#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BASE_URL="${BASE_URL:-http://127.0.0.1:5035}"

ADMIN_USER="${ADMIN_USER:-cnn_ai_admin}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"
NORMAL_USER="${NORMAL_USER:-cnn_ai_user}"
NORMAL_PASS="${NORMAL_PASS:-user123}"

ADMIN_USER_ID="${ADMIN_USER_ID:-2}"
NORMAL_USER_ID="${NORMAL_USER_ID:-3}"
NORMAL_PACKAGE_ID="${NORMAL_PACKAGE_ID:-2}"
RUN_BASE_SMOKE="${RUN_BASE_SMOKE:-0}"
SKIP_EXTERNAL_DNS="${SKIP_EXTERNAL_DNS:-0}"
STRICT_EXTERNAL_DNS="${STRICT_EXTERNAL_DNS:-0}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
EXTERNAL_DNS_SKIPPED=0
EXTERNAL_DNS_REASON=""
EXIT_EXTERNAL_DEPENDENCY=20

json_get() {
  local url="$1"
  local token="${2:-}"
  if [[ -n "$token" ]]; then
    curl -sS "${BASE_URL}${url}" -H "Authorization: Bearer ${token}"
  else
    curl -sS "${BASE_URL}${url}"
  fi
}

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

json_put() {
  local url="$1"
  local body="$2"
  local token="${3:-}"
  curl -sS -X PUT "${BASE_URL}${url}" -H 'Content-Type: application/json' -H "Authorization: Bearer ${token}" -d "$body"
}

json_delete() {
  local url="$1"
  local token="${2:-}"
  curl -sS -X DELETE "${BASE_URL}${url}" -H "Authorization: Bearer ${token}"
}

json_delete_body() {
  local url="$1"
  local body="$2"
  local token="${3:-}"
  curl -sS -X DELETE "${BASE_URL}${url}" -H 'Content-Type: application/json' -H "Authorization: Bearer ${token}" -d "$body"
}

assert_code_eq() {
  local json="$1"
  local expected="$2"
  if ! jq -e . >/dev/null 2>&1 <<<"$json"; then
    echo "Invalid JSON response (expected code=${expected}):"
    echo "$json"
    exit 1
  fi
  local got
  got="$(jq -r '.code' <<<"$json")"
  if [[ "$got" != "$expected" ]]; then
    echo "Expected code=${expected}, got code=${got}"
    echo "$json" | jq .
    exit 1
  fi
}

assert_code_ne() {
  local json="$1"
  local bad="$2"
  if ! jq -e . >/dev/null 2>&1 <<<"$json"; then
    echo "Invalid JSON response (expected code!=${bad}):"
    echo "$json"
    exit 1
  fi
  local got
  got="$(jq -r '.code' <<<"$json")"
  if [[ "$got" == "$bad" ]]; then
    echo "Expected code!=${bad}, got ${got}"
    echo "$json" | jq .
    exit 1
  fi
}

try_skip_external_dns_failure() {
  local phase="$1"
  local json="$2"
  if ! jq -e . >/dev/null 2>&1 <<<"$json"; then
    if [[ "$STRICT_EXTERNAL_DNS" == "1" ]]; then
      echo "external dns dependency failure at ${phase} (STRICT_EXTERNAL_DNS=1, non-json response)"
      echo "$json"
      exit "$EXIT_EXTERNAL_DEPENDENCY"
    fi

    EXTERNAL_DNS_SKIPPED=1
    EXTERNAL_DNS_REASON="${phase}:non_json_response"
    echo "external dns dependency unavailable at ${phase}; continue with controlled skip (non-json response)"
    return 0
  fi

  local code
  code="$(jq -r '.code // ""' <<<"$json")"

  if [[ "$STRICT_EXTERNAL_DNS" == "1" ]]; then
    echo "external dns dependency failure at ${phase} (STRICT_EXTERNAL_DNS=1)"
    echo "$json" | jq .
    exit "$EXIT_EXTERNAL_DEPENDENCY"
  fi

  case "$code" in
    401|40401)
      EXTERNAL_DNS_SKIPPED=1
      EXTERNAL_DNS_REASON="${phase}:code=${code}"
      echo "external dns dependency unavailable at ${phase}; continue with controlled skip (code=${code})"
      return 0
      ;;
  esac

  return 1
}

wait_cert_disabled() {
  local cert_id="$1"
  local keyword="$2"
  for _ in {1..30}; do
    local cert_list
    cert_list="$(json_get "/api/v1/admin/certs?keyword=${keyword}&page=1&limit=50" "$admin_token")"
    if [[ "$(jq -r '.code' <<<"$cert_list")" != "200" ]]; then
      echo "cert list failed while waiting disable:"
      echo "$cert_list" | jq .
      exit 1
    fi

    local enabled_state
    enabled_state="$(jq -r --argjson cid "$cert_id" '.data.list[] | select(.id==$cid) | .enable' <<<"$cert_list" | head -n1)"
    if [[ "$enabled_state" == "false" ]]; then
      return 0
    fi
    sleep 1
  done

  echo "timeout waiting cert disabled: id=${cert_id}"
  return 1
}

if [[ "$RUN_BASE_SMOKE" == "1" ]]; then
  echo "[A/15] Run base smoke"
  "$ROOT_DIR/scripts/smoke_live_mysql.sh"
else
  echo "[A/15] Skip base smoke (RUN_BASE_SMOKE=${RUN_BASE_SMOKE})"
fi

echo "[B/15] Login admin + user"
admin_login="$(json_post '/api/v1/admin/login' "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}")"
assert_code_eq "$admin_login" "200"
admin_token="$(jq -r '.data.token // ""' <<<"$admin_login")"
[[ -n "$admin_token" ]] || { echo "missing admin token"; exit 1; }

user_login="$(json_post '/api/v1/user/login' "{\"username\":\"${NORMAL_USER}\",\"password\":\"${NORMAL_PASS}\"}")"
assert_code_eq "$user_login" "200"
user_token="$(jq -r '.data.token // ""' <<<"$user_login")"
[[ -n "$user_token" ]] || { echo "missing user token"; exit 1; }

