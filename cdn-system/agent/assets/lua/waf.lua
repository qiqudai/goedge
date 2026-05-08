-- lua/waf.lua
local _M = {}
local CACHE_TTL = 5 -- seconds
local cache = ngx.shared.waf_cache
local ip_blacklist = ngx.shared.ip_blacklist

-- Simple Local Rules
local UA_BLACKLIST = { "sqlmap", "nikto", "w3af", "nmap" }

local cdnfly = require "lua.cdnfly_wrapper"
local geo_country = require "lua.geo_country"
local guard = require "lua.guard"

local function build_waf_reason(rule, extras)
    local parts = {"type=waf", "rule=" .. tostring(rule or "unknown"), "rule_id=0"}
    if type(extras) == "table" then
        for _, item in ipairs(extras) do
            if item and item ~= "" then
                table.insert(parts, tostring(item))
            end
        end
    end
    return table.concat(parts, ";")
end

local function mark_block_source(reason)
    local source = reason or build_waf_reason("unknown")
    ngx.header["X-Block-Source"] = source
end

local function to_string(value)
    if value == nil then
        return ""
    end
    if type(value) == "string" then
        return value
    end
    if type(value) == "number" or type(value) == "boolean" then
        return tostring(value)
    end
    return ""
end

local function lower_string(value)
    return string.lower(to_string(value))
end

local function to_int(value, fallback)
    local n = tonumber(value)
    if n == nil then
        return fallback
    end
    return n
end

local function match_patterns(value, patterns)
    if value == "" then
        return false
    end
    for _, pattern in ipairs(patterns) do
        if ngx.re.find(value, pattern, "ijo") then
            return true
        end
    end
    return false
end

local function match_pattern_list(value, patterns)
    if value == "" or type(patterns) ~= "table" then
        return false
    end
    for _, pattern in ipairs(patterns) do
        if pattern and pattern ~= "" and ngx.re.find(value, pattern, "ijo") then
            return true
        end
    end
    return false
end

local function has_syntactic_flag(config, key)
    if not config or not config.waf or not config.waf.syntactic then
        return false
    end
    return config.waf.syntactic[key] == true
end

local function is_waf_disabled(config)
    return config and config.waf and config.waf.enable == false
end

local function is_log_only(config)
    if not config or not config.waf then
        return false
    end
    return string.lower(tostring(config.waf.policy or "")) == "log_only"
end

local function waf_debug_enabled(config)
    return config and config.waf and config.waf.anti_cc_debug == true
end

local function waf_debug_log(config, ...)
    if waf_debug_enabled(config) then
        ngx.log(ngx.INFO, ...)
    end
end

local function get_blacklist_timeout(config)
    local ttl = to_int(config.waf.blacklist_timeout, 0)
    if ttl <= 0 then
        ttl = 3600
    end
    return ttl
end

local function set_ip_blacklist(config, ip)
    if not ip_blacklist or not ip or ip == "" then
        return
    end
    ip_blacklist:set(ip, true, get_blacklist_timeout(config))
end

local function normalize_action(action)
    action = string.lower(to_string(action))
    if action == "ipset" or action == "disconnect" or action == "page" then
        return action
    end
    return "disconnect"
end

local function temp_whitelist_key(ip)
    return "waf:temp:allow:" .. ip
end

local function is_temp_whitelisted(config, ip)
    if not cache or not config or not config.waf or not ip or ip == "" then
        return false
    end
    return cache:get(temp_whitelist_key(ip)) == 1
end

local function should_grant_temp_whitelist(config, ip, uri)
    if not cache or not config or not config.waf then
        return false
    end
    local limit_total = to_int(config.waf.temp_whitelist_limit_total, 0)
    local limit_url = to_int(config.waf.temp_whitelist_limit_url, 0)
    local timeout = to_int(config.waf.temp_whitelist_timeout, 0)
    if timeout <= 0 then
        return false
    end
    if limit_total <= 0 and limit_url <= 0 then
        return false
    end

    local total = 0
    local url_count = 0
    if limit_total > 0 then
        total = cache:incr("waf:temp:total:" .. ip, 1, 0, 5)
        if total > limit_total then
            return false
        end
    end
    if limit_url > 0 and uri and uri ~= "" then
        url_count = cache:incr("waf:temp:url:" .. ip .. ":" .. uri, 1, 0, 5)
        if url_count > limit_url then
            return false
        end
    end

    cache:set(temp_whitelist_key(ip), 1, timeout)
    return true
end

