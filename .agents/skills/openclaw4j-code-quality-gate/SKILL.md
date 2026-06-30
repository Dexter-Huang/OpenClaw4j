---
name: openclaw4j-code-quality-gate
description: Use when writing, editing, formatting, reviewing, or completing OpenClaw4j code and a final code quality, lint, format, test, build, or verification gate is required.
---

# OpenClaw4j Code Quality Gate

## 核心规则

每一次代码编写、修改、格式化、重构或修复，都必须在报告完成前执行格式检查和代码质量检查。如果检查失败，报告精确命令和关键错误，不要声称代码 clean。

## 工作流

1. 用 `git status --short` 和 `git diff --stat` 确认改动范围。
2. 按 `openclaw4j-fast-verification` 选择最小但足够的测试或构建命令。
3. 对每个被修改的代码区域运行格式检查。
4. 对每个被修改的代码区域运行质量检查。
5. 自动格式化或修复后，重新运行相关检查。
6. 汇总命令、退出状态和未解决的问题。

## Backend Java

从 `OpenClaw4j-Bankend/` 运行，使用 JDK 26 和固定 Maven 本地仓库：

```powershell
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' checkstyle:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotbugs:check
```

当用户要求格式化，或确实需要格式化时：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:apply
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:check
```

涉及行为变化时，先运行有针对性的 Maven 测试，再按 `openclaw4j-fast-verification` 扩大验证范围。

## Frontend TypeScript

从 `OpenClaw4j-Frontend/` 运行。优先使用能覆盖被修改 package 的最窄 formatter、linter 或 test 命令。如果没有明显更窄的命令，运行：

```powershell
npm run build:app
```

把 pre-commit lint-staged 失败视为质量门禁失败；修复报告的文件后重新运行相关命令。

## 仅文档或配置

仅修改文档或非运行时配置时，除非改动会影响运行时或构建行为，否则不要求构建。仍需验证：

```powershell
git status --short
git diff --check
```

如果仓库已有更具体的检查命令，优先使用它。

## 报告格式

使用基于证据的状态：

```text
Backend format: mvn ... spotless:check -> BUILD SUCCESS
Backend Checkstyle: mvn ... checkstyle:check -> BUILD FAILURE, N warnings/errors
Backend SpotBugs: mvn ... spotbugs:check -> BUILD FAILURE, 16 bugs
```

不要对本轮没有运行过的检查使用 "passed"、"clean" 或 "complete"。
