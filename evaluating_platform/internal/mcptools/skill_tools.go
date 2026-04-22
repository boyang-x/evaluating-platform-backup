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
	"evaluating_platform/internal/sample"
	skillpkg "evaluating_platform/internal/skill"
)

type skillRecommendation struct {
	SkillID           string  `json:"skill_id"`
	SkillName         string  `json:"skill_name"`
	Version           string  `json:"version"`
	Score             float64 `json:"score"`
	Reason            string  `json:"reason"`
	CapabilityProfile string  `json:"capability_profile"`
	InputSourceMode   string  `json:"input_source_mode"`
}

func buildSkillCard(bundle skillpkg.SkillBundle) resourceCard {
	name := ""
	description := ""
	expertID := ""
	status := ""
	if bundle.Skill != nil {
		name = bundle.Skill.Name
		description = bundle.Skill.Description
		expertID = bundle.Skill.ExpertID.String()
		status = string(bundle.Skill.Status)
	}
	if bundle.Version != nil {
		if strings.TrimSpace(bundle.Version.DisplayName) != "" {
			name = bundle.Version.DisplayName
		}
		if strings.TrimSpace(bundle.Version.Summary) != "" {
			description = bundle.Version.Summary
		}
	}
	tags := uniqueStrings(append(bundle.Manifest.AssessmentTypes, keywordTags(name, description, bundle.Manifest.Category, bundle.Manifest.CapabilityProfile)...))
	plannerSummary := strings.TrimSpace(description)
	if plannerSummary == "" {
		plannerSummary = fmt.Sprintf("已发布 generator skill，可直接生成攻击数据集并输出平台标准 payload_dataset，能力画像：%s。", bundle.Manifest.CapabilityProfile)
	}
	return resourceCard{
		ID:                        bundle.Skill.ID.String(),
		ExpertID:                  expertID,
		ResourceKind:              "skill",
		Name:                      name,
		SubType:                   bundle.Manifest.SkillType,
		Description:               description,
		Status:                    status,
		PlannerSummary:            plannerSummary,
		ScenarioTags:              tags,
		TargetTypes:               []string{"openai", "custom", "agent"},
		ApplicableAssessmentTypes: uniqueStrings(bundle.Manifest.AssessmentTypes),
		LanguageSupport:           []string{"zh", "multilingual"},
		AttackStyle:               firstNonEmptyString(bundle.Manifest.CapabilityProfile, "generator_skill"),
		Difficulty:                "high",
		ExpectedSignal:            fmt.Sprintf("该 skill 将通过独立沙箱生成 payload_dataset，输入来源模式为 %s。", bundle.Manifest.InputSourceMode),
		InputSourceMode:           bundle.Manifest.InputSourceMode,
		EstimatedDurationSecs:     90,
		EstimatedCost:             "skill_runner",
	}
}

func buildSkillRecommendations(goal string, assessmentTypes []string, cards []resourceCard, bundles map[string]skillpkg.SkillBundle, limit int) []skillRecommendation {
	scored := scoreCards(goal, assessmentTypes, "", cards)
	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	recommendations := make([]skillRecommendation, 0, len(scored))
	for _, item := range scored {
		bundle, ok := bundles[item.card.ID]
		if !ok || bundle.Version == nil {
			continue
		}
		reason := item.card.PlannerSummary
		if reason == "" {
			reason = fmt.Sprintf("skill `%s` 与目标和评估类型匹配。", item.card.Name)
		}
		recommendations = append(recommendations, skillRecommendation{
			SkillID:           item.card.ID,
			SkillName:         item.card.Name,
			Version:           bundle.Version.Version,
			Score:             roundScore(item.score + 0.12),
			Reason:            reason,
			CapabilityProfile: bundle.Manifest.CapabilityProfile,
			InputSourceMode:   bundle.Manifest.InputSourceMode,
		})
	}
	sort.Slice(recommendations, func(i, j int) bool { return recommendations[i].Score > recommendations[j].Score })
	return recommendations
}

