# OpenClaw4j Sandbox

`OpenClaw4j-Sandbox` 是工作流 `Script` 节点的独立代码执行服务。服务使用 Go + Hertz 暴露 HTTP API，在 Linux 容器中通过 sandlock Go SDK 执行 Python 和 JavaScript 脚本。

## 构建镜像

```powershell
docker build -t openclaw4j-sandbox:local OpenClaw4j-Sandbox
```

镜像构建使用 Docker BuildKit cache mount 缓存 Cargo、Go、pip、pnpm 和 sandlock 编译目录。后续重复构建时会复用本机 Docker builder 的缓存，不需要把这些构建产物挂成运行时 volume。

## 启动服务

```powershell
docker run --rm -p 127.0.0.1:9010:9010 openclaw4j-sandbox:local
```

或使用中间件 compose profile：

```powershell
docker compose -f deploy/docker-compose.middleware.yml --profile sandbox up -d openclaw4j-sandbox
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

## Agent 兼容接口

服务额外提供一组轻量 AIO 风格接口，供 Agent 读写 workspace 文件、运行命令和通过 MCP 调用工具。

默认 workspace：

```text
OPENCLAW_SANDBOX_WORKSPACE_DIR=/tmp/openclaw4j-workspace
```

可调整的 Agent 兼容参数：

```text
OPENCLAW_SANDBOX_BASH_DEFAULT_TIMEOUT_MS=30000
OPENCLAW_SANDBOX_BASH_HARD_TIMEOUT_MS=120000
OPENCLAW_SANDBOX_BASH_OUTPUT_LIMIT_BYTES=65536
OPENCLAW_SANDBOX_FILE_READ_LIMIT_BYTES=1048576
```

写入和读取文件：

```powershell
$write = @{
  file = 'notes/todo.txt'
  content = 'hello sandbox'
} | ConvertTo-Json
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/v1/file/write' -Method Post -ContentType 'application/json' -Body $write

$read = @{ file = 'notes/todo.txt' } | ConvertTo-Json
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/v1/file/read' -Method Post -ContentType 'application/json' -Body $read
```

执行 Bash：

```powershell
$body = @{
  command = 'pwd && ls -la'
  timeout = 30
} | ConvertTo-Json
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/v1/bash/exec' -Method Post -ContentType 'application/json' -Body $body
```

查询 MCP 工具列表：

```powershell
$body = @{
  jsonrpc = '2.0'
  id = 1
  method = 'tools/list'
} | ConvertTo-Json
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/mcp' -Method Post -ContentType 'application/json' -Body $body
```

当前内置 MCP server 名称为 `sandbox`，工具包括 `file_read`、`file_write`、`file_list`、`file_replace`、`file_search` 和 `sandbox_execute_bash`。

MCP SSE 入口：

```json
{
  "mcpServers": {
    "openclaw4j-sandbox": {
      "type": "sse",
      "url": "http://127.0.0.1:9010/mcp/sse"
    }
  }
}
```

`GET /mcp/sse` 会建立 SSE 连接，并通过 `endpoint` 事件返回本次会话的消息投递地址。客户端随后向该 endpoint 发送 JSON-RPC 请求，服务端会把响应通过 SSE `message` 事件返回。当前先实现传统 MCP SSE 传输，暂不实现 Streamable HTTP。

首轮兼容重点是 Agent 可用，不包含 upload/download、file watch、浏览器、Jupyter、Code Server、外部 MCP Hub 聚合、Streamable HTTP MCP 以及 `sudo` 提权语义。所有 file/bash 路径都会限制在 workspace 内。

## 内置依赖

沙箱镜像默认内置一组白名单依赖：

- Python: `numpy`, `pandas`, `scipy`
- JavaScript: `lodash`, `dayjs`, `decimal.js`, `uuid`

Python 依赖安装在 `/opt/openclaw4j-sandbox/deps/python` venv 中，JavaScript 依赖安装在 `/opt/openclaw4j-sandbox/deps/node/node_modules`。运行脚本时 sandlock SDK 只给 `/opt/openclaw4j-sandbox/deps` 只读访问权限，并通过下面环境变量暴露运行时：

```text
OPENCLAW_SANDBOX_DEPS_DIR=/opt/openclaw4j-sandbox/deps
OPENCLAW_SANDBOX_PYTHON_RUNTIME=/opt/openclaw4j-sandbox/deps/python/bin/python
OPENCLAW_SANDBOX_NODE_PATH=/opt/openclaw4j-sandbox/deps/node/node_modules
```

新增依赖时优先修改：

- `deps/python/requirements.txt`
- `deps/node/package.json`，并重新生成 `deps/node/pnpm-lock.yaml`

不建议默认把依赖目录挂成 volume。空 volume 会遮蔽镜像内置依赖，导致线上和本地行为不一致。

## 本地测试

```powershell
cd OpenClaw4j-Sandbox
go test ./...
```

Windows 和其他非 Linux 环境会使用直接执行 fallback，仅用于本地单元测试和基础调试，不提供安全沙箱边界。安全行为以 Linux 容器中的 sandlock SDK 验证为准。

## 日志

服务默认输出 `info` 级别日志到 stdout，Docker 下可以直接查看：

```powershell
docker logs -f openclaw4j-sandbox
```

执行日志会记录 `request_id`、脚本语言、超时时间、资源限制、工作目录、执行耗时、退出码以及 stdout/stderr 字节数。脚本输出内容不会默认写入日志，避免大输出或敏感数据污染容器日志。
