#!/usr/bin/env bash
set -euo pipefail
sed -n 's/.*"token":"\([^"]*\)".*/\1/p' /tmp/login_body.json