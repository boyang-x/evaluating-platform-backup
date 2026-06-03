package handler

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"

	"evaluating_platform/internal/maclaw"
)

func writeMaclawError(c *gin.Context, err error) {
	status, code, message := classifyMaclawError(err)
	c.JSON(status, gin.H{"error": message, "code": code})
}

func classifyMaclawError(err error) (int, string, string) {
	if err == nil {
		return http.StatusBadGateway, "maclaw_proxy_error", "maclaw request failed"
	}
	if errors.Is(err, maclaw.ErrNotConfigured) {
		return http.StatusServiceUnavailable, "maclaw_not_configured", "maclaw runtime is not configured"
	}

	var upstream *maclaw.UpstreamError
	if errors.As(err, &upstream) {
		switch upstream.StatusCode {
		case http.StatusBadRequest:
			return http.StatusBadRequest, "maclaw_bad_request", "maclaw rejected the request"
		case http.StatusUnauthorized, http.StatusForbidden:
			return http.StatusBadGateway, "maclaw_auth_failed", "maclaw authentication failed"
		case http.StatusNotFound:
			return http.StatusNotFound, "maclaw_not_found", "requested maclaw resource was not found"
		case http.StatusConflict:
			return http.StatusConflict, "maclaw_conflict", "maclaw resource conflict"
		case http.StatusUnprocessableEntity:
			return http.StatusUnprocessableEntity, "maclaw_validation_failed", "maclaw validation failed"
		case http.StatusTooManyRequests:
			return http.StatusTooManyRequests, "maclaw_rate_limited", "maclaw rate limit exceeded"
		case http.StatusServiceUnavailable:
			return http.StatusServiceUnavailable, "maclaw_unavailable", "maclaw runtime is unavailable"
		default:
			if upstream.StatusCode >= 500 {
				return http.StatusBadGateway, "maclaw_upstream_failed", "maclaw runtime request failed"
			}
			return http.StatusBadGateway, "maclaw_proxy_error", "maclaw request failed"
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, "maclaw_timeout", "maclaw runtime request timed out"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return http.StatusGatewayTimeout, "maclaw_timeout", "maclaw runtime request timed out"
	}
	return http.StatusServiceUnavailable, "maclaw_unavailable", "maclaw runtime is unavailable"
}
