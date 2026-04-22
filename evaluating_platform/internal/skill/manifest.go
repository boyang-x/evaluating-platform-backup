package skill

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultMaxArchiveBytes      = 20 << 20
	DefaultMaxUncompressedBytes = 200 << 20
)

type Manifest struct {
	ManifestVersion   string                 `yaml:"manifest_version" json:"manifest_version"`
	SkillType         string                 `yaml:"skill_type" json:"skill_type"`
	Name              string                 `yaml:"name" json:"name"`
	DisplayName       string                 `yaml:"display_name" json:"display_name"`
	Version           string                 `yaml:"version" json:"version"`
	Description       string                 `yaml:"description" json:"description"`
	Category          string                 `yaml:"category" json:"category"`
	CapabilityProfile string                 `yaml:"capability_profile" json:"capability_profile"`
	InputSourceMode   string                 `yaml:"input_source_mode" json:"input_source_mode"`
	AssessmentTypes   []string               `yaml:"assessment_types" json:"assessment_types"`
	Permissions       map[string]interface{} `yaml:"permissions" json:"permissions"`
	Metadata          map[string]interface{} `yaml:"metadata" json:"metadata"`
	ConfigSchema      []ConfigField          `yaml:"config_schema" json:"config_schema,omitempty"`
	Planner           PlannerManifest        `yaml:"planner" json:"planner"`
	Web               WebManifest            `yaml:"web" json:"web"`
	Execution         ExecutionManifest      `yaml:"execution" json:"execution"`
	EmbeddedResources EmbeddedResources      `yaml:"embedded_resources" json:"embedded_resources"`

	ConfigSchemaCompatMode bool   `yaml:"-" json:"-"`
	ConfigSchemaSource     string `yaml:"-" json:"-"`
}

type PlannerManifest struct {
	Summary           string   `yaml:"summary" json:"summary"`
	IntentExamples    []string `yaml:"intent_examples" json:"intent_examples"`
	DeliveryMode      string   `yaml:"delivery_mode" json:"delivery_mode"`
	EnterpriseVisible *bool    `yaml:"enterprise_visible" json:"enterprise_visible,omitempty"`
}

type WebManifest struct {
	Entrypoint string `yaml:"entrypoint" json:"entrypoint"`
}

type ExecutionManifest struct {
	Runtime            string `yaml:"runtime" json:"runtime"`
	Entrypoint         string `yaml:"entrypoint" json:"entrypoint"`
	SelfTestEntrypoint string `yaml:"self_test_entrypoint" json:"self_test_entrypoint"`
	TimeoutSeconds     int    `yaml:"timeout_seconds" json:"timeout_seconds"`
}

type EmbeddedResources struct {
	EmbeddedDatasetID string                 `yaml:"embedded_dataset_id" json:"embedded_dataset_id"`
	Summary           map[string]interface{} `yaml:"summary" json:"summary"`
}

type ImportedPackage struct {
	Manifest         Manifest
	PromptText       string
	ValidationReport json.RawMessage
	ExampleInput     json.RawMessage
	ExampleOutput    json.RawMessage
	ArchiveBytes     []byte
	ArchiveHash      string
	FileNames        []string
	Files            map[string][]byte
}

func ParsePackage(archive []byte, maxArchiveBytes int64) (*ImportedPackage, error) {
	if len(archive) == 0 {
		return nil, fmt.Errorf("empty archive")
	}
	if maxArchiveBytes <= 0 {
		maxArchiveBytes = DefaultMaxArchiveBytes
	}
	if int64(len(archive)) > maxArchiveBytes {
		return nil, fmt.Errorf("archive exceeds size limit")
	}

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}

	files := map[string][]byte{}
	fileNames := make([]string, 0, len(reader.File))
	var totalUncompressed uint64
	for _, file := range reader.File {
		cleaned, err := sanitizeZipPath(file.Name)
		if err != nil {
			return nil, err
		}
		if cleaned == "" {
			continue
		}
		fileNames = append(fileNames, cleaned)
		if file.FileInfo().IsDir() {
			continue
		}
		totalUncompressed += file.UncompressedSize64
		if totalUncompressed > DefaultMaxUncompressedBytes {
			return nil, fmt.Errorf("archive uncompressed size exceeds limit")
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", cleaned, err)
		}
		data, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", cleaned, readErr)
		}
		files[cleaned] = data
	}

	manifestBytes, ok := files["skill.yaml"]
	if !ok {
		return nil, fmt.Errorf("skill.yaml is required")
	}

	var manifest Manifest
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("parse skill.yaml: %w", err)
	}
	normalizeManifest(&manifest)
	if err := normalizeManifestConfigSchema(&manifest); err != nil {
		return nil, err
	}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}

	report := rawJSONOrDefault(files["submission/validation-report.json"], `{}`)
	exampleInput := rawJSONOrDefault(files["examples/input.json"], `{}`)
	exampleOutput := rawJSONOrDefault(files["examples/output.json"], `{}`)
	promptText := string(files["prompt.md"])

	sum := sha256.Sum256(archive)
	return &ImportedPackage{
		Manifest:         manifest,
		PromptText:       promptText,
		ValidationReport: report,
		ExampleInput:     exampleInput,
		ExampleOutput:    exampleOutput,
		ArchiveBytes:     archive,
		ArchiveHash:      hex.EncodeToString(sum[:]),
		FileNames:        fileNames,
		Files:            files,
	}, nil
}

