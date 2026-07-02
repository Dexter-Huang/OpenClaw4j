# ELK 本地观测栈设计

## 目标

在现有 `deploy/docker-compose.middleware.yml` 中接入本地 ELK 观测能力，使用 Elasticsearch 9.4.3 存储 OpenClaw4j 后端日志，并通过 Kibana 查看日志。当前范围只覆盖本地开发与联调，不引入生产安全、链路追踪或业务搜索改造。

## 架构

本地中间件栈新增 3 个服务：

```text
OpenClaw4j 后端日志目录
  -> Logstash 9.4.3
  -> Elasticsearch 9.4.3
  -> Kibana 9.4.3
```

Elasticsearch 以单节点开发模式运行，关闭内置安全能力，暴露 `127.0.0.1:9200`。Kibana 暴露 `127.0.0.1:5601`，连接 compose 网络内的 `http://elasticsearch:9200`。Logstash 挂载 `deploy/logstash/pipeline` 和 `deploy/data/logs/openclaw4j`，读取后端滚动日志文件并写入 Elasticsearch。

## 日志采集

后端现有 `logback-spring.xml` 已经将应用日志、监控日志和 trace 日志落盘。本次只将日志目录改成可配置：

- 默认路径仍然是 `${user.home}/${APP_NAME}/logs`。
- 本地 ELK 联调时可设置 `OPENCLAW_LOG_PATH`，例如 `deploy/data/logs/openclaw4j`。

Logstash 使用 `file` input 读取 `*.log`，使用 multiline codec 合并 Java 堆栈，输出到 `openclaw4j-logs-%{+YYYY.MM.dd}` 索引。

## 边界

- 不修改业务搜索、向量存储或 Spring AI 的 Elasticsearch 配置。
- 不接入 OpenTelemetry tracing。
- 不为本地开发栈配置 ES 用户、证书和密码。
- 不采集 MySQL、Redis、ShardingSphere 容器日志；后续需要时可单独增加采集规则。

## 验证

- `docker compose -f deploy/docker-compose.middleware.yml config`
- `git diff --check`
- 若修改后端运行时配置，按质量门禁执行必要的 Maven 格式或质量检查；若仅 XML/YAML 配置，则报告未运行 Java 编译检查的原因。
