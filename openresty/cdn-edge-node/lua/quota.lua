-- lua/quota.lua
local _M = {}
local config_loader = require "lua.config_loader"
local redis_conn = require "lua.redis_conn"

-- 1. Check Account Status (Distributed)
function _M.check_quota(host)
    local config = config_loader.get_config()
    if not config or not config.domain_map then return end
    
    local domain = config.domain_map[host]
    if not domain then return end
    
    -- Local static check first (Performance)
    if domain.status == "suspended" then
        ngx.exit(451)
    end
    
    -- Distributed Rate Limit REMOVED per user request
    -- Reason: High latency risk under attack; Local limit_req is preferred.
end

return _M
