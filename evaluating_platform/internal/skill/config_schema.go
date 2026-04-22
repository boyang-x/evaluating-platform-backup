package skill

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	ConfigFieldTypeText     = "text"
	ConfigFieldTypeTextarea = "textarea"
	ConfigFieldTypePassword = "password"
	ConfigFieldTypeNumber   = "number"
	ConfigFieldTypeBoolean  = "boolean"
	ConfigFieldTypeSelect   = "select"
)

type ConfigOption struct {
	Label string `yaml:"label" json:"label"`
	Value string `yaml:"value" json:"value"`
}

type ConfigField struct {
	Key         string         `yaml:"key" json:"key"`
	Label       string         `yaml:"label" json:"label"`
	Type        string         `yaml:"type" json:"type"`
	Required    bool           `yaml:"required" json:"required"`
	Description string         `yaml:"description" json:"description"`
	EnvName     string         `yaml:"env_name" json:"env_name"`
	Secret      bool           `yaml:"secret,omitempty" json:"secret,omitempty"`
	Placeholder string         `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	Default     interface{}    `yaml:"default,omitempty" json:"default,omitempty"`
	Options     []ConfigOption `yaml:"options,omitempty" json:"options,omitempty"`
	FallbackEnv []string       `yaml:"-" json:"fallback_env,omitempty"`
}

type ConfigSchemaEnvelope struct {
	Fields     []ConfigField `json:"fields"`
	CompatMode bool          `json:"compat_mode,omitempty"`
	Source     string        `json:"source,omitempty"`
}

type legacyRuntimeConfig struct {
	Env []legacyRuntimeConfigField `json:"env"`
}

type legacyRuntimeConfigField struct {
	Name        string   `json:"name"`
	Required    bool     `json:"required"`
	Secret      bool     `json:"secret"`
	Description string   `json:"description"`
	FallbackEnv []string `json:"fallback_env"`
}

func normalizeManifestConfigSchema(m *Manifest) error {
	fields := cloneConfigFields(m.ConfigSchema)
	compatMode := false
	source := "config_schema"

	if len(fields) == 0 {
		legacyFields, err := legacyConfigFieldsFromMetadata(m.Metadata)
		if err != nil {
			return err
		}
		if len(legacyFields) > 0 {
			fields = legacyFields
			compatMode = true
			source = "metadata.runtime_config"
		}
	}

	normalized, err := normalizeConfigFields(fields)
	if err != nil {
		return err
	}
	m.ConfigSchema = normalized
	m.ConfigSchemaCompatMode = compatMode
	m.ConfigSchemaSource = source
	return nil
}

func normalizeConfigFields(fields []ConfigField) ([]ConfigField, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	normalized := make([]ConfigField, 0, len(fields))
	seenKeys := make(map[string]struct{}, len(fields))
	seenEnvNames := make(map[string]struct{}, len(fields))
	for _, rawField := range fields {
		field, err := normalizeConfigField(rawField)
		if err != nil {
			return nil, err
		}
		if _, exists := seenKeys[field.Key]; exists {
			return nil, fmt.Errorf("duplicate config_schema key: %s", field.Key)
		}
		if _, exists := seenEnvNames[field.EnvName]; exists {
			return nil, fmt.Errorf("duplicate config_schema env_name: %s", field.EnvName)
		}
		seenKeys[field.Key] = struct{}{}
		seenEnvNames[field.EnvName] = struct{}{}
		normalized = append(normalized, field)
	}
	return normalized, nil
}

func normalizeConfigField(field ConfigField) (ConfigField, error) {
	field.Key = strings.TrimSpace(field.Key)
	field.Label = strings.TrimSpace(field.Label)
	field.Type = strings.TrimSpace(strings.ToLower(field.Type))
	field.Description = strings.TrimSpace(field.Description)
	field.EnvName = strings.TrimSpace(field.EnvName)
	field.Placeholder = strings.TrimSpace(field.Placeholder)
	field.FallbackEnv = uniqueNonEmptyStrings(field.FallbackEnv)

	if field.Key == "" {
		return ConfigField{}, fmt.Errorf("config_schema key is required")
	}
	if field.Label == "" {
		field.Label = humanizeConfigKey(field.Key)
	}
	if field.EnvName == "" {
		return ConfigField{}, fmt.Errorf("config_schema env_name is required for key %s", field.Key)
	}
	switch field.Type {
	case ConfigFieldTypeText, ConfigFieldTypeTextarea, ConfigFieldTypePassword, ConfigFieldTypeNumber, ConfigFieldTypeBoolean, ConfigFieldTypeSelect:
	default:
		return ConfigField{}, fmt.Errorf("unsupported config_schema type for key %s: %s", field.Key, field.Type)
	}
	if field.Type == ConfigFieldTypePassword {
		field.Secret = true
	}
	if field.Type == ConfigFieldTypeSelect {
		if len(field.Options) == 0 {
			return ConfigField{}, fmt.Errorf("config_schema select field %s requires options", field.Key)
		}
		seen := make(map[string]struct{}, len(field.Options))
		options := make([]ConfigOption, 0, len(field.Options))
		for _, option := range field.Options {
			option.Value = strings.TrimSpace(option.Value)
			option.Label = strings.TrimSpace(option.Label)
			if option.Value == "" {
				return ConfigField{}, fmt.Errorf("config_schema select field %s has empty option value", field.Key)
			}
			if option.Label == "" {
				option.Label = option.Value
			}
			if _, exists := seen[option.Value]; exists {
				return ConfigField{}, fmt.Errorf("config_schema select field %s has duplicate option value: %s", field.Key, option.Value)
			}
			seen[option.Value] = struct{}{}
			options = append(options, option)
		}
		field.Options = options
	} else if len(field.Options) > 0 {
		return ConfigField{}, fmt.Errorf("config_schema field %s only select type can define options", field.Key)
	}

	if field.Secret && field.Default != nil {
		return ConfigField{}, fmt.Errorf("config_schema secret field %s cannot declare default", field.Key)
	}
	if field.Default != nil {
		normalizedDefault, err := coerceConfigValue(field, field.Default)
		if err != nil {
			return ConfigField{}, fmt.Errorf("config_schema default for key %s is invalid: %w", field.Key, err)
		}
		field.Default = normalizedDefault
	}
	return field, nil
}

func legacyConfigFieldsFromMetadata(metadata map[string]interface{}) ([]ConfigField, error) {
	if metadata == nil {
		return nil, nil
	}
	rawRuntimeConfig, ok := metadata["runtime_config"]
	if !ok {
		return nil, nil
	}
	data, err := json.Marshal(rawRuntimeConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal legacy runtime_config: %w", err)
	}
	var runtimeConfig legacyRuntimeConfig
	if err := json.Unmarshal(data, &runtimeConfig); err != nil {
		return nil, fmt.Errorf("parse legacy runtime_config: %w", err)
	}
	fields := make([]ConfigField, 0, len(runtimeConfig.Env))
	for _, item := range runtimeConfig.Env {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return nil, fmt.Errorf("legacy runtime_config env.name is required")
		}
		fields = append(fields, ConfigField{
			Key:         name,
			Label:       humanizeConfigKey(name),
			Type:        ConfigFieldTypeText,
			Required:    item.Required,
			Description: strings.TrimSpace(item.Description),
			EnvName:     name,
			Secret:      item.Secret,
			FallbackEnv: uniqueNonEmptyStrings(item.FallbackEnv),
		})
	}
	return fields, nil
}

func configSchemaEnvelopeFromMetadata(metadata map[string]interface{}) (ConfigSchemaEnvelope, error) {
	if metadata == nil {
		return ConfigSchemaEnvelope{}, nil
	}
	if rawSchema, ok := metadata["config_schema"]; ok {
		data, err := json.Marshal(rawSchema)
		if err != nil {
			return ConfigSchemaEnvelope{}, fmt.Errorf("marshal metadata config_schema: %w", err)
		}

		var envelope ConfigSchemaEnvelope
		if err := json.Unmarshal(data, &envelope); err == nil && len(envelope.Fields) > 0 {
			fields, normErr := normalizeConfigFields(envelope.Fields)
			if normErr != nil {
				return ConfigSchemaEnvelope{}, normErr
			}
			envelope.Fields = fields
			if strings.TrimSpace(envelope.Source) == "" {
				envelope.Source = "config_schema"
			}
			return envelope, nil
		}

		var directFields []ConfigField
		if err := json.Unmarshal(data, &directFields); err == nil {
			fields, normErr := normalizeConfigFields(directFields)
			if normErr != nil {
				return ConfigSchemaEnvelope{}, normErr
			}
			return ConfigSchemaEnvelope{
				Fields:     fields,
				CompatMode: false,
				Source:     "config_schema",
			}, nil
		}
	}

	legacyFields, err := legacyConfigFieldsFromMetadata(metadata)
	if err != nil {
		return ConfigSchemaEnvelope{}, err
	}
	fields, err := normalizeConfigFields(legacyFields)
	if err != nil {
		return ConfigSchemaEnvelope{}, err
	}
	if len(fields) == 0 {
		return ConfigSchemaEnvelope{}, nil
	}
	return ConfigSchemaEnvelope{
		Fields:     fields,
		CompatMode: true,
		Source:     "metadata.runtime_config",
	}, nil
}

func buildConfigSchemaEnvelope(manifest Manifest) ConfigSchemaEnvelope {
	return ConfigSchemaEnvelope{
		Fields:     cloneConfigFields(manifest.ConfigSchema),
		CompatMode: manifest.ConfigSchemaCompatMode,
		Source:     firstNonEmptyString(strings.TrimSpace(manifest.ConfigSchemaSource), "config_schema"),
	}
}

func cloneConfigFields(fields []ConfigField) []ConfigField {
	if len(fields) == 0 {
		return nil
	}
	cloned := make([]ConfigField, 0, len(fields))
	for _, field := range fields {
		copyField := field
		copyField.Options = append([]ConfigOption(nil), field.Options...)
		copyField.FallbackEnv = append([]string(nil), field.FallbackEnv...)
		cloned = append(cloned, copyField)
	}
	return cloned
}

func humanizeConfigKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(strings.ToLower(value), "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	return strings.Join(strings.Fields(value), " ")
}

func configFieldSatisfied(field ConfigField, storedValue interface{}, stored bool) bool {
	if stored {
		return configValuePresent(field, storedValue)
	}
	if field.Default != nil {
		return configValuePresent(field, field.Default)
	}
	return false
}

func configValuePresent(field ConfigField, value interface{}) bool {
	switch field.Type {
	case ConfigFieldTypeBoolean:
		_, ok := value.(bool)
		return ok
	case ConfigFieldTypeNumber:
		switch typed := value.(type) {
		case float64:
			return !math.IsNaN(typed) && !math.IsInf(typed, 0)
		case float32:
			return !math.IsNaN(float64(typed)) && !math.IsInf(float64(typed), 0)
		case int, int32, int64, uint, uint32, uint64:
			return true
		default:
			return false
		}
	default:
		text, ok := value.(string)
		return ok && strings.TrimSpace(text) != ""
	}
}

func coerceConfigValue(field ConfigField, raw interface{}) (interface{}, error) {
	switch field.Type {
	case ConfigFieldTypeText, ConfigFieldTypeTextarea, ConfigFieldTypePassword:
		text, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string")
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("value cannot be empty")
		}
		return text, nil
	case ConfigFieldTypeSelect:
		text, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string")
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("value cannot be empty")
		}
		for _, option := range field.Options {
			if option.Value == text {
				return text, nil
			}
		}
		return nil, fmt.Errorf("value %q is not in select options", text)
	case ConfigFieldTypeNumber:
		switch typed := raw.(type) {
		case float64:
			if math.IsNaN(typed) || math.IsInf(typed, 0) {
				return nil, fmt.Errorf("invalid number")
			}
			return typed, nil
		case float32:
			value := float64(typed)
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("invalid number")
			}
			return value, nil
		case int:
			return float64(typed), nil
		case int32:
			return float64(typed), nil
		case int64:
			return float64(typed), nil
		case uint:
			return float64(typed), nil
		case uint32:
			return float64(typed), nil
		case uint64:
			return float64(typed), nil
		case json.Number:
			value, err := typed.Float64()
			if err != nil {
				return nil, fmt.Errorf("invalid number")
			}
			return value, nil
		case string:
			text := strings.TrimSpace(typed)
			if text == "" {
				return nil, fmt.Errorf("value cannot be empty")
			}
			value, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number")
			}
			return value, nil
		default:
			return nil, fmt.Errorf("expected number")
		}
	case ConfigFieldTypeBoolean:
		switch typed := raw.(type) {
		case bool:
			return typed, nil
		case string:
			text := strings.ToLower(strings.TrimSpace(typed))
			switch text {
			case "true", "1", "yes", "y", "on":
				return true, nil
			case "false", "0", "no", "n", "off":
				return false, nil
			default:
				return nil, fmt.Errorf("invalid boolean")
			}
		default:
			return nil, fmt.Errorf("expected boolean")
		}
	default:
		return nil, fmt.Errorf("unsupported type %s", field.Type)
	}
}

func stringifyConfigValue(field ConfigField, value interface{}) (string, error) {
	switch field.Type {
	case ConfigFieldTypeBoolean:
		typed, ok := value.(bool)
		if !ok {
			return "", fmt.Errorf("expected boolean value for %s", field.Key)
		}
		if typed {
			return "true", nil
		}
		return "false", nil
	case ConfigFieldTypeNumber:
		number, ok := value.(float64)
		if !ok {
			return "", fmt.Errorf("expected number value for %s", field.Key)
		}
		return strconv.FormatFloat(number, 'f', -1, 64), nil
	default:
		text, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("expected string value for %s", field.Key)
		}
		return text, nil
	}
}

func configFieldsCompatible(existing, next ConfigField) bool {
	if strings.TrimSpace(existing.Key) != strings.TrimSpace(next.Key) {
		return true
	}
	return strings.TrimSpace(strings.ToLower(existing.Type)) == strings.TrimSpace(strings.ToLower(next.Type)) &&
		effectiveFieldSecret(existing) == effectiveFieldSecret(next)
}

func effectiveFieldSecret(field ConfigField) bool {
	return field.Secret || strings.TrimSpace(strings.ToLower(field.Type)) == ConfigFieldTypePassword
}
