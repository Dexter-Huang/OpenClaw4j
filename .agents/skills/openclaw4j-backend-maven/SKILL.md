---
name: openclaw4j-backend-maven
description: Use when editing, testing, building, or committing OpenClaw4j-Bankend Java/Spring/Maven code inside the OpenClaw4j monorepo.
---

# OpenClaw4j 后端 Maven

## 路径

- 后端根目录：`D:\IDEA_project\OpenClaw4j\OpenClaw4j-Bankend`
- 项目没有 Maven wrapper，使用系统 `mvn`。
- 始终固定 Maven 本地仓库：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' test
```

## 定向测试

修改 admin compatibility API 时，优先运行：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=AdminCompatApiTest' test
```

修改后端构建、运行时结构或打包形态时，优先运行：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=LeydenBuildConfigurationTest' test
```

迭代时先运行定向测试；在声明重要后端工作完成前，再运行完整后端测试：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' test
```

## 质量检查

后端 Java 改动的本地迭代阶段，默认先运行能证明当前行为的定向测试或最小构建。不要在每一次小改后反复运行完整质量工具。

如果需要提前发现格式或明显静态分析问题，可以运行快速质量检查：

```powershell
.\scripts\check-backend-quality-fast.ps1
```

该脚本会运行 Spotless、仅针对变更 Java 文件的 Checkstyle，并且只在 main Java class 有变更时运行快速 SpotBugs。它不是最终质量门禁，也不要求每次小改后都运行；本轮代码稳定、声明后端代码完成前，按 `openclaw4j-code-quality-gate` 的要求运行完整 `spotless:check`、`checkstyle:check` 和 `spotbugs:check`。

## 预期警告

Maven 可能输出 Lombok builder、native-access、deprecated API 或缺失 POM metadata 等警告。如果退出码是 `0`，且 Surefire 没有测试失败，不要把这些警告当作构建失败。

## Leyden / HotSpot AOT 缓存

默认启动优化保持使用 HotSpot JVM，并使用 JDK 25 Project Leyden 风格的 AOT cache。

构建 Leyden-ready jar：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pleyden -DskipTests package
```

训练缓存：

```powershell
./scripts/train-leyden-aot.ps1
```

除非用户要求更窄或实验性的运行方式，否则使用 representative warmup profile：

```powershell
./scripts/train-leyden-aot.ps1 -WarmupProfile representative -WarmupRequestTimeoutSeconds 10
```

Warmup profiles：

- `minimal`：应用启动加 `/console/v1/system/health`。
- `representative`：稳定的 GET 端点，覆盖 Spring MVC、Jackson、MyBatis、Redis-adjacent 启动路径和 observability，不包含外部 LLM 调用。
- `extended`：用于实验的额外 admin list 端点。
- `custom`：需要传入 `-WarmupPaths`。

使用缓存运行：

```powershell
./scripts/run-leyden-aot.ps1
```

AOT cache 绑定精确的 JDK 版本、操作系统、CPU 架构、jar 内容、classpath 和解压后的应用布局。Java 代码或依赖变化后必须重新构建并重新训练。训练和运行必须使用同一个 JDK 25/runtime image。

默认训练模式是 `profiled`。它会启动真实的 `com.seaskyland.llm.LLMApplication`，启用 `openclaw4j.leyden.training.profiled=true`，预热 representative 本地端点，并正常退出，让 JVM 写入 AOT cache。`-TrainingMode classpath` 仍可作为快速 fallback，只运行 `LeydenTrainingApplication`；`-TrainingMode spring` 会在 context refresh 后退出，用于实验。

Profiled AOT recording 会让第一次请求比正常运行慢很多。不要把 warmup timeout 直接判断为端点损坏。常见 first-hit 成本包括 DispatcherServlet 初始化、Jackson 设置、Hikari/MySQL 连接创建、MyBatis 查询设置，以及训练时 JVM recording 开销。runner 会记录每个 warmup 的耗时和 `Leyden profiled warmup summary`；用这些日志确认实际训练了哪些路径。

不要默认训练全部 API 端点。全量端点 sweep 往往会引入认证、数据库状态、文件上传、外部模型和长时间 workflow 噪声，对启动收益很小。优先使用 representative GET 端点，再用 benchmark 脚本测量 first-request latency，之后再决定是否扩展 profile。

对比有无缓存的启动表现：

```powershell
./scripts/benchmark-leyden-startup.ps1
```

benchmark 每次运行前都会刷新 `target/leyden-extracted`，确保测量的是当前 jar，而不是过期解压布局。benchmark 同时报告启动耗时和可选 first-request probes。把这些指标分开理解：`springStartedSeconds` / `processRunningSeconds` 表示启动，`firstRequestMilliseconds` 表示启动后探测端点的首次请求成本。

训练脚本最长等待 900 秒，因为 profiled Spring training 会给 JDK 25/26 生成更大的 startup image，可能需要几分钟从临时 `openclaw4j.aot.config` 组装 `openclaw4j.aot`。

SQLite vector store 支持和内置 `sqlite-vec` native resources 已移除。除非用户明确要求新的 SQLite 实验，否则不要重新引入 `sqlite-jdbc`、`SqliteVectorStoreService`、`src/main/resources/sqlite/vec0.*` 或 `spring.ai.alibaba.studio.sqlite-vec-extension-path`。

## GraalVM Native 实验

GraalVM native-image 支持只保留在显式的 `native-experiment` Maven profile 中。除非用户要求专门的 native-image 实验，否则不要把它加到 `dev`、`prod`、`leyden` 或 Dockerfile。

准备 Spring AOT/native experiment 构建，但不编译 native executable：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pnative-experiment -DskipNativeBuild=true -DskipTests package
```

只有在已安装兼容的 GraalVM/native-image 工具链，且用户明确需要长时间实验时，才运行真实 native-image compile：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pnative-experiment -DskipNativeBuild=false -DskipTests native:compile
```

## Git 说明

从 monorepo 根目录运行 Git：

```powershell
git -C D:\IDEA_project\OpenClaw4j status --short --branch
```

不要跟踪 `target/`、`data/`、本地 SQLite 文件、`.idea/` 或 `*.iml`。
