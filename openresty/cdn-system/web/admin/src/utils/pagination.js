export const buildPageSizeKey = (routePath, key) => {
  const safeRoute = routePath || 'unknown'
  const safeKey = key || 'default'
  return `page-size:${safeRoute}:${safeKey}`
}

export const loadPageSize = (storageKey, fallback = 10) => {
  const raw = localStorage.getItem(storageKey)
  const parsed = Number.parseInt(raw || '', 10)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return fallback
  }
  return parsed
}

export const savePageSize = (storageKey, size) => {
  if (!size || !Number.isFinite(size)) {
    return
  }
  localStorage.setItem(storageKey, String(size))
}
