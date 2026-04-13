package mcptools

// Bug condition exploration tests for enterprise-assessment-bugs spec.
// These tests MUST FAIL on unfixed code — failure confirms the bugs exist.
//
// Bug 2: list_attack_samples with empty expert_id should return published samples,
// not "invalid expert_id" error.
//
// Validates: Requirements 1.3

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// callTool is a helper that sends a tools/call JSON-RPC request via HandleMessage.
func callTool(t *testing.T, s *server.MCPServer, toolName string, args map[string]interface{}) mcp.JSONRPCMessage {
	t.Helper()
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	}
	raw, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  params,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return s.HandleMessage(context.Background(), raw)
}

// TestListAttackSamples_EmptyExpertID_ShouldNotReturnInvalidError verifies that
// calling list_attack_samples with an empty expert_id does NOT return
// "invalid expert_id" error. On UNFIXED code this test FAILS because
// uuid.Parse("") fails immediately and the handler returns the error
// instead of falling back to ListPublished.
//
// On FIXED code: the handler takes the ListPublished path. With nil repo it
// panics (expected in test-only scenario), which proves the branching is correct.
//
// Validates: Requirements 1.3
func TestListAttackSamples_EmptyExpertID_ShouldNotReturnInvalidError(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerSampleQueryTools(s, nil, nil) // nil repo — only branching logic is tested

	// On fixed code, empty expert_id routes to ListPublished which panics with nil repo.
	// We recover the panic — if it came from ListPublished, the fix is working.
	// On unfixed code, no panic occurs (uuid.Parse("") returns error gracefully).
	var panicValue interface{}
	func() {
		defer func() { panicValue = recover() }()
		resp := callTool(t, s, "list_attack_samples", map[string]interface{}{
			"expert_id": "",
		})

		// If we reach here without panic, check the response for "invalid expert_id"
		raw, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}

		var rpcResp struct {
			Result *struct {
				IsError bool `json:"isError"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal(raw, &rpcResp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		if rpcResp.Result != nil {
			for _, c := range rpcResp.Result.Content {
				if c.Text == "invalid expert_id" {
					t.Errorf("Bug confirmed: list_attack_samples(expert_id=\"\") returned "+
						"\"invalid expert_id\" error instead of falling back to published samples")
				}
			}
		}
	}()

	// If we got a panic from ListPublished (nil repo), the fix is working correctly.
	// The handler took the ListPublished path instead of the uuid.Parse error path.
	if panicValue != nil {
		t.Logf("Fix verified: handler routed to ListPublished path (nil repo panic expected in test)")
	}
}

// ─── Preservation property tests ─────────────────────────────────────────────
// These tests MUST PASS on unfixed code to capture baseline behavior that must
// be preserved after the bugfix is applied.
//
// Validates: Requirements 3.1, 3.3, 3.4

// TestPreservation_ListAttackSamples_InvalidExpertID verifies that calling
// list_attack_samples with a non-empty but invalid UUID string returns
// "invalid expert_id" error. This behaviour must be preserved after the fix
// (only empty expert_id should trigger the fallback to ListPublished).
//
// Validates: Requirements 3.3
func TestPreservation_ListAttackSamples_InvalidExpertID(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerSampleQueryTools(s, nil, nil) // nil repo — only UUID validation is exercised

	invalidIDs := []string{
		"not-a-uuid",
		"12345",
		"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
		"abc",
		"hello world",
	}

	for _, id := range invalidIDs {
		t.Run("expert_id="+id, func(t *testing.T) {
			resp := callTool(t, s, "list_attack_samples", map[string]interface{}{
				"expert_id": id,
			})

			raw, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}

			var rpcResp struct {
				Result struct {
					IsError bool `json:"isError"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"result"`
				Error *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(raw, &rpcResp); err != nil {
				t.Fatalf("unmarshal response: %v\nraw: %s", err, string(raw))
			}

			if rpcResp.Error != nil {
				t.Fatalf("unexpected JSON-RPC error: code=%d msg=%s", rpcResp.Error.Code, rpcResp.Error.Message)
			}

			// The handler must flag the result as an error.
			if !rpcResp.Result.IsError {
				t.Errorf("expected IsError=true for invalid expert_id=%q, got false", id)
			}

			// The error text must be "invalid expert_id".
			found := false
			for _, c := range rpcResp.Result.Content {
				if c.Text == "invalid expert_id" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error text \"invalid expert_id\" for expert_id=%q, got content: %+v", id, rpcResp.Result.Content)
			}
		})
	}
}

// TestPreservation_GetAttackSample_InvalidSampleID verifies that calling
// get_attack_sample with an invalid UUID string returns "invalid sample_id"
// error. This behaviour must be preserved after the fix.
//
// Validates: Requirements 3.4
func TestPreservation_GetAttackSample_InvalidSampleID(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerSampleQueryTools(s, nil, nil) // nil repo — only UUID validation is exercised

	invalidIDs := []string{
		"not-a-uuid",
		"12345",
		"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
		"abc",
		"hello world",
	}

	for _, id := range invalidIDs {
		t.Run("sample_id="+id, func(t *testing.T) {
			resp := callTool(t, s, "get_attack_sample", map[string]interface{}{
				"sample_id": id,
			})

			raw, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}

			var rpcResp struct {
				Result struct {
					IsError bool `json:"isError"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"result"`
				Error *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(raw, &rpcResp); err != nil {
				t.Fatalf("unmarshal response: %v\nraw: %s", err, string(raw))
			}

			if rpcResp.Error != nil {
				t.Fatalf("unexpected JSON-RPC error: code=%d msg=%s", rpcResp.Error.Code, rpcResp.Error.Message)
			}

			// The handler must flag the result as an error.
			if !rpcResp.Result.IsError {
				t.Errorf("expected IsError=true for invalid sample_id=%q, got false", id)
			}

			// The error text must be "invalid sample_id".
			found := false
			for _, c := range rpcResp.Result.Content {
				if c.Text == "invalid sample_id" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error text \"invalid sample_id\" for sample_id=%q, got content: %+v", id, rpcResp.Result.Content)
			}
		})
	}
}

// TestPreservation_ListAttackSamples_NonEmptyInvalidExpertID_StillErrors is an
// explicit preservation check: after the fix adds a fallback for empty expert_id,
// non-empty invalid UUIDs must STILL return "invalid expert_id" error.
//
// Validates: Requirements 3.3
func TestPreservation_ListAttackSamples_NonEmptyInvalidExpertID_StillErrors(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerSampleQueryTools(s, nil, nil)

	resp := callTool(t, s, "list_attack_samples", map[string]interface{}{
		"expert_id": "not-a-uuid",
	})

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var rpcResp struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, string(raw))
	}

	if rpcResp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: code=%d msg=%s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Must be an error result.
	if !rpcResp.Result.IsError {
		t.Errorf("expected IsError=true for non-empty invalid expert_id, got false")
	}

	// Must contain "invalid expert_id" text.
	found := false
	for _, c := range rpcResp.Result.Content {
		if c.Text == "invalid expert_id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected \"invalid expert_id\" error text, got: %+v", rpcResp.Result.Content)
	}
}
