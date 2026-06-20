# Origin Keepalive Auto and First Visit Slow Diagnosis Implementation

本文档面向 AI 编码执行者。目标是无歧义地实现两个商业 CDN 能力：

1. 回源协议版本 `auto`：默认使用 HTTP/1.1 + keepalive 加速回源；发现源站不兼容时自动降级到 HTTP/1.0 兼容模式；冷却后自动探测恢复。
2. 首次访问慢诊断：访问日志能展示回源建连耗时、源站首包/响应耗时、缓存状态，并给出“慢在哪里”的诊断结论。

不要把本文档当建议书。执行时必须按本文档逐项完成、测试、回收。

## Terminology

- **边缘总耗时**：Nginx `$request_time`，从收到客户端请求到响应完成。
- **回源建连耗时**：Nginx `$upstream_connect_time`。包含到源站建立连接的耗时；HTTPS 回源时包含 TLS handshake。Nginx 无法在普通 `proxy_pass` 变量中单独暴露 DNS 解析耗时。
- **源站首包耗时**：Nginx `$upstream_header_time`，从开始回源到收到源站响应头。
- **源站响应耗时**：Nginx `$upstream_response_time`，从开始回源到源站响应完成。
- **缓存状态**：Nginx `$upstream_cache_status`，常见值 `HIT/MISS/BYPASS/EXPIRED/STALE/UPDATING/REVALIDATED`。
- **DNS/建连耗时**：商业后台展示名。第一阶段用 `$upstream_connect_time` 表示“DNS + TCP + TLS 建连综合耗时”。如需单独 DNS 耗时，见“可选增强：精确 DNS 耗时”。

## Part 1: Origin Protocol Auto

### Product Behavior

后台回源协议版本新增三档：

- `auto`：默认，推荐。正常使用 HTTP/1.1 + keepalive。连续异常达到阈值后自动降级兼容模式。冷却期后自动探测恢复。
- `http11`：强制 HTTP/1.1 + keepalive。不自动降级。
- `compat`：强制 HTTP/1.0/关闭 keepalive。用于老旧源站。

默认值必须是 `auto`。

WebSocket、实时返回/实时鉴别、显式关闭 buffering 的业务，不参与自动降级，保留现有逻辑。

### Data Model

在站点设置 `settings.advanced` 中新增：

```json
{
  "origin_http_version_policy": "auto",
  "origin_auto_downgrade": true,
  "origin_downgrade_threshold": 3,
  "origin_downgrade_window_seconds": 60,
  "origin_downgrade_cooldown_seconds": 600,
  "origin_keepalive_conn": 64,
  "origin_keepalive_timeout": 60
}
```

兼容旧字段：

- 旧 `proxy_http_version=1.1` 映射为 `http11`
- 旧 `ups_keepalive=true` 且无新字段时映射为 `auto`
- 无任何字段时映射为 `auto`

### API Config Payload

修改 `api/models/config_models.go` 的 `EdgeDomain`：

```go
OriginHTTPVersionPolicy string `json:"origin_http_version_policy,omitempty"`
OriginAutoDowngrade bool `json:"origin_auto_downgrade,omitempty"`
OriginDowngradeThreshold int `json:"origin_downgrade_threshold,omitempty"`
OriginDowngradeWindowSeconds int `json:"origin_downgrade_window_seconds,omitempty"`
OriginDowngradeCooldownSeconds int `json:"origin_downgrade_cooldown_seconds,omitempty"`
```

修改 `api/services/config_service.go`：

1. 在 `advancedConfig` 中加入上述字段。
2. 在 `extractAdvancedConfig` 中解析 `settings.advanced`。
3. 默认：
   - policy: `auto`
   - auto downgrade: `true`
   - threshold: `3`
   - window: `60`
   - cooldown: `600`
   - keepalive conn: `64`
   - keepalive timeout: `60`
4. 下发到 `models.EdgeDomain`。

### Agent Config Model

修改 `agent/config.go` 的 `edgeDomain`，新增同名 JSON 字段。

### Nginx Generation

修改 `agent/http_config.go`。

