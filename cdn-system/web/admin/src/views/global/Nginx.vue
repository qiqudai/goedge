<template>
  <div class="app-container" @focusin="cacheInputValue">
    <el-card>
      <template #header>
        <div class="card-header">
            <span>Nginx 全局配置</span>
        </div>
      </template>

      <el-form label-width="200px" v-if="config.nginx">
        <h3>工作进程配置</h3>
        <el-form-item label="工作进程数">
             <el-input v-model="config.nginx.worker_processes" placeholder="自动 (auto)" style="width: 200px;" @blur="saveConfig" />
             <div class="tip">工作进程数，建议设置为 CPU 核心数或 "auto"</div>
        </el-form-item>
        <el-form-item label="工作进程连接数">
             <el-input-number v-model="config.nginx.worker_connections" :min="1024" :step="1024" @blur="saveConfig" />
             <div class="tip">每个工作进程的最大连接数</div>
        </el-form-item>
        <el-form-item label="打开文件数限制">
             <el-input-number v-model="config.nginx.worker_rlimit_nofile" :min="1024" :step="1024" @blur="saveConfig" />
             <div class="tip">最大打开文件描述符数 (ulimit -n)</div>
        </el-form-item>
        <el-form-item label="优雅退出超时">
             <el-input v-model="config.nginx.worker_shutdown_timeout" placeholder="60s" style="width: 200px;" @blur="saveConfig" />
             <div class="tip">优雅退出超时时间</div>
        </el-form-item>

        <el-divider />

        <h3>路径配置</h3>
        <el-form-item label="日志目录（访问/错误）">
             <el-input v-model="config.nginx.log_directory" placeholder="/usr/local/nginx/logs/" @blur="saveConfig" />
             <div class="tip">Nginx 访问日志和错误日志的存放目录</div>
        </el-form-item>

        <el-divider />

        <h3>其他设置</h3>
         <el-form-item label="Keepalive 超时">
             <el-input-number v-model="config.nginx.keepalive_timeout" @blur="saveConfig" /> <span class="unit">秒</span>
        </el-form-item>
        <el-form-item label="开启 Gzip">
            <el-switch v-model="config.nginx.gzip" @change="saveConfig" />
        </el-form-item>
        <el-form-item label="可缓存状态码">
             <el-input v-model="config.nginx.cache_valid_statuses" placeholder="200 302" style="width: 280px;" @blur="saveConfig" />
             <div class="tip">仅这些回源状态码允许写入缓存，其他状态码将强制不缓存（空格或逗号分隔）</div>
        </el-form-item>
        <el-form-item label="404缓存探测开关">
            <el-switch v-model="config.nginx.cache_404_revalidate_enable" @change="saveConfig" />
        </el-form-item>
        <el-form-item label="404探测触发时长">
             <el-input-number v-model="config.nginx.cache_404_revalidate_after" :min="1" @blur="saveConfig" /> <span class="unit">秒</span>
        </el-form-item>
        <el-form-item label="同键探测间隔">
             <el-input-number v-model="config.nginx.cache_404_probe_interval" :min="1" @blur="saveConfig" /> <span class="unit">秒</span>
        </el-form-item>
        <el-form-item label="探测超时">
             <el-input-number v-model="config.nginx.cache_404_probe_timeout_ms" :min="200" :step="100" @blur="saveConfig" /> <span class="unit">毫秒</span>
        </el-form-item>
        <el-form-item label="自定义配置片段（HTTP 块）">
            <el-input type="textarea" v-model="config.nginx.custom_snippet" :rows="5" placeholder="# 自定义 Nginx 指令..." @blur="saveConfig" />
        </el-form-item>

      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
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
      cache_valid_statuses: '200 302',
      cache_404_revalidate_enable: true,
      cache_404_revalidate_after: 5,
      cache_404_probe_interval: 3,
      cache_404_probe_timeout_ms: 1200,
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
      cache_valid_statuses: (http.proxy_cache_valid_statuses || http.cache_valid_statuses || '200 302'),
      cache_404_revalidate_enable: parseBool(http.cache_404_revalidate_enable, true),
      cache_404_revalidate_after: Number(http.cache_404_revalidate_after || 5),
      cache_404_probe_interval: Number(http.cache_404_probe_interval || 3),
      cache_404_probe_timeout_ms: Number(http.cache_404_probe_timeout_ms || 1200),
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

const saving = ref(false)
let saveQueued = false

const cacheInputValue = (event) => {
  const el = event?.target
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
    return
  }
  el.dataset.lastValue = el.value ?? ''
}

const shouldSkipBlurSave = (event) => {
  const el = event?.target
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
    return false
  }
  const value = el.value ?? ''
  const lastValue = el.dataset.lastValue ?? ''
  if (value === '' || value === lastValue) {
    return true
  }
  el.dataset.lastValue = value
  return false
}

const saveConfig = async (event) => {
  if (shouldSkipBlurSave(event)) {
    return
  }
  if (saving.value) {
    saveQueued = true
    return
  }
  saving.value = true
  await nextTick()
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
  updated.http.proxy_cache_valid_statuses = String(config.value.nginx.cache_valid_statuses || '200 302').trim()
  updated.http.cache_404_revalidate_enable = !!config.value.nginx.cache_404_revalidate_enable
  updated.http.cache_404_revalidate_after = Number(config.value.nginx.cache_404_revalidate_after || 5)
  updated.http.cache_404_probe_interval = Number(config.value.nginx.cache_404_probe_interval || 3)
  updated.http.cache_404_probe_timeout_ms = Number(config.value.nginx.cache_404_probe_timeout_ms || 1200)
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
      saving.value = false
      if (saveQueued) {
        saveQueued = false
        saveConfig()
      }
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
