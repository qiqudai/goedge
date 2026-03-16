local cjson = require "cjson.safe"

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

local function normalize_raw_header(raw)
    if not raw or raw == "" then
        return ""
    end
    raw = string.gsub(raw, "\r", "")
    raw = string.gsub(raw, "\n", "\\n")
    return raw
end

local function find_default_domain(cfg)
    if not cfg or type(cfg.domains) ~= "table" then
        return nil
    end
    for _, domain in ipairs(cfg.domains) do
        if domain.default_site then
            return domain
        end
    end
    return nil
end

local function lookup_domain_conf(cfg)
    if not cfg then
        return nil
    end
    local host = ngx.var.host
    if cfg.domain_map and host and host ~= "" then
        local domain = cfg.domain_map[host]
        if domain then
            return domain
        end
    end
    if cfg.waf and cfg.waf.block_unbound_domain then
        return nil
    end
    return find_default_domain(cfg)
end

local domain_conf = lookup_domain_conf(config)
local realtime_return = true
if domain_conf and domain_conf.realtime_return == false then
    realtime_return = false
end

if realtime_return then
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
end

if domain_conf then
    if domain_conf.log_request_header then
        local current = ngx.var.cdn_req_headers or ""
        if current == "" then
            local encoded = encode_headers(ngx.req.get_headers())
            if encoded == "" then
                encoded = normalize_raw_header(ngx.req.raw_header())
            end
            ngx.var.cdn_req_headers = encoded
        end
    end
    if domain_conf.log_response_header then
        ngx.var.cdn_resp_headers = encode_headers(ngx.resp.get_headers())
    end
end
