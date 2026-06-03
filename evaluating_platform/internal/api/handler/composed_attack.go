package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/repository"
	"evaluating_platform/internal/sample"
	"evaluating_platform/pkg/logger"
	"evaluating_platform/pkg/storage"
)

type ComposedAttackHandler struct {
	repo    *repository.ComposedAttackRepository
	manager *sample.ComposedAttackManager
	loader  *sample.ComposedAttackLoader
	store   *storage.MinIOClient
}

func NewComposedAttackHandler(
	repo *repository.ComposedAttackRepository,
	manager *sample.ComposedAttackManager,
	loader *sample.ComposedAttackLoader,
	store *storage.MinIOClient,
) *ComposedAttackHandler {
	return &ComposedAttackHandler{
		repo:    repo,
		manager: manager,
		loader:  loader,
		store:   store,
	}
}

func (h *ComposedAttackHandler) Upload(c *gin.Context) {
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	subType := c.PostForm("sub_type")
	name := c.PostForm("name")
	description := c.PostForm("description")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if subType == "" {
		subType = "ready_to_run"
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	csvData, err := readLimitedUpload(file, maxExpertDataUploadBytes)
	if err != nil {
		writeUploadReadError(c, err)
		return
	}

	item, err := h.manager.Upload(c.Request.Context(), expertID, subType, name, description, csvData)
	if err != nil {
		logger.Error("upload composed attack failed", map[string]interface{}{
			"expert_id": expertID,
			"error":     err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           item.ID,
		"name":         item.Name,
		"sub_type":     item.SubType,
		"sample_count": item.SampleCount,
		"file_hash":    item.FileHash,
	})
}

func (h *ComposedAttackHandler) List(c *gin.Context) {
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	subType := c.Query("sub_type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.repo.ListByExpert(c.Request.Context(), expertID, subType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query composed attacks failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *ComposedAttackHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))

	storagePath, err := h.repo.Delete(c.Request.Context(), id, expertID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if storagePath != "" {
		_ = h.store.Delete(c.Request.Context(), storagePath)
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (h *ComposedAttackHandler) Preview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}

	payloads, err := h.loader.Preview(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"preview": payloads, "limit": limit})
}
