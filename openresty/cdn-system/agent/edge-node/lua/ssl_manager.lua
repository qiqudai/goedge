-- lua/ssl_manager.lua
local _M = {}
local ssl = require "ngx.ssl"

-- LRU cache for parsed certs/keys to avoid re-parsing on every handshake
-- Capacity: 1000 items (adjust based on memory)
local cert_cache = require "resty.lrucache":new(1000)
if not cert_cache then
    ngx.log(ngx.ERR, "failed to create ssl lrucache")
end

function _M.set_certificate()
    -- 1. Clear default fallback cert (optional but good practice)
    ssl.clear_certs()

    -- 2. Get SNI hostname
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
    
    local cert_pem = domain_info.ssl_cert_data
    local key_pem  = domain_info.ssl_key_data
    
    if not cert_pem or not key_pem then
        -- If data is not inline, try loading from file path helper
        -- (Ideally, the config loader should have pre-read these)
        ngx.log(ngx.ERR, "SSL: No cert data for ", server_name)
        return
    end
    
    -- 5. Parse Certificate (with Cache)
    local cached_cert = cert_cache:get(server_name .. ":cert")
    local parsed_cert
    
    if cached_cert then
        parsed_cert = cached_cert
    else
        -- Parse PEM
        local pcert, err = ssl.parse_pem_cert(cert_pem)
        if not pcert then
            ngx.log(ngx.ERR, "SSL: Failed to parse cert for ", server_name, ": ", err)
            return
        end
        parsed_cert = pcert
        cert_cache:set(server_name .. ":cert", parsed_cert, 3600) -- TTL 1h
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
    
    -- 7. Set Certificate & Key
    local ok, err = ssl.set_cert(parsed_cert)
    if not ok then
        ngx.log(ngx.ERR, "SSL: Failed to set cert: ", err)
        return
    end
    
    local ok, err = ssl.set_priv_key(parsed_key)
    if not ok then
        ngx.log(ngx.ERR, "SSL: Failed to set key: ", err)
        return
    end
end

return _M
