#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
WAF_SRC="$ROOT/agent/edge-node/lua/waf.lua"

if [[ ! -f "$WAF_SRC" ]]; then
  echo "[lua-waf-cidr] missing $WAF_SRC" >&2
  exit 1
fi

docker run --rm \
  openresty/openresty:1.25.3.1-0-alpine-fat \
  resty -e '
local bit = require "bit"

local function ipv4_to_num(ip)
    local a, b, c, d = ip:match("^(%d+)%.(%d+)%.(%d+)%.(%d+)$")
    if not a then return nil end
    a, b, c, d = tonumber(a), tonumber(b), tonumber(c), tonumber(d)
    if not a or not b or not c or not d then return nil end
    return a * 16777216 + b * 65536 + c * 256 + d
end

local function ip_in_cidr(ip, cidr)
    local base, prefix = cidr:match("^(%d+%.%d+%.%d+%.%d+)%s*/%s*(%d+)$")
    if not base or not prefix then return false end
    local ip_num = ipv4_to_num(ip)
    local base_num = ipv4_to_num(base)
    if not ip_num or not base_num then return false end
    local bits = tonumber(prefix)
    if not bits or bits < 0 or bits > 32 then return false end
    local mask = bits == 0 and 0 or bit.lshift(0xFFFFFFFF, 32 - bits)
    mask = bit.band(mask, 0xFFFFFFFF)
    return bit.band(ip_num, mask) == bit.band(base_num, mask)
end

local function ip_matches_list_entry(ip, entry)
    entry = string.match(tostring(entry or ""), "^%s*(.-)%s*$") or ""
    if entry == "" then return false end
    if ip == entry then return true end
    if string.find(entry, "/", 1, true) then return ip_in_cidr(ip, entry) end
    return false
end

local function in_list(list, ip)
    for line in string.gmatch(list, "[^\n]+") do
        line = string.gsub(line, "^%s+", "")
        line = string.gsub(line, "%s+$", "")
        if line ~= "" and ip_matches_list_entry(ip, line) then
            return true
        end
    end
    return false
end

local cases = {
    {list = "10.0.0.50", ip = "10.0.0.50", want = true},
    {list = "10.0.0.0/8", ip = "10.0.0.50", want = true},
    {list = "10.0.0.0/8", ip = "10.255.255.255", want = true},
    {list = "10.0.0.0/8", ip = "192.168.1.1", want = false},
    {list = "10.0.0.50\n10.0.0.0/8", ip = "10.0.0.99", want = true},
}

for i, c in ipairs(cases) do
    local got = in_list(c.list, c.ip)
    if got ~= c.want then
        io.stderr:write(string.format("case %d failed: list=%q ip=%s got=%s want=%s\n", i, c.list, c.ip, tostring(got), tostring(c.want)))
        os.exit(1)
    end
end
print("[lua-waf-cidr] all cases passed")
'
