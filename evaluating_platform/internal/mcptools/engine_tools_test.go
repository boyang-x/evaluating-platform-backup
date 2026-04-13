package mcptools

// Bug condition exploration tests for enterprise-content-compliance-fix spec.
// These tests MUST FAIL on unfixed code — failure confirms the bugs exist.
//
// Bug A: list_templates with empty expert_id should return published templates,
// not "invalid expert_id" error.
//
// Bug B: enhance_payloads with strategy="none" and empty expert_id should succeed,
// since the none strategy does not use expert_id at all.
//
// Validates: Requirements 1.1, 1.3, 1.4, 1.5

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// TestListTemplates_EmptyExpertID_ShouldNotReturnInvalidError verifies that
// calling list_templates with an empty expert_id does NOT return
// "invalid expert_id" error. On UNFIXED code this test FAILS because
// uuid.Parse("") fails and the handler returns "invalid expert_id"
// instead of falling back to ListPublished.
//
// Validates: Requirements 1.1
func TestListTemplates_EmptyExpertID_ShouldNotReturnInvalidError(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerListTemplates(s, nil) // nil repo — only branching logic is tested

	// On fixed code, empty expert_id routes to ListPublished which panics with nil repo.
	// We recover the panic — if it came from ListPublished, the fix is working.
	// On unfixed code, no panic occurs (uuid.Parse("") returns error gracefully).
	var panicValue interface{}
	func() {
		defer func() { panicValue = recover() }()
		resp := callTool(t, s, "list_templates", map[string]interface{}{
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
					t.Errorf("Bug confirmed: list_templates(expert_id=\"\") returned "+
						"\"invalid expert_id\" error instead of falling back to published templates")
				}
			}
		}
	}()

	// If we got a panic from ListPublished (nil repo), the fix is working correctly.
	if panicValue != nil {
		t.Logf("Fix verified: handler routed to ListPublished path (nil repo panic expected in test)")
	}
}

