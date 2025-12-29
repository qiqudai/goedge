<template>
  <div class="site-manage">
    <el-page-header v-if="site" @back="goBack" class="page-header" content="网站配置" style="margin-bottom: 16px;">
      <template #title>
        <span>{{ site.domains?.[0] || site.domain_raw || '网站' }}</span>
      </template>
      <template #content>
        <span>ID {{ site.id }} · {{ siteSettings.basic.cname }}</span>
      </template>
    </el-page-header>

    <el-card class="page-card" v-loading="loading">
      <el-tabs v-model="activeTab" class="manage-tabs" type="border-card">
        <el-tab-pane label="基本信息" name="basic">
          <el-descriptions border column="3" class="mb-16">
            <el-descriptions-item label="域名" :span="2">
              {{ siteSettings.basic.domain || '未配置' }}
            </el-descriptions-item>
            <el-descriptions-item label="CNAME">
              {{ siteSettings.basic.cname }}
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="siteSettings.basic.status ? 'success' : 'warning'">
                {{ siteSettings.basic.status ? '运行中' : '已停用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ siteSettings.basic.createdAt }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ siteSettings.basic.updatedAt }}</el-descriptions-item>
          </el-descriptions>
          <el-form label-width="120px" class="config-form">
            <el-form-item label="站点状态">
              <el-switch
                v-model="siteSettings.basic.status"
                @change="toggleSiteStatus"
                active-text="启用"
                inactive-text="停用"
              />
            </el-form-item>
            <el-form-item label="套餐">
              <el-input v-model="siteSettings.basic.planName" disabled />
            </el-form-item>
            <el-form-item label="所属分组">
              <el-input v-model="siteSettings.basic.groupName" disabled />
            </el-form-item>
            <el-form-item label="区域/线路">
              <el-input v-model="siteSettings.basic.nodeGroupName" disabled />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="回源设置" name="origin">
          <div class="section-title">源站列表</div>
          <el-table :data="siteSettings.origin.list" border size="small" style="margin-bottom: 12px;">
            <el-table-column prop="address" label="源地址">
              <template #default="{ row }">
                <el-input v-model="row.address" placeholder="IP 或域名" size="small" />
              </template>
            </el-table-column>
            <el-table-column prop="weight" label="权重" width="120">
              <template #default="{ row }">
                <el-input v-model="row.weight" size="small" />
              </template>
            </el-table-column>
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-switch v-model="row.enable" active-text="启用" inactive-text="停用" size="small" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ $index }">
                <el-button type="text" size="small" @click="removeOrigin($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button size="small" type="primary" @click="addOrigin">新增源站</el-button>

          <el-divider />
          <div class="section-title">条件源站</div>
          <el-table :data="siteSettings.origin.conditions" border size="small" style="margin-bottom: 12px;">
            <el-table-column label="匹配项" width="180">
              <template #default="{ row }">
                <el-select
                  v-model="row.item"
                  size="small"
                  placeholder="请选择"
                  @change="handleOriginConditionChange(row)"
                >
                  <el-option
                    v-for="opt in originConditionItems"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="条件" min-width="260">
              <template #default="{ row }">
                <div class="condition-origin-row">
                  <el-input
                    v-if="isOriginHeaderItem(row.item)"
                    v-model="row.header"
                    size="small"
                    placeholder="请求头名称，如 user-agent"
                  />
                  <el-input
                    v-else-if="isOriginStatItem(row.item)"
                    v-model="row.seconds"
                    size="small"
                    placeholder="统计秒数"
                  />
                  <el-input
                    v-else
                    v-model="row.value"
                    size="small"
                    :placeholder="getOriginConditionPlaceholder(row)"
                  />
                  <el-select
                    v-if="!isOriginStatItem(row.item)"
                    v-model="row.operator"
                    size="small"
                    placeholder="匹配方式"
                    style="width: 140px;"
                  >
                    <el-option
                      v-for="opt in originConditionOperators"
                      :key="opt.value"
                      :label="opt.label"
                      :value="opt.value"
                    />
                  </el-select>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="源站" min-width="220">
              <template #default="{ row }">
                <el-input v-model="row.origin" placeholder="源站地址，多个用 | 分隔" size="small" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ $index }">
                <el-button type="text" size="small" @click="removeConditionOrigin($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button size="small" type="primary" @click="addConditionOrigin">新增条件源站</el-button>

          <el-divider />
          <div class="section-title">回源健康检查</div>
          <el-form label-width="150px" class="config-form">
            <el-form-item label="启用健康检查">
              <el-switch v-model="siteSettings.origin.healthCheckEnabled" />
            </el-form-item>
            <el-form-item label="检查地址">
              <el-input v-model="siteSettings.origin.healthCheckHost" placeholder="域名或 IP" />
            </el-form-item>
            <el-form-item label="检查路径">
              <el-input v-model="siteSettings.origin.healthCheckPath" placeholder="/" />
            </el-form-item>
            <el-form-item label="有效状态码">
              <el-input v-model="siteSettings.origin.healthCheckStatus" placeholder="200 301 302" />
            </el-form-item>
            <el-form-item label="检测间隔(秒)">
              <el-input v-model="siteSettings.origin.healthCheckInterval" placeholder="60" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="HTTPS 设置" name="https">
          <el-form label-width="150px" class="config-form">
            <el-form-item label="HTTPS 开启">
              <el-switch v-model="siteSettings.https.enabled" />
            </el-form-item>
            <template v-if="siteSettings.https.enabled">
              <el-form-item label="监听端口">
                <el-input v-model="siteSettings.https.port" placeholder="443" />
              </el-form-item>
              <el-form-item label="证书选择">
                <el-select v-model="siteSettings.https.certId" placeholder="请选择证书">
                  <el-option v-for="cert in certList" :key="cert.id" :label="cert.domain" :value="cert.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="强制 HTTPS">
                <el-switch v-model="siteSettings.https.force" />
              </el-form-item>
              <el-form-item label="跳转端口" v-if="siteSettings.https.force">
                <el-input v-model="siteSettings.https.redirectPort" placeholder="443" />
              </el-form-item>
              <el-form-item label="开启 HSTS">
                <el-switch v-model="siteSettings.https.hsts" />
              </el-form-item>
              <el-form-item label="开启 HTTP2">
                <el-switch v-model="siteSettings.https.http2" />
              </el-form-item>
              <el-form-item label="开启 HTTP3">
                <el-switch v-model="siteSettings.https.http3" />
              </el-form-item>
              <el-form-item label="OCSP Stapling">
                <el-switch v-model="siteSettings.https.ocspStapling" />
              </el-form-item>
              <el-form-item label="SSL 协议" style="max-width: 600px;">
                <el-checkbox-group v-model="siteSettings.https.sslProtocols">
                  <el-checkbox v-for="proto in sslProtocolOptions" :key="proto" :label="proto">
                    {{ proto }}
                  </el-checkbox>
                </el-checkbox-group>
              </el-form-item>
              <el-form-item label="SSL 密钥套件">
                <el-input
                  type="textarea"
                  :rows="2"
                  v-model="siteSettings.https.sslCiphers"
                  placeholder="请输入 SSL ciphers"
                />
              </el-form-item>
              <el-form-item label="Prefer Server Ciphers">
                <el-switch v-model="siteSettings.https.sslPreferServerCiphers" />
              </el-form-item>
              <el-form-item label="SSL 配置">
                <el-radio-group v-model="siteSettings.https.sslProfile">
                  <el-radio value="compat">兼容旧浏览器</el-radio>
                  <el-radio value="modern">兼容大部分浏览器</el-radio>
                  <el-radio value="custom">自定义</el-radio>
                </el-radio-group>
              </el-form-item>
            </template>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="安全设置" name="security">
          <el-form label-width="150px" class="config-form">
            <el-form-item label="默认 CC 规则">
              <el-select v-model="siteSettings.security.defaultRule">
                <el-option v-for="item in ccRules" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="自动切换">
              <el-switch v-model="siteSettings.security.autoSwitch" />
            </el-form-item>
            <el-form-item label="搜索引擎爬虫">
              <el-radio-group v-model="siteSettings.security.bot">
                <el-radio value="none">不设置</el-radio>
                <el-radio value="allow">放行</el-radio>
                <el-radio value="block">拦截</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="黑名单">
              <el-input
                type="textarea"
                :rows="3"
                v-model="siteSettings.security.blacklist"
                placeholder="每行一个 IP"
              />
            </el-form-item>
            <el-form-item label="白名单">
              <el-input
                type="textarea"
                :rows="3"
                v-model="siteSettings.security.whitelist"
                placeholder="每行一个 IP"
              />
            </el-form-item>
            <el-form-item label="黑名单单时效">
              <el-radio-group v-model="siteSettings.security.blackTimeMode">
                <el-radio value="system">系统默认</el-radio>
                <el-radio value="custom">自定义</el-radio>
              </el-radio-group>
              <el-input
                v-if="siteSettings.security.blackTimeMode === 'custom'"
                v-model="siteSettings.security.blackTimeCustom"
                placeholder="请输入秒数"
                style="margin-top: 6px; width: 180px;"
              />
            </el-form-item>
            <el-form-item label="白名单单时效">
              <el-radio-group v-model="siteSettings.security.whiteTimeMode">
                <el-radio value="system">系统默认</el-radio>
                <el-radio value="custom">自定义</el-radio>
              </el-radio-group>
              <el-input
                v-if="siteSettings.security.whiteTimeMode === 'custom'"
                v-model="siteSettings.security.whiteTimeCustom"
                placeholder="请输入秒数"
                style="margin-top: 6px; width: 180px;"
              />
            </el-form-item>
            <el-form-item label="屏蔽透明代理">
              <el-switch v-model="siteSettings.security.shieldProxy" />
            </el-form-item>
            <el-form-item label="区域屏蔽">
              <el-select v-model="siteSettings.security.regionMode" @change="handleRegionModeChange">
                <el-option label="不设置" value="none" />
                <el-option label="国外（不含港澳台）" value="overseas_without_hk" />
                <el-option label="国外（含港澳台）" value="overseas_with_hk" />
                <el-option label="中国（含港澳台）" value="china_with_hk" />
                <el-option label="中国（不含港澳台）" value="china_without_hk" />
                <el-option label="自定义" value="custom" />
              </el-select>
              <country-selector
                v-if="siteSettings.security.regionMode === 'custom'"
                v-model="siteSettings.security.regionCustom"
              />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="缓存设置" name="cache">
          <div class="toolbar-row" style="margin-bottom: 12px;">
            <el-button type="primary" size="small" @click="openCacheRuleDialog('create')">新增规则</el-button>
            <el-select
              v-model="cacheQuickPreset"
              placeholder="快速添加缓存"
              size="small"
              style="width: 150px; margin-left: 12px;"
              @change="applyCachePreset"
            >
              <el-option label="首页缓存" value="index" />
              <el-option label="全站缓存" value="all" />
              <el-option label="静态资源缓存" value="static" />
              <el-option label="视频资源" value="video" />
              <el-option label="Wordpress 缓存" value="wordpress" />
            </el-select>
          </div>
          <el-table :data="siteSettings.cache.rules" border size="small">
            <el-table-column label="类型" min-width="120">
              <template #default="{ row }">{{ cacheTypeLabel(row.type) }}</template>
            </el-table-column>
            <el-table-column label="内容" min-width="240" prop="value" />
            <el-table-column label="TTL(秒)" width="120" prop="ttl" />
            <el-table-column label="操作" width="140">
              <template #default="{ row, $index }">
                <el-button type="text" size="small" @click="openCacheRuleDialog('edit', row, $index)">编辑</el-button>
                <el-button type="text" size="small" @click="removeCacheRule($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="访问控制" name="access">
          <el-form label-width="150px" class="config-form">
            <el-form-item label="ACL 设置">
              <el-select v-model="siteSettings.access.acl" placeholder="请选择">
                <el-option label="不设置" value="" />
                <el-option label="仅白名单" value="whitelist" />
                <el-option label="仅黑名单" value="blacklist" />
              </el-select>
            </el-form-item>
            <el-form-item label="防盗链">
              <el-switch v-model="siteSettings.access.hotlink" />
            </el-form-item>
            <el-form-item label="跨域访问">
              <el-switch v-model="siteSettings.access.cors" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="高级设置" name="advanced">
          <el-form label-width="180px" class="config-form">
            <el-form-item label="开启 Gzip">
              <el-switch v-model="siteSettings.advanced.gzip" />
            </el-form-item>
            <el-form-item label="开启 Websocket">
              <el-switch v-model="siteSettings.advanced.websocket" />
            </el-form-item>
            <el-form-item label="开启 IPv6">
              <el-switch v-model="siteSettings.advanced.ipv6" />
            </el-form-item>
            <el-form-item label="日志 - 请求头">
              <el-switch v-model="siteSettings.advanced.logRequestHeader" />
            </el-form-item>
            <el-form-item label="日志 - 响应头">
              <el-switch v-model="siteSettings.advanced.logResponseHeader" />
            </el-form-item>
            <el-form-item label="日志 - 请求体">
              <el-switch v-model="siteSettings.advanced.logRequestBody" />
            </el-form-item>
            <el-form-item label="请求体大小限制 (KB)">
              <el-input v-model="siteSettings.advanced.bodyLimit" placeholder="16" />
            </el-form-item>
            <el-form-item label="实时回源">
              <el-switch v-model="siteSettings.advanced.realtimeReturn" />
            </el-form-item>
            <el-form-item label="实时推送">
              <el-switch v-model="siteSettings.advanced.realtimeSend" />
            </el-form-item>
            <el-form-item label="ACME 回源">
              <el-switch v-model="siteSettings.advanced.acmeBacksource" />
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>

      <div class="site-manage-actions">
        <el-button type="primary" :loading="saving" @click="saveSettings">保存配置</el-button>
        <el-button plain @click="loadSite">重新加载</el-button>
      </div>
    </el-card>

    <el-dialog
      v-model="cacheRuleDialog.visible"
      :title="cacheRuleDialog.mode === 'edit' ? '编辑缓存规则' : '新增缓存规则'"
      width="520px"
    >
      <el-form label-width="120px">
        <el-form-item label="类型">
          <el-select v-model="cacheRuleForm.type">
            <el-option label="首页" value="index" />
            <el-option label="全站" value="all" />
            <el-option label="目录" value="dir" />
            <el-option label="后缀" value="suffix" />
            <el-option label="路径" value="path" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="cacheRuleForm.value" placeholder="支持正则或路径" />
        </el-form-item>
        <el-form-item label="TTL">
          <el-input v-model="cacheRuleForm.ttl" placeholder="单位：秒" />
        </el-form-item>
        <el-form-item label="忽略参数">
          <el-switch v-model="cacheRuleForm.ignore_query" />
        </el-form-item>
        <el-form-item label="强制缓存">
          <el-switch v-model="cacheRuleForm.force_cache" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="small" @click="cacheRuleDialog.visible = false">取消</el-button>
        <el-button size="small" type="primary" @click="saveCacheRule">保存规则</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import CountrySelector from '@/components/CountrySelector.vue'
