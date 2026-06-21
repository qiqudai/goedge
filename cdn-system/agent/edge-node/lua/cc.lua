-- lua/cc.lua
local cjson = require "cjson.safe"
local guard = require "lua.guard"
local cc_matcher = require "lua.cc_matcher"
local cc_stats = require "lua.cc_stats"
local ip_block = require "lua.ip_block"
local md5 = require "resty.md5"
local str = require "resty.string"

local _M = {}
local store = ngx.shared.cc_req_rate

local ACTION_TO_FILTER = {
    limit_rate = "req_rate",
    block = "block",
    allow = "allow",
    log = "log",
    invisible = "silent_captcha",
    ["5s"] = "five_seconds",
    click = "click_captcha",
    click_simple = "click_captcha_simple",
    slide = "slide_captcha",
    slide_simple = "slide_captcha_simple",
    captcha = "captcha",
    rotate = "rotate_captcha",
    ["302"] = "302",
    url_auth = "url_auth",
}

local function block_request(reason, status)
    local source = reason or "type=cc;module=lua.cc;rule=cc_rate_limit;rule_id=0;condition=unknown"
    ngx.header["X-Block-Source"] = source
    ngx.exit(status or 429)
end

local function build_cc_reason(rule_name, rule_id, rule, filter, extra)
    local parts = {"type=cc", "module=lua.cc", "rule=" .. tostring(rule_name or "cc"), "rule_id=" .. tostring(rule_id or 0)}
    if rule and rule.filter_id then
        table.insert(parts, "filter_id=" .. tostring(rule.filter_id))
    end
    if rule and rule.matcher_id then
        table.insert(parts, "matcher_id=" .. tostring(rule.matcher_id))
    end
    if filter then
        if filter.type ~= nil then
            table.insert(parts, "config=" .. tostring(filter.type))
        end
        if filter.within_second ~= nil then
            table.insert(parts, "window=" .. tostring(filter.within_second))
        end
        if filter.max_req ~= nil then
            table.insert(parts, "max_req=" .. tostring(filter.max_req))
        end
        if filter.max_req_per_uri ~= nil then
            table.insert(parts, "max_req_per_uri=" .. tostring(filter.max_req_per_uri))
        end
    end
    if extra and extra ~= "" then
        table.insert(parts, extra)
    end
    return table.concat(parts, ";")
end

local function parse_filter_extra(filter)
    if not filter or not filter.extra or filter.extra == "" then
        return {}
    end
    local ok, data = pcall(cjson.decode, filter.extra)
    if ok and type(data) == "table" then
        return data
    end
    return {}
end

local function md5_hex(value)
    local m = md5:new()
    if not m then
        return ""
    end
    m:update(tostring(value or ""))
    local digest = m:final()
    if not digest then
        return ""
    end
    return str.to_hex(digest)
end

local function rate_exceeded(filter, host, ip, uri)
    if not store or not filter then
        return false, ""
    end
    local window = tonumber(filter.within_second) or 0
    if window <= 0 then
        return false, ""
    end
    local max_req = tonumber(filter.max_req) or 0
    local max_req_per_uri = tonumber(filter.max_req_per_uri) or 0
    if max_req <= 0 and max_req_per_uri <= 0 then
        return false, ""
    end

    local base_key = host .. "|" .. ip
    if max_req > 0 then
        local current = store:incr(base_key, 1, 0, window)
        if current and current > max_req then
            return true, "scope=host_ip;current=" .. tostring(current)
        end
    end

    if max_req_per_uri > 0 and uri then
        local uri_key = base_key .. "|" .. uri
        local uri_count = store:incr(uri_key, 1, 0, window)
        if uri_count and uri_count > max_req_per_uri then
            return true, "scope=uri;current=" .. tostring(uri_count) .. ";uri=" .. tostring(uri)
        end
    end

    return false, ""
end

local function should_force_guard(filter)
    local window = tonumber(filter.within_second) or 0
    local max_req = tonumber(filter.max_req) or 0
    local max_req_per_uri = tonumber(filter.max_req_per_uri) or 0
    if window <= 0 then
        return max_req <= 0 and max_req_per_uri <= 0
    end
    return max_req <= 0 and max_req_per_uri <= 0
end

