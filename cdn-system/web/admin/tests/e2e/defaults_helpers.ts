import fs from 'node:fs'
import path from 'node:path'
import { APIRequestContext, Locator, Page, expect } from '@playwright/test'

const agentConfigPath = process.env.AGENT_CONFIG_PATH
  || '/www/server/go_project/openresty/cdn-system/agent/edge-node/conf/cdn_config.json'

const adminBase = '/api/v1/admin'

export type CreatedSite = { id: number; domain: string }

export async function gotoManage(page: Page, siteId: number) {
  await page.goto(`/website/manage?site_id=${siteId}`)
  await page.waitForSelector('.site-manage', { timeout: 30_000 })
}

export async function openManageTab(page: Page, label: string) {
  const tab = page.locator('.manage-tabs .el-tabs__item:visible').filter({ hasText: label }).first()
  if ((await tab.count()) === 0) {
    throw new Error(`Manage tab not found: ${label}`)
  }
  await tab.click()
  await page.waitForTimeout(200)
}

export async function createSite(page: Page, domain: string, origin: string, siteType = 'website'): Promise<CreatedSite> {
  await page.goto('/website/list')
  await page.waitForSelector('.filter-container', { timeout: 30_000 })
  await page.getByRole('button', { name: '添加网站' }).click()

  const dialog = page.locator('.el-dialog').filter({ hasText: '添加网站' })
  await dialog.waitFor({ state: 'visible', timeout: 30_000 })

  await fillDialogInput(dialog, '网站域名', domain)
  await fillDialogInput(dialog, '源站地址', origin)

  const siteTypeGroup = dialog.locator('.el-form-item__label', { hasText: '加速类型' }).locator('xpath=..')
  await siteTypeGroup.getByText(siteTypeLabel(siteType), { exact: true }).click()

  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes(`${adminBase}/sites`) && r.request().method() === 'POST',
      { timeout: 30_000 }
    ),
    dialog.getByRole('button', { name: '确定' }).click()
  ])
  const payload = await resp.json().catch(() => null)
  const siteId = payload?.data?.id
  if (!siteId) {
    throw new Error(`Failed to create site for ${domain}`)
  }
  return { id: siteId, domain }
}

export async function deleteSite(request: APIRequestContext, siteId: number) {
  if (!siteId) return
  await request.post(`${adminBase}/sites/batch_action`, {
    data: { action: 'delete', ids: [siteId] }
  }).catch(() => null)
}

export async function readAgentConfig() {
  const resolved = path.resolve(agentConfigPath)
  if (!fs.existsSync(resolved)) {
    throw new Error(`Agent config not found: ${resolved}`)
  }
  const raw = fs.readFileSync(resolved, 'utf-8')
  return JSON.parse(raw)
}

export async function waitForAgentDomain(domain: string, timeoutMs = 30_000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    try {
      const cfg = await readAgentConfig()
      const entry = (cfg.domains || []).find((item: { name: string }) => item.name === domain)
      if (entry) return { cfg, entry }
    } catch {
      // ignore transient read errors
    }
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(`Timed out waiting for domain in agent config: ${domain}`)
}

export async function waitForAgentPredicate(domain: string, predicate: (entry: any) => boolean, timeoutMs = 30_000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const { entry } = await waitForAgentDomain(domain, Math.min(2_000, timeoutMs))
    if (predicate(entry)) {
      return entry
    }
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(`Timed out waiting for agent predicate on ${domain}`)
}

export async function getGlobalDefaultsMap(request: APIRequestContext) {
  const res = await request.get(`${adminBase}/site_defaults`, {
    params: { scope_name: 'global', scope_id: 0 }
  })
  expect(res.ok()).toBeTruthy()
  const payload = await res.json()
  const list = payload?.data?.list || []
  const map: Record<string, string> = {}
  list.forEach((item: { name: string; value: string }) => {
    map[item.name] = item.value
  })
  return map
}

export async function getDefaultsList(request: APIRequestContext) {
  const res = await request.get(`${adminBase}/site_defaults`)
  expect(res.ok()).toBeTruthy()
  const payload = await res.json()
  return payload?.data?.list || []
}

export async function updateSiteDefaults(
  request: APIRequestContext,
  data: Record<string, any>,
  scopeName = 'global',
  scopeId = 0,
  userId?: number
) {
  const payload: Record<string, any> = {
    scope_name: scopeName,
    scope_id: scopeId,
    data
  }
  if (userId) payload.user_id = userId
  const res = await request.post(`${adminBase}/site_defaults`, { data: payload })
  expect(res.ok()).toBeTruthy()
}

export async function deleteDefault(request: APIRequestContext, name: string, scopeName: string, scopeId: number, userId?: number) {
  const params: Record<string, any> = { scope_name: scopeName, scope_id: scopeId }
  if (userId) params.user_id = userId
  await request.delete(`${adminBase}/site_defaults/${encodeURIComponent(name)}`, { params }).catch(() => null)
}

export async function getSiteDetail(request: APIRequestContext, siteId: number) {
  const res = await request.get(`${adminBase}/sites/${siteId}`)
  expect(res.ok()).toBeTruthy()
  const payload = await res.json()
  return payload?.data?.site || null
}

export async function getConfigItems(
  request: APIRequestContext,
  type: string,
  scopeName = 'global',
  scopeId = 0
) {
  const res = await request.get(`${adminBase}/config_items`, {
    params: { type, scope_name: scopeName, scope_id: scopeId }
  })
  expect(res.ok()).toBeTruthy()
  const payload = await res.json()
  const list = payload?.data?.list || payload?.list || []
  const map: Record<string, string> = {}
  list.forEach((item: { name: string; value: string }) => {
    map[item.name] = item.value
  })
  return map
}

export async function updateConfigItems(
  request: APIRequestContext,
  type: string,
  items: { name: string; value: string }[],
  scopeName = 'global',
  scopeId = 0
) {
  const res = await request.post(`${adminBase}/config_items`, {
    data: {
      type,
      scope_name: scopeName,
      scope_id: scopeId,
      items
    }
  })
  expect(res.ok()).toBeTruthy()
}

export async function getGlobalConfig(request: APIRequestContext) {
  const res = await request.get(`${adminBase}/global_config`)
  expect(res.ok()).toBeTruthy()
  const payload = await res.json()
  return payload?.data || null
}

export async function updateGlobalConfig(request: APIRequestContext, config: Record<string, any>) {
  const res = await request.post(`${adminBase}/global_config`, { data: config })
  expect(res.ok()).toBeTruthy()
}

function siteTypeLabel(type: string) {
  switch (type) {
    case 'api':
      return 'API加速'
    case 'download':
      return '下载加速'
    default:
      return '网页加速'
  }
}

async function fillDialogInput(dialog: Locator, label: string, value: string) {
  const item = dialog.locator('.el-form-item__label', { hasText: label }).first().locator('xpath=..')
  const input = item.locator('input, textarea').first()
  await input.fill(value)
  await input.blur()
}
