#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
DATA_DIR="$RUNTIME_DIR/clickhouse-data"
LOG_DIR="$RUNTIME_DIR/clickhouse-logs"
STATUS_FILE="$DATA_DIR/status"
PID_FILE="$RUNTIME_DIR/clickhouse.pid"

HTTP_PORT="${CLICKHOUSE_HTTP_PORT:-8123}"
TCP_PORT="${CLICKHOUSE_TCP_PORT:-9000}"
CLICKHOUSE_BIN="${CLICKHOUSE_BIN:-$(command -v clickhouse || true)}"

mkdir -p "$DATA_DIR" "$LOG_DIR"

if [[ -z "$CLICKHOUSE_BIN" ]]; then
  echo "clickhouse binary not found in PATH"
  exit 1
fi

if "$CLICKHOUSE_BIN" client --host 127.0.0.1 --port "$TCP_PORT" --query "SELECT 1" >/dev/null 2>&1; then
  echo "ClickHouse already running on 127.0.0.1:${TCP_PORT}"
  exit 0
fi

echo "Starting local ClickHouse ..."
"$CLICKHOUSE_BIN" server \
  --daemon \
  --pidfile "$PID_FILE" \
  -- \
  --http_port="$HTTP_PORT" \
  --tcp_port="$TCP_PORT" \
  --path="$DATA_DIR" \
  --logger.level=trace

for _ in {1..20}; do
  if "$CLICKHOUSE_BIN" client --host 127.0.0.1 --port "$TCP_PORT" --query "SELECT 1" >/dev/null 2>&1; then
    echo "ClickHouse started"
    echo "HTTP: http://127.0.0.1:${HTTP_PORT}"
    echo "TCP : 127.0.0.1:${TCP_PORT}"
    if [[ -f "$STATUS_FILE" ]]; then
      echo "Status:"
      sed -n '1,20p' "$STATUS_FILE"
    fi
    exit 0
  fi
  sleep 1
done

echo "ClickHouse failed to start in time"
if [[ -f "$STATUS_FILE" ]]; then
  sed -n '1,40p' "$STATUS_FILE" || true
fi
exit 1
