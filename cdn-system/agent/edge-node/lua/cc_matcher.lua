-- lua/cc_matcher.lua
-- Shared CC matcher engine used by rule groups and site custom rules.
local geo_country = require "lua.geo_country"
local cc_stats = require "lua.cc_stats"
local bit = require "bit"

local _M = {}

local KEY_ALIASES = {
    all = "all",
    ip = "ip",
    domain = "domain",
    uri = "uri",
    req_uri = "uri",
    request_uri = "uri",
    uri_no_args = "uri_no_args",
    request_path = "uri_no_args",
    header = "header",
    user_agent_unique_count = "ua_count",
    ua_count = "ua_count",
    status_404_count = "404_count",
    ["404_count"] = "404_count",
    method = "method",
    user_agent = "ua",
    ua = "ua",
    referer = "referer",
    country = "country",
    country_code = "country",
    asn = "asn",
    as = "asn",
    province = "province",
    city = "city",
    isp = "isp",
    http_version = "http_version",
    accept_language = "accept_language",
    header_accept_language = "accept_language",
}

local function split_lines(value)
    if not value or value == "" then
        return {}
    end
    local list = {}
    for line in string.gmatch(value, "([^\n]+)") do
        local item = string.match(line, "^%s*(.-)%s*$")
        if item ~= "" then
            table.insert(list, item)
        end
    end
    return list
end

local function split_pipe(value)
    local out = {}
    if not value or value == "" then
        return out
    end
    for part in string.gmatch(value, "([^|\n]+)") do
        local item = string.match(part, "^%s*(.-)%s*$")
        if item ~= "" then
            table.insert(out, item)
        end
    end
    return out
end

local function contains_any(haystack, needles)
    if not haystack or haystack == "" or not needles or #needles == 0 then
        return false
    end
    local lower = string.lower(haystack)
    for _, item in ipairs(needles) do
        local token = string.lower(tostring(item))
        if token ~= "" and string.find(lower, token, 1, true) then
            return true
        end
    end
    return false
end

