<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>用户默认配置</span>
          <el-button type="primary" @click="saveConfig" :loading="loading">保存配置</el-button>
        </div>
      </template>

      <el-tabs v-model="activeTab" v-if="config.default_config">
        <el-tab-pane label="网站默认配置" name="website">
          <el-form label-width="150px">
            <el-form-item label="缓存开关">
              <el-switch v-model="config.default_config.website.cache_enable" />
            </el-form-item>
            <el-form-item label="缓存 TTL (秒)" v-if="config.default_config.website.cache_enable">
              <el-input-number v-model="config.default_config.website.cache_ttl" />
            </el-form-item>
            <el-form-item label="开启 Gzip">
              <el-switch v-model="config.default_config.website.gzip" />
            </el-form-item>
            <el-form-item label="开启 WAF">
              <el-switch v-model="config.default_config.website.waf_enable" />
            </el-form-item>
            <el-alert title="该配置用于新用户的网站默认配置" type="info" :closable="false" />
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="API 默认配置" name="api">
          <el-form label-width="150px">
            <el-form-item label="缓存开关">
              <el-switch v-model="config.default_config.api.cache_enable" disabled active-text="不可修改" />
            </el-form-item>
            <el-form-item label="开启 Gzip">
              <el-switch v-model="config.default_config.api.gzip" />
            </el-form-item>
            <el-form-item label="开启 WAF">
              <el-switch v-model="config.default_config.api.waf_enable" />
            </el-form-item>
            <el-alert title="API 暂不支持缓存开关，只展示默认值" type="warning" :closable="false" />
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="下载默认配置" name="download">
          <el-form label-width="150px">
            <el-form-item label="缓存开关">
              <el-switch v-model="config.default_config.download.cache_enable" />
            </el-form-item>
            <el-form-item label="开启 Gzip">
              <el-switch v-model="config.default_config.download.gzip" />
            </el-form-item>
            <el-form-item label="开启 WAF">
              <el-switch v-model="config.default_config.download.waf_enable" />
            </el-form-item>
            <el-alert title="该配置用于新用户的下载默认配置" type="info" :closable="false" />
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
      config.value = res.data || {}
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
      ElMessage.success('\u4fdd\u5b58\u6210\u529f')
    } else {
      ElMessage.error(res.msg || '\u4fdd\u5b58\u5931\u8d25')
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
