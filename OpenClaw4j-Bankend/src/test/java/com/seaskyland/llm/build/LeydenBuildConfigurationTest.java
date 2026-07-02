package com.seaskyland.llm.build;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import org.junit.jupiter.api.Test;

class LeydenBuildConfigurationTest {

  private static final Path PROJECT_DIR = Path.of("").toAbsolutePath();

  @Test
  void mavenCompilerTargetsConfiguredJdkRelease() throws IOException {
    String pom = read("pom.xml");

    assertTrue(pom.contains("<java.version>26</java.version>"));
    assertTrue(pom.contains("<release>${java.version}</release>"));
    assertFalse(pom.contains("<source>17</source>"));
    assertFalse(pom.contains("<target>17</target>"));
  }

  @Test
  void defaultBackendConfigurationUsesMysqlAsPrimaryDatabase() throws IOException {
    String application = read("src/main/resources/application.yml");
    String pom = read("pom.xml");
    String trainingApplication =
        read("src/main/java/com/seaskyland/llm/LeydenTrainingApplication.java");

    assertTrue(
        application.contains("url: ${OPENCLAW_MYSQL_URL:jdbc:mysql://127.0.0.1:3306/openclaw4j"));
    assertTrue(application.contains("driver-class-name: com.mysql.cj.jdbc.Driver"));
    assertTrue(application.contains("username: ${OPENCLAW_MYSQL_USERNAME:openclaw}"));
    assertTrue(application.contains("password: ${OPENCLAW_MYSQL_PASSWORD:openclaw123}"));
    assertTrue(application.contains("schema-locations: classpath:sql/MySQL/V0.0.1__init.sql"));
    assertTrue(application.contains("mode: never"));
    assertTrue(application.contains("db-type: mysql"));
    assertTrue(application.contains("vector-store-type: simple"));
    assertFalse(application.contains("connection-init-sql: PRAGMA"));
    assertFalse(application.contains("url: jdbc:sqlite:./data/openclaw.db"));
    assertFalse(application.contains("sqlite-vec"));
    assertFalse(application.contains("sqlite-vec-extension-path"));

    assertTrue(pom.contains("<artifactId>mysql-connector-j</artifactId>"));
    assertFalse(pom.contains("<artifactId>sqlite-jdbc</artifactId>"));
    assertTrue(trainingApplication.contains("com.mysql.cj.jdbc.Driver"));
    assertFalse(trainingApplication.contains("org.sqlite.JDBC"));
  }

  @Test
  void dashScopeOpenAiCompatibleEndpointsIncludeVersionPath() throws IOException {
    String[] files = {
      "src/main/resources/application.yml",
      "src/main/resources/templates/default-application.yml",
      "src/main/resources/sql/MySQL/V0.0.1__init.sql",
      "src/main/resources/sql/H2/V0.0.1__init.sql",
      "src/main/resources/sql/H2/agentscope-schema.sql",
      "src/main/resources/prompt/agent.json"
    };

    for (String file : files) {
      String content = read(file);
      assertTrue(
          content.contains("https://dashscope.aliyuncs.com/compatible-mode/v1"),
          "DashScope endpoint should include /v1 in " + file);
      assertFalse(
          content.contains("https://dashscope.aliyuncs.com/compatible-mode\""),
          "DashScope JSON endpoint is missing /v1 in " + file);
      assertFalse(
          content.contains("https://dashscope.aliyuncs.com/compatible-mode'"),
          "DashScope SQL endpoint is missing /v1 in " + file);
      assertFalse(
          content.contains("https://dashscope.aliyuncs.com/compatible-mode\n"),
          "DashScope YAML endpoint is missing /v1 in " + file);
    }
  }

