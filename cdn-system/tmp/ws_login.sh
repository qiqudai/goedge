#!/usr/bin/env bash
set -euo pipefail

echo eyJ1c2VybmFtZSI6ImFkbWluIiwicGFzc3dvcmQiOiIxMjM0NTYifQ== | base64 -d > /tmp/login.json
curl -s -D /tmp/login_headers.txt -o /tmp/login_body.json \
  -H "Content-Type: application/json" \
  -X POST \
  --data-binary @/tmp/login.json \
  http://127.0.0.1:8080/api/v1/admin/login