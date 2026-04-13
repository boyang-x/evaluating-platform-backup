package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

type previewItem struct {
	Index            int    `json:"index"`
	OriginalQuestion string `json:"original_question,omitempty"`
	PayloadTail      string `json:"payload_tail"`
	StrategySummary  string `json:"strategy_summary,omitempty"`
}

func main() {
	cfg := loadConfig()
	store, err := NewDatasetStore(cfg.ArtifactDir)
	if err != nil {
		log.Fatalf("create dataset store: %v", err)
	}
	rewriter := newRewriteClient(cfg)

	server := mcpserver.NewMCPServer("cc-bos-mcp", "0.1.0")
	registerGenerateTool(server, rewriter, store, cfg)
	registerPreviewTool(server, store)
	registerExportTool(server, store)

	log.Printf("CC-BOS MCP server public base URL: %s/sse", strings.TrimRight(cfg.PublicBaseURL, "/"))
	log.Printf("CC-BOS MCP server listening on %s:%d", cfg.ListenHost, cfg.Port)
	log.Printf("artifact dir: %s", cfg.ArtifactDir)
	if strings.TrimSpace(cfg.APIKey) == "" {
		log.Printf("warning: CCBOS_API_KEY is empty; generation tool will return a configuration error until you set it")
	}

	sseServer := mcpserver.NewSSEServer(server, mcpserver.WithBaseURL(strings.TrimRight(cfg.PublicBaseURL, "/")))
	if err := sseServer.Start(fmt.Sprintf("%s:%d", cfg.ListenHost, cfg.Port)); err != nil {
		log.Fatalf("start sse server: %v", err)
	}
}

func registerGenerateTool(server *mcpserver.MCPServer, rewriter *rewriteClient, store *DatasetStore, cfg Config) {
	tool := mcp.Tool{
		Name:        "generate_classical_chinese_payloads",
		Description: "Rewrite seed attack questions into classical Chinese payload variants. Returns only a dataset handle and a small preview by default.",
		RawInputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "questions": {
                    "type": "array",
                    "description": "List of seed questions to rewrite",
                    "items": {"type": "string"},
                    "minItems": 1
                },
                "variant_count": {
                    "type": "integer",
                    "description": "How many classical-Chinese variants to generate per question. Recommended range: 1-5.",
                    "minimum": 1,
                    "maximum": 5,
                    "default": 1
                },
                "intent_hint": {
                    "type": "string",
                    "description": "Optional hint used to preserve the original malicious intent more accurately"
                },
                "optimization_mode": {
                    "type": "string",
                    "description": "Optional generation mode: single_pass or fruitfly.",
                    "enum": ["single_pass", "fruitfly"],
                    "default": "single_pass"
                },
                "population_size": {
                    "type": "integer",
                    "description": "Population size used by fruitfly optimization. Recommended range: 3-8.",
                    "minimum": 2,
                    "maximum": 8,
                    "default": 4
                },
                "max_iterations": {
                    "type": "integer",
                    "description": "Maximum optimization iterations for fruitfly mode.",
                    "minimum": 1,
                    "maximum": 5,
                    "default": 2
                },
                "early_stopping_threshold": {
                    "type": "number",
                    "description": "Stop optimization early once a candidate reaches this score. Range 0-120.",
                    "minimum": 0,
                    "maximum": 120,
                    "default": 80
                },
                "evaluation_target": {
                    "type": "object",
                    "description": "Optional target model config used for fruitfly evaluation.",
                    "properties": {
                        "connector_type": {"type": "string"},
                        "base_url": {"type": "string"},
                        "api_key": {"type": "string"},
                        "model": {"type": "string"},
                        "headers": {
                            "type": "object",
                            "additionalProperties": {"type": "string"}
                        }
                    }
                },
                "preview_count": {
                    "type": "integer",
                    "description": "How many preview items to return. Default is 3.",
                    "minimum": 1,
                    "maximum": 10,
                    "default": 3
                }
            },
            "required": ["questions"]
        }`),
	}

	server.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		questions := parseQuestions(req.GetArguments()["questions"])
		if len(questions) == 0 {
			return mcp.NewToolResultError("questions is required"), nil
		}
		variantCount := parseIntArg(req.GetArguments()["variant_count"], 1)
		previewCount := parseIntArg(req.GetArguments()["preview_count"], 3)
		intentHint, _ := req.GetArguments()["intent_hint"].(string)
		options, err := parseGenerationOptions(req.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		items, optimization, err := rewriter.GeneratePayloads(ctx, req.Header, questions, variantCount, intentHint, options)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		dataset := PayloadDataset{
			DatasetID:    newDatasetID(),
			ResourceType: "payload_dataset",
			Source:       "cc_bos_mcp",
			Model:        cfg.Model,
			CreatedAt:    time.Now(),
			Items:        items,
		}
		if err := store.Save(dataset); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result := map[string]interface{}{
			"status":        "success",
			"resource_type": dataset.ResourceType,
			"dataset_id":    dataset.DatasetID,
			"count":         len(dataset.Items),
			"preview":       buildPreview(dataset.Items, previewCount, false),
			"artifact_path": filepath.Join(cfg.ArtifactDir, "datasets", dataset.DatasetID+".json"),
			"usage_note":    "The full payload dataset has been stored locally. Only the handle and preview are returned by default.",
		}
		if optimization != nil {
			result["optimization"] = map[string]interface{}{
				"mode_requested": optimization.ModeRequested,
				"mode_used":      optimization.ModeUsed,
				"attempts":       optimization.Attempts,
				"best_score":     optimization.BestScore,
				"target_enabled": optimization.TargetEnabled,
			}
		}
		bytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(bytes)), nil
	})
}

func registerPreviewTool(server *mcpserver.MCPServer, store *DatasetStore) {
	tool := mcp.Tool{
		Name:        "preview_payload_dataset",
		Description: "Preview a previously generated payload dataset. Full payload text is hidden by default.",
		RawInputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "dataset_id": {
                    "type": "string",
                    "description": "Dataset handle returned by generate_classical_chinese_payloads"
                },
                "limit": {
                    "type": "integer",
                    "minimum": 1,
                    "maximum": 20,
                    "default": 5
                },
                "show_full_text": {
                    "type": "boolean",
                    "description": "Debug only. When true, return full payload text. Default false."
                }
            },
            "required": ["dataset_id"]
        }`),
	}

	server.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		datasetID, _ := req.GetArguments()["dataset_id"].(string)
		if strings.TrimSpace(datasetID) == "" {
			return mcp.NewToolResultError("dataset_id is required"), nil
		}
		limit := parseIntArg(req.GetArguments()["limit"], 5)
		showFullText, _ := req.GetArguments()["show_full_text"].(bool)

		dataset, err := store.Load(datasetID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result := map[string]interface{}{
			"status":        "success",
			"dataset_id":    dataset.DatasetID,
			"count":         len(dataset.Items),
			"resource_type": dataset.ResourceType,
			"preview":       buildPreview(dataset.Items, limit, showFullText),
		}
		bytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(bytes)), nil
	})
}

