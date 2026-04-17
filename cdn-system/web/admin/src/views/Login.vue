<template>
  <div class="login-container">
    <div class="login-shell">
      <div class="login-visual">
        <img v-if="bannerUrl" :src="bannerUrl" class="login-visual-image" alt="login-banner" />
        <div v-else class="login-visual-empty">
          <img v-if="brandLogoUrl" :src="brandLogoUrl" class="login-visual-logo" alt="logo" />
          <div class="login-visual-name">{{ brandName }}</div>
        </div>
      </div>

      <div class="login-panel">
        <div class="login-panel-top">
          <img v-if="brandLogoUrl" :src="brandLogoUrl" class="login-panel-logo" alt="brand-logo" />
          <div class="login-panel-title">{{ pageTitle }}</div>
        </div>

        <div class="login-form-wrap">
          <el-form :model="form" class="login-form" @keyup.enter="handleLogin">
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
              <el-button type="primary" :loading="loading" class="login-submit" @click="handleLogin">登录</el-button>
            </el-form-item>
          </el-form>
        </div>

        <div v-if="showFooter" class="login-footer">
          <div v-if="footerLinks.length" class="login-footer-links">
            <a
              v-for="item in footerLinks"
              :key="`${item.label}-${item.url}`"
              :href="item.url"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ item.label }}
            </a>
          </div>
          <div v-if="footerCopy" class="login-footer-copy">{{ footerCopy }}</div>
        </div>
      </div>
    </div>
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
const brandLogoUrl = computed(() => systemInfo.logo_file || '')
const brandName = computed(() => systemInfo.sys_name || 'CDN')
const showCaptcha = computed(() => systemInfo.enable_email_login || systemInfo.enable_sms_login)
const showCaptchaType = computed(() => systemInfo.enable_email_login && systemInfo.enable_sms_login)
const footerCopy = computed(() => systemInfo.footer_copyright || '')
const footerLinks = computed(() => {
  const raw = systemInfo.footer_link || ''
  if (!raw) return []
  return raw
    .split(/\r?\n/)
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const [label, url] = line.split('|').map(part => part.trim())
      return { label: label || url, url }
    })
    .filter(item => item.url)
})
const showFooter = computed(() => footerLinks.value.length > 0 || !!footerCopy.value)
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
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 32px;
  background:
    radial-gradient(circle at top left, rgba(124, 164, 255, 0.28), transparent 30%),
    radial-gradient(circle at bottom right, rgba(87, 174, 255, 0.16), transparent 28%),
    linear-gradient(135deg, #eef5ff 0%, #f7f9fc 48%, #edf2fb 100%);
  box-sizing: border-box;
}

.login-shell {
  width: min(1320px, 100%);
  min-height: min(760px, calc(100vh - 64px));
  display: grid;
  grid-template-columns: minmax(420px, 1fr) minmax(420px, 0.92fr);
  background: rgba(255, 255, 255, 0.96);
  border-radius: 28px;
  overflow: hidden;
  box-shadow: 0 28px 80px rgba(52, 72, 108, 0.18);
  border: 1px solid rgba(210, 220, 236, 0.8);
}

.login-visual {
  background: linear-gradient(180deg, #e9f2ff 0%, #f4f8ff 100%);
  display: flex;
  align-items: stretch;
  justify-content: center;
  padding: 28px;
}

.login-visual-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 18px;
  background: #eef4ff;
}

.login-visual-empty {
  width: 100%;
  border-radius: 18px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #58708f;
  background:
    linear-gradient(135deg, rgba(120, 160, 255, 0.14), rgba(255, 255, 255, 0.45)),
    linear-gradient(180deg, #edf4ff, #f8fbff);
}

.login-visual-logo {
  max-width: 240px;
  max-height: 96px;
  margin-bottom: 20px;
  object-fit: contain;
}

.login-visual-name {
  font-size: 42px;
  line-height: 1.1;
  color: #4d6284;
  letter-spacing: 0.02em;
}

.login-panel {
  display: flex;
  flex-direction: column;
  padding: 42px 58px 36px;
  background: rgba(255, 255, 255, 0.98);
}

.login-panel-top {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 16px;
  min-height: 160px;
  text-align: center;
}

.login-panel-logo {
  max-width: 120px;
  max-height: 48px;
  object-fit: contain;
}

.login-panel-title {
  font-size: clamp(34px, 3vw, 56px);
  line-height: 1.1;
  font-weight: 300;
  color: #59657f;
  text-align: center;
}

.login-form-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.login-form {
  width: min(100%, 560px);
}

.login-submit {
  width: 100%;
  height: 48px;
  font-size: 18px;
  border-radius: 10px;
}

.login-footer {
  padding-top: 18px;
  text-align: center;
  color: #8390a7;
}

.login-footer-links {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 18px;
  margin-bottom: 10px;
}

.login-footer-links a {
  color: #5d8fe9;
  text-decoration: none;
}

.login-footer-links a:hover {
  text-decoration: underline;
}

.login-footer-copy {
  font-size: 13px;
  line-height: 1.6;
}

@media (max-width: 980px) {
  .login-container {
    padding: 16px;
  }

  .login-shell {
    grid-template-columns: 1fr;
    min-height: auto;
  }

  .login-visual {
    min-height: 240px;
    padding: 20px;
  }

  .login-panel {
    padding: 28px 22px 24px;
  }

  .login-panel-top {
    justify-content: center;
    min-height: auto;
    margin-bottom: 12px;
  }

  .login-panel-title {
    text-align: center;
    font-size: 32px;
  }

  .login-form-wrap {
    align-items: flex-start;
  }
}
</style>
