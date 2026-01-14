import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

test.describe('user: usage and recharge', () => {
  test.setTimeout(120_000)

  test('usage endpoint', async () => {
    const { token } = await loginUser('ceshi', '123456')
    const api = await createUserApiContext(token)

    const usageBody = await expectApiSuccess(await api.get('/api/v1/user/usage', { params: { range: '7days' } }))
    const usage = usageBody.data || usageBody
    expect(Array.isArray(usage.x_axis)).toBeTruthy()
    expect(Array.isArray(usage.values)).toBeTruthy()
    expect(Array.isArray(usage.list)).toBeTruthy()
    expect(typeof usage.total).toBe('number')

    await api.dispose()
  })

  test('recharge endpoint', async () => {
    const { token } = await loginUser('ceshi', '123456')
    const api = await createUserApiContext(token)

    await expectApiSuccess(await api.post('/api/v1/user/recharge', { data: { amount: 1, remark: 'autotest' } }))

    await api.dispose()
  })
})
