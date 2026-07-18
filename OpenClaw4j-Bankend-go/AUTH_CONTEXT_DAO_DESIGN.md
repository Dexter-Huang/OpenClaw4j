# Go 后端鉴权上下文与 DAO 设计

评估日期：2026-07-18

## 设计目标

Go 版后端用 `context.Context` 承接 Java 后端的 `ScopedValue<RequestContext>`。鉴权 middleware 在请求入口构造 `RequestContext`，后续 controller、service、DAO、Eino callback、workflow goroutine 都显式接收并传递 `ctx`。

这套设计优先兼容现有 Java 后端表结构，允许 Java 和 Go 在迁移期共用 PostgreSQL、Redis 和前端接口契约。

## Java 到 Go 的语义映射

| Java 现状 | Go 设计 |
| --- | --- |
| Spring virtual threads | goroutine + `context.Context` |
| `ScopedValue<RequestContext>` | `context.WithValue(ctx, requestContextKey, *RequestContext)` |
| `OncePerRequestFilter` | HTTP middleware |
| `TokenAuthInterceptor` | console token middleware |
| `ApiKeyAuthInterceptor` | OpenAPI API key middleware |
| `RequestContextHolder.getRequestContext()` | `contextx.From(ctx)` |
| `RequestContextHolder.callWithRequestContext(...)` | 显式传入 `ctx` 或 `contextx.With(ctx, rc)` |

Go 不建议实现协程本地变量。原因是 goroutine 没有稳定的官方 goroutine-local storage，靠全局 map 绑定 goroutine id 会引入泄漏、竞争和调试困难。`context.Context` 更适合请求级元数据、取消信号、超时和 trace 传播。

## 上下文模型

建议 Go 侧定义：

```go
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
```

`Source` 默认值：

- `/console/v1/*`：`console`
- `/api/v1/*`：`openapi`
- 内部任务：`internal`

## goroutine 传播规则

任何异步任务都必须显式接收父 `ctx`：

```go
func GoWithContext(ctx context.Context, fn func(context.Context)) {
    go fn(ctx)
}
```

后台长任务如果不能继承请求取消信号，应复制 request metadata，但替换 cancel 生命周期：

```go
func DetachRequestContext(parent context.Context) context.Context {
    rc, ok := contextx.From(parent)
    if !ok {
        return context.Background()
    }
    return contextx.With(context.Background(), rc)
}
```

这样可以避免 SSE 断开后误杀必须继续运行的 workflow，同时保留 `request_id/account_id/workspace_id` 用于审计和日志。

## 鉴权入口

### console token

兼容 Java：

- URL：`/console/v1/*`、`/starter.zip`
- Header：`X-SAA-TOKEN: Bearer <access_token>`
- Query：`access_token=<access_token>`
- 白名单：`/console/v1/auth/login`、`/console/v1/auth/refresh-token`、`/console/v1/system/*`、Swagger、`/test/*`

校验流程：

1. 提取 access token。
2. 从 Redis 或 `auth_token_session` 查询 token。
3. 根据 `account_id` 查询有效账号。
4. 查询默认 workspace。
5. 构造 `RequestContext` 写入 `ctx`。

### API key

兼容 Java：

- URL：`/api/v1/*`
- Header：`Authorization: Bearer <api_key>`

校验流程：

1. 提取 API key。
2. 第一阶段兼容 Java：按现有 AES 规则加密后查询 `api_key.api_key`。
3. 第二阶段升级：优先按 `api_key_hash` 查询，兼容期保留旧字段 fallback。
4. 查询有效账号和默认 workspace。
5. 构造 `RequestContext`，`source=openapi`，`auth_type=api_key`。

## 表结构设计

### 复用现有表

第一阶段直接复用 Java 已有表：

- `account`
- `workspace`
- `api_key`

Go DAO 不新增字段即可完成基础鉴权。

### 新增登录态表

建议新增 `auth_token_session`，把 token 生命周期从 Redis-only 变成 Redis + DB 可审计模型。Redis 仍作为热路径，DB 用于审计、主动吊销和多实例一致性兜底。

```sql
CREATE TABLE IF NOT EXISTS auth_token_session (
  id BIGSERIAL NOT NULL,
  token_id varchar(64) NOT NULL,
  account_id varchar(64) NOT NULL,
  token_type varchar(32) NOT NULL,
  token_hash varchar(128) NOT NULL,
  expires_at timestamp NOT NULL,
  revoked smallint NOT NULL DEFAULT 0,
  source varchar(64) DEFAULT NULL,
  caller_ip varchar(64) DEFAULT NULL,
  user_agent varchar(512) DEFAULT NULL,
  gmt_create timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  gmt_modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  tenant_id bigint DEFAULT 0,
  PRIMARY KEY (id),
  CONSTRAINT uk_auth_token_session_token_hash UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_auth_token_session_account_id
  ON auth_token_session (account_id);

CREATE INDEX IF NOT EXISTS idx_auth_token_session_expires_at
  ON auth_token_session (expires_at);

CREATE INDEX IF NOT EXISTS idx_auth_token_session_account_revoked
  ON auth_token_session (account_id, revoked);
```

字段说明：

- `token_id`：非敏感 token 标识，用于日志与排查。
- `token_hash`：token 原文的 SHA-256/hex，不保存明文 token。
- `token_type`：`access` 或 `refresh`。
- `revoked`：软吊销标记。
- `expires_at`：过期时间，查询时必须过滤。

### API key 后续增强

