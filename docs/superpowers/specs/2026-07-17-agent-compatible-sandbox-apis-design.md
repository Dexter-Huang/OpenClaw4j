# Rust Sandbox Agent 可用兼容接口设计

## 背景

`OpenClaw4j-Sandbox` 当前是一个独立 Rust + Axum 服务，主要为工作流 `Script`
节点提供 `/v1/execute` 脚本执行接口。服务每次请求创建临时目录，通过
`sandlock run` 执行 Python 或 JavaScript，并在执行结束后返回 JSON 结果。

本地 `D:\IDEA_project\sandbox` 提供了 AIO Sandbox 的文档、SDK 和 OpenAPI
定义。AIO Sandbox 的能力范围很大，包含文件、Bash、MCP、浏览器、Jupyter、
Code Server、watcher、上传下载等多个子系统。本轮目标不是完整复刻 AIO
Sandbox，而是为 OpenClaw4j Agent 场景补齐最常用、最稳定的文件、命令执行和
MCP 工具接口。

## 目标

- 在现有 Rust sandbox 中新增 AIO 风格的 `/v1/file/*` 常用文件 API。
- 新增 `/v1/bash/*` 管道式命令执行 API，支持短命令、长命令轮询和 stdin 写入。
- 新增内置 MCP 入口，让 Agent 可以通过 MCP 工具调用文件和 Bash 能力。
- 保留现有 `/health` 和 `/v1/execute` 行为不变。
- 默认把文件和 Bash 工作范围限制在配置的 workspace 根目录内，避免任意访问宿主或容器敏感路径。
- 使用中文文档和中文注释解释关键边界、兼容取舍和安全约束。

## 非目标

- 不实现完整 AIO Sandbox。
- 不实现浏览器、Jupyter、Code Server、Node.js 会话、display、proxy、skills 等 AIO 子系统。
- 不实现 `/v1/file/upload`、`/v1/file/download`、file watch、SSE 事件流。
- 不实现 `sudo` 参数和受保护系统路径写入。
- 不实现外部 MCP Hub 聚合，只提供当前 Rust sandbox 内置工具。
- 不要求兼容 AIO SDK 的所有字段和边缘行为；首轮以 Agent 调用可用、响应结构相近为准。

## 方案选择

推荐采用“轻量兼容层”方案：在现有 Axum 应用中新增 `file`、`bash`、`mcp`
三个模块，每个模块封装自己的请求模型、响应模型和业务服务。`main.rs` 只负责组装路由和共享状态。

备选方案包括完整复刻 AIO REST schema，或只做 MCP、不做 REST。完整复刻会显著扩大首轮工作量；
只做 MCP 又会削弱 OpenClaw4j 后端或其他 HTTP 客户端直接调用 file/bash 的能力。因此本轮选择先补
Agent 最需要的 REST + MCP 子集。

## 配置

新增 workspace 配置：

```text
OPENCLAW_SANDBOX_WORKSPACE_DIR=/tmp/openclaw4j-workspace
OPENCLAW_SANDBOX_BASH_DEFAULT_TIMEOUT_MS=30000
OPENCLAW_SANDBOX_BASH_HARD_TIMEOUT_MS=120000
OPENCLAW_SANDBOX_BASH_OUTPUT_LIMIT_BYTES=65536
OPENCLAW_SANDBOX_FILE_READ_LIMIT_BYTES=1048576
```

`OPENCLAW_SANDBOX_WORK_DIR` 继续用于 `/v1/execute` 的临时执行目录。
`OPENCLAW_SANDBOX_WORKSPACE_DIR` 用于 file/bash/MCP 的共享工作区。服务启动时创建该目录。

路径规则：

- 客户端可以传绝对路径或相对路径。
- 相对路径按 workspace 根目录解析。
- 绝对路径必须位于 workspace 根目录内。
- 路径归一化后如果逃逸 workspace，返回业务失败响应，错误类型为 `invalid_path`。
- 首轮不支持 `sudo=true`，收到该参数时按普通权限执行，并在后续文档中说明不保证兼容 sudo。

## File API

首轮实现下面端点：

```text
POST /v1/file/read
POST /v1/file/write
POST /v1/file/replace
POST /v1/file/search
POST /v1/file/find
POST /v1/file/grep
POST /v1/file/glob
POST /v1/file/list
POST /v1/file/str_replace_editor
```

统一响应形态尽量贴近 AIO：

```json
{
  "success": true,
  "message": "File read successfully",
  "data": {}
}
```

预期内文件系统错误返回 `HTTP 200` 和 `success=false`：

```json
{
  "success": false,
  "message": "Failed to read file",
  "data": {
    "path": "/tmp/openclaw4j-workspace/missing.txt",
    "operation": "read",
    "message": "Failed to read file",
    "error_type": "not_found",
    "retryable": false,
    "errno_name": "ENOENT"
  }
}
```

