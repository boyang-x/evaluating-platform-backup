package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/detector"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/logger"
)

// AppDetectorHandler 应用检测工具处理器
type AppDetectorHandler struct {
	repo   *repository.AppDetectorRepository
	engine *detector.Engine
}

func NewAppDetectorHandler(repo *repository.AppDetectorRepository, engine *detector.Engine) *AppDetectorHandler {
	return &AppDetectorHandler{repo: repo, engine: engine}
}

type CreateDetectorRequest struct {
	Name         string                 `json:"name" binding:"required"`
	SubType      string                 `json:"sub_type" binding:"required"`
	Description  string                 `json:"description"`
	TargetConfig model.TargetAppConfig  `json:"target_config" binding:"required"`
	SampleIDs    []string               `json:"sample_ids"`
	TemplateIDs  []string               `json:"template_ids"`
	PackageIDs   []string               `json:"package_ids"`
}

// Create 创建应用检测工具
func (h *AppDetectorHandler) Create(c *gin.Context) {
	var req CreateDetectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))
	det := &model.AppDetector{
		ID:           uuid.New(),
		ExpertID:     expertID,
		Name:         req.Name,
		SubType:      model.AppDetectorSubType(req.SubType),
		Description:  req.Description,
		TargetConfig: req.TargetConfig,
		SampleIDs:    parseUUIDs(req.SampleIDs),
		TemplateIDs:  parseUUIDs(req.TemplateIDs),
		PackageIDs:   parseUUIDs(req.PackageIDs),
		Status:       "draft",
	}

	if err := h.repo.Create(c.Request.Context(), det); err != nil {
		logger.Error("create detector failed", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusCreated, det)
}

// List 列出检测工具
func (h *AppDetectorHandler) List(c *gin.Context) {
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.repo.ListByExpert(c.Request.Context(), expertID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "limit": limit, "offset": offset})
}

// Get 获取检测工具详情
func (h *AppDetectorHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	det, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "检测工具不存在"})
		return
	}
	c.JSON(http.StatusOK, det)
}

// Update 更新检测工具
func (h *AppDetectorHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req CreateDetectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	det := &model.AppDetector{
		ID:           id,
		Name:         req.Name,
		SubType:      model.AppDetectorSubType(req.SubType),
		Description:  req.Description,
		TargetConfig: req.TargetConfig,
		SampleIDs:    parseUUIDs(req.SampleIDs),
		TemplateIDs:  parseUUIDs(req.TemplateIDs),
		PackageIDs:   parseUUIDs(req.PackageIDs),
		Status:       "draft",
	}

	if err := h.repo.Update(c.Request.Context(), det); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// Delete 删除检测工具
func (h *AppDetectorHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.repo.Delete(c.Request.Context(), id, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// Run 执行检测
func (h *AppDetectorHandler) Run(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	det, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "检测工具不存在"})
		return
	}

	result, err := h.engine.Run(c.Request.Context(), det)
	if err != nil {
		logger.Error("run detector failed", map[string]interface{}{"id": id, "error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func parseUUIDs(ss []string) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		if id, err := uuid.Parse(s); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
