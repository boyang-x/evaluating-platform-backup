package mcptools

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"evaluating_platform/internal/model"
)

type resourceCard struct {
	ID                        string   `json:"id"`
	ExpertID                  string   `json:"expert_id,omitempty"`
	ResourceKind              string   `json:"resource_kind,omitempty"`
	AlreadyComposed           bool     `json:"already_composed,omitempty"`
	Name                      string   `json:"name"`
	SubType                   string   `json:"sub_type"`
	Description               string   `json:"description"`
	Status                    string   `json:"status,omitempty"`
	PlannerSummary            string   `json:"planner_summary"`
	ScenarioTags              []string `json:"scenario_tags"`
	TargetTypes               []string `json:"target_types"`
	ApplicableAssessmentTypes []string `json:"applicable_assessment_types"`
	LanguageSupport           []string `json:"language_support"`
	AttackStyle               string   `json:"attack_style"`
	Difficulty                string   `json:"difficulty"`
	ExpectedSignal            string   `json:"expected_signal"`
	InputSourceMode           string   `json:"input_source_mode,omitempty"`
	SampleCount               int      `json:"sample_count,omitempty"`
	Variables                 []string `json:"variables,omitempty"`
	RecommendedPairings       []string `json:"recommended_pairings,omitempty"`
	SanitizedExamples         []string `json:"sanitized_examples,omitempty"`
	EstimatedDurationSecs     int      `json:"estimated_duration_secs,omitempty"`
	EstimatedCost             string   `json:"estimated_cost,omitempty"`
}

type recommendationResult struct {
	SampleID      string  `json:"sample_id"`
	SampleName    string  `json:"sample_name"`
	TemplateID    string  `json:"template_id"`
	TemplateName  string  `json:"template_name"`
	Score         float64 `json:"score"`
	Reason        string  `json:"reason"`
	SampleSubType string  `json:"sample_sub_type"`
	TemplateType  string  `json:"template_sub_type"`
}

type composedAttackRecommendation struct {
	ComposedAttackID   string  `json:"composed_attack_id"`
	ComposedAttackName string  `json:"composed_attack_name"`
	Score              float64 `json:"score"`
	Reason             string  `json:"reason"`
	SubType            string  `json:"sub_type"`
}

var whitespaceRE = regexp.MustCompile(`\s+`)
var placeholderRE = regexp.MustCompile(`\{\{[^}]+\}\}`)

func buildSampleCard(sample model.AttackSample, preview []model.AttackPayload) resourceCard {
	assessmentTypes := sampleAssessmentTypes(sample.SubType)
	tags := uniqueStrings(append(sampleTags(sample.SubType), keywordTags(sample.Name, sample.Description)...))
	examples := make([]string, 0, len(preview))
	for _, item := range preview {
		examples = append(examples, sanitizeSnippet(item.Data, 96))
	}

	return resourceCard{
		ID:                        sample.ID.String(),
		ExpertID:                  sample.ExpertID.String(),
		ResourceKind:              "sample",
		Name:                      sample.Name,
		SubType:                   sample.SubType,
		Description:               sample.Description,
		Status:                    sample.Status,
		PlannerSummary:            samplePlannerSummary(sample, assessmentTypes, tags),
		ScenarioTags:              tags,
		TargetTypes:               []string{"openai", "custom", "agent"},
		ApplicableAssessmentTypes: assessmentTypes,
		LanguageSupport:           []string{"zh", "multilingual"},
		AttackStyle:               sampleAttackStyle(sample.SubType),
		Difficulty:                sampleDifficulty(sample.SubType),
		ExpectedSignal:            sampleExpectedSignal(sample.SubType),
		SampleCount:               sample.SampleCount,
		RecommendedPairings:       recommendedTemplateTypesForSample(sample.SubType),
		SanitizedExamples:         examples,
		EstimatedDurationSecs:     estimateDuration(sample.SampleCount, 1),
		EstimatedCost:             estimateCost(sample.SampleCount, 1),
	}
}

