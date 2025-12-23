<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>???????????</span>
          <el-button type="primary" @click="saveConfig" :loading="loading">??????</el-button>
        </div>
      </template>

      <el-tabs v-model="activeTab" v-if="config.default_config">
        <el-tab-pane label="?????????" name="website">
          <el-form label-width="150px">
            <el-form-item label="????">
              <el-switch v-model="config.default_config.website.cache_enable" />
            </el-form-item>
            <el-form-item label="?? TTL (?)" v-if="config.default_config.website.cache_enable">
              <el-input-number v-model="config.default_config.website.cache_ttl" />
            </el-form-item>
            <el-form-item label="?? Gzip">
              <el-switch v-model="config.default_config.website.gzip" />
            </el-form-item>
            <el-form-item label="?? WAF">
              <el-switch v-model="config.default_config.website.waf_enable" />
            </el-form-item>
            <el-alert title="??????????????????" type="info" :closable="false" />
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="API ???????" name="api">
          <el-form label-width="150px">
            <el-form-item label="????">
              <el-switch v-model="config.default_config.api.cache_enable" disabled active-text="????" />
            </el-form-item>
            <el-form-item label="?? Gzip">
              <el-switch v-model="config.default_config.api.gzip" />
            </el-form-item>
            <el-form-item label="?? WAF">
              <el-switch v-model="config.default_config.api.waf_enable" />
            </el-form-item>
            <el-alert title="??? API ???????????????" type="warning" :closable="false" />
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="??????????" name="download">
          <el-form label-width="150px">
            <el-form-item label="????">
              <el-switch v-model="config.default_config.download.cache_enable" />
            </el-form-item>
            <el-form-item label="?? Gzip">
              <el-switch v-model="config.default_config.download.gzip" />
            </el-form-item>
            <el-form-item label="?? WAF">
              <el-switch v-model="config.default_config.download.waf_enable" />
            </el-form-item>
            <el-alert title="???????????????????????????????" type="info" :closable="false" />
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
      ElMessage.success('?????')
    } else {
      ElMessage.error(res.msg || '????')
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
