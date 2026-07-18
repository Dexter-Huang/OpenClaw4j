# Go 后端交接说明

记录日期：2026-07-19

这份文档给后续接手 `OpenClaw4j-Bankend-go` 的 agent 看，重点是当前已经完成了什么、为什么这么做、下一步别踩哪些坑。

## 当前状态

- Go 后端已经能启动，并通过 `go test ./...`。
- HTTP 框架使用 Hertz。
- 认证链路已经从“新建 token 表”改成“Redis 会话存储”，和 Java 侧 `TokenManager` 的思路对齐。
- 主业务数据仍然走同一个 PostgreSQL，不需要为了 token 再单独建表。
- 当前 backend 监听端口是 `9004`。

## 已完成的关键改动

1. `internal/auth/service.go`
   - 登录和 refresh 都改成按原始 token 查 session。
   - access token TTL 是 2 小时，refresh token TTL 是 30 天。
   - 生成 session 时，`TokenID` 现在直接存原始 token，方便 Redis key 对齐。

2. `internal/dao/redisdao/store.go`
   - 新增 Redis token session DAO。
   - key 规则：
     - `access_token:<token>`
     - `refresh_token:<token>`
   - 存储内容包含 `account_id`、`token_type`、`token_hash`、`expires_at`、`source`、`caller_ip`、`user_agent`、`tenant_id`。

3. `cmd/openclaw4j-backend-go/main.go`
   - 启动时创建 Redis client 并 `Ping`。
   - 把 Redis token store 注入 bootstrap。

4. `internal/bootstrap/app.go`
   - 去掉了 token 表的启动建表逻辑。
   - bootstrap 改为接收 `TokenSessions` 依赖。

5. `internal/config/config.go`
   - 新增 Redis 配置解析：
     - `OPENCLAW_REDIS_HOST`
     - `OPENCLAW_REDIS_PORT`
     - `OPENCLAW_REDIS_PASSWORD`
     - `OPENCLAW_REDIS_DATABASE`

6. `internal/transport/hertz/middleware/auth.go`
   - 保留兼容的 console token / API key 提取方式。
   - console token 支持 `X-SAA-TOKEN` 和 query `access_token`。
   - API key 支持 `Authorization` 和 `X-API-Key`。

7. `internal/transport/hertz/server/server_compat_test.go`
   - 这类兼容测试要保留，它能防止路由和鉴权行为回退。

## 关键设计决策

- 不再新增 token 表。
- session 只作为短期状态放 Redis。
- PostgreSQL 继续承载账号、workspace、API key 等主业务数据。
- `internal/dao/pgdao/store.go` 里的 token DAO 先保留，主要用于兼容和测试，不要随手删。
- 认证链路以“兼容 Java 现有前端请求方式”为先，不要为了 Go 化改掉 header 语义。

## 主要文件地图

- `cmd/openclaw4j-backend-go/main.go`：入口、配置、DB/Redis 初始化。
- `internal/bootstrap/app.go`：依赖装配和 server 启动。
- `internal/config/config.go`：环境变量配置。
- `internal/auth/service.go`：登录、refresh、鉴权上下文装配。
- `internal/auth/token.go`：token hash / token id helper。
- `internal/dao/interfaces.go`：DAO 接口。
- `internal/dao/redisdao/store.go`：Redis token session 实现。
- `internal/dao/pgdao/store.go`：PostgreSQL DAO 兼容实现。
- `internal/transport/hertz/server/server.go`：Hertz server 组装。
- `internal/transport/hertz/middleware/auth.go`：鉴权中间件。

## 验证方式

最小验证：

```powershell
cd OpenClaw4j-Bankend-go
$env:GOPROXY='off'
$env:GOSUMDB='off'
go test ./...
```

运行验证时常用环境：

- `OPENCLAW_DATABASE_DSN=postgres://...`
- `OPENCLAW_REDIS_HOST=127.0.0.1`
- `OPENCLAW_REDIS_PORT=6379`
- `OPENCLAW_REDIS_DATABASE=0`

建议继续确认的接口：

- `GET /healthz`
- `POST /console/v1/auth/login`
- `GET /console/v1/accounts/profile`
- `POST /console/v1/auth/refresh-token`

## 下一步建议

1. 决定是否继续保留 `TokenID` / PG token DAO 的兼容分支。
2. 给 console auth、refresh、profile 补更完整的契约测试。
3. 如果继续扩 Go 后端，优先保持“同库 + Redis session”这个边界，不要重新引入 token 表。
4. 新增功能前先确认是否要兼容现有 Java 前端的请求头、响应体和错误码。

## 已知注意事项

- `OpenClaw4j-Sandbox/` 里还有用户自己的改动，不要回滚。
- 这条 Go 后端线已经走向“兼容 Java 行为 + 逐步替换”的路线，不是重新设计一套登录体系。
- 只要是会影响前端请求头、token 格式、refresh 逻辑、Redis key 规则的改动，都要先看现有测试，再动代码。
