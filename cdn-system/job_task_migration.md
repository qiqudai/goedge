# Job Removal + Task JSON Dispatch Plan

## 1) Job Usage Inventory (cdn-system/api)

- `cdn-system/api/main.go`: AutoMigrate includes `models.Job`.
- `cdn-system/api/models/job.go`: Job table model definition.
- `cdn-system/api/services/cleanup_worker.go`: clears `job` table by days.
- `cdn-system/api/services/user_package_service.go`: creates `job` rows for package sync.
- `cdn-system/api/controllers/agent_ws_controller.go`:
  - `TaskDispatchMsg`, `TaskAckMsg` structs (task payload only).
  - `handleJobAck` updates `job` table state.
  - `dispatchPendingJobsForNode` loads `job` rows by node.
  - `DispatchJobToNode` dispatches job payload.
  - WS message types: `task_dispatch`, `task_ack`.
- `cdn-system/api/services/expiration_worker.go`: comment referencing Job.
- `cdn-system/api/scripts/smoke_ws.ps1`: prints job state in WS test.

## 2) Task JSON (Replace Job Table)

### 2.1 Implemented schema

- `targets_json` (longtext) added to `task` table.
- Counters (`total/success/fail/pending`) stored inside `targets_json`.
- `task.progress` retained for existing batch workflows.
```
{
  "nodes": {
    "18": {"state":"waiting","retry_at":0,"tries":0,"ret":""},
    "22": {"state":"success","retry_at":0,"tries":1,"ret":""}
  },
  "total":2,
  "success":1,
  "fail":0,
  "pending":1
}
```

### 2.3 Retry policy (per node)
- On failure: `retry_at = now + 10~30s` (jitter).
- Max retries: 3 (tries >= 3 → mark node `failed_final`).
- Task completes when all nodes are `success` or `failed_final`.
- `task.ret` records last error; node-level errors stored in JSON.

## 3) Worker Pool (Implemented)

- Global dispatch queue with 10 worker goroutines.
- Per-task in-process lock to serialize `targets_json` updates and avoid lost updates.

## 4) WS Dispatch + ACK Changes

### 4.1 WS message types
- Use `task_dispatch/task_ack`.
- Task payload only; remove `job_*` fields from WS messages.

### 4.2 ACK handling
- Replace `handleTaskAck` updates:
  - Update task JSON node state (success/fail).
  - Increment counters.
  - If fail: set node retry time (10~30s); tries++.
  - If tries >= 3: mark `failed_final`.

## 5) Code Changes (Completed)

### 5.1 Database
- Remove `job` table (or leave empty, no writes).
- Add task columns (or reuse `task.progress` JSON).

### 5.2 API code updates
- Removed `models.Job` and AutoMigrate entry; drop `job` table on startup.
- `cleanup_worker.go`: removed `job` cleanup.
- `user_package_service.go`: stop creating `job` rows; populate `targets_json` and task payload.
- `agent_ws_controller.go`:
  - Task JSON dispatch + per-node retries.
  - ACK updates `targets_json` with 10–30s retry jitter and max 3 retries.
  - Dispatch worker pool + per-task lock.
  - WS payload uses task fields only (no `job_*` fields).
- `scripts/smoke_ws.ps1`: update output to task state.

### 5.3 Task JSON helper methods
- `loadTaskTargets()`, `saveTaskTargets()`
- `markNodeSuccess()`, `markNodeFail()`
- `taskShouldRetry()`, `taskIsDone()`

### 5.4 Retry logic (requirements)
- Failed node remains `waiting` but with `retry_at`.
- `retry_at` jitter 10–30 seconds.
- Max retries = 3; then node = `failed_final`.
- Task state → `success` if all success; else `fail` if any failed_final.

## 6) Validation Plan (Playwright + Curl)

1. Trigger task creation (e.g., config sync, package sync).
2. Verify task JSON includes all nodes.
3. Simulate node failure (disconnect/invalid node).
4. Confirm retries (10–30s jitter), max 3 attempts, final state recorded.
5. Confirm agent receives dispatch and ACK updates task state.
6. Playwright: confirm UI reflects task completion.

## 7) Decisions (Resolved)

