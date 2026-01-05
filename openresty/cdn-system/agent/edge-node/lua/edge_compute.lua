-- lua/edge_compute.lua
local _M = {}

function _M.run()
    -- Add custom headers or logic here
    ngx.req.set_header("X-Edge-Node", "OpenResty-1")
end

return _M
