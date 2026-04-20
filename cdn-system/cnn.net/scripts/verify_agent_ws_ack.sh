#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:5035}"
ADMIN_USER="${ADMIN_USER:-cnn_ai_admin}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"
NODE_ID="${NODE_ID:-}"
TASK_TYPE="${TASK_TYPE:-config_sync}"
WAIT_SECONDS="${WAIT_SECONDS:-10}"
PAYLOAD_JSON="${PAYLOAD_JSON:-{}}"

json_post() {
  local url="$1"
  local body="$2"
  local token="${3:-}"
  if [[ -n "$token" ]]; then
    curl -sS -X POST "${BASE_URL}${url}" -H 'Content-Type: application/json' -H "Authorization: Bearer ${token}" -d "$body"
  else
    curl -sS -X POST "${BASE_URL}${url}" -H 'Content-Type: application/json' -d "$body"
  fi
}

json_get() {
  local url="$1"
  local token="${2:-}"
  if [[ -n "$token" ]]; then
    curl -sS "${BASE_URL}${url}" -H "Authorization: Bearer ${token}"
  else
    curl -sS "${BASE_URL}${url}"
  fi
}

assert_json() {
  local raw="$1"
  if ! jq -e . >/dev/null 2>&1 <<<"$raw"; then
    echo "Invalid JSON response:"
    echo "$raw"
    exit 1
  fi
}

login_resp="$(json_post '/api/v1/admin/login' "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}")"
assert_json "$login_resp"
if [[ "$(jq -r '.code' <<<"$login_resp")" != "200" ]]; then
  echo "Admin login failed"
  echo "$login_resp" | jq .
  exit 1
fi
admin_token="$(jq -r '.data.token // ""' <<<"$login_resp")"
if [[ -z "$admin_token" ]]; then
  echo "Admin token missing"
  exit 1
fi

if [[ -z "$NODE_ID" ]]; then
  nodes_resp="$(json_get '/api/v1/admin/nodes?page=1&pageSize=100' "$admin_token")"
  assert_json "$nodes_resp"
  if [[ "$(jq -r '.code' <<<"$nodes_resp")" != "200" ]]; then
    echo "Failed to list nodes"
    echo "$nodes_resp" | jq .
    exit 1
  fi

  NODE_ID="$(jq -r '.data.list[] | select(.online==true) | .id' <<<"$nodes_resp" | head -n1)"
  if [[ -z "$NODE_ID" || "$NODE_ID" == "null" ]]; then
    NODE_ID="$(jq -r '.data.list[0].id // ""' <<<"$nodes_resp")"
  fi
fi

if [[ -z "$NODE_ID" || "$NODE_ID" == "null" || "$NODE_ID" == "0" ]]; then
  echo "No available node found. Start/register an agent node first."
  exit 1
fi

dispatch_body="$(jq -n \
  --argjson node_id "$NODE_ID" \
  --arg task_type "$TASK_TYPE" \
  --arg payload "$PAYLOAD_JSON" \
  --argjson wait_seconds "$WAIT_SECONDS" \
  '{node_id:$node_id, task_type:$task_type, payload:$payload, wait_seconds:$wait_seconds}')"

dispatch_resp="$(json_post '/api/v1/admin/ws/dispatch' "$dispatch_body" "$admin_token")"
assert_json "$dispatch_resp"

code="$(jq -r '.code' <<<"$dispatch_resp")"
connected="$(jq -r '.data.connected // false' <<<"$dispatch_resp")"
state="$(jq -r '.data.state // ""' <<<"$dispatch_resp")"
error="$(jq -r '.data.error // ""' <<<"$dispatch_resp")"

if [[ "$code" != "200" ]]; then
  echo "Dispatch API failed"
  echo "$dispatch_resp" | jq .
  exit 1
fi

if [[ "$connected" != "true" ]]; then
  echo "Node is not connected to /ws/agent"
  echo "$dispatch_resp" | jq .
  exit 1
fi

if [[ -z "$state" || "$state" == "timeout" ]]; then
  echo "No ACK received within ${WAIT_SECONDS}s"
  echo "$dispatch_resp" | jq .
  exit 1
fi

cat <<OUT
Agent WS ACK verified
- base_url : ${BASE_URL}
- node_id  : ${NODE_ID}
- task_type: ${TASK_TYPE}
- state    : ${state}
- error    : ${error}
OUT
