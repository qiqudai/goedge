-- lua/access.lua
local ip_block = require "lua.ip_block"
local anti_cc = require "lua.anti_cc"
local edge_compute = require "lua.edge_compute"
local waf = require "lua.waf"        -- Phase 3
local quota = require "lua.quota"    -- Phase 3
local balancer = require "lua.balancer" -- Phase 3
local cc = require "lua.cc"
local cache = require "lua.cache"
local geo_country = require "lua.geo_country"
local cjson = require "cjson.safe"

-- 1. IP Blocking Check (Legacy/Fallback)
if ip_block.is_blocked(ngx.var.remote_addr) then
    ngx.exit(418)
end

-- 2. Anti-CC Check (Legacy/Fallback)
if anti_cc.check_limit(ngx.var.remote_addr) then
    ngx.exit(503)
end

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

local function region_blocked(domain_conf, ip)
    if not domain_conf or not domain_conf.region_block or not ip then
        return false
    end
    local list = domain_conf.region_block
    if type(list) ~= "table" or #list == 0 then
        return false
    end
    local country = resolve_country(ip)
    if country == "" then
        return false
    end
    for _, code in ipairs(list) do
        if string.upper(code) == country then
            return true
        end
    end
    return false
end

local crawler_tokens = {
    "baiduspider",
    "googlebot",
    "bingbot",
    "yandex",
    "sogou",
    "360spider",
    "bytespider",
    "duckduckbot",
    "slurp",
    "facebot",
    "ia_archiver",
    "semrushbot",
}

local function is_crawler_ua(ua)
    if not ua or ua == "" then
        return false
    end
    local lower = string.lower(ua)
    for _, token in ipairs(crawler_tokens) do
        if string.find(lower, token, 1, true) then
            return true
        end
    end
    return false
end

local function split_pipe(value)
    local out = {}
    if not value or value == "" then
        return out
    end
    for part in string.gmatch(value, "([^|]+)") do
        local item = string.match(part, "^%s*(.-)%s*$")
        if item ~= "" then
            table.insert(out, item)
        end
    end
    return out
end

