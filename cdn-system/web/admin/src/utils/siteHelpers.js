import { parseBool } from './helpers'

/**
 * 网站设置相关的工具函数
 */

/**
 * 标准化源站条件配置
 * @param {Object} item - 源站条件对象
 * @returns {Object|null} 标准化后的条件对象
 */
export function normalizeOriginCondition(item) {
  if (!item) return null
  return {
    item: item.item || 'uri',
    operator: item.operator || 'eq',
    value: item.value || '',
    origin: item.origin || '',
    header: item.header || '',
    seconds: item.seconds || ''
  }
}

/**
 * 检查是否为请求头匹配项
 * @param {string} item - 匹配项类型
 * @returns {boolean}
 */
export function isOriginHeaderItem(item) {
  return item === 'header'
}

/**
 * 检查是否为统计类匹配项
 * @param {string} item - 匹配项类型
 * @returns {boolean}
 */
export function isOriginStatItem(item) {
  return item === 'ua_count' || item === 'status_404'
}

/**
 * 获取源站条件输入提示文本
 * @param {Object} row - 条件行数据
 * @returns {string} 提示文本
 */
export function getOriginConditionPlaceholder(row) {
  if (!row) return '输入匹配值，一行一个'
  
  switch (row.item) {
    case 'http_version':
      return '输入 HTTP/1.0、HTTP/1.1 等'
    case 'method':
      return '输入请求方法，如 GET'
    case 'client_ip':
      return '输入 IP 地址'
    case 'domain':
      return '输入域名，如 example.com'
    case 'uri':
    case 'uri_no_args':
      return '输入路径，如 /index.html'
    case 'node_country':
    case 'client_country':
      return '输入国家代码，如 CN'
    case 'node_isp':
    case 'client_isp':
      return '输入运营商，如 电信'
    case 'node_province':
    case 'client_province':
      return '输入省份，如 广东'
    case 'node_city':
    case 'client_city':
      return '输入城市，如 深圳'
    case 'ua_count':
    case 'status_404':
      return '输入次数'
    case 'header':
      return '输入请求头名称'
    default:
      return '输入匹配值，一行一个'
  }
}

/**
 * 处理源站条件变化
 * @param {Object} row - 条件行数据
 */
export function handleOriginConditionChange(row) {
  if (!row) return
  
  if (isOriginStatItem(row.item)) {
    row.operator = 'gt'
    row.seconds = row.seconds || '10'
  } else if (!row.operator) {
    row.operator = 'eq'
  }
}

/**
 * 标准化缓存规则
 * @param {Object} rule - 缓存规则对象
 * @returns {Object|null} 标准化后的规则对象
 */
export function normalizeCacheRule(rule) {
  if (!rule) return null
  
  const ttl = rule.ttl || '86400'
  return {
    type: rule.type || 'index',
    value: rule.value || '',
    ttl: String(ttl),
    ignore_query: !!rule.ignore_query,
    force_cache: !!rule.force_cache,
    enable_range: !!(rule.enable_range ?? rule.enable_slice),
    ignore_vary: !!rule.ignore_vary,
    skip_conditions: rule.skip_conditions || []
  }
}

const splitCacheRuleValues = (value) => {
  const raw = (value || '').trim()
  if (!raw) return []
  const parts = raw.includes('|') ? raw.split('|') : raw.split(/[\s\n]+/)
  return parts.map(part => part.trim()).filter(Boolean)
}

const normalizeCachePathValue = (value) => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  return raw.startsWith('/') ? raw : `/${raw}`
}

const normalizeCacheExtValue = (value) => {
  let raw = String(value || '').trim().toLowerCase()
  if (!raw) return ''
  while (raw.startsWith('*')) raw = raw.slice(1)
  if (raw.startsWith('.')) raw = raw.slice(1)
  return raw
}

const normalizeRuleLocation = (rule) => {
  const raw = String(rule || '').trim()
  if (!raw) return ''
  if (raw.startsWith('=') || raw.startsWith('^~') || raw.startsWith('~')) {
    return raw
  }
  if (raw.startsWith('/')) {
    return `^~ ${raw}`
  }
  if (raw.startsWith('.')) {
    return `~* \\${raw}$`
  }
  return `~* ${raw}`
}

