# Go 后端测试约定

## 测试分层

- 单元测试与被测代码同目录，文件名使用 `*_test.go`。默认测试只能使用 fake、内存实现或 `httptest`，不得依赖本机 PostgreSQL、Redis、Sandbox 或第三方网络服务。
- 集成测试同样与对应包放在一起，文件名使用 `*_integration_test.go`，并在文件首行使用 `//go:build integration`。真实服务地址只能通过运行时环境变量传入。
- 端到端测试仅在需要跨 HTTP 服务验证时放入 `test/e2e/`。测试数据、固定请求和响应样本放在所属包的 `testdata/`。

共享测试 helper 优先放在所属包的 `*_test.go` 文件中；只有被多个包使用时，才建立小而明确的 `internal/testutil` 包。不要让生产包为了测试而暴露内部实现。

## 执行命令

默认单元测试不需要启动任何外部服务：

```powershell
cd OpenClaw4j-Bankend-go
.\scripts\test-unit.ps1
```

也可直接执行：

```powershell
go test ./...
```

当前真实集成测试覆盖 pgvector。先在受控环境中设置测试专用 DSN，再显式执行：

```powershell
$env:OPENCLAW_TEST_PGVECTOR_DSN='postgres://...'
.\scripts\test-integration.ps1
```

集成测试会创建并删除自己的临时表；连接字符串必须指向专用测试数据库，不能使用生产数据库。不要将 DSN、密码、token 或其他凭据写入测试源码、文档或提交记录。

## 新增测试的判断

- 业务分支、变量转换、错误映射、图调度等确定性逻辑：增加单元测试。
- 数据库方言、Redis 过期语义、pgvector、真实 Sandbox 协议等必须依赖真实实现的行为：增加 `integration` 测试。
- 对外 API 的认证、路由、SSE 合约：优先使用 `httptest` 或 Hertz 请求上下文测试；只有完整部署链路才使用 E2E 测试。

提交前至少运行本次改动涉及包的测试；涉及多个包或公共契约时运行 `go test ./...`。集成测试不作为默认门禁，需由具备测试基础设施的本地环境或 CI 显式触发。
