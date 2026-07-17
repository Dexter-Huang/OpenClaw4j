# Agent-Compatible Sandbox APIs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 Go 版 `OpenClaw4j-Sandbox` 中实现 Agent 可用的 AIO 风格 file、bash 和 MCP 兼容接口。

**Architecture:** 在现有 Hertz 服务中新增 `internal/pathguard`、`internal/fileapi`、`internal/bashapi`、`internal/mcpapi` 四个内部包。`fileapi` 与 `bashapi` 共用 workspace 路径约束，`mcpapi` 通过内部 service 调用 file/bash 能力并提供 JSON-RPC 与 REST 包装，`cmd/openclaw4j-sandbox/main.go` 只负责组装路由和共享服务。

**Tech Stack:** Go 1.22、CloudWeGo Hertz、标准库 `os`/`path/filepath`/`regexp`/`os/exec`、`github.com/google/uuid`。

## Global Constraints

- 文档、设计说明和新增代码注释默认使用中文。
- Git 命令必须从 `D:\IDEA_project\OpenClaw4j` 根仓库运行。
- `OpenClaw4j-Bankend/`、`OpenClaw4j-Frontend/`、`OpenClaw4j-Sandbox/` 都是根仓库普通子目录。
- 本计划不实现完整 AIO Sandbox。
- 本计划不实现浏览器、Jupyter、Code Server、Node.js 会话、display、proxy、skills 等 AIO 子系统。
- 本计划不实现 `/v1/file/upload`、`/v1/file/download`、file watch、SSE 事件流。
- 本计划不实现 `sudo` 参数和受保护系统路径写入。
- 本计划不实现外部 MCP Hub 聚合，只提供当前 Go sandbox 内置工具。
- 所有 file/bash 路径默认限制在 `OPENCLAW_SANDBOX_WORKSPACE_DIR` 内。
- `/health` 和 `/v1/execute` 行为必须保持兼容。
- Go 改动完成前的最小验证以定向 `go test` 为准，最终运行 `go test ./...`。
- 不主动运行后端 Java `spotless:check`、`checkstyle:check`、`spotbugs:check`，除非用户明确要求。

---

## File Structure

- Modify: `OpenClaw4j-Sandbox/internal/config/config.go`
  - 增加 workspace、bash、file 限制配置字段和环境变量读取。
- Create: `OpenClaw4j-Sandbox/internal/pathguard/pathguard.go`
  - 负责 workspace 路径归一化、目录创建和逃逸检查。
- Create: `OpenClaw4j-Sandbox/internal/fileapi/service.go`
  - 负责 `/v1/file/*` 请求/响应模型和文件 service。
- Create: `OpenClaw4j-Sandbox/internal/fileapi/routes.go`
  - 负责 Hertz file API handler 注册。
- Create: `OpenClaw4j-Sandbox/internal/bashapi/service.go`
  - 负责 bash session/command 状态、进程管理和输出缓存。
- Create: `OpenClaw4j-Sandbox/internal/bashapi/routes.go`
  - 负责 Hertz bash API handler 注册。
- Create: `OpenClaw4j-Sandbox/internal/mcpapi/service.go`
  - 负责 MCP tool registry 和工具分发。
- Create: `OpenClaw4j-Sandbox/internal/mcpapi/routes.go`
  - 负责 `/mcp` JSON-RPC 和 `/v1/mcp/*` REST 包装。
- Modify: `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`
  - 注册新增服务和路由。
- Modify: `OpenClaw4j-Sandbox/README.md`
  - 补充 workspace、file、bash、MCP 调用示例和不兼容项。

## Task 1: Workspace Config And Path Guard

**Files:**
- Modify: `OpenClaw4j-Sandbox/internal/config/config.go`
- Modify: `OpenClaw4j-Sandbox/internal/config/config_test.go`
- Create: `OpenClaw4j-Sandbox/internal/pathguard/pathguard.go`
- Create: `OpenClaw4j-Sandbox/internal/pathguard/pathguard_test.go`

**Interfaces:**
- Produces: `config.Config.WorkspaceDir string`
- Produces: `config.Config.BashDefaultTimeoutMs uint64`
- Produces: `config.Config.BashHardTimeoutMs uint64`
- Produces: `config.Config.BashOutputLimitBytes int`
- Produces: `config.Config.FileReadLimitBytes int`
- Produces: `pathguard.New(root string) pathguard.Guard`
- Produces: `Guard.EnsureWorkspace() error`
- Produces: `Guard.Resolve(input string) (string, error)`
- Produces: `pathguard.Error{Input string, Message string, ErrorType string}`