func buildTemplateCard(tpl model.Template) resourceCard {
	assessmentTypes := templateAssessmentTypes(tpl.SubType)
	tags := uniqueStrings(append(templateTags(tpl.SubType), keywordTags(tpl.Name, tpl.Description, tpl.Content)...))

	return resourceCard{
		ID:                        tpl.ID.String(),
		ExpertID:                  tpl.ExpertID.String(),
		ResourceKind:              "template",
		Name:                      tpl.Name,
		SubType:                   tpl.SubType,
		Description:               tpl.Description,
		Status:                    tpl.Status,
		PlannerSummary:            templatePlannerSummary(tpl, assessmentTypes, tags),
		ScenarioTags:              tags,
		TargetTypes:               []string{"openai", "custom", "agent"},
		ApplicableAssessmentTypes: assessmentTypes,
		LanguageSupport:           templateLanguageSupport(tpl.SubType),
		AttackStyle:               templateAttackStyle(tpl.SubType),
		Difficulty:                templateDifficulty(tpl.SubType),
		ExpectedSignal:            templateExpectedSignal(tpl.SubType),
		Variables:                 templateVariables(tpl),
		RecommendedPairings:       recommendedSampleTypesForTemplate(tpl.SubType),
		EstimatedDurationSecs:     estimateDuration(5, templateExpansionFactor(tpl.SubType)),
		EstimatedCost:             estimateCost(5, templateExpansionFactor(tpl.SubType)),
	}
}

func buildComposedAttackCard(item model.ComposedAttack, preview []model.AttackPayload) resourceCard {
	assessmentTypes := sampleAssessmentTypes(item.SubType)
	tags := uniqueStrings(append(sampleTags(item.SubType), []string{"已组合攻击", "可直接执行"}...))
	examples := make([]string, 0, len(preview))
	for _, payload := range preview {
		examples = append(examples, sanitizeSnippet(payload.Data, 96))
	}

	return resourceCard{
		ID:                        item.ID.String(),
		ExpertID:                  item.ExpertID.String(),
		ResourceKind:              "composed_attack",
		AlreadyComposed:           true,
		Name:                      item.Name,
		SubType:                   item.SubType,
		Description:               item.Description,
		Status:                    item.Status,
		PlannerSummary:            composedAttackPlannerSummary(item, assessmentTypes, tags),
		ScenarioTags:              tags,
		TargetTypes:               []string{"openai", "custom", "agent"},
		ApplicableAssessmentTypes: assessmentTypes,
		LanguageSupport:           []string{"zh", "multilingual"},
		AttackStyle:               "precomposed_payload",
		Difficulty:                sampleDifficulty(item.SubType),
		ExpectedSignal:            "该资源已经完成模板拼接，可直接执行；命中后应跳过 combine_template_sample。",
		SampleCount:               item.SampleCount,
		SanitizedExamples:         examples,
		EstimatedDurationSecs:     estimateDuration(item.SampleCount, 1),
		EstimatedCost:             estimateCost(item.SampleCount, 1),
	}
}

func composedAttackPlannerSummary(item model.ComposedAttack, assessmentTypes, tags []string) string {
	return fmt.Sprintf(
		"该已组合攻击集属于 %s，包含 %d 条可直接执行的最终攻击载荷，主要覆盖 %s 场景。适合 %s 类评估，命中后可直接执行并跳过模板拼接。",
		item.SubType,
		item.SampleCount,
		strings.Join(tags, "、"),
		strings.Join(assessmentTypes, " / "),
	)
}

func samplePlannerSummary(sample model.AttackSample, assessmentTypes, tags []string) string {
	return fmt.Sprintf(
		"该样本集属于 %s，包含 %d 条测试载荷，主要覆盖 %s 场景。适合 %s 类评估，用于验证模型是否会输出违规或危险响应。",
		sample.SubType,
		sample.SampleCount,
		strings.Join(tags, "、"),
		strings.Join(assessmentTypes, " / "),
	)
}

func templatePlannerSummary(tpl model.Template, assessmentTypes, tags []string) string {
	variableHint := "无显式变量"
	if vars := templateVariables(tpl); len(vars) > 0 {
		variableHint = "变量：" + strings.Join(vars, "、")
	}
	return fmt.Sprintf(
		"该模板属于 %s，适合 %s 类评估。它强调 %s 风格，%s。编排层可据此判断其是否适合与目标样本组合。",
		tpl.SubType,
		strings.Join(assessmentTypes, " / "),
		strings.Join(tags, "、"),
		variableHint,
	)
}

func sampleAssessmentTypes(subType string) []string {
	switch subType {
	case string(model.SubTypeComplianceDetection):
		return []string{"compliance_check"}
	case string(model.SubTypeMaliciousPoisoning):
		return []string{"tool_poisoning", "goal_hijacking"}
	case string(model.SubTypeDirectInjection):
		return []string{"prompt_injection", "jailbreak"}
	case string(model.SubTypeMaliciousInstruction):
		return []string{"prompt_injection", "jailbreak", "goal_hijacking"}
	default:
		return []string{"prompt_injection"}
	}
}

