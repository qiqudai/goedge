import { APIRequestContext, expect, request, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

const adminUser = process.env.PW_ADMIN_USER || 'admin'
const adminPass = process.env.PW_ADMIN_PASS || '123456'
const apiBase = process.env.API_BASE || 'http://127.0.0.1:8080'

async function waitForResolve(api: APIRequestContext, domain: string, timeoutMs = 30_000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const res = await api.get('/api/v1/user/sites/resolve', { params: { domain } })
    if (res.ok()) {
      const body = await res.json()
      const cname = String(body?.cname || '').trim()
      const ips = Array.isArray(body?.ips) ? body.ips : []
      if (cname || ips.length > 0) {
        return { cname, ips }
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }
  throw new Error(`Timed out resolving ${domain}`)
}

test.describe('user: dns cname sync', () => {
  test.skip(process.env.E2E_DNS !== '1', 'DNS sync e2e requires E2E_DNS=1 with configured CNAME domains')

  test('site-based and package-based cname resolve', async () => {
    const { token } = await loginUser('ceshi', '123456')
    const userApi = await createUserApiContext(token)
    const adminLogin = await request.newContext({ baseURL: apiBase })
    let adminApi: APIRequestContext | null = null

    const createdIds: number[] = []
    try {
      const adminRes = await adminLogin.post('/api/v1/admin/login', {
        data: { username: adminUser, password: adminPass }
      })
      expect(adminRes.ok(), 'admin login').toBeTruthy()
      const adminBody = await adminRes.json()
      const adminToken = String(adminBody?.token || '')
      expect(adminToken, 'admin token').toBeTruthy()

      adminApi = await request.newContext({
        baseURL: apiBase,
        extraHTTPHeaders: { Authorization: `Bearer ${adminToken}` }
      })

      const cnameRes = await adminApi.get('/api/v1/admin/cname_domains')
      const cnameBody = await expectApiSuccess(cnameRes)
      const cnameList = cnameBody.data?.list || []
      const cnameDomain = cnameList.find((item: { domain?: string; dns_provider_id?: number }) => item?.dns_provider_id)?.domain
      expect(cnameDomain, 'cname domain with dns provider').toBeTruthy()

      const packagesBody = await expectApiSuccess(
        await userApi.get('/api/v1/user/user_packages', { params: { pageSize: 50 } })
      )
      const packages = packagesBody.data?.list || packagesBody.list || []
      expect(packages.length, 'user packages').toBeGreaterThan(0)
      const pkg = packages.find((item: { cname_hostname?: string }) => item?.cname_hostname) || packages[0]
      const pkgHost = String(pkg?.cname_hostname || '')
      expect(pkgHost, 'package cname hostname').toBeTruthy()
      const pkgDomain = String(pkg?.cname_domain || '')

      const stamp = Date.now()
      const customDomain = `autodns-custom-${stamp}.example.com`
      const packageDomain = `autodns-package-${stamp}.example.com`

      const customCreate = await expectApiSuccess(
        await userApi.post('/api/v1/user/sites', {
          data: { domains: [customDomain], backends: ['1.1.1.1'], user_package_id: pkg.id }
        })
      )
      const customSiteId = Number(customCreate.data?.id || customCreate.data?.site_id || 0)
      expect(customSiteId, 'custom site id').toBeTruthy()
      createdIds.push(customSiteId)

      const packageCreate = await expectApiSuccess(
        await userApi.post('/api/v1/user/sites', {
          data: { domains: [packageDomain], backends: ['1.1.1.1'], user_package_id: pkg.id }
        })
      )
      const packageSiteId = Number(packageCreate.data?.id || packageCreate.data?.site_id || 0)
      expect(packageSiteId, 'package site id').toBeTruthy()
      createdIds.push(packageSiteId)

      await expectApiSuccess(
        await userApi.post('/api/v1/user/sites/batch_update', {
          data: { ids: [customSiteId], cname_mode: 'custom', cname_domain: cnameDomain }
        })
      )
      await expectApiSuccess(
        await userApi.post('/api/v1/user/sites/batch_update', {
          data: { ids: [packageSiteId], cname_mode: 'package' }
        })
      )

      await expectApiSuccess(await adminApi.post('/api/v1/admin/dns/records/fix'))

      const customSiteRes = await expectApiSuccess(await userApi.get(`/api/v1/user/sites/${customSiteId}`))
      const customCname = String(customSiteRes.data?.site?.cname || '').trim()
      expect(customCname, 'custom cname').toBe(`${customDomain}.${cnameDomain}`)

      const packageSiteRes = await expectApiSuccess(await userApi.get(`/api/v1/user/sites/${packageSiteId}`))
      const packageCname = String(packageSiteRes.data?.site?.cname || '').trim()
      const expectedPkgDomain = pkgDomain || String(cnameDomain)
      expect(packageCname, 'package cname').toContain(pkgHost)
      expect(packageCname.endsWith(`.${expectedPkgDomain}`), 'package cname domain').toBeTruthy()

      await waitForResolve(userApi, customCname)
      await waitForResolve(userApi, packageCname)

      await expectApiSuccess(
        await userApi.post('/api/v1/user/sites/batch_action', {
          data: { action: 'delete', ids: [customSiteId, packageSiteId] }
        })
      )

    } finally {
      if (adminApi) {
        await adminApi.dispose()
      }
      if (createdIds.length > 0) {
        await userApi
          .post('/api/v1/user/sites/batch_action', { data: { action: 'delete', ids: createdIds } })
          .catch(() => null)
      }
      await userApi.dispose()
      await adminLogin.dispose()
    }
  })
})
