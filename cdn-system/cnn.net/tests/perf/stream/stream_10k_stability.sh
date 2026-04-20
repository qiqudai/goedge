#!/usr/bin/env bash
set -euo pipefail

HOST="${1:-127.0.0.1}"
PORT="${2:-10000}"
DURATION_SECONDS="${3:-600}"
CONCURRENCY="${4:-10000}"

if ! command -v tcpkali >/dev/null 2>&1; then
  echo "tcpkali not found. install from https://github.com/machinezone/tcpkali"
  exit 2
fi

echo "[stream_10k_stability] host=${HOST} port=${PORT} duration=${DURATION_SECONDS}s concurrency=${CONCURRENCY}"
tcpkali \
  --connect-rate=2000 \
  --connections="${CONCURRENCY}" \
  --duration="${DURATION_SECONDS}s" \
  --message-rate=1 \
  "${HOST}:${PORT}"
