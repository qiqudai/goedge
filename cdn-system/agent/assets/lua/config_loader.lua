-- lua/config_loader.lua
local cjson = require "cjson.safe"
local _M = {}

local function resolve_prefix()
    local prefix = ngx.config.prefix() or ""
    if prefix ~= "" and prefix:sub(-1) ~= "/" then
        prefix = prefix .. "/"
    end
    return prefix
end

local function resolve_config_path()
    return resolve_prefix() .. "conf/cdn_config.json"
end

-- Config file path (relative to nginx prefix)
local CONFIG_FILE = resolve_config_path()
local L2_STATUS_FILE = resolve_prefix() .. "conf/l2_status.json"
local CHECK_INTERVAL = 1 -- seconds
local L2_STATUS_INTERVAL = 2

-- Shared dictionary to store config version/metadata if needed
-- For worker-level cache, we use a local module variable (upvalue)
local current_config = nil
local last_version = 0

local function load_l2_status()
    local f = io.open(L2_STATUS_FILE, "r")
    if not f then
        return
    end
    local content = f:read("*a")
    f:close()
    if not content or content == "" then
        return
    end
    local data = cjson.decode(content)
    if not data then
        return
    end
    local status = data.nodes or data
    if type(status) ~= "table" then
        return
    end
    _G.cdn_l2_status = status
end

-- Redis reporting removed (use API-based reporting if needed).

-- Function to load config
function _M.load_config()
    local f = io.open(CONFIG_FILE, "r")
    if not f then
        ngx.log(ngx.ERR, "Failed to open config file for reading: ", CONFIG_FILE)
        return
    end

    local content = f:read("*a")
    f:close()

    if not content then return end

    local config, err = cjson.decode(content)
    if not config then
        ngx.log(ngx.ERR, "Failed to parse config JSON: ", err)
        return
    end

    local version = tonumber(config.version) or 0
    if version ~= 0 and last_version == version then
        return
    end

    -- 1. Indexing Domains for O(1) Lookup
    -- Structure: config.domain_map[hostname] = { upstream_key = "...", ssl_id = "..." }
    local domain_map = {}
    if config.domains then
        for _, d in ipairs(config.domains) do
            domain_map[d.name] = d
        end
    end
    config.domain_map = domain_map
    
    -- 2. Indexing Upstreams
    -- Structure: config.upstream_map[cluster_id] = { {ip=..., weight=...}, ... }
    local upstream_map = {}
    if config.upstreams then
        for _, u in ipairs(config.upstreams) do
            upstream_map[u.id] = u.targets
        end
    end
    config.upstream_map = upstream_map

    -- Update Global State
    current_config = config
    last_version = version
    
    -- 3. Export to _G for access.lua access
    _G.cdn_config = current_config
    
    ngx.log(ngx.INFO, "CDN Config Reloaded. Version: ", version)
    
    -- Reporting removed; sync via API if needed.
end

-- Timer callback
local function check_config(premature)
    if premature then return end
    
    local ok, err = pcall(_M.load_config)
    if not ok then
        ngx.log(ngx.ERR, "Error loading config: ", err)
    end
    
    local ok, err = ngx.timer.at(CHECK_INTERVAL, check_config)
    if not ok then
        ngx.log(ngx.ERR, "Failed to schedule config check timer: ", err)
    end
end

local function check_l2_status(premature)
    if premature then return end

    local ok, err = pcall(load_l2_status)
    if not ok then
        ngx.log(ngx.ERR, "Error loading L2 status: ", err)
    end
    local ok, err = ngx.timer.at(L2_STATUS_INTERVAL, check_l2_status)
    if not ok then
        ngx.log(ngx.ERR, "Failed to schedule L2 status timer: ", err)
    end
end

-- Public Init Function
function _M.init()
    -- Run immediately once
    _M.load_config()
    load_l2_status()
    -- Start polling loop
    local ok, err = ngx.timer.at(CHECK_INTERVAL, check_config)
    if not ok then
        ngx.log(ngx.ERR, "Failed to start config timer: ", err)
    end
    local ok, err = ngx.timer.at(L2_STATUS_INTERVAL, check_l2_status)
    if not ok then
        ngx.log(ngx.ERR, "Failed to start L2 status timer: ", err)
    end
end

-- Getter for other modules
function _M.get_config()
    return current_config
end

return _M
