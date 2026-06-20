#!/bin/sh
set -eu

CONF_DIR=/opt/cc/conf
CONF_FILE="${CONF_DIR}/cdn_config.json"
API_URL="${API_URL:-http://api:8080}"
AGENT_TOKEN="${AGENT_TOKEN:-cc-e2e-agent-token}"
NODE_ID="${NODE_ID:-1}"
SYNC_INTERVAL="${SYNC_INTERVAL:-3}"

mkdir -p "${CONF_DIR}"

sync_config() {
  tmp="$(mktemp)"
  code="$(curl -sS -m 10 \
    -H "Authorization: Bearer ${AGENT_TOKEN}" \
    -o "${tmp}" -w '%{http_code}' \
    "${API_URL}/api/v1/agent/config?node_id=${NODE_ID}" || echo "000")"

  if [ "${code}" = "200" ] && [ -s "${tmp}" ]; then
    mv "${tmp}" "${CONF_FILE}"
    chmod 644 "${CONF_FILE}"
    return 0
  fi

  rm -f "${tmp}"
  return 1
}

echo "[edge] waiting for control plane config..."
attempt=0
until sync_config; do
  attempt=$((attempt + 1))
  if [ "${attempt}" -ge 60 ]; then
    echo "[edge] failed to pull initial config from ${API_URL}" >&2
    exit 1
  fi
  sleep 2
done
echo "[edge] initial config synced to ${CONF_FILE}"

(
  while true; do
    sleep "${SYNC_INTERVAL}"
    if sync_config; then
      echo "[edge] config synced"
    else
      echo "[edge] config sync failed" >&2
    fi
  done
) &

exec openresty -p /opt/cc -c /opt/cc/nginx.conf -g 'daemon off;'
