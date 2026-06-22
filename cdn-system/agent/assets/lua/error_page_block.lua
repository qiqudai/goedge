-- lua/error_page_block.lua
-- Dedicated HTTP status for IP blacklist / regional access blocks (CF-style page).
local _M = {}

_M.ACCESS_BLOCKED_STATUS = 419

function _M.exit_blocked(reason)
    if reason and reason ~= "" then
        ngx.header["X-Block-Source"] = reason
    end
    ngx.exit(_M.ACCESS_BLOCKED_STATUS)
end

function _M.is_access_block_reason(reason)
    if type(reason) ~= "string" or reason == "" then
        return false
    end
    return reason == "blacklist"
        or reason == "region"
        or reason == "ip_block"
        or reason == "ip_deny"
        or reason == "region_block"
end

return _M
