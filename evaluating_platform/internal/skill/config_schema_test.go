package skill

import "testing"

func TestParsePackageSupportsConfigSchema(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"skill.yaml": `manifest_version: "1.0"
skill_type: generator_skill
name: configurable-skill
display_name: Configurable Skill
version: "1.0.0"
description: Configurable generator skill
category: example
capability_profile: configurable_skill
input_source_mode: platform_resource_only
assessment_types:
  - jailbreak
config_schema:
  - key: region
    label: Region
    type: select
    required: true
    description: Runtime region selector
    env_name: REGION
    options:
      - label: China
        value: cn
      - label: US
        value: us
  - key: api_key
    label: API Key
    type: text
    required: false
    description: Upstream API key
    env_name: API_KEY
    secret: true
permissions:
  code_execution: true
execution:
  runtime: python3.12
  entrypoint: runtime/main.py
  self_test_entrypoint: runtime/selfcheck.py
embedded_resources:
  summary:
    mode: platform_sample_only
`,
	})

	pkg, err := ParsePackage(archive, DefaultMaxArchiveBytes)
	if err != nil {
		t.Fatalf("ParsePackage returned error: %v", err)
	}
	if len(pkg.Manifest.ConfigSchema) != 2 {
		t.Fatalf("expected 2 config fields, got %d", len(pkg.Manifest.ConfigSchema))
	}
	if pkg.Manifest.ConfigSchemaSource != "config_schema" {
		t.Fatalf("expected config_schema source, got %q", pkg.Manifest.ConfigSchemaSource)
	}
	if pkg.Manifest.ConfigSchemaCompatMode {
		t.Fatal("expected explicit config_schema to not be compat mode")
	}
}

func TestParsePackageNormalizesLegacyRuntimeConfig(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"skill.yaml": `manifest_version: "1.0"
skill_type: generator_skill
name: legacy-runtime-config
display_name: Legacy Runtime Config
version: "1.0.0"
description: Legacy runtime config manifest
category: example
capability_profile: legacy_runtime_config
input_source_mode: platform_resource_only
assessment_types:
  - jailbreak
metadata:
  runtime_config:
    env:
      - name: SKILL_LLM_API_KEY
        required: false
        secret: true
        fallback_env:
          - LLM_API_KEY
        description: Upstream API key
permissions:
  code_execution: true
execution:
  runtime: python3.12
  entrypoint: runtime/main.py
  self_test_entrypoint: runtime/selfcheck.py
embedded_resources:
  summary:
    mode: platform_sample_only
`,
	})

	pkg, err := ParsePackage(archive, DefaultMaxArchiveBytes)
	if err != nil {
		t.Fatalf("ParsePackage returned error: %v", err)
	}
	if len(pkg.Manifest.ConfigSchema) != 1 {
		t.Fatalf("expected 1 config field, got %d", len(pkg.Manifest.ConfigSchema))
	}
	field := pkg.Manifest.ConfigSchema[0]
	if field.Key != "SKILL_LLM_API_KEY" || field.EnvName != "SKILL_LLM_API_KEY" {
		t.Fatalf("unexpected normalized legacy field: %+v", field)
	}
	if len(field.FallbackEnv) != 1 || field.FallbackEnv[0] != "LLM_API_KEY" {
		t.Fatalf("expected fallback env to be preserved, got %+v", field.FallbackEnv)
	}
	if !pkg.Manifest.ConfigSchemaCompatMode || pkg.Manifest.ConfigSchemaSource != "metadata.runtime_config" {
		t.Fatalf("expected compat metadata.runtime_config normalization, got compat=%v source=%q", pkg.Manifest.ConfigSchemaCompatMode, pkg.Manifest.ConfigSchemaSource)
	}
}

func TestParsePackageRejectsInteractiveConfigSchema(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"skill.yaml": `manifest_version: "1.0"
skill_type: interactive_web_skill
name: invalid-web-skill
display_name: Invalid Web Skill
version: "1.0.0"
description: Should be rejected
config_schema:
  - key: token
    label: Token
    type: text
    required: false
    description: Token
    env_name: TOKEN
web:
  entrypoint: templates/index.html
embedded_resources:
  summary:
    mode: embedded_static_html
`,
		"templates/index.html": `<html><body>ok</body></html>`,
	})

	if _, err := ParsePackage(archive, DefaultMaxArchiveBytes); err == nil {
		t.Fatal("expected ParsePackage to reject interactive_web_skill config_schema")
	}
}

func TestParsePackageRejectsSecretDefault(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"skill.yaml": `manifest_version: "1.0"
skill_type: generator_skill
name: invalid-default-skill
display_name: Invalid Default Skill
version: "1.0.0"
description: Should be rejected
config_schema:
  - key: api_key
    label: API Key
    type: text
    required: false
    description: Invalid secret default
    env_name: API_KEY
    secret: true
    default: leaked
permissions:
  code_execution: true
execution:
  runtime: python3.12
  entrypoint: runtime/main.py
  self_test_entrypoint: runtime/selfcheck.py
embedded_resources:
  summary:
    mode: platform_sample_only
`,
	})

	if _, err := ParsePackage(archive, DefaultMaxArchiveBytes); err == nil {
		t.Fatal("expected ParsePackage to reject secret default")
	}
}
