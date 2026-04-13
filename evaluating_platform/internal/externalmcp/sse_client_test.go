package externalmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestTimeoutAwareSSETransportUsesContextDeadlineOverFallback(t *testing.T) {
	transport, cleanup := newTestTimeoutAwareTransport(t, 100*time.Millisecond)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	req := mcptransport.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(int64(1)),
		Method:  "ping",
	}
	if _, err := transport.SendRequest(ctx, req); err != nil {
		t.Fatalf("SendRequest() error = %v, want success", err)
	}
}

func TestTimeoutAwareSSETransportUsesFallbackTimeoutWithoutContextDeadline(t *testing.T) {
	transport, cleanup := newTestTimeoutAwareTransport(t, 100*time.Millisecond)
	defer cleanup()

	if err := transport.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	req := mcptransport.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(int64(2)),
		Method:  "ping",
	}
	if _, err := transport.SendRequest(context.Background(), req); err == nil {
		t.Fatal("SendRequest() error = nil, want timeout")
	}
}

func newTestTimeoutAwareTransport(t *testing.T, responseDelay time.Duration) (*timeoutAwareSSETransport, func()) {
	t.Helper()

	var serverURL string
	messages := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "event: endpoint\ndata: %s/message\n\n", serverURL)
			flusher.Flush()

			for {
				select {
				case <-r.Context().Done():
					return
				case msg := <-messages:
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
					flusher.Flush()
				}
			}
		case "/message":
			var req mcptransport.JSONRPCRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)

			go func(id mcp.RequestId) {
				time.Sleep(responseDelay)
				resp := mcptransport.JSONRPCResponse{
					JSONRPC: mcp.JSONRPC_VERSION,
					ID:      id,
					Result:  json.RawMessage(`{"ok":true}`),
				}
				payload, _ := json.Marshal(resp)
				messages <- string(payload)
			}(req.ID)
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL = server.URL

	transport, err := newTimeoutAwareSSETransport(server.URL+"/sse", 50*time.Millisecond, nil, server.Client())
	if err != nil {
		server.Close()
		t.Fatalf("newTimeoutAwareSSETransport() error = %v", err)
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			_ = transport.Close()
			server.Close()
		})
	}
	return transport, cleanup
}
