package externalmcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/client"
	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	defaultSSEEndpointTimeout = 30 * time.Second
	defaultSSEResponseTimeout = 60 * time.Second
)

func NewTimeoutAwareSSEClient(
	baseURL string,
	responseTimeout time.Duration,
	headers map[string]string,
	httpClient *http.Client,
) (*client.Client, error) {
	transport, err := newTimeoutAwareSSETransport(baseURL, responseTimeout, headers, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE transport: %w", err)
	}
	return client.NewClient(transport), nil
}

type timeoutAwareSSETransport struct {
	baseURL    *url.URL
	endpoint   *url.URL
	httpClient *http.Client

	responseTimeout time.Duration

	mu        sync.RWMutex
	responses map[string]chan *mcptransport.JSONRPCResponse

	headers      map[string]string
	endpointChan chan struct{}
	endpointOnce sync.Once

	notifyMu       sync.RWMutex
	onNotification func(mcp.JSONRPCNotification)

	started         atomic.Bool
	closed          atomic.Bool
	cancelSSEStream context.CancelFunc
	protocolVersion atomic.Value
}

func newTimeoutAwareSSETransport(
	baseURL string,
	responseTimeout time.Duration,
	headers map[string]string,
	httpClient *http.Client,
) (*timeoutAwareSSETransport, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	transport := &timeoutAwareSSETransport{
		baseURL:         parsedURL,
		httpClient:      httpClient,
		responseTimeout: responseTimeout,
		responses:       make(map[string]chan *mcptransport.JSONRPCResponse),
		headers:         cloneHeaders(headers),
		endpointChan:    make(chan struct{}),
	}
	return transport, nil
}

func (c *timeoutAwareSSETransport) Start(ctx context.Context) error {
	if c.started.Load() {
		return nil
	}

	streamCtx, cancel := context.WithCancel(ctx)
	c.cancelSSEStream = cancel

	req, err := http.NewRequestWithContext(streamCtx, http.MethodGet, c.baseURL.String(), nil)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to connect to SSE stream: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, body)
	}

	go c.readSSE(resp.Body)

	endpointTimeout, err := c.waitTimeout(ctx, defaultSSEEndpointTimeout)
	if err != nil {
		cancel()
		return err
	}

	timer := time.NewTimer(endpointTimeout)
	defer timer.Stop()

	select {
	case <-c.endpointChan:
	case <-ctx.Done():
		cancel()
		return fmt.Errorf("context cancelled while waiting for endpoint: %w", ctx.Err())
	case <-timer.C:
		cancel()
		return fmt.Errorf("timeout waiting for endpoint after %v", endpointTimeout)
	}

	c.started.Store(true)
	return nil
}

