-- lua/cdnfly_wrapper.lua
local _M = {}

-- Add our local lib to package path to find the migrated 'resty.*' modules
-- We assume the base path is standard, but let's be safe.
-- In nginx.conf, we should ensure 'lua/lib/?.lua' is added.
-- For now, we dynamically append if needed (though discouraged in phase).

-- Try to load the core filter module
-- Based on the file list, 'filter_req.lua' seems to be 'resty.filter_req'
local status, filter_req = pcall(require, "resty.filter_req")

if not status then
    ngx.log(ngx.ERR, "CDNFLY: Failed to load resty.filter_req: ", filter_req)
else
    _M.filter_req = filter_req
end

function _M.run()
    if not _M.filter_req then
        return
    end

    -- The cdnfly scripts likely expect 'run' or similar method.
    -- Looking at the strings in the bytecode: "run", "apply_rule", "match_request"
    -- filter_req.lua strings: "filter_request", "get_cur_rule"
    
    -- Let's try calling entry point. 
    -- Usually these modules have a .run() or .check() method.
    -- We'll wrap in pcall to prevent crash.
    local ok, err = pcall(function()
        -- Attempt to invoke main logic
        -- Common patterns: filter_req.run(), filter_req.filter_request()
        if _M.filter_req.run then
            _M.filter_req.run()
        elseif _M.filter_req.filter_request then
            _M.filter_req.filter_request()
        else
            ngx.log(ngx.ERR, "CDNFLY: No known entry point in filter_req")
        end
    end)

    if not ok then
        ngx.log(ngx.ERR, "CDNFLY: Error execution: ", err)
    end
end

return _M
