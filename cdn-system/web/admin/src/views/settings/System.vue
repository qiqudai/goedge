<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" v-loading="loading" type="border-card">
      <el-tab-pane label="系统配置" name="system">
        <BasicConfig :configItems="configItems" @saved="loadData" />
        <el-divider />
        <PackageConfig :configItems="configItems" @saved="loadData" />
        <el-divider />
        <MaintenanceConfig :configItems="configItems" @saved="loadData" />
      </el-tab-pane>

      <el-tab-pane label="数据清理" name="cleaning">
        <CleaningConfig :configItems="configItems" @saved="loadData" />
      </el-tab-pane>

      <el-tab-pane label="用户相关" name="user">
        <UserConfig :configItems="configItems" @saved="loadData" />
      </el-tab-pane>

      <el-tab-pane label="通知配置" name="notify">
        <NotifyConfig :configItems="configItems" @saved="loadData" />
      </el-tab-pane>



      <el-tab-pane label="其它配置" name="other">
        <OtherConfig :configItems="configItems" @saved="loadData" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import request from '@/utils/request'
import { useSystemInfo } from '@/composables/useSystemInfo'
import BasicConfig from './components/BasicConfig.vue'
import PackageConfig from './components/PackageConfig.vue'
import MaintenanceConfig from './components/MaintenanceConfig.vue'
import CleaningConfig from './components/CleaningConfig.vue'
import UserConfig from './components/UserConfig.vue'
import NotifyConfig from './components/NotifyConfig.vue'
import OtherConfig from './components/OtherConfig.vue'

const activeTab = ref('system')
const configItems = ref([])
const loading = ref(false)
const { loadSystemInfo } = useSystemInfo()

const loadData = () => {
  loading.value = true
  request
    .get('/config_items', { params: { type: 'system' } })
    .then(res => {
      configItems.value = res.list || []
    })
    .then(() => {
      if (activeTab.value === 'system') {
        loadSystemInfo(true)
      }
    })
    .finally(() => {
      loading.value = false
    })
}

onMounted(() => {
  loadData()
})

watch(activeTab, () => {
  loadData()
})
</script>

<style scoped>
</style>