func templateAssessmentTypes(subType string) []string {
	switch subType {
	case string(model.SubTypeRolePlay):
		return []string{"compliance_check", "prompt_injection", "jailbreak", "goal_hijacking", "tool_poisoning"}
	case string(model.SubTypeMultilingual):
		return []string{"compliance_check", "prompt_injection", "jailbreak"}
	case string(model.SubTypeEncodingEvasion):
		return []string{"prompt_injection", "jailbreak", "tool_poisoning"}
	default:
		return []string{"prompt_injection"}
	}
}

func sampleTags(subType string) []string {
	switch subType {
	case string(model.SubTypeComplianceDetection):
		return []string{"合规", "违法", "色情", "暴力", "歧视", "隐私泄露"}
	case string(model.SubTypeMaliciousPoisoning):
		return []string{"工具投毒", "恶意指令", "目标漂移"}
	case string(model.SubTypeDirectInjection):
		return []string{"直接注入", "越狱", "系统提示词干扰"}
	case string(model.SubTypeMaliciousInstruction):
		return []string{"恶意指令", "提示词注入", "绕过约束"}
	default:
		return []string{"安全评测"}
	}
}

func templateTags(subType string) []string {
	switch subType {
	case string(model.SubTypeRolePlay):
		return []string{"角色扮演", "情境诱导", "叙事包装"}
	case string(model.SubTypeMultilingual):
		return []string{"多语言", "跨语言迁移", "翻译变体"}
	case string(model.SubTypeEncodingEvasion):
		return []string{"编码规避", "格式变换", "绕过过滤"}
	default:
		return []string{"模板"}
	}
}

func sampleAttackStyle(subType string) string {
	switch subType {
	case string(model.SubTypeComplianceDetection):
		return "policy_probe"
	case string(model.SubTypeMaliciousPoisoning):
		return "tool_poisoning"
	case string(model.SubTypeDirectInjection):
		return "direct"
	case string(model.SubTypeMaliciousInstruction):
		return "indirect"
	default:
		return "generic"
	}
}

func templateAttackStyle(subType string) string {
	switch subType {
	case string(model.SubTypeRolePlay):
		return "role_play"
	case string(model.SubTypeMultilingual):
		return "multilingual"
	case string(model.SubTypeEncodingEvasion):
		return "encoding_evasion"
	default:
		return "generic"
	}
}

func sampleDifficulty(subType string) string {
	switch subType {
	case string(model.SubTypeComplianceDetection):
		return "medium"
	case string(model.SubTypeMaliciousPoisoning):
		return "high"
	default:
		return "medium"
	}
}

func templateDifficulty(subType string) string {
	switch subType {
	case string(model.SubTypeEncodingEvasion):
		return "high"
	case string(model.SubTypeMultilingual):
		return "medium"
	default:
		return "medium"
	}
}

func sampleExpectedSignal(subType string) string {
	switch subType {
	case string(model.SubTypeComplianceDetection):
		return "理想情况下模型应拒绝、规避或安全改写违规请求。"
	case string(model.SubTypeMaliciousPoisoning):
		return "理想情况下模型应避免执行恶意工具指令或暴露系统目标。"
	default:
		return "理想情况下模型应拒绝危险请求，避免泄露或绕过安全约束。"
	}
}

func templateExpectedSignal(subType string) string {
	switch subType {
	case string(model.SubTypeRolePlay):
		return "用于观察模型在情境诱导下是否放松安全边界。"
	case string(model.SubTypeMultilingual):
		return "用于观察模型在跨语言条件下是否仍能维持安全策略。"
	case string(model.SubTypeEncodingEvasion):
		return "用于观察模型在格式变换或编码规避后是否仍能识别风险。"
	default:
		return "用于观察模型在模板包装下的安全策略稳定性。"
	}
}

func templateLanguageSupport(subType string) []string {
	switch subType {
	case string(model.SubTypeMultilingual):
		return []string{"zh", "en", "ja", "ko", "fr", "de"}
	default:
		return []string{"zh"}
	}
}

func recommendedTemplateTypesForSample(subType string) []string {
	switch subType {
	case string(model.SubTypeComplianceDetection):
		return []string{string(model.SubTypeRolePlay), string(model.SubTypeMultilingual)}
	case string(model.SubTypeMaliciousPoisoning):
		return []string{string(model.SubTypeEncodingEvasion), string(model.SubTypeRolePlay)}
	default:
		return []string{string(model.SubTypeRolePlay), string(model.SubTypeEncodingEvasion)}
	}
}

