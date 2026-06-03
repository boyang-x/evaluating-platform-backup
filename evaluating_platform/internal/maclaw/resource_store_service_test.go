package maclaw

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	appcrypto "evaluating_platform/internal/crypto"
	"evaluating_platform/internal/model"
	"evaluating_platform/pkg/config"
)

func TestResourceStoreServiceSavesSafeSummaryAndMaterializesServerSide(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	store := newFakeResourceStore()
	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatalf("keystore: %v", err)
	}
	service := NewResourceStoreService(store, keyStore)

	summary, err := service.SaveResource(ctx, userID, "tenant_1", EvaluationResourceInput{
		Name:         "Classical jailbreak samples",
		Kind:         EvaluationResourceKindSample,
		Version:      "v1",
		Status:       EvaluationResourceStatusPublished,
		Enabled:      true,
		Summary:      "Classical Chinese jailbreak prompts",
		Payload:      "SECRET_PAYLOAD_SHOULD_NOT_LEAK",
		Tags:         []string{"jailbreak", "classical_chinese"},
		Metadata:     map[string]string{"local_path": "C:/secret/path", "source": "expert"},
		HealthStatus: EvaluationResourceHealthHealthy,
	})
	if err != nil {
		t.Fatalf("SaveResource: %v", err)
	}
	if summary.Metadata["local_path"] != "" {
		t.Fatalf("summary leaked local path: %#v", summary.Metadata)
	}
	if store.last.EncryptedPayload == nil || strings.Contains(string(store.last.EncryptedPayload), "SECRET_PAYLOAD") {
		t.Fatalf("payload was not encrypted at rest: %q", string(store.last.EncryptedPayload))
	}

	materialized, err := service.MaterializeResource(ctx, userID, summary.Handle)
	if err != nil {
		t.Fatalf("MaterializeResource: %v", err)
	}
	if materialized.Payload != "SECRET_PAYLOAD_SHOULD_NOT_LEAK" {
		t.Fatalf("unexpected materialized payload: %#v", materialized)
	}
	if materialized.Metadata["local_path"] != "" {
		t.Fatalf("materialized metadata leaked local path: %#v", materialized.Metadata)
	}
}

func TestPlatformResourceGatewayOverridesUnsupportedRuntimeResourceAPIs(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	store := newFakeResourceStore()
	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatalf("keystore: %v", err)
	}
	service := NewResourceStoreService(store, keyStore)
	gateway := NewPlatformResourceGateway(nil, service, userID, "tenant_1")

	summary, err := gateway.SaveEvaluationResource(ctx, EvaluationResourceInput{
		Name:    "Prompt template",
		Kind:    EvaluationResourceKindTemplate,
		Enabled: true,
		Payload: "template body",
	})
	if err != nil {
		t.Fatalf("SaveEvaluationResource: %v", err)
	}
	items, err := gateway.SearchEvaluationResources(ctx, EvaluationResourceQuery{Query: "prompt"})
	if err != nil {
		t.Fatalf("SearchEvaluationResources: %v", err)
	}
	if len(items) != 1 || items[0].ID != summary.ID {
		t.Fatalf("unexpected search items: %#v", items)
	}
	preview, err := gateway.PreviewEvaluationResource(ctx, summary.ID)
	if err != nil {
		t.Fatalf("PreviewEvaluationResource: %v", err)
	}
	if preview.PayloadSHA256 == "" || preview.PayloadBytes == 0 {
		t.Fatalf("preview missing safe payload stats: %#v", preview)
	}
	materialized, err := gateway.MaterializeEvaluationResource(ctx, summary.Handle)
	if err != nil {
		t.Fatalf("MaterializeEvaluationResource: %v", err)
	}
	if materialized.Payload != "template body" {
		t.Fatalf("unexpected payload: %#v", materialized)
	}
}

type fakeResourceStore struct {
	items map[string]model.MaclawResourceRecord
	last  model.MaclawResourceRecord
}

func newFakeResourceStore() *fakeResourceStore {
	return &fakeResourceStore{items: map[string]model.MaclawResourceRecord{}}
}

func (s *fakeResourceStore) UpsertResource(_ context.Context, record model.MaclawResourceRecord) (*model.MaclawResourceRecord, error) {
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	s.last = record
	s.items[record.ID.String()] = record
	s.items[record.Handle] = record
	return &record, nil
}

func (s *fakeResourceStore) ListResources(_ context.Context, userID uuid.UUID, q EvaluationResourceQuery) ([]model.MaclawResourceRecord, error) {
	out := []model.MaclawResourceRecord{}
	for key, item := range s.items {
		if key != item.ID.String() || item.OwnerUserID != userID {
			continue
		}
		if q.Kind != "" && item.Kind != string(q.Kind) {
			continue
		}
		if strings.TrimSpace(q.Query) != "" && !strings.Contains(strings.ToLower(item.Name+" "+item.Summary), strings.ToLower(strings.TrimSpace(q.Query))) {
			continue
		}
		if !q.IncludeInactive && (!item.Enabled || item.Status == string(EvaluationResourceStatusArchived)) {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *fakeResourceStore) GetResource(_ context.Context, userID uuid.UUID, idOrHandle string) (*model.MaclawResourceRecord, error) {
	item, ok := s.items[strings.TrimSpace(idOrHandle)]
	if !ok || item.OwnerUserID != userID {
		return nil, nil
	}
	return &item, nil
}