func (c *timeoutAwareSSETransport) SendRequest(
	ctx context.Context,
	request mcptransport.JSONRPCRequest,
) (*mcptransport.JSONRPCResponse, error) {
	if !c.started.Load() {
		return nil, fmt.Errorf("transport not started yet")
	}
	if c.closed.Load() {
		return nil, fmt.Errorf("transport has been closed")
	}
	if c.endpoint == nil {
		return nil, fmt.Errorf("endpoint not received")
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint.String(), bytes.NewReader(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if v := c.protocolVersion.Load(); v != nil {
		if version, ok := v.(string); ok && version != "" {
			req.Header.Set(mcptransport.HeaderKeyProtocolVersion, version)
		}
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	for k, v := range request.Header {
		if _, ok := req.Header[k]; !ok {
			req.Header[k] = v
		}
	}

	idKey := request.ID.String()
	responseChan := make(chan *mcptransport.JSONRPCResponse, 1)
	c.mu.Lock()
	c.responses[idKey] = responseChan
	c.mu.Unlock()
	deleteResponseChan := func() {
		c.mu.Lock()
		delete(c.responses, idKey)
		c.mu.Unlock()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		deleteResponseChan()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		deleteResponseChan()
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		deleteResponseChan()
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, body)
	}

	responseTimeout, err := c.waitTimeout(ctx, defaultSSEResponseTimeout)
	if err != nil {
		deleteResponseChan()
		return nil, err
	}

	timer := time.NewTimer(responseTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		deleteResponseChan()
		return nil, ctx.Err()
	case <-timer.C:
		deleteResponseChan()
		return nil, fmt.Errorf("timeout waiting for SSE response after %v", responseTimeout)
	case response, ok := <-responseChan:
		if ok {
			return response, nil
		}
		return nil, fmt.Errorf("connection has been closed")
	}
}

func (c *timeoutAwareSSETransport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	if c.endpoint == nil {
		return fmt.Errorf("endpoint not received")
	}

	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint.String(), bytes.NewReader(notificationBytes))
	if err != nil {
		return fmt.Errorf("failed to create notification request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if v := c.protocolVersion.Load(); v != nil {
		if version, ok := v.(string); ok && version != "" {
			req.Header.Set(mcptransport.HeaderKeyProtocolVersion, version)
		}
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notification failed with status %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (c *timeoutAwareSSETransport) SetNotificationHandler(handler func(mcp.JSONRPCNotification)) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.onNotification = handler
}

func (c *timeoutAwareSSETransport) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	if c.cancelSSEStream != nil {
		c.cancelSSEStream()
	}

	c.mu.Lock()
	for _, ch := range c.responses {
		close(ch)
	}
	c.responses = make(map[string]chan *mcptransport.JSONRPCResponse)
	c.mu.Unlock()
	return nil
}

func (c *timeoutAwareSSETransport) GetSessionId() string {
	return ""
}

func (c *timeoutAwareSSETransport) SetProtocolVersion(version string) {
	c.protocolVersion.Store(version)
}

func (c *timeoutAwareSSETransport) readSSE(reader io.ReadCloser) {
	defer reader.Close()

	br := bufio.NewReader(reader)
	var event string
	var data strings.Builder

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF && data.Len() > 0 {
				c.handleSSEEvent(event, data.String())
			}
			return
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if data.Len() == 0 {
				event = ""
				continue
			}
			c.handleSSEEvent(event, data.String())
			event = ""
			data.Reset()
			continue
		}

		if after, ok := strings.CutPrefix(line, "event:"); ok {
			event = strings.TrimSpace(after)
			continue
		}
		if after, ok := strings.CutPrefix(line, "data:"); ok {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(after))
		}
	}
}

func (c *timeoutAwareSSETransport) handleSSEEvent(event string, data string) {
	switch event {
	case "endpoint":
		endpoint, err := c.baseURL.Parse(data)
		if err != nil {
			return
		}
		if endpoint.Host != c.baseURL.Host {
			return
		}
		c.endpoint = endpoint
		c.endpointOnce.Do(func() {
			close(c.endpointChan)
		})
	case "", "message":
		var baseMessage mcptransport.JSONRPCResponse
		if err := json.Unmarshal([]byte(data), &baseMessage); err != nil {
			return
		}
		if baseMessage.ID.IsNil() {
			var notification mcp.JSONRPCNotification
			if err := json.Unmarshal([]byte(data), &notification); err != nil {
				return
			}
			c.notifyMu.RLock()
			handler := c.onNotification
			c.notifyMu.RUnlock()
			if handler != nil {
				handler(notification)
			}
			return
		}

		idKey := baseMessage.ID.String()
		c.mu.RLock()
		ch, exists := c.responses[idKey]
		c.mu.RUnlock()
		if exists {
			ch <- &baseMessage
			c.mu.Lock()
			delete(c.responses, idKey)
			c.mu.Unlock()
		}
	}
}

func (c *timeoutAwareSSETransport) waitTimeout(ctx context.Context, fallback time.Duration) (time.Duration, error) {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, ctx.Err()
		}
		return remaining, nil
	}
	if c.responseTimeout > 0 {
		return c.responseTimeout, nil
	}
	return fallback, nil
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(headers))
	for k, v := range headers {
		cloned[k] = v
	}
	return cloned
}
