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

Run with the cache:

```powershell
./scripts/run-leyden-aot.ps1
```

The AOT cache is tied to the exact JDK version, OS, CPU architecture, jar content, classpath, and extracted application layout. Rebuild and retrain after code or dependency changes. Training and runtime must use the same JDK 25/runtime image.

The default training mode is `profiled`. It starts the real `com.seaskyland.llm.LLMApplication`, enables `openclaw4j.leyden.training.profiled=true`, warms a short list of local endpoints, and exits normally so the JVM can write a representative AOT cache. `-TrainingMode classpath` remains available as the fast fallback that only runs `LeydenTrainingApplication`; `-TrainingMode spring` exits on context refresh for experiments.

Benchmark startup with and without the cache:

```powershell
./scripts/benchmark-leyden-startup.ps1
```

The training script waits up to 900 seconds because profiled Spring training gives JDK 25 a larger startup image and may spend several minutes assembling `openclaw4j.aot` from the temporary `openclaw4j.aot.config`.

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
