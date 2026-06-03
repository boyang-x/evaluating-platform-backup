package maclaw

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"

	appcrypto "evaluating_platform/internal/crypto"
	"evaluating_platform/internal/model"
	"evaluating_platform/pkg/config"
)

func TestRuntimeConfigServiceEncryptsDefaultConfigAndReturnsMaskedSecrets(t *testing.T) {
	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{
		MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		KeyID:     "test",
	})
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	store := &fakeModelDefaultStore{}
	service := NewRuntimeConfigService(store, keyStore)
	adminID := uuid.New()

	out, err := service.SaveDefaultRuntimeConfig(context.Background(), RuntimeAppConfig{
		MaclawLLMProviders: []RuntimeLLMProvider{{
			Name:  "openai-prod",
			URL:   "https://api.example/v1",
			Key:   "sk-secret",
			Model: "gpt-test",
		}},
		MaclawLLMCurrentProvider: "openai-prod",
	}, &adminID)
	if err != nil {
		t.Fatalf("SaveDefaultRuntimeConfig: %v", err)
	}
	if out.AppConfig.MaclawLLMProviders[0].Key != "******" {
		t.Fatalf("returned key = %q, want masked", out.AppConfig.MaclawLLMProviders[0].Key)
	}
	if bytes.Contains(store.record.EncryptedConfig, []byte("sk-secret")) {
		t.Fatalf("secret was stored in plaintext")
	}

	raw, err := service.GetDefaultRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultRuntimeConfig: %v", err)
	}
	if raw.MaclawLLMProviders[0].Key != "sk-secret" {
		t.Fatalf("raw key = %q, want secret for server-side provisioning", raw.MaclawLLMProviders[0].Key)
	}
	masked, err := service.GetMaskedDefaultRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("GetMaskedDefaultRuntimeConfig: %v", err)
	}
	if masked.AppConfig.MaclawLLMProviders[0].Key != "******" {
		t.Fatalf("masked key = %q", masked.AppConfig.MaclawLLMProviders[0].Key)
	}
}

func TestRuntimeConfigServiceReturnsMergedPlaintextConfigForTenantSync(t *testing.T) {
	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{
		MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		KeyID:     "test",
	})
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	service := NewRuntimeConfigService(&fakeModelDefaultStore{}, keyStore)

	_, _, err = service.SaveDefaultRuntimeConfigForSync(context.Background(), RuntimeAppConfig{
		MaclawLLMProviders: []RuntimeLLMProvider{{
			Name:  "openai-prod",
			URL:   "https://api.example/v1",
			Key:   "sk-original",
			Model: "gpt-old",
		}},
		MaclawLLMCurrentProvider: "openai-prod",
	}, nil)
	if err != nil {
		t.Fatalf("initial SaveDefaultRuntimeConfigForSync: %v", err)
	}

	masked, syncConfig, err := service.SaveDefaultRuntimeConfigForSync(context.Background(), RuntimeAppConfig{
		MaclawLLMProviders: []RuntimeLLMProvider{{
			Name:  "openai-prod",
			URL:   "https://api.example/v1",
			Key:   "******",
			Model: "gpt-new",
		}},
		MaclawLLMCurrentProvider: "openai-prod",
	}, nil)
	if err != nil {
		t.Fatalf("masked SaveDefaultRuntimeConfigForSync: %v", err)
	}
	if masked.AppConfig.MaclawLLMProviders[0].Key != "******" {
		t.Fatalf("masked key = %q", masked.AppConfig.MaclawLLMProviders[0].Key)
	}
	if syncConfig.MaclawLLMProviders[0].Key != "sk-original" {
		t.Fatalf("sync key = %q, want preserved plaintext secret", syncConfig.MaclawLLMProviders[0].Key)
	}
	if syncConfig.MaclawLLMProviders[0].Model != "gpt-new" {
		t.Fatalf("sync model = %q, want updated model", syncConfig.MaclawLLMProviders[0].Model)
	}
}

type fakeModelDefaultStore struct {
	record *model.MaclawModelDefault
}

func (s *fakeModelDefaultStore) GetDefault(context.Context) (*model.MaclawModelDefault, error) {
	return s.record, nil
}

func (s *fakeModelDefaultStore) UpsertDefault(_ context.Context, encrypted []byte, keyID string, updatedBy *uuid.UUID) error {
	s.record = &model.MaclawModelDefault{
		ID:              "default",
		EncryptedConfig: encrypted,
		ConfigKeyID:     keyID,
		UpdatedBy:       updatedBy,
	}
	return nil
}
