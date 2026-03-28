-- resty/filter_req.lua
local _M = {}

local ok_ip_block, ip_block = pcall(require, "lua.ip_block")
local ok_anti_cc, anti_cc = pcall(require, "lua.anti_cc")
local ok_waf, waf = pcall(require, "lua.waf")
local ok_cc, cc = pcall(require, "lua.cc")

local function log(...)
    ngx.log(ngx.DEBUG, "filter_req: ", ...)
end

local function block(reason, status)
    status = status or 403
    ngx.log(ngx.WARN, "filter_req blocking request (", reason, ") status=", status)
    ngx.exit(status)
end

function _M.run()
    local ip = ngx.var.remote_addr

    if ok_ip_block and ip_block and ip then
        if ip_block.is_blocked(ip) then
            block("ip_block", 418)
            return
        end
    end

    if ok_anti_cc and anti_cc and ip then
        if anti_cc.check_limit(ip) then
            block("anti_cc", 503)
            return
        end
    end

    if ok_waf and waf then
        waf.check()
    end

    if ok_cc and cc and ip then
        cc.check(nil, ip, ngx.var.uri)
    end

    log("filters processed for ip=", ip)
end

function _M.filter_request()
    _M.run()
end

function _M.get_cur_rule()
    local config = _G.cdn_config or {}
    local waf_cfg = (config.waf or {})
    return {
        name = "filter_req_local",
        reason = "local logic",
        waf_mode = waf_cfg.default_block_action or "deny",
    }
end

return _M
