import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useLoading } from '@/composables/useLoading'

const DEFAULT_API_BASE = 'https://goai.665305.cc'
export const API_BASE = (() => {
  if (typeof window === 'undefined') {
    return DEFAULT_API_BASE
  }
  const host = window.location.hostname
  if (host === '127.0.0.1' || host === 'localhost') {
    return 'http://127.0.0.1:8080'
  }
  return DEFAULT_API_BASE
})()

// Create axios instance
const service = axios.create({
  baseURL: `${API_BASE}/api/v1/admin`, // Default, overridden per-role
  timeout: 5000
})

const { showLoading, hideLoading } = useLoading()

const redirectToMaintenance = payload => {
  const message = payload?.msg || payload?.data?.message || '系统维护中'
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
      return Promise.reject(new Error(res.msg || 'maintenance'))
    }
    const refreshedToken = response.headers?.['x-auth-token']
    if (refreshedToken) {
      localStorage.setItem('admin_token', refreshedToken)
    }
    // If backend returns code, check it (assuming 0 is success)
    if (res.code !== undefined && res.code !== 0) {
      ElMessage({
        message: res.msg || 'Error',
        type: 'error',
        duration: 5 * 1000
      })
      return Promise.reject(new Error(res.msg || 'Error'))
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
      ElMessage({
        message: '登录失效，请重新登录',
        type: 'error',
        duration: 3000
      })
      localStorage.removeItem('admin_token')
      localStorage.removeItem('role')
      setTimeout(() => {
        window.location.href = '/login'
      }, 1000)
    } else {
      const apiMessage = error.response?.data?.error || error.response?.data?.msg
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
