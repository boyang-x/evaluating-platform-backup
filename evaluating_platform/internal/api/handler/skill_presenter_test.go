package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	skillpkg "evaluating_platform/internal/skill"
)

func TestPresentSkillBundleDetailRedactsSensitiveFields(t *testing.T) {
	now := time.Now()
	skillID := uuid.New()
	latestVersionID := uuid.New()
	publishedVersionID := uuid.New()

	bundle := &skillpkg.SkillBundle{
		Skill: &model.Skill{
			ID:                 skillID,
			Name:               "Classical Skill",
			Slug:               "classical_skill",
			Description:        "generate classical chinese jailbreak prompts",
			SkillType:          "generator_skill",
			Category:           "jailbreak",
			CapabilityProfile:  "offline_generator",
			Status:             model.SkillStatusPublished,
			LatestVersionID:    &latestVersionID,
			PublishedVersionID: &publishedVersionID,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		Versions: []model.SkillVersion{
			{
				ID:                     latestVersionID,
				SkillID:                skillID,
				Version:                "1.1.0",
				ManifestVersion:        "1.0",
				DisplayName:            "Classical Skill 1.1",
				Summary:                "latest",
				PackageObjectPath:      "skills/secret.zip",
				PromptText:             "do not expose",
				InputSourceMode:        "embedded_dataset_only",
				ExecutionRuntime:       "python3.12",
				Permissions:            json.RawMessage(`{"code_execution":true}`),
				EmbeddedDatasetSummary: json.RawMessage(`{"dataset_id":"seed-v1"}`),
				AssessmentTypes:        json.RawMessage(`["jailbreak"]`),
				ValidationReport:       json.RawMessage(`{"all_passed":true}`),
				Examples:               json.RawMessage(`{"input":"secret"}`),
				Status:                 model.SkillVersionStatusSelfTestPassed,
				CreatedAt:              now,
				UpdatedAt:              now,
			},
			{
				ID:                     publishedVersionID,
				SkillID:                skillID,
				Version:                "1.0.0",
				ManifestVersion:        "1.0",
				DisplayName:            "Classical Skill 1.0",
				Summary:                "published",
				PackageObjectPath:      "skills/secret-old.zip",
				PromptText:             "still secret",
				InputSourceMode:        "embedded_dataset_only",
				ExecutionRuntime:       "python3.12",
				Permissions:            json.RawMessage(`{"code_execution":false}`),
				EmbeddedDatasetSummary: json.RawMessage(`{"dataset_id":"seed-v0"}`),
				AssessmentTypes:        json.RawMessage(`["jailbreak"]`),
				ValidationReport:       json.RawMessage(`{"all_passed":true}`),
				Examples:               json.RawMessage(`{"output":"secret"}`),
				Status:                 model.SkillVersionStatusPublished,
				CreatedAt:              now,
				UpdatedAt:              now,
			},
		},
		Runs: []model.SkillRun{
			{
				ID:                    uuid.New(),
				SkillID:               skillID,
				SkillVersionID:        latestVersionID,
				RunType:               model.SkillRunTypeGenerate,
				Status:                model.SkillRunStatusCompleted,
				ResultPayload:         json.RawMessage(`{"payloads":["secret"]}`),
				PayloadDatasetSummary: json.RawMessage(`{"count":2}`),
				StdoutLog:             "hidden stdout",
				StderrLog:             "hidden stderr",
				CreatedAt:             now,
				UpdatedAt:             now,
			},
		},
	}

	detail := presentSkillBundleDetail(bundle)
	body, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal detail: %v", err)
	}
	text := string(body)

	for _, forbidden := range []string{
		"prompt_text",
		"package_object_path",
		"examples",
		"result_payload",
		"stdout_log",
		"stderr_log",
		"do not expose",
		"skills/secret.zip",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("expected redacted detail to omit %q, got %s", forbidden, text)
		}
	}

	if detail.InputSourceMode != "embedded_dataset_only" {
		t.Fatalf("unexpected input source mode: %s", detail.InputSourceMode)
	}
	if detail.VersionCount != 2 || detail.RunCount != 1 {
		t.Fatalf("unexpected counts: versions=%d runs=%d", detail.VersionCount, detail.RunCount)
	}
	if len(detail.Runs) != 1 || detail.Runs[0].LogExcerpt != "" {
		t.Fatalf("expected generate run without log excerpt, got %+v", detail.Runs)
	}
}

func TestPresentSkillRunSelfTestUsesLogExcerptOnly(t *testing.T) {
	run := &model.SkillRun{
		ID:             uuid.New(),
		SkillVersionID: uuid.New(),
		RunType:        model.SkillRunTypeSelfTest,
		Status:         model.SkillRunStatusFailed,
		StdoutLog:      strings.Repeat("a", 1500),
		StderrLog:      strings.Repeat("b", 1500),
		ResultPayload:  json.RawMessage(`{"secret":"payload"}`),
	}

	presented := presentSkillRun(run, nil)
	if presented.LogExcerpt == "" {
		t.Fatal("expected self-test run to include log excerpt")
	}
	if len([]rune(presented.LogExcerpt)) > maxSkillLogExcerptChars+32 {
		t.Fatalf("expected log excerpt to be truncated, got length %d", len([]rune(presented.LogExcerpt)))
	}

	body, err := json.Marshal(presented)
	if err != nil {
		t.Fatalf("marshal presented run: %v", err)
	}
	if strings.Contains(string(body), "secret") {
		t.Fatalf("expected presented run JSON to omit result payload, got %s", string(body))
	}
}
