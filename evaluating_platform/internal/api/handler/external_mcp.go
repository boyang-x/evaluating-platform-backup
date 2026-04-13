package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/externalmcp"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
)

type ExternalMCPHandler struct {
	serverRepo *repository.ExternalMCPServerRepository
	toolRepo   *repository.ExternalMCPToolRepository
	manager    *externalmcp.Manager
}

func NewExternalMCPHandler(
	serverRepo *repository.ExternalMCPServerRepository,
	toolRepo *repository.ExternalMCPToolRepository,
	manager *externalmcp.Manager,
) *ExternalMCPHandler {
	return &ExternalMCPHandler{serverRepo: serverRepo, toolRepo: toolRepo, manager: manager}
}

type upsertExternalMCPServerRequest struct {
	Name            string `json:"name" binding:"required"`
	Namespace       string `json:"namespace"`
	Description     string `json:"description"`
	BaseURL         string `json:"base_url" binding:"required"`
	TransportType   string `json:"transport_type"`
	AuthType        string `json:"auth_type"`
	AuthKey         string `json:"auth_key"`
	AuthHeader      string `json:"auth_header"`
	AuthPrefix      string `json:"auth_prefix"`
	UpstreamBaseURL string `json:"upstream_base_url"`
	UpstreamAPIKey  string `json:"upstream_api_key"`
	UpstreamModel   string `json:"upstream_model"`
	UpstreamTimeout int    `json:"upstream_timeout_seconds"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	Enabled         *bool  `json:"enabled"`
	SkillPrompt     string `json:"skill_prompt"`
}

func (h *ExternalMCPHandler) ListServers(c *gin.Context) {
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	items, err := h.serverRepo.ListByExpert(c.Request.Context(), expertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]gin.H, 0, len(items))
	for _, item := range items {
		response = append(response, serializeExternalMCPServer(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": response})
}

func (h *ExternalMCPHandler) CreateServer(c *gin.Context) {
	var req upsertExternalMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item := buildExternalMCPServerModel(req, expertID)
	item.ID = uuid.New()
	item.Enabled = enabled
	item.Status = initialStatus(enabled)

	if err := h.serverRepo.Create(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	warning := ""
	if item.Enabled {
		if err := h.manager.SyncServer(c.Request.Context(), item.ID); err != nil {
			warning = err.Error()
		}
		fresh, _ := h.serverRepo.GetByID(c.Request.Context(), item.ID)
		if fresh != nil {
			item = fresh
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"item":    serializeExternalMCPServer(*item),
		"warning": warning,
	})
}

func (h *ExternalMCPHandler) UpdateServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req upsertExternalMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expertID, _ := uuid.Parse(c.GetString("user_id"))
	existing, err := h.serverRepo.GetByIDForExpert(c.Request.Context(), id, expertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "external mcp server not found"})
		return
	}

	updated := buildExternalMCPServerModel(req, expertID)
	updated.ID = id
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = existing.UpdatedAt
	updated.LastSyncAt = existing.LastSyncAt
	updated.LastError = existing.LastError
	updated.Status = existing.Status

	if req.Enabled == nil {
		updated.Enabled = existing.Enabled
	}
	if strings.TrimSpace(req.AuthKey) == "" && updated.AuthType != "none" {
		updated.AuthKey = existing.AuthKey
	}
	if strings.TrimSpace(req.UpstreamAPIKey) == "" && strings.TrimSpace(existing.UpstreamAPIKey) != "" {
		updated.UpstreamAPIKey = existing.UpstreamAPIKey
	}

	if !updated.Enabled {
		updated.Status = "disabled"
		updated.LastError = ""
		updated.LastSyncAt = nil
		if err := h.serverRepo.Update(c.Request.Context(), updated); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_ = h.manager.DisableServer(c.Request.Context(), id)
		fresh, _ := h.serverRepo.GetByID(c.Request.Context(), id)
		if fresh != nil {
			updated = fresh
		}
		c.JSON(http.StatusOK, gin.H{"item": serializeExternalMCPServer(*updated)})
		return
	}

	if err := h.serverRepo.Update(c.Request.Context(), updated); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	warning := ""
	if err := h.manager.SyncServer(c.Request.Context(), id); err != nil {
		warning = err.Error()
	}
	fresh, _ := h.serverRepo.GetByID(c.Request.Context(), id)
	if fresh != nil {
		updated = fresh
	}

	c.JSON(http.StatusOK, gin.H{
		"item":    serializeExternalMCPServer(*updated),
		"warning": warning,
	})
}

func (h *ExternalMCPHandler) DeleteServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	h.manager.RemoveServer(id)
	if err := h.serverRepo.Delete(c.Request.Context(), id, expertID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "external mcp server deleted"})
}

func (h *ExternalMCPHandler) TestServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	item, err := h.serverRepo.GetByIDForExpert(c.Request.Context(), id, expertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "external mcp server not found"})
		return
	}

	result, err := h.manager.TestConnection(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "result": result})
}

func (h *ExternalMCPHandler) SyncServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	item, err := h.serverRepo.GetByIDForExpert(c.Request.Context(), id, expertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "external mcp server not found"})
		return
	}

	if err := h.manager.SyncServer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	tools, err := h.toolRepo.ListByServer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fresh, _ := h.serverRepo.GetByID(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"item":  serializeExternalMCPServer(*fresh),
		"tools": tools,
	})
}

func (h *ExternalMCPHandler) ListTools(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	expertID, _ := uuid.Parse(c.GetString("user_id"))
	item, err := h.serverRepo.GetByIDForExpert(c.Request.Context(), id, expertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "external mcp server not found"})
		return
	}

	tools, err := h.toolRepo.ListByServer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tools})
}

func buildExternalMCPServerModel(req upsertExternalMCPServerRequest, expertID uuid.UUID) *model.ExternalMCPServer {
	namespace := sanitizeNamespace(req.Namespace)
	if namespace == "" {
		namespace = sanitizeNamespace(req.Name)
	}
	transportType := strings.TrimSpace(req.TransportType)
	if transportType == "" {
		transportType = "sse"
	}
	authType := strings.TrimSpace(req.AuthType)
	if authType == "" {
		authType = "none"
	}
	timeoutSeconds := req.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	return &model.ExternalMCPServer{
		ExpertID:        expertID,
		Name:            strings.TrimSpace(req.Name),
		Namespace:       namespace,
		Description:     strings.TrimSpace(req.Description),
		BaseURL:         strings.TrimSpace(req.BaseURL),
		TransportType:   transportType,
		AuthType:        authType,
		AuthKey:         strings.TrimSpace(req.AuthKey),
		AuthHeader:      strings.TrimSpace(req.AuthHeader),
		AuthPrefix:      strings.TrimSpace(req.AuthPrefix),
		UpstreamBaseURL: strings.TrimSpace(req.UpstreamBaseURL),
		UpstreamAPIKey:  strings.TrimSpace(req.UpstreamAPIKey),
		UpstreamModel:   strings.TrimSpace(req.UpstreamModel),
		UpstreamTimeout: req.UpstreamTimeout,
		TimeoutSeconds:  timeoutSeconds,
		SkillPrompt:     strings.TrimSpace(req.SkillPrompt),
		Enabled:         true,
	}
}

func serializeExternalMCPServer(item model.ExternalMCPServer) gin.H {
	return gin.H{
		"id":                          item.ID,
		"expert_id":                   item.ExpertID,
		"name":                        item.Name,
		"namespace":                   item.Namespace,
		"description":                 item.Description,
		"base_url":                    item.BaseURL,
		"transport_type":              item.TransportType,
		"auth_type":                   item.AuthType,
		"auth_header":                 item.AuthHeader,
		"auth_prefix":                 item.AuthPrefix,
		"upstream_base_url":           item.UpstreamBaseURL,
		"upstream_model":              item.UpstreamModel,
		"upstream_timeout_seconds":    item.UpstreamTimeout,
		"upstream_api_key_configured": strings.TrimSpace(item.UpstreamAPIKey) != "",
		"auth_key_configured":         strings.TrimSpace(item.AuthKey) != "",
		"timeout_seconds":             item.TimeoutSeconds,
		"enabled":                     item.Enabled,
		"skill_prompt":                item.SkillPrompt,
		"status":                      item.Status,
		"last_sync_at":                item.LastSyncAt,
		"last_error":                  item.LastError,
		"tool_count":                  item.ToolCount,
		"created_at":                  item.CreatedAt,
		"updated_at":                  item.UpdatedAt,
	}
}

func sanitizeNamespace(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("-", "_", " ", "_")
	value = replacer.Replace(value)
	builder := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		}
	}
	sanitized := builder.String()
	sanitized = strings.Trim(sanitized, "_")
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}
	return sanitized
}

func initialStatus(enabled bool) string {
	if enabled {
		return "pending"
	}
	return "disabled"
}
