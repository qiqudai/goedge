#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
API_BASE_URL="${BASE_URL:-http://127.0.0.1:5035}"
AGENT_URL="${AGENT_URL:-http://127.0.0.1:5091}"
AGENT_NODE_ID="${NODE_ID:-3}"
AGENT_NODE_TOKEN="${NODE_TOKEN:-token}"
AGENT_LOG="$RUNTIME_DIR/agent-nat.log"

mkdir -p "$RUNTIME_DIR"

cleanup() {
  if [[ -n "${AGENT_PID:-}" ]] && kill -0 "$AGENT_PID" >/dev/null 2>&1; then
    kill "$AGENT_PID" >/dev/null 2>&1 || true
    sleep 1
  fi
}
trap cleanup EXIT

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "SKIP: verify_stream_nat_mode requires Linux"
  exit 0
fi

if ! command -v iptables >/dev/null 2>&1; then
  echo "SKIP: iptables not found"
  exit 0
fi

if [[ "${EUID}" -ne 0 ]]; then
  echo "SKIP: nat mode verification requires root/CAP_NET_ADMIN"
  exit 0
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "Missing dependency: jq"
  exit 1
fi

if ! curl -fsS "${API_BASE_URL}/api/health" >/dev/null 2>&1; then
  "$ROOT_DIR/scripts/start_local_api_mysql.sh"
fi

echo "Starting agent in NAT mode..."
ASPNETCORE_ENVIRONMENT=Development \
ASPNETCORE_URLS="$AGENT_URL" \
Api__BaseUrl="$API_BASE_URL" \
Node__Id="$AGENT_NODE_ID" \
Node__Token="$AGENT_NODE_TOKEN" \
Stream__Mode=nat \
Stream__FallbackToUserspaceOnNatFailure=false \
dotnet run --no-launch-profile --project "$ROOT_DIR/src/Cnn.Agent/Cnn.Agent.csproj" >"$AGENT_LOG" 2>&1 &
AGENT_PID=$!

echo "Waiting for agent endpoint..."
for _ in {1..30}; do
  if ! kill -0 "$AGENT_PID" >/dev/null 2>&1; then
    echo "Agent exited unexpectedly"
    tail -n 120 "$AGENT_LOG" || true
    exit 1
  fi

  code="$(curl -s -o /dev/null -w '%{http_code}' "${AGENT_URL}/debug/stream/runtime" || true)"
  if [[ "$code" == "200" ]]; then
    break
  fi
  sleep 1
done

echo "Dispatching config_sync for runtime apply..."
NODE_ID="$AGENT_NODE_ID" WAIT_SECONDS=20 "$ROOT_DIR/scripts/verify_agent_ws_ack.sh"

report="$(curl -fsS "${AGENT_URL}/debug/stream/runtime")"
active_mode="$(jq -r '.activeMode // ""' <<<"$report")"
configured_mode="$(jq -r '.configuredMode // ""' <<<"$report")"
nat_active="$(jq -r '.natActive // false' <<<"$report")"

if [[ "$configured_mode" != "nat" || "$active_mode" != "nat" || "$nat_active" != "true" ]]; then
  echo "NAT mode not active as expected"
  echo "$report" | jq .
  exit 1
fi

echo "NAT mode verified"
echo "- configured_mode: $configured_mode"
echo "- active_mode    : $active_mode"
echo "- nat_active     : $nat_active"
