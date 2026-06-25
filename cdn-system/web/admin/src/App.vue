<template>
  <el-config-provider :locale="zhCn">
    <router-view />
    <GlobalLoading />
  </el-config-provider>
</template>

<script setup>
import { onMounted } from 'vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import GlobalLoading from '@/components/GlobalLoading.vue'
import { useSystemInfo } from '@/composables/useSystemInfo'

const { loadSystemInfo } = useSystemInfo()

onMounted(() => {
  loadSystemInfo()
})
</script>

<style>
body {
  margin: 0;
  padding: 0;
}
</style>
