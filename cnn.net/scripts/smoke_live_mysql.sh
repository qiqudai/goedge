#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:5035}"

ADMIN_USER="${ADMIN_USER:-cnn_ai_admin}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"
NORMAL_USER="${NORMAL_USER:-cnn_ai_user}"
NORMAL_PASS="${NORMAL_PASS:-user123}"

ADMIN_USER_ID="${ADMIN_USER_ID:-2}"
NORMAL_USER_ID="${NORMAL_USER_ID:-3}"
ADMIN_PACKAGE_ID="${ADMIN_PACKAGE_ID:-1}"
NORMAL_PACKAGE_ID="${NORMAL_PACKAGE_ID:-2}"
DNS_PROVIDER_ID="${DNS_PROVIDER_ID:-1}"

stamp="$(date +%s)"
admin_cname_from="smoke-${stamp}-a.test"
admin_cname_to="smoke-${stamp}-b.test"
admin_site_domain="smoke-admin-${stamp}.test"
user_site_domain="smoke-user-${stamp}.test"
forbidden_domain="smoke-forbidden-${stamp}.test"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

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

json_get() {
  local url="$1"
  local token="${2:-}"
  if [[ -n "$token" ]]; then
    curl -sS "${BASE_URL}${url}" -H "Authorization: Bearer ${token}"
  else
    curl -sS "${BASE_URL}${url}"
  fi
}

http_code() {
  local url="$1"
  curl -sS -o /dev/null -w '%{http_code}' "${BASE_URL}${url}"
}

assert_code_eq() {
  local json="$1"
  local expected="$2"
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
  local not_expected="$2"
  local got
  got="$(jq -r '.code' <<<"$json")"
  if [[ "$got" == "$not_expected" ]]; then
    echo "Expected code!=${not_expected}, got code=${got}"
    echo "$json" | jq .
    exit 1
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "Expected string to contain '${needle}', got: ${haystack}"
    exit 1
  fi
}

wait_site_disabled() {
  local scope="$1"
  local site_id="$2"
  local token="$3"

  for _ in {1..20}; do
    local detail
    detail="$(json_get "/api/v1/${scope}/sites/${site_id}" "$token")"
    local code
    code="$(jq -r '.code' <<<"$detail")"
    if [[ "$code" == "200" ]]; then
      local status
      status="$(jq -r 'if (.data | has("status")) then .data.status else "missing" end' <<<"$detail")"
      if [[ "$status" == "false" ]]; then
        return 0
      fi
    fi
    sleep 1
  done

  echo "site disable not applied in time: scope=${scope}, site_id=${site_id}"
  exit 1
}

echo "[1/8] Health + static assets"
health_code="$(http_code /api/health)"
[[ "$health_code" == "200" ]] || { echo "health failed: ${health_code}"; exit 1; }
css_code="$(http_code /css/site.css)"
[[ "$css_code" == "200" ]] || { echo "css failed: ${css_code}"; exit 1; }
js_code="$(http_code /js/app.js)"
[[ "$js_code" == "200" ]] || { echo "js failed: ${js_code}"; exit 1; }

echo "[2/8] Admin login + unauthorized check"
unauthorized_admin_sites="$(json_get '/api/v1/admin/sites?page=1&pageSize=10')"
assert_code_ne "$unauthorized_admin_sites" "200"

admin_login="$(json_post '/api/v1/admin/login' "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}")"
assert_code_eq "$admin_login" "200"
admin_token="$(jq -r '.data.token // ""' <<<"$admin_login")"
[[ -n "$admin_token" ]] || { echo "missing admin token"; exit 1; }

echo "[3/8] Admin defaults + cname CRUD"
set_global_default="$(json_post '/api/v1/admin/site_defaults' '{"name":"backend_protocol","value":"http","scope_name":"global","scope_id":0}' "$admin_token")"
assert_code_eq "$set_global_default" "200"

set_user_default="$(json_post '/api/v1/admin/site_defaults' "{\"user_id\":${NORMAL_USER_ID},\"name\":\"backend_protocol\",\"value\":\"https\",\"scope_name\":\"user\",\"scope_id\":${NORMAL_USER_ID}}" "$admin_token")"
assert_code_eq "$set_user_default" "200"

