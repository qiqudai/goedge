import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

const wildcardDomain = process.env.ACME_WILDCARD_DOMAIN || process.env.ACME_DOMAIN || '*.355686.cc'
const origin = process.env.ACME_ORIGIN || '127.0.0.1'
const dnsType = (process.env.ACME_DNS_TYPE || 'dnsla').toLowerCase()
const waitMinutes = Number(process.env.ACME_DNS_WAIT_MIN || '35')

test('user: wildcard cert via dns api', async () => {
  test.setTimeout(waitMinutes * 60_000 + 120_000)
  test.skip(process.env.ACME_DNS_E2E !== '1', 'ACME_DNS_E2E=1 required')

  const { token } = await loginUser('ceshi', '123456')
  const api = await createUserApiContext(token)

  await ensureDnsapiDefault(api)
  const pkgId = await ensureUserPackage(api)
  const siteId = await ensureSite(api, wildcardDomain, origin, pkgId)

  const existingCert = await findCert(api, wildcardDomain)
  if (existingCert?.id) {
    await expectApiSuccess(await api.post('/api/v1/user/certs/reissue', { data: { ids: [existingCert.id] } }))
  } else {
    await expectApiSuccess(await api.post('/api/v1/user/sites/apply_cert', { data: { ids: [siteId] } }))
  }

  await waitForCertReady(api, wildcardDomain, waitMinutes)

  await api.dispose()
})

async function ensureDnsapiDefault(api: any) {
  const dnsBody = await expectApiSuccess(await api.get('/api/v1/user/dnsapi', { params: { pageSize: 1000 } }))
  const dnsList = dnsBody.data?.list || dnsBody.list || []
  expect(dnsList.length).toBeGreaterThan(0)

  const selected = dnsList.find((item: any) => normalizeDnsType(item.type) === normalizeDnsType(dnsType)) || dnsList[0]
  expect(selected?.id, 'dnsapi id').toBeTruthy()

  const defaultsBody = await expectApiSuccess(await api.get('/api/v1/user/certs/default_settings'))
  const defaults = defaultsBody.data || {}
  const current = Number(defaults.dnsapi || 0)

  if (current !== Number(selected.id)) {
    await expectApiSuccess(
      await api.post('/api/v1/user/certs/default_settings', {
        data: { dnsapi: Number(selected.id), type: defaults.type || 'system' }
      })
    )
  }

  return Number(selected.id)
}

async function ensureUserPackage(api: any) {
  const packagesBody = await expectApiSuccess(await api.get('/api/v1/user/user_packages', { params: { pageSize: 10 } }))
  const packages = packagesBody.data?.list || packagesBody.list || []
  expect(packages.length).toBeGreaterThan(0)
  return packages[0].id
}

async function ensureSite(api: any, domain: string, backend: string, pkgId: number) {
  const listBody = await expectApiSuccess(
    await api.get('/api/v1/user/sites', { params: { search_field: 'domain', keyword: domain, pageSize: 10 } })
  )
  const list = listBody.data?.list || listBody.list || []
  if (list.length > 0) {
    return list[0].id
  }

  const createBody = await expectApiSuccess(
    await api.post('/api/v1/user/sites', {
      data: { domains: [domain], backends: [backend], user_package_id: pkgId }
    })
  )
  const siteId = createBody.data?.id || 0
  expect(siteId, 'site id').toBeTruthy()
  return siteId
}

async function findCert(api: any, domain: string) {
  const certBody = await expectApiSuccess(
    await api.get('/api/v1/user/certs', { params: { search_field: 'domain', keyword: domain, pageSize: 10 } })
  )
  const certs = certBody.data?.list || certBody.list || certBody.data || []
  if (certs.length > 0) {
    return certs[0]
  }
  return null
}

async function waitForCertReady(api: any, domain: string, minutes: number) {
  const deadline = Date.now() + minutes * 60_000
  while (Date.now() < deadline) {
    const cert = await findCert(api, domain)
    if (cert) {
      const state = String(cert.state || '').toLowerCase()
      if (state === 'ready' || state === 'success' || state === '') {
        return
      }
      if (state === 'fail') {
        const reason = cert.ret || cert.issue_task_ret || 'unknown error'
        throw new Error(`cert issue failed: ${reason}`)
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 15_000))
  }
  throw new Error('cert status did not reach ready before timeout')
}

function normalizeDnsType(value: string) {
  return String(value || '').toLowerCase().replace(/\./g, '')
}