现有 `api_key.api_key` 是可逆 AES 密文。为了兼容 Java 第一阶段先保留，后续建议增加：

```sql
ALTER TABLE api_key ADD COLUMN IF NOT EXISTS api_key_hash varchar(128);
ALTER TABLE api_key ADD COLUMN IF NOT EXISTS api_key_prefix varchar(32);
CREATE UNIQUE INDEX IF NOT EXISTS uk_api_key_hash
  ON api_key (api_key_hash)
  WHERE api_key_hash IS NOT NULL;
```

新建 API key 时仅返回一次明文，DB 保存 hash 和 prefix。旧数据继续通过 AES 字段兼容读取。

## DAO 设计

Web 框架选用 Hertz，数据访问选用 `pgx` + `sqlc`。理由：

- SQL 直接可审查，迁移期最容易保证和 Java MyBatis Plus 行为一致。
- 编译期生成类型，减少字段拼写错误。
- 后续复杂 pgvector/filter 查询也更适合手写 SQL。

目录建议：

```text
OpenClaw4j-Bankend-go/
  db/
    migrations/
    schema/
    query/
      account.sql
      workspace.sql
      api_key.sql
      auth_token_session.sql
  internal/
    auth/
    contextx/
    dao/
    service/
```

### AccountDAO

```go
type AccountDAO interface {
    FindActiveByAccountID(ctx context.Context, accountID string) (*Account, error)
    FindActiveByUsername(ctx context.Context, username string) (*Account, error)
    UpdateLastLogin(ctx context.Context, accountID string, lastLogin time.Time) error
}
```

关键 SQL：

```sql
SELECT id, account_id, username, email, mobile, password, nickname, icon,
       type, status, gmt_create, gmt_modified, gmt_last_login, creator, modifier, tenant_id
FROM account
WHERE account_id = $1 AND status <> 0
LIMIT 1;
```

### WorkspaceDAO

```go
type WorkspaceDAO interface {
    FindDefaultByAccountID(ctx context.Context, accountID string) (*Workspace, error)
}
```

现有 Java 逻辑没有显式 `is_default` 字段。第一阶段按账号查询第一个未删除 workspace：

```sql
SELECT id, workspace_id, account_id, status, name, description, config,
       gmt_create, gmt_modified, creator, modifier, tenant_id
FROM workspace
WHERE account_id = $1 AND status <> 0
ORDER BY id ASC
LIMIT 1;
```

后续如果需要多 workspace 切换，再补 `workspace_member` 或默认 workspace 字段。

### APIKeyDAO

```go
type APIKeyDAO interface {
    FindActiveByEncryptedKey(ctx context.Context, encryptedKey string) (*APIKey, error)
    FindActiveByHash(ctx context.Context, keyHash string) (*APIKey, error)
    CountActiveByAccountID(ctx context.Context, accountID string) (int64, error)
    ListActiveByAccountID(ctx context.Context, accountID string, limit int32, offset int32) ([]APIKey, error)
}
```

第一阶段必须支持 `FindActiveByEncryptedKey`，确保 Java 创建的 API key 在 Go 里可用。

### TokenSessionDAO

```go
type TokenSessionDAO interface {
    Create(ctx context.Context, session TokenSession) error
    FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*TokenSession, error)
    RevokeByTokenHash(ctx context.Context, tokenHash string) error
    RevokeAccountTokens(ctx context.Context, accountID string, tokenType string) error
    DeleteExpired(ctx context.Context, before time.Time) error
}
```

关键 SQL：

```sql
SELECT id, token_id, account_id, token_type, token_hash, expires_at, revoked,
       source, caller_ip, user_agent, gmt_create, gmt_modified, tenant_id
FROM auth_token_session
WHERE token_hash = $1
  AND token_type = $2
  AND revoked = 0
  AND expires_at > $3
LIMIT 1;
```

## Mapper/DAO 与业务边界

DAO 只负责数据访问，不读取 `RequestContext`，避免隐式依赖。需要 account/workspace 的地方由 service 显式传参：

```go
func (s *AuthService) AuthenticateConsoleToken(ctx context.Context, token string) (*RequestContext, error)
```

service 负责：

- 解密或 hash token。
- 查询 DAO。
- 校验状态。
- 装配 `RequestContext`。
- 写 Redis/token session。

middleware 负责：

- 从 HTTP 请求提取凭证。
- 调用 service。
- 将 `RequestContext` 写入 `ctx`。
- 返回兼容 Java `Result` 的 401 错误。

## 错误返回

Go 版鉴权错误要兼容 Java：

```json
{
  "success": false,
  "request_id": "uuid",
  "code": "...",
  "message": "..."
}
```

第一阶段至少区分：

- `INVALID_TOKEN`
- `INVALID_REFRESH_TOKEN`
- `INVALID_API_KEY`
- `ACCOUNT_NOT_FOUND`
- `PERMISSION_DENIED`

错误码文本以 Java `ErrorCode` 为准，避免前端判断分叉。

## 测试策略

优先补单元测试，不依赖真实 PostgreSQL：

- `contextx`：能写入和读取 request context；没有上下文时返回 false。
- token hash：同一 token hash 稳定，不泄露明文。
- auth service：mock DAO 下验证 token/API key 成功和失败路径。
- middleware：无 token、非法 token、合法 token 三条路径返回符合预期的 `Result`。

集成测试后置：

- 用 testcontainers 或本地 PostgreSQL 验证 sqlc query。
- Java 生成的 API key 可被 Go 识别。
- Java/Go 对同一个账号、workspace 的鉴权上下文一致。
