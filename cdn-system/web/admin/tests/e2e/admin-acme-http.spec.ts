import { expect, test } from '@playwright/test'
import { gotoAndWait, login } from './_helpers'

const adminUser = process.env.ADMIN_USER || 'admin'
const adminPass = process.env.ADMIN_PASS || '123456'
const siteDomain = process.env.ACME_DOMAIN || 'test.665305.cc'
const siteOrigin = process.env.ACME_ORIGIN || '127.0.0.1'
const certName = process.env.ACME_CERT_NAME || `acme-http-${siteDomain}`
const certUser = process.env.ACME_USER || adminUser

test('admin:申请 HTTP ACME 证书', async ({ page }) => {
  test.setTimeout(60_000)
  test.skip(process.env.ACME_E2E !== '1', 'ACME e2e requires ACME_E2E=1 with valid env')

  await login(page, adminUser, adminPass)
  await ensureSite(page, siteDomain, siteOrigin)
  await ensureHttpCert(page, siteDomain, certName)
})

async function ensureSite(page, domain, origin) {
  await gotoAndWait(page, '/website/list')

  const filter = page.locator('.filter-container').first()
  await selectOption(filter.locator('.el-select').first(), '域名')
  await filter.getByPlaceholder('输入关键字').fill(domain)
  await Promise.all([
    page.waitForResponse(resp => resp.url().includes('/sites') && resp.request().method() === 'GET'),
    filter.getByRole('button', { name: '查询' }).click()
  ])

  if (await hasRow(page, domain)) {
    return
  }

  await filter.getByRole('button', { name: '添加网站' }).click()
  const dialog = page.locator('.el-dialog__wrapper:visible').first()
  await fillOwnerIfNeeded(page, dialog)
  await dialog.getByPlaceholder('每行一个域名，支持泛域名如 *.example.com').fill(domain)
  await dialog.getByPlaceholder('每行一个，如 1.1.1.1 或 1.1.1.1:8080').fill(origin)

  const submit = dialog.getByRole('button', { name: '确定' })
  const saved = await Promise.all([
    page
      .waitForResponse(
        resp => resp.url().includes('/sites') && resp.request().method() === 'POST',
        { timeout: 15000 }
      )
      .then(() => true)
      .catch(() => false),
    submit.click()
  ]).then(([ok]) => ok)
  if (!saved) {
    const errors = await dialog.locator('.el-form-item__error:visible').allInnerTexts()
    const message = await page.locator('.el-message--error:visible').allInnerTexts()
    throw new Error(`站点提交失败: ${[...errors, ...message].join(' | ') || '未触发提交'}`)
  }

  await page.locator('.el-dialog__wrapper:visible').waitFor({ state: 'hidden', timeout: 10_000 })
  await filter.getByPlaceholder('输入关键字').fill(domain)
  await Promise.all([
    page.waitForResponse(resp => resp.url().includes('/sites') && resp.request().method() === 'GET'),
    filter.getByRole('button', { name: '查询' }).click()
  ])

  expect(await hasRow(page, domain)).toBeTruthy()
}

async function fillOwnerIfNeeded(page, dialog) {
  const ownerItem = dialog.locator('.el-form-item').filter({ hasText: '所属用户' }).first()
  if ((await ownerItem.count()) === 0) {
    return
  }
  const input = ownerItem.locator('input.el-input__inner').first()
  await input.click()
  await input.fill(adminUser)
  await input.press('Enter').catch(() => null)
  await page.waitForResponse(resp => resp.url().includes('/users') && resp.request().method() === 'GET').catch(() => null)
  const dropdown = page.locator('.el-select-dropdown:visible').first()
  let option = dropdown.locator('.el-select-dropdown__item').first()
  if ((await option.count()) === 0) {
    await input.click()
    await input.fill(adminUser)
    await page.waitForResponse(resp => resp.url().includes('/users') && resp.request().method() === 'GET').catch(() => null)
    option = dropdown.locator('.el-select-dropdown__item').first()
  }
  if (await option.count()) {
    await option.click()
  }
  const selected = (await input.inputValue()).trim()
  if (!selected) {
    throw new Error('未能选择所属用户')
  }
}