- Use `targets_json` (keep `task.progress` for existing flows).
- Switch to `task_dispatch/task_ack` only; remove `job_*` fields.
- Store aggregated per-node logs in `task.ret` via `appendTaskLog`.
- Drop `job` table on startup.

## 8) Execution Order

1. Confirm schema strategy (new columns vs reuse `task.progress`).
2. Implement task JSON helpers + worker pool.
3. Replace job dispatch/ack logic.
4. Update cleanup worker + AutoMigrate.
5. Update scripts/tests.
6. Rebuild + restart API/agent; run Playwright validation.

## 9) Progress

- [x] Module: schema/docs cleanup (drop job table + remove keep-job-days references + update dispatch wording).
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go vet ./...` ✅
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go test ./...` ✅
- [x] Module: admin 清理配置映射 + e2e 保护门禁 (ACME/smoke).
  - Tests: `npm run build` ✅
  - Tests: `node scripts/sync-wwwroot.cjs` ✅
  - Tests: `TMPDIR=/www/server/go_project/openresty/.tmp npm run test:e2e` ✅ (3 skipped: admin/user smoke + ACME unless env enabled)
- [x] Module: WS task dispatch payload + agent logging compatibility.
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go vet ./...` ✅ (cdn-system/api)
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go test ./...` ✅ (cdn-system/api)
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go vet ./...` ✅ (cdn-system/agent)
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go test ./...` ✅ (cdn-system/agent)
- [x] Module: API WS payload移除 job 字段.
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go vet ./...` ✅ (cdn-system/api)
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go test ./...` ✅ (cdn-system/api)
- [x] Module: agent WS payload移除 job 字段.
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go vet ./...` ✅ (cdn-system/agent)
  - Tests: `GOCACHE=/www/server/go_project/openresty/.cache/go-build GOMODCACHE=/www/server/go_project/openresty/.cache/go-mod go test ./...` ✅ (cdn-system/agent)
- [x] Module: WS 下发/ACK 快速验证（smoke）。
  - Tests: `FORCE=1 go run ./cmd/init_admin admin 123456` ✅ (cdn-system/api)
  - Tests: `curl + python (等价执行 cdn-system/api/scripts/smoke_ws.ps1)` ✅
- [x] Module: admin 前端本地 API 回退 + Playwright 稳定性修正。
  - Tests: `npm run build` ✅ (cdn-system/web/admin)
  - Tests: `node scripts/sync-wwwroot.cjs` ✅ (cdn-system/web/admin)
  - Tests: `ADMIN_BASE_URL=http://127.0.0.1:5173 ADMIN_USER=admin ADMIN_PASS=123456 ADMIN_API_BASE=http://127.0.0.1:8080/api/v1/admin npm test` ✅ (cdn-system/tests/ui_e2e)

## 10) Batch Site Settings (New)

- [x] Module: 后端批量设置合并（AdminBatchUpdate 支持局部 settings 合并 + group_ids/backends）。
  - Tests: `go vet ./...` ✅ (cdn-system/api)
  - Tests: `go test ./...` ✅ (cdn-system/api)
- [x] Module: 前端批量设置弹窗 + 列表接入（独立组件，不影响既有批量功能）。
  - Tests: `npm run build` ✅ (cdn-system/web/admin)
  - Tests: `node scripts/sync-wwwroot.cjs` ✅ (cdn-system/web/admin)
- [x] Module: 验证（smoke_ws.ps1 + Playwright 验证计划）。
  - Tests: `python (等价执行 cdn-system/api/scripts/smoke_ws.ps1)` ✅
  - Tests: `npm run test:e2e` ✅ (cdn-system/web/admin, baseURL=127.0.0.1:5176 via preview)
  - Tests: `go vet ./...` ✅ (cdn-system/agent)
  - Tests: `go test ./...` ✅ (cdn-system/agent)
- [x] Module: 批量分组设置生效修复（重建 API 以支持 group_ids）。
  - Tests: `go vet ./...` ✅ (cdn-system/api)
  - Tests: `go test ./...` ✅ (cdn-system/api)
  - Tests: `python (admin login + /sites/batch_update group_ids 验证)` ✅