stamp="$(date +%s)"
dns_name="smoke-dns-${stamp}"
dns_remark="smoke-dns-remark-${stamp}"
dns_remark_updated="smoke-dns-remark-updated-${stamp}"
cert_name="smoke-cert-${stamp}"
cert_name_updated="smoke-cert-updated-${stamp}"
cert_domain="cert-${stamp}.test"
acl_name="smoke-acl-${stamp}"
acl_name_updated="smoke-acl-updated-${stamp}"
cc_group_name="smoke-cc-group-${stamp}"
cc_group_name_updated="smoke-cc-group-updated-${stamp}"
cc_matcher_name="smoke-cc-matcher-${stamp}"
cc_matcher_name_updated="smoke-cc-matcher-updated-${stamp}"
cc_filter_name="smoke-cc-filter-${stamp}"
cc_filter_name_updated="smoke-cc-filter-updated-${stamp}"
task_domain="smoke-task-${stamp}.test"
plan_name="smoke-plan-${stamp}"
plan_name_updated="smoke-plan-updated-${stamp}"
node_name="smoke-node-${stamp}"
node_name_updated="smoke-node-updated-${stamp}"
node_ip="10.88.$((stamp % 200)).$((stamp % 240 + 10))"
dns_provider_name="smoke-provider-${stamp}"
dns_provider_name_updated="smoke-provider-updated-${stamp}"
forward_group_name="smoke-forward-group-${stamp}"
forward_group_name_updated="smoke-forward-group-updated-${stamp}"
forward_listen_initial="5$((stamp % 10000 + 1000))"
forward_listen_updated="6$((stamp % 10000 + 1000))"
apply_cert_domain="smoke-apply-cert-${stamp}.test"

region_list="$(json_get '/api/v1/admin/regions' "$admin_token")"
assert_code_eq "$region_list" "200"
region_id="$(jq -r '.data.list[0].id // 0' <<<"$region_list")"
[[ "$region_id" != "0" ]] || { echo "missing region id"; exit 1; }

node_group_list="$(json_get "/api/v1/admin/node-groups?region_id=${region_id}&page=1&limit=20" "$admin_token")"
assert_code_eq "$node_group_list" "200"
node_group_id="$(jq -r '.data.list[0].id // 0' <<<"$node_group_list")"
[[ "$node_group_id" != "0" ]] || { echo "missing node group id"; exit 1; }

echo "[C/15] DNS API types + CRUD + user permission boundary"
dns_types="$(json_get '/api/v1/admin/dnsapi/types' "$admin_token")"
assert_code_eq "$dns_types" "200"

dns_create_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$dns_name" \
  --arg remark "$dns_remark" \
  --arg type "huawei" \
  --arg auth "{\"access_key_id\":\"ak-smoke\",\"secret_access_key\":\"sk-smoke\"}" \
  '{user_id:$user_id,name:$name,remark:$remark,type:$type,auth:$auth}')"
dns_create="$(json_post '/api/v1/admin/dnsapi' "$dns_create_payload" "$admin_token")"
assert_code_eq "$dns_create" "200"
dns_id="$(jq -r '.data.id // ""' <<<"$dns_create")"
[[ -n "$dns_id" && "$dns_id" != "null" ]] || { echo "missing dns id"; exit 1; }

dns_update_payload="$(jq -n \
  --arg name "$dns_name" \
  --arg remark "$dns_remark_updated" \
  --arg type "huawei" \
  --arg auth "{\"access_key_id\":\"ak-smoke-2\",\"secret_access_key\":\"sk-smoke-2\"}" \
  '{name:$name,remark:$remark,type:$type,auth:$auth}')"
dns_update="$(json_put "/api/v1/admin/dnsapi/${dns_id}" "$dns_update_payload" "$admin_token")"
assert_code_eq "$dns_update" "200"

dns_list="$(json_get "/api/v1/admin/dnsapi?keyword=${dns_name}" "$admin_token")"
assert_code_eq "$dns_list" "200"
dns_list_id="$(jq -r --arg n "$dns_name" '.data.list[] | select(.name==$n) | .id' <<<"$dns_list" | head -n1)"
[[ "$dns_list_id" == "$dns_id" ]] || { echo "dns list lookup failed"; exit 1; }

user_forbidden_dns_delete="$(json_delete '/api/v1/user/dnsapi/1' "$user_token")"
assert_code_ne "$user_forbidden_dns_delete" "200"

echo "[D/15] Cert default settings + upload cert CRUD"
cert_default_set_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg type "system" \
  --argjson dnsapi "$dns_id" \
  '{user_id:$user_id,type:$type,dnsapi:$dnsapi}')"
cert_default_set="$(json_post '/api/v1/admin/certs/default_settings' "$cert_default_set_payload" "$admin_token")"
assert_code_eq "$cert_default_set" "200"

cert_default_get="$(json_get "/api/v1/admin/certs/default_settings?user_id=${NORMAL_USER_ID}" "$admin_token")"
assert_code_eq "$cert_default_get" "200"
dnsapi_in_default="$(jq -r '.data.dnsapi // 0' <<<"$cert_default_get")"
[[ "$dnsapi_in_default" == "$dns_id" ]] || { echo "cert default dnsapi mismatch"; exit 1; }

openssl req -x509 -newkey rsa:2048 -sha256 -days 7 -nodes \
  -keyout "$tmp_dir/cert.key" \
  -out "$tmp_dir/cert.pem" \
  -subj "/CN=${cert_domain}" >/dev/null 2>&1

cert_pem="$(cat "$tmp_dir/cert.pem")"
key_pem="$(cat "$tmp_dir/cert.key")"

cert_create_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$cert_name" \
  --arg des "smoke-cert" \
  --arg type "upload" \
  --arg domain "$cert_domain" \
  --argjson dnsapi "$dns_id" \
  --arg cert "$cert_pem" \
  --arg key "$key_pem" \
  '{user_id:$user_id,name:$name,des:$des,type:$type,domain:$domain,dnsapi:$dnsapi,cert:$cert,key:$key,auto_renew:true}')"
