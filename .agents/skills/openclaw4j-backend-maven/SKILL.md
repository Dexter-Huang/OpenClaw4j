---
name: openclaw4j-backend-maven
description: Use when editing, testing, building, or committing OpenClaw4j-Bankend Java/Spring/Maven code inside the OpenClaw4j monorepo.
---

# OpenClaw4j Backend Maven

## Paths

- Backend root: `D:\IDEA_project\OpenClaw4j\OpenClaw4j-Bankend`
- There is no Maven wrapper; use system `mvn`.
- Always pin the local Maven repository:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' test
```

## Targeted Tests

For admin compatibility API changes, prefer:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=AdminCompatApiTest' test
```

For backend build/runtime-shape changes, prefer:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=LeydenBuildConfigurationTest' test
```

Use targeted tests during iteration, then run the full backend test suite before claiming important backend work is complete:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' test
```

## Expected Warnings

Maven may emit Lombok builder, native-access, deprecated API, or missing POM metadata warnings. If the exit code is `0` and Surefire reports no failures, do not treat those warnings as failures.

## Leyden / HotSpot AOT Cache

The default startup optimization keeps HotSpot JVM and uses the JDK 25 Project Leyden style AOT cache.

Build the Leyden-ready jar:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pleyden -DskipTests package
```

Train the cache:

```powershell
./scripts/train-leyden-aot.ps1
```

Use the representative warmup profile unless the user asks for a narrower or experimental run:

```powershell
./scripts/train-leyden-aot.ps1 -WarmupProfile representative -WarmupRequestTimeoutSeconds 10
```

Warmup profiles:

- `minimal`: application startup plus `/console/v1/system/health`.
- `representative`: stable GET endpoints that cover Spring MVC, Jackson, MyBatis, Redis-adjacent startup paths, and observability without external LLM calls.
- `extended`: extra admin list endpoints for experiments.
- `custom`: requires `-WarmupPaths`.

Run with the cache:

```powershell
./scripts/run-leyden-aot.ps1
```

The AOT cache is tied to the exact JDK version, OS, CPU architecture, jar content, classpath, and extracted application layout. Rebuild and retrain after code or dependency changes. Training and runtime must use the same JDK 25/runtime image.

The default training mode is `profiled`. It starts the real `com.seaskyland.llm.LLMApplication`, enables `openclaw4j.leyden.training.profiled=true`, warms representative local endpoints, and exits normally so the JVM can write an AOT cache. `-TrainingMode classpath` remains available as the fast fallback that only runs `LeydenTrainingApplication`; `-TrainingMode spring` exits on context refresh for experiments.

Profiled AOT recording makes the first request much slower than normal runtime. Do not assume a warmup timeout means the endpoint is broken. Common first-hit costs include DispatcherServlet initialization, Jackson setup, Hikari/MySQL connection creation, MyBatis query setup, and training-time JVM recording overhead. The runner logs each warmup duration plus a `Leyden profiled warmup summary`; use that summary to confirm which paths actually trained.

Do not train all API endpoints by default. Full endpoint sweeps usually add authentication, DB state, file upload, external model, and long-running workflow noise while giving little extra startup benefit. Prefer representative GET endpoints, then measure first-request latency with the benchmark script before expanding the profile.

Benchmark startup with and without the cache:

```powershell
./scripts/benchmark-leyden-startup.ps1
```

The benchmark refreshes `target/leyden-extracted` before every run so it measures the current jar rather than a stale extracted layout. The benchmark also reports both startup timing and optional first-request probes. Treat these as separate metrics: `springStartedSeconds` / `processRunningSeconds` show startup, while `firstRequestMilliseconds` shows startup-aftershock cost for the probe endpoints.

The training script waits up to 900 seconds because profiled Spring training gives JDK 25/26 a larger startup image and may spend several minutes assembling `openclaw4j.aot` from the temporary `openclaw4j.aot.config`.

SQLite vector store support and bundled `sqlite-vec` native resources were removed. Do not reintroduce `sqlite-jdbc`, `SqliteVectorStoreService`, `src/main/resources/sqlite/vec0.*`, or `spring.ai.alibaba.studio.sqlite-vec-extension-path` unless the user explicitly asks for a new SQLite experiment.

## GraalVM Native Experiment

GraalVM native-image support is kept only in the explicit `native-experiment` Maven profile. Do not add it to `dev`, `prod`, `leyden`, or the Dockerfile unless the user asks for a dedicated native-image experiment.

Prepare the Spring AOT/native experiment build without compiling a native executable:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pnative-experiment -DskipNativeBuild=true -DskipTests package
```

Run a real native-image compile only when a compatible GraalVM/native-image toolchain is installed and the user explicitly wants the long experiment:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pnative-experiment -DskipNativeBuild=false -DskipTests native:compile
```

## Git Notes

Run Git from the monorepo root:

```powershell
git -C D:\IDEA_project\OpenClaw4j status --short --branch
```

Do not track `target/`, `data/`, local SQLite files, `.idea/`, or `*.iml`.
