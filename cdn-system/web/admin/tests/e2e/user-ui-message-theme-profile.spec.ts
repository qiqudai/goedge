import { expect, test } from '@playwright/test'
import { gotoAndWait, login } from './_helpers'

const apiBase = process.env.API_BASE || 'http://127.0.0.1:8080'

test.describe('user: ui message badge, theme, profile labels', () => {
  test.setTimeout(180_000)

  test('badge count, detail dot, theme toggle, password labels', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('theme', 'light')
    })

    await page.route('**/api/v1/user/messages/unread**', (route) => {
      if (route.request().method() !== 'GET') return route.fallback()
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            count: 3,
            latest: { id: 99, title: '流量超限' }
          }
        })
      })
    })

    await page.route('**/api/v1/user/messages**', (route) => {
      if (route.request().url().includes('/messages/unread')) return route.fallback()
      if (route.request().method() !== 'GET') return route.fallback()
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            total: 1,
            list: [
              {
                id: 99,
                type: 'traffic-exceed',
                type_label: '流量超限',
                title: '流量超限',
                content: '测试内容',
                phone: '',
                site_id: 37,
                created_at: '2025-01-01 00:00:00',
                is_read: false
              }
            ]
          }
        })
      })
    })

    await page.route('**/api/v1/user/messages/99/read', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 200))
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, msg: 'ok' }) })
    })

    const unreadRequest = page.waitForRequest('**/api/v1/user/messages/unread**').catch(() => null)
    const unreadResponse = page.waitForResponse('**/api/v1/user/messages/unread**').catch(() => null)

    if (process.env.E2E_USER_TOKEN) {
      await page.addInitScript(({ token, base }) => {
        localStorage.setItem('admin_token', token)
        localStorage.setItem('role', 'user')
        localStorage.setItem('api_base', base)
      }, { token: process.env.E2E_USER_TOKEN, base: apiBase })
      await gotoAndWait(page, '/dashboard')
    } else {
      await login(page, 'ceshi', '123456')
    }

    await unreadRequest
    const unreadResp = await unreadResponse
    if (unreadResp) {
      const rawText = await unreadResp.text()
      let body: any = {}
      try {
        body = JSON.parse(rawText)
      } catch {
        body = {}
      }
      expect(unreadResp.url()).toContain('/api/v1/user/messages/unread')
      expect(body.data?.count, `unread response: ${rawText}`).toBe(3)
    }

    const roleValue = await page.evaluate(() => localStorage.getItem('role'))
    await page.waitForTimeout(300)
    const badgeCount = await page.locator('.message-count').count()
    const badgeHtml = await page.locator('.message-badge').evaluate((el) => el.innerHTML)
    expect(badgeCount).toBe(1)
    const badgeValue = await page.locator('.message-count').first().innerText()
    expect(badgeValue.trim()).toBe('3')

    const themeSwitch = page.locator('.theme-switch')
    await themeSwitch.click()
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')

    await gotoAndWait(page, '/account/messages')
    await page.getByRole('button', { name: '详情' }).click()
    const dot = page.locator('.detail-unread-dot')
    await expect(dot).toBeVisible()
    await expect(dot).toBeHidden({ timeout: 5000 })

    const detailDialog = page.locator('.el-dialog:visible').filter({ hasText: '消息详情' })
    await detailDialog.locator('.el-dialog__footer button').filter({ hasText: '关闭' }).click()

    await gotoAndWait(page, '/account/profile')
    await page.getByRole('button', { name: '修改' }).click()
    const dialog = page.locator('.el-dialog:visible').first()
    const labels = dialog.locator('.el-form-item__label')
    await expect(labels).toContainText(['旧密码', '新密码', '确认新密码'])
  })
})
