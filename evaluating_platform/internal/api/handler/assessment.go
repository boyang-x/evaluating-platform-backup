package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"evaluating_platform/internal/agent"
	"evaluating_platform/internal/billing"
	"evaluating_platform/internal/connector"
	"evaluating_platform/internal/hub"
	"evaluating_platform/internal/mcptools"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/report"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/llm"
	"evaluating_platform/pkg/logger"
)

// AssessmentHandler 评估任务处理器
type AssessmentHandler struct {
	engine         *agent.Engine
	reporter       *report.Generator
	llmClient      *llm.Client
	pool           *mcptools.ConnectorPool
	assessmentRepo *repository.AssessmentRepository
	reportRepo     *repository.ReportRepository
	billingService *billing.Service
	logHub         *hub.LogHub
	jwtSecret      string
	cancelFuncs    sync.Map // map[assessmentID string] -> context.CancelFunc
}

// NewAssessmentHandler 创建处理器
func NewAssessmentHandler(
	engine *agent.Engine,
	reporter *report.Generator,
	llmClient *llm.Client,
	pool *mcptools.ConnectorPool,
	assessmentRepo *repository.AssessmentRepository,
	reportRepo *repository.ReportRepository,
	billingService *billing.Service,
	logHub *hub.LogHub,
	jwtSecret string,
) *AssessmentHandler {
	return &AssessmentHandler{
		engine:         engine,
		reporter:       reporter,
		llmClient:      llmClient,
		pool:           pool,
		assessmentRepo: assessmentRepo,
		reportRepo:     reportRepo,
		billingService: billingService,
		logHub:         logHub,
		jwtSecret:      jwtSecret,
	}
}

// CreateAssessmentRequest 创建评估请求体
type CreateAssessmentRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Goal        string `json:"goal" binding:"required"`
	TargetType  string `json:"target_type" binding:"required"` // openai | agent | custom
	TargetURL   string `json:"target_url" binding:"required"`
	TargetKey   string `json:"target_key"`
	TargetModel string `json:"target_model"`
	TemplateID  string `json:"template_id"`
}

// Create 创建并异步执行评估任务
func (h *AssessmentHandler) Create(c *gin.Context) {
	var req CreateAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从 JWT 中获取用户 ID
	userID, _ := uuid.Parse(c.GetString("user_id"))

	ctx := c.Request.Context()

	// 余额预检：确保余额满足最低要求
	if h.billingService != nil {
		if err := h.billingService.CheckSufficientBalance(ctx, userID); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":  "余额不足，无法发起评估",
				"detail": err.Error(),
			})
			return
		}
	}

	// 创建或复用目标系统记录
	targetID, err := h.assessmentRepo.CreateOrGetTargetSystem(ctx, userID, req.TargetType, req.TargetURL, req.TargetKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目标系统失败"})
		return
	}

	// 创建评估任务记录（pending 状态）
	assessment := &model.Assessment{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Goal:        req.Goal,
		TargetID:    targetID,
		Status:      model.StatusPending,
		Plan:        []string{},
	}
	if req.TemplateID != "" {
		tid, err := uuid.Parse(req.TemplateID)
		if err == nil {
			assessment.TemplateID = &tid
		}
	}

	if err := h.assessmentRepo.Create(ctx, assessment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建评估任务失败"})
		return
	}

	// 异步执行评估（使用独立 context，不受 HTTP 请求生命周期影响）
	go h.runAssessment(assessment, req)

	c.JSON(http.StatusAccepted, gin.H{
		"assessment_id": assessment.ID,
		"status":        assessment.Status,
		"message":       "评估任务已创建，正在后台执行",
	})
}

