export function toStr(value, fallback = '') {
  if (value === undefined || value === null) return fallback
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return fallback
}

export function parseBool(value, def = false) {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (typeof value === 'string') {
    const v = value.trim().toLowerCase()
    if (v === '') return def
    return v === '1' || v === 'true' || v === 'on' || v === 'yes'
  }
  return def
}

export function parseList(value) {
  if (Array.isArray(value)) return value
  if (!value) return []
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      if (Array.isArray(parsed)) return parsed
    } catch (err) {
      return value.split(/[\s,]+/).filter(Boolean)
    }
  }
  return []
}

export function toIntSafe(value, fallback = 0) {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const parsed = Number(value)
    return Number.isNaN(parsed) ? fallback : parsed
  }
  return fallback
}

export function convertSecondsToUnit(seconds) {
  const s = parseInt(seconds || 0)
  if (s <= 0) return { value: '0', unit: 'second' }
  if (s % 86400 === 0) return { value: String(s / 86400), unit: 'day' }
  if (s % 3600 === 0) return { value: String(s / 3600), unit: 'hour' }
  if (s % 60 === 0) return { value: String(s / 60), unit: 'minute' }
  return { value: String(s), unit: 'second' }
}

export function convertUnitToSeconds(value, unit) {
  const v = parseInt(value || 0)
  if (isNaN(v)) return 0
  switch (unit) {
    case 'day': return v * 86400
    case 'hour': return v * 3600
    case 'minute': return v * 60
    case 'second': return v
    default: return v
  }
}

export function normalizeCacheRule(rule) {
  if (!rule) return null
  const ttl = rule.ttl || rule.expire || rule.cache_time || ''
  return {
    type: rule.type || 'index',
    value: toStr(rule.value || rule.content || '', ''),
    ttl: toStr(ttl || '86400', '86400'),
    ignore_query: parseBool(rule.ignore_query, false),
    force_cache: parseBool(rule.force_cache, false),
    enable_range: parseBool(rule.enable_range, false),
    ignore_vary: parseBool(rule.ignore_vary) || false,
    skip_conditions: Array.isArray(rule.skip_conditions) ? rule.skip_conditions : []
  }
}

export function debounce(fn, wait) {
  let timer
  return (...args) => {
    clearTimeout(timer)
    timer = setTimeout(() => fn(...args), wait)
  }
}

export function formatTTL(seconds) {
  const s = parseInt(seconds)
  if (isNaN(s)) return seconds
  if (s % 86400 === 0) return (s / 86400) + ' 天'
  if (s % 3600 === 0) return (s / 3600) + ' 小时'
  if (s % 60 === 0) return (s / 60) + ' 分'
  return s + ' 秒'
}
