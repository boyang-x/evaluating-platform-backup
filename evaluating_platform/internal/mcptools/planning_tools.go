package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/internal/sample"
)

func registerRecommendResources(
	s *server.MCPServer,
	sampleRepo *repository.AttackSampleRepository,
	composedRepo *repository.ComposedAttackRepository,
	tplRepo *repository.TemplateRepository,
) {
	tool := mcp.NewTool("recommend_resources",
		mcp.WithDescription("根据评估目标和评估类型，推荐合适的样本+模板组合，以及可直接执行的已组合攻击。"),
		mcp.WithString("goal", mcp.Required(), mcp.Description("评估目标描述")),
		mcp.WithArray("assessment_types", mcp.Description("评估类型数组，如 [\"compliance_check\"]")),
		mcp.WithString("target_type", mcp.Description("目标系统类型，可选：openai | agent | custom")),
		mcp.WithNumber("max_results", mcp.Description("最多返回多少条推荐结果，默认 5")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		goal, _ := req.GetArguments()["goal"].(string)
		if strings.TrimSpace(goal) == "" {
			return mcp.NewToolResultError("goal is required"), nil
		}

		assessmentTypes := parseAssessmentTypes(req.GetArguments()["assessment_types"])
		if len(assessmentTypes) == 0 {
			assessmentTypes = inferAssessmentTypesFromGoal(goal)
		}
		targetType, _ := req.GetArguments()["target_type"].(string)
		maxResults := 5
		if rawLimit, ok := req.GetArguments()["max_results"].(float64); ok && rawLimit > 0 {
			maxResults = int(rawLimit)
		}

		samples, _, err := sampleRepo.ListPublished(ctx, "", 100, 0)
		if err != nil {
			return mcp.NewToolResultError("query samples failed: " + err.Error()), nil
		}
		templates, _, err := tplRepo.ListPublished(ctx, "", 100, 0)
		if err != nil {
			return mcp.NewToolResultError("query templates failed: " + err.Error()), nil
		}

		composedItems := make([]model.ComposedAttack, 0)
		if composedRepo != nil {
			composedItems, _, err = composedRepo.ListPublished(ctx, "", 100, 0)
			if err != nil {
				return mcp.NewToolResultError("query composed attacks failed: " + err.Error()), nil
			}
		}

		sampleCards := make([]resourceCard, 0, len(samples))
		for _, item := range samples {
			sampleCards = append(sampleCards, buildSampleCard(item, nil))
		}

		templateCards := make([]resourceCard, 0, len(templates))
		for _, item := range templates {
			templateCards = append(templateCards, buildTemplateCard(item))
		}

		composedCards := make([]resourceCard, 0, len(composedItems))
		for _, item := range composedItems {
			composedCards = append(composedCards, buildComposedAttackCard(item, nil))
		}

		pairs, composedRecommendations := buildRecommendations(goal, assessmentTypes, targetType, sampleCards, templateCards, composedCards, maxResults)
		result := map[string]interface{}{
			"assessment_types":             assessmentTypes,
			"sample_candidates":            sampleCards,
			"template_candidates":          templateCards,
			"composed_attack_candidates":   composedCards,
			"recommended_pairs":            pairs,
			"recommended_composed_attacks": composedRecommendations,
			"selection_guidance":           "优先从 recommended_pairs 或 recommended_composed_attacks 中选择真实资源 ID；若命中已组合攻击，应直接加载并执行，跳过模板拼接。",
		}

		b, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(b)), nil
	})
}

func registerPreviewAttackSample(s *server.MCPServer, sampleRepo *repository.AttackSampleRepository, loader *sample.Loader) {
	tool := mcp.NewTool("preview_attack_sample",
		mcp.WithDescription("返回样本的资源画像和少量脱敏示例，帮助编排 LLM 选择资源而不暴露完整内容"),
		mcp.WithString("sample_id", mcp.Required(), mcp.Description("攻击样本 ID")),
		mcp.WithNumber("preview_count", mcp.Description("预览条数，默认 3")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sampleIDStr, _ := req.GetArguments()["sample_id"].(string)
		sampleID, err := uuid.Parse(sampleIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid sample_id"), nil
		}

		previewCount := 3
		if rawLimit, ok := req.GetArguments()["preview_count"].(float64); ok && rawLimit > 0 {
			previewCount = int(rawLimit)
		}

		item, err := sampleRepo.GetByID(ctx, sampleID)
		if err != nil {
			return mcp.NewToolResultError("sample not found: " + sampleIDStr), nil
		}

		var preview []model.AttackPayload
		if loader != nil {
			preview, _ = loader.Preview(ctx, sampleID, previewCount)
		}

		card := buildSampleCard(*item, preview)
		result := map[string]interface{}{
			"sample":        card,
			"preview_count": len(card.SanitizedExamples),
		}
		b, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(b)), nil
	})
}

func registerPreviewTemplate(s *server.MCPServer, tplRepo *repository.TemplateRepository) {
	tool := mcp.NewTool("preview_template",
		mcp.WithDescription("返回模板的资源画像、变量和结构摘要，帮助编排 LLM 选择模板而不暴露完整内容"),
		mcp.WithString("template_id", mcp.Required(), mcp.Description("模板 ID")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templateIDStr, _ := req.GetArguments()["template_id"].(string)
		templateID, err := uuid.Parse(templateIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid template_id"), nil
		}

		item, err := tplRepo.GetByID(ctx, templateID)
		if err != nil {
			return mcp.NewToolResultError("template not found: " + templateIDStr), nil
		}

		card := buildTemplateCard(*item)
		result := map[string]interface{}{
			"template":          card,
			"structure_outline": buildTemplateOutline(*item),
		}
		b, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(b)), nil
	})
}

