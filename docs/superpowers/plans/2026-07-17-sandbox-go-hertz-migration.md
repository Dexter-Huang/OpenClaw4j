# OpenClaw4j Sandbox Go + Hertz Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `OpenClaw4j-Sandbox` 从 Rust + Axum 迁移为 Go + Hertz，并在 Linux 容器中通过 sandlock Go SDK 执行脚本。

**Architecture:** Go 服务保留现有 HTTP 契约，Hertz 只负责请求解析和响应输出；执行流程沉到 `internal/executor`，语言适配和 runner 生成沉到 `internal/wrapper`。Linux 使用 sandlock Go SDK，非 Linux 使用直接进程执行 fallback 供本地测试。

**Tech Stack:** Go 1.22+、CloudWeGo Hertz、sandlock Go SDK、cgo、Python、Node.js、Docker multi-stage build。

## Global Constraints

- 只把 `D:\IDEA_project\OpenClaw4j` 当作 Git 仓库根，所有 Git 命令从根目录运行。
- 项目文档和新增代码注释默认使用中文。
- 不改变 `GET /health` 和 `POST /v1/execute` API。
- 不改变 `ExecuteRequest` / `ExecuteResponse` JSON 字段。
- 不改变 `OPENCLAW_SANDBOX_*` 环境变量语义。
- Linux 容器内使用 sandlock Go SDK，不再依赖 `sandlock run` CLI。
- 非 Linux fallback 仅用于本地测试，不视为安全沙箱。
- 不扩展新的脚本语言。
- 不改变 Python/JavaScript 用户脚本的 `main(params)` 约定。
- 不清理 backend/frontend 既有脏改动。

---

## File Structure

- Create `OpenClaw4j-Sandbox/go.mod`: Go module 声明，依赖 Hertz、UUID 和 sandlock Go SDK。
- Create `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`: Hertz 服务入口和路由注册。
- Create `OpenClaw4j-Sandbox/internal/config/config.go`: 环境变量读取、默认值和 timeout clamp。
- Create `OpenClaw4j-Sandbox/internal/model/model.go`: HTTP 请求/响应模型。
- Create `OpenClaw4j-Sandbox/internal/wrapper/language.go`: 语言枚举、请求校验、runtime 参数。
- Create `OpenClaw4j-Sandbox/internal/wrapper/runner.go`: Python/JavaScript runner 生成。
- Create `OpenClaw4j-Sandbox/internal/executor/executor.go`: 执行主流程、临时目录、文件写入、响应转换。
- Create `OpenClaw4j-Sandbox/internal/executor/runtime.go`: runner 接口和公共结果类型。
- Create `OpenClaw4j-Sandbox/internal/executor/runtime_direct.go`: 非 Linux 直接执行实现。
- Create `OpenClaw4j-Sandbox/internal/executor/runtime_sandlock_linux.go`: Linux sandlock SDK 实现。
- Create tests under matching package paths.
- Modify `OpenClaw4j-Sandbox/Dockerfile`: Go 服务和 sandlock FFI 构建。
- Modify `OpenClaw4j-Sandbox/README.md`: 更新为 Go + Hertz + sandlock SDK 说明。
- Delete `OpenClaw4j-Sandbox/Cargo.toml`, `OpenClaw4j-Sandbox/Cargo.lock`, `OpenClaw4j-Sandbox/src/*.rs`: Go 实现完成并通过最小验证后删除 Rust 源。

---

### Task 1: Go Module Scaffold

**Files:**
- Create: `OpenClaw4j-Sandbox/go.mod`
- Create: `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`

**Interfaces:**
- Produces: module path `github.com/seaskyland/openclaw4j-sandbox`
- Produces: binary entry package `./cmd/openclaw4j-sandbox`

- [ ] **Step 1: Create minimal Go module**

Create `OpenClaw4j-Sandbox/go.mod`:

```go
module github.com/seaskyland/openclaw4j-sandbox

go 1.22

require (
	github.com/cloudwego/hertz v0.10.1
	github.com/google/uuid v1.6.0
	github.com/multikernel/sandlock/go v0.0.0
)

replace github.com/multikernel/sandlock/go => ../../sandlock/go
```

- [ ] **Step 2: Create a temporary compiling main**

Create `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`:

```go
package main

func main() {}
```

- [ ] **Step 3: Resolve dependencies**

Run:

```powershell
cd OpenClaw4j-Sandbox
go mod tidy
```

Expected: command exits 0 and creates `go.sum`.

- [ ] **Step 4: Verify scaffold compiles**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./...
```

Expected: PASS or `? github.com/seaskyland/openclaw4j-sandbox/cmd/openclaw4j-sandbox [no test files]`.

- [ ] **Step 5: Commit scaffold**

Run from repo root:

```powershell
git add OpenClaw4j-Sandbox/go.mod OpenClaw4j-Sandbox/go.sum OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go
git commit -m "chore: scaffold go sandbox module"
```

---

### Task 2: Config And Model Contracts

**Files:**
- Create: `OpenClaw4j-Sandbox/internal/config/config.go`
- Create: `OpenClaw4j-Sandbox/internal/config/config_test.go`
- Create: `OpenClaw4j-Sandbox/internal/model/model.go`
- Create: `OpenClaw4j-Sandbox/internal/model/model_test.go`

**Interfaces:**
- Produces: `type config.Config struct`
- Produces: `func config.FromEnv() Config`
- Produces: `func (c Config) ClampTimeout(timeoutMs *uint64) uint64`
- Produces: `type model.ExecuteRequest struct`
- Produces: `type model.ExecuteResponse struct`
- Produces: `func model.Success(data any, stdout string, stderr string, exitCode int, durationMs uint64) ExecuteResponse`
- Produces: `func model.Error(code string, message string, stdout string, stderr string, exitCode *int, durationMs uint64) ExecuteResponse`

- [ ] **Step 1: Write config tests**

Create `OpenClaw4j-Sandbox/internal/config/config_test.go`:

```go
package config

