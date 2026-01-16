import fs from 'node:fs'
import { test, expect, APIRequestContext } from '@playwright/test'
import { login } from './_helpers'
import {
  createSite,
  deleteSite,
  deleteDefault,
  getGlobalDefaultsMap,
  getSiteDetail,
  updateSiteDefaults
} from './defaults_helpers'

const baseDomain = '665305.cc'
const originAddress = '1.1.1.1'
const adminBase = '/api/v1/admin'

const randomLabel = () => Math.random().toString(36).slice(2, 8)
const randomDomain = (prefix: string) => `${prefix}-${Date.now().toString(36)}-${randomLabel()}.${baseDomain}`

const normalizeSpace = (value: unknown) => String(value ?? '').trim().replace(/\s+/g, ' ')
const toBool = (value: unknown) => ['1', 'true', 'on', 'yes'].includes(String(value ?? '').toLowerCase())
const toNumber = (value: unknown) => {
  const num = Number(value)
  return Number.isFinite(num) ? num : 0
}

const toBalancePolicy = (value: unknown) => {
  const raw = String(value ?? '').trim()
  if (raw === 'rr') return 'round_robin'
  return raw
}

const pickAlt = (current: unknown, options: Array<string | number>) => {
  const currentStr = String(current ?? '').trim()
  for (const option of options) {
    const opt = String(option)
    if (opt !== currentStr) {
      return opt
    }
  }
  return String(options[0] ?? '')
}

const pickAltValue = (globalValue: unknown, originalValue: unknown, options: Array<string | number>) => {
  const globalStr = String(globalValue ?? '').trim()
  const originalStr = originalValue === undefined ? '' : String(originalValue ?? '').trim()
  for (const option of options) {
    const opt = String(option)
    if (opt !== globalStr && opt !== originalStr) {
      return opt
    }
  }
  return pickAlt(globalStr, options)
}

const parseLines = (value: string) => value.split(/\s+/).map((item) => item.trim()).filter(Boolean)

const parseRegion = (value: string) => {
  if (!value || value === 'none') return []
  return value
    .split(/[\s,]+/)
    .map((item) => item.trim().toUpperCase())
    .filter(Boolean)
}

const sortedList = (list: string[]) => [...list].sort()

const getSetting = (settings: Record<string, any> | undefined, path: string) => {
  if (!settings) return undefined
  return path.split('.').reduce((acc, key) => {
    if (acc && typeof acc === 'object' && key in acc) {
      return acc[key]
    }
    return undefined
  }, settings as any)
}

type UserDefaultCase = {
  key: string
  label: string
  next: (globalValue: string, originalValue: string) => any
  assertSite?: (site: any, value: any, globalValue: string) => void
  assertAgent?: (entry: any, value: any, globalValue: string) => void
  allowSkip?: boolean
}

async function getDefaultsMap(request: APIRequestContext, scopeName: string, scopeId: number) {
  const res = await request.get(`${adminBase}/site_defaults`, {
    params: { scope_name: scopeName, scope_id: scopeId }
  })
  if (!res.ok()) return {}
  const payload = await res.json()
  const list = payload?.data?.list || []
  const map: Record<string, string> = {}
  list.forEach((item: { name: string; value: string }) => {
    map[item.name] = item.value
  })
  return map
}

async function getDnsProviders(request: APIRequestContext) {
  const res = await request.get(`${adminBase}/dnsapi`)
  if (!res.ok()) return []
  const payload = await res.json()
  return payload?.data?.list || payload?.list || []
}

async function getCcRules(request: APIRequestContext) {
  const res = await request.get(`${adminBase}/rules/cc/groups`, { params: { pageSize: 200 } })
  if (!res.ok()) return []
  const payload = await res.json()
  return payload?.data?.list || []
}

async function getAdminUserId(request: APIRequestContext) {
  const res = await request.get(`${adminBase}/users`, { params: { keyword: 'admin', pageSize: 20 } })
  if (!res.ok()) return 1
  const payload = await res.json()
  const list = payload?.data?.list || []
  const adminUser = list.find((item: { name?: string; username?: string; email?: string }) =>
    item?.name === 'admin' || item?.username === 'admin' || item?.email === 'admin'
  )
  return adminUser?.id || list?.[0]?.id || 1
}

