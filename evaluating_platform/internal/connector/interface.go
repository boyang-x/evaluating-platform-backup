package connector

import "context"

// AssessRequest 向目标系统发送的评估请求
type AssessRequest struct {
	Messages []Message         `json:"messages"`
	Extra    map[string]string `json:"extra,omitempty"` // 额外参数
}

// Message 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AssessResponse 目标系统的响应
type AssessResponse struct {
	Content    string            `json:"content"`
	RawBody    string            `json:"raw_body"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// TargetConnector 目标连接器统一接口
type TargetConnector interface {
	// SendMessage 向目标系统发送消息并获取响应
	SendMessage(ctx context.Context, req *AssessRequest) (*AssessResponse, error)
	// GetCapabilities 返回目标系统能力列表
	GetCapabilities() []string
	// HealthCheck 健康检查
	HealthCheck() error
}

// Config 连接器配置
type Config struct {
	Type     string            `json:"type"` // openai | agent | custom
	Endpoint string            `json:"endpoint"`
	APIKey   string            `json:"api_key,omitempty"`
	Model    string            `json:"model,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

// NewConnector 工厂函数，根据配置创建连接器
func NewConnector(cfg *Config) TargetConnector {
	switch cfg.Type {
	case "openai":
		return NewOpenAIConnector(cfg)
	case "dify":
		return NewDifyConnector(cfg)
	case "custom", "agent":
		return NewCustomConnector(cfg)
	default:
		return NewOpenAIConnector(cfg)
	}
}
