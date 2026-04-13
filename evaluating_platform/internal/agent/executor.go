package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"evaluating_platform/internal/hub"
)

// Executor 工具执行调度器（通过 MCP Client 调用工具）
type Executor struct {
	mcpClient    *client.Client
	assessmentID string
	hub          *hub.LogHub   // 可选，nil 时不广播
	iteration    int           // 当前迭代轮次（由 Engine 设置）
	toolTimeout  time.Duration // 单工具调用超时，0 表示不限制
}

// NewExecutor 创建执行器
func NewExecutor(mcpClient *client.Client, assessmentID string) *Executor {
	return &Executor{
		mcpClient:    mcpClient,
		assessmentID: assessmentID,
	}
}

// WithHub 注入 LogHub（链式调用）
func (e *Executor) WithHub(h *hub.LogHub) *Executor {
	e.hub = h
	return e
}

// WithToolTimeout 设置单工具调用的超时时间（链式调用）
func (e *Executor) WithToolTimeout(d time.Duration) *Executor {
	e.toolTimeout = d
	return e
}

// SetIteration 由 Engine 在每次 ReAct 循环中更新
func (e *Executor) SetIteration(n int) {
	e.iteration = n
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	ToolName   string
	Severity   string
	InputJSON  string
	OutputJSON string
	DurationMs int64
	Error      error
}

// Execute 通过 MCP Client 执行单个工具
// 自动注入 assessment_id 到参数中，工具处理函数通过它从 Pool 获取 connector
func (e *Executor) Execute(ctx context.Context, toolName string, params map[string]interface{}) *ExecuteResult {
	start := time.Now()

	// 拷贝参数并注入 assessment_id
	args := make(map[string]interface{}, len(params)+1)
	for k, v := range params {
		args[k] = v
	}
	args["assessment_id"] = e.assessmentID

	inputJSON, _ := json.Marshal(args)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	callCtx := ctx
	if timeout := e.timeoutForTool(toolName); timeout > 0 {
		var callCancel context.CancelFunc
		callCtx, callCancel = context.WithTimeout(ctx, timeout)
		defer callCancel()
	}
	callResult, err := e.mcpClient.CallTool(callCtx, req)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		result := &ExecuteResult{
			ToolName:   toolName,
			InputJSON:  string(inputJSON),
			DurationMs: durationMs,
			Error:      fmt.Errorf("MCP CallTool failed: %w", err),
		}
		e.publish(toolName, "error: "+err.Error(), "", durationMs, 0)
		return result
	}

	if callResult.IsError {
		errMsg := extractTextFromResult(callResult)
		result := &ExecuteResult{
			ToolName:   toolName,
			InputJSON:  string(inputJSON),
			DurationMs: durationMs,
			Error:      fmt.Errorf("tool error: %s", errMsg),
		}
		e.publish(toolName, "tool error: "+errMsg, "", durationMs, 0)
		return result
	}

	text := extractTextFromResult(callResult)
	severity := extractSeverity(text)

	e.publish(toolName, text, severity, durationMs, 0)

	return &ExecuteResult{
		ToolName:   toolName,
		Severity:   severity,
		InputJSON:  string(inputJSON),
		OutputJSON: text,
		DurationMs: durationMs,
	}
}

func (e *Executor) timeoutForTool(toolName string) time.Duration {
	timeout := e.toolTimeout
	switch toolName {
	case "execute_payloads":
		if timeout == 0 || timeout < 2*time.Minute {
			return 2 * time.Minute
		}
	case "rewrite_attack_sample_with_ccbos":
		if timeout == 0 || timeout < 5*time.Minute {
			return 5 * time.Minute
		}
	case "generate_report":
		if timeout == 0 || timeout < time.Minute {
			return time.Minute
		}
	}
	return timeout
}

// publish 向 LogHub 广播本次工具调用事件（hub 为 nil 时跳过）
func (e *Executor) publish(toolName, output, severity string, durationMs int64, tokensUsed int) {
	if e.hub == nil {
		return
	}
	e.hub.Publish(hub.LogEvent{
		AssessmentID: e.assessmentID,
		Iteration:    e.iteration,
		ToolName:     toolName,
		Output:       output,
		Severity:     severity,
		TokensUsed:   tokensUsed,
		DurationMs:   durationMs,
		Timestamp:    time.Now(),
	})
}

// extractTextFromResult 从 MCP CallToolResult 中提取文本内容
func extractTextFromResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, content := range result.Content {
		if tc, ok := content.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

// extractSeverity 从 JSON 结果字符串中提取 severity 字段
func extractSeverity(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return ""
	}
	if s, ok := m["severity"].(string); ok {
		return s
	}
	return ""
}
