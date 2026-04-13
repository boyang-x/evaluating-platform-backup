package billing

import (
	"testing"
)

// ─── ToolPrice ────────────────────────────────────────────────────────────────

func TestToolPrice_KnownTools(t *testing.T) {
	cases := []struct {
		name string
		want float64
	}{
		{"prompt_injection", 0.50},
		{"jailbreak", 0.30},
		{"compliance_check", 0.40},
		{"compliance_evaluator", 1.00},
		{"goal_hijacking", 0.60},
		{"tool_poisoning", 0.60},
		{"report_generator", 2.00},
	}
	for _, c := range cases {
		got := ToolPrice(c.name)
		if got != c.want {
			t.Errorf("ToolPrice(%q) = %.2f, want %.2f", c.name, got, c.want)
		}
	}
}

func TestToolPrice_UnknownTool(t *testing.T) {
	got := ToolPrice("nonexistent_tool")
	if got != DefaultToolPrice {
		t.Errorf("ToolPrice(unknown) = %.2f, want %.2f (DefaultToolPrice)", got, DefaultToolPrice)
	}
}

// ─── ToolCategory ─────────────────────────────────────────────────────────────

func TestToolCategory_KnownTools(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"prompt_injection", "llm_security"},
		{"jailbreak", "llm_security"},
		{"compliance_check", "compliance"},
		{"compliance_evaluator", "compliance"},
		{"goal_hijacking", "agent_security"},
		{"tool_poisoning", "agent_security"},
		{"report_generator", "reporting"},
	}
	for _, c := range cases {
		got := ToolCategory(c.name)
		if got != c.want {
			t.Errorf("ToolCategory(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestToolCategory_UnknownTool(t *testing.T) {
	got := ToolCategory("mystery_tool")
	if got != "unknown" {
		t.Errorf("ToolCategory(unknown) = %q, want 'unknown'", got)
	}
}

// ─── CalculateToolCost ────────────────────────────────────────────────────────

func TestCalculateToolCost_NoTokens(t *testing.T) {
	// 无 token 费用时，总费用等于工具单价
	got := CalculateToolCost("jailbreak", 0)
	if got != 0.30 {
		t.Errorf("CalculateToolCost(jailbreak, 0) = %.4f, want 0.30", got)
	}
}

func TestCalculateToolCost_WithTokens(t *testing.T) {
	// 1000 tokens = 0.01 元 token 费，加 jailbreak 工具价 0.30 元
	got := CalculateToolCost("jailbreak", 1000)
	want := 0.30 + 0.01
	if got != want {
		t.Errorf("CalculateToolCost(jailbreak, 1000) = %.4f, want %.4f", got, want)
	}
}

func TestCalculateToolCost_PartialThousandTokens(t *testing.T) {
	// 500 tokens = 0.005 元 token 费
	got := CalculateToolCost("report_generator", 500)
	want := 2.00 + 0.005
	if got != want {
		t.Errorf("CalculateToolCost(report_generator, 500) = %.4f, want %.4f", got, want)
	}
}

// ─── CalculateTokenCost ───────────────────────────────────────────────────────

func TestCalculateTokenCost(t *testing.T) {
	cases := []struct {
		tokens int
		want   float64
	}{
		{0, 0},
		{1000, 0.01},
		{5000, 0.05},
		{500, 0.005},
		{100000, 1.00},
	}
	for _, c := range cases {
		got := CalculateTokenCost(c.tokens)
		if got != c.want {
			t.Errorf("CalculateTokenCost(%d) = %.4f, want %.4f", c.tokens, got, c.want)
		}
	}
}

// ─── constants sanity ─────────────────────────────────────────────────────────

func TestConstants_SanityCheck(t *testing.T) {
	if MinAssessmentBalance <= 0 {
		t.Error("MinAssessmentBalance must be positive")
	}
	if TokenCostPer1K <= 0 {
		t.Error("TokenCostPer1K must be positive")
	}
	if DefaultToolPrice <= 0 {
		t.Error("DefaultToolPrice must be positive")
	}
}
