# Rust Sandlock 脚本沙箱设计

## 背景

OpenClaw4j 工作流的 `Script` 节点目前在 Java 后端内执行脚本：

- `javascript` 通过 `SandboxManager.executeJavaScript` 使用 Nashorn 执行。
- `python` 入口 `executePython3Script` 当前没有真实实现。
- `java` 通过 `ASMCodeExecutor` / `InMemoryCodeExecutor` 在 JVM 内执行。

这种方式把不可信脚本执行和主应用进程放在同一个运行时边界内，缺少进程级隔离、资源限制和 Linux 安全机制约束。用户希望在 `OpenClaw4j-Sandbox` 中基于 Rust 实现独立代码沙箱服务，使用 Axum 暴露 HTTP API，并基于 `multikernel/sandlock` 在 Linux 环境中执行 Python 和 JavaScript 脚本；Java 后端通过 HTTP 调用该 Rust 服务。

`sandlock` 是 Linux 进程沙箱，使用 Landlock、seccomp-bpf 和 seccomp user notification 提供文件系统、网络、进程、内存等约束。它要求 Linux 运行环境，因此本设计通过 Docker 构建和运行 Rust sandbox 服务，Windows 本地 Java 后端仍通过 HTTP 访问容器中的服务。

## 目标

- 新增 `OpenClaw4j-Sandbox/` Rust Axum 服务，作为独立代码执行沙箱。
- 用 Rust sandbox 替代工作流 `Script` 节点中 `python` 和 `javascript` 的执行方式。
- Java 后端通过 HTTP 调用 Rust sandbox，保持 `ScriptExecuteProcessor` 和前端节点配置的现有契约基本不变。
- 使用 Dockerfile 构建 Linux 镜像并运行 Rust web 服务，避免在 Windows 主机直接运行 sandlock。
- 更新 `.gitignore` 和部署示例，确保源码、配置被跟踪，构建产物、临时目录不被跟踪。
- 保留当前 `java` 脚本执行路径，避免扩大范围到 JVM 内 Java 动态代码隔离。

## 非目标

- 本轮不改造前端 Script 节点 UI，不新增脚本语言。
- 本轮不替换 `java` 脚本执行。
- 本轮不复用或改造已有 `aio-sandbox` MCP 方案；它与工作流 Script 节点执行是不同边界。
- 本轮不允许脚本默认访问外网；如后续确有需求，再单独设计按域名或 HTTP ACL 的放行策略。
- 本轮不提供持久化工作目录；每次执行使用临时目录，执行结束后清理。

## 方案选择

推荐方案是在 monorepo 中新增独立 Rust 服务：

```text
OpenClaw4j-Bankend  --HTTP-->  OpenClaw4j-Sandbox(Axum)  --sandlock-->  python3/node
```

备选方案包括让 Java 直接调用 `sandlock run` CLI，或复用已有 `aio-sandbox`。前者会把进程管理、Docker/Linux 差异和 stdout/stderr 解析压到 Java 后端里；后者偏向 MCP 工具聚合，不符合本轮 Rust + Axum + sandlock 的目标。因此采用独立 Rust web 服务。

## Rust 服务设计

### 项目结构

```text
OpenClaw4j-Sandbox/
  Cargo.toml
  Cargo.lock
  Dockerfile
  README.md
  src/
    main.rs
    config.rs
    error.rs
    executor.rs
    language.rs
    routes.rs
    wrapper.rs
```

首轮实现可以根据实际复杂度合并少量模块，但职责边界保持清晰：

- `routes`：Axum 路由和请求响应模型。
- `config`：环境变量、超时、资源限制、监听地址。
- `executor`：创建临时目录、调用 sandlock、清理资源。
- `wrapper`：为 Python / JavaScript 生成稳定入口脚本。
- `language`：语言枚举、runtime 命令、扩展名。
- `error`：统一错误码和 HTTP 响应转换。

### HTTP API

健康检查：

```http
GET /health
```

响应：

```json
{
  "status": "UP"
}
```

脚本执行：

```http
POST /v1/execute
Content-Type: application/json
```

请求：

```json
{
  "request_id": "workflow-request-id",
  "language": "python",
  "code": "def main(params):\n    return {\"result\": params[\"name\"]}",
  "params": {
    "name": "OpenClaw4j"
  },
  "timeout_ms": 30000
}
```

成功响应：

```json
{
  "success": true,
  "data": {
    "result": "OpenClaw4j"
  },
  "stdout": "",
  "stderr": "",
  "exit_code": 0,
  "duration_ms": 18
}
```

失败响应：

```json
{
  "success": false,
  "message": "脚本执行超时",
  "code": "TIMEOUT",
  "stdout": "",
  "stderr": "",
  "exit_code": null,
  "duration_ms": 30000
}
```

错误码首轮包括：

