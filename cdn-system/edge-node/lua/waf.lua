-- lua/waf.lua
local _M = {}
local CACHE_TTL = 5 -- seconds
local cache = ngx.shared.waf_cache

-- Simple Local Rules
local UA_BLACKLIST = { "sqlmap", "nikto", "w3af", "nmap" }

local cdnfly = require "lua.cdnfly_wrapper"

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
    if type(res) == "table" then
        country = res.country or res.region or res[1] or ""
    else
        local first = tostring(res):match("^[^|]+") or tostring(res)
        country = first
    end
    country = string.upper(country)
    local idx = string.find(country, "-", 1, true)
    if idx and idx > 1 then
        country = string.sub(country, 1, idx - 1)
    end
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
    if config and config.waf and config.waf.enable == false then
        return
    end
    if config and config.waf then
        if in_list(config.waf.whitelist_ips, ip) then
            return
        end
        if in_list(config.waf.blacklist_ips, ip) then
            ngx.exit(418)
        end
    end

    if config and config.waf and config.waf.access_control then
        local ac = config.waf.access_control
        local ua = ngx.var.http_user_agent or ""
        if ac.block_empty_ua == true and ua == "" then
            ngx.exit(418)
        end

        local ua_lower = string.lower(ua)
        local ua_whitelisted = match_pattern_list(ua_lower, ac.white_ua)
        if not ua_whitelisted and match_pattern_list(ua_lower, ac.black_ua) then
            ngx.exit(418)
        end

        local uri = ngx.var.request_uri or ngx.var.uri or ""
        local url_whitelisted = match_pattern_list(uri, ac.white_url)
        if not url_whitelisted and match_pattern_list(uri, ac.black_url) then
            ngx.exit(403)
        end

        if region_blocked(ac.region_block, ip) then
            ngx.exit(418)
        end
    end

    local uri = ngx.var.uri or ""
    if should_block_resource(config, ip, uri) then
        ngx.exit(403)
    end
    
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
                    ngx.exit(418)
                end
            end
            if has_syntactic_flag(config, "scanner") then
                local scanner_patterns = { "sqlmap", "nikto", "w3af", "nmap", "acunetix", "nessus", "masscan", "dirbuster", "zgrab", "zmap" }
                for _, pattern in ipairs(scanner_patterns) do
                    if string.find(ua_lower, pattern, 1, true) then
                        ngx.exit(418)
                    end
                end
            end
        end
        
        -- 3. Arg Checks - Fast CPU check
        local args = ngx.req.get_uri_args()
        if args then
            local sql_patterns = { "union%s+select", "select%s+.+%s+from", "insert%s+into", "update%s+.+%s+set", "delete%s+from", "drop%s+table", "sleep%(", "benchmark%(" }
            local xss_patterns = { "<script", "javascript:", "onerror%s*=", "onload%s*=", "<img", "<svg", "document%.cookie" }
            for key, val in pairs(args) do
                local key_lower = lower_string(key)
                local val_lower = lower_string(val)
                if has_syntactic_flag(config, "sql_injection") then
                    if match_patterns(key_lower, sql_patterns) or match_patterns(val_lower, sql_patterns) then
                        ngx.exit(403)
                    end
                end
                if has_syntactic_flag(config, "xss") then
                    if match_patterns(key_lower, xss_patterns) or match_patterns(val_lower, xss_patterns) then
                        ngx.exit(403)
                    end
                end
            end
        end

        -- 4. Redis lookup removed; rely on local rules only.
        if cache then cache:set("ip_bl:" .. ip, 0, CACHE_TTL) end
    end

    -- 5. Cdnfly Commercial WAF Engine

end

return _M