- [ ] **Step 1: Write failing path guard tests**

Create `OpenClaw4j-Sandbox/internal/pathguard/pathguard_test.go`:

```go
package pathguard

import "testing"

func TestResolveAllowsPathsInsideWorkspace(t *testing.T) {
	guard := New("/tmp/openclaw4j-workspace")

	relative, err := guard.Resolve("reports/out.txt")
	if err != nil {
		t.Fatalf("relative path should resolve: %v", err)
	}
	if relative != "/tmp/openclaw4j-workspace/reports/out.txt" {
		t.Fatalf("unexpected relative resolution: %s", relative)
	}

	absolute, err := guard.Resolve("/tmp/openclaw4j-workspace/reports/out.txt")
	if err != nil {
		t.Fatalf("absolute path inside workspace should resolve: %v", err)
	}
	if absolute != "/tmp/openclaw4j-workspace/reports/out.txt" {
		t.Fatalf("unexpected absolute resolution: %s", absolute)
	}
}

func TestResolveRejectsWorkspaceEscape(t *testing.T) {
	guard := New("/tmp/openclaw4j-workspace")

	for _, input := range []string{"../secret.txt", "/etc/passwd"} {
		_, err := guard.Resolve(input)
		if err == nil {
			t.Fatalf("expected %s to be rejected", input)
		}
		guardErr, ok := err.(Error)
		if !ok {
			t.Fatalf("expected pathguard.Error, got %T", err)
		}
		if guardErr.ErrorType != "invalid_path" {
			t.Fatalf("unexpected error type: %s", guardErr.ErrorType)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
go test ./internal/pathguard -run TestResolve -v
```

Expected: FAIL because `internal/pathguard` does not exist or `New` is undefined.

- [ ] **Step 3: Implement minimal path guard**

Create `OpenClaw4j-Sandbox/internal/pathguard/pathguard.go`:

```go
package pathguard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Guard struct {
	root string
}

type Error struct {
	Input     string
	Message   string
	ErrorType string
}

func (e Error) Error() string {
	return e.Message
}

func New(root string) Guard {
	return Guard{root: filepath.Clean(root)}
}

func (g Guard) Root() string {
	return g.root
}

func (g Guard) EnsureWorkspace() error {
	return os.MkdirAll(g.root, 0o755)
}

func (g Guard) Resolve(input string) (string, error) {
	cleanInput := filepath.Clean(input)
	candidate := cleanInput
	if !filepath.IsAbs(cleanInput) {
		candidate = filepath.Join(g.root, cleanInput)
	}
	candidate = filepath.Clean(candidate)

	rel, err := filepath.Rel(g.root, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", Error{
			Input:     input,
			Message:   fmt.Sprintf("path must stay inside workspace %s", g.root),
			ErrorType: "invalid_path",
		}
	}

	return candidate, nil
}
```

- [ ] **Step 4: Verify path guard passes**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
go test ./internal/pathguard -v
```

Expected: PASS.

- [ ] **Step 5: Write failing config defaults test**

Extend `OpenClaw4j-Sandbox/internal/config/config_test.go` with:

```go
func TestFromEnvUsesAgentCompatibilityDefaults(t *testing.T) {
	cfg := FromEnv()

	if cfg.WorkspaceDir != "/tmp/openclaw4j-workspace" {
		t.Fatalf("unexpected workspace dir: %s", cfg.WorkspaceDir)
	}
	if cfg.BashDefaultTimeoutMs != 30000 {
		t.Fatalf("unexpected bash default timeout: %d", cfg.BashDefaultTimeoutMs)
	}
	if cfg.BashHardTimeoutMs != 120000 {
		t.Fatalf("unexpected bash hard timeout: %d", cfg.BashHardTimeoutMs)
	}
	if cfg.BashOutputLimitBytes != 65536 {
		t.Fatalf("unexpected bash output limit: %d", cfg.BashOutputLimitBytes)
	}
	if cfg.FileReadLimitBytes != 1048576 {
		t.Fatalf("unexpected file read limit: %d", cfg.FileReadLimitBytes)
	}
}
```

- [ ] **Step 6: Run config test to verify it fails**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
go test ./internal/config -run TestFromEnvUsesAgentCompatibilityDefaults -v
```

Expected: FAIL because the new fields do not exist.

- [ ] **Step 7: Implement config fields**

