package mcptools

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/internal/sample"
)

func registerComposedAttackQueryTools(s *server.MCPServer, repo *repository.ComposedAttackRepository, loader *sample.ComposedAttackLoader) {
	listTool := mcp.NewTool("list_composed_attacks",
		mcp.WithDescription("列出已组合攻击资源卡片。这类资产已经是可直接执行的最终载荷，可跳过模板拼接。"),
		mcp.WithString("expert_id", mcp.Description("专家用户 ID，可选；为空时返回可用的公开资源")),
		mcp.WithString("sub_type", mcp.Description("按子类型过滤，可选")),
	)

	s.AddTool(listTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		expertIDStr, _ := req.GetArguments()["expert_id"].(string)
		subType, _ := req.GetArguments()["sub_type"].(string)

		var items []model.ComposedAttack
		var err error
		if expertIDStr == "" {
			items, _, err = repo.ListPublished(ctx, subType, 10000, 0)
		} else {
			expertID, parseErr := uuid.Parse(expertIDStr)
			if parseErr != nil {
				return mcp.NewToolResultError("invalid expert_id"), nil
			}
			items, _, err = repo.ListByExpert(ctx, expertID, subType, 10000, 0)
		}
		if err != nil {
			return mcp.NewToolResultError("query composed attacks failed: " + err.Error()), nil
		}

		cards := make([]resourceCard, 0, len(items))
		for _, item := range items {
			cards = append(cards, buildComposedAttackCard(item, nil))
		}
		b, _ := json.Marshal(cards)
		return mcp.NewToolResultText(string(b)), nil
	})

	getTool := mcp.NewTool("get_composed_attack",
		mcp.WithDescription("根据 ID 获取已组合攻击的基础信息；编排优先使用 preview_composed_attack"),
		mcp.WithString("composed_attack_id", mcp.Required(), mcp.Description("已组合攻击 ID")),
	)

	s.AddTool(getTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idStr, _ := req.GetArguments()["composed_attack_id"].(string)
		id, err := uuid.Parse(idStr)
		if err != nil {
			return mcp.NewToolResultError("invalid composed_attack_id"), nil
		}
		item, err := repo.GetByID(ctx, id)
		if err != nil {
			return mcp.NewToolResultError("composed attack not found: " + idStr), nil
		}
		b, _ := json.Marshal(item)
		return mcp.NewToolResultText(string(b)), nil
	})

	registerPreviewComposedAttack(s, repo, loader)
}

func registerPreviewComposedAttack(s *server.MCPServer, repo *repository.ComposedAttackRepository, loader *sample.ComposedAttackLoader) {
	tool := mcp.NewTool("preview_composed_attack",
		mcp.WithDescription("返回已组合攻击的资源画像和少量脱敏预览，帮助编排 LLM 判断是否应直接执行。"),
		mcp.WithString("composed_attack_id", mcp.Required(), mcp.Description("已组合攻击 ID")),
		mcp.WithNumber("preview_count", mcp.Description("预览条数，默认 3")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idStr, _ := req.GetArguments()["composed_attack_id"].(string)
		id, err := uuid.Parse(idStr)
		if err != nil {
			return mcp.NewToolResultError("invalid composed_attack_id"), nil
		}
		previewCount := 3
		if rawLimit, ok := req.GetArguments()["preview_count"].(float64); ok && rawLimit > 0 {
			previewCount = int(rawLimit)
		}

		item, err := repo.GetByID(ctx, id)
		if err != nil {
			return mcp.NewToolResultError("composed attack not found: " + idStr), nil
		}
		preview := []model.AttackPayload{}
		if loader != nil {
			preview, _ = loader.Preview(ctx, id, previewCount)
		}

		card := buildComposedAttackCard(*item, preview)
		result := map[string]interface{}{
			"composed_attack": card,
			"preview_count":   len(card.SanitizedExamples),
		}
		b, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(b)), nil
	})
}
