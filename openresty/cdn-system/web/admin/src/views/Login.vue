<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <div class="login-header">
          <h2>Edge Admin</h2>
        </div>
      </template>
      <el-form :model="form" @keyup.enter="handleLogin">
        <el-form-item>
          <el-input v-model="form.username" placeholder="Username" prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="Password" prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" style="width: 100%" @click="handleLogin">Login</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const router = useRouter()
const form = reactive({ username: '', password: '' })
const loading = ref(false)

const applyImpersonateFromQuery = () => {
  const params = new URLSearchParams(window.location.search)
  const token = params.get('token')
  const role = params.get('role')
  const redirect = params.get('redirect') || '/dashboard'
  if (!token || !role) return
  localStorage.setItem('admin_token', token)
  localStorage.setItem('role', role)
  router.replace(redirect)
}

const handleLogin = () => {
  if (!form.username || !form.password) return
  loading.value = true
  request.post('/login', form).then(res => {
    localStorage.setItem('admin_token', res.token)
    localStorage.setItem('role', res.role || 'user')
    localStorage.setItem('username', form.username)
    ElMessage.success('Login success')
    router.push('/dashboard')
  }).catch(() => {
    loading.value = false
  })
}

onMounted(() => {
  applyImpersonateFromQuery()
})
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #f0f2f5;
}
.login-card {
  width: 400px;
}
.login-header {
  text-align: center;
}
</style>