Modify `OpenClaw4j-Sandbox/internal/config/config.go`:

```go
type Config struct {
	Bind                 string
	WorkDir              string
	WorkspaceDir         string
	DefaultTimeoutMs     uint64
	MaxTimeoutMs         uint64
	MemoryLimit          string
	ProcessLimit         uint32
	StdoutLimitBytes     int
	StderrLimitBytes     int
	DepsDir              string
	PythonRuntime        string
	NodePath             string
	BashDefaultTimeoutMs uint64
	BashHardTimeoutMs    uint64
	BashOutputLimitBytes int
	FileReadLimitBytes   int
}
```

Add to `FromEnv()`:

```go
WorkspaceDir:         envString("OPENCLAW_SANDBOX_WORKSPACE_DIR", "/tmp/openclaw4j-workspace"),
BashDefaultTimeoutMs: envUint64("OPENCLAW_SANDBOX_BASH_DEFAULT_TIMEOUT_MS", 30000),
BashHardTimeoutMs:    envUint64("OPENCLAW_SANDBOX_BASH_HARD_TIMEOUT_MS", 120000),
BashOutputLimitBytes: envInt("OPENCLAW_SANDBOX_BASH_OUTPUT_LIMIT_BYTES", 65536),
FileReadLimitBytes:   envInt("OPENCLAW_SANDBOX_FILE_READ_LIMIT_BYTES", 1048576),
```

