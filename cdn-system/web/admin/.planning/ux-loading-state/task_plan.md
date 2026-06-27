# UX Loading State 改造计划

**Goal:** 替换旧版全屏 loading 遮罩为组件级 loading；自动轮询异步状态（证书签发、CNAME 解析）；批量操作进度窗口。

**Created:** 2026-06-27

---

## Phases

### Phase 1 — 基础设施
- [ ] InlineLoading 组件 (小 spinner + 文案)
- [ ] usePolling composable (通用轮询直到终态)
- [ ] BatchProgressDialog 组件 (图5)

### Phase 2 — Dashboard 改造
- [ ] 运营数据 / 网络概览 / 监控趋势 / TOP10 各自独立 loading
- [ ] 系统授权 / Agent 状态行内 loading
- [ ] 关闭全局遮罩对该页的影响（用 skipLoading + 本地 loading ref）

### Phase 3 — 证书签发/CNAME 自动刷新
- [ ] 证书列表：issuing/dns_pending/waiting 状态显示小 spinner
- [ ] 列表存在非终态证书时启动轮询 (10s)，全终态停止
- [ ] SiteHeader：状态/CNAME loading + 自动检测解析生效

### Phase 4 — 弹窗确定按钮 loading
- [ ] SiteEditDialog / CertEditPopup / BatchEditDialog / BatchSettingsDialog 等
- [ ] 提交期间 :loading=true，防止重复点击

### Phase 5 — 批量操作进度窗口
- [ ] 接入 BatchProgressDialog 到证书批量 / 站点批量
- [ ] 显示 0%→100% 进度条 + 取消按钮

### Phase 6 — 验证
- [ ] vue-tsc / vite build
- [ ] Playwright 冒烟
- [ ] 更新 progress.md