### read

请求字段：

- `file`: 文件路径。
- `start_line`: 可选，0-based 起始行。
- `end_line`: 可选，0-based 结束行，不包含。
- `sudo`: 可选，首轮忽略。

响应数据：

- `file`: 归一化后的路径。
- `content`: 文本内容。
- `line_count`: 返回内容行数。

读取大小超过 `OPENCLAW_SANDBOX_FILE_READ_LIMIT_BYTES` 时返回 `success=false`，
错误类型为 `too_large`，提示调用方用行范围分块读取。

### write

请求字段：

- `file`: 文件路径。
- `content`: 写入内容。
- `encoding`: `utf-8`、`base64` 或 `raw`，默认 `utf-8`。
- `append`: 是否追加，默认 `false`。
- `leading_newline`: 写入前是否补一个换行，默认 `false`。
- `trailing_newline`: 写入后是否补一个换行，默认 `false`。

响应数据：

- `file`: 归一化后的路径。
- `bytes_written`: 写入字节数。
- `created`: 写入前文件是否不存在。

### replace / search / grep / find / glob / list

- `replace` 在单个文件内替换文本，默认要求 `old_str` 存在，支持 `replace_all` 简化参数。
- `search` 在单个文件中使用 Rust `regex` 搜索，返回匹配文本、行号和列号。
- `grep` 在目录树中搜索文本，支持 `include`、`exclude`、`case_insensitive`、`max_results`。
- `find` 使用简单 glob 在目录中查找文件。
- `glob` 支持 `**` 递归匹配、隐藏文件控制、只返回文件或同时返回目录。
- `list` 列出目录，支持 `recursive`、`show_hidden`、`include_size`。

首轮可以用 `walkdir`、`globset`、`regex` 实现，不依赖 shell 命令拼接。

### str_replace_editor

实现 Agent 常用的编辑器工具语义：

- `view`: 查看文件或目录。
- `create`: 创建新文件，目标已存在时失败。
- `str_replace`: 替换文本，支持 `replace_mode=ALL|FIRST|LAST`；未指定时要求唯一匹配。
- `insert`: 按行号插入文本。

`undo_edit` 首轮返回 `success=false`，错误类型为 `unsupported_operation`。原因是当前服务没有持久编辑历史；
后续如确有需要，可在 workspace 内维护 per-file edit log。

## Bash API

首轮实现下面端点：

```text
POST /v1/bash/exec
POST /v1/bash/output
POST /v1/bash/write
POST /v1/bash/kill
GET /v1/bash/sessions
POST /v1/bash/sessions/create
POST /v1/bash/sessions/{session_id}/close
```

Bash 模型与 AIO 文档保持一致：每次 `exec` 创建一个新进程，同一 `session_id` 只保留 API
级状态，例如默认工作目录和历史命令状态。命令内部的 `cd`、`export` 不影响后续 `exec`。

### 状态模型

命令状态：

- `running`: 进程仍在运行。
- `completed`: 进程正常退出，`exit_code` 可用；非 0 仍是 completed。
- `timed_out`: 达到 `hard_timeout` 后被强制终止。
- `killed`: 被 `/v1/bash/kill` 或 session close 终止。

session 状态：

- `ready`: 可继续执行命令。
- `closed`: 已关闭，不可复用。

### exec

请求字段：

- `command`: Shell 命令。
- `session_id`: 可选；为空时自动创建。
- `exec_dir`: 可选工作目录，必须位于 workspace 内。
- `env`: 可选，仅本次命令生效。
- `async_mode`: 为 `true` 时立即返回 `running`。
- `timeout`: 同步等待软超时，单位秒。
- `hard_timeout`: 强制终止超时，单位秒。
- `max_output_length`: 同步响应 stdout/stderr 截断长度。

响应数据：

- `session_id`
- `command_id`
- `command`
- `status`
- `stdout`
- `stderr`
- `exit_code`
- `offset`
- `stderr_offset`

Linux 容器内使用 `/bin/bash -lc <command>`。Windows 本地调试时可以降级为 `powershell`
或 `cmd`，但 Docker/Linux 是生产目标；跨平台差异需要在 README 中说明。

### output

按 `offset` 和 `stderr_offset` 增量读取输出。`wait=true` 时长轮询，直到有新输出、命令结束或
`wait_timeout` 到期。输出缓存在内存中，首轮不持久化到磁盘。

### write / kill / sessions

- `write` 将 `input` 写入指定命令的 stdin；没有指定 `command_id` 时写入当前运行命令。
- `kill` 对 session 当前运行命令发送终止信号；首轮接受 `SIGTERM`、`SIGKILL`、`SIGINT`。
- `sessions` 返回当前内存中的 active sessions。
- `sessions/create` 创建 session 并设置默认 `exec_dir`。
- `sessions/{id}/close` 终止该 session 中运行的命令并标记 closed。

