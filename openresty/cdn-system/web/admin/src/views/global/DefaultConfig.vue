<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>默认配置 (新站点模板)</span>
          <el-button type="primary" @click="saveConfig" :loading="loading">保存所有配置</el-button>
        </div>
      </template>

      <el-tabs v-model="activeTab" v-if="config.default_config">
        <!-- 1. Website Acceleration -->
        <el-tab-pane label="网站加速 (带缓存)" name="website">
            <el-form label-width="150px">
                <el-form-item label="启用缓存">
                    <el-switch v-model="config.default_config.website.cache_enable" />
                </el-form-item>
                <el-form-item label="缓存 TTL (秒)" v-if="config.default_config.website.cache_enable">
                    <el-input-number v-model="config.default_config.website.cache_ttl" />
                </el-form-item>
                <el-form-item label="启用 Gzip">
                    <el-switch v-model="config.default_config.website.gzip" />
                </el-form-item>
                <el-form-item label="启用 WAF">
                    <el-switch v-model="config.default_config.website.waf_enable" />
                </el-form-item>
                <el-alert title="适用于常规网站，静态资源将默认被缓存" type="info" :closable="false" />
            </el-form>
        </el-tab-pane>

        <!-- 2. API Acceleration -->
        <el-tab-pane label="API 加速 (无缓存)" name="api">
             <el-form label-width="150px">
                <el-form-item label="启用缓存">
                    <el-switch v-model="config.default_config.api.cache_enable" disabled active-text="强制关闭" />
                </el-form-item>
                <el-form-item label="启用 Gzip">
                    <el-switch v-model="config.default_config.api.gzip" />
                </el-form-item>
                 <el-form-item label="启用 WAF">
                    <el-switch v-model="config.default_config.api.waf_enable" />
                </el-form-item>
                <el-alert title="适用于 API 接口，所有请求直接回源，不缓存" type="warning" :closable="false" />
            </el-form>
        </el-tab-pane>

        <!-- 3. Download Acceleration -->
        <el-tab-pane label="大文件下载 (无缓存)" name="download">
             <el-form label-width="150px">
                <el-form-item label="启用缓存">
                    <el-switch v-model="config.default_config.download.cache_enable" />
                </el-form-item>
                 <el-form-item label="启用 Gzip">
                    <el-switch v-model="config.default_config.download.gzip" />
                </el-form-item>
                 <el-form-item label="启用 WAF">
                    <el-switch v-model="config.default_config.download.waf_enable" />
                </el-form-item>
                <el-alert title="适用于大文件分发，通常建议直接回源或使用切片缓存（需高级配置）" type="info" :closable="false" />
            </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const activeTab = ref('website')
const loading = ref(false)
const config = ref({})

const loadConfig = () => {
    loading.value = true
    request.get('/global_config').then(res => {
        if (res.code === 0) {
            config.value = res.data
             // Ensure structure exists if backend only returned partial
            if (!config.value.default_config) {
                 config.value.default_config = {
                     website: { cache_enable: true, cache_ttl: 3600, gzip: true, waf_enable: true },
                     api: { cache_enable: false, gzip: true, waf_enable: true },
                     download: { cache_enable: false, gzip: false, waf_enable: true }
                 }
            }
        }
    }).finally(() => {
        loading.value = false
    })
}

const saveConfig = () => {
    loading.value = true
    request.post('/global_config', config.value).then(res => {
        if (res.code === 0) {
            ElMessage.success('配置已保存')
        } else {
             ElMessage.error(res.msg || '保存失败')
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
</style>
