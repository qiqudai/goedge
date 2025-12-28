<template>
  <div class="default-config">
    <el-card shadow="never" class="layout-card">
      <template #header>
        <div class="card-header">全局默认配置</div>
      </template>
      <el-tabs v-model="topTab" class="top-tabs">
        <el-tab-pane label="全局配置" name="global">
          <el-tabs v-model="globalTab" class="sub-tabs">
            <el-tab-pane label="网站" name="site">
              <el-form label-width="150px" class="config-form">
                <div class="section-title">HTTP</div>
                <el-form-item label="监听端口" style="max-width: 500px;">
                  <el-input v-model="siteForm.httpListen" @change="saveSiteConfig" />
                </el-form-item>
                <el-divider />
                <div class="section-title">HTTPS</div>
                <el-form-item label="监听端口" style="max-width: 500px;">
                  <el-input v-model="siteForm.httpsListen" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="开启HSTS" style="max-width: 500px;">
                  <el-switch v-model="siteForm.httpsHsts" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="开启HTTP2" style="max-width: 500px;">
                  <el-switch v-model="siteForm.httpsHttp2" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="开启HTTP3" style="max-width: 500px;">
                  <el-switch v-model="siteForm.httpsHttp3" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="强制HTTPS" style="max-width: 500px;">
                  <el-switch v-model="siteForm.httpsForce" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="ssl_protocols" style="max-width: 600px;">
                  <el-checkbox-group v-model="siteForm.sslProtocols" @change="saveSiteConfig">
                    <el-checkbox label="SSLv2" />
                    <el-checkbox label="SSLv3" />
                    <el-checkbox label="TLSv1" />
                    <el-checkbox label="TLSv1.1" />
                    <el-checkbox label="TLSv1.2" />
                    <el-checkbox label="TLSv1.3" />
                  </el-checkbox-group>
                </el-form-item>
                <el-form-item label="ssl_ciphers" style="max-width: 600px;">
                  <el-input
                    v-model="siteForm.sslCiphers"
                    type="textarea"
                    :rows="2"
                    @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="ssl_prefer_server_ciphers" style="max-width: 500px;">
                  <el-switch v-model="siteForm.sslPreferServerCiphers" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="ocsp_stapling" style="max-width: 500px;">
                  <el-switch v-model="siteForm.ocspStapling" @change="saveSiteConfig" />
                </el-form-item>
                <el-divider />
                <div class="section-title">回源设置</div>
                <el-form-item label="回源协议" style="max-width: 600px;">
                  <el-radio-group v-model="siteForm.backendProtocol" @change="saveSiteConfig">
                    <el-radio label="http">HTTP</el-radio>
                    <el-radio label="https">HTTPS</el-radio>
                    <el-radio label="follow">跟随协议</el-radio>
                    <el-radio label="follow_port">跟随端口和协议</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item label="回源http端口" style="max-width: 500px;">
                  <el-input v-model="siteForm.backendHttpPort" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="回源https端口" style="max-width: 500px;">
                  <el-input v-model="siteForm.backendHttpsPort" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="回源超时" style="max-width: 500px;">
                  <el-input v-model="siteForm.proxyTimeout" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="连接超时" style="max-width: 500px;">
                  <el-input v-model="siteForm.connectTimeout" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="回源SSL协议" style="max-width: 600px;">
                  <el-checkbox-group v-model="siteForm.proxySslProtocols" @change="saveSiteConfig">
                    <el-checkbox label="SSLv2" />
                    <el-checkbox label="SSLv3" />
                    <el-checkbox label="TLSv1" />
                    <el-checkbox label="TLSv1.1" />
                    <el-checkbox label="TLSv1.2" />
                    <el-checkbox label="TLSv1.3" />
                  </el-checkbox-group>
                </el-form-item>
                <el-divider />
                <div class="section-title">缓存</div>
                <div class="toolbar-row">
                  <el-button type="primary" size="normal" @click="openCacheRuleDialog('create')">新增规则</el-button>
                  <el-select
                    v-model="cacheQuickPreset"
                    placeholder="快速设置缓存"
                    style="max-width: 200px"
                    size="normal"
                    @change="applyCachePreset">
                    <el-option label="通用静态资源" value="static" />
                    <el-option label="视频资源" value="video" />
                    <el-option label="图片资源" value="image" />
                  </el-select>
                </div>
                <el-table :data="siteForm.cacheRules" border class="config-table">
                  <el-table-column type="selection" width="50" />
                  <el-table-column label="类型" min-width="120">
                    <template #default="{ row }">
                      <span>{{ cacheTypeLabel(row.type) }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="内容" min-width="220">
                    <template #default="{ row }">
                      <span>{{ row.value }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="有效期" width="120">
                    <template #default="{ row }">
                      <span>{{ row.ttl }} 天</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="忽略参数" width="120">
                    <template #default="{ row }">
                      <el-tag :type="row.ignore_query ? 'success' : 'warning'">
                        {{ row.ignore_query ? '是' : '否' }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="强制缓存" width="120">
                    <template #default="{ row }">
                      <el-tag :type="row.force_cache ? 'success' : 'warning'">
                        {{ row.force_cache ? '是' : '否' }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="140">
                    <template #default="{ row, $index }">
                      <el-button type="primary" link size="normal" @click="openCacheRuleDialog('edit', row, $index)">编辑</el-button>
                      <el-button type="danger" link size="normal" @click="removeCacheRule($index)">删除</el-button>
                    </template>
                  </el-table-column>
                </el-table>
                <div class="help-text">缓存规则由上到下匹配，匹配到即停止，可拖动调整顺序</div>
                <el-divider />
                <div class="section-title">源站请求头</div>
                <div class="toolbar-row">
                  <el-button type="primary" size="normal" @click="openHeaderDialog('create')">新增请求头</el-button>
                </div>
                <el-table :data="siteForm.originHeaders" border class="config-table">
                  <el-table-column type="selection" width="50" />
                  <el-table-column label="名称" min-width="200" prop="name" />
                  <el-table-column label="值" min-width="260" prop="value" />
                  <el-table-column label="操作" width="140">
                    <template #default="{ row, $index }">
                      <el-button type="primary" link size="normal" @click="openHeaderDialog('edit', row, $index)">编辑</el-button>
                      <el-button type="danger" link size="normal" @click="removeHeader($index)">删除</el-button>
                    </template>
                  </el-table-column>
                </el-table>
                <div class="help-text">这里添加的是节点请求源时带的请求头</div>
                <el-divider />
                <div class="section-title">访问日志</div>
                <el-form-item label="记录请求头" style="max-width: 500px;">
                  <el-switch v-model="siteForm.logRequestHeader" @change="saveSiteConfig" />
                  <div class="help-text">开启只会增加硬盘空间占用，可长期开启</div>
                </el-form-item>
                <el-form-item label="记录响应头" style="max-width: 500px;">
                  <el-switch v-model="siteForm.logResponseHeader" @change="saveSiteConfig" />
                  <div class="help-text">建议只在调试时开启，始终开启会增加CPU与硬盘占用</div>
                </el-form-item>
                <el-form-item label="记录请求体" style="max-width: 500px;">
                  <el-switch v-model="siteForm.logRequestBody" @change="saveSiteConfig" />
                  <div class="help-text">建议只在调试时开启，始终开启对节点性能消耗较大</div>
                </el-form-item>
                <el-form-item label="请求体大小限制" style="max-width: 500px;">
                  <el-input v-model="siteForm.postSizeLimit" @change="saveSiteConfig" />
                  <div class="help-text">单位KB</div>
                </el-form-item>
                <el-divider />
                <div class="section-title">其它</div>
                <el-form-item label="负载方式" style="max-width: 600px;">
                  <el-radio-group v-model="siteForm.balanceWay" @change="saveSiteConfig">
                    <el-radio label="rr">轮循</el-radio>
                    <el-radio label="ip_hash">定源</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item label="默认CC规则" style="max-width: 500px;">
                  <el-select v-model="siteForm.ccDefaultRule" @change="saveSiteConfig">
                    <el-option v-for="item in ccRules" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                </el-form-item>
                <el-form-item label="搜索引擎爬虫" style="max-width: 600px;">
                  <el-radio-group v-model="siteForm.securityBot" @change="saveSiteConfig">
                    <el-radio label="">不设置</el-radio>
                    <el-radio label="allow">放行</el-radio>
                    <el-radio label="block">拦截</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item label="开启Gzip" style="max-width: 500px;">
                  <el-switch v-model="siteForm.gzipEnable" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="gzip types" style="max-width: 600px;">
                  <el-input v-model="siteForm.gzipTypes" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="开启Websocket" style="max-width: 500px;">
                  <el-switch v-model="siteForm.websocketEnable" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="屏蔽透明代理" style="max-width: 500px;">
                  <el-switch v-model="siteForm.securityShieldProxy" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="数据实时返回" style="max-width: 500px;">
                  <el-switch v-model="siteForm.realtimeReturn" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="数据实时发送" style="max-width: 500px;">
                  <el-switch v-model="siteForm.realtimeSend" @change="saveSiteConfig" />
                </el-form-item>
                <el-form-item label="开启IPv6" style="max-width: 500px;">
                  <el-switch v-model="siteForm.ipv6Enable" @change="saveSiteConfig" />
                </el-form-item>
              </el-form>
            </el-tab-pane>
            <el-tab-pane label="转发" name="stream">
              <el-form label-width="150px" class="config-form">
                <el-form-item label="监听协议" style="max-width: 600px;">
                  <el-radio-group v-model="streamForm.listenProtocol" @change="saveStreamConfig">
                    <el-radio label="tcp">tcp</el-radio>
                    <el-radio label="udp">udp</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item label="负载方式" style="max-width: 600px;">
                  <el-radio-group v-model="streamForm.balanceWay" @change="saveStreamConfig">
                    <el-radio label="rr">轮循</el-radio>
                    <el-radio label="ip_hash">定源</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item label="开启proxy protocol" style="max-width: 500px;">
                  <el-switch v-model="streamForm.proxyProtocol" @change="saveStreamConfig" />
                </el-form-item>
              </el-form>
            </el-tab-pane>
            <el-tab-pane label="证书" name="cert">
              <el-form label-width="150px" class="config-form">
                <el-form-item label="默认证书类型" style="max-width: 600px;">
                  <el-radio-group v-model="certForm.provider" @change="saveCertConfig">
                    <el-radio label="zerossl">zerossl</el-radio>
                    <el-radio label="lets">lets</el-radio>
                    <el-radio label="buypass">buypass</el-radio>
                    <el-radio label="google">google</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item label="DNS API" style="max-width: 500px;">
                  <el-select v-model="certForm.dnsapiId" @change="saveCertConfig">
                    <el-option
                      v-for="item in dnsapis"
                      :key="item.id"
                      :label="item.name"
                      :value="item.id" />
                  </el-select>
                  <div class="help-text">设置后，申请证书将使用此DNS API</div>
                </el-form-item>
              </el-form>
            </el-tab-pane>
          </el-tabs>
        </el-tab-pane>
        <el-tab-pane label="缓存配置" name="cache">
          <el-tabs v-model="cacheTab" class="sub-tabs">
            <el-tab-pane label="网站默认配置" name="site">
              <el-form label-width="150px" class="config-form">
                <el-form-item label="缓存开关" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.site.enable" @change="saveCacheDefaults('site')" />
                </el-form-item>
                <el-form-item label="缓存 TTL (秒)" style="max-width: 500px;">
                  <el-input v-model="cacheDefaults.site.ttl" @change="saveCacheDefaults('site')" />
                </el-form-item>
                <el-form-item label="开启 Gzip" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.site.gzip" @change="saveCacheDefaults('site')" />
                </el-form-item>
                <el-form-item label="开启 WAF" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.site.waf" @change="saveCacheDefaults('site')" />
                </el-form-item>
              </el-form>
            </el-tab-pane>
            <el-tab-pane label="API 默认配置" name="api">
              <el-form label-width="150px" class="config-form">
                <el-form-item label="缓存开关" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.api.enable" @change="saveCacheDefaults('api')" />
                </el-form-item>
                <el-form-item label="缓存 TTL (秒)" style="max-width: 500px;">
                  <el-input v-model="cacheDefaults.api.ttl" @change="saveCacheDefaults('api')" />
                </el-form-item>
                <el-form-item label="开启 Gzip" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.api.gzip" @change="saveCacheDefaults('api')" />
                </el-form-item>
                <el-form-item label="开启 WAF" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.api.waf" @change="saveCacheDefaults('api')" />
                </el-form-item>
              </el-form>
            </el-tab-pane>
            <el-tab-pane label="下载默认配置" name="download">
              <el-form label-width="150px" class="config-form">
                <el-form-item label="缓存开关" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.download.enable" @change="saveCacheDefaults('download')" />
                </el-form-item>
                <el-form-item label="缓存 TTL (秒)" style="max-width: 500px;">
                  <el-input v-model="cacheDefaults.download.ttl" @change="saveCacheDefaults('download')" />
                </el-form-item>
                <el-form-item label="开启 Gzip" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.download.gzip" @change="saveCacheDefaults('download')" />
                </el-form-item>
                <el-form-item label="开启 WAF" style="max-width: 500px;">
                  <el-switch v-model="cacheDefaults.download.waf" @change="saveCacheDefaults('download')" />
                </el-form-item>
              </el-form>
            </el-tab-pane>
          </el-tabs>
        </el-tab-pane>
      </el-tabs>
    </el-card>
    <el-dialog
      v-model="cacheRuleDialog.visible"
      :title="cacheRuleDialog.mode === 'create' ? '新增规则' : '编辑规则'"
      width="520px"
      @close="cacheRuleDialog.visible = false">
      <el-form label-width="100px">
        <el-form-item label="类型">
          <el-select v-model="cacheRuleForm.type" style="width: 100%">
            <el-option label="首页" value="index" />
            <el-option label="全站" value="all" />
            <el-option label="目录" value="dir" />
            <el-option label="后缀" value="suffix" />
            <el-option label="单个路径" value="path" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="cacheRuleForm.value" placeholder="需要匹配的内容" />
        </el-form-item>
        <el-form-item label="有效期(天)">
          <el-input v-model="cacheRuleForm.ttl" />
        </el-form-item>
        <el-form-item label="忽略参数">
          <el-switch v-model="cacheRuleForm.ignore_query" />
        </el-form-item>
        <el-form-item label="强制缓存">
          <el-switch v-model="cacheRuleForm.force_cache" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="normal" @click="cacheRuleDialog.visible = false">取消</el-button>
        <el-button size="normal" type="primary" @click="saveCacheRule">确定</el-button>
      </template>
    </el-dialog>
    <el-dialog
      v-model="headerDialog.visible"
      :title="headerDialog.mode === 'create' ? '新增请求头' : '编辑请求头'"
      width="520px"
      @close="headerDialog.visible = false">
      <el-form label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="headerForm.name" placeholder="请求头名称" />
        </el-form-item>
        <el-form-item label="值">
          <el-input v-model="headerForm.value" placeholder="请求头值" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="normal" @click="headerDialog.visible = false">取消</el-button>
        <el-button size="normal" type="primary" @click="saveHeader">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const topTab = ref('global')
const globalTab = ref('site')
const cacheTab = ref('site')

const defaultSslCiphers = [
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
].join(':')

const defaultGzipTypes = [
  'text/plain',
  'text/css',
  'text/xml',
  'text/javascript',
  'application/javascript',
  'application/x-javascript',
  'application/json'
].join(' ')

const siteForm = reactive({
  httpListen: '80',
  httpsListen: '443',
  httpsHsts: false,
  httpsHttp2: false,
  httpsHttp3: false,
  httpsForce: false,
  sslProtocols: ['TLSv1', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3'],
  sslCiphers: defaultSslCiphers,
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
  gzipTypes: defaultGzipTypes,
  websocketEnable: false,
  securityShieldProxy: false,
  realtimeReturn: false,
  realtimeSend: false,
  ipv6Enable: false
})

const streamForm = reactive({
  listenProtocol: 'tcp',
  balanceWay: 'rr',
  proxyProtocol: false
})

const certForm = reactive({
  provider: 'lets',
  dnsapiId: 0
})

const cacheDefaults = reactive({
  site: { enable: true, ttl: '86400', gzip: true, waf: true },
  api: { enable: false, ttl: '0', gzip: true, waf: false },
  download: { enable: true, ttl: '86400', gzip: true, waf: true }
})

const cacheRuleDialog = reactive({
  visible: false,
  mode: 'create',
  index: -1
})

const cacheRuleForm = reactive({
  type: 'index',
  value: '',
  ttl: '1',
  ignore_query: false,
  force_cache: false
})

const headerDialog = reactive({
  visible: false,
  mode: 'create',
  index: -1
})

const headerForm = reactive({
  name: '',
  value: ''
})

const cacheQuickPreset = ref('')

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

const dnsapis = ref([])

const cacheTypeLabelMap = {
  index: '首页',
  all: '全站',
  dir: '目录',
  suffix: '后缀',
  path: '单个路径'
}

function cacheTypeLabel(type) {
  return cacheTypeLabelMap[type] || type
}

function parseBool(value, def = false) {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (typeof value === 'string') {
    const v = value.trim().toLowerCase()
    if (v === '') return def
    return v === '1' || v === 'true' || v === 'on' || v === 'yes'
  }
  return def
}

function parseList(value) {
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

function toStr(value, fallback = '') {
  if (value === undefined || value === null) return fallback
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return fallback
}

function toIntSafe(value, fallback = 0) {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const parsed = Number(value)
    return Number.isNaN(parsed) ? fallback : parsed
  }
  return fallback
}

function normalizeCacheRule(rule) {
  if (!rule) return null
  const ttl = rule.ttl || rule.expire || rule.cache_time || ''
  return {
    type: rule.type || 'index',
    value: rule.value || rule.content || '',
    ttl: String(ttl || '1'),
    ignore_query: parseBool(rule.ignore_query, false),
    force_cache: parseBool(rule.force_cache, false)
  }
}

function buildSiteConfigPayload() {
  return {
    'http_listen-port': siteForm.httpListen,
    'https_listen-port': siteForm.httpsListen,
    'https_listen-hsts': siteForm.httpsHsts,
    'https_listen-http2': siteForm.httpsHttp2,
    'https_listen-http3': siteForm.httpsHttp3,
    'https_listen-force_ssl_enable': siteForm.httpsForce,
    'https_listen-ssl_protocols': siteForm.sslProtocols.join(' '),
    'https_listen-ssl_ciphers': siteForm.sslCiphers,
    'https_listen-ssl_prefer_server_ciphers': siteForm.sslPreferServerCiphers ? 'on' : 'off',
    'https_listen-ocsp_stapling': siteForm.ocspStapling,
    'backend_protocol': siteForm.backendProtocol,
    'backend_http_port': siteForm.backendHttpPort,
    'backend_https_port': siteForm.backendHttpsPort,
    'proxy_timeout': siteForm.proxyTimeout,
    'connect_timeout': siteForm.connectTimeout,
    'proxy_ssl_protocols': siteForm.proxySslProtocols.join(' '),
    'proxy_cache': JSON.stringify(siteForm.cacheRules),
    'origin_headers': JSON.stringify(siteForm.originHeaders),
    'log_request_header': siteForm.logRequestHeader,
    'log_response_header': siteForm.logResponseHeader,
    'log_request_body': siteForm.logRequestBody,
    'post_size_limit': siteForm.postSizeLimit,
    'balance_way': siteForm.balanceWay,
    'cc_default_rule': siteForm.ccDefaultRule,
    'security_bot': siteForm.securityBot,
    'gzip_enable': siteForm.gzipEnable,
    'gzip_types': siteForm.gzipTypes,
    'websocket_enable': siteForm.websocketEnable,
    'security_shield_proxy': siteForm.securityShieldProxy,
    'realtime_send': siteForm.realtimeSend,
    'realtime_return': siteForm.realtimeReturn,
    'ipv6_enable': siteForm.ipv6Enable
  }
}

const saveSiteConfig = debounce(async () => {
  try {
    await request.post('/site_defaults', {
      scope_name: 'global',
      scope_id: 0,
      data: buildSiteConfigPayload()
    })
  } catch (err) {
    ElMessage.error('保存失败')
  }
}, 300)

const saveStreamConfig = debounce(async () => {
  try {
    await request.post('/config_items', {
      type: 'stream_default_config',
      scope_name: 'global',
      scope_id: 0,
      value: JSON.stringify({
        listen_protocol: streamForm.listenProtocol,
        balance_way: streamForm.balanceWay,
        proxy_protocol: streamForm.proxyProtocol
      })
    })
  } catch (err) {
    ElMessage.error('保存失败')
  }
}, 300)

const saveCertConfig = debounce(async () => {
  try {
    await request.post('/certs/default_settings', {
      cert_provider: certForm.provider,
      dnsapi_id: certForm.dnsapiId
    })
  } catch (err) {
    ElMessage.error('保存失败')
  }
}, 300)

const saveCacheDefaults = debounce(async (mode) => {
  try {
    await request.post('/global_config', {
      name: `cache_default_${mode}`,
      value: JSON.stringify(cacheDefaults[mode])
    })
  } catch (err) {
    ElMessage.error('保存失败')
  }
}, 300)

function openCacheRuleDialog(mode, rule, index) {
  cacheRuleDialog.mode = mode
  cacheRuleDialog.index = index ?? -1
  if (rule) {
    Object.assign(cacheRuleForm, normalizeCacheRule(rule))
  } else {
    Object.assign(cacheRuleForm, {
      type: 'index',
      value: '',
      ttl: '1',
      ignore_query: false,
      force_cache: false
    })
  }
  cacheRuleDialog.visible = true
}

function saveCacheRule() {
  const newRule = normalizeCacheRule(cacheRuleForm)
  if (!newRule) return
  if (cacheRuleDialog.mode === 'edit' && cacheRuleDialog.index >= 0) {
    siteForm.cacheRules.splice(cacheRuleDialog.index, 1, newRule)
  } else {
    siteForm.cacheRules.push(newRule)
  }
  cacheRuleDialog.visible = false
  saveSiteConfig()
}

function removeCacheRule(index) {
  siteForm.cacheRules.splice(index, 1)
  saveSiteConfig()
}

function openHeaderDialog(mode, row, index) {
  headerDialog.mode = mode
  headerDialog.index = index ?? -1
  if (row) {
    Object.assign(headerForm, row)
  } else {
    Object.assign(headerForm, { name: '', value: '' })
  }
  headerDialog.visible = true
}

function saveHeader() {
  const payload = { name: headerForm.name, value: headerForm.value }
  if (headerDialog.mode === 'edit' && headerDialog.index >= 0) {
    siteForm.originHeaders.splice(headerDialog.index, 1, payload)
  } else {
    siteForm.originHeaders.push(payload)
  }
  headerDialog.visible = false
  saveSiteConfig()
}

function removeHeader(index) {
  siteForm.originHeaders.splice(index, 1)
  saveSiteConfig()
}

function applyCachePreset() {
  if (!cacheQuickPreset.value) return
  if (cacheQuickPreset.value === 'static') {
    siteForm.cacheRules = [
      { type: 'suffix', value: 'jpg|jpeg|png|gif|ico|css|js|svg|bmp|webp|woff|woff2', ttl: '7', ignore_query: true, force_cache: false }
    ]
  } else if (cacheQuickPreset.value === 'video') {
    siteForm.cacheRules = [
      { type: 'suffix', value: 'mp4|avi|mov|webm|m3u8|ts', ttl: '30', ignore_query: false, force_cache: false }
    ]
  } else if (cacheQuickPreset.value === 'image') {
    siteForm.cacheRules = [
      { type: 'suffix', value: 'jpg|jpeg|png|gif|webp', ttl: '15', ignore_query: true, force_cache: false }
    ]
  }
  saveSiteConfig()
}

function debounce(fn, wait) {
  let timer
  return (...args) => {
    clearTimeout(timer)
    timer = setTimeout(() => fn(...args), wait)
  }
}
async function loadSiteConfig() {
  const res = await request.get('/site_defaults', {
    params: { scope_name: 'global', scope_id: 0 }
  })
  const data = res?.data?.data?.list || []
  const map = {}
  data.forEach((item) => {
    map[item.name] = item.value
  })
  if (map['http_listen-port'] !== undefined) siteForm.httpListen = toStr(map['http_listen-port'], siteForm.httpListen)
  if (map['https_listen-port'] !== undefined) siteForm.httpsListen = toStr(map['https_listen-port'], siteForm.httpsListen)
  siteForm.httpsHsts = parseBool(map['https_listen-hsts'], siteForm.httpsHsts)
  siteForm.httpsHttp2 = parseBool(map['https_listen-http2'], siteForm.httpsHttp2)
  siteForm.httpsHttp3 = parseBool(map['https_listen-http3'], siteForm.httpsHttp3)
  siteForm.httpsForce = parseBool(map['https_listen-force_ssl_enable'], siteForm.httpsForce)
  if (Array.isArray(map['https_listen-ssl_protocols'])) {
    siteForm.sslProtocols = map['https_listen-ssl_protocols']
      .map((item) => toStr(item))
      .filter((item) => item !== '')
  } else if (map['https_listen-ssl_protocols']) {
    siteForm.sslProtocols = String(map['https_listen-ssl_protocols']).split(/\s+/).filter(Boolean)
  }
  if (map['https_listen-ssl_ciphers'] !== undefined) {
    siteForm.sslCiphers = toStr(map['https_listen-ssl_ciphers'], siteForm.sslCiphers)
  }
  siteForm.sslPreferServerCiphers = parseBool(map['https_listen-ssl_prefer_server_ciphers'], siteForm.sslPreferServerCiphers)
  siteForm.ocspStapling = parseBool(map['https_listen-ocsp_stapling'], siteForm.ocspStapling)
  if (map['backend_protocol']) siteForm.backendProtocol = toStr(map['backend_protocol'], siteForm.backendProtocol)
  if (map['backend_http_port'] !== undefined) siteForm.backendHttpPort = toStr(map['backend_http_port'], siteForm.backendHttpPort)
  if (map['backend_https_port'] !== undefined) siteForm.backendHttpsPort = toStr(map['backend_https_port'], siteForm.backendHttpsPort)
  if (map['proxy_timeout'] !== undefined) siteForm.proxyTimeout = toStr(map['proxy_timeout'], siteForm.proxyTimeout)
  if (map['connect_timeout'] !== undefined) siteForm.connectTimeout = toStr(map['connect_timeout'], siteForm.connectTimeout)
  if (Array.isArray(map['proxy_ssl_protocols'])) {
    siteForm.proxySslProtocols = map['proxy_ssl_protocols']
      .map((item) => toStr(item))
      .filter((item) => item !== '')
  } else if (map['proxy_ssl_protocols']) {
    siteForm.proxySslProtocols = String(map['proxy_ssl_protocols']).split(/\s+/).filter(Boolean)
  }
  siteForm.cacheRules = parseList(map['proxy_cache']).map(normalizeCacheRule).filter(Boolean)
  siteForm.originHeaders = parseList(map['origin_headers'])
  siteForm.logRequestHeader = parseBool(map['log_request_header'], siteForm.logRequestHeader)
  siteForm.logResponseHeader = parseBool(map['log_response_header'], siteForm.logResponseHeader)
  siteForm.logRequestBody = parseBool(map['log_request_body'], siteForm.logRequestBody)
  if (map['post_size_limit'] !== undefined) siteForm.postSizeLimit = toStr(map['post_size_limit'], siteForm.postSizeLimit)
  if (map['balance_way']) siteForm.balanceWay = toStr(map['balance_way'], siteForm.balanceWay)
  if (map['cc_default_rule'] !== undefined && map['cc_default_rule'] !== null) {
    const parsed = Number(map['cc_default_rule'])
    if (!Number.isNaN(parsed)) siteForm.ccDefaultRule = parsed
  }
  if (map['security_bot'] !== undefined) siteForm.securityBot = toStr(map['security_bot'], siteForm.securityBot)
  siteForm.gzipEnable = parseBool(map['gzip_enable'], siteForm.gzipEnable)
  if (map['gzip_types'] !== undefined) siteForm.gzipTypes = toStr(map['gzip_types'], siteForm.gzipTypes)
  siteForm.websocketEnable = parseBool(map['websocket_enable'], siteForm.websocketEnable)
  siteForm.securityShieldProxy = parseBool(map['security_shield_proxy'], siteForm.securityShieldProxy)
  siteForm.realtimeSend = parseBool(map['realtime_send'], siteForm.realtimeSend)
  siteForm.realtimeReturn = parseBool(map['realtime_return'], siteForm.realtimeReturn)
  siteForm.ipv6Enable = parseBool(map['ipv6_enable'], siteForm.ipv6Enable)
}

async function loadStreamConfig() {
  const res = await request.get('/config_items', {
    params: { type: 'stream_default_config', scope_name: 'global', scope_id: 0 }
  })
  const item = res?.data?.data
  if (!item?.value) return
  try {
    const value = JSON.parse(item.value)
    if (value.listen_protocol) streamForm.listenProtocol = value.listen_protocol
    if (value.balance_way) streamForm.balanceWay = value.balance_way
    if (value.proxy_protocol !== undefined) streamForm.proxyProtocol = parseBool(value.proxy_protocol)
  } catch (err) {
    return
  }
}

async function loadCertConfig() {
  const res = await request.get('/certs/default_settings')
  const data = res?.data?.data || {}
  if (data.cert_provider) certForm.provider = data.cert_provider
  if (data.dnsapi_id !== undefined && data.dnsapi_id !== null) {
    certForm.dnsapiId = toIntSafe(data.dnsapi_id, 0)
  }
}

async function loadCacheDefaults() {
  const res = await request.get('/global_config')
  const items = res?.data?.data || []
  items.forEach((item) => {
    if (!item?.name) return
    if (item.name === 'cache_default_site') {
      Object.assign(cacheDefaults.site, safeParse(item.value))
    }
    if (item.name === 'cache_default_api') {
      Object.assign(cacheDefaults.api, safeParse(item.value))
    }
    if (item.name === 'cache_default_download') {
      Object.assign(cacheDefaults.download, safeParse(item.value))
    }
  })
}

function safeParse(value) {
  if (!value) return {}
  if (typeof value === 'object') return value
  try {
    return JSON.parse(value)
  } catch (err) {
    return {}
  }
}

async function loadDnsApis() {
  const res = await request.get('/dnsapi')
  dnsapis.value = res?.data?.data?.list || []
}

async function loadCcRules() {
  const res = await request.get('/rules/cc/groups')
  const list = res?.data?.data?.list || []
  if (list.length > 0) {
    ccRules.value = list.map((item) => ({
      label: item.name,
      value: item.id
    }))
  }
}

loadSiteConfig()
loadStreamConfig()
loadCertConfig()
loadCacheDefaults()
loadDnsApis()
loadCcRules()
</script>

<style scoped>
.default-config {
  padding: 10px;
}

.layout-card :deep(.el-card__header) {
  padding: 12px 20px;
}

.card-header {
  font-size: 16px;
  font-weight: 600;
}

.section-title {
  font-weight: 600;
  margin-bottom: 10px;
}

.config-form {
  padding: 10px 0;
}

.toolbar-row {
  display: flex;
  gap: 12px;
  margin-bottom: 10px;
}

.config-table {
  margin-bottom: 6px;
}

.help-text {
  color: #8c8c8c;
  font-size: 12px;
  margin-top: 6px;
}
</style>