const normalizeLocationKey = (location) => {
  const raw = String(location || '').trim()
  if (!raw) return ''
  const parts = raw.split(/\s+/).filter(Boolean)
  if (!parts.length) return ''
  const head = parts[0]
  if (head === '=') {
    return `exact ${parts.slice(1).join(' ')}`
  }
  if (head === '^~') {
    return `prefix ${parts.slice(1).join(' ')}`
  }
  if (head.startsWith('~')) {
    return `regex ${head}${parts.length > 1 ? ` ${parts.slice(1).join(' ')}` : ''}`
  }
  return `prefix ${parts.join(' ')}`
}

export function dedupeCacheRules(rules = []) {
  if (!Array.isArray(rules) || rules.length === 0) {
    return { rules: [], removed: 0 }
  }
  const normalized = rules.map((rule) => {
    const base = normalizeCacheRule(rule)
    if (!base) return null
    return { ...(rule || {}), ...base }
  }).filter(Boolean)
  const seen = new Set()
  const result = []
  let removed = 0

  for (let i = normalized.length - 1; i >= 0; i -= 1) {
    const rule = { ...normalized[i] }
    let kept = false

    const ruleExpr = (rule.rule || '').trim()
    const uri = (rule.uri || '').trim()
    const prefix = (rule.prefix || '').trim()
    const ext = (rule.ext || '').trim()
    const type = (rule.type || '').toLowerCase()

    const tryKeep = (location) => {
      const key = normalizeLocationKey(location)
      if (!key || seen.has(key)) {
        return false
      }
      seen.add(key)
      return true
    }

    if (ruleExpr) {
      const location = normalizeRuleLocation(ruleExpr)
      if (location && tryKeep(location)) {
        kept = true
      } else {
        removed += 1
      }
    } else if (uri) {
      const value = normalizeCachePathValue(uri)
      const location = value ? `= ${value}` : ''
      if (location && tryKeep(location)) {
        rule.uri = value
        kept = true
      } else {
        removed += 1
      }
    } else if (prefix) {
      const value = normalizeCachePathValue(prefix)
      const location = value ? `^~ ${value}` : ''
      if (location && tryKeep(location)) {
        rule.prefix = value
        kept = true
      } else {
        removed += 1
      }
    } else if (ext) {
      const value = normalizeCacheExtValue(ext)
      const location = value ? `~* \\.${value}$` : ''
      if (location && tryKeep(location)) {
        rule.ext = value
        kept = true
      } else {
        removed += 1
      }
    } else if (type === 'all') {
      if (tryKeep('^~ /')) {
        kept = true
      } else {
        removed += 1
      }
    } else if (type === 'index') {
      if (tryKeep('= /')) {
        kept = true
      } else {
        removed += 1
      }
    } else if (type === 'dir' || type === 'path' || type === 'suffix') {
      const values = splitCacheRuleValues(rule.value)
      const keptValues = []
      for (const val of values) {
        if (type === 'suffix') {
          const extValue = normalizeCacheExtValue(val)
          if (!extValue) continue
          const location = `~* \\.${extValue}$`
          if (!tryKeep(location)) {
            removed += 1
            continue
          }
          keptValues.push(extValue)
        } else if (type === 'dir') {
          const dirValue = normalizeCachePathValue(val)
          if (!dirValue) continue
          const location = `^~ ${dirValue}`
          if (!tryKeep(location)) {
            removed += 1
            continue
          }
          keptValues.push(dirValue)
        } else {
          const pathValue = normalizeCachePathValue(val)
          if (!pathValue) continue
          const location = `= ${pathValue}`
          if (!tryKeep(location)) {
            removed += 1
            continue
          }
          keptValues.push(pathValue)
        }
      }
      if (keptValues.length > 0) {
        rule.value = keptValues.join('|')
        kept = true
      } else {
        removed += 1
      }
    } else {
      kept = true
    }

    if (kept) {
      result.push(rule)
    }
  }

  result.reverse()
  return { rules: result, removed }
}

export function dedupeHeaderRules(list = []) {
  if (!Array.isArray(list) || list.length === 0) {
    return { list: [], removed: 0 }
  }
  const seen = new Set()
  const result = []
  let removed = 0
  for (let i = list.length - 1; i >= 0; i -= 1) {
    const item = list[i] || {}
    const name = String(item.name || '').trim()
    if (!name) {
      removed += 1
      continue
    }
    const key = name.toLowerCase()
    if (seen.has(key)) {
      removed += 1
      continue
    }
    seen.add(key)
    result.push({ ...item, name })
  }
  result.reverse()
  return { list: result, removed }
}

