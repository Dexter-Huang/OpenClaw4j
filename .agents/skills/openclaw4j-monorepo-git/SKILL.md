---
name: openclaw4j-monorepo-git
description: Use when working in OpenClaw4j after frontend/backend are managed by one root Git repository, especially when checking status, adding ignores, committing, or avoiding nested Git/submodule mistakes.
---

# OpenClaw4j Monorepo Git

## 核心原则

把 `D:\IDEA_project\OpenClaw4j` 当作唯一 Git 仓库。`OpenClaw4j-Frontend/` 和 `OpenClaw4j-Bankend/` 是普通子目录，不再是独立仓库。

## 工作流

1. 从根目录运行 Git 命令：

```powershell
git -C D:\IDEA_project\OpenClaw4j status --short --branch
```

2. 添加文件前先确认忽略规则：

```powershell
git -C D:\IDEA_project\OpenClaw4j status --ignored --short
git -C D:\IDEA_project\OpenClaw4j check-ignore -v <path>
```

3. 只添加应该被跟踪的源码、配置和文档。生成物和本地文件保持忽略：
   - 跟踪：源码、测试、`pom.xml`、`package.json`、`package-lock.json`、`.env.example`、SQL migrations、项目 skills
   - 忽略：`.env`、`node_modules/`、`dist/`、`.umi*`、`target/`、`data/`、IDE 文件、日志

4. 如果子目录里再次出现 `.git`，先停下来确认，不要直接删除，也不要把嵌套仓库元数据提交进去。

## 提交方式

每个关注点尽量单独提交：

```powershell
git add .gitignore .codex/skills
git commit -m "chore: configure monorepo ignores and project skills"
```

前端或后端代码改动要显式添加对应子目录路径，避免误带无关文件。

## 常见错误

- 在旧的子仓库路径里看 `git status`，然后误信过期上下文。
- 根 `.gitignore` 还没到位时添加 `OpenClaw4j-Frontend/node_modules` 或 `OpenClaw4j-Bankend/target`。
- 只依赖子目录 `.gitignore`；根仓库仍需要覆盖常见生成目录和本地配置。
