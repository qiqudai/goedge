<template>
  <div class="app-container" @focusin="cacheInputValue">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>资源限制配置</span>
        </div>
      </template>

      <el-tabs v-if="config.resources" v-model="activeTab" v-loading="loading" type="border-card" @tab-change="handleTabChange">
        <el-tab-pane label="网站 (Website)" name="website">
          <el-form label-width="220px">
            <h4>配置限制</h4>
            <el-form-item label="相关配置限制不低于">
              <el-input-number v-model="config.resources.website.min_limit" @blur="saveConfig" />
              <div class="tip">限制证书 DNS API、证书、ACL、CC 匹配器/过滤器/规则、网站分组等数量下限。</div>
            </el-form-item>
            <el-form-item label="相关配置限制的最大倍数">
              <el-input-number v-model="config.resources.website.max_limit_multiplier" @blur="saveConfig" />
              <div class="tip">与当前网站数相乘得到相关配置数量上限（不低于上面的下限）。</div>
            </el-form-item>

            <h4>黑白名单</h4>
            <el-form-item label="黑名单 IP 数量限制">
              <el-input-number v-model="config.resources.website.max_blacklist_ips" @blur="saveConfig" />
              <div class="tip">限制单个网站可配置的黑名单 IP 条数。</div>
            </el-form-item>
            <el-form-item label="白名单 IP 数量限制">
              <el-input-number v-model="config.resources.website.max_whitelist_ips" @blur="saveConfig" />
              <div class="tip">限制单个网站可配置的白名单 IP 条数。</div>
            </el-form-item>
            <el-form-item label="WAF 泛 IP 数量限制">
              <el-input-number v-model="config.resources.website.max_waf_pattern_ips" :min="1" @blur="saveConfig" />
              <div class="tip">限制全局 WAF 黑白名单中 CIDR 与通配符规则数量，精准 IP 不计入该限制。</div>
            </el-form-item>

            <h4>清缓存及解锁</h4>
            <el-form-item label="日清 URL 缓存次数限制">
              <el-input-number v-model="config.resources.website.daily_url_purge_limit" @blur="saveConfig" />
              <div class="tip">每个用户每日刷新 URL 缓存次数上限。</div>
            </el-form-item>
            <el-form-item label="日清目录缓存次数限制">
              <el-input-number v-model="config.resources.website.daily_dir_purge_limit" @blur="saveConfig" />
              <div class="tip">每个用户每日刷新目录缓存次数上限。</div>
            </el-form-item>
            <el-form-item label="日预热 URL 次数限制">
              <el-input-number v-model="config.resources.website.daily_preload_limit" @blur="saveConfig" />
              <div class="tip">每个用户每日预热 URL 次数上限。</div>
            </el-form-item>
            <el-form-item label="预热超时">
              <el-input-number v-model="config.resources.website.preload_timeout" :min="1" @blur="saveConfig" />
              <span class="unit">秒</span>
              <div class="tip">预热单个 URL 的最大等待时间，超时则中止。</div>
            </el-form-item>
            <el-form-item label="日解锁 IP 次数限制">
              <el-input-number v-model="config.resources.website.daily_unlock_ip_limit" @blur="saveConfig" />
              <div class="tip">每个用户每日解锁黑名单 IP 的总次数上限。</div>
            </el-form-item>
            <el-form-item label="每次解锁 IP 个数限制">
              <el-input-number v-model="config.resources.website.unlock_ip_batch_limit" @blur="saveConfig" />
              <div class="tip">单次解锁操作允许提交的 IP 数量上限。</div>
            </el-form-item>

            <h4>规则数限制</h4>
            <el-form-item label="单个 CC 规则数量">
              <el-input-number v-model="config.resources.website.max_cc_rules_per_group" @blur="saveConfig" />
              <div class="tip">单个 CC 规则组内允许配置的规则条数上限。</div>
            </el-form-item>
            <el-form-item label="单个 ACL 规则数量">
              <el-input-number v-model="config.resources.website.max_acl_rules" @blur="saveConfig" />
              <div class="tip">单个 ACL 允许配置的规则条数上限。</div>
            </el-form-item>

            <h4>下载日志</h4>
            <el-form-item label="每天允许下载日志次数">
              <el-input-number v-model="config.resources.website.daily_log_download_limit" @blur="saveConfig" />
              <div class="tip">每个用户每日申请下载访问日志的次数上限。</div>
            </el-form-item>
            <el-form-item label="日志文件存放目录">
              <el-input v-model="config.resources.website.log_storage_dir" @blur="saveConfig" />
              <div class="tip">日志打包文件的临时存放目录（控制面）。</div>
            </el-form-item>
            <el-form-item label="日志文件存放时长 (小时)">
              <el-input-number v-model="config.resources.website.log_storage_hours" @blur="saveConfig" />
              <div class="tip">临时日志文件保留时长，过期后自动清理。</div>
            </el-form-item>

            <h4>其它</h4>
            <el-form-item label="单个站点最大域名数限制">
              <el-input-number v-model="config.resources.website.max_domains_per_site" @blur="saveConfig" />
              <div class="tip">单个站点可绑定的域名数量上限，建议不超过 100（Let's Encrypt 上限）。</div>
            </el-form-item>
            <el-form-item label="默认监听 80 端口">
              <el-switch v-model="config.resources.website.default_listen_80" active-text="开启" inactive-text="关闭" @change="saveConfig" />
              <div class="tip">关闭后节点不再默认监听 80，适用于四层转发需占用 80 端口的场景。</div>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="转发 (Forwarding)" name="forward">
          <el-form label-width="220px">
            <el-form-item label="禁用的端口">
              <el-input v-model="config.resources.forward.disabled_ports" placeholder="80 443" @blur="saveConfig" />
              <div class="tip">禁止四层转发监听的端口，默认 80 443，预留给 HTTP/HTTPS 站点。</div>
            </el-form-item>

            <h4>配置限制</h4>
            <el-form-item label="相关配置限制不低于">
              <el-input-number v-model="config.resources.forward.min_limit" @blur="saveConfig" />
              <div class="tip">限制转发分组等相关配置数量的下限。</div>
            </el-form-item>
            <el-form-item label="相关配置限制的最大倍数">
              <el-input-number v-model="config.resources.forward.max_limit_multiplier" @blur="saveConfig" />
              <div class="tip">与当前转发数相乘得到相关配置数量上限。</div>
            </el-form-item>

            <h4>规则数限制</h4>
            <el-form-item label="ACL 规则数量限制">
              <el-input-number v-model="config.resources.forward.max_acl_rules" @blur="saveConfig" />
              <div class="tip">四层转发 ACL 规则条数上限，过多会影响性能。</div>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="公共 (Public)" name="public">
          <el-form label-width="220px">
            <el-form-item label="禁用的自定义端口">
              <el-input v-model="config.resources.public.disabled_custom_ports" placeholder="22 5000" @blur="saveConfig" />
              <div class="tip">同时禁止网站 HTTP/HTTPS 与四层转发监听的端口，多个端口以空格分隔，默认 22 5000。</div>
            </el-form-item>
            <el-form-item label="允许的自定义端口">
              <el-input v-model="config.resources.public.allowed_custom_ports" placeholder="1-65535" @blur="saveConfig" />
              <div class="tip">允许监听的自定义端口（网站与转发共用），可填范围如 81-222、单端口 99，多个值以逗号或空格分隔。</div>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const activeTab = ref('website')
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
    if (res.code === 0 || res.code === 200) {
      config.value = res.data || {}
      if (!config.value.resources) {
        config.value.resources = { website: {}, forward: {}, public: {} }
      }
      const website = config.value.resources.website || {}
      if (website.preload_timeout == null || website.preload_timeout === 0) {
        website.preload_timeout = 120
      }
      if (website.max_waf_pattern_ips == null || website.max_waf_pattern_ips === 0) {
        website.max_waf_pattern_ips = 100
      }
      config.value.resources.website = website
    }
  }).finally(() => {
    loading.value = false
  })
}

