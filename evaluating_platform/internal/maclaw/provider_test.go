package maclaw

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestMCPGatewayFromClientUnwrapsPlatformGateways(t *testing.T) {
	upstream := &fakeMCPGateway{}
	client := GatewayClient(upstream)
	client = NewPlatformTargetGateway(client, nil, uuid.New())
	client = NewPlatformArtifactGateway(client, nil, uuid.New())
	client = NewPlatformResourceGateway(client, nil, uuid.New(), "tenant_1")

	gateway, ok := MCPGatewayFromClient(client)
	if !ok || gateway == nil {
		t.Fatalf("expected MCP gateway through platform wrappers")
	}
	items, err := gateway.ListMCPServers(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListMCPServers: %v", err)
	}
	if len(items) != 1 || items[0].ID != "mcp_1" {
		t.Fatalf("items = %#v", items)
	}
}

type fakeMCPGateway struct {
	GatewayClient
	MCPGateway
}

func (g *fakeMCPGateway) Enabled() bool { return true }

func (g *fakeMCPGateway) ListMCPServers(context.Context, int) ([]MCPServerView, error) {
	return []MCPServerView{{ID: "mcp_1", Kind: "remote", Name: "test"}}, nil
}
