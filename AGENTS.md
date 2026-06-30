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

Use `spotless:apply` only when formatting the touched Java files is intended. For broad legacy cleanup, make a dedicated formatting commit.

Generated and legacy-problematic backend sources should stay excluded from quality checks until they are intentionally cleaned up:

- `src/main/java/com/seaskyland/llm/workflow/admin/generator/**`
- `src/main/java/com/seaskyland/llm/workflow/admin/utils/ContributorFileUtil.java`

## 代码质量门禁

- 每一次代码编写、修改、格式化、重构或修复后，在声明完成前都必须执行代码格式检查和代码质量检查。
- 使用项目 skill `openclaw4j-code-quality-gate` 选择具体流程和命令。
- 后端 Java 改动至少运行：

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotless:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' checkstyle:check
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' spotbugs:check
```

- 如果明确需要格式化，先运行 `spotless:apply`，再重新运行 `spotless:check`。
- 前端改动从 `OpenClaw4j-Frontend/` 运行相关 formatter、linter 或 build 脚本；如果没有更窄的脚本，运行 `npm run build:app`。
- 如果必要的质量检查因为已知历史基线问题无法通过，必须报告精确命令、退出码和剩余问题；不要把任务描述为 clean 或 fully passing。

## Leyden / AOT

- Keep the default startup optimization on HotSpot JVM with the JDK Project Leyden style AOT cache.
- Do not add GraalVM native-image behavior to `dev`, `prod`, `leyden`, or the Dockerfile unless explicitly requested.
- Rebuild and retrain the AOT cache after Java code, dependency, JVM flag, or extracted layout changes.
