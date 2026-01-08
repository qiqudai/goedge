<template>
  <el-form label-width="160px" @focusin="cacheInputValue">
    <el-card shadow="never" class="mb-20">
      <template #header>用户相关</template>
      <el-form-item label="Session有效时间">
        <el-input v-model.number="form.session_life" @blur="handleBlurSave"><template #append>秒</template></el-input>
      </el-form-item>
            <el-form-item label="????????????????>
        <el-input
          v-model="form.limit_user_login_domain"
          placeholder="?? user ? user.example.com"
          @blur="handleBlurSave"
        />
        <div class="form-helper">???????????????????????????????????bind-master-host???</div>
      </el-form-item>
            <el-form-item label="??????????????>
        <el-input
          v-model="form.limit_admin_login_domain"
          placeholder="?? admin.example.com"
          @blur="handleBlurSave"
        />
        <div class="form-helper">???????????????????????????????????bind-master-host???</div>
      </el-form-item>

      <el-divider>登录方式</el-divider>
      <el-form-item label="启用邮箱登录">
        <el-switch v-model="form.enable_email_login" @change="handleSave" />
      </el-form-item>
      <el-form-item label="启用短信登录">
        <el-switch v-model="form.enable_sms_login" @change="handleSave" />
      </el-form-item>

      <el-divider>注册设置</el-divider>
      <el-form-item label="开放注册">
        <el-switch v-model="form.open_register" @change="handleSave" />
      </el-form-item>

      <el-divider>邮件模板</el-divider>
      <el-form-item label="注册成功标题">
        <el-input v-model="form.register_mail_title" @blur="handleBlurSave" />
      </el-form-item>
      <el-form-item label="注册成功内容">
        <el-input type="textarea" v-model="form.register_mail_content" :rows="4" @blur="handleBlurSave" />
      </el-form-item>

      <el-form-item label="找回密码标题">
        <el-input v-model="form.reset_pwd_mail_title" @blur="handleBlurSave" />
      </el-form-item>
      <el-form-item label="找回密码内容">
        <el-input type="textarea" v-model="form.reset_pwd_mail_content" :rows="4" @blur="handleBlurSave" />
      </el-form-item>

      <el-form-item label="邮箱验证标题">
        <el-input v-model="form.verify_mail_title" @blur="handleBlurSave" />
      </el-form-item>
      <el-form-item label="邮箱验证内容">
        <el-input type="textarea" v-model="form.verify_mail_content" :rows="4" @blur="handleBlurSave" />
      </el-form-item>

      <el-divider>短信模板</el-divider>
      <el-form-item label="验证码模板ID">
        <el-input v-model="form.phone_captcha_templ_id" @blur="handleBlurSave" placeholder="如: SMS_123456789" />
      </el-form-item>
      <el-form-item label="验证码模板内容">
        <el-input type="textarea" v-model="form.phone_captcha_templ" :rows="3" @blur="handleBlurSave" placeholder="您的验证码是${code}，5分钟内有效。" />
      </el-form-item>
    </el-card>
  </el-form>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import { cacheInputValue, shouldSkipBlurSave } from '@/utils/saveGuard'

