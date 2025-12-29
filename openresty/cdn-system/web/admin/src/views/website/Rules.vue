<template>
  <div class="app-container">
    <el-card shadow="never">
      <el-tabs v-model="activeTab" type="card" class="rules-tabs">
        <el-tab-pane label="CC规则" name="cc">
          <CCRules />
        </el-tab-pane>

        <el-tab-pane label="ACL规则" name="acl">
          <AclList />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import CCRules from './rules/CCRules.vue'
import AclList from './rules/AclList.vue'

const activeTab = ref('cc')
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
