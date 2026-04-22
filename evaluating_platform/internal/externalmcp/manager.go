package externalmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
)

type TestResult struct {
	ServerName string   `json:"server_name"`
	ToolCount  int      `json:"tool_count"`
	ToolNames  []string `json:"tool_names"`
}

const hiddenRawProxyPrefix = "ext_ccbos_"

type runtimeClient struct {
	cfg        model.ExternalMCPServer
	client     *client.Client
	cancel     context.CancelFunc
	proxyNames []string
}

type Manager struct {
	serverRepo     *repository.ExternalMCPServerRepository
	toolRepo       *repository.ExternalMCPToolRepository
	internalServer *mcpserver.MCPServer

	mu      sync.RWMutex
	runtime map[uuid.UUID]*runtimeClient
}

func NewManager(
	serverRepo *repository.ExternalMCPServerRepository,
	toolRepo *repository.ExternalMCPToolRepository,
	internalServer *mcpserver.MCPServer,
) *Manager {
	return &Manager{
		serverRepo:     serverRepo,
		toolRepo:       toolRepo,
		internalServer: internalServer,
		runtime:        make(map[uuid.UUID]*runtimeClient),
	}
}

func (m *Manager) Bootstrap(ctx context.Context) error {
	servers, err := m.serverRepo.ListEnabled(ctx)
	if err != nil {
		return err
	}

	var errs []string
	for _, item := range servers {
		if err := m.SyncServer(ctx, item.ID); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", item.Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func (m *Manager) TestConnection(ctx context.Context, cfg *model.ExternalMCPServer) (*TestResult, error) {
	cli, cancel, err := m.connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer cli.Close()

	callCtx, callCancel := context.WithTimeout(ctx, time.Duration(maxInt(cfg.TimeoutSeconds, 5))*time.Second)
	defer callCancel()

	result, err := cli.ListTools(callCtx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list remote tools: %w", err)
	}

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)

	return &TestResult{
		ServerName: cfg.Name,
		ToolCount:  len(names),
		ToolNames:  names,
	}, nil
}

func (m *Manager) SyncServer(ctx context.Context, serverID uuid.UUID) error {
	cfg, err := m.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("external mcp server not found")
	}
	if !cfg.Enabled {
		return m.DisableServer(ctx, serverID)
	}

	runtime, err := m.reconnect(ctx, *cfg)
	if err != nil {
		_ = m.serverRepo.UpdateStatus(ctx, serverID, "error", err.Error(), cfg.LastSyncAt)
		return err
	}

	timeout := time.Duration(maxInt(cfg.TimeoutSeconds, 5)) * time.Second
	callCtx, callCancel := context.WithTimeout(ctx, timeout)
	defer callCancel()

	result, err := runtime.client.ListTools(callCtx, mcp.ListToolsRequest{})
	if err != nil {
		m.closeRuntime(serverID)
		_ = m.serverRepo.UpdateStatus(ctx, serverID, "error", err.Error(), cfg.LastSyncAt)
		return fmt.Errorf("list remote tools: %w", err)
	}

	syncedTools := make([]model.ExternalMCPTool, 0, len(result.Tools))
	for _, remoteTool := range result.Tools {
		schemaBytes := rawSchemaForTool(remoteTool)
		syncedTools = append(syncedTools, model.ExternalMCPTool{
			ID:             uuid.New(),
			ServerID:       serverID,
			RemoteToolName: remoteTool.Name,
			ProxyToolName:  buildProxyToolName(cfg.Namespace, remoteTool.Name),
			Description:    remoteTool.Description,
			InputSchema:    schemaBytes,
			CapabilityMetadata: mustJSON(map[string]interface{}{
				"source":           "external_mcp",
				"server_id":        serverID.String(),
				"server_name":      cfg.Name,
				"server_namespace": cfg.Namespace,
				"remote_tool_name": remoteTool.Name,
				"transport_type":   cfg.TransportType,
			}),
			SkillText: cfg.SkillPrompt,
			Enabled:   true,
		})
	}

	if err := m.toolRepo.ReplaceByServer(ctx, serverID, syncedTools); err != nil {
		return err
	}

	m.registerProxyTools(*cfg, syncedTools)

	now := time.Now()
	if err := m.serverRepo.UpdateStatus(ctx, serverID, "connected", "", &now); err != nil {
		return err
	}
	return nil
}

func (m *Manager) DisableServer(ctx context.Context, serverID uuid.UUID) error {
	m.unregisterProxyTools(serverID)
	m.closeRuntime(serverID)
	if err := m.toolRepo.DeleteByServer(ctx, serverID); err != nil {
		return err
	}
	return m.serverRepo.UpdateStatus(ctx, serverID, "disabled", "", nil)
}

func (m *Manager) RemoveServer(serverID uuid.UUID) {
	m.unregisterProxyTools(serverID)
	m.closeRuntime(serverID)
}