  @Test
  void localMysqlComposeInitializesSingleMysql97InstanceWithApplicationSchema() throws IOException {
    String compose =
        Files.readString(PROJECT_DIR.getParent().resolve("deploy/docker-compose.middleware.yml"));

    assertTrue(compose.contains("image: mysql:9.7.1"));
    assertTrue(compose.contains("mysql:"));
    assertTrue(compose.contains("container_name: openclaw4j-mysql"));
    assertTrue(compose.contains("\"3306:3306\""));
    assertTrue(compose.contains("MYSQL_DATABASE: openclaw4j"));
    assertTrue(compose.contains("MYSQL_USER: openclaw"));
    assertTrue(compose.contains("MYSQL_PASSWORD: openclaw123"));
    assertTrue(
        compose.contains(
            "../OpenClaw4j-Bankend/src/main/resources/sql/MySQL/V0.0.1__init.sql:/docker-entrypoint-initdb.d/01-openclaw4j-init.sql:ro"));
    assertTrue(compose.contains("./data/mysql:/var/lib/mysql"));
    assertFalse(compose.contains("mysql-0-primary"));
    assertFalse(compose.contains("mysql-0-replica"));
    assertFalse(compose.contains("mysql-1-primary"));
    assertFalse(compose.contains("mysql-1-replica"));
    assertFalse(compose.contains("mysql-replication-init"));
    assertFalse(compose.contains("openclaw4j_ds_"));
    assertFalse(compose.contains("--gtid-mode=ON"));
    assertFalse(compose.contains("--binlog-format=ROW"));

    String readme = read("README.md");
    assertTrue(
        readme.contains("docker compose -f ../deploy/docker-compose.middleware.yml up -d --build"));
    assertTrue(readme.contains("spring.sql.init.mode=never"));
  }

  @Test
  void defaultBackendConfigurationUsesRedisForPersistentLoginCache() throws IOException {
    String application = read("src/main/resources/application.yml");
    String redissonConfig =
        read("src/main/java/com/seaskyland/llm/workflow/core/config/RedissonConfig.java");
    String compose =
        Files.readString(PROJECT_DIR.getParent().resolve("deploy/docker-compose.middleware.yml"));
    String readme = read("README.md");

    assertTrue(application.contains("master: ${OPENCLAW_REDIS_SENTINEL_MASTER:openclaw4j-master}"));
    assertTrue(
        application.contains(
            "nodes: ${OPENCLAW_REDIS_SENTINEL_NODES:127.0.0.1:26379,127.0.0.1:26380,127.0.0.1:26381}"));
    assertTrue(
        application.contains(
            "nat-map: ${OPENCLAW_REDIS_SENTINEL_NAT_MAP:redis-master:6379=127.0.0.1:6379,redis-replica-1:6379=127.0.0.1:6380,redis-replica-2:6379=127.0.0.1:6381}"));
    assertTrue(application.contains("database: ${OPENCLAW_REDIS_DATABASE:0}"));
    assertTrue(application.contains("type: REDIS"));
    assertTrue(application.contains("type: JVM"));
    assertFalse(application.contains("cache:\n  type: JVM"));
    assertFalse(application.contains("host: ${OPENCLAW_REDIS_HOST:127.0.0.1}"));
    assertFalse(application.contains("port: ${OPENCLAW_REDIS_PORT:6379}"));

    assertTrue(redissonConfig.contains("useSentinelServers()"));
    assertTrue(redissonConfig.contains("setMasterName(sentinelMaster)"));
    assertTrue(redissonConfig.contains("addSentinelAddress(sentinelAddresses)"));
    assertTrue(redissonConfig.contains("setDatabase(database)"));
    assertTrue(redissonConfig.contains("setCheckSentinelsList(false)"));
    assertTrue(redissonConfig.contains("setSentinelsDiscovery(false)"));
    assertTrue(redissonConfig.contains("HostPortNatMapper"));
    assertTrue(redissonConfig.contains("setNatMapper"));

    assertTrue(compose.contains("image: redis:7"));
    assertTrue(compose.contains("container_name: openclaw4j-redis-master"));
    assertTrue(compose.contains("\"6379:6379\""));
    assertTrue(compose.contains("redis-cli ping"));
    assertTrue(compose.contains("./data/redis-master:/data"));
    assertTrue(compose.contains("redis-sentinel-1"));

    assertTrue(readme.contains("Redis master 仍可通过 `127.0.0.1:6379`、database `0`"));
    assertTrue(readme.contains("openclaw4j-master"));
    assertTrue(readme.contains("cache.type=REDIS"));
  }

