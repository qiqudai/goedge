-- lua/ip_block.lua
local _M = {}
local block_list = ngx.shared.ip_blacklist

function _M.is_blocked(ip)
    if not block_list then return false end
    return block_list:get(ip)
end

return _M
