import { expect, test } from '@playwright/test'
import { attachGuards, gotoAndWait } from './_helpers'

test('website list: batch clear cache creates task and shows monitor', async ({ page }) => {
  test.setTimeout(120_000)
  const guards = attachGuards(page)

  await page.addInitScript(() => {
    localStorage.setItem('admin_token', 'e2e-token')
    localStorage.setItem('role', 'admin')
  })

  await page.route(/\/api\/v1\/admin\/sites\?.+/, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: {
          total: 1,
          list: [
            {
              id: 1,
              domains: ['example.com'],
              domain_display: 'example.com',
              listen_ports: 'HTTP:80',
              origin_display: '1.1.1.1',
              cname: 'example.com.old-cname.test',
              https: false,
              user_package_id: 1,
              user_package_name: 'pkg',
              group_name: '-',
              node_group_name: '-',
              region_name: '-',
              status: true,
              created_at: new Date().toISOString()
            }
          ]
        }
      })
    })
  })

  await page.route('**/api/v1/admin/sites/batch_action', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    const data = route.request().postDataJSON()
    if (data?.action !== 'clear_cache') return route.fallback()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: { task_id: 123 } })
    })
  })

  let taskPoll = 0
  await page.route('**/api/v1/admin/tasks/123', async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    taskPoll += 1
    const state = taskPoll >= 2 ? 'done' : 'running'
    const progress = taskPoll >= 2 ? { wsl: 'done' } : { wsl: 'running' }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: {
          id: 123,
          name: '清空缓存',
          type: 'clear_cache',
          state,
          create_at: new Date().toISOString(),
          start_at: new Date().toISOString(),
          end_at: state === 'done' ? new Date().toISOString() : null,
          err_times: 0,
          progress: JSON.stringify(progress),
          ret: JSON.stringify([{ time: new Date().toISOString(), node_id: 'wsl', state, message: '', attempt: 0 }])
        }
      })
    })
  })

  await gotoAndWait(page, '/website/list')

  await expect(page.locator('.el-table__body-wrapper tbody tr')).toHaveCount(1)
  const firstRow = page.locator('.el-table__body-wrapper tbody tr').first()
  await firstRow.locator('.el-checkbox').first().click()

  const clearCacheReq = page.waitForRequest(
    (req) => req.url().includes('/api/v1/admin/sites/batch_action') && req.method() === 'POST'
  )

  await page.getByRole('button', { name: /更多操作/ }).click()
  await page.locator('.el-dropdown-menu__item').filter({ hasText: '清空缓存' }).click()

  const msgBox = page.locator('.el-message-box:visible')
  await expect(msgBox).toBeVisible()
  await msgBox.locator('button').filter({ hasText: '确定' }).click()

  const req = await clearCacheReq
  const posted = req.postDataJSON()
  expect(posted.action).toBe('clear_cache')
  expect(posted.ids).toEqual([1])

  const monitorDialog = page.locator('.el-dialog:visible').filter({ hasText: /清空缓存任务/ })
  await expect(monitorDialog).toBeVisible()
  await expect(monitorDialog.locator('.el-tag').filter({ hasText: '完成' }).first()).toBeVisible({ timeout: 10_000 })

  await guards.assertClean()
})

test('website list: batch CNAME domain uses dropdown and runs resolve check', async ({ page }) => {
  test.setTimeout(120_000)
  const guards = attachGuards(page)

  await page.addInitScript(() => {
    localStorage.setItem('admin_token', 'e2e-token')
    localStorage.setItem('role', 'admin')
  })

  let sitesCall = 0
  await page.route(/\/api\/v1\/admin\/sites\?.+/, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    sitesCall += 1
    const cname = sitesCall >= 2 ? 'example.com.new-cname.test' : 'example.com.old-cname.test'
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: {
          total: 1,
          list: [
            {
              id: 2,
              domains: ['example.com'],
              domain_display: 'example.com',
              listen_ports: 'HTTP:80',
              origin_display: '1.1.1.1',
              cname,
              https: false,
              user_package_id: 1,
              user_package_name: 'pkg',
              group_name: '-',
              node_group_name: '-',
              region_name: '-',
              status: true,
              created_at: new Date().toISOString()
            }
          ]
        }
      })
    })
  })

  await page.route('**/api/v1/admin/cname_domains**', async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: { list: [{ id: 1, domain: 'new-cname.test', note: '' }] }
      })
    })
  })

  await page.route('**/api/v1/admin/sites/batch_update', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    const body = route.request().postDataJSON()
    expect(body.ids).toEqual([2])
    expect(body.cname_domain).toBe('new-cname.test')
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ message: 'ok' }) })
  })

  await page.route(/\/api\/v1\/admin\/sites\/resolve\?.+/, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        domain: 'example.com',
        cname: 'example.com.new-cname.test',
        ips: ['127.0.0.1']
      })
    })
  })

  await gotoAndWait(page, '/website/list')

  await expect(page.locator('.el-table__body-wrapper tbody tr')).toHaveCount(1)
  const firstRow = page.locator('.el-table__body-wrapper tbody tr').first()
  await firstRow.locator('.el-checkbox').first().click()

  await page.getByRole('button', { name: /更多操作/ }).click()
  await page.locator('.el-dropdown-menu__item').filter({ hasText: 'CNAME域名' }).click()

  const editDialog = page.locator('.el-dialog:visible').filter({ hasText: '修改CNAME域名' })
  await expect(editDialog).toBeVisible()

  await expect(editDialog.locator('.el-select')).toBeVisible()
  await editDialog.locator('.el-select').click()
  await page.locator('.el-select-dropdown__item').filter({ hasText: 'new-cname.test' }).click()

  await editDialog.locator('button').filter({ hasText: '确定' }).click()

  const resolveDialog = page.locator('.el-dialog:visible').filter({ hasText: '解析检测结果' })
  await expect(resolveDialog).toBeVisible()
  await expect(resolveDialog.locator('.el-tag').filter({ hasText: '正常' })).toBeVisible()

  await guards.assertClean()
})
