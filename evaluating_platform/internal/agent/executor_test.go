package agent

// executor 内部辅助函数单元测试
// 直接测试 extractTextFromResult 和 extractSeverity，无需网络连接。

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// ─── extractTextFromResult ────────────────────────────────────────────────────

func TestExtractTextFromResult_TextContent(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "hello world"},
		},
	}
	got := extractTextFromResult(result)
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestExtractTextFromResult_NilResult(t *testing.T) {
	got := extractTextFromResult(nil)
	if got != "" {
		t.Errorf("expected empty string for nil result, got %q", got)
	}
}

func TestExtractTextFromResult_EmptyContent(t *testing.T) {
	result := &mcp.CallToolResult{Content: []mcp.Content{}}
	got := extractTextFromResult(result)
	if got != "" {
		t.Errorf("expected empty string for empty content, got %q", got)
	}
}

func TestExtractTextFromResult_ReturnsFirstTextContent(t *testing.T) {
	// 多个 TextContent 项，应返回第一个
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "first"},
			mcp.TextContent{Type: "text", Text: "second"},
		},
	}
	got := extractTextFromResult(result)
	if got != "first" {
		t.Errorf("expected 'first', got %q", got)
	}
}

func TestExtractTextFromResult_ImageContentReturnsEmpty(t *testing.T) {
	// ImageContent 不包含文本，应返回空字符串
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.ImageContent{Type: "image", Data: "base64...", MIMEType: "image/png"},
		},
	}
	got := extractTextFromResult(result)
	if got != "" {
		t.Errorf("expected empty string for image-only content, got %q", got)
	}
}

// ─── extractSeverity ──────────────────────────────────────────────────────────

func TestExtractSeverity_AllValidValues(t *testing.T) {
	cases := []struct {
		jsonStr string
		want    string
	}{
		{`{"severity":"critical","success":true}`, "critical"},
		{`{"severity":"high","data":{}}`, "high"},
		{`{"severity":"medium","evidence":"..."}`, "medium"},
		{`{"severity":"low"}`, "low"},
		{`{"severity":"info","data":null}`, "info"},
	}

	for _, c := range cases {
		got := extractSeverity(c.jsonStr)
		if got != c.want {
			t.Errorf("extractSeverity(%q) = %q, want %q", c.jsonStr, got, c.want)
		}
	}
}

func TestExtractSeverity_MissingSeverity(t *testing.T) {
	got := extractSeverity(`{"success":true,"evidence":"","data":{}}`)
	if got != "" {
		t.Errorf("expected empty string when severity missing, got %q", got)
	}
}

func TestExtractSeverity_InvalidJSON(t *testing.T) {
	got := extractSeverity("not valid json { }")
	if got != "" {
		t.Errorf("expected empty string for invalid JSON, got %q", got)
	}
}

func TestExtractSeverity_EmptyString(t *testing.T) {
	got := extractSeverity("")
	if got != "" {
		t.Errorf("expected empty string for empty input, got %q", got)
	}
}

func TestExtractSeverity_NonStringSeverity(t *testing.T) {
	// severity 存在但值为数字而非字符串
	got := extractSeverity(`{"severity":3}`)
	if got != "" {
		t.Errorf("expected empty string when severity is non-string, got %q", got)
	}
}
