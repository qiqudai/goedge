import { splitStr } from './siteHelpers'

/**
 * 网站设置数据转换工具
 * 负责将前端设置数据转换为后端API所需的格式
 */

/**
 * 构建后端API负载数据
 * @param {Object} siteSettings - 前端网站设置
 * @returns {Object} 后端API负载
 */
export function buildSettingsPayload(siteSettings) {
  return {
    origin: buildOriginPayload(siteSettings.origin),
    https: buildHttpsPayload(siteSettings.https),
    cache: buildCachePayload(siteSettings.cache),
    security: buildSecurityPayload(siteSettings.security),
    access: buildAccessPayload(siteSettings.access),
    http_enable: siteSettings.basic.httpEnable,
    
    // 高级设置（扁平化）
    gzip: siteSettings.advanced.gzip,
    websocket: siteSettings.advanced.websocket,
    search_engine_origin: siteSettings.advanced.searchEngineOrigin,
    search_engine_origin_ip: siteSettings.advanced.searchEngineOriginIp,
    url_rewrites: siteSettings.advanced.urlRewrites,
    url_redirects: siteSettings.advanced.urlRedirects,
    req_headers: siteSettings.advanced.reqHeaders,
    res_headers: siteSettings.advanced.resHeaders,
    origin_cert: siteSettings.advanced.originCert,
    realtime_identify: siteSettings.advanced.realtimeIdentify,
    realtime_send: siteSettings.advanced.realtimeSend,
    upload_limit: siteSettings.advanced.uploadLimitMode === 'none' 
      ? 0 
      : parseInt(siteSettings.advanced.uploadLimitValue || 0),
    log_request_header: siteSettings.advanced.logRequestHeader,
    log_response_header: siteSettings.advanced.logResponseHeader,
    log_request_body: siteSettings.advanced.logRequestBody,
    log_request_body_size_limit: parseInt(siteSettings.advanced.logRequestBodySizeLimit || 16),
    
    // Default Site & L2 Config
    default_site: siteSettings.advanced.defaultSite,
    l2_config: siteSettings.advanced.l2Config,

    // Deprecated fields removed: proxy timeouts, upstream keepalive, limit rate

    // 源站相关设置
    origin_host: siteSettings.origin.host === 'custom' 
      ? siteSettings.origin.hostValue 
      : siteSettings.origin.host,
    origin_timeout: siteSettings.origin.timeout,
    origin_http_port: siteSettings.origin.httpPort,
    origin_https_port: siteSettings.origin.httpsPort,
    backend_protocol: siteSettings.origin.protocol
  }
}

/**
 * 构建源站配置负载
 * @param {Object} originSettings - 源站设置
 * @returns {Object} 源站负载
 */
function buildOriginPayload(originSettings) {
  return {
    list: originSettings.list.map(item => ({
      address: item.address,
      weight: parseInt(item.weight || 10, 10),
      enable: item.enable
    })),
    conditions: originSettings.conditions.map(item => ({
      ...item,
      seconds: item.seconds ? parseInt(item.seconds) : 0
    })),
    connTimeout: parseInt(originSettings.connTimeout || 10),
    health_check: originSettings.healthCheckEnabled,
    health_host: originSettings.healthCheckHost,
    health_path: originSettings.healthCheckPath,
    health_status: originSettings.healthCheckStatus,
    health_interval: parseInt(originSettings.healthCheckInterval)
  }
}

/**
 * 构建HTTPS配置负载
 * @param {Object} httpsSettings - HTTPS设置
 * @returns {Object} HTTPS负载
 */
function buildHttpsPayload(httpsSettings) {
  return {
    enable: httpsSettings.enable,
    listen_port: httpsSettings.listenPorts,
    force: httpsSettings.force,
    redirect_port: httpsSettings.forcePort,
    hsts: httpsSettings.hsts,
    http2: httpsSettings.http2,
    http3: httpsSettings.http3,
    ocsp_stapling: httpsSettings.ocsp,
    ssl_profile: httpsSettings.sslPolicy,
    ssl_protocols: httpsSettings.sslProtocols,
    ssl_ciphers: httpsSettings.sslCiphers,
    ssl_prefer_server_ciphers: true,
    certificate_id: httpsSettings.certId
  }
}

