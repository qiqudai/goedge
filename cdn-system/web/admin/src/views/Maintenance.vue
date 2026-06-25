<template>
  <div class="maintenance">
    <div class="card">
      <div class="title">系统维护中</div>
      <div class="message">{{ message }}</div>
      <el-button type="primary" @click="reload">刷新</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const message = ref('系统维护中，请稍后再试。')

onMounted(() => {
  const stored = localStorage.getItem('maintenance_msg')
  if (stored) {
    message.value = stored
  }
})

const reload = () => {
  window.location.reload()
}
</script>

<style scoped>
.maintenance {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-bg-color-page);
}
.card {
  background: var(--el-bg-color);
  padding: 32px 40px;
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
  text-align: center;
  max-width: 520px;
}
.title {
  font-size: 22px;
  font-weight: 600;
  margin-bottom: 12px;
}
.message {
  color: var(--el-text-color-regular);
  margin-bottom: 20px;
  line-height: 1.6;
  word-break: break-word;
}
</style>
