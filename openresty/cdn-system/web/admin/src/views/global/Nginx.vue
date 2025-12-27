<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
            <span>Nginx 全局配置</span>
            <el-button type="primary" @click="saveConfig" :loading="loading">保存配置</el-button>
        </div>
      </template>

      <el-form label-width="200px" v-if="config.nginx">
        <h3>Worker 进程配置</h3>
        <el-form-item label="Worker Processes">
             <el-input v-model="config.nginx.worker_processes" placeholder="auto" style="width: 200px;" />
             <div class="tip">工作进程数，建议设置为 CPU 核心数 or "auto"</div>
        </el-form-item>
        <el-form-item label="Worker Connections">
             <el-input-number v-model="config.nginx.worker_connections" :min="1024" :step="1024" />
             <div class="tip">每个工作进程的最大连接数</div>
        </el-form-item>
        <el-form-item label="Worker Rlimit Nofile">
             <el-input-number v-model="config.nginx.worker_rlimit_nofile" :min="1024" :step="1024" />
             <div class="tip">最大打开文件描述符数 (ulimit -n)</div>
        </el-form-item>
        <el-form-item label="Worker Shutdown Timeout">
             <el-input v-model="config.nginx.worker_shutdown_timeout" placeholder="60s" style="width: 200px;" />
             <div class="tip">优雅退出超时时间</div>
        </el-form-item>

        <el-divider />

        <h3>路径配置</h3>
        <el-form-item label="日志目录 (Access/Error)">
             <el-input v-model="config.nginx.log_directory" placeholder="/usr/local/nginx/logs/" />
             <div class="tip">Nginx 访问日志和错误日志的存放目录</div>
        </el-form-item>

        <el-divider />

        <h3>其他设置</h3>
         <el-form-item label="Keepalive Timeout">
             <el-input-number v-model="config.nginx.keepalive_timeout" /> <span class="unit">秒</span>
        </el-form-item>
        <el-form-item label="开启 Gzip">
            <el-switch v-model="config.nginx.gzip" />
        </el-form-item>
        <el-form-item label="自定义配置片段 (http block)">
            <el-input type="textarea" v-model="config.nginx.custom_snippet" :rows="5" placeholder="# Custom nginx directives..." />
        </el-form-item>

      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const config = ref(defaultForm())
const rawNginxConfig = ref({})

function defaultForm() {
  return {
    nginx: {
      worker_processes: 'auto',
      worker_connections: 51200,
      worker_rlimit_nofile: 51200,
      worker_shutdown_timeout: '60s',
      log_directory: '',
      keepalive_timeout: 60,
      gzip: true,
      custom_snippet: ''
    }
  }
}

function parseKeepaliveTimeout(value, fallback = 60) {
  if (value === undefined || value === null || value === '') {
    return fallback
  }
  if (typeof value === 'number') {
    return value
  }
  const raw = String(value).trim()
  if (raw === '') {
    return fallback
  }
  const match = raw.match(/^(\d+)/)
  if (!match) {
    return fallback
  }
  return Number.parseInt(match[1], 10)
}

function formatKeepaliveTimeout(value) {
  if (value === undefined || value === null || value === '') {
    return ''
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return `${value}s`
  }
  const raw = String(value).trim()
  if (raw === '') {
    return ''
  }
  if (/^\d+$/.test(raw)) {
    return `${raw}s`
  }
  return raw
}

function parseBool(value, fallback = false) {
  if (typeof value === 'boolean') {
    return value
  }
  if (value === undefined || value === null) {
    return fallback
  }
  const raw = String(value).toLowerCase()
  if (raw === 'on' || raw === 'true' || raw === '1') {
    return true
  }
  if (raw === 'off' || raw === 'false' || raw === '0') {
    return false
  }
  return fallback
}

function mergeFormFromRaw(raw) {
  const http = raw.http || {}
  config.value = {
    nginx: {
      worker_processes: raw.worker_processes || 'auto',
      worker_connections: raw.worker_connections || 0,
      worker_rlimit_nofile: raw.worker_rlimit_nofile || 0,
      worker_shutdown_timeout: raw.worker_shutdown_timeout || '',
      log_directory: raw.logs_dir || '',
      keepalive_timeout: parseKeepaliveTimeout(http.keepalive_timeout, 60),
      gzip: parseBool(http.gzip, true),
      custom_snippet: http.custom_snippet || ''
    }
  }
}

const loadConfig = () => {
  loading.value = true
  request
    .get('/config_items', {
      params: { type: 'nginx_config', scope_name: 'global', scope_id: 0 }
    })
    .then(res => {
      const list = res.list || res.data?.list || []
      const item = list.find(entry => entry.name === 'nginx-config-file')
      if (item && item.value) {
        try {
          const parsed = JSON.parse(item.value)
          rawNginxConfig.value = parsed || {}
          mergeFormFromRaw(rawNginxConfig.value)
          return
        } catch (e) {
          rawNginxConfig.value = {}
        }
      }
      rawNginxConfig.value = {}
      config.value = defaultForm()
    })
    .finally(() => {
      loading.value = false
    })
}

const saveConfig = () => {
  loading.value = true
  const updated = {
    ...rawNginxConfig.value
  }
  updated.http = {
    ...(rawNginxConfig.value.http || {})
  }
  updated.worker_processes = config.value.nginx.worker_processes
  updated.worker_connections = config.value.nginx.worker_connections
  updated.worker_rlimit_nofile = config.value.nginx.worker_rlimit_nofile
  updated.worker_shutdown_timeout = config.value.nginx.worker_shutdown_timeout
  updated.logs_dir = config.value.nginx.log_directory
  updated.http.keepalive_timeout = formatKeepaliveTimeout(config.value.nginx.keepalive_timeout)
  updated.http.gzip = config.value.nginx.gzip ? 'on' : 'off'
  updated.http.custom_snippet = config.value.nginx.custom_snippet || ''

  const payload = {
    type: 'nginx_config',
    scope_name: 'global',
    scope_id: 0,
    items: [
      {
        name: 'nginx-config-file',
        value: JSON.stringify(updated),
        enable: true
      }
    ]
  }

  request
    .post('/config_items', payload)
    .then(() => {
      ElMessage.success('Nginx config saved')
    })
    .finally(() => {
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
.tip {
    font-size: 12px;
    color: #999;
}
.unit {
    margin-left: 10px;
}
h3 {
    margin-top: 0;
    margin-bottom: 20px;
    padding-left: 10px;
    border-left: 4px solid #67C23A; 
}
</style>
