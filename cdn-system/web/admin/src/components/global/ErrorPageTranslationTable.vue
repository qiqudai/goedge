<template>
  <div class="translation-table">
    <el-table :data="rows" border size="small" style="width: 100%">
      <el-table-column prop="key" label="变量" width="180" fixed />
      <el-table-column
        v-for="lang in enabledLangs"
        :key="lang"
        :label="lang"
        min-width="220"
      >
        <template #default="{ row }">
          <el-input
            :model-value="strings[lang]?.[row.key] || ''"
            type="textarea"
            :rows="2"
            @update:model-value="value => updateCell(lang, row.key, value)"
          />
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { extractTemplateKeys } from '@/services/errorPageService'

const props = defineProps({
  template: {
    type: String,
    default: ''
  },
  strings: {
    type: Object,
    default: () => ({})
  },
  enabledLangs: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:strings'])

const rows = computed(() => extractTemplateKeys(props.template).map(key => ({ key })))

const updateCell = (lang, key, value) => {
  const next = { ...props.strings }
  next[lang] = { ...(next[lang] || {}), [key]: value }
  emit('update:strings', next)
}
</script>