cert_create="$(json_post '/api/v1/admin/certs' "$cert_create_payload" "$admin_token")"
assert_code_eq "$cert_create" "200"
cert_id="$(jq -r '.data.id // ""' <<<"$cert_create")"
[[ -n "$cert_id" && "$cert_id" != "null" ]] || { echo "missing cert id"; exit 1; }

cert_update_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$cert_name_updated" \
  --arg des "smoke-cert-updated" \
  --arg type "upload" \
  --arg domain "$cert_domain" \
  --argjson dnsapi "$dns_id" \
  --arg cert "$cert_pem" \
  --arg key "$key_pem" \
  '{user_id:$user_id,name:$name,des:$des,type:$type,domain:$domain,dnsapi:$dnsapi,cert:$cert,key:$key,auto_renew:false}')"
cert_update="$(json_put "/api/v1/admin/certs/${cert_id}" "$cert_update_payload" "$admin_token")"
assert_code_eq "$cert_update" "200"

download_code="$(curl -sS -o "$tmp_dir/cert.zip" -w '%{http_code}' "${BASE_URL}/api/v1/admin/certs/${cert_id}/download" -H "Authorization: Bearer ${admin_token}")"
[[ "$download_code" == "200" ]] || { echo "cert download failed: ${download_code}"; exit 1; }
[[ -s "$tmp_dir/cert.zip" ]] || { echo "cert download empty"; exit 1; }

cert_disable="$(json_post '/api/v1/admin/certs/batch_action' "{\"action\":\"disable\",\"ids\":[${cert_id}]}" "$admin_token")"
assert_code_eq "$cert_disable" "200"
wait_cert_disabled "$cert_id" "$cert_name_updated"

cert_delete=""
for _ in {1..6}; do
  cert_delete="$(json_post '/api/v1/admin/certs/batch_action' "{\"action\":\"delete\",\"ids\":[${cert_id}]}" "$admin_token")"
  cert_delete_code="$(jq -r '.code' <<<"$cert_delete")"
  if [[ "$cert_delete_code" == "200" ]]; then
    break
  fi
  if [[ "$cert_delete_code" != "40903" ]]; then
    break
  fi
  sleep 1
done
assert_code_eq "$cert_delete" "200"

echo "[E/15] ACL CRUD"
acl_create_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$acl_name" \
  --arg des "smoke-acl" \
  --arg default_action "allow" \
  --argjson enable true \
  --argjson deny_status 403 \
  --arg redirect_url "" \
  '{user_id:$user_id,name:$name,des:$des,default_action:$default_action,enable:$enable,default_deny_status:$deny_status,default_redirect_url:$redirect_url,rules:[{conditions:[{item:"ip",operator:"match",value:"127.0.0.1"}],action:"deny",deny_status:403,redirect_url:""}]}')"
acl_create="$(json_post '/api/v1/admin/rules/acl' "$acl_create_payload" "$admin_token")"
assert_code_eq "$acl_create" "200"
acl_id="$(jq -r '.data.id // ""' <<<"$acl_create")"
[[ -n "$acl_id" && "$acl_id" != "null" ]] || { echo "missing acl id"; exit 1; }

acl_get="$(json_get "/api/v1/admin/rules/acl/${acl_id}" "$admin_token")"
assert_code_eq "$acl_get" "200"

acl_update_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$acl_name_updated" \
  --arg des "smoke-acl-updated" \
  --arg default_action "deny" \
  --argjson enable false \
  --argjson deny_status 444 \
  --arg redirect_url "https://example.test/deny" \
  '{user_id:$user_id,name:$name,des:$des,default_action:$default_action,enable:$enable,default_deny_status:$deny_status,default_redirect_url:$redirect_url,rules:[{conditions:[{item:"path",operator:"contain",value:"/admin"}],action:"deny",deny_status:444,redirect_url:"https://example.test/deny"}]}')"
acl_update="$(json_put "/api/v1/admin/rules/acl/${acl_id}" "$acl_update_payload" "$admin_token")"
assert_code_eq "$acl_update" "200"

acl_delete="$(json_delete "/api/v1/admin/rules/acl/${acl_id}" "$admin_token")"
assert_code_eq "$acl_delete" "200"

echo "[F/15] Global config (WAF/error page/resource limits)"
global_get="$(json_get '/api/v1/admin/global_config' "$admin_token")"
assert_code_eq "$global_get" "200"
error_page_403="<html><body>smoke-403-${stamp}</body></html>"
global_update_payload="$(jq -c \
  --arg e403 "$error_page_403" \
  '.data
    | .waf.enable = true
    | .waf.blacklist_timeout = 4321
    | .error_pages["403"] = $e403
    | .resources.website.max_blacklist_ips = 77' <<<"$global_get")"
global_update="$(json_post '/api/v1/admin/global_config' "$global_update_payload" "$admin_token")"
assert_code_eq "$global_update" "200"

global_verify="$(json_get '/api/v1/admin/global_config' "$admin_token")"
assert_code_eq "$global_verify" "200"
blacklist_timeout="$(jq -r '.data.waf.blacklist_timeout // 0' <<<"$global_verify")"
[[ "$blacklist_timeout" == "4321" ]] || { echo "global config blacklist_timeout mismatch"; exit 1; }
max_blacklist_ips="$(jq -r '.data.resources.website.max_blacklist_ips // 0' <<<"$global_verify")"
[[ "$max_blacklist_ips" == "77" ]] || { echo "global config max_blacklist_ips mismatch"; exit 1; }
page_403="$(jq -r '.data.error_pages["403"] // ""' <<<"$global_verify")"
[[ "$page_403" == "$error_page_403" ]] || { echo "global config error page mismatch"; exit 1; }

echo "[G/15] CC group/matcher/filter CRUD"
cc_group_create_payload="$(jq -n \
  --arg type "user" \
  --arg name "$cc_group_name" \
  --arg remark "smoke-cc-group" \
  --argjson user_id "$NORMAL_USER_ID" \
  '{type:$type,name:$name,remark:$remark,is_visible:true,visible_users:[$user_id],sort_order:10,rules:[{match:"path",operator:"contain",value:"/api"}],user_id:$user_id}')"
