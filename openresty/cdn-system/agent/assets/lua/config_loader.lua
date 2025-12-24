-- lua/config_loader.lua
local cjson = require "cjson.safe"
local _M = {}

-- Config file path
local CONFIG_FILE = "/usr/local/openresty/nginx/conf/cdn_config.json"
local CHECK_INTERVAL = 1 -- seconds

-- Shared dictionary to store config version/metadata if needed
-- For worker-level cache, we use a local module variable (upvalue)
local current_config = nil
local last_version = 0

local redis_conn = require "lua.redis_conn"

-- ... (Previous code)

-- New Helper: Report Config Version to Redis
local function report_version(config, version)
    if not config then return end
    
    -- Identify the node (Prefer node_id from config, else hostname/random)
    local node_id = config.node_id or "edge-node-default"
    
    local red, err = redis_conn.get_connect()
    if not red then
        ngx.log(ngx.ERR, "Reporting: Config loaded but failed to connect to Redis: ", err)
        return
    end
    
    -- HSET cluster_config_status <node_id> <timestamp>
    local ok, err = red:hset("cluster_config_status", node_id, version)
    if not ok then
        ngx.log(ngx.ERR, "Reporting: Failed to update status in Redis: ", err)
    else
        ngx.log(ngx.INFO, "Reporting: Config version reported to Redis. Node: ", node_id, " Ver: ", version)
    end
    
    redis_conn.close(red)
end

-- Function to load config
function _M.load_config()
    local f = io.open(CONFIG_FILE, "r")
    if not f then
        ngx.log(ngx.ERR, "Failed to open config file for reading")
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
    
    -- 4. Phase 6: Sync Reporting
    -- Run in pcall to avoid crashing the loader if network fails
    local ok, err = pcall(report_version, config, version)
    if not ok then
        ngx.log(ngx.ERR, "Reporting Crashed: ", err)
    end
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

-- Public Init Function
function _M.init()
    -- Run immediately once
    _M.load_config()
    -- Start polling loop
    local ok, err = ngx.timer.at(CHECK_INTERVAL, check_config)
    if not ok then
        ngx.log(ngx.ERR, "Failed to start config timer: ", err)
    end
end

-- Getter for other modules
function _M.get_config()
    return current_config
end

return _M
