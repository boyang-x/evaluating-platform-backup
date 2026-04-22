package skill

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestParsePackageSupportsInteractiveWebSkill(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"skill.yaml": `manifest_version: "1.0"
skill_type: interactive_web_skill
name: ai-risk-viz
display_name: AI Risk Viz
version: "1.0.0"
description: Open an interactive risk page
category: ai_risk_viz
capability_profile: interactive_risk_viz
planner:
  summary: Open the risk page
  intent_examples:
    - 打开 AI 风险页面
web:
  entrypoint: templates/index.html
embedded_resources:
  summary:
    mode: embedded_static_html
`,
		"templates/index.html": `<html><head><title>Risk Viz</title><script src="https://example.com/app.js"></script></head><body>ok</body></html>`,
	})

	pkg, err := ParsePackage(archive, DefaultMaxArchiveBytes)
	if err != nil {
		t.Fatalf("ParsePackage returned error: %v", err)
	}

	if pkg.Manifest.SkillType != "interactive_web_skill" {
		t.Fatalf("expected interactive_web_skill, got %q", pkg.Manifest.SkillType)
	}
	if pkg.Manifest.Execution.Runtime != "managed_web" {
		t.Fatalf("expected managed_web runtime, got %q", pkg.Manifest.Execution.Runtime)
	}
	if pkg.Manifest.Execution.Entrypoint != "templates/index.html" {
		t.Fatalf("expected execution entrypoint to mirror web entrypoint, got %q", pkg.Manifest.Execution.Entrypoint)
	}
	if pkg.Manifest.Planner.DeliveryMode != "open_url" {
		t.Fatalf("expected default delivery mode open_url, got %q", pkg.Manifest.Planner.DeliveryMode)
	}
}

func TestPrepareInteractiveLaunchRejectsRelativeAssets(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"skill.yaml": `manifest_version: "1.0"
skill_type: interactive_web_skill
name: relative-asset-skill
display_name: Relative Asset Skill
version: "1.0.0"
description: Relative asset demo
web:
  entrypoint: templates/index.html
embedded_resources:
  summary:
    mode: embedded_static_html
`,
		"templates/index.html": `<html><head><title>Relative Asset</title><script src="libs/app.js"></script></head><body>broken</body></html>`,
	})

	pkg, err := ParsePackage(archive, DefaultMaxArchiveBytes)
	if err != nil {
		t.Fatalf("ParsePackage returned error: %v", err)
	}

	_, _, report, err := prepareInteractiveLaunch(pkg)
	if err == nil {
		t.Fatal("expected relative asset validation to fail")
	}
	if !strings.Contains(err.Error(), "libs/app.js") {
		t.Fatalf("expected error to mention relative asset, got %v", err)
	}
	if !strings.Contains(string(report), "relative_asset_refs") {
		t.Fatalf("expected validation report to include relative asset refs, got %s", string(report))
	}
}

func TestPrepareInteractiveLaunchAcceptsAbsoluteAssets(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"skill.yaml": `manifest_version: "1.0"
skill_type: interactive_web_skill
name: absolute-asset-skill
display_name: Absolute Asset Skill
version: "1.0.0"
description: Absolute asset demo
planner:
  summary: Open the absolute-asset page
web:
  entrypoint: templates/index.html
embedded_resources:
  summary:
    mode: embedded_static_html
`,
		"templates/index.html": `<html><head><title>Absolute Asset</title><script src="https://cdn.example.com/app.js"></script></head><body>ok</body></html>`,
	})

	pkg, err := ParsePackage(archive, DefaultMaxArchiveBytes)
	if err != nil {
		t.Fatalf("ParsePackage returned error: %v", err)
	}

	title, htmlText, report, err := prepareInteractiveLaunch(pkg)
	if err != nil {
		t.Fatalf("prepareInteractiveLaunch returned error: %v", err)
	}
	if title != "Absolute Asset" {
		t.Fatalf("expected title to be extracted, got %q", title)
	}
	if !strings.Contains(htmlText, "https://cdn.example.com/app.js") {
		t.Fatalf("expected html to be returned unchanged, got %q", htmlText)
	}
	if !strings.Contains(string(report), "remote_asset_refs") {
		t.Fatalf("expected validation report to include remote asset refs, got %s", string(report))
	}
}

func buildSkillArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, content := range files {
		fileWriter, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := fileWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}
