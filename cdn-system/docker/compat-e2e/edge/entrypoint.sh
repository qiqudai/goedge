#!/bin/sh
set -eu

CERT_DIR=/opt/compat/cert
mkdir -p "$CERT_DIR"

if [ ! -f "$CERT_DIR/test.key" ] || [ ! -f "$CERT_DIR/test.crt" ]; then
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "$CERT_DIR/test.key" \
    -out "$CERT_DIR/test.crt" \
    -days 1 \
    -subj "/CN=compat.test" >/dev/null 2>&1
fi

exec openresty -p /opt/compat -c /opt/compat/nginx.conf -g 'daemon off;'
