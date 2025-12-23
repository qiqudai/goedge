<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>自定义错误页面</span>
          <el-button type="primary" @click="saveConfig" :loading="loading">保存配置</el-button>
        </div>
      </template>

      <div class="error-page-container">
        <el-tabs v-model="activeCode" tab-position="left" class="error-tabs" style="height: calc(100vh - 180px);">
          <el-tab-pane v-for="code in errorCodes" :key="code.key" :label="code.label" :name="code.key">
            <div class="tab-content-scroll">
              <div class="editor-header">
                <h3>{{ code.label }} ({{ code.key }})</h3>
                <span class="tip">编辑内容:</span>
              </div>

              <el-input
                v-model="errorPages[code.key]"
                type="textarea"
                :rows="25"
                placeholder="<!-- 请输入 HTML 内容 -->"
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
  { key: '400', label: '400 \u9519\u8bef\u9875\u9762' },
  { key: '403', label: '403 \u9519\u8bef\u9875\u9762' },
  { key: '502', label: '502 \u9519\u8bef\u9875\u9762' },
  { key: '504', label: '504 \u9519\u8bef\u9875\u9762' },
  { key: 'traffic_limit', label: '\u6d41\u91cf\u8d85\u9650' },
  { key: 'site_locked', label: '\u7f51\u7ad9\u88ab\u9501' },
  { key: 'domain_invalid', label: '\u57df\u540d\u65e0\u6548' },
  { key: 'conn_limit', label: '\u8fde\u63a5\u6570\u8d85\u9650' }
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
      ElMessage.success('\u4fdd\u5b58\u6210\u529f')
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
