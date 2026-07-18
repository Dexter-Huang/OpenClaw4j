# Spring AI 替换为 Eino 的可行性评估

评估日期：2026-07-18

## 结论

可以用 Eino 承接 Go 版 OpenClaw4j 后端里的 AI runtime，但不建议把它理解成 Spring AI 的一比一替换。

更准确的判断是：如果 `OpenClaw4j-Bankend-go` 目标是新建 Go 后端，Eino 适合作为模型调用、Agent、Tool Calling、RAG 编排和工作流图执行的核心框架；如果目标是保持现有 Java 后端，只局部替换 Spring AI，则 Eino 不能直接嵌入 Java 调用链，最多作为独立 Go sidecar 暴露 HTTP/RPC 接口。

整体难度：中高。普通模型调用和流式响应难度中等，Agent Tool Calling 与 RAG 中等偏高，完整 workflow runtime 迁移难度高。

## 当前后端实际使用的 Spring AI 能力

Java 后端不是只用了简单 chat completion。当前 Spring AI 相关依赖集中在这些链路：

- 模型调用：`ChatClient`、`ChatModel`、`OpenAiChatModel`、`OpenAiChatOptions`，用于 OpenAI-compatible 模型和 DashScope 兼容模式。
- 流式响应：大量使用 Reactor `Flux<AgentResponse>`，HTTP 层通过 SSE/`TEXT_EVENT_STREAM` 输出。
- Tool Calling：`ToolCallback`、`ToolCallbackProvider`、`ToolCallingManager`，承载 plugin、skill、MCP、app component 等工具。
- 记忆：`ChatMemory`、`MessageChatMemoryAdvisor`，用于会话上下文。
- RAG：`Document`、`DocumentRetriever`、`VectorStore`、`SearchRequest`、`FilterExpressionBuilder`、`PgVectorStore`。
- 文档解析：`MarkdownDocumentReader`、`PagePdfDocumentReader`、`TikaDocumentReader`、`TokenTextSplitter`。
- Advisor：`SimpleLoggerAdvisor` 和自定义 `KnowledgeBaseRetrievalAdvisor`，用于日志、知识库召回和提示词注入。
- 多模态：`UserMessage` + `Media`，支持图片 URL、本地文件、base64 数据。

这些能力在本项目中主要落在：

- `BasicAgentExecutor`：Agent 消息构造、流式调用、递归工具调用、RAG advisor、记忆、多模态。
- `ModelExecuteManager`：workflow LLM 节点底层模型调用。
- `LLMExecuteProcessor`、`ClassifierExecuteProcessor`、`ParameterExtractorExecuteProcessor`：workflow 节点调用模型并消费流式结果。
- `KnowledgeBaseIndexPipeline`、`DocumentServiceImpl`、`PgVectorStoreService`：文档解析、切分、向量写入、检索和 chunk 管理。
- `CompositeToolCallbackProvider` 及多个 `*ToolCallback`：把 OpenClaw4j 的 plugin、skill、MCP、component 封装为模型工具。

## Eino 能覆盖的部分

根据 Eino 官方文档和 CloudWeGo 开源仓库，Eino 是面向 Go 的 LLM 应用开发框架，核心模块包含 components、chain/graph、flow integration、callback、schema，并提供 Eino ADK、Eino Compose、Eino Dev 等配套能力。官方文档中明确有 ChatModel、Tool、Retriever、Indexer、Embedding、Document Loader、Transformer、Graph、Lambda、Callback、Checkpoint、Interrupt 等概念。

对应到 OpenClaw4j：

- ChatModel：可以替代 Spring AI 的 `ChatModel` / `ChatClient` 作为 Go 侧模型抽象。
- Tool Calling：可以承接 plugin、skill、MCP、component 的工具定义与调用，但需要重写 OpenClaw4j 的工具 schema、参数合并、调用记录和失败兜底逻辑。
- RAG：Eino 有 Retriever、Indexer、Embedding、Document Loader、Transformer 等模块，适合重建 RAG pipeline。
- Workflow/Graph：Eino 的 Graph 能承载部分 workflow 编排思想，但 OpenClaw4j 当前已有 22 类 workflow processor 和大量业务节点状态逻辑，不能直接迁过去，需要重新映射节点语义。
- Callback/Trace：Eino 有 callback 机制，可以承接日志、trace 和 token usage 记录，但要重新对齐当前 `Result`、`AgentResponse`、`WorkflowResponse`、observability 页面需要的数据结构。
- 中断/检查点：Eino 的 checkpoint/interrupt 对 workflow debug、resume、async result 有潜在价值，但要按现有前端接口契约适配。

## 不能直接等价替换的部分

### Spring 生态能力

Eino 不替代 Spring Boot、MVC、MyBatis Plus、Redisson、Spring 配置绑定、Spring Bean 生命周期、Spring exception handler、interceptor、validation 等后端基础设施。Go 版需要自己选型：

- HTTP：`net/http`、Gin、Chi、Fiber 等。
- ORM/SQL：`sqlc`、GORM、Ent、原生 `database/sql`。
- PostgreSQL/pgvector：Go 驱动和手写 SQL，或接入 Eino 扩展组件。
- Redis/MQ：`go-redis`、asynq、nats、或保留 Redis pub/sub。
- 配置：Viper、koanf、envconfig 或自研。
- 鉴权：JWT 中间件、API key 中间件、request context 透传。

### Spring AI 的高级封装

当前 Java 代码依赖 Spring AI 的几个“省心层”：

