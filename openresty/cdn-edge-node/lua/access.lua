-- lua/access.lua
local ip_block = require "lua.ip_block"
local anti_cc = require "lua.anti_cc"
local edge_compute = require "lua.edge_compute"
local waf = require "lua.waf"        -- Phase 3
local quota = require "lua.quota"    -- Phase 3
local balancer = require "lua.balancer" -- Phase 3

-- 1. IP Blocking Check (Legacy/Fallback)
if ip_block.is_blocked(ngx.var.remote_addr) then
    ngx.exit(403)
end

-- 2. Anti-CC Check (Legacy/Fallback)
if anti_cc.check_limit(ngx.var.remote_addr) then
    ngx.exit(503)
end

-- 3. WAF Protection (Phase 3: Requirement #4)
-- Checks User-Agent and Query Args for malicious patterns
waf.check()

-- 4. Dynamic Routing & Config Lookup
local host = ngx.var.host
local config = _G.cdn_config 
local domain_conf = nil

if not config then
    ngx.log(ngx.ERR, "Config not loaded")
    ngx.exit(503)
end

if config.domain_map then
    domain_conf = config.domain_map[host]
end

if not domain_conf then
    ngx.log(ngx.WARN, "Unknown domain: ", host)
    ngx.exit(404)
else
    -- 5. Quota & Commercial Status (Phase 3: Requirement #8)
    -- Checks if account is suspended or limits exceeded
    quota.check_quota(host)

    -- 6. Upstream Selection (Phase 3: Requirement #7)
    local upstream_key = domain_conf.upstream_key
    if config.upstream_map and config.upstream_map[upstream_key] then
        local targets = config.upstream_map[upstream_key]
        
        -- Get Policy from Domain Config (default to round_robin)
        local policy = domain_conf.load_balance_policy or "round_robin"
        
        -- Use Balancer Logic
        local target_addr = balancer.get_target(upstream_key, targets, policy)
        
        if target_addr then
            ngx.var.backend_target = "http://" .. target_addr
            
             -- Add Custom Headers
            if domain_conf.headers then
                 for k, v in pairs(domain_conf.headers) do
                     ngx.req.set_header(k, v)
                 end
            end
        else
            ngx.log(ngx.ERR, "Balancer returned no target for: ", upstream_key)
            ngx.exit(502)
        end
    else
        ngx.log(ngx.ERR, "Upstream not found: ", upstream_key)
        ngx.exit(502)
    end
end

-- 7. Edge Logic
edge_compute.run()
