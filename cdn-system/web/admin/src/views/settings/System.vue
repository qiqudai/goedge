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

const resolveConfigItemList = (response) => {
  if (Array.isArray(response?.data?.list)) {
    return response.data.list
  }
  if (Array.isArray(response?.list)) {
    return response.list
  }
  return []
}

const normalizeConfigItems = (items) => {
  if (!Array.isArray(items)) {
    return []
  }

  const preferredByName = new Map()
  items.forEach((item) => {
    if (!item?.name) {
      return
    }

    const scopeName = String(item.scope_name || '').trim()
    const scopeID = Number(item.scope_id || 0)
    const score = scopeName === 'global' && scopeID === 0 ? 3 : scopeID === 0 ? 2 : 1
    const current = preferredByName.get(item.name)
    if (!current || score > current.score) {
      preferredByName.set(item.name, { item, score })
    }
  })

  return Array.from(preferredByName.values()).map(entry => entry.item)
}

const loadData = () => {
  loading.value = true
  request
    .get('/config_items', { params: { type: 'system', scope_name: 'global', scope_id: 0 } })
    .then(res => {
      configItems.value = normalizeConfigItems(resolveConfigItemList(res))
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