cc_group_create="$(json_post '/api/v1/admin/rules/cc/groups' "$cc_group_create_payload" "$admin_token")"
assert_code_eq "$cc_group_create" "200"

cc_group_list="$(json_get "/api/v1/admin/rules/cc/groups?name=${cc_group_name}" "$admin_token")"
assert_code_eq "$cc_group_list" "200"
cc_group_id="$(jq -r --arg n "$cc_group_name" '.data.list[] | select(.name==$n) | .id' <<<"$cc_group_list" | head -n1)"
[[ -n "$cc_group_id" && "$cc_group_id" != "null" ]] || { echo "missing cc_group id"; exit 1; }

cc_group_get="$(json_get "/api/v1/admin/rules/cc/groups/${cc_group_id}" "$admin_token")"
assert_code_eq "$cc_group_get" "200"

cc_group_update_payload="$(jq -n \
  --arg type "user" \
  --arg name "$cc_group_name_updated" \
  --arg remark "smoke-cc-group-updated" \
  --argjson user_id "$NORMAL_USER_ID" \
  '{type:$type,name:$name,remark:$remark,is_visible:true,visible_users:[$user_id],sort_order:20,rules:[{match:"ip",operator:"match",value:"127.0.0.1"}],user_id:$user_id}')"
cc_group_update="$(json_put "/api/v1/admin/rules/cc/groups/${cc_group_id}" "$cc_group_update_payload" "$admin_token")"
assert_code_eq "$cc_group_update" "200"

cc_matcher_create_payload="$(jq -n \
  --arg type "user" \
  --arg name "$cc_matcher_name" \
  --arg remark "smoke-cc-matcher" \
  --argjson user_id "$NORMAL_USER_ID" \
  '{type:$type,name:$name,remark:$remark,is_on:true,rules:[{scope:"request",key:"uri",operator:"contain",value:"/api"}],user_id:$user_id}')"
cc_matcher_create="$(json_post '/api/v1/admin/rules/cc/matchers' "$cc_matcher_create_payload" "$admin_token")"
assert_code_eq "$cc_matcher_create" "200"

cc_matcher_list="$(json_get "/api/v1/admin/rules/cc/matchers?name=${cc_matcher_name}" "$admin_token")"
assert_code_eq "$cc_matcher_list" "200"
cc_matcher_id="$(jq -r --arg n "$cc_matcher_name" '.data.list[] | select(.name==$n) | .id' <<<"$cc_matcher_list" | head -n1)"
[[ -n "$cc_matcher_id" && "$cc_matcher_id" != "null" ]] || { echo "missing cc_matcher id"; exit 1; }

cc_matcher_update_payload="$(jq -n \
  --arg type "user" \
  --arg name "$cc_matcher_name_updated" \
  --arg remark "smoke-cc-matcher-updated" \
  --argjson user_id "$NORMAL_USER_ID" \
  '{type:$type,name:$name,remark:$remark,is_on:false,rules:[{scope:"request",key:"method",operator:"eq",value:"POST"}],user_id:$user_id}')"
cc_matcher_update="$(json_put "/api/v1/admin/rules/cc/matchers/${cc_matcher_id}" "$cc_matcher_update_payload" "$admin_token")"
assert_code_eq "$cc_matcher_update" "200"

cc_filter_create_payload="$(jq -n \
  --arg type "user" \
  --arg name "$cc_filter_name" \
  --arg remark "smoke-cc-filter" \
  --arg action "deny" \
  --arg match_mode "path" \
  --argjson user_id "$NORMAL_USER_ID" \
  '{type:$type,name:$name,remark:$remark,enable:true,action:$action,match_mode:$match_mode,blacklist:true,within_second:30,max_req:200,max_req_per_uri:60,auth:{"mode":"token"},user_id:$user_id}')"
cc_filter_create="$(json_post '/api/v1/admin/rules/cc/filters' "$cc_filter_create_payload" "$admin_token")"
assert_code_eq "$cc_filter_create" "200"

cc_filter_list="$(json_get "/api/v1/admin/rules/cc/filters?name=${cc_filter_name}" "$admin_token")"
assert_code_eq "$cc_filter_list" "200"
cc_filter_id="$(jq -r --arg n "$cc_filter_name" '.data.list[] | select(.name==$n) | .id' <<<"$cc_filter_list" | head -n1)"
[[ -n "$cc_filter_id" && "$cc_filter_id" != "null" ]] || { echo "missing cc_filter id"; exit 1; }

cc_filter_update_payload="$(jq -n \
  --arg type "user" \
  --arg name "$cc_filter_name_updated" \
  --arg remark "smoke-cc-filter-updated" \
  --arg action "captcha" \
  --arg match_mode "host" \
  '{type:$type,name:$name,remark:$remark,enable:false,action:$action,match_mode:$match_mode,blacklist:false,within_second:15,max_req:80,max_req_per_uri:20,auth:{"mode":"cookie"}}')"
cc_filter_update="$(json_put "/api/v1/admin/rules/cc/filters/${cc_filter_id}" "$cc_filter_update_payload" "$admin_token")"
assert_code_eq "$cc_filter_update" "200"

cc_filter_delete="$(json_delete "/api/v1/admin/rules/cc/filters/${cc_filter_id}" "$admin_token")"
assert_code_eq "$cc_filter_delete" "200"
cc_matcher_delete="$(json_delete "/api/v1/admin/rules/cc/matchers/${cc_matcher_id}" "$admin_token")"
assert_code_eq "$cc_matcher_delete" "200"
cc_group_delete="$(json_delete "/api/v1/admin/rules/cc/groups/${cc_group_id}" "$admin_token")"
assert_code_eq "$cc_group_delete" "200"

echo "[H/15] Purge + preheat tasks and usage accounting"
task_site_create="$(json_post '/api/v1/admin/sites' "{\"user_id\":${NORMAL_USER_ID},\"user_package_id\":${NORMAL_PACKAGE_ID},\"domains\":[\"${task_domain}\"],\"backends\":[\"3.3.3.3\"]}" "$admin_token")"
assert_code_eq "$task_site_create" "200"
task_site_id="$(jq -r '.data.id // ""' <<<"$task_site_create")"
[[ -n "$task_site_id" && "$task_site_id" != "null" ]] || { echo "missing task site id"; exit 1; }

