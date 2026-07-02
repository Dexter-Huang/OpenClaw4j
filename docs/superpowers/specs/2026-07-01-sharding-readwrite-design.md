# 分库分表与读写分离设计

## 目标

在本地中间件环境中一次性搭建分库分表与读写分离能力。应用仍只连接 ShardingSphere Proxy 的 MySQL 协议入口 `127.0.0.1:3306`，由 Proxy 负责路由到后端 MySQL 分片和主从节点。

## 拓扑

采用 2 个分片，每个分片 1 主 1 从，共 4 个 MySQL 9.7.1 容器：

```text
ShardingSphere Proxy
  ├─ ds_0
  │  ├─ ds_0_primary -> mysql-0-primary
  │  └─ ds_0_replica -> mysql-0-replica
  └─ ds_1
     ├─ ds_1_primary -> mysql-1-primary
     └─ ds_1_replica -> mysql-1-replica
```

对外端口：

- `3306`: ShardingSphere Proxy
- `33061`: `mysql-0-primary`
- `33062`: `mysql-1-primary`
- `33063`: `mysql-0-replica`
- `33064`: `mysql-1-replica`

## 数据路由

- `workspace` 表继续按 `workspace_id` 哈希分片到 `ds_0` 或 `ds_1`。
- 其它单表默认落在 `ds_0`。
- `SELECT` 默认路由到对应分片的 replica。
- `INSERT`、`UPDATE`、`DELETE` 路由到对应分片的 primary。

## 复制策略

MySQL 使用 binlog row 格式和 GTID 基础配置。`mysql-replication-init` 是一次性初始化容器，会等待四个 MySQL 节点健康后：

1. 在 primary 创建复制用户。
2. 从 primary 导出对应分片数据库。
3. 导入 replica。
4. 配置 `CHANGE REPLICATION SOURCE TO`。
5. 启动复制，并将 replica 设置为 `read_only` 和 `super_read_only`。

## 约束

- 当前是从 0 开发阶段，MySQL 数据目录可以清理重建。
- Redis 数据不属于本次变更范围。
- ShardingSphere Proxy 继续使用官方 5.5.2 镜像派生镜像，运行 JDK 21。
- 不引入 MySQL 自动故障转移；当前读写分离只解决读写路由和基础复制。
