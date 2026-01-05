import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useLoading } from '@/composables/useLoading'

// Create axios instance
const service = axios.create({
  baseURL: '/api/v1/admin', // Default, overridden per-role
  timeout: 5000
})

const { showLoading, hideLoading } = useLoading()

// Request interceptor
service.interceptors.request.use(
  config => {
    if (!config.skipLoading) {
      showLoading()
    }
    const role = localStorage.getItem('role') || 'user'
    if (config.baseURL === '/api/v1/admin') {
      config.baseURL = role === 'admin' ? '/api/v1/admin' : '/api/v1/user'
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
      hideLoading()
    }
    return Promise.reject(error)
  }
)

// Response interceptor
service.interceptors.response.use(
  response => {
    if (!response.config?.skipLoading) {
      hideLoading()
    }
    const res = response.data
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
      hideLoading()
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
      ElMessage({
        message: error.message,
        type: 'error',
        duration: 5 * 1000
      })
    }
    return Promise.reject(error)
  }
)

export default service