usage_before="$(json_get "/api/v1/admin/tasks/usage?user_id=${NORMAL_USER_ID}" "$admin_token")"
assert_code_eq "$usage_before" "200"
used_refresh_before="$(jq -r '.data.used.refresh_url // 0' <<<"$usage_before")"
used_preheat_before="$(jq -r '.data.used.preheat // 0' <<<"$usage_before")"

task_refresh="$(json_post '/api/v1/admin/tasks' "{\"type\":\"refresh_url\",\"urls\":\"https://${task_domain}/index.html\",\"user_id\":${NORMAL_USER_ID}}" "$admin_token")"
assert_code_eq "$task_refresh" "200"
task_preheat="$(json_post '/api/v1/admin/tasks' "{\"type\":\"preheat\",\"urls\":\"https://${task_domain}/video.mp4\",\"user_id\":${NORMAL_USER_ID}}" "$admin_token")"
assert_code_eq "$task_preheat" "200"

usage_after="$(json_get "/api/v1/admin/tasks/usage?user_id=${NORMAL_USER_ID}" "$admin_token")"
assert_code_eq "$usage_after" "200"
used_refresh_after="$(jq -r '.data.used.refresh_url // 0' <<<"$usage_after")"
used_preheat_after="$(jq -r '.data.used.preheat // 0' <<<"$usage_after")"
(( used_refresh_after >= used_refresh_before + 1 )) || { echo "refresh_url usage not increased"; exit 1; }
(( used_preheat_after >= used_preheat_before + 1 )) || { echo "preheat usage not increased"; exit 1; }

task_list="$(json_get "/api/v1/admin/tasks?keyword=${task_domain}&type=refresh_url&page=1&pageSize=20" "$admin_token")"
assert_code_eq "$task_list" "200"
task_id="$(jq -r '.list[0].id // .data.list[0].id // ""' <<<"$task_list")"
[[ -n "$task_id" && "$task_id" != "null" ]] || { echo "missing task id for resubmit"; exit 1; }
task_resubmit="$(json_post "/api/v1/admin/tasks/${task_id}/resubmit" '{}' "$admin_token")"
assert_code_eq "$task_resubmit" "200"

echo "[I/15] Site cache config save/load/compile"
cache_get_before="$(json_get "/api/sites/${task_site_id}/cache" "$admin_token")"
assert_code_eq "$cache_get_before" "200"

cache_save_payload="$(jq -n \
  '{profiles:{Static:{ttl:7200,ignore_query:true,force_cache:true,query_ignore_list:["utm_*","fbclid"]}},rules:[{path_prefix:"/static/",profile:"Static"}]}')"
cache_save="$(json_post "/api/sites/${task_site_id}/cache?compile=true" "$cache_save_payload" "$admin_token")"
assert_code_eq "$cache_save" "200"
cache_compiled_site_id="$(jq -r '.data.compiled.site_id // 0' <<<"$cache_save")"
[[ "$cache_compiled_site_id" == "$task_site_id" ]] || { echo "cache compiled site_id mismatch"; exit 1; }

cache_get_after="$(json_get "/api/sites/${task_site_id}/cache" "$admin_token")"
assert_code_eq "$cache_get_after" "200"
cache_static_ttl="$(jq -r '.data.config.profiles | to_entries | map(select((.key|ascii_downcase)=="static")) | .[0].value.ttl // 0' <<<"$cache_get_after")"
[[ "$cache_static_ttl" == "7200" ]] || { echo "cache static ttl mismatch"; exit 1; }

cache_compile="$(json_post "/api/sites/${task_site_id}/cache/compile" '{}' "$admin_token")"
assert_code_eq "$cache_compile" "200"
cache_host_found="$(jq -r --arg h "$task_domain" '.data.hosts // [] | map(select(.==$h)) | length' <<<"$cache_compile")"
(( cache_host_found >= 1 )) || { echo "cache compile hosts missing task domain"; echo "$cache_compile" | jq .; exit 1; }

echo "[J/15] Plan + user_plan + package sync"
plan_create_payload="$(jq -n \
  --arg name "$plan_name" \
  --arg desc "smoke plan" \
  --arg cname_domain "cnn-ai.test" \
  --arg cname_mode "site" \
  --argjson region "$region_id" \
  --argjson line_group "$node_group_id" \
  '{name:$name,desc:$desc,region:$region,line_group:$line_group,backup_group:0,cname_domain:$cname_domain,cname_mode:$cname_mode,price_monthly:10,price_quarterly:20,price_yearly:30,traffic_limit:1024,bandwidth_limit:"100",connection_limit:200,domain_limit:50,http_port:80,stream_port:0,buy_num_limit:1,id_verify:false,before_exp_days_renew:7,websocket:true,custom_cc_rules:true,l2_origin:false,sort_order:9,status:true,owner:"smoke"}')"
plan_create="$(json_post '/api/v1/admin/plans' "$plan_create_payload" "$admin_token")"
assert_code_eq "$plan_create" "200"

plan_list="$(json_get '/api/v1/admin/plans' "$admin_token")"
assert_code_eq "$plan_list" "200"
plan_id="$(jq -r --arg n "$plan_name" '.data.list[] | select(.name==$n) | .id' <<<"$plan_list" | head -n1)"
[[ -n "$plan_id" && "$plan_id" != "null" ]] || { echo "missing plan id"; exit 1; }

plan_update_payload="$(jq -n --arg name "$plan_name_updated" --arg cname_domain "cnn-ai.test" --argjson region "$region_id" --argjson line_group "$node_group_id" '{name:$name,cname_domain:$cname_domain,region:$region,line_group:$line_group,traffic_limit:2048,domain_limit:80,status:true}')"
plan_update="$(json_put "/api/v1/admin/plans/${plan_id}" "$plan_update_payload" "$admin_token")"
assert_code_eq "$plan_update" "200"

assign_user_plan="$(json_post '/api/v1/admin/user_plans/assign' "{\"plan_id\":${plan_id},\"user_id\":${NORMAL_USER_ID},\"duration_months\":1}" "$admin_token")"
assert_code_eq "$assign_user_plan" "200"