func registerPreviewSkill(s *server.MCPServer, skillService *skillpkg.Service) {
	if skillService == nil {
		return
	}

	tool := mcp.NewTool("preview_skill",
		mcp.WithDescription("返回已发布 skill 的非敏感摘要，包括版本、能力画像、输入来源和嵌入数据集摘要。"),
		mcp.WithString("skill_id", mcp.Required(), mcp.Description("skill ID")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		skillIDStr, _ := req.GetArguments()["skill_id"].(string)
		skillID, err := uuid.Parse(skillIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid skill_id"), nil
		}

		bundle, err := skillService.GetPublishedByID(ctx, skillID)
		if err != nil || bundle == nil || bundle.Skill == nil || bundle.Version == nil {
			return mcp.NewToolResultError("published skill not found"), nil
		}
		if bundle.Skill.SkillType != "generator_skill" {
			return mcp.NewToolResultError("preview_skill only supports published generator_skill"), nil
		}
		card := buildSkillCard(*bundle)
		result := map[string]interface{}{
			"skill": map[string]interface{}{
				"id":                          card.ID,
				"name":                        card.Name,
				"description":                 card.Description,
				"skill_type":                  bundle.Manifest.SkillType,
				"version":                     bundle.Version.Version,
				"capability_profile":          bundle.Manifest.CapabilityProfile,
				"input_source_mode":           bundle.Manifest.InputSourceMode,
				"planner_summary":             card.PlannerSummary,
				"permissions":                 json.RawMessage(bundle.Version.Permissions),
				"embedded_dataset_summary":    json.RawMessage(bundle.Version.EmbeddedDatasetSummary),
				"applicable_assessment_types": bundle.Manifest.AssessmentTypes,
				"redaction_notice":            "完整 payload 正文不会返回给编排层，只返回摘要和统计信息。",
			},
		}
		b, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(b)), nil
	})
}

const defaultSkillSourceSampleLimit = 20

func registerRunGeneratorSkill(s *server.MCPServer, skillService *skillpkg.Service, loader *sample.Loader, store *SessionStore) {
	if skillService == nil || store == nil {
		return
	}

	tool := mcp.NewTool("run_generator_skill",
		mcp.WithDescription("运行已发布 generator skill，在独立 skill_runner 中生成平台标准 payload_dataset，并写入当前 session。"),
		mcp.WithString("skill_id", mcp.Required(), mcp.Description("skill ID")),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("session / assessment ID")),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("enterprise user ID")),
		mcp.WithString("sample_id", mcp.Description("optional expert sample ID for platform_resource_only skills")),
		mcp.WithString("goal", mcp.Description("assessment goal")),
		mcp.WithArray("assessment_types", mcp.Description("assessment types")),
		mcp.WithNumber("requested_count", mcp.Description("desired payload count")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		skillIDStr, _ := req.GetArguments()["skill_id"].(string)
		sessionID, _ := req.GetArguments()["session_id"].(string)
		userID, _ := req.GetArguments()["user_id"].(string)
		sampleIDStr, _ := req.GetArguments()["sample_id"].(string)
		goal, _ := req.GetArguments()["goal"].(string)
		if skillIDStr == "" || sessionID == "" || userID == "" {
			return mcp.NewToolResultError("skill_id, session_id and user_id are required"), nil
		}
		skillID, err := uuid.Parse(skillIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid skill_id"), nil
		}

		requestedCount := 0
		if raw, ok := req.GetArguments()["requested_count"].(float64); ok && raw > 0 {
			requestedCount = int(raw)
		}
		assessmentTypes := parseAssessmentTypes(req.GetArguments()["assessment_types"])

		bundle, err := skillService.GetPublishedByID(ctx, skillID)
		if err != nil || bundle == nil || bundle.Skill == nil || bundle.Version == nil {
			return mcp.NewToolResultError("published skill not found"), nil
		}
		if bundle.Skill.SkillType != "generator_skill" {
			return mcp.NewToolResultError("run_generator_skill only supports published generator_skill"), nil
		}

		sourceSamples := make([]skillpkg.SourceSampleItem, 0)
		if strings.TrimSpace(sampleIDStr) != "" {
			if loader == nil {
				return mcp.NewToolResultError("sample loader is unavailable"), nil
			}
			sampleID, parseErr := uuid.Parse(sampleIDStr)
			if parseErr != nil {
				return mcp.NewToolResultError("invalid sample_id"), nil
			}
			loaded, loadErr := loader.LoadSamples(ctx, sampleID)
			if loadErr != nil {
				return mcp.NewToolResultError("load sample failed: " + loadErr.Error()), nil
			}
			defer loaded.Close()
			sourceSamples = buildSkillSourceSamples(loaded.Payloads, skillSourceSampleLimit(requestedCount))
			if len(sourceSamples) == 0 {
				return mcp.NewToolResultError("sample has no usable items"), nil
			}
		}
		if bundle.Version.InputSourceMode == "platform_resource_only" && len(sourceSamples) == 0 {
			return mcp.NewToolResultError("this skill requires sample_id because it consumes platform samples"), nil
		}

		generated, run, err := skillService.GeneratePayloads(ctx, skillID, skillpkg.GenerateRequest{
			AssessmentID:    sessionID,
			UserID:          userID,
			Goal:            goal,
			AssessmentTypes: assessmentTypes,
			RequestedCount:  requestedCount,
			SourceSampleID:  strings.TrimSpace(sampleIDStr),
			SourceSamples:   sourceSamples,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		payloads := adaptSkillPayloads(generated.PayloadDataset.Payloads)
		store.SetPayloads(sessionID, payloads)

		skillName := skillIDStr
		version := ""
		if bundle != nil && bundle.Skill != nil {
			skillName = bundle.Skill.Name
		}
		if bundle != nil && bundle.Version != nil {
			version = bundle.Version.Version
		}

		result := map[string]interface{}{
			"session_id":        sessionID,
			"skill_id":          skillIDStr,
			"skill_name":        skillName,
			"version":           version,
			"run_id":            run.ID.String(),
			"generated_count":   len(payloads),
			"dataset_summary":   generated.PayloadDataset.Summary,
			"input_source_mode": bundle.Version.InputSourceMode,
			"source_sample_id":  strings.TrimSpace(sampleIDStr),
		}
		b, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(b)), nil
	})
}

