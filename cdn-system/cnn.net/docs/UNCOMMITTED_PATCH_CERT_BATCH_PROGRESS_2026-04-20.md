# 未提交改动清单（持续更新：cert batch progress / clickhouse / geo）

日期: 2026-04-20
范围: `.NET` 证书批次进度接口补齐、ClickHouse 本地回归、geo 统计链路修复

## 改动文件
- `src/Cnn.Common/Contracts/Admin/CertDtos.cs`
- `src/Cnn.Api/Services/Admin/CertService.cs`
- `src/Cnn.Api/Services/Admin/CertService.Batch.cs` (新增)
- `src/Cnn.Api/Controllers/Admin/CertsController.cs`
- `src/Cnn.Api/Controllers/User/CertsController.cs`
- `src/Cnn.Api/Services/Admin/BlockLogService.cs`
- `src/Cnn.Api/Services/Stats/RankingService.cs`
- `src/Cnn.Api/Services/Stats/AccessLogGeoExpressions.cs` (新增)
- `src/Cnn.Api/Services/Admin/ForwardGroupService.cs`
- `scripts/start_local_api_mysql.sh`
- `scripts/run_local_api_mysql.sh`
- `scripts/verify_live_mysql.sh`
- `scripts/verify_live_mysql_extended.sh`
- `scripts/smoke_live_mysql_extended.sh`
- `scripts/start_local_clickhouse.sh` (新增)
- `scripts/stop_local_clickhouse.sh` (新增)
- `scripts/init_local_clickhouse_schema.sh` (新增)
- `scripts/verify_local_clickhouse.sh` (新增)
- `tests/Cnn.Api.Tests/CertServiceBehaviorTests.cs`
- `tests/Cnn.Api.Tests/SiteServiceOwnershipTests.cs`
- `tests/Cnn.Api.Tests/AccessLogGeoExpressionsTests.cs` (新增)
- `tests/Cnn.Api.Tests/AgentLogGeoWriteTests.cs` (新增)
- `tests/Cnn.Api.Tests/ClickHouseGeoQueryTests.cs` (新增)
- `docs/GO_VS_DOTNET_PARITY_REPORT_2026-04-20.md`
- `docs/REGRESSION_STATUS_2026-04-20.md`

## 验证命令
- `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "CertServiceBehaviorTests|SiteServiceOwnershipTests|GoRouterParityTests|RouteParityCoverageTests"`
- `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "ClickHouseGeoQueryTests|AgentLogGeoWriteTests|AccessLogGeoExpressionsTests"`
- `dotnet test tests/Cnn.Agent.Tests/Cnn.Agent.Tests.csproj`
- `./scripts/start_local_clickhouse.sh`
- `./scripts/init_local_clickhouse_schema.sh`
- `./scripts/verify_local_clickhouse.sh`
- `./scripts/run_local_api_mysql.sh`
- `./scripts/verify_live_mysql_extended.sh`

结果:
- `CertServiceBehaviorTests|SiteServiceOwnershipTests|GoRouterParityTests|RouteParityCoverageTests`: 5/5 通过
- `ClickHouseGeoQueryTests|AgentLogGeoWriteTests|AccessLogGeoExpressionsTests`: 15/15 通过
- `Cnn.Agent.Tests`: 23/23 通过
- 本地真实接口验证通过：
  - `/api/v1/admin/stats/ranking?type=country&range=7d`
  - `/api/v1/admin/logs/block/history?page=1&pageSize=10`
  - `/api/v1/admin/logs/block/current?page=1&pageSize=10&type=ip&keyword=`
- 扩展全链路回归通过：
  - `verify_live_mysql_extended.sh` 全 15 步通过
  - 覆盖 DNS API / Cert / ACL / GlobalConfig / CC / Purge / SiteCache / Plan / Forward / ApplyCert / Node / DNS Provider / Cleanup

## 本轮新增修复
- 修复 `forward_group` 删除时的外键崩溃：
  - 现象：删除 `stream_group` 时命中 `merge_stream_group` FK，返回 500
  - 修复：`ForwardGroupService.DeleteAsync` 事务内先删 `merge_stream_group`，再删 `stream_group`
- 修复扩展回归脚本稳定性：
  - `cert disable` 后增加状态等待与删除重试，消除偶发 `40903`
  - 验证入口脚本增加“强制清理残留 API 进程”步骤，避免跨轮限流/状态污染

