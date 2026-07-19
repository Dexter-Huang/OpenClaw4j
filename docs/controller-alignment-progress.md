# Java Controller 对齐进度

记录日期：2026-07-19

## 已完成并验证

- Console 基础管理接口：账户、工作区、API Key、Provider/Model、应用、Agent Schema、Tool、Plugin、Skill、知识库和文件。
- 应用组件接口：CRUD、按应用或编码查询、配置/Schema、可发布应用与引用查询。
- 文档及分块接口：文档 CRUD、重建索引状态、分块 CRUD、启停与预览。`document_chunk` 迁移已用于本地运行库。
- Skill 包上传：校验 ZIP、`SKILL.md`、可选 `manifest.json`、文件数量和解压大小，并安全解压到本地存储。
- 文件上传策略：`/console/v1/files/upload-policies` 与本地上传/下载闭环。
- MCP Streamable HTTP：CRUD、`tools/list` 工具发现和 `tools/call` 调试；当前不包含 stdio/SSE transport。
- Chat/AppChat：`/console/v1/apps/chat/completions` 与 `/api/v1/apps/chat/completions` 已注册。支持从应用配置解析 `model_provider`、`model.model_id`，使用 Java 兼容 RSA 私钥解密 Provider API key，并转发至 OpenAI 兼容的 `/chat/completions`。流式请求返回 SparkChat 所需的 SSE 消息结构。
- 应用发布已兼容前端保存的 `model` 对象配置，不再只接受字符串模型 ID。
- `Oauth2Controller`：已实现 GitHub OAuth2 授权地址、回调兑换、外部账号绑定或创建及登录 token 签发；未配置 GitHub 参数时返回明确的不可用响应。
- `WorkflowController`：已实现调试初始化、任务运行、轮询、恢复、局部图运行、停止，以及 `/api/v1/apps/workflow/*` 同步/异步调用。任务和节点执行状态由 Go 的工作流服务统一管理，返回 Java 前端所需的 `node_results`、`batches` 与 JSON 字符串输入输出结构。
- Java 遗留 `AdminCompatController`：已完成 Prompt、Dataset、Evaluator、Experiment 和模型列表的未版本化 `/api/*` 路由。持久化逻辑复用 PostgreSQL，表名和排序列均采用白名单。
- `ObservabilityCompatController`：已完成进程内 Trace 写入、查询、详情、服务聚合与概览指标；行为与 Java 的 `CopyOnWriteArrayList` 语义一致。

## 运行与验证证据

- 本轮 Go 后端：`http://127.0.0.1:9005`，健康检查 `GET /healthz` 返回 `{"status":"ok"}`；前端开发服务为 `http://127.0.0.1:8006`。
- 最终 `go test ./...` 通过。
- HTTP 冒烟验证：`/api/models` 返回 449 条模型；Prompt 写入和删除、Trace 写入与 token 聚合均返回 `code: 200`。
- Playwright 实际登录后，`/app`、Workflow 编辑页、`/admin/prompts` 和 `/admin/tracing` 均通过 9005 返回 HTTP 200。Prompt 页面实际渲染 PostgreSQL 记录；Trace 页面实际渲染本轮写入的 `compat-smoke-trace`。
- Workflow 浏览器路径成功调用 `debug/init`、`debug/run-task` 和 `debug/get-task-process`，节点结果面板能正常渲染。测试工件位于 `output/playwright/go-controller-e2e-20260719/`，浏览器已关闭。
- Playwright CLI 已实际登录并进入 `/app`；端到端脚本覆盖 `/app`、模型服务、API Key、Agent Schema、知识库、MCP、Skill 页面。
- 最新端到端脚本验证了 Skill、文件策略、Plugin/Tool、知识库/文档/分块、应用组件、MCP 工具发现和调试、Chat Provider/Model/应用发布/普通调用/SSE 调用，关键请求均为 HTTP 200。
- Chat 验证使用临时 OpenAI mock：普通响应为 `mock answer:hello`，SSE 响应为 `text/event-stream`；临时 Provider、Model、应用均在测试结束时删除。
- 最后一轮浏览器脚本无 `requestfailed`；有 2 条现有 Ant Design `useForm` 控制台警告，未见页面错误。

## 尚存运行依赖

- `ChatController` 的完整 Agent 行为：工具调用、RAG、记忆、Skills、组件编排与模型流式 token 透传尚未迁移；当前是已配置基础应用的 OpenAI 兼容完成调用。
- MCP 的 stdio 和传统 SSE transport 尚未实现；当前仅验证 Streamable HTTP。
- 知识库检索尚未接入向量索引/召回，现有文档分块仅提供持久化管理能力。
- Workflow 模型调用已对齐 Java `OpenAiChatOptions.baseUrl` 语义：Provider endpoint 中的路径前缀会被保留，避免 `/compatible-mode/v1` 一类前缀因 URL 合成而丢失。失败时当前节点会回写 `fail` 与 `error_code/error_info`，不会再长期显示为 `executing`。

## 继续前的约束

- 不应把上述运行依赖以空成功响应或本地假数据标记为已对齐。
- Chat 运行需要设置 `OPENCLAW_PROVIDER_PRIVATE_KEY_FILE` 指向与 Provider 公钥配套的 PKCS#8 私钥文件；本地验证使用 Java 后端资源中的 `keys/private.pem`。
- 前后端当前仍在运行，后续接手时应先确认端口和进程归属，避免启动旧的 9004/8000 服务。
