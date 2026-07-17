# OpenClaw4j Sandbox Go + Hertz 迁移设计

## 背景

`OpenClaw4j-Sandbox` 当前是工作流 `Script` 节点的独立脚本执行服务，使用 Rust + Axum 暴露 HTTP API，并在 Linux 容器中通过 `sandlock run` 执行 Python 和 JavaScript 脚本。用户希望将该服务迁移为 Go 项目，HTTP 框架使用 Hertz。

本次迁移的目标是替换 sandbox 服务自身的实现语言，不改变后端与 sandbox 之间的接口契约，也不改变容器中的脚本运行语义。`D:\IDEA_project\sandlock\go` 已提供 sandlock Go SDK，它通过 cgo 链接 `libsandlock_ffi`，可以直接以 Go API 运行受限进程。

## 目标

- 使用 Go + Hertz 重写 `OpenClaw4j-Sandbox` 服务。
- 保持 `GET /health` 和 `POST /v1/execute` API 兼容。
- 保持请求、响应 JSON 字段兼容。
- 保持现有 `OPENCLAW_SANDBOX_*` 环境变量语义兼容。
- Linux 容器内优先使用 sandlock Go SDK，而不是继续拼接 `sandlock run` CLI 参数。
- 非 Linux 环境提供直接执行 fallback，便于 Windows 本地单元测试和基础调试。
- Docker 镜像仍内置 Python、Node.js、Python 依赖和 Node 依赖。

## 非目标

- 不扩展新的脚本语言。
- 不改变后端 `SandboxManager` 调用契约。
- 不引入网络访问白名单、HTTP ACL、COW 分支、checkpoint/restore 等 sandlock 高级能力。
- 不改变 Python/JavaScript 用户脚本的 `main(params)` 约定。
- 不在本轮清理 backend/frontend 的既有改动。

## 方案比较

### 方案 A：Go + Hertz + sandlock Go SDK

服务使用 Hertz 处理 HTTP，请求进入执行器后在 Linux 下构造 `sandlock.Sandbox`，调用 `Run(ctx, cmd...)` 执行 runner。

优点：

- 执行层是原生 Go API，避免手工拼 CLI 参数。
- 可以拿到结构化的 `Result`，stdout、stderr、exit code 更直接。
- 后续如果需要动态策略、资源限制或进程生命周期控制，可以沿用 SDK 能力。

缺点：

- Docker 构建需要同时构建 Go 服务和 Rust FFI 动态库。
- Linux 构建依赖 cgo、`sandlock.h`、`libsandlock_ffi.so` 和 `pkg-config`。

### 方案 B：Go + Hertz + sandlock CLI

服务迁移到 Go，但执行层继续通过 `os/exec` 调用 `sandlock run`。

优点：

- Docker 迁移简单，沿用当前 CLI 模式。
- 不需要在 Go 服务内引入 cgo。

缺点：

- 只是把 Rust HTTP 壳替换成 Go HTTP 壳，执行层仍然依赖字符串参数组织。
- 未来使用 sandlock 结构化能力时仍需二次迁移。

### 方案 C：先 CLI 后 SDK

第一阶段使用 Go + CLI 保持低风险，第二阶段再切换 SDK。

优点：

- 初始迁移风险较低。

缺点：

- 执行层会做两遍，测试和排障成本更高。

## 推荐方案

采用方案 A：Go + Hertz + sandlock Go SDK。

理由是 sandbox 服务本身边界很窄，迁移时最有价值的是把执行层也变成 Go 原生结构，而不是只替换 HTTP 框架。SDK 已覆盖本服务需要的静态文件系统权限、资源限制、环境变量清理、超时和 stdout/stderr 捕获能力，适合一次性迁移。

## 模块设计

Go 项目保留在 `OpenClaw4j-Sandbox/` 下，建议结构如下：

```text
OpenClaw4j-Sandbox/
  cmd/openclaw4j-sandbox/main.go
  internal/config/
  internal/executor/
  internal/model/
  internal/wrapper/
  deps/python/requirements.txt
  deps/node/package.json
  deps/node/pnpm-lock.yaml
  Dockerfile
  README.md
  go.mod
  go.sum
```

### config

负责读取环境变量并给出默认值：

- `OPENCLAW_SANDBOX_BIND`
- `OPENCLAW_SANDBOX_WORK_DIR`
- `OPENCLAW_SANDBOX_DEFAULT_TIMEOUT_MS`
- `OPENCLAW_SANDBOX_MAX_TIMEOUT_MS`
- `OPENCLAW_SANDBOX_MEMORY_LIMIT`
- `OPENCLAW_SANDBOX_PROCESS_LIMIT`
- `OPENCLAW_SANDBOX_STDOUT_LIMIT_BYTES`
- `OPENCLAW_SANDBOX_STDERR_LIMIT_BYTES`
- `OPENCLAW_SANDBOX_DEPS_DIR`
- `OPENCLAW_SANDBOX_PYTHON_RUNTIME`
- `OPENCLAW_SANDBOX_NODE_PATH`

`ClampTimeout` 保持当前语义：请求未传 timeout 时使用默认值，请求 timeout 超过最大值时截断为最大值。

### model

定义 HTTP JSON 契约：

- `ExecuteRequest`
- `ExecuteResponse`

响应字段保持现有 Rust 服务兼容。`duration_ms` 在 Go 中使用整数毫秒，JSON 字段名保持不变。

### wrapper

负责语言识别、请求校验和 runner 生成：

- 支持 `python`、`python3`、`javascript`、`js`。
- 空脚本返回 `INVALID_REQUEST`。
- Python runner 继续使用 `runpy` 加载用户脚本，并要求存在可调用的 `main`。
- JavaScript runner 继续使用 Node `vm` 模块隔离用户代码，并支持 `main`、`module.exports.main` 和 `exports.main`。
- JavaScript 运行参数保持 `--jitless`，避免 V8 JIT 与 seccomp 限制冲突。

