#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_DIR="$ROOT/docker/compat-e2e"
COMPOSE="docker compose -f $COMPOSE_DIR/docker-compose.yml"
if [[ -z "${COMPAT_E2E_HTTPS_PORT:-}" ]]; then
  PORT="$(python3 - <<'PY'
import socket
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
)"
else
  PORT="$COMPAT_E2E_HTTPS_PORT"
fi
export COMPAT_E2E_HTTPS_PORT="$PORT"
BASE_URL="https://127.0.0.1:${PORT}"

cleanup() {
  $COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$COMPOSE_DIR/.build"
}
trap cleanup EXIT

prepare_context() {
  rm -rf "$COMPOSE_DIR/.build"
  mkdir -p "$COMPOSE_DIR/.build/lua"
  cp "$ROOT/agent/edge-node/lua/response_headers.lua" "$COMPOSE_DIR/.build/lua/response_headers.lua"
}

wait_edge() {
  local i code
  for i in $(seq 1 60); do
    code=$(curl -k -s -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 \
      -H "Host: compat.test" "$BASE_URL/nginx-health" || echo "000")
    if [[ "$code" == "200" ]]; then
      code=$(curl -k -s -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 \
        -H "Host: compat.test" "$BASE_URL/" || echo "000")
    fi
    if [[ "$code" == "200" ]]; then
      return 0
    fi
    sleep 1
  done
  $COMPOSE logs --no-color edge origin | tail -n 120
  return 1
}

assert_no_header() {
  local name="$1"
  local headers="$2"
  if printf '%s\n' "$headers" | grep -Eiq "^${name}:"; then
    echo "[FAIL] unexpected response header: $name"
    printf '%s\n' "$headers"
    exit 1
  fi
  echo "[PASS] stripped $name"
}

echo "[compat-e2e] starting stack"
$COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
prepare_context
$COMPOSE up -d --build
wait_edge

echo "[compat-e2e] HTTPS/HTTP2 normal response strips origin hop-by-hop headers"
headers=$(curl -k --http2 -sS -D - -o /tmp/compat_e2e_body.txt \
  --connect-timeout 5 --max-time 15 \
  -H "Host: compat.test" "$BASE_URL/" | tr -d '\r')
if ! printf '%s\n' "$headers" | grep -q '^HTTP/2 200'; then
  echo "[FAIL] expected HTTP/2 200"
  printf '%s\n' "$headers"
  exit 1
fi
if [[ "$(cat /tmp/compat_e2e_body.txt)" != "origin-ok" ]]; then
  echo "[FAIL] unexpected body: $(cat /tmp/compat_e2e_body.txt)"
  exit 1
fi
for h in Upgrade Connection Keep-Alive Proxy-Connection TE Trailer Transfer-Encoding; do
  assert_no_header "$h" "$headers"
done

echo "[compat-e2e] HTTP/1.1 h2c/Expect request headers are not forwarded to origin"
request_echo=$(curl -k --http1.1 -sS -o - \
  --connect-timeout 5 --max-time 15 \
  -H "Host: compat.test" \
  -H "Connection: Upgrade" \
  -H "Upgrade: h2,h2c" \
  -H "Expect: 100-continue" \
  -H "TE: trailers" \
  -H "Trailer: Expires" \
  -H "Proxy-Connection: keep-alive" \
  "$BASE_URL/echo-request")
for h in Upgrade Expect TE Trailer Proxy-Connection Keep-Alive; do
  if ! printf '%s\n' "$request_echo" | grep -Eq "^${h}: $"; then
    echo "[FAIL] expected origin to receive empty $h header"
    printf '%s\n' "$request_echo"
    exit 1
  fi
done
if ! printf '%s\n' "$request_echo" | grep -Eq '^Connection: $'; then
  echo "[FAIL] expected h2c Connection upgrade to be suppressed"
  printf '%s\n' "$request_echo"
  exit 1
fi
echo "[PASS] suppressed non-websocket request upgrade headers"

echo "[compat-e2e] WebSocket 101 keeps upgrade headers on HTTP/1.1"
ws_headers=$(curl -k --http1.1 -sS -D - -o /tmp/compat_e2e_ws_body.txt \
  --connect-timeout 5 --max-time 15 \
  -H "Host: compat.test" \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  "$BASE_URL/ws" || true)
ws_headers=$(printf '%s' "$ws_headers" | tr -d '\r')
if ! printf '%s\n' "$ws_headers" | grep -q '^HTTP/1.1 101'; then
  echo "[FAIL] expected HTTP/1.1 101"
  printf '%s\n' "$ws_headers"
  exit 1
fi
if ! printf '%s\n' "$ws_headers" | grep -Eiq '^Upgrade: websocket'; then
  echo "[FAIL] expected websocket Upgrade header"
  printf '%s\n' "$ws_headers"
  exit 1
fi
if ! printf '%s\n' "$ws_headers" | grep -Eiq '^Connection: upgrade'; then
  echo "[FAIL] expected Connection upgrade header"
  printf '%s\n' "$ws_headers"
  exit 1
fi

echo "[compat-e2e] ALL PASSED"
