<template>
  <el-page-header @back="$emit('back')" class="page-header" content="网站配置" style="margin-bottom: 16px;">
    <template #title>
      <span>{{ site.domains?.[0] || site.domain_raw || '网站' }}</span>
    </template>
    <template #content>
      <span>ID {{ site.id }} · {{ siteSettings.basic.cname }}</span>
    </template>
  </el-page-header>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  site: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['back'])

// 从props.site中提取设置
const siteSettings = computed(() => {
  return {
    basic: {
      cname: props.site.cname || '-'
    }
  }
})
</script>

<style scoped>
.page-header {
  padding: 16px;
  background: var(--el-bg-color);
  border-radius: 4px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}
</style>