### executor

执行流程保持现有顺序：

1. 记录请求基础信息。
2. 校验请求并解析语言。
3. 创建工作根目录。
4. 按 `request_id` 生成安全的临时目录前缀；没有 `request_id` 时生成 UUID。
5. 写入用户脚本、`params.json` 和语言 runner。
6. 根据配置和请求计算 timeout。
7. Linux 下通过 sandlock Go SDK 执行 runner。
8. 非 Linux 下直接执行 runtime，供本地测试使用。
9. 截断 stdout/stderr。
10. 非 0 退出码返回 `SCRIPT_ERROR`。
11. 成功时读取 `result.json` 并解析 JSON。

错误码保持：

- `INVALID_REQUEST`
- `TIMEOUT`
- `SCRIPT_ERROR`
- `SANDBOX_ERROR`

## sandlock SDK 集成

Linux 构建文件使用 build tag 引入 `github.com/multikernel/sandlock/go`。执行时构造：

```go
sandbox := &sandlock.Sandbox{
    FSReadable:   []string{"/usr", "/lib", "/lib64", "/bin", "/etc", config.DepsDir},
    FSWritable:   []string{workDir},
    MaxMemory:    config.MemoryLimit,
    MaxProcesses: config.ProcessLimit,
    CleanEnv:     true,
    Env: map[string]string{
        "PATH":             computedPath,
        "LANG":             "C.UTF-8",
        "PYTHONIOENCODING": "utf-8",
        "NODE_PATH":        config.NodePath,
    },
    Cwd: workDir,
}
```

命令参数保持与当前 Rust 版本一致：

- Python：`config.PythonRuntime runner.py`
- JavaScript：`node --jitless runner.js`

SDK 的 `Run(ctx, cmd...)` 使用 `context.WithTimeout` 控制超时。若 SDK 返回 timeout 或上下文超时，转换为 `TIMEOUT`；其他 SDK 错误转换为 `SANDBOX_ERROR`。

## Docker 设计

Dockerfile 需要同时产出 Go 服务和 sandlock FFI：

1. `go-build` stage：编译 `openclaw4j-sandbox`。
2. `sandlock-ffi-build` stage：从 sandlock 源码构建 `libsandlock_ffi.so`，并提供 `sandlock.h`、`sandlock.pc`。
3. `python-deps` stage：构建 Python venv。
4. `node-deps` stage：安装 Node 依赖。
5. `runtime` stage：安装 `ca-certificates`、`nodejs`、`python3`、`python3-venv`，复制 Go 二进制、FFI 动态库和依赖目录。

为了让 Go 二进制在运行时找到 `libsandlock_ffi.so`，优先通过 `pkg-config` 的 rpath 或设置系统库路径解决；Dockerfile 中需要显式验证最终镜像启动时不会缺少动态库。

## 日志与可观测性

Go 服务需要保留当前日志意图：

- 请求接收时记录 request id、language、code bytes、params JSON 类型。
- 执行开始时记录 request id、language、timeout、memory limit、process limit 和 work dir。
- 成功时记录 duration、exit code、stdout/stderr 字节数。
- 失败时记录错误码、duration、exit code、stdout/stderr 字节数。

脚本 stdout/stderr 内容不写入服务日志，避免敏感信息或大输出污染容器日志。

## 测试与验证

最小验证：

- `go test ./...`
- wrapper 单测覆盖语言解析、空代码校验、Python runner、JavaScript runner。
- config 单测覆盖 timeout clamp。
- executor helper 单测覆盖 request id 清洗、stdout/stderr 截断、环境变量构造。

容器验证：

- 构建镜像。
- 调用 `GET /health` 返回 `{"status":"UP"}`。
- 调用 Python 示例，验证 `main(params)` 返回 JSON。
- 调用 JavaScript 示例，验证 `module.exports.main` 返回 JSON。
- 调用超时脚本，验证返回 `TIMEOUT`。
- 调用抛错脚本，验证返回 `SCRIPT_ERROR` 且保留 stderr。

## 兼容性影响

后端调用方不需要修改 API 契约。部署侧需要注意 Dockerfile 的 sandlock 依赖从 CLI 二进制转为 FFI 动态库；如果仍希望保留 `sandlock` CLI 作为诊断工具，可以在 runtime 镜像中额外复制 CLI，但这不是服务运行必需条件。

Windows 本地运行不会使用 sandlock SDK，只用于基础逻辑和单元测试。安全边界必须以 Linux 容器验证为准。

## 风险与缓解

- cgo/动态库链接失败：在 Dockerfile 中固定复制 `libsandlock_ffi.so`、`sandlock.h` 和 `sandlock.pc`，并在构建阶段运行最小 Go 测试或二进制启动检查。
- SDK timeout 语义与 CLI 略有差异：用超时脚本做容器级回归验证，并在 executor 中统一转换为 `TIMEOUT`。
- JavaScript `vm` runner 不是完整安全边界：继续依赖 sandlock 作为进程级边界，runner 只负责用户脚本入口适配。
- 非 Linux fallback 被误认为安全执行：通过文件名、注释和 README 明确说明 fallback 仅用于本地测试。

## 实施顺序

1. 初始化 Go module，引入 Hertz 和 sandlock Go SDK。
2. 搭建 model、config、wrapper，并补充单元测试。
3. 实现 executor 的通用流程和非 Linux fallback。
4. 实现 Linux sandlock SDK executor。
5. 替换 main 入口为 Hertz 服务。
6. 更新 Dockerfile 构建链路。
7. 更新 README。
8. 运行 Go 单测和可行的容器验证。
