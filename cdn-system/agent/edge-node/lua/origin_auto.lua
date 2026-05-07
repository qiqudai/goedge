-- lua/origin_auto.lua
local _M = {}

local function trim(value)
    if not value then
        return ""
    end
    return tostring(value):match("^%s*(.-)%s*$") or ""
end

local function policy_enabled()
    return trim(ngx.var.origin_http_policy) == "auto" and trim(ngx.var.origin_auto_downgrade) ~= "0"
end

local function state_key(kind)
    local host = trim(ngx.var.host)
    local upstream = trim(ngx.ctx.origin_auto_upstream_key or ngx.var.backend_target)
    if host == "" or upstream == "" then
        return nil
    end
    return "origin:auto:" .. kind .. ":" .. host .. ":" .. upstream
end

function _M.before_proxy(domain_conf, backend_target)
    ngx.var.origin_compat = "0"
    ngx.var.origin_connection = ""
    if not policy_enabled() then
        return
    end
    ngx.ctx.origin_auto_upstream_key = trim((domain_conf and domain_conf.upstream_key) or backend_target)
    local dict = ngx.shared.config_store
    if not dict then
        return
    end
    local compat_key = state_key("compat_until")
    if not compat_key then
        return
    end
    local until_ts = tonumber(dict:get(compat_key)) or 0
    if until_ts > ngx.time() then
        ngx.var.origin_compat = "1"
        ngx.var.origin_connection = "close"
    elseif until_ts > 0 then
        dict:delete(compat_key)
        ngx.log(ngx.NOTICE, "[OriginAuto] restore host=", ngx.var.host or "", " upstream=", ngx.ctx.origin_auto_upstream_key or "")
    end
end

local function failed_status()
    local status = tonumber(ngx.status) or 0
    if status == 502 or status == 503 or status == 504 then
        return true
    end
    local upstream_status = trim(ngx.var.upstream_status)
    if upstream_status ~= "" and upstream_status ~= "-" then
        for code in string.gmatch(upstream_status, "([^,]+)") do
            code = trim(code)
            if code == "502" or code == "503" or code == "504" then
                return true
            end
        end
    end
    local upstream_rt = trim(ngx.var.upstream_response_time)
    if (upstream_rt == "" or upstream_rt == "-") and status >= 500 then
        return true
    end
    return false
end

function _M.after_proxy()
    if not policy_enabled() then
        return
    end
    if ngx.var.origin_compat == "1" then
        return
    end
    if not failed_status() then
        local error_key = state_key("error")
        if error_key and ngx.shared.config_store then
            ngx.shared.config_store:delete(error_key)
        end
        return
    end
    local dict = ngx.shared.config_store
    if not dict then
        return
    end
    local error_key = state_key("error")
    local compat_key = state_key("compat_until")
    if not error_key or not compat_key then
        return
    end
    local threshold = tonumber(ngx.var.origin_downgrade_threshold) or 3
    local window = tonumber(ngx.var.origin_downgrade_window) or 60
    local cooldown = tonumber(ngx.var.origin_downgrade_cooldown) or 600
    if threshold < 1 then threshold = 3 end
    if window < 1 then window = 60 end
    if cooldown < 1 then cooldown = 600 end

    local count, err = dict:incr(error_key, 1, 0, window)
    if not count then
        ngx.log(ngx.WARN, "[OriginAuto] error counter failed: ", err or "unknown")
        return
    end
    if count >= threshold then
        dict:set(compat_key, ngx.time() + cooldown, cooldown)
        dict:delete(error_key)
        ngx.log(ngx.NOTICE, "[OriginAuto] downgrade host=", ngx.var.host or "", " upstream=", ngx.ctx.origin_auto_upstream_key or "", " cooldown=", cooldown, " reason=5xx_threshold")
    end
end

return _M
