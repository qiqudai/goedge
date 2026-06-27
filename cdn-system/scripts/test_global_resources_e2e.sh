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
TMP_JSON="$(mktemp)"

pass=0
fail=0

cleanup() {
  rm -f "$TMP_JSON"
  $COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

assert_eq() {
  local name="$1"
  local got="$2"
  local want="$3"
  if [[ "$got" == "$want" ]]; then
    echo "[PASS] $name"
    pass=$((pass + 1))
  else
    echo "[FAIL] $name -> got=$got want=$want"
    fail=$((fail + 1))
  fi
}

assert_status() {
  local name="$1"
  local expect="$2"
  local client_ip="${3:-10.0.0.50}"
  local code
  code=$(curl -s -o /tmp/gr_e2e_body.txt -w '%{http_code}' \
    -H "Host: ${HOST}" \
    -H "X-Forwarded-For: ${client_ip}" \
    "${BASE_URL}/" || echo "000")
  if [[ "$code" == "$expect" ]]; then
    echo "[PASS] $name -> HTTP $code"
    pass=$((pass + 1))
  else
    echo "[FAIL] $name -> HTTP $code, want $expect"
    echo "       body: $(tr '\n' ' ' < /tmp/gr_e2e_body.txt | head -c 200)"
    fail=$((fail + 1))
  fi
}

wait_edge() {
  local i
  for i in $(seq 1 90); do
    code=$(curl -s -o /dev/null -w '%{http_code}' -H "Host: ${HOST}" "${BASE_URL}/nginx-health" || echo "000")
    if [[ "$code" == "200" ]]; then
      echo "[global-resources-e2e] edge ready"
      return 0
    fi
    sleep 2
  done
  echo "[global-resources-e2e] timeout waiting for edge" >&2
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

write_global_config() {
  python3 - "$TMP_JSON" "$@" <<'PY'
import json, sys
out_path = sys.argv[1]
resources = json.loads(sys.argv[2])
waf = json.loads(sys.argv[3]) if len(sys.argv) > 3 and sys.argv[3] else {"enable": True, "default_block_action": "page"}
payload = {
    "resources": resources,
    "waf": waf,
    "nginx": {"worker_processes": "auto"},
    "default_config": {"website": {}, "api": {}, "download": {}},
}
with open(out_path, "w", encoding="utf-8") as f:
    json.dump(payload, f, ensure_ascii=False)
PY
  local sql
  sql=$(python3 - <<PY
raw=open("$TMP_JSON", encoding="utf-8").read().replace("'", "''")
print(f"DELETE FROM config WHERE name='global_config' AND type='system';")
print(f"INSERT INTO config (name, value, type, scope_id, scope_name, create_at, update_at, enable) VALUES ('global_config', '{raw}', 'system', 0, 'global', NOW(), NOW(), 1);")
PY
)
  $MYSQL -e "$sql" >/dev/null
  $MYSQL -e "UPDATE site SET update_at = NOW(3) WHERE id = 1;" >/dev/null
}

fetch_agent_config() {
  $COMPOSE exec -T edge wget -qO- \
    --header="Authorization: Bearer ${AGENT_TOKEN}" \
    "http://api:8080/api/v1/agent/config?node_id=1" 2>/dev/null || true
}

recreate_edge() {
  $COMPOSE up -d --no-deps --force-recreate edge >/dev/null
  wait_edge
  sleep 8
}

blacklist_count() {
  local cfg="$1"
  CFG="$cfg" python3 - <<'PY'
import json, os
raw = os.environ.get("CFG", "")
try:
    data = json.loads(raw)
except Exception:
    print(0)
    raise SystemExit
domains = data.get("domains") or []
if not domains:
    print(0)
else:
    print(len(domains[0].get("black_ips") or []))
PY
}

echo "[global-resources-e2e] preparing edge build context"
prepare_edge_stage

echo "[global-resources-e2e] starting stack"
$COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
if ! docker image inspect cc-e2e-api:latest >/dev/null 2>&1; then
  $COMPOSE build mysql origin api
else
  $COMPOSE build api
fi
docker build -f "$COMPOSE_DIR/Dockerfile.edge" -t cc-e2e-edge "$COMPOSE_DIR"
$COMPOSE up -d --no-build
$COMPOSE up -d --no-deps --force-recreate edge
wait_edge || exit 1
sleep 8

echo "[global-resources-e2e] phase 1: max_blacklist_ips trims edge config"
write_global_config '{"website":{"max_blacklist_ips":2,"max_whitelist_ips":50,"default_listen_80":true,"min_limit":1000,"max_limit_multiplier":200,"daily_url_purge_limit":2000,"daily_dir_purge_limit":500,"daily_preload_limit":2000,"preload_timeout":120,"daily_unlock_ip_limit":1000,"unlock_ip_batch_limit":50,"max_cc_rules_per_group":5,"max_acl_rules":5,"daily_log_download_limit":10,"log_storage_dir":"/data/download-temp/","log_storage_hours":12,"max_domains_per_site":100},"forward":{"disabled_ports":"80 443","min_limit":1000,"max_limit_multiplier":200,"max_acl_rules":10},"public":{"disabled_custom_ports":"22 5000","allowed_custom_ports":"1-65535"}}'
$MYSQL -e "UPDATE site SET black_ip = '[\"1.1.1.1\",\"2.2.2.2\",\"3.3.3.3\"]', update_at = NOW() WHERE id = 1;" >/dev/null
recreate_edge
cfg=$(fetch_agent_config)
count=$(blacklist_count "$cfg")
assert_eq "blacklist trimmed to 2 on edge" "$count" "2"

echo "[global-resources-e2e] phase 2: default_listen_80=false in agent payload"
write_global_config '{"website":{"max_blacklist_ips":50,"default_listen_80":false,"min_limit":1000,"max_limit_multiplier":200,"daily_url_purge_limit":2000,"daily_dir_purge_limit":500,"daily_preload_limit":2000,"preload_timeout":120,"daily_unlock_ip_limit":1000,"unlock_ip_batch_limit":50,"max_cc_rules_per_group":5,"max_acl_rules":5,"daily_log_download_limit":10,"log_storage_dir":"/data/download-temp/","log_storage_hours":12,"max_domains_per_site":100},"forward":{"disabled_ports":"80 443","min_limit":1000,"max_limit_multiplier":200,"max_acl_rules":10},"public":{"disabled_custom_ports":"22 5000","allowed_custom_ports":"1-65535"}}'
recreate_edge
cfg=$(fetch_agent_config)
if echo "$cfg" | grep -q '"default_listen_80":false'; then
  echo "[PASS] default_listen_80=false in agent payload"
  pass=$((pass + 1))
else
  echo "[FAIL] default_listen_80=false missing in agent payload"
  fail=$((fail + 1))
fi

echo "[global-resources-e2e] phase 3: global WAF blacklist blocks client IP"
write_global_config '{"website":{"max_blacklist_ips":50,"default_listen_80":true,"min_limit":1000,"max_limit_multiplier":200,"daily_url_purge_limit":2000,"daily_dir_purge_limit":500,"daily_preload_limit":2000,"preload_timeout":120,"daily_unlock_ip_limit":1000,"unlock_ip_batch_limit":50,"max_cc_rules_per_group":5,"max_acl_rules":5,"daily_log_download_limit":10,"log_storage_dir":"/data/download-temp/","log_storage_hours":12,"max_domains_per_site":100},"forward":{"disabled_ports":"80 443","min_limit":1000,"max_limit_multiplier":200,"max_acl_rules":10},"public":{"disabled_custom_ports":"22 5000","allowed_custom_ports":"1-65535"}}' '{"enable":true,"default_block_action":"page","blacklist_ips":"10.0.0.50","whitelist_ips":""}'
$MYSQL -e "UPDATE site SET cc_default_rule = 0, update_at = NOW() WHERE id = 1;" >/dev/null
recreate_edge
cfg=$(fetch_agent_config)
if echo "$cfg" | grep -q '"blacklist_ips":"10.0.0.50"'; then
  echo "[PASS] global WAF blacklist present in agent config"
  pass=$((pass + 1))
else
  echo "[FAIL] global WAF blacklist missing in agent config"
  fail=$((fail + 1))
fi
echo "[global-resources-e2e] WAF CIDR matching covered by docker/acceptance-e2e/scripts/test_lua_waf_cidr.sh"

echo "[global-resources-e2e] phase 4: custom purge limit value reaches agent resources"
write_global_config '{"website":{"daily_url_purge_limit":7,"max_blacklist_ips":50,"default_listen_80":true,"min_limit":1000,"max_limit_multiplier":200,"daily_dir_purge_limit":500,"daily_preload_limit":2000,"preload_timeout":120,"daily_unlock_ip_limit":1000,"unlock_ip_batch_limit":50,"max_cc_rules_per_group":5,"max_acl_rules":5,"daily_log_download_limit":10,"log_storage_dir":"/data/download-temp/","log_storage_hours":12,"max_domains_per_site":100},"forward":{"disabled_ports":"80 443","min_limit":1000,"max_limit_multiplier":200,"max_acl_rules":10},"public":{"disabled_custom_ports":"22 5000","allowed_custom_ports":"1-65535"}}'
recreate_edge
cfg=$(fetch_agent_config)
if echo "$cfg" | grep -q '"daily_url_purge_limit":7'; then
  echo "[PASS] daily_url_purge_limit=7 in agent resources"
  pass=$((pass + 1))
else
  echo "[FAIL] daily_url_purge_limit=7 missing in agent resources"
  fail=$((fail + 1))
fi

echo ""
echo "[global-resources-e2e] results pass=$pass fail=$fail"
if [[ "$fail" -gt 0 ]]; then
  exit 1
fi
echo "[global-resources-e2e] ALL PASSED"
