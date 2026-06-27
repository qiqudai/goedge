#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

pass=0
fail=0
sections=()

run_section() {
  local name="$1"
  shift
  echo ""
  echo "============================================================"
  echo "[acceptance] $name"
  echo "============================================================"
  if "$@"; then
    echo "[acceptance][OK] $name"
    pass=$((pass + 1))
    sections+=("OK  $name")
  else
    echo "[acceptance][FAIL] $name" >&2
    fail=$((fail + 1))
    sections+=("FAIL $name")
  fi
}

run_section "API unit tests (forward/blacklist/models)" \
  bash -c 'cd api && pkgs=$(go list ./controllers/... ./models/... ./services/... | grep -v "/services/dns") && go test $pkgs -count=1'

run_section "Agent unit tests (stream + full package)" \
  bash -c 'cd agent && go test ./... -count=1'

run_section "Lua blacklist matcher smoke test" \
  bash docker/acceptance-e2e/scripts/test_lua_blacklist.sh

run_section "Lua global WAF CIDR matcher smoke test" \
  bash docker/acceptance-e2e/scripts/test_lua_waf_cidr.sh

run_section "Docker stream TCP/UDP forward e2e" \
  bash -c 'cd docker/stream-e2e && docker compose down -v --remove-orphans >/dev/null 2>&1 || true; docker compose up --abort-on-container-exit --exit-code-from test; code=$?; docker compose down -v --remove-orphans >/dev/null 2>&1 || true; exit $code'

run_section "Docker protocol compatibility e2e" \
  bash scripts/test_compat_e2e.sh

run_section "Docker cc-e2e (CC + whitelist + blacklist intercept)" \
  bash scripts/test_cc_e2e.sh

run_section "Docker global resources e2e" \
  bash scripts/test_global_resources_e2e.sh

echo ""
echo "============================================================"
echo "[acceptance] summary"
echo "============================================================"
for line in "${sections[@]}"; do
  echo "  $line"
done

if [[ "$fail" -gt 0 ]]; then
  echo "[acceptance] FAILED sections=$fail passed=$pass" >&2
  exit 1
fi

echo "[acceptance] ALL PASSED sections=$pass"
