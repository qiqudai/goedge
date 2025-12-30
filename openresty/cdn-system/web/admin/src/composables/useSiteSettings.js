import { reactive, ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { debounce } from 'lodash-es'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import { formatDate, parseBool } from '@/utils/helpers'
import { buildSettingsPayload } from '@/utils/configTransform'

import { useLoading } from './useLoading'

const { withLoading, loading, loadingText } = useLoading()

// 全局共享状态 (单例模式)
const isInitialized = ref(false)
const activeTab = ref('basic')
const site = ref(null)

// 证书列表和套餐列表
const certList = ref([])
const userPackages = ref([])
const aclList = ref([])

// 网站设置数据结构
const siteSettings = reactive({
  basic: {
    status: true,
    domain: '',
    cname: '',
    httpEnable: true,
    httpPorts: '80',
    userPackageId: null,
    groupId: null,
    expireTime: '-',
    createdAt: '-',
    updatedAt: '-',
    originList: [],
    originConditions: []
  },
  origin: {
    list: [],
    conditions: [],
    protocol: 'follow',
    httpPort: '80',
    httpsPort: '443',
    timeout: 60,
    connTimeout: 10,
    healthCheckEnabled: true,
    healthCheckHost: '',
    healthCheckPath: '/',
    healthCheckStatus: '200 301 302',
    healthCheckInterval: 60
  },
  https: {
    enable: false,
    listenPorts: '443',
    certId: null,
    force: false,
    forcePort: '443',
    hsts: false,
    http2: false,
    http3: false,
    ocsp: false,
    sslPolicy: 'compat',
    sslCiphers: '',
    sslProtocols: ''
  },
  security: {
    cc: {
      mode: 10002,
      autoSwitch: {
        enable: false,
        qps: 200,
        rule: 'close'
      },
      customRules: []
    },
    crawlers: {
      action: 'none'
    },
    ip: {
      white: '',
      black: '',
      blackTime: 3600,
      blackTimeCustom: false,
      whiteTime: 21600,
      whiteTimeCustom: false
    },
    ua: { white: '', black: '' },
    cookie: { enable: false, domain: '' },
    regions: [],
    block: {
      transparentProxy: false
    }
  },
  cache: { rules: [] },
  access: {
    acl: '',
    hotlink: {
      enable: false,
      scope: 'all',
      value: '',
      allowEmpty: false,
      domains: ''
    },
    cors: {
      enable: false,
      allowOrigin: '*',
      allowMethods: '*',
      allowHeaders: '*',
      exposeHeaders: '*',
      maxAge: 1728000
    },
    regionBlock: {
      mode: 'disabled',
      countries: []
    }
  },
  advanced: {
    uploadLimitMode: 'none',
    uploadLimitValue: 100,
    gzip: false,
    websocket: false,
    searchEngineOrigin: false,
    searchEngineOriginIp: '',
    urlRewrites: [],
    reqHeaders: [],
    resHeaders: [],
    originCert: false,
    realtimeIdentify: false,
    realtimeSend: false,
    logRequestHeader: false,
    logResponseHeader: false,
    logRequestBody: false,
    logRequestBodySizeLimit: 16,
    defaultSite: false,
    l2Config: 'current'
  }
})

// 初始化控制
let isInitializing = false

/**
 * 网站设置状态管理
 * 提供统一的状态管理和API调用逻辑
 */
export function useSiteSettings() {
  const route = useRoute()
  const siteId = computed(() => {
    return parseInt(route.query.site_id || route.params.site_id || 0, 10)
  })

  // 自动保存逻辑
  const triggerSave = debounce(() => {
    if (isInitialized.value) {
      saveSettings()
    }
  }, 200) // 延迟缩短为 500ms

  // 监听并同步
  watch(siteSettings, (newVal) => {
    // 始终进行数据一致性检查
    if (newVal.basic.originList && newVal.basic.originList !== newVal.origin.list) {
      newVal.origin.list = newVal.basic.originList
    }
    if (newVal.basic.originConditions && newVal.basic.originConditions !== newVal.origin.conditions) {
      newVal.origin.conditions = newVal.basic.originConditions
    }

    // 监听跳转端口变化，确保有效
    if (newVal.https.listenPorts && newVal.https.enable) {
      const ports = (newVal.https.listenPorts || '').split(' ').filter(Boolean)
      if (ports.length > 0 && !ports.includes(newVal.https.forcePort)) {
        newVal.https.forcePort = ports[0]
      }
    }

    if (!isInitialized.value) return
    if (!isInitialized.value) return
    // triggerSave() // Disabled global auto-save to support save-on-blur
  }, { deep: true })

  // 保存设置
  const saveSettings = async (isAutoSave = false) => {
    if (!siteId.value) return

    const doSave = async () => {
      try {
        const payload = {
          ids: [siteId.value],
          settings: buildSettingsPayload(siteSettings),
          enable: siteSettings.basic.status,
          http_listen: siteSettings.basic.httpEnable ? splitStr(siteSettings.basic.httpPorts) : [],
          https_listen: siteSettings.https.enable ? splitStr(siteSettings.https.listenPorts) : [],
          user_package_id: siteSettings.basic.userPackageId || 0,
          group_id: siteSettings.basic.groupId || 0,
          domains: splitStr(siteSettings.basic.domain),
          backends: siteSettings.origin.list.map(o => o.address).filter(Boolean),
          backend_protocol: siteSettings.origin.protocol
        }
        await request.put(`/sites/${siteId.value}`, payload)
        if (!isAutoSave) {
          ElMessage.success('配置已保存')
        } else {
          // 自动保存时可以选择不弹窗，或者使用更轻量的提示，这里暂且保留但可考虑 console.log
          // ElMessage.success({ message: '已自动保存', type: 'success', duration: 1000 }) 
        }
      } catch (error) {
        ElMessage.error('保存失败: ' + error.message)
      }
    }

    if (isAutoSave) {
      // 自动保存不显示全屏 loading
      await doSave()
    } else {
      // 手动保存（如果有的话）显示 loading
      await withLoading(doSave, '正在保存配置...')
    }
  }

  // 初始化方法
  const init = async () => {
    if (isInitializing) return
    isInitializing = true
    isInitialized.value = false

    // 不要在这里重置 loadingCount，让 withLoading 自己处理计数
    await withLoading(async () => {
      try {
        console.log('[SiteSettings] Starting init for site:', siteId.value)

        // 加载基本数据
        const res = await request.get(`/sites/${siteId.value}`)
        const data = res.data?.site || res.site || res.data || res
        if (data && data.id) {
          site.value = data
          applySiteData(data)
        }

        // 加载辅助数据 (允许部分失败，静默处理)
        const loadAux = async (url, refVar) => {
          try {
            const r = await request.get(url)
            refVar.value = r.data?.list || r.list || []
          } catch (e) {
            console.warn(`[SiteSettings] Load ${url} failed`, e)
          }
        }

        await Promise.allSettled([
          loadAux('/certs', certList),
          loadAux('/user_packages', userPackages),
          loadAcls()
        ])
      } catch (error) {
        console.error('[SiteSettings] Init error:', error)
      } finally {
        isInitializing = false
        // 延迟开启保存监听，避开初始化的响应式数据波峰
        setTimeout(() => {
          isInitialized.value = true
          console.log('[SiteSettings] Init complete, auto-save enabled.')
        }, 1500)
      }
    })
  }

  // 计算属性：分发站点数据
  const applySiteData = (data) => {
    // 基本信息
    siteSettings.basic.userPackageId = data.user_package_id || null
    siteSettings.basic.groupId = data.group_id || null
    siteSettings.basic.domain = (data.domains || []).join('\n')
    siteSettings.basic.status = parseBool(data.enable, true)
    siteSettings.basic.httpEnable = !!(data.http_listen && data.http_listen.length)
    siteSettings.basic.httpPorts = (data.http_listen || []).join(' ')
    
    // 源站设置
    let finalOriginList = []
    if (data.settings?.origin?.list && Array.isArray(data.settings.origin.list)) {
      finalOriginList = data.settings.origin.list.map(o => ({
        address: o.address || '',
        weight: o.weight || '10',
        enable: o.enable !== false
      }))
    } else {
      finalOriginList = (data.backends || []).map(b => ({
        address: b,
        weight: '10',
        enable: true
      }))
    }
    siteSettings.origin.list = finalOriginList
    siteSettings.basic.originList = JSON.parse(JSON.stringify(finalOriginList))

    let finalConditions = []
    if (data.settings?.origin?.conditions && Array.isArray(data.settings.origin.conditions)) {
      finalConditions = data.settings.origin.conditions
    }
    siteSettings.origin.conditions = finalConditions
    siteSettings.basic.originConditions = JSON.parse(JSON.stringify(finalConditions))

    // 回源详情设置
    const s = data.settings || {}
    const sOrigin = s.origin || {}

    siteSettings.origin.protocol = data.backend_protocol || s.backend_protocol || sOrigin.protocol || 'follow'

    // 映射 host
    const rawHost = s.origin_host || sOrigin.host || 'follow'
    if (['follow', 'domain'].includes(rawHost)) {
      siteSettings.origin.host = rawHost
      siteSettings.origin.hostValue = ''
    } else {
      siteSettings.origin.host = 'custom'
      siteSettings.origin.hostValue = rawHost
    }

    siteSettings.origin.httpPort = s.origin_http_port || data.origin_http_port || sOrigin.httpPort || '80'
    siteSettings.origin.httpsPort = s.origin_https_port || data.origin_https_port || sOrigin.httpsPort || '443'
    siteSettings.origin.timeout = parseInt(s.origin_timeout || data.origin_timeout || sOrigin.timeout || 60)
    siteSettings.origin.connTimeout = parseInt(sOrigin.connTimeout || 10)

    // 健康检查
    siteSettings.origin.healthCheckEnabled = sOrigin.health_check !== false
    siteSettings.origin.healthCheckHost = sOrigin.health_host || ''
    siteSettings.origin.healthCheckPath = sOrigin.health_path || '/'
    siteSettings.origin.healthCheckStatus = sOrigin.health_status || '200 301 302'
    siteSettings.origin.healthCheckInterval = parseInt(sOrigin.health_interval || 60)

    // HTTPS设置
    siteSettings.https.enable = !!(data.https_listen && data.https_listen.length)
    siteSettings.https.listenPorts = (data.https_listen || []).join(' ')
    siteSettings.https.certId = data.cert_id || null
    
    // 如果 settings 中有明确配置，则覆盖
    if (data.settings?.https) {
      const h = data.settings.https
      if (h.enable !== undefined) {
        siteSettings.https.enable = parseBool(h.enable, false)
      }
    }

    // 基本设置中显示的字段
    siteSettings.basic.cname = data.cname || data.domain_cname || ''
    siteSettings.basic.expireTime = data.expire_time || data.user_plan_expire_time || '-'
    siteSettings.basic.createdAt = data.created_at ? formatDate(data.created_at) : '-'
    siteSettings.basic.updatedAt = data.updated_at ? formatDate(data.updated_at) : '-'
    
    // 将源站数据也映射到basic中，供BasicConfig组件使用
    siteSettings.basic.originList = siteSettings.origin.list
    siteSettings.basic.originConditions = siteSettings.origin.conditions
    
    // 映射 Settings 中的高级配置
    if (data.settings && typeof data.settings === 'object') {
      const s = data.settings

      // HTTPS 高级设置
      siteSettings.https.force = !!s.https?.force || !!s.force
      siteSettings.https.forcePort = s.https?.redirect_port || s.redirect_port || '443'
      siteSettings.https.hsts = !!s.https?.hsts || !!s.hsts
      siteSettings.https.http2 = !!s.https?.http2 || !!s.http2
      siteSettings.https.http3 = !!s.https?.http3 || !!s.http3
      siteSettings.https.ocsp = !!s.https?.ocsp_stapling || !!s.ocsp_stapling
      siteSettings.https.sslPolicy = s.https?.ssl_profile || s.ssl_profile || 'compat'
      siteSettings.https.sslCiphers = s.https?.ssl_ciphers || s.ssl_ciphers || ''
      siteSettings.https.sslProtocols = s.https?.ssl_protocols || s.ssl_protocols || ''

      // 高级设置
      siteSettings.advanced.gzip = !!s.gzip
      siteSettings.advanced.websocket = !!s.websocket
      siteSettings.advanced.searchEngineOrigin = !!s.search_engine_origin
      siteSettings.advanced.searchEngineOriginIp = s.search_engine_origin_ip || ''
      siteSettings.advanced.uploadLimitMode = (s.upload_limit && s.upload_limit > 0) ? 'custom' : 'none'
      siteSettings.advanced.uploadLimitValue = s.upload_limit || 100
      siteSettings.advanced.logRequestHeader = !!s.log_request_header
      siteSettings.advanced.logResponseHeader = !!s.log_response_header
      siteSettings.advanced.logRequestBody = !!s.log_request_body
      siteSettings.advanced.logRequestBodySizeLimit = s.log_request_body_size_limit || 16

      siteSettings.advanced.originCert = !!s.origin_cert
      siteSettings.advanced.realtimeIdentify = !!s.realtime_identify
      siteSettings.advanced.realtimeSend = !!s.realtime_send

      siteSettings.advanced.defaultSite = !!s.default_site
      siteSettings.advanced.l2Config = s.l2_config || 'current'

      // 缓存设置
      if (s.cache?.rules) {
        siteSettings.cache.rules = s.cache.rules
      }

      // 安全设置
      if (s.security) {
        const sec = s.security
        siteSettings.security.cc.mode = sec.default_rule || 10002
        if (sec.auto_switch) {
          try {
            siteSettings.security.cc.autoSwitch = JSON.parse(sec.auto_switch)
          } catch (e) {
            console.error('Parse auto_switch failed', e)
          }
        }

        // Custom Rules
        siteSettings.security.cc.customRules = sec.custom_rules || []

        // Crawlers
        siteSettings.security.crawlers.action = sec.crawlers_action || 'none'

        // IP List Timeouts
        siteSettings.security.ip.black = (sec.blacklist || []).join('\n')
        siteSettings.security.ip.white = (sec.whitelist || []).join('\n')

        const blackTime = parseInt(sec.ip_black_timeout)
        if (blackTime > 0 && blackTime !== 3600) {
          siteSettings.security.ip.blackTime = blackTime
          siteSettings.security.ip.blackTimeCustom = true
        } else {
          siteSettings.security.ip.blackTime = 3600
          siteSettings.security.ip.blackTimeCustom = false
        }

        const whiteTime = parseInt(sec.ip_white_timeout)
        if (whiteTime > 0 && whiteTime !== 21600) {
          siteSettings.security.ip.whiteTime = whiteTime
          siteSettings.security.ip.whiteTimeCustom = true
        } else {
          siteSettings.security.ip.whiteTime = 21600
          siteSettings.security.ip.whiteTimeCustom = false
        }

        siteSettings.security.regions = sec.region_block || []

        // Block Settings
        siteSettings.security.block.transparentProxy = !!sec.block_transparent_proxy
      }

      // 访问控制
      if (s.access) {
        const acc = s.access
        siteSettings.access.acl = acc.acl || ''
        if (acc.hotlink) {
          siteSettings.access.hotlink = { ...siteSettings.access.hotlink, ...acc.hotlink }
        }
        if (acc.cors) {
          siteSettings.access.cors = { ...siteSettings.access.cors, ...acc.cors }
        }
        if (acc.region_block) {
          siteSettings.access.regionBlock = { ...siteSettings.access.regionBlock, ...acc.region_block }
        }
      }

      // 列表类数据
      siteSettings.advanced.urlRedirects = s.url_redirects || []
      siteSettings.advanced.reqHeaders = s.req_headers || []
      siteSettings.advanced.resHeaders = s.res_headers || []
      siteSettings.advanced.urlRewrites = s.url_rewrites || []
    }
  }

  // 工具函数：分割字符串
  const splitStr = (str) => (str || '').split(/[\s\n]+/).filter(Boolean)


  // 辅助加载函数 (带 loading 效果)
  const loadSite = () => init()

  const loadCerts = () => withLoading(async () => {
    try {
      const r = await request.get('/certs')
      certList.value = r.data?.list || r.list || []
    } catch (e) { console.error('Load certs failed', e) }
  })

  const loadUserPackages = () => withLoading(async () => {
    try {
      const r = await request.get('/user_packages')
      userPackages.value = r.data?.list || r.list || []
    } catch (e) { console.error('Load user packages failed', e) }
  })

  const loadAcls = () => withLoading(async () => {
    try {
      const r = await request.get('/acls', { baseURL: '/api/v1' })
      aclList.value = r.data?.list || r.list || []
    } catch (e) { console.error('Load acls failed', e) }
  })

  // 计算证书剩余天数
  const calcCertDays = (certId) => {
    const c = certList.value.find(i => i.id === certId)
    if (!c || !c.end_time) return null
    const end = new Date(c.end_time).getTime()
    const now = new Date().getTime()
    return Math.max(0, Math.ceil((end - now) / (1000 * 60 * 60 * 24)))
  }

  // 返回所有状态和方法
  return {
    // 状态
    site,
    siteSettings,
    loading,
    activeTab,
    certList,
    userPackages,
    aclList,
    
    // 方法
    loadSite,
    saveSettings,
    loadCerts,
    loadUserPackages,
    loadAcls,
    calcCertDays,
    
    // 计算属性
    siteId
  }
}
