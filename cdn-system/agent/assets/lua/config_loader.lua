-- lua/config_loader.lua
local cjson = require "cjson.safe"
local _M = {}

local function resolve_config_path()
    local prefix = ngx.config.prefix() or ""
    if prefix ~= "" and prefix:sub(-1) ~= "/" then
        prefix = prefix .. "/"
    end
    return prefix .. "conf/cdn_config.json"
end

-- Config file path (relative to nginx prefix)
local CONFIG_FILE = resolve_config_path()
local CHECK_INTERVAL = 1 -- seconds

-- Shared dictionary to store config version/metadata if needed
-- For worker-level cache, we use a local module variable (upvalue)
local current_config = nil
local last_version = 0

local function normalize_domain_host(input)
    local host = tostring(input or "")
    host = string.lower(host)
    host = string.gsub(host, "^%s+", "")
    host = string.gsub(host, "%s+$", "")
    host = string.gsub(host, "^https?://", "")
    host = string.gsub(host, "[/#?].*$", "")
    host = string.gsub(host, ":%d+$", "")
    host = string.gsub(host, "%.+$", "")
    return host
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

    local function resolve_site_template(site_type, defaults)
        if not defaults then
            return nil
        end
        site_type = tostring(site_type or "website")
        site_type = string.lower(site_type)
        if site_type == "api" then
            return defaults.api
        elseif site_type == "download" then
            return defaults.download
        end
        return defaults.website
    end

    local function apply_domain_defaults(cfg)
        if not cfg or type(cfg.domains) ~= "table" or not cfg.default_config then
            return
        end
        for _, d in ipairs(cfg.domains) do
            d.name = normalize_domain_host(d.name)
            local tpl = resolve_site_template(d.site_type, cfg.default_config)
            if tpl then
                if type(d.cache) ~= "table" then
                    d.cache = nil
                end
                local ttl = tonumber(tpl.cache_ttl) or 0
                if d.waf_enable == nil and tpl.waf_enable ~= nil then
                    d.waf_enable = tpl.waf_enable
                end
                if d.cache == nil then
                    if tpl.cache_enable or ttl > 0 then
                        d.cache = {
                            enable = tpl.cache_enable or false,
                            default_ttl = ttl,
                            rules = {},
                        }
                    end
                else
                    if d.cache.enable == nil and tpl.cache_enable ~= nil then
                        d.cache.enable = tpl.cache_enable
                    end
                    if (d.cache.default_ttl == nil or d.cache.default_ttl == 0) and ttl > 0 then
                        d.cache.default_ttl = ttl
                    end
                end
            end
        end
    end

    local function trim_list(list, max)
        if type(list) ~= "table" then
            return list
        end
        if max <= 0 or #list <= max then
            return list
        end
        local out = {}
        for i = 1, max do
            out[i] = list[i]
        end
        return out
    end

    local function apply_resource_limits(cfg)
        if not cfg or type(cfg.domains) ~= "table" then
            return
        end
        local resources = cfg.resources
        if type(resources) ~= "table" or type(resources.website) ~= "table" then
            return
        end
        local max_black = tonumber(resources.website.max_blacklist_ips) or 0
        local max_white = tonumber(resources.website.max_whitelist_ips) or 0
        local max_acl = tonumber(resources.website.max_acl_rules) or 0
        if max_black <= 0 and max_white <= 0 and max_acl <= 0 then
            return
        end
        for _, d in ipairs(cfg.domains) do
            if max_black > 0 then
                d.black_ips = trim_list(d.black_ips, max_black)
            end
            if max_white > 0 then
                d.white_ips = trim_list(d.white_ips, max_white)
            end
            if max_acl > 0 then
                d.acl_rules = trim_list(d.acl_rules, max_acl)
            end
        end
    end

    local function map_cc_action(action)
        action = string.lower(tostring(action or ""))
        if action == "slide" or action == "slide_simple" or action == "click" or action == "rotate" then
            return action
        end
        if action == "5s" or action == "5s_shield" or action == "js_challenge" then
            return "5s"
        end
        if action == "captcha" then
            return "slide"
        end
        return ""
    end

    local function normalize_waf_legacy(cfg)
        if not cfg or type(cfg.waf) ~= "table" then
            return
        end
        local waf = cfg.waf
        local mode = string.lower(tostring(waf.mode or ""))
        if (not waf.default_block_action or waf.default_block_action == "") and mode ~= "" then
            waf.default_block_action = mode
        end

        local policy = string.lower(tostring(waf.policy or ""))
        if policy == "strict" then
            if not waf.default_page_protection or waf.default_page_protection == "" then
                waf.default_page_protection = "force"
            end
        elseif policy == "loose" then
            if not waf.default_page_protection or waf.default_page_protection == "" then
                waf.default_page_protection = "auto"
            end
        end

        local cc = waf.cc
        if type(cc) ~= "table" then
            return
        end
        if cc.enable == true then
            if not waf.default_page_protection or waf.default_page_protection == "" then
                waf.default_page_protection = "force"
            end
        end
        if cc.threshold and tonumber(cc.threshold) and (not waf.default_page_protection_threshold or tonumber(waf.default_page_protection_threshold) == 0) then
            waf.default_page_protection_threshold = tonumber(cc.threshold)
        end
        if cc.action and (not waf.anti_cc_type or waf.anti_cc_type == "") then
            local mapped = map_cc_action(cc.action)
            if mapped ~= "" then
                waf.anti_cc_type = mapped
            end
        end
        if cc.block_timeout and tonumber(cc.block_timeout) and (not waf.blacklist_timeout or tonumber(waf.blacklist_timeout) == 0) then
            waf.blacklist_timeout = tonumber(cc.block_timeout)
        end
        if cc.emergency_mode == true then
            if not waf.default_page_protection or waf.default_page_protection == "" then
                waf.default_page_protection = "force"
            end
        end
    end

    local version = tonumber(config.version) or 0
    if version ~= 0 and last_version == version then
        return
    end

    apply_domain_defaults(config)
    apply_resource_limits(config)
    normalize_waf_legacy(config)

    -- 1. Indexing Domains for O(1) Lookup
    -- Structure: config.domain_map[hostname] = { upstream_key = "...", ssl_id = "..." }
    local domain_map = {}
    if config.domains then
        for _, d in ipairs(config.domains) do
            local host = normalize_domain_host(d.name)
            if host ~= "" then
                d.name = host
                domain_map[host] = d
            end
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
