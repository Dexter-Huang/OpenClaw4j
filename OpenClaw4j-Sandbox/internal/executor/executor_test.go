package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/model"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type fakeRuntime struct {
	result RuntimeResult
	err    error
	seen   string
}

func (f *fakeRuntime) Run(ctx context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error) {
	f.seen = filepath.ToSlash(workDir)
	if f.result.Success {
		if err := os.WriteFile(filepath.Join(workDir, "result.json"), []byte(`{"ok":true}`), 0o644); err != nil {
			return RuntimeResult{}, err
		}
	}
	return f.result, f.err
}

func executorConfig(t *testing.T) config.Config {
	return config.Config{
		WorkDir:          t.TempDir(),
		DefaultTimeoutMs: 30000,
		MaxTimeoutMs:     60000,
		MemoryLimit:      "256M",
		ProcessLimit:     16,
		StdoutLimitBytes: 1024,
		StderrLimitBytes: 1024,
		DepsDir:          "/opt/openclaw4j-sandbox/deps",
		PythonRuntime:    "python",
		NodePath:         "/opt/openclaw4j-sandbox/deps/node/node_modules",
	}
}

func TestExecuteRejectsInvalidRequest(t *testing.T) {
	e := New(executorConfig(t), &fakeRuntime{})
	res := e.Execute(context.Background(), model.ExecuteRequest{Language: "python", Code: " "})

	if res.Success || res.Code == nil || *res.Code != "INVALID_REQUEST" {
		t.Fatalf("response = %+v", res)
	}
}

func TestExecuteReturnsScriptResult(t *testing.T) {
	rt := &fakeRuntime{result: RuntimeResult{Stdout: []byte("out"), Stderr: []byte(""), ExitCode: intPtr(0), Success: true}}
	e := New(executorConfig(t), rt)
	reqID := "abc/../中文"
	res := e.Execute(context.Background(), model.ExecuteRequest{
		RequestID: &reqID,
		Language:  "python",
		Code:      "def main(params):\n    return {'sum': params['a'] + params['b']}",
		Params:    map[string]any{"a": float64(1), "b": float64(2)},
	})

	if !res.Success || res.Data == nil {
		t.Fatalf("response = %+v", res)
	}
	if !strings.Contains(rt.seen, "abc------") {
		t.Fatalf("work dir prefix should include sanitized request id, got %q", rt.seen)
	}
}

func TestExecuteConvertsRuntimeFailure(t *testing.T) {
	rt := &fakeRuntime{err: errors.New("native failure")}
	e := New(executorConfig(t), rt)
	res := e.Execute(context.Background(), model.ExecuteRequest{Language: "python", Code: "def main():\n    return 1"})

	if res.Success || res.Code == nil || *res.Code != "SANDBOX_ERROR" {
		t.Fatalf("response = %+v", res)
	}
}

func TestExecuteConvertsTimeout(t *testing.T) {
	rt := &fakeRuntime{result: RuntimeResult{Timeout: true}}
	e := New(executorConfig(t), rt)
	res := e.Execute(context.Background(), model.ExecuteRequest{Language: "python", Code: "def main():\n    return 1"})

	if res.Success || res.Code == nil || *res.Code != "TIMEOUT" {
		t.Fatalf("response = %+v", res)
	}
}

func intPtr(value int) *int {
	return &value
}
