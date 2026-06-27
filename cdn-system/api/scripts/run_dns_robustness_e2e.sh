#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE_DIR="$ROOT/docker/cc-e2e"
API_DIR="$ROOT/api"
BASE_URL="${ADMIN_URL:-http://127.0.0.1:18088}"
ADMIN_USER="${CDN_ADMIN_USER:-dnsadmin}"
ADMIN_PASS="${CDN_ADMIN_PASS:-dns-e2e-pass}"
GROUP_ID="${GROUP_ID:-1}"
LINE_ID="${LINE_ID:-default}"
CYCLES="${CYCLES:-5}"
DB_DSN="root:cc_test_root@tcp(127.0.0.1:13306)/cdnfy?charset=utf8mb4&parseTime=True&loc=Local"

echo "[dns-e2e] starting cc-e2e stack (api :18088, mysql :13306)..."
cd "$COMPOSE_DIR"
docker compose -f docker-compose.yml -f docker-compose.dns-e2e.yml up -d --build mysql origin api

echo "[dns-e2e] waiting for api health..."
for i in $(seq 1 90); do
  if curl -sf "$BASE_URL/health" >/dev/null; then
    echo "[dns-e2e] api is healthy"
    break
  fi
  if [[ "$i" -eq 90 ]]; then
    echo "[dns-e2e] api health check timed out" >&2
    exit 1
  fi
  sleep 2
done

echo "[dns-e2e] seeding dns robustness fixtures..."
docker compose -f docker-compose.yml -f docker-compose.dns-e2e.yml exec -T mysql \
  mysql -uroot -pcc_test_root cdnfy < "$API_DIR/scripts/dns_robustness_seed.sql"

echo "[dns-e2e] creating admin user..."
(
  cd "$API_DIR"
  go run ./cmd/init_admin -config ./scripts/dns_e2e_local.yaml "$ADMIN_USER" "$ADMIN_PASS" "dns-e2e@local.test"
) || (
  cd "$API_DIR"
  FORCE=1 go run ./cmd/init_admin -config ./scripts/dns_e2e_local.yaml "$ADMIN_USER" "$ADMIN_PASS" "dns-e2e@local.test"
)

echo "[dns-e2e] cleaning bogus init_admin users (if any)..."
docker compose -f docker-compose.yml -f docker-compose.dns-e2e.yml exec -T mysql \
  mysql -uroot -pcc_test_root cdnfy -e "DELETE FROM user WHERE name IN ('-config','-db');" 2>/dev/null || true

echo "[dns-e2e] logging in..."
TOKEN="$(
  curl -sf "$BASE_URL/api/v1/admin/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"dns-e2e@local.test\",\"password\":\"$ADMIN_PASS\"}" \
  | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("token") or (d.get("data") or {}).get("token",""))'
)"
if [[ -z "$TOKEN" ]]; then
  echo "[dns-e2e] failed to obtain admin token" >&2
  exit 1
fi

echo "[dns-e2e] running unit tests..."
(
  cd "$API_DIR"
  go test ./services/dns/... -count=1
)

echo "[dns-e2e] running api chaos test (${CYCLES} cycles)..."
(
  cd "$API_DIR"
  ADMIN_URL="$BASE_URL" \
  ADMIN_TOKEN="$TOKEN" \
  GROUP_ID="$GROUP_ID" \
  LINE_ID="$LINE_ID" \
  CYCLES="$CYCLES" \
  go run ./scripts/dns_robustness
)

echo "[dns-e2e] all dns robustness checks passed"
