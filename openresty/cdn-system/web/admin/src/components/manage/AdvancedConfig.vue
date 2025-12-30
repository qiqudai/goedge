<template>
  <div class="advanced-config">
    <el-form label-width="150px" class="config-form">
      <div class="section-title">上传大小限制</div>
      <el-form-item label="大小限制">
        <div style="display: flex; align-items: center; gap: 10px;">
          <el-radio-group v-model="advancedSettings.uploadLimitMode">
            <el-radio value="none">不限制</el-radio>
            <el-radio value="custom">自定义</el-radio>
          </el-radio-group>
          <el-input 
            v-if="advancedSettings.uploadLimitMode === 'custom'" 
            v-model="advancedSettings.uploadLimitValue" 
            style="width: 150px" 
            placeholder="100"
          >
            <template #append>MB</template>
          </el-input>
        </div>
      </el-form-item>

      <div class="divider"></div>
      <div class="section-title">压缩设置</div>
      <el-form-item label="Gzip压缩">
        <el-switch v-model="advancedSettings.gzip" />
      </el-form-item>

      <div class="divider"></div>

      <div class="section-title">Websocket设置</div>
      <el-form-item label="Websocket">
        <el-switch v-model="advancedSettings.websocket" />
      </el-form-item>

      <div class="divider"></div>

      <div class="section-title">搜索引擎回源配置</div>
      <el-form-item label="开关">
        <el-switch v-model="advancedSettings.searchEngineOrigin" />
      </el-form-item>

      <div class="divider"></div>

      <div class="section-title">URL转向设置</div>
      <el-button type="primary" size="small" style="margin-bottom: 12px;" @click="openRedirectDialog()">新增转向</el-button>
      <el-table :data="advancedSettings.urlRedirects" border size="small">
        <el-table-column label="域名端口" prop="domain" />
        <el-table-column label="匹配" prop="match" />
        <el-table-column label="转向到" prop="redirect" />
        <el-table-column label="响应码" prop="code" width="100" />
        <el-table-column label="操作" width="140">
          <template #default="{ row, $index }">
            <el-button link type="primary" size="small" @click="openRedirectDialog(row, $index)">编辑</el-button>
            <el-button link type="danger" size="small" @click="removeRedirect($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="form-helper">这里的转向可以设置301，302转向到地址，也可以对uri重写再回源</div>

      <div class="divider"></div>
      
      <div class="section-title">请求头设置</div>
      <el-button type="primary" size="small" style="margin-bottom: 12px;" @click="openHeaderDialog('req')">新增请求头</el-button>
      <el-table :data="advancedSettings.reqHeaders" border size="small">
        <el-table-column label="名称" prop="name" />
        <el-table-column label="值" prop="value" />
        <el-table-column label="操作" width="140">
          <template #default="{ row, $index }">
            <el-button link type="primary" size="small" @click="openHeaderDialog('req', row, $index)">编辑</el-button>
            <el-button link type="danger" size="small" @click="removeHeader('req', $index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="divider"></div>

      <div class="section-title">CDN响应头设置</div>
      <el-button type="primary" size="small" style="margin-bottom: 12px;" @click="openHeaderDialog('res')">新增响应头</el-button>
      <el-table :data="advancedSettings.resHeaders" border size="small">
        <el-table-column label="名称" prop="name" />
        <el-table-column label="值" prop="value" />
        <el-table-column label="操作" width="140">
          <template #default="{ row, $index }">
            <el-button link type="primary" size="small" @click="openHeaderDialog('res', row, $index)">编辑</el-button>
            <el-button link type="danger" size="small" @click="removeHeader('res', $index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="divider"></div>
      
      <div class="section-title">访问日志</div>
      <el-form-item label="记录请求头">
        <el-switch v-model="advancedSettings.logRequestHeader" />
        <div class="form-helper">开启只会增加硬盘空间占用，可长期开启</div>
      </el-form-item>
      <el-form-item label="记录响应头">
        <el-switch v-model="advancedSettings.logResponseHeader" />
        <div class="form-helper">建议只在调试时开启，始终开启会增加cpu, 硬盘的占用</div>
      </el-form-item>
      <el-form-item label="记录请求体">
        <el-switch v-model="advancedSettings.logRequestBody" />
        <div class="form-helper">建议只在调试时开启，始终开启对节点性能消耗较大</div>
      </el-form-item>
      <el-form-item label="请求体大小限制">
        <el-input v-model="advancedSettings.logRequestBodySizeLimit" placeholder="16" style="width: 200px;">
          <template #append>KB</template>
        </el-input>
      </el-form-item>

      <div class="divider"></div>

      <div class="divider"></div>

      <div class="section-title">代理超时设置</div>
      <el-form-item label="连接超时">
        <el-input v-model="advancedSettings.proxyConnectTimeout" style="width: 200px;" placeholder="30s">
          <template #append>秒/单位</template>
        </el-input>
        <div class="form-helper">连接源站的超时时间，默认为30s</div>
      </el-form-item>
      <el-form-item label="读取超时">
        <el-input v-model="advancedSettings.proxyReadTimeout" style="width: 200px;" placeholder="60s">
          <template #append>秒/单位</template>
        </el-input>
        <div class="form-helper">读取源站响应的超时时间，默认为60s</div>
      </el-form-item>
      <el-form-item label="发送超时">
        <el-input v-model="advancedSettings.proxySendTimeout" style="width: 200px;" placeholder="60s">
          <template #append>秒/单位</template>
        </el-input>
        <div class="form-helper">发送请求到源站的超时时间，默认为60s</div>
      </el-form-item>

      <div class="divider"></div>

      <div class="section-title">上游长连接</div>
      <el-form-item label="开关">
        <el-switch v-model="advancedSettings.upstreamKeepalive" />
        <div class="form-helper">开启后，回源连接将复用HTTP连接，减少三次握手开销</div>
      </el-form-item>
      <template v-if="advancedSettings.upstreamKeepalive">
        <el-form-item label="最大空闲连接">
          <el-input v-model="advancedSettings.upstreamKeepaliveConn" style="width: 200px;" placeholder="100" />
          <div class="form-helper">每个Worker进程保留的最大空闲连接数</div>
        </el-form-item>
        <el-form-item label="超时时间">
          <el-input v-model="advancedSettings.upstreamKeepaliveTimeout" style="width: 200px;" placeholder="60">
            <template #append>秒</template>
          </el-input>
          <div class="form-helper">空闲连接的超时时间</div>
        </el-form-item>
      </template>

      <div class="divider"></div>

      <div class="section-title">流量限制</div>
      <el-form-item label="单连接限速">
        <el-input v-model="advancedSettings.limitRate" style="width: 200px;" placeholder="0">
          <template #append>KB/s</template>
        </el-input>
        <div class="form-helper">限制单个连接的下载速度，0表示不限制</div>
      </el-form-item>

      <div class="divider"></div>

      <div class="section-title">其它</div>
      <el-form-item label="源站证书">
        <el-switch v-model="advancedSettings.originCert" />
        <div class="form-helper">用于回源连接（HTTPS）验证源站证书</div>
      </el-form-item>
      <el-form-item label="数据实时鉴别">
        <el-switch v-model="advancedSettings.realtimeIdentify" />
      </el-form-item>
      <el-form-item label="数据实时发送">
        <el-switch v-model="advancedSettings.realtimeSend" />
      </el-form-item>
    </el-form>

    <!-- 转向规则弹窗 -->
    <RedirectRuleDialog
      v-model="redirectDialogVisible"
      :rule="editingRedirectRule"
      @submit="handleRedirectSubmit"
    />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import RedirectRuleDialog from '@/components/RedirectRuleDialog.vue'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  }
})

