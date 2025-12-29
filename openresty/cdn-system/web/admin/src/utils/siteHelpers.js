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
    force_cache: !!rule.force_cache
  }
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