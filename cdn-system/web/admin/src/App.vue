<template>
  <router-view />
  <GlobalLoading />
</template>

<script setup>
import { onMounted } from 'vue'
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
