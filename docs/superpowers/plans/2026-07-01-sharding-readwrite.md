# 分库分表与读写分离 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将本地中间件环境改造成 2 分片、每分片 1 主 1 从的 ShardingSphere Proxy 拓扑。

**Architecture:** Docker Compose 管理 4 个 MySQL 9.7.1 容器和一个一次性复制初始化容器。ShardingSphere Proxy 暴露 MySQL 协议入口，组合 `READWRITE_SPLITTING`、`SHARDING` 和 `SINGLE` 规则完成读写分离与分片路由。

**Tech Stack:** Docker Compose, MySQL 9.7.1, ShardingSphere Proxy 5.5.2, shell, PowerShell

## Global Constraints

- Git 根目录固定为 `D:\IDEA_project\OpenClaw4j`。
- 项目文档默认使用中文。
- 生成数据目录 `deploy/data/**` 不进入 Git。
- MySQL 当前按从 0 开发阶段处理，允许清理旧数据目录。
- Redis 拓扑和数据不在本次变更范围内。

---

### Task 1: MySQL 复制拓扑

**Files:**
- Modify: `deploy/docker-compose.middleware.yml`
- Create: `deploy/mysql/setup-replication.sh`

**Interfaces:**
- Produces: `mysql-0-primary`, `mysql-0-replica`, `mysql-1-primary`, `mysql-1-replica`, `mysql-replication-init`

- [x] **Step 1: 新增四个 MySQL 9.7.1 服务**

保留 `33061` 和 `33062` 给两个 primary，新增 `33063` 和 `33064` 给两个 replica。

- [x] **Step 2: 新增复制初始化脚本**

脚本等待四个 MySQL 健康，创建复制用户，导出 primary 数据，导入 replica，并启动复制。

- [x] **Step 3: 验证复制状态**

Run: `docker exec openclaw4j-mysql-0-replica mysql -uroot -proot123456 -Nse "SELECT SERVICE_STATE FROM performance_schema.replication_connection_status; SELECT SERVICE_STATE FROM performance_schema.replication_applier_status; SELECT @@read_only, @@super_read_only;"`

Expected: 两个 `ON`，并且 `read_only/super_read_only` 为 `1 1`。

### Task 2: ShardingSphere 规则

**Files:**
- Modify: `deploy/shardingsphere/conf/database-openclaw4j.yaml`

**Interfaces:**
- Consumes: Task 1 的四个 MySQL 服务名
- Produces: 逻辑数据源 `ds_0`、`ds_1`

- [x] **Step 1: 定义四个物理数据源**

`ds_0_primary`、`ds_0_replica`、`ds_1_primary`、`ds_1_replica` 分别指向四个 MySQL 容器。

- [x] **Step 2: 增加 `READWRITE_SPLITTING`**

`ds_0` 由 `ds_0_primary` 写和 `ds_0_replica` 读组成，`ds_1` 同理。

- [x] **Step 3: 保留原有分片规则**

`workspace` 继续使用 `actualDataNodes: ds_${0..1}.workspace` 和 `workspace_id` 哈希。

### Task 3: 部署与验证

**Files:**
- Runtime only: `deploy/data/**`

- [x] **Step 1: 清理旧 MySQL 数据目录**

只删除 `deploy/data/mysql-0*` 和 `deploy/data/mysql-1*`，保留 Redis 数据。

- [x] **Step 2: 启动拓扑**

Run: `docker compose -f deploy/docker-compose.middleware.yml up -d mysql-0-primary mysql-0-replica mysql-1-primary mysql-1-replica mysql-replication-init shardingsphere-proxy`

- [x] **Step 3: 验证读写分离**

通过 Proxy 插入测试账号时日志显示 `ds_0_primary`，查询测试账号时日志显示 `ds_0_replica`。

- [x] **Step 4: 验证分片与读写分离组合**

通过 Proxy 插入两个不同 `workspace_id`，日志显示分别写入 `ds_1_primary` 和 `ds_0_primary`，查询时走 `ds_0_replica` 和 `ds_1_replica`。
