package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CustomConnector 通用 HTTP 适配器，支持自定义 Header、Body 模板
type CustomConnector struct {
	cfg        *Config
	httpClient *http.Client
}

// NewCustomConnector 创建自定义连接器
func NewCustomConnector(cfg *Config) *CustomConnector {
	return &CustomConnector{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *CustomConnector) SendMessage(ctx context.Context, req *AssessRequest) (*AssessResponse, error) {
	// 将消息序列化为请求体
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	for k, v := range c.cfg.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, _ := io.ReadAll(resp.Body)
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	// 尝试提取 content 字段
	var result map[string]interface{}
	content := ""
	if err := json.Unmarshal(rawBody, &result); err == nil {
		if v, ok := result["content"]; ok {
			content = fmt.Sprintf("%v", v)
		} else if v, ok := result["response"]; ok {
			content = fmt.Sprintf("%v", v)
		} else if v, ok := result["message"]; ok {
			content = fmt.Sprintf("%v", v)
		}
	}

	return &AssessResponse{
		Content:    content,
		RawBody:    string(rawBody),
		StatusCode: resp.StatusCode,
		Headers:    headers,
	}, nil
}

func (c *CustomConnector) GetCapabilities() []string {
	return []string{"chat"}
}

func (c *CustomConnector) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.SendMessage(ctx, &AssessRequest{
		Messages: []Message{{Role: "user", Content: "ping"}},
	})
	return err
}
