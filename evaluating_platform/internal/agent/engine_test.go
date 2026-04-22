package agent

import (
	"strings"
	"testing"
)

func TestApplyResourceModePreferenceSampleRewrite(t *testing.T) {
	recommendResp := recommendResourcesResponse{
		RecommendedPairs: []recommendationPair{
			{
				SampleID: "sample-1",
				Reason:   "best rewrite candidate",
			},
		},
		SampleCandidates: []pipelineResourceCard{
			{ID: "sample-1", PlannerSummary: "sample summary"},
		},
	}

	decision := applyResourceModePreference(pipelineSelectionDecision{}, recommendResp, "sample_rewrite")
	if decision.Mode != "sample_rewrite" {
		t.Fatalf("expected sample_rewrite mode, got %q", decision.Mode)
	}
	if decision.SampleID != "sample-1" {
		t.Fatalf("expected sample-1, got %q", decision.SampleID)
	}
	if decision.TemplateID != "" {
		t.Fatalf("expected empty template id, got %q", decision.TemplateID)
	}
}

func TestDecisionMatchesRecommendationsAllowsSampleRewrite(t *testing.T) {
	recommendResp := recommendResourcesResponse{
		SampleCandidates: []pipelineResourceCard{
			{ID: "sample-2"},
		},
	}

	decision := pipelineSelectionDecision{
		Mode:     "sample_rewrite",
		SampleID: "sample-2",
	}
	if !decisionMatchesRecommendations(recommendResp, decision) {
		t.Fatalf("expected sample rewrite decision to match recommendations")
	}
}

func TestApplyResourceModePreferenceSkillGeneratedWithPlatformSample(t *testing.T) {
	recommendResp := recommendResourcesResponse{
		RecommendedSkills: []skillRecommendation{
			{
				SkillID:         "skill-1",
				SkillName:       "CCBOS Skill",
				InputSourceMode: "platform_resource_only",
			},
		},
		SkillCandidates: []pipelineResourceCard{
			{
				ID:              "skill-1",
				Name:            "CCBOS Skill",
				InputSourceMode: "platform_resource_only",
			},
		},
		SampleCandidates: []pipelineResourceCard{
			{ID: "sample-1"},
		},
	}

	decision := applyResourceModePreference(pipelineSelectionDecision{}, recommendResp, "skill_generated")
	if decision.Mode != "skill_generated" {
		t.Fatalf("expected skill_generated mode, got %q", decision.Mode)
	}
	if decision.SkillID != "skill-1" {
		t.Fatalf("expected skill-1, got %q", decision.SkillID)
	}
	if decision.SampleID != "sample-1" {
		t.Fatalf("expected sample-1 for platform_resource_only skill, got %q", decision.SampleID)
	}
}

func TestDecisionMatchesRecommendationsRequiresSampleForPlatformSkill(t *testing.T) {
	recommendResp := recommendResourcesResponse{
		RecommendedSkills: []skillRecommendation{
			{
				SkillID:         "skill-1",
				SkillName:       "CCBOS Skill",
				InputSourceMode: "platform_resource_only",
			},
		},
		SkillCandidates: []pipelineResourceCard{
			{
				ID:              "skill-1",
				Name:            "CCBOS Skill",
				InputSourceMode: "platform_resource_only",
			},
		},
		SampleCandidates: []pipelineResourceCard{
			{ID: "sample-1"},
		},
	}

	if decisionMatchesRecommendations(recommendResp, pipelineSelectionDecision{
		Mode:    "skill_generated",
		SkillID: "skill-1",
	}) {
		t.Fatalf("expected platform_resource_only skill without sample to be rejected")
	}
	if !decisionMatchesRecommendations(recommendResp, pipelineSelectionDecision{
		Mode:     "skill_generated",
		SkillID:  "skill-1",
		SampleID: "sample-1",
	}) {
		t.Fatalf("expected platform_resource_only skill with sample to be accepted")
	}
}

func TestBuildPipelineSummaryForSampleRewrite(t *testing.T) {
	summary := buildPipelineSummary(pipelineSelectionDecision{
		Mode:     "sample_rewrite",
		SampleID: "sample-1",
	}, "none", 3)

	if !strings.Contains(summary, "CC-BOS") {
		t.Fatalf("expected CC-BOS summary, got %q", summary)
	}
	if !strings.Contains(summary, "\u6587\u8a00\u6587") {
		t.Fatalf("expected wenyanwen summary, got %q", summary)
	}
}

func TestBuildPipelineSummaryForSkillGeneratedWithSample(t *testing.T) {
	summary := buildPipelineSummary(pipelineSelectionDecision{
		Mode:      "skill_generated",
		SkillID:   "skill-1",
		SkillName: "CCBOS Skill",
		SampleID:  "sample-1",
	}, "none", 2)

	if !strings.Contains(summary, "CCBOS Skill") {
		t.Fatalf("expected skill name in summary, got %q", summary)
	}
	if !strings.Contains(summary, "专家样本") {
		t.Fatalf("expected sample-backed skill summary, got %q", summary)
	}
}