- [x] Module: agent Nginx 重载修复 + ACME HTTP-01 端口补齐（80 + HTTPSForce）+ 系统 Nginx 证书挑战透传。
  - Tests: `go vet ./...` ✅ (cdn-system/agent)
  - Tests: `go test ./...` ✅ (cdn-system/agent)
  - Tests: `/www/server/nginx/sbin/nginx -t` ✅
  - Tests: `/www/server/nginx/sbin/nginx -s reload` ✅
  - Tests: `python (config_sync + cert reissue + status ready)` ✅
- [x] Module: ACME HTTP-01 改为使用网站 HTTP 监听端口（移除固定 80 端口补位）。
  - Tests: `go vet ./...` ✅ (cdn-system/agent)
  - Tests: `go test ./...` ✅ (cdn-system/agent)
- [x] Module: 节点禁用状态前端不刷新修复（节点列表状态同步）。
  - Tests: `npm run build` ✅ (cdn-system/web/admin)
  - Tests: `node scripts/sync-wwwroot.cjs` ✅ (cdn-system/web/admin)

### 基本设置
- [x] 套餐设置：批量切换网站套餐 | backend: `user_package_id` | key: `user_package_id`
- [x] 所属分组：批量设置网站分组 | backend: `group_ids` (fallback `group_id`) | key: `group_ids`

### HTTP 设置
- [x] 开关：启用/禁用 HTTP 访问 | backend: `settings.http_enable` | key: `http_enable`
- [x] 监听端口：设置 HTTP 端口列表 | backend: `http_listen` | key: `http_listen`

### HTTPS 设置
- [x] 开关：启用/禁用 HTTPS | backend: `settings.https.enable` | key: `https.enable`
- [x] 证书：批量绑定证书 | backend: `cert_id` | key: `cert_id`
- [x] 监听端口：设置 HTTPS 端口列表 | backend: `https_listen` + `settings.https.listen_port` | key: `https_listen` / `https.listen_port`
- [x] 强制 HTTPS：HTTP 跳转 HTTPS + 端口 | backend: `settings.https.force` + `settings.https.redirect_port` | key: `https.force` / `https.redirect_port`
- [x] HSTS：启用/禁用 HSTS | backend: `settings.https.hsts` | key: `https.hsts`
- [x] HTTP2：启用/禁用 HTTP2 | backend: `settings.https.http2` | key: `https.http2`
- [x] HTTP3：启用/禁用 HTTP3 | backend: `settings.https.http3` | key: `https.http3`
- [x] OCSP Stapling：启用/禁用 OCSP | backend: `settings.https.ocsp_stapling` | key: `https.ocsp_stapling`
- [x] SSL 配置：SSL 策略/协议/加密套件 | backend: `settings.https.ssl_profile` + `settings.https.ssl_protocols` + `settings.https.ssl_ciphers` | key: `https.ssl_profile` / `https.ssl_protocols` / `https.ssl_ciphers`

### 回源设置
- [x] 回源协议：HTTP/HTTPS/跟随 | backend: `backend_protocol` + `settings.backend_protocol` | key: `backend_protocol`
- [x] HTTP 回源端口：设置回源端口 | backend: `settings.origin_http_port` | key: `origin_http_port`
- [x] HTTPS 回源端口：设置回源端口 | backend: `settings.origin_https_port` | key: `origin_https_port`
- [x] 回源 HOST：跟随/网站域名/自定义 | backend: `settings.origin_host` | key: `origin_host`
- [x] 回源超时：回源超时秒数 | backend: `settings.origin_timeout` | key: `origin_timeout`
- [x] 连接超时：回源连接超时 | backend: `settings.origin.connTimeout` | key: `origin.connTimeout`
- [x] 源站列表：源站 IP/权重/状态 | backend: `settings.origin.list` (+ `backends`) | key: `origin.list`
- [x] 条件源站：按条件匹配源站 | backend: `settings.origin.conditions` | key: `origin.conditions`
- [x] 负载方式：源站负载策略 | backend: `balance_way` | key: `balance_way`

### 缓存设置
- [x] 缓存规则：缓存规则列表 | backend: `settings.cache.rules` | key: `cache.rules`