const saving = ref(false)
let saveQueued = false

const cacheInputValue = (event) => {
  const el = event?.target
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
    return
  }
  el.dataset.lastValue = el.value ?? ''
}

const shouldSkipBlurSave = (event) => {
  const el = event?.target
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
    return false
  }
  const value = el.value ?? ''
  const lastValue = el.dataset.lastValue ?? ''
  if (value === '' || value === lastValue) {
    return true
  }
  el.dataset.lastValue = value
  return false
}

const saveConfig = async (event) => {
  if (shouldSkipBlurSave(event)) {
    return
  }
  if (saving.value) {
    saveQueued = true
    return
  }
  saving.value = true
  await nextTick()
  request.post('/global_config', config.value).then(res => {
    if (res.code === 0 || res.code === 200) {
      ElMessage.success('资源配置已保存')
    }
  }).finally(() => {
    saving.value = false
    if (saveQueued) {
      saveQueued = false
      saveConfig()
    }
  })
}

const handleTabChange = () => {
  loadConfig()
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
  color: var(--el-text-color-secondary, #999);
  margin-top: 4px;
  line-height: 1.5;
}
.unit {
  margin-left: 8px;
  color: var(--el-text-color-secondary, #999);
}
h4 {
  margin-top: 10px;
  margin-bottom: 20px;
  padding-left: 10px;
  border-left: 4px solid var(--el-color-primary, #409eff);
}
</style>
