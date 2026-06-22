-- lua/access_guard.lua
local ip_block = require "lua.ip_block"
local waf = require "lua.waf"
local quota = require "lua.quota"
local cc = require "lua.cc"
local acl = require "lua.acl"
local balancer = require "lua.balancer"
local cache = require "lua.cache"
local geo_country = require "lua.geo_country"
local error_page_block = require "lua.error_page_block"
local origin_auto = require "lua.origin_auto"
local bit = require "bit"
local cjson = require "cjson.safe"

local function build_reason(source_type, rule, extras)
    local parts = {
        "type=" .. tostring(source_type or "local_protection"),
        "module=lua.access_guard",
        "rule=" .. tostring(rule or source_type or "unknown"),
        "rule_id=0"
    }
    if type(extras) == "table" then
        for _, item in ipairs(extras) do
            if item and item ~= "" then
                table.insert(parts, tostring(item))
            end
        end
    end
    return table.concat(parts, ";")
end

local function block_request(reason, status)
    local source = reason or build_reason("local_protection", "unknown")
    ngx.header["X-Block-Source"] = source
    ngx.exit(status or 403)
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
        country = geo_country.from_ip2region(raw)
    end
    if country == "" then
        return ""
    end
    country = string.upper(country)
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
        local normalized = geo_country.to_iso(tostring(code or ""), tostring(code or ""))
        if normalized ~= "" and normalized == country then
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

local function clean_geo_value(value)
    if value == nil then
        return ""
    end
    value = tostring(value)
    value = string.match(value, "^%s*(.-)%s*$") or ""
    if value == "" or value == "0" or value == "-" then
        return ""
    end
    return value
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
    local cache = _G.IP_LRU_CACHE
    if cache then
        local cached = cache:get("geo_record:" .. ip)
        if cached ~= nil then
            return cached
        end
    end
    local ok, res = pcall(function()
        return _G.IP_SEARCHER:search(ip)
    end)
    if not ok or not res then
        return nil
    end
    local record = nil
    if type(res) == "table" then
        record = {
            country = clean_geo_value(res.country or res[1] or ""),
            region = clean_geo_value(res.region or res.province or res[2] or ""),
            province = clean_geo_value(res.province or res.region or res[2] or ""),
            city = clean_geo_value(res.city or res[3] or ""),
            isp = clean_geo_value(res.isp or res[4] or ""),
            raw = clean_geo_value(res.raw or ""),
        }
    else
        local raw = tostring(res)
        local parts = {}
        for part in string.gmatch(raw, "([^|]+)") do
            table.insert(parts, part)
        end
        record = {
            country = clean_geo_value(parts[1] or ""),
            region = clean_geo_value(parts[2] or ""),
            province = clean_geo_value(parts[2] or ""),
            city = clean_geo_value(parts[3] or ""),
            isp = clean_geo_value(parts[4] or ""),
            raw = raw,
        }
    end
    if cache and record then
        cache:set("geo_record:" .. ip, record, 600)
    end
    return record
end

local function apply_geo_logging(ip)
    ngx.var.client_country = ""
    ngx.var.client_province = ""
    ngx.var.client_city = ""
    ngx.var.client_isp = ""
    local geo = resolve_geo_record(ip)
    if not geo then
        return
    end
    ngx.var.client_country = geo.country or ""
    ngx.var.client_province = geo.province or ""
    ngx.var.client_city = geo.city or ""
    ngx.var.client_isp = geo.isp or ""
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

local function should_handle_url_rule(rule, rewrite)
    if type(rule) ~= "table" then
        return false
    end
    local conditions = rule.conditions
    if type(conditions) == "table" and #conditions > 0 then
        return true
    end
    local code = tostring(rule.code or "")
    if code == "" or code == "internal" or code == "301" or code == "302" then
        if rewrite and (code == "" or code == "internal") then
            local target = rule.replace or rule.redirect or ""
            return not string.find(target, "^/", 1)
        end
        return false
    end
    return true
end

local function apply_url_redirects(domain_conf, uri, args, host, port, ip)
    local rules = domain_conf and domain_conf.url_redirects
    if type(rules) ~= "table" then
        return false
    end
    for _, rule in ipairs(rules) do
        if should_handle_url_rule(rule, false) then
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
    end
    return false
end

local function apply_url_rewrites(domain_conf, uri, args)
    local rules = domain_conf and domain_conf.url_rewrites
    if type(rules) ~= "table" then
        return false
    end
    for _, rule in ipairs(rules) do
        if should_handle_url_rule(rule, true) then
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
    end
    return false
