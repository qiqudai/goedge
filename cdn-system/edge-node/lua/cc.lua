-- lua/cc.lua
local cjson = require "cjson.safe"
local guard = require "lua.guard"

local _M = {}
local store = ngx.shared.cc_req_rate

local function split_lines(value)
    if not value or value == "" then
        return nil
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

local function contains_any(haystack, needles)
    if not haystack or haystack == "" or not needles then
        return false
    end
    for _, item in ipairs(needles) do
        if item ~= "" and string.find(haystack, item, 1, true) then
            return true
        end
    end
    return false
end

local function match_rule(matcher_data, uri)
    if not matcher_data then
        return true
    end
    local function match_single(rule)
        if not rule then
            return true
        end
        local key = rule.key or rule.item or "uri"
        if key ~= "uri" and key ~= "req_uri" then
            return true
        end
        if not uri then
            return false
        end
        local operator = rule.operator or "contain"
        local value = rule.value or ""
        local list = split_lines(value) or {}
        if operator == "equals" then
            if #list == 0 then
                return uri == value
            end
            for _, item in ipairs(list) do
                if uri == item then
                    return true
                end
            end
            return false
        elseif operator == "prefix" then
            if #list == 0 then
                return string.sub(uri, 1, #value) == value
            end
            for _, item in ipairs(list) do
                if string.sub(uri, 1, #item) == item then
                    return true
                end
            end
            return false
        else
            return contains_any(uri, list)
        end
    end

    local rule = matcher_data.req_uri or matcher_data.uri
    if rule then
        return match_single(rule)
    end

    local rules = matcher_data.rules
    if type(rules) ~= "table" then
        return true
    end
    local has_or = false
    local or_matched = false
    for _, item in ipairs(rules) do
        local logic = string.lower(item.logic or "and")
        local matched = match_single(item)
        if logic == "or" then
            has_or = true
            if matched then
                or_matched = true
            end
        else
            if not matched then
                return false
            end
        end
    end
    if has_or then
        return or_matched
    end
    return true
end

local function rate_exceeded(filter, host, ip, uri)
    if not store or not filter then
        return false
    end
    if filter.within_second == 0 or filter.max_req == 0 then
        return false
    end

    local window = tonumber(filter.within_second) or 0
    if window <= 0 then
        return false
    end
    local max_req = tonumber(filter.max_req) or 0
    local max_req_per_uri = tonumber(filter.max_req_per_uri) or 0

    local base_key = host .. "|" .. ip
    local current = store:incr(base_key, 1, 0, window)
    if current and max_req > 0 and current > max_req then
        return true
    end

    if max_req_per_uri > 0 and uri then
        local uri_key = base_key .. "|" .. uri
        local uri_count = store:incr(uri_key, 1, 0, window)
        if uri_count and uri_count > max_req_per_uri then
            return true
        end
    end

    return false
end

local function check_rule_id(rule_id, host, ip, uri)
    if rule_id == 0 or not ip then
        return
    end
    if guard.is_guard_request(uri) then
        return
    end
    local config = _G.cdn_config
    if not config then
        return
    end
    local rules = nil
    if config.cc_rules then
        rules = config.cc_rules[tostring(rule_id)] or config.cc_rules[rule_id]
    end
    if not rules then
        return
    end
    for _, rule in ipairs(rules) do
        if rule.enabled == false then
            goto continue
        end
        local matcher_data = nil
        if config.cc_matchers and rule.matcher_id then
            local matcher = config.cc_matchers[tostring(rule.matcher_id)] or config.cc_matchers[rule.matcher_id]
            if matcher and matcher.data and matcher.data ~= "" then
                matcher_data = cjson.decode(matcher.data)
            end
        end
        if not match_rule(matcher_data, uri) then
            goto continue
        end

        local filter = nil
        if config.cc_filters and rule.filter_id then
            filter = config.cc_filters[tostring(rule.filter_id)] or config.cc_filters[rule.filter_id]
        end
        if filter then
            local filter_type = (filter.type or "")
            if guard.is_guard_filter(filter_type) then
                if rate_exceeded(filter, host, ip, uri) then
                    if not guard.ensure_passed(filter, host, ip) then
                        guard.challenge(filter, host, ip)
                        ngx.exit(200)
                    end
                end
            elseif filter_type == "" or filter_type == "req_rate" or filter_type == "block" then
                if rate_exceeded(filter, host, ip, uri) then
                    ngx.exit(429)
                end
            end
        end

        ::continue::
    end
end

function _M.check(domain_conf, ip, uri)
    if not domain_conf or not ip then
        return
    end
    local rule_id = tonumber(domain_conf.cc_rule_id or 0) or 0
    check_rule_id(rule_id, domain_conf.name or "", ip, uri)
end

function _M.check_rule_id(rule_id, host, ip, uri)
    check_rule_id(rule_id, host, ip, uri)
end

return _M
