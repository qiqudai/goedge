-- Test harness for L3 parent fetch logic.
-- Keep in sync with agent/assets/lua/access_guard.lua (resolve_l3_upstream, filter_online_targets, has_online_targets).

local _M = {}

function _M.filter_online_targets(targets, status_map)
    if type(targets) ~= "table" or #targets == 0 then
        return targets
    end
    local has_status = false
    local out = {}
    for _, t in ipairs(targets) do
        local node_id = t.node_id
        if node_id and node_id ~= "" then
            local val = status_map[tostring(node_id)]
            if val ~= nil then
                has_status = true
                if val == true or val == 1 or val == "true" then
                    table.insert(out, t)
                end
            else
                table.insert(out, t)
            end
        else
            table.insert(out, t)
        end
    end
    if has_status then
        if #out > 0 then
            return out
        end
        return {}
    end
    return targets
end

function _M.has_online_targets(upstream_key, status_map, config)
    if not upstream_key or upstream_key == "" or not config or not config.upstream_map then
        return false
    end
    local targets = config.upstream_map[upstream_key]
    if type(targets) ~= "table" or #targets == 0 then
        return false
    end
    return #_M.filter_online_targets(targets, status_map) > 0
end

function _M.parse_bool_flag(v, default_value)
    if v == nil then
        return default_value
    end
    if type(v) == "boolean" then
        return v
    end
    local s = string.lower(tostring(v))
    if s == "1" or s == "true" or s == "on" or s == "yes" then
        return true
    end
    if s == "0" or s == "false" or s == "off" or s == "no" then
        return false
    end
    return default_value
end

function _M.resolve_l3_upstream(domain_conf, config)
    local mode = string.lower(tostring(domain_conf.parent_fetch_mode or "origin"))
    if mode == "" or mode == "origin" then
        return domain_conf.upstream_key, "origin"
    end
    local l1_key = domain_conf.parent_l1_upstream_key
    local l2_key = domain_conf.parent_l2_upstream_key
    local l1_status = (config.parent_status and config.parent_status.l1) or {}
    local l2_status = (config.l2_status and config.l2_status.nodes) or {}
    if mode == "l1" then
        if l1_key and l1_key ~= "" and _M.has_online_targets(l1_key, l1_status, config) then
            return l1_key, "parent"
        end
        if l2_key and l2_key ~= "" and _M.has_online_targets(l2_key, l2_status, config) then
            return l2_key, "parent"
        end
        return domain_conf.upstream_key, "origin"
    end
    if mode == "l2" then
        if l2_key and l2_key ~= "" and _M.has_online_targets(l2_key, l2_status, config) then
            return l2_key, "parent"
        end
        return domain_conf.upstream_key, "origin"
    end
    return domain_conf.upstream_key, "origin"
end

function _M.resolve_l1_use_l2(domain_conf, config, skip_l2)
    if skip_l2 or not domain_conf.use_l2 or not domain_conf.l2_upstream_key or domain_conf.l2_upstream_key == "" then
        return false
    end
    if not config or not config.upstream_map or not config.upstream_map[domain_conf.l2_upstream_key] then
        return false
    end
    local l2_status = (config.l2_status and config.l2_status.nodes) or {}
    return _M.has_online_targets(domain_conf.l2_upstream_key, l2_status, config)
end

return _M
