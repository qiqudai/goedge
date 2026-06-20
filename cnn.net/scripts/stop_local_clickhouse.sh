#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
DATA_DIR="$RUNTIME_DIR/clickhouse-data"
STATUS_FILE="$DATA_DIR/status"
PID_FILE="$RUNTIME_DIR/clickhouse.pid"

pid=""
if [[ -f "$STATUS_FILE" ]]; then
  pid="$(awk '/^PID:/ {print $2}' "$STATUS_FILE" | head -n1)"
fi
if [[ -z "$pid" && -f "$PID_FILE" ]]; then
  pid="$(cat "$PID_FILE" 2>/dev/null || true)"
fi

if [[ -z "$pid" ]]; then
  echo "No ClickHouse pid found"
  exit 0
fi

if ! kill -0 "$pid" >/dev/null 2>&1; then
  rm -f "$PID_FILE"
  echo "ClickHouse process not running, stale pid removed"
  exit 0
fi

echo "Stopping ClickHouse pid=$pid ..."
kill "$pid" >/dev/null 2>&1 || true

for _ in {1..20}; do
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    rm -f "$PID_FILE"
    echo "Stopped"
    exit 0
  fi
  sleep 1
done

echo "Force stopping ClickHouse pid=$pid ..."
kill -9 "$pid" >/dev/null 2>&1 || true
rm -f "$PID_FILE"
echo "Stopped (force)"
