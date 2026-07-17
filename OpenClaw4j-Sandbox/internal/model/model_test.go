package model

import (
	"encoding/json"
	"testing"
)

func TestExecuteRequestJSONDefaultsParams(t *testing.T) {
	body := []byte(`{"language":"python","code":"def main(): return 1"}`)
	var req ExecuteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Params != nil {
		t.Fatalf("Params = %#v, want nil so executor writes JSON null like the Rust service", req.Params)
	}
}

func TestSuccessAndErrorResponses(t *testing.T) {
	ok := Success(map[string]int{"sum": 3}, "out", "", 0, 12)
	if !ok.Success || ok.Data == nil || ok.ExitCode == nil || *ok.ExitCode != 0 {
		t.Fatalf("unexpected success response: %+v", ok)
	}

	exitCode := 1
	err := Error("SCRIPT_ERROR", "script execution failed", "out", "err", &exitCode, 15)
	if err.Success || err.Code == nil || *err.Code != "SCRIPT_ERROR" || err.ExitCode == nil || *err.ExitCode != 1 {
		t.Fatalf("unexpected error response: %+v", err)
	}
}