/**
 * 构建缓存配置负载
 * @param {Object} cacheSettings - 缓存设置
 * @returns {Object} 缓存负载
 */
function buildCachePayload(cacheSettings) {
  return {
    rules: cacheSettings.rules
  }
}

/**
 * 构建安全配置负载
 * @param {Object} securitySettings - 安全设置
 * @returns {Object} 安全负载
 */
function buildSecurityPayload(securitySettings) {
  return {
    default_rule: securitySettings.cc.mode,
    auto_switch: securitySettings.cc.autoSwitch.enable 
      ? JSON.stringify(securitySettings.cc.autoSwitch) 
      : '',
    custom_rules: securitySettings.cc.customRules,

    // IP Lists
    blacklist: splitStr(securitySettings.ip.black),
    whitelist: splitStr(securitySettings.ip.white),
    ip_black_timeout: securitySettings.ip.blackTime,
    ip_white_timeout: securitySettings.ip.whiteTime,

    // Crawlers
    crawlers_action: securitySettings.crawlers.action,

    // Block
    block_transparent_proxy: securitySettings.block.transparentProxy,

    // Cookie
    cookie: securitySettings.cookie,

    shield_proxy: false, // 默认值
    region_block: securitySettings.regions
  }
}

/**
 * 构建访问控制配置负载
 * @param {Object} accessSettings - 访问控制设置
 * @returns {Object} 访问控制负载
 */
function buildAccessPayload(accessSettings) {
  return {
    acl: accessSettings.acl,
    hotlink: accessSettings.hotlink,
    cors: accessSettings.cors,
    region_block: {
      mode: accessSettings.regionBlock.mode,
      countries: accessSettings.regionBlock.countries
    }
  }
}

/**
 * 应用站点数据到设置对象
 * @param {Object} siteSettings - 设置对象
 * @param {Object} data - 站点数据
 */
export function applySiteData(siteSettings, data) {
  // 基本信息
  siteSettings.basic.userPackageId = data.user_package_id || null
  siteSettings.basic.planName = data.user_package_id ? `套餐ID ${data.user_package_id}` : '商业版(飞扬)'
  siteSettings.basic.groupIds = data.group_ids || (data.group_id ? [data.group_id] : [])
  siteSettings.basic.groupName = siteSettings.basic.groupIds.length ? `分组ID ${siteSettings.basic.groupIds[0]}` : ''
  siteSettings.basic.domain = (data.domains || []).join('\n')
  siteSettings.basic.status = parseBool(data.enable, true)
  siteSettings.basic.httpEnable = !!(data.http_listen && data.http_listen.length)
  siteSettings.basic.httpPorts = (data.http_listen || []).join(' ')

  // 源站设置
  siteSettings.origin.list = (data.backends || []).map(b => ({
    address: b,
    weight: 1,
    enable: true
  }))
  siteSettings.origin.protocol = data.backend_protocol || 'follow'
  
  // HTTPS设置
  siteSettings.https.enable = !!(data.https_listen && data.https_listen.length)
  siteSettings.https.listenPorts = (data.https_listen || []).join(' ')
  siteSettings.https.certId = data.cert_id || null

  // 应用其他设置的逻辑...
}

/**
 * 转换缓存预设数据
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
 * 转换站点设置为表单数据
 * @param {Object} data - 站点数据
 * @returns {Object} 转换后的设置
 */
export function transformSiteToSettings(data) {
  return {
    basic: {
      status: parseBool(data.enable, true),
      domain: (data.domains || []).join('\n'),
      httpEnable: !!(data.http_listen && data.http_listen.length),
      httpPorts: (data.http_listen || []).join(' ')
    },
    origin: {
      list: (data.backends || []).map(b => ({
        address: b,
        weight: 1,
        enable: true
      }))
    },
    https: {
      enable: !!(data.https_listen && data.https_listen.length),
      listenPorts: (data.https_listen || []).join(' '),
      certId: data.cert_id || null
    }
  }
}