## Git 添加结果（2026-04-20）
- 已执行：将 `cnn.net` 项目相关目录加入暂存区
  - `cdn-system/cnn.net/src/**`
  - `cdn-system/cnn.net/tests/**`
  - 本次相关文档：
    - `cdn-system/cnn.net/docs/GO_VS_DOTNET_PARITY_REPORT_2026-04-20.md`
    - `cdn-system/cnn.net/docs/PAGINATION_FINAL_ACCEPTANCE_CHECKLIST.md`
    - `cdn-system/cnn.net/docs/PAGINATION_GUIDELINES.md`
    - `cdn-system/cnn.net/docs/UNCOMMITTED_PATCH_CERT_BATCH_PROGRESS_2026-04-20.md`
    - `cdn-system/cnn.net/docs/UNCOMMITTED_PATCH_CONTINUE_2026-04-20.md`
- 已排除（未加入暂存）：非项目核心/中间内容（例如 `docs/**/node_modules/**` 等镜像资料）
- 当前暂存区中 `cdn-system/cnn.net/(src|tests|docs)` 文件数：`749`

## 仓库边界处理（2026-04-20）
- 已确认：`cnn.net` 原先挂在父仓库 `/Users/fake/code/goedge/.git` 下。
- 已处理：先从父仓库暂存区撤掉 `cdn-system/cnn.net` 路径，再在 `cdn-system/cnn.net` 下初始化独立 `.git`。
- 当前状态：`cnn.net` 已可独立执行 `git status/git add/commit`，不再把父目录文件纳入本仓库暂存。

## 其他 cs/前端代码（未纳入本次 add）
- 统计：`417` 条
- 明细文件：`cdn-system/cnn.net/docs/OTHER_CS_FRONTEND_REMAINING_2026-04-20.txt`
- 说明：该列表主要来自 `cdn-system/web/admin/dist-publish/assets/**` 的前端构建产物变更（删除/替换），属于与本次 `.NET` 迁移提交范围分离的候选项。

## 本次新增说明
- 本次没有触碰父目录 `../web/admin` 的 TypeScript 源码。
- 当前新增的脚本与代码都限定在 `cnn.net` 仓库内，用于：
  - 本地安装后的 ClickHouse 启停与建表
  - `.NET` API 启动时自动接入本地 ClickHouse
  - 统计/封禁日志页面与 Go 侧一致地读取 `client_country` / `client_province`
- 真实回归已经证明：本机 `Cnn.Api` 可以直接从本地 ClickHouse 读出 IP 归属地并在 admin API 中返回。

## 本仓库未纳入（已忽略）
- 根目录临时重构脚本：
  - `apply_constants.py`
  - `do_split.py`
  - `extract_common.py`
  - `extract_di.py`
  - `extract_endpoints.py`
  - `extract_ws_handler.py`
  - `fix_constructors.py`
  - `fix_dns_methods.py`
  - `fix_endpoints_2.py`
  - `fix_modules.py`
  - `fix_razor.py`
  - `fix_razor2.py`
  - `fix_syntax.py`
  - `fix_tasks.py`
  - `refactor_controllers.py`
  - `refactor_dns.py`
  - `refactor_domain_usage.py`
  - `refactor_forward.py`
  - `update_task_service.py`
  - `update_task_service2.py`
- 文档镜像产物：
  - `docs/**/node_modules/`
  - `docs/**/test-results/`

## 本轮补充（80机真实环境 + 统计链路修复）

### 新增代码改动
- `src/Cnn.Api/Services/Admin/CertIssueProcessor.cs`
  - 证书签发完成回写时，保证 `auto_renew` 不为 `null`，避免任务回写失败。
- `src/Cnn.Api/Services/Stats/ClickHouseHttpHelper.cs`
  - ClickHouse HTTP 请求改为 POST 携带空 body，兼容 22.1 的 Content-Length 要求。
- `src/Cnn.Api/Services/Stats/AccessStatsService.cs`
  - `bytes` 聚合字段改为显式转义；别名 `bytes` 改为 `out_bytes`，适配旧版 CH 聚合解析。
- `src/Cnn.Api/Services/Stats/RankingService.cs`
  - 统一 `sum("bytes")/sumIf("bytes",...)` 聚合写法。
- `src/Cnn.Api/Services/Common/UserPackageTrafficWorker.cs`
  - 统一 `sum("bytes")` 写法。

### 80机隔离环境实测补充
- API: `http://127.0.0.1:15035`
- MySQL: `cnn_test_20260420`
- ClickHouse: `cnn_test_20260420`
- 通过项：
  - DNSAPI 配置 CRUD
  - 证书默认设置 set/get
  - 全局默认设置 get/update/get
  - 站点缓存 save/load/compile
  - 站点 apply_cert 后 HTTPS 状态与 certificate_id
  - 日志统计 basic/quality/origin（写入样本后数值核对通过）

### 当前仍是外部依赖导致的非代码失败
- ACME 对 `.test` 伪域名签发失败（预期行为，非代码缺陷）。
- 外部 DNS 厂商鉴权若使用无效测试密钥会失败（环境配置问题，非代码缺陷）。