user_plan_list="$(json_get '/api/v1/admin/user_plans' "$admin_token")"
assert_code_eq "$user_plan_list" "200"
user_plan_id="$(jq -r --argjson uid "$NORMAL_USER_ID" --argjson pid "$plan_id" '.data.list | map(select(.user_id==$uid and .package_id==$pid)) | sort_by(.id) | last | .id // ""' <<<"$user_plan_list")"
[[ -n "$user_plan_id" && "$user_plan_id" != "null" ]] || { echo "missing user_plan id"; exit 1; }

user_pkg_before="$(json_get "/api/v1/admin/user_packages?user_id=${NORMAL_USER_ID}" "$admin_token")"
assert_code_eq "$user_pkg_before" "200"
pkg_version_before="$(jq -r --argjson id "$user_plan_id" '.data.list[] | select(.id==$id) | .version // 0' <<<"$user_pkg_before" | head -n1)"
[[ -n "$pkg_version_before" ]] || pkg_version_before="0"

user_plan_update_payload="$(jq -n '{cname_mode:"site",cname_domain:"cnn-ai.test",main_domain_limit:66,enable_backup_group:false}')"
user_plan_update="$(json_put "/api/v1/admin/user_plans/${user_plan_id}" "$user_plan_update_payload" "$admin_token")"
assert_code_eq "$user_plan_update" "200"

user_pkg_after="$(json_get "/api/v1/admin/user_packages?user_id=${NORMAL_USER_ID}" "$admin_token")"
assert_code_eq "$user_pkg_after" "200"
pkg_version_after="$(jq -r --argjson id "$user_plan_id" '.data.list[] | select(.id==$id) | .version // 0' <<<"$user_pkg_after" | head -n1)"
[[ -n "$pkg_version_after" ]] || { echo "missing pkg version after"; exit 1; }
(( pkg_version_after > pkg_version_before )) || { echo "user package version not bumped"; exit 1; }

sync_task_list="$(json_get "/api/v1/admin/tasks?keyword=${user_plan_id}&page=1&pageSize=20" "$admin_token")"
assert_code_eq "$sync_task_list" "200"
sync_total="$(jq -r '.total // .data.total // 0' <<<"$sync_task_list")"
(( sync_total >= 1 )) || { echo "missing sync task records"; exit 1; }

delete_user_plan="$(json_delete_body '/api/v1/admin/user_plans' "{\"ids\":[${user_plan_id}]}" "$admin_token")"
assert_code_eq "$delete_user_plan" "200"
delete_plan="$(json_delete "/api/v1/admin/plans/${plan_id}" "$admin_token")"
assert_code_eq "$delete_plan" "200"

echo "[K/15] Forward proxy (group/default/stream) CRUD"
forward_group_create_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$forward_group_name" \
  --arg remark "smoke-forward-group" \
  '{user_id:$user_id,name:$name,remark:$remark}')"
forward_group_create="$(json_post '/api/v1/admin/forward_groups' "$forward_group_create_payload" "$admin_token")"
assert_code_eq "$forward_group_create" "200"
forward_group_id="$(jq -r '.data.id // ""' <<<"$forward_group_create")"
[[ -n "$forward_group_id" && "$forward_group_id" != "null" ]] || { echo "missing forward_group id"; exit 1; }

forward_group_update_payload="$(jq -n \
  --argjson id "$forward_group_id" \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$forward_group_name_updated" \
  --arg remark "smoke-forward-group-updated" \
  '{id:$id,user_id:$user_id,name:$name,remark:$remark}')"
forward_group_update="$(json_put '/api/v1/admin/forward_groups' "$forward_group_update_payload" "$admin_token")"
assert_code_eq "$forward_group_update" "200"

forward_default_create_payload="$(jq -n \
  --arg key "listen_protocol" \
  --arg value "tcp" \
  --arg scope "global" \
  --argjson group_id "$forward_group_id" \
  '{key:$key,value:$value,scope:$scope,group_id:$group_id}')"
forward_default_create="$(json_post '/api/v1/admin/forward_defaults' "$forward_default_create_payload" "$admin_token")"
assert_code_eq "$forward_default_create" "200"

forward_default_list="$(json_get '/api/v1/admin/forward_defaults' "$admin_token")"
assert_code_eq "$forward_default_list" "200"
forward_default_id="$(jq -r --arg key "listen_protocol" --argjson gid "$forward_group_id" '.data.list[] | select(.key==$key and .group_id==$gid) | .id' <<<"$forward_default_list" | head -n1)"
[[ -n "$forward_default_id" && "$forward_default_id" != "null" ]] || { echo "missing forward default id"; exit 1; }

forward_create_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --argjson user_package_id "$NORMAL_PACKAGE_ID" \
  --argjson node_group_id "$node_group_id" \
  --argjson group_id "$forward_group_id" \
  --arg listen "$forward_listen_initial" \
  --arg origin_input "1.1.1.1:8080" \
  --arg remark "smoke-forward" \
  '{user_id:$user_id,user_package_id:$user_package_id,node_group_id:$node_group_id,group_id:$group_id,listen_ports:[$listen],origin_input:$origin_input,remark:$remark}')"
forward_create="$(json_post '/api/v1/admin/forwards' "$forward_create_payload" "$admin_token")"
assert_code_eq "$forward_create" "200"
forward_id="$(jq -r '.data.id // ""' <<<"$forward_create")"
[[ -n "$forward_id" && "$forward_id" != "null" ]] || { echo "missing forward id"; exit 1; }

forward_update_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --argjson user_package_id "$NORMAL_PACKAGE_ID" \
  --argjson group_id "$forward_group_id" \
  --arg listen "$forward_listen_updated" \
  --arg origin_input "1.1.1.2:8081" \
  --arg remark "smoke-forward-updated" \
  '{user_id:$user_id,user_package_id:$user_package_id,group_id:$group_id,listen_ports_input:$listen,origin_input:$origin_input,remark:$remark}')"
forward_update="$(json_put "/api/v1/admin/forwards/${forward_id}" "$forward_update_payload" "$admin_token")"
assert_code_eq "$forward_update" "200"

