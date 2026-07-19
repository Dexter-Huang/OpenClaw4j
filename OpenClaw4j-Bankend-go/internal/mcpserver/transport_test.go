package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCallSSECompletesEndpointHandshake(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/sse":
			writer.Header().Set("Content-Type", "text/event-stream")
			_, _ = writer.Write([]byte("event: endpoint\ndata: /message\n\n"))
		case "/message":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","id":"go","result":{"tools":[{"name":"weather"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	service := NewService(nil, nil)
	body, err := service.callSSE(context.Background(), deployConfig{RemoteAddress: server.URL}, "tools/list", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	tools, err := parseToolListResponse(body)
	if err != nil || len(tools) != 1 {
		t.Fatalf("tools=%#v err=%v", tools, err)
	}
}

func TestCallStdioInitializesBeforeToolCall(t *testing.T) {
	t.Setenv("GO_WANT_MCP_HELPER", "1")
	service := NewService(nil, nil)
	body, err := service.callStdio(context.Background(), deployConfig{Command: os.Args[0], Args: []string{"-test.run=TestMCPStdioHelperProcess", "--"}}, "tools/list", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	tools, err := parseToolListResponse(body)
	if err != nil || len(tools) != 1 {
		t.Fatalf("tools=%#v err=%v", tools, err)
	}
}

func TestMCPStdioHelperProcess(t *testing.T) {
	if !strings.Contains(strings.Join(os.Args, " "), "TestMCPStdioHelperProcess") || os.Getenv("GO_WANT_MCP_HELPER") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(scanner.Bytes(), &request) != nil || request.Method == "notifications/initialized" {
			continue
		}
		if request.Method == "initialize" {
			_, _ = os.Stdout.WriteString(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05"}}` + "\n")
			continue
		}
		if request.Method == "tools/list" {
			_, _ = os.Stdout.WriteString(`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"stdio-tool"}]}}` + "\n")
			return
		}
	}
}
