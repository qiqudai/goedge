#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
APP_URLS="${ASPNETCORE_URLS:-http://127.0.0.1:5035}"

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-123456}"
MYSQL_DB="${MYSQL_DB:-cdnfy}"
CLICKHOUSE_DSN="${ClickHouse__Dsn:-${ClickHouse__HttpDsn:-}}"

"$ROOT_DIR/scripts/bootstrap_local_mysql.sh" >/dev/null

export ASPNETCORE_ENVIRONMENT="${ASPNETCORE_ENVIRONMENT:-Development}"
export ASPNETCORE_URLS="$APP_URLS"
export Database__Provider="mysql"
export ConnectionStrings__Default="server=${MYSQL_HOST};port=${MYSQL_PORT};database=${MYSQL_DB};user=${MYSQL_USER};password=${MYSQL_PASSWORD};"
export Jwt__Secret="${Jwt__Secret:-cnn-local-jwt-secret-32bytes-min-length-2026}"
if [[ -z "$CLICKHOUSE_DSN" ]]; then
  if curl -sS "http://127.0.0.1:8123/?query=SELECT%201" >/dev/null 2>&1; then
    CLICKHOUSE_DSN="http://127.0.0.1:8123/default"
  fi
fi
export ClickHouse__Dsn="$CLICKHOUSE_DSN"

cd "$ROOT_DIR"
echo "Starting Cnn.Api (MySQL) ..."
echo "URL: $APP_URLS"
echo "DB : ${MYSQL_USER}@${MYSQL_HOST}:${MYSQL_PORT}/${MYSQL_DB}"
if [[ -n "$CLICKHOUSE_DSN" ]]; then
  echo "CK : ${CLICKHOUSE_DSN}"
fi
echo "Admin credentials: cnn_ai_admin / admin123"
echo "User credentials : cnn_ai_user / user123"

exec dotnet run --no-launch-profile --project src/Cnn.Api/Cnn.Api.csproj