local function auto_ipset_active(config, host)
    if not cache or not config or not config.waf or config.waf.auto_ipset_enable ~= true then
        return false
    end
    local threshold = to_int(config.waf.auto_ipset_threshold, 0)
    if threshold <= 0 then
        return false
    end
    host = host or "_"
    local flag_key = "waf:auto_ipset:flag:" .. host
    if cache:get(flag_key) == 1 then
        return true
    end
    local count = cache:incr("waf:auto_ipset:count:" .. host, 1, 0, 1)
    if count >= threshold then
        cache:set(flag_key, 1, 60)
        return true
    end
    return false
end

local function should_block_page_rate_limit(config, ip)
    if not cache or not config or not config.waf then
        return false
    end
    if config.waf.block_page_rate_limit_enable ~= true then
        return false
    end
    local limit = to_int(config.waf.block_page_rate_limit, 0)
    if limit <= 0 then
        return false
    end
    local count = cache:incr("waf:block_page:" .. ip, 1, 0, 60)
    return count > limit
end

local function block_request(config, ip, reason, status, extras)
    if not config or not config.waf then
        mark_block_source(build_waf_reason(reason or "waf", extras))
        ngx.exit(status or 403)
    end
    if is_log_only(config) then
        waf_debug_log(config, "WAF log_only skip: ", reason or "")
        return false
    end
    if should_grant_temp_whitelist(config, ip, ngx.var.uri or "") then
        return false
    end

    local host = ngx.var.host or "_"
    local action = normalize_action(config.waf.default_block_action)
    if auto_ipset_active(config, host) then
        action = "ipset"
    end

    if action == "page" and should_block_page_rate_limit(config, ip) then
        action = "ipset"
    end

    local details = {}
    if type(extras) == "table" then
        for _, item in ipairs(extras) do
            if item and item ~= "" then
                table.insert(details, tostring(item))
            end
        end
    end
    table.insert(details, "mode=" .. tostring(action))
    mark_block_source(build_waf_reason(reason or "waf", details))

    if action == "ipset" then
        set_ip_blacklist(config, ip)
        ngx.exit(status or 403)
    elseif action == "page" then
        if config.waf.block_page_traffic_free == true then
            ngx.ctx.waf_block_page = true
        end
        ngx.exit(status or 418)
    end

    ngx.exit(444)
end

local function should_block_resource(config, ip, uri)
    if not cache or not config or not config.waf then
        return false
    end
    if config.waf.resource_protection_enable ~= true then
        return false
    end

    local block_timeout = tonumber(config.waf.resource_protection_block_timeout) or 0
    if block_timeout <= 0 then
        block_timeout = 300
    end

    local block_key = "rp:block:" .. ip
    local blocked = cache:get(block_key)
    if blocked == 1 then
        return true
    end

    local rules = config.waf.resource_protection_rules
    if type(rules) == "table" and #rules > 0 then
        for _, rule in ipairs(rules) do
            local duration = tonumber(rule.duration) or 0
            local max_requests = tonumber(rule.max_requests) or 0
            if duration > 0 and max_requests > 0 then
                local key = "rp:" .. duration .. ":" .. ip .. ":" .. uri
                local count, err = cache:incr(key, 1, 0, duration)
                if err then
                    ngx.log(ngx.WARN, "WAF: resource_protection incr failed: ", err)
                elseif count > max_requests then
                    cache:set(block_key, 1, block_timeout)
                    return true
                end
            end
        end
        return false
    end

    local threshold = tonumber(config.waf.resource_protection_threshold) or 0
    if threshold > 0 then
        local key = "rp:threshold:" .. ip .. ":" .. uri
        local count, err = cache:incr(key, 1, 0, 1)
        if err then
            ngx.log(ngx.WARN, "WAF: resource_protection threshold incr failed: ", err)
        elseif count > threshold then
            cache:set(block_key, 1, block_timeout)
            return true
        end
    end
    return false
end

local function in_list(list, ip)
    if not list or list == "" then return false end
    for line in string.gmatch(list, "[^\n]+") do
        line = string.gsub(line, "^%s+", "")
        line = string.gsub(line, "%s+$", "")
        if line ~= "" and line == ip then
            return true
        end
    end
    return false
end

