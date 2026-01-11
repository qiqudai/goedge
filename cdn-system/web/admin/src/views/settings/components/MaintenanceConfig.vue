<template>
  <el-form label-width="120px" @focusin="cacheInputValue">
    <el-card shadow="never" class="mb-20">
      <template #header>维护升级</template>
      <el-form-item label="系统维护模式">
        <el-switch v-model="maintain.enable" :active-value="1" :inactive-value="0" @change="handleSave" />
        <div class="form-helper">开启后，用户访问将显示维护页面</div>
      </el-form-item>
      <el-form-item label="维护公告">
        <el-input v-model="maintain.msg" placeholder="系统维护中，请稍后重试..." @blur="handleBlurSave" />
      </el-form-item>
      
      <el-divider />
      
      <el-form-item label="Agent自动升级">
        <el-switch v-model="autoUpgradeAgent" active-value="1" inactive-value="0" @change="handleSave" />
        <div class="form-helper">开启后，节点Agent将尝试自动升级到最新版本</div>
      </el-form-item>

    </el-card>
  </el-form>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import { cacheInputValue, shouldSkipBlurSave } from '@/utils/saveGuard'

const props = defineProps({
  configItems: {
    type: Array,
    default: () => []
  }
})

const maintain = ref({
  enable: 0,
  msg: ''
})
const autoUpgradeAgent = ref('1')

watch(() => props.configItems, (items) => {
  if (!items) return

  // maintain
  const mItem = items.find(i => i.name === 'maintain')
  if (mItem && mItem.value) {
    try {
      const parsed = JSON.parse(mItem.value)
      maintain.value = { ...maintain.value, ...parsed }
    } catch (e) {
      console.error('Failed to parse maintain config', e)
    }
  }

  // auto_upgrade_agent
  const agentItem = items.find(i => i.name === 'auto_upgrade_agent')
  if (agentItem) {
    autoUpgradeAgent.value = agentItem.value
  }
}, { immediate: true, deep: true })

const save = () => {
  const items = []
  
  // maintain
  items.push({
    name: 'maintain',
    value: JSON.stringify(maintain.value)
  })

  // auto_upgrade_agent
  items.push({
    name: 'auto_upgrade_agent',
    value: autoUpgradeAgent.value
  })

  return request.post('/config_items', {
    type: 'system',
    scope_name: 'global',
    items: items
  }).then(() => {
    ElMessage.success('保存成功')
  })
}

const saving = ref(false)
let saveQueued = false

const queueSave = async () => {
  if (saving.value) {
    saveQueued = true
    return
  }
  saving.value = true
  await nextTick()
  save().finally(() => {
    saving.value = false
    if (saveQueued) {
      saveQueued = false
      queueSave()
    }
  })
}

const handleSave = () => {
  queueSave()
}

const handleBlurSave = (event) => {
  if (shouldSkipBlurSave(event)) {
    return
  }
  queueSave()
}
</script>

<style scoped>
.mb-20 { margin-bottom: 20px; }
.form-helper { color: #999; font-size: 12px; margin-left: 10px; display: inline-block; }
</style>
