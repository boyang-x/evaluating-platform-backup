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

// OpenAIConnector 标准 OpenAI Chat Completion API 连接器
type OpenAIConnector struct {
	cfg        *Config
	httpClient *http.Client
}

// NewOpenAIConnector 创建 OpenAI 连接器
func NewOpenAIConnector(cfg *Config) *OpenAIConnector {
	return &OpenAIConnector{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *OpenAIConnector) SendMessage(ctx context.Context, req *AssessRequest) (*AssessResponse, error) {
	model := c.cfg.Model
	if model == "" {
		model = "gpt-4o"
	}

	type openAIMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type openAIReq struct {
		Model    string      `json:"model"`
		Messages []openAIMsg `json:"messages"`
	}

	msgs := make([]openAIMsg, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openAIMsg{Role: m.Role, Content: m.Content}
	}

	body, _ := json.Marshal(openAIReq{Model: model, Messages: msgs})
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.cfg.Endpoint+"/chat/completions", bytes.NewReader(body))
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

	if resp.StatusCode != http.StatusOK {
		return &AssessResponse{
			RawBody:    string(rawBody),
			StatusCode: resp.StatusCode,
			Headers:    headers,
		}, fmt.Errorf("target API error %d", resp.StatusCode)
	}

	// 解析 OpenAI 响应格式
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	content := ""
	if err := json.Unmarshal(rawBody, &result); err == nil && len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return &AssessResponse{
		Content:    content,
		RawBody:    string(rawBody),
		StatusCode: resp.StatusCode,
		Headers:    headers,
	}, nil
}

func (c *OpenAIConnector) GetCapabilities() []string {
	return []string{"chat", "completion", "function_calling"}
}

func (c *OpenAIConnector) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.SendMessage(ctx, &AssessRequest{
		Messages: []Message{{Role: "user", Content: "ping"}},
	})
	return err
}
