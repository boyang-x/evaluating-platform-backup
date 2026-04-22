package chat

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
)

func TestBuildMCPCapabilityUsesFriendlyRewriteLabel(t *testing.T) {
	item := model.ExternalMCPServer{
		ID:          uuid.New(),
		Name:        "CCBOS",
		Namespace:   "classical_rewrite",
		Description: "提供文言文改写与文风转换能力",
		Enabled:     true,
		Status:      "connected",
	}

	result := buildMCPCapability(item)
	if !strings.Contains(result.Label, "文言文") {
		t.Fatalf("expected rewrite-friendly label, got %q", result.Label)
	}
	if !strings.Contains(result.Prompt, "文言文") {
		t.Fatalf("expected rewrite-friendly prompt, got %q", result.Prompt)
	}
}

func TestAssembleWelcomeCapabilitiesKeepsBalancedMix(t *testing.T) {
	skills := []welcomeCandidate{
		{WelcomeCapability: WelcomeCapability{ID: "skill-1", Label: "Skill1", SourceKind: "skill", SourceType: "generator_skill"}, priority: 120},
		{WelcomeCapability: WelcomeCapability{ID: "skill-2", Label: "Skill2", SourceKind: "skill", SourceType: "generator_skill"}, priority: 118},
		{WelcomeCapability: WelcomeCapability{ID: "skill-3", Label: "Skill3", SourceKind: "skill", SourceType: "generator_skill"}, priority: 116},
	}
	mcps := []welcomeCandidate{
		{WelcomeCapability: WelcomeCapability{ID: "mcp-1", Label: "MCP1", SourceKind: "mcp", SourceType: "external_mcp_server"}, priority: 114},
		{WelcomeCapability: WelcomeCapability{ID: "mcp-2", Label: "MCP2", SourceKind: "mcp", SourceType: "external_mcp_server"}, priority: 112},
		{WelcomeCapability: WelcomeCapability{ID: "mcp-3", Label: "MCP3", SourceKind: "mcp", SourceType: "external_mcp_server"}, priority: 110},
	}
	resources := []welcomeCandidate{
		{WelcomeCapability: WelcomeCapability{ID: "resource-1", Label: "Res1", SourceKind: "resource", SourceType: "sample_overview"}, priority: 108},
		{WelcomeCapability: WelcomeCapability{ID: "resource-2", Label: "Res2", SourceKind: "resource", SourceType: "template_overview"}, priority: 106},
		{WelcomeCapability: WelcomeCapability{ID: "resource-3", Label: "Res3", SourceKind: "resource", SourceType: "multilingual_template"}, priority: 104},
	}

	result := assembleWelcomeCapabilities(6, skills, mcps, resources)
	if len(result) != 6 {
		t.Fatalf("expected 6 items, got %d", len(result))
	}

	gotIDs := make([]string, 0, len(result))
	for _, item := range result {
		gotIDs = append(gotIDs, item.ID)
	}
	expectedPrefix := []string{"skill-1", "skill-2", "mcp-1", "mcp-2", "resource-1", "resource-2"}
	for idx, id := range expectedPrefix {
		if gotIDs[idx] != id {
			t.Fatalf("expected item %d to be %q, got %q", idx, id, gotIDs[idx])
		}
	}
}