local function ends_with(value, suffix)
    if not value or not suffix or suffix == "" then
        return false
    end
    return value:sub(-#suffix) == suffix
end

local function parse_referer_host(referer)
    if not referer or referer == "" then
        return ""
    end
    local m = ngx.re.match(referer, [[^https?://([^/]+)]], "jo")
    local host = m and m[1] or referer
    host = string.gsub(host, ":%d+$", "")
    return host
end

local function host_allowed(host, domain)
    if host == "" or domain == "" then
        return false
    end
    if host == domain then
        return true
    end
    if #host > #domain and ends_with(host, "." .. domain) then
        return true
    end
    return false
end

local function hotlink_scope_matches(scope, value, uri)
    if not scope or scope == "" or scope == "all" then
        return true
    end
    if not value or value == "" then
        return false
    end
    local items = split_pipe(value)
    if #items == 0 then
        return false
    end
    if scope == "suffix" then
        for _, suffix in ipairs(items) do
            if suffix ~= "" then
                if suffix:sub(1, 1) ~= "." and ends_with(uri, "." .. suffix) then
                    return true
                end
                if ends_with(uri, suffix) then
                    return true
                end
            end
        end
    elseif scope == "dir" then
        for _, dir in ipairs(items) do
            if dir ~= "" and uri:sub(1, #dir) == dir then
                return true
            end
        end
    elseif scope == "path" then
        for _, path in ipairs(items) do
            if uri == path then
                return true
            end
        end
    end
    return false
end

local function hotlink_allowed(domain_conf, uri)
    local hotlink = domain_conf and domain_conf.hotlink
    if not hotlink or not hotlink.enable then
        return true
    end
    if not hotlink_scope_matches(hotlink.scope, hotlink.value, uri) then
        return true
    end
    local referer = ngx.var.http_referer
    if not referer or referer == "" then
        return hotlink.allow_empty == true
    end
    local ref_host = parse_referer_host(referer)
    if ref_host == "" then
        return hotlink.allow_empty == true
    end
    if host_allowed(ref_host, domain_conf.name or "") then
        return true
    end
    if hotlink.domains and type(hotlink.domains) == "table" then
        for _, domain in ipairs(hotlink.domains) do
            if host_allowed(ref_host, domain) then
                return true
            end
        end
    end
    return false
end

local function apply_cors_headers(cors)
    if not cors or cors.enable ~= true then
        return false
    end
    local allow_origin = cors.allow_origin or "*"
    ngx.header["Access-Control-Allow-Origin"] = allow_origin
    if cors.allow_methods and cors.allow_methods ~= "" then
        ngx.header["Access-Control-Allow-Methods"] = cors.allow_methods
    end
    if cors.allow_headers and cors.allow_headers ~= "" then
        ngx.header["Access-Control-Allow-Headers"] = cors.allow_headers
    end
    if cors.expose_headers and cors.expose_headers ~= "" then
        ngx.header["Access-Control-Expose-Headers"] = cors.expose_headers
    end
    if cors.allow_credentials == true then
        ngx.header["Access-Control-Allow-Credentials"] = "true"
    end
    if cors.max_age and cors.max_age ~= "" then
        ngx.header["Access-Control-Max-Age"] = cors.max_age
    end
    return true
end

local function encode_headers(headers)
    if type(headers) ~= "table" then
        return ""
    end
    local out = {}
    for k, v in pairs(headers) do
        if type(v) == "table" then
            out[k] = table.concat(v, ",")
        elseif v ~= nil then
            out[k] = tostring(v)
        end
    end
    local ok, encoded = pcall(cjson.encode, out)
    if ok and encoded then
        return encoded
    end
    return ""
end

local function read_body(limit_bytes)
    ngx.req.read_body()
    local data = ngx.req.get_body_data()
    if data then
        return data
    end
    local file = ngx.req.get_body_file()
    if not file or file == "" then
        return ""
    end
    local f = io.open(file, "rb")
    if not f then
        return ""
    end
    local content = f:read(limit_bytes or "*a")
    f:close()
    return content or ""
end

local function normalize_raw_header(raw)
    if not raw or raw == "" then
        return ""
    end
    raw = string.gsub(raw, "\r", "")
    raw = string.gsub(raw, "\n", "\\n")
    return raw
end

local function apply_realtime_send(domain_conf)
    if not domain_conf then
        return
    end
    if domain_conf.realtime_send == false then
        ngx.var.cdn_realtime_send = "0"
    else
        ngx.var.cdn_realtime_send = "1"
    end
end

local function apply_request_logging(domain_conf)
    if not domain_conf then
        return
    end
    local log_request_header = domain_conf.log_request_header
    local log_request_body = domain_conf.log_request_body
    if not log_request_header and not log_request_body then
        return
    end
    if log_request_header then
        local encoded = encode_headers(ngx.req.get_headers())
        if encoded == "" then
            encoded = normalize_raw_header(ngx.req.raw_header())
        end
        ngx.var.cdn_req_headers = encoded
    end
    if log_request_body then
        local limit_kb = tonumber(domain_conf.log_request_body_size_limit) or 0
        if limit_kb <= 0 then
            limit_kb = 16
        end
        local limit_bytes = limit_kb * 1024
        local body = read_body(limit_bytes)
        if body and #body > limit_bytes then
            body = string.sub(body, 1, limit_bytes)
        end
        ngx.var.cdn_req_body = body or ""
    end
end

local function resolve_geo_record(ip)
    if not ip or not _G.IP_SEARCHER then
        return nil
    end
    local ok, res = pcall(function()
        return _G.IP_SEARCHER:search(ip)
    end)
    if not ok or not res then
        return nil
    end
    if type(res) == "table" then
        return res
    end
    local raw = tostring(res)
    local parts = {}
    for part in string.gmatch(raw, "([^|]+)") do
        table.insert(parts, part)
    end
    return {
        country = parts[1] or "",
        region = parts[2] or "",
        province = parts[3] or "",
        city = parts[4] or "",
        isp = parts[5] or "",
        raw = raw,
    }
end

local function contains_any(haystack, needles)
    if not haystack or haystack == "" or not needles or #needles == 0 then
        return false
    end
    local lower = string.lower(haystack)
    for _, item in ipairs(needles) do
        local token = string.lower(item)
        if token ~= "" and string.find(lower, token, 1, true) then
            return true
        end
    end
    return false
end

local function match_redirect_conditions(rule, host, port, ip)
    local conditions = rule and rule.conditions
    if type(conditions) ~= "table" or #conditions == 0 then
        return true
    end
    local ua = ngx.var.http_user_agent or ""
    local referer = ngx.var.http_referer or ""
    local accept_lang = ngx.var.http_accept_language or ""
    local geo = resolve_geo_record(ip)
    local country = resolve_country(ip)
    for _, cond in ipairs(conditions) do
        local key = cond.key or cond.item or ""
        local value = cond.value or ""
        local list = split_pipe(value)
        if key == "domain_port" then
            local candidate = host .. ":" .. (port or "")
            if not contains_any(candidate, list) then
                return false
            end
        elseif key == "user_agent" then
            if not contains_any(ua, list) then
                return false
            end
        elseif key == "referer" then
            if not contains_any(referer, list) then
                return false
            end
        elseif key == "accept_language" then
            if not contains_any(accept_lang, list) then
                return false
            end
        elseif key == "country_code" then
            if not contains_any(country or "", list) then
                return false
            end
        elseif key == "province" then
            if not geo or not contains_any(geo.province or "", list) then
                return false
            end
        elseif key == "city" then
            if not geo or not contains_any(geo.city or "", list) then
                return false
            end
        elseif key == "isp" then
            if not geo or not contains_any(geo.isp or "", list) then
                return false
            end
        elseif key == "asn" or key == "as" then
            return false
        end
    end
    return true
end

local function apply_url_redirects(domain_conf, uri, args, host, port, ip)
    local rules = domain_conf and domain_conf.url_redirects
    if type(rules) ~= "table" then
        return false
    end
    for _, rule in ipairs(rules) do
        local pattern = rule.match or ""
        local target = rule.redirect or ""
        if pattern ~= "" and target ~= "" then
            local ok, m = pcall(ngx.re.find, uri, pattern, "jo")
            if ok and m then
                if match_redirect_conditions(rule, host, port, ip) then
                    local replaced, _, err = ngx.re.sub(uri, pattern, target, "jo")
                    if replaced then
                        target = replaced
                    else
                        ngx.log(ngx.WARN, "redirect regex failed: ", err or "")
                    end
                    if args and args ~= "" and not string.find(target, "?", 1, true) then
                        target = target .. "?" .. args
                    end
                    local code = tostring(rule.code or "")
                    if code == "internal" then
                        ngx.req.set_uri(target, false)
                        return false
                    end
                    local status = tonumber(code) or 302
                    return ngx.redirect(target, status)
                end
            end
        end
    end
    return false
end

local function apply_url_rewrites(domain_conf, uri, args)
    local rules = domain_conf and domain_conf.url_rewrites
    if type(rules) ~= "table" then
        return false
    end
    for _, rule in ipairs(rules) do
        local pattern = rule.match or ""
        local target = rule.replace or rule.redirect or ""
        if pattern ~= "" and target ~= "" then
            local ok, m = pcall(ngx.re.find, uri, pattern, "jo")
            if ok and m then
                local replaced, _, err = ngx.re.sub(uri, pattern, target, "jo")
                if replaced then
                    target = replaced
                else
                    ngx.log(ngx.WARN, "rewrite regex failed: ", err or "")
                end
                if args and args ~= "" and not string.find(target, "?", 1, true) then
                    target = target .. "?" .. args
                end
                local code = tostring(rule.code or "")
                if code == "" or code == "internal" then
                    ngx.req.set_uri(target, false)
                    return false
                end
                local status = tonumber(code) or 302
                return ngx.redirect(target, status)
            end
        end
    end
    return false
end

local function find_default_domain(config)
    if not config or type(config.domains) ~= "table" then
        return nil
    end
    for _, domain in ipairs(config.domains) do
        if domain.default_site then
            return domain
        end
    end
    return nil
end

local function lookup_domain_conf(config, host)
    if not config then
        return nil
    end
    if config.domain_map and host and host ~= "" then
        local domain = config.domain_map[host]
        if domain then
            return domain
        end
    end
    if config.waf and config.waf.block_unbound_domain then
        return nil
    end
    return find_default_domain(config)
end

local host = ngx.var.host
local config = _G.cdn_config 
local domain_conf = nil

if not config then
    ngx.log(ngx.ERR, "Config not loaded")
    ngx.exit(503)
end

domain_conf = lookup_domain_conf(config, host)

if not domain_conf then
    ngx.log(ngx.WARN, "Unknown domain: ", host)
    ngx.exit(404)
else
    if domain_conf.waf_enable ~= false then
        waf.check()
    end
    apply_realtime_send(domain_conf)
    apply_request_logging(domain_conf)
    local client_ip = ngx.var.remote_addr
    local whitelisted = ip_in_list(domain_conf.white_ips, client_ip)
    local crawler_allowed = false
    local crawler_action = domain_conf.crawler_action or ""
    if crawler_action ~= "" then
        local ua = ngx.var.http_user_agent or ""
        local is_crawler = is_crawler_ua(ua)
        if crawler_action == "block" and is_crawler then
            ngx.exit(403)
        end
        if crawler_action == "allow" and is_crawler then
            crawler_allowed = true
        end
    end

    ngx.ctx.guard_pass_ttl = domain_conf.guard_pass_ttl
    ngx.ctx.guard_block_ttl = domain_conf.guard_block_ttl
    if domain_conf.cookie and domain_conf.cookie.enable and domain_conf.cookie.domain then
        ngx.ctx.guard_cookie_domain = domain_conf.cookie.domain
    else
        ngx.ctx.guard_cookie_domain = nil
    end

    if not whitelisted and ip_in_list(domain_conf.black_ips, client_ip) then
        ngx.exit(403)
    end
    if not whitelisted and region_blocked(domain_conf, client_ip) then
        ngx.exit(403)
    end
    if not whitelisted and domain_conf.block_transparent_proxy then
        local xff = ngx.var.http_x_forwarded_for
        local via = ngx.var.http_via
        if (xff and xff ~= "") or (via and via ~= "") then
            ngx.exit(403)
        end
    end
    if not whitelisted and not crawler_allowed and not hotlink_allowed(domain_conf, ngx.var.uri or "") then
        ngx.exit(403)
    end

    if apply_url_rewrites(domain_conf, ngx.var.uri or "", ngx.var.args or "") then
        return
    end

    if apply_url_redirects(domain_conf, ngx.var.uri or "", ngx.var.args or "", host or "", ngx.var.server_port or "", client_ip) then
        return
    end

    local cors_enabled = apply_cors_headers(domain_conf.cors)
    if cors_enabled and ngx.req.get_method() == "OPTIONS" then
        ngx.status = 204
        ngx.header["Content-Length"] = "0"
        ngx.exit(204)
    end

    if not crawler_allowed then
        cc.check(domain_conf, client_ip, ngx.var.uri)
    end

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
    local targets = nil
    local use_l2 = false
    if config.upstream_map then
        if domain_conf.use_l2 and domain_conf.l2_upstream_key and domain_conf.l2_upstream_key ~= "" then
            if config.upstream_map[domain_conf.l2_upstream_key] then
                upstream_key = domain_conf.l2_upstream_key
                use_l2 = true
            end
        end
        targets = config.upstream_map[upstream_key]
    end
    if targets then
        
        -- Get Policy from Domain Config (default to round_robin)
        local policy = domain_conf.load_balance_policy or "round_robin"
        
        -- Use Balancer Logic
        local target_addr = balancer.get_target(upstream_key, targets, policy)
        
        if target_addr then
            local scheme = domain_conf.origin_protocol or "http"
            scheme = string.lower(scheme)
            local target = target_addr
            if not string.find(target_addr, ":", 1, true) then
                local port = ""
                if use_l2 then
                    if scheme == "https" and domain_conf.l2_https_port and domain_conf.l2_https_port ~= "" then
                        port = domain_conf.l2_https_port
                    elseif scheme == "http" and domain_conf.l2_http_port and domain_conf.l2_http_port ~= "" then
                        port = domain_conf.l2_http_port
                    end
                end
                if port == "" then
                    if scheme == "https" and domain_conf.origin_https_port and domain_conf.origin_https_port ~= "" then
                        port = domain_conf.origin_https_port
                    elseif scheme == "http" and domain_conf.origin_http_port and domain_conf.origin_http_port ~= "" then
                        port = domain_conf.origin_http_port
                    end
                end
                if port ~= "" then
                    target = target_addr .. ":" .. port
                end
            end
            ngx.var.backend_target = scheme .. "://" .. target
            ngx.ctx.l2_used = use_l2
            
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
