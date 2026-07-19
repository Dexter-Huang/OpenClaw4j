package scriptsandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientExecutesWorkflowScript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/execute" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		var body executeRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.RequestID != "req_1" || body.Language != "python" || body.Params["number"] != float64(2) || body.TimeoutMs != 1500 {
			t.Fatalf("unexpected request: %#v", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"output":3}}`))
	}))
	defer server.Close()

	result, err := NewClient(server.URL+"/", 1500*time.Millisecond).Execute(context.Background(), "python", "def main(params): return {}", map[string]any{"number": 2}, "req_1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.Data.(map[string]any)["output"] != float64(3) {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestClientReturnsSandboxFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"success":false,"code":"SCRIPT_ERROR","message":"syntax error"}`))
	}))
	defer server.Close()

	result, err := NewClient(server.URL, time.Second).Execute(context.Background(), "javascript", "invalid", nil, "req_2")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success || result.Code != "SCRIPT_ERROR" || result.Message != "syntax error" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
