<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>防火墙全局配置 (WAF)</span>
          <el-button type="primary" @click="saveConfig" :loading="loading">保存配置</el-button>
        </div>
      </template>

      <el-tabs type="border-card" v-if="config.waf">
        <el-tab-pane label="基础防护 & 拉黑策略">
          <el-form label-width="180px">
            <el-form-item label="全局 WAF 开启">
              <el-switch v-model="config.waf.enable" active-text="开启" inactive-text="关闭" />
            </el-form-item>

            <h4>默认拉黑方式</h4>
            <el-form-item label="拉黑动作">
              <el-radio-group v-model="config.waf.default_block_action">
                <el-radio label="ipset">IPSet (系统防火墙)</el-radio>
                <el-radio label="disconnect">断开连接</el-radio>
                <el-radio label="page">显示拦截页面</el-radio>
              </el-radio-group>
              <div class="tip">建议默认选非 IPSet 方式，配合自动 IPSet 切换使用。</div>
            </el-form-item>

            <h4>IPSet 自动切换</h4>
            <el-form-item label="自动启用 IPSet">
              <el-switch v-model="config.waf.auto_ipset_enable" />
            </el-form-item>
            <el-form-item label="触发阈值" v-if="config.waf.auto_ipset_enable">
              <el-input-number v-model="config.waf.auto_ipset_threshold" :min="1" />
              <span class="unit">次/秒</span>
              <div class="tip">当单站每秒拉黑次数超过阈值时，自动升级为 IPSet 拉黑。</div>
            </el-form-item>

            <h4>拉黑页面限制 (防刷)</h4>
            <el-form-item label="限制访问频率">
              <el-switch v-model="config.waf.block_page_rate_limit_enable" />
            </el-form-item>
            <el-form-item label="频率阈值" v-if="config.waf.block_page_rate_limit_enable">
              <el-input-number v-model="config.waf.block_page_rate_limit" :min="1" />
              <span class="unit">次/60秒</span>
              <div class="tip">单 IP 访问拉黑页面超过此频率，直接升级 IPSet 拉黑。</div>
            </el-form-item>
            <el-form-item label="拉黑页不计流量">
              <el-switch v-model="config.waf.block_page_traffic_free" />
            </el-form-item>

            <h4>封禁与白名单时长</h4>
            <el-form-item label="黑名单封禁时长">
              <el-input-number v-model="config.waf.blacklist_timeout" />
              <span class="unit">秒</span>
            </el-form-item>
            <el-form-item label="临时白名单时长">
              <el-input-number v-model="config.waf.temp_whitelist_timeout" />
              <span class="unit">秒</span>
            </el-form-item>

            <h4>临时白名单自动加入条件(5秒内)</h4>
            <el-form-item label="总请求数限制">
              <el-input-number v-model="config.waf.temp_whitelist_limit_total" />
            </el-form-item>
            <el-form-item label="同URL请求限制">
              <el-input-number v-model="config.waf.temp_whitelist_limit_url" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="安全控制 & 名单">
          <el-form label-width="180px">
            <h4>黑白名单 (一行一个，支持 CIDR)</h4>
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="白名单 IP">
                  <el-input type="textarea" v-model="config.waf.whitelist_ips" :rows="10" placeholder="192.168.1.10&#10;10.0.0.0/24" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="黑名单 IP">
                  <el-input type="textarea" v-model="config.waf.blacklist_ips" :rows="10" placeholder="1.2.3.4&#10;5.0.0.0/8" />
                </el-form-item>
              </el-col>
            </el-row>

            <h4>系统安全</h4>
            <el-form-item label="防止 TLS 握手攻击">
              <el-switch v-model="config.waf.prevent_tls_handshake" />
            </el-form-item>
            <el-form-item label="禁止未绑定域名访问">
              <el-switch v-model="config.waf.block_unbound_domain" />
              <span class="tip">禁止直接通过节点 IP 访问。</span>
            </el-form-item>
            <el-form-item label="禁止 PING">
              <el-switch v-model="config.waf.disable_ping" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="CC 防护 & 验证">
          <el-form label-width="180px">
            <h4>默认页防护</h4>
            <el-form-item label="开启模式">
              <el-radio-group v-model="config.waf.default_page_protection">
                <el-radio label="force">强制开启</el-radio>
                <el-radio label="auto">自动开启</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="自动开启阈值" v-if="config.waf.default_page_protection === 'auto'">
              <el-input-number v-model="config.waf.default_page_protection_threshold" />
              <span class="unit">请求/秒</span>
            </el-form-item>

            <h4>反 CC 页面设置</h4>
            <el-form-item label="验证方式">
              <el-select v-model="config.waf.anti_cc_type">
                <el-option label="滑动验证" value="slide" />
                <el-option label="点击验证" value="click" />
                <el-option label="5秒盾 (自动跳转)" value="5s" />
                <el-option label="图片旋转" value="rotate" />
                <el-option label="简单滑动" value="slide_simple" />
              </el-select>
            </el-form-item>
            <el-form-item label="验证图片来源">
              <el-radio-group v-model="config.waf.anti_cc_image_source">
                <el-radio label="system">系统默认</el-radio>
                <el-radio label="custom">自定义 URL</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="图片 URL" v-if="config.waf.anti_cc_image_source === 'custom'">
              <el-input v-model="config.waf.anti_cc_image_custom_url" placeholder="http://..." />
            </el-form-item>
            <el-form-item label="开启调试日志">
              <el-switch v-model="config.waf.anti_cc_debug" />
            </el-form-item>

            <h4>规则设置</h4>
            <el-form-item label="CC 规则自动切换">
              <el-switch v-model="config.waf.cc_rule_auto_switch" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="高级防护 & 系统">
          <el-form label-width="180px">
            <h4>系统配置</h4>
            <el-form-item label="通讯密钥">
              <el-input v-model="config.waf.secret_key" show-password />
            </el-form-item>
            <el-form-item label="节点日志清理">
              <el-select v-model="config.waf.node_log_clean_strategy">
                <el-option label="不清理" value="none" />
                <el-option label="仅清理日志" value="log_only" />
                <el-option label="清理日志和缓存" value="log_cache" />
              </el-select>
              <span class="tip">当节点空间不足时触发。</span>
            </el-form-item>

            <h4>特殊路径防护 (.well-known)</h4>
            <el-form-item label="404 阈值">
              <el-input-number v-model="config.waf.well_known_protection_threshold" />
              <span class="unit">次/60秒</span>
              <div class="tip">超过阈值 IP 将不再回源 300 秒（防止 ACME 验证泛滥）。</div>
            </el-form-item>

            <h4>内置资源防护 (防CC图片/JS)</h4>
            <el-form-item label="开启防护">
              <el-switch v-model="config.waf.resource_protection_enable" />
            </el-form-item>
            <el-form-item label="开启阈值">
              <el-input-number v-model="config.waf.resource_protection_threshold" />
              <span class="unit">QPS</span>
            </el-form-item>
            <el-form-item label="拉黑时长">
              <el-input-number v-model="config.waf.resource_protection_block_timeout" />
              <span class="unit">秒</span>
            </el-form-item>

            <el-form-item label="限流规则">
              <el-table :data="config.waf.resource_protection_rules" border style="width: 100%">
                <el-table-column label="统计时长(秒)" width="150">
                  <template #default="{ row }">
                    <el-input-number v-model="row.duration" size="small" :controls="false" />
                  </template>
                </el-table-column>
                <el-table-column label="最大请求数" width="150">
                  <template #default="{ row }">
                    <el-input-number v-model="row.max_requests" size="small" :controls="false" />
                  </template>
                </el-table-column>
                <el-table-column label="操作">
                  <template #default="{ $index }">
                    <el-button size="small" type="danger" link @click="config.waf.resource_protection_rules.splice($index, 1)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <el-button size="small" style="margin-top:5px" @click="config.waf.resource_protection_rules.push({ duration: 60, max_requests: 100 })">+ 添加规则</el-button>
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
const defaultWaf = {
  enable: false,
  default_block_action: 'disconnect',
  auto_ipset_enable: false,
  auto_ipset_threshold: 60,
  block_page_rate_limit_enable: false,
  block_page_rate_limit: 60,
  block_page_traffic_free: false,
  blacklist_timeout: 3600,
  temp_whitelist_timeout: 300,
  temp_whitelist_limit_total: 20,
  temp_whitelist_limit_url: 10,
  whitelist_ips: '',
  blacklist_ips: '',
  prevent_tls_handshake: false,
  block_unbound_domain: false,
  disable_ping: false,
  default_page_protection: 'auto',
  default_page_protection_threshold: 200,
  anti_cc_type: 'slide',
  anti_cc_image_source: 'system',
  anti_cc_image_custom_url: '',
  anti_cc_debug: false,
  cc_rule_auto_switch: false,
  secret_key: '',
  node_log_clean_strategy: 'none',
  well_known_protection_threshold: 60,
  resource_protection_enable: false,
  resource_protection_threshold: 200,
  resource_protection_block_timeout: 300,
  resource_protection_rules: []
}

const config = ref({
  waf: { ...defaultWaf }
})

const loadConfig = () => {
  loading.value = true
  request.get('/global_config').then(res => {
    if (res.code === 0) {
      config.value = res.data || {}
      if (!config.value.waf) {
        config.value.waf = { ...defaultWaf }
      } else {
        config.value.waf = { ...defaultWaf, ...config.value.waf }
      }
      if (!Array.isArray(config.value.waf.resource_protection_rules)) {
        config.value.waf.resource_protection_rules = []
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
      ElMessage.success('WAF 配置已保存')
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
  margin-top: 5px;
  line-height: 1.4;
}
.unit {
  margin-left: 10px;
}
h4 {
  margin: 20px 0 10px;
  padding-left: 10px;
  border-left: 4px solid #409eff;
  font-size: 14px;
}
</style>