local function ends_with(value, suffix)
    if not value or not suffix or suffix == "" then
        return false
    end
    return string.sub(value, -#suffix) == suffix
end

local function normalize_operator(operator)
    operator = string.lower(tostring(operator or "contains"))
    if operator == "contain" or operator == "contains" then
        return "contains"
    end
    if operator == "equals" or operator == "eq" then
        return "eq"
    end
    if operator == "not_equals" or operator == "neq" then
        return "neq"
    end
    if operator == "not_contains" then
        return "not_contains"
    end
    if operator == "not_regex" then
        return "not_regex"
    end
    if operator == "not_ip_range" then
        return "not_ip_range"
    end
    return operator
end

local function normalize_key(key)
    key = string.lower(tostring(key or ""))
    return KEY_ALIASES[key] or key
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
    if #list == 0 and value and value ~= "" then
        list = split_lines(value)
    end
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
    local record
    if type(res) == "table" then
        record = {
            country = clean_geo_value(res.country or res[1] or ""),
            province = clean_geo_value(res.province or res.region or res[2] or ""),
            city = clean_geo_value(res.city or res[3] or ""),
            isp = clean_geo_value(res.isp or res[4] or ""),
            asn = clean_geo_value(res.asn or res.as or ""),
        }
    else
        local raw = tostring(res)
        local parts = {}
        for part in string.gmatch(raw, "([^|]+)") do
            table.insert(parts, part)
        end
        record = {
            country = clean_geo_value(parts[1] or ""),
            province = clean_geo_value(parts[2] or ""),
            city = clean_geo_value(parts[3] or ""),
            isp = clean_geo_value(parts[4] or ""),
            asn = "",
        }
    end
    if cache and record then
        cache:set("geo_record:" .. ip, record, 600)
    end
    return record
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
    return tostring(val)
end

function _M.build_context(host, ip, uri)
    local request_uri = ngx.var.request_uri or uri or ""
    uri = uri or ngx.var.uri or ""
    local geo = resolve_geo_record(ip)
    local ctx = {
        host = host or ngx.var.host or "",
        ip = ip or ngx.var.remote_addr or "",
        uri = uri,
        request_uri = request_uri,
        method = ngx.req.get_method() or "",
        http_version = tostring(ngx.req.http_version() or ngx.var.server_protocol or ""),
        ua = ngx.var.http_user_agent or "",
        referer = ngx.var.http_referer or "",
        accept_language = ngx.var.http_accept_language or "",
        country = resolve_country(ip),
        province = geo and geo.province or "",
        city = geo and geo.city or "",
        isp = geo and geo.isp or "",
        asn = geo and geo.asn or "",
    }
    return ctx
end

local function match_operator(candidate, operator, value)
    candidate = tostring(candidate or "")
    operator = normalize_operator(operator)
    value = tostring(value or "")

    if operator == "exists" then
        return candidate ~= ""
    end
    if operator == "not_exists" then
        return candidate == ""
    end
    if operator == "ip_range" then
        return ip_in_ranges(candidate, value)
    end
    if operator == "not_ip_range" then
        return not ip_in_ranges(candidate, value)
    end

    local list = split_pipe(value)
    if #list == 0 then
        list = split_lines(value)
    end

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
    end
    if operator == "neq" then
        if #list == 0 then
            return candidate ~= value
        end
        for _, item in ipairs(list) do
            if candidate == item then
                return false
            end
        end
        return true
    end
    if operator == "contains" then
        return contains_any(candidate, list)
    end
    if operator == "not_contains" then
        return not contains_any(candidate, list)
    end
    if operator == "prefix" then
        if #list == 0 and value ~= "" then
            list = { value }
        end
        for _, item in ipairs(list) do
            if item ~= "" and candidate:sub(1, #item) == item then
                return true
            end
        end
        return false
    end
    if operator == "suffix" then
        if #list == 0 and value ~= "" then
            list = { value }
        end
        for _, item in ipairs(list) do
            if item ~= "" and ends_with(candidate, item) then
                return true
            end
        end
        return false
    end
    if operator == "regex" or operator == "not_regex" then
        if #list == 0 and value ~= "" then
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

    return contains_any(candidate, list)
end

local function resolve_candidate(key, rule, ctx)
    key = normalize_key(rule.key or rule.item or key)
    if key == "all" then
        return "1"
    end
    if key == "ip" then
        return ctx.ip
    end
    if key == "domain" then
        return ctx.host
    end
    if key == "uri" then
        return ctx.request_uri
    end
    if key == "uri_no_args" then
        return ctx.uri
    end
    if key == "method" then
        return ctx.method
    end
    if key == "ua" then
        return ctx.ua
    end
    if key == "referer" then
        return ctx.referer
    end
    if key == "accept_language" then
        return ctx.accept_language
    end
    if key == "country" then
        return ctx.country
    end
    if key == "province" then
        return ctx.province
    end
    if key == "city" then
        return ctx.city
    end
    if key == "isp" then
        return ctx.isp
    end
    if key == "asn" then
        return ctx.asn
    end
    if key == "http_version" then
        return ctx.http_version
    end
    if key == "header" then
        return get_header_value(rule.header or rule.name or rule.value_name or rule.header_name)
    end
    if key == "404_count" then
        local window = tonumber(rule.window or rule.within_second) or 60
        return tostring(cc_stats.get_404_count(ctx.ip, window))
    end
    if key == "ua_count" then
        local window = tonumber(rule.window or rule.within_second) or 60
        cc_stats.record_response(ctx.ip, 0, ctx.ua, window)
        return tostring(cc_stats.get_ua_unique_count(ctx.ip, window))
    end
    return ""
end

local function match_single(rule, ctx)
    if not rule then
        return true
    end
    local key = normalize_key(rule.key or rule.item or "all")
    if key == "all" then
        return true
    end
    local candidate = resolve_candidate(key, rule, ctx)
    return match_operator(candidate, rule.operator, rule.value)
end

local function match_rules_list(rules, ctx)
    if type(rules) ~= "table" or #rules == 0 then
        return true
    end
    local has_or = false
    local or_matched = false
    for _, item in ipairs(rules) do
        local logic = string.lower(item.logic or "and")
        local matched = match_single(item, ctx)
        if logic == "or" then
            has_or = true
            if matched then
                or_matched = true
            end
        elseif not matched then
            return false
        end
    end
    if has_or then
        return or_matched
    end
    return true
end

function _M.match_data(matcher_data, ctx)
    if not matcher_data then
        return true
    end
    if type(matcher_data) == "string" then
        return true
    end

    local legacy = matcher_data.req_uri or matcher_data.uri or matcher_data.request_uri
    if legacy then
        local rule = legacy
        if not rule.key and not rule.item then
            rule = { key = "uri", operator = rule.operator, value = rule.value }
        end
        return match_single(rule, ctx)
    end

    if matcher_data.rules then
        return match_rules_list(matcher_data.rules, ctx)
    end

    if matcher_data.matchers then
        return match_rules_list(matcher_data.matchers, ctx)
    end

    return match_single(matcher_data, ctx)
end

function _M.match_operator(candidate, operator, value)
    return match_operator(candidate, operator, value)
end

return _M
