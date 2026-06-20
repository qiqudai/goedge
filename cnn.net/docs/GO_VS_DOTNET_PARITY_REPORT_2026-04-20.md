# Go 与 .NET 功能对齐审计报告

审计日期: 2026-04-20

## 审计范围
- Go 后端: `/Users/fake/code/goedge/cdn-system/api`
- Go Agent: `/Users/fake/code/goedge/cdn-system/agent`
- Go 前端(Vue): `/Users/fake/code/goedge/cdn-system/web/admin/src`
- .NET 后端/前端/Agent: `/Users/fake/code/goedge/cdn-system/cnn.net`

## 一、后端 API 对齐结论
- 结论: **主体功能已基本补全**，核心接口覆盖高。
- 路由对齐统计（归一化参数后）:
  - Go 路由: `321`
  - .NET 路由(含 `/health` 与 `/ws/agent` 模块路由): `383`
  - 交集: `319`
  - 仅 Go 存在: `2`
  - 仅 .NET 存在: `64`

### 仅 Go 存在（需确认是否补齐）
- `GET /api/v1/admin/certs/batch/{}/progress`
- `GET /api/v1/user/certs/batch/{}/progress`

### 仅 .NET 存在（增强项，不视为缺失）
- 主要为删除工作流与可观测增强接口，例如 `delete_preview/delete_request`、`logs/events`、调试开关等。
- 典型示例:
  - `DELETE /api/v1/admin/forwards/{}`
  - `DELETE /api/v1/admin/sites/{}`
  - `DELETE /api/v1/admin/user_packages/{}`
  - `GET /api/v1/admin/certs/{}/delete_preview`
  - `GET /api/v1/admin/debug/server_switches`
  - `GET /api/v1/admin/forward_groups/{}/delete_preview`
  - `GET /api/v1/admin/forwards/{}/delete_preview`
  - `GET /api/v1/admin/logs/access/downloads`
  - `GET /api/v1/admin/logs/events`
  - `GET /api/v1/admin/node-groups/{}/delete_preview`
  - `GET /api/v1/admin/nodes/{}/delete_preview`
  - `GET /api/v1/admin/plans/{}/delete_preview`
  - `GET /api/v1/admin/rules/acl/{}/delete_preview`
  - `GET /api/v1/admin/rules/cc/filters/{}/delete_preview`
  - `GET /api/v1/admin/rules/cc/groups/{}/delete_preview`

## 二、前端对齐结论（Vue 路由 vs Blazor 页面）
- 结论: **核心业务页面已覆盖**（节点、网站、转发、套餐、系统、账户、登录/维护）。
- 依据: `web/admin/src/router/index.js` 与 `cnn.net/src/Cnn.Api/Pages/**/*.razor` 对照。
- 核心路由均可在 .NET 找到对应，例如:
  - `/node/list`, `/node/groups`, `/node/dns`, `/node/monitor`, `/node/realtime`
  - `/website/list`, `/website/groups`, `/website/resolve`, `/website/certs`, `/website/purge`, `/website/rules`, `/website/logs/block`, `/website/logs/access`
  - `/forward/list`, `/forward/groups`, `/forward/default`, `/forward/monitor`
  - `/plans/basic`, `/plans/sold`, `/plans/my`, `/plans/usage`
  - `/system/config`, `/system/tasks`, `/system/users`, `/system/logs`, `/system/upgrade`, `/system/announcements`, `/system/messages`
  - `/account/profile`, `/account/recharge`, `/account/bills`, `/account/logs`, `/account/messages`, `/account/subscribe`, `/account/apikey`
- .NET 还提供别名/扩展页（如 `/website/statistics` 与 `/website/monitor` 双路由）。

## 三、Agent 对齐结论
- 结论: **关键链路已对齐**，并且 .NET Agent 在可靠性层面有增强。
- Go 与 .NET 都具备以下能力:
  - `heartbeat` 心跳与 `heartbeat_ack` 处理
  - `/ws/agent` 长连接任务分发
  - `node/sync`、`l2_heartbeat`
  - `logs/access|metrics|events` 上报
  - `certs/issued` 回报
  - `acme/tokens` 写入/删除
  - 升级包下载与应用 `/api/v1/agent/upgrade/package`
- 关键证据文件:
  - Go: `agent/ws_client.go`, `agent/heartbeat.go`, `agent/config.go`, `agent/tasks.go`, `agent/upgrade.go`, `agent/acme_token_store.go`
  - .NET: `src/Cnn.Agent/Ws/AgentWsClient.cs`, `src/Cnn.Agent/Program.cs`, `src/Cnn.Agent/Acme/AcmeTokenStore.cs`

## 四、逻辑一致性判断
- 当前判断: **主要流程逻辑一致，未发现系统性缺口**。
- 补齐进展（2026-04-20 二次补充）:
  1. `GET /api/v1/admin/certs/batch/{id}/progress` 已在 .NET 补齐。
  2. `GET /api/v1/user/certs/batch/{id}/progress` 已在 .NET 补齐，并带用户归属过滤。
  3. 新增批次进度聚合逻辑：按 `task.type=issue_cert` 与 `task.pid=batchId` 统计 `success/fail/running/pending`，返回 `fail_items`。
  4. 新增行为测试覆盖 admin 聚合与 user 权限过滤。

## 五、已执行验证
- `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj` -> `65/65` 通过
- `dotnet test tests/Cnn.Agent.Tests/Cnn.Agent.Tests.csproj` -> `23/23` 通过
- `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "CertServiceBehaviorTests|SiteServiceOwnershipTests|GoRouterParityTests|RouteParityCoverageTests"` -> `5/5` 通过
