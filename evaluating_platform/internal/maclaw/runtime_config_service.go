package maclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	appcrypto "evaluating_platform/internal/crypto"
	"evaluating_platform/internal/model"
)

type ModelDefaultStore interface {
	GetDefault(context.Context) (*model.MaclawModelDefault, error)
	UpsertDefault(context.Context, []byte, string, *uuid.UUID) error
}

type RuntimeConfigService struct {
	store    ModelDefaultStore
	keyStore *appcrypto.KeyStore
}

func NewRuntimeConfigService(store ModelDefaultStore, keyStore *appcrypto.KeyStore) *RuntimeConfigService {
	return &RuntimeConfigService{store: store, keyStore: keyStore}
}

func (s *RuntimeConfigService) Enabled() bool {
	return s != nil && s.store != nil && s.keyStore != nil
}

func (s *RuntimeConfigService) GetDefaultRuntimeConfig(ctx context.Context) (*RuntimeAppConfig, error) {
	if !s.Enabled() {
		return nil, nil
	}
	record, err := s.store.GetDefault(ctx)
	if err != nil {
		return nil, err
	}
	if record == nil || len(record.EncryptedConfig) == 0 {
		return nil, nil
	}
	return s.decryptConfig(record)
}

func (s *RuntimeConfigService) GetMaskedDefaultRuntimeConfig(ctx context.Context) (*RuntimeUserConfig, error) {
	cfg, err := s.GetDefaultRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &RuntimeUserConfig{AppConfig: RuntimeAppConfig{}}, nil
	}
	masked := MaskRuntimeAppConfig(*cfg)
	return &RuntimeUserConfig{AppConfig: masked}, nil
}

func (s *RuntimeConfigService) SaveDefaultRuntimeConfig(ctx context.Context, next RuntimeAppConfig, updatedBy *uuid.UUID) (*RuntimeUserConfig, error) {
	masked, _, err := s.SaveDefaultRuntimeConfigForSync(ctx, next, updatedBy)
	return masked, err
}

func (s *RuntimeConfigService) SaveDefaultRuntimeConfigForSync(ctx context.Context, next RuntimeAppConfig, updatedBy *uuid.UUID) (*RuntimeUserConfig, *RuntimeAppConfig, error) {
	if !s.Enabled() {
		return nil, nil, ErrNotConfigured
	}
	current, err := s.GetDefaultRuntimeConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	if current != nil {
		next = MergeRuntimeConfigSecrets(*current, next)
	}
	keyID, key := s.keyStore.CurrentKey()
	data, err := json.Marshal(next)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal maclaw default model config: %w", err)
	}
	encrypted, err := appcrypto.Encrypt(data, key)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt maclaw default model config: %w", err)
	}
	if err := s.store.UpsertDefault(ctx, encrypted, keyID, updatedBy); err != nil {
		return nil, nil, err
	}
	return &RuntimeUserConfig{AppConfig: MaskRuntimeAppConfig(next), UpdatedAt: time.Now()}, &next, nil
}

func (s *RuntimeConfigService) decryptConfig(record *model.MaclawModelDefault) (*RuntimeAppConfig, error) {
	key, err := s.keyStore.GetKey(record.ConfigKeyID)
	if err != nil {
		return nil, fmt.Errorf("load maclaw default model config key: %w", err)
	}
	plain, err := appcrypto.Decrypt(record.EncryptedConfig, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt maclaw default model config: %w", err)
	}
	var cfg RuntimeAppConfig
	if err := json.Unmarshal(plain, &cfg); err != nil {
		return nil, fmt.Errorf("decode maclaw default model config: %w", err)
	}
	return &cfg, nil
}

func MaskRuntimeAppConfig(cfg RuntimeAppConfig) RuntimeAppConfig {
	if cfg.MaclawLLMProviders != nil {
		cfg.MaclawLLMProviders = append([]RuntimeLLMProvider(nil), cfg.MaclawLLMProviders...)
	}
	if strings.TrimSpace(cfg.MaclawLLMKey) != "" {
		cfg.MaclawLLMKey = "******"
	}
	for i := range cfg.MaclawLLMProviders {
		if strings.TrimSpace(cfg.MaclawLLMProviders[i].Key) != "" {
			cfg.MaclawLLMProviders[i].Key = "******"
		}
		if strings.TrimSpace(cfg.MaclawLLMProviders[i].OAuthAccessToken) != "" {
			cfg.MaclawLLMProviders[i].OAuthAccessToken = "******"
		}
		if strings.TrimSpace(cfg.MaclawLLMProviders[i].RefreshToken) != "" {
			cfg.MaclawLLMProviders[i].RefreshToken = "******"
		}
	}
	return cfg
}

func MergeRuntimeConfigSecrets(current, next RuntimeAppConfig) RuntimeAppConfig {
	if isMaskedOrEmpty(next.MaclawLLMKey) {
		next.MaclawLLMKey = current.MaclawLLMKey
	}
	currentProviders := make(map[string]RuntimeLLMProvider, len(current.MaclawLLMProviders))
	for _, provider := range current.MaclawLLMProviders {
		currentProviders[provider.Name] = provider
	}
	for i := range next.MaclawLLMProviders {
		currentProvider, ok := currentProviders[next.MaclawLLMProviders[i].Name]
		if !ok {
			continue
		}
		if isMaskedOrEmpty(next.MaclawLLMProviders[i].Key) {
			next.MaclawLLMProviders[i].Key = currentProvider.Key
		}
		if isMaskedOrEmpty(next.MaclawLLMProviders[i].OAuthAccessToken) {
			next.MaclawLLMProviders[i].OAuthAccessToken = currentProvider.OAuthAccessToken
		}
		if isMaskedOrEmpty(next.MaclawLLMProviders[i].RefreshToken) {
			next.MaclawLLMProviders[i].RefreshToken = currentProvider.RefreshToken
		}
	}
	return next
}

func isMaskedOrEmpty(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || value == "******"
}
