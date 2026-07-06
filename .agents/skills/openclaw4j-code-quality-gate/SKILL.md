---
name: openclaw4j-code-quality-gate
description: Use when writing, editing, formatting, reviewing, or completing OpenClaw4j code and a final code quality, lint, format, test, build, or verification gate is required.
---

# OpenClaw4j Code Quality Gate

## 核心规则

代码编写、修改、格式化、重构或修复过程中，优先用能证明当前假设的最小测试、构建或手工复现命令迭代。不要在每一次小改后反复运行全量 `spotless:check`、`checkstyle:check`、`spotbugs:check` 或前端完整 build。

本轮所有代码改完、准备声明完成前，必须执行与改动范围匹配的最终格式检查和代码质量检查。检查失败时，报告精确命令、退出码和剩余问题，不要把结果描述为 clean、passed 或 complete。

## 工作流

1. 用 `git status --short` 和 `git diff --stat` 确认改动范围。
2. 按 `openclaw4j-fast-verification` 选择最小但足够的测试或构建命令。
3. 迭代阶段按需运行定向测试、局部构建或手工复现；除非正在专门排查格式/静态分析问题，不要每改一点就跑全量质量工具。
4. 本轮代码改完、声明最终完成前，运行一次全量质量门禁，除非改动确实只是文档或非运行时配置。
5. 自动格式化或修复后，重新运行相关检查。
6. 汇总实际命令、退出状态和未解决问题。

## Backend Java

从 `OpenClaw4j-Bankend/` 运行，使用 JDK 26 和固定 Maven 本地仓库：

```powershell
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
```

日常迭代默认先运行定向测试或最小构建。需要提前发现格式或明显质量问题时，可以运行快检：

```powershell
.\scripts\check-backend-quality-fast.ps1
```

快检会运行 `spotless:check`、只针对变更 Java 文件的 Checkstyle，并在 `src/main/java` 有变更时用 `-Dspotbugs.effort=Default` 运行 SpotBugs。它不能替代最终全量门禁，也不要求每次小改后都运行。

后端 Java 改动的最终质量门禁：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' checkstyle:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotbugs:check
```

需要格式化时，优先在本轮代码改完后运行：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:apply
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:check
```

涉及行为变化时，迭代中先运行有针对性的 Maven 测试；等代码稳定后，再按 `openclaw4j-fast-verification` 扩大验证范围并执行最终质量门禁。

## Frontend TypeScript

从 `OpenClaw4j-Frontend/` 运行。迭代中优先使用能覆盖被修改 package 的最窄 test 或 build 命令；本轮代码改完后再运行相关 formatter、linter 或最终 build。如果没有更窄的命令，最终运行：

```powershell
npm run build:app
```

把 pre-commit lint-staged 失败视为质量门禁失败；修复报告的文件后重新运行相关命令。

## 文档或配置

仅修改文档或非运行时配置时，除非改动会影响运行时或构建行为，否则不要要求完整构建。仍需验证：

```powershell
git status --short
git diff --check
```

如果仓库已有更具体的检查命令，优先使用它。

## 报告格式

使用基于证据的状态：

```text
Backend fast quality: .\scripts\check-backend-quality-fast.ps1 -> SUCCESS
Backend format: mvn ... spotless:check -> BUILD SUCCESS
Backend Checkstyle: mvn ... checkstyle:check -> BUILD FAILURE, N warnings/errors
Backend SpotBugs: mvn ... spotbugs:check -> BUILD FAILURE, 16 bugs
```

不要对本轮没有运行过的检查使用 passed、clean 或 complete。
