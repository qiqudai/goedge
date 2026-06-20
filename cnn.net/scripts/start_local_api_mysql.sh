#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
PID_FILE="$RUNTIME_DIR/api-mysql.pid"
LOG_FILE="$RUNTIME_DIR/api-mysql.log"

APP_URLS="${ASPNETCORE_URLS:-http://127.0.0.1:5035}"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-123456}"
MYSQL_DB="${MYSQL_DB:-cdnfy}"
JWT_SECRET="${Jwt__Secret:-cnn-local-jwt-secret-32bytes-min-length-2026}"
CLICKHOUSE_DSN="${ClickHouse__Dsn:-${ClickHouse__HttpDsn:-}}"

mkdir -p "$RUNTIME_DIR"

if [[ -f "$PID_FILE" ]]; then
  old_pid="$(cat "$PID_FILE" || true)"
  if [[ -n "${old_pid}" ]] && kill -0 "$old_pid" >/dev/null 2>&1; then
    echo "Cnn.Api (MySQL) already running, pid=${old_pid}"
    exit 0
  fi
  rm -f "$PID_FILE"
fi

"$ROOT_DIR/scripts/bootstrap_local_mysql.sh" >/dev/null

echo "Building Cnn.Api ..."
dotnet build "$ROOT_DIR/src/Cnn.Api/Cnn.Api.csproj" --nologo >/dev/null

DLL_PATH="$ROOT_DIR/src/Cnn.Api/bin/Debug/net9.0/Cnn.Api.dll"
if [[ ! -f "$DLL_PATH" ]]; then
  echo "Build output missing: $DLL_PATH"
  exit 1
fi

echo "Starting Cnn.Api (MySQL) in background ..."
if [[ -z "$CLICKHOUSE_DSN" ]]; then
  if curl -sS "http://127.0.0.1:8123/?query=SELECT%201" >/dev/null 2>&1; then
    CLICKHOUSE_DSN="http://127.0.0.1:8123/default"
  fi
fi

ASPNETCORE_ENVIRONMENT=Development \
ASPNETCORE_URLS="$APP_URLS" \
Database__Provider=mysql \
ConnectionStrings__Default="server=${MYSQL_HOST};port=${MYSQL_PORT};database=${MYSQL_DB};user=${MYSQL_USER};password=${MYSQL_PASSWORD};" \
Jwt__Secret="$JWT_SECRET" \
ClickHouse__Dsn="$CLICKHOUSE_DSN" \
nohup dotnet "$DLL_PATH" >"$LOG_FILE" 2>&1 &

pid="$!"
echo "$pid" >"$PID_FILE"

echo "Waiting for health endpoint ..."
for _ in {1..30}; do
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    echo "Cnn.Api exited unexpectedly. Recent log:"
    tail -n 120 "$LOG_FILE" || true
    exit 1
  fi

  code="$(curl -s -o /dev/null -w '%{http_code}' "$APP_URLS/api/health" || true)"
  if [[ "$code" == "200" ]]; then
    echo "Cnn.Api started successfully"
    echo "URL: $APP_URLS"
    echo "PID: $pid"
    echo "LOG: $LOG_FILE"
    if [[ -n "$CLICKHOUSE_DSN" ]]; then
      echo "CK : $CLICKHOUSE_DSN"
    fi
    echo "Admin: cnn_ai_admin / admin123"
    echo "User : cnn_ai_user / user123"
    exit 0
  fi
  sleep 1
done

echo "Health check timeout. Recent log:"
tail -n 120 "$LOG_FILE" || true
exit 1
