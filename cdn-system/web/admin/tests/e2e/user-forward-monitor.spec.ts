import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

test.describe('user: forward monitor', () => {
  test.setTimeout(120_000)

  test('traffic and ranking endpoints return data', async () => {
    const { token } = await loginUser('ceshi', '123456')
    const api = await createUserApiContext(token)

    const trafficBody = await expectApiSuccess(await api.get('/api/v1/user/forward/traffic', { params: { range: '1h' } }))
    const traffic = trafficBody.data || trafficBody
    expect(Array.isArray(traffic.x_axis)).toBeTruthy()
    expect(Array.isArray(traffic.bandwidth)).toBeTruthy()
    expect(Array.isArray(traffic.traffic)).toBeTruthy()

    const rankingBody = await expectApiSuccess(await api.get('/api/v1/user/forward/ranking', { params: { range: '1h' } }))
    const ranking = rankingBody.data || rankingBody
    expect(Array.isArray(ranking.list)).toBeTruthy()

    await api.dispose()
  })
})
