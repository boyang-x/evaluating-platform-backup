package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	chatpkg "evaluating_platform/internal/chat"
)

type EnterpriseWelcomeHandler struct {
	service *chatpkg.WelcomeCapabilitiesService
}

func NewEnterpriseWelcomeHandler(service *chatpkg.WelcomeCapabilitiesService) *EnterpriseWelcomeHandler {
	return &EnterpriseWelcomeHandler{service: service}
}

func (h *EnterpriseWelcomeHandler) ListCapabilities(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusOK, gin.H{"items": []chatpkg.WelcomeCapability{}})
		return
	}

	limit := 6
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= limit {
			limit = parsed
		}
	}

	items, err := h.service.List(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
