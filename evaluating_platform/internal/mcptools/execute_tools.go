package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/connector"
	"evaluating_platform/internal/repository"
)

// registerExecutePayloads registers the execute_payloads MCP tool that sends
// payloads to the target LLM, detects attack success, and stores results.
func registerExecutePayloads(s *server.MCPServer, targetLLMRepo *repository.TargetLLMRepository, store *SessionStore) {
	tool := mcp.NewTool("execute_payloads",
		mcp.WithDescription("????? payload ??????? LLM??????????????"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("?? ID"),
		),
		mcp.WithString("user_id",
			mcp.Required(),
			mcp.Description("???? ID??????? LLM ??"),
		),
		mcp.WithNumber("max_payloads",
			mcp.Description("???? payload ????????????"),
		),
		mcp.WithNumber("offset",
			mcp.Description("? 0 ??? payload ??????????"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, _ := req.GetArguments()["session_id"].(string)
		userIDStr, _ := req.GetArguments()["user_id"].(string)

		if sessionID == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}
		if userIDStr == "" {
			return mcp.NewToolResultError("user_id is required"), nil
		}

		maxPayloads := 0
		if rawLimit, ok := req.GetArguments()["max_payloads"].(float64); ok && rawLimit > 0 {
			maxPayloads = int(rawLimit)
		}
		offset := 0
		if rawOffset, ok := req.GetArguments()["offset"].(float64); ok && rawOffset > 0 {
			offset = int(rawOffset)
		}

		payloads, err := store.GetPayloads(sessionID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		totalAvailable := len(payloads)
		if offset >= totalAvailable {
			summary := map[string]interface{}{
				"session_id":      sessionID,
				"offset":          offset,
				"total_count":     0,
				"processed_count": 0,
				"total_available": totalAvailable,
				"success_count":   0,
				"success_rate":    0,
				"results":         []ExecutionResult{},
			}
			b, _ := json.Marshal(summary)
			return mcp.NewToolResultText(string(b)), nil
		}
		if offset > 0 {
			payloads = payloads[offset:]
		}
		if maxPayloads > 0 && len(payloads) > maxPayloads {
			payloads = payloads[:maxPayloads]
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid user_id"), nil
		}

		cfg, err := targetLLMRepo.GetByUserID(ctx, userID)
		if err != nil {
			return mcp.NewToolResultError("query target LLM config failed: " + err.Error()), nil
		}
		if cfg == nil {
			return mcp.NewToolResultError(fmt.Sprintf("target LLM not configured for user: %s", userIDStr)), nil
		}

		conn := connector.NewConnector(&connector.Config{
			Type:     cfg.ConnectorType,
			Endpoint: cfg.BaseURL,
			APIKey:   cfg.APIKey,
			Model:    cfg.Model,
		})

		detector := &AttackDetector{}
		results := make([]ExecutionResult, len(payloads))
		var wg sync.WaitGroup
		workerLimit := 4
		if len(payloads) < workerLimit {
			workerLimit = len(payloads)
		}
		if workerLimit <= 0 {
			workerLimit = 1
		}
		sem := make(chan struct{}, workerLimit)

		for i, payload := range payloads {
			wg.Add(1)
			go func(index int, p PayloadItem) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				text := p.EnhancedText
				if text == "" {
					text = p.CombinedText
				}

				result := ExecutionResult{
					Index:     p.Index,
					Sensitive: p.Sensitive,
				}
				if p.Sensitive {
					result.QuestionSummary = firstNonEmpty(strings.TrimSpace(p.QuestionSummary), buildSensitiveQuestionSummary(p.Index))
					result.PayloadSummary = firstNonEmpty(strings.TrimSpace(p.PayloadSummary), buildSensitivePayloadSummary(p.Index, text))
					result.OriginalContent = result.QuestionSummary
					result.EnhancedContent = result.PayloadSummary
				} else {
					result.OriginalContent = p.OriginalContent
					result.EnhancedContent = text
				}

				resp, err := conn.SendMessage(ctx, &connector.AssessRequest{
					Messages: []connector.Message{{Role: "user", Content: text}},
				})
				if err != nil {
					result.Error = fmt.Sprintf("send payload failed: %s", err.Error())
					results[index] = result
					return
				}

				result.TargetResponse = resp.Content
				judgmentInput := p.OriginalContent
				if judgmentInput == "" {
					judgmentInput = text
				}
				detection := detector.Detect(ctx, conn, judgmentInput, resp.Content)
				result.AttackSuccess = detection.Success
				result.AttackReason = detection.Reason
				results[index] = result
			}(i, payload)
		}
		wg.Wait()

		successCount := 0
		for _, result := range results {
			if result.AttackSuccess {
				successCount++
			}
		}

		store.SetResults(sessionID, results)

		totalCount := len(results)
		var successRate float64
		if totalCount > 0 {
			successRate = float64(successCount) / float64(totalCount)
		}

		summary := map[string]interface{}{
			"session_id":      sessionID,
			"offset":          offset,
			"total_count":     totalCount,
			"processed_count": totalCount,
			"total_available": totalAvailable,
			"success_count":   successCount,
			"success_rate":    successRate,
			"results":         results,
		}

		b, _ := json.Marshal(summary)
		return mcp.NewToolResultText(string(b)), nil
	})
}

func buildSensitiveQuestionSummary(index int) string {
	return fmt.Sprintf("CC-BOS 文言文改写样本问题 #%d（已脱敏）", index)
}

func buildSensitivePayloadSummary(index int, payload string) string {
	return fmt.Sprintf("CC-BOS 文言文改写载荷 #%d（长度 %d 字）", index, len([]rune(strings.TrimSpace(payload))))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
