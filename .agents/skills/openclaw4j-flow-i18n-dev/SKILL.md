---
name: openclaw4j-flow-i18n-dev
description: Use when changing OpenClaw4j Frontend spark-flow, @spark-ai/flow, spark-i18n, i18n locale JSON, translation extraction scripts, FLOW_DEV, dev:flow, flow hot reload, or flow/i18n development workflow.
---

# OpenClaw4j Flow / I18n Dev

## 核心原则

改 `spark-flow` 或 `spark-i18n` 时，不要只改源码就结束。必须确认这次改动属于哪一类，并运行对应的同步和验证命令。

本 skill 需要和这些项目 skill 配合使用：

- **REQUIRED SUB-SKILL:** Use `openclaw4j-frontend-umi`
- **REQUIRED SUB-SKILL:** Use `openclaw4j-code-quality-gate` before reporting completion
- **REQUIRED SUB-SKILL:** Use `openclaw4j-fast-verification` to choose the smallest sufficient checks

## 路径

- 前端根目录：`D:\IDEA_project\OpenClaw4j\OpenClaw4j-Frontend`
- 主应用：`packages/main`
- Flow 源码：`packages/spark-flow/src`
- Flow 构建产物：`packages/spark-flow/dist`
- I18n 工具：`packages/spark-i18n`
- 主应用语言包：`packages/main/src/i18n/locales`
- Flow 语言包：`packages/spark-flow/src/i18n/locales`

## 开发模式

频繁改 `spark-flow` 时优先使用源码直连模式：

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Frontend\packages\main
npm run dev:flow
```

`dev:flow` 设置 `FLOW_DEV=true`，让 `@spark-ai/flow` 指向 `../spark-flow/src`。普通 `npm run dev` 和 `npm run build:app` 仍使用 `../spark-flow/dist`。

`spark-flow/src` 内部根路径必须使用 `@spark-flow/...`，不要新增 `@/...` 内部引用；`@` 在 `main` 中表示 `packages/main/src`，会和源码直连模式冲突。

## I18n 同步

`spark-i18n/src/index.ts` 已由 `packages/main/src/i18n/index.ts` 源码直连导入；改运行时类通常不需要安装依赖。

改代码里的 i18n key、`dm`、中文文案或 locale JSON 后，按影响范围运行：

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Frontend\packages\spark-i18n
npm run translate:main
npm run translate:spark-flow
```

只影响主应用时运行 `translate:main`；只影响 flow 时运行 `translate:spark-flow`。不要把自动生成的 locale 改动漏掉。

## 验证选择

按最小充分原则选择命令：

| 改动范围 | 先运行 | 最终验证 |
| --- | --- | --- |
| 只改 `spark-i18n/src/index.ts` 运行时 | `cd packages/main && npm run build` | `npm run build:app` |
| 改 `main` i18n 文案或 locale | `cd packages/spark-i18n && npm run translate:main` | `npm run build:app` |
| 改 `spark-flow` i18n 文案或 locale | `cd packages/spark-i18n && npm run translate:spark-flow` | `npm run build:flow` 和 `FLOW_DEV=true npm run build` |
| 改 `spark-flow/src` 组件、hooks、types | `cd packages/main; $env:FLOW_DEV='true'; npm run build` | `npm run build:flow` 和 `npm run build:app` |
| 改 `FLOW_DEV`、alias、构建脚本 | `cd packages/main; $env:FLOW_DEV='true'; npm run build` | `npm run build:flow` 和 `npm run build:app` |

PowerShell 下的源码直连构建命令：

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Frontend\packages\main
$env:FLOW_DEV='true'
npm run build
```

## 完成前检查

完成前至少确认：

- `rg "@/" packages/spark-flow/src --glob "*.ts" --glob "*.tsx"` 无结果。
- `rg "�" packages/spark-flow/src --glob "*.ts" --glob "*.tsx"` 无结果。
- `git diff --check` 退出码为 0。
- 相关 Prettier 检查通过，或报告精确失败文件。
- 没有把 `packages/*/dist`、`src/.umi*`、`node_modules` 作为可追踪改动带入。

如果质量检查失败，报告精确命令、退出码和关键错误；不要声称 clean 或 fully passing。
