<template>
  <div class="app-container">
    <el-card shadow="never">
      <div class="key-row">
        <div class="label">????</div>
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
        <div class="label">IP???</div>
        <el-input v-model="whitelist" type="textarea" rows="3" placeholder="??IP???????" />
      </div>

      <div class="key-actions">
        <el-button @click="saveWhitelist">?????</el-button>
        <el-button type="primary" @click="resetSecret">????</el-button>
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
    ElMessage.success('???')
  })
}

const saveWhitelist = () => {
  request.put('/api_key', { api_ip: whitelist.value }).then(() => {
    ElMessage.success('???')
  })
}

const resetSecret = () => {
  ElMessageBox.confirm('???????', '??', {
    confirmButtonText: '??',
    cancelButtonText: '??',
    type: 'warning'
  }).then(() => {
    request.post('/api_key/reset').then(res => {
      apiSecret.value = res.data?.api_secret || apiSecret.value
      ElMessage.success('?????')
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
