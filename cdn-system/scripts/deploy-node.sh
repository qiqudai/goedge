#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd "${SCRIPT_DIR}/.." && pwd)

AGENT_BIN="${ROOT_DIR}/agent/cdn-agent"
API_CONFIG="${ROOT_DIR}/api/config.yaml"
DEFAULT_REMOTE_DIR="/opt/cdn-system/agent"
DEFAULT_CONF="${SCRIPT_DIR}/deploy-node.env"

if [[ -f "${DEPLOY_NODE_CONF:-$DEFAULT_CONF}" ]]; then
  # shellcheck disable=SC1090
  source "${DEPLOY_NODE_CONF:-$DEFAULT_CONF}"
fi

err() {
  echo "Error: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || err "$1 is required"
}

maybe_install() {
  local cmd="$1"
  if command -v "$cmd" >/dev/null 2>&1; then
    return 0
  fi
  local install_cmd=""
  if command -v apt-get >/dev/null 2>&1; then
    install_cmd="apt-get update && apt-get install -y $cmd"
  elif command -v yum >/dev/null 2>&1; then
    install_cmd="yum install -y $cmd"
  fi
  if [[ -z "$install_cmd" ]]; then
    err "$cmd not found and no supported package manager"
  fi
  if [[ "$(id -u)" -ne 0 ]]; then
    command -v sudo >/dev/null 2>&1 || err "$cmd not found and sudo is not available"
    install_cmd="sudo $install_cmd"
  fi
  echo "Installing $cmd..."
  bash -c "$install_cmd"
}

detect_local_ip() {
  local ip=""
  if command -v ip >/dev/null 2>&1; then
    ip=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}')
  fi
  if [[ -z "$ip" ]] && command -v hostname >/dev/null 2>&1; then
    ip=$(hostname -I 2>/dev/null | awk '{print $1}')
  fi
  if [[ -z "$ip" ]] && command -v ifconfig >/dev/null 2>&1; then
    ip=$(ifconfig 2>/dev/null | awk '/inet / {print $2; exit}' | sed 's/addr://')
  fi
  echo "$ip"
}

prompt_default() {
  local var="$1"
  local prompt="$2"
  local default="${3:-}"
  local input=""
  if [[ -n "$default" ]]; then
    read -r -p "$prompt [$default]: " input
    input="${input:-$default}"
  else
    read -r -p "$prompt: " input
  fi
  printf -v "$var" '%s' "$input"
}

prompt_secret() {
  local var="$1"
  local prompt="$2"
  local input=""
  read -r -s -p "$prompt: " input
  echo
  printf -v "$var" '%s' "$input"
}

