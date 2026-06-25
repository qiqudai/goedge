<template>
  <div class="origin-config" @focusin="cacheInputValue">
    <!-- 回源协议与端口 -->
    <div class="section-title">回源协议与端口</div>
    <el-form label-width="150px" class="config-form">
      <el-form-item label="回源协议">
        <div>
          <el-radio-group v-model="originSettings.protocol" @change="handleSave">
            <el-radio value="http">HTTP</el-radio>
            <el-radio value="https">HTTPS</el-radio>
            <el-radio value="follow">跟随协议</el-radio>
            <el-radio value="follow_port">跟随端口和协议</el-radio>
          </el-radio-group>
          <div class="form-helper">
            <div>1. 当选择HTTP，即节点与源的连接使用HTTP协议；</div>
            <div>2. 当选择HTTPS时，节点使用HTTPS连接；</div>
            <div>3. 当选择跟随协议时，当用户使用HTTP访问你在cdn上的网站时，节点也使用HTTP连接源，用户使用HTTPS访问时，节点也使用HTTPS连接源；</div>
            <div>4. 当选择跟随端口和协议时，即用户访问的协议和端口，节点也使用同样的协议和端口与源连接，一般用于当监听多个端口时，也希望以同样的访问端口回源</div>
          </div>
        </div>
      </el-form-item>
      
      <el-form-item 
        label="HTTP回源端口" 
        v-if="['http', 'follow'].includes(originSettings.protocol)" 
        style="width: 520px"
      >
        <el-input v-model="originSettings.httpPort"  @blur="handleBlurSave" />
        <div class="form-helper">当节点与源使用HTTP连接时所使用的端口</div>
      </el-form-item>
      
      <el-form-item 
        label="HTTPS回源端口" 
        v-if="['https', 'follow'].includes(originSettings.protocol)" 
        style="width: 520px"
      >
        <el-input v-model="originSettings.httpsPort"  @blur="handleBlurSave" />
        <div class="form-helper">当节点与源使用HTTPS连接时所使用的端口</div>
      </el-form-item>

      <el-form-item label="回源HOST" style="width: 520px">
        <el-radio-group v-model="originSettings.host" @change="handleSave">
          <el-radio value="follow">自动跟随</el-radio>
          <el-radio value="domain">网站域名</el-radio>
          <el-radio value="custom">自定义</el-radio>
        </el-radio-group>
        <el-input 
          v-if="originSettings.host === 'custom'" 
          v-model="originSettings.hostValue" 
          placeholder="请输入自定义回源HOST"
          style="margin-top: 10px"
         @blur="handleBlurSave" />
        <div class="form-helper">节点回源时发送的 Host 头部。自动跟随：跟随用户请求的 Host；网站域名：使用当前网站配置的第一个域名。</div>
      </el-form-item>

      <el-form-item label="回源SNI" style="width: 520px">
        <el-input
          v-model="originSettings.sni"
          placeholder="留空则跟随回源HOST"
          @blur="handleBlurSave"
        />
        <div class="form-helper">HTTPS 回源握手使用的 SNI。源站地址为 IP、负载均衡域名与证书域名不一致时，请填写源站证书域名。</div>
      </el-form-item>

      <el-form-item label="校验源站证书">
        <el-switch v-model="originSettings.verifyTLS" @change="handleSave" />
        <div class="form-helper">开启后节点会校验源站证书链和域名，证书不匹配时回源失败；关闭可兼容自签名或内网源。</div>
      </el-form-item>

      <el-form-item label="回源HTTP版本" style="width: 520px">
        <el-radio-group v-model="originSettings.httpVersionPolicy" @change="handleSave">
          <el-radio value="auto">自动加速</el-radio>
          <el-radio value="http11">HTTP/1.1 keepalive</el-radio>
          <el-radio value="compat">HTTP/1.0兼容</el-radio>
        </el-radio-group>
        <div class="form-helper">自动加速默认使用 HTTP/1.1 长连接；源站连续异常时临时降级，冷却后自动恢复。</div>
      </el-form-item>

      <template v-if="originSettings.httpVersionPolicy === 'auto'">
        <el-form-item label="自动降级">
          <el-switch v-model="originSettings.autoDowngrade" @change="handleSave" />
        </el-form-item>
        <el-form-item label="降级阈值" style="width: 320px">
          <el-input v-model="originSettings.downgradeThreshold" @blur="handleBlurSave">
            <template #append>次</template>
          </el-input>
        </el-form-item>
        <el-form-item label="统计窗口" style="width: 320px">
          <el-input v-model="originSettings.downgradeWindowSeconds" @blur="handleBlurSave">
            <template #append>秒</template>
          </el-input>
        </el-form-item>
        <el-form-item label="冷却时间" style="width: 320px">
          <el-input v-model="originSettings.downgradeCooldownSeconds" @blur="handleBlurSave">
            <template #append>秒</template>
          </el-input>
        </el-form-item>
      </template>
    </el-form>

    <div class="divider"></div>

    <!-- 回源超时 -->
    <div class="section-title">回源超时</div>
    <el-form label-width="150px" class="config-form">
      <el-form-item label="回源超时" style="width: 320px">
        <el-input v-model="originSettings.timeout" @blur="handleBlurSave">
          <template #append>秒</template>
        </el-input>
      </el-form-item>
      <el-form-item label="连接超时" style="width: 320px">
        <el-input style="width: 320px" v-model="originSettings.connTimeout" @blur="handleBlurSave">
          <template #append>秒</template>
        </el-input>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { cacheInputValue, shouldSkipBlurSave } from '@/utils/saveGuard'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  }
})