func (m *Manager) CallTool(ctx context.Context, serverID uuid.UUID, remoteToolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	runtime, err := m.ensureRuntime(ctx, serverID)
	if err != nil {
		return nil, err
	}

	result, err := runtime.client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      remoteToolName,
			Arguments: sanitizeForwardArgs(args),
		},
	})
	if err == nil {
		return result, nil
	}

	runtime, reconnectErr := m.reconnect(ctx, runtime.cfg)
	if reconnectErr != nil {
		return nil, fmt.Errorf("call external tool failed: %w; reconnect failed: %v", err, reconnectErr)
	}

	return runtime.client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      remoteToolName,
			Arguments: sanitizeForwardArgs(args),
		},
	})
}

func (m *Manager) ensureRuntime(ctx context.Context, serverID uuid.UUID) (*runtimeClient, error) {
	m.mu.RLock()
	existing := m.runtime[serverID]
	m.mu.RUnlock()
	if existing != nil {
		return existing, nil
	}

	cfg, err := m.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("external mcp server not found")
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("external mcp server is disabled")
	}
	return m.reconnect(ctx, *cfg)
}

func (m *Manager) reconnect(ctx context.Context, cfg model.ExternalMCPServer) (*runtimeClient, error) {
	m.closeRuntime(cfg.ID)

	cli, cancel, err := m.connect(ctx, &cfg)
	if err != nil {
		return nil, err
	}

	runtime := &runtimeClient{
		cfg:    cfg,
		client: cli,
		cancel: cancel,
	}

	m.mu.Lock()
	m.runtime[cfg.ID] = runtime
	m.mu.Unlock()
	return runtime, nil
}

func (m *Manager) connect(_ context.Context, cfg *model.ExternalMCPServer) (*client.Client, context.CancelFunc, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, nil, fmt.Errorf("base_url is required")
	}
	if cfg.TransportType != "" && cfg.TransportType != "sse" {
		return nil, nil, fmt.Errorf("unsupported transport_type: %s", cfg.TransportType)
	}

	headers, err := buildHeaders(cfg)
	if err != nil {
		return nil, nil, err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   time.Duration(maxInt(cfg.TimeoutSeconds, 5)) * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(maxInt(cfg.TimeoutSeconds, 5)) * time.Second,
			Proxy:                 http.ProxyFromEnvironment,
		},
	}

	cli, err := NewTimeoutAwareSSEClient(
		cfg.BaseURL,
		time.Duration(maxInt(cfg.TimeoutSeconds, 5))*time.Second,
		headers,
		httpClient,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create external mcp client: %w", err)
	}

	runtimeCtx, cancel := context.WithCancel(context.Background())
	if err := cli.Start(runtimeCtx); err != nil {
		cancel()
		_ = cli.Close()
		return nil, nil, fmt.Errorf("start external mcp client: %w", err)
	}

	initCtx, initCancel := context.WithTimeout(context.Background(), time.Duration(maxInt(cfg.TimeoutSeconds, 5))*time.Second)
	defer initCancel()

	_, err = cli.Initialize(initCtx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "ai-security-evaluator-external-mcp",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		cancel()
		_ = cli.Close()
		return nil, nil, fmt.Errorf("initialize external mcp client: %w", err)
	}

	return cli, cancel, nil
}

func (m *Manager) registerProxyTools(cfg model.ExternalMCPServer, tools []model.ExternalMCPTool) {
	proxyNames := make([]string, 0, len(tools))
	for _, item := range tools {
		proxyNames = append(proxyNames, item.ProxyToolName)
	}

	m.unregisterProxyTools(cfg.ID)

	for _, item := range tools {
		toolDef := mcp.Tool{
			Name:           item.ProxyToolName,
			Description:    proxyDescription(cfg, item),
			RawInputSchema: cloneJSON(item.InputSchema),
		}
		serverID := cfg.ID
		remoteName := item.RemoteToolName
		m.internalServer.AddTool(toolDef, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := m.CallTool(ctx, serverID, remoteName, req.GetArguments())
			if err != nil {
				return mcp.NewToolResultError("external MCP tool call failed: " + err.Error()), nil
			}
			return normalizeResult(result), nil
		})
	}

	m.mu.Lock()
	runtime := m.runtime[cfg.ID]
	if runtime != nil {
		runtime.proxyNames = proxyNames
		runtime.cfg = cfg
	}
	m.mu.Unlock()
}

func (m *Manager) unregisterProxyTools(serverID uuid.UUID) {
	m.mu.RLock()
	runtime := m.runtime[serverID]
	m.mu.RUnlock()
	if runtime == nil || len(runtime.proxyNames) == 0 {
		return
	}
	m.internalServer.DeleteTools(runtime.proxyNames...)
	m.mu.Lock()
	if current := m.runtime[serverID]; current != nil {
		current.proxyNames = nil
	}
	m.mu.Unlock()
}

