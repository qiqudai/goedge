import { expect, test } from '@playwright/test'
import { createUserApiContext, expectApiSuccess, loginUser } from './_api'

const hasNonAscii = (text: string) => {
  for (const ch of text) {
    if (ch.charCodeAt(0) > 127) return true
  }
  return false
}

test.describe('user: message localization', () => {
  test.setTimeout(120_000)

  test('message titles return chinese labels', async () => {
    const { token } = await loginUser('ceshi', '123456')
    const api = await createUserApiContext(token)

    const listBody = await expectApiSuccess(await api.get('/api/v1/user/messages', { params: { pageSize: 10 } }))
    const list = listBody.data?.list || listBody.list || []
    expect(list.length).toBeGreaterThan(0)
    for (const item of list) {
      expect(hasNonAscii(item.title)).toBeTruthy()
    }

    await api.dispose()
  })
})