create_cname="$(json_post '/api/v1/admin/cname_domains' "{\"domain\":\"${admin_cname_from}\",\"note\":\"smoke-create\",\"dns_provider_id\":${DNS_PROVIDER_ID}}" "$admin_token")"
assert_code_eq "$create_cname" "200"

cname_list="$(json_get '/api/v1/admin/cname_domains' "$admin_token")"
assert_code_eq "$cname_list" "200"
cname_id="$(jq -r --arg d "$admin_cname_from" '.data.list[] | select(.domain==$d) | .id' <<<"$cname_list" | head -n1)"
[[ -n "$cname_id" ]] || { echo "failed to find created cname"; exit 1; }

update_cname="$(json_put "/api/v1/admin/cname_domains/${cname_id}" "{\"domain\":\"${admin_cname_to}\",\"note\":\"smoke-update\",\"dns_provider_id\":${DNS_PROVIDER_ID}}" "$admin_token")"
assert_code_eq "$update_cname" "200"

echo "[4/8] Admin site create/update/cname/delete"
create_site="$(json_post '/api/v1/admin/sites' "{\"user_id\":${NORMAL_USER_ID},\"user_package_id\":${NORMAL_PACKAGE_ID},\"domains\":[\"${admin_site_domain}\"],\"backends\":[\"1.1.1.1\"]}" "$admin_token")"
assert_code_eq "$create_site" "200"
admin_site_id="$(jq -r '.data.id // ""' <<<"$create_site")"
[[ -n "$admin_site_id" && "$admin_site_id" != "null" ]] || { echo "missing admin site id"; exit 1; }

admin_site_detail="$(json_get "/api/v1/admin/sites/${admin_site_id}" "$admin_token")"
assert_code_eq "$admin_site_detail" "200"
default_protocol="$(jq -r '.data.settings.backsource.protocol // ""' <<<"$admin_site_detail")"
[[ "$default_protocol" == "https" ]] || { echo "expected user default https, got ${default_protocol}"; exit 1; }

site_override="$(json_put "/api/v1/admin/sites/${admin_site_id}" '{"settings":{"backsource":{"protocol":"grpc"}}}' "$admin_token")"
assert_code_eq "$site_override" "200"

site_after_override="$(json_get "/api/v1/admin/sites/${admin_site_id}" "$admin_token")"
assert_code_eq "$site_after_override" "200"
override_protocol="$(jq -r '.data.settings.backsource.protocol // ""' <<<"$site_after_override")"
[[ "$override_protocol" == "grpc" ]] || { echo "expected site override grpc, got ${override_protocol}"; exit 1; }

batch_cname="$(json_post '/api/v1/admin/sites/batch_update' "{\"ids\":[${admin_site_id}],\"cname_domain\":\"${admin_cname_to}\",\"cname_mode\":\"site\"}" "$admin_token")"
assert_code_eq "$batch_cname" "200"

site_after_cname="$(json_get "/api/v1/admin/sites/${admin_site_id}" "$admin_token")"
assert_code_eq "$site_after_cname" "200"
site_cname="$(jq -r '.data.cname // ""' <<<"$site_after_cname")"
assert_contains "$site_cname" "$admin_cname_to"

disable_site="$(json_post '/api/v1/admin/sites/batch_action' "{\"action\":\"disable\",\"ids\":[${admin_site_id}]}" "$admin_token")"
assert_code_eq "$disable_site" "200"
wait_site_disabled "admin" "$admin_site_id" "$admin_token"
delete_site="$(json_post '/api/v1/admin/sites/batch_action' "{\"action\":\"delete\",\"ids\":[${admin_site_id}]}" "$admin_token")"
assert_code_eq "$delete_site" "200"

echo "[5/8] User login + permission boundary"
user_login="$(json_post '/api/v1/user/login' "{\"username\":\"${NORMAL_USER}\",\"password\":\"${NORMAL_PASS}\"}")"
assert_code_eq "$user_login" "200"
user_token="$(jq -r '.data.token // ""' <<<"$user_login")"
[[ -n "$user_token" ]] || { echo "missing user token"; exit 1; }

