# cccadmin 功能修复进度

## 总览
- [x] 网站列表与管理（站点增删改、批量、清缓存、申请证书、解析检测、默认设置、DNS 接口）
- [x] 四层转发列表（增删改/批量/启用禁用）
- [x] 证书管理与 DNS 接口（含 DNS API 列表/新增/删除/默认设置）
- [x] 转发默认设置
- [x] 转发实时监控
- [x] 账户充值
- [x] 套餐用量（真实 CK 数据）
- [x] 消息红点/数字、消息详情红点、消息中文化
- [x] 主题切换按钮动效与图标
- [x] 个人资料改密弹窗标签修正

## 测试记录
- `npx playwright test tests/e2e/user-site-actions.spec.ts` ✅
- `npx playwright test tests/e2e/user-forward-actions.spec.ts` ✅
- `npx playwright test tests/e2e/user-dns-certs.spec.ts` ✅
- `npx playwright test tests/e2e/user-forward-default.spec.ts` ✅
- `npx playwright test tests/e2e/user-forward-monitor.spec.ts` ✅
- `npx playwright test tests/e2e/user-usage-recharge.spec.ts -g "usage endpoint"` ✅
- `E2E_USER_TOKEN=*** E2E_USER_ID=2 npx playwright test tests/e2e/user-usage-recharge.spec.ts -g "recharge endpoint"` ✅
- `E2E_USER_TOKEN=*** E2E_USER_ID=2 npx playwright test tests/e2e/user-messages.spec.ts` ✅
- `E2E_USER_TOKEN=*** npx playwright test tests/e2e/user-ui-message-theme-profile.spec.ts` ✅
- `E2E_USER_TOKEN=*** E2E_USER_ID=2 npx playwright test tests/e2e/user-ui-message-theme-profile.spec.ts` ✅