- `ToolCallingManager` 自动把模型 tool call 转成 tool response conversation history。
- `MessageChatMemoryAdvisor` 自动接入会话记忆。
- `PgVectorStore` 自动初始化 schema/table/index，并处理 Spring AI filter expression。
- `Tika/PDF/Markdown reader` 和 `TokenTextSplitter` 做文档解析和切分。
- `OpenAiChatOptions` 封装 stream usage、parallel tool calls、response format、模型参数。

Eino 有对应方向的能力，但 OpenClaw4j 需要自己写适配层，尤其是工具调用循环、RAG filter、pgvector 表结构、usage 归一化、reasoning content 透传、多模态消息构造。

## 推荐落地方式

推荐分三层做 Go 版，不建议一开始就全量复刻 Java 后端。

### 第一阶段：Eino AI Core 原型

在 `OpenClaw4j-Bankend-go` 先实现一个独立 AI core：

- `internal/ai/model`：模型配置、OpenAI-compatible client、stream/non-stream 调用。
- `internal/ai/message`：OpenClaw4j `ChatMessage` 与 Eino message schema 互转。
- `internal/ai/tool`：工具定义、工具调用记录、工具失败兜底。
- `internal/ai/agent`：非 workflow 的 Agent 调用，先覆盖 `BasicAgentExecutor` 的核心行为。
- `internal/ai/rag`：只做 knowledge base retrieval 的最小闭环，先不追求完整文档导入。

这一阶段验证：同一个 app 配置下，Go 版可以完成 chat completion、stream completion、基础 tool call、基础知识库召回。

### 第二阶段：RAG 与 MCP 适配

把 Java 后端里最依赖 Spring AI 的 RAG/MCP 能力补齐：

- pgvector 表结构与 Java 版兼容，优先复用已有 PostgreSQL schema。
- 实现 metadata filter，至少覆盖 `kb_id`、`document_id`、`enabled` 等当前业务查询。
- 文档解析先分层处理：文本/Markdown 先做，PDF/Office/Tika 类能力后补。
- MCP 先保留项目现有 SSE/Streamable HTTP 语义，不要只依赖框架默认实现。

这一阶段验证：知识库增删改查、文档 chunk、检索、MCP 工具列表和 call tool 与前端契约一致。

### 第三阶段：Workflow Runtime

最后迁 workflow，因为这里最不适合机械翻译：

- 用 Eino Graph 或自研 DAG executor 承接节点执行。
- 为每种节点建立明确 adapter：LLM、Classifier、Retrieval、Script、MCP、Plugin、Parallel、Iterator、Judge、Variable、Input/Output。
- 保留现有 `WorkflowResponse`、`NodeResult`、debug/init/resume/run_stream 等接口契约。
- 对齐 async result、stop task、part graph debug 和 node result cache。

这一阶段验证：前端 workflow debug 页面不改或少改即可跑通主路径。

## 风险点

- **接口契约风险**：前端已经绑定 `/console/v1`、`/api/v1/apps`、`Result<T>`、分页结构、SSE 数据结构。Go 版必须兼容这些 JSON 字段。
- **工具调用语义风险**：当前 Java 版对 plugin/skill/MCP/component 的 tool call 类型、参数合并、结果事件有业务定制，不能只用 Eino 默认 tool 调用。
- **RAG 兼容风险**：Spring AI `PgVectorStore` 的表结构、metadata、filter 表达式和 Eino/Go pgvector 方案不一定天然一致。
- **文档解析风险**：Java 的 Tika/PDF/Markdown reader 能力在 Go 侧需要另选库，PDF/Office 文档质量可能下降。
- **流式事件风险**：当前响应里包含 content、reasoning_content、tool_calls、usage、status。Go 版需要逐项还原。
- **测试基线风险**：现有后端测试只有 21 个 Java 测试文件，覆盖不足。Go 迁移前要先补接口契约测试，否则很容易“看起来跑通，前端细节坏掉”。

## 难度分级

- 简单 chat/stream：中等，约 1-2 周可做可演示原型。
- Agent + tool calling：中高，约 2-4 周形成可用核心。
- RAG + pgvector + 文档导入：中高，约 3-5 周。
- MCP + plugin + skill 全兼容：高，约 3-6 周。
- Workflow runtime 全量迁移：高，约 6-10 周，取决于需要覆盖多少节点和 debug 行为。
- 全后端 Go 化并替代 Java：高，建议按模块长期分期，不建议一次性切换。

## 建议

可以用 Eino，但建议定位为 `OpenClaw4j-Bankend-go` 的 AI runtime 基座，而不是 Spring AI 的直接替代品。

最稳的路线是先做 Go sidecar/实验后端，只接 `chat completion + tool calling + RAG retrieve` 三条链路；Java 后端继续承担管理端 CRUD、账户、工作区、provider/model 配置、文档管理和现有 workflow。等 Go 侧行为稳定后，再逐步把 Agent 和 Workflow runtime 从 Java 切过去。

若目标是追求启动速度、部署体积、并发和 Go 生态统一，Eino 是值得试的。如果目标只是减少 Spring AI 依赖，而后端仍保持 Java/Spring，改造收益不高，直接抽象一层自研 `ModelRuntime` 反而更现实。

## 参考资料

- Eino 官方文档：https://www.cloudwego.io/docs/eino/
- Eino GitHub 仓库：https://github.com/cloudwego/eino
- Eino ChatModel 指南：https://www.cloudwego.io/docs/eino/core_modules/components/chat_model_guide/
