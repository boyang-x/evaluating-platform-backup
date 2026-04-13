package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:18191/sse", "MCP SSE endpoint")
	question := flag.String("question", "", "Seed question to rewrite")
	variantCount := flag.Int("variant-count", 1, "How many variants to generate")
	upstreamBaseURL := flag.String("upstream-base-url", "", "Upstream LLM base URL override")
	upstreamAPIKey := flag.String("upstream-api-key", "", "Upstream LLM API key override")
	upstreamModel := flag.String("upstream-model", "", "Upstream LLM model override")
	upstreamTimeout := flag.Int("upstream-timeout", 60, "Upstream LLM timeout override in seconds")
	flag.Parse()

	if *question == "" {
		panic("question is required")
	}

	headers := map[string]string{}
	if *upstreamBaseURL != "" {
		headers["X-MCP-Upstream-Base-Url"] = *upstreamBaseURL
	}
	if *upstreamAPIKey != "" {
		headers["X-MCP-Upstream-Api-Key"] = *upstreamAPIKey
	}
	if *upstreamModel != "" {
		headers["X-MCP-Upstream-Model"] = *upstreamModel
	}
	if *upstreamTimeout > 0 {
		headers["X-MCP-Upstream-Timeout-Seconds"] = fmt.Sprintf("%d", *upstreamTimeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cli, err := client.NewSSEMCPClient(*baseURL, client.WithHeaders(headers))
	if err != nil {
		panic(err)
	}
	if err := cli.Start(ctx); err != nil {
		panic(err)
	}
	defer cli.Close()

	_, err = cli.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "ccbos-test-client",
				Version: "0.1.0",
			},
		},
	})
	if err != nil {
		panic(err)
	}

	result, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "generate_classical_chinese_payloads",
			Arguments: map[string]any{
				"questions":     []string{*question},
				"variant_count": *variantCount,
				"preview_count": 2,
			},
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("is_error=%v\n", result.IsError)
	for _, content := range result.Content {
		if text, ok := content.(mcp.TextContent); ok {
			fmt.Println(text.Text)
		}
	}
}
