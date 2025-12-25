-- lua/access.lua
local ip_block = require "lua.ip_block"
local anti_cc = require "lua.anti_cc"
local edge_compute = require "lua.edge_compute"
local waf = require "lua.waf"        -- Phase 3
local quota = require "lua.quota"    -- Phase 3
local balancer = require "lua.balancer" -- Phase 3
local cc = require "lua.cc"
local cache = require "lua.cache"

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

local function acl_check(domain_conf, ip)
    if not domain_conf then return end
    local rules = domain_conf.acl_rules
    if rules then
        for _, rule in ipairs(rules) do
            if rule.ip == ip then
                if rule.action == "deny" then
                    ngx.exit(403)
                else
                    return
                end
            end
        end
    end
    local default_action = domain_conf.acl_default_action
    if default_action == "deny" then
        ngx.exit(403)
    end
end

local function ip_in_list(list, ip)
    if not list or not ip then
        return false
    end
    for _, item in ipairs(list) do
        if item == ip then
            return true
        end
    end
    return false
end

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
    local client_ip = ngx.var.remote_addr
    local whitelisted = ip_in_list(domain_conf.white_ips, client_ip)
    if not whitelisted and ip_in_list(domain_conf.black_ips, client_ip) then
        ngx.exit(403)
    end

    cc.check(domain_conf, client_ip, ngx.var.uri)

    local bypass, ttl = cache.resolve(domain_conf, ngx.var.uri)
    if bypass then
        ngx.var.cache_bypass = "1"
    else
        ngx.var.cache_bypass = "0"
    end
    if ttl and tonumber(ttl) and tonumber(ttl) > 0 then
        ngx.var.cache_ttl = tostring(ttl)
    end

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
            local scheme = domain_conf.origin_protocol or "http"
            scheme = string.lower(scheme)
            local target = target_addr
            if not string.find(target_addr, ":", 1, true) then
                if scheme == "https" and domain_conf.origin_https_port and domain_conf.origin_https_port ~= "" then
                    target = target_addr .. ":" .. domain_conf.origin_https_port
                elseif scheme == "http" and domain_conf.origin_http_port and domain_conf.origin_http_port ~= "" then
                    target = target_addr .. ":" .. domain_conf.origin_http_port
                end
            end
            ngx.var.backend_target = scheme .. "://" .. target
            
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

    if not whitelisted then
        acl_check(domain_conf, client_ip)
    end
end

-- 7. Edge Logic
edge_compute.run()
