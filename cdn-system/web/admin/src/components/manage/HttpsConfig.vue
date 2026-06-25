<template>
  <div class="https-config" @focusin="cacheInputValue">
      <div class="section-title">HTTPS证书</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="开关">
        <el-switch v-model="httpsSettings.enable" @change="handleEnableChange" />
        <div v-if="httpsStatusText" class="form-helper">
          当前状态：{{ httpsStatusText }}<span v-if="httpsSettings.error">，{{ httpsSettings.error }}</span>
        </div>
      </el-form-item>
      
      <el-form-item label="证书选择" style="width: 520px">
        <div class="cert-row">
          <el-select v-model="httpsSettings.certId" placeholder="请选择证书" @change="handleSave">
            <el-option 
              v-for="cert in certList" 
              :key="cert.id" 
              :label="cert.name" 
              :value="cert.id"
            >
              <span style="float: left">{{ cert.name }}</span>
              <span style="float: right; color: #8492a6; font-size: 13px">{{ cert.domains }}</span>
            </el-option>
          </el-select>
          <el-button size="small" @click="applyCert">申请证书</el-button>
        </div>
        <div class="form-helper" v-if="httpsSettings.certId">
          <span class="status-dot active"></span> 
          有效期剩余 {{ calcCertDays({ id: httpsSettings.certId }, certList) }} 天
        </div>
        <div class="form-helper" v-else>请选择或上传证书</div>
      </el-form-item>

      <template v-if="httpsSettings.enable && httpsSettings.certId">
        <el-form-item label="监听端口" style="width: 520px">
          <el-input v-model="httpsSettings.listenPorts" placeholder="443" @blur="handleBlurSave" />
          <div class="form-helper">
            多个端口空格分隔。如果需要https://www.example.com和https://www.example.com:8433访问，则填443 8433
          </div>
        </el-form-item>
        
        <div class="divider"></div>
        
        <div class="section-title">强制HTTPS</div>
        <el-form-item label="开关">
          <el-switch v-model="httpsSettings.force" @change="handleSave" />
          <div class="form-helper">开启后，访问http将会301跳转到https</div>
        </el-form-item>
        
        <el-form-item label="跳转端口" v-if="httpsSettings.force" style="width: 320px">
          <el-select v-model="httpsSettings.forcePort" placeholder="443" @change="handleSave">
            <el-option
              v-for="option in listenPortOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          <div class="form-helper">如果https监听有多个端口，可以择其一个跳转</div>
        </el-form-item>

        <div class="divider"></div>

        <div class="section-title">HSTS</div>
        <el-form-item label="开关">
          <el-switch v-model="httpsSettings.hsts" @change="handleSave" />
          <div class="form-helper">
            开启后，访问使用浏览器访问http时，将不用请求服务器直接转向https，这可以减少http会话劫持风险
          </div>
        </el-form-item>

        <div class="divider"></div>

        <div class="section-title">HTTP2设置</div>
        <el-form-item label="开关">
          <el-switch v-model="httpsSettings.http2" @change="handleSave" />
          <div class="form-helper">
            HTTP2.0协议是HTTP1.1协议的升级版本，在Web数据交互性能上具备更多的优势，开启前您需要先配置HTTPS证书。
          </div>
        </el-form-item>

        <div class="divider"></div>

        <div class="section-title">OCSP 装订</div>
        <el-form-item label="开关">
          <el-switch v-model="httpsSettings.ocsp" @change="handleSave" />
          <div class="form-helper">
            OCSP 装订功能可实现由CDN预先缓存在线证书验证结果并下发给客户端，无需浏览器直接向CA站点查询证书状态，从而减少用户验证时间。
          </div>
        </el-form-item>

        <div class="divider"></div>

        <div class="section-title">HTTP3设置</div>
        <el-form-item label="开关">
          <el-switch v-model="httpsSettings.http3" @change="handleSave" />
        </el-form-item>

        <div class="divider"></div>

        <div class="section-title">SSL配置</div>
        <el-form-item label="SSL配置">
          <el-radio-group v-model="httpsSettings.sslPolicy" @change="handleSave">
            <el-radio value="compat">兼容旧浏览器（安全性降低）</el-radio>
            <el-radio value="modern">兼容大部分浏览器（更安全）</el-radio>
            <el-radio value="custom">自定义</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <template v-if="httpsSettings.sslPolicy === 'custom'">
          <el-form-item label="加密算法">
            <el-input 
              v-model="httpsSettings.sslCiphers" 
              type="textarea" 
              :rows="3" 
              placeholder="EECDH+AESGCM:EDH+AESGCM:AES256+EECDH:AES256+EDH"
              @blur="handleBlurSave"
            />
            <div class="form-helper">OpenSSL支持的加密算法，多个算法之间使用冒号(:)分隔</div>
          </el-form-item>
          <el-form-item label="SSL协议">
            <el-input 
              v-model="httpsSettings.sslProtocols" 
              type="textarea" 
              :rows="2" 
              placeholder="TLSv1 TLSv1.1 TLSv1.2 TLSv1.3" 
              @blur="handleBlurSave"
            />
            <div class="form-helper">空格分隔，如 TLSv1.2 TLSv1.3</div>
          </el-form-item>
        </template>
      </template>
    </el-form>
  </div>
</template>



<script setup>
import { ref, watch, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { getCertDays } from '@/utils/helpers'
import { cacheInputValue, shouldSkipBlurSave } from '@/utils/saveGuard'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  },
  certList: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue', 'calc-cert-days'])

