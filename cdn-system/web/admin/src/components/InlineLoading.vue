<template>
  <span class="inline-loading" :class="sizeClass" role="status" aria-live="polite">
    <span class="inline-loading__spinner" aria-hidden="true"></span>
    <span v-if="text" class="inline-loading__text">{{ text }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  text: {
    type: String,
    default: '数据加载中...'
  },
  size: {
    type: String,
    default: 'sm' // 'xs' | 'sm' | 'md'
  }
})

const sizeClass = computed(() => `is-${props.size}`)
</script>

<style scoped>
.inline-loading {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--el-color-primary, #409eff);
  font-size: 12px;
  line-height: 1;
  vertical-align: middle;
}

.inline-loading__spinner {
  display: inline-block;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  box-sizing: border-box;
  animation: inline-loading-spin 0.8s linear infinite;
}

.is-xs .inline-loading__spinner {
  width: 10px;
  height: 10px;
  border-width: 1.5px;
}
.is-xs { font-size: 11px; }

.is-sm .inline-loading__spinner {
  width: 14px;
  height: 14px;
}
.is-sm { font-size: 12px; }

.is-md .inline-loading__spinner {
  width: 18px;
  height: 18px;
  border-width: 2.5px;
}
.is-md { font-size: 13px; }

.inline-loading__text {
  color: var(--el-text-color-secondary, #909399);
  white-space: nowrap;
}

@keyframes inline-loading-spin {
  to { transform: rotate(360deg); }
}

:global(:root[data-theme="dark"] .inline-loading) {
  color: var(--el-color-primary, #8ab5ff);
}
:global(:root[data-theme="dark"] .inline-loading__text) {
  color: var(--el-text-color-secondary, #a8b1bd);
}
</style>
