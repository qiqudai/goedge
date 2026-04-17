import { API_BASE } from '@/utils/request'

const ABSOLUTE_URL_RE = /^(https?:)?\/\//i

export const resolveAssetUrl = (value) => {
  const raw = String(value || '').trim()
  if (!raw) {
    return ''
  }
  if (ABSOLUTE_URL_RE.test(raw) || raw.startsWith('data:') || raw.startsWith('blob:')) {
    return raw
  }
  if (raw.startsWith('/')) {
    return `${API_BASE}${raw}`
  }
  return `${API_BASE}/${raw}`
}

export const toStoredAssetUrl = (value) => {
  const raw = String(value || '').trim()
  if (!raw) {
    return ''
  }
  if (raw.startsWith('data:') || raw.startsWith('blob:')) {
    return raw
  }
  const apiBase = String(API_BASE || '').trim().replace(/\/+$/, '')
  if (apiBase) {
    if (raw === apiBase) {
      return ''
    }
    if (raw.startsWith(`${apiBase}/`)) {
      return raw.slice(apiBase.length)
    }
  }
  return raw
}
