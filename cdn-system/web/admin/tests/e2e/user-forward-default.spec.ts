import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

test.describe('user: forward defaults', () => {
  test.setTimeout(120_000)

  test('create/list/delete default', async () => {
    const { token } = await loginUser('ceshi', '123456')
    const api = await createUserApiContext(token)

    await expectApiSuccess(
      await api.post('/api/v1/user/forward_defaults', {
        data: { key: 'proxy_protocol', value: true, scope: 'global', group_id: 0 }
      })
    )

    const listBody = await expectApiSuccess(await api.get('/api/v1/user/forward_defaults'))
    const list = listBody.list || listBody.data?.list || []
    expect(list.some((item: any) => item.key === 'proxy_protocol')).toBeTruthy()

    await expectApiSuccess(await api.delete('/api/v1/user/forward_defaults', { data: { id_str: 'proxy_protocol' } }))

    await api.dispose()
  })
})
