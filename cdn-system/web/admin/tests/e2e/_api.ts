import fs from 'node:fs'
import path from 'node:path'
import { APIRequestContext, expect, request } from '@playwright/test'

const API_BASE = process.env.API_BASE || 'http://127.0.0.1:8080'

type LoginResult = { token: string; uid: number }

const loginCache = new Map<string, LoginResult>()
const loginPromises = new Map<string, Promise<LoginResult>>()
const tokenCachePath = process.env.E2E_USER_TOKEN_CACHE
  || path.resolve(process.cwd(), '..', '..', '.tmp', 'e2e-user-token.json')
const tokenLockPath = `${tokenCachePath}.lock`

const isSuccessCode = (code: any) => code === 0 || code === 200

const clearTokenCache = () => {
  try {
    if (fs.existsSync(tokenCachePath)) {
      fs.unlinkSync(tokenCachePath)
    }
  } catch {
    // ignore
  }
}

const verifyToken = async (token: string) => {
  if (!token) return false
  const ctx = await request.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}` }
  })
  try {
    const res = await ctx.get('/api/v1/user/profile')
    if (!res.ok()) return false
    const body = await readJsonSafe(res)
    if (body && body.code !== undefined) {
      return isSuccessCode(body.code)
    }
    return true
  } catch {
    return false
  } finally {
    await ctx.dispose()
  }
}

const unwrapPayload = (body: any) => (body && typeof body === 'object' && 'data' in body ? body.data : body)

const extractToken = (body: any) => {
  const payload = unwrapPayload(body)
  if (!payload) return null
  return {
    token: String(payload.token || ''),
    uid: Number(payload.uid || 0)
  }
}

const readJsonSafe = async (res: any) => {
  try {
    return await res.json()
  } catch {
    return null
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

const readTokenCache = () => {
  try {
    if (!fs.existsSync(tokenCachePath)) return null
    const raw = fs.readFileSync(tokenCachePath, 'utf-8')
    const parsed = JSON.parse(raw)
    if (parsed && parsed.token) {
      return { token: String(parsed.token), uid: Number(parsed.uid || 0) }
    }
  } catch {
    // ignore cache errors
  }
  return null
}

const writeTokenCache = (token: string, uid: number) => {
  try {
    fs.mkdirSync(path.dirname(tokenCachePath), { recursive: true })
    fs.writeFileSync(tokenCachePath, JSON.stringify({ token, uid, created_at: Date.now() }))
  } catch {
    // ignore cache write errors
  }
}

const withTokenLock = async <T>(action: () => Promise<T>) => {
  const start = Date.now()
  while (true) {
    try {
      const fd = fs.openSync(tokenLockPath, 'wx')
      try {
        return await action()
      } finally {
        fs.closeSync(fd)
        fs.unlinkSync(tokenLockPath)
      }
    } catch (err: any) {
      if (err?.code !== 'EEXIST') {
        throw err
      }
      if (Date.now() - start > 30_000) {
        return action()
      }
      await sleep(200)
    }
  }
}

async function postWithRetry(ctx: APIRequestContext, url: string, data: Record<string, any>, retries = 2) {
  let res: any = null
  for (let attempt = 0; attempt <= retries; attempt += 1) {
    res = await ctx.post(url, { data })
    if (res.status() !== 429) {
      return res
    }
    const retryBody = await readJsonSafe(res)
    const payload = unwrapPayload(retryBody) || {}
    const waitMs = typeof payload.rate_cooldown === 'number' ? payload.rate_cooldown * 1000 : 1000
    await sleep(waitMs + attempt * 250)
  }
  return res
}

async function tryLogin(ctx: APIRequestContext, username: string, password: string) {
  const res = await postWithRetry(ctx, '/api/v1/user/login', { username, password })
  const body = await readJsonSafe(res)
  const token = extractToken(body)
  return { res, body, token }
}

async function tryRegister(ctx: APIRequestContext, username: string, password: string) {
  const res = await postWithRetry(ctx, '/api/v1/user/register', { username, password })
  const body = await readJsonSafe(res)
  return { res, body }
}

export async function loginUser(username: string, password: string): Promise<LoginResult> {
  if (process.env.E2E_USER_TOKEN) {
    return {
      token: process.env.E2E_USER_TOKEN,
      uid: Number(process.env.E2E_USER_ID || 0)
    }
  }
  const cacheKey = username
  if (loginCache.has(cacheKey)) {
    const cached = loginCache.get(cacheKey) as LoginResult
    if (await verifyToken(cached.token)) {
      return cached
    }
    loginCache.delete(cacheKey)
  }
  const cached = readTokenCache()
  if (cached) {
    if (await verifyToken(cached.token)) {
      loginCache.set(cacheKey, cached)
      return cached
    }
    clearTokenCache()
  }
  if (loginPromises.has(cacheKey)) {
    return loginPromises.get(cacheKey) as Promise<LoginResult>
  }

  const promise = withTokenLock(async () => {
    const cachedAgain = readTokenCache()
    if (cachedAgain) return cachedAgain

    const ctx = await request.newContext({ baseURL: API_BASE })
    try {
      let attempt = await tryLogin(ctx, username, password)
      if (attempt.res.ok() && attempt.token?.token) {
        writeTokenCache(attempt.token.token, attempt.token.uid)
        return { token: attempt.token.token, uid: attempt.token.uid }
      }

      const register = await tryRegister(ctx, username, password)
      if (!register.res.ok()) {
        const msg = register.body?.message || register.body?.msg || register.body?.error || ''
        if (msg && msg.toLowerCase().includes('exists')) {
          // ignore if already exists
        }
      }

      attempt = await tryLogin(ctx, username, password)
      if (attempt.res.ok() && attempt.token?.token) {
        writeTokenCache(attempt.token.token, attempt.token.uid)
        return { token: attempt.token.token, uid: attempt.token.uid }
      }

      const finalMsg = attempt.body?.message || attempt.body?.msg || attempt.body?.error || ''
      throw new Error(`login token: ${finalMsg || attempt.res.status()}`)
    } finally {
      await ctx.dispose()
    }
  })

  loginPromises.set(cacheKey, promise)
  try {
    const result = await promise
    loginCache.set(cacheKey, result)
    return result
  } finally {
    loginPromises.delete(cacheKey)
  }
}

export async function createUserApiContext(token: string): Promise<APIRequestContext> {
  const ctx = await request.newContext({
    baseURL: API_BASE
  })
  let currentToken = token

  const wrap = (method: 'get' | 'post' | 'put' | 'delete') => {
    const original = (ctx as any)[method].bind(ctx)
    return async (url: string, options: any = {}) => {
      const headers = {
        ...(options.headers || {}),
        Authorization: `Bearer ${currentToken}`
      }
      const res = await original(url, { ...options, headers })
      const responseHeaders = res.headers()
      const refreshed = responseHeaders['x-auth-token']
      if (refreshed) {
        currentToken = refreshed
        writeTokenCache(refreshed, 0)
      }
      return res
    }
  }

  ;(ctx as any).get = wrap('get')
  ;(ctx as any).post = wrap('post')
  ;(ctx as any).put = wrap('put')
  ;(ctx as any).delete = wrap('delete')
  return ctx
}

export async function expectApiSuccess(res: any) {
  if (!res.ok()) {
    const text = await res.text()
    throw new Error(`api ${res.status()} ${text}`)
  }
  const body = await res.json()
  if (body.code !== undefined) {
    expect([0, 200]).toContain(body.code)
  }
  if (body.error || body.msg === 'error' || body.message === 'Error') {
    throw new Error(body.error || body.msg || body.message)
  }
  return body
}