local function resolve_country(ip)
    if not ip then
        return ""
    end
    local cache = _G.IP_LRU_CACHE
    if cache then
        local cached = cache:get("geo:" .. ip)
        if cached ~= nil then
            return cached
        end
    end
    if not _G.IP_SEARCHER then
        return ""
    end
    local ok, res = pcall(function()
        return _G.IP_SEARCHER:search(ip)
    end)
    if not ok or not res then
        return ""
    end
    local country = ""
    local raw = ""
    if type(res) == "table" then
        country = res.country or res.region or res[1] or ""
    else
        raw = tostring(res)
        local first = raw:match("^[^|]+") or raw
        country = first
    end
    country = string.upper(country)
    local idx = string.find(country, "-", 1, true)
    if idx and idx > 1 then
        country = string.sub(country, 1, idx - 1)
    end
    country = geo_country.to_iso(country, raw)
    if cache and country ~= "" then
        cache:set("geo:" .. ip, country, 600)
    end
    return country
end

local function region_blocked(region_list, ip)
    if type(region_list) ~= "table" or #region_list == 0 or not ip then
        return false
    end
    local country = resolve_country(ip)
    if country == "" then
        return false
    end
    for _, item in ipairs(region_list) do
        local code = string.upper(tostring(item or ""))
        local idx = string.find(code, "-", 1, true)
        if idx and idx > 1 then
            code = string.sub(code, 1, idx - 1)
        end
        if code ~= "" and code == country then
            return true
        end
    end
    return false
end

local function is_default_page(uri)
    if not uri or uri == "" then
        return false
    end
    if uri == "/" then
        return true
    end
    return uri == "/index.html" or uri == "/index.htm"
end

local function normalize_guard_type(value)
    local v = string.lower(to_string(value))
    if v == "5s" or v == "five_seconds" then
        return "five_seconds"
    end
    if v == "slide" then
        return "slide_captcha"
    end
    if v == "slide_simple" then
        return "slide_captcha_simple"
    end
    if v == "click" then
        return "click_captcha"
    end
    if v == "click_simple" then
        return "click_captcha_simple"
    end
    if v == "rotate" then
        return "rotate_captcha"
    end
    return "slide_captcha"
end

local function apply_default_page_protection(config, ip, host, uri)
    if not config or not config.waf then
        return
    end
    local mode = to_string(config.waf.default_page_protection)
    if mode == "" or not is_default_page(uri) or guard.is_guard_request(uri) then
        return
    end

    local force = mode == "force"
    if mode == "auto" then
        local threshold = to_int(config.waf.default_page_protection_threshold, 0)
        if threshold <= 0 then
            return
        end
        if cache then
            local count = cache:incr("waf:default_page:" .. (host or "_"), 1, 0, 1)
            if count >= threshold then
                force = true
                if config.waf.cc_rule_auto_switch == true then
                    cache:set("waf:default_page:force:" .. (host or "_"), 1, 60)
                end
            end
            if config.waf.cc_rule_auto_switch == true then
                if cache:get("waf:default_page:force:" .. (host or "_")) == 1 then
                    force = true
                end
            end
        end
    end

    if not force then
        return
    end

    local filter = { type = normalize_guard_type(config.waf.anti_cc_type), id = 0 }
    if guard.ensure_passed(filter, host or "", ip or "") then
        return
    end
    waf_debug_log(config, "WAF: default page guard triggered host=", host or "", " ip=", ip or "")
    guard.challenge(filter, host or "", ip or "")
    ngx.exit(200)
end

local function apply_well_known_protection(config, ip, uri)
    if not cache or not config or not config.waf then
        return
    end
    local threshold = to_int(config.waf.well_known_protection_threshold, 0)
    if threshold <= 0 then
        return
    end
    if not uri or string.sub(uri, 1, 13) ~= "/.well-known/" then
        return
    end

    local block_key = "waf:well_known:block:" .. ip
    if cache:get(block_key) == 1 then
        mark_block_source(build_waf_reason("well_known", {"condition=well_known_path"}))
        ngx.exit(404)
    end

    local count = cache:incr("waf:well_known:count:" .. ip, 1, 0, 60)
    if count > threshold then
        cache:set(block_key, 1, 300)
        mark_block_source(build_waf_reason("well_known", {"condition=well_known_path"}))
        ngx.exit(404)
    end
end

-- Ensure we can find the migrated libraries
-- We append 'lua/lib/?.lua' to the search path
-- This assumes the running directory is roughly the base of cdn-edge-node or similar
-- or that we can use relative paths.
if not string.find(package.path, "lua/lib/?.lua", 1, true) then
    package.path = package.path .. ";lua/lib/?.lua"
end

