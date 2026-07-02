# cbes-llm

## 本地单实例 MySQL 与 Redis Sentinel

后端使用 MySQL 作为主数据库，并使用 Redis 持久化登录 token 和缓存数据。本地中间件栈放在仓库根目录的 `deploy/` 目录中，当前部署方案只启动一个 MySQL 实例，不再使用 ShardingSphere、分库分表或读写分离；Redis 仍使用 Sentinel 拓扑：

```powershell
docker compose -f ../deploy/docker-compose.middleware.yml up -d --build
```

默认本地连接配置：

```text
url      = jdbc:mysql://127.0.0.1:3306/openclaw4j?useUnicode=true&characterEncoding=utf8&serverTimezone=Asia/Shanghai&useSSL=false&allowPublicKeyRetrieval=true
username = openclaw
password = openclaw123
```

`127.0.0.1:3306` 直接连接单个 MySQL 容器，数据库名为 `openclaw4j`。

Redis master 仍可通过 `127.0.0.1:6379`、database `0` 供简单本地客户端访问。Sentinel 入口为 `127.0.0.1:26379`，master name 为 `openclaw4j-master`；另外两个 Sentinel 发布在 `26380` 和 `26381`。

MySQL 容器只会在 `deploy/data/mysql` 第一次创建时导入 `src/main/resources/sql/MySQL/V0.0.1__init.sql`。Spring Boot 启动保持 `spring.sql.init.mode=never`，避免每次应用重启都重复执行初始化 SQL。运行时数据保存在已忽略的 `deploy/data/` 目录下。

后端默认 `cache.type=REDIS`，access token 和 refresh token 映射会写入 Redis，所以只重启后端进程不会立即让用户退出登录。本地开发的消息队列仍保持 `mq.type=JVM`。

重建干净的本地数据库：

```powershell
docker compose -f ../deploy/docker-compose.middleware.yml down
Remove-Item -Recurse -Force ..\deploy\data\mysql
docker compose -f ../deploy/docker-compose.middleware.yml up -d --build
```

如需覆盖后端连接配置：

```powershell
$env:OPENCLAW_MYSQL_URL='jdbc:mysql://127.0.0.1:3306/openclaw4j?useUnicode=true&characterEncoding=utf8&serverTimezone=Asia/Shanghai&useSSL=false&allowPublicKeyRetrieval=true'
$env:OPENCLAW_MYSQL_USERNAME='openclaw'
$env:OPENCLAW_MYSQL_PASSWORD='openclaw123'
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
