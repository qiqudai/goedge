import { reactive, ref, computed, watch } from 'vue'
import { debounce } from 'lodash-es'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import {
  toStr, parseBool, parseList,
  convertSecondsToUnit, convertUnitToSeconds,
  normalizeCacheRule, formatTTL
} from '@/views/global/utils'

import { useLoading } from './useLoading'

const { withLoading, loading, loadingText } = useLoading()

// 全局共享状态 (单例模式)
let isInitialized = false

// 全局配置数据结构
const globalSettings = reactive({
  site: {
    httpListen: '80',
    httpsListen: '443',
    httpsHsts: false,
    httpsHttp2: false,
    httpsHttp3: false,
    httpsForce: false,
    sslProtocols: ['TLSv1', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3'],
    sslCiphers: [
      'ECDHE-ECDSA-AES128-GCM-SHA256',
      'ECDHE-RSA-AES128-GCM-SHA256',
      'ECDHE-ECDSA-AES256-GCM-SHA384',
      'ECDHE-RSA-AES256-GCM-SHA384',
      'ECDHE-ECDSA-CHACHA20-POLY1305',
      'ECDHE-RSA-CHACHA20-POLY1305',
      'DHE-RSA-AES128-GCM-SHA256',
      'DHE-RSA-AES256-GCM-SHA384',
      'DHE-RSA-CHACHA20-POLY1305',
      'ECDHE-ECDSA-AES128-SHA256',
      'ECDHE-RSA-AES128-SHA256',
      'ECDHE-ECDSA-AES128-SHA',
      'ECDHE-RSA-AES128-SHA',
      'ECDHE-ECDSA-AES256-SHA384',
      'ECDHE-RSA-AES256-SHA384',
      'ECDHE-ECDSA-AES256-SHA',
      'ECDHE-RSA-AES256-SHA',
      'DHE-RSA-AES128-SHA256',
      'DHE-RSA-AES256-SHA256',
      'AES128-GCM-SHA256',
      'AES256-GCM-SHA384',
      'AES128-SHA256',
      'AES256-SHA256',
      'AES128-SHA',
      'AES256-SHA',
      'DES-CBC3-SHA'
    ].join(':'),
    sslPreferServerCiphers: true,
    ocspStapling: true,
    backendProtocol: 'http',
    backendHttpPort: '80',
    backendHttpsPort: '443',
    proxyTimeout: '60',
    connectTimeout: '10',
    proxySslProtocols: ['TLSv1', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3'],
    cacheRules: [],
    originHeaders: [],
    logRequestHeader: true,
    logResponseHeader: false,
    logRequestBody: false,
    postSizeLimit: '16',
    balanceWay: 'rr',
    ccDefaultRule: 10002,
    securityBot: 'allow',
    gzipEnable: true,
    gzipTypes: [
      'text/plain',
      'text/css',
      'text/xml',
      'text/javascript',
      'application/javascript',
      'application/x-javascript',
      'application/json'
    ].join(' '),
    websocketEnable: false,
    securityShieldProxy: false,
    realtimeReturn: false,
    realtimeSend: false,
    ipv6Enable: false
  },
  stream: {
    // TODO: 流媒体配置
  },
  cert: {
    // TODO: 证书配置
  },
  cache: {
    // TODO: 缓存配置
  }
})

// CC规则列表
const ccRules = ref([
  { label: '关闭', value: 10002 },
  { label: '宽松', value: 10003 },
  { label: 'JS验证', value: 10004 },
  { label: '5秒盾', value: 10005 },
  { label: '点击验证', value: 10006 },
  { label: '滑块验证', value: 10007 },
  { label: '验证码', value: 10008 },
  { label: '旋转图片', value: 10009 },
  { label: '点击验证(简单)', value: 10010 },
  { label: '滑块验证(简单)', value: 10011 },
  { label: '临时白名单', value: 10012 }
])

/**
 * 全局配置状态管理
 * 提供统一的状态管理和API调用逻辑
 */
export function useGlobalConfig() {
  // 自动保存逻辑
  const triggerSave = debounce(() => {
    if (isInitialized) {
      saveGlobalSettings()
    }
  }, 500) // 延迟500ms

  // 监听并同步
  watch(globalSettings, () => {
    if (!isInitialized) return
    triggerSave()
  }, { deep: true })

  // 保存全局设置
  const saveGlobalSettings = async () => {
    try {
      const sitePayload = {
        'http_listen-port': globalSettings.site.httpListen,
        'https_listen-port': globalSettings.site.httpsListen,
        'https_listen-hsts': globalSettings.site.httpsHsts,
        'https_listen-http2': globalSettings.site.httpsHttp2,
        'https_listen-http3': globalSettings.site.httpsHttp3,
        'https_listen-force_ssl_enable': globalSettings.site.httpsForce,
        'https_listen-ssl_protocols': globalSettings.site.sslProtocols.join(' '),
        'https_listen-ssl_ciphers': globalSettings.site.sslCiphers,
        'https_listen-ssl_prefer_server_ciphers': globalSettings.site.sslPreferServerCiphers ? 'on' : 'off',
        'https_listen-ocsp_stapling': globalSettings.site.ocspStapling,
        'backend_protocol': globalSettings.site.backendProtocol,
        'backend_http_port': globalSettings.site.backendHttpPort,
        'backend_https_port': globalSettings.site.backendHttpsPort,
        'proxy_timeout': globalSettings.site.proxyTimeout,
        'connect_timeout': globalSettings.site.connectTimeout,
        'proxy_ssl_protocols': globalSettings.site.proxySslProtocols.join(' '),
        'proxy_cache': JSON.stringify(globalSettings.site.cacheRules),
        'origin_headers': JSON.stringify(globalSettings.site.originHeaders),
        'log_request_header': globalSettings.site.logRequestHeader,
        'log_response_header': globalSettings.site.logResponseHeader,
        'log_request_body': globalSettings.site.logRequestBody,
        'post_size_limit': globalSettings.site.postSizeLimit,
        'balance_way': globalSettings.site.balanceWay,
        'cc_default_rule': globalSettings.site.ccDefaultRule,
        'security_bot': globalSettings.site.securityBot,
        'gzip_enable': globalSettings.site.gzipEnable,
        'gzip_types': globalSettings.site.gzipTypes,
        'websocket_enable': globalSettings.site.websocketEnable,
        'security_shield_proxy': globalSettings.site.securityShieldProxy,
        'realtime_send': globalSettings.site.realtimeSend,
        'realtime_return': globalSettings.site.realtimeReturn,
        'ipv6_enable': globalSettings.site.ipv6Enable
      }

      // 同时保存所有部分，这里先只保存site部分
      await request.post('/site_defaults', { scope_name: 'global', scope_id: 0, data: sitePayload })
      ElMessage.success('全局配置已保存')
    } catch (error) {
      ElMessage.error('保存失败: ' + error.message)
    }
  }

  // 加载全局配置
  const loadGlobalSettings = async () => {
    if (isInitialized) return
    isInitialized = true

    await withLoading(async () => {
      try {
        const res = await request.get('/site_defaults', { params: { scope_name: 'global', scope_id: 0 } })
        const data = res?.data?.list || []
        const map = {}
        data.forEach((item) => { map[item.name] = item.value })

        // 加载site配置
        if (map['http_listen-port'] !== undefined) globalSettings.site.httpListen = toStr(map['http_listen-port'], globalSettings.site.httpListen)
        if (map['https_listen-port'] !== undefined) globalSettings.site.httpsListen = toStr(map['https_listen-port'], globalSettings.site.httpsListen)
        globalSettings.site.httpsHsts = parseBool(map['https_listen-hsts'], globalSettings.site.httpsHsts)
        globalSettings.site.httpsHttp2 = parseBool(map['https_listen-http2'], globalSettings.site.httpsHttp2)
        globalSettings.site.httpsHttp3 = parseBool(map['https_listen-http3'], globalSettings.site.httpsHttp3)
        globalSettings.site.httpsForce = parseBool(map['https_listen-force_ssl_enable'], globalSettings.site.httpsForce)
        if (map['https_listen-ssl_protocols']) {
          globalSettings.site.sslProtocols = toStr(map['https_listen-ssl_protocols']).split(/\s+/).filter(Boolean)
        }
        if (map['https_listen-ssl_ciphers'] !== undefined) globalSettings.site.sslCiphers = toStr(map['https_listen-ssl_ciphers'], globalSettings.site.sslCiphers)
        globalSettings.site.sslPreferServerCiphers = parseBool(map['https_listen-ssl_prefer_server_ciphers'], globalSettings.site.sslPreferServerCiphers)
        globalSettings.site.ocspStapling = parseBool(map['https_listen-ocsp_stapling'], globalSettings.site.ocspStapling)
        if (map['backend_protocol']) globalSettings.site.backendProtocol = toStr(map['backend_protocol'], globalSettings.site.backendProtocol)
        if (map['backend_http_port'] !== undefined) globalSettings.site.backendHttpPort = toStr(map['backend_http_port'], globalSettings.site.backendHttpPort)
        if (map['backend_https_port'] !== undefined) globalSettings.site.backendHttpsPort = toStr(map['backend_https_port'], globalSettings.site.backendHttpsPort)
        if (map['proxy_timeout'] !== undefined) globalSettings.site.proxyTimeout = toStr(map['proxy_timeout'], globalSettings.site.proxyTimeout)
        if (map['connect_timeout'] !== undefined) globalSettings.site.connectTimeout = toStr(map['connect_timeout'], globalSettings.site.connectTimeout)
        if (map['proxy_ssl_protocols']) {
          globalSettings.site.proxySslProtocols = toStr(map['proxy_ssl_protocols']).split(/\s+/).filter(Boolean)
        }
        globalSettings.site.cacheRules = parseList(map['proxy_cache']).map(normalizeCacheRule).filter(Boolean)
        globalSettings.site.originHeaders = parseList(map['origin_headers']).map((item) => ({
          name: toStr(item?.name || item?.key || '', ''),
          value: toStr(item?.value || '', '')
        }))
        globalSettings.site.logRequestHeader = parseBool(map['log_request_header'], globalSettings.site.logRequestHeader)
        globalSettings.site.logResponseHeader = parseBool(map['log_response_header'], globalSettings.site.logResponseHeader)
        globalSettings.site.logRequestBody = parseBool(map['log_request_body'], globalSettings.site.logRequestBody)
        if (map['post_size_limit'] !== undefined) globalSettings.site.postSizeLimit = toStr(map['post_size_limit'], globalSettings.site.postSizeLimit)
        if (map['balance_way']) globalSettings.site.balanceWay = toStr(map['balance_way'], globalSettings.site.balanceWay)
        if (map['cc_default_rule'] !== undefined) globalSettings.site.ccDefaultRule = Number(map['cc_default_rule']) || globalSettings.site.ccDefaultRule
        if (map['security_bot'] !== undefined) globalSettings.site.securityBot = toStr(map['security_bot'], globalSettings.site.securityBot)
        globalSettings.site.gzipEnable = parseBool(map['gzip_enable'], globalSettings.site.gzipEnable)
        if (map['gzip_types'] !== undefined) globalSettings.site.gzipTypes = toStr(map['gzip_types'], globalSettings.site.gzipTypes)
        globalSettings.site.websocketEnable = parseBool(map['websocket_enable'], globalSettings.site.websocketEnable)
        globalSettings.site.securityShieldProxy = parseBool(map['security_shield_proxy'], globalSettings.site.securityShieldProxy)
        globalSettings.site.realtimeSend = parseBool(map['realtime_send'], globalSettings.site.realtimeSend)
        globalSettings.site.realtimeReturn = parseBool(map['realtime_return'], globalSettings.site.realtimeReturn)
        globalSettings.site.ipv6Enable = parseBool(map['ipv6_enable'], globalSettings.site.ipv6Enable)

        // 加载CC规则
        await loadCcRules()
      } catch (error) {
        console.error('[GlobalConfig] Load error:', error)
      }
    })
  }

  // 加载CC规则
  const loadCcRules = async () => {
    try {
      const res = await request.get('/rules/cc/groups')
      const list = res?.data?.data?.list || []
      if (list.length > 0) ccRules.value = list.map((item) => ({ label: item.name, value: item.id }))
    } catch (error) {
      console.warn('[GlobalConfig] Load CC rules failed', error)
    }
  }

  // 返回所有状态和方法
  return {
    // 状态
    globalSettings,
    loading,
    ccRules,

    // 方法
    loadGlobalSettings,
    saveGlobalSettings,
    loadCcRules
  }
}