const props = defineProps({
  configItems: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['saved'])

const form = ref({
  session_life: 86400,
  limit_user_login_domain: '',
  limit_admin_login_domain: '',
  enable_email_login: false,
  enable_sms_login: false,
  open_register: true,
  register_mail_title: '',
  register_mail_content: '',
  reset_pwd_mail_title: '',
  reset_pwd_mail_content: '',
  verify_mail_title: '',
  verify_mail_content: '',
  phone_captcha_templ_id: '',
  phone_captcha_templ: ''
})

// Helper to safely parse JSON
const safeParse = (str) => {
    try {
        return str ? JSON.parse(str) : {}
    } catch(e) {
        return {}
    }
}

watch(() => props.configItems, (items) => {
  if (!items) return

  items.forEach(item => {
    const val = item.value
    switch (item.name) {
      case 'login_session_valid_time':
        form.value.session_life = parseInt(val) || 86400
        break
      case 'limit_user_login_domain':
        form.value.limit_user_login_domain = normalizeDomainValue(val)
        break
      case 'limit_admin_login_domain':
        form.value.limit_admin_login_domain = normalizeDomainValue(val)
        break
      case 'allow-enable-email-captcha-login':
        form.value.enable_email_login = val === '1' || val === 'true'
        break
      case 'allow-enable-sms-captcha-login':
        form.value.enable_sms_login = val === '1' || val === 'true'
        break
      case 'allow_register':
        form.value.open_register = val === '1' || val === 'true' || val === 1
        break
      case 'register_success_templ':
        {
           const obj = safeParse(val)
           form.value.register_mail_title = obj.title || ''
           form.value.register_mail_content = obj.data || '' // Dump uses 'data' key for content? "data":"<p>..."
        }
        break
      case 'forget_password_templ':
        {
           const obj = safeParse(val)
           form.value.reset_pwd_mail_title = obj.title || ''
           form.value.reset_pwd_mail_content = obj.data || ''
        }
        break
      case 'email_captcha_templ':
        {
           const obj = safeParse(val)
           form.value.verify_mail_title = obj.title || ''
           form.value.verify_mail_content = obj.data || ''
        }
        break
      case 'phone_captcha_templ':
         // Dump shows string: "【cdn】您的验证码..."
         form.value.phone_captcha_templ = val
         break
      // Configs requiring ID (not in dump but maybe needed?)
      // We will skip phone_captcha_templ_id if not present in dump mapping
    }
  })
}, { immediate: true, deep: true })

const save = () => {
  const items = []
  
  items.push({ name: 'login_session_valid_time', value: String(form.value.session_life) })
  items.push({ name: 'limit_user_login_domain', value: String(form.value.limit_user_login_domain || '') })
  items.push({ name: 'limit_admin_login_domain', value: String(form.value.limit_admin_login_domain || '') })
  items.push({ name: 'allow-enable-email-captcha-login', value: form.value.enable_email_login ? '1' : '0' })
  items.push({ name: 'allow-enable-sms-captcha-login', value: form.value.enable_sms_login ? '1' : '0' })
  items.push({ name: 'allow_register', value: form.value.open_register ? '1' : '0' })

  // Templates JSON
  items.push({
      name: 'register_success_templ',
      value: JSON.stringify({
          title: form.value.register_mail_title,
          data: form.value.register_mail_content
      })
  })
  items.push({
      name: 'forget_password_templ',
      value: JSON.stringify({
          title: form.value.reset_pwd_mail_title,
          data: form.value.reset_pwd_mail_content
      })
  })
  items.push({
      name: 'email_captcha_templ',
      value: JSON.stringify({
          title: form.value.verify_mail_title,
          data: form.value.verify_mail_content
      })
  })
  
  items.push({
      name: 'phone_captcha_templ',
      value: form.value.phone_captcha_templ
  })

  return request.post('/config_items', {
    type: 'system',
    scope_name: 'global',
    items: items
  }).then(() => {
    ElMessage.success('保存成功')
    emit('saved')
  })
}

const normalizeDomainValue = (value) => {
  const raw = String(value || '').trim()
  if (['0', '1', 'true', 'false'].includes(raw)) {
    return ''
  }
  return raw
}

const saving = ref(false)
let saveQueued = false

const queueSave = async () => {
  if (saving.value) {
    saveQueued = true
    return
  }
  saving.value = true
  await nextTick()
  save().finally(() => {
    saving.value = false
    if (saveQueued) {
      saveQueued = false
      queueSave()
    }
  })
}

const handleSave = () => {
  queueSave()
}

const handleBlurSave = (event) => {
  if (shouldSkipBlurSave(event)) {
    return
  }
  queueSave()
}
</script>

<style scoped>
.mb-20 { margin-bottom: 20px; }
.form-helper { color: #999; font-size: 12px; margin-left: 10px; display: inline-block; }
</style>
