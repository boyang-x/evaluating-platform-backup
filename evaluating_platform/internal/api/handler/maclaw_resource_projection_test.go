package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/maclaw"
	"evaluating_platform/internal/model"
)

func TestMaclawEvaluationHandlerProjectsExpertResourceBeforeStartingRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	expertID := uuid.New()
	enterpriseID := uuid.New()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/evaluation/resources/materialize" {
			t.Fatalf("unexpected source request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(maclaw.EvaluationResourceMaterialization{
			ResourceID: "res_pub",
			Handle:     "expert_handle",
			Name:       "Expert prompt",
			Kind:       maclaw.EvaluationResourceKindSample,
			Version:    "v1",
			Payload:    "SECRET_PAYLOAD",
		})
	}))
	defer source.Close()

	var runInput maclaw.EvaluationRunInput
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/evaluation/resources":
			var in maclaw.EvaluationResourceInput
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Fatalf("decode shadow resource: %v", err)
			}
			if in.Payload != "SECRET_PAYLOAD" {
				t.Fatalf("shadow payload = %q", in.Payload)
			}
			_ = json.NewEncoder(w).Encode(maclaw.EvaluationResourceSummary{
				ID:      "shadow_res",
				Handle:  "shadow_handle",
				Name:    in.Name,
				Kind:    in.Kind,
				Version: in.Version,
				Status:  in.Status,
				Enabled: in.Enabled,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/evaluation/runs":
			if err := json.NewDecoder(r.Body).Decode(&runInput); err != nil {
				t.Fatalf("decode run input: %v", err)
			}
			_ = json.NewEncoder(w).Encode(maclaw.EvaluationRunResult{
				Run: &maclaw.RuntimeRun{ID: "run_1", Status: "running"},
			})
		default:
			t.Fatalf("unexpected enterprise request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer target.Close()

	sourceClient, err := maclaw.NewClient(maclaw.Config{BaseURL: source.URL, APIToken: "source-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("source client: %v", err)
	}
	targetClient, err := maclaw.NewClient(maclaw.Config{BaseURL: target.URL, APIToken: "target-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("target client: %v", err)
	}
	provider := &projectionHandlerProvider{sessions: map[string]*maclaw.GatewaySession{
		expertID.String():     {Client: sourceClient, Mapping: &model.MaclawAccountMapping{PlatformUserID: expertID, PlatformRole: model.RoleExpert, MaclawTenantID: "tenant_expert"}},
		enterpriseID.String(): {Client: targetClient, InstanceID: "inst_enterprise", Mapping: &model.MaclawAccountMapping{PlatformUserID: enterpriseID, PlatformRole: model.RoleEnterprise, MaclawTenantID: "tenant_enterprise"}},
	}}
	publications := newProjectionHandlerPublicationStore()
	if err := publications.UpsertPublication(ctx, &model.MaclawResourcePublication{
		SourceExpertUserID:   expertID,
		SourceMaclawTenantID: "tenant_expert",
		SourceResourceID:     "res_pub",
		SourceResourceHandle: "expert_handle",
		SourceVersion:        "v1",
		Name:                 "Expert prompt",
		Kind:                 string(maclaw.EvaluationResourceKindSample),
		Status:               string(maclaw.EvaluationResourceStatusPublished),
		Enabled:              true,
		Summary:              "safe summary",
	}); err != nil {
		t.Fatalf("seed publication: %v", err)
	}
	projection := maclaw.NewResourceProjectionService(provider, publications, newProjectionHandlerShadowStore())
	h := NewMaclawEvaluationHandlerWithProviderAndProjection(provider, projection)

	router := gin.New()
	router.POST("/maclaw/evaluation/runs", func(c *gin.Context) {
		c.Set("user_id", enterpriseID.String())
		c.Set("user_role", "enterprise")
		h.StartRun(c)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/runs", strings.NewReader(`{
		"session_id":"sess_1",
		"target_id":"target_1",
		"resource_handles":["expert_handle","own_handle"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if runInput.InstanceID != "inst_enterprise" {
		t.Fatalf("run instance id = %q", runInput.InstanceID)
	}
	if len(runInput.ResourceHandles) != 2 || runInput.ResourceHandles[0] != "shadow_handle" || runInput.ResourceHandles[1] != "own_handle" {
		t.Fatalf("run resource handles = %#v", runInput.ResourceHandles)
	}
}

type projectionHandlerProvider struct {
	sessions map[string]*maclaw.GatewaySession
}

func (p *projectionHandlerProvider) Enabled() bool { return true }

func (p *projectionHandlerProvider) Resolve(_ context.Context, identity maclaw.RuntimeIdentity) (*maclaw.GatewaySession, error) {
	return p.sessions[identity.UserID], nil
}

type projectionHandlerPublicationStore struct {
	items map[string]*model.MaclawResourcePublication
}

func newProjectionHandlerPublicationStore() *projectionHandlerPublicationStore {
	return &projectionHandlerPublicationStore{items: map[string]*model.MaclawResourcePublication{}}
}

func (s *projectionHandlerPublicationStore) UpsertPublication(_ context.Context, item *model.MaclawResourcePublication) error {
	copied := *item
	s.items[copied.SourceResourceHandle] = &copied
	s.items[copied.SourceResourceID] = &copied
	return nil
}

func (s *projectionHandlerPublicationStore) MarkPublicationUnavailable(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (s *projectionHandlerPublicationStore) ListPublished(_ context.Context, _ maclaw.EvaluationResourceQuery) ([]model.MaclawResourcePublication, error) {
	seen := map[string]bool{}
	out := []model.MaclawResourcePublication{}
	for _, item := range s.items {
		if seen[item.SourceResourceID] || !item.Enabled || item.Status != string(maclaw.EvaluationResourceStatusPublished) {
			continue
		}
		seen[item.SourceResourceID] = true
		out = append(out, *item)
	}
	return out, nil
}

func (s *projectionHandlerPublicationStore) GetPublishedByHandle(_ context.Context, handle string) (*model.MaclawResourcePublication, error) {
	item := s.items[handle]
	if item == nil || !item.Enabled || item.Status != string(maclaw.EvaluationResourceStatusPublished) {
		return nil, nil
	}
	copied := *item
	return &copied, nil
}

type projectionHandlerShadowStore struct {
	items map[string]*model.MaclawResourceShadow
}

func newProjectionHandlerShadowStore() *projectionHandlerShadowStore {
	return &projectionHandlerShadowStore{items: map[string]*model.MaclawResourceShadow{}}
}

func (s *projectionHandlerShadowStore) GetShadow(_ context.Context, enterpriseUserID uuid.UUID, sourceResourceID, sourceVersion string) (*model.MaclawResourceShadow, error) {
	item := s.items[enterpriseUserID.String()+"|"+sourceResourceID+"|"+sourceVersion]
	if item == nil {
		return nil, nil
	}
	copied := *item
	return &copied, nil
}

func (s *projectionHandlerShadowStore) UpsertShadow(_ context.Context, item *model.MaclawResourceShadow) error {
	copied := *item
	s.items[item.EnterpriseUserID.String()+"|"+item.SourceResourceID+"|"+item.SourceVersion] = &copied
	return nil
}
