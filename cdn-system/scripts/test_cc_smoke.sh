#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SMOKE_DIR="$ROOT/agent/testdata/cc_smoke"
BUILD_DIR="$SMOKE_DIR/docker-build"
IMAGE="${CC_SMOKE_IMAGE:-cc-smoke-openresty:local}"
PORT="${CC_SMOKE_PORT:-18088}"

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/lua" "$BUILD_DIR/conf/guard"

cp "$SMOKE_DIR/nginx.conf" "$BUILD_DIR/nginx.conf"
cp "$SMOKE_DIR/cdn_config.json" "$BUILD_DIR/cdn_config.json"
cp -R "$ROOT/agent/edge-node/lua/." "$BUILD_DIR/lua/"
cp -R "$ROOT/agent/edge-node/conf/guard/." "$BUILD_DIR/conf/guard/" 2>/dev/null || true

cat > "$BUILD_DIR/Dockerfile" <<'EOF'
FROM openresty/openresty:1.25.3.1-0-alpine-fat
COPY . /opt/cc/
EXPOSE 18088
CMD ["openresty", "-c", "/opt/cc/nginx.conf", "-g", "daemon off;"]
EOF

echo "[cc-smoke] building image $IMAGE"
docker build -t "$IMAGE" "$BUILD_DIR" >/tmp/cc-smoke-build.log 2>&1 || {
  tail -n 40 /tmp/cc-smoke-build.log
  exit 1
}

echo "[cc-smoke] starting origin backend on :18080"
python3 - <<'PY' &
import http.server
import socketserver

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"origin-ok\n")
    def log_message(self, fmt, *args):
        return

with socketserver.TCPServer(("0.0.0.0", 18080), Handler) as httpd:
    httpd.serve_forever()
PY
ORIGIN_PID=$!
cleanup() {
  kill "$ORIGIN_PID" 2>/dev/null || true
  docker rm -f cc-smoke-openresty >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "[cc-smoke] starting openresty container on :$PORT"
docker rm -f cc-smoke-openresty >/dev/null 2>&1 || true
docker run -d --name cc-smoke-openresty \
  --add-host=host.docker.internal:host-gateway \
  -p "${PORT}:18088" \
  "$IMAGE" >/dev/null

sleep 3
if ! docker ps --format '{{.Names}}' | grep -qx cc-smoke-openresty; then
  echo "[cc-smoke] openresty container failed to stay up"
  docker logs cc-smoke-openresty 2>&1 | tail -n 40 || true
  exit 1
fi

pass=0
fail=0
assert_status() {
  local name="$1"
  local path="$2"
  local expect="$3"
  local code
  code=$(curl -s -o /tmp/cc_smoke_body.txt -w '%{http_code}' -H 'Host: cc-test.local' "http://127.0.0.1:${PORT}${path}" || echo "000")
  if [[ "$code" == "$expect" ]]; then
    echo "[PASS] $name -> HTTP $code"
    pass=$((pass + 1))
  else
    echo "[FAIL] $name -> HTTP $code, want $expect"
    echo "       body: $(tr '\n' ' ' < /tmp/cc_smoke_body.txt | head -c 200)"
    fail=$((fail + 1))
  fi
}

echo "[cc-smoke] custom allow rule on /api/health"
assert_status "allow /api/health" "/api/health" "200"

echo "[cc-smoke] rate limit / after 2 requests in window"
assert_status "rate / #1" "/" "200"
assert_status "rate / #2" "/" "200"
assert_status "rate / #3 blocked" "/" "429"

if [[ "$fail" -gt 0 ]]; then
  echo "[cc-smoke] FAILED pass=$pass fail=$fail"
  docker logs cc-smoke-openresty 2>&1 | tail -n 40 || true
  exit 1
fi

echo "[cc-smoke] SUCCESS pass=$pass fail=$fail"
