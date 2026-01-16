local function normalize_cache_status(status)
    status = tostring(status or "")
    status = string.upper(status)
    if status == "" then
        return "MISS"
    end
    return status
end

local function build_cache_entry(role, node_id, status)
    if node_id == "" or role == "" then
        return ""
    end
    return status .. " from " .. role .. ":" .. node_id
end

local config = _G.cdn_config or {}
local node_id = tostring(config.node_id or "")
local node_level = tonumber(config.node_level) or 1
local role = node_level == 2 and "L2" or "L1"

local cache_status = normalize_cache_status(ngx.var.upstream_cache_status)
local self_entry = build_cache_entry(role, node_id, cache_status)
local l2_used = ngx.ctx.l2_used

if role == "L1" and l2_used then
    local upstream_status = ngx.var.upstream_http_x_cache_status or ""
    if upstream_status ~= "" then
        if self_entry ~= "" then
            ngx.header["X-Cache-Status"] = upstream_status .. ", " .. self_entry
        else
            ngx.header["X-Cache-Status"] = upstream_status
        end
    elseif self_entry ~= "" then
        ngx.header["X-Cache-Status"] = self_entry
    end
elseif self_entry ~= "" then
    ngx.header["X-Cache-Status"] = self_entry
end

local via = ""
if node_id ~= "" then
    via = role .. ":" .. node_id
end
local upstream_via = ngx.var.upstream_http_via or ""
if upstream_via ~= "" then
    if via ~= "" then
        via = via .. ", " .. upstream_via
    else
        via = upstream_via
    end
end
if via ~= "" then
    ngx.header["Via"] = via
end