func buildSkillSourceSamples(items []model.AttackPayload, limit int) []skillpkg.SourceSampleItem {
	sourceSamples := make([]skillpkg.SourceSampleItem, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item.Data)
		if text == "" {
			continue
		}
		sourceSamples = append(sourceSamples, skillpkg.SourceSampleItem{
			Index: item.Index,
			Text:  text,
		})
		if limit > 0 && len(sourceSamples) >= limit {
			break
		}
	}
	return sourceSamples
}

func skillSourceSampleLimit(requestedCount int) int {
	if requestedCount > 0 {
		return requestedCount
	}
	return defaultSkillSourceSampleLimit
}

func adaptSkillPayloads(items []skillpkg.PayloadItem) []PayloadItem {
	payloads := make([]PayloadItem, 0, len(items))
	for idx, item := range items {
		text := strings.TrimSpace(item.PayloadText)
		if text == "" {
			continue
		}
		index := idx + 1
		payloads = append(payloads, PayloadItem{
			Index:           index,
			OriginalContent: strings.TrimSpace(item.OriginalQuestion),
			CombinedText:    text,
			Language:        firstNonEmptyString(strings.TrimSpace(item.Language), "zh"),
			Sensitive:       true,
			QuestionSummary: firstNonEmptyString(strings.TrimSpace(item.QuestionSummary), buildSkillQuestionSummary(index)),
			PayloadSummary:  firstNonEmptyString(strings.TrimSpace(item.PayloadSummary), buildSkillPayloadSummary(index, text)),
		})
	}
	return payloads
}

func buildSkillQuestionSummary(index int) string {
	return fmt.Sprintf("Skill generated question #%d (redacted)", index)
}

func buildSkillPayloadSummary(index int, payload string) string {
	return fmt.Sprintf("Skill generated payload #%d (%d chars)", index, len([]rune(strings.TrimSpace(payload))))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
