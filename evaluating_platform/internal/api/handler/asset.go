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

// AssetHandler handles expert asset upload and management endpoints.
type AssetHandler struct {
	assetRepo *repository.AssetRepository
}

func NewAssetHandler(assetRepo *repository.AssetRepository) *AssetHandler {
	return &AssetHandler{assetRepo: assetRepo}
}

type CreateAssetRequest struct {
	Name        string                `json:"name" binding:"required"`
	Description string                `json:"description"`
	Type        model.AssetType       `json:"type" binding:"required"`
	Visibility  model.AssetVisibility `json:"visibility"`
	Version     string                `json:"version"`
	Config      model.WorkflowConfig  `json:"config"`
	PriceUnit   float64               `json:"price_unit"`
}

// UploadTool creates an asset and makes it immediately available.
// POST /api/v1/tools
// POST /api/v1/assets
func (h *AssetHandler) UploadTool(c *gin.Context) {
	var req CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.Type {
	case model.AssetTypeToolConfig, model.AssetTypeSuite:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be tool_config or suite"})
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
		Status:      model.AssetStatusPublished,
		Version:     version,
		Config:      req.Config,
		PriceUnit:   req.PriceUnit,
	}

	if err := h.assetRepo.Create(c.Request.Context(), asset); err != nil {
		logger.Error("create asset failed", map[string]interface{}{
			"expert_id": expertID,
			"error":     err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create asset failed"})
		return
	}

	logger.Info("asset created", map[string]interface{}{
		"asset_id":  asset.ID,
		"expert_id": expertID,
		"type":      asset.Type,
		"status":    asset.Status,
	})
	c.JSON(http.StatusCreated, gin.H{
		"asset_id": asset.ID,
		"status":   asset.Status,
		"message":  "asset created and available immediately",
	})
}

// ListTools lists public assets or the current expert's assets.
// GET /api/v1/tools
// GET /api/v1/assets
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
	if mine && (role == "expert" || role == "admin") {
		expertID, _ := uuid.Parse(c.GetString("user_id"))
		assets, total, err := h.assetRepo.ListByExpert(ctx, expertID, limit, offset)
		if err != nil {
			logger.Error("list expert assets failed", map[string]interface{}{
				"expert_id": expertID,
				"error":     err.Error(),
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query assets failed"})
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

	assets, total, err := h.assetRepo.ListPublic(ctx, assetType, limit, offset)
	if err != nil {
		logger.Error("list public assets failed", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query assets failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  assets,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// PublishTool remains as a compatibility endpoint and now simply ensures the asset is available.
// PUT /api/v1/tools/:id/publish
func (h *AssetHandler) PublishTool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.assetRepo.UpdateStatus(c.Request.Context(), id, expertID, model.AssetStatusPublished); err != nil {
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
		"message":  "asset is available",
	})
}

// SubmitForReview is retained as a compatibility alias after removing the expert review flow.
// PUT /api/v1/tools/:id/submit
func (h *AssetHandler) SubmitForReview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.assetRepo.UpdateStatus(c.Request.Context(), id, expertID, model.AssetStatusPublished); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("asset marked available", map[string]interface{}{
		"asset_id":  id,
		"expert_id": expertID,
	})
	c.JSON(http.StatusOK, gin.H{
		"asset_id": id,
		"status":   model.AssetStatusPublished,
		"message":  "asset is available immediately",
	})
}

// DeprecateTool disables an asset for future use.
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
