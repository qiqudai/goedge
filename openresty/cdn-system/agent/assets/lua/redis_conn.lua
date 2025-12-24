-- lua/redis_conn.lua
local redis = require "resty.redis"
local config_loader = require "lua.config_loader"
local _M = {}

local DEFAULT_HOST = "127.0.0.1"
local DEFAULT_PORT = 6379
local DEFAULT_TIMEOUT = 1000 -- 1 sec

function _M.get_connect()
    local config = config_loader.get_config()
    
    -- In production, redis config should be in cdn_config.json
    -- For MVP, we fallback to defaults if not found
    local redis_conf = config and config.redis or {}
    local host = redis_conf.host or DEFAULT_HOST
    local port = redis_conf.port or DEFAULT_PORT
    local password = redis_conf.password
    
    local red, err = redis:new()
    if not red then
        ngx.log(ngx.ERR, "failed to instantiate redis: ", err)
        return nil, err
    end

    red:set_timeout(DEFAULT_TIMEOUT)

    local ok, err = red:connect(host, port)
    if not ok then
        ngx.log(ngx.ERR, "failed to connect to redis: ", err)
        return nil, err
    end

    if password then
        local res, err = red:auth(password)
        if not res then
            ngx.log(ngx.ERR, "failed to authenticate to redis: ", err)
            return nil, err
        end
    end

    return red, nil
end

function _M.close(red)
    if not red then return end
    -- Put it into the connection pool
    -- max_idle_timeout 10 sec, pool_size 100
    local ok, err = red:set_keepalive(10000, 100)
    if not ok then
        ngx.log(ngx.WARN, "failed to set redis keepalive: ", err)
        -- if keepalive fails, explicitly close
        red:close() 
    end
end

return _M
