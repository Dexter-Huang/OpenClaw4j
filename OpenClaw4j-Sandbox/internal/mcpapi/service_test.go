package mcpapi

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/seaskyland/openclaw4j-sandbox/internal/bashapi"
	"github.com/seaskyland/openclaw4j-sandbox/internal/fileapi"
	"github.com/seaskyland/openclaw4j-sandbox/internal/pathguard"
)

func newTestService(t *testing.T) Service {
	t.Helper()

	guard := pathguard.New(t.TempDir())
	return NewService(
		fileapi.NewService(guard, 1024*1024),
		bashapi.NewService(guard, 3000, 5000, 64*1024),
	)
}

func TestListToolsIncludesFileAndBashTools(t *testing.T) {
	tools := ListTools()
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}

	for _, name := range []string{"file_read", "file_write", "file_list", "file_replace", "file_search", "sandbox_execute_bash"} {
		if !names[name] {
			t.Fatalf("missing tool %s", name)
		}
	}
}

func TestCallBuiltinToolWritesFile(t *testing.T) {
	service := newTestService(t)

	result := service.CallTool("file_write", map[string]any{
		"file":    "mcp.txt",
		"content": "hello",
	})
	if result.IsError {
		t.Fatalf("tool failed: %#v", result)
	}
	if !strings.Contains(result.Content[0].Text, "File written successfully") {
		t.Fatalf("text = %s", result.Content[0].Text)
	}
}

func TestJSONRPCRequestParses(t *testing.T) {
	var req JSONRPCRequest
	err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`), &req)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "tools/list" {
		t.Fatalf("method = %s", req.Method)
	}
}

func TestRoutesExposeMcpEndpoints(t *testing.T) {
	service := newTestService(t)
	broker := NewSSEBroker(service)
	routes := Routes(service, broker)

	seen := make(map[string]string, len(routes))
	for _, route := range routes {
		seen[route.Path] = route.Method
	}

	want := map[string]string{
		"/mcp":                                  "POST",
		"/mcp/sse":                              "GET",
		"/mcp/message":                          "POST",
		"/v1/mcp/servers":                       "GET",
		"/v1/mcp/:server_name/tools":            "GET",
		"/v1/mcp/:server_name/tools/:tool_name": "POST",
	}
	for path, method := range want {
		if seen[path] != method {
			t.Fatalf("route %s = %q, want %s", path, seen[path], method)
		}
	}
}

func TestSSEBrokerOpensSessionWithMessageEndpoint(t *testing.T) {
	service := newTestService(t)
	broker := NewSSEBroker(service)

	session := broker.OpenSession()
	defer broker.CloseSession(session.ID)

	if session.ID == "" {
		t.Fatal("session id is empty")
	}
	if !strings.HasPrefix(session.Endpoint, "/mcp/message?session_id=") {
		t.Fatalf("endpoint = %s", session.Endpoint)
	}
	if !strings.Contains(session.Endpoint, session.ID) {
		t.Fatalf("endpoint %q does not contain session id %q", session.Endpoint, session.ID)
	}
}

func TestSSEBrokerDispatchSendsJSONRPCResponseEvent(t *testing.T) {
	service := newTestService(t)
	broker := NewSSEBroker(service)
	session := broker.OpenSession()
	defer broker.CloseSession(session.ID)

	err := broker.Dispatch(session.ID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "tools/list",
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-session.Events:
		if event.Type != "message" {
			t.Fatalf("event type = %s", event.Type)
		}
		text := string(event.Data)
		if !strings.Contains(text, `"jsonrpc":"2.0"`) || !strings.Contains(text, `"tools"`) {
			t.Fatalf("event data = %s", text)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for SSE response event")
	}
}

func TestSSEBrokerDispatchIgnoresInitializedNotification(t *testing.T) {
	service := newTestService(t)
	broker := NewSSEBroker(service)
	session := broker.OpenSession()
	defer broker.CloseSession(session.ID)

	err := broker.Dispatch(session.ID, JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-session.Events:
		t.Fatalf("unexpected event: %#v", event)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSSEBrokerDispatchRejectsUnknownSession(t *testing.T) {
	service := newTestService(t)
	broker := NewSSEBroker(service)

	err := broker.Dispatch("missing", JSONRPCRequest{JSONRPC: "2.0", ID: float64(1), Method: "tools/list"})
	if !errors.Is(err, ErrSSESessionNotFound) {
		t.Fatalf("err = %v", err)
	}
}
