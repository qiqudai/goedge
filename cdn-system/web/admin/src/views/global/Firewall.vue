<template>
  <div class="app-container firewall-page" @focusin="cacheInputValue">
    <el-card class="firewall-card">
      <template #header>
        <div class="card-header">
          <span>防火墙全局配置 (WAF)</span>
        </div>
      </template>

      <el-tabs v-if="config.waf" v-model="activeTab" v-loading="loading" type="border-card" class="firewall-tabs" @tab-change="handleTabChange">
        <el-tab-pane label="基础防护 & 拉黑策略" name="basic">
          <el-form label-width="180px" class="firewall-form">
            <el-form-item label="全局 WAF 开启">
              <el-switch v-model="config.waf.enable" active-text="开启" inactive-text="关闭" @change="saveConfig" />
            </el-form-item>

            <SectionTitle>默认拉黑方式</SectionTitle>
            <el-form-item label="拉黑动作">
              <el-radio-group v-model="config.waf.default_block_action" @change="saveConfig">
                <el-radio value="ipset">IPSet (系统防火墙)</el-radio>
                <el-radio value="disconnect">断开连接</el-radio>
                <el-radio value="page">显示拦截页面</el-radio>
              </el-radio-group>
              <div class="form-tip">建议默认选非 IPSet 方式，配合自动 IPSet 切换使用。</div>
            </el-form-item>

            <SectionTitle>IPSet 自动切换</SectionTitle>
            <el-form-item label="自动启用 IPSet">
              <el-switch v-model="config.waf.auto_ipset_enable" @change="saveConfig" />
            </el-form-item>
            <el-form-item label="触发阈值" v-if="config.waf.auto_ipset_enable">
              <el-input-number v-model="config.waf.auto_ipset_threshold" :min="1" @blur="saveConfig" />
              <span class="unit">次/秒</span>
              <div class="form-tip">当单站每秒拉黑次数超过阈值时，自动升级为 IPSet 拉黑。</div>
            </el-form-item>

            <SectionTitle>拉黑页面限制 (防刷)</SectionTitle>
            <el-form-item label="限制访问频率">
              <el-switch v-model="config.waf.block_page_rate_limit_enable" @change="saveConfig" />
            </el-form-item>
            <el-form-item label="频率阈值" v-if="config.waf.block_page_rate_limit_enable">
              <el-input-number v-model="config.waf.block_page_rate_limit" :min="1" @blur="saveConfig" />
              <span class="unit">次/60秒</span>
              <div class="form-tip">单 IP 访问拉黑页面超过此频率，直接升级 IPSet 拉黑。</div>
            </el-form-item>
            <el-form-item label="拉黑页不计流量">
              <el-switch v-model="config.waf.block_page_traffic_free" @change="saveConfig" />
            </el-form-item>

            <SectionTitle>封禁与白名单时长</SectionTitle>
            <el-form-item label="黑名单封禁时长">
              <el-input-number v-model="config.waf.blacklist_timeout" @blur="saveConfig" />
              <span class="unit">秒</span>
            </el-form-item>
            <el-form-item label="临时白名单时长">
              <el-input-number v-model="config.waf.temp_whitelist_timeout" @blur="saveConfig" />
              <span class="unit">秒</span>
            </el-form-item>

            <SectionTitle>临时白名单自动加入条件(5秒内)</SectionTitle>
            <el-form-item label="总请求数限制">
              <el-input-number v-model="config.waf.temp_whitelist_limit_total" @blur="saveConfig" />
            </el-form-item>
            <el-form-item label="同URL请求限制">
              <el-input-number v-model="config.waf.temp_whitelist_limit_url" @blur="saveConfig" />
            </el-form-item>

            <SectionTitle>自动清理节点日志</SectionTitle>
            <el-form-item label="清理策略">
              <el-radio-group v-model="config.waf.node_log_clean_strategy" @change="saveConfig">
                <el-radio value="none">不清理</el-radio>
                <el-radio value="log_only">节点空间不足时，清空访问日志</el-radio>
                <el-radio value="log_cache">即清空访问日志，也清空缓存</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="安全控制 & 名单" name="access">
          <el-form label-width="180px" class="firewall-form">
            <SectionTitle>黑白名单 (一行一个，支持 CIDR)</SectionTitle>
            <div class="risk-grid">
              <div class="risk-item">
                <span class="risk-label">泛 IP 上限</span>
                <strong>{{ wafPatternLimit }}</strong>
              </div>
              <div class="risk-item">
                <span class="risk-label">黑名单精准 IP</span>
                <strong>{{ blacklistRisk.exact }}</strong>
              </div>
              <div class="risk-item" :class="{ danger: blacklistRisk.pattern > wafPatternLimit }">
                <span class="risk-label">黑名单泛 IP</span>
                <strong>{{ blacklistRisk.pattern }}</strong>
              </div>
              <div class="risk-item" :class="{ danger: whitelistRisk.pattern > wafPatternLimit }">
                <span class="risk-label">白名单泛 IP</span>
                <strong>{{ whitelistRisk.pattern }}</strong>
              </div>
              <div class="risk-item" :class="{ danger: totalInvalidEntries > 0 }">
                <span class="risk-label">无效条目</span>
                <strong>{{ totalInvalidEntries }}</strong>
              </div>
              <div class="risk-item" :class="{ warn: totalOverbroadEntries > 0 }">
                <span class="risk-label">过宽规则</span>
                <strong>{{ totalOverbroadEntries }}</strong>
              </div>
            </div>
            <el-alert
              v-if="wafListRiskMessage"
              class="risk-alert"
              :title="wafListRiskMessage"
              type="warning"
              show-icon
              :closable="false"
            />
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="白名单 IP">
                  <el-input type="textarea" v-model="config.waf.whitelist_ips" :rows="10" placeholder="192.168.1.10&#10;10.0.0.0/24" @blur="saveConfig" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="黑名单 IP">
                  <el-input type="textarea" v-model="config.waf.blacklist_ips" :rows="10" placeholder="1.2.3.4&#10;5.0.0.0/8" @blur="saveConfig" />
                </el-form-item>
              </el-col>
            </el-row>

            <SectionTitle>系统安全</SectionTitle>
            <el-form-item label="防止 TLS 握手攻击">
              <el-switch v-model="config.waf.prevent_tls_handshake" @change="saveConfig" />
            </el-form-item>
            <el-form-item label="禁止未绑定域名访问">
              <el-switch v-model="config.waf.block_unbound_domain" @change="saveConfig" />
              <span class="inline-tip">禁止直接通过节点 IP 访问。</span>
            </el-form-item>
            <el-form-item label="禁止 PING">
              <el-switch v-model="config.waf.disable_ping" @change="saveConfig" />
            </el-form-item>

            <SectionTitle>通讯密钥</SectionTitle>
            <el-form-item label="通讯密钥">
              <el-input v-model="config.waf.secret_key" show-password @blur="saveConfig" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="CC 防护 & 验证" name="cc">
          <el-form label-width="180px" class="firewall-form">
            <SectionTitle>默认页防护</SectionTitle>
            <el-form-item label="开启模式">
              <el-radio-group v-model="config.waf.default_page_protection" @change="saveConfig">
                <el-radio value="force">强制开启</el-radio>
                <el-radio value="auto">自动开启</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="自动开启阈值" v-if="config.waf.default_page_protection === 'auto'">
              <el-input-number v-model="config.waf.default_page_protection_threshold" @blur="saveConfig" />
              <span class="unit">请求/秒</span>
            </el-form-item>

            <SectionTitle>CC 规则自动切换</SectionTitle>
            <el-form-item label="CC 规则自动切换">
              <el-switch
                v-model="config.waf.cc_auto_switch.enable"
                active-text="开启"
                inactive-text="关闭"
                @change="syncCcRuleAutoSwitch(); saveConfig()"
              />
            </el-form-item>
            <template v-if="config.waf.cc_auto_switch.enable">
              <el-form-item label="502/504 QPS">
                <el-input-number v-model="config.waf.cc_auto_switch.qps_502504" :min="0" @blur="saveConfig" />
                <span class="inline-tip">切换条件</span>
              </el-form-item>
              <el-form-item label="总 QPS">
                <el-input-number v-model="config.waf.cc_auto_switch.total_qps" :min="0" @blur="saveConfig" />
                <span class="inline-tip">切换条件</span>
              </el-form-item>
              <el-form-item label="切换规则组">
                <el-select v-model="config.waf.cc_auto_switch.rule" style="width: 280px" @change="saveConfig">
                  <el-option v-for="item in ccSwitchRuleOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="切换时长">
                <el-input-number v-model="config.waf.cc_auto_switch.duration" :min="1" @blur="saveConfig" />
                <span class="unit">秒</span>
              </el-form-item>
            </template>

            <SectionTitle>防 CC 图片更新 URL</SectionTitle>
            <el-form-item label="图片来源">
              <el-radio-group v-model="config.waf.anti_cc_image_source" @change="saveConfig">
                <el-radio value="system">系统默认</el-radio>
                <el-radio value="custom">自定义</el-radio>
              </el-radio-group>
              <el-button type="primary" plain style="margin-left: 12px" @click="handleUpdateGuardImages">立即更新</el-button>
            </el-form-item>
            <template v-if="config.waf.anti_cc_image_source === 'custom'">
              <el-form-item label="图片 URL">
                <el-input v-model="config.waf.anti_cc_image_custom_url" placeholder="http://..." @blur="saveConfig">
                  <template #prepend>URL</template>
                </el-input>
              </el-form-item>
              <el-form-item label="定时更新">
                <el-input-number v-model="config.waf.anti_cc_image_update_hour" :min="0" :max="23" @blur="saveConfig" />
                <span class="unit">点更新（0-23，留空或 0 表示不自动更新）</span>
              </el-form-item>
              <el-form-item>
                <div class="form-tip form-tip-warn">
                  强烈建议自己搭建生成图片的服务器，并选择自定义填入自己的图片下载地址。
                </div>
              </el-form-item>
            </template>

            <SectionTitle>防 CC 页面设置</SectionTitle>
            <el-form-item label="防 CC 页面">
              <el-radio-group v-model="config.waf.anti_cc_type" class="anti-cc-type-group" @change="handleAntiCcTypeChange">
                <el-radio v-for="item in antiCcTypeOptions" :key="item.value" :value="item.value">{{ item.label }}</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="页面模板">
              <el-tabs v-model="guardEditorTab" class="guard-editor-tabs">
                <el-tab-pane label="HTML 模板" name="template">
                  <ErrorPageTemplateEditor
                    :model-value="currentGuardPage.template"
                    @update:model-value="updateGuardTemplate"
                    @blur="saveConfig"
                  />
                </el-tab-pane>
                <el-tab-pane label="多语言文案" name="strings">
                  <ErrorPageTranslationTable
                    :template="currentGuardPage.template"
                    :strings="currentGuardPage.strings"
                    :enabled-langs="enabledGuardLangs"
                    @update:strings="updateGuardStrings"
                  />
                </el-tab-pane>
                <el-tab-pane label="预览" name="preview">
                  <div class="preview-toolbar">
                    <span>预览语言</span>
                    <el-select v-model="guardPreviewLang" style="width: 200px">
                      <el-option v-for="lang in enabledGuardLangs" :key="lang" :label="lang" :value="lang" />
                    </el-select>
                  </div>
                  <div class="guard-preview" v-html="guardPreviewHtml"></div>
                </el-tab-pane>
              </el-tabs>
              <div class="form-tip">模板使用 &#123;&#123;变量名&#125;&#125; 占位符；运行时语言跟随全局错误页语言设置。</div>
            </el-form-item>

            <SectionTitle>调试与路径防护</SectionTitle>
            <el-form-item label="开启调试日志">
              <el-switch v-model="config.waf.anti_cc_debug" @change="saveConfig" />
            </el-form-item>
            <el-form-item label=".well-known 防护">
              <el-input-number v-model="config.waf.well_known_protection_threshold" @blur="saveConfig" />
              <span class="unit">次/60秒</span>
              <div class="form-tip">
                当 /.well-known/acme-challenge/ 路径的 404 请求数在 60 秒内超过以上次数时，300 秒内除已验证 IP 外不再回源到主控，仍可正常申请证书。
              </div>
            </el-form-item>

            <SectionTitle>内置资源防护</SectionTitle>
            <el-form-item label="开启防护">
              <el-switch v-model="config.waf.resource_protection_enable" @change="saveConfig" />
              <span class="inline-tip">内置资源是指防 CC 页面里的图片、JS 等资源。</span>
            </el-form-item>
            <el-form-item label="开启阈值" v-if="config.waf.resource_protection_enable">
              <el-input-number v-model="config.waf.resource_protection_threshold" @blur="saveConfig" />
              <span class="unit">QPS</span>
            </el-form-item>
            <el-form-item label="拉黑时长" v-if="config.waf.resource_protection_enable">
              <el-input-number v-model="config.waf.resource_protection_block_timeout" @blur="saveConfig" />
              <span class="unit">秒</span>
            </el-form-item>
            <el-form-item label="限流规则" v-if="config.waf.resource_protection_enable">
              <el-table :data="config.waf.resource_protection_rules" border class="resource-rule-table">
                <el-table-column label="统计时长(秒)" width="180">
                  <template #default="{ row }">
                    <el-input-number v-model="row.duration" size="small" :controls="false" placeholder="为空则不启用" @blur="saveConfig" />
                  </template>
                </el-table-column>
                <el-table-column label="最大次数" width="180">
                  <template #default="{ row }">
                    <el-input-number v-model="row.max_requests" size="small" :controls="false" placeholder="为空则不启用" @blur="saveConfig" />
                  </template>
                </el-table-column>
                <el-table-column label="操作">
                  <template #default="{ $index }">
                    <el-button size="small" type="danger" link @click="removeResourceRule($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <el-button size="small" class="table-action-btn" @click="addResourceRule">+ 添加规则</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, h, defineComponent, computed } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import ErrorPageTemplateEditor from '@/components/global/ErrorPageTemplateEditor.vue'
