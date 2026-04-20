#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PID_FILE="$ROOT_DIR/.runtime/api-mysql.pid"

if [[ ! -f "$PID_FILE" ]]; then
  echo "No pid file found: $PID_FILE"
  exit 0
fi

pid="$(cat "$PID_FILE" || true)"
if [[ -z "$pid" ]]; then
  rm -f "$PID_FILE"
  echo "Empty pid file removed"
  exit 0
fi

if ! kill -0 "$pid" >/dev/null 2>&1; then
  rm -f "$PID_FILE"
  echo "Process not running, stale pid file removed"
  exit 0
fi

echo "Stopping Cnn.Api pid=$pid ..."
kill "$pid" >/dev/null 2>&1 || true

for _ in {1..15}; do
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    rm -f "$PID_FILE"
    echo "Stopped"
    exit 0
  fi
  sleep 1
done

echo "Force killing pid=$pid ..."
kill -9 "$pid" >/dev/null 2>&1 || true
rm -f "$PID_FILE"
echo "Stopped (force)"
