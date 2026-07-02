#!/bin/sh
set -eu

ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root123456}"
REPLICATION_USER="${MYSQL_REPLICATION_USER:-openclaw_repl}"
REPLICATION_PASSWORD="${MYSQL_REPLICATION_PASSWORD:-openclaw_repl123}"

wait_mysql() {
  host="$1"
  until MYSQL_PWD="$ROOT_PASSWORD" mysqladmin ping -h "$host" -uroot --silent; do
    echo "Waiting for MySQL at $host..."
    sleep 2
  done
}

mysql_root() {
  host="$1"
  shift
  MYSQL_PWD="$ROOT_PASSWORD" mysql -h "$host" -uroot "$@"
}

configure_pair() {
  shard="$1"
  primary_host="$2"
  replica_host="$3"
  database_name="$4"

  echo "Configuring replication for shard $shard: $primary_host -> $replica_host"
  wait_mysql "$primary_host"
  wait_mysql "$replica_host"

  mysql_root "$primary_host" <<SQL
CREATE USER IF NOT EXISTS '$REPLICATION_USER'@'%' IDENTIFIED BY '$REPLICATION_PASSWORD';
GRANT REPLICATION SLAVE ON *.* TO '$REPLICATION_USER'@'%';
FLUSH PRIVILEGES;
SQL

  status="$(mysql_root "$primary_host" -Nse 'SHOW BINARY LOG STATUS')"
  source_log_file="$(echo "$status" | awk '{print $1}')"
  source_log_pos="$(echo "$status" | awk '{print $2}')"

  dump_file="/tmp/${database_name}.sql"
  MYSQL_PWD="$ROOT_PASSWORD" mysqldump \
    -h "$primary_host" \
    -uroot \
    --single-transaction \
    --routines \
    --events \
    --triggers \
    --hex-blob \
    --set-gtid-purged=OFF \
    "$database_name" > "$dump_file"

  mysql_root "$replica_host" <<SQL
STOP REPLICA;
RESET REPLICA ALL;
SET GLOBAL super_read_only = OFF;
SET GLOBAL read_only = OFF;
DROP DATABASE IF EXISTS $database_name;
CREATE DATABASE $database_name DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
SQL

  MYSQL_PWD="$ROOT_PASSWORD" mysql -h "$replica_host" -uroot "$database_name" < "$dump_file"

  mysql_root "$replica_host" <<SQL
CHANGE REPLICATION SOURCE TO
  SOURCE_HOST = '$primary_host',
  SOURCE_PORT = 3306,
  SOURCE_USER = '$REPLICATION_USER',
  SOURCE_PASSWORD = '$REPLICATION_PASSWORD',
  SOURCE_LOG_FILE = '$source_log_file',
  SOURCE_LOG_POS = $source_log_pos,
  GET_SOURCE_PUBLIC_KEY = 1;
START REPLICA;
SET GLOBAL read_only = ON;
SET GLOBAL super_read_only = ON;
SQL

  rm -f "$dump_file"
  mysql_root "$replica_host" -e 'SHOW REPLICA STATUS'
}

configure_pair 0 mysql-0-primary mysql-0-replica openclaw4j_ds_0
configure_pair 1 mysql-1-primary mysql-1-replica openclaw4j_ds_1
