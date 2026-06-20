#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_DIR="$ROOT/docker/cc-e2e"
COMPOSE="docker compose -f $COMPOSE_DIR/docker-compose.yml"
PORT="${CC_E2E_PORT:-18089}"
HOST="cc-test.local"
BASE_URL="http://127.0.0.1:${PORT}"
AGENT_TOKEN="cc-e2e-agent-token"
MYSQL="$COMPOSE exec -T mysql mysql -uroot -pcc_test_root cdnfy"
TEST_CLIENT_IP="${CC_E2E_CLIENT_IP:-10.0.0.50}"
WHITELIST_IP="${CC_E2E_WHITELIST_IP:-10.0.0.99}"
ACL_ALLOW_IP="${CC_E2E_ACL_ALLOW_IP:-10.0.0.60}"

pass=0
fail=0

cleanup() {
  $COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

assert_status() {
  local name="$1"
  local expect="$2"
  local client_ip="${3:-$TEST_CLIENT_IP}"
  local path="${4:-/}"
  local code
  code=$(curl -s -o /tmp/cc_e2e_body.txt -w '%{http_code}' \
    -H "Host: ${HOST}" \
    -H "X-Forwarded-For: ${client_ip}" \
    "${BASE_URL}${path}" || echo "000")
  if [[ "$code" == "$expect" ]]; then
    echo "[PASS] $name -> HTTP $code"
    pass=$((pass + 1))
  else
    echo "[FAIL] $name -> HTTP $code, want $expect"
    echo "       body: $(tr '\n' ' ' < /tmp/cc_e2e_body.txt | head -c 200)"
    fail=$((fail + 1))
  fi
}

wait_edge() {
  local i
  for i in $(seq 1 90); do
    code=$(curl -s -o /dev/null -w '%{http_code}' -H "Host: ${HOST}" "${BASE_URL}/nginx-health" || echo "000")
    if [[ "$code" == "200" ]]; then
      echo "[cc-e2e] edge ready"
      return 0
    fi
    sleep 2
  done
  echo "[cc-e2e] timeout waiting for edge" >&2
  return 1
}

prepare_edge_stage() {
  local stage="$COMPOSE_DIR/stage"
  rm -rf "$stage"
  mkdir -p "$stage/lua" "$stage/conf/guard"
  cp -R "$ROOT/agent/edge-node/lua/." "$stage/lua/"
  cp -R "$ROOT/agent/edge-node/conf/guard/." "$stage/conf/guard/" 2>/dev/null || true
  cp "$COMPOSE_DIR/edge/nginx.conf" "$stage/nginx.conf"
  cp "$COMPOSE_DIR/edge/entrypoint.sh" "$stage/entrypoint.sh"
}

echo "[cc-e2e] preparing edge build context"
prepare_edge_stage

echo "[cc-e2e] building and starting stack (edge :${PORT})"
$COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true

if [[ "${CC_E2E_FULL_BUILD:-0}" == "1" ]]; then
  $COMPOSE up -d --build
else
  if ! docker image inspect cc-e2e-api:latest >/dev/null 2>&1; then
    echo "[cc-e2e] first run: building mysql/origin/api images"
    $COMPOSE build mysql origin api
  fi
  docker build -f "$COMPOSE_DIR/Dockerfile.edge" -t cc-e2e-edge "$COMPOSE_DIR"
  $COMPOSE up -d --no-build
  $COMPOSE up -d --no-deps --force-recreate edge
fi

wait_edge || {
  $COMPOSE logs --no-color edge api mysql | tail -n 80
  exit 1
}

echo "[cc-e2e] verify control plane pushes CC rule to edge config"
cfg=$($COMPOSE exec -T edge wget -qO- \
  --header="Authorization: Bearer ${AGENT_TOKEN}" \
  "http://api:8080/api/v1/agent/config?node_id=1" 2>/dev/null || true)

if echo "$cfg" | grep -q '"cc_rule_id":10001'; then
  echo "[PASS] API config includes cc_rule_id=10001"
  pass=$((pass + 1))
else
  echo "[FAIL] API config missing cc_rule_id=10001"
  echo "       snippet: $(echo "$cfg" | head -c 300)"
  fail=$((fail + 1))
fi

echo "[cc-e2e] phase 1: CC ON -> node intercepts on 3rd refresh (client ${TEST_CLIENT_IP})"
assert_status "CC ON refresh #1" "200"
assert_status "CC ON refresh #2" "200"
assert_status "CC ON refresh #3 blocked" "403"

echo "[cc-e2e] phase 2: disable CC in control plane, sync to edge"
$MYSQL -e "UPDATE site SET cc_default_rule = 0, update_at = NOW() WHERE id = 1;" >/dev/null
sleep 8

assert_status "CC OFF refresh #1" "200"
assert_status "CC OFF refresh #2" "200"
assert_status "CC OFF refresh #3 still allowed" "200"

echo "[cc-e2e] phase 3: re-enable CC after rate window resets"
sleep 6
$MYSQL -e "UPDATE site SET cc_default_rule = 10001, update_at = NOW() WHERE id = 1;" >/dev/null
sleep 8

assert_status "CC ON again #1" "200"
assert_status "CC ON again #2" "200"
assert_status "CC ON again #3 blocked" "403"

echo "[cc-e2e] phase 4: whitelist IP bypasses CC (access_guard.lua)"
sleep 6
$MYSQL -e "UPDATE site SET white_ip = '[\"${WHITELIST_IP}\"]', update_at = NOW() WHERE id = 1;" >/dev/null
sleep 8

cfg=$($COMPOSE exec -T edge wget -qO- \
  --header="Authorization: Bearer ${AGENT_TOKEN}" \
  "http://api:8080/api/v1/agent/config?node_id=1" 2>/dev/null || true)
if echo "$cfg" | grep -q "\"${WHITELIST_IP}\""; then
  echo "[PASS] API config includes whitelist IP ${WHITELIST_IP}"
  pass=$((pass + 1))
else
  echo "[FAIL] API config missing whitelist IP ${WHITELIST_IP}"
  echo "       snippet: $(echo "$cfg" | head -c 400)"
  fail=$((fail + 1))
fi

assert_status "whitelist refresh #1" "200" "$WHITELIST_IP"
assert_status "whitelist refresh #2" "200" "$WHITELIST_IP"
assert_status "whitelist refresh #3" "200" "$WHITELIST_IP"
assert_status "whitelist refresh #4" "200" "$WHITELIST_IP"
assert_status "whitelist refresh #5" "200" "$WHITELIST_IP"

assert_status "non-whitelist refresh #1" "200" "$TEST_CLIENT_IP"
assert_status "non-whitelist refresh #2" "200" "$TEST_CLIENT_IP"
assert_status "non-whitelist refresh #3 blocked" "403" "$TEST_CLIENT_IP"

echo "[cc-e2e] phase 5: custom CC allow /api bypasses system CC on same client"
sleep 6
assert_status "custom allow /api #1" "200" "$TEST_CLIENT_IP" "/api"
assert_status "custom allow /api #2" "200" "$TEST_CLIENT_IP" "/api"
assert_status "custom allow /api #3 still allowed" "200" "$TEST_CLIENT_IP" "/api"
assert_status "disabled custom rule path ignored" "200" "$TEST_CLIENT_IP" "/disabled-rule"

echo "[cc-e2e] phase 6: ACL default deny with single-IP allow"
$MYSQL -e "UPDATE site SET white_ip = NULL, settings = '{\"access\":{\"acl\":10001}}', update_at = NOW() WHERE id = 1;" >/dev/null
sleep 8

assert_status "ACL allowed IP" "200" "$ACL_ALLOW_IP"
assert_status "ACL default deny IP" "403" "$TEST_CLIENT_IP"

echo "[cc-e2e] phase 7: dual filter1+filter2 — filter2 pass avoids block on /dual"
sleep 6
$MYSQL -e "UPDATE site SET cc_default_rule = 10003, settings = '{\"access\":{\"acl\":0}}', update_at = NOW() WHERE id = 1;" >/dev/null
sleep 8

assert_status "dual filter /dual #1" "200" "$TEST_CLIENT_IP" "/dual"
assert_status "dual filter /dual #2" "200" "$TEST_CLIENT_IP" "/dual"
assert_status "dual filter /dual #3 still allowed" "200" "$TEST_CLIENT_IP" "/dual"

echo "[cc-e2e] phase 8: ACL deny redirect"
$MYSQL -e "UPDATE site SET cc_default_rule = 0, settings = '{\"access\":{\"acl\":10002}}', update_at = NOW() WHERE id = 1;" >/dev/null
sleep 8

code=$(curl -s -o /tmp/cc_e2e_body.txt -w '%{http_code}' \
  -H "Host: ${HOST}" \
  -H "X-Forwarded-For: ${TEST_CLIENT_IP}" \
  "${BASE_URL}/blocked" || echo "000")
if [[ "$code" == "302" ]]; then
  echo "[PASS] ACL redirect deny -> HTTP 302"
  pass=$((pass + 1))
else
  echo "[FAIL] ACL redirect deny -> HTTP $code, want 302"
  fail=$((fail + 1))
fi

if [[ "$fail" -gt 0 ]]; then
  echo "[cc-e2e] FAILED pass=$pass fail=$fail"
  $COMPOSE logs --no-color edge api | tail -n 60
  exit 1
fi

echo "[cc-e2e] SUCCESS pass=$pass fail=$fail"
