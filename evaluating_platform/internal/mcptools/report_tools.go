package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/agent"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/report"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/llm"
	"evaluating_platform/pkg/logger"
	"evaluating_platform/pkg/storage"
)

// registerGenerateReport registers the generate_report MCP tool that reads
// execution results from the SessionStore, uses report.Generator to produce
// a PDF report (with MinIO upload), persists the report to the database,
// and returns a JSON response containing the PDF URL.
func registerGenerateReport(
	s *server.MCPServer,
	auxLLMRepo *repository.AuxiliaryLLMRepository,
	store *SessionStore,
	minioClient *storage.MinIOClient,
	reportRepo *repository.ReportRepository,
) {
	tool := mcp.NewTool("generate_report",
		mcp.WithDescription("根据执行结果，使用辅助 LLM 生成 PDF 安全评估报告并返回下载 URL"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("会话 ID"),
		),
		mcp.WithString("expert_id",
			mcp.Description("专家用户 ID（可选，为空时跳过辅助 LLM 报告生成）"),
		),
		mcp.WithString("goal",
			mcp.Required(),
			mcp.Description("本次评估的目标描述"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, _ := req.GetArguments()["session_id"].(string)
		expertIDStr, _ := req.GetArguments()["expert_id"].(string)
		goal, _ := req.GetArguments()["goal"].(string)

		if sessionID == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}
		if expertIDStr == "" {
			// 无 expert_id：跳过辅助 LLM 报告生成，由 runAssessment 使用编排 LLM 生成
			skipResp := map[string]interface{}{
				"message": "expert_id 未提供，报告将由 runAssessment 使用编排 LLM 生成",
				"skipped": true,
			}
			b, _ := json.Marshal(skipResp)
			return mcp.NewToolResultText(string(b)), nil
		}
		if goal == "" {
			return mcp.NewToolResultError("goal is required"), nil
		}

		// Read execution results from SessionStore
		results, err := store.GetResults(sessionID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Parse expert_id and look up auxiliary LLM config
		expertID, err := uuid.Parse(expertIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid expert_id"), nil
		}

		cfg, err := auxLLMRepo.GetByUserID(ctx, expertID)
		if err != nil {
			return mcp.NewToolResultError("query auxiliary LLM config failed: " + err.Error()), nil
		}
		if cfg == nil {
			return mcp.NewToolResultError("auxiliary LLM not configured, please configure it first"), nil
		}

		// Create an LLM client from the auxiliary LLM config
		auxLLMClient := llm.NewClient(cfg.BaseURL, cfg.APIKey, cfg.Model)

		// Build a report.Generator with the auxiliary LLM and MinIO client
		gen := report.NewGenerator(auxLLMClient, minioClient)

		// Convert ExecutionResults to agent.ToolResult for the Generator
		toolResults := executionResultsToToolResults(results)

		// Build a temporary assessment model for the generator
		assessmentID := uuid.New()
		assessment := &model.Assessment{
			ID:   assessmentID,
			Name: "安全评估",
			Goal: goal,
		}

		// Generate report (includes PDF rendering + MinIO upload)
		rpt, err := gen.Generate(ctx, assessment, toolResults)
		if err != nil {
			// Fallback: generate a degraded report without LLM
			logger.Warn("report generation via LLM failed, using fallback", map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			rpt = gen.GenerateFallback(ctx, assessment, toolResults)
		}

		// Persist report to database
		if reportRepo != nil {
			if err := reportRepo.Create(ctx, rpt); err != nil {
				logger.Warn("persist report to database failed", map[string]interface{}{
					"report_id": rpt.ID.String(),
					"error":     err.Error(),
				})
			}
		}

		// Return JSON with pdf_url, report_id, risk_level, summary
		response := map[string]interface{}{
			"pdf_url":    rpt.PDFURL,
			"report_id":  rpt.ID.String(),
			"risk_level": rpt.RiskLevel,
			"summary":    rpt.Summary,
		}

		b, _ := json.Marshal(response)
		return mcp.NewToolResultText(string(b)), nil
	})
}

// executionResultsToToolResults converts MCP SessionStore ExecutionResults
// to agent.ToolResult slice for use with report.Generator.
func executionResultsToToolResults(results []ExecutionResult) []agent.ToolResult {
	toolResults := make([]agent.ToolResult, 0, len(results))
	for _, r := range results {
		severity := "info"
		if r.AttackSuccess {
			severity = "high"
		}
		output := r.TargetResponse
		if r.Error != "" {
			output = fmt.Sprintf("error: %s", r.Error)
		}
		toolResults = append(toolResults, agent.ToolResult{
			ToolName: fmt.Sprintf("payload_%d", r.Index),
			Input:    formatReportToolInput(preferredReportToolValue(r.QuestionSummary, r.OriginalContent), preferredReportToolValue(r.PayloadSummary, r.EnhancedContent)),
			Output:   output,
			Severity: severity,
		})
	}
	return toolResults
}

func preferredReportToolValue(primary, fallback string) string {
	primary = strings.TrimSpace(primary)
	if primary != "" {
		return primary
	}
	return strings.TrimSpace(fallback)
}

func formatReportToolInput(originalContent, actualContent string) string {
	originalContent = strings.TrimSpace(originalContent)
	actualContent = strings.TrimSpace(actualContent)
	if originalContent == "" && actualContent == "" {
		return ""
	}
	if actualContent == "" || actualContent == originalContent {
		return "样本问题:\n" + originalContent
	}
	return fmt.Sprintf("样本问题:\n%s\n\n实际发送内容:\n%s", originalContent, actualContent)
}

// buildReportPrompt constructs the prompt sent to the auxiliary LLM, including
// the assessment goal, overall statistics, and per-question execution data.
// Uses the standardized report template from internal/report/template.go.
func buildReportPrompt(goal string, results []ExecutionResult) string {
	totalCount := len(results)
	successCount := 0
	for _, r := range results {
		if r.AttackSuccess {
			successCount++
		}
	}
	var successRate float64
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount) * 100
	}

	// Build per-question details
	perQuestion := ""
	for _, r := range results {
		perQuestion += fmt.Sprintf(`
--- 第 %d 题 ---
原始样本: %s
被测 LLM 响应: %s
攻击成功: %v
判定理由: %s
`, r.Index, r.OriginalContent, r.TargetResponse, r.AttackSuccess, r.AttackReason)
		if r.Error != "" {
			perQuestion += fmt.Sprintf("执行错误: %s\n", r.Error)
		}
	}

	return fmt.Sprintf(report.StandardReportPromptTemplate,
		"安全评估", goal,
		totalCount, successCount, successRate,
		perQuestion)
}
