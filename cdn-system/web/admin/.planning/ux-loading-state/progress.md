# Progress — UX Loading State

## Session 2026-06-27

### 已完成
- 新增 `InlineLoading.vue`：小 spinner + 文案，支持 xs/sm/md；匹配图1/图2 风格
- 新增 `usePolling.js`：通用轮询 composable，shouldRun 控制启停
- 新增 `BatchProgressDialog.vue`：图5 操作进度窗（标题/百分比/取消按钮/失败列表）
- Dashboard 改造：
  - 运营数据 / 网络概览 / 监控趋势 / TOP10 / 系统状态 / 系统授权 各自 `v-loading` 行内遮罩
  - 拆分为 6 个独立 fetch (ops/overview/chart/top/system/license)，全部 `skipLoading: true`
  - 时间范围切换只刷新对应区块
  - 系统授权「刷新授权」/ Agent「立即检查」按钮自带 loading
- 证书列表：
  - `dns_pending / waiting / issuing` 状态使用 `<InlineLoading>` 替代静态文案
  - 列表存在非终态证书时每 10s 自动轮询，全部稳定后停止
  - 列表请求改为 `skipLoading: true`，不再触发全局遮罩
- 站点列表：
  - CNAME 单元格新增行内 loading（pending）或成功图标（ok）
  - 列表存在 `cname_status/resolve_status=pending|generating|syncing|waiting` 时自动轮询
- 弹窗确定按钮 `:loading`：
  - CertEditPopup（单个/批量/泛证书）
  - DnsApiList（DNS 接口新增/编辑）
  - DefaultSettings（站点默认设置新增/编辑）
  - Certs.vue 默认设置保存
  - SiteEditDialog / BatchEditDialog 已有 loading（保留）
  - BatchSettingsDialog 各 section 批量修改按钮已有 loading
- 批量操作进度窗口接入：
  - List.vue 启用/禁用/删除/解除黑名单使用 `BatchProgressDialog`
  - 进度按总数推进 0→90%，API 返回后置 100%
  - 支持取消（取消令牌 + 停止轮询）
  - 完成关闭后自动 fetchSites 刷新

### 验证
- `vue-tsc --noEmit -p tsconfig.app.json` ✅
- `vite build` ✅ (35.39s, dist 已生成)
- ReadLints：改动 10 个文件均无 linter 报错

### 待确认项
- Dashboard 后端是否提供 `/dashboard/ops_summary`、`/dashboard/overview`、`/dashboard/charts`、`/dashboard/top`、`/dashboard/system_status`、`/dashboard/license`、`/dashboard/sidebar` 拆分接口；若仍只提供 `/dashboard` 聚合接口，需要在后端补拆分端点，或前端将 6 个独立 fetch 合并回 1 个但仍保留各区块的本地 loading ref
- 站点行 `cname_status/resolve_status` 字段需后端在 `/sites` 列表项中返回；当前前端做了字段兜底，若后端未提供则不会显示 loading/ok 图标（向后兼容）
- 批量进度窗口当前为前端按总数推进的视觉进度；若后端 `/sites/batch_action` 改为异步任务返回 task_id，可平滑切换到 `TaskMonitorDialog` 已有实现
