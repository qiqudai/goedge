-- lua/anti_cc.lua
local _M = {}
local limit_req = ngx.shared.limit_req_store

function _M.check_limit(ip)
    if not limit_req then return false end
    
    local key = ip
    local rate = 10 -- 10 requests per second
    
    local current, flags = limit_req:get(key)
    if current and current > rate then
        return true
    end
    
    local new_val, err = limit_req:incr(key, 1, 0, 1) -- increment, default 0, expire 1s
    if not new_val then
        limit_req:set(key, 1, 1)
    end
    
    return false
end

return _M
