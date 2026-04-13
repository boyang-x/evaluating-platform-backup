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

const (
	defaultCCBOSNamespace       = "ccbos"
	ccbosGenerateToolName       = "generate_classical_chinese_payloads"
	ccbosPreviewDatasetToolName = "preview_payload_dataset"

	defaultCCBOSOptimizationMode = "single_pass"
	optimizedCCBOSMode           = "fruitfly"
	defaultCCBOSPopulationSize   = 1
	defaultCCBOSMaxIterations    = 1
	defaultCCBOSEarlyStopScore   = 20.0
)

type externalToolCaller interface {
	CallTool(ctx context.Context, serverID uuid.UUID, remoteToolName string, args map[string]interface{}) (*mcp.CallToolResult, error)
}

type ccbosDatasetCreateResponse struct {
	Status       string `json:"status"`
	DatasetID    string `json:"dataset_id"`
	Count        int    `json:"count"`
	Optimization struct {
		ModeRequested string  `json:"mode_requested"`
		ModeUsed      string  `json:"mode_used"`
		Attempts      int     `json:"attempts"`
		BestScore     float64 `json:"best_score"`
		TargetEnabled bool    `json:"target_enabled"`
	} `json:"optimization"`
}

type ccbosDatasetPreviewResponse struct {
	Status  string                    `json:"status"`
	Count   int                       `json:"count"`
	Preview []ccbosDatasetPreviewItem `json:"preview"`
}

type ccbosDatasetPreviewItem struct {
	Index            int    `json:"index"`
	OriginalQuestion string `json:"original_question"`
	PayloadTail      string `json:"payload_tail"`
	StrategySummary  string `json:"strategy_summary"`
}

