package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Client OpenAI-compatible LLM 客户端
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient 创建 LLM 客户端
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
				TLSHandshakeTimeout:   15 * time.Second,
				ResponseHeaderTimeout: 60 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				IdleConnTimeout:       30 * time.Second,
				ForceAttemptHTTP2:     false,
				DisableKeepAlives:     true,
			},
		},
	}
}

// Chat 发送对话请求
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var resp *http.Response
	for attempt := 0; attempt < 2; attempt++ {
		httpReq, reqErr := http.NewRequestWithContext(ctx, "POST",
			c.baseURL+"/chat/completions", bytes.NewReader(body))
		if reqErr != nil {
			return nil, fmt.Errorf("create request: %w", reqErr)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err = c.httpClient.Do(httpReq)
		if err == nil {
			break
		}
		if attempt == 1 || !isRetryableTransportError(err) {
			return nil, fmt.Errorf("send request: %w", err)
		}
		select {
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(b))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// StreamChat 流式对话，通过 channel 推送增量内容
func (c *Client) StreamChat(ctx context.Context, req *ChatRequest, ch chan<- StreamChunk) error {
	req.Stream = true
	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		select {
		case ch <- chunk:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return scanner.Err()
}

func isRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "bad record mac"),
		strings.Contains(text, "unexpected eof"),
		strings.Contains(text, " eof"),
		strings.Contains(text, "connection reset"),
		strings.Contains(text, "broken pipe"),
		strings.Contains(text, "stream error"):
		return true
	default:
		return false
	}
}
