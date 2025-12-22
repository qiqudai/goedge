-- lua/waf.lua
local _M = {}
local redis_conn = require "lua.redis_conn"

local CACHE_TTL = 5 -- seconds
local cache = ngx.shared.waf_cache

-- Simple Local Rules
local UA_BLACKLIST = { "sqlmap", "nikto", "w3af", "nmap" }

local cdnfly = require "lua.cdnfly_wrapper"

-- Ensure we can find the migrated libraries
-- We append 'lua/lib/?.lua' to the search path
-- This assumes the running directory is roughly the base of cdn-edge-node or similar
-- or that we can use relative paths.
if not string.find(package.path, "lua/lib/?.lua", 1, true) then
    package.path = package.path .. ";lua/lib/?.lua"
end

function _M.check()
    local ip = ngx.var.remote_addr
    
    -- 1. Cache-Aside IP Blacklist Check (Zero Latency Path)
    local is_blocked = nil
    if cache then
        is_blocked = cache:get("ip_bl:" .. ip)
    end
    
    if is_blocked == 1 then
        ngx.log(ngx.WARN, "WAF: IP Blocked (Cache Hit): ", ip)
        ngx.exit(403)
    elseif is_blocked == 0 then
        -- Clean IP in cache, proceed to Cdnfly/Next but skip Redis
    else
        -- 2. Local Regex Checks (UA) - Fast CPU check (Prioritize CPU over Network)
        local ua = ngx.var.http_user_agent
        if ua then
            local ua_lower = string.lower(ua)
            for _, pattern in ipairs(UA_BLACKLIST) do
                if string.find(ua_lower, pattern, 1, true) then
                    ngx.exit(403)
                end
            end
        end
        
        -- 3. Arg Checks - Fast CPU check
        local args = ngx.req.get_uri_args()
        if args then
            for key, val in pairs(args) do
                if type(val) == "string" and ngx.re.find(val, "union%s+select", "jo") then
                    ngx.exit(403)
                end
            end
        end

         -- 4. Cache Miss: Query Redis (Slow Path)
         local red, err = redis_conn.get_connect()
         if red then
             local res, err = red:sismember("global_blacklist", ip)
             redis_conn.close(red)
             
             if res == 1 then
                 ngx.log(ngx.WARN, "WAF: IP Blocked (Redis Hit): ", ip)
                 if cache then cache:set("ip_bl:" .. ip, 1, CACHE_TTL) end
                 ngx.exit(403)
             else
                 if cache then cache:set("ip_bl:" .. ip, 0, CACHE_TTL) end
             end
         end
    end

    -- 5. Cdnfly Commercial WAF Engine

end

return _M
