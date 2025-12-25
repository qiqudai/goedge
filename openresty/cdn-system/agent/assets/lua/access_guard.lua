-- lua/access_guard.lua
local waf = require "lua.waf"
local quota = require "lua.quota"
local cc = require "lua.cc"

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

local function lookup_domain_conf()
    local config = _G.cdn_config
    if not config or not config.domain_map then
        return nil
    end
    return config.domain_map[ngx.var.host]
end

waf.check()

local domain_conf = lookup_domain_conf()
if domain_conf then
    local client_ip = ngx.var.remote_addr
    local whitelisted = ip_in_list(domain_conf.white_ips, client_ip)
    if not whitelisted and ip_in_list(domain_conf.black_ips, client_ip) then
        ngx.exit(403)
    end
    if not whitelisted then
        acl_check(domain_conf, client_ip)
    end

    local rule_id = tonumber(ngx.var.cc_rule_id or 0) or 0
    if rule_id > 0 then
        cc.check_rule_id(rule_id, domain_conf.name or "", client_ip, ngx.var.uri)
    end

    quota.check_quota(ngx.var.host)
end
