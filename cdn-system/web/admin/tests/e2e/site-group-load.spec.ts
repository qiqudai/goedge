import { test, expect } from '@playwright/test'

const adminUser = process.env.PW_ADMIN_USER || 'admin'
const adminPass = process.env.PW_ADMIN_PASS || '123456'

const setupNoCache = async (page) => {
  await page.setExtraHTTPHeaders({
    'Cache-Control': 'no-cache',
    Pragma: 'no-cache'
  })
}

test('website list: admin add site loads groups after user select', async ({ page }) => {
  test.setTimeout(60000)
  await setupNoCache(page)

  const authRes = await page.request.post('https://goai.665305.cc/api/v1/admin/login', {
    data: { username: adminUser, password: adminPass }
  })
  const authData = await authRes.json()
  const token = authData?.token || ''
  await page.addInitScript((value) => {
    localStorage.setItem('admin_token', value)
    localStorage.setItem('role', 'admin')
  }, token)

  await page.goto('/website/list', { waitUntil: 'domcontentloaded' })
  await page.getByRole('button', { name: '添加网站' }).click()

  const dialog = page.locator('.el-dialog').first()
  await dialog.waitFor({ state: 'visible' })

  const expand = page.getByText('展开更多')
  if (await expand.isVisible()) {
    await expand.click()
  }

  const userRow = dialog.locator('.el-form-item').filter({ hasText: '所属用户' }).first()
  const userInput = userRow.locator('input').first()
  await userInput.click()

  const userDropdownId = await userInput.getAttribute('aria-controls')
  const userOptions = page.locator(`#${userDropdownId} .el-select-dropdown__item`)
  await expect(userOptions.first()).toBeVisible()

  const waitGroups = page.waitForResponse((res) => {
    return res.url().includes('/api/v1/admin/site_groups') && res.request().method() === 'GET'
  }, { timeout: 15000 })

  await userOptions.first().click()

  const groupRes = await waitGroups
  const body = await groupRes.json()
  const list = body?.data?.list || body?.list || []

  const groupSelect = dialog.locator('.el-form-item').filter({ hasText: '网站分组' }).locator('.el-select__wrapper').first()
  await groupSelect.click()
  const groupInput = dialog.locator('.el-form-item').filter({ hasText: '网站分组' }).locator('input').first()
  const groupDropdownId = await groupInput.getAttribute('aria-controls')
  const groupOptions = page.locator(`#${groupDropdownId} .el-select-dropdown__item`)
  const optionCount = await groupOptions.count()

  expect(Array.isArray(list)).toBeTruthy()
  expect(optionCount).toBe(list.length)
})
