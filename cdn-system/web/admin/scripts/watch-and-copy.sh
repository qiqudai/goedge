#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

cleanup() {
  if [ -n "${BUILD_PID:-}" ]; then
    kill "$BUILD_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

cd "$ROOT_DIR"
npm run build -- --watch > /tmp/vite-build-watch.log 2>&1 &
BUILD_PID=$!

node scripts/sync-wwwroot.cjs --watch