func normalizeManifest(m *Manifest) {
	m.ManifestVersion = strings.TrimSpace(defaultString(m.ManifestVersion, "1.0"))
	m.SkillType = strings.TrimSpace(defaultString(m.SkillType, "generator_skill"))
	m.DisplayName = strings.TrimSpace(defaultString(m.DisplayName, m.Name))
	if m.Execution.TimeoutSeconds <= 0 {
		m.Execution.TimeoutSeconds = 180
	}
	if m.Permissions == nil {
		m.Permissions = map[string]interface{}{}
	}
	if m.Metadata == nil {
		m.Metadata = map[string]interface{}{}
	}
	if m.EmbeddedResources.Summary == nil {
		m.EmbeddedResources.Summary = map[string]interface{}{}
	}
	switch m.SkillType {
	case "interactive_web_skill":
		m.InputSourceMode = strings.TrimSpace(defaultString(m.InputSourceMode, "embedded_dataset_only"))
		m.Web.Entrypoint = strings.TrimSpace(defaultString(m.Web.Entrypoint, m.Execution.Entrypoint))
		m.Execution.Runtime = strings.TrimSpace(defaultString(m.Execution.Runtime, "managed_web"))
		m.Execution.Entrypoint = strings.TrimSpace(defaultString(m.Execution.Entrypoint, m.Web.Entrypoint))
		m.Execution.SelfTestEntrypoint = strings.TrimSpace(defaultString(m.Execution.SelfTestEntrypoint, "platform_managed"))
		m.Planner.Summary = strings.TrimSpace(defaultString(m.Planner.Summary, m.Description))
		m.Planner.DeliveryMode = strings.TrimSpace(defaultString(m.Planner.DeliveryMode, "open_url"))
		if m.Planner.EnterpriseVisible == nil {
			m.Planner.EnterpriseVisible = boolPtr(true)
		}
	default:
		m.InputSourceMode = strings.TrimSpace(defaultString(m.InputSourceMode, "embedded_dataset_only"))
		m.Execution.Runtime = strings.TrimSpace(defaultString(m.Execution.Runtime, "python3.12"))
	}
}

func validateManifest(m Manifest) error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("skill name is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("skill version is required")
	}
	switch m.InputSourceMode {
	case "embedded_dataset_only", "platform_resource_only", "hybrid":
	default:
		return fmt.Errorf("unsupported input_source_mode: %s", m.InputSourceMode)
	}
	switch m.SkillType {
	case "generator_skill":
		if m.Execution.Runtime != "python3.12" {
			return fmt.Errorf("only python3.12 runtime is supported for generator_skill")
		}
		if strings.TrimSpace(m.Execution.Entrypoint) == "" {
			return fmt.Errorf("execution.entrypoint is required")
		}
		if strings.TrimSpace(m.Execution.SelfTestEntrypoint) == "" {
			return fmt.Errorf("execution.self_test_entrypoint is required")
		}
	case "interactive_web_skill":
		if m.Execution.Runtime != "managed_web" {
			return fmt.Errorf("interactive_web_skill must use managed_web runtime")
		}
		if len(m.ConfigSchema) > 0 {
			return fmt.Errorf("interactive_web_skill does not support config_schema")
		}
		if strings.TrimSpace(m.Web.Entrypoint) == "" {
			return fmt.Errorf("web.entrypoint is required for interactive_web_skill")
		}
		if strings.TrimSpace(m.Planner.DeliveryMode) != "" && m.Planner.DeliveryMode != "open_url" {
			return fmt.Errorf("interactive_web_skill only supports planner.delivery_mode=open_url")
		}
	default:
		return fmt.Errorf("unsupported skill_type: %s", m.SkillType)
	}
	if _, err := normalizeConfigFields(m.ConfigSchema); err != nil {
		return err
	}
	return nil
}

func sanitizeZipPath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	cleaned := path.Clean(strings.TrimSpace(name))
	if cleaned == "." || cleaned == "/" {
		return "", nil
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("zip contains unsafe path: %s", name)
	}
	return cleaned, nil
}

func rawJSONOrDefault(data []byte, fallback string) json.RawMessage {
	if len(bytes.TrimSpace(data)) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(data)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func boolPtr(value bool) *bool {
	return &value
}
