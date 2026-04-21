#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
API_BASE_URL="${BASE_URL:-http://127.0.0.1:5035}"
AGENT_URL="${AGENT_URL:-http://127.0.0.1:5091}"
AGENT_NODE_ID="${NODE_ID:-3}"
AGENT_NODE_TOKEN="${NODE_TOKEN:-token}"
AGENT_LOG="$RUNTIME_DIR/agent-nat.log"
ROUNDS="${ROUNDS:-1}"
ROUND_INTERVAL_SECONDS="${ROUND_INTERVAL_SECONDS:-2}"
START_AGENT="${START_AGENT:-1}"
PROBE_URL="${PROBE_URL:-}"
PROBE_PATH="${PROBE_PATH:-/}"
ACK_WAIT_SECONDS="${ACK_WAIT_SECONDS:-20}"
NAT_CHAIN="${NAT_CHAIN:-CNN_STREAM_DNAT}"
API_HEALTH_PATH="${API_HEALTH_PATH:-/api/health}"
SKIP_API_BOOTSTRAP="${SKIP_API_BOOTSTRAP:-0}"
CONFIG_PAYLOAD_MODE="${CONFIG_PAYLOAD_MODE:-agent_config}"

mkdir -p "$RUNTIME_DIR"

cleanup() {
  if [[ -n "${AGENT_PID:-}" ]] && kill -0 "$AGENT_PID" >/dev/null 2>&1; then
    kill "$AGENT_PID" >/dev/null 2>&1 || true
    sleep 1
  fi
  if [[ "$START_AGENT" == "1" ]]; then
    local agent_port
    agent_port="$(sed -E 's#^https?://[^:/]+:([0-9]+).*$#\1#' <<<"$AGENT_URL")"
    if [[ -n "$agent_port" && "$agent_port" =~ ^[0-9]+$ ]]; then
      if command -v lsof >/dev/null 2>&1; then
        lsof -ti "tcp:${agent_port}" | xargs -r kill >/dev/null 2>&1 || true
      else
        fuser -k "${agent_port}/tcp" >/dev/null 2>&1 || true
      fi
    fi
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

if ! [[ "$ROUNDS" =~ ^[0-9]+$ ]] || [[ "$ROUNDS" -le 0 ]]; then
  echo "Invalid ROUNDS=${ROUNDS}, expected positive integer"
  exit 1
fi

if ! curl -fsS "${API_BASE_URL}${API_HEALTH_PATH}" >/dev/null 2>&1; then
  if [[ "$SKIP_API_BOOTSTRAP" == "1" ]]; then
    echo "SKIP_API_BOOTSTRAP=1 and API health check failed, continue anyway: ${API_BASE_URL}${API_HEALTH_PATH}"
  else
    "$ROOT_DIR/scripts/start_local_api_mysql.sh"
  fi
fi

if [[ "$START_AGENT" == "1" ]]; then
  # Avoid bind conflicts when previous test runs left an agent process behind.
  agent_port="$(sed -E 's#^https?://[^:/]+:([0-9]+).*$#\1#' <<<"$AGENT_URL")"
  if [[ -n "$agent_port" && "$agent_port" =~ ^[0-9]+$ ]]; then
    if command -v lsof >/dev/null 2>&1; then
      lsof -ti "tcp:${agent_port}" | xargs -r kill >/dev/null 2>&1 || true
    else
      fuser -k "${agent_port}/tcp" >/dev/null 2>&1 || true
    fi
    sleep 1
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
else
  echo "START_AGENT=0, will use existing agent at ${AGENT_URL}"
fi

wait_agent_ready() {
  echo "Waiting for agent endpoint..."
  for _ in {1..30}; do
    if [[ -n "${AGENT_PID:-}" ]] && ! kill -0 "$AGENT_PID" >/dev/null 2>&1; then
      echo "Agent exited unexpectedly"
      tail -n 120 "$AGENT_LOG" || true
      return 1
    fi

    code="$(curl -s -o /dev/null -w '%{http_code}' "${AGENT_URL}/debug/stream/runtime" || true)"
    # 200 means runtime has active route; 503 means endpoint is alive but waiting for config.
    if [[ "$code" == "200" || "$code" == "503" ]]; then
      return 0
    fi
    sleep 1
  done
  echo "Agent endpoint not ready after timeout"
  return 1
}

extract_probe_port() {
  local report="$1"
  local listen
  listen="$(jq -r '.states[0].listen // ""' <<<"$report")"
  if [[ -z "$listen" ]]; then
    return 1
  fi

  if [[ "$listen" != *:* ]]; then
    return 1
  fi

  echo "${listen##*:}"
}

build_probe_url() {
  local port="$1"
  if [[ -n "$PROBE_URL" ]]; then
    echo "$PROBE_URL"
    return
  fi

  local host
  host="$(sed -E 's#^https?://([^:/]+).*$#\1#' <<<"$AGENT_URL")"
  if [[ -z "$host" || "$host" == "$AGENT_URL" ]]; then
    host="127.0.0.1"
  fi

  echo "http://${host}:${port}${PROBE_PATH}"
}

fetch_config_payload() {
  if [[ "$CONFIG_PAYLOAD_MODE" == "empty" ]]; then
    echo "{}"
    return 0
  fi

  if [[ "$CONFIG_PAYLOAD_MODE" != "agent_config" ]]; then
    echo "Unsupported CONFIG_PAYLOAD_MODE=$CONFIG_PAYLOAD_MODE"
    return 1
  fi

  local cfg
  cfg="$(curl -fsS "${API_BASE_URL}/api/v1/agent/config?node_id=${AGENT_NODE_ID}" \
    -H "Authorization: Bearer ${AGENT_NODE_TOKEN}")"

  local compact
  compact="$(jq -c '.data' <<<"$cfg" 2>/dev/null || true)"
  if [[ -z "$compact" || "$compact" == "null" || "$compact" == "[]" || "$compact" == "{}" ]]; then
    echo "Failed to parse agent config payload.data"
    return 1
  fi

  if ! jq -e 'has("version")' >/dev/null 2>&1 <<<"$compact"; then
    echo "agent config payload.data missing version"
    return 1
  fi

  echo "$compact"
}

verify_round() {
  local round="$1"
  echo "==== Round ${round}/${ROUNDS} ===="
  echo "Dispatching config_sync for runtime apply..."
  local payload
  payload="$(fetch_config_payload)"
  local payload_file
  payload_file="$(mktemp "${RUNTIME_DIR}/config-payload.XXXXXX.json")"
  printf '%s' "$payload" > "$payload_file"
  PAYLOAD_FILE="$payload_file" NODE_ID="$AGENT_NODE_ID" WAIT_SECONDS="$ACK_WAIT_SECONDS" "$ROOT_DIR/scripts/verify_agent_ws_ack.sh"
  rm -f "$payload_file"

  local report
  local code
  code="$(curl -s -o /tmp/cnn-stream-runtime.json -w '%{http_code}' "${AGENT_URL}/debug/stream/runtime" || true)"
  report="$(cat /tmp/cnn-stream-runtime.json)"
  if [[ "$code" != "200" ]]; then
    echo "runtime endpoint not ready after ACK, status=${code}"
    echo "$report"
    return 1
  fi
  local active_mode configured_mode nat_active
  active_mode="$(jq -r '.activeMode // ""' <<<"$report")"
  configured_mode="$(jq -r '.configuredMode // ""' <<<"$report")"
  nat_active="$(jq -r '.natActive // false' <<<"$report")"

  if [[ "$configured_mode" != "nat" || "$active_mode" != "nat" || "$nat_active" != "true" ]]; then
    echo "NAT mode not active as expected"
    echo "$report" | jq .
    return 1
  fi

  local port
  if ! port="$(extract_probe_port "$report")"; then
    echo "No stream listen port found in runtime report"
    echo "$report" | jq .
    return 1
  fi

  if ! iptables -t nat -S "$NAT_CHAIN" | grep -Eq -- "--dport[[:space:]]+${port}([[:space:]]|$)"; then
    echo "iptables NAT rule missing for dport=${port} in chain=${NAT_CHAIN}"
    iptables -t nat -S "$NAT_CHAIN" || true
    return 1
  fi

  local probe_url probe_code
  probe_url="$(build_probe_url "$port")"
  probe_code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 8 "$probe_url" || true)"
  if [[ "$probe_code" != "200" ]]; then
    echo "Probe failed: url=${probe_url} http_code=${probe_code}"
    return 1
  fi

  echo "Round ${round} passed: ACK + iptables(dport=${port}) + probe(200)"
  echo "- configured_mode: $configured_mode"
  echo "- active_mode    : $active_mode"
  echo "- nat_active     : $nat_active"
  echo "- probe_url      : $probe_url"
}

wait_agent_ready

for round in $(seq 1 "$ROUNDS"); do
  verify_round "$round"
  if [[ "$round" -lt "$ROUNDS" ]]; then
    sleep "$ROUND_INTERVAL_SECONDS"
  fi
done

echo "NAT verification passed for ${ROUNDS} round(s)"
