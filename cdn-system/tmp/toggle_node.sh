#!/usr/bin/env bash
set -euo pipefail
TOKEN="$(sed -n 's/.*\"token\":\"\([^\"]*\)\".*/\1/p' /tmp/login_body.json)"
if [ -z "$TOKEN" ]; then
  echo "missing token" >&2
  exit 1
fi

echo "-- disable node 1"
curl -s -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -X PUT \
  http://127.0.0.1:8080/api/v1/admin/nodes/1/status \
  -d '{"enable":false}'

echo
echo "-- enable node 1"
curl -s -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -X PUT \
  http://127.0.0.1:8080/api/v1/admin/nodes/1/status \
  -d '{"enable":true}'

echo