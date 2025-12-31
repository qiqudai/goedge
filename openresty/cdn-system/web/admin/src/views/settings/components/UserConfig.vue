<template>
  <el-form label-width="160px">
    <el-card shadow="never" class="mb-20">
      <template #header>用户相关</template>
      <el-form-item label="Session有效时间">
        <el-input v-model.number="form.session_life" @blur="save"><template #append>秒</template></el-input>
      </el-form-item>
      <el-form-item label="限制普通用户登录域名">
        <el-switch v-model="form.limit_user_login_domain" @change="save" />
        <div class="form-helper">开启后，普通用户只能通过绑定的域名登录</div>
      </el-form-item>
      <el-form-item label="限制管理员登录域名">
        <el-switch v-model="form.limit_admin_login_domain" @change="save" />
        <div class="form-helper">开启后，管理员只能通过绑定的域名登录</div>
      </el-form-item>

      <el-divider>登录方式</el-divider>
      <el-form-item label="启用邮箱登录">
        <el-switch v-model="form.enable_email_login" @change="save" />
      </el-form-item>
      <el-form-item label="启用短信登录">
        <el-switch v-model="form.enable_sms_login" @change="save" />
      </el-form-item>

      <el-divider>注册设置</el-divider>
      <el-form-item label="开放注册">
        <el-switch v-model="form.open_register" @change="save" />
      </el-form-item>

      <el-divider>邮件模板</el-divider>
      <el-form-item label="注册成功标题">
        <el-input v-model="form.register_mail_title" @blur="save" />
      </el-form-item>
      <el-form-item label="注册成功内容">
        <el-input type="textarea" v-model="form.register_mail_content" :rows="4" @blur="save" />
      </el-form-item>

      <el-form-item label="找回密码标题">
        <el-input v-model="form.reset_pwd_mail_title" @blur="save" />
      </el-form-item>
      <el-form-item label="找回密码内容">
        <el-input type="textarea" v-model="form.reset_pwd_mail_content" :rows="4" @blur="save" />
      </el-form-item>

      <el-form-item label="邮箱验证标题">
        <el-input v-model="form.verify_mail_title" @blur="save" />
      </el-form-item>
      <el-form-item label="邮箱验证内容">
        <el-input type="textarea" v-model="form.verify_mail_content" :rows="4" @blur="save" />
      </el-form-item>

      <el-divider>短信模板</el-divider>
      <el-form-item label="验证码模板ID">
        <el-input v-model="form.phone_captcha_templ_id" @blur="save" placeholder="如: SMS_123456789" />
      </el-form-item>
      <el-form-item label="验证码模板内容">
        <el-input type="textarea" v-model="form.phone_captcha_templ" :rows="3" @blur="save" placeholder="您的验证码是${code}，5分钟内有效。" />
      </el-form-item>
    </el-card>
  </el-form>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const props = defineProps({
  configItems: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['saved'])

const form = ref({
  session_life: 86400,
  limit_user_login_domain: false,
  limit_admin_login_domain: false,
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
        form.value.limit_user_login_domain = val === '1' || val === 'true'
        break
      case 'limit_admin_login_domain':
        form.value.limit_admin_login_domain = val === '1' || val === 'true'
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
  items.push({ name: 'limit_user_login_domain', value: form.value.limit_user_login_domain ? '1' : '0' })
  items.push({ name: 'limit_admin_login_domain', value: form.value.limit_admin_login_domain ? '1' : '0' })
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

  request.post('/config_items', {
    type: 'system',
    scope_name: 'global',
    items: items
  }).then(() => {
    ElMessage.success('保存成功')
    emit('saved')
  })
}
</script>

<style scoped>
.mb-20 { margin-bottom: 20px; }
.form-helper { color: #999; font-size: 12px; margin-left: 10px; display: inline-block; }
</style>