forward_batch_update_payload="$(jq -n \
  --argjson fid "$forward_id" \
  '{ids:[$fid],settings:{origin:{balance_way:"least_conn",proxy_protocol:true,backsource_port:"8082",origins:[{address:"1.1.1.3:8082",weight:1,enable:true}]}}}')"
forward_batch_update="$(json_post '/api/v1/admin/forwards/batch_update' "$forward_batch_update_payload" "$admin_token")"
assert_code_eq "$forward_batch_update" "200"

forward_list_verify="$(json_get "/api/v1/admin/forwards?user_id=${NORMAL_USER_ID}&search_field=forward_id&keyword=${forward_id}&page=1&pageSize=20" "$admin_token")"
assert_code_eq "$forward_list_verify" "200"
forward_id_in_list="$(jq -r --argjson id "$forward_id" '.data.list[] | select(.id==$id) | .id' <<<"$forward_list_verify" | head -n1)"
[[ "$forward_id_in_list" == "$forward_id" ]] || { echo "forward not found in list"; exit 1; }

forward_disable="$(json_post '/api/v1/admin/forwards/batch_action' "{\"action\":\"disable\",\"ids\":[${forward_id}]}" "$admin_token")"
assert_code_eq "$forward_disable" "200"
forward_enable="$(json_post '/api/v1/admin/forwards/batch_action' "{\"action\":\"enable\",\"ids\":[${forward_id}]}" "$admin_token")"
assert_code_eq "$forward_enable" "200"
forward_delete="$(json_post '/api/v1/admin/forwards/batch_action' "{\"action\":\"delete\",\"ids\":[${forward_id}]}" "$admin_token")"
assert_code_eq "$forward_delete" "200"

forward_default_delete_payload="$(jq -n --argjson id "$forward_default_id" '{id:$id}')"
forward_default_delete="$(json_delete_body '/api/v1/admin/forward_defaults' "$forward_default_delete_payload" "$admin_token")"
assert_code_eq "$forward_default_delete" "200"

forward_group_delete_payload="$(jq -n --argjson id "$forward_group_id" '{id:$id}')"
forward_group_delete="$(json_delete_body '/api/v1/admin/forward_groups' "$forward_group_delete_payload" "$admin_token")"
assert_code_eq "$forward_group_delete" "200"

echo "[L/15] Site apply-cert flow"
cert_apply_site_id=""
apply_cert_created_id=0
if [[ "$SKIP_EXTERNAL_DNS" == "1" ]]; then
  EXTERNAL_DNS_SKIPPED=1
  EXTERNAL_DNS_REASON="manual_skip"
  echo "skip site apply-cert flow (SKIP_EXTERNAL_DNS=1)"
else
  cert_apply_site_create="$(json_post '/api/v1/admin/sites' "{\"user_id\":${NORMAL_USER_ID},\"user_package_id\":${NORMAL_PACKAGE_ID},\"domains\":[\"${apply_cert_domain}\"],\"backends\":[\"4.4.4.4\"]}" "$admin_token")"
  assert_code_eq "$cert_apply_site_create" "200"
  cert_apply_site_id="$(jq -r '.data.id // ""' <<<"$cert_apply_site_create")"
  [[ -n "$cert_apply_site_id" && "$cert_apply_site_id" != "null" ]] || { echo "missing cert apply site id"; exit 1; }

  apply_cert_result="$(json_post '/api/v1/admin/sites/apply_cert' "{\"ids\":[${cert_apply_site_id}]}" "$admin_token")"
  apply_cert_code="$(jq -r '.code // ""' <<<"$apply_cert_result" 2>/dev/null || true)"
  if [[ -z "$apply_cert_code" ]]; then
    if try_skip_external_dns_failure "site_apply_cert" "$apply_cert_result"; then
      apply_cert_code="skipped"
    else
      echo "unexpected non-json apply_cert response:"
      echo "$apply_cert_result"
      exit 1
    fi
  fi

  if [[ "$apply_cert_code" != "200" ]]; then
    if try_skip_external_dns_failure "site_apply_cert" "$apply_cert_result"; then
      :
    else
      assert_code_eq "$apply_cert_result" "200"
    fi
  fi

  if [[ "$apply_cert_code" == "200" ]]; then
    apply_cert_created_id="$(jq -r '.data.created_ids[0] // 0' <<<"$apply_cert_result")"
    (( apply_cert_created_id > 0 )) || { echo "site apply_cert did not create certificate"; echo "$apply_cert_result" | jq .; exit 1; }

    cert_apply_site_detail="$(json_get "/api/v1/admin/sites/${cert_apply_site_id}" "$admin_token")"
    assert_code_eq "$cert_apply_site_detail" "200"
    site_https_enabled="$(jq -r '.data.https // false' <<<"$cert_apply_site_detail")"
    [[ "$site_https_enabled" == "true" ]] || { echo "site https not enabled after apply cert"; exit 1; }
    site_cert_id="$(jq -r '.data.settings.https.certificate_id // .data.cert_id // 0' <<<"$cert_apply_site_detail")"
    (( site_cert_id > 0 )) || { echo "site certificate_id not set after apply cert"; echo "$cert_apply_site_detail" | jq .; exit 1; }
  fi
fi

echo "[M/15] Node + DNS provider CRUD"
node_create_payload="$(jq -n --argjson region_id "$region_id" --arg name "$node_name" --arg ip "$node_ip" '{region_id:$region_id,name:$name,remark:"smoke-node",ip:$ip,enable:true,check_on:false,type:1,auto_install:false}')"
node_create="$(json_post '/api/v1/admin/nodes' "$node_create_payload" "$admin_token")"
assert_code_eq "$node_create" "200"
node_id="$(jq -r '.data.id // ""' <<<"$node_create")"
[[ -n "$node_id" && "$node_id" != "null" ]] || { echo "missing node id"; exit 1; }

node_update_payload="$(jq -n --argjson region_id "$region_id" --arg name "$node_name_updated" --arg ip "$node_ip" '{region_id:$region_id,name:$name,remark:"smoke-node-updated",ip:$ip,enable:true,type:1}')"
node_update="$(json_put "/api/v1/admin/nodes/${node_id}" "$node_update_payload" "$admin_token")"
assert_code_eq "$node_update" "200"

