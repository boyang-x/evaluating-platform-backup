package agent

import "testing"

func TestApplyClassicalChineseRewritePreferencePrefersSkillGenerated(t *testing.T) {
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

	decision := applyClassicalChineseRewritePreference(pipelineSelectionDecision{}, recommendResp, "请做一轮文言文绕过测试", "sample_rewrite")
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

func TestFallbackDecisionFromRewriteFailureUsesSkillWhenRewriteUnavailable(t *testing.T) {
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

	decision, ok := fallbackDecisionFromRewriteFailure(recommendResp, pipelineSelectionDecision{
		Mode:     "sample_rewrite",
		SampleID: "sample-1",
	}, errString("tool error: enabled external MCP server not found for namespace: ccbos"))
	if !ok {
		t.Fatalf("expected rewrite failure fallback to be available")
	}
	if decision.Mode != "skill_generated" {
		t.Fatalf("expected skill_generated mode, got %q", decision.Mode)
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}
