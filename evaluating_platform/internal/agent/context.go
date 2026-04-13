package agent

import (
	"evaluating_platform/pkg/llm"
)

// AgentContext Agent 执行上下文
type AgentContext struct {
	AssessmentID string
	Goal         string
	Messages     []llm.Message
	ToolResults  []ToolResult
	Iteration    int
	MaxIter      int
	Done         bool
	TokensUsed   int
}

// ToolResult 工具执行结果
type ToolResult struct {
	ToolName   string
	Input      string
	Output     string
	Severity   string
	TokensUsed int
}

// NewAgentContext 创建 Agent 上下文
func NewAgentContext(assessmentID, goal, systemPrompt string, maxIter int) *AgentContext {
	ctx := &AgentContext{
		AssessmentID: assessmentID,
		Goal:         goal,
		MaxIter:      maxIter,
		Messages:     []llm.Message{},
	}
	if systemPrompt != "" {
		ctx.Messages = append(ctx.Messages, llm.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	ctx.Messages = append(ctx.Messages, llm.Message{
		Role:    "user",
		Content: goal,
	})
	return ctx
}

// AddAssistantMessage 添加助手消息
func (c *AgentContext) AddAssistantMessage(msg llm.Message) {
	c.Messages = append(c.Messages, msg)
}

// AddToolResult 添加工具执行结果消息
func (c *AgentContext) AddToolResult(toolCallID, result string) {
	c.Messages = append(c.Messages, llm.Message{
		Role:       "tool",
		ToolCallID: toolCallID,
		Content:    result,
	})
}

// ShouldContinue 判断是否继续循环
func (c *AgentContext) ShouldContinue() bool {
	return !c.Done && c.Iteration < c.MaxIter
}