local function verify_url_auth(filter, host, ip, uri)
    local extra = parse_filter_extra(filter)
    local auth = extra.auth or {}
    local key = tostring(auth.key or "")
    if key == "" then
        return false, "missing_key"
    end
    local sign_param = tostring(auth.sign_param or "sign")
    local time_param = tostring(auth.time_param or "t")
    local method = string.upper(tostring(auth.method or "A"))
    local max_diff = tonumber(auth.max_time_diff) or 180
    local max_usage = tonumber(auth.max_sign_usage) or 0

    local args = ngx.req.get_uri_args() or {}
    local sign = tostring(args[sign_param] or "")
    if sign == "" then
        return false, "missing_sign"
    end

    local path = uri or ngx.var.uri or "/"
    local expected = ""
    local raw = ""

    if method == "B" then
        raw = key .. path
        expected = md5_hex(raw)
    else
        local ts = tostring(args[time_param] or "")
        if ts == "" then
            return false, "missing_time"
        end
        local ts_num = tonumber(ts)
        if not ts_num then
            return false, "invalid_time"
        end
        if math.abs(ngx.time() - ts_num) > max_diff then
            return false, "expired"
        end
        raw = key .. path .. ts
        expected = md5_hex(raw)
    end

    if string.lower(sign) ~= string.lower(expected) then
        return false, "bad_sign"
    end

    if auth.ip_auth == true then
        local bound_ip = tostring(args.ip or args.client_ip or "")
        if bound_ip ~= "" and bound_ip ~= ip then
            return false, "ip_mismatch"
        end
    end

    if max_usage > 0 and store then
        local usage_key = "cc:auth:" .. host .. ":" .. md5_hex(sign .. "|" .. path)
        local used = store:incr(usage_key, 1, 0, max_diff > 0 and max_diff or 180)
        if used and used > max_usage then
            return false, "sign_reuse"
        end
    end

    return true, ""
end

local function block_ttl()
    local ttl = tonumber(ngx.ctx.guard_block_ttl) or 3600
    if ttl <= 0 then
        ttl = 3600
    end
    return ttl
end

local function blacklist_on_trigger(ip)
    if ip and ip ~= "" then
        ip_block.block(ip, block_ttl())
    end
end

local function apply_redirect_302(filter, host, ip)
    blacklist_on_trigger(ip)
    local extra = parse_filter_extra(filter)
    local target = tostring(extra.redirect_url or extra.url or "")
    if target == "" then
            if not guard.ensure_passed(filter, host, ip) then
                ngx.header["X-Block-Source"] = build_cc_reason("cc_guard", filter.id or 0, nil, filter, "config=302")
                guard.challenge(filter, host, ip)
                ngx.exit(200)
            end
        return true
    end
    ngx.header["X-Block-Source"] = build_cc_reason("cc_redirect", filter.id or 0, nil, filter, "config=302")
    return ngx.redirect(target, 302)
end

local function rule_should_stop(rule)
    if not rule then
        return false
    end
    local mode = string.lower(tostring(rule.mode or ""))
    if mode == "stop" or rule.breakMatch == true or rule.break_match == true then
        return true
    end
    return false
end

local function finalize_allow(rule)
    if rule_should_stop(rule) then
        return "allow"
    end
    return "continue"
end

local function enforce_action(action, rule, rule_id, host, ip, uri, filter, detail)
    action = string.lower(tostring(action or "block"))
    if action == "allow" then
        return finalize_allow(rule)
    end
    if action == "log" then
        ngx.log(ngx.INFO, "cc log action host=", host or "", " ip=", ip or "", " uri=", uri or "", " detail=", detail or "")
        return "continue"
    end

    blacklist_on_trigger(ip)

    if action == "exit" then
        ngx.header["X-Block-Source"] = build_cc_reason("cc_exit", rule_id, rule, filter, detail or "action=exit")
        ngx.exit(444)
    end

    local reason_key = "cc_block"
    if action == "ipset" then
        reason_key = "cc_ipset"
    elseif action == "limit_rate" then
        reason_key = "cc_rate_limit"
    end
    block_request(build_cc_reason(reason_key, rule_id, rule, filter, detail or ("action=" .. action)), 403)
end