import (
	"path/filepath"
	"testing"
)

func TestClampTimeout(t *testing.T) {
	c := Config{DefaultTimeoutMs: 30000, MaxTimeoutMs: 60000}
	value := uint64(90000)

	if got := c.ClampTimeout(&value); got != 60000 {
		t.Fatalf("ClampTimeout() = %d, want 60000", got)
	}
	if got := c.ClampTimeout(nil); got != 30000 {
		t.Fatalf("ClampTimeout(nil) = %d, want 30000", got)
	}
}

func TestFromEnvUsesDefaults(t *testing.T) {
	t.Setenv("OPENCLAW_SANDBOX_BIND", "")
	c := FromEnv()

	if c.Bind != "0.0.0.0:9010" {
		t.Fatalf("Bind = %q, want default", c.Bind)
	}
	if filepath.ToSlash(c.WorkDir) != "/tmp/openclaw4j-sandbox" {
		t.Fatalf("WorkDir = %q", c.WorkDir)
	}
	if c.PythonRuntime != "/opt/openclaw4j-sandbox/deps/python/bin/python" {
		t.Fatalf("PythonRuntime = %q", c.PythonRuntime)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./internal/config
```

Expected: FAIL because `Config` and `FromEnv` are not defined.

- [ ] **Step 3: Implement config**

Create `OpenClaw4j-Sandbox/internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Bind             string
	WorkDir          string
	DefaultTimeoutMs uint64
	MaxTimeoutMs     uint64
	MemoryLimit      string
	ProcessLimit     uint32
	StdoutLimitBytes int
	StderrLimitBytes int
	DepsDir          string
	PythonRuntime    string
	NodePath         string
}

func FromEnv() Config {
	return Config{
		Bind:             envValue("OPENCLAW_SANDBOX_BIND", "0.0.0.0:9010"),
		WorkDir:          filepath.Clean(envValue("OPENCLAW_SANDBOX_WORK_DIR", "/tmp/openclaw4j-sandbox")),
		DefaultTimeoutMs: envUint64("OPENCLAW_SANDBOX_DEFAULT_TIMEOUT_MS", 30000),
		MaxTimeoutMs:     envUint64("OPENCLAW_SANDBOX_MAX_TIMEOUT_MS", 120000),
		MemoryLimit:      envValue("OPENCLAW_SANDBOX_MEMORY_LIMIT", "256M"),
		ProcessLimit:     envUint32("OPENCLAW_SANDBOX_PROCESS_LIMIT", 16),
		StdoutLimitBytes: envInt("OPENCLAW_SANDBOX_STDOUT_LIMIT_BYTES", 65536),
		StderrLimitBytes: envInt("OPENCLAW_SANDBOX_STDERR_LIMIT_BYTES", 65536),
		DepsDir:          envValue("OPENCLAW_SANDBOX_DEPS_DIR", "/opt/openclaw4j-sandbox/deps"),
		PythonRuntime:    envValue("OPENCLAW_SANDBOX_PYTHON_RUNTIME", "/opt/openclaw4j-sandbox/deps/python/bin/python"),
		NodePath:         envValue("OPENCLAW_SANDBOX_NODE_PATH", "/opt/openclaw4j-sandbox/deps/node/node_modules"),
	}
}

func (c Config) ClampTimeout(timeoutMs *uint64) uint64 {
	if timeoutMs == nil {
		return c.DefaultTimeoutMs
	}
	if *timeoutMs > c.MaxTimeoutMs {
		return c.MaxTimeoutMs
	}
	return *timeoutMs
}

func envValue(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envUint64(key string, fallback uint64) uint64 {
	value, err := strconv.ParseUint(os.Getenv(key), 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func envUint32(key string, fallback uint32) uint32 {
	value, err := strconv.ParseUint(os.Getenv(key), 10, 32)
	if err != nil {
		return fallback
	}
	return uint32(value)
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
```

- [ ] **Step 4: Write model tests**

Create `OpenClaw4j-Sandbox/internal/model/model_test.go`:

```go
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
```

- [ ] **Step 5: Implement model**

Create `OpenClaw4j-Sandbox/internal/model/model.go`:

```go
package model

type ExecuteRequest struct {
	RequestID *string `json:"request_id,omitempty"`
	Language  string  `json:"language"`
	Code      string  `json:"code"`
	Params    any     `json:"params,omitempty"`
	TimeoutMs *uint64 `json:"timeout_ms,omitempty"`
}

type ExecuteResponse struct {
	Success    bool    `json:"success"`
	Data       any     `json:"data,omitempty"`
	Message    *string `json:"message,omitempty"`
	Code       *string `json:"code,omitempty"`
	Stdout     string  `json:"stdout"`
	Stderr     string  `json:"stderr"`
	ExitCode   *int    `json:"exit_code,omitempty"`
	DurationMs uint64  `json:"duration_ms"`
}

func Success(data any, stdout string, stderr string, exitCode int, durationMs uint64) ExecuteResponse {
	return ExecuteResponse{
		Success:    true,
		Data:       data,
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   &exitCode,
		DurationMs: durationMs,
	}
}

func Error(code string, message string, stdout string, stderr string, exitCode *int, durationMs uint64) ExecuteResponse {
	return ExecuteResponse{
		Success:    false,
		Message:    &message,
		Code:       &code,
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		DurationMs: durationMs,
	}
}
```

- [ ] **Step 6: Run package tests**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./internal/config ./internal/model
```

Expected: PASS.

- [ ] **Step 7: Commit config and model**

Run from repo root:

```powershell
git add OpenClaw4j-Sandbox/internal/config OpenClaw4j-Sandbox/internal/model
git commit -m "feat: add sandbox config and model contracts"
```

---

### Task 3: Language Wrapper And Runners

**Files:**
- Create: `OpenClaw4j-Sandbox/internal/wrapper/language.go`
- Create: `OpenClaw4j-Sandbox/internal/wrapper/runner.go`
- Create: `OpenClaw4j-Sandbox/internal/wrapper/wrapper_test.go`

**Interfaces:**
- Consumes: `model.ExecuteRequest`
- Produces: `type wrapper.Language string`
- Produces: `func ValidateRequest(req model.ExecuteRequest) (Language, error)`
- Produces: `func BuildRunner(language Language, workDir string) string`
- Produces: `func (l Language) UserFileName() string`
- Produces: `func (l Language) RunnerFileName() string`
- Produces: `func (l Language) RuntimeCommand(pythonRuntime string) string`
- Produces: `func (l Language) RuntimeArgs() []string`

- [ ] **Step 1: Write wrapper tests**

Create `OpenClaw4j-Sandbox/internal/wrapper/wrapper_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./internal/wrapper
```

Expected: FAIL because wrapper package files are not implemented.

- [ ] **Step 3: Implement language wrapper**

Create `OpenClaw4j-Sandbox/internal/wrapper/language.go`:

```go
package wrapper

import (
	"fmt"
	"strings"

	"github.com/seaskyland/openclaw4j-sandbox/internal/model"
)

type Language string

const (
	Python     Language = "python"
	JavaScript Language = "javascript"
)

func ParseLanguage(value string) (Language, bool) {
	switch strings.ToLower(value) {
	case "python", "python3":
		return Python, true
	case "javascript", "js":
		return JavaScript, true
	default:
		return "", false
	}
}

func ValidateRequest(req model.ExecuteRequest) (Language, error) {
	if strings.TrimSpace(req.Code) == "" {
		return "", fmt.Errorf("script code cannot be empty")
	}
	language, ok := ParseLanguage(req.Language)
	if !ok {
		return "", fmt.Errorf("unsupported script language: %s", req.Language)
	}
	return language, nil
}

func (l Language) UserFileName() string {
	switch l {
	case Python:
		return "user.py"
	case JavaScript:
		return "user.js"
	default:
		return "user.txt"
	}
}

func (l Language) RunnerFileName() string {
	switch l {
	case Python:
		return "runner.py"
	case JavaScript:
		return "runner.js"
	default:
		return "runner.txt"
	}
}

func (l Language) RuntimeCommand(pythonRuntime string) string {
	if l == Python {
		return pythonRuntime
	}
	return "node"
}

func (l Language) RuntimeArgs() []string {
	if l == JavaScript {
		return []string{"--jitless"}
	}
	return nil
}
```

- [ ] **Step 4: Implement runner generation**

Create `OpenClaw4j-Sandbox/internal/wrapper/runner.go`:

```go
package wrapper

import (
	"path/filepath"
	"strings"
)

func BuildRunner(language Language, workDir string) string {
	switch language {
	case Python:
		return buildPythonRunner(workDir)
	case JavaScript:
		return buildJavaScriptRunner(workDir)
	default:
		return ""
	}
}

func slashPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), `\`, `\\`)
}

func buildPythonRunner(workDir string) string {
	paramsPath := slashPath(filepath.Join(workDir, "params.json"))
	userPath := slashPath(filepath.Join(workDir, "user.py"))
	resultPath := slashPath(filepath.Join(workDir, "result.json"))
	return `
import inspect
import json
import runpy
import sys
import traceback

params_path = r"` + paramsPath + `"
user_path = r"` + userPath + `"
result_path = r"` + resultPath + `"

try:
    with open(params_path, "r", encoding="utf-8") as f:
        params = json.load(f)
    namespace = runpy.run_path(user_path)
    main_fn = namespace.get("main")
    if not callable(main_fn):
        raise RuntimeError("main function is required")
    signature = inspect.signature(main_fn)
    if len(signature.parameters) == 0:
        result = main_fn()
    else:
        result = main_fn(params)
    with open(result_path, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False)
except Exception:
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
`
}

func buildJavaScriptRunner(workDir string) string {
	paramsPath := slashPath(filepath.Join(workDir, "params.json"))
	userPath := slashPath(filepath.Join(workDir, "user.js"))
	resultPath := slashPath(filepath.Join(workDir, "result.json"))
	return `
const fs = require('fs');
const vm = require('vm');

const paramsPath = "` + paramsPath + `";
const userPath = "` + userPath + `";
const resultPath = "` + resultPath + `";

(async () => {
  const params = JSON.parse(fs.readFileSync(paramsPath, 'utf8'));
  const sandbox = {
    console,
    module: { exports: {} },
    exports: {},
    require,
    process: { env: process.env },
    setTimeout,
    clearTimeout,
    setInterval,
    clearInterval,
  };
  sandbox.globalThis = sandbox;
  const context = vm.createContext(sandbox);
  const userCode = fs.readFileSync(userPath, 'utf8');
  vm.runInContext(userCode, context, { filename: userPath, timeout: 1000 });
  const mainFn = context.main || context.module.exports.main || context.exports.main;
  if (typeof mainFn !== 'function') {
    throw new Error('main function is required');
  }
  const result = mainFn.length === 0 ? await mainFn() : await mainFn(params);
  fs.writeFileSync(resultPath, JSON.stringify(result), 'utf8');
})().catch((err) => {
  console.error(err && err.stack ? err.stack : String(err));
  process.exit(1);
});
`
}
```

- [ ] **Step 5: Run wrapper tests**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./internal/wrapper
```

Expected: PASS.

- [ ] **Step 6: Commit wrapper**

Run from repo root:

```powershell
git add OpenClaw4j-Sandbox/internal/wrapper
git commit -m "feat: add sandbox language wrappers"
```

---

### Task 4: Executor Helpers And Runtime Abstraction

**Files:**
- Create: `OpenClaw4j-Sandbox/internal/executor/runtime.go`
- Create: `OpenClaw4j-Sandbox/internal/executor/helpers.go`
- Create: `OpenClaw4j-Sandbox/internal/executor/helpers_test.go`

**Interfaces:**
- Consumes: `config.Config`
- Consumes: `wrapper.Language`
- Produces: `type Runtime interface { Run(ctx context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error) }`
- Produces: `type RuntimeResult struct`
- Produces: `func SanitizeName(value string) string`
- Produces: `func LimitBytes(bytes []byte, maxBytes int) string`
- Produces: `func SandboxEnvVars(cfg config.Config) map[string]string`
- Produces: `func SandboxReadPaths(cfg config.Config) []string`

- [ ] **Step 1: Write helper tests**

Create `OpenClaw4j-Sandbox/internal/executor/helpers_test.go`:

```go
package executor

import (
	"reflect"
	"strings"
	"testing"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
)

func testConfig() config.Config {
	return config.Config{
		WorkDir:          "/tmp/openclaw4j-sandbox",
		DefaultTimeoutMs: 30000,
		MaxTimeoutMs:     60000,
		MemoryLimit:      "512M",
		ProcessLimit:     16,
		StdoutLimitBytes: 1024,
		StderrLimitBytes: 1024,
		DepsDir:          "/opt/openclaw4j-sandbox/deps",
		PythonRuntime:    "/opt/openclaw4j-sandbox/deps/python/bin/python",
		NodePath:         "/opt/openclaw4j-sandbox/deps/node/node_modules",
	}
}

func TestSanitizeName(t *testing.T) {
	if got := SanitizeName("abc/../中文"); got != "abc------" {
		t.Fatalf("SanitizeName() = %q", got)
	}
}

func TestLimitBytes(t *testing.T) {
	if got := LimitBytes([]byte("abcdef"), 3); got != "abc\n... truncated ..." {
		t.Fatalf("LimitBytes() = %q", got)
	}
}

func TestSandboxReadPathsIncludesDeps(t *testing.T) {
	paths := SandboxReadPaths(testConfig())
	want := []string{"/usr", "/lib", "/lib64", "/bin", "/etc", "/opt/openclaw4j-sandbox/deps"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("SandboxReadPaths() = %#v", paths)
	}
}

func TestSandboxEnvVars(t *testing.T) {
	env := SandboxEnvVars(testConfig())
	if env["NODE_PATH"] != "/opt/openclaw4j-sandbox/deps/node/node_modules" {
		t.Fatalf("NODE_PATH = %q", env["NODE_PATH"])
	}
	if env["PYTHONIOENCODING"] != "utf-8" {
		t.Fatalf("PYTHONIOENCODING = %q", env["PYTHONIOENCODING"])
	}
	if !strings.HasPrefix(env["PATH"], "/opt/openclaw4j-sandbox/deps/python/bin:") {
		t.Fatalf("PATH = %q", env["PATH"])
	}
}
```

- [ ] **Step 2: Run helper tests to verify failure**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./internal/executor
```

Expected: FAIL because helper functions are not defined.

- [ ] **Step 3: Implement runtime abstraction**

Create `OpenClaw4j-Sandbox/internal/executor/runtime.go`:

```go
package executor

import (
	"context"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type RuntimeResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode *int
	Success  bool
	Timeout  bool
}

type Runtime interface {
	Run(ctx context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error)
}
```

- [ ] **Step 4: Implement helpers**

Create `OpenClaw4j-Sandbox/internal/executor/helpers.go`:

```go
package executor

import (
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
)

func SanitizeName(value string) string {
	var builder strings.Builder
	for _, ch := range value {
		if builder.Len() >= 48 {
			break
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			builder.WriteRune(ch)
		} else {
			builder.WriteByte('-')
		}
	}
	return builder.String()
}

func LimitBytes(bytes []byte, maxBytes int) string {
	if len(bytes) <= maxBytes {
		return string(bytes)
	}
	truncated := bytes[:maxBytes]
	for !utf8.Valid(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return string(truncated) + "\n... truncated ..."
}

func SandboxReadPaths(cfg config.Config) []string {
	return []string{"/usr", "/lib", "/lib64", "/bin", "/etc", cfg.DepsDir}
}

func SandboxEnvVars(cfg config.Config) map[string]string {
	pythonBin := filepath.Dir(cfg.PythonRuntime)
	if pythonBin == "." || pythonBin == string(filepath.Separator) {
		pythonBin = "/usr/local/bin"
	}
	return map[string]string{
		"PATH":             pythonBin + ":/usr/local/bin:/usr/bin:/bin",
		"LANG":             "C.UTF-8",
		"PYTHONIOENCODING": "utf-8",
		"NODE_PATH":        cfg.NodePath,
	}
}
```

- [ ] **Step 5: Run executor helper tests**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./internal/executor
```

Expected: PASS.

- [ ] **Step 6: Commit helper layer**

Run from repo root:

```powershell
git add OpenClaw4j-Sandbox/internal/executor
git commit -m "feat: add sandbox executor helpers"
```

---

### Task 5: Direct Runtime And Executor Workflow

**Files:**
- Create: `OpenClaw4j-Sandbox/internal/executor/runtime_direct.go`
- Create: `OpenClaw4j-Sandbox/internal/executor/executor.go`
- Create: `OpenClaw4j-Sandbox/internal/executor/executor_test.go`

**Interfaces:**
- Consumes: `Runtime`
- Produces: `type Executor struct`
- Produces: `func New(cfg config.Config, runtime Runtime) *Executor`
- Produces: `func (e *Executor) Execute(ctx context.Context, req model.ExecuteRequest) model.ExecuteResponse`
- Produces: `type DirectRuntime struct{}`

- [ ] **Step 1: Write executor tests**

Create `OpenClaw4j-Sandbox/internal/executor/executor_test.go`:

```go
package executor

import (
	"context"
	"errors"
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
	seen  string
}

func (f *fakeRuntime) Run(ctx context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error) {
	f.seen = filepath.ToSlash(workDir)
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
```

- [ ] **Step 2: Run executor tests to verify failure**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./internal/executor
```

Expected: FAIL because `New`, `Executor`, and direct runtime are not implemented.

- [ ] **Step 3: Implement executor workflow**

Create `OpenClaw4j-Sandbox/internal/executor/executor.go`:

```go
package executor

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/model"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type Executor struct {
	cfg     config.Config
	runtime Runtime
}

func New(cfg config.Config, runtime Runtime) *Executor {
	return &Executor{cfg: cfg, runtime: runtime}
}

func (e *Executor) Execute(ctx context.Context, req model.ExecuteRequest) model.ExecuteResponse {
	started := time.Now()
	requestID := "<none>"
	if req.RequestID != nil {
		requestID = *req.RequestID
	}
	slog.Debug("script execution request received", "request_id", requestID, "language", req.Language, "code_bytes", len(req.Code))

	language, err := wrapper.ValidateRequest(req)
	if err != nil {
		slog.Warn("script execution request rejected", "request_id", requestID, "language", req.Language, "error", err.Error())
		return model.Error("INVALID_REQUEST", err.Error(), "", "", nil, elapsedMs(started))
	}

	res, err := e.executeValidated(ctx, req, language, started)
	if err != nil {
		out := model.Error("SANDBOX_ERROR", err.Error(), "", "", nil, elapsedMs(started))
		logResponse(requestID, req.Language, out)
		return out
	}
	logResponse(requestID, req.Language, res)
	return res
}

func (e *Executor) executeValidated(ctx context.Context, req model.ExecuteRequest, language wrapper.Language, started time.Time) (model.ExecuteResponse, error) {
	if err := os.MkdirAll(e.cfg.WorkDir, 0o755); err != nil {
		return model.ExecuteResponse{}, err
	}
	prefix := uuid.NewString()
	if req.RequestID != nil {
		if sanitized := SanitizeName(*req.RequestID); sanitized != "" {
			prefix = sanitized
		}
	}
	workDir, err := os.MkdirTemp(e.cfg.WorkDir, prefix+"-")
	if err != nil {
		return model.ExecuteResponse{}, err
	}
	defer os.RemoveAll(workDir)

	if err := writeExecutionFiles(workDir, language, req); err != nil {
		return model.ExecuteResponse{}, err
	}

	timeoutMs := e.cfg.ClampTimeout(req.TimeoutMs)
	slog.Info("script execution started", "request_id", prefix, "language", string(language), "timeout_ms", timeoutMs, "memory_limit", e.cfg.MemoryLimit, "process_limit", e.cfg.ProcessLimit, "work_dir", workDir)

	runtimeResult, err := e.runtime.Run(ctx, e.cfg, language, workDir, timeoutMs)
	if err != nil {
		return model.ExecuteResponse{}, err
	}
	if runtimeResult.Timeout {
		return model.Error("TIMEOUT", "script execution timed out", "", "", nil, elapsedMs(started)), nil
	}

	stdout := LimitBytes(runtimeResult.Stdout, e.cfg.StdoutLimitBytes)
	stderr := LimitBytes(runtimeResult.Stderr, e.cfg.StderrLimitBytes)
	if !runtimeResult.Success {
		return model.Error("SCRIPT_ERROR", "script execution failed", stdout, stderr, runtimeResult.ExitCode, elapsedMs(started)), nil
	}

	resultText, err := os.ReadFile(filepath.Join(workDir, "result.json"))
	if err != nil {
		return model.ExecuteResponse{}, err
	}
	var data any
	if err := json.Unmarshal(resultText, &data); err != nil {
		return model.ExecuteResponse{}, err
	}
	exitCode := 0
	if runtimeResult.ExitCode != nil {
		exitCode = *runtimeResult.ExitCode
	}
	return model.Success(data, stdout, stderr, exitCode, elapsedMs(started)), nil
}

func writeExecutionFiles(workDir string, language wrapper.Language, req model.ExecuteRequest) error {
	if err := os.WriteFile(filepath.Join(workDir, language.UserFileName()), []byte(req.Code), 0o644); err != nil {
		return err
	}
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workDir, "params.json"), paramsBytes, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(workDir, language.RunnerFileName()), []byte(wrapper.BuildRunner(language, workDir)), 0o644)
}

func elapsedMs(started time.Time) uint64 {
	return uint64(time.Since(started).Milliseconds())
}

func logResponse(requestID string, language string, response model.ExecuteResponse) {
	attrs := []any{
		"request_id", requestID,
		"language", language,
		"duration_ms", response.DurationMs,
		"stdout_bytes", len(response.Stdout),
		"stderr_bytes", len(response.Stderr),
	}
	if response.ExitCode != nil {
		attrs = append(attrs, "exit_code", *response.ExitCode)
	}
	if response.Success {
		slog.Info("script execution completed", attrs...)
		return
	}
	if response.Code != nil {
		attrs = append(attrs, "code", *response.Code)
	}
	slog.Warn("script execution failed", attrs...)
}
```

- [ ] **Step 4: Implement direct runtime**

Create `OpenClaw4j-Sandbox/internal/executor/runtime_direct.go`:

```go
//go:build !linux

package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type DirectRuntime struct{}

func (DirectRuntime) Run(parent context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error) {
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	args := append([]string{}, language.RuntimeArgs()...)
	args = append(args, language.RunnerFileName())
	cmd := exec.CommandContext(ctx, language.RuntimeCommand(cfg.PythonRuntime), args...)
	cmd.Dir = workDir
	for key, value := range SandboxEnvVars(cfg) {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return RuntimeResult{Timeout: true}, nil
	}
	exitCode := 0
	success := true
	if err != nil {
		success = false
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return RuntimeResult{}, err
		}
	}
	return RuntimeResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: &exitCode, Success: success}, nil
}
```

- [ ] **Step 5: Run executor tests**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./internal/executor
```

Expected: PASS on Windows because `DirectRuntime` is available under `!linux`.

- [ ] **Step 6: Commit executor workflow**

Run from repo root:

```powershell
git add OpenClaw4j-Sandbox/internal/executor
git commit -m "feat: implement sandbox executor workflow"
```

---

### Task 6: Linux Sandlock Runtime

**Files:**
- Create: `OpenClaw4j-Sandbox/internal/executor/runtime_sandlock_linux.go`

**Interfaces:**
- Consumes: `Runtime`
- Produces: `type SandlockRuntime struct{}`

- [ ] **Step 1: Create Linux sandlock runtime**

Create `OpenClaw4j-Sandbox/internal/executor/runtime_sandlock_linux.go`:

```go
//go:build linux

package executor

import (
	"context"
	"errors"
	"time"

	sandlock "github.com/multikernel/sandlock/go"
	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/wrapper"
)

type SandlockRuntime struct{}

func (SandlockRuntime) Run(parent context.Context, cfg config.Config, language wrapper.Language, workDir string, timeoutMs uint64) (RuntimeResult, error) {
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	sb := &sandlock.Sandbox{
		FSReadable:   SandboxReadPaths(cfg),
		FSWritable:   []string{workDir},
		MaxMemory:    cfg.MemoryLimit,
		MaxProcesses: cfg.ProcessLimit,
		CleanEnv:     true,
		Env:          SandboxEnvVars(cfg),
		Cwd:          workDir,
	}

	cmd := append([]string{language.RuntimeCommand(cfg.PythonRuntime)}, language.RuntimeArgs()...)
	cmd = append(cmd, language.RunnerFileName())
	res, err := sb.Run(ctx, cmd...)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return RuntimeResult{Timeout: true}, nil
	}
	if err != nil {
		return RuntimeResult{}, err
	}
	exitCode := res.ExitCode
	timeout := res.Reason == sandlock.ReasonTimeout
	return RuntimeResult{
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		ExitCode: &exitCode,
		Success:  res.Success,
		Timeout:  timeout,
	}, nil
}
```

- [ ] **Step 2: Verify non-Linux tests still pass**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./...
```

Expected: PASS on Windows. The Linux file is excluded by build tags.

- [ ] **Step 3: Commit sandlock runtime**

Run from repo root:

```powershell
git add OpenClaw4j-Sandbox/internal/executor/runtime_sandlock_linux.go
git commit -m "feat: add linux sandlock runtime"
```

---

### Task 7: Hertz HTTP Entry

**Files:**
- Modify: `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`
- Create: `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/runtime_linux.go`
- Create: `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/runtime_other.go`

**Interfaces:**
- Consumes: `executor.New`
- Consumes: `executor.SandlockRuntime` on Linux
- Consumes: `executor.DirectRuntime` on non-Linux
- Produces: `/health`
- Produces: `/v1/execute`

- [ ] **Step 1: Add platform runtime selectors**

Create `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/runtime_linux.go`:

```go
//go:build linux

package main

import "github.com/seaskyland/openclaw4j-sandbox/internal/executor"

func defaultRuntime() executor.Runtime {
	return executor.SandlockRuntime{}
}
```

Create `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/runtime_other.go`:

```go
//go:build !linux

package main

import "github.com/seaskyland/openclaw4j-sandbox/internal/executor"

func defaultRuntime() executor.Runtime {
	return executor.DirectRuntime{}
}
```

- [ ] **Step 2: Replace main with Hertz server**

Modify `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`:

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/seaskyland/openclaw4j-sandbox/internal/config"
	"github.com/seaskyland/openclaw4j-sandbox/internal/executor"
	"github.com/seaskyland/openclaw4j-sandbox/internal/model"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.FromEnv()
	exec := executor.New(cfg, defaultRuntime())
	h := server.Default(server.WithHostPorts(cfg.Bind))

	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(http.StatusOK, utils.H{"status": "UP"})
	})

	h.POST("/v1/execute", func(ctx context.Context, c *app.RequestContext) {
		var req model.ExecuteRequest
		if err := c.BindAndValidate(&req); err != nil {
			res := model.Error("INVALID_REQUEST", err.Error(), "", "", nil, 0)
			c.JSON(http.StatusOK, res)
			return
		}
		c.JSON(http.StatusOK, exec.Execute(ctx, req))
	})

	slog.Info("OpenClaw4j sandbox listening", "bind", cfg.Bind)
	h.Spin()
}
```

- [ ] **Step 3: Run full Go tests**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Run binary help-free compile check**

Run:

```powershell
cd OpenClaw4j-Sandbox
go build ./cmd/openclaw4j-sandbox
```

Expected: command exits 0 and produces `openclaw4j-sandbox.exe` on Windows.

- [ ] **Step 5: Commit Hertz entry**

Run from repo root:

```powershell
git add OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox
git commit -m "feat: expose sandbox api with hertz"
```

---

### Task 8: Dockerfile, README, And Rust Removal

**Files:**
- Modify: `OpenClaw4j-Sandbox/Dockerfile`
- Modify: `OpenClaw4j-Sandbox/README.md`
- Delete: `OpenClaw4j-Sandbox/Cargo.toml`
- Delete: `OpenClaw4j-Sandbox/Cargo.lock`
- Delete: `OpenClaw4j-Sandbox/src/config.rs`
- Delete: `OpenClaw4j-Sandbox/src/executor.rs`
- Delete: `OpenClaw4j-Sandbox/src/main.rs`
- Delete: `OpenClaw4j-Sandbox/src/model.rs`
- Delete: `OpenClaw4j-Sandbox/src/wrapper.rs`

**Interfaces:**
- Consumes: Go binary `openclaw4j-sandbox`
- Produces: Docker image `openclaw4j-sandbox:local`

- [ ] **Step 1: Replace Dockerfile**

Modify `OpenClaw4j-Sandbox/Dockerfile`:

```dockerfile
FROM rust:1.96 AS sandlock-ffi-build

WORKDIR /sandlock
RUN git clone --depth 1 https://github.com/multikernel/sandlock.git .
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    --mount=type=cache,target=/sandlock/target \
    set -eux; \
    cargo build --release -p sandlock-ffi; \
    mkdir -p /sandlock-install/lib /sandlock-install/include /sandlock-install/lib/pkgconfig; \
    cp /sandlock/target/release/libsandlock_ffi.so /sandlock-install/lib/libsandlock_ffi.so; \
    cp /sandlock/crates/sandlock-ffi/include/sandlock.h /sandlock-install/include/sandlock.h; \
    sed "s|@PREFIX@|/usr/local|g; s|@LIBDIR@|/usr/local/lib|g; s|@INCLUDEDIR@|/usr/local/include|g" \
        /sandlock/go/sandlock.pc.in > /sandlock-install/lib/pkgconfig/sandlock.pc

FROM golang:1.24 AS go-build

WORKDIR /workspace/OpenClaw4j-Sandbox
COPY OpenClaw4j-Sandbox/go.mod OpenClaw4j-Sandbox/go.sum ./
COPY --from=sandlock-ffi-build /sandlock-install /usr/local
COPY --from=sandlock-ffi-build /sandlock/go /sandlock/go
ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY OpenClaw4j-Sandbox ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -o /usr/local/bin/openclaw4j-sandbox ./cmd/openclaw4j-sandbox

FROM python:3.12-slim AS python-deps

COPY OpenClaw4j-Sandbox/deps/python/requirements.txt /deps/python/requirements.txt
RUN --mount=type=cache,target=/root/.cache/pip \
    set -eux; \
    python3 -m venv /opt/openclaw4j-sandbox/deps/python; \
    /opt/openclaw4j-sandbox/deps/python/bin/python -m pip install --upgrade pip setuptools wheel; \
    /opt/openclaw4j-sandbox/deps/python/bin/pip install -r /deps/python/requirements.txt

FROM node:22-bookworm-slim AS node-deps

COPY OpenClaw4j-Sandbox/deps/node/package.json OpenClaw4j-Sandbox/deps/node/pnpm-lock.yaml /deps/node/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    set -eux; \
    corepack enable; \
    corepack prepare pnpm@8.6.12 --activate; \
    pnpm install --prod --frozen-lockfile --dir /deps/node; \
    mkdir -p /opt/openclaw4j-sandbox/deps/node; \
    cp -a /deps/node/node_modules /opt/openclaw4j-sandbox/deps/node/node_modules; \
    cp /deps/node/package.json /deps/node/pnpm-lock.yaml /opt/openclaw4j-sandbox/deps/node/

FROM debian:bookworm-slim AS runtime

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates nodejs python3 python3-venv; \
    rm -rf /var/lib/apt/lists/*

COPY --from=go-build /usr/local/bin/openclaw4j-sandbox /usr/local/bin/openclaw4j-sandbox
COPY --from=sandlock-ffi-build /sandlock-install/lib/libsandlock_ffi.so /usr/local/lib/libsandlock_ffi.so
COPY --from=python-deps /opt/openclaw4j-sandbox/deps/python /opt/openclaw4j-sandbox/deps/python
COPY --from=node-deps /opt/openclaw4j-sandbox/deps/node /opt/openclaw4j-sandbox/deps/node
RUN ldconfig

ENV OPENCLAW_SANDBOX_BIND=0.0.0.0:9010 \
    OPENCLAW_SANDBOX_DEPS_DIR=/opt/openclaw4j-sandbox/deps \
    OPENCLAW_SANDBOX_PYTHON_RUNTIME=/opt/openclaw4j-sandbox/deps/python/bin/python \
    OPENCLAW_SANDBOX_NODE_PATH=/opt/openclaw4j-sandbox/deps/node/node_modules
EXPOSE 9010

ENTRYPOINT ["/usr/local/bin/openclaw4j-sandbox"]
```

- [ ] **Step 2: Update README**

Replace `OpenClaw4j-Sandbox/README.md` with Chinese documentation that states:

```markdown
# OpenClaw4j Sandbox

`OpenClaw4j-Sandbox` 是工作流 `Script` 节点的独立代码执行服务。服务使用 Go + Hertz 暴露 HTTP API，在 Linux 容器中通过 sandlock Go SDK 执行 Python 和 JavaScript 脚本。

## 构建镜像

```powershell
docker build -t openclaw4j-sandbox:local -f OpenClaw4j-Sandbox/Dockerfile .
```

## 启动服务

```powershell
docker run --rm -p 127.0.0.1:9010:9010 openclaw4j-sandbox:local
```

## 调用示例

```powershell
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/health'
```

```powershell
$body = @{
  language = 'python'
  code = "def main(params):`n    return {'sum': params['a'] + params['b']}"
  params = @{ a = 1; b = 2 }
} | ConvertTo-Json -Depth 10

Invoke-RestMethod -Uri 'http://127.0.0.1:9010/v1/execute' -Method Post -ContentType 'application/json' -Body $body
```

## 本地测试

```powershell
cd OpenClaw4j-Sandbox
go test ./...
```

Windows 和其他非 Linux 环境会使用直接执行 fallback，仅用于本地单元测试和基础调试，不提供安全沙箱边界。安全行为以 Linux 容器中的 sandlock SDK 验证为准。
```

- [ ] **Step 3: Remove Rust files**

Run from repo root:

```powershell
Remove-Item -LiteralPath OpenClaw4j-Sandbox\Cargo.toml
Remove-Item -LiteralPath OpenClaw4j-Sandbox\Cargo.lock
Remove-Item -LiteralPath OpenClaw4j-Sandbox\src\config.rs
Remove-Item -LiteralPath OpenClaw4j-Sandbox\src\executor.rs
Remove-Item -LiteralPath OpenClaw4j-Sandbox\src\main.rs
Remove-Item -LiteralPath OpenClaw4j-Sandbox\src\model.rs
Remove-Item -LiteralPath OpenClaw4j-Sandbox\src\wrapper.rs
Remove-Item -LiteralPath OpenClaw4j-Sandbox\src
```

Expected: Rust source and Cargo files are removed; `deps/`, `Dockerfile`, `README.md`, Go sources remain.

- [ ] **Step 4: Run Go tests**

Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Optionally build Docker image when Docker is available**

Run from repo root:

```powershell
docker build -t openclaw4j-sandbox:local -f OpenClaw4j-Sandbox/Dockerfile .
```

Expected: image builds successfully. If Docker is unavailable, record the exact command failure in the completion report.

- [ ] **Step 6: Commit Docker, docs, and Rust removal**

Run from repo root:

```powershell
git add OpenClaw4j-Sandbox
git commit -m "chore: migrate sandbox packaging to go"
```

---

## Final Verification

- [ ] Run:

```powershell
cd OpenClaw4j-Sandbox
go test ./...
```

Expected: PASS.

- [ ] Run from repo root when Docker is available:

```powershell
docker build -t openclaw4j-sandbox:local -f OpenClaw4j-Sandbox/Dockerfile .
```

Expected: image builds successfully.

- [ ] If the image builds, run:

```powershell
docker run --rm -d --name openclaw4j-sandbox-test -p 127.0.0.1:9010:9010 openclaw4j-sandbox:local
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/health'
docker stop openclaw4j-sandbox-test
```

Expected: health response contains `status = UP`, and container stops cleanly.
