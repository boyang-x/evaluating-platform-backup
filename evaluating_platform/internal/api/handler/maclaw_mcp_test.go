package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"evaluating_platform/internal/maclaw"
)

func TestMaclawMCPHandlerProxiesExpertTenantMCPWithoutLeakingSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var created maclaw.MCPServerInput
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/mcp/servers":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []maclaw.MCPServerView{{
				ID:            "mcp_1",
				Kind:          "remote",
				Name:          "expert-data",
				EndpointURL:   "https://user:secret@mcp.example.test/path?token=secret-value&ok=1",
				AuthType:      "bearer",
				HasAuthSecret: true,
				HeaderNames:   []string{"X-Expert"},
				Running:       true,
				HealthStatus:  "healthy",
			}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/mcp/servers":
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Fatalf("decode create: %v", err)
			}
			_ = json.NewEncoder(w).Encode(maclaw.MCPServerView{
				ID:            "mcp_new",
				Kind:          created.Kind,
				Name:          created.Name,
				EndpointURL:   created.EndpointURL,
				AuthType:      created.AuthType,
				HasAuthSecret: created.AuthSecret != "",
				HeaderNames:   []string{"X-Expert"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/mcp/servers/mcp_new/tools":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []maclaw.MCPToolView{{Name: "search_samples", Description: "Search expert samples"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	client, err := maclaw.NewMaclawSrvClient(maclaw.Config{BaseURL: upstream.URL, APIToken: "request-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	provider := &fakeGatewayProvider{session: &maclaw.GatewaySession{Client: client, InstanceID: "inst_expert"}}
	handler := NewMaclawMCPHandler(provider)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "expert-user")
		c.Set("user_role", "expert")
		c.Next()
	})
	router.GET("/maclaw/mcp/servers", handler.ListServers)
	router.POST("/maclaw/mcp/servers", handler.CreateServer)
	router.GET("/maclaw/mcp/servers/:id/tools", handler.ListServerTools)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/maclaw/mcp/servers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "has_auth_secret") || strings.Contains(w.Body.String(), "secret-value") || strings.Contains(w.Body.String(), "user:secret") || strings.Contains(w.Body.String(), "token=") || strings.Contains(w.Body.String(), "env_keys") {
		t.Fatalf("list response leaked or missed secret marker: %s", w.Body.String())
	}

	body := `{"kind":"remote","name":"expert-data","endpoint_url":"https://mcp.example.test","auth_type":"bearer","auth_secret":"secret-value","headers":{"X-Expert":"demo"},"auto_start":true}`
	req := httptest.NewRequest(http.MethodPost, "/maclaw/mcp/servers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if created.Kind != "remote" || created.EndpointURL != "https://mcp.example.test" {
		t.Fatalf("created = %#v", created)
	}
	if strings.Contains(w.Body.String(), "secret-value") {
		t.Fatalf("create response leaked secret: %s", w.Body.String())
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/maclaw/mcp/servers/mcp_new/tools", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "search_samples") {
		t.Fatalf("tools status = %d body=%s", w.Code, w.Body.String())
	}
}

func TestMaclawMCPHandlerRejectsEnterpriseAndLocalCommandMCP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, err := maclaw.NewMaclawSrvClient(maclaw.Config{BaseURL: "http://maclaw.local", APIToken: "request-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	provider := &fakeGatewayProvider{session: &maclaw.GatewaySession{Client: client, InstanceID: "inst_enterprise"}}
	handler := NewMaclawMCPHandler(provider)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "enterprise-user")
		c.Set("user_role", "enterprise")
		c.Next()
	})
	router.POST("/maclaw/mcp/servers", handler.CreateServer)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/mcp/servers", strings.NewReader(`{"kind":"remote","name":"blocked","endpoint_url":"https://mcp.example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("enterprise status = %d body=%s", w.Code, w.Body.String())
	}

	router = gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "expert-user")
		c.Set("user_role", "expert")
		c.Next()
	})
	router.POST("/maclaw/mcp/servers", handler.CreateServer)
	req = httptest.NewRequest(http.MethodPost, "/maclaw/mcp/servers", strings.NewReader(`{"kind":"stdio","name":"local","command":"node","env":{"TOKEN":"secret"}}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("local status = %d body=%s", w.Code, w.Body.String())
	}
}

func TestMaclawMCPHandlerRejectsNonHTTPRemoteEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, err := maclaw.NewMaclawSrvClient(maclaw.Config{BaseURL: "http://maclaw.local", APIToken: "request-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	provider := &fakeGatewayProvider{session: &maclaw.GatewaySession{Client: client, InstanceID: "inst_expert"}}
	handler := NewMaclawMCPHandler(provider)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "expert-user")
		c.Set("user_role", "expert")
		c.Next()
	})
	router.POST("/maclaw/mcp/servers", handler.CreateServer)

	for _, endpoint := range []string{"file:///tmp/server", "javascript:alert(1)", "ftp://mcp.example.test"} {
		req := httptest.NewRequest(http.MethodPost, "/maclaw/mcp/servers", strings.NewReader(`{"kind":"remote","name":"bad","endpoint_url":"`+endpoint+`"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("endpoint %q status = %d body=%s", endpoint, w.Code, w.Body.String())
		}
	}
}

func TestMaclawMCPHandlerRejectsPrivateRemoteEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, err := maclaw.NewMaclawSrvClient(maclaw.Config{BaseURL: "http://maclaw.local", APIToken: "request-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	provider := &fakeGatewayProvider{session: &maclaw.GatewaySession{Client: client, InstanceID: "inst_expert"}}
	handler := NewMaclawMCPHandler(provider)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "expert-user")
		c.Set("user_role", "expert")
		c.Next()
	})
	router.POST("/maclaw/mcp/servers", handler.CreateServer)

	for _, endpoint := range []string{
		"http://localhost:8080/mcp",
		"http://127.0.0.1:8080/mcp",
		"http://[::1]:8080/mcp",
		"http://10.0.0.5/mcp",
		"http://172.16.0.5/mcp",
		"http://192.168.1.5/mcp",
		"http://169.254.169.254/latest/meta-data",
	} {
		req := httptest.NewRequest(http.MethodPost, "/maclaw/mcp/servers", strings.NewReader(`{"kind":"remote","name":"private","endpoint_url":"`+endpoint+`"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("endpoint %q status = %d body=%s", endpoint, w.Code, w.Body.String())
		}
	}
}
