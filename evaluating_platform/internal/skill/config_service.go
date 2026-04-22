package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	cryptopkg "evaluating_platform/internal/crypto"
	"evaluating_platform/internal/model"
)

const missingCryptoMasterKeyMessage = "crypto.master_key is required for skill config storage"

type SkillConfigFieldView struct {
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Required    bool           `json:"required"`
	Secret      bool           `json:"secret"`
	Description string         `json:"description"`
	Placeholder string         `json:"placeholder,omitempty"`
	Default     interface{}    `json:"default,omitempty"`
	EnvName     string         `json:"env_name"`
	Options     []ConfigOption `json:"options,omitempty"`
	Configured  bool           `json:"configured"`
	Value       interface{}    `json:"value,omitempty"`
}

type SkillConfigView struct {
	SkillID             uuid.UUID              `json:"skill_id"`
	SelectedVersion     *model.SkillVersion    `json:"-"`
	ConfigComplete      bool                   `json:"config_complete"`
	MissingRequiredKeys []string               `json:"missing_required_keys"`
	Fields              []SkillConfigFieldView `json:"fields"`
}

type SkillConfigUpdateRequest struct {
	VersionID uuid.UUID
	Values    []SkillConfigUpdateValue
}

type SkillConfigUpdateValue struct {
	Key   string
	Value interface{}
	Clear bool
}

type resolvedSkillConfigState struct {
	Schema              ConfigSchemaEnvelope
	StoredValues        map[string]interface{}
	MissingRequiredKeys []string
	ConfigComplete      bool
}

func (s *Service) GetSkillConfig(ctx context.Context, skillID, expertID, versionID uuid.UUID) (*SkillConfigView, error) {
	item, version, err := s.resolveExpertSkillVersion(ctx, skillID, expertID, versionID)
	if err != nil {
		return nil, err
	}
	if item == nil || version == nil {
		return nil, fmt.Errorf("skill version not found")
	}
	return s.buildSkillConfigView(ctx, item.ID, version)
}

func (s *Service) UpdateSkillConfig(ctx context.Context, skillID, expertID uuid.UUID, req SkillConfigUpdateRequest) (*SkillConfigView, error) {
	item, version, err := s.resolveExpertSkillVersion(ctx, skillID, expertID, req.VersionID)
	if err != nil {
		return nil, err
	}
	if item == nil || version == nil {
		return nil, fmt.Errorf("skill version not found")
	}

	schema, err := s.configSchemaFromVersion(version)
	if err != nil {
		return nil, err
	}
	if len(schema.Fields) == 0 {
		if len(req.Values) > 0 {
			return nil, fmt.Errorf("selected skill version does not declare config_schema")
		}
		return s.buildSkillConfigView(ctx, item.ID, version)
	}
	if s.keyStore == nil {
		return nil, fmt.Errorf(missingCryptoMasterKeyMessage)
	}

	fieldsByKey := make(map[string]ConfigField, len(schema.Fields))
	for _, field := range schema.Fields {
		fieldsByKey[field.Key] = field
	}

	seen := make(map[string]struct{}, len(req.Values))
	for _, update := range req.Values {
		key := strings.TrimSpace(update.Key)
		if key == "" {
			return nil, fmt.Errorf("config key is required")
		}
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate config key: %s", key)
		}
		seen[key] = struct{}{}

		field, ok := fieldsByKey[key]
		if !ok {
			return nil, fmt.Errorf("unknown config key: %s", key)
		}
		if update.Clear {
			if err := s.configRepo.Delete(ctx, item.ID, key); err != nil {
				return nil, err
			}
			continue
		}

		coercedValue, err := coerceConfigValue(field, update.Value)
		if err != nil {
			return nil, fmt.Errorf("config %s: %w", key, err)
		}
		ciphertext, keyID, err := s.encryptConfigValue(coercedValue)
		if err != nil {
			return nil, err
		}
		record := &model.SkillConfigValue{
			SkillID:         item.ID,
			FieldKey:        key,
			ValueCiphertext: ciphertext,
			KeyID:           keyID,
		}
		if err := s.configRepo.Upsert(ctx, record); err != nil {
			return nil, err
		}
	}

	return s.buildSkillConfigView(ctx, item.ID, version)
}

