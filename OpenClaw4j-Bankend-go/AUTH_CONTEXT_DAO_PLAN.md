# Auth Context And DAO Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `OpenClaw4j-Bankend-go` 建立 Go 版鉴权上下文、账号/API key/token session 数据访问边界和最小可测试骨架。

**Architecture:** HTTP middleware 负责提取 token/API key，auth service 负责校验和装配 `RequestContext`，DAO 只做显式 SQL 数据访问。Go 版用 `context.Context` 替代 Java `ScopedValue<RequestContext>`，所有 goroutine、Eino callback、service、DAO 都显式传递 `ctx`。

**Tech Stack:** Go 1.22+、Hertz、`context.Context`、`pgx`、`sqlc`、PostgreSQL、Redis、标准 `testing`。

## Global Constraints

- 仓库根目录是 `D:\IDEA_project\OpenClaw4j`。
- Go 后端目录是 `OpenClaw4j-Bankend-go/`。
- 文档和注释默认使用中文。
- 第一阶段兼容现有 Java 表：`account`、`workspace`、`api_key`。
- DAO 不读取全局上下文，不依赖 goroutine-local storage。
- 不改动 `OpenClaw4j-Sandbox/` 的现有脏文件。

---

## File Structure

- Create: `OpenClaw4j-Bankend-go/go.mod`
  - 定义 Go 后端模块。
- Create: `OpenClaw4j-Bankend-go/internal/contextx/request_context.go`
  - `RequestContext` 类型、`With`、`From`、`MustFrom`、`Detach`。
- Create: `OpenClaw4j-Bankend-go/internal/contextx/request_context_test.go`
  - 验证上下文读写和 detach 行为。
- Create: `OpenClaw4j-Bankend-go/db/migrations/000001_auth_token_session.up.sql`
  - 新增 `auth_token_session`。
- Create: `OpenClaw4j-Bankend-go/db/migrations/000001_auth_token_session.down.sql`
  - 回滚 `auth_token_session`。
- Create: `OpenClaw4j-Bankend-go/db/migrations/000002_api_key_hash_columns.up.sql`
  - 为后续不可逆 API key 存储增加 hash/prefix 字段。
- Create: `OpenClaw4j-Bankend-go/db/migrations/000002_api_key_hash_columns.down.sql`
  - 回滚 API key hash/prefix 字段。
- Create: `OpenClaw4j-Bankend-go/db/schema/go_auth_extensions.sql`
  - sqlc schema 扩展文件，只描述 Go 侧新增表和字段，不包含 down migration。
- Create: `OpenClaw4j-Bankend-go/sqlc.yaml`
  - sqlc 配置。
- Create: `OpenClaw4j-Bankend-go/db/query/account.sql`
  - 账号查询 SQL。
- Create: `OpenClaw4j-Bankend-go/db/query/workspace.sql`
  - 默认 workspace 查询 SQL。
- Create: `OpenClaw4j-Bankend-go/db/query/api_key.sql`
  - API key 查询 SQL。
- Create: `OpenClaw4j-Bankend-go/db/query/auth_token_session.sql`
  - token session SQL。
- Create: `OpenClaw4j-Bankend-go/internal/dao/interfaces.go`
  - DAO 接口定义，供 service 层依赖。
- Create: `OpenClaw4j-Bankend-go/internal/auth/token.go`
  - token hash 和 token id helper。
- Create: `OpenClaw4j-Bankend-go/internal/auth/token_test.go`
  - token helper 单元测试。

---

### Task 1: Request Context

**Files:**
- Create: `OpenClaw4j-Bankend-go/go.mod`
- Create: `OpenClaw4j-Bankend-go/internal/contextx/request_context.go`
- Create: `OpenClaw4j-Bankend-go/internal/contextx/request_context_test.go`

**Interfaces:**
- Produces: `contextx.RequestContext`
- Produces: `contextx.With(ctx context.Context, rc *RequestContext) context.Context`
- Produces: `contextx.From(ctx context.Context) (*RequestContext, bool)`
- Produces: `contextx.MustFrom(ctx context.Context) *RequestContext`
- Produces: `contextx.Detach(ctx context.Context) context.Context`

- [ ] **Step 1: Write failing tests**

```go
package contextx

import (
	"context"
	"testing"
	"time"
)

func TestWithAndFromReturnsRequestContext(t *testing.T) {
	rc := &RequestContext{
		RequestID:   "req-1",
		AccountID:   "10000",
		Username:    "saa",
		AccountType: "admin",
		WorkspaceID: "1",
		CallerIP:    "127.0.0.1",
		Source:      "console",
		AuthType:    AuthTypeConsoleToken,
		StartTime:   time.Unix(1, 0),
	}

	got, ok := From(With(context.Background(), rc))
	if !ok {
		t.Fatal("expected request context to exist")
	}
	if got.RequestID != "req-1" || got.AccountID != "10000" || got.WorkspaceID != "1" {
		t.Fatalf("unexpected request context: %#v", got)
	}
}

func TestFromMissingContextReturnsFalse(t *testing.T) {
	if got, ok := From(context.Background()); ok || got != nil {
		t.Fatalf("expected missing context, got=%#v ok=%v", got, ok)
	}
}

func TestDetachKeepsRequestContextWithoutParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	rc := &RequestContext{RequestID: "req-2", AccountID: "10000"}
	detached := Detach(With(parent, rc))
	cancel()

	select {
	case <-detached.Done():
		t.Fatal("detached context should not inherit parent cancellation")
	default:
	}

	got, ok := From(detached)
	if !ok || got.RequestID != "req-2" {
		t.Fatalf("unexpected detached request context: %#v ok=%v", got, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/contextx`

Expected: FAIL because `RequestContext` and helper functions do not exist.

