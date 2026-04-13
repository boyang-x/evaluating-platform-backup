package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type evaluationTargetConfig struct {
	ConnectorType string            `json:"connector_type"`
	BaseURL       string            `json:"base_url"`
	APIKey        string            `json:"api_key"`
	Model         string            `json:"model"`
	Headers       map[string]string `json:"headers,omitempty"`
}

type evaluationTargetClient struct {
	cfg        evaluationTargetConfig
	httpClient *http.Client
}

type targetResponse struct {
	Content    string
	RawBody    string
	StatusCode int
}

func newEvaluationTargetClient(cfg evaluationTargetConfig, timeout time.Duration) *evaluationTargetClient {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &evaluationTargetClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *evaluationTargetClient) SendPrompt(ctx context.Context, prompt string) (*targetResponse, error) {
	connectorType := strings.ToLower(strings.TrimSpace(c.cfg.ConnectorType))
	switch connectorType {
	case "", "openai":
		return c.sendOpenAI(ctx, prompt)
	case "dify":
		return c.sendDify(ctx, prompt)
	case "custom", "agent":
		return c.sendCustom(ctx, prompt)
	default:
		return nil, fmt.Errorf("unsupported evaluation target connector_type: %s", c.cfg.ConnectorType)
	}
}

func (c *evaluationTargetClient) sendOpenAI(ctx context.Context, prompt string) (*targetResponse, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model       string    `json:"model"`
		Messages    []message `json:"messages"`
		Temperature float64   `json:"temperature,omitempty"`
	}

	baseURL := strings.TrimRight(strings.TrimSpace(c.cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("evaluation target base_url is required")
	}
	if strings.TrimSpace(c.cfg.Model) == "" {
		return nil, fmt.Errorf("evaluation target model is required for openai connector")
	}

	body, _ := json.Marshal(request{
		Model: c.cfg.Model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create openai target request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.cfg.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	for key, value := range c.cfg.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send openai target request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &targetResponse{
			RawBody:    string(rawBody),
			StatusCode: resp.StatusCode,
		}, fmt.Errorf("openai target API error: %d", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	content := ""
	if err := json.Unmarshal(rawBody, &parsed); err == nil && len(parsed.Choices) > 0 {
		content = parsed.Choices[0].Message.Content
	}
	return &targetResponse{
		Content:    content,
		RawBody:    string(rawBody),
		StatusCode: resp.StatusCode,
	}, nil
}

func (c *evaluationTargetClient) sendDify(ctx context.Context, prompt string) (*targetResponse, error) {
	type request struct {
		Inputs       map[string]interface{} `json:"inputs"`
		Query        string                 `json:"query"`
		ResponseMode string                 `json:"response_mode"`
		User         string                 `json:"user"`
	}

	baseURL := strings.TrimRight(strings.TrimSpace(c.cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("evaluation target base_url is required")
	}

	body, _ := json.Marshal(request{
		Inputs:       map[string]interface{}{},
		Query:        prompt,
		ResponseMode: "blocking",
		User:         "ccbos-mcp",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat-messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create dify target request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.cfg.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	for key, value := range c.cfg.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send dify target request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &targetResponse{
			RawBody:    string(rawBody),
			StatusCode: resp.StatusCode,
		}, fmt.Errorf("dify target API error: %d", resp.StatusCode)
	}

	var parsed struct {
		Answer string `json:"answer"`
	}
	content := ""
	if err := json.Unmarshal(rawBody, &parsed); err == nil {
		content = parsed.Answer
	}
	return &targetResponse{
		Content:    content,
		RawBody:    string(rawBody),
		StatusCode: resp.StatusCode,
	}, nil
}

func (c *evaluationTargetClient) sendCustom(ctx context.Context, prompt string) (*targetResponse, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Messages []message `json:"messages"`
	}

	baseURL := strings.TrimSpace(c.cfg.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("evaluation target base_url is required")
	}

	body, _ := json.Marshal(request{
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create custom target request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.cfg.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	for key, value := range c.cfg.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send custom target request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &targetResponse{
			RawBody:    string(rawBody),
			StatusCode: resp.StatusCode,
		}, fmt.Errorf("custom target API error: %d", resp.StatusCode)
	}

	var parsed map[string]interface{}
	content := ""
	if err := json.Unmarshal(rawBody, &parsed); err == nil {
		switch {
		case parsed["content"] != nil:
			content = fmt.Sprintf("%v", parsed["content"])
		case parsed["response"] != nil:
			content = fmt.Sprintf("%v", parsed["response"])
		case parsed["message"] != nil:
			content = fmt.Sprintf("%v", parsed["message"])
		default:
			content = string(rawBody)
		}
	}
	if content == "" {
		content = string(rawBody)
	}
	return &targetResponse{
		Content:    content,
		RawBody:    string(rawBody),
		StatusCode: resp.StatusCode,
	}, nil
}