#### Upstream keepalive

当前只有 `domain.UpstreamKeepalive` 才写：

```nginx
keepalive 32;
keepalive_timeout 30s;
```

调整为：

- `policy=auto`：写 upstream keepalive。
- `policy=http11`：写 upstream keepalive。
- `policy=compat`：不写 upstream keepalive。
- `WebSocket=true`：按现有 WebSocket 逻辑，不受 auto 降级控制。

默认 keepalive：

```nginx
keepalive 64;
keepalive_timeout 60s;
```

#### Location proxy_http_version

`writeProxyProtocol` 改成：

```text
if websocket:
  proxy_http_version 1.1
  Upgrade/Connection 保持现有逻辑
else if policy == compat:
  proxy_http_version 1.0
  proxy_set_header Connection close
else if policy == http11:
  proxy_http_version 1.1
  proxy_set_header Connection ""
else if policy == auto:
  由 Lua 变量 $origin_http_version 和 $origin_connection 控制
```

`auto` 生成：

```nginx
set $origin_http_version "1.1";
set $origin_connection "";
access_by_lua_file lua/access_guard.lua;
proxy_http_version $origin_http_version;
proxy_set_header Connection $origin_connection;
```

注意：确认当前 Nginx 支持 `proxy_http_version` 使用变量。如果不支持变量，改用两个 named location：

```nginx
location / {
  access_by_lua_file lua/access_guard.lua;
  error_page 418 = @origin_compat;
  if ($origin_compat = 1) { return 418; }
  proxy_http_version 1.1;
  proxy_set_header Connection "";
  proxy_pass $backend_target;
}

location @origin_compat {
  proxy_http_version 1.0;
  proxy_set_header Connection close;
  proxy_pass $backend_target;
}
```

推荐先用 named location 方案，兼容性更高。

### Auto Downgrade State

在 `agent/edge-node/lua/access.lua` 或新增 `agent/edge-node/lua/origin_compat.lua` 中实现。

使用 `ngx.shared.config_store` 存状态，不引入外部 DB：

Key 设计：

```text
origin:auto:error:{host}:{upstream_key}
origin:auto:compat_until:{host}:{upstream_key}
origin:auto:last_probe:{host}:{upstream_key}
```

请求前逻辑：

```lua
if policy != "auto" then return end

compat_until = dict:get(compat_key)
if compat_until and compat_until > ngx.time() then
  ngx.var.origin_compat = "1"
else
  ngx.var.origin_compat = "0"
end
```

请求后逻辑必须放在 `log_by_lua` 或现有 `metrics_log.lua` 中，因为这时有 `$status`、`$upstream_status`、`$upstream_response_time`。

失败判定：

- HTTP 状态为 `502/503/504`
- `$upstream_status` 包含 `502/503/504`
- `$upstream_response_time` 为空或 `-` 且状态为 5xx
- error log 中的 connection reset/invalid header 无法可靠从 log phase 获得，第一版不依赖 error log。

计数规则：

```text
threshold = 3
window = 60s
cooldown = 600s

同一 host + upstream_key 在 window 内失败 >= threshold:
  compat_until = now + cooldown
  error_count = 0
```

恢复规则：

- 冷却时间到期后自动恢复 HTTP/1.1。
- 恢复后如果继续失败，再进入降级。
- 不要做同步阻塞探测，避免拖慢用户请求。

可选：后台 timer 每 60 秒对 compat 状态做 HEAD 探测。第一版可以不做，只依赖冷却后真实请求恢复。

### Agent Logs

降级时写 notice 日志：

```text
[OriginAuto] downgrade host=example.com upstream=upstream_1 cooldown=600 reason=5xx_threshold
```

恢复时写：

```text
[OriginAuto] restore host=example.com upstream=upstream_1
```

### Frontend UI

修改 `web/admin/src/components/manage/OriginConfig.vue`：

新增“回源 HTTP 版本”：

- 自动加速（推荐）
- HTTP/1.1 keepalive
- HTTP/1.0 兼容

新增高级项：

- 自动降级开关
- 降级阈值
- 统计窗口
- 冷却时间

