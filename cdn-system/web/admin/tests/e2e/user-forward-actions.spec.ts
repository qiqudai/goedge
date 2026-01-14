import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

test.describe('user: forward actions', () => {
  test.setTimeout(180_000)

  test('forward create/update/batch actions', async () => {
    const { token } = await loginUser('ceshi', '123456')
    const api = await createUserApiContext(token)

    const packagesBody = await expectApiSuccess(await api.get('/api/v1/user/user_packages', { params: { pageSize: 10 } }))
    const packages = packagesBody.data?.list || packagesBody.list || []
    expect(packages.length).toBeGreaterThan(0)
    const pkgId = packages[0].id

    const port = 20000 + Math.floor(Math.random() * 500)
    const createBody = await expectApiSuccess(
      await api.post('/api/v1/user/forwards', {
        data: {
          listen_ports_input: String(port),
          origin_input: '1.1.1.1:8080',
          user_package_id: pkgId,
          remark: 'autotest'
        }
      })
    )
    const forward = createBody.data || {}
    const forwardId = forward.id
    expect(forwardId).toBeTruthy()

    await expectApiSuccess(
      await api.put(`/api/v1/user/forwards/${forwardId}`, {
        data: {
          listen_ports_input: String(port),
          origin_input: '1.1.1.1:8080',
          user_package_id: pkgId,
          remark: 'autotest-update'
        }
      })
    )

    await expectApiSuccess(await api.post('/api/v1/user/forwards/batch_action', { data: { action: 'disable', ids: [forwardId] } }))
    await expectApiSuccess(await api.post('/api/v1/user/forwards/batch_action', { data: { action: 'enable', ids: [forwardId] } }))

    await expectApiSuccess(
      await api.post('/api/v1/user/forwards/batch_update', { data: { ids: [forwardId], remark: 'autotest-batch' } })
    )

    await expectApiSuccess(await api.post('/api/v1/user/forwards/batch_action', { data: { action: 'delete', ids: [forwardId] } }))

    await api.dispose()
  })
})