func buildRecommendations(
	goal string,
	assessmentTypes []string,
	targetType string,
	samples []resourceCard,
	templates []resourceCard,
	composedAttacks []resourceCard,
	maxResults int,
) ([]recommendationResult, []composedAttackRecommendation) {
	type scoredCard struct {
		card  resourceCard
		score float64
	}

	scoredSamples := scoreCards(goal, assessmentTypes, targetType, samples)
	scoredTemplates := scoreCards(goal, assessmentTypes, targetType, templates)
	scoredComposed := scoreCards(goal, assessmentTypes, targetType, composedAttacks)

	limitSamples := minInt(len(scoredSamples), 3)
	limitTemplates := minInt(len(scoredTemplates), 3)
	pairs := make([]recommendationResult, 0, maxResults)

	for i := 0; i < limitSamples; i++ {
		for j := 0; j < limitTemplates; j++ {
			sampleCard := scoredSamples[i]
			templateCard := scoredTemplates[j]
			score := (sampleCard.score + templateCard.score) / 2
			if containsString(sampleCard.card.RecommendedPairings, templateCard.card.SubType) {
				score += 0.12
			}
			pairs = append(pairs, recommendationResult{
				SampleID:      sampleCard.card.ID,
				SampleName:    sampleCard.card.Name,
				TemplateID:    templateCard.card.ID,
				TemplateName:  templateCard.card.Name,
				Score:         roundScore(score),
				Reason:        recommendationReason(goal, assessmentTypes, sampleCard.card, templateCard.card),
				SampleSubType: sampleCard.card.SubType,
				TemplateType:  templateCard.card.SubType,
			})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Score > pairs[j].Score })
	if len(pairs) > maxResults {
		pairs = pairs[:maxResults]
	}

	limitComposed := minInt(len(scoredComposed), maxResults)
	composedRecommendations := make([]composedAttackRecommendation, 0, limitComposed)
	for i := 0; i < limitComposed; i++ {
		card := scoredComposed[i].card
		composedRecommendations = append(composedRecommendations, composedAttackRecommendation{
			ComposedAttackID:   card.ID,
			ComposedAttackName: card.Name,
			Score:              roundScore(scoredComposed[i].score + 0.08),
			Reason:             directAttackReason(goal, assessmentTypes, card),
			SubType:            card.SubType,
		})
	}

	return pairs, composedRecommendations
}

func scoreCards(goal string, assessmentTypes []string, targetType string, cards []resourceCard) []struct {
	card  resourceCard
	score float64
} {
	scored := make([]struct {
		card  resourceCard
		score float64
	}, 0, len(cards))
	for _, card := range cards {
		scored = append(scored, struct {
			card  resourceCard
			score float64
		}{
			card:  card,
			score: resourceScore(goal, assessmentTypes, targetType, card),
		})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	return scored
}

func resourceScore(goal string, assessmentTypes []string, targetType string, card resourceCard) float64 {
	score := 0.15
	goalTags := keywordTags(goal)
	if overlapCount(goalTags, card.ScenarioTags) > 0 {
		score += 0.25
	}
	if overlapCount(assessmentTypes, card.ApplicableAssessmentTypes) > 0 {
		score += 0.4
	}
	if targetType == "" || containsString(card.TargetTypes, targetType) {
		score += 0.1
	}
	switch card.Difficulty {
	case "high":
		score += 0.05
	case "medium":
		score += 0.03
	}
	if card.SampleCount > 0 {
		score += 0.05
	}
	if card.AlreadyComposed {
		score += 0.06
	}
	return score
}

func recommendationReason(goal string, assessmentTypes []string, sample resourceCard, tpl resourceCard) string {
	reasons := []string{
		fmt.Sprintf("样本 `%s` 覆盖 %s", sample.Name, strings.Join(sample.ScenarioTags, "、")),
		fmt.Sprintf("模板 `%s` 提供 %s 包装方式", tpl.Name, tpl.AttackStyle),
	}
	if overlapCount(assessmentTypes, sample.ApplicableAssessmentTypes) > 0 {
		reasons = append(reasons, "样本与评估类型高度匹配")
	}
	if overlapCount(keywordTags(goal), tpl.ScenarioTags) > 0 {
		reasons = append(reasons, "模板风格与目标描述中的关键词相符")
	}
	return strings.Join(reasons, "；")
}

func directAttackReason(goal string, assessmentTypes []string, card resourceCard) string {
	reasons := []string{
		fmt.Sprintf("已组合攻击 `%s` 已包含最终载荷，可直接执行", card.Name),
		fmt.Sprintf("覆盖 %s", strings.Join(card.ScenarioTags, "、")),
	}
	if overlapCount(assessmentTypes, card.ApplicableAssessmentTypes) > 0 {
		reasons = append(reasons, "与当前评估类型匹配")
	}
	if overlapCount(keywordTags(goal), card.ScenarioTags) > 0 {
		reasons = append(reasons, "与目标描述关键词相符")
	}
	return strings.Join(reasons, "；")
}

func buildTemplateOutline(tpl model.Template) map[string]interface{} {
	outline := []string{"角色/场景设定", "输出风格与约束", "载荷插入位"}
	if len(templateVariables(tpl)) > 0 {
		outline = append(outline, "变量绑定")
	}
	return map[string]interface{}{
		"sections":       outline,
		"variables":      templateVariables(tpl),
		"content_digest": sanitizeSnippet(tpl.Content, 120),
	}
}

func overlapCount(a, b []string) int {
	count := 0
	for _, item := range uniqueStrings(a) {
		if containsString(b, item) {
			count++
		}
	}
	return count
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func roundScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		score = 1
	}
	return float64(int(score*100)) / 100
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
