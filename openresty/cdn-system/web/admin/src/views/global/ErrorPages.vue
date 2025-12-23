<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>??????</span>
          <el-button type="primary" @click="saveConfig" :loading="loading">??????</el-button>
        </div>
      </template>

      <div class="error-page-container">
        <el-tabs v-model="activeCode" tab-position="left" class="error-tabs" style="height: calc(100vh - 180px);">
          <el-tab-pane v-for="code in errorCodes" :key="code.key" :label="code.label" :name="code.key">
            <div class="tab-content-scroll">
              <div class="editor-header">
                <h3>{{ code.label }} ({{ code.key }})</h3>
                <span class="tip">????:</span>
              </div>

              <el-input
                v-model="errorPages[code.key]"
                type="textarea"
                :rows="25"
                placeholder="<!-- ?? HTML ?? -->"
                font-family="monospace"
              />

              <div class="preview" v-html="errorPages[code.key]"></div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const activeCode = ref('403')
const errorPages = reactive({})
const fullConfig = ref({})

const errorCodes = [
  { key: '400', label: '400 ????' },
  { key: '403', label: '403 ????' },
  { key: '502', label: '502 ????' },
  { key: '504', label: '504 ????' },
  { key: 'traffic_limit', label: '??????' },
  { key: 'site_locked', label: '????' },
  { key: 'domain_invalid', label: '?????' },
  { key: 'conn_limit', label: '?????' }
]

const loadConfig = () => {
  loading.value = true
  request.get('/global_config').then(res => {
    if (res.code === 0) {
      fullConfig.value = res.data || {}
      if (res.data?.error_pages) {
        Object.assign(errorPages, res.data.error_pages)
      }
      errorCodes.forEach(c => {
        if (!errorPages[c.key]) {
          errorPages[c.key] = `<h1>${c.label}</h1><p>System Default Page</p>`
        }
      })
    }
  }).finally(() => {
    loading.value = false
  })
}

const saveConfig = () => {
  loading.value = true
  fullConfig.value.error_pages = errorPages
  request.post('/global_config', fullConfig.value).then(res => {
    if (res.code === 0) {
      ElMessage.success('?????')
    }
  }).finally(() => {
    loading.value = false
  })
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.error-tabs {
  border: 1px solid #eee;
}
:deep(.el-tabs__content) {
  height: 100%;
  overflow-y: auto;
  padding-right: 10px;
}
.tab-content-scroll {
  padding-bottom: 20px;
}
.editor-header {
  display: flex;
  align-items: center;
  gap: 12px;
}
.preview {
  margin-top: 20px;
  border: 1px dashed #ccc;
  padding: 10px;
  background: #f9f9f9;
  min-height: 100px;
}
.tip {
  font-size: 12px;
  color: #909399;
}
</style>
