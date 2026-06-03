package maclaw

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appcrypto "evaluating_platform/internal/crypto"
	"evaluating_platform/pkg/config"

	"github.com/google/uuid"
)

type memoryTargetConfigStore struct {
	record *TargetConfigRecord
}

func (s *memoryTargetConfigStore) GetByUserID(_ context.Context, userID uuid.UUID) (*TargetConfigRecord, error) {
	if s.record == nil || s.record.UserID != userID {
		return nil, nil
	}
	copy := *s.record
	return &copy, nil
}

func (s *memoryTargetConfigStore) Upsert(_ context.Context, record TargetConfigRecord) (*TargetConfigRecord, error) {
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	s.record = &record
	copy := record
	return &copy, nil
}

func TestPlatformTargetGatewayStoresTargetEncryptedAndReturnsSafeSummary(t *testing.T) {
	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	store := &memoryTargetConfigStore{}
	service := NewTargetConfigService(store, keyStore)
	userID := uuid.New()
	gateway := NewPlatformTargetGateway(nil, service, userID)

	summary, err := gateway.SaveEvaluationTarget(context.Background(), EvaluationTargetInput{
		Name:             "Default target",
		Kind:             EvaluationTargetKindLLM,
		Provider:         "openai",
		BaseURL:          "http://example.test/v1",
		Model:            "gpt-test",
		AuthType:         EvaluationTargetAuthTypeBearer,
		CredentialSecret: "sk-secret",
		Enabled:          true,
	})
	if err != nil {
		t.Fatalf("SaveEvaluationTarget: %v", err)
	}
	if summary.ID == "" || !summary.CredentialSecretSet || summary.Provider != "openai" || summary.Model != "gpt-test" {
		t.Fatalf("summary = %#v", summary)
	}
	if store.record == nil || bytes.Contains(store.record.EncryptedConfig, []byte("sk-secret")) {
		t.Fatalf("target config was not encrypted: %#v", store.record)
	}

	items, err := gateway.SearchEvaluationTargets(context.Background(), EvaluationTargetQuery{Kind: EvaluationTargetKindLLM})
	if err != nil {
		t.Fatalf("SearchEvaluationTargets: %v", err)
	}
	if len(items) != 1 || items[0].ID != summary.ID || !items[0].CredentialSecretSet {
		t.Fatalf("items = %#v", items)
	}

	got, err := gateway.GetEvaluationTarget(context.Background(), summary.ID)
	if err != nil {
		t.Fatalf("GetEvaluationTarget: %v", err)
	}
	if got.ID != summary.ID || got.CredentialSecretSet != true {
		t.Fatalf("got = %#v", got)
	}
}

func TestPlatformTargetGatewayPreservesExistingSecretWhenMasked(t *testing.T) {
	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	store := &memoryTargetConfigStore{}
	service := NewTargetConfigService(store, keyStore)
	userID := uuid.New()
	gateway := NewPlatformTargetGateway(nil, service, userID)

	first, err := gateway.SaveEvaluationTarget(context.Background(), EvaluationTargetInput{
		Name: "Target", Kind: EvaluationTargetKindLLM, BaseURL: "http://example.test/v1", Model: "gpt-test", AuthType: EvaluationTargetAuthTypeBearer, CredentialSecret: "sk-original", Enabled: true,
	})
	if err != nil {
		t.Fatalf("Save first target: %v", err)
	}
	second, err := gateway.SaveEvaluationTarget(context.Background(), EvaluationTargetInput{
		ID: first.ID, Name: "Target v2", Kind: EvaluationTargetKindLLM, BaseURL: "http://example.test/v1", Model: "gpt-test", AuthType: EvaluationTargetAuthTypeBearer, CredentialSecret: "******", Enabled: true,
	})
	if err != nil {
		t.Fatalf("Save masked target: %v", err)
	}
	if !second.CredentialSecretSet {
		t.Fatalf("secret should remain set: %#v", second)
	}
	plain, err := service.GetTargetWithSecret(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetTargetWithSecret: %v", err)
	}
	if plain.CredentialSecret != "sk-original" {
		t.Fatalf("secret = %q", plain.CredentialSecret)
	}
}

