package wrapper

import (
	"strings"
	"testing"

	"github.com/seaskyland/openclaw4j-sandbox/internal/model"
)

func TestValidateRequest(t *testing.T) {
	req := model.ExecuteRequest{Language: "javascript", Code: "function main(params) { return params; }"}
	language, err := ValidateRequest(req)
	if err != nil {
		t.Fatalf("ValidateRequest() error = %v", err)
	}
	if language != JavaScript {
		t.Fatalf("language = %v, want JavaScript", language)
	}

	_, err = ValidateRequest(model.ExecuteRequest{Language: "python", Code: " "})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty code error = %v", err)
	}
}

func TestPythonRunnerUsesInspect(t *testing.T) {
	runner := BuildRunner(Python, "/tmp/openclaw4j-test")
	if !strings.Contains(runner, "inspect.signature") {
		t.Fatalf("runner should inspect main signature:\n%s", runner)
	}
	if !strings.Contains(runner, "main_fn(params)") {
		t.Fatalf("runner should pass params when main accepts arguments")
	}
	if strings.Contains(runner, "except TypeError") {
		t.Fatalf("runner must not swallow user TypeError")
	}
}

func TestJavaScriptRunnerUsesVMAndModuleExports(t *testing.T) {
	runner := BuildRunner(JavaScript, "/tmp/openclaw4j-test")
	if !strings.Contains(runner, "vm.createContext") {
		t.Fatalf("runner should use vm context")
	}
	if !strings.Contains(runner, "context.main || context.module.exports.main || context.exports.main") {
		t.Fatalf("runner should support module.exports.main")
	}
}

func TestRuntimeArgs(t *testing.T) {
	if got := JavaScript.RuntimeArgs(); len(got) != 1 || got[0] != "--jitless" {
		t.Fatalf("JavaScript.RuntimeArgs() = %#v", got)
	}
	if got := Python.RuntimeArgs(); len(got) != 0 {
		t.Fatalf("Python.RuntimeArgs() = %#v", got)
	}
}