func registerExportTool(server *mcpserver.MCPServer, store *DatasetStore) {
	tool := mcp.Tool{
		Name:        "export_payload_dataset_csv",
		Description: "Export a generated payload dataset to a local CSV file and return the file path.",
		RawInputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "dataset_id": {
                    "type": "string",
                    "description": "Dataset handle returned by generate_classical_chinese_payloads"
                },
                "output_name": {
                    "type": "string",
                    "description": "Optional CSV file name. If omitted, one is generated automatically."
                }
            },
            "required": ["dataset_id"]
        }`),
	}

	server.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		datasetID, _ := req.GetArguments()["dataset_id"].(string)
		outputName, _ := req.GetArguments()["output_name"].(string)
		if strings.TrimSpace(datasetID) == "" {
			return mcp.NewToolResultError("dataset_id is required"), nil
		}
		path, count, err := store.ExportCSV(datasetID, outputName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result := map[string]interface{}{
			"status":     "success",
			"dataset_id": datasetID,
			"count":      count,
			"file_path":  path,
		}
		bytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(bytes)), nil
	})
}

func buildPreview(items []PayloadItem, limit int, showFullText bool) []previewItem {
	if limit <= 0 {
		limit = 5
	}
	if limit > len(items) {
		limit = len(items)
	}
	preview := make([]previewItem, 0, limit)
	for _, item := range items[:limit] {
		payloadTail := summarizePayload(item.FinalPayloadText, showFullText)
		preview = append(preview, previewItem{
			Index:            item.Index,
			OriginalQuestion: item.OriginalQuestion,
			PayloadTail:      payloadTail,
			StrategySummary:  item.StrategySummary,
		})
	}
	return preview
}

func summarizePayload(text string, showFullText bool) string {
	text = strings.TrimSpace(text)
	if showFullText || len([]rune(text)) <= 30 {
		return text
	}
	runes := []rune(text)
	return fmt.Sprintf("Tail preview (%d chars total): %s", len(runes), string(runes[len(runes)-30:]))
}

func parseQuestions(raw interface{}) []string {
	switch value := raw.(type) {
	case []interface{}:
		items := make([]string, 0, len(value))
		for _, entry := range value {
			if str, ok := entry.(string); ok && strings.TrimSpace(str) != "" {
				items = append(items, strings.TrimSpace(str))
			}
		}
		return items
	case []string:
		items := make([]string, 0, len(value))
		for _, entry := range value {
			if strings.TrimSpace(entry) != "" {
				items = append(items, strings.TrimSpace(entry))
			}
		}
		return items
	default:
		return nil
	}
}

func parseGenerationOptions(args map[string]interface{}) (generationOptions, error) {
	options := generationOptions{
		OptimizationMode:       parseStringArg(args["optimization_mode"]),
		PopulationSize:         parseIntArg(args["population_size"], defaultPopulationSize),
		MaxIterations:          parseIntArg(args["max_iterations"], defaultMaxIterations),
		EarlyStoppingThreshold: parseFloatArg(args["early_stopping_threshold"], defaultEarlyStoppingScore),
	}
	if rawTarget, ok := args["evaluation_target"]; ok && rawTarget != nil {
		target, err := parseEvaluationTarget(rawTarget)
		if err != nil {
			return generationOptions{}, err
		}
		options.EvaluationTarget = target
	}
	return normalizeGenerationOptions(options), nil
}

func parseEvaluationTarget(raw interface{}) (*evaluationTargetConfig, error) {
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal evaluation_target: %w", err)
	}
	var target evaluationTargetConfig
	if err := json.Unmarshal(bytes, &target); err != nil {
		return nil, fmt.Errorf("parse evaluation_target: %w", err)
	}
	if strings.TrimSpace(target.BaseURL) == "" {
		return nil, fmt.Errorf("evaluation_target.base_url is required when optimization is enabled")
	}
	if strings.TrimSpace(target.ConnectorType) == "" {
		target.ConnectorType = "openai"
	}
	return &target, nil
}

func parseStringArg(raw interface{}) string {
	if value, ok := raw.(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func parseFloatArg(raw interface{}, fallback float64) float64 {
	switch value := raw.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	}
	return fallback
}

func parseIntArg(raw interface{}, fallback int) int {
	switch value := raw.(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return fallback
	}
}