func (s *Service) buildSkillConfigView(ctx context.Context, skillID uuid.UUID, version *model.SkillVersion) (*SkillConfigView, error) {
	state, err := s.resolveSkillConfigState(ctx, skillID, version)
	if err != nil {
		return nil, err
	}

	fields := make([]SkillConfigFieldView, 0, len(state.Schema.Fields))
	for _, field := range state.Schema.Fields {
		storedValue, configured := state.StoredValues[field.Key]
		viewField := SkillConfigFieldView{
			Key:         field.Key,
			Label:       field.Label,
			Type:        field.Type,
			Required:    field.Required,
			Secret:      effectiveFieldSecret(field),
			Description: field.Description,
			Placeholder: field.Placeholder,
			Default:     field.Default,
			EnvName:     field.EnvName,
			Options:     append([]ConfigOption(nil), field.Options...),
			Configured:  configured,
		}
		if !effectiveFieldSecret(field) {
			if configured {
				viewField.Value = storedValue
			} else if field.Default != nil {
				viewField.Value = field.Default
			}
		}
		fields = append(fields, viewField)
	}

	return &SkillConfigView{
		SkillID:             skillID,
		SelectedVersion:     version,
		ConfigComplete:      state.ConfigComplete,
		MissingRequiredKeys: append([]string(nil), state.MissingRequiredKeys...),
		Fields:              fields,
	}, nil
}

func (s *Service) resolveSkillConfigState(ctx context.Context, skillID uuid.UUID, version *model.SkillVersion) (*resolvedSkillConfigState, error) {
	schema, err := s.configSchemaFromVersion(version)
	if err != nil {
		return nil, err
	}
	storedValues, err := s.loadStoredConfigValues(ctx, skillID)
	if err != nil {
		return nil, err
	}

	missing := make([]string, 0)
	for _, field := range schema.Fields {
		value, configured := storedValues[field.Key]
		if field.Required && !configFieldSatisfied(field, value, configured) {
			missing = append(missing, field.Key)
		}
	}
	return &resolvedSkillConfigState{
		Schema:              schema,
		StoredValues:        storedValues,
		MissingRequiredKeys: missing,
		ConfigComplete:      len(missing) == 0,
	}, nil
}

func (s *Service) configSchemaFromVersion(version *model.SkillVersion) (ConfigSchemaEnvelope, error) {
	if version == nil {
		return ConfigSchemaEnvelope{}, fmt.Errorf("skill version is required")
	}
	metadata := map[string]interface{}{}
	if len(version.Metadata) > 0 {
		if err := json.Unmarshal(version.Metadata, &metadata); err != nil {
			return ConfigSchemaEnvelope{}, fmt.Errorf("parse skill metadata: %w", err)
		}
	}
	return configSchemaEnvelopeFromMetadata(metadata)
}

func (s *Service) loadStoredConfigValues(ctx context.Context, skillID uuid.UUID) (map[string]interface{}, error) {
	if s.configRepo == nil {
		return map[string]interface{}{}, nil
	}
	items, err := s.configRepo.ListBySkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return map[string]interface{}{}, nil
	}
	if s.keyStore == nil {
		return nil, fmt.Errorf(missingCryptoMasterKeyMessage)
	}

	values := make(map[string]interface{}, len(items))
	for _, item := range items {
		value, err := s.decryptConfigValue(item)
		if err != nil {
			return nil, err
		}
		values[item.FieldKey] = value
	}
	return values, nil
}

func (s *Service) decryptConfigValue(item model.SkillConfigValue) (interface{}, error) {
	if s.keyStore == nil {
		return nil, fmt.Errorf(missingCryptoMasterKeyMessage)
	}
	key, err := s.keyStore.GetKey(item.KeyID)
	if err != nil {
		return nil, fmt.Errorf("load skill config key %s: %w", item.KeyID, err)
	}
	plaintext, err := cryptopkg.Decrypt(item.ValueCiphertext, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt skill config %s: %w", item.FieldKey, err)
	}

	var value interface{}
	if err := json.Unmarshal(plaintext, &value); err != nil {
		return nil, fmt.Errorf("parse skill config %s: %w", item.FieldKey, err)
	}
	return value, nil
}

func (s *Service) encryptConfigValue(value interface{}) ([]byte, string, error) {
	if s.keyStore == nil {
		return nil, "", fmt.Errorf(missingCryptoMasterKeyMessage)
	}
	plaintext, err := json.Marshal(value)
	if err != nil {
		return nil, "", fmt.Errorf("marshal skill config value: %w", err)
	}
	keyID, key := s.keyStore.CurrentKey()
	if keyID == "" || len(key) == 0 {
		return nil, "", fmt.Errorf(missingCryptoMasterKeyMessage)
	}
	ciphertext, err := cryptopkg.Encrypt(plaintext, key)
	if err != nil {
		return nil, "", fmt.Errorf("encrypt skill config value: %w", err)
	}
	return ciphertext, keyID, nil
}

