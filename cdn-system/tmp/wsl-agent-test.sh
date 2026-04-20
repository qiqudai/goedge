#!/usr/bin/env bash
set -euo pipefail
TEST_DIR="/mnt/e/cdn/goedge/cdn-system/tmp/edge-test"
LOG="/mnt/e/cdn/goedge/cdn-system/tmp/wsl-agent-test.log"
rm -f "$LOG"
exec > "$LOG" 2>&1
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR/app"
cp "/mnt/e/cdn/goedge/cdn-system/build/linux-amd64/agent/cdn-agent" "$TEST_DIR/app/cdn-agent"
chmod 755 "$TEST_DIR/app/cdn-agent"
cat > "$TEST_DIR/app/agent.json" <<EOF
{
  "api": "http://127.0.0.1:8080",
  "token": "dummy-token",
  "node_id": "test-node",
  "reset_resources": true,
  "bootstrap_sync": false,
  "bootstrap_start": false
}
EOF
( timeout 2s "$TEST_DIR/app/cdn-agent" -config "$TEST_DIR/app/agent.json" ) || true
echo "APP:"
ls -la "$TEST_DIR/app" || true
echo "EDGE:"
ls -la "$TEST_DIR/app/edge-node" || true