- [ ] **Step 8: Verify Task 1 passes**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
go test ./internal/config ./internal/pathguard -v
go test ./...
```

Expected: PASS.

- [ ] **Step 9: Commit Task 1**

Run from repo root:

```powershell
cd D:\IDEA_project\OpenClaw4j
git add OpenClaw4j-Sandbox/internal/config OpenClaw4j-Sandbox/internal/pathguard
git commit -m "feat: add sandbox workspace path guard"
```

## Task 2: Basic File API

**Files:**
- Create: `OpenClaw4j-Sandbox/internal/fileapi/service.go`
- Create: `OpenClaw4j-Sandbox/internal/fileapi/service_test.go`
- Create: `OpenClaw4j-Sandbox/internal/fileapi/routes.go`
- Modify: `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`

**Interfaces:**
- Consumes: `pathguard.Guard.Resolve(input string) (string, error)`
- Produces: `fileapi.Service`
- Produces methods: `Read`, `Write`, `Replace`, `List`
- Produces routes: `POST /v1/file/read`, `POST /v1/file/write`, `POST /v1/file/replace`, `POST /v1/file/list`

- [ ] Write failing service tests for write/read, invalid path, replace, and list.
- [ ] Run `go test ./internal/fileapi -v` and verify failure.
- [ ] Implement `Response`, request structs, and `Service` methods using `os`, `io/fs`, `path/filepath`.
- [ ] Add Hertz handlers in `routes.go`; handlers bind JSON and call service.
- [ ] Wire service/routes from `cmd/openclaw4j-sandbox/main.go`.
- [ ] Run `go test ./internal/fileapi -v` and `go test ./...`.
- [ ] Commit: `feat: add sandbox file APIs`.

## Task 3: File Search And Agent Editor API

**Files:**
- Modify: `OpenClaw4j-Sandbox/internal/fileapi/service.go`
- Modify: `OpenClaw4j-Sandbox/internal/fileapi/service_test.go`
- Modify: `OpenClaw4j-Sandbox/internal/fileapi/routes.go`

**Interfaces:**
- Produces methods: `Search`, `Find`, `Grep`, `Glob`, `StrReplaceEditor`
- Produces routes: `POST /v1/file/search`, `POST /v1/file/find`, `POST /v1/file/grep`, `POST /v1/file/glob`, `POST /v1/file/str_replace_editor`

- [ ] Write failing tests for regex search, recursive grep, glob, and `str_replace_editor` commands `view/create/str_replace/insert/undo_edit` unsupported.
- [ ] Run `go test ./internal/fileapi -run 'TestSearch|TestGrep|TestGlob|TestStrReplaceEditor' -v` and verify failure.
- [ ] Implement search using `regexp`, recursive traversal using `filepath.WalkDir`, and editor commands with explicit operation validation.
- [ ] Register the remaining file routes.
- [ ] Run `go test ./internal/fileapi -v` and `go test ./...`.
- [ ] Commit: `feat: add sandbox file search APIs`.

## Task 4: Bash API

**Files:**
- Create: `OpenClaw4j-Sandbox/internal/bashapi/service.go`
- Create: `OpenClaw4j-Sandbox/internal/bashapi/service_test.go`
- Create: `OpenClaw4j-Sandbox/internal/bashapi/routes.go`
- Modify: `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`

**Interfaces:**
- Consumes: `pathguard.Guard`
- Produces: `bashapi.Service`
- Produces methods: `Exec`, `Output`, `Write`, `Kill`, `Sessions`, `CreateSession`, `CloseSession`
- Produces routes: `POST /v1/bash/exec`, `POST /v1/bash/output`, `POST /v1/bash/write`, `POST /v1/bash/kill`, `GET /v1/bash/sessions`, `POST /v1/bash/sessions/create`, `POST /v1/bash/sessions/:session_id/close`

- [ ] Write failing tests for short command, stderr, non-zero exit, async output polling, stdin write, and kill.
- [ ] Run `go test ./internal/bashapi -v` and verify failure.
- [ ] Implement session and command state with `sync.Mutex`, `exec.CommandContext`, stdout/stderr readers, and bounded output buffers.
- [ ] Add Windows local fallback using PowerShell and Linux production command using `/bin/bash -lc`.
- [ ] Add Hertz handlers in `routes.go`; handlers bind JSON and call service.
- [ ] Wire service/routes from `cmd/openclaw4j-sandbox/main.go`.
- [ ] Run `go test ./internal/bashapi -v` and `go test ./...`.
- [ ] Commit: `feat: add sandbox bash APIs`.

## Task 5: MCP REST Wrapper And JSON-RPC Entry

**Files:**
- Create: `OpenClaw4j-Sandbox/internal/mcpapi/service.go`
- Create: `OpenClaw4j-Sandbox/internal/mcpapi/service_test.go`
- Create: `OpenClaw4j-Sandbox/internal/mcpapi/routes.go`
- Modify: `OpenClaw4j-Sandbox/cmd/openclaw4j-sandbox/main.go`

**Interfaces:**
- Consumes: `fileapi.Service`
- Consumes: `bashapi.Service`
- Produces routes: `POST /mcp`, `GET /v1/mcp/servers`, `GET /v1/mcp/:server_name/tools`, `POST /v1/mcp/:server_name/tools/:tool_name`
- Produces tools: `file_read`, `file_write`, `file_list`, `file_replace`, `file_search`, `sandbox_execute_bash`

- [ ] Write failing tests for `tools/list`, `tools/call file_write`, `tools/call sandbox_execute_bash`, and unknown tool.
- [ ] Run `go test ./internal/mcpapi -v` and verify failure.
- [ ] Implement JSON-RPC request/response structs, tool registry, and tool dispatch to file/bash services.
- [ ] Add REST wrapper handlers for server list, tool list, and tool call.
- [ ] Wire MCP routes from `cmd/openclaw4j-sandbox/main.go`.
- [ ] Run `go test ./internal/mcpapi -v` and `go test ./...`.
- [ ] Commit: `feat: add sandbox MCP tool APIs`.

## Task 6: Documentation And Final Verification

**Files:**
- Modify: `OpenClaw4j-Sandbox/README.md`

**Interfaces:**
- Consumes: file, bash, MCP routes.
- Produces: documented examples for `/v1/file/read`, `/v1/bash/exec`, `/mcp`.

- [ ] Write README sections for workspace config, file examples, bash examples, MCP examples, and known non-goals.
- [ ] Run `go test ./...`.
- [ ] Optionally run a local HTTP smoke test if no port conflict exists: `/health`, `/v1/file/write`, `/v1/file/read`, `/v1/bash/exec`, `/mcp` `tools/list`.
- [ ] Commit: `docs: document sandbox agent-compatible APIs`.

## Final Verification

- [ ] Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
go test ./...
```

Expected: PASS.

- [ ] From root, inspect tracked changes:

```powershell
cd D:\IDEA_project\OpenClaw4j
git status --short
```

Expected: only intentional sandbox files remain modified or untracked. Existing unrelated backend/frontend dirty files may remain; do not revert them.

## Self-Review Notes

- Spec coverage: path guard, file API, bash API, MCP API, route registration, README, and `go test` verification are covered.
- Scope control: upload/download/watch/browser/Jupyter/external MCP Hub/sudo are explicitly excluded from implementation tasks.
- Type consistency: `pathguard.Guard`, `fileapi.Service`, `bashapi.Service`, and MCP service adapters are the shared interfaces between tasks.
