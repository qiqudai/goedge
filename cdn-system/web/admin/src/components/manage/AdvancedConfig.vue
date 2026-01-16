<template>
  <div class="advanced-config" @focusin="cacheInputValue">
    <el-form label-width="150px" class="config-form">
      <div class="section-title">上传大小限制</div>
      <el-form-item label="大小限制">
        <div style="display: flex; align-items: center; gap: 10px;">
          <el-radio-group v-model="advancedSettings.uploadLimitMode" @change="handleSave">
            <el-radio value="none">不限制</el-radio>
            <el-radio value="custom">自定义</el-radio>
          </el-radio-group>
          <el-input 
            v-if="advancedSettings.uploadLimitMode === 'custom'" 
            v-model="advancedSettings.uploadLimitValue" 
            style="width: 150px" 
            placeholder="100"
           @blur="handleBlurSave">
            <template #append>MB</template>
          </el-input>
        </div>
      </el-form-item>

      <div class="divider"></div>
      <div class="section-title">压缩设置</div>
      <el-form-item label="Gzip压缩">
        <el-switch v-model="advancedSettings.gzip" @change="handleSave" />
      </el-form-item>

      <div class="divider"></div>

      <div class="section-title">WebSocket 设置</div>
      <el-form-item label="WebSocket">
        <el-switch v-model="advancedSettings.websocket" @change="handleSave" />
      </el-form-item>

      <div class="divider"></div>

      <div class="section-title">搜索引擎回源配置</div>
      <el-form-item label="开关">
        <el-switch v-model="advancedSettings.searchEngineOrigin" @change="handleSave" />
      </el-form-item>
      <el-form-item label="回源IP" v-if="advancedSettings.searchEngineOrigin">
        <el-input v-model="advancedSettings.searchEngineOriginIp" placeholder="请输入源IP" style="width: 200px;"  @blur="handleBlurSave" />
        <div class="form-helper" style="color: #F56C6C;">谨慎使用，有泄露源IP的风险!</div>
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
        <el-switch v-model="advancedSettings.logRequestHeader" @change="handleSave" />
        <div class="form-helper">开启只会增加硬盘空间占用，可长期开启</div>
      </el-form-item>
      <el-form-item label="记录响应头">
        <el-switch v-model="advancedSettings.logResponseHeader" @change="handleSave" />
        <div class="form-helper">建议只在调试时开启，始终开启会增加cpu, 硬盘的占用</div>
      </el-form-item>
      <el-form-item label="记录请求体">
        <el-switch v-model="advancedSettings.logRequestBody" @change="handleSave" />
        <div class="form-helper">建议只在调试时开启，始终开启对节点性能消耗较大</div>
      </el-form-item>
      <el-form-item label="请求体大小限制">
        <el-input v-model="advancedSettings.logRequestBodySizeLimit" placeholder="16" style="width: 200px;" @blur="handleBlurSave">
          <template #append>KB</template>
        </el-input>
      </el-form-item>

      <div class="divider"></div>

      <div class="divider"></div>

      <div class="section-title">其它</div>
      <el-form-item label="源站证书">
        <div>
          <el-switch v-model="advancedSettings.originCert" @change="handleSave" />
          <div class="form-helper">用于回源连接（HTTPS）验证源站证书</div>
        </div>
      </el-form-item>
      <el-form-item label="数据实时鉴别">
        <div>
          <el-switch v-model="advancedSettings.realtimeIdentify" @change="handleSave" />
          <div class="form-helper">开启后，节点一收到源返回的数据，立即发送到用户。</div>
        </div>
      </el-form-item>
      <el-form-item label="数据实时发送">
        <div>
          <el-switch v-model="advancedSettings.realtimeSend" @change="handleSave" />
          <div class="form-helper">开启后，节点一收到用户发来的数据就会立即发送给源服务器。</div>
        </div>
      </el-form-item>
      <el-form-item label="默认站点">
        <div>
          <el-switch v-model="advancedSettings.defaultSite" @change="handleSave" />
          <div class="form-helper">开启后，不属于cdn上的域名将会使用这个站点；另外如果要使用IP证书，也请开启这个选项</div>
        </div>
      </el-form-item>
      <el-form-item label="L2配置">
        <el-radio-group v-model="advancedSettings.l2Config" @change="handleSave">
          <el-radio value="current">当前套餐配置</el-radio>
          <el-radio value="none">不配置L2</el-radio>
          <el-radio value="custom">自定义L2配置</el-radio>
        </el-radio-group>
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
import { useSiteSettings } from '@/composables/useSiteSettings'
import { cacheInputValue, shouldSkipBlurSave } from '@/utils/saveGuard'

const { saveSettings } = useSiteSettings()

const handleSave = () => {
  saveSettings(true)
}

const handleBlurSave = (event) => {
  if (shouldSkipBlurSave(event, { skipEmpty: true })) {
    return
  }
  handleSave()
}

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
  searchEngineOriginIp: props.modelValue?.searchEngineOriginIp || '',
  urlRedirects: JSON.parse(JSON.stringify(props.modelValue?.urlRedirects || [])),
  reqHeaders: JSON.parse(JSON.stringify(props.modelValue?.reqHeaders || [])),
  resHeaders: JSON.parse(JSON.stringify(props.modelValue?.resHeaders || [])),
  logRequestHeader: props.modelValue?.logRequestHeader || false,
  logResponseHeader: props.modelValue?.logResponseHeader || false,
  logRequestBody: props.modelValue?.logRequestBody || false,
  logRequestBodySizeLimit: props.modelValue?.logRequestBodySizeLimit || 16,
  originCert: props.modelValue?.originCert || false,
  realtimeIdentify: props.modelValue?.realtimeIdentify || false,
  realtimeSend: props.modelValue?.realtimeSend || false,
  defaultSite: props.modelValue?.defaultSite || false,
  l2Config: props.modelValue?.l2Config || 'current'
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
      searchEngineOriginIp: newVal.searchEngineOriginIp || '',
      urlRedirects: JSON.parse(JSON.stringify(newVal.urlRedirects || [])),
      reqHeaders: JSON.parse(JSON.stringify(newVal.reqHeaders || [])),
      resHeaders: JSON.parse(JSON.stringify(newVal.resHeaders || [])),
      logRequestHeader: newVal.logRequestHeader || false,
      logResponseHeader: newVal.logResponseHeader || false,
      logRequestBody: newVal.logRequestBody || false,
      logRequestBodySizeLimit: newVal.logRequestBodySizeLimit || 16,
      originCert: newVal.originCert || false,
      realtimeIdentify: newVal.realtimeIdentify || false,
      realtimeSend: newVal.realtimeSend || false,
      defaultSite: newVal.defaultSite || false,
      l2Config: newVal.l2Config || 'current'
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

  handleSave()
}

const removeRedirect = (index) => {
  const urlRedirects = advancedSettings.value.urlRedirects || []
  urlRedirects.splice(index, 1)
  advancedSettings.value = {
    ...advancedSettings.value,
    urlRedirects
  }
  handleSave()
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
  width: 100%;
  display: block;
}
</style>
