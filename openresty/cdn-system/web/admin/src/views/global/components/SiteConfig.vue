<template>
  <el-form v-loading="loading" label-width="150px" class="config-form">
    <div class="section-title">HTTP</div>
    <el-form-item label="监听端口" style="max-width: 500px;">
      <el-input v-model="form.httpListen" @change="saveConfig" />
    </el-form-item>
    <el-divider />
    <div class="section-title">HTTPS</div>
    <el-form-item label="监听端口" style="max-width: 500px;">
      <el-input v-model="form.httpsListen" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="开启HSTS" style="max-width: 500px;">
      <el-switch v-model="form.httpsHsts" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="开启HTTP2" style="max-width: 500px;">
      <el-switch v-model="form.httpsHttp2" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="开启HTTP3" style="max-width: 500px;">
      <el-switch v-model="form.httpsHttp3" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="强制HTTPS" style="max-width: 500px;">
      <el-switch v-model="form.httpsForce" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="ssl_protocols" style="max-width: 600px;">
      <el-checkbox-group v-model="form.sslProtocols" @change="saveConfig">
        <el-checkbox value="SSLv2">SSLv2</el-checkbox>
        <el-checkbox value="SSLv3">SSLv3</el-checkbox>
        <el-checkbox value="TLSv1">TLSv1</el-checkbox>
        <el-checkbox value="TLSv1.1">TLSv1.1</el-checkbox>
        <el-checkbox value="TLSv1.2">TLSv1.2</el-checkbox>
        <el-checkbox value="TLSv1.3">TLSv1.3</el-checkbox>
      </el-checkbox-group>
    </el-form-item>
    <el-form-item label="ssl_ciphers" style="max-width: 600px;">
      <el-input
        v-model="form.sslCiphers"
        type="textarea"
        :rows="2"
        @change="saveConfig" />
    </el-form-item>
    <el-form-item label="ssl_prefer_server_ciphers" style="max-width: 500px;">
      <el-switch v-model="form.sslPreferServerCiphers" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="ocsp_stapling" style="max-width: 500px;">
      <el-switch v-model="form.ocspStapling" @change="saveConfig" />
    </el-form-item>
    <el-divider />
    <div class="section-title">回源设置</div>
    <el-form-item label="回源协议" style="max-width: 600px;">
      <el-radio-group v-model="form.backendProtocol" @change="saveConfig">
        <el-radio value="http">HTTP</el-radio>
        <el-radio value="https">HTTPS</el-radio>
        <el-radio value="follow">跟随协议</el-radio>
        <el-radio value="follow_port">跟随端口和协议</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="回源http端口" style="max-width: 500px;">
      <el-input v-model="form.backendHttpPort" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="回源https端口" style="max-width: 500px;">
      <el-input v-model="form.backendHttpsPort" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="回源超时" style="max-width: 500px;">
      <el-input v-model="form.proxyTimeout" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="连接超时" style="max-width: 500px;">
      <el-input v-model="form.connectTimeout" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="回源SSL协议" style="max-width: 600px;">
      <el-checkbox-group v-model="form.proxySslProtocols" @change="saveConfig">
        <el-checkbox value="SSLv2">SSLv2</el-checkbox>
        <el-checkbox value="SSLv3">SSLv3</el-checkbox>
        <el-checkbox value="TLSv1">TLSv1</el-checkbox>
        <el-checkbox value="TLSv1.1">TLSv1.1</el-checkbox>
        <el-checkbox value="TLSv1.2">TLSv1.2</el-checkbox>
        <el-checkbox value="TLSv1.3">TLSv1.3</el-checkbox>
      </el-checkbox-group>
    </el-form-item>
    <el-divider />
    <div class="section-title">缓存</div>
    <div class="toolbar-row">
      <el-button type="primary" size="default" @click="openCacheRuleDialog('create')">新增规则</el-button>
      <el-form-item label="快速添加缓存配置" label-width="130px">
        <el-select
          v-model="cacheQuickPreset"
          style="width: 140px; margin-left: 10px;"
          @change="applyCachePreset">
          <el-option label="首页缓存" value="index" />
          <el-option label="全站缓存" value="all" />
          <el-option label="静态资源缓存" value="static" />
          <el-option label="视频文件缓存" value="video" />
          <el-option label="Wordpress缓存" value="wordpress" />
        </el-select>
      </el-form-item>
    </div>
    <el-table :data="form.cacheRules" border class="config-table">
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
          <span>{{ formatTTL(row.ttl) }}</span>
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
          <el-button type="primary" link size="default" @click="openCacheRuleDialog('edit', row, $index)">编辑</el-button>
          <el-button type="danger" link size="default" @click="removeCacheRule($index)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="help-text">缓存规则由上到下匹配，匹配到即停止，可拖动调整顺序</div>
    <el-divider />
    <div class="section-title">源站请求头</div>
    <div class="toolbar-row">
      <el-button type="primary" size="default" @click="openHeaderDialog('create')">新增请求头</el-button>
    </div>
    <el-table :data="form.originHeaders" border class="config-table">
      <el-table-column type="selection" width="50" />
      <el-table-column label="名称" min-width="200" prop="name" />
      <el-table-column label="值" min-width="260" prop="value" />
      <el-table-column label="操作" width="140">
        <template #default="{ row, $index }">
          <el-button type="primary" link size="default" @click="openHeaderDialog('edit', row, $index)">编辑</el-button>
          <el-button type="danger" link size="default" @click="removeHeader($index)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="help-text">这里添加的是节点请求源时带的请求头</div>
    <el-divider />
    <div class="section-title">访问日志</div>
    <el-form-item label="记录请求头" style="max-width: 500px;">
      <el-switch v-model="form.logRequestHeader" @change="saveConfig" />
      <div class="help-text">开启只会增加硬盘空间占用，可长期开启</div>
    </el-form-item>
    <el-form-item label="记录响应头" style="max-width: 500px;">
      <el-switch v-model="form.logResponseHeader" @change="saveConfig" />
      <div class="help-text">建议只在调试时开启，始终开启会增加CPU与硬盘占用</div>
    </el-form-item>
    <el-form-item label="记录请求体" style="max-width: 500px;">
      <el-switch v-model="form.logRequestBody" @change="saveConfig" />
      <div class="help-text">建议只在调试时开启，始终开启对节点性能消耗较大</div>
    </el-form-item>
    <el-form-item label="请求体大小限制" style="max-width: 500px;">
      <el-input v-model="form.postSizeLimit" @change="saveConfig" />
      <div class="help-text">单位KB</div>
    </el-form-item>
    <el-divider />
    <div class="section-title">其它</div>
    <el-form-item label="负载方式" style="max-width: 600px;">
      <el-radio-group v-model="form.balanceWay" @change="saveConfig">
        <el-radio value="rr">轮循</el-radio>
        <el-radio value="ip_hash">定源</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="默认CC规则" style="max-width: 500px;">
      <el-select v-model="form.ccDefaultRule" @change="saveConfig">
        <el-option v-for="item in ccRules" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
    </el-form-item>
    <el-form-item label="搜索引擎爬虫" style="max-width: 600px;">
      <el-radio-group v-model="form.securityBot" @change="saveConfig">
        <el-radio value="">不设置</el-radio>
        <el-radio value="allow">放行</el-radio>
        <el-radio value="block">拦截</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="开启Gzip" style="max-width: 500px;">
      <el-switch v-model="form.gzipEnable" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="gzip types" style="max-width: 600px;">
      <el-input v-model="form.gzipTypes" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="开启Websocket" style="max-width: 500px;">
      <el-switch v-model="form.websocketEnable" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="屏蔽透明代理" style="max-width: 500px;">
      <el-switch v-model="form.securityShieldProxy" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="数据实时返回" style="max-width: 500px;">
      <el-switch v-model="form.realtimeReturn" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="数据实时发送" style="max-width: 500px;">
      <el-switch v-model="form.realtimeSend" @change="saveConfig" />
    </el-form-item>
    <el-form-item label="开启IPv6" style="max-width: 500px;">
      <el-switch v-model="form.ipv6Enable" @change="saveConfig" />
    </el-form-item>

    <!-- Dialogs -->
    <el-dialog
      v-model="cacheRuleDialog.visible"
      :title="cacheRuleDialog.mode === 'create' ? '新增缓存规则' : '编辑缓存规则'"
      width="600px">
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
          <el-input v-model="cacheRuleForm.value" placeholder="多个用|分隔" />
        </el-form-item>
        <el-form-item label="有效期">
          <el-input v-model="cacheRuleForm.ttl_value" placeholder="时长">
            <template #append>
              <el-select v-model="cacheRuleForm.ttl_unit" style="width: 80px">
                <el-option label="天" value="day" />
                <el-option label="时" value="hour" />
                <el-option label="分" value="minute" />
                <el-option label="秒" value="second" />
              </el-select>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="忽略参数">
          <el-switch v-model="cacheRuleForm.ignore_query" />
        </el-form-item>
        <el-form-item label="强制缓存">
          <el-switch v-model="cacheRuleForm.force_cache" />
        </el-form-item>

        <div class="more-settings-divider">
           <el-divider border-style="dashed">
             <span class="more-settings-toggle" @click="moreSettingsVisible = !moreSettingsVisible" style="cursor: pointer; color: #666; font-size: 12px;">
                更多设置 <el-icon><ArrowDown v-if="!moreSettingsVisible" /><ArrowUp v-else /></el-icon>
             </span>
           </el-divider>
        </div>

        <div v-show="moreSettingsVisible">
            <el-form-item label="分片回源">
                <el-switch v-model="cacheRuleForm.enable_range" />
            </el-form-item>
            <el-form-item label="忽略Vary">
                <el-switch v-model="cacheRuleForm.ignore_vary" />
            </el-form-item>
            <el-form-item label-width="0">
                <div style="margin-bottom: 5px; font-weight: bold; color: #606266;">不缓存条件：</div>
                <div class="condition-list">
                    <el-table :data="cacheRuleForm.skip_conditions" size="small" border empty-text="暂无数据">
                        <el-table-column prop="type" label="匹配项" width="100">
                             <template #default="{ row }">{{ matchTypeLabel(row.type) }}</template>
                        </el-table-column>
                        <el-table-column prop="value" label="匹配值" />
                        <el-table-column label="操作" width="60" align="center">
                             <template #default="{ $index }">
                                 <el-button link type="danger" @click="removeCondition($index)">删除</el-button>
                             </template>
                        </el-table-column>
                    </el-table>
                    <div style="display: flex; margin-top: 5px; gap: 5px;">
                         <el-select v-model="newCondition.type" placeholder="选择匹配项" size="small" style="width: 140px">
                             <el-option label="请求URI" value="request_uri" />
                             <el-option label="请求URI(不带参数)" value="uri" />
                             <el-option label="客户IP地址" value="remote_addr" />
                             <el-option label="请求协议" value="scheme" />
                             <el-option label="请求参数" value="args" />
                             <el-option label="域名" value="host" />
                             <el-option label="自定义" value="custom" />
                         </el-select>
                         <el-input v-model="newCondition.value" placeholder="请输入匹配值" size="small" style="flex: 1" />
                         <el-button type="primary" size="small" @click="addCondition">添加</el-button>
                    </div>
                </div>
            </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button size="default" @click="cacheRuleDialog.visible = false">取消</el-button>
        <el-button size="default" type="primary" @click="saveCacheRule">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="headerDialog.visible"
      :title="headerDialog.mode === 'create' ? '新增请求头' : '编辑请求头'"
      width="520px">
      <el-form label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="headerForm.name" placeholder="请求头名称" />
        </el-form-item>
        <el-form-item label="值">
          <el-input v-model="headerForm.value" placeholder="请求头值" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="default" @click="headerDialog.visible = false">取消</el-button>
        <el-button size="default" type="primary" @click="saveHeader">确定</el-button>
      </template>
    </el-dialog>
  </el-form>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { 
  toStr, parseBool, parseList, 
  convertSecondsToUnit, convertUnitToSeconds, 
  normalizeCacheRule, debounce, formatTTL 
} from '../utils'

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