  @Test
  void deployMiddlewareComposeUsesSingleMysqlAndRedisSentinel() throws IOException {
    String compose =
        Files.readString(PROJECT_DIR.getParent().resolve("deploy/docker-compose.middleware.yml"));
    String sentinel =
        Files.readString(PROJECT_DIR.getParent().resolve("deploy/redis/sentinel-1.conf"));
    String redisMaster =
        Files.readString(PROJECT_DIR.getParent().resolve("deploy/redis/redis-master.conf"));
    String redisReplica1 =
        Files.readString(PROJECT_DIR.getParent().resolve("deploy/redis/redis-replica-1.conf"));
    String redisReplica2 =
        Files.readString(PROJECT_DIR.getParent().resolve("deploy/redis/redis-replica-2.conf"));

    assertTrue(compose.contains("mysql:"));
    assertTrue(compose.contains("container_name: openclaw4j-mysql"));
    assertTrue(compose.contains("\"3306:3306\""));
    assertTrue(compose.contains("./data/mysql:/var/lib/mysql"));
    assertFalse(compose.contains("shardingsphere"));
    assertFalse(compose.contains("3306:3307"));
    assertFalse(compose.contains("mysql-0-primary"));
    assertFalse(compose.contains("mysql-0-replica"));
    assertFalse(compose.contains("mysql-1-primary"));
    assertFalse(compose.contains("mysql-1-replica"));
    assertFalse(compose.contains("./shardingsphere/conf:/opt/shardingsphere-proxy/conf"));
    assertTrue(compose.contains("redis-master"));
    assertTrue(compose.contains("redis-replica-1"));
    assertTrue(compose.contains("redis-replica-2"));
    assertTrue(compose.contains("redis-sentinel-1"));
    assertTrue(compose.contains("redis-sentinel-2"));
    assertTrue(compose.contains("redis-sentinel-3"));
    assertTrue(compose.contains("26379:26379"));
    assertTrue(compose.contains("cp /usr/local/etc/redis/sentinel.conf /data/sentinel.conf"));
    assertTrue(compose.contains("./redis/redis-replica-1.conf:/usr/local/etc/redis/redis.conf:ro"));
    assertTrue(compose.contains("./redis/redis-replica-2.conf:/usr/local/etc/redis/redis.conf:ro"));
    assertTrue(sentinel.contains("sentinel resolve-hostnames yes"));
    assertTrue(sentinel.contains("sentinel announce-hostnames yes"));
    assertTrue(sentinel.contains("sentinel monitor openclaw4j-master redis-master 6379 2"));
    assertTrue(redisMaster.contains("replica-announce-ip redis-master"));
    assertTrue(redisReplica1.contains("replica-announce-ip redis-replica-1"));
    assertTrue(redisReplica2.contains("replica-announce-ip redis-replica-2"));
    assertTrue(redisReplica1.contains("appendonly no"));
    assertTrue(redisReplica2.contains("appendonly no"));
    assertTrue(redisReplica1.contains("repl-diskless-load on-empty-db"));
    assertTrue(redisReplica2.contains("repl-diskless-load on-empty-db"));
    assertFalse(redisReplica1.contains("appendonly yes"));
    assertFalse(redisReplica2.contains("appendonly yes"));
    assertTrue(compose.contains("master_link_status:up"));
    assertFalse(
        Files.exists(
            PROJECT_DIR
                .getParent()
                .resolve("deploy/shardingsphere/conf/database-openclaw4j.yaml")));
    assertFalse(compose.contains("openclaw4j-mysql-data:"));
    assertFalse(compose.contains("openclaw4j-redis-data:"));
  }

