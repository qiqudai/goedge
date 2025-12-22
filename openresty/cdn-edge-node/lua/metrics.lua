-- lua/metrics.lua
local _M = {}

local shared_dict_name = "metrics_store"
local dict = ngx.shared[shared_dict_name]

-- Use a safe logger that doesn't crash if dict is missing
local function log_err(...)
    ngx.log(ngx.ERR, "metrics: ", ...)
end

function _M.log_request(host, status, bytes_sent, upstream_time)
    if not dict then
        -- Silent fail is better than crash in production metrics
        return
    end

    -- Values from Nginx config are strings, need conversion
    local bytes = tonumber(bytes_sent) or 0
    local upstream_lat = tonumber(upstream_time) or 0
    
    -- Keys Design for atomic increments
    -- We use a simple prefix-based approach. 
    -- Cleaning/Rotation is done by external agent scraping or expire logic (TBD)
    -- For now, we assume scrape-and-reset or cumulative counter.
    -- Prometheus standard: Counters never decrease. We just increment.
    
    -- 1. QPS (Total Requests)
    local qps_key = "qps:" .. (host or "_")
    dict:incr(qps_key, 1, 0)
    
    -- 2. Traffic (Bytes)
    local bytes_key = "bytes:" .. (host or "_")
    dict:incr(bytes_key, bytes, 0)

    -- 3. Status Code Distribution
    local status_key = "status:" .. (host or "_") .. ":" .. (status or "000")
    dict:incr(status_key, 1, 0)
    
    -- 4. Upstream Latency Bucket (Simplified Histogram)
    -- <50ms, <100ms, <500ms, >500ms
    local bucket = "slow"
    if upstream_lat < 0.05 then bucket = "50ms"
    elseif upstream_lat < 0.1 then bucket = "100ms"
    elseif upstream_lat < 0.5 then bucket = "500ms"
    end
    local lat_key = "latency:" .. (host or "_") .. ":" .. bucket
    dict:incr(lat_key, 1, 0)
end

function _M.get_prometheus_data()
    if not dict then return "# Error: metrics_store dict missing\n" end
    
    local keys = dict:get_keys(0)
    local lines = {}
    
    table.insert(lines, "# HELP edge_requests_total Total requests")
    table.insert(lines, "# TYPE edge_requests_total counter")
    
    for _, key in ipairs(keys) do
        local val = dict:get(key)
        if val then
            -- QPS
            local _, _, host = string.find(key, "^qps:(.+)")
            if host then
                table.insert(lines, string.format('edge_requests_total{host="%s"} %d', host, val))
            end
            
            -- Bytes
            local _, _, b_host = string.find(key, "^bytes:(.+)")
            if b_host then
                 table.insert(lines, string.format('edge_bytes_total{host="%s"} %d', b_host, val))
            end
            
            -- Status
            local _, _, s_host, s_code = string.find(key, "^status:([^:]+):(%d+)")
            if s_host then
                 table.insert(lines, string.format('edge_response_status{host="%s",code="%s"} %d', s_host, s_code, val))
            end
            
            -- Latency
            local _, _, l_host, l_bucket = string.find(key, "^latency:([^:]+):(.+)")
            if l_host then
                 table.insert(lines, string.format('edge_upstream_latency_bucket{host="%s",le="%s"} %d', l_host, l_bucket, val))
            end
        end
    end
    
    return table.concat(lines, "\n")
end

return _M