import ErrorPageTranslationTable from '@/components/global/ErrorPageTranslationTable.vue'
import { DEFAULT_ERROR_PAGE_I18N } from '@/constants/errorPageLocales'
import {
  ensureGuardPageStructure,
  migrateLegacyAntiCCPageCustom,
  renderGuardPagePreview,
  resolveGuardPageKey,
  resolveGuardPreviewStrings
} from '@/services/guardPageService'

const SectionTitle = defineComponent({
  name: 'SectionTitle',
  setup(_, { slots }) {
    return () => h('h4', { class: 'section-title' }, slots.default?.())
  }
})

const ccSwitchRuleOptions = [
  { value: 'close', label: '关闭' },
  { value: 'lenient', label: '宽松' },
  { value: 'js', label: 'JS 验证' },
  { value: '5s', label: '5 秒盾' },
  { value: 'click', label: '点击验证' },
  { value: 'slide', label: '滑块验证' },
  { value: 'captcha', label: '验证码' },
  { value: 'rotate', label: '旋转图片' },
  { value: 'click_simple', label: '点击验证(简单)' },
  { value: 'slide_simple', label: '滑块验证(简单)' },
  { value: 'temp_whitelist', label: '临时白名单' }
]

const antiCcTypeOptions = [
  { value: 'slide', label: '滑动' },
  { value: 'captcha', label: '验证码' },
  { value: 'click', label: '点击' },
  { value: '5s', label: '5 秒盾' },
  { value: 'rotate', label: '图片旋转' },
  { value: 'click_simple', label: '点击(简单)' },
  { value: 'slide_simple', label: '滑动(简单)' }
]