修改 `web/admin/src/composables/useSiteSettings.js`：

增加状态字段。

修改 `web/admin/src/utils/configTransform.js`：

写入 `settings.advanced.origin_http_version_policy` 等字段。

批量设置 `BatchSettingsDialog.vue` 也必须支持这三个档位，避免批量站点无法配置。

### Tests

#### Agent unit tests

新增/修改：

- `agent/http_config_headers_test.go`
- 新增 `agent/origin_auto_test.go`

必须覆盖：

1. `auto` 生成 HTTP/1.1 keepalive upstream。
2. `http11` 强制 HTTP/1.1，不生成 auto 降级变量。
3. `compat` 生成 HTTP/1.0 + `Connection close`，不生成 upstream keepalive。
4. WebSocket 保持现有 Upgrade/Connection 行为。
5. auto 失败计数达到阈值后进入 compat。
6. cooldown 过期后恢复 auto。

#### API tests

新增/修改 `api/services/config_service_test.go`：

1. 无配置默认 `auto`。
2. 旧字段 `ups_keepalive=true` 映射到 `auto`。
3. 显式 `compat/http11` 原样下发。

#### Frontend build

必须通过：

```bash
cd web/admin && npm run build
```

## Part 2: First Visit Slow Diagnosis

### Goal

访问日志后台能回答：

- 是缓存没命中导致首次回源？
- 是节点到源站建连慢？
- 是源站首包慢？
- 是源站整体响应慢？
- 是 HTTPS 客户端握手/链路慢？
- 是节点规则/WAF/日志等边缘处理慢？

### Nginx Log Fields

修改：

- `agent/assets/conf/nginx.conf`
- `agent/edge-node/conf/nginx.conf`

在 JSON log 中加入：

```nginx
'"upstream_connect_time": "$upstream_connect_time",'
'"upstream_header_time": "$upstream_header_time",'
'"upstream_response_time": "$upstream_response_time",'
'"upstream_cache_status": "$upstream_cache_status",'
'"request_time": $request_time,'
'"ssl_protocol": "$ssl_protocol",'
'"ssl_cipher": "$ssl_cipher"'
```

已有字段不要删除。新增字段必须放在合法 JSON 位置，注意逗号。

### ClickHouse Schema

修改 `api/db/clickhouse.go`。

`CREATE TABLE node_access_logs` 增加：

```sql
upstream_connect_time Float64,
upstream_header_time Float64,
slow_reason String,
slow_advice String,
```

同时增加兼容旧表的 ALTER：

```sql
ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS upstream_connect_time Float64 AFTER upstream_addr;
ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS upstream_header_time Float64 AFTER upstream_connect_time;
ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS slow_reason String AFTER upstream_cache_status;
ALTER TABLE node_access_logs ADD COLUMN IF NOT EXISTS slow_advice String AFTER slow_reason;
```

### Log Ingestion

修改 `api/services/ck_service.go`。

`rawAccessLog` 新增：

```go
UpstreamConnectTime string `json:"upstream_connect_time"`
UpstreamHeaderTime string `json:"upstream_header_time"`
```

插入前解析：

```go
connectTime := parseFloatFirst(raw.UpstreamConnectTime)
headerTime := parseFloatFirst(raw.UpstreamHeaderTime)
responseTime := parseFloatFirst(raw.UpstreamResponseTime)
cacheStatus := normalizeCacheStatus(raw.UpstreamCacheStatus)
reason, advice := DiagnoseAccessLogSlowReason(DiagnoseInput{
    RequestTime: raw.RequestTime,
    UpstreamConnectTime: connectTime,
    UpstreamHeaderTime: headerTime,
    UpstreamResponseTime: responseTime,
    UpstreamCacheStatus: cacheStatus,
    Status: raw.Status,
    Scheme: raw.Scheme,
    SSLProtocol: raw.SSLProtocol,
})
```

HTTP ClickHouse insert map 和 native insert SQL 都必须包含新字段。

### Diagnosis Function

新增 `api/services/access_slow_diagnosis.go`：

