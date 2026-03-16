import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useLoading } from '@/composables/useLoading'

const DEFAULT_API_BASE = 'https://goai.665305.cc'
const resolveApiBase = () => {
  const envBase = String(import.meta.env?.VITE_API_BASE || '').trim()
  if (envBase) return envBase
  if (typeof window !== 'undefined') {
    try {
      const stored = window.localStorage?.getItem('api_base')
      if (stored) return stored
    } catch {
      // ignore storage errors
    }
    const host = window.location?.hostname || ''
    if (host === '127.0.0.1' || host === 'localhost') {
      return 'http://127.0.0.1:8080'
    }
  }
  return DEFAULT_API_BASE
}
export const API_BASE = resolveApiBase()

// Create axios instance
const service = axios.create({
  baseURL: `${API_BASE}/api/v1/admin`, // Default, overridden per-role
  timeout: 120000
})

const { showLoading, hideLoading } = useLoading()

const isSuccessCode = code => code === 0 || code === 200

const resolveMessage = payload => payload?.message || payload?.msg || payload?.data?.message || payload?.error || ''

const AUTH_INVALID_CODES = new Set([401, 40101])
const AUTH_INVALID_MESSAGES = new Set([
  'Invalid or expired token',
  'Authorization header missing',
  'Invalid authorization format',
  '令牌无效或已过期',
  '授权信息缺失',
  '登录失效',
  '请重新登录'
])
let isRedirectingToLogin = false

const isAuthInvalidResponse = payload => {
  const code = Number(payload?.code)
  if (AUTH_INVALID_CODES.has(code)) {
    return true
  }
  const message = resolveMessage(payload)
  if (!message) {
    return false
  }
  if (AUTH_INVALID_MESSAGES.has(message)) {
    return true
  }
  const lowered = message.toLowerCase()
  return lowered.includes('invalid or expired token') ||
    lowered.includes('authorization header missing') ||
    lowered.includes('invalid authorization format') ||
    lowered.includes('unauthorized')
}

const redirectToLogin = message => {
  localStorage.removeItem('admin_token')
  localStorage.removeItem('role')
  const msg = message || '令牌无效或已过期，请重新登录'
  if (window.location.pathname === '/login') {
    if (msg) {
      ElMessage.closeAll?.()
      ElMessage({ message: msg, type: 'error', duration: 3000 })
    }
    return
  }
  if (isRedirectingToLogin) {
    return
  }
  isRedirectingToLogin = true
  ElMessage.closeAll?.()
  ElMessage({ message: msg, type: 'error', duration: 3000 })
  setTimeout(() => {
    window.location.replace('/login')
  }, 300)
}

const redirectToMaintenance = payload => {
  const message = resolveMessage(payload) || 'Maintenance'
  localStorage.setItem('maintenance_msg', message)
  if (window.location.pathname !== '/maintenance') {
    window.location.href = '/maintenance'
  }
}

// Request interceptor
service.interceptors.request.use(
  config => {
    if (!config.skipLoading) {
      config.__loadingToken = showLoading()
    }
    const role = localStorage.getItem('role') || 'user'
    if (config.baseURL === `${API_BASE}/api/v1/admin`) {
      config.baseURL = role === 'admin'
        ? `${API_BASE}/api/v1/admin`
        : `${API_BASE}/api/v1/user`
    }
    // Inject Token if exists
    const token = localStorage.getItem('admin_token')
    if (token) {
      config.headers['Authorization'] = 'Bearer ' + token
    }
    return config
  },
  error => {
    if (!error.config?.skipLoading) {
      hideLoading(error.config?.__loadingToken)
    }
    return Promise.reject(error)
  }
)

// Response interceptor
service.interceptors.response.use(
  response => {
    if (!response.config?.skipLoading) {
      hideLoading(response.config?.__loadingToken)
    }
    const res = response.data
    if (res?.maintenance) {
      redirectToMaintenance(res)
      return Promise.reject(new Error(resolveMessage(res) || 'maintenance'))
    }
    const refreshedToken = response.headers?.['x-auth-token']
    if (refreshedToken) {
      localStorage.setItem('admin_token', refreshedToken)
    }
    if (res.code === undefined) {
      if (isAuthInvalidResponse(res)) {
        redirectToLogin(resolveMessage(res))
        return Promise.reject(new Error(resolveMessage(res) || 'Error'))
      }
      return res
    }
    // If backend returns code, check it (assuming 0 is success)
    if (!isSuccessCode(res.code)) {
      if (isAuthInvalidResponse(res)) {
        redirectToLogin(resolveMessage(res))
        return Promise.reject(new Error(resolveMessage(res) || 'Error'))
      }
      ElMessage({
        message: resolveMessage(res) || 'Error',
        type: 'error',
        duration: 5 * 1000
      })
      return Promise.reject(new Error(resolveMessage(res) || 'Error'))
    } else {
      return res
    }
  },
  error => {
    if (!error.config?.skipLoading) {
      hideLoading(error.config?.__loadingToken)
    }
    if (error.response && error.response.status === 503 && error.response.data?.maintenance) {
      redirectToMaintenance(error.response.data)
      return Promise.reject(new Error(error.response.data?.msg || 'maintenance'))
    }
    if (error.response && error.response.status === 401) {
      redirectToLogin(resolveMessage(error.response.data))
      return Promise.reject(error)
    } else {
      const apiMessage = error.response?.data?.error || error.response?.data?.message || error.response?.data?.msg
      ElMessage({
        message: apiMessage || error.message,
        type: 'error',
        duration: 5 * 1000
      })
    }
    return Promise.reject(error)
  }
)

export default service


