import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

test.describe('user: site actions', () => {
  test.setTimeout(180_000)

  test('site create/update/batch actions/export/resolve', async () => {
    const { token } = await loginUser('ceshi', '123456')
    const api = await createUserApiContext(token)

    const packagesBody = await expectApiSuccess(await api.get('/api/v1/user/user_packages', { params: { pageSize: 10 } }))
    const packages = packagesBody.data?.list || packagesBody.list || []
    expect(packages.length).toBeGreaterThan(0)
    const pkgId = packages[0].id

    const domain = `autotest-${Date.now()}.example.com`
    const createBody = await expectApiSuccess(
      await api.post('/api/v1/user/sites', { data: { domains: [domain], backends: ['1.1.1.1'], user_package_id: pkgId } })
    )
    const site = createBody.data || {}
    const siteId = site.id
    expect(siteId).toBeTruthy()

    await expectApiSuccess(await api.get(`/api/v1/user/sites/${siteId}`))

    await expectApiSuccess(await api.put(`/api/v1/user/sites/${siteId}`, { data: { enable: false } }))
    await expectApiSuccess(await api.put(`/api/v1/user/sites/${siteId}`, { data: { enable: true } }))

    await expectApiSuccess(await api.post('/api/v1/user/sites/batch_action', { data: { action: 'disable', ids: [siteId] } }))
    await expectApiSuccess(await api.post('/api/v1/user/sites/batch_action', { data: { action: 'enable', ids: [siteId] } }))

    await expectApiSuccess(
      await api.post('/api/v1/user/sites/batch_update', { data: { ids: [siteId], balance_way: 'rr' } })
    )

    await expectApiSuccess(await api.post('/api/v1/user/sites/batch_action', { data: { action: 'clear_cache', ids: [siteId] } }))

    await expectApiSuccess(await api.post('/api/v1/user/sites/apply_cert', { data: { ids: [siteId] } }))

    const resolveRes = await api.get('/api/v1/user/sites/resolve', { params: { domain } })
    expect(resolveRes.ok()).toBeTruthy()

    const exportRes = await api.get('/api/v1/user/sites/export')
    expect(exportRes.ok()).toBeTruthy()
    const contentType = exportRes.headers()['content-type'] || ''
    expect(contentType).toContain('text/csv')

    const batchDomain = `autotest-batch-${Date.now()}.example.com`
    await expectApiSuccess(
      await api.post('/api/v1/user/sites/batch', {
        data: {
          user_package_id: pkgId,
          data: `domain=${batchDomain}|ip=1.1.1.1`
        }
      })
    )

    await expectApiSuccess(await api.post('/api/v1/user/sites/batch_action', { data: { action: 'delete', ids: [siteId] } }))

    await api.dispose()
  })
})
