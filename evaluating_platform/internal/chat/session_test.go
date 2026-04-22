package chat

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	skillpkg "evaluating_platform/internal/skill"
)

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

func TestBuildIntentSystemMessageMergesLaunchPromptIntoSingleSystemMessage(t *testing.T) {
	candidates := map[string]skillpkg.LaunchSkillCandidate{
		uuid.NewString(): {
			SkillID:        uuid.New(),
			SkillName:      "AI 风险检索",
			SkillSlug:      "ai-risk-viz",
			Description:    "Open a published interactive HTML skill in a new tab.",
			PlannerSummary: "打开一个互动检索页面",
			DeliveryMode:   "open_url",
		},
	}

	msg := buildIntentSystemMessage(candidates)
	if msg.Role != "system" {
		t.Fatalf("expected system role, got %q", msg.Role)
	}
	if !strings.Contains(msg.Content, "Published interactive_web_skill candidates") {
		t.Fatalf("expected launch prompt to be merged into system content")
	}
	if count := strings.Count(msg.Content, "Published interactive_web_skill candidates"); count != 1 {
		t.Fatalf("expected one merged launch prompt, got %d", count)
	}
}

func TestResolveClassicalChineseRewriteModePreferencePrefersSkill(t *testing.T) {
	got := resolveClassicalChineseRewriteModePreference("sample_rewrite", true, false)
	if got != "skill_generated" {
		t.Fatalf("expected skill_generated, got %q", got)
	}
}

func TestResolveClassicalChineseRewriteModePreferenceFallsBackToRewriteOnlyWhenMCPEnabled(t *testing.T) {
	got := resolveClassicalChineseRewriteModePreference("", false, true)
	if got != "sample_rewrite" {
		t.Fatalf("expected sample_rewrite, got %q", got)
	}
}

func TestResolveClassicalChineseRewriteModePreferenceClearsRewriteWhenNoCapability(t *testing.T) {
	got := resolveClassicalChineseRewriteModePreference("sample_rewrite", false, false)
	if got != "" {
		t.Fatalf("expected empty preference, got %q", got)
	}
}

func TestCanonicalizePlanGoalRebuildsSkillGeneratedGoalWhenLLMStillMentionsMCP(t *testing.T) {
	got := canonicalizePlanGoal("通过 CCBOS MCP 改写能力生成文言文测试问题", []string{"jailbreak"}, "skill_generated")
	if strings.Contains(strings.ToLower(got), "mcp") {
		t.Fatalf("expected canonicalized goal to avoid MCP wording, got %q", got)
	}
}

func TestNormalizedPlanTextValueTreatsNilPlaceholdersAsEmpty(t *testing.T) {
	for _, value := range []any{nil, "<nil>", " nil ", "null", "undefined"} {
		if got := normalizedPlanTextValue(value); got != "" {
			t.Fatalf("expected empty normalized text for %#v, got %q", value, got)
		}
	}
	if got := normalizedPlanTextValue("  合规检测  "); got != "合规检测" {
		t.Fatalf("expected trimmed text, got %q", got)
	}
}

func TestCanonicalizePlanGoalRebuildsWhenGoalWasNilPlaceholder(t *testing.T) {
	got := canonicalizePlanGoal(normalizedPlanTextValue(nil), []string{"jailbreak"}, "skill_generated")
	if got == "" || strings.Contains(strings.ToLower(got), "<nil>") {
		t.Fatalf("expected rebuilt goal, got %q", got)
	}
}

func TestCanonicalizePlanConfirmationMessageRebuildsSkillGeneratedMessageWhenLLMStillMentionsMCP(t *testing.T) {
	planInfo := map[string]any{
		"resource_mode_preference": "skill_generated",
		"test_count":               20,
	}
	got := canonicalizePlanConfirmationMessage("我会调用 CCBOS MCP 改写能力生成测试问题。", planInfo, []string{"jailbreak"}, false)
	if strings.Contains(strings.ToLower(got), "mcp") {
		t.Fatalf("expected canonicalized message to avoid MCP wording, got %q", got)
	}
}

func TestLooksLikePlanNarrative(t *testing.T) {
	text := "评估计划概要：\n- 评估名称：文言文越狱安全评估\n- 资源模式：sample_rewrite\n- 测试数量：20\n是否确认执行此计划？"
	if !looksLikePlanNarrative(text) {
		t.Fatalf("expected narrative plan text to be detected")
	}
}

func TestFallbackPlanSeedFromText(t *testing.T) {
	userText := "请使用文言文改写能力，为目标应用设计一轮越狱安全评估。"
	rawContent := "评估计划概要：\n- 评估类型：jailbreak\n- 资源模式：sample_rewrite\n- 测试数量：20\n是否确认执行此计划？"
	planInfo, ok := fallbackPlanSeedFromText(userText, rawContent)
	if !ok {
		t.Fatalf("expected fallback plan seed to be built")
	}
	if got := stringSliceFromAny(planInfo["assessment_types"]); len(got) != 1 || got[0] != "jailbreak" {
		t.Fatalf("expected jailbreak assessment type, got %#v", got)
	}
	if got, _ := planInfo["resource_mode_preference"].(string); got != "sample_rewrite" {
		t.Fatalf("expected sample_rewrite preference, got %q", got)
	}
}

func TestRecoverPlanNarrativeFromHistory(t *testing.T) {
	history := []*model.ChatMessage{
		{Role: model.RoleUser, Content: "请使用文言文改写能力，为目标应用设计一轮越狱安全评估。"},
		{Role: model.RoleAssistant, Content: "评估计划概要：\n- 评估类型：jailbreak\n- 资源模式：sample_rewrite\n是否确认执行此计划？"},
		{Role: model.RoleUser, Content: "执行"},
	}
	planNarrative, requestText, ok := recoverPlanNarrativeFromHistory(history)
	if !ok {
		t.Fatalf("expected to recover plan narrative from history")
	}
	if !strings.Contains(planNarrative, "评估计划概要") {
		t.Fatalf("unexpected plan narrative: %q", planNarrative)
	}
	if requestText != "请使用文言文改写能力，为目标应用设计一轮越狱安全评估。" {
		t.Fatalf("unexpected recovered request text: %q", requestText)
	}
}
