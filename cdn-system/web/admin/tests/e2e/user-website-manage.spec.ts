import fs from 'node:fs'
import path from 'node:path'
import { APIRequestContext, expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'
import { attachGuards } from './_helpers'
import { gotoManage, openManageTab } from './defaults_helpers'

test.describe.serial('user: website manage', () => {
  test.setTimeout(180_000)

  let token = ''
  let api: APIRequestContext
  let siteId = 0
  let certId = 0
  let certName = ''

  const certPath = process.env.E2E_CERT_PEM || '/www/server/go_project/openresty/cdn-system/agent/edge-node/cert/fallback.pem'
  const keyPath = process.env.E2E_CERT_KEY || '/www/server/go_project/openresty/cdn-system/agent/edge-node/cert/fallback.key'

  test.beforeAll(async () => {
    const login = await loginUser('ceshi', '123456')
    token = login.token
    api = await createUserApiContext(token)

    const packagesBody = await expectApiSuccess(await api.get('/api/v1/user/user_packages', { params: { pageSize: 10 } }))
    const packages = packagesBody.data?.list || packagesBody.list || []
    expect(packages.length).toBeGreaterThan(0)
    const pkg = packages.find((item: { domain?: number }) => Number(item?.domain || 0) === 0) || packages[0]
    const pkgId = pkg.id

    const domain = `autotest-manage-${Date.now()}.example.com`
    const createBody = await expectApiSuccess(
      await api.post('/api/v1/user/sites', { data: { domains: [domain], backends: ['1.1.1.1'], user_package_id: pkgId } })
    )
    siteId = createBody.data?.id || 0
    expect(siteId).toBeTruthy()

    const certPem = fs.readFileSync(path.resolve(certPath), 'utf-8')
    const keyPem = fs.readFileSync(path.resolve(keyPath), 'utf-8')
    const certBody = await expectApiSuccess(
      await api.post('/api/v1/user/certs', {
        data: { name: `autotest-cert-${Date.now()}`, type: 'upload', cert: certPem, key: keyPem }
      })
    )
    certId = certBody.data?.id || 0
    certName = certBody.data?.name || ''
  })

  test.afterAll(async () => {
    if (certId) {
      await api.delete(`/api/v1/user/certs/${certId}`).catch(() => null)
    }
    if (siteId) {
      await api.post('/api/v1/user/sites/batch_action', { data: { action: 'delete', ids: [siteId] } }).catch(() => null)
    }
    await api.dispose()
  })

  async function loginWithToken(page: any) {
    await page.addInitScript((value: string) => {
      localStorage.setItem('admin_token', value)
      localStorage.setItem('role', 'user')
    }, token)
  }

  async function waitForSiteUpdate(page: any) {
    const resp = await page.waitForResponse(
      (r: any) => r.url().includes(`/api/v1/user/sites/${siteId}`) && r.request().method() === 'PUT',
      { timeout: 30_000 }
    )
    expect(resp.ok()).toBeTruthy()
  }

  test('basic config: status toggle', async ({ page }) => {
    await loginWithToken(page)
    const guards = attachGuards(page)
    await gotoManage(page, siteId)
    await openManageTab(page, '基本配置')

    const statusItem = page
      .locator('.basic-config .el-form-item')
      .filter({ has: page.locator('.el-form-item__label', { hasText: '状态' }) })
    const statusSwitch = statusItem.locator('.el-switch')

    await Promise.all([waitForSiteUpdate(page), statusSwitch.click()])
    await Promise.all([waitForSiteUpdate(page), statusSwitch.click()])

    await guards.assertClean()
  })

  test('origin config: protocol change', async ({ page }) => {
    await loginWithToken(page)
    const guards = attachGuards(page)
    await gotoManage(page, siteId)
    await openManageTab(page, '回源设置')

    const httpRadio = page.locator('.origin-config .el-radio').filter({ hasText: /^HTTP$/ })
    const followRadio = page.locator('.origin-config .el-radio').filter({ hasText: /^跟随协议$/ })

    await Promise.all([waitForSiteUpdate(page), httpRadio.click()])
    await Promise.all([waitForSiteUpdate(page), followRadio.click()])

    await guards.assertClean()
  })

  test('https config: select cert and toggle', async ({ page }) => {
    await loginWithToken(page)
    const guards = attachGuards(page)
    await gotoManage(page, siteId)
    await openManageTab(page, 'HTTPS配置')

    const certSelect = page
      .locator('.https-config .el-form-item')
      .filter({ hasText: '证书选择' })
      .locator('.el-select')
      .first()

    await certSelect.click()
    const certOption = page.locator('.el-select-dropdown__item').filter({ hasText: certName }).first()
    await Promise.all([waitForSiteUpdate(page), certOption.click()])

    const toggle = page.locator('.https-config .el-form-item .el-switch').first()

    await Promise.all([waitForSiteUpdate(page), toggle.click()])
    await Promise.all([waitForSiteUpdate(page), toggle.click()])

    await guards.assertClean()
  })

  test('security config: auto switch', async ({ page }) => {
    await loginWithToken(page)
    const guards = attachGuards(page)
    await gotoManage(page, siteId)
    await openManageTab(page, '安全设置')

    const autoItem = page
      .locator('.security-config .el-form-item')
      .filter({ has: page.locator('.el-form-item__label', { hasText: '自动切换' }) })
    const autoSwitch = autoItem.locator('.el-switch')

    await Promise.all([waitForSiteUpdate(page), autoSwitch.click()])
    await Promise.all([waitForSiteUpdate(page), autoSwitch.click()])

    await guards.assertClean()
  })

  test('cache config: quick preset add/remove', async ({ page }) => {
    await loginWithToken(page)
    const guards = attachGuards(page)
    await gotoManage(page, siteId)
    await openManageTab(page, '缓存设置')

    const rowLocator = page.locator('.cache-config .el-table__body-wrapper tbody tr.el-table__row')
    const beforeCount = await rowLocator.count()

    const presetSelect = page.locator('.cache-config .el-select').first()
    await presetSelect.click()
    const option = page.locator('.el-select-dropdown__item').filter({ hasText: '全站缓存' }).first()
    await Promise.all([waitForSiteUpdate(page), option.click()])

    await page.waitForTimeout(300)
    const afterCount = await rowLocator.count()
    expect(afterCount).toBe(beforeCount + 1)

    const targetRow = rowLocator.nth(afterCount - 1)
    const deleteBtn = targetRow.locator('button').filter({ hasText: '删除' }).first()
    await Promise.all([waitForSiteUpdate(page), deleteBtn.click()])

    await page.waitForTimeout(300)
    const finalCount = await rowLocator.count()
    expect(finalCount).toBe(beforeCount)

    await guards.assertClean()
  })

  test('access config: hotlink toggle', async ({ page }) => {
    await loginWithToken(page)
    const guards = attachGuards(page)
    await gotoManage(page, siteId)
    await openManageTab(page, '访问控制')

    const hotlinkSwitch = page.locator('.access-config .el-switch').first()

    await Promise.all([waitForSiteUpdate(page), hotlinkSwitch.click()])
    await Promise.all([waitForSiteUpdate(page), hotlinkSwitch.click()])

    await guards.assertClean()
  })

  test('advanced config: gzip toggle', async ({ page }) => {
    await loginWithToken(page)
    const guards = attachGuards(page)
    await gotoManage(page, siteId)
    await openManageTab(page, '高级设置')

    const gzipItem = page
      .locator('.advanced-config .el-form-item')
      .filter({ has: page.locator('.el-form-item__label', { hasText: 'Gzip压缩' }) })
    const gzipSwitch = gzipItem.locator('.el-switch')

    await Promise.all([waitForSiteUpdate(page), gzipSwitch.click()])
    await Promise.all([waitForSiteUpdate(page), gzipSwitch.click()])

    await guards.assertClean()
  })
})
