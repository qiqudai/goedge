<template>
  <div class="app-container">
    <el-card shadow="never">
      <div class="key-row">
        <div class="label">密钥状态</div>
        <el-switch v-model="keyEnabled" disabled />
      </div>

      <div class="key-row">
        <div class="label">api_key</div>
        <div class="value">
          {{ apiKey }}
          <el-button link type="primary" @click="copyText(apiKey)">
            <el-icon><DocumentCopy /></el-icon>
          </el-button>
        </div>
      </div>

      <div class="key-row">
        <div class="label">api_secret</div>
        <div class="value">
          {{ apiSecret }}
          <el-button link type="primary" @click="copyText(apiSecret)">
            <el-icon><DocumentCopy /></el-icon>
          </el-button>
        </div>
      </div>

      <div class="key-row">
        <div class="label">IP白名单</div>
        <el-input v-model="whitelist" type="textarea" rows="3" placeholder="多个IP换行分隔" />
      </div>

      <div class="key-actions">
        <el-button @click="saveWhitelist">保存白名单</el-button>
        <el-button type="primary" @click="resetSecret">重置密钥</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DocumentCopy } from '@element-plus/icons-vue'
import request from '@/utils/request'

const keyEnabled = ref(false)
const apiKey = ref('')
const apiSecret = ref('')
const whitelist = ref('')

const loadKey = () => {
  request.get('/api_key').then(res => {
    const data = res.data || {}
    apiKey.value = data.api_key || ''
    apiSecret.value = data.api_secret || ''
    whitelist.value = data.api_ip || ''
    keyEnabled.value = Boolean(apiKey.value)
  })
}

const copyText = value => {
  navigator.clipboard?.writeText(value).then(() => {
    ElMessage.success('\u590d\u5236\u6210\u529f')
  })
}

const saveWhitelist = () => {
  request.put('/api_key', { api_ip: whitelist.value }).then(() => {
    ElMessage.success('\u4fdd\u5b58\u6210\u529f')
  })
}

const resetSecret = () => {
  ElMessageBox.confirm('\u786e\u5b9a\u91cd\u7f6e\u5bc6\u94a5\uff1f', '\u63d0\u793a', {
    confirmButtonText: '\u786e\u5b9a',
    cancelButtonText: '\u53d6\u6d88',
    type: 'warning'
  }).then(() => {
    request.post('/api_key/reset').then(res => {
      apiSecret.value = res.data?.api_secret || apiSecret.value
      ElMessage.success('\u91cd\u7f6e\u6210\u529f')
    })
  })
}

onMounted(() => loadKey())
</script>

<style scoped>
.key-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.label {
  width: 90px;
  color: #606266;
}
.value {
  display: flex;
  align-items: center;
  gap: 6px;
}
.key-actions {
  margin-top: 6px;
  display: flex;
  gap: 10px;
}
</style>