func RegisterCCBOSRewriteTools(
	s *server.MCPServer,
	loader *sample.Loader,
	store *SessionStore,
	assessmentRepo *repository.AssessmentRepository,
	targetLLMRepo *repository.TargetLLMRepository,
	serverRepo *repository.ExternalMCPServerRepository,
	caller externalToolCaller,
) {
	tool := mcp.NewTool("rewrite_attack_sample_with_ccbos",
		mcp.WithDescription("Safely rewrite a stored attack sample with CC-BOS and import only the resulting payload handles into the current session."),
		mcp.WithString("sample_id", mcp.Required(), mcp.Description("Attack sample ID loaded internally by the backend.")),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Current pipeline session ID.")),
		mcp.WithString("server_namespace", mcp.Description("External MCP namespace. Defaults to ccbos.")),
		mcp.WithNumber("variant_count", mcp.Description("Number of variants to generate per question. Default 1.")),
		mcp.WithNumber("question_limit", mcp.Description("Optional limit on how many sample questions to rewrite.")),
		mcp.WithString("intent_hint", mcp.Description("Optional hint to preserve the original attack intent.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if loader == nil {
			return mcp.NewToolResultError("sample loader is unavailable"), nil
		}
		if store == nil {
			return mcp.NewToolResultError("session store is unavailable"), nil
		}
		if serverRepo == nil || caller == nil {
			return mcp.NewToolResultError("external MCP bridge is unavailable"), nil
		}

		args := req.GetArguments()
		sampleIDStr, _ := args["sample_id"].(string)
		sessionID, _ := args["session_id"].(string)
		assessmentIDStr, _ := args["assessment_id"].(string)
		namespace, _ := args["server_namespace"].(string)
		intentHint, _ := args["intent_hint"].(string)
		variantCount := parsePositiveInt(args["variant_count"], 1)
		questionLimit := parsePositiveInt(args["question_limit"], 0)

		if strings.TrimSpace(sessionID) == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}

		sampleID, err := uuid.Parse(strings.TrimSpace(sampleIDStr))
		if err != nil {
			return mcp.NewToolResultError("invalid sample_id"), nil
		}

		if strings.TrimSpace(namespace) == "" {
			namespace = defaultCCBOSNamespace
		}
		serverCfg, err := findEnabledExternalServerByNamespace(ctx, serverRepo, namespace)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		loaded, err := loader.LoadSamples(ctx, sampleID)
		if err != nil {
			return mcp.NewToolResultError("load sample failed: " + err.Error()), nil
		}
		defer loaded.Close()

		questions := collectSampleQuestions(loaded.Payloads, questionLimit)
		if len(questions) == 0 {
			return mcp.NewToolResultError("sample has no usable questions"), nil
		}

		generateArgs := map[string]interface{}{
			"questions":     questions,
			"variant_count": variantCount,
			"preview_count": 1,
			"intent_hint":   intentHint,
		}

		targetAttached := false
		if evaluationTarget := buildCCBOSEvaluationTarget(ctx, assessmentRepo, targetLLMRepo, assessmentIDStr); evaluationTarget != nil {
			generateArgs["optimization_mode"] = optimizedCCBOSMode
			generateArgs["population_size"] = defaultCCBOSPopulationSize
			generateArgs["max_iterations"] = defaultCCBOSMaxIterations
			generateArgs["early_stopping_threshold"] = defaultCCBOSEarlyStopScore
			generateArgs["evaluation_target"] = evaluationTarget
			targetAttached = true
		}

		createResult, err := caller.CallTool(ctx, serverCfg.ID, ccbosGenerateToolName, generateArgs)
		if err != nil {
			return mcp.NewToolResultError("CC-BOS generation failed: " + err.Error()), nil
		}
		if createResult != nil && createResult.IsError {
			return mcp.NewToolResultError("CC-BOS generation failed: " + extractTextResult(createResult)), nil
		}

		createText := extractTextResult(createResult)
		if createText == "" {
			return mcp.NewToolResultError("CC-BOS generation returned empty result"), nil
		}

		var createResp ccbosDatasetCreateResponse
		if err := json.Unmarshal([]byte(createText), &createResp); err != nil {
			return mcp.NewToolResultError("parse CC-BOS generation result failed: " + err.Error()), nil
		}
		if strings.TrimSpace(createResp.DatasetID) == "" {
			return mcp.NewToolResultError("CC-BOS generation did not return dataset_id"), nil
		}

		previewResult, err := caller.CallTool(ctx, serverCfg.ID, ccbosPreviewDatasetToolName, map[string]interface{}{
			"dataset_id":     createResp.DatasetID,
			"limit":          createResp.Count,
			"show_full_text": true,
		})
		if err != nil {
			return mcp.NewToolResultError("CC-BOS preview failed: " + err.Error()), nil
		}
		if previewResult != nil && previewResult.IsError {
			return mcp.NewToolResultError("CC-BOS preview failed: " + extractTextResult(previewResult)), nil
		}

		previewText := extractTextResult(previewResult)
		if previewText == "" {
			return mcp.NewToolResultError("CC-BOS preview returned empty result"), nil
		}

		var previewResp ccbosDatasetPreviewResponse
		if err := json.Unmarshal([]byte(previewText), &previewResp); err != nil {
			return mcp.NewToolResultError("parse CC-BOS preview result failed: " + err.Error()), nil
		}

		payloads := buildCCBOSPayloads(previewResp.Preview)
		if len(payloads) == 0 {
			return mcp.NewToolResultError("CC-BOS returned no payloads"), nil
		}
		store.SetPayloads(sessionID, payloads)

		modeUsed := strings.TrimSpace(createResp.Optimization.ModeUsed)
		if modeUsed == "" {
			modeUsed = defaultCCBOSOptimizationMode
		}

		resp := map[string]interface{}{
			"session_id":        sessionID,
			"assessment_id":     strings.TrimSpace(assessmentIDStr),
			"sample_id":         sampleID.String(),
			"server_namespace":  strings.TrimSpace(namespace),
			"dataset_id":        createResp.DatasetID,
			"question_count":    len(questions),
			"imported_count":    len(payloads),
			"privacy_mode":      "handle_only",
			"optimization_mode": modeUsed,
			"target_attached":   targetAttached || createResp.Optimization.TargetEnabled,
			"best_score":        createResp.Optimization.BestScore,
			"strategy_summary":  "CC-BOS rewrite results were imported into the current session without exposing full payload text to the orchestration LLM.",
		}
		bytes, _ := json.Marshal(resp)
		return mcp.NewToolResultText(string(bytes)), nil
	})
}

