#!/bin/sh
set -eu

ROOT="$(cd "$(dirname "$0")" && pwd)"
export LUA_PATH="$ROOT/lua/?.lua;;"

resty -e "
package.path = '$ROOT/lua/?.lua;;'
local core = require 'parent_fetch_core'

local upstreams = {
  upstream_map = {
    upstream_1 = {{addr = '9.9.9.9:80'}},
    l1_upstream_5 = {{addr = '1.1.1.1:80', node_id = 41}},
    l2_upstream_5 = {{addr = '2.2.2.2:80', node_id = 56}},
    l2_upstream_3 = {{addr = '3.3.3.3:80', node_id = 70}},
  },
  parent_status = { l1 = { ['41'] = true, ['42'] = false } },
  l2_status = { nodes = { ['56'] = true, ['70'] = false } },
}

local function assert_eq(name, got_key, got_layer, want_key, want_layer)
  if got_key ~= want_key or got_layer ~= want_layer then
    io.write(string.format('FAIL %s: got key=%s layer=%s want key=%s layer=%s\n',
      name, tostring(got_key), tostring(got_layer), want_key, want_layer))
    os.exit(1)
  end
  io.write('PASS ', name, '\n')
end

local base = {
  upstream_key = 'upstream_1',
  parent_l1_upstream_key = 'l1_upstream_5',
  parent_l2_upstream_key = 'l2_upstream_5',
  use_l2 = true,
  l2_upstream_key = 'l2_upstream_5',
}

-- 1) L3 -> Origin
local k, l = core.resolve_l3_upstream({upstream_key='upstream_1', parent_fetch_mode='origin'}, upstreams)
assert_eq('L3_origin', k, l, 'upstream_1', 'origin')

-- 2) L3 -> L2
k, l = core.resolve_l3_upstream({upstream_key='upstream_1', parent_fetch_mode='l2',
  parent_l2_upstream_key='l2_upstream_5'}, upstreams)
assert_eq('L3_l2', k, l, 'l2_upstream_5', 'parent')

-- 3) L3 -> L1
k, l = core.resolve_l3_upstream({upstream_key='upstream_1', parent_fetch_mode='l1',
  parent_l1_upstream_key='l1_upstream_5', parent_l2_upstream_key='l2_upstream_5'}, upstreams)
assert_eq('L3_l1', k, l, 'l1_upstream_5', 'parent')

-- 4) L1 -> L2 (healthy)
if not core.resolve_l1_use_l2(base, upstreams, false) then
  io.write('FAIL L1_use_l2_healthy\n'); os.exit(1)
end
io.write('PASS L1_use_l2_healthy\n')

-- 5) L1 skip L2 via header flag
if core.resolve_l1_use_l2(base, upstreams, true) then
  io.write('FAIL L1_skip_l2_header\n'); os.exit(1)
end
io.write('PASS L1_skip_l2_header\n')

-- 6) L3 l1 failover to L2
local fail = {
  upstream_map = upstreams.upstream_map,
  parent_status = { l1 = { ['41'] = false } },
  l2_status = upstreams.l2_status,
}
k, l = core.resolve_l3_upstream({upstream_key='upstream_1', parent_fetch_mode='l1',
  parent_l1_upstream_key='l1_upstream_5', parent_l2_upstream_key='l2_upstream_5'}, fail)
assert_eq('L3_l1_failover_l2', k, l, 'l2_upstream_5', 'parent')

-- 7) L3 l1 both parent tiers down -> origin
fail.parent_status.l1['41'] = false
fail.l2_status = { nodes = { ['56'] = false } }
k, l = core.resolve_l3_upstream({upstream_key='upstream_1', parent_fetch_mode='l1',
  parent_l1_upstream_key='l1_upstream_5', parent_l2_upstream_key='l2_upstream_5'}, fail)
assert_eq('L3_l1_both_down_origin', k, l, 'upstream_1', 'origin')

-- 8) L3 l2 offline -> origin
k, l = core.resolve_l3_upstream({upstream_key='upstream_1', parent_fetch_mode='l2',
  parent_l2_upstream_key='l2_upstream_5'}, fail)
assert_eq('L3_l2_down_origin', k, l, 'upstream_1', 'origin')

-- 9) L3 l1 recovery
fail.parent_status.l1['41'] = true
fail.l2_status = upstreams.l2_status
k, l = core.resolve_l3_upstream({upstream_key='upstream_1', parent_fetch_mode='l1',
  parent_l1_upstream_key='l1_upstream_5', parent_l2_upstream_key='l2_upstream_5'}, fail)
assert_eq('L3_l1_recovery', k, l, 'l1_upstream_5', 'parent')

-- 10) l1_respect_l2 flag parsing
if not core.parse_bool_flag(nil, true) then io.write('FAIL parse_default\n'); os.exit(1) end
if core.parse_bool_flag('false', true) then io.write('FAIL parse_false\n'); os.exit(1) end
io.write('PASS L1_respect_l2_parse\n')

print('ALL_SCENARIOS_OK')
"