node_disable="$(json_put "/api/v1/admin/nodes/${node_id}/status" '{"enable":false}' "$admin_token")"
assert_code_eq "$node_disable" "200"
node_enable="$(json_put "/api/v1/admin/nodes/${node_id}/status" '{"enable":true}' "$admin_token")"
assert_code_eq "$node_enable" "200"

node_anti_off="$(json_put "/api/v1/admin/nodes/${node_id}/anti_blocking" '{"enable":false}' "$admin_token")"
assert_code_eq "$node_anti_off" "200"
node_anti_on="$(json_put "/api/v1/admin/nodes/${node_id}/anti_blocking" '{"enable":true}' "$admin_token")"
assert_code_eq "$node_anti_on" "200"

node_list_verify="$(json_get "/api/v1/admin/nodes?keyword=${node_name_updated}&page=1&pageSize=20" "$admin_token")"
assert_code_eq "$node_list_verify" "200"
node_id_in_list="$(jq -r --argjson id "$node_id" '.data.list[] | select(.id==$id) | .id' <<<"$node_list_verify" | head -n1)"
[[ "$node_id_in_list" == "$node_id" ]] || { echo "node not found in list"; exit 1; }

dns_provider_create_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$dns_provider_name" \
  --arg type "huawei" \
  --arg credentials "{\"id\":\"provider-ak\",\"secret\":\"provider-sk\"}" \
  '{user_id:$user_id,name:$name,type:$type,credentials:$credentials}')"
dns_provider_create="$(json_post '/api/v1/admin/dns/providers' "$dns_provider_create_payload" "$admin_token")"
assert_code_eq "$dns_provider_create" "200"

dns_provider_list="$(json_get "/api/v1/admin/dns/providers?user_id=${NORMAL_USER_ID}" "$admin_token")"
assert_code_eq "$dns_provider_list" "200"
dns_provider_id="$(jq -r --arg n "$dns_provider_name" '.data.list[] | select(.name==$n) | .id' <<<"$dns_provider_list" | head -n1)"
[[ -n "$dns_provider_id" && "$dns_provider_id" != "null" ]] || { echo "missing dns provider id"; exit 1; }

dns_provider_update_payload="$(jq -n \
  --argjson user_id "$NORMAL_USER_ID" \
  --arg name "$dns_provider_name_updated" \
  --arg type "huawei" \
  --arg credentials "{\"id\":\"provider-ak-up\",\"secret\":\"provider-sk-up\"}" \
  '{user_id:$user_id,name:$name,type:$type,credentials:$credentials}')"
dns_provider_update="$(json_put "/api/v1/admin/dns/providers/${dns_provider_id}" "$dns_provider_update_payload" "$admin_token")"
assert_code_eq "$dns_provider_update" "200"

dns_provider_delete="$(json_delete "/api/v1/admin/dns/providers/${dns_provider_id}" "$admin_token")"
assert_code_eq "$dns_provider_delete" "200"
node_delete="$(json_delete "/api/v1/admin/nodes/${node_id}" "$admin_token")"
assert_code_eq "$node_delete" "200"

echo "[N/15] Cleanup temporary sites + DNS API"
if [[ -n "${cert_apply_site_id:-}" && "${cert_apply_site_id}" != "null" ]]; then
  cleanup_cert_site="$(json_post '/api/v1/admin/sites/batch_action' "{\"action\":\"delete\",\"ids\":[${cert_apply_site_id}]}" "$admin_token")"
  cleanup_cert_site_code="$(jq -r '.code // ""' <<<"$cleanup_cert_site")"
  if [[ "$cleanup_cert_site_code" == "200" ]]; then
    :
  elif [[ "$cleanup_cert_site_code" == "41201" ]]; then
    cleanup_cert_site_disable="$(json_post '/api/v1/admin/sites/batch_action' "{\"action\":\"disable\",\"ids\":[${cert_apply_site_id}]}" "$admin_token")"
    assert_code_eq "$cleanup_cert_site_disable" "200"
  else
    echo "unexpected cleanup cert site response:"
    echo "$cleanup_cert_site" | jq .
    exit 1
  fi
fi

if [[ -n "${task_site_id:-}" && "${task_site_id}" != "null" ]]; then
  cleanup_task_site="$(json_post '/api/v1/admin/sites/batch_action' "{\"action\":\"delete\",\"ids\":[${task_site_id}]}" "$admin_token")"
  cleanup_task_site_code="$(jq -r '.code // ""' <<<"$cleanup_task_site")"
  if [[ "$cleanup_task_site_code" == "200" ]]; then
    :
  elif [[ "$cleanup_task_site_code" == "41201" ]]; then
    cleanup_task_site_disable="$(json_post '/api/v1/admin/sites/batch_action' "{\"action\":\"disable\",\"ids\":[${task_site_id}]}" "$admin_token")"
    assert_code_eq "$cleanup_task_site_disable" "200"
  else
    echo "unexpected cleanup site response:"
    echo "$cleanup_task_site" | jq .
    exit 1
  fi
fi

dns_delete="$(json_delete "/api/v1/admin/dnsapi/${dns_id}" "$admin_token")"
assert_code_eq "$dns_delete" "200"

echo "[O/15] Extended smoke passed"
cat <<EOF
Extended smoke test passed (live MySQL API):
- base: ${BASE_URL}
- region/node_group : ${region_id}/${node_group_id}
- dnsapi tested id: ${dns_id}
- cert tested id : ${cert_id}
- acl tested id  : ${acl_id}
- task site id   : ${task_site_id}
- apply cert site : ${cert_apply_site_id}
- apply cert id   : ${apply_cert_created_id}
- plan id        : ${plan_id}
- user_plan id   : ${user_plan_id}
- forward group id: ${forward_group_id}
- forward id      : ${forward_id}
- node id        : ${node_id}
- dns provider id: ${dns_provider_id}
- external dns skipped: ${EXTERNAL_DNS_SKIPPED}
- external dns reason : ${EXTERNAL_DNS_REASON}
EOF
