import fs from 'node:fs'
import path from 'node:path'
import { APIRequestContext, Locator, Page, expect, request, test } from '@playwright/test'

const agentConfigPath = process.env.AGENT_CONFIG_PATH
  || '/www/server/go_project/openresty/cdn-system/agent/edge-node/conf/cdn_config.json'

const adminBase = '/api/v1/admin'

export type CreatedSite = { id: number; domain: string }

export async function loginAdminToken(apiBase: string, username = 'admin', password = '123456') {
  const ctx = await request.newContext({ baseURL: apiBase })
  try {
    const res = await ctx.post('/api/v1/admin/login', { data: { username, password } })
    const body = await res.json().catch(() => null)
    const data = body?.data || body
    const token = String(data?.token || '')
    if (!token) {
      throw new Error('Missing admin token')
    }
    return token
  } finally {
    await ctx.dispose()
  }
}

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
  const addButton = page.locator('.filter-container .el-button').first()
  await addButton.click()

  const dialog = page.locator('.el-dialog:visible').first()
  await dialog.waitFor({ state: 'visible', timeout: 30_000 })

  const domainInput = dialog.locator('input[placeholder*="example.com"], textarea[placeholder*="example.com"]').first()
  await domainInput.fill(domain)
  await domainInput.blur()

  const originInput = dialog.locator('input[placeholder*="1.1.1.1"], textarea[placeholder*="1.1.1.1"]').first()
  await originInput.fill(origin)
  await originInput.blur()

  const radioInput = dialog.locator(`input[type="radio"][value="${siteType}"]`).first()
  if (await radioInput.count()) {
    await radioInput.check()
  } else if (siteType === 'api') {
    const apiRadio = dialog.locator('.el-radio').filter({ hasText: /API/i }).first()
    if (await apiRadio.count()) {
      await apiRadio.click()
    }
  }

  const confirmButton = dialog.locator('.el-dialog__footer .el-button--primary').first()
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes(`${adminBase}/sites`) && r.request().method() === 'POST',
      { timeout: 30_000 }
    ),
    confirmButton.click()
  ])
  const payload = await resp.json().catch(() => null)
  const siteId = payload?.data?.id
  if (!siteId) {
    throw new Error(`Failed to create site for ${domain}`)
  }
  return { id: siteId, domain }
}


export async function createSiteByApi(
  request: APIRequestContext,
  domain: string,
  origin: string,
  siteType = 'website',
  userId?: number
): Promise<CreatedSite> {
  const pickUserId = async () => {
    const userRes = await request.get(`${adminBase}/users`, { params: { pageSize: 50 } })
    if (!userRes.ok()) return 0
    const userBody = await userRes.json().catch(() => null)
    const list = userBody?.data?.list || []
    const preferred = list.find((item: any) =>
      item?.name === 'admin' || item?.username === 'admin' || item?.email === 'admin'
    )
    return Number(preferred?.id || list?.[0]?.id || 0)
  }

  const pickPackageId = async (uid: number) => {
    if (!uid) return 0
    const pkgRes = await request.get(`${adminBase}/user_packages`, { params: { user_id: uid, pageSize: 20 } })
    if (!pkgRes.ok()) return 0
    const pkgBody = await pkgRes.json().catch(() => null)
    const pkgs = pkgBody?.data?.list || []
    return Number(pkgs?.[0]?.id || 0)
  }

  const buildPayload = (uid?: number, pkgId?: number) => {
    const payload: Record<string, any> = {
      domains: [domain],
      backends: [origin],
      site_type: siteType
    }
    if (uid) {
      payload.user_id = uid
    }
    if (pkgId) {
      payload.user_package_id = pkgId
    }
    return payload
  }

  const extractMessage = (body: any) => body?.message || body?.msg || body?.error || ''
  const isSuccessCode = (code: any) => code === 0 || code === 200

  const createOnce = async (payload: Record<string, any>) => {
    const res = await request.post(`${adminBase}/sites`, { data: payload })
    expect(res.ok()).toBeTruthy()
    const body = await res.json().catch(() => null)
    const code = body?.code
    const data = body?.data || body
    return { body, code, data }
  }

  const attempt = await createOnce(buildPayload(userId))
  if (attempt.data?.id && (attempt.code === undefined || isSuccessCode(attempt.code))) {
    return { id: attempt.data.id, domain }
  }

  const msg = extractMessage(attempt.body)
  const lowerMsg = String(msg || '').toLowerCase()
  const needUserId =
    lowerMsg.includes('user_id') ||
    lowerMsg.includes('userid') ||
    msg.includes('用户ID')

  let resolvedUserId = Number(userId || 0)
  if (!resolvedUserId && needUserId) {
    resolvedUserId = await pickUserId()
    if (resolvedUserId) {
      const retry = await createOnce(buildPayload(resolvedUserId))
      if (retry.data?.id && (retry.code === undefined || isSuccessCode(retry.code))) {
        return { id: retry.data.id, domain }
      }
    }
  }

  const finalMsg = extractMessage(attempt.body)
  const finalLowerMsg = String(finalMsg || '').toLowerCase()
  const needPkg = finalLowerMsg.includes('user_package') || finalLowerMsg.includes('package')
  if (needPkg) {
    if (!resolvedUserId) {
      resolvedUserId = await pickUserId()
    }
    if (resolvedUserId) {
      const pkgId = await pickPackageId(resolvedUserId)
      if (pkgId) {
        const retry = await createOnce(buildPayload(resolvedUserId, pkgId))
        if (retry.data?.id && (retry.code === undefined || isSuccessCode(retry.code))) {
          return { id: retry.data.id, domain }
        }
        const retryMsg = extractMessage(retry.body)
        throw new Error(`Failed to create site for ${domain}: ${retryMsg || retry.code || 'unknown'}`)
      }
    }
  }

  if (finalMsg && finalLowerMsg.includes('domain limit')) {
    test.skip(true, `domain limit reached: ${finalMsg}`)
  }

  throw new Error(`Failed to create site for ${domain}: ${finalMsg || attempt.code || 'unknown'}`)
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
  return ''
}

async function fillDialogInput(dialog: Locator, label: string, value: string) {
  const item = dialog.locator('.el-form-item__label', { hasText: label }).first().locator('xpath=..')
  const input = item.locator('input, textarea').first()
  await input.fill(value)
  await input.blur()
}
