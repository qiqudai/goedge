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
            <el-input type="textarea" v-model="config.nginx.custom_snippet" rows="5" placeholder="# Custom nginx directives..." />
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
const config = ref({})

const loadConfig = () => {
    loading.value = true
    request.get('/global_config').then(res => {
        if (res.code === 0) {
            config.value = res.data
        }
    }).finally(() => {
        loading.value = false
    })
}

const saveConfig = () => {
    loading.value = true
    request.post('/global_config', config.value).then(res => {
        if (res.code === 0) {
            ElMessage.success('Nginx 配置已保存')
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
