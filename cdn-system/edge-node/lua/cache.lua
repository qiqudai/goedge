-- lua/cache.lua
local _M = {}

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