## MCP API

首轮同时提供 AIO 风格 REST 包装和标准 JSON-RPC `/mcp` 入口。

### REST 包装

```text
GET /v1/mcp/servers
GET /v1/mcp/{server_name}/tools
POST /v1/mcp/{server_name}/tools/{tool_name}
```

内置 server 名称：

```text
sandbox
```

工具列表：

- `file_read`
- `file_write`
- `file_list`
- `file_replace`
- `file_search`
- `sandbox_execute_bash`

工具 schema 使用 JSON Schema object 表达，字段与对应 REST API 的核心字段一致。

### `/mcp` JSON-RPC

支持方法：

- `initialize`
- `tools/list`
- `tools/call`

`initialize` 返回基础 protocol 信息和 capabilities。`tools/list` 返回内置工具。`tools/call`
按工具名转发到 file/bash service，并把结果包装为 MCP content：

```json
{
  "content": [
    {
      "type": "text",
      "text": "{...}"
    }
  ],
  "isError": false
}
```

如果工具执行失败，`isError=true`，`text` 中包含结构化错误 JSON 字符串。首轮不实现 SSE
或 streamable HTTP 的长连接行为；`POST /mcp` 以普通 JSON-RPC request/response 工作。

## 模块边界

建议新增或调整文件：

```text
OpenClaw4j-Sandbox/src/main.rs
OpenClaw4j-Sandbox/src/config.rs
OpenClaw4j-Sandbox/src/model.rs
OpenClaw4j-Sandbox/src/file_api.rs
OpenClaw4j-Sandbox/src/bash_api.rs
OpenClaw4j-Sandbox/src/mcp_api.rs
OpenClaw4j-Sandbox/src/path_guard.rs
```

职责：

- `file_api`: 文件请求模型、文件响应模型、文件操作服务和路由 handler。
- `bash_api`: bash session/command 状态、进程管理、输出缓存和路由 handler。
- `mcp_api`: MCP tool registry、REST 包装、JSON-RPC 转发。
- `path_guard`: workspace 路径归一化和逃逸检查，供 file/bash 复用。
- `model`: 保留 `/v1/execute` 模型；如公共响应模型明显复用，再放入共享类型。

## 错误处理

- JSON 请求体缺失必填字段：返回 `HTTP 422` 或 Axum 默认反序列化错误。
- 路径逃逸 workspace：返回 `HTTP 200`、`success=false`、`error_type=invalid_path`。
- 文件不存在：返回 `success=false`、`error_type=not_found`。
- 权限不足：返回 `success=false`、`error_type=permission_denied`。
- Bash 命令退出非 0：返回 `success=true`，`data.status=completed`，`exit_code` 为实际退出码。
- Bash hard timeout：返回 `success=true`，`data.status=timed_out`。
- MCP 未知工具：JSON-RPC 返回 `error.code=-32601` 或 REST 返回 `success=false`。

## 测试策略

本轮采用测试优先方式实现：

1. `path_guard` 单元测试先覆盖相对路径、workspace 内绝对路径和 `..` 逃逸。
2. `file_api` 单元/集成测试覆盖 read、write、replace、list、grep 和 `str_replace_editor` 的常用路径。
3. `bash_api` 测试覆盖短命令、stderr、非 0 exit code、async output、stdin write 和 kill。
4. `mcp_api` 测试覆盖 `tools/list`、`tools/call file_read`、`tools/call sandbox_execute_bash` 和未知工具。
5. 运行 Rust 定向测试：`cargo test`。如果后续改 Docker/runtime 行为，再补 Docker 手工验证。

不主动运行后端 Java 完整质量门禁，除非用户明确要求。Rust 改动完成前的最小验证以
`cargo test` 为准。

## 兼容性与影响范围

- `/v1/execute` 保持兼容，现有 OpenClaw4j 后端脚本执行链路不受影响。
- 新增 file/bash/MCP 能力主要服务 Agent 工具调用和后续 OpenClaw4j 后端集成。
- 由于首轮路径限制在 workspace 内，AIO 示例中的 `/home/gem/workspace` 需要通过配置或文档映射到
  `OPENCLAW_SANDBOX_WORKSPACE_DIR`。
- 首轮 MCP 是内置工具注册表，不会连接外部 MCP server，也不会聚合浏览器等工具。

## 后续扩展

- 增加 `/v1/file/upload`、`/v1/file/download`。
- 增加 file watch 和 SSE。
- 增加 Bash session TTL、输出落盘、后台任务恢复。
- 增加外部 MCP server 配置和 Hub 聚合。
- 增加 workspace 磁盘配额和更细粒度的只读/可写目录策略。
