#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
LUA_SRC="$ROOT/agent/edge-node/lua/access_guard.lua"

if [[ ! -f "$LUA_SRC" ]]; then
  echo "[lua-blacklist] missing $LUA_SRC" >&2
  exit 1
fi

docker run --rm \
  -v "$LUA_SRC:/lua/access_guard.lua:ro" \
  openresty/openresty:1.25.3.1-0-alpine-fat \
  resty -e '
local bit = require "bit"

local function split_ipv4_octets(ip)
    local parts = {}
    if not ip or ip == "" then return parts end
    for oct in string.gmatch(ip, "(%d+)") do parts[#parts + 1] = oct end
    return parts
end

local function ipv4_to_num(ip)
    if not ip or ip == "" then return nil end
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

local function ip_matches_blacklist_entry(ip, entry)
    if not ip or not entry then return false end
    entry = string.match(tostring(entry), "^%s*(.-)%s*$") or ""
    if entry == "" then return false end
    if ip == entry then return true end
    if string.find(entry, "/", 1, true) then return ip_in_cidr(ip, entry) end
    if not string.find(entry, "*", 1, true) then return false end
    local ip_parts = split_ipv4_octets(ip)
    if #ip_parts ~= 4 then return false end
    local pat_parts = {}
    for part in string.gmatch(entry, "([^%.]+)") do pat_parts[#pat_parts + 1] = part end
    if #pat_parts == 0 or #pat_parts > 4 then return false end
    for i = 1, #pat_parts do
        if pat_parts[i] ~= "*" and pat_parts[i] ~= ip_parts[i] then return false end
    end
    return true
end

local cases = {
    {ip = "127.0.0.1", entry = "127.*.*.*", want = true},
    {ip = "127.9.9.9", entry = "127.*.*.*", want = true},
    {ip = "128.0.0.1", entry = "127.*.*.*", want = false},
    {ip = "10.0.0.50", entry = "10.0.0.0/24", want = true},
    {ip = "10.0.0.50", entry = "10.0.0.0/8", want = true},
    {ip = "10.0.1.50", entry = "10.0.0.0/24", want = false},
    {ip = "192.168.1.10", entry = "192.168.*", want = true},
    {ip = "192.169.1.10", entry = "192.168.*", want = false},
    {ip = "1.2.3.4", entry = "1.2.3.4", want = true},
}

for i, c in ipairs(cases) do
    local got = ip_matches_blacklist_entry(c.ip, c.entry)
    if got ~= c.want then
        io.stderr:write(string.format("case %d failed: ip=%s entry=%s got=%s want=%s\n", i, c.ip, c.entry, tostring(got), tostring(c.want)))
        os.exit(1)
    end
end

if not string.find(io.open("/lua/access_guard.lua"):read("*a"), "ip_in_blacklist", 1, true) then
    io.stderr:write("access_guard.lua missing ip_in_blacklist\n")
    os.exit(1)
end

print("[lua-blacklist] all cases passed")
'
