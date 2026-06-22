#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="${ROOT_DIR}/assets/data/ip2region.xdb"
META="${ROOT_DIR}/assets/data/ip2region.meta.json"
SOURCE_URL="https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb"

mkdir -p "$(dirname "$DEST")"
curl -fsSL "$SOURCE_URL" -o "$DEST"

SHA256="$(shasum -a 256 "$DEST" | awk '{print $1}')"
SIZE="$(wc -c < "$DEST" | tr -d ' ')"
DATE="$(date +%Y-%m-%d)"

cat > "$META" <<EOF
{
  "source": "https://github.com/lionsoul2014/ip2region",
  "file": "data/ip2region_v4.xdb",
  "runtime_name": "ip2region.xdb",
  "sha256": "${SHA256}",
  "size_bytes": ${SIZE},
  "updated_at": "${DATE}"
}
EOF

echo "Updated ${DEST}"
echo "sha256=${SHA256}"
echo "size=${SIZE}"
