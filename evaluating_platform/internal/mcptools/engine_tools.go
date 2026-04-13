package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/connector"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/internal/sample"
)



// ---------- list_templates ----------

func registerListTemplates(s *server.MCPServer, tplRepo *repository.TemplateRepository) {
	tool := mcp.NewTool("list_templates",
		mcp.WithDescription("列出评测模板资源卡片，返回标签、适用评估类型和规划摘要，帮助编排 LLM 选择模板"),
		mcp.WithString("expert_id",
			mcp.Description("专家用户 ID（可选，为空时返回已发布的公开模板）"),
		),
		mcp.WithString("sub_type",
			mcp.Description("按子类型过滤（可选）"),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		expertIDStr, _ := req.GetArguments()["expert_id"].(string)
		subType, _ := req.GetArguments()["sub_type"].(string)

		var templates []model.Template
		var err error

		if expertIDStr == "" {
			// 无 expert_id：回退查询已发布的公开模板
			templates, _, err = tplRepo.ListPublished(ctx, subType, 10000, 0)
		} else {
			// 有 expert_id：查询该专家的模板
			expertID, parseErr := uuid.Parse(expertIDStr)
			if parseErr != nil {
				return mcp.NewToolResultError("invalid expert_id"), nil
			}
			templates, _, err = tplRepo.ListByExpert(ctx, expertID, subType, 10000, 0)
		}

		if err != nil {
			return mcp.NewToolResultError("query templates failed: " + err.Error()), nil
		}

		items := make([]resourceCard, 0, len(templates))
		for _, t := range templates {
			items = append(items, buildTemplateCard(t))
		}

		b, _ := json.Marshal(items)
		return mcp.NewToolResultText(string(b)), nil
	})
}

// ---------- get_template ----------

func registerGetTemplate(s *server.MCPServer, tplRepo *repository.TemplateRepository) {
	tool := mcp.NewTool("get_template",
		mcp.WithDescription("根据模板 ID 获取评测模板的详细信息（含内容）；编排决策优先使用 preview_template"),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("评测模板 ID"),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tplIDStr, _ := req.GetArguments()["template_id"].(string)

		tplID, err := uuid.Parse(tplIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid template_id"), nil
		}

		t, err := tplRepo.GetByID(ctx, tplID)
		if err != nil {
			return mcp.NewToolResultError("template not found: " + tplIDStr), nil
		}

		type templateDetail struct {
			ID          uuid.UUID `json:"id"`
			Name        string    `json:"name"`
			SubType     string    `json:"sub_type"`
			Description string    `json:"description"`
			Content     string    `json:"content"`
		}

		detail := templateDetail{
			ID:          t.ID,
			Name:        t.Name,
			SubType:     t.SubType,
			Description: t.Description,
			Content:     t.Content,
		}

		b, _ := json.Marshal(detail)
		return mcp.NewToolResultText(string(b)), nil
	})

	registerPreviewTemplate(s, tplRepo)
}

// ---------- combine_template_sample ----------

func registerCombineTemplateSample(s *server.MCPServer, tplRepo *repository.TemplateRepository, loader *sample.Loader, store *SessionStore) {
	tool := mcp.NewTool("combine_template_sample",
		mcp.WithDescription("将模板内容与样本数据组合，生成 payload 列表并存入会话存储"),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("评测模板 ID"),
		),
		mcp.WithString("sample_id",
			mcp.Required(),
			mcp.Description("攻击样本 ID"),
		),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("会话 ID"),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tplIDStr, _ := req.GetArguments()["template_id"].(string)
		sampleIDStr, _ := req.GetArguments()["sample_id"].(string)
		sessionID, _ := req.GetArguments()["session_id"].(string)

		if sessionID == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}

		tplID, err := uuid.Parse(tplIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid template_id"), nil
		}
		sampleID, err := uuid.Parse(sampleIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid sample_id"), nil
		}

		// 读取模板
		tpl, err := tplRepo.GetByID(ctx, tplID)
		if err != nil {
			return mcp.NewToolResultError("template not found: " + tplIDStr), nil
		}

		// 读取样本
		loaded, err := loader.LoadSamples(ctx, sampleID)
		if err != nil {
			return mcp.NewToolResultError("sample not found: " + sampleIDStr), nil
		}
		defer loaded.Close()

		payloads := make([]PayloadItem, 0, len(loaded.Payloads))
		for _, p := range loaded.Payloads {
			payloads = append(payloads, PayloadItem{
				Index:           p.Index,
				OriginalContent: p.Data,
				CombinedText:    tpl.Content + "\n" + p.Data,
			})
		}

		// Store payloads in SessionStore
		store.SetPayloads(sessionID, payloads)

		result := map[string]interface{}{
			"session_id":  sessionID,
			"total_count": len(payloads),
		}

		b, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(b)), nil
	})
}

// ---------- enhance_payloads ----------

