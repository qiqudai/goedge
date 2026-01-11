import { expect, Page } from '@playwright/test'

export type ApiError = { url: string; status: number; method?: string }

export function attachGuards(page: Page) {
  const apiErrors: ApiError[] = []
  const consoleErrors: string[] = []

  page.on('pageerror', (err) => {
    consoleErrors.push(err.message)
  })
  page.on('console', (msg) => {
    if (msg.type() !== 'error') return
    const text = msg.text()
    if (text.includes('Unable to preventDefault inside passive event listener invocation.')) {
      return
    }
    consoleErrors.push(text)
  })
  page.on('response', (resp) => {
    const url = resp.url()
    if (!url.includes('/api/v1/')) return
    const status = resp.status()
    if (status >= 400) {
      apiErrors.push({ url, status, method: resp.request().method() })
    }
  })

  return {
    apiErrors,
    consoleErrors,
    assertClean: async () => {
      expect(consoleErrors, `Console errors:\n${consoleErrors.join('\n')}`).toEqual([])
      expect(apiErrors, `API errors:\n${apiErrors.map((e) => `${e.status} ${e.method || ''} ${e.url}`).join('\n')}`).toEqual([])
    }
  }
}

export async function login(page: Page, username: string, password: string) {
  await page.goto('/login')
  await page.getByPlaceholder('Username').fill(username)
  await page.getByPlaceholder('Password').fill(password)
  await page.getByRole('button', { name: 'Login' }).click()
  await page.waitForURL('**/dashboard')
}

export async function gotoAndWait(page: Page, path: string) {
  await page.goto(path)
  await page.waitForLoadState('domcontentloaded')
  // Let Vue render and requests settle.
  await page.waitForTimeout(150)
}

export async function clickTab(page: Page, label: string) {
  const tab = page.locator('.el-tabs__item:visible').filter({ hasText: label }).first()
  if ((await tab.count()) === 0) return
  await tab.click()
  await page.waitForTimeout(250)
}

export function createLoadingTracker(page: Page) {
  const missing: string[] = []

  const expectLoadingOnRequest = async (label: string, action: () => Promise<void>) => {
    const requestPromise = page
      .waitForRequest(req => req.url().includes('/api/v1/'), { timeout: 30_000 })
      .catch(() => null)

    await action()

    const req = await requestPromise
    if (!req) return

    try {
      await page.locator('.global-loading-overlay').waitFor({ state: 'visible', timeout: 30_000 })
    } catch {
      missing.push(label)
    }
    await page.locator('.global-loading-overlay').waitFor({ state: 'hidden', timeout: 30_000 }).catch(() => null)
  }

  const assertNoMissing = () => {
    expect(missing, `Missing global loading for: ${missing.join(', ')}`).toEqual([])
  }

  return { missing, expectLoadingOnRequest, assertNoMissing }
}

export async function clickToolbarButtons(page: Page, clicker: (label: string, action: () => Promise<void>) => Promise<void>) {
  const selectors = [
    '.app-container .filter-container .el-button',
    '.app-container .toolbar .el-button',
    '.app-container .filter-row .el-button'
  ]
  const allowLabel = /查询|搜索|刷新|筛选|统计|查看|重试/i
  const skipLabel = /删除|移除|退出|注销|关闭|取消|导出|下载|重置|清空|清除|禁用|停用|解绑|批量|新增|添加|创建|上传|保存|提交|确定|确认|登录/i

  for (const selector of selectors) {
    const buttons = page.locator(`${selector}:visible`)
    const count = await buttons.count()
    for (let i = 0; i < count; i += 1) {
      const button = buttons.nth(i)
      let enabled = false
      try {
        enabled = await button.isEnabled()
      } catch {
        enabled = false
      }
      if (!enabled) continue
      const label = (await button.innerText()).trim()
      if (!label || skipLabel.test(label) || !allowLabel.test(label)) {
        continue
      }
      const safeLabel = label || `button-${selector}-${i}`
      await clicker(safeLabel, async () => {
        await dismissDialogs(page)
        await page.locator('.global-loading-overlay').waitFor({ state: 'hidden', timeout: 800 }).catch(() => null)
        await button.click()
        await page.waitForTimeout(80)
      })
      await dismissDialogs(page)
      await page.locator('.global-loading-overlay').waitFor({ state: 'hidden', timeout: 800 }).catch(() => null)
    }
  }
}

async function dismissDialogs(page: Page) {
  await page.keyboard.press('Escape').catch(() => null)

  const messageBox = page.locator('.el-message-box__wrapper:visible')
  if (await messageBox.count()) {
    const cancel = messageBox.locator('button').filter({ hasText: /取消|关闭|Cancel/i }).first()
    if (await cancel.count()) {
      await cancel.click()
      return
    }
    const close = messageBox.locator('.el-message-box__headerbtn').first()
    if (await close.count()) {
      await close.click()
    }
  }

  const dialog = page.locator('.el-dialog__wrapper:visible')
  if (await dialog.count()) {
    const cancel = dialog.locator('button').filter({ hasText: /取消|关闭|Cancel/i }).first()
    if (await cancel.count()) {
      await cancel.click()
      return
    }
    const close = dialog.locator('.el-dialog__headerbtn').first()
    if (await close.count()) {
      await close.click()
    }
  }

  const overlay = page.locator('.el-overlay:visible')
  if (await overlay.count()) {
    await overlay.first().click({ position: { x: 5, y: 5 } }).catch(() => null)
  }
}
