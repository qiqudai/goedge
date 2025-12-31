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
import { ref, onMounted } from 'vue'
import request from '@/utils/request'

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

const fetchData = async () => {
  try {
    const res = await request.get('/api/v1/admin/config_items', {
      params: { type: 'system', scope_name: 'global' }
    })
    const items = res.data.items || []
    const infoItem = items.find(i => i.name === 'system_info')
    if (infoItem && infoItem.value) {
      try {
        const parsed = JSON.parse(infoItem.value)
        const keys = Object.keys(form.value)
        keys.forEach(k => {
          if (parsed[k] !== undefined) form.value[k] = parsed[k]
        })
      } catch (e) { /* ignore */ }
    }
  } catch (e) {
    console.error('获取配置失败', e)
  }
}

const save = () => {
  const items = []
  items.push({
    name: 'system_info',
    value: JSON.stringify(form.value)
  })

  request.post('/api/v1/admin/config_items', {
    type: 'system',
    scope_name: 'global',
    items: items
  }).catch((e) => {
    console.error('保存配置失败', e)
  })
}

onMounted(fetchData)
</script>

<style scoped>
.mb-20 { margin-bottom: 20px; }
.form-helper { color: #999; font-size: 12px; margin-left: 10px; display: inline-block; }
</style>
