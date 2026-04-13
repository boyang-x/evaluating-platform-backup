package mcptools

// MCP Server/Client 集成测试
// 启动真实的 SSE Server，通过 MCP Client 进行端到端验证。

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// setupIntegration 启动 MCP Server + Client，返回已初始化的 client
func setupIntegration(t *testing.T) *client.Client {
	t.Helper()

	port := startTestMCPServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	c, err := client.NewSSEMCPClient(fmt.Sprintf("http://localhost:%d/sse", port))
	if err != nil {
		t.Fatalf("NewSSEMCPClient: %v", err)
	}
	if err := c.Start(ctx); err != nil {
		t.Fatalf("client.Start: %v", err)
	}
	if _, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test-client", Version: "1.0"},
		},
	}); err != nil {
		t.Fatalf("client.Initialize: %v", err)
	}

	return c
}

// TestMCPIntegration_ServerStartsSuccessfully verifies the MCP server starts and completes handshake
func TestMCPIntegration_ServerStartsSuccessfully(t *testing.T) {
	// If setupIntegration succeeds without fatal, the server is running and handshake completed
	_ = setupIntegration(t)
}