function _M.check()
    local ip = ngx.var.remote_addr
    local config = _G.cdn_config
    if is_waf_disabled(config) then
        return
    end
    if is_log_only(config) then
        return
    end

    if config and config.waf then
        if in_list(config.waf.whitelist_ips, ip) then
            return
        end
        if is_temp_whitelisted(config, ip) then
            return
        end
        if in_list(config.waf.blacklist_ips, ip) then
            block_request(config, ip, "blacklist", 418)
            return
        end

        if config.waf.prevent_tls_handshake == true then
            if ngx.var.https == "on" or ngx.var.scheme == "https" then
                local sni = ngx.var.ssl_server_name or ""
                if sni == "" then
                    block_request(config, ip, "tls_handshake", 444)
                    return
                end
            end
        end

        if config.waf.disable_ping == true then
            local uri = ngx.var.uri or ""
            if uri == "/ping" or uri == "/_ping" or uri == "/ping/" or uri == "/_ping/" then
                block_request(config, ip, "ping", 403)
                return
            end
        end
    end

    local uri = ngx.var.uri or ""
    apply_well_known_protection(config, ip, uri)

    if config and config.waf and config.waf.access_control then
        local ac = config.waf.access_control
        local ua = ngx.var.http_user_agent or ""
        if ac.block_empty_ua == true and ua == "" then
            block_request(config, ip, "empty_ua", 418)
            return
        end

        local ua_lower = string.lower(ua)
        local ua_whitelisted = match_pattern_list(ua_lower, ac.white_ua)
        if not ua_whitelisted and match_pattern_list(ua_lower, ac.black_ua) then
            block_request(config, ip, "ua_black", 418)
            return
        end

        local request_uri = ngx.var.request_uri or ngx.var.uri or ""
        local url_whitelisted = match_pattern_list(request_uri, ac.white_url)
        if not url_whitelisted and match_pattern_list(request_uri, ac.black_url) then
            block_request(config, ip, "url_black", 403)
            return
        end

        if region_blocked(ac.region_block, ip) then
            block_request(config, ip, "region", 418, {"condition=access_control.region_block"})
            return
        end
    end

    if should_block_resource(config, ip, uri) then
        block_request(config, ip, "resource", 403, {"condition=resource_protection"})
        return
    end

    -- 1. Cache-Aside IP Blacklist Check (Zero Latency Path)
    local is_blocked = nil
    if cache then
        is_blocked = cache:get("ip_bl:" .. ip)
    end

    if is_blocked == 1 then
        ngx.log(ngx.WARN, "WAF: IP Blocked (Cache Hit): ", ip)
        mark_block_source(build_waf_reason("waf_cache", {"condition=ip_bl_cache_hit"}))
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
                    block_request(config, ip, "ua_black", 418)
                    return
                end
            end
            if has_syntactic_flag(config, "scanner") then
                local scanner_patterns = { "sqlmap", "nikto", "w3af", "nmap", "acunetix", "nessus", "masscan", "dirbuster", "zgrab", "zmap" }
                for _, pattern in ipairs(scanner_patterns) do
                    if string.find(ua_lower, pattern, 1, true) then
                        block_request(config, ip, "scanner", 418)
                        return
                    end
                end
            end
        end

        -- 3. Arg Checks - Fast CPU check
        local args = ngx.req.get_uri_args()
        if args then
            local sql_patterns = { "union%s+select", "select%s+.+%s+from", "insert%s+into", "update%s+.+%s+set", "delete%s+from", "drop%s+table", "sleep%(", "benchmark%(", }
            local xss_patterns = { "<script", "javascript:", "onerror%s*=", "onload%s*=", "<img", "<svg", "document%.cookie" }
            for key, val in pairs(args) do
                local key_lower = lower_string(key)
                local val_lower = lower_string(val)
                if has_syntactic_flag(config, "sql_injection") then
                    if match_patterns(key_lower, sql_patterns) or match_patterns(val_lower, sql_patterns) then
                        block_request(config, ip, "sql_injection", 403)
                        return
                    end
                end
                if has_syntactic_flag(config, "xss") then
                    if match_patterns(key_lower, xss_patterns) or match_patterns(val_lower, xss_patterns) then
                        block_request(config, ip, "xss", 403)
                        return
                    end
                end
            end
        end

        -- 4. Redis lookup removed; rely on local rules only.
        if cache then cache:set("ip_bl:" .. ip, 0, CACHE_TTL) end
    end

    apply_default_page_protection(config, ip, ngx.var.host or "", uri)

    -- 5. Cdnfly Commercial WAF Engine

end

return _M