const emit = defineEmits(['update:modelValue'])

const localSettings = ref({
  protocol: props.modelValue?.protocol || 'follow',
  host: props.modelValue?.host || 'follow',
  hostValue: props.modelValue?.hostValue || '',
  httpPort: props.modelValue?.httpPort || '80',
  httpsPort: props.modelValue?.httpsPort || '443',
  sni: props.modelValue?.sni || '',
  verifyTLS: props.modelValue?.verifyTLS || false,
  httpVersionPolicy: props.modelValue?.httpVersionPolicy || 'auto',
  autoDowngrade: props.modelValue?.autoDowngrade !== false,
  downgradeThreshold: props.modelValue?.downgradeThreshold || 3,
  downgradeWindowSeconds: props.modelValue?.downgradeWindowSeconds || 60,
  downgradeCooldownSeconds: props.modelValue?.downgradeCooldownSeconds || 600,
  keepaliveConn: props.modelValue?.keepaliveConn || 64,
  keepaliveTimeout: props.modelValue?.keepaliveTimeout || 60,
  timeout: props.modelValue?.timeout || 60,
  connTimeout: props.modelValue?.connTimeout || 10
})

import { validateDomain } from '@/utils/siteHelpers'
import { useSiteSettings } from '@/composables/useSiteSettings'

const { saveSettings } = useSiteSettings()

let isInternalUpdate = false

const handleSave = () => {
   // Wait for sync to happen (watch triggers sync)
   // But watch is sync in Vue 3 for reactive objects?
   // Let's use nextTick or just assume sync.
   // Also handle custom validation check again just in case watch blocked it?
   // If watch blocked it, parent state is stale (old valid one).
   // Calling saveSettings saves the STALE state.
   // This is CORRECT: we don't save invalid state.
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
  // 如果是自定义 HOST，必须验证通过才保存
  if (newVal.host === 'custom') {
    if (!newVal.hostValue || !validateDomain(newVal.hostValue)) {
      return
    }
  }
  if (newVal.sni && !validateDomain(newVal.sni)) {
    return
  }

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
      protocol: newVal.protocol || 'follow',
      host: newVal.host || 'follow',
      hostValue: newVal.hostValue || '',
      httpPort: newVal.httpPort || '80',
      httpsPort: newVal.httpsPort || '443',
      sni: newVal.sni || '',
      verifyTLS: newVal.verifyTLS || false,
      httpVersionPolicy: newVal.httpVersionPolicy || 'auto',
      autoDowngrade: newVal.autoDowngrade !== false,
      downgradeThreshold: newVal.downgradeThreshold || 3,
      downgradeWindowSeconds: newVal.downgradeWindowSeconds || 60,
      downgradeCooldownSeconds: newVal.downgradeCooldownSeconds || 600,
      keepaliveConn: newVal.keepaliveConn || 64,
      keepaliveTimeout: newVal.keepaliveTimeout || 60,
      timeout: newVal.timeout || 60,
      connTimeout: newVal.connTimeout || 10
    }
  }
  isInternalUpdate = false
}, { deep: true })

const originSettings = localSettings
</script>

<style scoped>
.origin-config {
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
</style>