// runAssessment 异步执行评估任务
func (h *AssessmentHandler) runAssessment(assessment *model.Assessment, req CreateAssessmentRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 注册取消函数，供 Cancel handler 调用
	h.cancelFuncs.Store(assessment.ID.String(), cancel)
	defer h.cancelFuncs.Delete(assessment.ID.String())

	// 更新状态为 running
	_ = h.assessmentRepo.UpdateStatus(ctx, assessment.ID, model.StatusRunning, "")

	// 创建目标连接器并注册到 ConnectorPool
	conn := connector.NewConnector(&connector.Config{
		Type:     req.TargetType,
		Endpoint: req.TargetURL,
		APIKey:   req.TargetKey,
		Model:    req.TargetModel,
	})
	h.pool.Register(assessment.ID.String(), conn)
	defer h.pool.Remove(assessment.ID.String())

	// 注入 LogHub 到 Agent Engine（让实时日志可以广播给 SSE 客户端）
	if h.logHub != nil {
		h.engine.SetHub(h.logHub)
	}

	// 执行评估：统一走 ReAct 路径，已移除 workflow DAG 专门执行链路
	var result *agent.RunResult
	var err error
	result, err = h.engine.Run(ctx, assessment.ID.String(), req.Goal)

	if err != nil {
		logger.Error("assessment failed", map[string]interface{}{
			"assessment_id": assessment.ID,
			"error":         err.Error(),
		})
		_ = h.assessmentRepo.UpdateStatus(ctx, assessment.ID, model.StatusFailed, err.Error())
		return
	}

	// 持久化工具调用日志
	for i, log := range result.Logs {
		alog := &model.AssessmentLog{
			ID:           uuid.New(),
			AssessmentID: assessment.ID,
			Iteration:    i + 1,
			ToolName:     log.ToolName,
			Input:        log.Input,
			Output:       log.Output,
			Thinking:     log.Severity,
			TokensUsed:   log.TokensUsed,
		}
		_ = h.assessmentRepo.CreateLog(ctx, alog)
	}

	// 幂等检查：若报告已存在则跳过生成
	var rpt *model.Report
	if existing, err := h.reportRepo.GetByAssessmentID(ctx, assessment.ID); err == nil && existing != nil {
		rpt = existing
	} else {
		// 尝试 LLM 报告生成
		generated, genErr := h.reporter.Generate(ctx, assessment, result.Logs)
		if genErr != nil {
			// 降级：使用无 LLM 的基础报告，不标 failed
			logger.Warn("LLM report failed, using fallback", map[string]interface{}{
				"assessment_id": assessment.ID,
				"error":         genErr.Error(),
			})
			generated = h.reporter.GenerateFallback(ctx, assessment, result.Logs)
		}
		rpt = generated

		// 保存报告（降级报告也保存）
		if err := h.reportRepo.Create(ctx, rpt); err != nil {
			logger.Error("save report failed", map[string]interface{}{
				"assessment_id": assessment.ID,
				"error":         err.Error(),
			})
		}
	}

	// 标记完成
	_ = h.assessmentRepo.Complete(ctx, assessment.ID)

	// 计费处理：扣减用户余额并持久化计费记录
	if h.billingService != nil {
		toolCalls := make([]billing.ToolCallInfo, 0, len(result.Logs))
		for _, log := range result.Logs {
			toolCalls = append(toolCalls, billing.ToolCallInfo{
				ToolName:   log.ToolName,
				TokensUsed: log.TokensUsed,
			})
		}
		if err := h.billingService.ProcessAssessmentBilling(
			ctx, assessment.UserID, assessment.ID, toolCalls, result.TokensUsed,
			nil, nil,
		); err != nil {
			// 计费失败仅记录告警，不影响评估完成状态
			logger.Error("billing failed (non-fatal)", map[string]interface{}{
				"assessment_id": assessment.ID,
				"user_id":       assessment.UserID,
				"error":         err.Error(),
			})
		}
	}

	logger.Info("assessment completed", map[string]interface{}{
		"assessment_id": assessment.ID,
		"risk_level":    rpt.RiskLevel,
		"tokens_used":   result.TokensUsed,
	})
}

