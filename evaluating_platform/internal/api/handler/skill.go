package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	skillpkg "evaluating_platform/internal/skill"
)

type SkillHandler struct {
	service *skillpkg.Service
}

func NewSkillHandler(service *skillpkg.Service) *SkillHandler {
	return &SkillHandler{service: service}
}

func (h *SkillHandler) Import(c *gin.Context) {
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zip file is required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read zip file failed"})
		return
	}

	bundle, err := h.service.ImportArchive(c.Request.Context(), expertID, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": presentSkillBundleSummary(bundle)})
}

func (h *SkillHandler) List(c *gin.Context) {
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	items, err := h.service.ListByExpert(c.Request.Context(), expertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": presentSkillBundleSummaries(items)})
}

func (h *SkillHandler) Get(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	item, err := h.service.GetByID(c.Request.Context(), skillID, expertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": presentSkillBundleDetail(item)})
}

func (h *SkillHandler) SelfTest(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	versionID, err := uuid.Parse(c.Param("version_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	run, err := h.service.RunSelfTest(c.Request.Context(), skillID, versionID, expertID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": presentSkillRun(run, nil)})
}

func (h *SkillHandler) Publish(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	var req struct {
		VersionID string `json:"version_id"`
	}
	_ = c.ShouldBindJSON(&req)
	versionID := uuid.Nil
	if strings.TrimSpace(req.VersionID) != "" {
		versionID, err = uuid.Parse(req.VersionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version_id"})
			return
		}
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.service.Publish(c.Request.Context(), skillID, versionID, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "skill published"})
}

func (h *SkillHandler) Deprecate(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.service.Deprecate(c.Request.Context(), skillID, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "skill disabled"})
}

func (h *SkillHandler) Disable(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.service.Disable(c.Request.Context(), skillID, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "skill disabled"})
}

func (h *SkillHandler) Enable(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.service.Enable(c.Request.Context(), skillID, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "skill enabled"})
}

func (h *SkillHandler) Delete(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	if err := h.service.Delete(c.Request.Context(), skillID, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "skill deleted"})
}

func (h *SkillHandler) GetConfig(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	versionID := uuid.Nil
	if raw := strings.TrimSpace(c.Query("version_id")); raw != "" {
		versionID, err = uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version_id"})
			return
		}
	}
	view, err := h.service.GetSkillConfig(c.Request.Context(), skillID, expertID, versionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": presentSkillConfigView(view)})
}

func (h *SkillHandler) UpdateConfig(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	var req struct {
		VersionID string `json:"version_id"`
		Values    []struct {
			Key   string      `json:"key"`
			Value interface{} `json:"value"`
			Clear bool        `json:"clear"`
		} `json:"values"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	versionID := uuid.Nil
	if strings.TrimSpace(req.VersionID) != "" {
		versionParsed, parseErr := uuid.Parse(req.VersionID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version_id"})
			return
		}
		versionID = versionParsed
	}

	values := make([]skillpkg.SkillConfigUpdateValue, 0, len(req.Values))
	for _, item := range req.Values {
		values = append(values, skillpkg.SkillConfigUpdateValue{
			Key:   item.Key,
			Value: item.Value,
			Clear: item.Clear,
		})
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	view, err := h.service.UpdateSkillConfig(c.Request.Context(), skillID, expertID, skillpkg.SkillConfigUpdateRequest{
		VersionID: versionID,
		Values:    values,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": presentSkillConfigView(view)})
}

func (h *SkillHandler) ListRuns(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	items, err := h.service.ListRuns(c.Request.Context(), skillID, expertID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": presentSkillRuns(items, nil)})
}

func (h *SkillHandler) GetLaunchDocument(c *gin.Context) {
	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id"})
		return
	}

	doc, err := h.service.BuildLaunchDocument(c.Request.Context(), skillID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": doc})
}
