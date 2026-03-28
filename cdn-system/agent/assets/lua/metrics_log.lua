-- lua/metrics_log.lua
local metrics = require "lua.metrics"

local config = _G.cdn_config or {}
if config.waf and config.waf.block_page_traffic_free and ngx.ctx.waf_block_page then
    return
end
metrics.log_request(ngx.var.host, ngx.status, ngx.var.body_bytes_sent, ngx.var.upstream_response_time)
