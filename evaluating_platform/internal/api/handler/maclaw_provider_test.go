package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"evaluating_platform/internal/maclaw"
)

func TestMaclawEvaluationHandlerUsesRequestScopedProviderClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var authHeader string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer upstream.Close()

	client, err := maclaw.NewClient(maclaw.Config{BaseURL: upstream.URL, APIToken: "request-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	provider := &fakeGatewayProvider{session: &maclaw.GatewaySession{Client: client, InstanceID: "inst_user"}}
	h := NewMaclawEvaluationHandlerWithProvider(provider)

	router := gin.New()
	router.GET("/resources", func(c *gin.Context) {
		c.Set("user_id", "platform-user-1")
		c.Set("user_role", "enterprise")
		h.ListResources(c)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resources", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if provider.identity.UserID != "platform-user-1" || provider.identity.Role != "enterprise" {
		t.Fatalf("identity = %#v", provider.identity)
	}
	if authHeader != "Bearer request-token" {
		t.Fatalf("authorization header = %q", authHeader)
	}
}

type fakeGatewayProvider struct {
	session  *maclaw.GatewaySession
	identity maclaw.RuntimeIdentity
}

func (p *fakeGatewayProvider) Enabled() bool {
	return p != nil && p.session != nil
}

func (p *fakeGatewayProvider) Resolve(_ context.Context, identity maclaw.RuntimeIdentity) (*maclaw.GatewaySession, error) {
	p.identity = identity
	return p.session, nil
}
