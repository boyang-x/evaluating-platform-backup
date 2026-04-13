package report

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"evaluating_platform/internal/agent"
	"evaluating_platform/internal/model"
)

func TestCleanJSONResponse_StripsMarkdownFence(t *testing.T) {
	input := "before\n```json\n{\"title\":\"测试报告\",\"risk_level\":\"low\"}\n```\nafter"
	got := cleanJSONResponse(input)
	want := "{\"title\":\"测试报告\",\"risk_level\":\"low\"}"
	if got != want {
		t.Fatalf("cleanJSONResponse() = %q, want %q", got, want)
	}
}

func TestGenerateFallback_UsesDeterministicTitleAndMetrics(t *testing.T) {
	gen := &Generator{pdfRenderer: NewPDFRenderer("")}
	assessment := &model.Assessment{
		ID:        uuid.New(),
		Name:      "内容合规测试",
		Goal:      "测试内容合规风险",
		CreatedAt: time.Now().Add(-2 * time.Minute),
	}
	logs := []agent.ToolResult{
		{ToolName: "payload_1", Input: "样本问题:\n问题1", Output: "违规回答", Severity: "high"},
		{ToolName: "payload_2", Input: "样本问题:\n问题2", Output: "正常拒绝", Severity: "info"},
		{ToolName: "payload_3", Input: "样本问题:\n问题3", Output: "可疑内容", Severity: "medium"},
	}

	report := gen.GenerateFallback(context.Background(), assessment, logs)
	if !strings.Contains(report.Title, "AI 系统安全评估报告") {
		t.Fatalf("unexpected report title: %s", report.Title)
	}
	if report.RiskLevel != "high" {
		t.Fatalf("unexpected risk level: %s", report.RiskLevel)
	}
	if report.Metrics.TotalTests != 3 {
		t.Fatalf("unexpected total tests: %d", report.Metrics.TotalTests)
	}
	if report.Metrics.SuccessCount != 1 {
		t.Fatalf("unexpected success count: %d", report.Metrics.SuccessCount)
	}
	if report.Metrics.RiskScore < 45 || report.Metrics.RiskScore > 100 {
		t.Fatalf("unexpected risk score: %v", report.Metrics.RiskScore)
	}
	if len(report.Findings) != 3 {
		t.Fatalf("unexpected findings count: %d", len(report.Findings))
	}
	if !strings.Contains(report.Summary, "攻击成功 1 条") {
		t.Fatalf("unexpected summary: %s", report.Summary)
	}
}

func TestBuildReportQuestion_MasksComposedAttackPrompt(t *testing.T) {
	input := "实际发送内容:\n这是一个非常长的已组合攻击载荷文本用于测试报告脱敏展示"
	got := buildReportQuestion(input)
	if !strings.Contains(got, "已组合攻击尾部摘要") {
		t.Fatalf("expected masked composed attack summary, got: %s", got)
	}
	if strings.Contains(got, "这是一个非常长的已组合攻击载荷文本用于测试报告脱敏展示") {
		t.Fatalf("full composed attack prompt should be masked, got: %s", got)
	}
}

func TestAttackExamplesFromLogs_UsesMaskedQuestionForComposedAttack(t *testing.T) {
	logs := []agent.ToolResult{
		{ToolName: "payload_1", Input: "实际发送内容:\n这是一个非常长的已组合攻击载荷文本用于测试报告脱敏展示", Output: "模型回答", Severity: "high"},
	}
	items := attackExamplesFromLogs(logs, 5)
	if len(items) != 1 {
		t.Fatalf("expected 1 attack example, got %d", len(items))
	}
	if !strings.Contains(items[0].SampleQuestion, "已组合攻击尾部摘要") {
		t.Fatalf("expected masked sample question, got: %s", items[0].SampleQuestion)
	}
}

func TestFindingsFromLogs_StoresQuestionAndResponseOnly(t *testing.T) {
	logs := []agent.ToolResult{
		{ToolName: "payload_1", Input: "样本问题:\n用户问题", Output: "模型原回答", Severity: "info"},
	}
	findings := findingsFromLogs(logs)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Description != "用户问题" {
		t.Fatalf("unexpected finding description: %s", findings[0].Description)
	}
	if findings[0].Evidence != "模型原回答" {
		t.Fatalf("unexpected finding evidence: %s", findings[0].Evidence)
	}
	if findings[0].Category != "" {
		t.Fatalf("expected empty category, got: %s", findings[0].Category)
	}
}
