import { test } from '@playwright/test'
import { attachGuards, clickTab, clickToolbarButtons, createLoadingTracker, gotoAndWait, login } from './_helpers'

test('admin: tab pages load and refresh', async ({ page }) => {
  test.setTimeout(600000)
  const guards = attachGuards(page)
  const loadingTracker = createLoadingTracker(page)

  await loadingTracker.expectLoadingOnRequest('login', () => login(page, 'admin', '123456'))

  await loadingTracker.expectLoadingOnRequest('goto /system/config', () => gotoAndWait(page, '/system/config'))
  await loadingTracker.expectLoadingOnRequest('tab /system/config 数据清理', () => clickTab(page, '数据清理'))
  await loadingTracker.expectLoadingOnRequest('tab /system/config 用户相关', () => clickTab(page, '用户相关'))
  await loadingTracker.expectLoadingOnRequest('tab /system/config 通知配置', () => clickTab(page, '通知配置'))
  await loadingTracker.expectLoadingOnRequest('tab /system/config 其它配置', () => clickTab(page, '其它配置'))
  await loadingTracker.expectLoadingOnRequest('tab /system/config 系统配置', () => clickTab(page, '系统配置'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/system/config:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /system/tasks', () => gotoAndWait(page, '/system/tasks'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/system/tasks:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /system/users', () => gotoAndWait(page, '/system/users'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/system/users:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /system/logs', () => gotoAndWait(page, '/system/logs'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/system/logs:${label}`, action)
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
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/website/logs/block:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /website/logs/access', () => gotoAndWait(page, '/website/logs/access'))
  await loadingTracker.expectLoadingOnRequest('tab /website/logs/access 申请记录', () => clickTab(page, '申请记录'))
  await loadingTracker.expectLoadingOnRequest('tab /website/logs/access 日志查询', () => clickTab(page, '日志查询'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/website/logs/access:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /global/firewall', () => gotoAndWait(page, '/global/firewall'))
  await loadingTracker.expectLoadingOnRequest('tab /global/firewall 安全控制', () => clickTab(page, '安全控制'))
  await loadingTracker.expectLoadingOnRequest('tab /global/firewall CC 防护', () => clickTab(page, 'CC 防护'))
  await loadingTracker.expectLoadingOnRequest('tab /global/firewall 高级防护', () => clickTab(page, '高级防护'))
  await loadingTracker.expectLoadingOnRequest('tab /global/firewall 基础防护', () => clickTab(page, '基础防护'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/global/firewall:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /global/resources', () => gotoAndWait(page, '/global/resources'))
  await loadingTracker.expectLoadingOnRequest('tab /global/resources 转发', () => clickTab(page, '转发'))
  await loadingTracker.expectLoadingOnRequest('tab /global/resources 公共', () => clickTab(page, '公共'))
  await loadingTracker.expectLoadingOnRequest('tab /global/resources 网站', () => clickTab(page, '网站'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/global/resources:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /global/default', () => gotoAndWait(page, '/global/default'))
  await loadingTracker.expectLoadingOnRequest('tab /global/default 缓存配置', () => clickTab(page, '缓存配置'))
  await loadingTracker.expectLoadingOnRequest('tab /global/default 全局配置', () => clickTab(page, '全局配置'))
  await loadingTracker.expectLoadingOnRequest('tab /global/default 网站', () => clickTab(page, '网站'))
  await loadingTracker.expectLoadingOnRequest('tab /global/default 转发', () => clickTab(page, '转发'))
  await loadingTracker.expectLoadingOnRequest('tab /global/default 证书', () => clickTab(page, '证书'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/global/default:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /node/list', () => gotoAndWait(page, '/node/list'))
  await loadingTracker.expectLoadingOnRequest('tab /node/list 区域管理', () => clickTab(page, '区域管理'))
  await loadingTracker.expectLoadingOnRequest('tab /node/list 节点列表', () => clickTab(page, '节点列表'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/node/list:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /node/dns', () => gotoAndWait(page, '/node/dns'))
  await loadingTracker.expectLoadingOnRequest('tab /node/dns CNAME 域名', () => clickTab(page, 'CNAME 域名'))
  await loadingTracker.expectLoadingOnRequest('tab /node/dns DNS配置', () => clickTab(page, 'DNS配置'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/node/dns:${label}`, action)
  )

  await loadingTracker.expectLoadingOnRequest('goto /node/realtime', () => gotoAndWait(page, '/node/realtime'))
  await loadingTracker.expectLoadingOnRequest('tab /node/realtime 监控指标', () => clickTab(page, '监控指标'))
  await loadingTracker.expectLoadingOnRequest('tab /node/realtime 节点流量', () => clickTab(page, '节点流量'))
  await loadingTracker.expectLoadingOnRequest('tab /node/realtime 资源排行', () => clickTab(page, '资源排行'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/node/realtime:${label}`, action)
  )

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

  await loadingTracker.expectLoadingOnRequest('goto /plans/basic', () => gotoAndWait(page, '/plans/basic'))
  await loadingTracker.expectLoadingOnRequest('goto /plans/sold', () => gotoAndWait(page, '/plans/sold'))
  await clickToolbarButtons(page, (label, action) =>
    loadingTracker.expectLoadingOnRequest(`/plans:${label}`, action)
  )

  loadingTracker.assertNoMissing()
  await guards.assertClean()
})