import request from '@/utils/request'

const route = useRoute()
const router = useRouter()
const activeTab = ref('basic')
const loading = ref(false)
const saving = ref(false)
const cacheQuickPreset = ref('')
const certList = ref([])
const site = ref(null)
const sslProtocolOptions = ['SSLv2', 'SSLv3', 'TLSv1', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3']
const defaultSslProtocols = ['TLSv1', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3']
const defaultSslCiphers = [
  'ECDHE-ECDSA-AES128-GCM-SHA256',
  'ECDHE-RSA-AES128-GCM-SHA256',
  'ECDHE-ECDSA-AES256-GCM-SHA384',
  'ECDHE-RSA-AES256-GCM-SHA384',
  'ECDHE-ECDSA-CHACHA20-POLY1305',
  'ECDHE-RSA-CHACHA20-POLY1305'
].join(':')
const siteSettings = reactive(createDefaultSettings())
const cacheRuleDialog = reactive({ visible: false, mode: 'create', index: -1 })
const cacheRuleForm = reactive({ type: 'index', value: '', ttl: '86400', ignore_query: false, force_cache: false })
const siteId = computed(() => parseInt(route.query.site_id || route.params.site_id || 0, 10))

const ccRules = ref([
  { label: '关闭', value: 10002 },
  { label: '宽松', value: 10003 },
  { label: 'JS 验证', value: 10004 },
  { label: '5 秒盾', value: 10005 },
  { label: '点击验证', value: 10006 },
  { label: '滑块验证', value: 10007 },
  { label: '验证码', value: 10008 }
])
const cacheTypeLabelMap = { index: '首页', all: '全站', dir: '目录', suffix: '后缀', path: '路径' }
const originConditionItems = [
  { label: '请求URI', value: 'uri' },
  { label: '请求URI(不带参数)', value: 'uri_no_args' },
  { label: '节点国家代码', value: 'node_country' },
  { label: '节点运营商', value: 'node_isp' },
  { label: '节点省份', value: 'node_province' },
  { label: '节点城市', value: 'node_city' },
  { label: '客户端国家代码', value: 'client_country' },
  { label: '客户端运营商', value: 'client_isp' },
  { label: '客户端省份', value: 'client_province' },
  { label: '客户端城市', value: 'client_city' },
  { label: '用户 IP', value: 'client_ip' },
  { label: '域名', value: 'domain' },
  { label: '请求头', value: 'header' },
  { label: '请求方法', value: 'method' },
  { label: 'HTTP 版本', value: 'http_version' },
  { label: '独立 UA 数量', value: 'ua_count' },
  { label: '404 状态码数量', value: 'status_404' }
]
const originConditionOperators = [
  { label: '等于', value: 'eq' },
  { label: '不等于', value: 'neq' },
  { label: '包含', value: 'contains' },
  { label: '不包含', value: 'not_contains' },
  { label: '前缀匹配', value: 'prefix' },
  { label: '后缀匹配', value: 'suffix' },
  { label: '正则匹配', value: 'regex' },
  { label: '正则不匹配', value: 'not_regex' },
  { label: '存在', value: 'exists' },
  { label: '不存在', value: 'not_exists' },
  { label: '在 IP 段', value: 'in_ip' },
  { label: '不在 IP 段', value: 'not_in_ip' }
]

function createDefaultSettings() {
  return {
    basic: {
      planName: '-',
      groupName: '-',
      nodeGroupName: '-',
      domain: '',
      cname: '-',
      status: true,
      createdAt: '-',
      updatedAt: '-'
    },
    origin: {
      list: [],
      conditions: [],
      healthCheckEnabled: true,
      healthCheckHost: '',
      healthCheckPath: '/',
      healthCheckStatus: '200 301 302',
      healthCheckInterval: 60
    },
    https: {
      enabled: true,
      port: '443',
      certId: null,
      force: false,
      redirectPort: '443',
      hsts: false,
      http2: false,
      http3: false,
      ocspStapling: false,
      sslProfile: 'compat',
      sslProtocols: [...defaultSslProtocols],
      sslCiphers: defaultSslCiphers,
      sslPreferServerCiphers: true
    },
    cache: {
      rules: []
    },
    security: {
      defaultRule: 10002,
      autoSwitch: false,
      bot: 'none',
      blacklist: '',
      whitelist: '',
      blackTimeMode: 'system',
      blackTimeCustom: '',
      whiteTimeMode: 'system',
      whiteTimeCustom: '',
      shieldProxy: false,
      regionMode: 'none',
      regionCustom: []
    },
    access: {
      acl: '',
      hotlink: false,
      cors: false
    },
    advanced: {
      gzip: true,
      websocket: false,
      ipv6: false,
      logRequestHeader: false,
      logResponseHeader: false,
      logRequestBody: false,
      bodyLimit: '16',
      realtimeReturn: false,
      realtimeSend: false,
      acmeBacksource: false
    }
  }
}

function cacheTypeLabel(type) {
  return cacheTypeLabelMap[type] || type
}

function parseBool(value, fallback = false) {
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') {
    const v = value.trim().toLowerCase()
    return v === '1' || v === 'true' || v === 'on'
  }
  if (typeof value === 'number') return value !== 0
  return fallback
}

function splitLines(value) {
  if (!value) return []
  return String(value)
    .split(/\\r?\\n/)
    .map(item => item.trim())
    .filter(Boolean)
}

function loadSite() {
  if (!siteId.value) {
    ElMessage.warning('缺少 site_id')
    router.push({ path: '/website/list' })
    return
  }
  loading.value = true
  request
    .get(`/sites/${siteId.value}`)
    .then(res => {
      const data = res.data?.site || res.site || res.data || res
      if (!data || !data.id) {
        ElMessage.error('站点信息载入失败')
        return
      }
      site.value = data
      applySiteData(data)
    })
    .finally(() => {
      loading.value = false
    })
}

function loadCerts() {
  request.get('/certs').then(res => {
    certList.value = res.data?.list || res.list || []
  })
}

function normalizeOriginCondition(item) {
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

function applySiteData(data) {
  const defaults = createDefaultSettings()
  Object.assign(siteSettings, defaults)
  const settings = data.settings || {}
  siteSettings.basic.domain = (data.domains || []).join('\\n')
  siteSettings.basic.cname = computeCname(data)
  siteSettings.basic.status = parseBool(data.enable, true)
  siteSettings.basic.planName = data.user_package_id ? `套餐ID ${data.user_package_id}` : '-'
  siteSettings.basic.groupName = data.group_name || `分组ID ${data.group_id || 0}`
  siteSettings.basic.nodeGroupName = data.node_group_id ? `线路组${data.node_group_id}` : '-'
  siteSettings.basic.createdAt = formatDate(data.create_at)
  siteSettings.basic.updatedAt = formatDate(data.update_at)

  siteSettings.origin.list = (settings.origin?.list || []).map(item => ({
    address: item.address || item.backend || '',
    weight: item.weight || '10',
    enable: item.enable !== false
  }))
  siteSettings.origin.conditions = (settings.origin?.conditions || []).map(normalizeOriginCondition).filter(Boolean)
  siteSettings.origin.healthCheckEnabled = parseBool(settings.origin?.health_check, true)
  siteSettings.origin.healthCheckHost = settings.origin?.health_host || data.domains?.[0] || ''
  siteSettings.origin.healthCheckPath = settings.origin?.health_path || '/'
  siteSettings.origin.healthCheckStatus = settings.origin?.health_status || '200 301 302'
  siteSettings.origin.healthCheckInterval = settings.origin?.health_interval || 60

  const httpsCfg = settings.https || {}
  siteSettings.https.enabled = Boolean(data.https_listen?.length || parseBool(httpsCfg.http3))
  siteSettings.https.port = (data.https_listen && data.https_listen[0]) || httpsCfg.listen_port || '443'
  siteSettings.https.force = parseBool(httpsCfg.force, false)
  siteSettings.https.redirectPort = httpsCfg.redirect_port || '443'
  siteSettings.https.hsts = parseBool(httpsCfg.hsts, false)
  siteSettings.https.http2 = parseBool(httpsCfg.http2, false)
  siteSettings.https.http3 = parseBool(httpsCfg.http3, false)
  siteSettings.https.ocspStapling = parseBool(httpsCfg.ocsp_stapling, false)
  siteSettings.https.sslProfile = httpsCfg.ssl_profile || 'compat'
  siteSettings.https.sslProtocols = httpsCfg.ssl_protocols
    ? httpsCfg.ssl_protocols.split(/\\s+/).filter(Boolean)
    : [...defaultSslProtocols]
  siteSettings.https.sslCiphers = httpsCfg.ssl_ciphers || defaultSslCiphers
  siteSettings.https.sslPreferServerCiphers = parseBool(httpsCfg.ssl_prefer_server_ciphers, true)
  siteSettings.https.certId = settings.certificate_id || null

  siteSettings.cache.rules = (settings.cache?.rules || []).map(normalizeCacheRule).filter(Boolean)

  const sec = settings.security || {}
  siteSettings.security.defaultRule = sec.default_rule || 10002
  siteSettings.security.autoSwitch = parseBool(sec.auto_switch, false)
  siteSettings.security.bot = sec.bot || 'none'
  siteSettings.security.blacklist = (sec.blacklist || []).join('\\n')
  siteSettings.security.whitelist = (sec.whitelist || []).join('\\n')
  siteSettings.security.blackTimeMode = sec.black_time_mode || 'system'
  siteSettings.security.blackTimeCustom = sec.black_time_custom || ''
  siteSettings.security.whiteTimeMode = sec.white_time_mode || 'system'
  siteSettings.security.whiteTimeCustom = sec.white_time_custom || ''
  siteSettings.security.shieldProxy = parseBool(sec.shield_proxy, false)
  siteSettings.security.regionMode = sec.region_mode || 'none'
  siteSettings.security.regionCustom = sec.region_custom || []

  const access = settings.access || {}
  siteSettings.access.acl = access.acl || ''
  siteSettings.access.hotlink = parseBool(access.hotlink, false)
  siteSettings.access.cors = parseBool(access.cors, false)

  const adv = settings.advanced || {}
  siteSettings.advanced.gzip = parseBool(adv.gzip, siteSettings.advanced.gzip)
  siteSettings.advanced.websocket = parseBool(adv.websocket, siteSettings.advanced.websocket)
  siteSettings.advanced.ipv6 = parseBool(adv.ipv6, siteSettings.advanced.ipv6)
  siteSettings.advanced.logRequestHeader = parseBool(adv.log_request_header, false)
  siteSettings.advanced.logResponseHeader = parseBool(adv.log_response_header, false)
  siteSettings.advanced.logRequestBody = parseBool(adv.log_request_body, false)
  siteSettings.advanced.bodyLimit = adv.body_limit || siteSettings.advanced.bodyLimit
  siteSettings.advanced.realtimeReturn = parseBool(adv.realtime_return, false)
  siteSettings.advanced.realtimeSend = parseBool(adv.realtime_send, false)
  siteSettings.advanced.acmeBacksource = parseBool(adv.acme_backsource, false)
}

function computeCname(data) {
  if (data.cname_hostname) return data.cname_hostname
  if (data.domains?.length && data.cname_domain) {
    return `${data.domains[0]}.${data.cname_domain}`
  }
  return '-'
}

function formatDate(value) {
  if (!value) return '-'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return '-'
  return parsed.toLocaleString()
}

function toggleSiteStatus(enabled) {
  if (!siteId.value) return
  const action = enabled ? 'enable' : 'disable'
  request
    .post('/sites/batch_action', { action, ids: [siteId.value] })
    .then(() => {
      ElMessage.success('状态已更新')
      loadSite()
    })
}

function addOrigin() {
  siteSettings.origin.list.push({ address: '', weight: '10', enable: true })
}

function removeOrigin(index) {
  siteSettings.origin.list.splice(index, 1)
}

function addConditionOrigin() {
  siteSettings.origin.conditions.push({
    item: 'uri',
    operator: 'eq',
    value: '',
    origin: '',
    header: '',
    seconds: ''
  })
}

function removeConditionOrigin(index) {
  siteSettings.origin.conditions.splice(index, 1)
}

function handleRegionModeChange() {
  if (siteSettings.security.regionMode !== 'custom') {
    siteSettings.security.regionCustom = []
  }
}

function isOriginHeaderItem(item) {
  return item === 'header'
}

function isOriginStatItem(item) {
  return item === 'ua_count' || item === 'status_404'
}

function getOriginConditionPlaceholder(row) {
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

function handleOriginConditionChange(row) {
  if (!row) return
  if (isOriginStatItem(row.item)) {
    row.operator = 'gt'
    row.seconds = row.seconds || '10'
  } else if (!row.operator) {
    row.operator = 'eq'
  }
}

function openCacheRuleDialog(mode, rule, index) {
  cacheRuleDialog.mode = mode
  cacheRuleDialog.index = index ?? -1
  if (rule) {
    Object.assign(cacheRuleForm, { ...rule })
  } else {
    Object.assign(cacheRuleForm, { type: 'index', value: '', ttl: '86400', ignore_query: false, force_cache: false })
  }
  cacheRuleDialog.visible = true
}

function saveCacheRule() {
  const rule = normalizeCacheRule({
    type: cacheRuleForm.type,
    value: cacheRuleForm.value,
    ttl: cacheRuleForm.ttl,
    ignore_query: cacheRuleForm.ignore_query,
    force_cache: cacheRuleForm.force_cache
  })
  if (!rule) return
  if (cacheRuleDialog.mode === 'edit' && cacheRuleDialog.index >= 0) {
    siteSettings.cache.rules.splice(cacheRuleDialog.index, 1, rule)
  } else {
    siteSettings.cache.rules.push(rule)
  }
  cacheRuleDialog.visible = false
}

function removeCacheRule(index) {
  siteSettings.cache.rules.splice(index, 1)
}

function applyCachePreset(val) {
  if (!val) return
  let preset = null
  switch (val) {
    case 'index':
      preset = { type: 'index', value: '', ttl: '86400' }
      break
    case 'all':
      preset = { type: 'all', value: '', ttl: '259200' }
      break
    case 'static':
      preset = { type: 'suffix', value: 'jpg|jpeg|png|gif|ico|css|js|svg|bmp|webp|woff|woff2', ttl: '604800', ignore_query: true }
      break
    case 'video':
      preset = { type: 'suffix', value: 'mp4|avi|mov|webm|m3u8|ts', ttl: '2592000' }
      break
    case 'wordpress':
      preset = { type: 'all', value: '', ttl: '259200' }
      break
  }
  if (preset) {
    siteSettings.cache.rules.push(normalizeCacheRule(preset))
    ElMessage.success('已添加缓存规则')
  }
  cacheQuickPreset.value = ''
}

function normalizeCacheRule(rule) {
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

function parsePortList(raw) {
  if (!raw) return []
  return raw
    .split(/[\s,]+/)
    .map(item => item.trim())
    .filter(Boolean)
}

function buildSettingsPayload() {
  return {
    origin: {
      list: siteSettings.origin.list,
      conditions: siteSettings.origin.conditions,
      health_check: siteSettings.origin.healthCheckEnabled,
      health_host: siteSettings.origin.healthCheckHost,
      health_path: siteSettings.origin.healthCheckPath,
      health_status: siteSettings.origin.healthCheckStatus,
      health_interval: siteSettings.origin.healthCheckInterval
    },
    https: {
      enabled: siteSettings.https.enabled,
      listen_port: siteSettings.https.port,
      force: siteSettings.https.force,
      redirect_port: siteSettings.https.redirectPort,
      hsts: siteSettings.https.hsts,
      http2: siteSettings.https.http2,
      http3: siteSettings.https.http3,
      ocsp_stapling: siteSettings.https.ocspStapling,
      ssl_profile: siteSettings.https.sslProfile,
      ssl_protocols: siteSettings.https.sslProtocols.join(' '),
      ssl_ciphers: siteSettings.https.sslCiphers,
      ssl_prefer_server_ciphers: siteSettings.https.sslPreferServerCiphers ? 'on' : 'off'
    },
    cache: { rules: siteSettings.cache.rules },
    security: {
      default_rule: siteSettings.security.defaultRule,
      auto_switch: siteSettings.security.autoSwitch,
      bot: siteSettings.security.bot,
      blacklist: splitLines(siteSettings.security.blacklist),
      whitelist: splitLines(siteSettings.security.whitelist),
      black_time_mode: siteSettings.security.blackTimeMode,
      black_time_custom: siteSettings.security.blackTimeCustom,
      white_time_mode: siteSettings.security.whiteTimeMode,
      white_time_custom: siteSettings.security.whiteTimeCustom,
      shield_proxy: siteSettings.security.shieldProxy,
      region_mode: siteSettings.security.regionMode,
      region_custom: siteSettings.security.regionCustom
    },
    access: {
      acl: siteSettings.access.acl,
      hotlink: siteSettings.access.hotlink,
      cors: siteSettings.access.cors
    },
    advanced: {
      gzip: siteSettings.advanced.gzip,
      websocket: siteSettings.advanced.websocket,
      ipv6: siteSettings.advanced.ipv6,
      log_request_header: siteSettings.advanced.logRequestHeader,
      log_response_header: siteSettings.advanced.logResponseHeader,
      log_request_body: siteSettings.advanced.logRequestBody,
      body_limit: siteSettings.advanced.bodyLimit,
      realtime_return: siteSettings.advanced.realtimeReturn,
      realtime_send: siteSettings.advanced.realtimeSend,
      acme_backsource: siteSettings.advanced.acmeBacksource
    },
    certificate_id: siteSettings.https.certId
  }
}

function saveSettings() {
  if (!siteId.value) return
  saving.value = true
  const payload = {
    ids: [siteId.value],
    settings: buildSettingsPayload(),
    https_listen: siteSettings.https.enabled ? parsePortList(siteSettings.https.port) : []
  }
  request
    .post('/sites/batch_update', payload)
    .then(() => {
      ElMessage.success('配置已保存')
      loadSite()
    })
    .catch(() => {
      ElMessage.error('保存失败，请稍后再试')
    })
    .finally(() => {
      saving.value = false
    })
}

function goBack() {
  router.push({ path: '/website/list' })
}

onMounted(() => {
  loadSite()
  loadCerts()
})
</script>
<style scoped>
.site-manage {
  padding: 16px;
}
.page-card {
  background: #fff;
}
.manage-tabs {
  margin-bottom: 20px;
}
.site-manage-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}
.toolbar-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.mb-16 {
  margin-bottom: 16px;
}
.config-form {
  margin-top: 12px;
}
.section-title {
  font-weight: 600;
  margin: 12px 0 8px;
}
.condition-origin-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.country-selector {
  margin-top: 12px;
}
</style>
