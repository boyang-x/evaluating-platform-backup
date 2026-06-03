package maclaw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
)

func TestResourceProjectionListsPublishedExpertResourcesForEnterpriseCatalog(t *testing.T) {
	ctx := context.Background()
	expertID := uuid.New()
	store := newFakePublicationStore()
	shadows := newFakeShadowStore()
	service := NewResourceProjectionService(&fakeProjectionProvider{}, store, shadows)

	err := service.RecordExpertResource(ctx, RuntimeIdentity{UserID: expertID.String(), Role: "expert"}, &GatewaySession{
		Mapping: &model.MaclawAccountMapping{PlatformUserID: expertID, MaclawTenantID: "tenant_expert"},
	}, &EvaluationResourceSummary{
		ID:              "res_pub",
		Handle:          "handle_pub",
		Name:            "Expert sample",
		Kind:            EvaluationResourceKindSample,
		Version:         "v1",
		Status:          EvaluationResourceStatusPublished,
		Enabled:         true,
		Summary:         "safe summary",
		AssessmentTypes: []string{"prompt_injection"},
		Tags:            []string{"expert"},
	})
	if err != nil {
		t.Fatalf("RecordExpertResource published: %v", err)
	}
	err = service.RecordExpertResource(ctx, RuntimeIdentity{UserID: expertID.String(), Role: "expert"}, &GatewaySession{
		Mapping: &model.MaclawAccountMapping{PlatformUserID: expertID, MaclawTenantID: "tenant_expert"},
	}, &EvaluationResourceSummary{
		ID:      "res_draft",
		Handle:  "handle_draft",
		Name:    "Draft",
		Kind:    EvaluationResourceKindSample,
		Version: "v1",
		Status:  EvaluationResourceStatusDraft,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("RecordExpertResource draft: %v", err)
	}

	items, err := service.ListEnterpriseCatalog(ctx, []EvaluationResourceSummary{{ID: "own", Handle: "own_handle", Name: "Own resource"}}, EvaluationResourceQuery{})
	if err != nil {
		t.Fatalf("ListEnterpriseCatalog: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %#v, want own + one published expert resource", items)
	}
	expertItem := items[1]
	if expertItem.ID != "res_pub" || expertItem.Handle != "handle_pub" {
		t.Fatalf("expert item = %#v", expertItem)
	}
	if expertItem.Metadata["requires_shadow_copy"] != "true" || expertItem.Metadata["source_expert_user_id"] != expertID.String() {
		t.Fatalf("expert metadata = %#v", expertItem.Metadata)
	}
}

func TestResourceProjectionCreatesEnterpriseShadowResourceOnce(t *testing.T) {
	ctx := context.Background()
	expertID := uuid.New()
	enterpriseID := uuid.New()
	store := newFakePublicationStore()
	shadows := newFakeShadowStore()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/evaluation/resources/materialize" {
			t.Fatalf("unexpected source request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(EvaluationResourceMaterialization{
			ResourceID: "res_pub",
			Handle:     "handle_pub",
			Name:       "Expert sample",
			Kind:       EvaluationResourceKindSample,
			Version:    "v1",
			Payload:    "SECRET_PAYLOAD",
		})
	}))
	defer source.Close()

	var saveCount int
	var saved EvaluationResourceInput
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/evaluation/resources" {
			t.Fatalf("unexpected target request %s %s", r.Method, r.URL.Path)
		}
		saveCount++
		if err := json.NewDecoder(r.Body).Decode(&saved); err != nil {
			t.Fatalf("decode shadow input: %v", err)
		}
		_ = json.NewEncoder(w).Encode(EvaluationResourceSummary{
			ID:      "shadow_res",
			Handle:  "shadow_handle",
			Name:    saved.Name,
			Kind:    saved.Kind,
			Version: saved.Version,
			Status:  saved.Status,
			Enabled: saved.Enabled,
		})
	}))
	defer target.Close()

	sourceClient, err := NewClient(Config{BaseURL: source.URL, APIToken: "source-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("source client: %v", err)
	}
	targetClient, err := NewClient(Config{BaseURL: target.URL, APIToken: "target-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("target client: %v", err)
	}
	provider := &fakeProjectionProvider{sessions: map[string]*GatewaySession{
		expertID.String():     {Client: sourceClient, InstanceID: "inst_expert", Mapping: &model.MaclawAccountMapping{PlatformUserID: expertID, MaclawTenantID: "tenant_expert"}},
		enterpriseID.String(): {Client: targetClient, InstanceID: "inst_enterprise", Mapping: &model.MaclawAccountMapping{PlatformUserID: enterpriseID, MaclawTenantID: "tenant_enterprise"}},
	}}
	service := NewResourceProjectionService(provider, store, shadows)
	if err := store.UpsertPublication(ctx, &model.MaclawResourcePublication{
		SourceExpertUserID:   expertID,
		SourceMaclawTenantID: "tenant_expert",
		SourceResourceID:     "res_pub",
		SourceResourceHandle: "handle_pub",
		SourceVersion:        "v1",
		Name:                 "Expert sample",
		Kind:                 string(EvaluationResourceKindSample),
		Status:               string(EvaluationResourceStatusPublished),
		Enabled:              true,
		Summary:              "safe summary",
	}); err != nil {
		t.Fatalf("seed publication: %v", err)
	}

	resolved, err := service.ResolveEnterpriseRunResourceHandles(ctx, RuntimeIdentity{UserID: enterpriseID.String(), Role: "enterprise"}, provider.sessions[enterpriseID.String()], []string{"handle_pub", "own_handle"})
	if err != nil {
		t.Fatalf("ResolveEnterpriseRunResourceHandles: %v", err)
	}
	if got, want := resolved[0], "shadow_handle"; got != want {
		t.Fatalf("resolved[0] = %q, want %q", got, want)
	}
	if got, want := resolved[1], "own_handle"; got != want {
		t.Fatalf("resolved[1] = %q, want %q", got, want)
	}
	if saveCount != 1 {
		t.Fatalf("saveCount = %d, want 1", saveCount)
	}
	if saved.Payload != "SECRET_PAYLOAD" {
		t.Fatalf("shadow payload was not copied through server side")
	}
	if saved.Metadata["source_expert_user_id"] != expertID.String() || saved.Metadata["source_resource_id"] != "res_pub" || saved.Metadata["shadow_created_by"] != "bff" {
		t.Fatalf("shadow metadata = %#v", saved.Metadata)
	}

	resolvedAgain, err := service.ResolveEnterpriseRunResourceHandles(ctx, RuntimeIdentity{UserID: enterpriseID.String(), Role: "enterprise"}, provider.sessions[enterpriseID.String()], []string{"handle_pub"})
	if err != nil {
		t.Fatalf("ResolveEnterpriseRunResourceHandles second: %v", err)
	}
	if resolvedAgain[0] != "shadow_handle" {
		t.Fatalf("resolvedAgain = %#v", resolvedAgain)
	}
	if saveCount != 1 {
		t.Fatalf("saveCount after idempotent reuse = %d, want 1", saveCount)
	}
}

func TestResourceProjectionNativeCatalogProfileFallsBackToShadowUntilGrantSyncExists(t *testing.T) {
	ctx := context.Background()
	expertID := uuid.New()
	enterpriseID := uuid.New()
	store := newFakePublicationStore()
	shadows := newFakeShadowStore()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/evaluation/resources/materialize" {
			t.Fatalf("unexpected source request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(EvaluationResourceMaterialization{
			ResourceID: "res_pub",
			Handle:     "handle_pub",
			Name:       "Expert sample",
			Kind:       EvaluationResourceKindSample,
			Version:    "v1",
			Payload:    "SECRET_PAYLOAD",
		})
	}))
	defer source.Close()

	var saveCount int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/evaluation/resources" {
			t.Fatalf("unexpected target request %s %s", r.Method, r.URL.Path)
		}
		saveCount++
		_ = json.NewEncoder(w).Encode(EvaluationResourceSummary{
			ID:      "shadow_res",
			Handle:  "shadow_handle",
			Name:    "Expert sample",
			Kind:    EvaluationResourceKindSample,
			Version: "v1",
			Status:  EvaluationResourceStatusPublished,
			Enabled: true,
		})
	}))
	defer target.Close()

	sourceClient, err := NewClient(Config{BaseURL: source.URL, APIToken: "source-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("source client: %v", err)
	}
	targetClient, err := NewClient(Config{BaseURL: target.URL, APIToken: "target-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("target client: %v", err)
	}
	provider := &fakeProjectionProvider{sessions: map[string]*GatewaySession{
		expertID.String():     {Client: sourceClient, InstanceID: "inst_expert", Mapping: &model.MaclawAccountMapping{PlatformUserID: expertID, MaclawTenantID: "tenant_expert"}},
		enterpriseID.String(): {Client: targetClient, InstanceID: "inst_enterprise", Mapping: &model.MaclawAccountMapping{PlatformUserID: enterpriseID, MaclawTenantID: "tenant_enterprise"}},
	}}
	service := NewResourceProjectionService(provider, store, shadows)
	service.SetRuntimeCapabilities(RuntimeCapabilities{
		Profile:                     RuntimeCapabilityProfileNativeCatalog,
		NativeResourceCatalog:       true,
		NativeCrossTenantReferences: true,
	})
	if err := store.UpsertPublication(ctx, &model.MaclawResourcePublication{
		SourceExpertUserID:   expertID,
		SourceMaclawTenantID: "tenant_expert",
		SourceResourceID:     "res_pub",
		SourceResourceHandle: "handle_pub",
		SourceVersion:        "v1",
		Name:                 "Expert sample",
		Kind:                 string(EvaluationResourceKindSample),
		Status:               string(EvaluationResourceStatusPublished),
		Enabled:              true,
	}); err != nil {
		t.Fatalf("seed publication: %v", err)
	}

	if err := service.SyncEnterprisePublishedResources(ctx, RuntimeIdentity{UserID: enterpriseID.String(), Role: "enterprise"}, provider.sessions[enterpriseID.String()]); err != nil {
		t.Fatalf("SyncEnterprisePublishedResources: %v", err)
	}
	resolved, err := service.ResolveEnterpriseRunResourceHandles(ctx, RuntimeIdentity{UserID: enterpriseID.String(), Role: "enterprise"}, provider.sessions[enterpriseID.String()], []string{"handle_pub"})
	if err != nil {
		t.Fatalf("ResolveEnterpriseRunResourceHandles: %v", err)
	}
	if resolved[0] != "shadow_handle" {
		t.Fatalf("native catalog without grant sync should use fallback shadow handle, got %#v", resolved)
	}
	if saveCount != 1 {
		t.Fatalf("saveCount = %d, want fallback shadow creation once", saveCount)
	}
}

type fakeProjectionProvider struct {
	sessions map[string]*GatewaySession
}

func (p *fakeProjectionProvider) Enabled() bool { return true }

func (p *fakeProjectionProvider) Resolve(_ context.Context, identity RuntimeIdentity) (*GatewaySession, error) {
	return p.sessions[identity.UserID], nil
}

type fakePublicationStore struct {
	items map[string]*model.MaclawResourcePublication
}

func newFakePublicationStore() *fakePublicationStore {
	return &fakePublicationStore{items: make(map[string]*model.MaclawResourcePublication)}
}

func (s *fakePublicationStore) UpsertPublication(_ context.Context, item *model.MaclawResourcePublication) error {
	now := time.Now()
	copied := *item
	copied.UpdatedAt = now
	if copied.CreatedAt.IsZero() {
		copied.CreatedAt = now
	}
	s.items[copied.SourceResourceHandle] = &copied
	return nil
}

func (s *fakePublicationStore) MarkPublicationUnavailable(_ context.Context, sourceExpertUserID uuid.UUID, sourceResourceID string) error {
	for _, item := range s.items {
		if item.SourceExpertUserID == sourceExpertUserID && item.SourceResourceID == sourceResourceID {
			item.Enabled = false
			item.Status = string(EvaluationResourceStatusArchived)
		}
	}
	return nil
}

func (s *fakePublicationStore) ListPublished(_ context.Context, _ EvaluationResourceQuery) ([]model.MaclawResourcePublication, error) {
	out := make([]model.MaclawResourcePublication, 0, len(s.items))
	for _, item := range s.items {
		if item.Enabled && item.Status == string(EvaluationResourceStatusPublished) {
			out = append(out, *item)
		}
	}
	return out, nil
}

func (s *fakePublicationStore) GetPublishedByHandle(_ context.Context, handle string) (*model.MaclawResourcePublication, error) {
	item := s.items[handle]
	if item == nil || !item.Enabled || item.Status != string(EvaluationResourceStatusPublished) {
		return nil, nil
	}
	copied := *item
	return &copied, nil
}

type fakeShadowStore struct {
	items map[string]*model.MaclawResourceShadow
}

func newFakeShadowStore() *fakeShadowStore {
	return &fakeShadowStore{items: make(map[string]*model.MaclawResourceShadow)}
}

func (s *fakeShadowStore) GetShadow(_ context.Context, enterpriseUserID uuid.UUID, sourceResourceID, sourceVersion string) (*model.MaclawResourceShadow, error) {
	item := s.items[enterpriseUserID.String()+"|"+sourceResourceID+"|"+sourceVersion]
	if item == nil {
		return nil, nil
	}
	copied := *item
	return &copied, nil
}

func (s *fakeShadowStore) UpsertShadow(_ context.Context, item *model.MaclawResourceShadow) error {
	copied := *item
	s.items[item.EnterpriseUserID.String()+"|"+item.SourceResourceID+"|"+item.SourceVersion] = &copied
	return nil
}