const emit = defineEmits([
  'update:modelValue', 
  'open-header-dialog',
  'remove-header'
])

const localSettings = ref({
  uploadLimitMode: props.modelValue?.uploadLimitMode || 'none',
  uploadLimitValue: props.modelValue?.uploadLimitValue || 100,
  gzip: props.modelValue?.gzip || false,
  websocket: props.modelValue?.websocket || false,
  searchEngineOrigin: props.modelValue?.searchEngineOrigin || false,
  urlRedirects: JSON.parse(JSON.stringify(props.modelValue?.urlRedirects || [])),
  reqHeaders: JSON.parse(JSON.stringify(props.modelValue?.reqHeaders || [])),
  resHeaders: JSON.parse(JSON.stringify(props.modelValue?.resHeaders || [])),
  logRequestHeader: props.modelValue?.logRequestHeader || false,
  logResponseHeader: props.modelValue?.logResponseHeader || false,
  logRequestBody: props.modelValue?.logRequestBody || false,
  logRequestBodySizeLimit: props.modelValue?.logRequestBodySizeLimit || 16,
  proxyConnectTimeout: props.modelValue?.proxyConnectTimeout || '30',
  proxyReadTimeout: props.modelValue?.proxyReadTimeout || '60',
  proxySendTimeout: props.modelValue?.proxySendTimeout || '60',
  upstreamKeepalive: props.modelValue?.upstreamKeepalive || false,
  upstreamKeepaliveConn: props.modelValue?.upstreamKeepaliveConn || 100,
  upstreamKeepaliveTimeout: props.modelValue?.upstreamKeepaliveTimeout || 60,
  limitRate: props.modelValue?.limitRate || 0,
  originCert: props.modelValue?.originCert || false,
  realtimeIdentify: props.modelValue?.realtimeIdentify || false,
  realtimeSend: props.modelValue?.realtimeSend || false
})

