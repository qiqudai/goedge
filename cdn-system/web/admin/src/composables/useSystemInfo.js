import { reactive } from 'vue'
import axios from 'axios'
import { API_BASE } from '@/utils/request'

const systemInfo = reactive({
  sys_name: '',
  user_console_title: '',
  admin_console_title: '',
  footer_link: '',
  footer_copyright: '',
  favicon_file: '',
  logo_file: '',
  login_ad_file: '',
  enable_email_login: false,
  enable_sms_login: false,
  allow_register: false
})

let loadingPromise = null

const applyFavicon = (href) => {
  if (!href || typeof document === 'undefined') return
  let link = document.querySelector("link[rel~='icon']")
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.href = href
}

const applySystemInfo = (info) => {
  const next = info || {}
  systemInfo.sys_name = next.sys_name || ''
  systemInfo.user_console_title = next.user_console_title || ''
  systemInfo.admin_console_title = next.admin_console_title || ''
  systemInfo.footer_link = next.footer_link || ''
  systemInfo.footer_copyright = next.footer_copyright || ''
  systemInfo.favicon_file = next.favicon_file || ''
  systemInfo.logo_file = next.logo_file || ''
  systemInfo.login_ad_file = next.login_ad_file || ''
  systemInfo.enable_email_login = Boolean(next.enable_email_login)
  systemInfo.enable_sms_login = Boolean(next.enable_sms_login)
  systemInfo.allow_register = Boolean(next.allow_register)
  if (systemInfo.favicon_file) {
    applyFavicon(systemInfo.favicon_file)
  }
}

const loadSystemInfo = async (force = false) => {
  if (!force && loadingPromise) {
    return loadingPromise
  }
  loadingPromise = axios
    .get(`${API_BASE}/api/v1/system_info`)
    .then((res) => {
      if (res?.data?.code === 0 || res?.data?.code === 200) {
        applySystemInfo(res.data.data || {})
      }
    })
    .catch(() => {})
    .finally(() => {
      loadingPromise = null
    })
  return loadingPromise
}

export function useSystemInfo() {
  return {
    systemInfo,
    loadSystemInfo
  }
}
