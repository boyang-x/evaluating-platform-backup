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

// TargetLLMHandler 被测 LLM 配置处理器
type TargetLLMHandler struct {
	repo *repository.TargetLLMRepository
}

// NewTargetLLMHandler 创建被测 LLM 配置处理器
func NewTargetLLMHandler(repo *repository.TargetLLMRepository) *TargetLLMHandler {
	return &TargetLLMHandler{repo: repo}
}

// UpdateTargetConfigRequest 更新被测 LLM 配置请求
type UpdateTargetConfigRequest struct {
	BaseURL       string `json:"base_url" binding:"required"`
	APIKey        string `json:"api_key"`                      // 可选，留空则保持原有 key
	Model         string `json:"model"`
	ConnectorType string `json:"connector_type"`               // openai | dify | custom
}

// TestTargetConnectionRequest 测试被测 LLM 连接请求
type TestTargetConnectionRequest struct {
	BaseURL       string `json:"base_url" binding:"required"`
	APIKey        string `json:"api_key"`                      // 可选，留空则使用已保存的 key
	Model         string `json:"model"`
	ConnectorType string `json:"connector_type"`               // openai | dify | custom
}

// GetConfig 获取当前用户的被测 LLM 配置
// GET /api/v1/target-llm/config
func (h *TargetLLMHandler) GetConfig(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	cfg, err := h.repo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		logger.Error("get target LLM config failed", map[string]interface{}{
			"user_id": userID, "error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "target LLM not configured"})
		return
	}

	// 返回脱敏后的配置（含 connector_type）
	c.JSON(http.StatusOK, gin.H{
		"id":             cfg.ID,
		"user_id":        cfg.UserID,
		"base_url":       cfg.BaseURL,
		"api_key":        maskAPIKey(cfg.APIKey),
		"model":          cfg.Model,
		"connector_type": cfg.ConnectorType,
		"created_at":     cfg.CreatedAt,
		"updated_at":     cfg.UpdatedAt,
	})
}

// UpdateConfig 更新被测 LLM 配置
// PUT /api/v1/target-llm/config
func (h *TargetLLMHandler) UpdateConfig(c *gin.Context) {
	var req UpdateTargetConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	// connector_type 默认为 openai
	connectorType := req.ConnectorType
	if connectorType == "" {
		connectorType = "openai"
	}

	cfg := &model.TargetLLMConfig{
		ID:            uuid.New(),
		UserID:        userID,
		BaseURL:       req.BaseURL,
		APIKey:        apiKey,
		Model:         req.Model,
		ConnectorType: connectorType,
	}

	if err := h.repo.Upsert(c.Request.Context(), cfg); err != nil {
		logger.Error("upsert target LLM config failed", map[string]interface{}{
			"user_id": userID, "error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             cfg.ID,
		"base_url":       cfg.BaseURL,
		"model":          cfg.Model,
		"connector_type": cfg.ConnectorType,
		"updated_at":     cfg.UpdatedAt,
	})
}

// TestConnection 测试被测 LLM 连接
// POST /api/v1/target-llm/test
func (h *TargetLLMHandler) TestConnection(c *gin.Context) {
	var req TestTargetConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	// connector_type 默认为 openai
	connectorType := req.ConnectorType
	if connectorType == "" {
		connectorType = "openai"
	}

	// 使用 connector.NewConnector 创建对应类型的连接器测试
	conn := connector.NewConnector(&connector.Config{
		Type:     connectorType,
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
		logger.Warn("target LLM connection test failed", map[string]interface{}{
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