  @Test
  void deployMiddlewareComposeIncludesElasticsearchOnly() throws IOException {
    String compose =
        Files.readString(PROJECT_DIR.getParent().resolve("deploy/docker-compose.middleware.yml"));
    String logback = read("src/main/resources/logback-spring.xml");

    assertTrue(compose.contains("image: docker.elastic.co/elasticsearch/elasticsearch:9.4.3"));
    assertTrue(compose.contains("container_name: openclaw4j-elasticsearch"));
    assertTrue(compose.contains("discovery.type: single-node"));
    assertTrue(compose.contains("ELASTIC_PASSWORD: ${ELASTIC_PASSWORD:"));
    assertTrue(compose.contains("xpack.security.enabled: \"true\""));
    assertTrue(compose.contains("xpack.security.http.ssl.enabled: \"false\""));
    assertTrue(compose.contains("-XX:+UnlockExperimentalVMOptions"));
    assertTrue(compose.contains("-XX:+UseCompactObjectHeaders"));
    assertTrue(compose.contains("\"39200:9200\""));
    assertTrue(compose.contains("./data/elasticsearch:/usr/share/elasticsearch/data"));
    assertFalse(compose.contains("container_name: openclaw4j-kibana"));
    assertFalse(compose.contains("container_name: openclaw4j-logstash"));
    assertFalse(compose.contains("elasticsearch-security-init"));
    assertFalse(compose.contains("docker.elastic.co/kibana"));
    assertFalse(compose.contains("docker.elastic.co/logstash"));
    assertFalse(compose.contains("KIBANA_"));
    assertFalse(compose.contains("LOGSTASH_"));
    assertTrue(logback.contains("${APP_NAME:-OpenClaw4j-Bankend}"));
    assertTrue(logback.contains("${OPENCLAW_LOG_PATH:-${openclaw.logging.path:-"));
  }

  @Test
  void mysqlSchemaIncludesLegacyAdminCompatibilityTables() throws IOException {
    String mysqlSchema = read("src/main/resources/sql/MySQL/V0.0.1__init.sql");

    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `prompt`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `prompt_version`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `prompt_build_template`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `dataset`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `dataset_version`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `dataset_item`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `evaluator`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `evaluator_version`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `evaluator_template`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `experiment`"));
    assertTrue(mysqlSchema.contains("CREATE TABLE IF NOT EXISTS `experiment_result`"));
    assertTrue(mysqlSchema.contains("INSERT IGNORE INTO `prompt_build_template`"));
    assertTrue(mysqlSchema.contains("INSERT IGNORE INTO `evaluator_template`"));
  }

  @Test
  void adminCompatibilityWritesAvoidSqliteOnlySyntaxForMysqlBackend() throws IOException {
    String service =
        read("src/main/java/com/seaskyland/llm/workflow/admin/compat/AdminCompatService.java");

    assertFalse(service.contains("datetime('now')"));
    assertFalse(service.contains("last_insert_rowid()"));
    assertFalse(service.contains("ON CONFLICT"));
    assertFalse(service.contains("ON DUPLICATE KEY"));
    assertTrue(service.contains("CURRENT_TIMESTAMP"));
    assertTrue(service.contains("GeneratedKeyHolder"));
  }

  @Test
  void sqliteVectorStoreSupportIsRemovedFromRuntime() throws IOException {
    String vectorStoreType =
        read("src/main/java/com/seaskyland/llm/workflow/core/rag/vectorstore/VectorStoreType.java");
    String vectorStoreFactory =
        read(
            "src/main/java/com/seaskyland/llm/workflow/core/rag/vectorstore/VectorStoreFactory.java");
    String studioProperties =
        read("src/main/java/com/seaskyland/llm/workflow/core/config/StudioProperties.java");

    assertFalse(
        Files.exists(
            PROJECT_DIR.resolve(
                "src/main/java/com/seaskyland/llm/workflow/core/rag/vectorstore/sqlite/SqliteVectorStoreService.java")));
    assertFalse(
        Files.exists(
            PROJECT_DIR.resolve(
                "src/main/java/com/seaskyland/llm/workflow/core/rag/vectorstore/sqlite/SqliteVectorStore.java")));
    assertFalse(Files.exists(PROJECT_DIR.resolve("src/main/resources/sqlite/vec0.dll")));
    assertFalse(
        Files.exists(PROJECT_DIR.resolve("src/main/resources/sql/Sqlite/V0.0.1__init.sql")));
    assertFalse(vectorStoreType.contains("SQLITE"));
    assertFalse(vectorStoreType.contains("sqlite"));
    assertFalse(vectorStoreFactory.contains("sqliteVectorStoreService"));
    assertFalse(studioProperties.contains("sqliteVecExtensionPath"));
  }

