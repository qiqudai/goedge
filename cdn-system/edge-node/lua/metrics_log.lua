-- lua/metrics_log.lua
local metrics = require "lua.metrics"
local origin_auto = require "lua.origin_auto"
local cc_stats = require "lua.cc_stats"

local config = _G.cdn_config or {}
if config.waf and config.waf.block_page_traffic_free and ngx.ctx.waf_block_page then
    return
end
origin_auto.after_proxy()
cc_stats.record_response(ngx.var.remote_addr, ngx.status, ngx.var.http_user_agent, 60)
metrics.log_request(ngx.var.host, ngx.status, ngx.var.body_bytes_sent, ngx.var.upstream_response_time)