func (s *Service) buildRuntimeEnv(ctx context.Context, skillID uuid.UUID, version *model.SkillVersion) (map[string]string, error) {
	state, err := s.resolveSkillConfigState(ctx, skillID, version)
	if err != nil {
		return nil, err
	}
	if len(state.Schema.Fields) == 0 {
		return nil, nil
	}

	env := make(map[string]string)
	for _, field := range state.Schema.Fields {
		value, ok := state.StoredValues[field.Key]
		if !ok && field.Default != nil {
			value = field.Default
			ok = true
		}
		if !ok && state.Schema.CompatMode {
			for _, fallbackEnv := range field.FallbackEnv {
				raw := strings.TrimSpace(os.Getenv(fallbackEnv))
				if raw == "" {
					continue
				}
				coerced, err := coerceConfigValue(field, raw)
				if err != nil {
					continue
				}
				value = coerced
				ok = true
				break
			}
		}
		if !ok || !configValuePresent(field, value) {
			continue
		}
		stringified, err := stringifyConfigValue(field, value)
		if err != nil {
			return nil, err
		}
		env[field.EnvName] = stringified
	}
	return env, nil
}

func (s *Service) validateSkillConfigForPublish(ctx context.Context, skillID uuid.UUID, version *model.SkillVersion) error {
	schema, err := s.configSchemaFromVersion(version)
	if err != nil {
		return err
	}
	if len(schema.Fields) == 0 {
		return nil
	}
	if s.keyStore == nil {
		return fmt.Errorf(missingCryptoMasterKeyMessage)
	}
	state, err := s.resolveSkillConfigState(ctx, skillID, version)
	if err != nil {
		return err
	}
	if state.ConfigComplete {
		return nil
	}
	return fmt.Errorf("missing required skill config keys: %s", strings.Join(state.MissingRequiredKeys, ", "))
}

func (s *Service) publishedSkillConfigReady(ctx context.Context, skillID uuid.UUID, version *model.SkillVersion) bool {
	schema, err := s.configSchemaFromVersion(version)
	if err != nil {
		return false
	}
	if len(schema.Fields) == 0 {
		return true
	}
	if s.keyStore == nil {
		return false
	}
	state, err := s.resolveSkillConfigState(ctx, skillID, version)
	if err != nil {
		return false
	}
	return state.ConfigComplete
}

func (s *Service) resolveExpertSkillVersion(ctx context.Context, skillID, expertID, versionID uuid.UUID) (*model.Skill, *model.SkillVersion, error) {
	item, err := s.skillRepo.GetByIDForExpert(ctx, skillID, expertID)
	if err != nil || item == nil {
		return nil, nil, fmt.Errorf("skill not found")
	}
	version, err := s.resolveVersionForSkill(ctx, item, versionID)
	if err != nil {
		return nil, nil, err
	}
	return item, version, nil
}

func (s *Service) resolveVersionForSkill(ctx context.Context, item *model.Skill, versionID uuid.UUID) (*model.SkillVersion, error) {
	if item == nil {
		return nil, fmt.Errorf("skill not found")
	}
	if versionID != uuid.Nil {
		version, err := s.versionRepo.GetByID(ctx, versionID)
		if err != nil || version == nil || version.SkillID != item.ID {
			return nil, fmt.Errorf("skill version not found")
		}
		return version, nil
	}
	if item.LatestVersionID != nil {
		if version, err := s.versionRepo.GetByID(ctx, *item.LatestVersionID); err == nil && version != nil {
			return version, nil
		}
	}
	if item.PublishedVersionID != nil {
		if version, err := s.versionRepo.GetByID(ctx, *item.PublishedVersionID); err == nil && version != nil {
			return version, nil
		}
	}
	version, err := s.versionRepo.GetLatestBySkillID(ctx, item.ID)
	if err != nil || version == nil {
		return nil, fmt.Errorf("skill version not found")
	}
	return version, nil
}

func (s *Service) ensureConfigSchemaCompatibility(ctx context.Context, skillID uuid.UUID, manifest Manifest) error {
	if s.versionRepo == nil {
		return nil
	}
	versions, err := s.versionRepo.ListBySkill(ctx, skillID)
	if err != nil {
		return err
	}
	if len(versions) == 0 || len(manifest.ConfigSchema) == 0 {
		return nil
	}

	existingByKey := make(map[string]ConfigField)
	for _, version := range versions {
		schema, err := s.configSchemaFromVersion(&version)
		if err != nil {
			return err
		}
		for _, field := range schema.Fields {
			if _, ok := existingByKey[field.Key]; !ok {
				existingByKey[field.Key] = field
			}
		}
	}

	for _, nextField := range manifest.ConfigSchema {
		existingField, ok := existingByKey[nextField.Key]
		if !ok {
			continue
		}
		if !configFieldsCompatible(existingField, nextField) {
			return fmt.Errorf(
				"config_schema key %s is incompatible with previous versions: type/secret changes are not allowed",
				nextField.Key,
			)
		}
	}
	return nil
}
