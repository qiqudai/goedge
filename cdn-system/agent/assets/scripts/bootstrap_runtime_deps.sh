#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

if [[ $# -eq 0 ]]; then
  echo "No packages requested."
  exit 0
fi

pkgs=("$@")

if command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y --no-install-recommends "${pkgs[@]}"
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y "${pkgs[@]}"
elif command -v yum >/dev/null 2>&1; then
  yum install -y "${pkgs[@]}"
else
  echo "No supported package manager found." >&2
  exit 1
fi

ldconfig || true