  @Test
  void jdkFinalFieldMutationWarningIsSuppressedForMybatisRuntimePaths() throws IOException {
    String pom = read("pom.xml");
    String trainScript = read("scripts/train-leyden-aot.ps1");
    String runScript = read("scripts/run-leyden-aot.ps1");
    String benchmarkScript = read("scripts/benchmark-leyden-startup.ps1");
    String dockerfile = read("Dockerfile");
    String mavenJvmConfig = read(".mvn/jvm.config");

    String flag = "--enable-final-field-mutation=ALL-UNNAMED";
    assertTrue(
        pom.contains(
            "<mybatis.final.field.mutation.jvmArg>"
                + flag
                + "</mybatis.final.field.mutation.jvmArg>"));
    assertTrue(pom.contains("<artifactId>maven-surefire-plugin</artifactId>"));
    assertTrue(pom.contains("<argLine>${mybatis.final.field.mutation.jvmArg}</argLine>"));
    assertTrue(pom.contains("<jvmArguments>${mybatis.final.field.mutation.jvmArg}</jvmArguments>"));
    assertTrue(trainScript.contains("$JavaExe = \"D:\\jdk-26\\bin\\java.exe\""));
    assertTrue(runScript.contains("$JavaExe = \"D:\\jdk-26\\bin\\java.exe\""));
    assertTrue(benchmarkScript.contains("$JavaExe = \"D:\\jdk-26\\bin\\java.exe\""));
    assertTrue(trainScript.contains(flag));
    assertTrue(runScript.contains(flag));
    assertTrue(benchmarkScript.contains(flag));
    assertTrue(dockerfile.contains("ENV JAVA_OPTS=\"" + flag + "\""));
    assertTrue(dockerfile.contains(flag + " \\"));
    assertTrue(mavenJvmConfig.contains(flag));
  }

  @Test
  void localLeydenScriptsUseSameCompactObjectHeaderOptionsAsDockerRuntime() throws IOException {
    String trainScript = read("scripts/train-leyden-aot.ps1");
    String runScript = read("scripts/run-leyden-aot.ps1");
    String benchmarkScript = read("scripts/benchmark-leyden-startup.ps1");
    String dockerfile = read("Dockerfile");

    String compactHeaderOptions =
        "[string]$LeydenJvmOptions = \"-XX:+UnlockExperimentalVMOptions -XX:+UseCompactObjectHeaders\"";
    assertTrue(trainScript.contains(compactHeaderOptions));
    assertTrue(runScript.contains(compactHeaderOptions));
    assertTrue(benchmarkScript.contains(compactHeaderOptions));
    assertTrue(trainScript.contains("Get-LeydenJvmOptions"));
    assertTrue(runScript.contains("Get-LeydenJvmOptions"));
    assertTrue(benchmarkScript.contains("Get-LeydenJvmOptions"));
    assertTrue(
        trainScript.contains("$javaArgs = @(Get-LeydenJvmOptions -Options $LeydenJvmOptions)"));
    assertTrue(
        runScript.contains("$javaArgs = @(Get-LeydenJvmOptions -Options $LeydenJvmOptions)"));
    assertTrue(
        benchmarkScript.contains("$javaArgs = @(\"--enable-final-field-mutation=ALL-UNNAMED\")"));
    assertTrue(benchmarkScript.contains("if ($UseLeyden)"));
    assertTrue(
        benchmarkScript.contains("$javaArgs += Get-LeydenJvmOptions -Options $LeydenJvmOptions"));
    assertTrue(
        dockerfile.contains(
            "OPENCLAW4J_LEYDEN_OPTS=\"-XX:+UnlockExperimentalVMOptions -XX:+UseCompactObjectHeaders\""));
  }

