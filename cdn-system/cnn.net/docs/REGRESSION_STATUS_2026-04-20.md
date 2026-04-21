# Regression Status 2026-04-20

## Scope
- Repository: `cnn.net`
- Runtime under test: `Cnn.Api` + `Cnn.Agent`
- Excluded: parent Go/Vue project under `../web/admin`

## Automated results
- `dotnet test tests/Cnn.Agent.Tests/Cnn.Agent.Tests.csproj`
  - passed: 23/23
- `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "PaginationPageRegressionTests|PaginationUiConventionsTests|TaskPagesPagingDiagnosticsTests|TablePagerTests|RealDataClickHouseIntegrationTests|DnsApiServiceOwnershipTests|SiteServiceOwnershipTests|TaskDispatchWorkerTests|CertServiceBehaviorTests"`
  - passed: 48/48
- `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "ClickHouseGeoQueryTests|AgentLogGeoWriteTests|AccessLogGeoExpressionsTests"`
  - passed: 15/15
- Latest full-suite rerun in this workspace session:
  - `tests/Cnn.Api.Tests`: 75/75 passed
  - `tests/Cnn.Agent.Tests`: 23/23 passed

## Live API verification
- Startup command that works in current workspace: `./scripts/run_local_api_mysql.sh`
- Full extended live regression command (MySQL + API):
  - `./scripts/verify_live_mysql_extended.sh`
  - result: passed end-to-end (`[A/15]` ~ `[O/15]`)
- Local ClickHouse installation verified:
  - binary: `/usr/local/bin/clickhouse`
  - version: `26.3.3.20-lts`
- Local ClickHouse runtime verified:
  - startup command: `./scripts/start_local_clickhouse.sh`
  - schema init: `./scripts/init_local_clickhouse_schema.sh`
  - health check: `./scripts/verify_local_clickhouse.sh`
  - HTTP DSN in use during regression: `http://127.0.0.1:8123/default`
- Health endpoint verified:
  - `GET /api/health` -> 200
- Real ClickHouse-backed statistics verified with seeded local dataset:
  - inserted sample rows into `default.node_access_logs`
  - `GET /api/v1/admin/stats/ranking?type=country&range=7d` -> 200
  - result included `item = 中国`, `request_count = 2`, `out_traffic = 1.00 MB`
- Real ClickHouse-backed block history verified:
  - `GET /api/v1/admin/logs/block/history?page=1&pageSize=10` -> 200
  - result included `ip = 1.2.3.5`, `location = 中国-广东省`, `filter = HTTP_403`
- Real ClickHouse-backed current block list verified:
  - `GET /api/v1/admin/logs/block/current?page=1&pageSize=10&type=ip&keyword=` -> 200
  - result included `ip = 1.2.3.5`, `location = 中国-广东省`, `release_time = PERMANENT`
- Static assets verified through smoke script:
  - `/css/site.css` -> 200
  - `/js/app.js` -> 200
- Admin site create flow verified manually:
  - `POST /api/v1/admin/sites` -> 200
  - site detail returned expected default protocol `https`
- Admin site batch CNAME update verified manually:
  - `POST /api/v1/admin/sites/batch_update` with valid existing `cname_domain` -> 200
- Site disable task verified end-to-end at DB level:
  - task row `SITE_DISABLE` reached `success`
  - `site.enable` changed to `0`
  - `site.state` changed to `stop`
- Forward proxy chain verified in extended run:
  - forward group/default/stream create/update/delete all passed
  - site apply-cert flow passed (created cert + site https enabled)
  - node + DNS provider CRUD passed

## Script fixes made during regression
- Fixed race in `scripts/smoke_live_mysql.sh`
  - after `disable`, script now polls site detail before issuing `delete`
- Fixed false-negative status parsing in `scripts/smoke_live_mysql.sh`
  - previous jq expression treated `false` as fallback `true`
- Added local ClickHouse helper scripts:
  - `scripts/start_local_clickhouse.sh`
  - `scripts/stop_local_clickhouse.sh`
  - `scripts/init_local_clickhouse_schema.sh`
  - `scripts/verify_local_clickhouse.sh`
- Updated API startup helpers to auto-detect local ClickHouse:
  - `scripts/start_local_api_mysql.sh`
  - `scripts/run_local_api_mysql.sh`
- Hardened full verification entry scripts:
  - `scripts/verify_live_mysql.sh`
  - `scripts/verify_live_mysql_extended.sh`
  - before verification now force-clean stale API process to avoid cross-run pollution
- Hardened extended smoke script:
  - `scripts/smoke_live_mysql_extended.sh`
  - cert disable->delete now waits for disable state and retries on transient state conflict
  - base smoke step can be toggled via `RUN_BASE_SMOKE` (default skip)
  - JSON assertion now reports raw invalid response payload for faster debugging
- Fixed backend delete behavior for forward group:
  - `ForwardGroupService.DeleteAsync` now deletes `merge_stream_group` relations in transaction before deleting `stream_group`
  - resolved previous MySQL FK crash (`merge_stream_group` -> `stream_group`) seen during extended regression

## .NET Agent/API compatibility baseline (2026-04-20)
- Agent compatibility relaxed to align with Go control-plane payload:
  - `ProxyConfigValidator` no longer rejects negative `version` values.
  - `origin_protocol=follow` is accepted and no longer treated as invalid.
