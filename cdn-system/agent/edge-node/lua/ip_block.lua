-- lua/ip_block.lua
local _M = {}
local block_list = ngx.shared.ip_blacklist

local function temp_whitelist_key(ip)
    return "waf:temp:allow:" .. ip
end

function _M.is_blocked(ip)
    if not block_list then return false end
    return block_list:get(ip)
end

function _M.block(ip, ttl)
    if not block_list or not ip or ip == "" then
        return false
    end
    ttl = tonumber(ttl) or 3600
    if ttl <= 0 then
        ttl = 3600
    end
    return block_list:set(ip, 1, ttl)
end

function _M.unblock(ip)
    if not ip or ip == "" then
        return false
    end
    if block_list then
        block_list:delete(ip)
    end
    local cache = ngx.shared.waf_cache
    if cache then
        cache:delete(temp_whitelist_key(ip))
    end
    return true
end

return _M