const { saveSettings, siteId, loadSite, loadCerts } = useSiteSettings()

const localSettings = ref({
  enable: props.modelValue?.enable || false,
  state: props.modelValue?.state || 'off',
  error: props.modelValue?.error || '',
  activeCertId: props.modelValue?.activeCertId || null,
  pendingCertId: props.modelValue?.pendingCertId || null,
  certId: props.modelValue?.certId || null,
  listenPorts: props.modelValue?.listenPorts || '443',
  force: props.modelValue?.force || false,
  forcePort: props.modelValue?.forcePort || '443',
  hsts: props.modelValue?.hsts || false,
  http2: props.modelValue?.http2 || false,
  http3: props.modelValue?.http3 || false,
  ocsp: props.modelValue?.ocsp || false,
  sslPolicy: props.modelValue?.sslPolicy || 'compat',
  sslCiphers: props.modelValue?.sslCiphers || '',
  sslProtocols: props.modelValue?.sslProtocols || ''
})

const listenPortOptions = computed(() => {
  return (localSettings.value.listenPorts || '443').split(' ').filter(Boolean).map(port => ({
    label: port,
    value: port
  }))
})

const httpsStatusText = computed(() => {
  const state = String(localSettings.value.state || '').toLowerCase()
  if (state === 'active') return '已启用'
  if (state === 'pending_issue') return '证书申请中'
  if (state === 'probing') return '节点证书验证中'
  if (state === 'failed') return '启用失败'
  return ''
})

const handleEnableChange = (newVal) => {
  if (newVal && !localSettings.value.certId) {
    ElMessage.warning('请先选择证书')
    localSettings.value.enable = false
    return
  }
  if (newVal && !localSettings.value.listenPorts) {
    localSettings.value.listenPorts = '443'
  }
  handleSave()
}

let isInternalUpdate = false

const applyCert = async () => {
  if (!siteId.value) {
    return
  }
  await ElMessageBox.confirm('确定申请证书吗？', '提示')
  const res = await request.post('/sites/apply_cert', { ids: [siteId.value] })
  const payload = res?.data || res || {}
  const created = Array.isArray(payload.created_ids) ? payload.created_ids : []
  const reissued = Array.isArray(payload.reissued_ids) ? payload.reissued_ids : []
  const skipped = Array.isArray(payload.skipped) ? payload.skipped : []
  const summary = res?.message || payload.message
  const issuedCertID = created[0] || reissued[0]
  if (issuedCertID) {
    ElMessage.success(
      reissued.length > 0 && created.length === 0
        ? '失败证书已重新提交签发，签发并通过节点探测后才会启用 HTTPS'
        : '证书申请已提交，签发并通过节点探测后才会启用 HTTPS'
    )
    localSettings.value.enable = false
    localSettings.value.state = 'pending_issue'
    localSettings.value.pendingCertId = issuedCertID
    localSettings.value.certId = issuedCertID
    if (!localSettings.value.listenPorts) {
      localSettings.value.listenPorts = '443'
    }
    await Promise.allSettled([loadCerts(), loadSite()])
  } else if (skipped.length > 0) {
    const msg = skipped.map(item => item.reason || '已忽略').join('\n')
    ElMessage.warning(summary || msg)
  }
}

const handleSave = () => {
  emit('update:modelValue', {
    ...props.modelValue,
    ...localSettings.value
  })
  saveSettings(true)
}

const handleBlurSave = (event) => {
  if (shouldSkipBlurSave(event, { skipEmpty: true })) {
    return
  }
  handleSave()
}

// 监听本地变化并同步
watch(localSettings, (newVal) => {
  isInternalUpdate = true
  emit('update:modelValue', {
    ...props.modelValue,
    ...newVal
  })
}, { deep: true })

// 监听外部变化并更新本地
watch(() => props.modelValue, (newVal) => {
  if (newVal && !isInternalUpdate) {
    localSettings.value = {
      enable: newVal.enable || false,
      state: newVal.state || 'off',
      error: newVal.error || '',
      activeCertId: newVal.activeCertId || null,
      pendingCertId: newVal.pendingCertId || null,
      certId: newVal.certId || null,
      listenPorts: newVal.listenPorts || '443',
      force: newVal.force || false,
      forcePort: newVal.forcePort || '443',
      hsts: newVal.hsts || false,
      http2: newVal.http2 || false,
      http3: newVal.http3 || false,
      ocsp: newVal.ocsp || false,
      sslPolicy: newVal.sslPolicy || 'compat',
      sslCiphers: newVal.sslCiphers || '',
      sslProtocols: newVal.sslProtocols || ''
    }
  }
  isInternalUpdate = false
}, { deep: true })

// 监听监听端口变化，确保跳转端口有效
watch(() => localSettings.value.listenPorts, (newPorts) => {
  const ports = (newPorts || '').split(' ').filter(Boolean)
  if (ports.length > 0 && !ports.includes(localSettings.value.forcePort)) {
    localSettings.value.forcePort = ports[0]
  }
})

const httpsSettings = localSettings

const calcCertDays = (cert, certs) => {
    return getCertDays(cert, certs)
}
</script>

<style scoped>
.https-config {
  padding: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 20px;
  padding-left: 10px;
  border-left: 3px solid #409eff;
}

.divider {
  height: 1px;
  background-color: var(--el-border-color-lighter);
  margin: 24px 0;
}

.form-helper {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  margin-top: 6px;
}

.cert-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.cert-row .el-select {
  width: 220px;
  flex: 0 0 220px;
}

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #67c23a;
  margin-right: 6px;
}
</style>
