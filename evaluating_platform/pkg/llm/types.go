package llm

// Message OpenAI-compatible 消息结构
type Message struct {
	Role       string     `json:"role"` // system | user | assistant | tool
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // function
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Tool 工具定义（传给 LLM 的 function schema）
type Tool struct {
	Type     string      `json:"type"` // function
	Function FunctionDef `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// ChatRequest 对话请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"` // auto | none | required
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 响应选项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"` // stop | tool_calls | length
}

// Usage Token 用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	ID      string        `json:"id"`
	Choices []StreamDelta `json:"choices"`
	Usage   *Usage        `json:"usage,omitempty"`
}

// StreamDelta 流式增量
type StreamDelta struct {
	Index        int          `json:"index"`
	Delta        DeltaMessage `json:"delta"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

// DeltaMessage 增量消息
type DeltaMessage struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
