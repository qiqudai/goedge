# 分页最终验收清单

## 1. 功能正确性
- [x] 分页页面首次进入时：恢复历史配置后再发首个列表请求。
- [x] “搜索”动作总是回到第 1 页。
- [x] “刷新”动作保持当前页与当前 pageSize。
- [x] 切页、改 pageSize、开关分页后，请求参数与 UI 展示一致。
- [x] 关闭分页时请求量受 `UnpagedLimit` 保护（默认 1000）。

## 2. 并发与竞态
- [x] 快速连续切页仅落最后状态，不出现回跳。
- [x] 快速切换 pageSize 仅落最后状态，不出现旧 pageSize 覆盖。
- [x] 连续“搜索 + 刷新 + 切页”不产生请求风暴（无无效重复请求堆积）。
- [x] 页面层未额外叠加与 `TablePager` 冲突的并发控制逻辑。

## 3. 性能与体验
- [ ] 常规分页请求响应时间在可接受范围内（按页面 SLO）。（当前仅有功能回归，未做 SLO 压测）
- [ ] 列表加载中状态可见、结束后及时恢复交互。（当前未补专门体验断言）
- [ ] 大页码和大 pageSize 下无明显卡顿或长时间白屏。（当前未补性能压测）
- [x] 关键按钮文案统一：检索为“搜索”，拉新为“刷新”。

> 说明：第 3 章为“性能与体验”专项，当前轮次仅完成功能回归，不将未执行的压测项误标为通过。

## 4. 可观测性
- [x] `TablePager` 结构化日志可检索到：
  - `uiEvent=table_pager_query`
  - `stateKey`
  - `page`
  - `pageSize`
  - `pagingEnabled`
  - `total`
- [x] 关键分页页可通过 `stateKey` 维度做聚合分析。
- [x] 任务相关页“分页诊断”默认关闭，且可手动显示/隐藏。
- [x] 故障排查时可通过日志与诊断面板快速对齐同一时刻的分页状态。

## 5. 回归建议（最小集合）
- [x] 页面级自动化覆盖：
  - `website/list`
  - `node/list`（含监控日志弹窗）
  - `system/tasks`
  - `website/logs/access`
  - `website/logs/block`
- [x] 核心断言覆盖：
  - 恢复 pageSize 再首请
  - `SearchAsync()` 回第 1 页
  - `RefreshAsync()` 保持当前页
  - 快速操作只落最新状态

## 6. 当前自动化覆盖（2026-04-20）
- [x] `website/list`
- [x] `website/groups`（admin 选用户前置）
- [x] `website/dnsapi`
- [x] `website/resolve`
- [x] `forward/list`
- [x] `node/list`（含监控日志弹窗）
- [x] `system/tasks`
- [x] `system/messages`
- [x] `system/announcements`
- [x] `account/logs`
- [x] `account/messages`
- [x] `account/bills`
- [x] `website/logs/access`（含申请记录子列表）
- [x] `website/logs/block/current`
- [x] `website/logs/block/stats`
- [x] `website/logs/block/history`
- [x] `website/purge`（操作记录列表）
- [x] `website/certs`
- [x] `system/logs/login`
- [x] `system/logs/operation`
- [x] `finance/orders`
- [x] `system/logs`（多 tab 场景）
- [x] `node/groups`
- [x] `system/users`
- [x] UI 规范校验（分页页“搜索/刷新”文案与刷新语义）
- [x] `TablePager` 结构化日志字段校验（含关闭分页场景）

## 7. 页面-用例矩阵（页面级回归）
| 页面标识 | 页面文件/组件 | 回归测试方法 |
| --- | --- | --- |
| `website:list:table` | `Website/List.razor` | `WebsiteList_Pagination_Regression` |
| `website:groups:table` | `Website/Groups.razor` | `WebsiteGroups_Pagination_Regression_WithAdminUserSelection` |
| `website:dnsapi:table` | `Website/DnsApiTab.razor` | `WebsiteDnsApiTab_Pagination_Regression` |
| `website:resolve:table` | `Website/SiteResolvePanel.razor` | `WebsiteResolvePanel_Pagination_Regression` |
| `forward:list:table` | `Forward/List.razor` | `ForwardList_Pagination_Regression` |
| `node:list:table` | `Node/List.razor` | `NodeList_AndMonitorDialog_Pagination_Regression` |
| `node:list:monitor:table` | `Node/List.razor`（监控弹窗） | `NodeList_AndMonitorDialog_Pagination_Regression` |
| `system:tasks:table` | `System/Tasks.razor` | `SystemTasks_Pagination_Regression` |
| `system:messages:table` | `System/Messages.razor` | `SystemMessages_Pagination_Regression` |
| `system:announcements:table` | `System/Announcements.razor` | `SystemAnnouncements_Pagination_Regression` |
| `account:logs:table` | `Account/Logs.razor` | `AccountLogs_Pagination_Regression` |
| `account:messages:table` | `Account/Messages.razor` | `AccountMessages_Pagination_Regression` |
| `account:bills:table` | `Account/Bills.razor` | `AccountBills_Pagination_Regression` |
| `website:access_logs:query:table` | `Website/AccessLogs.razor` | `WebsiteAccessLogs_Pagination_Regression` |
| `website:access_logs:history:table` | `Website/AccessLogs.razor`（申请记录） | `WebsiteAccessLogs_History_Pagination_Regression` |
| `website:block_logs:current:table` | `Website/BlockLogs.razor`（当前封禁） | `WebsiteBlockLogs_Pagination_Regression` |
| `website:block_logs:stats:table` | `Website/BlockLogs.razor`（统计） | `WebsiteBlockLogs_Stats_Pagination_Regression` |
| `website:block_logs:history:table` | `Website/BlockLogs.razor`（历史记录） | `WebsiteBlockLogs_History_Pagination_Regression` |
| `website:purge:list:table` | `Website/Purge.razor`（操作记录） | `WebsitePurgeList_Pagination_Regression` |
| `website:certs:table` | `Website/Certs.razor` | `WebsiteCerts_Pagination_Regression` |
| `system:login_logs:table` | `System/LoginLogs.razor` | `SystemLoginLogs_Pagination_Regression` |
| `system:operation_logs:table` | `System/OpLogs.razor` | `SystemOperationLogs_Pagination_Regression` |
| `finance:orders:table` | `Finance/Orders.razor` | `FinanceOrders_Pagination_Regression` |
| `system:logs:table` | `System/Logs.razor`（多 Tab） | `SystemLogs_Pagination_Regression` |
| `node:groups:table` | `Node/Groups.razor` | `NodeGroups_Pagination_Regression` |
| `system:users:table` | `System/Users.razor` | `SystemUsers_Pagination_Regression` |

## 8. 验证命令与执行记录
- 执行日期：`2026-04-20`
- 命令：
  - `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "PaginationPageRegressionTests"`
  - `dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter "TablePagerTests|PaginationPageRegressionTests|TaskPagesPagingDiagnosticsTests|PaginationUiConventionsTests"`
- 结果：
  - `PaginationPageRegressionTests`: `25/25` 通过
  - 组合筛选集：`36/36` 通过
  - 备注：并行运行 `dotnet test` 会触发 `MvcTestingAppManifest.json` 文件锁，验收记录按串行执行结果统计。