let isInternalUpdate = false

// 监听本地同步
watch(localSettings, (newVal) => {
  isInternalUpdate = true
  emit('update:modelValue', {
    ...props.modelValue,
    ...newVal
  })
}, { deep: true })

// 监听外部更新
watch(() => props.modelValue, (newVal) => {
  if (newVal && !isInternalUpdate) {
    localSettings.value = {
      uploadLimitMode: newVal.uploadLimitMode || 'none',
      uploadLimitValue: newVal.uploadLimitValue || 100,
      gzip: newVal.gzip || false,
      websocket: newVal.websocket || false,
      searchEngineOrigin: newVal.searchEngineOrigin || false,
      urlRedirects: JSON.parse(JSON.stringify(newVal.urlRedirects || [])),
      reqHeaders: JSON.parse(JSON.stringify(newVal.reqHeaders || [])),
      resHeaders: JSON.parse(JSON.stringify(newVal.resHeaders || [])),
      logRequestHeader: newVal.logRequestHeader || false,
      logResponseHeader: newVal.logResponseHeader || false,
      logRequestBody: newVal.logRequestBody || false,
      logRequestBodySizeLimit: newVal.logRequestBodySizeLimit || 16,
      proxyConnectTimeout: newVal.proxyConnectTimeout || '30',
      proxyReadTimeout: newVal.proxyReadTimeout || '60',
      proxySendTimeout: newVal.proxySendTimeout || '60',
      upstreamKeepalive: newVal.upstreamKeepalive || false,
      upstreamKeepaliveConn: newVal.upstreamKeepaliveConn || 100,
      upstreamKeepaliveTimeout: newVal.upstreamKeepaliveTimeout || 60,
      limitRate: newVal.limitRate || 0,
      originCert: newVal.originCert || false,
      realtimeIdentify: newVal.realtimeIdentify || false,
      realtimeSend: newVal.realtimeSend || false
    }
  }
  isInternalUpdate = false
}, { deep: true })

const advancedSettings = localSettings

// 转向规则弹窗状态
const redirectDialogVisible = ref(false)
const editingRedirectRule = ref(null)
const editingRedirectIndex = ref(-1)

const openRedirectDialog = (rule = null, index = -1) => {
  editingRedirectRule.value = rule
  editingRedirectIndex.value = index
  redirectDialogVisible.value = true
}

const handleRedirectSubmit = (ruleData) => {
  const urlRedirects = advancedSettings.value.urlRedirects || []
  
  if (editingRedirectIndex.value >= 0) {
    // 编辑模式
    urlRedirects[editingRedirectIndex.value] = {
      ...urlRedirects[editingRedirectIndex.value],
      ...ruleData
    }
  } else {
    // 新增模式
    urlRedirects.push({
      domain: '', // 默认空，可以后续编辑
      ...ruleData
    })
  }
  
  advancedSettings.value = {
    ...advancedSettings.value,
    urlRedirects
  }
  
  // 重置状态
  editingRedirectRule.value = null
  editingRedirectIndex.value = -1
}

const removeRedirect = (index) => {
  const urlRedirects = advancedSettings.value.urlRedirects || []
  urlRedirects.splice(index, 1)
  advancedSettings.value = {
    ...advancedSettings.value,
    urlRedirects
  }
}

const openHeaderDialog = (type, rule = null, index = -1) => {
  emit('open-header-dialog', type, rule, index)
}

const removeHeader = (type, index) => {
  emit('remove-header', type, index)
}
</script>

<style scoped>
.advanced-config {
  padding: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 20px;
  padding-left: 10px;
  border-left: 3px solid #409eff;
}

.divider {
  height: 1px;
  background-color: #ebeef5;
  margin: 24px 0;
}

.form-helper {
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
  margin-top: 6px;
}
</style>