// Get 获取评估任务详情
func (h *AssessmentHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assessment id"})
		return
	}

	assessment, err := h.assessmentRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "评估任务不存在"})
		return
	}

	// 非 admin 用户只能查看自己的评估（返回 404 而非 403，避免信息泄漏）
	userIDStr := c.GetString("user_id")
	userRole := c.GetString("user_role")
	if userRole != "admin" && assessment.UserID.String() != userIDStr {
		c.JSON(http.StatusNotFound, gin.H{"error": "评估任务不存在"})
		return
	}

	// 如果已完成，附带报告摘要
	resp := gin.H{"assessment": assessment}
	if assessment.Status == model.StatusCompleted {
		if rpt, err := h.reportRepo.GetByAssessmentID(c.Request.Context(), id); err == nil {
			resp["report"] = gin.H{
				"id":         rpt.ID,
				"risk_level": rpt.RiskLevel,
				"summary":    rpt.Summary,
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

// List 列出评估任务（分页）
func (h *AssessmentHandler) List(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 100 {
		limit = 100
	}

	assessments, total, err := h.assessmentRepo.ListByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  assessments,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Cancel 取消评估任务
// POST /api/v1/assessments/:id/cancel
func (h *AssessmentHandler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assessment id"})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))

	// 如果任务正在运行，中断其 goroutine context
	if cancelFn, ok := h.cancelFuncs.Load(id.String()); ok {
		cancelFn.(context.CancelFunc)()
	}

	// 更新数据库状态（仅 pending/running 可取消，且只能取消自己的任务）
	if err := h.assessmentRepo.Cancel(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("assessment canceled", map[string]interface{}{
		"assessment_id": id,
		"user_id":       userID,
	})
	c.JSON(http.StatusOK, gin.H{"message": "评估任务已取消"})
}

// StreamLogs SSE 实时日志流
// GET /api/v1/assessments/:id/stream?token=<jwt>
// Delete 删除已结束的评估任务
// DELETE /api/v1/assessments/:id
func (h *AssessmentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assessment id"})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.assessmentRepo.Delete(c.Request.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, repository.ErrAssessmentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "assessment not found"})
		case errors.Is(err, repository.ErrAssessmentActive):
			c.JSON(http.StatusConflict, gin.H{"error": "assessment is still running, cancel it before deleting"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "delete assessment failed"})
		}
		return
	}

	logger.Info("assessment deleted", map[string]interface{}{
		"assessment_id": id,
		"user_id":       userID,
	})
	c.JSON(http.StatusOK, gin.H{"message": "assessment deleted"})
}

// DeleteAll 删除当前用户全部已结束的评估记录。
// DELETE /api/v1/assessments
func (h *AssessmentHandler) DeleteAll(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	result, err := h.assessmentRepo.DeleteAll(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete assessments failed"})
		return
	}

	logger.Info("assessments bulk deleted", map[string]interface{}{
		"user_id":       userID,
		"deleted_count": result.DeletedCount,
		"active_count":  result.ActiveCount,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":       "assessments deleted",
		"deleted_count": result.DeletedCount,
		"active_count":  result.ActiveCount,
	})
}

func (h *AssessmentHandler) StreamLogs(c *gin.Context) {
	assessmentID := c.Param("id")
	if _, err := uuid.Parse(assessmentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assessment id"})
		return
	}

	// 支持 query param token（EventSource 不支持自定义 header）
	tokenStr := c.Query("token")
	if tokenStr == "" {
		authHeader := c.GetHeader("Authorization")
		if parts := strings.SplitN(authHeader, " ", 2); len(parts) == 2 {
			tokenStr = parts[1]
		}
	}
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	// 验证 JWT
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
		return
	}

	userRole, _ := claims["role"].(string)
	userIDStr, _ := claims["sub"].(string)

	// 如果不是 admin，验证任务归属
	if userRole != "admin" {
		assessment, err := h.assessmentRepo.GetByID(c.Request.Context(), uuid.MustParse(assessmentID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "assessment not found"})
			return
		}
		if assessment.UserID.String() != userIDStr {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	if h.logHub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "log hub not initialized"})
		return
	}

	// 订阅日志
	logCh, unsub := h.logHub.Subscribe(assessmentID)
	defer unsub()

	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	// 立即发送初始注释，触发响应头刷新（EventSource 需要这个才能确认连接）
	fmt.Fprintf(c.Writer, ": connected\n\n")
	c.Writer.Flush()

	writeEvent := func(data interface{}) bool {
		b, _ := json.Marshal(data)
		_, err := fmt.Fprintf(c.Writer, "data: %s\n\n", b)
		if err != nil {
			return false
		}
		c.Writer.Flush()
		return true
	}

	clientDone := c.Request.Context().Done()
	ticker := time.NewTicker(25 * time.Second) // heartbeat
	defer ticker.Stop()

	for {
		select {
		case <-clientDone:
			return
		case <-ticker.C:
			// 发送心跳，防止连接超时
			fmt.Fprintf(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		case event, open := <-logCh:
			if !open {
				writeEvent(map[string]string{"type": "done"})
				return
			}
			if !writeEvent(event) {
				return
			}
		}
	}
}