- `INVALID_REQUEST`：请求参数缺失、语言不支持、代码为空。
- `SCRIPT_ERROR`：脚本运行返回非 0、抛异常或输出无法解析。
- `TIMEOUT`：超过执行超时。
- `SANDBOX_ERROR`：sandlock 启动、策略应用或 runtime 调用异常。
- `INTERNAL_ERROR`：未分类服务内部错误。

### 脚本入口与输出协议

Java 后端当前会在 Python / JavaScript 脚本末尾追加 `main()`，新方案应避免继续由 Java 拼接入口。Rust wrapper 统一负责调用：

- 优先调用 `main(params)`。
- 如果函数签名不接受参数，则兼容调用 `main()`。
- `main` 返回值必须能序列化为 JSON。
- wrapper 将返回值写入专用结果文件，例如 `result.json`。
- 用户脚本 stdout/stderr 单独捕获，不与结果 JSON 混合，避免 `print` 破坏协议。

Python wrapper 语义：

```text
1. 读取 params.json。
2. 执行用户代码文件。
3. 获取 main 函数。
4. 使用 inspect.signature 判断 main 是否声明入参；有入参时调用 main(params)，无入参时调用 main()。
5. 将返回值 JSON 序列化到 result.json。
```

JavaScript wrapper 语义：

```text
1. 读取 params.json。
2. 使用 Node.js vm 在隔离上下文中执行用户代码，避免 CommonJS 模块作用域隐藏普通 function main()。
3. 获取上下文中的 main，或兼容 module.exports.main。
4. 支持同步或 Promise 返回。
5. 将返回值 JSON 序列化到 result.json。
```

## sandlock 策略

每次执行创建独立临时目录：

```text
/tmp/openclaw4j-sandbox/<request_id或随机id>/
  user.py / user.js
  runner.py / runner.js
  params.json
  result.json
```

默认策略：

- 可读目录：`/usr`、`/lib`、`/lib64`、`/bin`、`/etc`，满足 Python / Node runtime 启动。
- 可写目录：当前请求临时目录。
- 默认禁止网络访问。
- 默认最大内存：`256M`。
- 默认最大进程数：`16`。
- 默认超时：`30000ms`。
- 清理环境变量，只保留必要 runtime 环境，例如 `PATH`、`LANG`、`PYTHONIOENCODING`。

配置通过环境变量覆盖：

```text
OPENCLAW_SANDBOX_BIND=0.0.0.0:9010
OPENCLAW_SANDBOX_WORK_DIR=/tmp/openclaw4j-sandbox
OPENCLAW_SANDBOX_DEFAULT_TIMEOUT_MS=30000
OPENCLAW_SANDBOX_MAX_TIMEOUT_MS=120000
OPENCLAW_SANDBOX_MEMORY_LIMIT=256M
OPENCLAW_SANDBOX_PROCESS_LIMIT=16
OPENCLAW_SANDBOX_STDOUT_LIMIT_BYTES=65536
OPENCLAW_SANDBOX_STDERR_LIMIT_BYTES=65536
```

## Java 后端集成

新增配置类，例如 `SandboxProperties`：

```yaml
sandbox:
  enabled: true
  base-url: ${OPENCLAW_SANDBOX_BASE_URL:http://127.0.0.1:9010}
  timeout-ms: ${OPENCLAW_SANDBOX_TIMEOUT_MS:30000}
```

`SandboxManager` 调整：

- `executePython3Script`：改为 HTTP 调用 Rust sandbox。
- `executeJavaScript`：改为 HTTP 调用 Rust sandbox。
- `executeJava`：保留现有 `ASMCodeExecutor` 路径。
- Java 不再给 Python / JavaScript 代码追加 `main()`，入口调用由 Rust wrapper 负责。
- Rust 服务不可用、HTTP 非 2xx、响应 JSON 无法解析时，返回符合现有 `ScriptExecutionResponse` 的失败 JSON，让 `ScriptExecuteProcessor` 走当前脚本失败分支。
- 日志记录 `requestId`、`language`、duration、错误码，不打印完整脚本内容和完整参数，避免泄漏敏感数据。

保持 `ScriptExecuteProcessor.ScriptExecutionResponse` 契约：

```json
{
  "success": true,
  "data": {}
}
```

因此 `ScriptExecuteProcessor` 需要删除 Python / JavaScript 分支中追加 `main()` 的行为。入口调用统一由 Rust wrapper 完成，避免 Java 与 Rust 双方重复拼接调用语句。

## Docker 与部署

`OpenClaw4j-Sandbox/Dockerfile` 使用多阶段构建：

1. Rust builder 阶段编译 release binary。
2. runtime 阶段安装 Python 3、Node.js、CA certificates 和必要动态库。
3. 复制 sandbox binary。
4. 暴露 `9010`。

`deploy/docker-compose.middleware.yml` 增加可选 profile：

