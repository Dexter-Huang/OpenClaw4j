# Middleware Topology Design

## Goal

Provide a local middleware stack under `deploy/` that stores runtime data in `deploy/data/`, exposes MySQL through ShardingSphere Proxy, and runs Redis in Sentinel mode.

## Architecture

`deploy/docker-compose.middleware.yml` owns the local topology. MySQL is split into two backend data nodes, `mysql-0` and `mysql-1`, whose data directories are bind-mounted below `deploy/data/mysql-*`. ShardingSphere Proxy exposes `127.0.0.1:3306` so the backend can keep using its existing JDBC default.

Redis runs as one master, two replicas, and three Sentinel containers. The master remains reachable on `6379` for simple local clients, while Sentinel is exposed on `26379` with master name `openclaw4j-master`.

## Sharding Scope

The first implementation uses ShardingSphere Proxy as the compatibility boundary and keeps most tables on the default data source. A small `workspace` sharding rule demonstrates the intended pattern with `workspace_id` as the sharding key. Broader table sharding should be added table-by-table after validating query patterns and key relationships.

## Data And Configuration

Runtime data stays under ignored `deploy/data/` paths. Versioned configuration lives under:

- `deploy/shardingsphere/conf/`
- `deploy/shardingsphere/Dockerfile`
- `deploy/redis/`

The ShardingSphere image is built locally so MySQL Connector/J is present in `ext-lib`.

## Testing

Backend configuration-shape tests assert that the deploy compose uses bind-mounted `deploy/data` paths, includes ShardingSphere Proxy, defines two MySQL shards, and defines Redis Sentinel services.