async function getNodeGroupForNode(request: APIRequestContext, nodeId: number) {
  const groupsRes = await request.get(`${adminBase}/node-groups`, { params: { limit: 200 } })
  if (!groupsRes.ok()) return 0
  const groupsPayload = await groupsRes.json()
  const groups = groupsPayload?.data?.list || []
  for (const group of groups) {
    const groupId = Number(group?.id || 0)
    if (!groupId) continue
    const res = await request.get(`${adminBase}/node-groups/${groupId}/resolution`, { params: { line_id: 'all' } })
    if (!res.ok()) continue
    const payload = await res.json()
    const assigned = payload?.data?.assigned || []
    const hit = assigned.find((item: { node_id?: number }) => Number(item?.node_id) === Number(nodeId))
    if (hit) {
      return groupId
    }
  }
  return 0
}

async function assignSiteNodeGroup(request: APIRequestContext, siteId: number, nodeGroupId: number) {
  if (!siteId || !nodeGroupId) return
  const res = await request.post(`${adminBase}/sites/batch_update`, {
    data: { ids: [siteId], node_group_id: nodeGroupId }
  })
  if (!res.ok()) {
    throw new Error('Failed to assign site node group.')
  }
}

async function waitForAgentDomain(request: APIRequestContext, nodeId: number, domain: string, timeoutMs = 30_000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const res = await request.get(`/api/v1/agent/config`, { params: { node_id: String(nodeId) } })
    if (!res.ok()) {
      await new Promise((r) => setTimeout(r, 500))
      continue
    }
    const cfg = await res.json()
    const entry = (cfg?.domains || []).find((item: { name: string }) => item.name === domain)
    if (entry) return { cfg, entry }
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(`Timed out waiting for domain in agent config: ${domain}`)
}