yaml_get() {
  local key="$1"
  [[ -f "$API_CONFIG" ]] || return 0
  awk -F: -v k="$key" '
    $1 ~ "^[[:space:]]*"k"[[:space:]]*$" {
      $1=""
      sub(/^:[[:space:]]*/, "", $0)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", $0)
      gsub(/"/, "", $0)
      print $0
      exit
    }
  ' "$API_CONFIG"
}

PYTHON_BIN=""
if command -v python3 >/dev/null 2>&1; then
  PYTHON_BIN="python3"
elif command -v python >/dev/null 2>&1; then
  PYTHON_BIN="python"
else
  err "python3 or python is required"
fi

json_get() {
  local key="$1"
  "$PYTHON_BIN" - "$key" <<'PY'
import json
import sys

path = sys.argv[1].strip(".")
data = json.load(sys.stdin)
if path:
    for part in path.split("."):
        if isinstance(data, dict) and part in data:
            data = data[part]
        else:
            sys.exit(1)
if data is None:
    sys.exit(1)
if isinstance(data, (dict, list)):
    sys.stdout.write(json.dumps(data))
else:
    sys.stdout.write(str(data))
PY
}

[[ -x "$AGENT_BIN" ]] || err "cdn-agent not found or not executable: $AGENT_BIN"

maybe_install sshpass
need_cmd curl
need_cmd ssh
need_cmd scp

prompt_default SSH_HOST "Target SSH host/IP" ""
[[ -n "$SSH_HOST" ]] || err "SSH host/IP is required"
prompt_default SSH_PORT "SSH port" "22"
prompt_default SSH_USER "SSH user" "root"
prompt_secret SSH_PASS "SSH password"
[[ -n "$SSH_PASS" ]] || err "SSH password is required"

if [[ -z "${NODE_TYPE:-}" ]]; then
  prompt_default NODE_TYPE "Node type (L1/L2)" "L1"
else
  echo "Using node type: $NODE_TYPE"
fi
NODE_TYPE_UPPER=$(echo "$NODE_TYPE" | tr '[:lower:]' '[:upper:]')
if [[ "$NODE_TYPE_UPPER" == "L1" ]]; then
  NODE_LEVEL=1
elif [[ "$NODE_TYPE_UPPER" == "L2" ]]; then
  NODE_LEVEL=2
else
  err "Node type must be L1 or L2"
fi

DEFAULT_API_PORT="${API_PORT:-$(yaml_get port || true)}"
if [[ -z "$DEFAULT_API_PORT" ]]; then
  DEFAULT_API_PORT="8080"
fi
DEFAULT_API_HOST="${API_HOST:-}"
if [[ -z "$DEFAULT_API_HOST" ]]; then
  DEFAULT_API_HOST="$(detect_local_ip || true)"
fi
if [[ -z "$DEFAULT_API_HOST" ]]; then
  DEFAULT_API_HOST="127.0.0.1"
fi
DEFAULT_API_SCHEME="${API_SCHEME:-http}"
DEFAULT_API_BASE="${DEFAULT_API_SCHEME}://${DEFAULT_API_HOST}:${DEFAULT_API_PORT}"
if [[ -z "${API_BASE:-}" ]]; then
  API_BASE="$DEFAULT_API_BASE"
  echo "Using API base: $API_BASE"
else
  API_BASE="${API_BASE%/}"
  echo "Using API base (override): $API_BASE"
fi

CURL_INSECURE="${CURL_INSECURE:-n}"
CURL_FLAGS=()
if [[ "$(echo "$CURL_INSECURE" | tr '[:upper:]' '[:lower:]')" =~ ^y ]]; then
  CURL_FLAGS+=(-k)
fi

CONFIG_AGENT_TOKEN="$(yaml_get agent_token || true)"
if [[ -z "${AGENT_TOKEN:-}" ]]; then
  AGENT_TOKEN="$CONFIG_AGENT_TOKEN"
  if [[ -n "$AGENT_TOKEN" ]]; then
    echo "Using agent token from config.yaml"
  else
    prompt_default AGENT_TOKEN "Agent token" ""
  fi
fi
[[ -n "$AGENT_TOKEN" ]] || err "Agent token is required"

if [[ -z "${AUTO_CREATE:-}" ]]; then
  if [[ -n "${ADMIN_TOKEN:-}" || (-n "${ADMIN_USER:-}" && -n "${ADMIN_PASS:-}") ]]; then
    AUTO_CREATE="y"
  else
    prompt_default AUTO_CREATE "Create node via API? (y/N)" "n"
  fi
fi
AUTO_CREATE_LOWER=$(echo "$AUTO_CREATE" | tr '[:upper:]' '[:lower:]')

NODE_ID=""
if [[ "$AUTO_CREATE_LOWER" =~ ^y ]]; then
  if [[ -z "${ADMIN_TOKEN:-}" ]]; then
    if [[ -z "${ADMIN_USER:-}" ]]; then
      prompt_default ADMIN_USER "Admin username" "admin"
    fi
    if [[ -z "${ADMIN_PASS:-}" ]]; then
      prompt_secret ADMIN_PASS "Admin password"
    fi
    [[ -n "$ADMIN_PASS" ]] || err "Admin password is required"
  fi

  NODE_NAME="${NODE_NAME:-node-${SSH_HOST}}"
  NODE_IP="${NODE_IP:-$SSH_HOST}"
  NODE_PORT="${NODE_PORT:-80}"
  [[ "$NODE_PORT" =~ ^[0-9]+$ ]] || err "Node port must be numeric"

  ADMIN_TOKEN_VALUE="${ADMIN_TOKEN:-}"
  if [[ -z "$ADMIN_TOKEN_VALUE" ]]; then
    LOGIN_PAYLOAD=$(
      ADMIN_USER="$ADMIN_USER" ADMIN_PASS="$ADMIN_PASS" \
        "$PYTHON_BIN" - <<'PY'
import json
import os
print(json.dumps({
    "username": os.environ["ADMIN_USER"],
    "password": os.environ["ADMIN_PASS"],
}))
PY
    )

    LOGIN_RESP=$(curl -sS "${CURL_FLAGS[@]}" -H "Content-Type: application/json" \
      -d "$LOGIN_PAYLOAD" \
      "${API_BASE}/api/v1/admin/login" || true)
    ADMIN_TOKEN_VALUE=$(printf '%s' "$LOGIN_RESP" | json_get token 2>/dev/null || true)
    [[ -n "$ADMIN_TOKEN_VALUE" ]] || err "Admin login failed: $LOGIN_RESP"
  fi

  CREATE_PAYLOAD=$(
    NODE_NAME="$NODE_NAME" NODE_IP="$NODE_IP" NODE_LEVEL="$NODE_LEVEL" NODE_PORT="$NODE_PORT" \
      "$PYTHON_BIN" - <<'PY'
import json
import os
print(json.dumps({
    "name": os.environ["NODE_NAME"],
    "ip": os.environ["NODE_IP"],
    "type": int(os.environ["NODE_LEVEL"]),
    "port": int(os.environ["NODE_PORT"]),
    "enable": True,
}))
PY
  )

  CREATE_RESP=$(curl -sS "${CURL_FLAGS[@]}" -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${ADMIN_TOKEN_VALUE}" \
    -d "$CREATE_PAYLOAD" \
    "${API_BASE}/api/v1/admin/nodes" || true)
  NODE_ID=$(printf '%s' "$CREATE_RESP" | json_get data.id 2>/dev/null || true)
  [[ -n "$NODE_ID" ]] || err "Node create failed: $CREATE_RESP"
else
  if [[ -z "${NODE_ID:-}" ]]; then
    prompt_default NODE_ID "Existing node ID" ""
  fi
  [[ -n "$NODE_ID" ]] || err "Node ID is required"
fi

if [[ -z "${REMOTE_DIR:-}" ]]; then
  prompt_default REMOTE_DIR "Remote install dir" "$DEFAULT_REMOTE_DIR"
fi
REMOTE_DIR="${REMOTE_DIR%/}"
[[ -n "$REMOTE_DIR" ]] || err "Remote install dir is required"

tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

AGENT_JSON="${tmp_dir}/agent.json"
API_BASE="$API_BASE" AGENT_TOKEN="$AGENT_TOKEN" NODE_ID="$NODE_ID" \
  "$PYTHON_BIN" - <<'PY' > "$AGENT_JSON"
import json
import os

data = {
    "api": os.environ["API_BASE"],
    "token": os.environ["AGENT_TOKEN"],
    "node_id": os.environ["NODE_ID"],
    "debug": False,
}
print(json.dumps(data, indent=2))
PY

SERVICE_FILE="${tmp_dir}/cdn-agent.service"
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=CDN Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${REMOTE_DIR}
ExecStart=${REMOTE_DIR}/cdn-agent -config ${REMOTE_DIR}/agent.json
Restart=always
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF

SSH_OPTS=(-p "$SSH_PORT" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null)
SSH_BASE=(sshpass -p "$SSH_PASS" ssh "${SSH_OPTS[@]}")
SCP_BASE=(sshpass -p "$SSH_PASS" scp -P "$SSH_PORT" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null)
REMOTE_HOST="${SSH_USER}@${SSH_HOST}"

"${SSH_BASE[@]}" "$REMOTE_HOST" "echo connected" >/dev/null
REMOTE_UID=$("${SSH_BASE[@]}" "$REMOTE_HOST" "id -u")
REMOTE_IS_ROOT=0
if [[ "$REMOTE_UID" == "0" ]]; then
  REMOTE_IS_ROOT=1
fi

run_remote() {
  local cmd="$1"
  "${SSH_BASE[@]}" "$REMOTE_HOST" "$cmd"
}

run_remote_sudo() {
  local cmd="$1"
  if [[ "$REMOTE_IS_ROOT" == "1" ]]; then
    run_remote "$cmd"
  else
    printf '%s\n' "$SSH_PASS" | "${SSH_BASE[@]}" "$REMOTE_HOST" "sudo -S -p '' $cmd"
  fi
}

REMOTE_TMP="/tmp/cdn-agent-deploy-$$"
run_remote "mkdir -p '$REMOTE_TMP'"

"${SCP_BASE[@]}" "$AGENT_BIN" "${REMOTE_HOST}:${REMOTE_TMP}/cdn-agent"
"${SCP_BASE[@]}" "$AGENT_JSON" "${REMOTE_HOST}:${REMOTE_TMP}/agent.json"
"${SCP_BASE[@]}" "$SERVICE_FILE" "${REMOTE_HOST}:${REMOTE_TMP}/cdn-agent.service"

run_remote_sudo "mkdir -p '$REMOTE_DIR'"
run_remote_sudo "mv '$REMOTE_TMP/cdn-agent' '$REMOTE_DIR/cdn-agent'"
run_remote_sudo "mv '$REMOTE_TMP/agent.json' '$REMOTE_DIR/agent.json'"
run_remote_sudo "chmod 755 '$REMOTE_DIR/cdn-agent'"
run_remote_sudo "chmod 600 '$REMOTE_DIR/agent.json'"

if "${SSH_BASE[@]}" "$REMOTE_HOST" "command -v systemctl >/dev/null 2>&1"; then
  run_remote_sudo "mv '$REMOTE_TMP/cdn-agent.service' /etc/systemd/system/cdn-agent.service"
  run_remote_sudo "systemctl daemon-reload"
  run_remote_sudo "systemctl enable --now cdn-agent"
  run_remote_sudo "systemctl is-active --quiet cdn-agent"
  run_remote_sudo "rm -rf '$REMOTE_TMP'"
  echo "Deploy OK. Node ID: ${NODE_ID}. Service: cdn-agent"
else
  run_remote_sudo "rm -f /tmp/cdn-agent.log /tmp/cdn-agent.err"
  run_remote_sudo "nohup '$REMOTE_DIR/cdn-agent' -config '$REMOTE_DIR/agent.json' >/tmp/cdn-agent.log 2>/tmp/cdn-agent.err &"
  run_remote_sudo "rm -rf '$REMOTE_TMP'"
  echo "Deploy OK. Node ID: ${NODE_ID}. Started with nohup (logs: /tmp/cdn-agent.log)."
fi
