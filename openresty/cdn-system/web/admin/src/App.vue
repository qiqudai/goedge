<template>
  <router-view />
  <GlobalLoading />
</template>

<script setup>
import GlobalLoading from '@/components/GlobalLoading.vue'
</script>

<style>
body {
  margin: 0;
  padding: 0;
}
</style>