const defaultCcAutoSwitch = () => ({
  enable: false,
  qps_502504: 0,
  total_qps: 200,
  rule: 'slide',
  duration: 60
})

const defaultWaf = {
  enable: false,
  default_block_action: 'disconnect',
  auto_ipset_enable: false,
  auto_ipset_threshold: 60,
  block_page_rate_limit_enable: false,
  block_page_rate_limit: 60,
  block_page_traffic_free: false,
  blacklist_timeout: 3600,
  temp_whitelist_timeout: 300,
  temp_whitelist_limit_total: 20,
  temp_whitelist_limit_url: 10,
  whitelist_ips: '',
  blacklist_ips: '',
  prevent_tls_handshake: false,
  block_unbound_domain: false,
  disable_ping: false,
  default_page_protection: 'auto',
  default_page_protection_threshold: 200,
  anti_cc_type: 'slide',
  anti_cc_image_source: 'system',
  anti_cc_image_custom_url: '',
  anti_cc_image_update_hour: 0,
  anti_cc_page_custom: '',
  anti_cc_debug: false,
  cc_rule_auto_switch: false,
  cc_auto_switch: defaultCcAutoSwitch(),
  secret_key: '',
  node_log_clean_strategy: 'none',
  well_known_protection_threshold: 60,
  resource_protection_enable: false,
  resource_protection_threshold: 200,
  resource_protection_block_timeout: 300,
  resource_protection_rules: []
}