// TestEnhancePayloads_NoneStrategy_NoExpertID_ShouldSucceed verifies that
// calling enhance_payloads with strategy="none" and expert_id completely omitted
// succeeds, since the none strategy does not use expert_id at all.
//
// The real bug is that expert_id is marked Required() in the tool schema,
// which causes the LLM Agent to refuse to call the tool when no expert_id
// is available (enterprise user context). The mcp-go framework doesn't
// enforce Required() at runtime, but the LLM sees it in the JSON schema
// and won't call the tool without it.
//
// This test verifies that expert_id is NOT in the tool's required parameters,
// which is the expected behavior after the fix.
//
// On UNFIXED code: test FAILS because expert_id IS in the required array.
//
// Validates: Requirements 1.3
func TestEnhancePayloads_NoneStrategy_NoExpertID_ShouldSucceed(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	store := NewSessionStore(5 * time.Minute)

	registerEnhancePayloads(s, nil, store)

	// Use the MCP tools/list method to inspect the tool schema
	listReq, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("marshal list request: %v", err)
	}

	resp := s.HandleMessage(context.Background(), listReq)
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var listResp struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Required []string `json:"required"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &listResp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}

	// Find the enhance_payloads tool
	var found bool
	for _, tool := range listResp.Result.Tools {
		if tool.Name == "enhance_payloads" {
			found = true
			// Check if expert_id is in the required array
			for _, req := range tool.InputSchema.Required {
				if req == "expert_id" {
					t.Errorf("Bug confirmed: enhance_payloads tool schema marks expert_id as Required, "+
						"but none strategy does not need expert_id. "+
						"The LLM Agent will refuse to call this tool without expert_id in enterprise context. "+
						"Required params: %v", tool.InputSchema.Required)
				}
			}
			break
		}
	}
	if !found {
		t.Fatalf("enhance_payloads tool not found in tools/list response")
	}
}

// ─── Preservation property tests ─────────────────────────────────────────────
// These tests MUST PASS on unfixed code to capture baseline behavior that must
// be preserved after the bugfix is applied.
//
// Validates: Requirements 3.1, 3.2, 3.5, 3.6

// TestPreservation_ListTemplates_InvalidExpertID verifies that calling
// list_templates with a non-empty but invalid UUID string returns
// "invalid expert_id" error. This behaviour must be preserved after the fix.
//
// Validates: Requirements 3.1
func TestPreservation_ListTemplates_InvalidExpertID(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerListTemplates(s, nil)

	invalidIDs := []string{
		"not-a-uuid",
		"12345",
		"abc",
		"hello world",
	}

	for _, id := range invalidIDs {
		t.Run("expert_id="+id, func(t *testing.T) {
			resp := callTool(t, s, "list_templates", map[string]interface{}{
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
			}
			if err := json.Unmarshal(raw, &rpcResp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}

			if !rpcResp.Result.IsError {
				t.Errorf("expected IsError=true for invalid expert_id=%q, got false", id)
			}

			found := false
			for _, c := range rpcResp.Result.Content {
				if c.Text == "invalid expert_id" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error text \"invalid expert_id\" for expert_id=%q", id)
			}
		})
	}
}

// TestPreservation_EnhancePayloads_NoneStrategy_WithExpertID verifies that
// enhance_payloads with strategy="none" and a valid expert_id works correctly.
// The none strategy sets EnhancedText = CombinedText for all payloads.
//
// Validates: Requirements 3.5
func TestPreservation_EnhancePayloads_NoneStrategy_WithExpertID(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	store := NewSessionStore(5 * time.Minute)

	// Pre-populate session with test payloads
	store.SetPayloads("test-session", []PayloadItem{
		{Index: 1, OriginalContent: "sample1", CombinedText: "template+sample1"},
		{Index: 2, OriginalContent: "sample2", CombinedText: "template+sample2"},
	})

	registerEnhancePayloads(s, nil, store)

	resp := callTool(t, s, "enhance_payloads", map[string]interface{}{
		"session_id": "test-session",
		"strategy":   "none",
		"expert_id":  "11111111-1111-1111-1111-111111111111",
	})

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

	if rpcResp.Result == nil {
		t.Fatalf("expected non-nil result")
	}
	if rpcResp.Result.IsError {
		t.Errorf("expected success for none strategy with valid expert_id")
	}

	// Verify the response contains total_count=2
	for _, c := range rpcResp.Result.Content {
		if c.Type == "text" {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(c.Text), &result); err == nil {
				if count, ok := result["total_count"].(float64); ok && count != 2 {
					t.Errorf("expected total_count=2, got %v", count)
				}
			}
		}
	}

	// Verify payloads in store have EnhancedText == CombinedText
	payloads, err := store.GetPayloads("test-session")
	if err != nil {
		t.Fatalf("get payloads: %v", err)
	}
	for _, p := range payloads {
		if p.EnhancedText != p.CombinedText {
			t.Errorf("payload %d: EnhancedText=%q != CombinedText=%q", p.Index, p.EnhancedText, p.CombinedText)
		}
	}
}

// TestPreservation_ListTemplates_ValidExpertID_ReachesListByExpert verifies that
// list_templates with a valid expert_id UUID reaches the ListByExpert path.
// With nil repo, this panics — proving the correct code path is taken.
//
// Validates: Requirements 3.1
func TestPreservation_ListTemplates_ValidExpertID_ReachesListByExpert(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerListTemplates(s, nil) // nil repo — will panic when ListByExpert is called

	var panicValue interface{}
	func() {
		defer func() { panicValue = recover() }()
		callTool(t, s, "list_templates", map[string]interface{}{
			"expert_id": "11111111-1111-1111-1111-111111111111",
		})
	}()

	// With nil repo, reaching ListByExpert causes a nil pointer panic.
	// This proves the valid expert_id path is taken.
	if panicValue == nil {
		t.Errorf("expected panic from nil repo ListByExpert call, but no panic occurred")
	}
}

// TestPreservation_GetTemplate_Unchanged verifies that get_template with an
// invalid template_id returns "invalid template_id" error. This tool should
// not be affected by the fix.
//
// Validates: Requirements 3.6
func TestPreservation_GetTemplate_Unchanged(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerGetTemplate(s, nil)

	resp := callTool(t, s, "get_template", map[string]interface{}{
		"template_id": "not-a-uuid",
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
	}
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !rpcResp.Result.IsError {
		t.Errorf("expected IsError=true for invalid template_id")
	}

	found := false
	for _, c := range rpcResp.Result.Content {
		if c.Text == "invalid template_id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error text \"invalid template_id\"")
	}
}
