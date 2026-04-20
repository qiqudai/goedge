#!/usr/bin/env bash
set -euo pipefail

HOST="${1:-127.0.0.1}"
PORT="${2:-10000}"
DURATION_SECONDS="${3:-300}"
CONCURRENCY="${4:-2000}"
RELOAD_COUNT="${5:-30}"
RELOAD_INTERVAL_SECONDS="${6:-5}"
CONFIG_PUSH_CMD="${CONFIG_PUSH_CMD:-}"

if ! command -v tcpkali >/dev/null 2>&1; then
  echo "tcpkali not found. install from https://github.com/machinezone/tcpkali"
  exit 2
fi

if [[ -z "${CONFIG_PUSH_CMD}" ]]; then
  echo "CONFIG_PUSH_CMD is required, for example:"
  echo "  CONFIG_PUSH_CMD='./scripts/push_stream_config.sh'"
  exit 2
fi

echo "[stream_hot_reload_survival] start traffic host=${HOST} port=${PORT} duration=${DURATION_SECONDS}s concurrency=${CONCURRENCY}"
tcpkali \
  --connect-rate=800 \
  --connections="${CONCURRENCY}" \
  --duration="${DURATION_SECONDS}s" \
  --message-rate=1 \
  "${HOST}:${PORT}" >/tmp/stream_hot_reload_survival.log 2>&1 &
TRAFFIC_PID=$!

cleanup() {
  kill "${TRAFFIC_PID}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

for ((i = 1; i <= RELOAD_COUNT; i++)); do
  echo "[stream_hot_reload_survival] reload ${i}/${RELOAD_COUNT}"
  bash -lc "${CONFIG_PUSH_CMD}"
  sleep "${RELOAD_INTERVAL_SECONDS}"
done

wait "${TRAFFIC_PID}"
echo "[stream_hot_reload_survival] completed, log: /tmp/stream_hot_reload_survival.log"