const buildRedirectConditionKey = (conditions) => {
  if (!Array.isArray(conditions) || conditions.length === 0) return ''
  const items = []
  for (const condition of conditions) {
    if (!condition) continue
    const key = String(condition.key || condition.item || '').trim().toLowerCase()
    const value = String(condition.value || '').trim()
    if (!key && !value) continue
    items.push(`${key}=${value}`)
  }
  items.sort()
  return items.join('&')
}

export function dedupeUrlRedirects(list = []) {
  if (!Array.isArray(list) || list.length === 0) {
    return { list: [], removed: 0 }
  }
  const seen = new Set()
  const result = []
  let removed = 0
  for (let i = list.length - 1; i >= 0; i -= 1) {
    const item = list[i] || {}
    const domain = String(item.domain || '').trim()
    const match = String(item.match || '').trim()
    const redirect = String(item.redirect || '').trim()
    const code = String(item.code || '').trim()
    const condKey = buildRedirectConditionKey(item.conditions)
    const key = `${domain.toLowerCase()}|${match}|${redirect}|${code}|${condKey}`
    if (!match || !redirect) {
      removed += 1
      continue
    }
    if (seen.has(key)) {
      removed += 1
      continue
    }
    seen.add(key)
    result.push({ ...item, domain, match, redirect, code })
  }
  result.reverse()
  return { list: result, removed }
}

/**
 * 分割字符串为数组
 * @param {string} str - 待分割的字符串
 * @returns {Array<string>} 分割后的数组
 */
export function splitStr(str) {
  return (str || '').split(/[\s\n]+/).filter(Boolean)
}

/**
 * 解析端口列表
 * @param {string} raw - 端口字符串
 * @returns {Array<string>} 端口数组
 */
export function parsePortList(raw) {
  if (!raw) return []
  
  return raw
    .split(/[\s,]+/)
    .map(item => item.trim())
    .filter(Boolean)
}

/**
 * 计算CNAME地址
 * @param {Object} data - 站点数据
 * @returns {string} CNAME地址
 */
export function computeCname(data) {
  if (data.cname_hostname) return data.cname_hostname
  
  if (data.domains?.length && data.cname_domain) {
    return `${data.domains[0]}.${data.cname_domain}`
  }
  
  return '-'
}

/**
 * 获取防盗链输入提示
 * @param {string} scope - 防盗链范围
 * @returns {string} 提示文本
 */
export function getHotlinkPlaceholder(scope) {
  switch (scope) {
    case 'suffix':
      return '请输入后缀，如 png|jpg|gif'
    case 'dir':
      return '请输入目录，如 /image/|/static/|/upload/'
    case 'path':
      return '请输入路径，如 /index.html'
    default:
      return ''
  }
}

/**
 * 验证配置数据
 * @param {Object} settings - 网站设置
 * @param {string} activeTab - 当前激活的标签页
 * @returns {boolean} 是否有效
 */
export function validateSettings(settings, activeTab) {
  // 安全设置验证
  if (activeTab === 'security') {
    const autoSwitch = settings.security.cc.autoSwitch
    if (autoSwitch.enable && (!autoSwitch.qps || !autoSwitch.rule)) {
      return false
    }
  }
  
  return true
}

/**
 * 获取缓存预设规则
 * @param {string} preset - 预设类型
 * @returns {Object|null} 预设规则
 */
export function getCachePreset(preset) {
  const presets = {
    index: { 
      type: 'index', 
      value: '', 
      ttl: '86400' 
    },
    all: { 
      type: 'all', 
      value: '', 
      ttl: '259200' 
    },
    static: { 
      type: 'suffix', 
      value: 'jpg|jpeg|png|gif|ico|css|js|svg|bmp|webp|woff|woff2', 
      ttl: '604800', 
      ignore_query: true 
    },
    video: { 
      type: 'suffix', 
      value: 'mp4|avi|mov|webm|m3u8|ts', 
      ttl: '2592000' 
    },
    wordpress: { 
      type: 'all', 
      value: '', 
      ttl: '259200' 
    }
  }
  
  return presets[preset] || null
}

/**
 * 验证域名格式
 * @param {string} domain - 域名字符串
 * @returns {boolean} 是否有效
 */
export function validateDomain(domain) {
  if (!domain) return false
  // 不允许带有协议前缀
  if (domain.includes('://')) return false

  // IP地址验证
  const ipRegex = /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/
  if (ipRegex.test(domain)) return true

  // 域名验证 (支持泛域名)
  const domainRegex = /^(?:\*\.)?(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/
  return domainRegex.test(domain)
}
