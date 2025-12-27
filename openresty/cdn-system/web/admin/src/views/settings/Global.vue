<template>
  <div class="app-container">
    <el-tabs type="border-card">
      <el-tab-pane label="Core Settings">
        <el-form label-width="180px">
          <el-form-item label="Worker Processes">
             <el-input v-model="form.worker_processes" placeholder="auto" />
          </el-form-item>
          <el-form-item label="Worker Connections">
             <el-input-number v-model="form.worker_connections" :step="1024" />
          </el-form-item>
          <el-form-item label="Shutdown Timeout (s)">
             <el-input-number v-model="form.worker_shutdown_timeout" />
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="WAF & Firewall">
        <el-form label-width="180px">
            <el-divider content-position="left">Block Strategy</el-divider>
            <el-form-item label="Block Mode">
                <el-radio-group v-model="form.waf_mode">
                  <el-radio value="ipset">IPSet (Zero CPU Cost)</el-radio>
                  <el-radio value="page">Return 403 Page</el-radio>
                  <el-radio value="drop">Disconnect</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item label="Blacklist Duration (s)">
                <el-input-number v-model="form.blacklist_timeout" />
            </el-form-item>
            
            <el-divider content-position="left">CC Protection</el-divider>
            <el-form-item label="CC Threshold (req/s)">
                <el-input-number v-model="form.cc_threshold" />
            </el-form-item>
            <el-form-item label="Action">
                <el-select v-model="form.cc_action">
                    <el-option label="Show Captcha" value="captcha" />
                    <el-option label="5s Shield" value="shield_5s" />
                    <el-option label="Block IP" value="block" />
                </el-select>
            </el-form-item>
            
             <el-divider content-position="left">Resources Protection</el-divider>
             <el-form-item label=".well-known Limit">
                 <el-input placeholder="600 req / 60s" disabled />
             </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="HTTPS & Protocol">
          <el-form label-width="180px">
            <el-form-item label="Enable HTTP/2">
                <el-switch v-model="form.http2" />
            </el-form-item>
            <el-form-item label="Enable HTTP/3 (QUIC)">
                <el-switch v-model="form.http3" />
            </el-form-item>
            <el-form-item label="Force HSTS">
                <el-switch v-model="form.hsts" />
            </el-form-item>
            <el-form-item label="SSL Ciphers">
                <el-input type="textarea" :rows="4" v-model="form.ssl_ciphers" />
            </el-form-item>
          </el-form>
      </el-tab-pane>
    </el-tabs>
    
    <div style="margin-top: 20px;">
        <el-button type="primary" @click="saveConfig">Save Global Config</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const form = reactive({
    worker_processes: 'auto',
    worker_connections: 51200,
    worker_shutdown_timeout: 60,
    waf_mode: 'ipset',
    blacklist_timeout: 3600,
    cc_threshold: 100,
    cc_action: 'captcha',
    http2: true,
    http3: true,
    hsts: false,
    ssl_ciphers: ''
})

const getConfig = () => {
    request.get('/global_config').then(res => {
         if(res.data) Object.assign(form, res.data)
    })
}

const saveConfig = () => {
    request.post('/global_config', form).then(() => {
        ElMessage.success('Saved')
    })
}

onMounted(() => getConfig())
</script>