const loading = ref(false)
const activeTab = ref('basic')
const guardEditorTab = ref('template')
const guardPreviewLang = ref('zh-CN')
const config = ref({ waf: { ...defaultWaf }, guard_pages: {}, error_page_i18n: { ...DEFAULT_ERROR_PAGE_I18N } })

const wafPatternLimit = computed(() => {
  return Number(config.value.resources?.website?.max_waf_pattern_ips || 100)
})

const parseIPListEntries = (text = '') => {
  const seen = new Set()
  return String(text || '')
    .replace(/\r\n/g, '\n')
    .replace(/\r/g, '\n')
    .split('\n')
    .map(item => item.trim())
    .filter(item => {
      if (!item || seen.has(item)) return false
      seen.add(item)
      return true
    })
}

const isIPv4 = (entry) => {
  const parts = entry.split('.')
  return parts.length === 4 && parts.every(part => {
    if (!/^\d+$/.test(part)) return false
    const n = Number(part)
    return n >= 0 && n <= 255
  })
}

const isIPv6 = (entry) => entry.includes(':') && /^[0-9a-fA-F:]+$/.test(entry)

const isCIDR = (entry) => {
  const [ip, prefix] = entry.split('/')
  if (!ip || prefix == null || !/^\d+$/.test(prefix)) return false
  const bits = Number(prefix)
  if (isIPv4(ip)) return bits >= 0 && bits <= 32
  if (isIPv6(ip)) return bits >= 0 && bits <= 128
  return false
}

