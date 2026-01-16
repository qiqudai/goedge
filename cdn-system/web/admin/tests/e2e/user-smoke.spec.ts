import { test } from '@playwright/test'
import { attachGuards, clickTab, clickToolbarButtons, createLoadingTracker, enableFastSmokeStubs, gotoAndWait, login } from './_helpers'

test('user: pages load without API errors', async ({ page }) => {
  test.setTimeout(600000)
  test.skip(process.env.E2E_SMOKE !== '1', 'Smoke e2e requires E2E_SMOKE=1 with seeded data')
  const guards = attachGuards(page)
  const loadingTracker = createLoadingTracker(page, { allowMissing: true })
  await enableFastSmokeStubs(page)

  await loadingTracker.expectLoadingOnRequest('login', () => login(page, 'ceshi', '123456'))

  await loadingTracker.expectLoadingOnRequest('goto /dashboard', () => gotoAndWait(page, '/dashboard'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/dashboard:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /website/list', () => gotoAndWait(page, '/website/list'))
  await loadingTracker.expectLoadingOnRequest('tab /website/list 默认设置', () => clickTab(page, '默认设置'))
  await loadingTracker.expectLoadingOnRequest('tab /website/list DNS API', () => clickTab(page, 'DNS API'))
  await loadingTracker.expectLoadingOnRequest('tab /website/list 解析检测', () => clickTab(page, '解析检测'))
  await loadingTracker.expectLoadingOnRequest('tab /website/list 网站列表', () => clickTab(page, '网站列表'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/website/list:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /website/certs', () => gotoAndWait(page, '/website/certs'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/website/certs:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /website/purge', () => gotoAndWait(page, '/website/purge'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/website/purge:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /website/rules', () => gotoAndWait(page, '/website/rules'))
  await loadingTracker.expectLoadingOnRequest('tab /website/rules ACL规则', () => clickTab(page, 'ACL规则'))
  await loadingTracker.expectLoadingOnRequest('tab /website/rules CC规则', () => clickTab(page, 'CC规则'))
  await loadingTracker.expectLoadingOnRequest('tab /website/rules 匹配器', () => clickTab(page, '匹配器'))
  await loadingTracker.expectLoadingOnRequest('tab /website/rules 过滤器', () => clickTab(page, '过滤器'))
  await loadingTracker.expectLoadingOnRequest('tab /website/rules 规则组', () => clickTab(page, '规则组'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/website/rules:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /website/monitor', () => gotoAndWait(page, '/website/monitor'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/website/monitor:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /website/logs/block', () => gotoAndWait(page, '/website/logs/block'))
  await loadingTracker.expectLoadingOnRequest('tab /website/logs/block 统计', () => clickTab(page, '统计'))
  await loadingTracker.expectLoadingOnRequest('tab /website/logs/block 历史记录', () => clickTab(page, '历史记录'))
  await loadingTracker.expectLoadingOnRequest('tab /website/logs/block 当前封禁', () => clickTab(page, '当前封禁'))

  await loadingTracker.expectLoadingOnRequest('goto /website/logs/access', () => gotoAndWait(page, '/website/logs/access'))
  await loadingTracker.expectLoadingOnRequest('tab /website/logs/access 申请记录', () => clickTab(page, '申请记录'))
  await loadingTracker.expectLoadingOnRequest('tab /website/logs/access 日志查询', () => clickTab(page, '日志查询'))

  await loadingTracker.expectLoadingOnRequest('goto /forward/list', () => gotoAndWait(page, '/forward/list'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/forward/list:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /forward/groups', () => gotoAndWait(page, '/forward/groups'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/forward/groups:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /forward/default', () => gotoAndWait(page, '/forward/default'))
  await loadingTracker.expectLoadingOnRequest('tab /forward/default 转发列表', () => clickTab(page, '转发列表'))
  await loadingTracker.expectLoadingOnRequest('tab /forward/default 默认设置', () => clickTab(page, '默认设置'))
  await loadingTracker.expectLoadingOnRequest('tab /forward/default 实时监控', () => clickTab(page, '实时监控'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/forward/default:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /forward/monitor', () => gotoAndWait(page, '/forward/monitor'))
  await loadingTracker.expectLoadingOnRequest('tab /forward/monitor 带宽流量', () => clickTab(page, '带宽流量'))
  await loadingTracker.expectLoadingOnRequest('tab /forward/monitor 端口排行', () => clickTab(page, '端口排行'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/forward/monitor:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /plans/my', () => gotoAndWait(page, '/plans/my'))
  await loadingTracker.expectLoadingOnRequest('goto /plans/usage', () => gotoAndWait(page, '/plans/usage'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/plans:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /account/profile', () => gotoAndWait(page, '/account/profile'))
  await loadingTracker.expectLoadingOnRequest('goto /account/recharge', () => gotoAndWait(page, '/account/recharge'))
  await loadingTracker.expectLoadingOnRequest('goto /account/bills', () => gotoAndWait(page, '/account/bills'))
  await loadingTracker.expectLoadingOnRequest('goto /account/logs', () => gotoAndWait(page, '/account/logs'))
  await loadingTracker.expectLoadingOnRequest('goto /account/messages', () => gotoAndWait(page, '/account/messages'))
  await loadingTracker.expectLoadingOnRequest('goto /account/subscribe', () => gotoAndWait(page, '/account/subscribe'))
  await loadingTracker.expectLoadingOnRequest('goto /account/apikey', () => gotoAndWait(page, '/account/apikey'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/account:${label}`, action)
  )

  loadingTracker.assertNoMissing()
  await guards.assertClean()
})