- [ ] **Step 3: Implement contextx**

```go
package contextx

import (
	"context"
	"time"
)

type AuthType string

const (
	AuthTypeConsoleToken AuthType = "console_token"
	AuthTypeAPIKey       AuthType = "api_key"
)

type RequestContext struct {
	StartTime   time.Time `json:"start_time"`
	RequestID   string    `json:"request_id"`
	AccountID   string    `json:"account_id"`
	Username    string    `json:"username"`
	AccountType string    `json:"account_type"`
	WorkspaceID string    `json:"workspace_id"`
	CallerIP    string    `json:"caller_ip"`
	Source      string    `json:"source"`
	AuthType    AuthType  `json:"auth_type"`
}

type requestContextKey struct{}

func With(ctx context.Context, rc *RequestContext) context.Context {
	if rc == nil {
		return ctx
	}
	return context.WithValue(ctx, requestContextKey{}, rc)
}

func From(ctx context.Context) (*RequestContext, bool) {
	if ctx == nil {
		return nil, false
	}
	rc, ok := ctx.Value(requestContextKey{}).(*RequestContext)
	return rc, ok
}

func MustFrom(ctx context.Context) *RequestContext {
	rc, ok := From(ctx)
	if !ok {
		panic("request context is not bound")
	}
	return rc
}

func Detach(ctx context.Context) context.Context {
	rc, ok := From(ctx)
	if !ok {
		return context.Background()
	}
	return With(context.Background(), rc)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/contextx`

Expected: PASS.

---

### Task 2: SQL Schema And Queries

**Files:**
- Create: `OpenClaw4j-Bankend-go/db/migrations/000001_auth_token_session.up.sql`
- Create: `OpenClaw4j-Bankend-go/db/migrations/000001_auth_token_session.down.sql`
- Create: `OpenClaw4j-Bankend-go/sqlc.yaml`
- Create: `OpenClaw4j-Bankend-go/db/query/account.sql`
- Create: `OpenClaw4j-Bankend-go/db/query/workspace.sql`
- Create: `OpenClaw4j-Bankend-go/db/query/api_key.sql`
- Create: `OpenClaw4j-Bankend-go/db/query/auth_token_session.sql`

**Interfaces:**
- Produces SQL query names: `FindActiveAccountByID`, `FindActiveAccountByUsername`, `UpdateAccountLastLogin`, `FindDefaultWorkspaceByAccountID`, `FindActiveAPIKeyByEncryptedKey`, `FindActiveAPIKeyByHash`, `CountActiveAPIKeysByAccountID`, `CreateTokenSession`, `FindActiveTokenSessionByHash`, `RevokeTokenSessionByHash`, `DeleteExpiredTokenSessions`.

- [ ] **Step 1: Create migration**

Use the DDL from `AUTH_CONTEXT_DAO_DESIGN.md` for `auth_token_session`.

- [ ] **Step 2: Create sqlc config**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/query"
    schema:
      - "../OpenClaw4j-Bankend/src/main/resources/sql/PostgreSQL/V0.0.1__init.sql"
      - "db/schema"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_interface: true
```

- [ ] **Step 3: Create query SQL**

Write each named query exactly as described in `AUTH_CONTEXT_DAO_DESIGN.md`, with `-- name: QueryName :one`, `:many`, or `:exec`.

- [ ] **Step 4: Validate sqlc files are discoverable**

Run: `Get-ChildItem -Recurse OpenClaw4j-Bankend-go\db,OpenClaw4j-Bankend-go\sqlc.yaml`

Expected: migration and query files are present.

---

### Task 3: DAO Interfaces

**Files:**
- Create: `OpenClaw4j-Bankend-go/internal/dao/interfaces.go`

**Interfaces:**
- Produces: `AccountDAO`
- Produces: `WorkspaceDAO`
- Produces: `APIKeyDAO`
- Produces: `TokenSessionDAO`

- [ ] **Step 1: Write interface definitions**

Use the interface signatures from `AUTH_CONTEXT_DAO_DESIGN.md`. Keep domain structs minimal and aligned with table columns required by auth.

- [ ] **Step 2: Run compile check**

Run: `go test ./internal/dao`

Expected: PASS after `go.mod` exists.

---

### Task 4: Token Helper

**Files:**
- Create: `OpenClaw4j-Bankend-go/internal/auth/token.go`
- Create: `OpenClaw4j-Bankend-go/internal/auth/token_test.go`

**Interfaces:**
- Produces: `HashToken(token string) string`
- Produces: `TokenID(token string) string`

- [ ] **Step 1: Write failing tests**

```go
package auth

import "testing"

func TestHashTokenIsStableAndDoesNotExposePlaintext(t *testing.T) {
	token := "oc_access_secret"
	first := HashToken(token)
	second := HashToken(token)
	if first != second {
		t.Fatal("expected stable token hash")
	}
	if first == token || len(first) != 64 {
		t.Fatalf("unexpected hash: %q", first)
	}
}

func TestTokenIDUsesHashPrefix(t *testing.T) {
	id := TokenID("oc_access_secret")
	if len(id) != 16 {
		t.Fatalf("expected 16-char token id, got %q", id)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth`

Expected: FAIL because helper functions do not exist.

- [ ] **Step 3: Implement token helper**

```go
package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TokenID(token string) string {
	hash := HashToken(token)
	return hash[:16]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/auth`

Expected: PASS.

---

## Self-Review

- Spec coverage: 覆盖 Go 上下文传播、console token、API key、token session、DAO 边界和测试策略。
- Placeholder scan: 本计划不包含 TBD/TODO/implement later。
- Type consistency: `RequestContext`、DAO 接口和 token helper 名称在各任务中保持一致。