```go
package services

type DiagnoseInput struct {
    RequestTime float64
    UpstreamConnectTime float64
    UpstreamHeaderTime float64
    UpstreamResponseTime float64
    UpstreamCacheStatus string
    Status int
    Scheme string
    SSLProtocol string
}

func DiagnoseAccessLogSlowReason(in DiagnoseInput) (reason string, advice string) {
    // implementation
}
```

规则顺序必须如下：

1. `request_time < 1` 且 `upstream_response_time < 1` 且 `upstream_connect_time < 0.3`
   - cache `HIT`: `正常命中`
   - other: `正常`
2. cache `MISS/EXPIRED/BYPASS` 且 `upstream_response_time >= 1`
   - reason: `缓存未命中回源慢`
   - advice: `首次访问或缓存过期正在回源；建议开启预热、延长缓存 TTL，或优化源站响应`
3. cache `MISS/EXPIRED/BYPASS`
   - reason: `缓存未命中`
   - advice: `请求需要回源；热门 URL 可使用预热降低首次访问等待`
4. cache `UPDATING`
   - reason: `后台更新中`
   - advice: `边缘正在后台刷新缓存；如频繁出现可检查 TTL 是否过短或源站波动`
5. cache `STALE`
   - reason: `使用过期缓存兜底`
   - advice: `源站可能超时或返回异常，边缘已使用 stale 缓存保护用户访问`
6. `upstream_connect_time >= 0.5`
   - reason: `回源建连慢`
   - advice: `节点到源站 TCP/TLS 建连耗时较高；建议开启回源长连接、检查源站网络或跨境链路`
7. `upstream_header_time >= 1`
   - reason: `源站首包慢`
   - advice: `源站处理或数据库耗时较高；建议检查源站首包时间、后端接口和数据库`
8. `upstream_response_time >= 1`
   - reason: `源站响应慢`
   - advice: `源站传输或大文件响应慢；建议检查源站带宽、对象大小和缓存策略`
9. `scheme=https` 且 `request_time - upstream_response_time >= 0.5`
   - reason: `客户端链路或 TLS 握手慢`
   - advice: `总耗时明显高于回源耗时；建议检查客户端网络、TLS 会话复用和证书链`
10. `status >= 500`
    - reason: `源站或节点错误`
    - advice: `5xx 可能来自源站错误、回源失败或节点配置异常；建议结合错误日志排查`
11. default
    - reason: `边缘处理慢`
    - advice: `总耗时偏高但回源耗时不高；建议检查 WAF/规则、日志采集和节点负载`

阈值第一版写常量，后续可做全局配置。

### API List Response

修改 `api/controllers/log_controller.go`。

`AccessLogRow` 新增：

```go
UpstreamConnectTime float64 `json:"upstream_connect_time"`
UpstreamHeaderTime float64 `json:"upstream_header_time"`
SlowReason string `json:"slow_reason"`
SlowAdvice string `json:"slow_advice"`
```

SELECT 增加：

```sql
upstream_connect_time,
upstream_header_time,
slow_reason,
slow_advice
```

兼容旧数据：

- 如果表中已有新列，直接 SELECT。
- 如果可能存在旧 ClickHouse 节点尚未 ALTER，启动时已经 `ADD COLUMN IF NOT EXISTS`，所以无需运行时降级。

用户端权限：

- 普通用户仍隐藏 `upstream_addr` 和 `node_ip`。
- 但可以显示耗时、缓存状态、慢因和建议。

### Frontend UI

修改 `web/admin/src/views/website/AccessLogs.vue`。

新增列：

```vue
<el-table-column prop="upstream_connect_time" label="建连耗时" width="90" />
<el-table-column prop="upstream_header_time" label="首包耗时" width="90" />
<el-table-column prop="upstream_response_time" label="源站耗时" width="90" />
<el-table-column label="慢因诊断" min-width="160">
  <template #default="{ row }">
    <el-tooltip :content="row.slow_advice || '-'" placement="top">
      <el-tag :type="slowReasonTag(row.slow_reason)" size="small">
        {{ row.slow_reason || '-' }}
      </el-tag>
    </el-tooltip>
  </template>
</el-table-column>
```