func TestPlatformTargetGatewayStripsCredentialURLParts(t *testing.T) {
	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	store := &memoryTargetConfigStore{}
	service := NewTargetConfigService(store, keyStore)
	userID := uuid.New()
	gateway := NewPlatformTargetGateway(nil, service, userID)

	summary, err := gateway.SaveEvaluationTarget(context.Background(), EvaluationTargetInput{
		Name:             "Target",
		Kind:             EvaluationTargetKindLLM,
		BaseURL:          "https://url-secret@example.test/v1?api_key=query-secret&token=token-secret&ok=1#access_token=fragment-secret",
		Model:            "gpt-test",
		AuthType:         EvaluationTargetAuthTypeBearer,
		CredentialSecret: "sk-field-secret",
		Enabled:          true,
		Metadata: map[string]string{
			"health_url": "https://health-secret@example.test/v1/models?credential_secret=health-secret&ok=1#token=health-fragment-secret",
		},
	})
	if err != nil {
		t.Fatalf("SaveEvaluationTarget: %v", err)
	}
	got, err := gateway.GetEvaluationTarget(context.Background(), summary.ID)
	if err != nil {
		t.Fatalf("GetEvaluationTarget: %v", err)
	}
	plain, err := service.GetTargetWithSecret(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetTargetWithSecret: %v", err)
	}
	serialized := summary.BaseURL + " " + got.BaseURL + " " + got.Metadata["health_url"] + " " + plain.BaseURL + " " + plain.Metadata["health_url"]
	for _, forbidden := range []string{"url-secret", "query-secret", "token-secret", "health-secret", "fragment-secret", "health-fragment-secret", "api_key=", "token=", "credential_secret=", "#"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("target URL leaked credential marker %q in %q", forbidden, serialized)
		}
	}
	if got.BaseURL != "https://example.test/v1?ok=1" || got.Metadata["health_url"] != "https://example.test/v1/models?ok=1" {
		t.Fatalf("sanitized urls = base %q health %q", got.BaseURL, got.Metadata["health_url"])
	}
}

func TestPlatformTargetGatewayProbesTargetWithSecretWithoutLeakingIt(t *testing.T) {
	var auth string
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	store := &memoryTargetConfigStore{}
	service := NewTargetConfigService(store, keyStore)
	userID := uuid.New()
	gateway := NewPlatformTargetGateway(nil, service, userID)
	summary, err := gateway.SaveEvaluationTarget(context.Background(), EvaluationTargetInput{
		Name: "Target", Kind: EvaluationTargetKindLLM, BaseURL: targetServer.URL, Model: "gpt-test", AuthType: EvaluationTargetAuthTypeBearer, CredentialSecret: "sk-probe", Enabled: true,
		Metadata: map[string]string{"health_url": targetServer.URL + "/models"},
	})
	if err != nil {
		t.Fatalf("SaveEvaluationTarget: %v", err)
	}

	probe, err := gateway.ProbeEvaluationTarget(context.Background(), summary.ID)
	if err != nil {
		t.Fatalf("ProbeEvaluationTarget: %v", err)
	}
	if probe.Status != EvaluationTargetHealthHealthy || !probe.Target.CredentialSecretSet {
		t.Fatalf("probe = %#v", probe)
	}
	if auth != "Bearer sk-probe" {
		t.Fatalf("auth = %q", auth)
	}
	if probe.Target.Metadata["health_url"] != targetServer.URL+"/models" {
		t.Fatalf("metadata = %#v", probe.Target.Metadata)
	}
}

func TestPlatformTargetGatewayFallsBackToRuntimeWhenServiceMissing(t *testing.T) {
	upstream := &targetFallbackGateway{}
	gateway := NewPlatformTargetGateway(upstream, nil, uuid.New())
	if _, err := gateway.SearchEvaluationTargets(context.Background(), EvaluationTargetQuery{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("error = %v, want fallback ErrUnsupported", err)
	}
}

type targetFallbackGateway struct {
	GatewayClient
}

func (g *targetFallbackGateway) Enabled() bool { return true }

func (g *targetFallbackGateway) SearchEvaluationTargets(context.Context, EvaluationTargetQuery) ([]EvaluationTargetSummary, error) {
	return nil, ErrUnsupported
}