end

local function ipv4_to_num(ip)
    if not ip or ip == "" then
        return nil
    end
    local a, b, c, d = ip:match("^(%d+)%.(%d+)%.(%d+)%.(%d+)$")
    if not a then
        return nil
    end
    a, b, c, d = tonumber(a), tonumber(b), tonumber(c), tonumber(d)
    if not a or not b or not c or not d then
        return nil
    end
    return a * 16777216 + b * 65536 + c * 256 + d
end

local function ip_in_cidr(ip, cidr)
    local base, prefix = cidr:match("^(%d+%.%d+%.%d+%.%d+)%s*/%s*(%d+)$")
    if not base or not prefix then
        return false
    end
    local ip_num = ipv4_to_num(ip)
    local base_num = ipv4_to_num(base)
    if not ip_num or not base_num then
        return false
    end
    local bits = tonumber(prefix)
    if not bits or bits < 0 or bits > 32 then
        return false
    end
    local mask = bits == 0 and 0 or bit.lshift(0xFFFFFFFF, 32 - bits)
    mask = bit.band(mask, 0xFFFFFFFF)
    return bit.band(ip_num, mask) == bit.band(base_num, mask)
end

local function ip_in_ranges(ip, value)
    local list = split_pipe(value)
    if #list == 0 then
        return false
    end
    for _, item in ipairs(list) do
        local trimmed = string.match(item, "^%s*(.-)%s*$")
        if trimmed ~= "" then
            if string.find(trimmed, "/", 1, true) then
                if ip_in_cidr(ip, trimmed) then
                    return true
                end
            elseif ip == trimmed then
                return true
            end
        end
    end
    return false
end

local function get_header_value(name)
    if not name or name == "" then
        return ""
    end
    local headers = ngx.req.get_headers()
    local val = headers[name] or headers[string.lower(name)] or ""
    if type(val) == "table" then
        return table.concat(val, ",")
    end
    return val
end

