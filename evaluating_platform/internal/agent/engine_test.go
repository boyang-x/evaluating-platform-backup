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
