<template>
  <el-form label-width="150px" @focusin="cacheInputValue">
    <el-card shadow="never" class="mb-20">
      <template #header>套餐控制</template>
      <el-form-item label="套餐到期关闭网站">
        <el-switch v-model="form.package_expire_close_site" active-value="1" inactive-value="0" @change="handleSave" />
        <div class="form-helper">开启后，过期套餐的网站将无法访问</div>
      </el-form-item>
      <el-form-item label="流量超限关闭网站">
        <el-switch v-model="form.traffic_excceed_close_site" active-value="1" inactive-value="0" @change="handleSave" />
        <div class="form-helper">开启后，超出套餐流量限制的网站将无法访问</div>
      </el-form-item>
      <el-form-item label="允许自主升级">
        <el-switch v-model="form.package_allow_upgrade" active-value="1" inactive-value="0" @change="handleSave" />
      </el-form-item>
      <el-form-item label="允许自主降级">
        <el-switch v-model="form.package_allow_downgrade" active-value="1" inactive-value="0" @change="handleSave" />
      </el-form-item>
      
    </el-card>
  </el-form>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import { cacheInputValue } from '@/utils/saveGuard'

const props = defineProps({
  configItems: {
    type: Array,
    default: () => []
  }
})

const form = ref({
  package_expire_close_site: '0',
  traffic_excceed_close_site: '0',
  package_allow_upgrade: '1',
  package_allow_downgrade: '0'
})

watch(() => props.configItems, (items) => {
  if (!items) return
  const keys = Object.keys(form.value)
  keys.forEach(key => {
    const found = items.find(i => i.name === key)
    if (found) {
      form.value[key] = found.value
    }
  })
}, { immediate: true, deep: true })

const save = () => {
  const items = []
  Object.keys(form.value).forEach(key => {
    items.push({
      name: key,
      value: form.value[key]
    })
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
</script>

<style scoped>
.mb-20 { margin-bottom: 20px; }
.form-helper { color: #999; font-size: 12px; margin-left: 10px; display: inline-block; }
</style>
