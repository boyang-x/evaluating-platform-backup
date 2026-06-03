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

func TestMaclawHubConfigServiceEncryptsAndReturnsRuntimePatch(t *testing.T) {
	keyStore, err := appcrypto.NewKeyStore(config.CryptoConfig{
		MasterKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		KeyID:     "test",
	})
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	store := &fakeHubConfigStore{}
	service := NewMaclawHubConfigService(store, keyStore)
	adminID := uuid.New()

	cfg, err := service.SaveHubConfig(context.Background(), MaclawHubConfig{
		Enabled:        true,
		HubURL:         "https://hub.example.test/",
		AllowedSources: []string{"skillhub", "github", "skillmarket", "skillhub"},
	}, &adminID)
	if err != nil {
		t.Fatalf("SaveHubConfig: %v", err)
	}
	if cfg.HubURL != "https://hub.example.test" {
		t.Fatalf("hub url = %q", cfg.HubURL)
	}
	if len(cfg.AllowedSources) != 2 || cfg.AllowedSources[0] != "skillhub" || cfg.AllowedSources[1] != "github" {
		t.Fatalf("allowed sources = %#v", cfg.AllowedSources)
	}
	if bytes.Contains(store.record.EncryptedConfig, []byte("hub.example.test")) {
		t.Fatalf("hub config was stored in plaintext")
	}

	patch, err := service.RuntimeConfigPatch(context.Background())
	if err != nil {
		t.Fatalf("RuntimeConfigPatch: %v", err)
	}
	if patch.RemoteHubURL != "https://hub.example.test" {
		t.Fatalf("patch hub url = %q", patch.RemoteHubURL)
	}
	if len(patch.SkillSourcesAllowed) != 2 || patch.SkillSourcesAllowed[0] != "skillhub" || patch.SkillSourcesAllowed[1] != "github" {
		t.Fatalf("patch skill sources = %#v", patch.SkillSourcesAllowed)
	}
}

func TestMaclawHubConfigServiceDefaultsToSkillHubOnly(t *testing.T) {
	cfg, err := NormalizeHubConfig(MaclawHubConfig{Enabled: false})
	if err != nil {
		t.Fatalf("NormalizeHubConfig: %v", err)
	}
	if len(cfg.AllowedSources) != 1 || cfg.AllowedSources[0] != "skillhub" {
		t.Fatalf("allowed sources = %#v", cfg.AllowedSources)
	}
}

type fakeHubConfigStore struct {
	record *model.MaclawHubConfigRecord
}

func (s *fakeHubConfigStore) GetHubConfig(context.Context) (*model.MaclawHubConfigRecord, error) {
	return s.record, nil
}

func (s *fakeHubConfigStore) UpsertHubConfig(_ context.Context, encrypted []byte, keyID string, updatedBy *uuid.UUID) error {
	s.record = &model.MaclawHubConfigRecord{
		ID:              "default",
		EncryptedConfig: encrypted,
		ConfigKeyID:     keyID,
		UpdatedBy:       updatedBy,
	}
	return nil
}
