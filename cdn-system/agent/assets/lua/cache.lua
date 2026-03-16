-- lua/cache.lua
local _M = {}
local bit = require "bit"

local function ends_with(str, suffix)
    if not str or not suffix then
        return false
    end
    return string.sub(str, -#suffix) == suffix
end

local function normalize_ext(ext)
    if not ext or ext == "" then
        return ""
    end
    if string.sub(ext, 1, 1) ~= "." then
        return "." .. ext
    end
    return ext
end

local function split_list(value)
    local raw = tostring(value or "")
    raw = string.gsub(raw, "^%s+", "")
    raw = string.gsub(raw, "%s+$", "")
    if raw == "" then
        return {}
    end
    local list = {}
    if string.find(raw, "|", 1, true) then
        for part in string.gmatch(raw, "([^|]+)") do
            local item = string.match(part, "^%s*(.-)%s*$")
            if item ~= "" then
                table.insert(list, item)
            end
        end
        return list
    end
    for part in string.gmatch(raw, "%S+") do
        local item = string.match(part, "^%s*(.-)%s*$")
        if item ~= "" then
            table.insert(list, item)
        end
    end
    return list
end

local function match_string(candidate, value)
    local list = split_list(value)
    if #list == 0 then
        return false
    end
    candidate = tostring(candidate or "")
    if candidate == "" then
        return false
    end
    for _, item in ipairs(list) do
        if item ~= "" and (candidate == item or string.find(candidate, item, 1, true)) then
            return true
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

local function match_ip(ip, value)
    local list = split_list(value)
    if #list == 0 or not ip or ip == "" then
        return false
    end
    for _, item in ipairs(list) do
        if item ~= "" then
            if string.find(item, "/", 1, true) then
                if ip_in_cidr(ip, item) then
                    return true
                end
            elseif ip == item then
                return true
            end
        end
    end
    return false
end

local function match_custom(value)
    local raw = tostring(value or "")
    raw = string.match(raw, "^%s*(.-)%s*$")
    if raw == "" then
        return false
    end
    local name, expected = raw:match("^(.-)=(.*)$")
    if name then
        name = string.match(name, "^%s*(.-)%s*$")
        expected = string.match(expected or "", "^%s*(.-)%s*$")
        if name == "" then
            return false
        end
        local candidate = ngx.var[name] or ""
        if expected == "" then
            return candidate ~= ""
        end
        return match_string(candidate, expected)
    end
    local candidate = ngx.var[raw] or ""
    return candidate ~= ""
end

local function match_skip_condition(cond)
    if type(cond) ~= "table" then
        return false
    end
    local cond_type = tostring(cond.type or cond.item or "")
    local value = cond.value or ""
    if cond_type == "request_uri" then
        return match_string(ngx.var.request_uri or "", value)
    elseif cond_type == "uri" then
        return match_string(ngx.var.uri or "", value)
    elseif cond_type == "ip" or cond_type == "remote_addr" then
        return match_ip(ngx.var.remote_addr or "", value)
    elseif cond_type == "scheme" then
        return match_string(ngx.var.scheme or "", value)
    elseif cond_type == "args" then
        return match_string(ngx.var.args or "", value)
    elseif cond_type == "domain" or cond_type == "host" then
        return match_string(ngx.var.host or "", value)
    elseif cond_type == "custom" then
        return match_custom(value)
    end
    return false
end

local function match_rule(rule, uri)
    if not rule or not uri then
        return false
    end
    local ext = rule.ext
    if ext and ext ~= "" then
        if ends_with(uri, normalize_ext(ext)) then
            return true
        end
    end
    local prefix = rule.prefix
    if prefix and prefix ~= "" then
        if string.sub(uri, 1, #prefix) == prefix then
            return true
        end
    end
    local rule_uri = rule.uri
    if rule_uri and rule_uri ~= "" then
        if string.find(uri, rule_uri, 1, true) then
            return true
        end
    end
    return false
end

function _M.resolve(domain_conf, uri)
    local cache_cfg = domain_conf and domain_conf.cache
    if not cache_cfg then
        return true, nil
    end

    local enabled = cache_cfg.enable
    if enabled == false or enabled == 0 or enabled == "0" then
        return true, nil
    end

    local ttl = cache_cfg.default_ttl
    local rules = cache_cfg.rules
    if rules then
        for _, rule in ipairs(rules) do
            if match_rule(rule, uri) then
                if rule.skip_conditions and type(rule.skip_conditions) == "table" then
                    for _, cond in ipairs(rule.skip_conditions) do
                        if match_skip_condition(cond) then
                            return true, ttl
                        end
                    end
                end
                if rule.enable == false or rule.enable == 0 or rule.enable == "0" then
                    return true, ttl
                end
                if rule.ttl and tonumber(rule.ttl) then
                    ttl = tonumber(rule.ttl)
                end
                return false, ttl
            end
        end
    end

    return false, ttl
end

return _M
