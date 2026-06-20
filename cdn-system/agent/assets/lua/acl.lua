-- lua/acl.lua
-- Site ACL engine: ordered rules with AND conditions and default action fallback.
local cc_matcher = require "lua.cc_matcher"

local _M = {}

local function build_reason(rule_index, action, extra)
    local parts = {
        "type=acl",
        "module=lua.acl",
        "rule=acl",
        "rule_id=" .. tostring(rule_index or 0),
        "action=" .. tostring(action or "unknown"),
    }
    if extra and extra ~= "" then
        table.insert(parts, tostring(extra))
    end
    return table.concat(parts, ";")
end

local function deny_request(status, redirect_url, rule_index, extra)
    status = tonumber(status) or 403
    redirect_url = tostring(redirect_url or "")
    if status == 302 and redirect_url ~= "" then
        ngx.header["X-Block-Source"] = build_reason(rule_index, "deny", extra or "redirect")
        return ngx.redirect(redirect_url, 302)
    end
    ngx.header["X-Block-Source"] = build_reason(rule_index, "deny", extra or ("status=" .. tostring(status)))
    ngx.exit(status)
end

local function normalize_action(action)
    action = string.lower(tostring(action or ""))
    if action == "reject" then
        return "deny"
    end
    return action
end

local function build_matcher_data(rule)
    if type(rule.conditions) ~= "table" or #rule.conditions == 0 then
        if rule.ip and rule.ip ~= "" then
            return {
                rules = {{
                    key = "ip",
                    operator = "eq",
                    value = rule.ip,
                }},
            }
        end
        return { rules = {{ key = "all" }} }
    end

    local matchers = {}
    for _, cond in ipairs(rule.conditions) do
        if type(cond) == "table" then
            table.insert(matchers, {
                key = cond.item,
                operator = cond.operator,
                value = cond.value,
            })
        end
    end
    if #matchers == 0 then
        return nil
    end
    return { rules = matchers }
end

local function match_rule(rule, ctx)
    local matcher_data = build_matcher_data(rule)
    if not matcher_data then
        return false
    end
    return cc_matcher.match_data(matcher_data, ctx)
end

function _M.check(domain_conf, ip, uri)
    if not domain_conf then
        return
    end
    local rules = domain_conf.acl_rules
    if type(rules) ~= "table" or #rules == 0 then
        local default_action = normalize_action(domain_conf.acl_default_action)
        if default_action == "deny" then
            deny_request(
                domain_conf.acl_default_deny_status,
                domain_conf.acl_default_redirect_url,
                0,
                "condition=acl_default_action=deny"
            )
        end
        return
    end

    local host = domain_conf.name or ngx.var.host or ""
    uri = uri or ngx.var.uri or ""
    local ctx = cc_matcher.build_context(host, ip, uri)

    for index, rule in ipairs(rules) do
        if match_rule(rule, ctx) then
            local action = normalize_action(rule.action)
            if action == "allow" then
                return
            end
            if action == "deny" then
                deny_request(rule.deny_status, rule.redirect_url, index, "condition=acl_rules")
            end
            return
        end
    end

    local default_action = normalize_action(domain_conf.acl_default_action)
    if default_action == "deny" then
        deny_request(
            domain_conf.acl_default_deny_status,
            domain_conf.acl_default_redirect_url,
            0,
            "condition=acl_default_action=deny"
        )
    end
end

return _M
