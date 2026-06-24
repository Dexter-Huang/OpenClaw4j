---
name: openclaw4j-frontend-umi
description: Use when editing, testing, building, linting, or committing OpenClaw4j-Frontend Umi/React/TypeScript code inside the OpenClaw4j monorepo.
---

# OpenClaw4j Frontend Umi

## 路径

- 前端根目录：`D:\IDEA_project\OpenClaw4j\OpenClaw4j-Frontend`
- 主应用：`OpenClaw4j-Frontend\packages\main`

## 构建

使用前端根目录脚本：

```powershell
npm run build:app
```

构建通过的关键证据是 `Webpack: Compiled successfully`。

## 提交 Hook

pre-commit 会通过 lint-staged 对暂存的 TypeScript/TSX 文件运行 `umi lint` 和 prettier。提交失败时：

1. 先读清楚具体 lint 错误、文件和行号。
2. 只修这个问题；除非它暴露了紧邻的格式问题。
3. 重新提交，让 hook 再跑一次。

除非用户明确要求，不要绕过 hooks。

## 忽略规则

不要跟踪：

- `node_modules/`
- `packages/*/node_modules/`
- `packages/*/dist/`
- `packages/*/src/.umi*`
- `.env`, `.env.local`, `.umirc.local.ts`, `config/config.local.ts`

需要跟踪 `.env.example` 和 lock 文件。