const isWildcard = (entry) => {
  if (!entry.includes('*')) return false
  const parts = entry.split('.')
  if (!parts.length || parts.length > 4) return false
  return parts.every(part => part === '*' || (/^\d+$/.test(part) && Number(part) >= 0 && Number(part) <= 255))
}

const isPatternEntry = (entry) => entry.includes('/') || entry.includes('*')

const isValidIPEntry = (entry) => isIPv4(entry) || isIPv6(entry) || isCIDR(entry) || isWildcard(entry)

const isOverbroadEntry = (entry) => {
  if (entry.includes('*')) {
    return entry.split('.').filter(part => part !== '*').length === 0
  }
  if (entry.includes('/')) {
    const [ip, prefix] = entry.split('/')
    const bits = Number(prefix)
    if (isIPv4(ip)) return bits <= 8
    if (isIPv6(ip)) return bits <= 32
  }
  return false
}

const analyzeIPList = (text = '') => {
  const stats = { total: 0, exact: 0, pattern: 0, invalid: 0, overbroad: 0 }
  parseIPListEntries(text).forEach(entry => {
    stats.total++
    if (!isValidIPEntry(entry)) {
      stats.invalid++
      return
    }
    if (isPatternEntry(entry)) {
      stats.pattern++
      if (isOverbroadEntry(entry)) {
        stats.overbroad++
      }
      return
    }
    stats.exact++
  })
  return stats
}

