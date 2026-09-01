package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"evaluating_platform/internal/maclaw"
)

func maclawRuntimeIdentity(c *gin.Context) maclaw.RuntimeIdentity {
	return maclaw.RuntimeIdentity{
		UserID: strings.TrimSpace(c.GetString("user_id")),
		Role:   strings.TrimSpace(c.GetString("user_role")),
	}
}

func resolveMaclawGatewaySession(c *gin.Context, provider maclaw.GatewayProvider) (*maclaw.GatewaySession, bool) {
	if provider == nil || !provider.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw runtime is not configured", "code": "maclaw_not_configured"})
		return nil, false
	}
	userID := strings.TrimSpace(c.GetString("user_id"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication is required", "code": "unauthorized"})
		return nil, false
	}
	session, err := provider.Resolve(c.Request.Context(), maclawRuntimeIdentity(c))
	if err != nil {
		log.Printf("[WARN] maclaw gateway resolve failed for role=%s user=%s: %v", c.GetString("user_role"), userID, err)
		if errors.Is(err, maclaw.ErrNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw runtime is not configured", "code": "maclaw_not_configured"})
			return nil, false
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw account provisioning failed", "code": "maclaw_provisioning_failed"})
		return nil, false
	}
	if session == nil || session.Client == nil || !session.Client.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw runtime is not configured", "code": "maclaw_not_configured"})
		return nil, false
	}
	return session, true
}