func registerEnhancePayloads(s *server.MCPServer, auxLLMRepo *repository.AuxiliaryLLMRepository, store *SessionStore) {
	tool := mcp.NewTool("enhance_payloads",
		mcp.WithDescription("对 payload 列表进行增强处理，支持不增强和多语言增强策略"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("会话 ID"),
		),
		mcp.WithString("strategy",
			mcp.Required(),
			mcp.Description("增强策略：none（不增强）或 multilingual（多语言增强）"),
		),
		mcp.WithString("expert_id",
			mcp.Description("专家用户 ID（可选，仅 multilingual 策略需要，用于查询辅助 LLM 配置）"),
		),
		mcp.WithString("target_languages",
			mcp.Description("目标语言 JSON 数组（可选，仅 multilingual 策略，默认 [\"en\",\"ja\",\"ko\",\"fr\",\"de\"]）"),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, _ := req.GetArguments()["session_id"].(string)
		strategy, _ := req.GetArguments()["strategy"].(string)
		expertIDStr, _ := req.GetArguments()["expert_id"].(string)
		targetLangsStr, _ := req.GetArguments()["target_languages"].(string)

		if sessionID == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}

		// Read payloads from SessionStore
		payloads, err := store.GetPayloads(sessionID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		switch strategy {
		case "none":
			return handleNoneStrategy(store, sessionID, payloads)
		case "multilingual":
			if expertIDStr == "" {
				return mcp.NewToolResultError("multilingual 策略需要 expert_id 以查询辅助 LLM 配置"), nil
			}
			return handleMultilingualStrategy(ctx, store, sessionID, payloads, expertIDStr, targetLangsStr, auxLLMRepo)
		default:
			return mcp.NewToolResultError(
				fmt.Sprintf("unsupported strategy: %s, available: none, multilingual", strategy),
			), nil
		}
	})
}

// handleNoneStrategy 不增强策略：设置 EnhancedText = CombinedText
func handleNoneStrategy(store *SessionStore, sessionID string, payloads []PayloadItem) (*mcp.CallToolResult, error) {
	for i := range payloads {
		payloads[i].EnhancedText = payloads[i].CombinedText
	}

	store.SetPayloads(sessionID, payloads)

	result := map[string]interface{}{
		"session_id":       sessionID,
		"total_count":      len(payloads),
		"strategy_summary": "none: payloads unchanged",
	}

	b, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(b)), nil
}

// handleMultilingualStrategy 多语言增强策略
func handleMultilingualStrategy(
	ctx context.Context,
	store *SessionStore,
	sessionID string,
	payloads []PayloadItem,
	expertIDStr string,
	targetLangsStr string,
	auxLLMRepo *repository.AuxiliaryLLMRepository,
) (*mcp.CallToolResult, error) {
	expertID, err := uuid.Parse(expertIDStr)
	if err != nil {
		return mcp.NewToolResultError("invalid expert_id"), nil
	}

	// 读取辅助 LLM 配置
	cfg, err := auxLLMRepo.GetByUserID(ctx, expertID)
	if err != nil {
		return mcp.NewToolResultError("query auxiliary LLM config failed: " + err.Error()), nil
	}
	if cfg == nil {
		return mcp.NewToolResultError("auxiliary LLM not configured, please configure it first"), nil
	}

	// 解析目标语言
	targetLangs := []string{"en", "ja", "ko", "fr", "de"}
	if targetLangsStr != "" {
		if err := json.Unmarshal([]byte(targetLangsStr), &targetLangs); err != nil {
			return mcp.NewToolResultError("invalid target_languages JSON: " + err.Error()), nil
		}
	}

	// 创建临时连接器
	conn := connector.NewConnector(&connector.Config{
		Type:     "openai",
		Endpoint: cfg.BaseURL,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
	})

	var enhanced []PayloadItem
	for _, p := range payloads {
		for _, lang := range targetLangs {
			prompt := fmt.Sprintf(
				"Translate the following text to %s. Keep the original meaning and intent. Only output the translation, nothing else.\n\n%s",
				lang, p.CombinedText,
			)

			resp, err := conn.SendMessage(ctx, &connector.AssessRequest{
				Messages: []connector.Message{{Role: "user", Content: prompt}},
			})

			ep := PayloadItem{
				Index:           p.Index,
				OriginalContent: p.OriginalContent,
				CombinedText:    p.CombinedText,
				Language:        lang,
			}
			if err != nil {
				ep.EnhancedText = ""
			} else {
				ep.EnhancedText = resp.Content
			}
			enhanced = append(enhanced, ep)
		}
	}

	// Update SessionStore with enhanced payloads
	store.SetPayloads(sessionID, enhanced)

	result := map[string]interface{}{
		"session_id":       sessionID,
		"total_count":      len(enhanced),
		"strategy_summary": fmt.Sprintf("multilingual: %d payloads × %d languages = %d total", len(payloads), len(targetLangs), len(enhanced)),
	}

	b, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(b)), nil
}