const blacklistRisk = computed(() => analyzeIPList(config.value.waf?.blacklist_ips || ''))
const whitelistRisk = computed(() => analyzeIPList(config.value.waf?.whitelist_ips || ''))
const totalInvalidEntries = computed(() => blacklistRisk.value.invalid + whitelistRisk.value.invalid)
const totalOverbroadEntries = computed(() => blacklistRisk.value.overbroad + whitelistRisk.value.overbroad)
const wafListRiskMessage = computed(() => {
  if (totalInvalidEntries.value > 0) {
    return `存在 ${totalInvalidEntries.value} 条无效 IP 规则，保存时会被后端拒绝。`
  }
  if (blacklistRisk.value.pattern > wafPatternLimit.value || whitelistRisk.value.pattern > wafPatternLimit.value) {
    return `泛 IP 规则超过上限 ${wafPatternLimit.value}，请减少 CIDR 或通配符规则。`
  }
  if (totalOverbroadEntries.value > 0) {
    return `存在 ${totalOverbroadEntries.value} 条覆盖范围过大的规则，请确认不会误封大范围用户。`
  }
  return ''
})

const enabledGuardLangs = computed(() => {
  const langs = config.value.error_page_i18n?.enabled_langs
  return Array.isArray(langs) && langs.length ? langs : DEFAULT_ERROR_PAGE_I18N.enabled_langs
})

const currentGuardPageKey = computed(() => resolveGuardPageKey(config.value.waf?.anti_cc_type || 'slide'))

const currentGuardPage = computed(() => {
  const pages = config.value.guard_pages || {}
  return pages[currentGuardPageKey.value] || { template: '', strings: {} }
})

const guardPreviewHtml = computed(() => {
  const page = currentGuardPage.value
  const i18n = config.value.error_page_i18n || DEFAULT_ERROR_PAGE_I18N
  const lang = guardPreviewLang.value || i18n.default_lang || 'zh-CN'
  const strings = resolveGuardPreviewStrings(page, lang, i18n.default_lang || 'zh-CN')
  return renderGuardPagePreview(page.template, strings, lang)
})

const normalizeConfig = (raw = {}) => {
  const merged = { ...raw }
  merged.waf = normalizeWaf(merged.waf)
  merged.resources = merged.resources || {}
  merged.resources.website = {
    max_waf_pattern_ips: 100,
    ...(merged.resources.website || {})
  }
  merged.error_page_i18n = {
    ...DEFAULT_ERROR_PAGE_I18N,
    ...(merged.error_page_i18n || {})
  }
  merged.guard_pages = ensureGuardPageStructure(merged.guard_pages, merged.error_page_i18n)
  if (merged.waf?.anti_cc_page_custom) {
    merged.guard_pages = migrateLegacyAntiCCPageCustom(
      merged.guard_pages,
      merged.waf.anti_cc_type,
      merged.waf.anti_cc_page_custom
    )
    merged.waf.anti_cc_page_custom = ''
  }
  const langs = merged.error_page_i18n?.enabled_langs || DEFAULT_ERROR_PAGE_I18N.enabled_langs
  if (!langs.includes(guardPreviewLang.value)) {
    guardPreviewLang.value = merged.error_page_i18n.default_lang || 'zh-CN'
  }
  return merged
}

const updateGuardTemplate = (value) => {
  const key = currentGuardPageKey.value
  if (!config.value.guard_pages[key]) {
    config.value.guard_pages[key] = { template: '', strings: {} }
  }
  config.value.guard_pages[key].template = value
}

const updateGuardStrings = (value) => {
  const key = currentGuardPageKey.value
  if (!config.value.guard_pages[key]) {
    config.value.guard_pages[key] = { template: '', strings: {} }
  }
  config.value.guard_pages[key].strings = value
  saveConfig()
}

const handleAntiCcTypeChange = () => {
  guardEditorTab.value = 'template'
  saveConfig()
}

const normalizeWaf = (raw = {}) => {
  const merged = { ...defaultWaf, ...raw }
  merged.cc_auto_switch = {
    ...defaultCcAutoSwitch(),
    ...(raw.cc_auto_switch || {})
  }
  if (raw.cc_rule_auto_switch === true && merged.cc_auto_switch.enable !== true) {
    merged.cc_auto_switch.enable = true
  }
  merged.cc_rule_auto_switch = !!merged.cc_auto_switch.enable
  if (!Array.isArray(merged.resource_protection_rules)) {
    merged.resource_protection_rules = []
  }
  while (merged.resource_protection_rules.length < 3) {
    merged.resource_protection_rules.push({ duration: undefined, max_requests: undefined })
  }
  return merged
}

