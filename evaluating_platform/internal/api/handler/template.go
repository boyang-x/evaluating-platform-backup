package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/internal/sample"
	"evaluating_platform/pkg/logger"
)

type TemplateHandler struct {
	tplRepo *repository.TemplateRepository
}

func NewTemplateHandler(tplRepo *repository.TemplateRepository) *TemplateHandler {
	return &TemplateHandler{tplRepo: tplRepo}
}

type CreateTemplateRequest struct {
	SubType     string              `json:"sub_type" binding:"required"`
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description"`
	Content     string              `json:"content" binding:"required"`
	Variables   []model.TemplateVar `json:"variables"`
}

func (h *TemplateHandler) Create(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))
	batchID := uuid.New()

	tpl := &model.Template{
		ID:              uuid.New(),
		ExpertID:        expertID,
		UploadBatchID:   batchID,
		UploadBatchName: req.Name,
		SubType:         req.SubType,
		Name:            req.Name,
		Description:     req.Description,
		Content:         req.Content,
		Variables:       req.Variables,
		Status:          "published",
		Visibility:      "public",
	}

	if err := h.tplRepo.Create(c.Request.Context(), tpl); err != nil {
		logger.Error("create template failed", map[string]interface{}{
			"expert_id": expertID,
			"error":     err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create template failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                tpl.ID,
		"name":              tpl.Name,
		"upload_batch_id":   tpl.UploadBatchID,
		"upload_batch_name": tpl.UploadBatchName,
	})
}

func (h *TemplateHandler) List(c *gin.Context) {
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	subType := c.Query("sub_type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	templates, total, err := h.tplRepo.ListByExpert(c.Request.Context(), expertID, subType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query templates failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  templates,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *TemplateHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))

	tpl := &model.Template{
		ID:          id,
		ExpertID:    expertID,
		SubType:     req.SubType,
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Variables:   req.Variables,
	}

	if err := h.tplRepo.Update(c.Request.Context(), tpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "message": "template updated"})
}

func (h *TemplateHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.tplRepo.Delete(c.Request.Context(), id, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}

func (h *TemplateHandler) UploadCSV(c *gin.Context) {
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	subType := c.PostForm("sub_type")
	name := c.PostForm("name")

	if name == "" {
		name = "template batch"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	fileData, err := readLimitedUpload(file, maxExpertDataUploadBytes)
	if err != nil {
		writeUploadReadError(c, err)
		return
	}

	templateRows, err := sample.ParseTemplateRows(fileData, header.Filename)
	if err != nil {
		if subType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		legacyRows, legacyErr := sample.ParseTwoColumnCSV(fileData)
		if legacyErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		templateRows = make([]sample.TemplateRow, 0, len(legacyRows))
		for _, row := range legacyRows {
			templateRows = append(templateRows, sample.TemplateRow{
				Index:    row.Index,
				Category: subType,
				Content:  row.Data,
			})
		}
	}

	batchID := uuid.New()
	created := make([]uuid.UUID, 0, len(templateRows))
	for _, row := range templateRows {
		tpl := &model.Template{
			ID:              uuid.New(),
			ExpertID:        expertID,
			UploadBatchID:   batchID,
			UploadBatchName: name,
			SubType:         row.Category,
			Name:            name,
			Content:         row.Content,
			Status:          "published",
			Visibility:      "public",
		}
		if err := h.tplRepo.Create(c.Request.Context(), tpl); err != nil {
			logger.Error("create template from CSV failed", map[string]interface{}{
				"expert_id": expertID,
				"row_index": row.Index,
				"error":     err.Error(),
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create template failed"})
			return
		}
		created = append(created, tpl.ID)
	}

	c.JSON(http.StatusCreated, gin.H{
		"count":             len(created),
		"ids":               created,
		"upload_batch_id":   batchID,
		"upload_batch_name": name,
	})
}