func recommendedSampleTypesForTemplate(subType string) []string {
	switch subType {
	case string(model.SubTypeMultilingual):
		return []string{string(model.SubTypeComplianceDetection), string(model.SubTypeMaliciousInstruction)}
	case string(model.SubTypeEncodingEvasion):
		return []string{string(model.SubTypeDirectInjection), string(model.SubTypeMaliciousPoisoning)}
	default:
		return []string{string(model.SubTypeComplianceDetection), string(model.SubTypeMaliciousInstruction), string(model.SubTypeDirectInjection)}
	}
}

func templateVariables(tpl model.Template) []string {
	if len(tpl.Variables) > 0 {
		out := make([]string, 0, len(tpl.Variables))
		for _, variable := range tpl.Variables {
			if variable.Name != "" {
				out = append(out, variable.Name)
			}
		}
		return uniqueStrings(out)
	}

	matches := placeholderRE.FindAllString(tpl.Content, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
	}
	return uniqueStrings(out)
}

func keywordTags(parts ...string) []string {
	text := strings.ToLower(strings.Join(parts, " "))
	keywordMap := map[string]string{
		"合规":         "合规",
		"compliance": "合规",
		"违法":         "违法",
		"色情":         "色情",
		"porn":       "色情",
		"暴力":         "暴力",
		"violence":   "暴力",
		"隐私":         "隐私泄露",
		"privacy":    "隐私泄露",
		"歧视":         "歧视",
		"hate":       "歧视",
		"注入":         "提示词注入",
		"injection":  "提示词注入",
		"越狱":         "越狱",
		"jailbreak":  "越狱",
		"role":       "角色扮演",
		"角色":         "角色扮演",
		"multi":      "多语言",
		"语言":         "多语言",
		"编码":         "编码规避",
		"encode":     "编码规避",
		"poison":     "工具投毒",
		"投毒":         "工具投毒",
	}

	tags := make([]string, 0, 6)
	for keyword, tag := range keywordMap {
		if strings.Contains(text, keyword) {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	return uniqueStrings(tags)
}

func sanitizeSnippet(text string, maxLen int) string {
	text = whitespaceRE.ReplaceAllString(strings.TrimSpace(text), " ")
	replacements := []struct {
		old string
		new string
	}{
		{"http://", "[url]"},
		{"https://", "[url]"},
		{"sk-", "[key]-"},
	}
	for _, item := range replacements {
		text = strings.ReplaceAll(text, item.old, item.new)
	}
	if len([]rune(text)) <= maxLen {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxLen]) + "..."
}

func estimateDuration(sampleCount, expansionFactor int) int {
	if sampleCount <= 0 {
		sampleCount = 5
	}
	if expansionFactor <= 0 {
		expansionFactor = 1
	}
	return sampleCount * expansionFactor * 6
}

func estimateCost(sampleCount, expansionFactor int) string {
	if sampleCount <= 0 {
		sampleCount = 5
	}
	if expansionFactor <= 0 {
		expansionFactor = 1
	}
	return fmt.Sprintf("low-%d calls", sampleCount*expansionFactor)
}

func templateExpansionFactor(subType string) int {
	switch subType {
	case string(model.SubTypeMultilingual):
		return 3
	case string(model.SubTypeEncodingEvasion):
		return 2
	default:
		return 1
	}
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func parseAssessmentTypes(value interface{}) []string {
	switch typed := value.(type) {
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return uniqueStrings(result)
	case []string:
		return uniqueStrings(typed)
	default:
		return nil
	}
}

func inferAssessmentTypesFromGoal(goal string) []string {
	goal = strings.ToLower(goal)
	types := make([]string, 0, 3)
	switch {
	case strings.Contains(goal, "合规") || strings.Contains(goal, "违法") || strings.Contains(goal, "色情") || strings.Contains(goal, "暴力"):
		types = append(types, "compliance_check")
	case strings.Contains(goal, "投毒"):
		types = append(types, "tool_poisoning")
	case strings.Contains(goal, "劫持"):
		types = append(types, "goal_hijacking")
	}
	if strings.Contains(goal, "越狱") {
		types = append(types, "jailbreak")
	}
	if strings.Contains(goal, "注入") {
		types = append(types, "prompt_injection")
	}
	if len(types) == 0 {
		types = append(types, "prompt_injection")
	}
	return uniqueStrings(types)
}
