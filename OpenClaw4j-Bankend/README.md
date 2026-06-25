# cbes-llm

## Leyden / HotSpot AOT cache

The default optimized backend packaging keeps the normal JVM runtime and uses the Project Leyden style AOT cache available in JDK 25.

Build the Leyden-ready jar:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pleyden -DskipTests package
```

Train a cache from the profiled startup path. The training and runtime must use the same JDK version, operating system, architecture, jar, and extracted application layout.

```powershell
./scripts/train-leyden-aot.ps1
```

Run with the trained cache:

```powershell
./scripts/run-leyden-aot.ps1
```

The Dockerfile follows the same pattern: build with JDK 25, extract the Spring Boot jar with `-Djarmode=tools`, create `openclaw4j.aot` during the image build by running the real `LLMApplication` in profiled training mode, and start `LLMApplication` with the same explicit classpath plus `-XX:AOTCache=openclaw4j.aot`.

The default training mode is `profiled`. It starts the full Spring web context, warms a short list of local endpoints, then exits normally so the JVM can write a representative AOT cache. This follows the same operational shape as production Leyden deployments: train representative startup behavior during the image/build step and run the same extracted layout with the generated archive.

`./scripts/train-leyden-aot.ps1 -TrainingMode classpath` remains available as a fast fallback that only runs `LeydenTrainingApplication`. `./scripts/train-leyden-aot.ps1 -TrainingMode spring` exits on context refresh and is useful for experiments. Profiled cache assembly can take several minutes, so the script waits up to 900 seconds by default.

Compare startup with and without the cache:

```powershell
./scripts/benchmark-leyden-startup.ps1
```

## GraalVM native-image experiment

GraalVM native-image support is available only as an explicit experiment profile. It is not used by the Leyden profile or Dockerfile.

Prepare the Spring AOT/native experiment build without creating a native executable:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pnative-experiment -DskipNativeBuild=true -DskipTests package
```

Build the native executable when a compatible GraalVM/native-image toolchain is installed:

```powershell
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' -Pnative-experiment -DskipNativeBuild=false -DskipTests native:compile
```

langfuse 配置
```python
import os
import base64
import urllib.parse

# 从环境变量获取凭证
LANGFUSE_PUBLIC_KEY = 'pk-lf-b8ddc404-72e0-4734-a8aa-aa172cda43d9'
LANGFUSE_SECRET_KEY = 'sk-lf-a058cb10-a3ea-42a8-9eee-276e9cca6956'

# 对凭证进行Base64编码
LANGFUSE_AUTH = base64.b64encode(
    f'{LANGFUSE_PUBLIC_KEY}:{LANGFUSE_SECRET_KEY}'.encode()
).decode()

# 构造认证头
auth_header = f'Basic {LANGFUSE_AUTH}'
print(auth_header)
```

如何关闭otel
```yaml
# 如果要关闭检测功能
management:
  tracing:
    enabled: false
  observations:
    annotations:
      enabled: false

otel:
  traces:
    exporter: none
  metrics:
    exporter: none
  logs:
    exporter: none
```

spel血泪教训：
```java
// 下面是正确的
List<String> docIds = List.of("1");
var b = new FilterExpressionBuilder();
var exp = b
        .and(b.eq(RagConstants.KEY_WORKSPACE_ID, "1"),  // 确保使用字符串类型
                b.in(RagConstants.KEY_DOC_ID, docIds.toArray()))
        .build();
// 下面会被错误解析成二位数组
List<String> docIds = List.of("1");
var b = new FilterExpressionBuilder();
var exp = b
        .and(b.eq(RagConstants.KEY_WORKSPACE_ID, "1"),  // 确保使用字符串类型
                b.in(RagConstants.KEY_DOC_ID, docIds))
        .build();
```