async function ensureHttpCert(page, domain, name) {
  await gotoAndWait(page, '/website/certs')

  const filter = page.locator('.filter-container').first()
  await filter.getByPlaceholder('输入名称/域名, 模糊搜索').fill(domain)
  await Promise.all([
    page.waitForResponse(resp => resp.url().includes('/certs') && resp.request().method() === 'GET'),
    filter.getByRole('button', { name: '搜索' }).click()
  ])

  if (await hasRow(page, domain)) {
    await selectRow(page, domain)
    await Promise.all([
      page.waitForResponse(resp => resp.url().includes('/certs/reissue') && resp.request().method() === 'POST'),
      filter.getByRole('button', { name: '重新申请' }).click()
    ])
  } else {
    await filter.getByRole('button', { name: '添加证书' }).click()
    const dialog = page.locator('.el-dialog__wrapper:visible').first()
    await fillCertUserIfNeeded(page, dialog)
    await dialog.getByPlaceholder('输入证书名称').fill(name)
    await dialog.locator('.el-radio').filter({ hasText: "Let's Encrypt" }).click()
    await dialog.getByPlaceholder('输入域名, 多个域名空格分隔').fill(domain)
    await selectOption(dialog.locator('.el-select').filter({ hasText: 'HTTP验证' }).first(), '不选择 (HTTP验证)')
    await Promise.all([
      page.waitForResponse(resp => resp.url().includes('/certs') && resp.request().method() === 'POST'),
      dialog.getByRole('button', { name: '确定' }).click()
    ])
  }

  await waitForCertSuccess(page, domain)
}

async function selectOption(selectLocator, label) {
  await selectLocator.click()
  const dropdown = selectLocator.page().locator('.el-select-dropdown:visible').first()
  await dropdown.getByRole('option', { name: label }).click()
}

async function hasRow(page, keyword) {
  const rows = page.locator('.el-table__body-wrapper .el-table__row')
  return (await rows.filter({ hasText: keyword }).count()) > 0
}

async function selectRow(page, keyword) {
  const row = page.locator('.el-table__body-wrapper .el-table__row').filter({ hasText: keyword }).first()
  await row.locator('.el-checkbox').first().click()
}

async function fillCertUserIfNeeded(page, dialog) {
  const userInput = dialog.getByPlaceholder('搜索用户ID或账号')
  if ((await userInput.count()) === 0) {
    return
  }
  await userInput.click()
  await userInput.fill(certUser)
  await userInput.press('Enter').catch(() => null)
  await page.waitForResponse(resp => resp.url().includes('/users') && resp.request().method() === 'GET').catch(() => null)
  const dropdown = page.locator('.el-select-dropdown:visible').first()
  const option = dropdown.locator('.el-select-dropdown__item').first()
  if (await option.count()) {
    await option.click()
  }
}

async function waitForCertSuccess(page, domain) {
  const deadline = Date.now() + 8 * 60_000
  const filter = page.locator('.filter-container').first()
  const searchInput = filter.getByPlaceholder('输入名称/域名, 模糊搜索')
  const searchButton = filter.getByRole('button', { name: '搜索' })

  while (Date.now() < deadline) {
    await searchInput.fill(domain)
    await Promise.all([
      page.waitForResponse(resp => resp.url().includes('/certs') && resp.request().method() === 'GET').catch(() => null),
      searchButton.click()
    ])

    const row = page.locator('.el-table__body-wrapper .el-table__row').filter({ hasText: domain }).first()
    if (await row.count()) {
      const rowText = (await row.innerText()).replace(/\s+/g, ' ')
      if (rowText.includes('失败')) {
        throw new Error(`证书签发失败: ${rowText}`)
      }
      if (rowText.includes('已签发')) {
        return
      }
    }

    await page.waitForTimeout(5000)
  }

  throw new Error('证书状态未在预期时间内变为已签发')
}
