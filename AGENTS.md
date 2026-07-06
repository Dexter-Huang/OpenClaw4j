# OpenClaw4j Agent Notes

## Repository

- Treat `D:\IDEA_project\OpenClaw4j` as the only Git repository root.
- `OpenClaw4j-Bankend/` and `OpenClaw4j-Frontend/` are normal subdirectories.
- Run Git commands from the repository root.
- Do not track generated output such as `target/`, `data/`, `.idea/`, `*.iml`, `node_modules/`, `dist/`, or `.umi*`.

## Documentation

- Write project documentation in Chinese by default, including new docs, design notes, implementation plans, README updates, and deployment instructions.
- Keep code identifiers, API paths, command examples, configuration keys, error messages, and third-party product names in their original spelling when that is clearer.
- If an existing document is already in English, prefer converting touched sections to Chinese when making substantive edits, unless the user explicitly asks to keep that document in English.

## Development Design And Comments

- 任何代码改动都必须先理解所在模块的职责边界、调用链路和现有风格，再按项目既有架构落位；不要为了局部方便绕过已有 service、manager、repository、component、hook、request 封装等分层约定。
- 新增能力优先复用项目已有抽象、配置入口、错误处理、日志、国际化、请求响应结构和测试模式。只有当现有抽象无法清晰表达新行为，或会制造明显重复和耦合时，才新增小而明确的抽象。
- 设计代码时保持单一职责和可持续演进：公共逻辑应沉淀到合适的共享模块，行为分支应靠清晰的数据结构、枚举、策略或 helper 表达，避免把临时判断堆进大型方法或 UI 组件。
- 修改行为时同步考虑兼容性、失败路径、可观测性和后续维护成本；涉及接口契约、持久化结构、启动流程、构建部署或跨端联动时，必须显式说明影响范围并补充相应验证。
- 代码注释默认使用中文，注释应解释功能意图、架构原因、边界条件、兼容逻辑或非直观实现。不要为一眼可见的赋值、简单 getter/setter、普通分支写重复代码含义的空注释。
- 对复杂流程、重要兼容分支、外部系统约束、临时权衡和后续清理点，应写足够详细的中文注释，让后续开发者能理解“为什么这样做”以及“改动时需要注意什么”。如需保留第三方 API 名称、配置键、异常消息或协议术语，保持原文拼写。

## Backend Maven

- Backend root: `OpenClaw4j-Bankend/`.
- There is no Maven wrapper; use system `mvn`.
- This project defaults to JDK 26 at `D:\jdk-26\bin`.
- Before running Maven for backend work, make the current shell use that JDK:

```powershell
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
```

- Always pin the local Maven repository:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' test
```

- Use the smallest verification command that proves the current backend change first, then broaden only when needed.
- For backend build/runtime-shape changes, prefer:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=LeydenBuildConfigurationTest' test
```

## Backend Java Quality Tools

The backend uses Maven quality plugins as the Java equivalent of a Python `ruff`/`mypy` toolchain:

- `spotless-maven-plugin` with `google-java-format` for formatting.
- `maven-checkstyle-plugin` for style checks.
- `spotbugs-maven-plugin` for bytecode-level bug detection.

These tools are intentionally not bound to the default Maven lifecycle yet. Run them explicitly while the existing codebase is being baselined.

Common commands:

```powershell
cd OpenClaw4j-Bankend
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:apply
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' checkstyle:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotbugs:check
```

日常迭代时优先运行能证明当前改动的最小测试或构建命令。不要在每一次小改后反复运行
`spotless:check`、`checkstyle:check`、`spotbugs:check` 三件套；这些全量质量工具默认留到本轮代码改完、准备声明完成前再运行一次。

如果迭代中需要提前发现格式或明显质量问题，可以临时运行后端快速质量检查：

```powershell
cd OpenClaw4j-Bankend
.\scripts\check-backend-quality-fast.ps1
```

该脚本会：

- 固定使用 JDK 26 和 `D:\apache-maven-3.9.1\m2\repository`。
- 始终运行 `spotless:check`。
- 只对相对 `HEAD` 有变更的后端 Java 文件运行 Checkstyle。
- 只有 `src/main/java` 有变更时，才用 `-Dspotbugs.effort=Default` 运行 SpotBugs 快检；仅测试代码变更时跳过 SpotBugs。

