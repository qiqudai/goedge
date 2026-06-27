# Findings — UX Loading State

## 现状
- `useLoading` 是单例全局遮罩，会覆盖整个 dialog 或主区域
- `request.js` 已支持 `skipLoading: true` 跳过全局遮罩 → 可用于轮询
- `TaskProgressBar.vue` / `TaskMonitorDialog.vue` 已存在，但未在证书/站点批量操作中接入
- 证书状态字段：`state` + `issue_task_state`；非终态 = `waiting|issuing|dns_pending`
- 站点 CNAME 状态字段：`cname` + `resolve_status`（待与后端对齐）

## 改造原则
- 行内小 spinner + 文案 `数据加载中...`，颜色与主色一致 (#409eff)
- 自动轮询：存在非终态行时每 10s 拉取一次；全部稳定后停止
- 弹窗确定按钮：`<el-button :loading="submitting">`，提交期间禁用
- 批量进度窗口：标题 `操作进度` + 百分比 + 取消按钮 (图5)
