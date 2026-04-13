package mcptools

import (
	"strings"
	"testing"
)

func TestBuildReportPrompt_WithResults(t *testing.T) {
	results := []ExecutionResult{
		{Index: 1, OriginalContent: "样本1", TargetResponse: "回答1", AttackSuccess: true, AttackReason: "成功原因"},
		{Index: 2, OriginalContent: "样本2", TargetResponse: "拒绝", AttackSuccess: false, AttackReason: "被拒绝"},
	}

	prompt := buildReportPrompt("测试目标", results)
	for _, expected := range []string{"测试目标", "risk_level", "findings", "回答1", "拒绝"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt should contain %q", expected)
		}
	}
}

func TestBuildReportPrompt_EmptyResults(t *testing.T) {
	prompt := buildReportPrompt("空测试", nil)
	if !strings.Contains(prompt, "空测试") {
		t.Fatalf("prompt should contain goal")
	}
	if !strings.Contains(prompt, "0") {
		t.Fatalf("prompt should contain zero-value statistics")
	}
}

func TestBuildReportPrompt_AllSuccess(t *testing.T) {
	results := []ExecutionResult{
		{Index: 1, OriginalContent: "s1", TargetResponse: "r1", AttackSuccess: true, AttackReason: "ok"},
		{Index: 2, OriginalContent: "s2", TargetResponse: "r2", AttackSuccess: true, AttackReason: "ok"},
	}

	prompt := buildReportPrompt("全部成功", results)
	if !strings.Contains(prompt, "100.00%") {
		t.Fatalf("prompt should show 100%% success rate")
	}
}

func TestBuildReportPrompt_WithErrors(t *testing.T) {
	results := []ExecutionResult{
		{Index: 1, OriginalContent: "s1", Error: "connection timeout"},
	}

	prompt := buildReportPrompt("错误测试", results)
	if !strings.Contains(prompt, "connection timeout") {
		t.Fatalf("prompt should include error information")
	}
}
