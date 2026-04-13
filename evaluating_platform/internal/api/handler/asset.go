package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/logger"
)

// AssetHandler 资产处理器（专家工具上传/管理）
type AssetHandler struct {
	assetRepo *repository.AssetRepository
}

// NewAssetHandler 创建资产处理器
func NewAssetHandler(assetRepo *repository.AssetRepository) *AssetHandler {
	return &AssetHandler{assetRepo: assetRepo}
}

// CreateAssetRequest 创建资产请求体
type CreateAssetRequest struct {
	Name        string                `json:"name" binding:"required"`
	Description string                `json:"description"`
	Type        model.AssetType       `json:"type" binding:"required"`
	Visibility  model.AssetVisibility `json:"visibility"`
	Version     string                `json:"version"`
	Config      model.WorkflowConfig  `json:"config"`
	PriceUnit   float64               `json:"price_unit"`
}

// UploadTool 专家创建/上传资产
// POST /api/v1/tools
// POST /api/v1/assets
func (h *AssetHandler) UploadTool(c *gin.Context) {
	var req CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.Type {
	case model.AssetTypeToolConfig, model.AssetTypeWorkflow, model.AssetTypeSuite:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type 必须为 tool_config | workflow | suite"})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))

	visibility := req.Visibility
	if visibility == "" {
		visibility = model.VisibilityPrivate
	}
	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	asset := &model.Asset{
		ID:          uuid.New(),
		ExpertID:    expertID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Visibility:  visibility,
		Status:      model.AssetStatusDraft,
		Version:     version,
		Config:      req.Config,
		PriceUnit:   req.PriceUnit,
	}

	if err := h.assetRepo.Create(c.Request.Context(), asset); err != nil {
		logger.Error("create asset failed", map[string]interface{}{
			"expert_id": expertID,
			"error":     err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建资产失败"})
		return
	}

	logger.Info("asset created", map[string]interface{}{
		"asset_id":  asset.ID,
		"expert_id": expertID,
		"type":      asset.Type,
	})
	c.JSON(http.StatusCreated, gin.H{
		"asset_id": asset.ID,
		"status":   asset.Status,
		"message":  "资产已创建，待发布",
	})
}

// ListTools 列出资产
// GET /api/v1/tools  - 公开市场（已发布的 public 资产）
// GET /api/v1/assets - 专家查看自己的资产（传 ?mine=1）
//
// Query params:
//
//	type=workflow|tool_config|suite  按类型过滤（公开市场有效）
//	mine=1                           专家/管理员查看自己的所有资产（含 draft）
//	limit=20  offset=0               分页
func (h *AssetHandler) ListTools(c *gin.Context) {
	assetType := c.Query("type")
	mine := c.Query("mine") == "1" || c.Query("mine") == "true"
	role := c.GetString("user_role")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 100 {
		limit = 100
	}

	ctx := c.Request.Context()

	// 专家/管理员查看自己的资产
	if mine && (role == "expert" || role == "admin") {
		expertID, _ := uuid.Parse(c.GetString("user_id"))
		assets, total, err := h.assetRepo.ListByExpert(ctx, expertID, limit, offset)
		if err != nil {
			logger.Error("list expert assets failed", map[string]interface{}{
				"expert_id": expertID,
				"error":     err.Error(),
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"items":  assets,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
		return
	}

	// 默认：公开市场（published + public）
	assets, total, err := h.assetRepo.ListPublic(ctx, assetType, limit, offset)
	if err != nil {
		logger.Error("list public assets failed", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  assets,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// PublishTool 发布资产（仅资产所有者）
// 允许从 draft 或 testing 状态发布。
// PUT /api/v1/tools/:id/publish
func (h *AssetHandler) PublishTool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.assetRepo.PublishAsset(c.Request.Context(), id, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("asset published", map[string]interface{}{
		"asset_id":  id,
		"expert_id": expertID,
	})
	c.JSON(http.StatusOK, gin.H{
		"asset_id": id,
		"status":   model.AssetStatusPublished,
	})
}

// SubmitForReview 提交资产审核（draft → testing）
// PUT /api/v1/tools/:id/submit
func (h *AssetHandler) SubmitForReview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.assetRepo.UpdateStatusFrom(
		c.Request.Context(), id, expertID,
		model.AssetStatusDraft, model.AssetStatusTesting,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("asset submitted for review", map[string]interface{}{
		"asset_id":  id,
		"expert_id": expertID,
	})
	c.JSON(http.StatusOK, gin.H{
		"asset_id": id,
		"status":   model.AssetStatusTesting,
		"message":  "资产已提交审核",
	})
}

// DeprecateTool 废弃资产（可从任意状态转换）
// PUT /api/v1/tools/:id/deprecate
func (h *AssetHandler) DeprecateTool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.assetRepo.UpdateStatus(c.Request.Context(), id, expertID, model.AssetStatusDeprecated); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("asset deprecated", map[string]interface{}{
		"asset_id":  id,
		"expert_id": expertID,
	})
	c.JSON(http.StatusOK, gin.H{
		"asset_id": id,
		"status":   model.AssetStatusDeprecated,
	})
}
