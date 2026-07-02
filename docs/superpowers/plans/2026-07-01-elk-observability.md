# ELK Observability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在本地 Docker Compose 中加入 Elasticsearch 9.4.3、Kibana 和 Logstash，并让后端日志可被采集到 Elasticsearch。

**Architecture:** Compose 管理 ELK 服务，Logstash 从 `deploy/data/logs/openclaw4j` 读取后端日志并写入 Elasticsearch。后端 Logback 只开放日志目录配置，不改变已有 logger 和 appender 结构。

**Tech Stack:** Docker Compose, Elasticsearch 9.4.3, Kibana 9.4.3, Logstash 9.4.3, Logback

## Global Constraints

- `D:\IDEA_project\OpenClaw4j` 是唯一 Git 仓库根目录。
- 文档使用中文；配置键、镜像名和命令保持原拼写。
- 不跟踪 `deploy/data/` 这类运行时生成目录。
- 本地 ELK 开发栈关闭 Elasticsearch 安全能力；生产安全配置不在本次范围内。

---

### Task 1: Compose ELK 服务

**Files:**
- Modify: `deploy/docker-compose.middleware.yml`

**Interfaces:**
- Consumes: 现有 compose 默认网络。
- Produces: `elasticsearch`、`kibana`、`logstash` 服务；ES HTTP 端口 `9200`，Kibana 端口 `5601`。

- [x] **Step 1: 新增 Elasticsearch 9.4.3 服务**

使用 `docker.elastic.co/elasticsearch/elasticsearch:9.4.3`，配置单节点开发模式、关闭安全、挂载 `./data/elasticsearch:/usr/share/elasticsearch/data`。

- [x] **Step 2: 新增 Kibana 9.4.3 服务**

使用 `docker.elastic.co/kibana/kibana:9.4.3`，依赖 Elasticsearch 健康后启动，配置 `ELASTICSEARCH_HOSTS=http://elasticsearch:9200`。

- [x] **Step 3: 新增 Logstash 9.4.3 服务**

使用 `docker.elastic.co/logstash/logstash:9.4.3`，挂载 pipeline 和日志目录，依赖 Elasticsearch 健康后启动。

### Task 2: Logstash Pipeline

**Files:**
- Create: `deploy/logstash/pipeline/openclaw4j-logs.conf`

**Interfaces:**
- Consumes: `/var/log/openclaw4j/**/*.log`
- Produces: Elasticsearch index `openclaw4j-logs-%{+YYYY.MM.dd}`

- [x] **Step 1: 配置 file input**

读取应用日志、监控日志和 trace 日志，使用 multiline codec 合并 Java 异常堆栈。

- [x] **Step 2: 配置 grok filter**

解析时间、线程、日志级别、logger 和消息，并补充 `service.name=openclaw4j-backend`。

- [x] **Step 3: 配置 Elasticsearch output**

输出到 `http://elasticsearch:9200`，索引按日期滚动。

### Task 3: 后端日志路径可配置

**Files:**
- Modify: `OpenClaw4j-Bankend/src/main/resources/logback-spring.xml`

**Interfaces:**
- Consumes: `OPENCLAW_LOG_PATH` 环境变量。
- Produces: 可被 Logstash 挂载采集的日志目录。

- [x] **Step 1: 给 APP_NAME 增加默认值**

默认值为 `OpenClaw4j-Bankend`，避免未设置 `APP_NAME` 时路径不稳定。

- [x] **Step 2: 给 APP_LOG_PATH 增加环境变量覆盖**

`OPENCLAW_LOG_PATH` 存在时优先使用，否则保持 `${user.home}/${APP_NAME}/logs`。

### Task 4: 验证

**Files:**
- Verify: `deploy/docker-compose.middleware.yml`
- Verify: `deploy/logstash/pipeline/openclaw4j-logs.conf`
- Verify: `OpenClaw4j-Bankend/src/main/resources/logback-spring.xml`

- [ ] **Step 1: 校验 compose 配置**

Run: `docker compose -f deploy/docker-compose.middleware.yml config`
Expected: 输出规范化 compose 配置且退出码为 0。

- [ ] **Step 2: 校验 diff 空白错误**

Run: `git diff --check`
Expected: 无 trailing whitespace 或冲突标记。

- [ ] **Step 3: 汇总质量门禁**

报告已运行的命令、退出码和未运行 Maven/前端构建的原因。
