#!/usr/bin/env bash
# Local CC matrix test: edge intercepts before origin, matcher/filter combinations.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_DIR="$ROOT/docker/cc-e2e"
COMPOSE="docker compose -f $COMPOSE_DIR/docker-compose.yml"
PORT="${CC_E2E_PORT:-18089}"
HOST="cc-test.local"
BASE_URL="http://127.0.0.1:${PORT}"
AGENT_TOKEN="cc-e2e-agent-token"
MYSQL="$COMPOSE exec -T mysql mysql -uroot -pcc_test_root cdnfy"
CLIENT_IP="${CC_E2E_CLIENT_IP:-10.0.0.50}"
SYNC_WAIT="${CC_SYNC_WAIT:-8}"
RATE_WAIT="${CC_RATE_WAIT:-6}"

pass=0
fail=0

cleanup() {
  if [[ "${CC_LOCAL_KEEP:-0}" != "1" ]]; then
    $COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

prepare_edge_stage() {
  local stage="$COMPOSE_DIR/stage"
  rm -rf "$stage"
  mkdir -p "$stage/lua" "$stage/conf/guard"
  cp -R "$ROOT/agent/edge-node/lua/." "$stage/lua/"
  cp -R "$ROOT/agent/edge-node/conf/guard/." "$stage/conf/guard/" 2>/dev/null || true
  cp "$COMPOSE_DIR/edge/nginx.conf" "$stage/nginx.conf"
  cp "$COMPOSE_DIR/edge/entrypoint.sh" "$stage/entrypoint.sh"
}

origin_total() {
  $COMPOSE exec -T origin wget -qO- http://127.0.0.1:8080/_origin/stats 2>/dev/null \
    | python3 -c "import json,sys; print(json.load(sys.stdin).get('total',0))" 2>/dev/null || echo "0"
}

wait_edge() {
  local i
  for i in $(seq 1 90); do
    local code
    code=$(curl -s -o /dev/null -w '%{http_code}' -H "Host: ${HOST}" "${BASE_URL}/nginx-health" || echo "000")
    if [[ "$code" == "200" ]]; then
      echo "[cc-local] edge ready"
      return 0
    fi
    sleep 2
  done
  echo "[cc-local] timeout waiting for edge" >&2
  return 1
}

sync_site() {
  local sql="$1"
  $MYSQL -e "$sql" >/dev/null 2>&1
  sleep "$SYNC_WAIT"
}

assert_http() {
  local name="$1"
  local expect="$2"
  local ip="${3:-$CLIENT_IP}"
  local path="${4:-/}"
  local ua="${5:-curl/cc-local-test}"
  local code
  code=$(curl -s -o /tmp/cc_local_body.txt -w '%{http_code}' \
    -H "Host: ${HOST}" \
    -H "X-Forwarded-For: ${ip}" \
    -H "User-Agent: ${ua}" \
    "${BASE_URL}${path}" || echo "000")
  if [[ "$code" == "$expect" ]]; then
    echo "[PASS] $name -> HTTP $code"
    pass=$((pass + 1))
  else
    echo "[FAIL] $name -> HTTP $code, want $expect"
    echo "       body: $(tr '\n' ' ' < /tmp/cc_local_body.txt | head -c 160)"
    fail=$((fail + 1))
  fi
}

assert_origin_delta() {
  local name="$1"
  local max_delta="$2"
  local before="$3"
  local after
  after=$(origin_total)
  local delta=$((after - before))
  if [[ "$delta" -le "$max_delta" ]]; then
    echo "[PASS] $name -> origin +${delta} (max ${max_delta})"
    pass=$((pass + 1))
  else
    echo "[FAIL] $name -> origin +${delta}, want <= ${max_delta} (before=${before} after=${after})"
    fail=$((fail + 1))
  fi
}

burst_requests() {
  local n="$1"
  local ip="$2"
  local path="$3"
  local ua="${4:-curl/cc-local-burst}"
  local i
  for i in $(seq 1 "$n"); do
    curl -s -o /dev/null \
      -H "Host: ${HOST}" \
      -H "X-Forwarded-For: ${ip}" \
      -H "User-Agent: ${ua}" \
      "${BASE_URL}${path}" >/dev/null || true
  done
}

echo "[cc-local] preparing stack"
prepare_edge_stage
$COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true

if [[ "${CC_LOCAL_FULL_BUILD:-0}" == "1" ]] || ! docker image inspect cc-e2e-api:latest >/dev/null 2>&1; then
  echo "[cc-local] building mysql/origin/api images"
  $COMPOSE build mysql origin api
else
  echo "[cc-local] rebuilding mysql/origin (seed + stats)"
  $COMPOSE build mysql origin
fi

docker build -f "$COMPOSE_DIR/Dockerfile.edge" -t cc-e2e-edge "$COMPOSE_DIR"
$COMPOSE up -d --no-build
$COMPOSE up -d --no-deps --force-recreate edge origin

wait_edge || {
  $COMPOSE logs --no-color edge api origin mysql | tail -n 80
  exit 1
}

echo "[cc-local] === 1) burst / : edge blocks, origin not flooded ==="
sync_site "UPDATE site SET cc_default_rule=10001, white_ip=NULL, settings='{\"access\":{\"acl\":0},\"security\":{\"custom_rules\":[{\"action\":\"allow\",\"on\":true,\"matchers\":[{\"key\":\"uri\",\"operator\":\"eq\",\"value\":\"/api\"}]},{\"action\":\"block\",\"on\":false,\"matchers\":[{\"key\":\"uri\",\"operator\":\"eq\",\"value\":\"/disabled-rule\"}]}]}}', update_at=NOW() WHERE id=1;"
sleep "$RATE_WAIT"
before=$(origin_total)
burst_requests 30 "$CLIENT_IP" "/"
assert_origin_delta "burst 30x / origin hits" 2 "$before"
assert_http "immediately after burst blocked at edge" "403" "$CLIENT_IP" "/"

echo "[cc-local] === 2) req_rate matcher=/ : 2 allow then block ==="
sync_site "UPDATE site SET cc_default_rule=10001, update_at=NOW() WHERE id=1;"
sleep "$RATE_WAIT"
assert_http "rate / #1" "200" "$CLIENT_IP" "/"
assert_http "rate / #2" "200" "$CLIENT_IP" "/"
assert_http "rate / #3 blocked" "403" "$CLIENT_IP" "/"

echo "[cc-local] === 3) custom allow /api bypasses system CC ==="
sleep "$RATE_WAIT"
assert_http "custom allow /api x3" "200" "$CLIENT_IP" "/api"
assert_http "custom allow /api x3" "200" "$CLIENT_IP" "/api"
assert_http "custom allow /api x3" "200" "$CLIENT_IP" "/api"

echo "[cc-local] === 4) prefix matcher /admin + req_rate ==="
sync_site "UPDATE site SET cc_default_rule=10004, update_at=NOW() WHERE id=1;"
sleep "$RATE_WAIT"
assert_http "admin prefix #1" "200" "$CLIENT_IP" "/admin/panel"
assert_http "admin prefix #2" "200" "$CLIENT_IP" "/admin/settings"
assert_http "admin prefix #3 blocked" "403" "$CLIENT_IP" "/admin/logs"
assert_http "non-admin path unaffected" "200" "$CLIENT_IP" "/public"

echo "[cc-local] === 5) per-uri rate on /static* (independent URIs) ==="
sync_site "UPDATE site SET cc_default_rule=10005, update_at=NOW() WHERE id=1;"
sleep "$RATE_WAIT"
assert_http "static/a #1" "200" "$CLIENT_IP" "/static/a"
assert_http "static/a #2" "200" "$CLIENT_IP" "/static/a"
assert_http "static/a #3 blocked" "403" "$CLIENT_IP" "/static/a"
assert_http "static/b still allowed" "200" "$CLIENT_IP" "/static/b"

echo "[cc-local] === 6) AND matcher uri=/secure + method=GET ==="
sync_site "UPDATE site SET cc_default_rule=10006, update_at=NOW() WHERE id=1;"
sleep "$SYNC_WAIT"
assert_http "secure GET blocked" "403" "$CLIENT_IP" "/secure"
code=$(curl -s -o /tmp/cc_local_body.txt -w '%{http_code}' \
  -X POST -H "Host: ${HOST}" -H "X-Forwarded-For: ${CLIENT_IP}" \
  "${BASE_URL}/secure" || echo "000")
if [[ "$code" == "200" ]]; then
  echo "[PASS] secure POST not matched -> HTTP 200"
  pass=$((pass + 1))
else
  echo "[FAIL] secure POST not matched -> HTTP $code, want 200"
  fail=$((fail + 1))
fi

echo "[cc-local] === 7) UA matcher BadBot + instant block filter ==="
sync_site "UPDATE site SET cc_default_rule=10007, update_at=NOW() WHERE id=1;"
sleep "$SYNC_WAIT"
assert_http "normal UA allowed" "200" "$CLIENT_IP" "/probe" "Mozilla/5.0"
assert_http "BadBot blocked" "403" "$CLIENT_IP" "/probe" "BadBot/1.0 scanner"

echo "[cc-local] === 8) dual filter: filter2 log prevents block on /dual ==="
sync_site "UPDATE site SET cc_default_rule=10003, update_at=NOW() WHERE id=1;"
sleep "$RATE_WAIT"
assert_http "dual /dual #1" "200" "$CLIENT_IP" "/dual"
assert_http "dual /dual #2" "200" "$CLIENT_IP" "/dual"
assert_http "dual /dual #3 still allowed (filter2 log)" "200" "$CLIENT_IP" "/dual"

echo "[cc-local] === 9) whitelist bypasses CC ==="
sync_site "UPDATE site SET cc_default_rule=10001, white_ip='[\"10.0.0.99\"]', update_at=NOW() WHERE id=1;"
sleep "$SYNC_WAIT"
burst_requests 10 "10.0.0.99" "/"
assert_http "whitelist burst still 200" "200" "10.0.0.99" "/"

if [[ "$fail" -gt 0 ]]; then
  echo "[cc-local] FAILED pass=$pass fail=$fail"
  $COMPOSE logs --no-color edge origin | tail -n 50
  exit 1
fi

echo "[cc-local] SUCCESS pass=$pass fail=$fail (CC intercepts at edge; origin protected)"