local function match_operator(candidate, operator, value)
    candidate = candidate or ""
    operator = operator or "eq"
    value = value or ""
    if operator == "exists" then
        return candidate ~= ""
    elseif operator == "not_exists" then
        return candidate == ""
    elseif operator == "ip_range" then
        return ip_in_ranges(candidate, value)
    elseif operator == "not_ip_range" then
        return not ip_in_ranges(candidate, value)
    end

    local list = split_pipe(value)
    if operator == "eq" then
        if #list == 0 then
            return candidate == value
        end
        for _, item in ipairs(list) do
            if candidate == item then
                return true
            end
        end
        return false
    elseif operator == "neq" then
        if #list == 0 then
            return candidate ~= value
        end
        for _, item in ipairs(list) do
            if candidate == item then
                return false
            end
        end
        return true
    elseif operator == "contains" then
        return contains_any(candidate, list)
    elseif operator == "not_contains" then
        return not contains_any(candidate, list)
    elseif operator == "prefix" then
        for _, item in ipairs(list) do
            if item ~= "" and candidate:sub(1, #item) == item then
                return true
            end
        end
        return false
    elseif operator == "suffix" then
        for _, item in ipairs(list) do
            if item ~= "" and ends_with(candidate, item) then
                return true
            end
        end
        return false
    elseif operator == "regex" or operator == "not_regex" then
        if #list == 0 then
            list = { value }
        end
        local matched = false
        for _, pattern in ipairs(list) do
            if pattern ~= "" then
                local ok, found = pcall(ngx.re.find, candidate, pattern, "jo")
                if ok and found then
                    matched = true
                    break
                end
            end
        end
        if operator == "regex" then
            return matched
        end
        return not matched
    end
    return false
end

local function match_origin_condition(cond, ctx)
    if not cond or not ctx then
        return false
    end
    local item = cond.item or ""
    local operator = cond.operator or "eq"
    local value = cond.value or ""
    local candidate = ""
    if item == "uri" then
        candidate = ctx.request_uri
    elseif item == "uri_no_args" then
        candidate = ctx.uri
    elseif item == "domain" then
        candidate = ctx.host
    elseif item == "client_ip" then
        candidate = ctx.client_ip
    elseif item == "method" then
        candidate = ctx.method
    elseif item == "http_version" then
        candidate = ctx.http_version
    elseif item == "header" then
        candidate = get_header_value(cond.header)
    elseif item == "client_country" then
        candidate = ctx.country
    elseif item == "client_province" then
        candidate = ctx.province
    elseif item == "client_city" then
        candidate = ctx.city
    elseif item == "client_isp" then
        candidate = ctx.isp
    elseif item == "node_country" then
        candidate = ctx.node_country
    elseif item == "node_province" then
        candidate = ctx.node_province
    elseif item == "node_city" then
        candidate = ctx.node_city
    elseif item == "node_isp" then
        candidate = ctx.node_isp
    else
        return false
    end
    return match_operator(candidate or "", operator, value)
end

local function split_origin_list(value)
    if not value or value == "" then
        return {}
    end
    local items = split_pipe(value)
    if #items > 0 then
        return items
    end
    local out = {}
    for part in string.gmatch(value, "%S+") do
        table.insert(out, part)
    end
    return out
end

local function select_condition_origin(domain_conf, ctx)
    local conds = domain_conf and domain_conf.origin_conditions
    if type(conds) ~= "table" then
        return nil
    end
    for _, cond in ipairs(conds) do
        if match_origin_condition(cond, ctx) then
            return cond.origin or ""
        end
    end
    return nil
end

local function resolve_origin_scheme(protocol)
    local proto = string.lower(protocol or "http")
    if proto == "follow" or proto == "follow_port" then
        return ngx.var.scheme or "http", proto
    end
    return proto, proto
end

local function build_backend_target(addr, domain_conf, use_l2)
    local scheme, proto = resolve_origin_scheme(domain_conf.origin_protocol)
    local target = addr
    if not string.find(addr, ":", 1, true) then
        if proto == "follow_port" then
            local port = ngx.var.server_port
            if port and port ~= "" then
                target = addr .. ":" .. port
            end
        else
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
                target = addr .. ":" .. port
            end
        end
    end
    return scheme .. "://" .. target
end

local function select_backend_target(domain_conf, client_ip)
    local config = _G.cdn_config
    if not config then
        return nil
    end
    local use_l2 = false
    if config.upstream_map and domain_conf.use_l2 and domain_conf.l2_upstream_key and domain_conf.l2_upstream_key ~= "" then
        if config.upstream_map[domain_conf.l2_upstream_key] then
            use_l2 = true
        end
    end
    ngx.ctx.l2_used = use_l2

    local ctx = {
        request_uri = ngx.var.request_uri or "",
        uri = ngx.var.uri or "",
        host = ngx.var.host or "",
        client_ip = client_ip or "",
        method = ngx.req.get_method(),
        http_version = tostring(ngx.req.http_version() or ""),
        country = resolve_country(client_ip),
    }
    local geo = resolve_geo_record(client_ip)
    if geo then
        ctx.province = geo.province or ""
        ctx.city = geo.city or ""
        ctx.isp = geo.isp or ""
    end
    local node_ip = ngx.var.server_addr or ""
    local node_geo = resolve_geo_record(node_ip)
    ctx.node_country = resolve_country(node_ip)
    if node_geo then
        ctx.node_province = node_geo.province or ""
        ctx.node_city = node_geo.city or ""
        ctx.node_isp = node_geo.isp or ""
    end
    if not use_l2 then
        local override = select_condition_origin(domain_conf, ctx)
        if override and override ~= "" then
            local items = split_origin_list(override)
            if #items > 0 then
                local targets = {}
                for _, item in ipairs(items) do
                    table.insert(targets, { addr = item })
                end
                local key = "cond:" .. override
                local addr = balancer.get_target(key, targets, "round_robin")
                if addr then
                    return build_backend_target(addr, domain_conf, false)
                end
            end
        end
    end

    if not config.upstream_map then
        return nil
    end
    local upstream_key = domain_conf.upstream_key
    if use_l2 then
        upstream_key = domain_conf.l2_upstream_key
    end
    if not upstream_key or upstream_key == "" then
        return nil
    end
    local targets = config.upstream_map[upstream_key]
    if not targets then
        return nil
    end
    local policy = domain_conf.load_balance_policy or "round_robin"
    local addr = balancer.get_target(upstream_key, targets, policy)
    if not addr then
        return nil
    end
    return build_backend_target(addr, domain_conf, use_l2)
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

local function lookup_domain_conf()
    local config = _G.cdn_config
    if not config then
        return nil
    end
    local host = ngx.var.host
    if config.domain_map and host and host ~= "" then
        local domain = config.domain_map[host]
        if domain then
            return domain
        end
        local matched = nil
        local matched_len = 0
        for pattern, conf in pairs(config.domain_map) do
            if type(pattern) == "string" and string.sub(pattern, 1, 2) == "*." then
                local suffix = string.sub(pattern, 2)
                if suffix and suffix ~= "" and string.sub(host, -string.len(suffix)) == suffix then
                    local plen = string.len(pattern)
                    if plen > matched_len then
                        matched = conf
                        matched_len = plen
                    end
                end
            end
        end
        if matched then
            return matched
        end
    end
    if config.waf and config.waf.block_unbound_domain then
        return nil
    end
    return find_default_domain(config)
end

if ip_block.is_blocked(ngx.var.remote_addr) then
    error_page_block.exit_blocked(build_reason("ip_block", "ip_block", {"condition=ip_in_blacklist"}))
end

local function normalize_domain_conf(conf)
    if not conf then
        return
    end
    if type(conf.cookie) ~= "table" then
        conf.cookie = nil
    end
    if type(conf.cors) ~= "table" then
        conf.cors = nil
    end
    if type(conf.hotlink) ~= "table" then
        conf.hotlink = nil
    end
    if type(conf.url_redirects) ~= "table" then
        conf.url_redirects = nil
    end
    if type(conf.url_rewrites) ~= "table" then
        conf.url_rewrites = nil
    end
    if type(conf.origin_conditions) ~= "table" then
        conf.origin_conditions = nil
    end
    if type(conf.headers) ~= "table" then
        conf.headers = nil
    end
    if type(conf.response_headers) ~= "table" then
        conf.response_headers = nil
    end
    if type(conf.acl_rules) ~= "table" then
        conf.acl_rules = nil
    end
    if type(conf.white_ips) ~= "table" then
        conf.white_ips = nil
    end
    if type(conf.black_ips) ~= "table" then
        conf.black_ips = nil
    end
end

local domain_conf = lookup_domain_conf()
if domain_conf then
    normalize_domain_conf(domain_conf)
    apply_realtime_send(domain_conf)
    apply_request_logging(domain_conf)
    local client_ip = ngx.var.remote_addr
    apply_geo_logging(client_ip)
    local whitelisted = ip_in_list(domain_conf.white_ips, client_ip)
    if not whitelisted and domain_conf.waf_enable ~= false then
        waf.check()
    end
    if not whitelisted then
        acl.check(domain_conf, client_ip, ngx.var.uri)
    end
    local crawler_allowed = false
    local crawler_action = domain_conf.crawler_action or ""
    if crawler_action ~= "" then
        local ua = ngx.var.http_user_agent or ""
        local is_crawler = is_crawler_ua(ua)
        if crawler_action == "block" and is_crawler then
            block_request(build_reason("local_protection", "crawler_block", {"condition=user_agent", "action=block"}), 403)
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
        error_page_block.exit_blocked(build_reason("local_protection", "ip_deny", {"condition=site_black_ips"}))
    end
    if not whitelisted and region_blocked(domain_conf, client_ip) then
        error_page_block.exit_blocked(build_reason("local_protection", "region_block", {"condition=region_block"}))
    end
    if not whitelisted and domain_conf.block_transparent_proxy then
        local xff = ngx.var.http_x_forwarded_for
        local via = ngx.var.http_via
        if (xff and xff ~= "") or (via and via ~= "") then
            block_request(build_reason("local_protection", "transparent_proxy", {"condition=xff_or_via_present"}), 403)
        end
    end
    if not whitelisted and not crawler_allowed and not hotlink_allowed(domain_conf, ngx.var.uri or "") then
        block_request(build_reason("local_protection", "hotlink", {"condition=referer_not_allowed"}), 403)
    end

    if apply_url_rewrites(domain_conf, ngx.var.uri or "", ngx.var.args or "") then
        return
    end

    if apply_url_redirects(domain_conf, ngx.var.uri or "", ngx.var.args or "", ngx.var.host or "", ngx.var.server_port or "", client_ip) then
        return
    end

    if not whitelisted and not crawler_allowed then
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

    local backend_target = select_backend_target(domain_conf, client_ip)
    if not backend_target or backend_target == "" then
        ngx.log(ngx.ERR, "Upstream not found: ", domain_conf.upstream_key or "")
        return ngx.exit(ngx.HTTP_BAD_GATEWAY)
    end
    ngx.var.backend_target = backend_target
    origin_auto.before_proxy(domain_conf, backend_target)

    quota.check_quota(ngx.var.host)
end
