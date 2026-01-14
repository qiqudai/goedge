import { APIRequestContext, expect, request } from '@playwright/test'

const API_BASE = process.env.API_BASE || 'http://127.0.0.1:8080'

export async function loginUser(username: string, password: string) {
  if (process.env.E2E_USER_TOKEN) {
    return {
      token: process.env.E2E_USER_TOKEN,
      uid: Number(process.env.E2E_USER_ID || 0)
    }
  }
  const ctx = await request.newContext({ baseURL: API_BASE })
  let res = await ctx.post('/api/v1/user/login', { data: { username, password } })
  if (res.status() === 429) {
    const retryBody = await res.json().catch(() => ({}))
    const waitMs = typeof retryBody.rate_cooldown === 'number' ? retryBody.rate_cooldown * 1000 : 1000
    await new Promise((resolve) => setTimeout(resolve, waitMs))
    res = await ctx.post('/api/v1/user/login', { data: { username, password } })
  }
  expect(res.ok()).toBeTruthy()
  const body = await res.json()
  expect(body.token, 'login token').toBeTruthy()
  await ctx.dispose()
  return { token: body.token as string, uid: body.uid as number }
}

export async function createUserApiContext(token: string): Promise<APIRequestContext> {
  return request.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}` }
  })
}

export async function expectApiSuccess(res: any) {
  if (!res.ok()) {
    const text = await res.text()
    throw new Error(`api ${res.status()} ${text}`)
  }
  const body = await res.json()
  if (body.code !== undefined) {
    expect(body.code).toBe(0)
  }
  if (body.error || body.msg === 'error') {
    throw new Error(body.error || body.msg)
  }
  return body
}
