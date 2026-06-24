---
name: openclaw4j-fast-verification
description: Use when changing OpenClaw4j code and choosing what tests, builds, lint, or commit hooks to run without wasting time, especially after slow Maven/Umi verification cycles.
---

# OpenClaw4j Fast Verification

## 核心原则

先运行能证明当前改动的最小命令；在声明完成或提交重要改动前，再做一次足够宽的最终校验。

## 决策表

| 改动类型 | 首轮检查 | 最终检查 |
| --- | --- | --- |
| 后端 controller/service/test | `mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=<TestClass>' test` | `mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' test` |
| 后端 POM 或共享配置 | 优先定向 compile/test | 完整 `mvn ... test` |
| 前端 TypeScript 工具代码 | 没有更小 test/lint 脚本时跑 `npm run build:app` | 最后再跑一次 `npm run build:app` |
| 前端已暂存 TS/TSX | pre-commit 会跑 `umi lint` 和 prettier | hook 修改了运行时代码时再重跑 build |
| 仅 `.gitignore` 或文档 | `git status --ignored --short` / skill 校验 | 没有代码改动就不跑 build |

## 避免重复

- 后端专属改动不要反复跑前端 build。
- 小步迭代时先用定向 Maven 测试，不要每改一次就跑完整测试。
- pre-commit 失败时修具体 lint 错误，再重试提交；不要绕过 hook。
- hook 自动格式化后先看 `git diff --stat`；只有影响运行时或构建相关文件时才重跑 build。

## 完成证据

汇报时给出实际命令和结果：

```text
Backend: mvn ... test -> BUILD SUCCESS, Tests run: N, Failures: 0
Frontend: npm run build:app -> Webpack compiled successfully
```

没有在当前轮次运行过的检查，不要说它通过了。
