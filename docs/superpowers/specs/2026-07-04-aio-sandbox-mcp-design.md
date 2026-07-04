# AIO Sandbox MCP 接入设计

## 背景

OpenClaw4j 已有 MCP Server 管理能力，前端通过 MCP 页面创建、更新、调试 MCP Server，后端由 `MCPManager` 使用 `HttpClientSseClientTransport` 连接远程 MCP 服务。现有实现只支持 SSE 安装类型，并在 `processInstallConfig` 中要求 URL path 以 `/sse` 结尾。

AIO Sandbox 官方 MCP Hub 入口是 `http://<host>:8080/mcp`，文档描述为可流式 HTTP 协议，并聚合浏览器、文件、终端、Markitdown 等工具。直接将 `/mcp` 填入现有 MCP 页面会被 URL 校验拒绝，也不能复用 SSE transport。

## 目标

- 允许用户在现有 MCP 页面注册 AIO Sandbox 的 `/mcp` endpoint。
- 保持现有 SSE MCP Server 行为不变。
- AIO MCP 注册后能在现有列表、详情、工具拉取、调试工具和 Agent/Workflow MCP 调用链路中使用。
- 支持在 deploy config 中配置请求头，便于后续接入 AIO Sandbox 鉴权。

## 非目标

- 不在本轮实现 AIO Sandbox REST SDK 或 REST 工具适配器。
- 将 AIO Sandbox 作为可选 profile 集成进 `deploy/docker-compose.middleware.yml`，默认不随基础中间件自动启动。
- 不改造现有 MCP 数据库表结构，优先复用 `install_type`、`deploy_config`、`host` 字段。
- 不改变已有 SSE 配置格式。

## 接入方案

### 安装类型

新增 MCP 安装类型：

```text
STREAMABLE_HTTP
```

前端显示为 `Streamable HTTP / AIO Sandbox`，用于注册 AIO Sandbox `/mcp`。后端枚举接受该类型，并在配置解析、工具列表、工具调用阶段按类型分流。

### 配置格式

沿用现有 `mcpServers` 单服务配置格式：

```json
{
  "mcpServers": {
    "aio-sandbox": {
      "url": "http://localhost:8080/mcp",
      "headers": {}
    }
  }
}
```

如果需要鉴权，可填：

```json
{
  "mcpServers": {
    "aio-sandbox": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer <token>"
      }
    }
  }
}
```

### 后端配置解析

`MCPManager.processInstallConfig` 保持单个 `mcpServers` 的限制。解析 URL 时：

- `SSE`：继续要求 path 以 `/sse` 结尾。
- `STREAMABLE_HTTP`：允许 path 非空，典型值为 `/mcp`，不要求 `/sse`。
- 两种类型都继续写入 `remote_address`、`remote_endpoint`、`remote_header` 和原始 `install_config`。

### 后端调用分流

`MCPManager` 增加内部客户端分流：

- `SSE`：继续使用当前 `McpSyncClient` 和 `HttpClientSseClientTransport`。
- `STREAMABLE_HTTP`：新增基于 JDK `HttpClient` 的 JSON-RPC 客户端，向 `remote_address + remote_endpoint` 发送 HTTP POST。

客户端需要支持：

- `tools/list`：转换为现有 `List<McpTool>`。
- `tools/call`：转换为现有 `McpServerCallToolResponse`。
- 请求头：从 `remote_header` 带入每次 HTTP 请求。
- 超时：沿用现有 60 秒请求超时。

为兼容不同 MCP HTTP 返回形态，解析时按 JSON-RPC 结构优先读取：

- 成功：`result.tools` 或 `result.content`。
- 失败：`error.message` 写入 `TextContent`，并标记 `is_error=true`。

### 前端页面

MCP 创建页的安装类型增加 `Streamable HTTP / AIO Sandbox`。选择该类型时：

- deploy config 示例使用 `/mcp`。
- 原有 `SSE` 选项和配置方式保持不变。

本轮不新增独立 AIO 专属页面，避免扩散范围。

## 错误处理

- URL 缺失、`mcpServers` 数量不为 1：继续返回现有 MCP 配置解析错误。
- `SSE` 使用非 `/sse` path：继续返回现有 MCP URL 解析错误。
- `STREAMABLE_HTTP` 的 HTTP 状态码非 2xx：工具拉取返回空列表，工具调用返回 `TOOL_EXECUTION_ERROR` 或带错误文本的 MCP response。
- JSON-RPC 返回 `error`：工具调用返回 `is_error=true`，并将错误消息放入 `TextContent`。
- 响应 JSON 无法解析：记录日志，保持现有调用失败语义。

## 测试与验证

后端：

- 为 `processInstallConfig` 增加 `/mcp` + `STREAMABLE_HTTP` 解析测试。
- 确认 `SSE` `/sse` 配置仍通过，`SSE` `/mcp` 仍被拒绝。
- 为 streamable HTTP 客户端增加 mock HTTP 测试，覆盖 `tools/list`、`tools/call` 和 JSON-RPC error。
- 运行后端定向测试，再按质量门禁运行 Spotless、Checkstyle、SpotBugs。

前端：

- 确认 MCP 创建页能选择新安装类型，并提交 `install_type=STREAMABLE_HTTP`。
- 运行能覆盖该 package 的构建或 `npm run build:app`。

端到端手工验证：

- 启动 AIO Sandbox。
- 在 MCP 页面注册 `http://localhost:8080/mcp`。
- 获取工具列表，至少看到浏览器、文件或终端类工具。
- 使用调试工具调用一个低风险工具，例如列目录或读取沙盒内安全文件。

## 部署提示

AIO Sandbox 的 8080 端口应保持在后端可访问的私网范围内。生产部署不要直接向公网暴露 8080，应通过 Nginx、Ingress 或负载均衡处理 TLS 与访问控制。
