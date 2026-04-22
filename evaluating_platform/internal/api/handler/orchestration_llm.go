package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/connector"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/logger"
)

// OrchestrationLLMHandler 编排 LLM 配置处理器
type OrchestrationLLMHandler struct {
	repo *repository.OrchestrationLLMRepository
}

// NewOrchestrationLLMHandler 创建编排 LLM 配置处理器
func NewOrchestrationLLMHandler(repo *repository.OrchestrationLLMRepository) *OrchestrationLLMHandler {
	return &OrchestrationLLMHandler{repo: repo}
}

// GetConfig 获取当前用户的编排 LLM 配置
// GET /api/v1/orchestration-llm/config
func (h *OrchestrationLLMHandler) GetConfig(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	cfg, err := h.repo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		logger.Error("get orchestration LLM config failed", map[string]interface{}{
			"user_id": userID, "error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "orchestration LLM not configured"})
		return
	}

	// 返回脱敏后的配置
	c.JSON(http.StatusOK, gin.H{
		"id":         cfg.ID,
		"user_id":    cfg.UserID,
		"base_url":   cfg.BaseURL,
		"api_key":    maskAPIKey(cfg.APIKey),
		"model":      cfg.Model,
		"created_at": cfg.CreatedAt,
		"updated_at": cfg.UpdatedAt,
	})
}

// UpdateConfig 更新编排 LLM 配置
// PUT /api/v1/orchestration-llm/config
func (h *OrchestrationLLMHandler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.BaseURL = normalizeLLMBaseURL(req.BaseURL)
	if err := validateOpenAICompatibleProviderConfig(req.BaseURL, req.Model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))

	// 如果 api_key 为空，尝试保留已有的 key
	apiKey := req.APIKey
	if apiKey == "" {
		existing, err := h.repo.GetByUserID(c.Request.Context(), userID)
		if err == nil && existing != nil {
			apiKey = existing.APIKey
		}
		if apiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "api_key is required"})
			return
		}
	}

	cfg := &model.OrchestrationLLMConfig{
		ID:      uuid.New(),
		UserID:  userID,
		BaseURL: req.BaseURL,
		APIKey:  apiKey,
		Model:   req.Model,
	}

	if err := h.repo.Upsert(c.Request.Context(), cfg); err != nil {
		logger.Error("upsert orchestration LLM config failed", map[string]interface{}{
			"user_id": userID, "error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         cfg.ID,
		"base_url":   cfg.BaseURL,
		"model":      cfg.Model,
		"updated_at": cfg.UpdatedAt,
	})
}

// TestConnection 测试编排 LLM 连接
// POST /api/v1/orchestration-llm/test
func (h *OrchestrationLLMHandler) TestConnection(c *gin.Context) {
	var req TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.BaseURL = normalizeLLMBaseURL(req.BaseURL)
	if err := validateOpenAICompatibleProviderConfig(req.BaseURL, req.Model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))

	// 如果 api_key 为空，尝试使用已保存的 key
	apiKey := req.APIKey
	if apiKey == "" {
		existing, err := h.repo.GetByUserID(c.Request.Context(), userID)
		if err == nil && existing != nil {
			apiKey = existing.APIKey
		}
		if apiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "api_key is required"})
			return
		}
	}

	// 使用 connector.NewConnector 创建临时连接器测试（编排 LLM 固定使用 openai 类型）
	conn := connector.NewConnector(&connector.Config{
		Type:     "openai",
		Endpoint: req.BaseURL,
		APIKey:   apiKey,
		Model:    req.Model,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	resp, err := conn.SendMessage(ctx, &connector.AssessRequest{
		Messages: []connector.Message{
			{Role: "user", Content: "Hello, this is a connection test. Reply with 'OK'."},
		},
	})
	if err != nil {
		logger.Warn("orchestration LLM connection test failed", map[string]interface{}{
			"base_url": req.BaseURL, "error": err.Error(),
		})
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"error":   "connection test failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"response": resp.Content,
	})
}
