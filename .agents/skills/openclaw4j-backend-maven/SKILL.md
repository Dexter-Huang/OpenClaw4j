---
name: openclaw4j-backend-maven
description: Use when editing, testing, building, or committing OpenClaw4j-Bankend Java/Spring/Maven code inside the OpenClaw4j monorepo.
---

# OpenClaw4j Backend Maven

## 路径

- 后端根目录：`D:\IDEA_project\OpenClaw4j\OpenClaw4j-Bankend`
- 当前没有 Maven wrapper，使用系统 `mvn`。
- 始终固定本地 Maven 仓库：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' test
```

## 定向测试

admin 兼容接口相关改动优先跑：

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=AdminCompatApiTest' test
```

迭代阶段先用定向测试，最终完成或准备提交重要后端改动前再跑完整 `mvn ... test`。

## 可预期警告

Maven 可能出现 Lombok builder、native-access、`aspose-word-crack` 缺失 POM metadata 等警告。只要退出码为 `0` 且 Surefire 没有失败，不要把这些警告当成失败。

## Git 注意事项

从根仓库运行 Git：

```powershell
git -C D:\IDEA_project\OpenClaw4j status --short --branch
```

不要跟踪 `target/`、`data/`、本地 SQLite 文件、`.idea/` 或 `*.iml`。
