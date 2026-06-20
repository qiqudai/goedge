#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
KEEP_RUNNING="${KEEP_RUNNING:-0}"
SKIP_EXTERNAL_DNS="${SKIP_EXTERNAL_DNS:-0}"
STRICT_EXTERNAL_DNS="${STRICT_EXTERNAL_DNS:-0}"
EXIT_CODE_FAILURE=1
EXIT_EXTERNAL_DEPENDENCY=20

# Ensure a clean API process for deterministic regression runs.
"$ROOT_DIR/scripts/stop_local_api_mysql.sh" >/dev/null 2>&1 || true
pkill -f 'Cnn.Api.dll' >/dev/null 2>&1 || true
pkill -f '/src/Cnn.Api/bin/Debug/net9.0/Cnn.Api' >/dev/null 2>&1 || true
sleep 1

"$ROOT_DIR/scripts/start_local_api_mysql.sh"
echo "verify_live_mysql_extended.sh: SKIP_EXTERNAL_DNS=${SKIP_EXTERNAL_DNS} STRICT_EXTERNAL_DNS=${STRICT_EXTERNAL_DNS}"
set +e
"$ROOT_DIR/scripts/smoke_live_mysql_extended.sh"
smoke_exit=$?
set -e

case "$smoke_exit" in
  0)
    echo "verify result: success"
    ;;
  "$EXIT_EXTERNAL_DEPENDENCY")
    echo "verify result: external_dependency_failure (exit=${smoke_exit})"
    ;;
  *)
    echo "verify result: code_failure (exit=${smoke_exit})"
    ;;
esac

if [[ "$KEEP_RUNNING" == "1" ]]; then
  echo "Extended verification finished, service kept running (KEEP_RUNNING=1, exit=${smoke_exit})."
  exit "$smoke_exit"
fi

"$ROOT_DIR/scripts/stop_local_api_mysql.sh"
echo "Extended verification finished, service stopped (exit=${smoke_exit})."
exit "$smoke_exit"