local function probe_filter(filter, rule, rule_id, host, ip, uri, action_override)
    if not filter then
        local action = string.lower(tostring(action_override or rule and rule.action or ""))
        if action == "allow" then
            return "allow", ""
        end
        if action == "log" then
            ngx.log(ngx.INFO, "cc log rule matched host=", host or "", " ip=", ip or "", " uri=", uri or "")
            return "continue", ""
        end
        if action == "block" or action == "ipset" or action == "exit" then
            return "fail", "action=" .. action
        end
        return "continue", ""
    end

    local filter_type = string.lower(tostring(filter.type or ""))
    local extra = parse_filter_extra(filter)
    local action = string.lower(tostring(action_override or rule and rule.action or ""))

    if filter_type == "allow" or action == "allow" then
        return "allow", ""
    end
    if filter_type == "log" or action == "log" then
        ngx.log(ngx.INFO, "cc log filter host=", host or "", " ip=", ip or "", " uri=", uri or "", " type=", filter_type)
        return "continue", ""
    end

    if filter_type == "url_auth" then
        local ok_auth, detail = verify_url_auth(filter, host, ip, uri)
        if ok_auth then
            return "continue", ""
        end
        local exceeded, rate_detail = rate_exceeded(filter, host, ip, uri)
        if not exceeded and tonumber(filter.max_req or 0) > 0 then
            return "continue", ""
        end
        return "fail", detail or rate_detail
    end

    if filter_type == "302" then
        local exceeded, detail = rate_exceeded(filter, host, ip, uri)
        if should_force_guard(filter) or exceeded then
            apply_redirect_302(filter, host, ip)
        end
        return "continue", ""
    end

    if guard.is_guard_filter(filter_type) then
        local non_browser, non_browser_detail = guard.is_common_non_browser_request()
        if non_browser and extra.block_non_browser ~= false then
            return "fail", non_browser_detail
        end
        local exceeded, detail = rate_exceeded(filter, host, ip, uri)
        if should_force_guard(filter) or exceeded then
            if not guard.ensure_passed(filter, host, ip) then
                ngx.header["X-Block-Source"] = build_cc_reason("cc_guard", rule_id, rule, filter, detail)
                guard.challenge(filter, host, ip)
                ngx.exit(200)
            end
        end
        return "continue", ""
    end

    if filter_type == "" or filter_type == "req_rate" or filter_type == "block" then
        if filter_type == "block" and (tonumber(filter.max_req or 0) <= 0 and tonumber(filter.max_req_per_uri or 0) <= 0) then
            return "fail", "config=block"
        end
        local exceeded, detail = rate_exceeded(filter, host, ip, uri)
        if exceeded then
            return "fail", detail
        end
        return "continue", ""
    end

    ngx.log(ngx.WARN, "cc unknown filter type=", filter_type, " id=", tostring(filter.id or 0))
    return "continue", ""
end

local function apply_filter(filter, rule, rule_id, host, ip, uri, action_override)
    local result, detail = probe_filter(filter, rule, rule_id, host, ip, uri, action_override)
    if result == "fail" then
        local action = action_override or rule and rule.action
        return enforce_action(action, rule, rule_id, host, ip, uri, filter, detail)
    end
    return result
end

local function resolve_filter(config, filter_id)
    if not config or not config.cc_filters or not filter_id or filter_id == 0 then
        return nil
    end
    return config.cc_filters[tostring(filter_id)] or config.cc_filters[filter_id]
end

local function resolve_matcher_data(config, matcher_id)
    if not config or not config.cc_matchers or not matcher_id or matcher_id == 0 then
        return nil
    end
    local matcher = config.cc_matchers[tostring(matcher_id)] or config.cc_matchers[matcher_id]
    if not matcher or not matcher.data or matcher.data == "" then
        return nil
    end
    return cjson.decode(matcher.data)
end

local function rule_enabled(rule)
    if rule == nil then
        return false
    end
    if rule.enabled == false or rule.on == false or rule.is_on == false or rule.state == false then
        return false
    end
    if rule.enabled == true or rule.on == true or rule.is_on == true or rule.state == true then
        return true
    end
    if rule.state == "off" or rule.state == "0" then
        return false
    end
    return true
end

local function build_inline_filter(rule)
    local action = string.lower(tostring(rule.action or "block"))
    local filter_type = ACTION_TO_FILTER[action] or action
    local params = rule.actionParams or rule.action_params or {}
    return {
        id = 0,
        type = filter_type,
        within_second = tonumber(params.seconds or params.within_second) or 10,
        max_req = tonumber(params.requests or params.max_req) or 0,
        max_req_per_uri = tonumber(params.urlRequests or params.max_req_per_uri) or 0,
        extra = cjson.encode({
            block_non_browser = params.blockOnFail ~= false,
            redirect_url = params.redirect_url or params.redirect,
        }),
    }, action
end