  @Test
  void mavenBuildKeepsGraalVmNativeImageInExperimentProfileOnly() throws IOException {
    String pom = read("pom.xml");

    assertTrue(pom.contains("<id>leyden</id>"));
    assertTrue(pom.contains("<id>native-experiment</id>"));
    assertTrue(pom.contains("org.graalvm.buildtools"));
    assertTrue(pom.contains("native-maven-plugin"));
    assertTrue(pom.contains("<skipNativeBuild>true</skipNativeBuild>"));

    String leydenProfile = profileBlock(pom, "leyden");
    String nativeProfile = profileBlock(pom, "native-experiment");
    assertFalse(leydenProfile.contains("native-maven-plugin"));
    assertFalse(leydenProfile.contains("org.graalvm.buildtools"));
    assertTrue(nativeProfile.contains("native-maven-plugin"));
    assertTrue(nativeProfile.contains("<imageName>OpenClaw4j-Bankend-native</imageName>"));
    assertTrue(nativeProfile.contains("maven-compiler-plugin"));
    assertTrue(nativeProfile.contains("annotationProcessorPaths"));
    assertTrue(nativeProfile.contains("org.projectlombok"));
  }

  @Test
  void leydenTrainingAndRuntimeScriptsUseHotSpotAotCache() throws IOException {
    String trainScript = read("scripts/train-leyden-aot.ps1");
    String runScript = read("scripts/run-leyden-aot.ps1");

    assertTrue(trainScript.contains("-XX:AOTCacheOutput="));
    assertTrue(trainScript.contains("-Djarmode=tools"));
    assertTrue(trainScript.contains("TrainingTimeoutSeconds"));
    assertTrue(trainScript.contains("TrainingTimeoutSeconds = 900"));
    assertTrue(trainScript.contains("Start-Job"));
    assertTrue(trainScript.contains("profiled"));
    assertTrue(trainScript.contains("WarmupProfile"));
    assertTrue(trainScript.contains("WarmupRequestTimeoutSeconds"));
    assertTrue(trainScript.contains("representative"));
    assertTrue(trainScript.contains("openclaw4j.leyden.training.profiled=true"));
    assertTrue(trainScript.contains("openclaw4j.leyden.training.request-timeout="));
    assertTrue(trainScript.contains("LeydenTrainingApplication"));
    assertTrue(trainScript.contains("Get-LeydenClasspath"));
    assertTrue(trainScript.contains("$appJar"));
    assertTrue(runScript.contains("-XX:AOTCache="));
    assertTrue(runScript.contains("Get-LeydenClasspath"));
    assertTrue(runScript.contains("LLMApplication"));
    assertTrue(runScript.contains("$appJar"));
    assertFalse(trainScript.contains("native-image"));
    assertFalse(runScript.contains("native-image"));
  }