const form = reactive({
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

const loading = ref(false)

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

const cacheRuleDialog = reactive({ visible: false, mode: 'create', index: -1 })
const cacheRuleForm = reactive({
  type: 'index', value: '', ttl: '86400', ttl_value: '1', ttl_unit: 'day',
  ignore_query: false, force_cache: false, enable_range: false, ignore_vary: false, skip_conditions: []
})
const moreSettingsVisible = ref(false)
const newCondition = reactive({ type: 'request_uri', value: '' })
const headerDialog = reactive({ visible: false, mode: 'create', index: -1 })
const headerForm = reactive({ name: '', value: '' })
const cacheQuickPreset = ref('')

const cacheTypeLabelMap = { index: '首页', all: '全站', dir: '目录', suffix: '后缀', path: '单个路径' }
const matchTypeLabelMap = {
    request_uri: '请求URI', uri: '请求URI(不带参数)', remote_addr: '客户IP地址',
    scheme: '请求协议', args: '请求参数', host: '域名', custom: '自定义'
}

const matchTypeLabel = (type) => matchTypeLabelMap[type] || type
const cacheTypeLabel = (type) => cacheTypeLabelMap[type] || type

const saveConfig = debounce(async () => {
  try {
    const payload = {
      'http_listen-port': form.httpListen,
      'https_listen-port': form.httpsListen,
      'https_listen-hsts': form.httpsHsts,
      'https_listen-http2': form.httpsHttp2,
      'https_listen-http3': form.httpsHttp3,
      'https_listen-force_ssl_enable': form.httpsForce,
      'https_listen-ssl_protocols': form.sslProtocols.join(' '),
      'https_listen-ssl_ciphers': form.sslCiphers,
      'https_listen-ssl_prefer_server_ciphers': form.sslPreferServerCiphers ? 'on' : 'off',
      'https_listen-ocsp_stapling': form.ocspStapling,
      'backend_protocol': form.backendProtocol,
      'backend_http_port': form.backendHttpPort,
      'backend_https_port': form.backendHttpsPort,
      'proxy_timeout': form.proxyTimeout,
      'connect_timeout': form.connectTimeout,
      'proxy_ssl_protocols': form.proxySslProtocols.join(' '),
      'proxy_cache': JSON.stringify(form.cacheRules),
      'origin_headers': JSON.stringify(form.originHeaders),
      'log_request_header': form.logRequestHeader,
      'log_response_header': form.logResponseHeader,
      'log_request_body': form.logRequestBody,
      'post_size_limit': form.postSizeLimit,
      'balance_way': form.balanceWay,
      'cc_default_rule': form.ccDefaultRule,
      'security_bot': form.securityBot,
      'gzip_enable': form.gzipEnable,
      'gzip_types': form.gzipTypes,
      'websocket_enable': form.websocketEnable,
      'security_shield_proxy': form.securityShieldProxy,
      'realtime_send': form.realtimeSend,
      'realtime_return': form.realtimeReturn,
      'ipv6_enable': form.ipv6Enable
    }
    await request.post('/site_defaults', { scope_name: 'global', scope_id: 0, data: payload })
    ElMessage.success('保存成功')
  } catch (err) {
    ElMessage.error('保存失败')
  }
}, 300)

const loadConfig = async () => {
  const res = await request.get('/site_defaults', { params: { scope_name: 'global', scope_id: 0 } })
  const data = res?.data?.list || []
  const map = {}
  data.forEach((item) => { map[item.name] = item.value })

  if (map['http_listen-port'] !== undefined) form.httpListen = toStr(map['http_listen-port'], form.httpListen)
  if (map['https_listen-port'] !== undefined) form.httpsListen = toStr(map['https_listen-port'], form.httpsListen)
  form.httpsHsts = parseBool(map['https_listen-hsts'], form.httpsHsts)
  form.httpsHttp2 = parseBool(map['https_listen-http2'], form.httpsHttp2)
  form.httpsHttp3 = parseBool(map['https_listen-http3'], form.httpsHttp3)
  form.httpsForce = parseBool(map['https_listen-force_ssl_enable'], form.httpsForce)
  if (map['https_listen-ssl_protocols']) {
     form.sslProtocols = toStr(map['https_listen-ssl_protocols']).split(/\s+/).filter(Boolean)
  }
  if (map['https_listen-ssl_ciphers'] !== undefined) form.sslCiphers = toStr(map['https_listen-ssl_ciphers'], form.sslCiphers)
  form.sslPreferServerCiphers = parseBool(map['https_listen-ssl_prefer_server_ciphers'], form.sslPreferServerCiphers)
  form.ocspStapling = parseBool(map['https_listen-ocsp_stapling'], form.ocspStapling)
  if (map['backend_protocol']) form.backendProtocol = toStr(map['backend_protocol'], form.backendProtocol)
  if (map['backend_http_port'] !== undefined) form.backendHttpPort = toStr(map['backend_http_port'], form.backendHttpPort)
  if (map['backend_https_port'] !== undefined) form.backendHttpsPort = toStr(map['backend_https_port'], form.backendHttpsPort)
  if (map['proxy_timeout'] !== undefined) form.proxyTimeout = toStr(map['proxy_timeout'], form.proxyTimeout)
  if (map['connect_timeout'] !== undefined) form.connectTimeout = toStr(map['connect_timeout'], form.connectTimeout)
  if (map['proxy_ssl_protocols']) {
      form.proxySslProtocols = toStr(map['proxy_ssl_protocols']).split(/\s+/).filter(Boolean)
  }
  form.cacheRules = parseList(map['proxy_cache']).map(normalizeCacheRule).filter(Boolean)
  form.originHeaders = parseList(map['origin_headers']).map((item) => ({
    name: toStr(item?.name || item?.key || '', ''),
    value: toStr(item?.value || '', '')
  }))
  form.logRequestHeader = parseBool(map['log_request_header'], form.logRequestHeader)
  form.logResponseHeader = parseBool(map['log_response_header'], form.logResponseHeader)
  form.logRequestBody = parseBool(map['log_request_body'], form.logRequestBody)
  if (map['post_size_limit'] !== undefined) form.postSizeLimit = toStr(map['post_size_limit'], form.postSizeLimit)
  if (map['balance_way']) form.balanceWay = toStr(map['balance_way'], form.balanceWay)
  if (map['cc_default_rule'] !== undefined) form.ccDefaultRule = Number(map['cc_default_rule']) || form.ccDefaultRule
  if (map['security_bot'] !== undefined) form.securityBot = toStr(map['security_bot'], form.securityBot)
  form.gzipEnable = parseBool(map['gzip_enable'], form.gzipEnable)
  if (map['gzip_types'] !== undefined) form.gzipTypes = toStr(map['gzip_types'], form.gzipTypes)
  form.websocketEnable = parseBool(map['websocket_enable'], form.websocketEnable)
  form.securityShieldProxy = parseBool(map['security_shield_proxy'], form.securityShieldProxy)
  form.realtimeSend = parseBool(map['realtime_send'], form.realtimeSend)
  form.realtimeReturn = parseBool(map['realtime_return'], form.realtimeReturn)
  form.ipv6Enable = parseBool(map['ipv6_enable'], form.ipv6Enable)
}

function openCacheRuleDialog(mode, rule, index) {
  cacheRuleDialog.mode = mode
  cacheRuleDialog.index = index ?? -1
  if (rule) {
    const norm = normalizeCacheRule(rule)
    Object.assign(cacheRuleForm, norm)
    const { value, unit } = convertSecondsToUnit(norm.ttl)
    cacheRuleForm.ttl_value = value
    cacheRuleForm.ttl_unit = unit
    moreSettingsVisible.value = norm.enable_range || norm.ignore_vary || (norm.skip_conditions && norm.skip_conditions.length > 0)
  } else {
    Object.assign(cacheRuleForm, {
      type: 'index', value: '', ttl: '86400', ttl_value: '1', ttl_unit: 'day',
      ignore_query: false, force_cache: false, enable_range: false, ignore_vary: false, skip_conditions: []
    })
    moreSettingsVisible.value = false
  }
  cacheRuleDialog.visible = true
}

function addCondition() { if (newCondition.value) cacheRuleForm.skip_conditions.push({ ...newCondition }); newCondition.value = '' }
function removeCondition(index) { cacheRuleForm.skip_conditions.splice(index, 1) }

function saveCacheRule() {
  cacheRuleForm.ttl = String(convertUnitToSeconds(cacheRuleForm.ttl_value, cacheRuleForm.ttl_unit))
  const newRule = normalizeCacheRule(cacheRuleForm)
  if (!newRule) return
  if (cacheRuleDialog.mode === 'edit' && cacheRuleDialog.index >= 0) {
    form.cacheRules.splice(cacheRuleDialog.index, 1, newRule)
  } else {
    form.cacheRules.push(newRule)
  }
  cacheRuleDialog.visible = false
  saveConfig()
}

function removeCacheRule(index) { form.cacheRules.splice(index, 1); saveConfig() }

function applyCachePreset(val) {
  if (!val) return
  let rule = null
  switch (val) {
    case 'index': rule = { type: 'index', value: '', ttl: '86400' }; break
    case 'all': rule = { type: 'all', value: '', ttl: '259200' }; break
    case 'static': rule = { type: 'suffix', value: 'jpg|jpeg|png|gif|ico|css|js|svg|bmp|webp|woff|woff2', ttl: '604800', ignore_query: true }; break
    case 'video': rule = { type: 'suffix', value: 'mp4|avi|mov|webm|m3u8|ts', ttl: '2592000' }; break
    case 'wordpress': rule = { type: 'all', value: '', ttl: '259200' }; break
  }
  if (rule) { form.cacheRules.push(normalizeCacheRule(rule)); saveConfig(); ElMessage.success('已添加规则') }
  cacheQuickPreset.value = ''
}

function openHeaderDialog(mode, row, index) {
  headerDialog.mode = mode; headerDialog.index = index ?? -1
  if (row) Object.assign(headerForm, row); else Object.assign(headerForm, { name: '', value: '' })
  headerDialog.visible = true
}

function saveHeader() {
  const payload = { name: headerForm.name, value: headerForm.value }
  if (headerDialog.mode === 'edit' && headerDialog.index >= 0) form.originHeaders.splice(headerDialog.index, 1, payload)
  else form.originHeaders.push(payload)
  headerDialog.visible = false; saveConfig()
}

function removeHeader(index) { form.originHeaders.splice(index, 1); saveConfig() }

async function loadCcRules() {
  const res = await request.get('/rules/cc/groups')
  const list = res?.data?.data?.list || []
  if (list.length > 0) ccRules.value = list.map((item) => ({ label: item.name, value: item.id }))
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadConfig(), loadCcRules()])
  } finally {
    loading.value = false
  }
})
</script>
