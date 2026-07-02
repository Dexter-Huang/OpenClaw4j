# cbes-llm

## 本地 PostgreSQL/pgvector 与 Redis Sentinel

后端使用 PostgreSQL 作为主数据库，并使用同一个 PostgreSQL 实例里的 pgvector 承载项目 RAG 向量数据。中间件栈在仓库根目录的 `deploy/` 目录中，只启动一个 `pgvector/pgvector:pg18` 实例，不使用 ShardingSphere、分库分表或读写分离；Redis 使用 Sentinel 拓扑。

```powershell
docker compose --env-file ../deploy/.env -f ../deploy/docker-compose.middleware.yml up -d --build
```

默认本地连接配置：

```text
url      = jdbc:postgresql://127.0.0.1:5432/openclaw4j
username = openclaw
password = openclaw
```

`127.0.0.1:5432` 直接连接 pgvector 容器，数据库名为 `openclaw4j`。这是从 0 开始的 PostgreSQL 开发库，不做 MySQL 数据迁移。

Redis master 可通过 `127.0.0.1:6379`、database `0` 供简单本地客户端访问。Sentinel 入口为 `127.0.0.1:26379`，master name 为 `openclaw4j-master`；另外两个 Sentinel 发布在 `26380` 和 `26381`。

Spring Boot 启动保持 `spring.sql.init.mode=never`，避免应用重启时重复执行初始化 SQL。pgvector 容器第一次创建 `deploy/data/pgvector` 时会自动执行 `src/main/resources/sql/PostgreSQL/V0.0.1__init.sql`。运行时数据保存在已忽略的 `deploy/data/` 目录下。

后端默认 `cache.type=REDIS`，消息队列默认 `mq.type=REDISSON`，两者都通过 Redis Sentinel 访问 Redis。

重建干净的本地 PostgreSQL/pgvector 数据库：

```powershell
docker compose --env-file ../deploy/.env -f ../deploy/docker-compose.middleware.yml down
Remove-Item -Recurse -Force ..\deploy\data\pgvector
docker compose --env-file ../deploy/.env -f ../deploy/docker-compose.middleware.yml up -d --build
```

如需覆盖后端连接配置：

```powershell
$env:OPENCLAW_DB_URL='jdbc:postgresql://127.0.0.1:5432/openclaw4j'
$env:OPENCLAW_DB_USERNAME='openclaw'
$env:OPENCLAW_DB_PASSWORD='openclaw'
$env:OPENCLAW_REDIS_SENTINEL_MASTER='openclaw4j-master'
$env:OPENCLAW_REDIS_SENTINEL_NODES='127.0.0.1:26379,127.0.0.1:26380,127.0.0.1:26381'
$env:OPENCLAW_REDIS_DATABASE='0'
```
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

langfuse 閰嶇疆
```python
import os
import base64
import urllib.parse

# 浠庣幆澧冨彉閲忚幏鍙栧嚟璇?
LANGFUSE_PUBLIC_KEY = 'pk-lf-b8ddc404-72e0-4734-a8aa-aa172cda43d9'
LANGFUSE_SECRET_KEY = 'sk-lf-a058cb10-a3ea-42a8-9eee-276e9cca6956'

# 瀵瑰嚟璇佽繘琛孊ase64缂栫爜
LANGFUSE_AUTH = base64.b64encode(
    f'{LANGFUSE_PUBLIC_KEY}:{LANGFUSE_SECRET_KEY}'.encode()
).decode()

# 鏋勯€犺璇佸ご
auth_header = f'Basic {LANGFUSE_AUTH}'
print(auth_header)
```

濡備綍鍏抽棴otel
```yaml
# 濡傛灉瑕佸叧闂娴嬪姛鑳?
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

spel琛€娉暀璁細
```java
// 涓嬮潰鏄纭殑
List<String> docIds = List.of("1");
var b = new FilterExpressionBuilder();
var exp = b
        .and(b.eq(RagConstants.KEY_WORKSPACE_ID, "1"),  // 纭繚浣跨敤瀛楃涓茬被鍨?
                b.in(RagConstants.KEY_DOC_ID, docIds.toArray()))
        .build();
// 涓嬮潰浼氳閿欒瑙ｆ瀽鎴愪簩浣嶆暟缁?
List<String> docIds = List.of("1");
var b = new FilterExpressionBuilder();
var exp = b
        .and(b.eq(RagConstants.KEY_WORKSPACE_ID, "1"),  // 纭繚浣跨敤瀛楃涓茬被鍨?
                b.in(RagConstants.KEY_DOC_ID, docIds))
        .build();
```

