<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>资源限制配置</span>
          <el-button type="primary" @click="saveConfig" :loading="loading">保存配置</el-button>
        </div>
      </template>

      <el-tabs type="border-card" v-if="config.resources">
        <el-tab-pane label="网站 (Website)">
          <el-form label-width="220px">
            <h4>配置限制</h4>
            <el-form-item label="相关配置限制不低于">
              <el-input-number v-model="config.resources.website.min_limit" />
            </el-form-item>
            <el-form-item label="相关配置限制的最大倍数">
              <el-input-number v-model="config.resources.website.max_limit_multiplier" />
            </el-form-item>

            <h4>黑白名单</h4>
            <el-form-item label="黑名单 IP 数量限制">
              <el-input-number v-model="config.resources.website.max_blacklist_ips" />
            </el-form-item>
            <el-form-item label="白名单 IP 数量限制">
              <el-input-number v-model="config.resources.website.max_whitelist_ips" />
            </el-form-item>

            <h4>清缓存及解锁</h4>
            <el-form-item label="日清 URL 缓存次数限制">
              <el-input-number v-model="config.resources.website.daily_url_purge_limit" />
            </el-form-item>
            <el-form-item label="日清目录缓存次数限制">
              <el-input-number v-model="config.resources.website.daily_dir_purge_limit" />
            </el-form-item>
            <el-form-item label="日预热 URL 次数限制">
              <el-input-number v-model="config.resources.website.daily_preload_limit" />
            </el-form-item>
            <el-form-item label="日解锁 IP 次数限制">
              <el-input-number v-model="config.resources.website.daily_unlock_ip_limit" />
            </el-form-item>
            <el-form-item label="每次解锁 IP 个数限制">
              <el-input-number v-model="config.resources.website.unlock_ip_batch_limit" />
            </el-form-item>

            <h4>规则数限制</h4>
            <el-form-item label="单个 CC 规则数量">
              <el-input-number v-model="config.resources.website.max_cc_rules_per_group" />
            </el-form-item>
            <el-form-item label="单个 ACL 规则数量">
              <el-input-number v-model="config.resources.website.max_acl_rules" />
            </el-form-item>

            <h4>下载日志</h4>
            <el-form-item label="每天允许下载日志次数">
              <el-input-number v-model="config.resources.website.daily_log_download_limit" />
            </el-form-item>
            <el-form-item label="日志文件存放目录">
              <el-input v-model="config.resources.website.log_storage_dir" />
            </el-form-item>
            <el-form-item label="日志文件存放时长 (小时)">
              <el-input-number v-model="config.resources.website.log_storage_hours" />
            </el-form-item>

            <h4>其它</h4>
            <el-form-item label="单个站点最大域名数限制">
              <el-input-number v-model="config.resources.website.max_domains_per_site" />
            </el-form-item>
            <el-form-item label="默认监听 80 端口">
              <el-switch v-model="config.resources.website.default_listen_80" active-text="开启" inactive-text="关闭" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="转发 (Forwarding)">
          <el-form label-width="220px">
            <el-form-item label="禁用的端口">
              <el-input v-model="config.resources.forward.disabled_ports" placeholder="80 443" />
              <div class="tip">空格分隔，例如 "80 443"。</div>
            </el-form-item>

            <h4>配置限制</h4>
            <el-form-item label="相关配置限制不低于">
              <el-input-number v-model="config.resources.forward.min_limit" />
            </el-form-item>
            <el-form-item label="相关配置限制的最大倍数">
              <el-input-number v-model="config.resources.forward.max_limit_multiplier" />
            </el-form-item>

            <h4>规则数限制</h4>
            <el-form-item label="ACL 规则数量限制">
              <el-input-number v-model="config.resources.forward.max_acl_rules" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="公共 (Public)">
          <el-form label-width="220px">
            <el-form-item label="禁用的自定义端口">
              <el-input v-model="config.resources.public.disabled_custom_ports" placeholder="22" />
              <div class="tip">空格分隔，例如 "22"。</div>
            </el-form-item>
            <el-form-item label="允许的自定义端口">
              <el-input v-model="config.resources.public.allowed_custom_ports" placeholder="1-65535" />
              <div class="tip">范围或列表，例如 "1-65535"。</div>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const config = ref({
  resources: {
    website: {},
    forward: {},
    public: {}
  }
})

const loadConfig = () => {
  loading.value = true
  request.get('/global_config').then(res => {
    if (res.code === 0) {
      config.value = res.data || {}
      if (!config.value.resources) {
        config.value.resources = { website: {}, forward: {}, public: {} }
      }
    }
  }).finally(() => {
    loading.value = false
  })
}

const saveConfig = () => {
  loading.value = true
  request.post('/global_config', config.value).then(res => {
    if (res.code === 0) {
      ElMessage.success('资源配置已保存')
    }
  }).finally(() => {
    loading.value = false
  })
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tip {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}
h4 {
  margin-top: 10px;
  margin-bottom: 20px;
  padding-left: 10px;
  border-left: 4px solid #409eff;
}
</style>
