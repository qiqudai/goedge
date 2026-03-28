<template>
  <div class="app-container">
    <el-tabs type="border-card">
      <el-tab-pane label="核心设置">
        <el-form label-width="180px">
          <el-form-item label="工作进程数">
             <el-input v-model="form.worker_processes" placeholder="自动 (auto)" />
          </el-form-item>
          <el-form-item label="工作进程连接数">
             <el-input-number v-model="form.worker_connections" :step="1024" />
          </el-form-item>
          <el-form-item label="关闭超时（秒）">
             <el-input-number v-model="form.worker_shutdown_timeout" />
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="WAF 与防火墙">
        <el-form label-width="180px">
            <el-divider content-position="left">拦截策略</el-divider>
            <el-form-item label="拦截模式">
                <el-radio-group v-model="form.waf_mode">
                  <el-radio value="ipset">IPSet（零 CPU 开销）</el-radio>
                  <el-radio value="page">返回 403 页面</el-radio>
                  <el-radio value="drop">断开连接</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item label="黑名单时长（秒）">
                <el-input-number v-model="form.blacklist_timeout" />
            </el-form-item>
            
            <el-divider content-position="left">CC 防护</el-divider>
            <el-form-item label="CC 阈值（请求/秒）">
                <el-input-number v-model="form.cc_threshold" />
            </el-form-item>
            <el-form-item label="动作">
                <el-select v-model="form.cc_action">
                    <el-option label="显示验证码" value="captcha" />
                    <el-option label="5 秒盾" value="shield_5s" />
                    <el-option label="封禁 IP" value="block" />
                </el-select>
            </el-form-item>
            
             <el-divider content-position="left">资源防护</el-divider>
             <el-form-item label=".well-known 限制">
                 <el-input placeholder="600 请求 / 60 秒" disabled />
             </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="HTTPS 与协议">
          <el-form label-width="180px">
            <el-form-item label="启用 HTTP/2">
                <el-switch v-model="form.http2" />
            </el-form-item>
            <el-form-item label="启用 HTTP/3（QUIC）">
                <el-switch v-model="form.http3" />
            </el-form-item>
            <el-form-item label="强制 HSTS">
                <el-switch v-model="form.hsts" />
            </el-form-item>
            <el-form-item label="SSL 加密套件">
                <el-input type="textarea" :rows="4" v-model="form.ssl_ciphers" />
            </el-form-item>
            <el-form-item label="可缓存状态码">
                <el-input v-model="form.cache_valid_statuses" placeholder="200 302" />
                <div style="font-size: 12px; color: #999;">空格或逗号分隔。仅这些回源状态码允许写入缓存，其它状态码强制不缓存。</div>
            </el-form-item>
            <el-form-item label="404缓存探测开关">
                <el-switch v-model="form.cache_404_revalidate_enable" />
            </el-form-item>
            <el-form-item label="404探测触发时长">
                <el-input-number v-model="form.cache_404_revalidate_after" :min="1" /> <span style="margin-left:10px;">秒</span>
            </el-form-item>
            <el-form-item label="同键探测间隔">
                <el-input-number v-model="form.cache_404_probe_interval" :min="1" /> <span style="margin-left:10px;">秒</span>
            </el-form-item>
            <el-form-item label="探测超时">
                <el-input-number v-model="form.cache_404_probe_timeout_ms" :min="200" :step="100" /> <span style="margin-left:10px;">毫秒</span>
            </el-form-item>
          </el-form>
      </el-tab-pane>
    </el-tabs>
    
    <div style="margin-top: 20px;">
        <el-button type="primary" @click="saveConfig">保存全局配置</el-button>
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
    ssl_ciphers: '',
    cache_valid_statuses: '200 302',
    cache_404_revalidate_enable: true,
    cache_404_revalidate_after: 5,
    cache_404_probe_interval: 3,
    cache_404_probe_timeout_ms: 1200
})

const nginxRawConfig = ref({})

const getConfig = () => {
    Promise.all([
      request.get('/global_config'),
      request.get('/config_items', { params: { type: 'nginx_config', scope_name: 'global', scope_id: 0 } })
    ]).then(([globalRes, nginxRes]) => {
      if (globalRes.data) Object.assign(form, globalRes.data)
      const list = nginxRes?.list || nginxRes?.data?.list || []
      const item = list.find(entry => entry.name === 'nginx-config-file')
      if (item?.value) {
        try {
          nginxRawConfig.value = JSON.parse(item.value) || {}
          const http = nginxRawConfig.value.http || {}
          form.cache_valid_statuses = (http.proxy_cache_valid_statuses || http.cache_valid_statuses || form.cache_valid_statuses || '200 302')
          form.cache_404_revalidate_enable = typeof http.cache_404_revalidate_enable === 'boolean' ? http.cache_404_revalidate_enable : form.cache_404_revalidate_enable
          form.cache_404_revalidate_after = Number(http.cache_404_revalidate_after || form.cache_404_revalidate_after || 5)
          form.cache_404_probe_interval = Number(http.cache_404_probe_interval || form.cache_404_probe_interval || 3)
          form.cache_404_probe_timeout_ms = Number(http.cache_404_probe_timeout_ms || form.cache_404_probe_timeout_ms || 1200)
        } catch (e) {
          nginxRawConfig.value = {}
        }
      }
    })
}

const saveConfig = () => {
    const updatedNginx = {
      ...nginxRawConfig.value,
      http: {
        ...(nginxRawConfig.value.http || {}),
        proxy_cache_valid_statuses: String(form.cache_valid_statuses || '200 302').trim(),
        cache_404_revalidate_enable: !!form.cache_404_revalidate_enable,
        cache_404_revalidate_after: Number(form.cache_404_revalidate_after || 5),
        cache_404_probe_interval: Number(form.cache_404_probe_interval || 3),
        cache_404_probe_timeout_ms: Number(form.cache_404_probe_timeout_ms || 1200)
      }
    }
    const nginxPayload = {
      type: 'nginx_config',
      scope_name: 'global',
      scope_id: 0,
      items: [
        {
          name: 'nginx-config-file',
          value: JSON.stringify(updatedNginx),
          enable: true
        }
      ]
    }

    Promise.all([
      request.post('/global_config', form),
      request.post('/config_items', nginxPayload)
    ]).then(() => {
        ElMessage.success('已保存')
    })
}

onMounted(() => getConfig())
</script>
