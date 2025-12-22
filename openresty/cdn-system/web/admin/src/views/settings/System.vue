<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" type="border-card">
      <el-tab-pane label="系统配置" name="system">
        <el-form label-width="120px" :model="form">
          <el-form-item label="系统名称"><el-input v-model="form.system_name" /></el-form-item>
          <el-form-item label="普通用户标题"><el-input v-model="form.user_title" /></el-form-item>
          <el-form-item label="管理员标题"><el-input v-model="form.admin_title" /></el-form-item>
          <el-form-item label="底部链接">
            <el-input v-model="form.footer_links" type="textarea" placeholder="格式: 名称|URL" rows="3" />
          </el-form-item>
          <el-form-item label="底部文字"><el-input v-model="form.footer_text" /></el-form-item>
          <el-form-item label="全站JS"><el-input v-model="form.global_js" type="textarea" rows="4" /></el-form-item>

          <el-divider>套餐相关</el-divider>
          <el-form-item label="套餐到期关闭"><el-switch v-model="form.expire_close_site" /></el-form-item>
          <el-form-item label="流量超限关闭"><el-switch v-model="form.traffic_close_site" /></el-form-item>
          <el-form-item label="允许自主升级"><el-switch v-model="form.allow_upgrade" /></el-form-item>
          <el-form-item label="允许自主降级"><el-switch v-model="form.allow_downgrade" /></el-form-item>

          <el-divider>维护升级</el-divider>
          <el-form-item label="维护状态"><el-switch v-model="form.maintenance_status" /></el-form-item>
          <el-form-item label="维护提示"><el-input v-model="form.maintenance_msg" /></el-form-item>
          <el-form-item label="自动升级节点"><el-switch v-model="form.auto_upgrade_node" /></el-form-item>

          <el-form-item>
            <el-button type="primary" @click="save">保存</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="数据清理" name="cleaning">
        <el-form label-width="180px" :model="form">
          <el-form-item label="清缓存/解锁IP记录">
            <el-input v-model="form.clean_cache_days"><template #append>天</template></el-input>
          </el-form-item>
          <el-form-item label="登录记录">
            <el-input v-model="form.clean_login_log_days"><template #append>天</template></el-input>
          </el-form-item>
          <el-form-item label="操作记录">
            <el-input v-model="form.clean_op_log_days"><template #append>天</template></el-input>
          </el-form-item>
          <el-form-item label="网站访问日志(ES)">
            <el-input v-model="form.clean_site_log_days"><template #append>天</template></el-input>
          </el-form-item>
          <el-form-item label="节点监控数据">
            <el-input v-model="form.clean_node_monitor_days"><template #append>天</template></el-input>
          </el-form-item>
          <el-form-item label="流量带宽历史">
            <el-input v-model="form.clean_traffic_days"><template #append>天</template></el-input>
          </el-form-item>

          <el-divider>数据备份</el-divider>
          <el-form-item label="备份频率">
            <el-input v-model="form.backup_frequency"><template #append>天</template></el-input>
          </el-form-item>
          <el-form-item label="保留天数">
            <el-input v-model="form.backup_retention"><template #append>天</template></el-input>
          </el-form-item>
          <el-form-item label="备份目录"><el-input v-model="form.backup_dir" /></el-form-item>

          <el-form-item>
            <el-button type="primary" @click="save">保存</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="用户相关" name="user">
        <el-form label-width="150px" :model="form">
          <el-form-item label="Session有效时间"><el-input v-model="form.session_life"><template #append>秒</template></el-input></el-form-item>
          <el-form-item label="限制普通用户登录域名"><el-switch v-model="form.limit_user_login_domain" /></el-form-item>
          <el-form-item label="开放注册"><el-switch v-model="form.open_register" /></el-form-item>

          <el-divider>邮件模板</el-divider>
          <el-form-item label="注册成功标题"><el-input v-model="form.register_mail_title" /></el-form-item>
          <el-form-item label="注册成功内容"><el-input type="textarea" v-model="form.register_mail_content" rows="4" /></el-form-item>

          <el-form-item>
            <el-button type="primary" @click="save">保存</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="通知配置" name="notify">
        <el-form label-width="180px" :model="form">
          <el-card shadow="never" class="mb-20">
            <template #header>流量已超限通知</template>
            <el-form-item label="开启"><el-switch v-model="form.notify_traffic_exceeded" /></el-form-item>
            <el-form-item label="通知模板标题"><el-input v-model="form.notify_traffic_exceeded_title" /></el-form-item>
          </el-card>

          <el-card shadow="never" class="mb-20">
            <template #header>流量不足通知</template>
            <el-form-item label="开启"><el-switch v-model="form.notify_traffic_low" /></el-form-item>
            <el-form-item label="阈值(GB)"><el-input v-model="form.notify_traffic_low_threshold" /></el-form-item>
          </el-card>

          <el-card shadow="never" class="mb-20">
            <template #header>套餐过期通知</template>
            <el-form-item label="开启"><el-switch v-model="form.notify_expire" /></el-form-item>
          </el-card>

          <el-form-item>
            <el-button type="primary" @click="save">保存</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="HTTPS设置" name="https">
        <el-form label-width="120px" :model="form">
          <el-form-item label="强制HTTPS"><el-switch v-model="form.force_ssl" /></el-form-item>
          <el-form-item label="公钥 (PEM)"><el-input type="textarea" v-model="form.cert_content" rows="6" /></el-form-item>
          <el-form-item label="私钥 (PEM)"><el-input type="textarea" v-model="form.key_content" rows="6" /></el-form-item>

          <el-alert title="保存后需要重启Master服务生效" type="info" show-icon />

          <el-form-item>
            <el-button type="primary" @click="save">保存</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const activeTab = ref('system')
const form = ref({})

const loadData = () => {
  request.get('/system_info').then(res => {
    form.value = res.data || {}
  })
}

const save = () => {
  request.post('/system_info', form.value).then(() => {
    ElMessage.success('配置已保存')
  })
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.mb-20 {
  margin-bottom: 20px;
}
</style>
