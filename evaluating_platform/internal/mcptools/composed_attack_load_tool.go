package mcptools

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/sample"
)

func registerLoadComposedAttack(s *server.MCPServer, loader *sample.ComposedAttackLoader, store *SessionStore) {
	tool := mcp.NewTool("load_composed_attack",
		mcp.WithDescription("把已组合攻击数据集加载为当前会话的待执行载荷。该数据集已是最终 prompt，可跳过模板拼接。"),
		mcp.WithString("composed_attack_id", mcp.Required(), mcp.Description("已组合攻击 ID")),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("会话 ID")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idStr, _ := req.GetArguments()["composed_attack_id"].(string)
		sessionID, _ := req.GetArguments()["session_id"].(string)
		if sessionID == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return mcp.NewToolResultError("invalid composed_attack_id"), nil
		}

		loaded, err := loader.Load(ctx, id)
		if err != nil {
			return mcp.NewToolResultError("composed attack not found: " + idStr), nil
		}
		defer loaded.Close()

		payloads := make([]PayloadItem, 0, len(loaded.Payloads))
		for _, payload := range loaded.Payloads {
			payloads = append(payloads, PayloadItem{
				Index:           payload.Index,
				OriginalContent: "",
				CombinedText:    payload.Data,
			})
		}

		store.SetPayloads(sessionID, payloads)

		result := map[string]interface{}{
			"session_id":       sessionID,
			"total_count":      len(payloads),
			"strategy_summary": "loaded ready-to-run composed attacks",
		}
		b, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(b)), nil
	})
}
