#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DB_PATH="${CNN_LOCAL_DB:-$ROOT_DIR/.runtime/cnn-local.db}"
APP_URLS="${ASPNETCORE_URLS:-http://127.0.0.1:5035}"

"$ROOT_DIR/scripts/bootstrap_local_sqlite.sh" "$DB_PATH" >/dev/null

export ASPNETCORE_ENVIRONMENT="${ASPNETCORE_ENVIRONMENT:-Development}"
export ASPNETCORE_URLS="$APP_URLS"
export Database__Provider="sqlite"
export ConnectionStrings__Default="DataSource=$DB_PATH"
export Jwt__Secret="${Jwt__Secret:-cnn-local-jwt-secret}"

cd "$ROOT_DIR"
echo "Starting Cnn.Api ..."
echo "URL: $APP_URLS"
echo "DB : $DB_PATH"
echo "Admin credentials: admin / admin123"
echo "User credentials : user2 / user123"

exec dotnet run --project src/Cnn.Api/Cnn.Api.csproj
