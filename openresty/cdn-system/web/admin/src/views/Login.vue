<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <div class="login-header">
           <h2>Edge 管理后台</h2>
        </div>
      </template>
      <el-form :model="form" @keyup.enter="handleLogin">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户�? prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" style="width: 100%" @click="handleLogin">登录</el-button>
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

const handleLogin = () => {
    if (!form.username || !form.password) return
    loading.value = true
    
    // Call API 
    // Backend: /api/v1/login
    request.post('/login', form).then(res => {
        localStorage.setItem('admin_token', res.token) // Fix: res IS the data object
        localStorage.setItem('role', res.role || 'user') // Store Role
        localStorage.setItem('username', form.username)
        ElMessage.success('登录成功')
        
        // Redirect to Dashboard for all users as requested
        router.push('/dashboard')
    }).catch(() => {
        loading.value = false
    })
}
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

