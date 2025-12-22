<template>
  <div class="app-container">
    <el-card shadow="never">
      <div class="key-row">
        <div class="label">密钥状态</div>
        <el-switch v-model="keyEnabled" />
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
        <el-input v-model="whitelist" type="textarea" rows="3" placeholder="多个IP以分隔" />
      </div>

      <div class="key-actions">
        <el-button type="primary" @click="resetSecret">重置密钥</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DocumentCopy } from '@element-plus/icons-vue'

const keyEnabled = ref(true)
const apiKey = ref('WUBHA9ISzOuNRZ8d')
const apiSecret = ref('JU3ETXfvD19kNjAC0HgOW2oRlpbQy')
const whitelist = ref('')

const copyText = value => {
  navigator.clipboard?.writeText(value).then(() => {
    ElMessage.success('已复制')
  })
}

const resetSecret = () => {
  ElMessageBox.confirm('确认重置密钥?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    apiSecret.value = Math.random().toString(36).slice(2, 18)
    ElMessage.success('密钥已重置')
  })
}
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
}
</style>