快速质量检查只用于本地迭代提速，不能替代最终门禁。声明后端代码完成前，仍按本节后面的要求运行全量 `spotless:check`、`checkstyle:check`、`spotbugs:check`。

Use `spotless:apply` only when formatting the touched Java files is intended. For broad legacy cleanup, make a dedicated formatting commit.

Generated and legacy-problematic backend sources should stay excluded from quality checks until they are intentionally cleaned up:

- `src/main/java/com/seaskyland/llm/workflow/admin/generator/**`
- `src/main/java/com/seaskyland/llm/workflow/admin/utils/ContributorFileUtil.java`

## 代码质量门禁

- 代码编写、修改、格式化、重构或修复过程中，先用能证明当前行为的最小验证命令迭代。
- 不要在每一次小改后都运行 `spotless:check`、`checkstyle:check`、`spotbugs:check`；本轮所有代码改完、准备声明完成前，再运行一次最终格式检查和代码质量检查。
- 使用项目 skill `openclaw4j-code-quality-gate` 选择具体流程和命令。
- 后端 Java 改动的最终门禁至少运行：

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' checkstyle:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotbugs:check
```

- 如果明确需要格式化，可以在本轮代码改完后运行 `spotless:apply`，再重新运行 `spotless:check`。
- 前端改动从 `OpenClaw4j-Frontend/` 运行相关 formatter、linter 或 build 脚本；如果没有更窄的脚本，运行 `npm run build:app`。
- 如果必要的质量检查因为已知历史基线问题无法通过，必须报告精确命令、退出码和剩余问题；不要把任务描述为 clean 或 fully passing。

## Playwright 本地调试

- 需要验证前端真实页面时，优先用 Playwright 打开本机页面复现用户路径；动态应用必须等到必要接口返回或关键元素出现后再判断，不要只看初始 HTML。
- 临时 Playwright 脚本、截图、trace 和日志统一放在 `output/playwright/`，不要新增顶层调试目录，也不要提交这些生成物。
- 默认使用短小的 Node/Playwright 诊断脚本或 CLI 操作，避免为了临时排查创建长期 `@playwright/test` spec；除非用户明确要求端到端测试用例。
- 能用接口或测试辅助入口建立登录态时，不要手动登录；不要在 `AGENTS.md`、脚本模板或长期文档中固化具体账号、密码、接口路径或 token。
- 调试接口链路时，优先记录关键请求的 `status/url/resourceType/content-type` 和必要 JSON 字段；避免整页 `page.content()`、超长 DOM、完整响应体或大截图直接刷屏。
- 排查 `Unexpected token '<'` 这类前端脚本错误时，优先检查“`.js` 资源是否返回了 `text/html`”。可通过 Playwright CDP `Runtime.exceptionThrown` 定位异常脚本 URL，并记录脚本响应的 `content-type` 与开头片段。
- Umi/Utoopack dev server 如果把旧 chunk 或 overlay 脚本回退成 `index.html`，先确认本地依赖包产物是否已生成；必要时运行对应项目的 clean dev 脚本清理 `.umi*` 和缓存后重启。
- Playwright 脚本必须在 `finally` 或等效收尾路径里关闭 `page/context/browser`；使用 CLI 打开的标签页测试完要显式关闭，避免遗留标签页、浏览器进程或占用端口影响下一次调试。
- 对同一问题优先复用已有脚本并只调整输入或断言；如果必须新增诊断脚本，保持单一目的，输出尽量是结构化 JSON，字段只包含定位问题所需信息。
- 完成页面验证后，在回复里说明实际打开的 URL、关键接口状态、是否有 `pageerror/requestfailed/console:error`，以及测试结束时是否已关闭浏览器资源。

## Leyden / AOT

- Keep the default startup optimization on HotSpot JVM with the JDK Project Leyden style AOT cache.
- Do not add GraalVM native-image behavior to `dev`, `prod`, `leyden`, or the Dockerfile unless explicitly requested.
- Rebuild and retrain the AOT cache after Java code, dependency, JVM flag, or extracted layout changes.
