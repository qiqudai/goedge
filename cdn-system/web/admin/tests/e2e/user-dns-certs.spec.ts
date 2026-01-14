import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

test.describe('user: dns api and certs', () => {
  test.setTimeout(120_000)

  test('dns api list and cert list', async () => {
    const { token, uid } = await loginUser('ceshi', '123456')
    const api = await createUserApiContext(token)

    const dnsBody = await expectApiSuccess(await api.get('/api/v1/user/dnsapi', { params: { pageSize: 1000 } }))
    const dnsList = dnsBody.data?.list || dnsBody.list || []
    expect(dnsList.length).toBeGreaterThan(0)
    const hasMasked = dnsList.some((item: any) => item.uid !== uid && item.auth === '')
    expect(hasMasked).toBeTruthy()

    await expectApiSuccess(await api.get('/api/v1/user/certs', { params: { pageSize: 10 } }))

    await api.dispose()
  })
})
