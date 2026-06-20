#!/usr/bin/env bash
set -euo pipefail

CLICKHOUSE_BIN="${CLICKHOUSE_BIN:-$(command -v clickhouse || true)}"
TCP_PORT="${CLICKHOUSE_TCP_PORT:-9000}"
DB_NAME="${CLICKHOUSE_DB:-default}"

if [[ -z "$CLICKHOUSE_BIN" ]]; then
  echo "clickhouse binary not found in PATH"
  exit 1
fi

"$CLICKHOUSE_BIN" client --host 127.0.0.1 --port "$TCP_PORT" --query "
CREATE DATABASE IF NOT EXISTS ${DB_NAME};

CREATE TABLE IF NOT EXISTS ${DB_NAME}.node_access_logs (
  ts DateTime,
  node_id String,
  node_ip String,
  remote_addr String,
  client_country String,
  client_province String,
  client_city String,
  client_isp String,
  site_name String,
  host String,
  method String,
  uri String,
  status UInt16,
  bytes UInt64,
  request_time Float64,
  upstream_addr String,
  upstream_response_time Float64,
  upstream_cache_status String,
  http_referer String,
  http_user_agent String,
  scheme String,
  ssl_protocol String,
  ssl_cipher String,
  raw String
) ENGINE = MergeTree
PARTITION BY toDate(ts)
ORDER BY (host, node_id, ts);

CREATE TABLE IF NOT EXISTS ${DB_NAME}.node_stream_logs (
  ts DateTime,
  node_id String,
  node_ip String,
  remote_addr String,
  server_port UInt16,
  protocol String,
  status UInt16,
  bytes_sent UInt64,
  bytes_received UInt64,
  session_time Float64,
  upstream_addr String,
  upstream_bytes_sent UInt64,
  upstream_bytes_received UInt64,
  upstream_connect_time Float64,
  upstream_session_time Float64,
  raw String
) ENGINE = MergeTree
PARTITION BY toDate(ts)
ORDER BY (server_port, node_id, ts);

CREATE TABLE IF NOT EXISTS ${DB_NAME}.node_metrics (
  ts DateTime,
  node_id String,
  node_ip String,
  metric String,
  labels String,
  value Float64
) ENGINE = MergeTree
PARTITION BY toDate(ts)
ORDER BY (metric, node_id, ts);

CREATE TABLE IF NOT EXISTS ${DB_NAME}.node_events (
  ts DateTime,
  node_id String,
  node_ip String,
  event_type String,
  payload String
) ENGINE = MergeTree
PARTITION BY toDate(ts)
ORDER BY (event_type, node_id, ts);
"

echo "ClickHouse schema initialized in database: ${DB_NAME}"
