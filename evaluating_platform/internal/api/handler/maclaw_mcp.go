package handler

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"evaluating_platform/internal/maclaw"
)

type MaclawMCPHandler struct {
	provider maclaw.GatewayProvider
}

func NewMaclawMCPHandler(provider maclaw.GatewayProvider) *MaclawMCPHandler {
	return &MaclawMCPHandler{provider: provider}
}

func (h *MaclawMCPHandler) ListServers(c *gin.Context) {
	gateway, ok := h.resolveExpertMCPGateway(c)
	if !ok {
		return
	}
	limit, err := parseOptionalPositiveInt(c.Query("limit"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}
	items, err := gateway.ListMCPServers(c.Request.Context(), limit)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": safeMCPServerViews(items)})
}

func (h *MaclawMCPHandler) CreateServer(c *gin.Context) {
	gateway, ok := h.resolveExpertMCPGateway(c)
	if !ok {
		return
	}
	var in maclaw.MCPServerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in = expertRemoteMCPInput(in)
	if err := validateExpertRemoteMCPInput(in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "invalid_mcp_server"})
		return
	}
	out, err := gateway.CreateMCPServer(c.Request.Context(), in)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusCreated, safeMCPServerView(out))
}

func (h *MaclawMCPHandler) GetServer(c *gin.Context) {
	gateway, ok := h.resolveExpertMCPGateway(c)
	if !ok {
		return
	}
	out, err := gateway.GetMCPServer(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, safeMCPServerView(out))
}

func (h *MaclawMCPHandler) UpdateServer(c *gin.Context) {
	gateway, ok := h.resolveExpertMCPGateway(c)
	if !ok {
		return
	}
	var in maclaw.MCPServerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in = expertRemoteMCPInput(in)
	if err := validateExpertRemoteMCPInput(in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "invalid_mcp_server"})
		return
	}
	out, err := gateway.UpdateMCPServer(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, safeMCPServerView(out))
}

func (h *MaclawMCPHandler) DeleteServer(c *gin.Context) {
	gateway, ok := h.resolveExpertMCPGateway(c)
	if !ok {
		return
	}
	if err := gateway.DeleteMCPServer(c.Request.Context(), c.Param("id")); err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *MaclawMCPHandler) StartServer(c *gin.Context) {
	h.mutateServer(c, func(g maclaw.MCPGateway) (*maclaw.MCPServerView, error) {
		return g.StartMCPServer(c.Request.Context(), c.Param("id"))
	})
}

func (h *MaclawMCPHandler) StopServer(c *gin.Context) {
	h.mutateServer(c, func(g maclaw.MCPGateway) (*maclaw.MCPServerView, error) {
		return g.StopMCPServer(c.Request.Context(), c.Param("id"))
	})
}

func (h *MaclawMCPHandler) HealthCheckServer(c *gin.Context) {
	h.mutateServer(c, func(g maclaw.MCPGateway) (*maclaw.MCPServerView, error) {
		return g.HealthCheckMCPServer(c.Request.Context(), c.Param("id"))
	})
}

func (h *MaclawMCPHandler) ListServerTools(c *gin.Context) {
	gateway, ok := h.resolveExpertMCPGateway(c)
	if !ok {
		return
	}
	items, err := gateway.ListMCPServerTools(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *MaclawMCPHandler) mutateServer(c *gin.Context, fn func(maclaw.MCPGateway) (*maclaw.MCPServerView, error)) {
	gateway, ok := h.resolveExpertMCPGateway(c)
	if !ok {
		return
	}
	out, err := fn(gateway)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, safeMCPServerView(out))
}

func (h *MaclawMCPHandler) resolveExpertMCPGateway(c *gin.Context) (maclaw.MCPGateway, bool) {
	role := strings.TrimSpace(c.GetString("user_role"))
	if role != "expert" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "maclaw mcp server management is available to expert accounts", "code": "forbidden"})
		return nil, false
	}
	session, ok := resolveMaclawGatewaySession(c, h.provider)
	if !ok {
		return nil, false
	}
	gateway, ok := maclaw.MCPGatewayFromClient(session.Client)
	if !ok || gateway == nil || !gateway.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw mcp server API is not supported", "code": "maclaw_mcp_not_supported"})
		return nil, false
	}
	return gateway, true
}

func expertRemoteMCPInput(in maclaw.MCPServerInput) maclaw.MCPServerInput {
	in.Kind = strings.TrimSpace(strings.ToLower(in.Kind))
	if in.Kind == "" {
		in.Kind = "remote"
	}
	in.Name = strings.TrimSpace(in.Name)
	in.EndpointURL = strings.TrimSpace(in.EndpointURL)
	in.AuthType = strings.TrimSpace(strings.ToLower(in.AuthType))
	in.AuthSecret = strings.TrimSpace(in.AuthSecret)
	in.Command = ""
	in.Args = nil
	in.Env = nil
	return in
}

func validateExpertRemoteMCPInput(in maclaw.MCPServerInput) error {
	if in.Kind != "remote" {
		return mcpValidationError("only remote HTTP MCP servers are supported")
	}
	if in.Name == "" {
		return mcpValidationError("mcp name is required")
	}
	if in.EndpointURL == "" {
		return mcpValidationError("mcp endpoint url is required")
	}
	parsed, err := url.ParseRequestURI(in.EndpointURL)
	if err != nil || parsed == nil || parsed.Host == "" {
		return mcpValidationError("mcp endpoint url must be a valid HTTP or HTTPS URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return mcpValidationError("mcp endpoint url must use http or https")
	}
	if blocked, err := isPrivateMCPRemoteHost(parsed.Hostname()); err != nil {
		return mcpValidationError(err.Error())
	} else if blocked {
		return mcpValidationError("mcp endpoint url must not target localhost, link-local, or private network hosts")
	}
	return nil
}

func isPrivateMCPRemoteHost(host string) (bool, error) {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" {
		return false, mcpValidationError("mcp endpoint url host is required")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true, nil
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return blockedMCPRemoteIP(ip), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false, nil
	}
	for _, addr := range addrs {
		ip, ok := netip.AddrFromSlice(addr.IP)
		if ok && blockedMCPRemoteIP(ip) {
			return true, nil
		}
	}
	return false, nil
}

func blockedMCPRemoteIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

type mcpValidationError string

func (e mcpValidationError) Error() string { return string(e) }

func safeMCPServerViews(items []maclaw.MCPServerView) []maclaw.MCPServerView {
	out := make([]maclaw.MCPServerView, 0, len(items))
	for i := range items {
		out = append(out, *safeMCPServerView(&items[i]))
	}
	return out
}

func safeMCPServerView(item *maclaw.MCPServerView) *maclaw.MCPServerView {
	if item == nil {
		return nil
	}
	out := *item
	out.Command = ""
	out.Args = nil
	out.EnvKeys = nil
	out.HasEnv = false
	if strings.TrimSpace(out.Kind) != "remote" {
		out.EndpointURL = ""
	} else {
		out.EndpointURL = safeMCPRemoteEndpointURL(out.EndpointURL)
	}
	return &out
}

func safeMCPRemoteEndpointURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(strings.TrimSpace(key))
		if strings.Contains(lower, "token") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "key") ||
			strings.Contains(lower, "signature") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "credential") {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