### 安全设置
- [x] 默认 CC 规则：设置默认 CC 规则 | backend: `settings.security.default_rule` (+ `cc_default_rule`) | key: `security.default_rule`
- [x] 自动防护：QPS 自动切换规则 | backend: `settings.security.auto_switch` | key: `security.auto_switch`
- [x] 自定义规则：批量设置 CC 自定义规则 | backend: `settings.security.custom_rules` | key: `security.custom_rules`
- [x] 搜索引擎爬虫：爬虫放行/拦截 | backend: `settings.security.crawlers_action` | key: `security.crawlers_action`
- [x] 黑名单：IP 黑名单 | backend: `settings.security.blacklist` (+ `black_ip`) | key: `security.blacklist`
- [x] 白名单：IP 白名单 | backend: `settings.security.whitelist` (+ `white_ip`) | key: `security.whitelist`
- [x] 黑名单时间：IP 黑名单时间 | backend: `settings.security.ip_black_timeout` | key: `security.ip_black_timeout`
- [x] 白名单时间：IP 白名单时间 | backend: `settings.security.ip_white_timeout` | key: `security.ip_white_timeout`
- [x] Cookie 域名：共享 Cookie 域名 | backend: `settings.security.cookie` | key: `security.cookie`
- [x] 屏蔽透明代理：拦截透明代理 | backend: `settings.security.block_transparent_proxy` | key: `security.block_transparent_proxy`
- [x] 区域屏蔽：国家/地区屏蔽 | backend: `settings.security.region_block` | key: `security.region_block`

### 访问控制
- [x] ACL 规则：批量绑定 ACL | backend: `settings.access.acl` | key: `access.acl`
- [x] 防盗链：防盗链开关/范围/来源 | backend: `settings.access.hotlink` | key: `access.hotlink`
- [x] 跨域访问：CORS 配置 | backend: `settings.access.cors` | key: `access.cors`

### 高级设置
- [x] 上传大小限制：限制上传大小 | backend: `settings.upload_limit` | key: `upload_limit`
- [x] Gzip 压缩：启用/禁用 Gzip | backend: `settings.gzip` | key: `gzip`
- [x] Websocket：启用/禁用 Websocket | backend: `settings.websocket` | key: `websocket`
- [x] 搜索引擎回源：开启 + 回源 IP | backend: `settings.search_engine_origin` + `settings.search_engine_origin_ip` | key: `search_engine_origin` / `search_engine_origin_ip`
- [x] URL 转向：重写/转向规则 | backend: `settings.url_redirects` | key: `url_redirects`
- [x] 源站请求头：请求头列表 | backend: `settings.req_headers` | key: `req_headers`
- [x] CDN 响应头：响应头列表 | backend: `settings.res_headers` | key: `res_headers`
- [x] 访问日志：请求/响应头/请求体 | backend: `settings.log_request_header` + `settings.log_response_header` + `settings.log_request_body` + `settings.log_request_body_size_limit` | key: `log_request_header` / `log_response_header` / `log_request_body` / `log_request_body_size_limit`
- [x] 源站证书：回源证书校验 | backend: `settings.origin_cert` | key: `origin_cert`
- [x] 数据实时鉴别：实时鉴别 | backend: `settings.realtime_identify` | key: `realtime_identify`
- [x] 数据实时发送：实时发送 | backend: `settings.realtime_send` | key: `realtime_send`
- [x] 默认站点：默认站点开关 | backend: `settings.default_site` | key: `default_site`
- [x] L2 配置：L2 配置策略 | backend: `settings.l2_config` | key: `l2_config`

## 11) Admin Pagination Fixes

- [x] Module: 证书列表分页跳转修复 + 分页 size 选择警告修复。
  - Tests: `npm run build` ✅ (cdn-system/web/admin)
  - Tests: `node scripts/sync-wwwroot.cjs` ✅ (cdn-system/web/admin)
- [x] Module: 网站列表批量申请证书移除确认框。
  - Tests: `npm run build` ✅ (cdn-system/web/admin)
  - Tests: `node scripts/sync-wwwroot.cjs` ✅ (cdn-system/web/admin)

## 12) ACME Issue via Edge Nodes

- [x] Module: HTTP-01 签发改为节点执行（轮询节点 + 失败状态回写）。
  - Tests: `go test ./...` ✅ (cdn-system/api)
  - Build: `go build -o cdn-api .` ✅ (cdn-system/api)
  - Build: `npm run build` ✅ (cdn-system/web/admin)
  - Build: `node scripts/sync-wwwroot.cjs` ✅ (cdn-system/web/admin)
