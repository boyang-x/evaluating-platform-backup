package skill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type RunnerClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewRunnerClient(baseURL string, timeout time.Duration) *RunnerClient {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &RunnerClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *RunnerClient) Run(ctx context.Context, req *RuntimeRequest) (*RuntimeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal runner request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/run", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create runner request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call skill runner: %w", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Data  RuntimeResponse `json:"data"`
		Error string          `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode runner response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("skill runner returned %d: %s", resp.StatusCode, payload.Error)
	}
	if payload.Error != "" {
		return nil, fmt.Errorf("%s", payload.Error)
	}
	return &payload.Data, nil
}
