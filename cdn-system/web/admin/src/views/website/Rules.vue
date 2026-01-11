<template>
  <div class="app-container">
    <el-card shadow="never">
      <el-tabs v-model="activeTab" type="card" class="rules-tabs">
        <el-tab-pane label="CC规则" name="cc">
          <el-alert
            v-if="activeTab === 'cc' && permissionLoaded && !canCustomCCRule"
            type="warning"
            show-icon
            title="当前套餐未开启自定义 CC 规则"
            description="请联系管理员开通套餐权限。"
          />
          <CCRules v-else-if="activeTab === 'cc' && canCustomCCRule" />
        </el-tab-pane>

        <el-tab-pane label="ACL规则" name="acl">
          <AclList v-if="activeTab === 'acl'" />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import request from '@/utils/request'
import CCRules from './rules/CCRules.vue'
import AclList from './rules/AclList.vue'

const activeTab = ref('cc')
const canCustomCCRule = ref(false)
const permissionLoaded = ref(false)
const isAdmin = (localStorage.getItem('role') || 'user') === 'admin'

const loadPermission = async () => {
  if (isAdmin) {
    canCustomCCRule.value = true
    permissionLoaded.value = true
    return
  }
  try {
    const res = await request.get('/user_packages')
    const list = res.data?.list || res.list || []
    canCustomCCRule.value = list.some(item => item.custom_cc_rule && item.status !== 'expired')
  } catch {
    canCustomCCRule.value = false
  } finally {
    permissionLoaded.value = true
  }
}

onMounted(() => {
  loadPermission()
})
</script>

<style scoped>
.app-container {
  padding: 20px;
}
:deep(.filter-container) {
  padding-bottom: 20px;
}
:deep(.text-success) {
  color: #67C23A;
}
:deep(.text-danger) {
  color: #F56C6C;
}
</style>
