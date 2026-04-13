package chat

import "testing"

func TestExtractJSONCommandFromEmbeddedText(t *testing.T) {
	raw := "好的，我为您生成计划。\n{\"action\":\"confirm_plan\",\"message\":\"请确认\",\"plan\":{\"name\":\"内容合规检测\",\"goal\":\"检测合规风险\",\"target_type\":\"openai\",\"assessment_types\":[\"compliance_check\"]}}\n请点击确认。"
	cmd, ok := extractJSONCommand(raw)
	if !ok {
		t.Fatalf("expected embedded command to be parsed")
	}
	if cmd["action"] != "confirm_plan" {
		t.Fatalf("unexpected action: %v", cmd["action"])
	}
}

func TestDetectAssessmentTypes(t *testing.T) {
	text := "请对我的 AI 系统进行内容合规性检查，并补充越狱攻击和提示词注入测试。"
	types := detectAssessmentTypes(text)
	expected := map[string]bool{
		"compliance_check": true,
		"prompt_injection": true,
		"jailbreak":        true,
	}
	for _, item := range types {
		delete(expected, item)
	}
	if len(expected) != 0 {
		t.Fatalf("missing assessment types: %v", expected)
	}
}

func TestLooksLikeConfirmation(t *testing.T) {
	if !looksLikeConfirmation("评估吧，现在开始执行") {
		t.Fatalf("expected confirmation phrase to be detected")
	}
	if looksLikeConfirmation("我想再看看计划细节") {
		t.Fatalf("did not expect non-confirmation phrase to be detected")
	}
}

func TestDetectResourceModePreferenceSampleRewrite(t *testing.T) {
	text := "\u6211\u662f\u8981\u8ba9\u6837\u672c\u95ee\u9898\u7ecf\u8fc7 ccbos mcp \u7684\u8fed\u4ee3\u4f18\u5316\u53d8\u6210\u6587\u8a00\u6587\u5f62\u5f0f\uff0c\u4e0d\u662f\u6a21\u677f+\u6837\u672c\uff0c\u4e5f\u4e0d\u662f\u5df2\u7ec4\u5408\u653b\u51fb"
	if got := detectResourceModePreference(text); got != "sample_rewrite" {
		t.Fatalf("expected sample_rewrite, got %q", got)
	}
}

func TestNormalizePlanResourceModePreferenceRewriteOverridesLegacyMode(t *testing.T) {
	goal := "\u901a\u8fc7 CCBOS \u5c06\u6837\u672c\u95ee\u9898\u8fed\u4ee3\u6539\u5199\u4e3a\u6587\u8a00\u6587\u5f62\u5f0f\u540e\u518d\u6267\u884c\u6d4b\u8bd5"
	if got := normalizePlanResourceModePreference(goal, "sample_template"); got != "sample_rewrite" {
		t.Fatalf("expected sample_rewrite, got %q", got)
	}
}

func TestResolvePlanResourceModePreferenceUsesUserIntent(t *testing.T) {
	intent := "\u8bf7\u4f7f\u7528 CCBOS MCP \u628a\u6837\u672c\u95ee\u9898\u8fed\u4ee3\u6539\u5199\u6210\u6587\u8a00\u6587\uff0c\u4e0d\u8981\u6a21\u677f+\u6837\u672c"
	goal := "\u68c0\u6d4b\u76ee\u6807 AI Agent \u7684\u63d0\u793a\u8bcd\u6ce8\u5165\u98ce\u9669"
	if got := resolvePlanResourceModePreference(intent, goal, "sample_template"); got != "sample_rewrite" {
		t.Fatalf("expected sample_rewrite from user intent, got %q", got)
	}
}

func TestGoalDescribesSampleRewriteRequiresRewriteLanguage(t *testing.T) {
	goal := "\u901a\u8fc7 CCBOS MCP \u5c06\u6837\u672c\u95ee\u9898\u8fed\u4ee3\u6539\u5199\u4e3a\u6587\u8a00\u6587\u540e\u6267\u884c\u6d4b\u8bd5"
	if !goalDescribesSampleRewrite(goal) {
		t.Fatalf("expected goal to describe sample rewrite")
	}
	if goalDescribesSampleRewrite("\u8bc4\u4f30 CCBOS MCP AI Agent \u7cfb\u7edf\u7684\u5b89\u5168\u6027") {
		t.Fatalf("did not expect generic CCBOS goal to describe sample rewrite")
	}
}