```yaml
  openclaw4j-sandbox:
    build:
      context: ../OpenClaw4j-Sandbox
      dockerfile: Dockerfile
    image: ${OPENCLAW_SANDBOX_IMAGE:-openclaw4j-sandbox:local}
    container_name: openclaw4j-sandbox
    profiles: ["sandbox"]
    restart: unless-stopped
    ports:
      - "127.0.0.1:${OPENCLAW_SANDBOX_PORT:-9010}:9010"
    environment:
      OPENCLAW_SANDBOX_BIND: 0.0.0.0:9010
      OPENCLAW_SANDBOX_DEFAULT_TIMEOUT_MS: ${OPENCLAW_SANDBOX_DEFAULT_TIMEOUT_MS:-30000}
      OPENCLAW_SANDBOX_MAX_TIMEOUT_MS: ${OPENCLAW_SANDBOX_MAX_TIMEOUT_MS:-120000}
      OPENCLAW_SANDBOX_MEMORY_LIMIT: ${OPENCLAW_SANDBOX_MEMORY_LIMIT:-256M}
      OPENCLAW_SANDBOX_PROCESS_LIMIT: ${OPENCLAW_SANDBOX_PROCESS_LIMIT:-16}
      TZ: ${OPENCLAW_SANDBOX_TZ:-Asia/Shanghai}
```

后端如果也运行在 Docker 网络内，可以将：

```text
OPENCLAW_SANDBOX_BASE_URL=http://openclaw4j-sandbox:9010
```

## .gitignore

新增忽略规则：

```gitignore
OpenClaw4j-Sandbox/target/
OpenClaw4j-Sandbox/.env
OpenClaw4j-Sandbox/.env.*
!OpenClaw4j-Sandbox/.env.example
OpenClaw4j-Sandbox/tmp/
OpenClaw4j-Sandbox/output/
```

保留跟踪：

- Rust 源码。
- `Cargo.toml`。
- `Cargo.lock`。
- `Dockerfile`。
- 中文 README / 使用文档。

## 错误处理

- 请求语言不是 `python` 或 `javascript`：Rust 返回 `INVALID_REQUEST`。
- `code` 为空：Rust 返回 `INVALID_REQUEST`。
- 用户代码语法错误或运行时异常：Rust 返回 `SCRIPT_ERROR`，`stderr` 中保留截断后的错误信息。
- `main` 不存在：Rust 返回 `SCRIPT_ERROR`。
- 返回值不能 JSON 序列化：Rust 返回 `SCRIPT_ERROR`。
- 超时：Rust 停止进程组并返回 `TIMEOUT`。
- sandlock 策略无法应用：Rust 返回 `SANDBOX_ERROR`。
- Java 调用超时或连接失败：Java 包装为 `success=false` 的脚本失败响应，错误码可用 `SANDBOX_UNAVAILABLE`。

## 测试与验证

Rust：

- 单元测试：语言解析、配置默认值、wrapper 生成、响应模型序列化。
- 集成测试：在 Linux/Docker 环境运行 Python 和 JavaScript 样例。
- 手工验证：

```bash
curl http://127.0.0.1:9010/health
curl -X POST http://127.0.0.1:9010/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{"language":"python","code":"def main(params):\n    return {\"x\": params[\"x\"] + 1}","params":{"x":1}}'
```

Java：

- 为 `SandboxManager` 增加定向测试，使用 mock HTTP server 覆盖：
  - Python 成功响应。
  - JavaScript 成功响应。
  - Rust 返回脚本失败。
  - Rust 服务不可用或响应无法解析。
- 为 `ScriptExecuteProcessor` 增加或调整测试，确认 Python / JavaScript 不再依赖 Java 侧追加 `main()`。

部署：

- 构建 Rust 镜像。
- 使用 compose profile 启动 sandbox。
- Java 后端设置 `OPENCLAW_SANDBOX_BASE_URL` 后运行 Script 节点单测或手工调试。

质量门禁：

- 迭代阶段优先运行最小验证。
- 后端不主动运行完整 `spotless:check`、`checkstyle:check`、`spotbugs:check`，除非用户明确要求。

## 兼容性与影响范围

- 前端 Script 节点配置不变，已有 `python`、`javascript`、`java` 选项继续存在。
- Python 从未完成实现变为真实可执行。
- JavaScript 从 Nashorn 切换到 Node.js，语言特性更接近现代 JavaScript；少量依赖 Nashorn 特性的旧脚本可能需要调整。
- `java` 脚本继续使用当前 JVM 内执行方式，不纳入本轮沙箱隔离。
- 生产部署必须确保 Rust sandbox 只暴露给后端或可信内网，不直接暴露公网。

## 后续扩展

- 增加可配置网络 allowlist 或 HTTP ACL。
- 增加脚本依赖白名单和预装包策略。
- 增加执行指标，例如耗时、超时次数、语言分布、错误码分布。
- 将 Java 脚本执行也迁移到进程级隔离，但需单独设计 JDK/runtime 镜像、编译缓存和安全边界。
