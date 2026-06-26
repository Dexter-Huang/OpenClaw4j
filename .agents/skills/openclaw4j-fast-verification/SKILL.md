---
name: openclaw4j-fast-verification
description: Use when changing OpenClaw4j code and choosing what tests, builds, lint, or commit hooks to run without wasting time, especially after slow Maven/Umi verification cycles.
---

# OpenClaw4j Fast Verification

## Core Principle

Run the smallest command that proves the current change first. Before claiming completion or preparing an important backend change, run a broad enough final verification.

## Decision Table

| Change type | First check | Final check |
| --- | --- | --- |
| Backend controller/service/test | `mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=<TestClass>' test` | `mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' test` |
| Backend POM/shared config | Targeted compile/test that covers the touched behavior | Full `mvn ... test` |
| Backend Leyden/JDK AOT cache config | `mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=LeydenBuildConfigurationTest' test` | `mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pleyden -DskipTests package`; when JDK 25/26 is available, also run `./scripts/train-leyden-aot.ps1 -WarmupProfile representative -WarmupRequestTimeoutSeconds 10`, `./scripts/run-leyden-aot.ps1 -ServerPort 0`, and for performance-sensitive changes `./scripts/benchmark-leyden-startup.ps1` |
| Backend GraalVM native experiment config | `mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=LeydenBuildConfigurationTest' test` | `mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pnative-experiment -DskipNativeBuild=true -DskipTests package`; do not run `native:compile` unless the user explicitly wants the long GraalVM experiment |
| Frontend TypeScript app code | If there is no smaller test/lint script, run `npm run build:app` | Run `npm run build:app` again at the end |
| `.gitignore` or docs only | `git status --ignored --short` / relevant doc checks | No build needed unless runtime/build files changed |

## Avoid Repetition

- Backend-only changes do not require frontend builds.
- Iterate with targeted Maven tests before the full suite.
- If a hook auto-formats files, check `git diff --stat`; rerun builds only when runtime or build-related files changed.

## Leyden/JDK AOT Benchmark Guardrails

- Use an explicit JDK executable for JDK comparison work, for example `D:\jdk-26\bin\java.exe`; do not trust `java` from `PATH` on Windows because `javapath` shims can hide the real process and distort memory sampling.
- Keep AOT caches tied to the exact JDK and flag set that created them. Do not run a JDK 26 process with a JDK 25 cache, or run a compact-object-header cache without `-XX:+UseCompactObjectHeaders`.
- Rebuild and retrain after changing Java code, dependencies, JVM flags, extracted layout, or the Leyden scripts. A previous `target/openclaw4j.aot` is not valid evidence after a fresh jar package.
- Ensure benchmark runs refresh `target/leyden-extracted`; stale extracted layouts can make logs and startup timings come from an older jar even after a successful package.
- Profiled AOT training for this Spring backend can take 10-15 minutes locally. Run it with progress logging and phase markers instead of an opaque short-timeout shell command.
- Start long AOT training with `Start-Process` and redirected stdout/stderr files. Do not treat stderr warnings such as native-access or `JAVA_TOOL_OPTIONS` output as process failure.
- On Windows, be careful when launching nested PowerShell with `$env:Path`; accidental outer-shell interpolation can turn semicolon-separated PATH entries into commands. Prefer direct synchronous training unless the quoting is already proven.
- Windows PowerShell may require `Add-Type -AssemblyName System.Net.Http` before benchmark probe code can use `[System.Net.Http.HttpClient]`.
- Prefer representative, deterministic GET warmup endpoints. Do not train all endpoints by default; full sweeps add auth, DB state, upload, external LLM, and long workflow noise with little startup benefit.
- A warmup timeout is not automatically an endpoint bug. In AOT recording, first-hit costs include DispatcherServlet, Jackson, Hikari/MySQL connection creation, MyBatis query setup, and JVM recording overhead. Read the runner's per-path durations and `Leyden profiled warmup summary`.
- SQLite vector store support and bundled `sqlite-vec` native resources were removed. Treat any reappearance of `sqlite-jdbc`, `SqliteVectorStoreService`, `src/main/resources/sqlite/vec0.*`, or `spring.ai.alibaba.studio.sqlite-vec-extension-path` as a regression unless the user asks for a new SQLite experiment.
- For two-mode startup comparisons, keep exactly two fixed command lines: one ordinary baseline with no JVM optimization flags, and one optimized command with the intended AOT/GC/heap flags. Report startup and memory as separate results.
- Separate startup from first-request cost. `springStartedSeconds` and `processRunningSeconds` measure startup; `firstRequestMilliseconds` measures the first probe calls after startup and can show benefits that startup-only numbers hide.
- When startup time regresses, isolate JVM options before changing code: benchmark baseline, compact headers only, AOT only, GC/heap flags only, then the combined command. ZGC, string deduplication, heap caps, `AlwaysPreTouch`, and large pages are latency/memory/runtime tuning knobs; they can hide AOT startup gains in a cold-start benchmark.
- Sample memory from the real Java process launched by the explicit executable, then use `jcmd GC.heap_info` for heap details. Parse G1 output as committed/used heap, and ZGC output as `ZHeap used/capacity`; missing G1 fields in ZGC output must not be reported as zero heap.

## Completion Evidence

Report actual commands and results, for example:

```text
Backend: mvn ... test -> BUILD SUCCESS, Tests run: N, Failures: 0
Leyden package: mvn ... -Pleyden -DskipTests package -> BUILD SUCCESS
```

Do not say a check passed unless it was run in the current turn.
