#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
KEEP_RUNNING="${KEEP_RUNNING:-0}"

# Ensure a clean API process for deterministic regression runs.
"$ROOT_DIR/scripts/stop_local_api_mysql.sh" >/dev/null 2>&1 || true
pkill -f 'Cnn.Api.dll' >/dev/null 2>&1 || true
pkill -f '/src/Cnn.Api/bin/Debug/net9.0/Cnn.Api' >/dev/null 2>&1 || true
sleep 1

"$ROOT_DIR/scripts/start_local_api_mysql.sh"
"$ROOT_DIR/scripts/smoke_live_mysql_extended.sh"

if [[ "$KEEP_RUNNING" == "1" ]]; then
  echo "Extended verification finished, service kept running (KEEP_RUNNING=1)."
  exit 0
fi

"$ROOT_DIR/scripts/stop_local_api_mysql.sh"
echo "Extended verification finished, service stopped."
