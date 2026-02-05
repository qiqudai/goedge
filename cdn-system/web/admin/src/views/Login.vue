<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <div class="login-header">
          <img v-if="bannerUrl" :src="bannerUrl" class="login-logo" alt="logo" />
          <h2>{{ pageTitle }}</h2>
        </div>
      </template>
      <el-form :model="form" @keyup.enter="handleLogin">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item v-if="showCaptchaType">
          <el-radio-group v-model="captchaType">
            <el-radio label="email">邮箱验证码</el-radio>
            <el-radio label="sms">短信验证码</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="showCaptcha">
          <el-row :gutter="10" style="width: 100%">
            <el-col :span="14">
              <el-input v-model="form.captcha" :placeholder="captchaPlaceholder" />
            </el-col>
            <el-col :span="10">
              <el-button :disabled="captchaCountdown > 0 || captchaSending" style="width: 100%" @click="sendLoginCaptcha">
                {{ captchaButtonText }}
              </el-button>
            </el-col>
          </el-row>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" style="width: 100%" @click="handleLogin">登录</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, computed, watch, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import { useSystemInfo } from '@/composables/useSystemInfo'
import { sha256Hex } from '@/utils/crypto'

const router = useRouter()
const form = reactive({ username: '', password: '', captcha: '' })
const loading = ref(false)
const { systemInfo, loadSystemInfo } = useSystemInfo()
const pageTitle = computed(() => systemInfo.admin_console_title || systemInfo.sys_name || 'Edge Admin')
const bannerUrl = computed(() => systemInfo.login_ad_file || systemInfo.logo_file || '')
const showCaptcha = computed(() => systemInfo.enable_email_login || systemInfo.enable_sms_login)
const showCaptchaType = computed(() => systemInfo.enable_email_login && systemInfo.enable_sms_login)
const captchaType = ref('')
const captchaSending = ref(false)
const captchaCountdown = ref(0)
let captchaTimer = null

const captchaPlaceholder = computed(() => {
  if (!showCaptcha.value) return ''
  return captchaType.value === 'sms' ? '短信验证码' : '邮箱验证码'
})

const captchaButtonText = computed(() => {
  if (captchaCountdown.value > 0) {
    return `${captchaCountdown.value}s`
  }
  return '获取验证码'
})

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

const handleLogin = async () => {
  if (!form.username || !form.password) return
  if (showCaptcha.value && !form.captcha) {
    ElMessage.warning('请输入验证码')
    return
  }
  loading.value = true
  let passwordHash = ''
  try {
    passwordHash = await sha256Hex(form.password)
  } catch (err) {
    loading.value = false
    ElMessage.error('请使用 HTTPS 访问以启用密码加密')
    return
  }
  request.post('/login', {
    username: form.username,
    password: passwordHash,
    password_hash: 'sha256',
    captcha: form.captcha,
    captcha_type: captchaType.value
  }).then(res => {
    const payload = res?.data || res || {}
    if (!payload.token) {
      ElMessage.error(res?.message || '登录失败')
      loading.value = false
      return
    }
    localStorage.setItem('admin_token', payload.token)
    localStorage.setItem('role', payload.role || 'user')
    localStorage.setItem('username', form.username)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  }).catch(() => {
    loading.value = false
  })
}

const sendLoginCaptcha = () => {
  if (!form.username) {
    ElMessage.warning('请输入用户名')
    return
  }
  if (!showCaptcha.value) {
    return
  }
  captchaSending.value = true
  request.post('/login/captcha', {
    username: form.username,
    type: captchaType.value
  }).then(() => {
    ElMessage.success('验证码已发送')
    startCaptchaCountdown()
  }).finally(() => {
    captchaSending.value = false
  })
}

const startCaptchaCountdown = () => {
  captchaCountdown.value = 60
  if (captchaTimer) {
    clearInterval(captchaTimer)
  }
  captchaTimer = setInterval(() => {
    captchaCountdown.value -= 1
    if (captchaCountdown.value <= 0) {
      clearInterval(captchaTimer)
      captchaTimer = null
      captchaCountdown.value = 0
    }
  }, 1000)
}

onMounted(() => {
  applyImpersonateFromQuery()
  loadSystemInfo()
})

watch(
  () => [systemInfo.enable_email_login, systemInfo.enable_sms_login],
  ([enableEmail, enableSms]) => {
    if (enableEmail) {
      captchaType.value = 'email'
    } else if (enableSms) {
      captchaType.value = 'sms'
    } else {
      captchaType.value = ''
    }
  },
  { immediate: true }
)

watch(pageTitle, (val) => {
  if (val) {
    document.title = val
  }
}, { immediate: true })

onBeforeUnmount(() => {
  if (captchaTimer) {
    clearInterval(captchaTimer)
  }
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
.login-logo {
  display: block;
  max-width: 180px;
  max-height: 72px;
  margin: 0 auto 12px;
  object-fit: contain;
}
</style>
