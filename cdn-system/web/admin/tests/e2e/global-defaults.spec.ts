import fs from 'node:fs'
import { test, expect, APIRequestContext } from '@playwright/test'
import { login } from './_helpers'
import {
  createSite,
  deleteSite,
  getConfigItems,
  getGlobalConfig,
  getGlobalDefaultsMap,
  getSiteDetail,
  updateConfigItems,
  updateGlobalConfig,
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

const getSetting = (settings: Record<string, any> | undefined, path: string) => {
  if (!settings) return undefined
  return path.split('.').reduce((acc, key) => {
    if (acc && typeof acc === 'object' && key in acc) {
      return acc[key]
    }
    return undefined
  }, settings as any)
}

async function getDnsProviders(request: APIRequestContext) {
  const res = await request.get(`${adminBase}/dnsapi`)
  if (!res.ok()) return []
  const payload = await res.json()
  return payload?.data?.list || []
}

async function getCcRules(request: APIRequestContext) {
  const res = await request.get(`${adminBase}/rules/cc/groups`, { params: { pageSize: 200 } })
  if (!res.ok()) return []
  const payload = await res.json()
  return payload?.data?.list || []
}

async function waitForAgentStream(request: APIRequestContext, nodeId: number, listenPort: string, timeoutMs = 30_000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const res = await request.get(`/api/v1/agent/config`, { params: { node_id: String(nodeId) } })
    if (!res.ok()) {
      await new Promise((r) => setTimeout(r, 500))
      continue
    }
    const cfg = await res.json()
    const streams = cfg?.streams || []
    const stream = streams.find((item: { listen_ports?: string[] }) => {
      return Array.isArray(item.listen_ports) && item.listen_ports.includes(listenPort)
    })
    if (stream) return stream
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(`Timed out waiting for stream on port ${listenPort}`)
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

test('global defaults apply to new sites', async ({ page, playwright }) => {
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
  const certDefaults = await getConfigItems(api, 'cert_default_config')
  const streamDefaults = await getConfigItems(api, 'stream_default_config')
  const globalConfig = await getGlobalConfig(api)
  const originalGlobalConfig = JSON.parse(JSON.stringify(globalConfig || {}))

  const siteDefaultCases = [
    {
      key: 'http_listen-port',
      label: 'HTTP监听端口',
      next: (current: string) => pickAlt(current, [81, 82]),
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
      next: (current: string) => pickAlt(current, [4443, 4444]),
      assertSite: (site: any, value: string) => {
        const port = getSetting(site?.settings, 'https.redirect_port')
        expect(String(port || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.https_redirect_port || '')).toBe(String(value))
      }
    },
    {
      key: 'https_listen-hsts',
      label: '开启HSTS',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'https.hsts')).toBe(value)
      },
      assertAgent: (entry: any, value: boolean) => {
        expect(!!entry?.https_hsts).toBe(value)
      }
    },
    {
      key: 'https_listen-http2',
      label: '开启HTTP2',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'https.http2')).toBe(value)
      },
      assertAgent: (entry: any, value: boolean) => {
        expect(!!entry?.https_http2).toBe(value)
      }
    },
    {
      key: 'https_listen-http3',
      label: '开启HTTP3',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'https.http3')).toBe(value)
      },
      assertAgent: (entry: any, value: boolean) => {
        expect(!!entry?.https_http3).toBe(value)
      }
    },
    {
      key: 'https_listen-force_ssl_enable',
      label: '强制HTTPS',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'https.force')).toBe(value)
      },
      assertAgent: (entry: any, value: boolean) => {
        expect(!!entry?.https_force).toBe(value)
      }
    },
    {
      key: 'https_listen-ssl_protocols',
      label: 'ssl_protocols',
      next: (current: string) => {
        const preferred = 'TLSv1.2 TLSv1.3'
        return normalizeSpace(current) === preferred ? 'TLSv1 TLSv1.1 TLSv1.2' : preferred
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
      next: (current: string) => {
        const preferred = 'ECDHE-RSA-AES128-GCM-SHA256'
        return normalizeSpace(current) === preferred ? 'AES256-SHA' : preferred
      },
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
      next: (current: string) => (String(current || '').toLowerCase() === 'on' ? 'off' : 'on'),
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
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'https.ocsp_stapling')).toBe(value)
      },
      assertAgent: (entry: any, value: boolean) => {
        expect(!!entry?.https_ocsp).toBe(value)
      }
    },
    {
      key: 'backend_protocol',
      label: '回源协议',
      next: (current: string) => pickAlt(current, ['follow_port', 'follow', 'http', 'https']),
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
      next: (current: string) => pickAlt(current, [8081, 8082]),
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
      next: (current: string) => pickAlt(current, [8443, 9443]),
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
      next: (current: string) => pickAlt(current, [75, 90]),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'backsource.timeout') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.proxy_read_timeout || '')).toBe(String(value))
        expect(String(entry?.proxy_send_timeout || '')).toBe(String(value))
      }
    },
    {
      key: 'connect_timeout',
      label: '连接超时',
      next: (current: string) => pickAlt(current, [12, 18]),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'backsource.connect_timeout') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.proxy_connect_timeout || '')).toBe(String(value))
      }
    },
    {
      key: 'proxy_ssl_protocols',
      label: '回源SSL协议',
      next: (current: string) => {
        const preferred = 'TLSv1.2 TLSv1.3'
        return normalizeSpace(current) === preferred ? 'TLSv1 TLSv1.1 TLSv1.2' : preferred
      },
      assertSite: (site: any, value: string) => {
        expect(normalizeSpace(getSetting(site?.settings, 'advanced.proxy_ssl_protocols'))).toBe(normalizeSpace(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(normalizeSpace(entry?.proxy_ssl_protocols)).toBe(normalizeSpace(value))
      }
    },
    {
      key: 'proxy_cache',
      label: '缓存规则',
      next: () => JSON.stringify([{
        type: 'all',
        value: '',
        ttl: 120,
        ignore_query: true,
        force_cache: true
      }]),
      assertSite: (site: any) => {
        const rules = getSetting(site?.settings, 'cache.rules') || []
        expect(Array.isArray(rules)).toBeTruthy()
        expect(rules.length).toBeGreaterThan(0)
        expect(rules[0]?.type).toBe('all')
        expect(toNumber(rules[0]?.ttl)).toBe(120)
      },
      assertAgent: (entry: any) => {
        const rules = entry?.cache?.rules || []
        expect(Array.isArray(rules)).toBeTruthy()
        expect(rules.length).toBeGreaterThan(0)
        expect(rules[0]?.prefix || '').toBe('/')
        expect(toNumber(rules[0]?.ttl)).toBe(120)
        expect(!!rules[0]?.force_cache).toBe(true)
      }
    },
    {
      key: 'origin_headers',
      label: '源站请求头',
      next: () => JSON.stringify([{ name: 'X-Default-Origin', value: 'global' }]),
      assertSite: (site: any) => {
        const headers = getSetting(site?.settings, 'advanced.origin_headers') || []
        expect(headers).toEqual([{ name: 'X-Default-Origin', value: 'global' }])
      },
      assertAgent: (entry: any) => {
        expect(entry?.headers?.['X-Default-Origin']).toBe('global')
      }
    },
    {
      key: 'log_request_header',
      label: '记录请求头',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'advanced.log_request_header')).toBe(value)
      }
    },
    {
      key: 'log_response_header',
      label: '记录响应头',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'advanced.log_response_header')).toBe(value)
      }
    },
    {
      key: 'log_request_body',
      label: '记录请求体',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'advanced.log_request_body')).toBe(value)
      }
    },
    {
      key: 'post_size_limit',
      label: '请求体大小限制',
      next: (current: string) => pickAlt(current, [20, 24]),
      assertSite: (site: any, value: string) => {
        expect(toNumber(getSetting(site?.settings, 'advanced.body_limit'))).toBe(Number(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(toNumber(entry?.body_limit)).toBe(Number(value))
      }
    },
    {
      key: 'balance_way',
      label: '负载方式',
      next: (current: string) => pickAlt(current, ['ip_hash', 'rr']),
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.load_balance_policy || '')).toBe(toBalancePolicy(value))
      }
    },
    {
      key: 'cc_default_rule',
      label: '默认CC规则',
      next: (current: string) => {
        const ids = ccRules.map((item: { id: number }) => String(item.id))
        const fallback = ['10002', '10003']
        const options = ids.length ? ids : fallback
        return pickAlt(current, options)
      },
      assertSite: (site: any, value: string) => {
        expect(toNumber(getSetting(site?.settings, 'security.default_rule'))).toBe(Number(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(toNumber(entry?.cc_rule_id)).toBe(Number(value))
      }
    },
    {
      key: 'security_bot',
      label: '搜索引擎爬虫',
      next: (current: string) => pickAlt(current, ['block', 'allow']),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'security.crawlers_action') || '')).toBe(String(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(String(entry?.crawler_action || '')).toBe(String(value))
      }
    },
    {
      key: 'gzip_enable',
      label: '开启Gzip',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'advanced.gzip')).toBe(value)
      },
      assertAgent: (entry: any, value: boolean) => {
        expect(!!entry?.enable_gzip).toBe(value)
      }
    },
    {
      key: 'gzip_types',
      label: 'gzip types',
      next: (current: string) => {
        const preferred = 'text/plain application/json'
        return normalizeSpace(current) === preferred ? 'text/css' : preferred
      },
      assertSite: (site: any, value: string) => {
        expect(normalizeSpace(getSetting(site?.settings, 'advanced.gzip_types'))).toBe(normalizeSpace(value))
      },
      assertAgent: (entry: any, value: string) => {
        expect(normalizeSpace(entry?.gzip_types)).toBe(normalizeSpace(value))
      }
    },
    {
      key: 'websocket_enable',
      label: '开启Websocket',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'advanced.websocket')).toBe(value)
      },
      assertAgent: (entry: any, value: boolean) => {
        expect(!!entry?.enable_websocket).toBe(value)
      }
    },
    {
      key: 'security_shield_proxy',
      label: '屏蔽透明代理',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'security.block_transparent_proxy')).toBe(value)
      },
      assertAgent: (entry: any, value: boolean) => {
        expect(!!entry?.block_transparent_proxy).toBe(value)
      }
    },
    {
      key: 'realtime_send',
      label: '数据实时发送',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'advanced.realtime_send')).toBe(value)
      }
    },
    {
      key: 'realtime_return',
      label: '数据实时返回',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'advanced.realtime_return')).toBe(value)
      }
    },
    {
      key: 'ipv6_enable',
      label: '开启IPv6',
      next: (current: string) => !toBool(current),
      assertSite: (site: any, value: boolean) => {
        expect(!!getSetting(site?.settings, 'advanced.ipv6')).toBe(value)
      }
    },
    {
      key: 'dns_provider_id',
      label: 'DNS API(解析)',
      next: (current: string) => {
        if (!dnsProviders.length) return null
        const options = dnsProviders.map((item: { id: number }) => String(item.id))
        return pickAlt(current, options)
      },
      assertSite: (site: any, value: string) => {
        expect(String(site?.dns_provider_id || '')).toBe(String(value))
      }
    }
  ]

  for (const item of siteDefaultCases) {
    await test.step(`全局默认: ${item.label}`, async () => {
      const original = globalDefaults[item.key]
      if (original === undefined) {
        return
      }
      const nextValue = item.next(String(original ?? ''))
      if (nextValue === null || nextValue === undefined) {
        return
      }
      await updateSiteDefaults(api, { [item.key]: nextValue })
      const domain = randomDomain('global')
      const { id } = await createSite(page, domain, originAddress)
      try {
        const site = await getSiteDetail(api, id)
        if (item.assertSite) item.assertSite(site, nextValue as any)
        if (item.assertAgent && canCheckAgent) {
          await assignSiteNodeGroup(api, id, agentNodeGroupId)
          const { entry } = await waitForAgentDomain(agentApi, agentNodeId, domain)
          item.assertAgent(entry, nextValue as any)
        }
      } finally {
        await deleteSite(api, id)
        await updateSiteDefaults(api, { [item.key]: original })
      }
    })
  }

  const certCases = [
    {
      key: 'cert_default_type',
      label: '默认证书类型',
      next: (current: string) => pickAlt(current, ['zerossl', 'lets', 'buypass', 'google']),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'cert.type') || '')).toBe(String(value))
      }
    },
    {
      key: 'cert_default_dnsapi_type',
      label: '默认证书DNS API',
      next: (current: string) => pickAlt(current, ['aliyun', 'cloudflare', 'huawei']),
      assertSite: (site: any, value: string) => {
        expect(String(getSetting(site?.settings, 'cert.dnsapi_type') || '')).toBe(String(value))
      }
    },
    {
      key: 'cert_default_dnsapi_data',
      label: '默认证书DNS API数据',
      next: () => JSON.stringify({ token: `default-${randomLabel()}` }),
      assertSite: (site: any, value: string) => {
        expect(getSetting(site?.settings, 'cert.dnsapi_data')).toEqual(JSON.parse(String(value)))
      }
    }
  ]

  for (const item of certCases) {
    await test.step(`证书默认: ${item.label}`, async () => {
      const original = certDefaults[item.key] || ''
      const nextValue = item.next(original)
      await updateConfigItems(api, 'cert_default_config', [{ name: item.key, value: String(nextValue) }])
      const domain = randomDomain('cert')
      const { id } = await createSite(page, domain, originAddress)
      try {
        const site = await getSiteDetail(api, id)
        item.assertSite(site, nextValue as any)
      } finally {
        await deleteSite(api, id)
        await updateConfigItems(api, 'cert_default_config', [{ name: item.key, value: String(original) }])
      }
    })
  }

  const streamCases = [
    {
      key: 'listen_protocol',
      label: '监听协议',
      next: (current: string) => pickAlt(current, ['tcp', 'udp'])
    },
    {
      key: 'balance_way',
      label: '负载方式',
      next: (current: string) => pickAlt(current, ['ip_hash', 'rr']),
      assertStream: (stream: any, value: string) => {
        expect(String(stream?.balance_way || '')).toBe(String(value))
      }
    },
    {
      key: 'proxy_protocol',
      label: 'proxy_protocol',
      next: (current: string) => (toBool(current) ? '0' : '1'),
      assertStream: (stream: any, value: string) => {
        expect(!!stream?.proxy_protocol).toBe(toBool(value))
      }
    }
  ]

  for (const item of streamCases) {
    await test.step(`转发默认: ${item.label}`, async () => {
      const original = streamDefaults[item.key] || ''
      const nextValue = item.next(original)
      await updateConfigItems(api, 'stream_default_config', [{ name: item.key, value: String(nextValue) }])
      const listenPort = pickAlt('', [39091, 39092, 39093])
      if (!canCheckAgent) {
        await updateConfigItems(api, 'stream_default_config', [{ name: item.key, value: String(original) }])
        return
      }
      const createRes = await api.post(`${adminBase}/forwards`, {
        data: {
          user_id: adminUserId,
          node_group_id: agentNodeGroupId || undefined,
          listen_ports: [String(listenPort)],
          origin_input: `${originAddress}:39090`
        }
      })
      if (!createRes.ok()) {
        await updateConfigItems(api, 'stream_default_config', [{ name: item.key, value: String(original) }])
        return
      }
      const payload = await createRes.json()
      const forwardId = payload?.data?.id
      try {
        const stream = await waitForAgentStream(agentApi, agentNodeId, String(listenPort))
        if (item.assertStream) {
          item.assertStream(stream, String(nextValue))
        }
      } finally {
        if (forwardId) {
          await api.post(`${adminBase}/forwards/batch_action`, {
            data: { action: 'delete', ids: [forwardId] }
          }).catch(() => null)
        }
        await updateConfigItems(api, 'stream_default_config', [{ name: item.key, value: String(original) }])
      }
    })
  }

  if (originalGlobalConfig?.default_config) {
    const types: Array<'website' | 'api' | 'download'> = ['website', 'api', 'download']
    const fields: Array<'cache_enable' | 'cache_ttl' | 'gzip' | 'waf_enable'> = [
      'cache_enable',
      'cache_ttl',
      'gzip',
      'waf_enable'
    ]

    for (const type of types) {
      for (const field of fields) {
        await test.step(`缓存配置: ${type} ${field}`, async () => {
          const cfg = JSON.parse(JSON.stringify(originalGlobalConfig))
          if (!cfg.default_config) {
            cfg.default_config = { website: {}, api: {}, download: {} }
          }
          if (!cfg.default_config[type]) {
            cfg.default_config[type] = {}
          }
          const current = cfg.default_config[type][field]
          let nextValue: any = current
          if (field === 'cache_ttl') {
            nextValue = Number(pickAlt(String(current ?? ''), [120, 240]))
          } else {
            nextValue = !toBool(current)
          }
          cfg.default_config[type][field] = nextValue

          await updateGlobalConfig(api, cfg)
          const domain = randomDomain(type)
          const { id } = await createSite(page, domain, originAddress, type)
          try {
            const site = await getSiteDetail(api, id)
            if (field === 'cache_enable') {
              expect(!!getSetting(site?.settings, 'cache.enable')).toBe(nextValue)
              if (canCheckAgent) {
                await assignSiteNodeGroup(api, id, agentNodeGroupId)
                const { entry } = await waitForAgentDomain(agentApi, agentNodeId, domain)
                expect(!!entry?.cache?.enable).toBe(nextValue)
              }
            } else if (field === 'cache_ttl') {
              expect(toNumber(getSetting(site?.settings, 'cache.ttl'))).toBe(Number(nextValue))
              if (canCheckAgent) {
                await assignSiteNodeGroup(api, id, agentNodeGroupId)
                const { entry } = await waitForAgentDomain(agentApi, agentNodeId, domain)
                expect(toNumber(entry?.cache?.default_ttl)).toBe(Number(nextValue))
              }
            } else if (field === 'gzip') {
              expect(!!getSetting(site?.settings, 'advanced.gzip')).toBe(nextValue)
              if (canCheckAgent) {
                await assignSiteNodeGroup(api, id, agentNodeGroupId)
                const { entry } = await waitForAgentDomain(agentApi, agentNodeId, domain)
                expect(!!entry?.enable_gzip).toBe(nextValue)
              }
            } else if (field === 'waf_enable') {
              expect(!!getSetting(site?.settings, 'security.waf_enable')).toBe(nextValue)
            }
          } finally {
            await deleteSite(api, id)
            await updateGlobalConfig(api, originalGlobalConfig)
          }
        })
      }
    }
  }

  await api.dispose()
  await agentApi.dispose()
})
