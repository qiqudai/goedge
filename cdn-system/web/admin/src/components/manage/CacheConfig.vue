<template>
  <div class="cache-config">
    <div class="toolbar-row" style="margin-bottom: 12px;">
      <el-button type="primary" size="small" @click="openCacheRuleDialog()">新增规则</el-button>
      <el-button
        type="danger"
        size="small"
        :disabled="!selectedRules.length"
        @click="removeCacheRulesBatch"
      >删除</el-button>
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
        <el-option label="WordPress 缓存" value="wordpress" />
      </el-select>
    </div>
    
  <el-table ref="tableRef" :data="cacheSettings.rules" border size="small" @selection-change="handleSelectionChange">
      <el-table-column type="selection" width="50" />
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
import { ref, watch, computed, onMounted, nextTick, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { cacheTypeLabelMap } from '@/constants/origin'
import { getCachePreset, normalizeCacheRule, dedupeCacheRules } from '@/utils/siteHelpers'

import { useSiteSettings } from '@/composables/useSiteSettings'

const { saveSettings, siteId } = useSiteSettings()

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  }
})

const emit = defineEmits(['update:modelValue', 'open-rule-dialog'])

const localSettings = ref({
  rules: JSON.parse(JSON.stringify(props.modelValue?.rules || []))
})

let isInternalUpdate = false

const updateSettings = () => {
    isInternalUpdate = true
    emit('update:modelValue', {
        ...props.modelValue,
        ...localSettings.value
    })
}

const handleSave = () => {
    updateSettings()
    saveSettings(true)
}

watch(localSettings, (newVal) => {
  isInternalUpdate = true
  emit('update:modelValue', {
    ...props.modelValue,
    ...newVal
  })
}, { deep: true })

watch(() => props.modelValue, (newVal) => {
  if (newVal && !isInternalUpdate) {
    localSettings.value = {
      rules: JSON.parse(JSON.stringify(newVal.rules || []))
    }
  }
  isInternalUpdate = false
}, { deep: true })

const cacheSettings = localSettings
const cacheQuickPreset = ref('')
const selectedRules = ref([])
const tableRef = ref(null)

const selectionKey = computed(() => `table-selection:cache-rules:${siteId.value || 'site'}`)

const getRuleKey = (rule) => {
  if (!rule) return ''
  try {
    return JSON.stringify(rule)
  } catch {
    return ''
  }
}

const saveSelection = (rows) => {
  const key = selectionKey.value
  if (!key) return
  try {
    const ids = (rows || []).map(getRuleKey).filter(Boolean)
    sessionStorage.setItem(key, JSON.stringify(ids))
  } catch (e) {
    console.error('Failed to save cache selection', e)
  }
}

const restoreSelection = () => {
  const key = selectionKey.value
  if (!key) return
  let saved = []
  try {
    saved = JSON.parse(sessionStorage.getItem(key) || '[]')
  } catch {
    saved = []
  }
  if (!saved.length) return
  const table = tableRef.value
  if (!table) return
  table.clearSelection()
  cacheSettings.value.rules.forEach((row) => {
    if (saved.includes(getRuleKey(row))) {
      table.toggleRowSelection(row, true)
    }
  })
}

const openCacheRuleDialog = (rule = null) => {
  emit('open-rule-dialog', rule)
  // Editing rule is handled by parent Dialog SAVE. 
  // Parent should call saveSettings() when dialog saves.
  // Wait, `Manage.vue` handles `saveCacheRule`. 
  // We need to check `Manage.vue`.
}

const applyCachePreset = (val) => {
  if (!val) return
  
  const preset = getCachePreset(val)
  if (preset) {
    const rule = normalizeCacheRule(preset)
    if (rule) {
      const nextRules = [...cacheSettings.value.rules, rule]
      const { rules: mergedRules, removed } = dedupeCacheRules(nextRules)
      cacheSettings.value.rules = mergedRules
      if (removed > 0) {
        ElMessage.warning('检测到重复缓存规则，已自动合并')
      }
      handleSave()
    }
  }
  cacheQuickPreset.value = ''
}

const removeCacheRule = (index) => {
  cacheSettings.value.rules.splice(index, 1)
  handleSave()
}

const handleSelectionChange = (rows) => {
  selectedRules.value = rows
  saveSelection(rows)
}

const removeCacheRulesBatch = async () => {
  if (!selectedRules.value.length) return
  await ElMessageBox.confirm('确定删除选中的缓存规则吗？', '提示', { type: 'warning' })
  const toRemove = new Set(selectedRules.value)
  cacheSettings.value.rules = cacheSettings.value.rules.filter(rule => !toRemove.has(rule))
  selectedRules.value = []
  handleSave()
}

watch(
  () => cacheSettings.value.rules,
  () => {
    nextTick(() => {
      restoreSelection()
    })
  },
  { deep: true }
)

onMounted(() => {
  nextTick(() => {
    restoreSelection()
  })
})

onBeforeUnmount(() => {
  const key = selectionKey.value
  if (key) {
    sessionStorage.removeItem(key)
  }
})
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
