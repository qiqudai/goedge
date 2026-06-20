<template>
  <div class="template-editor">
    <div class="editor-toolbar">
      <span class="tip">使用 {{key}} 占位符，运行时变量：{client_ip}、{node_ip}</span>
    </div>
    <div class="key-list" v-if="templateKeys.length">
      <span class="key-label">可用变量：</span>
      <el-tag v-for="key in templateKeys" :key="key" size="small" class="key-tag">{{ '{{' + key + '}}' }}</el-tag>
    </div>
    <el-input
      :model-value="modelValue"
      type="textarea"
      :rows="22"
      placeholder="请输入 HTML 模板"
      font-family="monospace"
      @update:model-value="$emit('update:modelValue', $event)"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { extractTemplateKeys } from '@/services/errorPageService'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  }
})

defineEmits(['update:modelValue'])

const templateKeys = computed(() => extractTemplateKeys(props.modelValue))
</script>

<style scoped>
.editor-toolbar {
  margin-bottom: 8px;
}
.tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.key-list {
  margin-bottom: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.key-label {
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.key-tag {
  font-family: monospace;
}
</style>
