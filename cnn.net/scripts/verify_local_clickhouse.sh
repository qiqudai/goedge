#!/usr/bin/env bash
set -euo pipefail

CLICKHOUSE_BIN="${CLICKHOUSE_BIN:-$(command -v clickhouse || true)}"
TCP_PORT="${CLICKHOUSE_TCP_PORT:-9000}"
DB_NAME="${CLICKHOUSE_DB:-default}"

if [[ -z "$CLICKHOUSE_BIN" ]]; then
  echo "clickhouse binary not found in PATH"
  exit 1
fi

echo "[1/3] Version"
"$CLICKHOUSE_BIN" client --host 127.0.0.1 --port "$TCP_PORT" --query "SELECT version()"

echo "[2/3] Tables"
"$CLICKHOUSE_BIN" client --host 127.0.0.1 --port "$TCP_PORT" --query "SHOW TABLES FROM ${DB_NAME}"

echo "[3/3] Health"
"$CLICKHOUSE_BIN" client --host 127.0.0.1 --port "$TCP_PORT" --query "SELECT 1"
