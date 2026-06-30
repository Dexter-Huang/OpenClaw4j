# Middleware Topology Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local deploy middleware stack with bind-mounted data, ShardingSphere-backed MySQL, and Redis Sentinel.

**Architecture:** `deploy/docker-compose.middleware.yml` defines the runtime topology. Versioned ShardingSphere and Redis config live under `deploy/shardingsphere/` and `deploy/redis/`; mutable data is bind-mounted under ignored `deploy/data/`.

**Tech Stack:** Docker Compose, MySQL 8.0, Apache ShardingSphere Proxy, Redis 7, Spring Boot Maven configuration tests.

---

### Task 1: Configuration Shape Test

**Files:**
- Modify: `OpenClaw4j-Bankend/src/test/java/com/seaskyland/llm/build/LeydenBuildConfigurationTest.java`

- [ ] **Step 1: Add a failing test for the deploy topology**

Add a JUnit test that reads `../deploy/docker-compose.middleware.yml` and asserts that it contains ShardingSphere Proxy, two MySQL data nodes, bind-mounted `./data` paths, and Redis Sentinel services.

- [ ] **Step 2: Run targeted Maven test and verify failure**

Run:

```powershell
cd OpenClaw4j-Bankend
$env:JAVA_HOME='D:\jdk-26'
$env:Path='D:\jdk-26\bin;' + $env:Path
mvn '-Dmaven.repo.local=D:\apache-maven-3.9.1\m2\repository' '-Dtest=LeydenBuildConfigurationTest#deployMiddlewareComposeUsesShardingSphereAndRedisSentinel' test
```

Expected: the new test fails because the compose file still has a single `mysql` and single `redis` service.

### Task 2: Compose And Middleware Config

**Files:**
- Modify: `deploy/docker-compose.middleware.yml`
- Create: `deploy/shardingsphere/Dockerfile`
- Create: `deploy/shardingsphere/conf/global.yaml`
- Create: `deploy/shardingsphere/conf/database-openclaw4j.yaml`
- Create: `deploy/redis/redis-master.conf`
- Create: `deploy/redis/redis-replica.conf`
- Create: `deploy/redis/sentinel-1.conf`
- Create: `deploy/redis/sentinel-2.conf`
- Create: `deploy/redis/sentinel-3.conf`

- [ ] **Step 1: Replace named Docker volumes with bind mounts**

Use `./data/mysql-0`, `./data/mysql-1`, `./data/redis-master`, `./data/redis-replica-*`, and `./data/shardingsphere` paths.

- [ ] **Step 2: Add MySQL data nodes and ShardingSphere Proxy**

Define `mysql-0`, `mysql-1`, and `shardingsphere-proxy`. Publish proxy `3306:3307`, keep physical MySQL ports on `33061` and `33062`, and mount the backend SQL init file into both MySQL containers.

- [ ] **Step 3: Add Redis Sentinel topology**

Define `redis-master`, `redis-replica-1`, `redis-replica-2`, and three Sentinel services with master name `openclaw4j-master`.

- [ ] **Step 4: Add ShardingSphere config files**

Configure two MySQL data sources, default route `ds_0`, and a conservative example sharding rule for the `workspace` table using `workspace_id`.

- [ ] **Step 5: Add Redis config files**

Enable appendonly persistence for Redis data nodes and configure each Sentinel to monitor `redis-master 6379` with quorum `2`.

### Task 3: Docs And Verification

**Files:**
- Modify: `OpenClaw4j-Bankend/README.md`

- [ ] **Step 1: Update local middleware docs**

Point developers to `docker compose -f deploy/docker-compose.middleware.yml up -d --build`, document ShardingSphere Proxy on `3306`, and document Redis Sentinel on `26379`.

- [ ] **Step 2: Run targeted Maven test**

Run the same targeted `LeydenBuildConfigurationTest` method and expect `BUILD SUCCESS`.

- [ ] **Step 3: Validate compose syntax**

Run:

```powershell
docker compose -f deploy/docker-compose.middleware.yml config
```

Expected: Docker Compose renders the merged configuration successfully.
