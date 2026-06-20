-- lua/cc_stats.lua
-- Tracks per-IP counters used by CC matchers (404 count, unique UA count).
local _M = {}

local DEFAULT_WINDOW = 60

local function store()
    return ngx.shared.cc_req_rate or ngx.shared.waf_cache
end

function _M.record_response(ip, status, ua, window)
    local s = store()
    if not s or not ip or ip == "" then
        return
    end
    window = tonumber(window) or DEFAULT_WINDOW
    if window <= 0 then
        window = DEFAULT_WINDOW
    end

    if tonumber(status) == 404 then
        s:incr("cc:404:" .. ip, 1, 0, window)
    end

    ua = tostring(ua or "")
    if ua ~= "" then
        local hash = ngx.md5(ua)
        local item_key = "cc:ua:" .. ip .. ":" .. hash
        if not s:get(item_key) then
            s:set(item_key, 1, window)
            s:incr("cc:ua_cnt:" .. ip, 1, 0, window)
        end
    end
end

function _M.get_404_count(ip, window)
    local s = store()
    if not s or not ip or ip == "" then
        return 0
    end
    return tonumber(s:get("cc:404:" .. ip)) or 0
end

function _M.get_ua_unique_count(ip, window)
    local s = store()
    if not s or not ip or ip == "" then
        return 0
    end
    window = tonumber(window) or DEFAULT_WINDOW
    if window <= 0 then
        window = DEFAULT_WINDOW
    end
    -- Refresh TTL on read so active windows stay warm.
    local count = tonumber(s:get("cc:ua_cnt:" .. ip)) or 0
    if count > 0 then
        s:incr("cc:ua_cnt:" .. ip, 0, 0, window)
    end
    return count
end

return _M
