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
local shared_store = ngx.shared.config_store

local function parse_bool(v, default_value)
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

local function parse_number(v, default_value)
    local n = tonumber(v)
    if not n then
        return default_value
    end
    return n
end

local function resolve_revalidate_settings(cfg)
    local enabled = true
    local age_after = 5
    local probe_interval = 3
    local timeout_ms = 1200
    local cache_dir = "/data/nginx/cache"
    if type(cfg) == "table" then
        if type(cfg.nginx) == "table" and type(cfg.nginx.http) == "table" then
            local http = cfg.nginx.http
            enabled = parse_bool(http.cache_404_revalidate_enable, enabled)
            enabled = parse_bool(http.proxy_cache_404_revalidate_enable, enabled)
            age_after = parse_number(http.cache_404_revalidate_after, age_after)
            age_after = parse_number(http.proxy_cache_404_revalidate_after, age_after)
            probe_interval = parse_number(http.cache_404_probe_interval, probe_interval)
            timeout_ms = parse_number(http.cache_404_probe_timeout_ms, timeout_ms)
            if http.proxy_cache_dir and tostring(http.proxy_cache_dir) ~= "" then
                cache_dir = tostring(http.proxy_cache_dir)
            end
        end
    end
    if cache_dir:sub(-1) == "/" then
        cache_dir = cache_dir:sub(1, -2)
    end
    if age_after < 1 then
        age_after = 1
    end
    if probe_interval < 1 then
        probe_interval = 1
    end
    if timeout_ms < 200 then
        timeout_ms = 200
    end
    return {
        enabled = enabled,
        age_after = age_after,
        probe_interval = probe_interval,
        timeout_ms = timeout_ms,
        cache_dir = cache_dir,
    }
end

local function parse_backend_target(target)
    if not target or target == "" then
        return nil
    end
    local m, err = ngx.re.match(target, [[^(https?)://([^:/]+)(?::(\d+))?$]], "jo")
    if not m then
        return nil
    end
    local scheme = m[1]
    local host = m[2]
    local port = tonumber(m[3])
    if not port then
        port = scheme == "https" and 443 or 80
    end
    return {
        scheme = scheme,
        host = host,
        port = port,
    }
end

local function should_probe_once(cache_key, probe_interval)
    if not shared_store or not cache_key or cache_key == "" then
        return true
    end
    local lock_key = "reval404:" .. ngx.md5(cache_key)
    local ok = shared_store:add(lock_key, 1, probe_interval)
    return ok == true
end

local function purge_cache_entry(cache_dir, cache_key)
    if not cache_key or cache_key == "" then
        return false
    end
    local md5 = ngx.md5(cache_key)
    local path = cache_dir .. "/" .. string.sub(md5, 1, 1) .. "/" .. string.sub(md5, 2, 3) .. "/" .. md5
    local ok, err = os.remove(path)
    if ok then
        ngx.log(ngx.NOTICE, "cache revalidate: purged stale 404 cache ", path)
        return true
    end
    if err and not string.find(tostring(err), "No such file", 1, true) then
        ngx.log(ngx.WARN, "cache revalidate: failed to purge cache file ", path, ": ", err)
    end
    return false
end

local function probe_origin_and_revalidate(premature, payload)
    if premature or type(payload) ~= "table" then
        return
    end
    local target = parse_backend_target(payload.backend_target)
    if not target then
        return
    end
    local sock, err = ngx.socket.tcp()
    if not sock then
        ngx.log(ngx.WARN, "cache revalidate: tcp socket create failed: ", err)
        return
    end
    sock:settimeout(payload.timeout_ms or 1200)
    local ok, conn_err = sock:connect(target.host, target.port)
    if not ok then
        return
    end
    if target.scheme == "https" then
        local sess, ssl_err = sock:sslhandshake(false, payload.host or target.host, false)
        if not sess then
            sock:close()
            ngx.log(ngx.WARN, "cache revalidate: ssl handshake failed: ", ssl_err)
            return
        end
    end
    local request_uri = payload.request_uri or "/"
    local host = payload.host or target.host
    local req = "HEAD " .. request_uri .. " HTTP/1.1\r\nHost: " .. host .. "\r\nConnection: close\r\nUser-Agent: CDN-Revalidate/1.0\r\n\r\n"
    local bytes, send_err = sock:send(req)
    if not bytes then
        sock:close()
        return
    end
    local line, read_err = sock:receive("*l")
    sock:close()
    if not line then
        return
    end
    local status = tonumber(string.match(line, "^HTTP/%d+%.%d+ (%d%d%d)"))
    if status and status >= 200 and status < 400 then
        purge_cache_entry(payload.cache_dir or "/data/nginx/cache", payload.cache_key or "")
    end
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

-- Hide the concrete server product name. `server_tokens off` only strips
-- version details and still leaves `openresty` in the Server header.
ngx.header["Server"] = nil

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

local function maybe_revalidate_404_cache(cfg, domain)
    local settings = resolve_revalidate_settings(cfg)
    if not settings.enabled then
        return
    end
    if normalize_cache_status(ngx.var.upstream_cache_status) ~= "HIT" then
        return
    end
    if tonumber(ngx.status) ~= 404 then
        return
    end
    local age = parse_number(ngx.header["Age"], 0)
    if age < settings.age_after then
        return
    end
    local cache_key = ngx.var.cdn_cache_key or ""
    if cache_key == "" then
        return
    end
    if not should_probe_once(cache_key, settings.probe_interval) then
        return
    end
    local payload = {
        backend_target = ngx.var.backend_target or "",
        host = ngx.var.host or "",
        request_uri = ngx.var.request_uri or "/",
        cache_key = cache_key,
        cache_dir = settings.cache_dir,
        timeout_ms = settings.timeout_ms,
    }
    local ok, err = ngx.timer.at(0, probe_origin_and_revalidate, payload)
    if not ok then
        ngx.log(ngx.WARN, "cache revalidate: timer create failed: ", err)
    end
end

maybe_revalidate_404_cache(config, domain_conf)

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