func (m *Manager) closeRuntime(serverID uuid.UUID) {
	m.mu.Lock()
	runtime := m.runtime[serverID]
	delete(m.runtime, serverID)
	m.mu.Unlock()

	if runtime == nil {
		return
	}
	if runtime.cancel != nil {
		runtime.cancel()
	}
	if runtime.client != nil {
		_ = runtime.client.Close()
	}
}

func buildHeaders(cfg *model.ExternalMCPServer) (map[string]string, error) {
	headers := map[string]string{}
	switch cfg.AuthType {
	case "", "none":
	case "bearer":
		if strings.TrimSpace(cfg.AuthKey) == "" {
			return nil, fmt.Errorf("auth_key is required for bearer auth")
		}
		headers["Authorization"] = strings.TrimSpace(cfg.AuthPrefix) + " " + strings.TrimSpace(cfg.AuthKey)
		if strings.TrimSpace(cfg.AuthPrefix) == "" {
			headers["Authorization"] = strings.TrimSpace(cfg.AuthKey)
		}
	case "custom_header":
		if strings.TrimSpace(cfg.AuthHeader) == "" {
			return nil, fmt.Errorf("auth_header is required for custom_header auth")
		}
		if strings.TrimSpace(cfg.AuthKey) == "" {
			return nil, fmt.Errorf("auth_key is required for custom_header auth")
		}
		value := strings.TrimSpace(cfg.AuthKey)
		if prefix := strings.TrimSpace(cfg.AuthPrefix); prefix != "" {
			value = prefix + " " + value
		}
		headers[cfg.AuthHeader] = value
	default:
		return nil, fmt.Errorf("unsupported auth_type: %s", cfg.AuthType)
	}

	if value := strings.TrimSpace(cfg.UpstreamAPIKey); value != "" {
		headers["X-MCP-Upstream-Api-Key"] = value
	}
	if value := strings.TrimSpace(cfg.UpstreamBaseURL); value != "" {
		headers["X-MCP-Upstream-Base-Url"] = value
	}
	if value := strings.TrimSpace(cfg.UpstreamModel); value != "" {
		headers["X-MCP-Upstream-Model"] = value
	}
	if cfg.UpstreamTimeout > 0 {
		headers["X-MCP-Upstream-Timeout-Seconds"] = fmt.Sprintf("%d", cfg.UpstreamTimeout)
	}
	return headers, nil
}

var proxyNameSanitizer = regexp.MustCompile(`[^a-z0-9_]+`)

func buildProxyToolName(namespace, remoteName string) string {
	ns := sanitizeIdentifier(namespace)
	name := sanitizeIdentifier(remoteName)
	if ns == "" {
		ns = "external"
	}
	if name == "" {
		name = "tool"
	}
	return fmt.Sprintf("ext_%s_%s", ns, name)
}

func sanitizeIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = proxyNameSanitizer.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return value
}

func proxyDescription(cfg model.ExternalMCPServer, tool model.ExternalMCPTool) string {
	parts := []string{fmt.Sprintf("[外部 MCP:%s]", cfg.Name)}
	if strings.TrimSpace(tool.Description) != "" {
		parts = append(parts, tool.Description)
	} else {
		parts = append(parts, fmt.Sprintf("代理外部工具 %s。", tool.RemoteToolName))
	}
	parts = append(parts, fmt.Sprintf("源工具名: %s。", tool.RemoteToolName))
	if strings.TrimSpace(cfg.SkillPrompt) != "" {
		parts = append(parts, fmt.Sprintf("使用提示: %s", cfg.SkillPrompt))
	}
	return strings.Join(parts, " ")
}

func sanitizeForwardArgs(args map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(args))
	for key, value := range args {
		if key == "assessment_id" {
			continue
		}
		cloned[key] = value
	}
	return cloned
}

func normalizeResult(result *mcp.CallToolResult) *mcp.CallToolResult {
	if result == nil {
		return mcp.NewToolResultError("empty result from external MCP tool")
	}
	for _, content := range result.Content {
		if _, ok := content.(mcp.TextContent); ok {
			return result
		}
	}

	payload, _ := json.Marshal(result)
	if result.IsError {
		return mcp.NewToolResultError(string(payload))
	}
	return mcp.NewToolResultText(string(payload))
}

func rawSchemaForTool(tool mcp.Tool) json.RawMessage {
	if len(tool.RawInputSchema) > 0 {
		return cloneJSON(tool.RawInputSchema)
	}
	data, err := json.Marshal(tool.InputSchema)
	if err != nil || len(data) == 0 {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return data
}

func cloneJSON(data json.RawMessage) json.RawMessage {
	if len(data) == 0 {
		return nil
	}
	copied := make([]byte, len(data))
	copy(copied, data)
	return copied
}

func mustJSON(value map[string]interface{}) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

// IsLLMVisibleTool reports whether a tool should be exposed to orchestration LLMs.
func IsLLMVisibleTool(name string, _ string) bool {
	return !strings.HasPrefix(strings.TrimSpace(strings.ToLower(name)), hiddenRawProxyPrefix)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