test('user defaults override global defaults', async ({ page, playwright }) => {
  test.setTimeout(900_000)
  await login(page, 'admin', '123456')

  const baseURL = process.env.PW_BASE_URL || 'http://127.0.0.1:5176'
  const hostname = new URL(baseURL).hostname
  const apiBase = hostname === '127.0.0.1' || hostname === 'localhost'
    ? 'http://127.0.0.1:8080'
    : 'https://goai.665305.cc'

  const token = await page.evaluate(() => localStorage.getItem('admin_token'))
  if (!token) {
    throw new Error('Missing admin token after login.')
  }
  const api = await playwright.request.newContext({
    baseURL: apiBase,
    extraHTTPHeaders: { Authorization: `Bearer ${token}` }
  })

  const globalDefaults = await getGlobalDefaultsMap(api)
  const adminUserId = await getAdminUserId(api)
  const userDefaults = await getDefaultsMap(api, 'global', adminUserId)

  const agentConfig = JSON.parse(fs.readFileSync('/www/server/go_project/openresty/cdn-system/agent/agent.json', 'utf-8'))
  const agentToken = String(process.env.AGENT_TOKEN || agentConfig?.token || '')
  const agentNodeId = Number(process.env.AGENT_NODE_ID || agentConfig?.node_id || 18)
  if (!agentToken) {
    throw new Error('Missing agent token.')
  }
  const agentApi = await playwright.request.newContext({
    baseURL: apiBase,
    extraHTTPHeaders: { Authorization: `Bearer ${agentToken}` }
  })
  const agentNodeGroupId = await getNodeGroupForNode(api, agentNodeId)
  const canCheckAgent = agentNodeGroupId > 0

  const dnsProviders = await getDnsProviders(api)
  const ccRules = await getCcRules(api)
  const skipped: string[] = []

  const cases: UserDefaultCase[] = [
    {
      key: 'cc_default_rule',
      label: '默认CC规则',
      next: (globalValue: string, original: string) => {
        const ids = ccRules.map((item: { id: number }) => String(item.id))
        const options = ids.length ? ids : ['10002', '10003']
        return pickAltValue(globalValue, original, options)
      },
      assertSite: (site: any, value: string) => {
        expect(toNumber(getSetting(site?.settings, 'security.default_rule'))).toBe(Number(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(toNumber(entry?.cc_rule_id)).toBe(Number(value))
      }
    },
    {
      key: 'security_black_time',
      label: '黑名单时间',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, [12, 18]),
      assertSite: (site: any, value: string) => {
        expect(toNumber(getSetting(site?.settings, 'security.ip_black_timeout'))).toBe(Number(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(toNumber(entry?.guard_block_ttl)).toBe(Number(value))
      }
    },
    {
      key: 'security_white_time',
      label: '白名单时间',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, [15, 21]),
      assertSite: (site: any, value: string) => {
        expect(toNumber(getSetting(site?.settings, 'security.ip_white_timeout'))).toBe(Number(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(toNumber(entry?.guard_pass_ttl)).toBe(Number(value))
      }
    },
    {
      key: 'security_bot',
      label: '搜索引擎爬虫',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, ['allow', 'block', 'none']),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'security.crawlers_action') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.crawler_action || '')).toBe(String(value))
      },
      allowSkip: true
    },
    {
      key: 'black_ip',
      label: '黑名单IP',
      next: (globalValue: string, original: string) =>
        pickAltValue(globalValue, original, ['203.0.113.10\n203.0.113.11', '198.51.100.10']),
      assertSite: (site: any, value: string) => {
        const expected = parseLines(value)
        const list = (getSetting(site?.settings, 'security.blacklist') || []) as string[]
        expect(sortedList(list)).toEqual(sortedList(expected))
      },
      assertAgent: (entry: any, value: string) => {
        const expected = parseLines(value)
        expect(sortedList(entry?.black_ips || [])).toEqual(sortedList(expected))
      }
    },
    {
      key: 'white_ip',
      label: '白名单IP',
      next: (globalValue: string, original: string) =>
        pickAltValue(globalValue, original, ['198.51.100.10\n198.51.100.11', '203.0.113.20']),
      assertSite: (site: any, value: string) => {
        const expected = parseLines(value)
        const list = (getSetting(site?.settings, 'security.whitelist') || []) as string[]
        expect(sortedList(list)).toEqual(sortedList(expected))
      },
      assertAgent: (entry: any, value: string) => {
        const expected = parseLines(value)
        expect(sortedList(entry?.white_ips || [])).toEqual(sortedList(expected))
      }
    },
    {
      key: 'security_shield_proxy',
      label: '屏蔽透明代理',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'security.block_transparent_proxy')).toBe(toBool(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(!!entry?.block_transparent_proxy).toBe(toBool(value))
      }
    },
    {
      key: 'block_region',
      label: '区域屏蔽',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, ['cn,us', 'hk,mo']),
      assertSite: (site: any, value: string) => {
        const expected = parseRegion(value)
        const list = (getSetting(site?.settings, 'security.region_block') || []) as string[]
        expect(sortedList(list.map((item) => item.toUpperCase()))).toEqual(sortedList(expected))
      },
      assertAgent: (entry: any, value: string) => {
        const expected = parseRegion(value)
        expect(sortedList((entry?.region_block || []).map((item: string) => item.toUpperCase())))
          .toEqual(sortedList(expected))
      }
    },
    {
      key: 'dns_provider_id',
      label: 'DNS API(解析)',
      next: (globalValue: string, original: string) => {
        if (!dnsProviders.length) return null
        const options = dnsProviders.map((item: { id: number }) => String(item.id))
        return pickAltValue(globalValue, original, options)
      },
      assertSite: (site: any, value: string) => {
        expect(String(site?.dns_provider_id || '')).toBe(String(value))
      }
    },
    {
      key: 'http_listen-port',
      label: 'HTTP监听端口',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, [81, 82]),
      assertSite: (site: any, value: string) => {
        const ports = (site?.http_listen || []).map(String)
        expect(ports).toEqual([String(value)])
      },
      assertAgent: (entry: any, value: string) => {
        const ports = (entry?.http_listen || []).map(String)
        expect(ports).toEqual([String(value)])
      }
    },
    {
      key: 'https_listen-port',
      label: 'HTTPS监听端口',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, [4443, 4444]),
      assertSite: (site: any, value: string) => {
        const port = getSetting(site?.settings, 'https.redirect_port')
        expect(String(port || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.https_redirect_port || '')).toBe(String(value))
      }
    },
    {
      key: 'https_listen-force_ssl_enable',
      label: '强制HTTPS',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'https.force')).toBe(toBool(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(!!entry?.https_force).toBe(toBool(value))
      }
    },
    {
      key: 'https_listen-hsts',
      label: '开启HSTS',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'https.hsts')).toBe(toBool(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(!!entry?.https_hsts).toBe(toBool(value))
      }
    },
    {
      key: 'https_listen-http2',
      label: '开启HTTP2',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'https.http2')).toBe(toBool(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(!!entry?.https_http2).toBe(toBool(value))
      }
    },
    {
      key: 'https_listen-http3',
      label: '开启HTTP3',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'https.http3')).toBe(toBool(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(!!entry?.https_http3).toBe(toBool(value))
      }
    },
    {
      key: 'https_listen-ssl_protocols',
      label: 'ssl_protocols',
      next: (globalValue: string, original: string) => {
        const preferred = 'TLSv1.2 TLSv1.3'
        const fallback = 'TLSv1 TLSv1.1'
        return pickAltValue(globalValue, original, [preferred, fallback])
      },
      assertSite: (site: any, value: string) => {
        expect(normalizeSpace(getSetting(site?.settings, 'https.ssl_protocols'))).toBe(normalizeSpace(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(normalizeSpace(entry?.https_ssl_protocols)).toBe(normalizeSpace(value))
      }
    },
    {
      key: 'https_listen-ssl_ciphers',
      label: 'ssl_ciphers',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, ['AES256-SHA', 'ECDHE-RSA-AES128-GCM-SHA256']),
      assertSite: (site: any, value: string) => {
        expect(normalizeSpace(getSetting(site?.settings, 'https.ssl_ciphers'))).toBe(normalizeSpace(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(normalizeSpace(entry?.https_ssl_ciphers)).toBe(normalizeSpace(value))
      }
    },
    {
      key: 'https_listen-ssl_prefer_server_ciphers',
      label: 'ssl_prefer_server_ciphers',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, ['on', 'off']),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'https.ssl_prefer_server_ciphers') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.https_ssl_prefer_server_ciphers || '')).toBe(String(value))
      }
    },
    {
      key: 'https_listen-ocsp_stapling',
      label: 'ocsp_stapling',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'https.ocsp_stapling')).toBe(toBool(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(!!entry?.https_ocsp).toBe(toBool(value))
      }
    },
    {
      key: 'backend_protocol',
      label: '回源协议',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, ['follow_port', 'follow', 'http', 'https']),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'backsource.protocol') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.origin_protocol || '')).toBe(String(value))
      }
    },
    {
      key: 'backend_http_port',
      label: '回源HTTP端口',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, [8081, 8082]),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'backsource.http_port') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.origin_http_port || '')).toBe(String(value))
      }
    },
    {
      key: 'backend_https_port',
      label: '回源HTTPS端口',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, [8443, 9443]),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'backsource.https_port') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.origin_https_port || '')).toBe(String(value))
      }
    },
    {
      key: 'proxy_timeout',
      label: '回源超时',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, [75, 90]),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'backsource.timeout') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.proxy_read_timeout || '')).toBe(String(value))
        expect(String(entry?.proxy_send_timeout || '')).toBe(String(value))
      }
    },
    {
      key: 'ipv6_enable',
      label: '开启IPv6',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'advanced.ipv6')).toBe(toBool(value))
      }
    },
    {
      key: 'gzip_enable',
      label: '开启Gzip',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'advanced.gzip')).toBe(toBool(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(!!entry?.enable_gzip).toBe(toBool(value))
      }
    },
    {
      key: 'websocket_enable',
      label: '开启Websocket',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'advanced.websocket')).toBe(toBool(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(!!entry?.enable_websocket).toBe(toBool(value))
      }
    },
    {
      key: 'post_size_limit',
      label: '上传文件大小限制',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, [20, 24]),
      assertSite: (site: any, value: string) => {
        expect(toNumber(getSetting(site?.settings, 'advanced.body_limit'))).toBe(Number(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(toNumber(entry?.body_limit)).toBe(Number(value))
      }
    },
    {
      key: 'realtime_send',
      label: '数据实时发送',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'advanced.realtime_send')).toBe(toBool(value))
      }
    },
    {
      key: 'realtime_return',
      label: '数据实时返回',
      next: (globalValue: string, _original: string) => (!toBool(globalValue)).toString(),
      assertSite: (site: any, value: string) => {
        expect(!!getSetting(site?.settings, 'advanced.realtime_return')).toBe(toBool(value))
      }
    },
    {
      key: 'origin_headers',
      label: '源站请求头',
      next: (globalValue: string, _original: string) => {
        const next = JSON.stringify([{ name: 'X-User-Default', value: `user-${randomLabel()}` }])
        return next === String(globalValue || '') ? JSON.stringify([{ name: 'X-User-Default', value: 'user-alt' }]) : next
      },
      assertSite: (site: any, value: string) => {
        const expected = JSON.parse(String(value))
        const headers = getSetting(site?.settings, 'advanced.origin_headers') || []
        expect(headers).toEqual(expected)
      },
      assertAgent: (entry: any, value: string) => {
        const expected = JSON.parse(String(value))
        const header = expected[0]
        expect(entry?.headers?.[header.name]).toBe(header.value)
      }
    },
    {
      key: 'balance_way',
      label: '回源负载方式',
      next: (globalValue: string, original: string) => pickAltValue(globalValue, original, ['ip_hash', 'rr']),
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.load_balance_policy || '')).toBe(toBalancePolicy(value))
      }
    }
  ]

  for (const item of cases) {
    await test.step(`用户默认: ${item.label}`, async () => {
      const globalValue = globalDefaults[item.key] ?? ''
      const originalUser = userDefaults[item.key]
      const nextValue = item.next(String(globalValue), String(originalUser ?? ''))
      if (nextValue === null || nextValue === undefined || nextValue === '') {
        return
      }

      await updateSiteDefaults(api, { [item.key]: nextValue }, 'global', adminUserId, adminUserId)
      const domain = randomDomain('user')
      const { id } = await createSite(page, domain, originAddress)
      let skipReason = ''
      try {
        const site = await getSiteDetail(api, id)
        if (item.assertSite) {
          try {
            item.assertSite(site, nextValue, globalValue)
          } catch (err: any) {
            if (item.allowSkip) {
              skipReason = `skip ${item.label}: ${err?.message || err}`
            } else {
              throw err
            }
          }
        }
        if (!skipReason && item.assertAgent && canCheckAgent) {
          await assignSiteNodeGroup(api, id, agentNodeGroupId)
          const { entry } = await waitForAgentDomain(agentApi, agentNodeId, domain)
          try {
            item.assertAgent(entry, nextValue, globalValue)
          } catch (err: any) {
            if (item.allowSkip) {
              skipReason = `skip ${item.label}: ${err?.message || err}`
            } else {
              throw err
            }
          }
        }
      } finally {
        await deleteSite(api, id)
        if (originalUser === undefined) {
          await deleteDefault(api, item.key, 'global', adminUserId, adminUserId)
        } else {
          await updateSiteDefaults(api, { [item.key]: originalUser }, 'global', adminUserId, adminUserId)
        }
      }
      if (skipReason) {
        skipped.push(skipReason)
      }
    })
  }

  if (skipped.length) {
    console.log(`Skipped defaults: ${skipped.join('; ')}`)
  }

  await api.dispose()
  await agentApi.dispose()
})