forbidden_admin_cname="$(json_get '/api/v1/admin/cname_domains' "$user_token")"
forbidden_code="$(jq -r '.code' <<<"$forbidden_admin_cname")"
[[ "$forbidden_code" == "403" || "$forbidden_code" == "4011" || "$forbidden_code" == "40301" ]] || {
  echo "expected permission denied for user->admin cname, got ${forbidden_code}"
  echo "$forbidden_admin_cname" | jq .
  exit 1
}

echo "[6/8] User defaults + package isolation"
user_set_default="$(json_post '/api/v1/user/site_defaults' "{\"user_id\":${NORMAL_USER_ID},\"name\":\"backend_protocol\",\"value\":\"https\",\"scope_name\":\"global\",\"scope_id\":${NORMAL_USER_ID}}" "$user_token")"
assert_code_eq "$user_set_default" "200"

user_forbidden_create="$(json_post '/api/v1/user/sites' "{\"user_id\":${NORMAL_USER_ID},\"user_package_id\":${ADMIN_PACKAGE_ID},\"domains\":[\"${forbidden_domain}\"],\"backends\":[\"2.2.2.2\"]}" "$user_token")"
user_forbidden_code="$(jq -r '.code' <<<"$user_forbidden_create")"
[[ "$user_forbidden_code" == "403" || "$user_forbidden_code" == "4011" || "$user_forbidden_code" == "40301" ]] || {
  echo "expected permission denied for package isolation, got ${user_forbidden_code}"
  echo "$user_forbidden_create" | jq .
  exit 1
}

echo "[7/8] User site create/update/delete"
user_create_site="$(json_post '/api/v1/user/sites' "{\"user_id\":${NORMAL_USER_ID},\"user_package_id\":${NORMAL_PACKAGE_ID},\"domains\":[\"${user_site_domain}\"],\"backends\":[\"2.2.2.2\"]}" "$user_token")"
assert_code_eq "$user_create_site" "200"
user_site_id="$(jq -r '.data.id // ""' <<<"$user_create_site")"
[[ -n "$user_site_id" && "$user_site_id" != "null" ]] || { echo "missing user site id"; exit 1; }

user_site_detail="$(json_get "/api/v1/user/sites/${user_site_id}" "$user_token")"
assert_code_eq "$user_site_detail" "200"
user_default_protocol="$(jq -r '.data.settings.backsource.protocol // ""' <<<"$user_site_detail")"
[[ "$user_default_protocol" == "https" ]] || { echo "expected user default https, got ${user_default_protocol}"; exit 1; }

user_override="$(json_put "/api/v1/user/sites/${user_site_id}" '{"settings":{"backsource":{"protocol":"grpc"}}}' "$user_token")"
assert_code_eq "$user_override" "200"

user_after_override="$(json_get "/api/v1/user/sites/${user_site_id}" "$user_token")"
assert_code_eq "$user_after_override" "200"
user_override_protocol="$(jq -r '.data.settings.backsource.protocol // ""' <<<"$user_after_override")"
[[ "$user_override_protocol" == "grpc" ]] || { echo "expected user override grpc, got ${user_override_protocol}"; exit 1; }

user_disable="$(json_post '/api/v1/user/sites/batch_action' "{\"action\":\"disable\",\"ids\":[${user_site_id}]}" "$user_token")"
assert_code_eq "$user_disable" "200"
wait_site_disabled "user" "$user_site_id" "$user_token"
user_delete="$(json_post '/api/v1/user/sites/batch_action' "{\"action\":\"delete\",\"ids\":[${user_site_id}]}" "$user_token")"
assert_code_eq "$user_delete" "200"

echo "[8/8] Logout simulation"
post_logout_access="$(json_get '/api/v1/user/sites?page=1&pageSize=10')"
assert_code_ne "$post_logout_access" "200"

cat <<EOF
Smoke test passed (live MySQL API):
- base: ${BASE_URL}
- admin: ${ADMIN_USER}
- user : ${NORMAL_USER}
- created/deleted admin site id: ${admin_site_id}
- created/deleted user site id : ${user_site_id}
EOF
