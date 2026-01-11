#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
TARGET_DIR="/www/wwwroot/www"

if [ ! -d "$DIST_DIR" ]; then
  echo "dist not found; run npm run build first" >&2
  exit 1
fi

mkdir -p "$TARGET_DIR"
cp -a "$DIST_DIR/." "$TARGET_DIR/"