local function execute_rule(rule, host, ip, uri, config, rule_id, ctx)
    if not rule_enabled(rule) then
        return "continue"
    end

    local matcher_data
    local filter
    local action_override

    if rule.matcher_id and rule.matcher_id > 0 then
        matcher_data = resolve_matcher_data(config, rule.matcher_id)
    elseif rule.matchers or rule.matcher then
        matcher_data = { rules = rule.matchers or rule.matcher }
    end

    if not cc_matcher.match_data(matcher_data, ctx) then
        return "continue"
    end

    local filter1_id = tonumber(rule.filter_id or rule.filter1_id or 0) or 0
    local filter2_id = tonumber(rule.filter2_id or 0) or 0
    local filter1 = resolve_filter(config, filter1_id)
    local filter2 = resolve_filter(config, filter2_id)
    local filter
    local action_override

    if filter1 and filter2 then
        local result1, detail1 = probe_filter(filter1, rule, rule_id, host, ip, uri, nil)
        if result1 == "allow" then
            return finalize_allow(rule)
        end
        if result1 == "continue" then
            if rule_should_stop(rule) then
                return "stop"
            end
            return "continue"
        end
        local result2, detail2 = probe_filter(filter2, rule, rule_id, host, ip, uri, nil)
        if result2 == "allow" then
            return finalize_allow(rule)
        end
        if result2 == "continue" then
            if rule_should_stop(rule) then
                return "stop"
            end
            return "continue"
        end
        return enforce_action(rule.action, rule, rule_id, host, ip, uri, filter2, detail2 or detail1)
    end

    if filter1 then
        filter = filter1
    elseif filter2 then
        filter = filter2
    elseif rule.action then
        filter, action_override = build_inline_filter(rule)
    end

    local result = apply_filter(filter, rule, rule_id, host, ip, uri, action_override)
    if result == "allow" then
        return finalize_allow(rule)
    end
    if rule_should_stop(rule) then
        return "stop"
    end
    return "continue"
end

local function run_rules(rules, host, ip, uri, config, rule_id)
    if type(rules) ~= "table" or #rules == 0 then
        return
    end
    if ngx.ctx.cc_allowed then
        return
    end
    if guard.is_guard_request(uri) then
        return
    end

    local ctx = cc_matcher.build_context(host, ip, uri)
    cc_stats.record_response(ip, 0, ctx.ua, 60)

    for _, rule in ipairs(rules) do
        local result = execute_rule(rule, host, ip, uri, config, rule_id, ctx)
        if result == "allow" then
            ngx.ctx.cc_allowed = true
            return
        end
        if result == "stop" then
            return
        end
    end
end

local function check_rule_id(rule_id, host, ip, uri)
    if rule_id == 0 or not ip then
        return
    end
    local config = _G.cdn_config
    if not config or not config.cc_rules then
        return
    end
    local rules = config.cc_rules[tostring(rule_id)] or config.cc_rules[rule_id]
    if not rules then
        return
    end
    run_rules(rules, host, ip, uri, config, rule_id)
end

local function check_custom_rules(domain_conf, host, ip, uri)
    if not domain_conf then
        return
    end
    local rules = domain_conf.custom_cc_rules
    if type(rules) ~= "table" or #rules == 0 then
        return
    end
    run_rules(rules, host, ip, uri, _G.cdn_config, 0)
end

local function resolve_effective_rule_id(domain_conf)
    local rule_id = 0
    if domain_conf then
        rule_id = tonumber(domain_conf.cc_rule_id or 0) or 0
    end
    if rule_id == 0 then
        rule_id = tonumber(ngx.var.cc_rule_id or 0) or 0
    end
    local auto_switch = domain_conf and domain_conf.cc_auto_switch
    if type(auto_switch) ~= "table" or auto_switch.enable ~= true then
        return rule_id
    end
    local qps_limit = tonumber(auto_switch.qps) or 0
    local switch_rule = tonumber(auto_switch.rule_id) or 0
    if qps_limit <= 0 or switch_rule <= 0 or not store then
        return rule_id
    end
    local host = (domain_conf and domain_conf.name) or ngx.var.host or ""
    local key = "cc:auto_switch:" .. host
    local count = store:incr(key, 1, 0, 1)
    if count and count > qps_limit then
        return switch_rule
    end
    return rule_id
end

function _M.check(domain_conf, ip, uri)
    if not ip then
        return
    end
    local host = (domain_conf and domain_conf.name) or ngx.var.host or ""
    uri = uri or ngx.var.uri

    check_custom_rules(domain_conf, host, ip, uri)
    if ngx.ctx.cc_allowed then
        return
    end

    local rule_id = resolve_effective_rule_id(domain_conf)
    check_rule_id(rule_id, host, ip, uri)
end

function _M.check_rule_id(rule_id, host, ip, uri)
    check_rule_id(rule_id, host, ip, uri)
end

return _M