const syncCcRuleAutoSwitch = () => {
  config.value.waf.cc_rule_auto_switch = !!config.value.waf.cc_auto_switch.enable
}

const loadConfig = () => {
  loading.value = true
  request.get('/global_config').then(res => {
    if (res.code === 0 || res.code === 200) {
      config.value = normalizeConfig(res.data || {})
    }
  }).finally(() => {
    loading.value = false
  })
}

const saving = ref(false)
let saveQueued = false

const cacheInputValue = (event) => {
  const el = event?.target
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
    return
  }
  el.dataset.lastValue = el.value ?? ''
}

const shouldSkipBlurSave = (event) => {
  const el = event?.target
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
    return false
  }
  const value = el.value ?? ''
  const lastValue = el.dataset.lastValue ?? ''
  if (value === '' || value === lastValue) {
    return true
  }
  el.dataset.lastValue = value
  return false
}

const saveConfig = async (event) => {
  if (shouldSkipBlurSave(event)) {
    return
  }
  syncCcRuleAutoSwitch()
  if (saving.value) {
    saveQueued = true
    return
  }
  saving.value = true
  await nextTick()
  request.post('/global_config', config.value).then(res => {
    if (res.code === 0 || res.code === 200) {
      ElMessage.success('WAF 配置已保存')
    }
  }).finally(() => {
    saving.value = false
    if (saveQueued) {
      saveQueued = false
      saveConfig()
    }
  })
}

const addResourceRule = () => {
  config.value.waf.resource_protection_rules.push({ duration: 60, max_requests: 100 })
  saveConfig()
}

const removeResourceRule = (index) => {
  config.value.waf.resource_protection_rules.splice(index, 1)
  saveConfig()
}

const handleUpdateGuardImages = () => {
  ElMessage.info('已记录更新请求，节点将在下次同步配置后拉取最新防 CC 图片')
  saveConfig()
}

const handleTabChange = () => {
  loadConfig()
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.firewall-page {
  color: var(--el-text-color-primary);
}

.firewall-card,
.firewall-tabs {
  background: var(--el-bg-color);
  border-color: var(--el-border-color-light);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.firewall-form :deep(.section-title) {
  margin: 20px 0 10px;
  padding-left: 10px;
  border-left: 4px solid var(--el-color-primary);
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 6px;
  line-height: 1.5;
}

.form-tip-warn {
  color: var(--el-color-danger);
}

.inline-tip {
  margin-left: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.risk-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 10px;
  margin: 6px 0 14px;
}

.risk-item {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
}

.risk-item strong {
  display: block;
  margin-top: 4px;
  font-size: 18px;
  color: var(--el-text-color-primary);
}

.risk-item.warn strong {
  color: var(--el-color-warning);
}

.risk-item.danger strong {
  color: var(--el-color-danger);
}

.risk-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.risk-alert {
  margin: 0 0 14px;
}

.unit {
  margin-left: 10px;
  color: var(--el-text-color-secondary);
}

.anti-cc-type-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
}

.guard-editor-tabs {
  width: 100%;
  max-width: 960px;
}

.preview-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  color: var(--el-text-color-regular);
}

.guard-preview {
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--el-border-radius-base);
  background: var(--el-bg-color);
  min-height: 280px;
  overflow: auto;
}

:deep(.guard-preview iframe) {
  width: 100%;
  min-height: 280px;
  border: 0;
}

.resource-rule-table {
  width: 100%;
  max-width: 720px;
}

.table-action-btn {
  margin-top: 8px;
}

:deep(.el-tabs--border-card) {
  background: var(--el-bg-color);
  border-color: var(--el-border-color-light);
}

:deep(.el-tabs--border-card > .el-tabs__header) {
  background: var(--el-fill-color-light);
  border-bottom-color: var(--el-border-color-light);
}

:deep(.el-tabs--border-card > .el-tabs__header .el-tabs__item.is-active) {
  background: var(--el-bg-color);
  color: var(--el-color-primary);
}

:deep(.el-radio-group .el-radio) {
  margin-right: 16px;
  color: var(--el-text-color-primary);
}
</style>