func findEnabledExternalServerByNamespace(
	ctx context.Context,
	repo *repository.ExternalMCPServerRepository,
	namespace string,
) (*model.ExternalMCPServer, error) {
	items, err := repo.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled external mcp servers: %w", err)
	}
	target := strings.ToLower(strings.TrimSpace(namespace))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Namespace)) == target {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("enabled external MCP server not found for namespace: %s", namespace)
}

func buildCCBOSEvaluationTarget(
	ctx context.Context,
	assessmentRepo *repository.AssessmentRepository,
	targetLLMRepo *repository.TargetLLMRepository,
	assessmentIDStr string,
) map[string]interface{} {
	if assessmentRepo == nil || targetLLMRepo == nil {
		return nil
	}
	assessmentIDStr = strings.TrimSpace(assessmentIDStr)
	if assessmentIDStr == "" {
		return nil
	}
	assessmentID, err := uuid.Parse(assessmentIDStr)
	if err != nil {
		return nil
	}
	assessment, err := assessmentRepo.GetByID(ctx, assessmentID)
	if err != nil || assessment == nil {
		return nil
	}
	targetCfg, err := targetLLMRepo.GetByUserID(ctx, assessment.UserID)
	if err != nil || targetCfg == nil {
		return nil
	}
	return targetLLMToArgs(targetCfg)
}

func targetLLMToArgs(cfg *model.TargetLLMConfig) map[string]interface{} {
	if cfg == nil {
		return nil
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil
	}
	connectorType := strings.TrimSpace(cfg.ConnectorType)
	if connectorType == "" {
		connectorType = "openai"
	}
	args := map[string]interface{}{
		"connector_type": connectorType,
		"base_url":       baseURL,
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		args["api_key"] = cfg.APIKey
	}
	if strings.TrimSpace(cfg.Model) != "" {
		args["model"] = cfg.Model
	}
	return args
}

func collectSampleQuestions(payloads []model.AttackPayload, limit int) []string {
	questions := make([]string, 0, len(payloads))
	for _, payload := range payloads {
		text := strings.TrimSpace(payload.Data)
		if text == "" {
			continue
		}
		questions = append(questions, text)
		if limit > 0 && len(questions) >= limit {
			break
		}
	}
	return questions
}

func buildCCBOSPayloads(items []ccbosDatasetPreviewItem) []PayloadItem {
	sorted := append([]ccbosDatasetPreviewItem(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Index < sorted[j].Index
	})

	payloads := make([]PayloadItem, 0, len(sorted))
	for _, item := range sorted {
		text := strings.TrimSpace(item.PayloadTail)
		if text == "" {
			continue
		}
		index := item.Index
		if index <= 0 {
			index = len(payloads) + 1
		}
		payloads = append(payloads, PayloadItem{
			Index:           index,
			OriginalContent: strings.TrimSpace(item.OriginalQuestion),
			CombinedText:    text,
			Sensitive:       true,
			QuestionSummary: buildCCBOSQuestionSummary(index),
			PayloadSummary:  buildCCBOSPayloadSummary(index, text, item.StrategySummary),
		})
	}
	return payloads
}

func buildCCBOSQuestionSummary(index int) string {
	return fmt.Sprintf("CC-BOS rewritten sample question #%d (redacted)", index)
}

func buildCCBOSPayloadSummary(index int, payload string, strategy string) string {
	strategy = strings.TrimSpace(strategy)
	length := len([]rune(strings.TrimSpace(payload)))
	if strategy == "" {
		return fmt.Sprintf("CC-BOS payload #%d (%d chars)", index, length)
	}
	return fmt.Sprintf("CC-BOS payload #%d (%d chars, strategy: %s)", index, length, strategy)
}

func extractTextResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, content := range result.Content {
		if text, ok := content.(mcp.TextContent); ok {
			return text.Text
		}
	}
	return ""
}

func parsePositiveInt(raw interface{}, fallback int) int {
	switch value := raw.(type) {
	case float64:
		if int(value) > 0 {
			return int(value)
		}
	case int:
		if value > 0 {
			return value
		}
	}
	return fallback
}
