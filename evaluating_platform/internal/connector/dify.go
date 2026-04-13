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

// DifyConnector 通过 Dify API 与 Dify 智能体交互
type DifyConnector struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

// NewDifyConnector 创建 Dify 连接器
func NewDifyConnector(cfg *Config) *DifyConnector {
	return &DifyConnector{
		endpoint: cfg.Endpoint,
		apiKey:   cfg.APIKey,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

type difyRequest struct {
	Inputs       map[string]interface{} `json:"inputs"`
	Query        string                 `json:"query"`
	ResponseMode string                 `json:"response_mode"`
	User         string                 `json:"user"`
}

type difyResponse struct {
	Answer string `json:"answer"`
	Event  string `json:"event"`
}

func (d *DifyConnector) SendMessage(ctx context.Context, req *AssessRequest) (*AssessResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("no messages")
	}

	lastMsg := req.Messages[len(req.Messages)-1]
	body := difyRequest{
		Inputs:       map[string]interface{}{},
		Query:        lastMsg.Content,
		ResponseMode: "blocking",
		User:         "security-eval",
	}

	jsonBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", d.endpoint+"/v1/chat-messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &AssessResponse{
			Content:    string(respBody),
			RawBody:    string(respBody),
			StatusCode: resp.StatusCode,
		}, fmt.Errorf("dify API error: %d", resp.StatusCode)
	}

	var difyResp difyResponse
	if err := json.Unmarshal(respBody, &difyResp); err != nil {
		return &AssessResponse{
			Content:    string(respBody),
			RawBody:    string(respBody),
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &AssessResponse{
		Content:    difyResp.Answer,
		RawBody:    string(respBody),
		StatusCode: resp.StatusCode,
	}, nil
}

func (d *DifyConnector) GetCapabilities() []string {
	return []string{"chat", "dify-agent"}
}

func (d *DifyConnector) HealthCheck() error {
	return nil
}