  @Test
  void dockerfileBuildsAndRunsExtractedApplicationWithAotCache() throws IOException {
    String dockerfile = read("Dockerfile");

    assertTrue(dockerfile.contains("maven:3.9-eclipse-temurin-26"));
    assertTrue(dockerfile.contains("eclipse-temurin:26-jre"));
    assertTrue(dockerfile.contains("cp app.jar runtime/OpenClaw4j-Bankend.jar"));
    assertTrue(dockerfile.contains("WORKDIR /app/runtime"));
    assertTrue(dockerfile.contains("leyden-training.log"));
    assertTrue(dockerfile.contains("-XX:+UnlockExperimentalVMOptions"));
    assertTrue(dockerfile.contains("-XX:+UseCompactObjectHeaders"));
    assertTrue(dockerfile.contains("-XX:AOTCacheOutput=openclaw4j.aot"));
    assertTrue(dockerfile.contains("-XX:AOTCache=openclaw4j.aot"));
    assertTrue(dockerfile.contains("test -f openclaw4j.aot"));
    assertTrue(dockerfile.contains("ls -lh openclaw4j.aot"));
    assertTrue(dockerfile.contains("openclaw4j.leyden.training.profiled=true"));
    assertTrue(dockerfile.contains("-jar OpenClaw4j-Bankend.jar"));
    assertTrue(dockerfile.contains("JAVA_OPTS"));
    assertFalse(dockerfile.contains("classpath.txt"));
    assertFalse(dockerfile.contains("jar xf"));
    assertFalse(dockerfile.contains("META-INF/MANIFEST.MF"));
    assertFalse(dockerfile.contains("native-image"));
    assertFalse(dockerfile.contains("graalvm"));
    assertFalse(dockerfile.contains("UseZGC"));
    assertFalse(dockerfile.contains("UseG1GC"));
  }

  @Test
  void profiledTrainingRunnerWarmsLocalEndpointsAndExitsNormally() throws IOException {
    String runner = read("src/main/java/com/seaskyland/llm/LeydenProfiledTrainingRunner.java");
    String benchmarkScript = read("scripts/benchmark-leyden-startup.ps1");

    assertTrue(runner.contains("@ConditionalOnProperty"));
    assertTrue(runner.contains("ApplicationReadyEvent"));
    assertTrue(runner.contains("local.server.port"));
    assertTrue(runner.contains("/console/v1/system/health"));
    assertTrue(runner.contains("openclaw4j.leyden.training.request-timeout"));
    assertTrue(runner.contains("WarmupResult"));
    assertTrue(runner.contains("Leyden profiled warmup summary"));
    assertTrue(runner.contains("SpringApplication.exit"));
    assertTrue(benchmarkScript.contains("Started LLMApplication in"));
    assertTrue(benchmarkScript.contains("ProbePaths"));
    assertTrue(benchmarkScript.contains("firstRequestMilliseconds"));
    assertTrue(benchmarkScript.contains("Group-Object leyden"));
    assertTrue(benchmarkScript.contains("Refusing to clean extract directory outside target"));
    assertTrue(benchmarkScript.contains("Remove-Item -LiteralPath $extractDir -Recurse -Force"));
  }

  @Test
  void githubActionsBuildsAndPublishesOptimizedLeydenDockerImageOnPush() throws IOException {
    String workflow =
        Files.readString(PROJECT_DIR.getParent().resolve(".github/workflows/backend-native.yml"));

    assertTrue(workflow.contains("packages: write"));
    assertTrue(workflow.contains("docker/login-action@v3"));
    assertTrue(workflow.contains("docker/metadata-action@v5"));
    assertTrue(workflow.contains("docker/build-push-action@v6"));
    assertTrue(workflow.contains("context: ./OpenClaw4j-Bankend"));
    assertTrue(workflow.contains("push: ${{ github.event_name != 'pull_request' }}"));
    assertTrue(workflow.contains("ghcr.io/dexter-huang/openclaw4j-backend"));
    assertTrue(workflow.contains("OPENCLAW4J_IMAGE_DIGEST"));
    assertFalse(workflow.contains("graalvm/setup-graalvm"));
    assertFalse(workflow.contains("native:compile"));
    assertFalse(workflow.contains("OpenClaw4j-Bankend-native"));
  }

  private String read(String relativePath) throws IOException {
    return Files.readString(PROJECT_DIR.resolve(relativePath));
  }

  private String profileBlock(String pom, String profileId) {
    String marker = "<id>" + profileId + "</id>";
    int idIndex = pom.indexOf(marker);
    assertTrue(idIndex >= 0, "Missing Maven profile: " + profileId);

    int start = pom.lastIndexOf("<profile>", idIndex);
    int end = pom.indexOf("</profile>", idIndex);
    assertTrue(start >= 0 && end > idIndex, "Invalid Maven profile block: " + profileId);
    return pom.substring(start, end + "</profile>".length());
  }
}