- Agent config apply observability expanded:
  - `config_sync` ACK `applied` now includes stream summary (`received/planned/applied/skipped/skip_reasons`).
  - stream runtime report now records last apply snapshot and skip reasons.
- API ACK audit field validation added:
  - for `task_type=config_sync`, API validates and records `streams_received/streams_applied/streams_skipped/streams_reason`.
  - invalid/missing stream audit fields are marked with `streams_audit_valid=false` and diagnostic reason.
- Dispatch/verification script baseline solidified:
  - `scripts/verify_agent_ws_ack.sh` enforces ACK `state=success`.
  - large payload path supports `PAYLOAD_FILE` and sends by file body to avoid argument length issues.
  - supports `AUTH_X_FORWARDED_FOR` pass-through.

## Confirmed blockers / gaps
1. Agent real-time effectiveness is not fully provable in current local environment.
- There are node records in MySQL, but no validated live agent heartbeat/session evidence was established in this run.
- Multiple `CONFIG_SYNC` tasks remain in `waiting`, which means config sync generation exists, but no live local agent consumption was demonstrated.

2. ClickHouse statistics are now wired through runtime environment overrides, not static appsettings.
- `src/Cnn.Api/appsettings.json` and `src/Cnn.Api/appsettings.Development.json` still do not hardcode `ClickHouse:Dsn`.
- Local verification now proves real `node_access_logs` ingestion and API aggregation when `ClickHouse__Dsn=http://127.0.0.1:8123/default` is provided by startup scripts.
- Remaining gap: this is validated for the tested admin endpoints above, not yet for every dashboard/chart page in a browser walkthrough.

3. IP geolocation / region attribution has been aligned to the Go-side ClickHouse model.
- `RankingService` and `BlockLogService` now read `client_country` / `client_province` from ClickHouse records instead of calling the previous stub `IIpRegionService.Lookup()`.
- `AgentLogService.InsertAccessLogsAsync` now forwards `client_country` / `client_province` into `node_access_logs`.
- Targeted tests passed:
  - `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "AgentLogGeoWriteTests|AccessLogGeoExpressionsTests"`
- Remaining prerequisite: ClickHouse must actually be configured and ingest those geo fields in the runtime environment under test.

4. Full browser-level verification of every front-end page was not completed.
- API and page backend contracts were exercised through tests and live HTTP calls.
- A true browser-driven walkthrough still requires a stable browser automation harness inside `cnn.net` plus a live agent/data source for end-to-end confirmation.

## Practical conclusion
- `cnn.net` API and core page contract regressions are in mostly good shape.
- Site/task workflow works for create/update/disable on the API side.
- It is not technically correct to claim that "every front-end setting already takes effect on agent in real time" in this local environment.
- It is technically correct to claim that local ClickHouse statistics and block-log geo aggregation are working end-to-end for the verified admin APIs on this machine.
- It is still not technically correct to claim that every browser page and every live agent-consumed setting has been fully proven end-to-end in this local environment.

## 80 服务器真实环境补充验证（2026-04-20 23:xx CST）
- 测试目标主机：`202.73.4.80`
- 严格使用隔离测试库：
  - MySQL: `cnn_test_20260420`（`127.0.0.1:13306`）
  - ClickHouse: `cnn_test_20260420`（`http://127.0.0.1:32770`）
- 测试 API 实例：`http://127.0.0.1:15035`（未触碰线上 `:8080` 实例重启）

### 核心结论（新增）
- `DNS API` 配置、证书默认设置、全局默认设置、站点缓存设置、站点 HTTPS 应用：已在 80 机隔离实例实测通过。
- `4 层转发` 两种模式在 80 机隔离环境均已验证：
  - userspace 模式：监听端口转发到源站成功（返回 200）
  - nat 模式：`iptables` DNAT 规则下发成功，数据面访问成功（返回 200）
- `日志统计正确性` 新增实测：
  - 先向 ClickHouse 写入可计算样本（3 条访问日志：200/HIT, 404/MISS, 503/MISS）
  - 再调用 `/api/v1/admin/stats/basic|quality|origin`（custom time range）
  - 结果与预期一致：`qps=0.05`, `hit_rate=33.33`, `status_4xx=1`, `status_5xx=1`, `traffic=0.01MB`

### 统计链路缺陷与修复（新增）
- 发现问题：在 80 机当前 ClickHouse 版本（`22.1.3.7`）下，统计链路出现“接口返回 0”的兼容性问题。
- 根因：
  1. ClickHouse HTTP 调用使用 `POST` 但无 body，触发 `HTTP_LENGTH_REQUIRED`。
  2. AccessStats 聚合 SQL 中 `bytes` 字段/别名在该版本解析存在兼容性问题。
- 修复：
  - `ClickHouseHttpHelper`：POST 请求显式附加空 body（带 Content-Length）。
  - `AccessStatsService`：`sum("bytes")` + `sumIf("bytes",...)`，并将别名从 `bytes` 调整为 `out_bytes`。
  - `RankingService` / `UserPackageTrafficWorker`：统一 `bytes` 聚合字段转义。
- 修复后同机复测：通过（见上文“日志统计正确性新增实测”）。
