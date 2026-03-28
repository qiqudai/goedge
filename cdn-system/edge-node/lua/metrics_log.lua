-- lua/metrics_log.lua
local metrics = require "lua.metrics"

metrics.log_request(ngx.var.host, ngx.status, ngx.var.body_bytes_sent, ngx.var.upstream_response_time)
