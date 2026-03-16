-- lua/ssl_manager.lua
local _M = {}
local ssl = require "ngx.ssl"

-- LRU cache for parsed certs/keys to avoid re-parsing on every handshake
-- Capacity: 1000 items (adjust based on memory)
local lrucache = require "resty.lrucache"
local cert_cache, cache_err = lrucache.new(1000)
if not cert_cache then
    ngx.log(ngx.ERR, "failed to create ssl lrucache: ", cache_err or "unknown error")
end

local function trim(value)
    if value == nil then
        return ""
    end
    value = tostring(value)
    value = string.gsub(value, "^%s+", "")
    value = string.gsub(value, "%s+$", "")
    return value
end

local function strip_bom(value)
    if value == nil then
        return ""
    end
    if value:sub(1, 3) == "\239\187\191" then
        return value:sub(4)
    end
    return value
end

local function normalize_pem(value)
    value = trim(value)
    if value == "" then
        return ""
    end
    value = value:gsub("\\\\n", "\n"):gsub("\\\\r", "\r")
    return strip_bom(value)
end

local function read_pem_from_path(path)
    path = trim(path)
    if path == "" then
        return ""
    end
    if cert_cache then
        local cached = cert_cache:get("file:" .. path)
        if cached ~= nil then
            return cached
        end
    end
    local file, err = io.open(path, "r")
    if not file then
        ngx.log(ngx.ERR, "SSL: Failed to open cert file: ", path, ": ", err or "")
        return ""
    end
    local data = file:read("*a") or ""
    file:close()
    data = normalize_pem(data)
    if data == "" then
        return ""
    end
    if cert_cache then
        cert_cache:set("file:" .. path, data, 3600)
    end
    return data
end

local function split_pem_certificates(pem)
    local certs = {}
    if pem == "" then
        return certs
    end
    local begin_mark = "-----BEGIN CERTIFICATE-----"
    local end_mark = "-----END CERTIFICATE-----"
    local start = 1
    while true do
        local b, e = pem:find(begin_mark, start, true)
        if not b then
            break
        end
        local eb, ee = pem:find(end_mark, e + 1, true)
        if not eb then
            break
        end
        table.insert(certs, pem:sub(b, ee))
        start = ee + 1
    end
    if #certs == 0 and pem ~= "" then
        table.insert(certs, pem)
    end
    return certs
end

local function parse_pem_chain(pem)
    local certs = split_pem_certificates(pem)
    if #certs == 0 then
        return nil, "empty cert chain"
    end
    local parsed = {}
    for idx, cert in ipairs(certs) do
        if cert:sub(-1) ~= "\n" then
            cert = cert .. "\n"
        end
        local pcert, err = ssl.parse_pem_cert(cert)
        if not pcert then
            return nil, (err or "parse failed") .. " (cert_count=" .. #certs .. ", index=" .. idx .. ", len=" .. #cert .. ")"
        end
        table.insert(parsed, pcert)
    end
    return parsed, nil
end

function _M.set_certificate()
    -- 1. Get SNI hostname
    local server_name, err = ssl.server_name()
    if not server_name then
        -- No SNI, or non-SSL handshake? 
        -- If missing SNI, Nginx usually serves default cert if configured.
        -- We exit here to let default behavior take over or drop.
        return
    end

    -- 3. Lookup Domain Config
    -- Using the global cdn_config populated by config_loader
    local config = _G.cdn_config
    if not config or not config.domain_map then
        -- Config not ready, can't verify.
        ngx.log(ngx.ERR, "SSL: config missing")
        return
    end

    local domain_info = config.domain_map[server_name]
    if not domain_info then
        -- Domain unknown? Fallback or Log
        ngx.log(ngx.WARN, "SSL: Unknown SNI domain: ", server_name)
        return
    end

    -- 4. Get Cert Data
    -- We assume the config contains either raw PEM content or path.
    -- For high perf, content should be pre-loaded or cached.
    -- Here we implement a simple Path-based loader with LRU.
    
    local cert_pem = normalize_pem(domain_info.ssl_cert_data)
    local key_pem  = normalize_pem(domain_info.ssl_key_data)

    if cert_pem == "" then
        cert_pem = read_pem_from_path(domain_info.ssl_cert_path)
    end
    if key_pem == "" then
        key_pem = read_pem_from_path(domain_info.ssl_key_path)
    end
    if cert_pem == "" or key_pem == "" then
        ngx.log(ngx.ERR, "SSL: No cert data for ", server_name)
        return
    end
    
    -- 5. Parse Certificate (with Cache)
    local cached_cert = cert_cache:get(server_name .. ":certs")
    local parsed_certs

    if cached_cert then
        parsed_certs = cached_cert
    else
        local pcerts, err = parse_pem_chain(cert_pem)
        if not pcerts then
            local head = cert_pem:sub(1, 30)
            head = head:gsub("\n", "\\n")
            local has_escape = cert_pem:find("\\n", 1, true) ~= nil
            ngx.log(ngx.ERR, "SSL: Failed to parse cert for ", server_name, ": ", err, ", len=", #cert_pem, ", escape=", has_escape and "1" or "0", ", head=", head)
            return
        end
        parsed_certs = pcerts
        cert_cache:set(server_name .. ":certs", parsed_certs, 3600) -- TTL 1h
    end
    
    -- 6. Parse Private Key (with Cache)
    local cached_key = cert_cache:get(server_name .. ":key")
    local parsed_key
    
    if cached_key then
        parsed_key = cached_key
    else
        local pkey, err = ssl.parse_pem_priv_key(key_pem)
        if not pkey then
             ngx.log(ngx.ERR, "SSL: Failed to parse key for ", server_name, ": ", err)
             return
        end
        parsed_key = pkey
        cert_cache:set(server_name .. ":key", parsed_key, 3600)
    end
    
    -- 7. Replace fallback cert with dynamic cert
    ssl.clear_certs()
    local ok, err = ssl.set_cert(parsed_certs[1])
    if not ok then
        ngx.log(ngx.ERR, "SSL: Failed to set cert: ", err)
        return
    end
    
    ok, err = ssl.set_priv_key(parsed_key)
    if not ok then
        ngx.log(ngx.ERR, "SSL: Failed to set key: ", err)
        return
    end

    for i = 2, #parsed_certs do
        local okc, errc = ssl.set_cert(parsed_certs[i])
        if not okc then
            ngx.log(ngx.ERR, "SSL: Failed to set chain cert: ", errc)
            return
        end
    end
end

return _M