缓存状态列用 tag：

- `HIT`: success
- `MISS/EXPIRED/BYPASS`: warning
- `STALE/UPDATING`: info
- empty: default

新增高级筛选：

- 慢因类型 `slow_reason`
- 最小总耗时 `min_request_time`
- 最小源站耗时 `min_upstream_response_time`
- 最小建连耗时 `min_upstream_connect_time`

API 也要支持这些 query 参数。

### Optional Enhancement: Exact DNS Timing

Nginx 原生变量不能稳定单独给出 DNS 解析耗时，`$upstream_connect_time` 更接近“DNS + TCP + TLS 建连综合耗时”。

如果必须精确 DNS：

1. 在 OpenResty Lua 中用 `resty.dns.resolver` 解析源站域名。
2. 记录 `ngx.now()` 差值到 `ngx.ctx.origin_dns_time`。
3. 将解析结果 IP 写入 `backend_target`。
4. log format 增加变量需要通过 `set $origin_dns_time ""` 并在 Lua 中 `ngx.var.origin_dns_time = value`。

第一版不建议实现精确 DNS，因为当前回源选择和 Nginx upstream 机制已经稳定，强行改 Lua DNS 解析会增加兼容风险。

### Tests

#### API tests

新增 `api/services/access_slow_diagnosis_test.go`：

必须覆盖：

1. HIT 快请求 -> `正常命中`
2. MISS + upstream_response_time 2s -> `缓存未命中回源慢`
3. connect_time 0.8s -> `回源建连慢`
4. header_time 1.5s -> `源站首包慢`
5. response_time 2s -> `源站响应慢`
6. https request_time 比 upstream_response_time 大 0.8s -> `客户端链路或 TLS 握手慢`
7. status 502 -> `源站或节点错误`

修改/新增 ClickHouse schema test：

- 确认 `clickHouseTableStmts()` 包含四个新字段和 `ADD COLUMN IF NOT EXISTS`。

#### Agent tests

新增 `agent/nginx_log_format_test.go`：

- 生成或读取 `agent/assets/conf/nginx.conf`，确认包含：
  - `upstream_connect_time`
  - `upstream_header_time`
  - `upstream_response_time`
  - `upstream_cache_status`

#### Frontend build

必须通过：

```bash
cd web/admin && npm run build
```

### Deployment Acceptance

部署后必须验证：

```bash
cd api && go test ./...
cd agent && go test ./...
cd web/admin && npm run build
git diff --check
```

线上验证：

1. API 重启后日志无 ClickHouse schema error。
2. 节点 agent 重启后 `nginx -t` 成功。
3. 访问任意站点 3 次：
   - 第一次通常 `MISS`
   - 后续可出现 `HIT` 或业务配置允许的缓存状态
4. 后台访问日志能看到：
   - 建连耗时
   - 首包耗时
   - 源站耗时
   - 缓存状态
   - 慢因诊断
5. 使用一个故意慢源站验证 `源站首包慢/源站响应慢` 能被识别。

### Rollback

如果上线后日志写入失败：

1. 回滚 API 二进制。
2. 回滚 agent 二进制和 nginx.conf 模板。
3. ClickHouse 新增列无需删除；新增列兼容旧代码。

如果 auto 回源导致源站异常：

1. 将全局默认 policy 临时改为 `compat`。
2. 批量站点设置 `origin_http_version_policy=compat`。
3. 保留诊断日志，用于定位不兼容源站。

### Final Definition of Done

全部满足才算完成：

- 默认新站点使用 `origin_http_version_policy=auto`。
- auto 模式正常生成 HTTP/1.1 keepalive。
- auto 模式连续失败可降级 compat，并在冷却后恢复。
- compat 模式不使用 keepalive。
- WebSocket 不被 auto 降级破坏。
- 访问日志包含建连、首包、源站响应、缓存状态。
- 后台访问日志展示慢因诊断和建议。
- 所有新增字段兼容旧 ClickHouse 表。
- API、agent、前端构建测试全部通过。
