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

	"github.com/seaskyland/openclaw4j-backend-go/internal/dao"
)

func TestCallSSECompletesEndpointHandshake(t *testing.T) {
	messagePosted := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/mcp/sse":
			writer.Header().Set("Content-Type", "text/event-stream")
			_, _ = writer.Write([]byte("event: endpoint\ndata: /mcp/message\n\n"))
			writer.(http.Flusher).Flush()
			<-messagePosted
			_, _ = writer.Write([]byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":\"go\",\"result\":{\"tools\":[{\"name\":\"weather\"}]}}\n\n"))
		case "/mcp/message":
			messagePosted <- struct{}{}
			writer.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	service := NewService(nil, nil)
	body, err := service.callSSE(context.Background(), deployConfig{RemoteAddress: server.URL, RemoteEndpoint: "/mcp/sse"}, "tools/list", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	tools, err := parseToolListResponse(body)
	if err != nil || len(tools) != 1 {
		t.Fatalf("tools=%#v err=%v", tools, err)
	}
}

func TestParseToolListResponseAddsLegacyInputSchemaAlias(t *testing.T) {
	tools, err := parseToolListResponse([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"weather","inputSchema":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("tools=%#v", tools)
	}
	tool, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("tool=%#v", tools[0])
	}
	schema, ok := tool["input_schema"].(map[string]any)
	if !ok || schema["type"] != "object" {
		t.Fatalf("legacy input_schema was not populated: %#v", tool)
	}
}

func TestResolveTransportUsesInstallTypeForLegacyBusinessType(t *testing.T) {
	installType := "SSE"
	server := dao.McpServer{Type: "CUSTOMER", InstallType: &installType}
	if transport := resolveTransport(deployConfig{}, server); transport != "sse" {
		t.Fatalf("transport=%q, want sse", transport)
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
