-- lua/unbound_guard.lua
-- Escalating blacklist policy for unbound-domain/default-server hits:
-- 5 hits => 5m, 10 => 10m, ... (linear steps)
local _M = {}

local block_list = ngx.shared.ip_blacklist
local counter_store = ngx.shared.guard_store or ngx.shared.limit_req_store

local STEP = 5
local STEP_SECONDS = 300
local COUNTER_TTL_SECONDS = 24 * 60 * 60
local MAX_BLOCK_SECONDS = 24 * 60 * 60

local function calc_ttl(count)
    local level = math.floor((tonumber(count) or 0) / STEP)
    if level <= 0 then
        return 0
    end
    local ttl = level * STEP_SECONDS
    if ttl > MAX_BLOCK_SECONDS then
        ttl = MAX_BLOCK_SECONDS
    end
    return ttl
end

local function run(ip)
    if not ip or ip == "" then
        return
    end
    if not block_list or not counter_store then
        return
    end
    if block_list:get(ip) then
        return
    end

    local key = "unbound:count:" .. ip
    local count, err = counter_store:incr(key, 1, 0, COUNTER_TTL_SECONDS)
    if not count then
        counter_store:set(key, 1, COUNTER_TTL_SECONDS)
        count = 1
    end
    if (count % STEP) ~= 0 then
        return
    end

    local ttl = calc_ttl(count)
    if ttl > 0 then
        block_list:set(ip, true, ttl)
        ngx.log(ngx.WARN, "unbound_guard blacklist ip=", ip, " count=", count, " ttl=", ttl, " err=", err or "")
    end
end

function _M.enforce(status)
    run(ngx.var.remote_addr)
    ngx.exit(status or 418)
end

function _M.run_for_ip(ip)
    run(ip)
end

return _M
