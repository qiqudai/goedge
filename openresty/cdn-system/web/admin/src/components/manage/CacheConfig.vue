<template>
  <div class="cache-config">
    <div class="toolbar-row" style="margin-bottom: 12px;">
      <el-button type="primary" size="small" @click="openCacheRuleDialog()">新增规则</el-button>
      <el-select
        v-model="cacheQuickPreset"
        placeholder="快速添加缓存"
        size="small"
        style="width: 150px; margin-left: 12px;"
        @change="applyCachePreset"
      >
        <el-option label="首页缓存" value="index" />
        <el-option label="全站缓存" value="all" />
        <el-option label="静态资源缓存" value="static" />
        <el-option label="视频资源" value="video" />
        <el-option label="Wordpress 缓存" value="wordpress" />
      </el-select>
    </div>
    
    <el-table :data="cacheSettings.rules" border size="small">
      <el-table-column label="类型" min-width="120">
        <template #default="{ row }">{{ cacheTypeLabelMap[row.type] || row.type }}</template>
      </el-table-column>
      <el-table-column label="内容" min-width="240" prop="value" />
      <el-table-column label="TTL(秒)" width="120" prop="ttl" />
      <el-table-column label="操作" width="140">
        <template #default="{ row, $index }">
          <el-button link type="primary" size="small" @click="openCacheRuleDialog(row, $index)">编辑</el-button>
          <el-button link type="danger" size="small" @click="removeCacheRule($index)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { cacheTypeLabelMap } from '@/constants/origin'
import { getCachePreset, normalizeCacheRule } from '@/utils/siteHelpers'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  }
})

const emit = defineEmits(['update:modelValue', 'open-rule-dialog'])

const cacheSettings = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const cacheQuickPreset = ref('')

const openCacheRuleDialog = (rule = null) => {
  emit('open-rule-dialog', rule)
}

const applyCachePreset = (val) => {
  if (!val) return
  
  const preset = getCachePreset(val)
  if (preset) {
    const rule = normalizeCacheRule(preset)
    if (rule) {
      cacheSettings.value.rules.push(rule)
    }
  }
  cacheQuickPreset.value = ''
}

const removeCacheRule = (index) => {
  cacheSettings.value.rules.splice(index, 1)
}
</script>

<style scoped>
.cache-config {
  padding: 16px;
}

.toolbar-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
</style>