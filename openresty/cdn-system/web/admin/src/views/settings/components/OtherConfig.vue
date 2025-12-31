<template>
  <el-form label-width="180px">
    <!-- Group 1: Master Source IP -->
    <el-card shadow="never" class="mb-20">
      <template #header>主控获取源IP</template>
      <el-form-item label="源IP请求头">
        <el-input v-model="form.master_client_ip_header" placeholder="X-Real-IP" />
      </el-form-item>
    </el-card>

    <!-- Group 2: Logging -->
    <el-card shadow="never" class="mb-20">
      <template #header>记录相关</template>
      <el-form-item label="记录定时修复">
        <el-radio-group v-model.number="form.record_repair_enable">
          <el-radio :label="0">关闭</el-radio>
          <el-radio :label="1">定时修复记录</el-radio>
          <el-radio :label="2">定时修复并删除多余记录</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="DNS记录保护">
        <el-input v-model="form.dns_rs_protect" placeholder="输入主机名,即域名的前面部分,多个以逗号分隔" />
      </el-form-item>
    </el-card>

    <!-- Group 3: Config Sync -->
    <el-card shadow="never" class="mb-20">
      <template #header>配置同步</template>
      <el-form-item label="单次同步站点最大个数">
        <el-input v-model.number="form.max_site_stream_sync_one_time" />
      </el-form-item>
      <el-form-item label="同步范围">
        <el-radio-group v-model="form.sync_site_config_scope">
          <el-radio label="region">按区域</el-radio>
          <el-radio label="group">按线路组</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-card>

    <!-- Group 4: Resource Limit -->
    <el-card shadow="never" class="mb-20">
      <template #header>资源限制</template>
      <el-form-item label="资源排行显示的数量">
        <el-input v-model.number="form.res_rank_size" />
      </el-form-item>
    </el-card>

    <!-- Group 5: HTTP Proxy -->
    <el-card shadow="never" class="mb-20">
      <template #header>http代理设置</template>
      <el-form-item label="http代理">
        <el-input v-model="form.http_proxy" placeholder="格式为 http://用户名:密码@代理ip:代理端口" />
        <div class="form-helper">当设置时，用户添加的dns api使用代理连接，系统提供的免费代理为:http://cdn:6d0d3e31@proxy.lotcdn.com:8888</div>
      </el-form-item>
    </el-card>
    
    <!-- Group 6: API Key -->
    <el-card shadow="never" class="mb-20">
      <template #header>API密钥</template>
        <el-form-item label="密钥状态">
           <el-switch v-model="form.api_key_status" active-value="1" inactive-value="0" @change="handleApiKeyStatusChange" />
        </el-form-item>
        
        <div v-if="form.api_key_status === '1'" class="pl-20">
           <el-form-item label="api_key">
             <span>{{ apiKeyInfo.api_key }}</span>
             <el-button link type="primary" class="ml-10" @click="copy(apiKeyInfo.api_key)"><el-icon><CopyDocument /></el-icon></el-button>
           </el-form-item>
           <el-form-item label="api_secret">
             <span>{{ apiKeyInfo.api_secret }}</span>
             <el-button link type="primary" class="ml-10" @click="copy(apiKeyInfo.api_secret)"><el-icon><CopyDocument /></el-icon></el-button>
           </el-form-item>
           <el-form-item label="IP白名单">
             <el-input v-model="apiKeyInfo.api_ip" placeholder="多个IP以|分隔 (例如: 1.2.3.4|5.6.7.8)" />
             <div class="form-helper">只允许指定IP访问API，留空表示不限制</div>
           </el-form-item>
           <el-form-item>
             <el-button type="danger" plain size="small" @click="resetSecret">重置密钥</el-button>
           </el-form-item>
        </div>
    </el-card>

    <!-- Group 7: Traffic Calculation -->
    <el-card shadow="never" class="mb-20">
      <template #header>流量计算</template>
      <el-form-item label="TCP系数">
        <el-radio-group v-model.number="form.tcp_traffic_factor">
          <el-radio :label="1.0">1.0</el-radio>
          <el-radio :label="1.1">1.1</el-radio>
        </el-radio-group>
        <div class="form-helper mt-10 text-gray-500 line-height-1.5">
          由于TCP/IP包头和TCP重传等原因，实际节点流量消耗要比从Nginx日志文件里统计的要大，所以提供一个系数选择。<br>
          当tcp系数为1.0时，只统计应用层流量，不统计TCP头的消耗，此时统计到的流量要比节点实际的流量消耗要小；<br>
          当tcp系数为1.1时，在应用层流量的基础上，再增加10%的流量消耗得出最终计费流量。这样最接近节点实际的流量消耗。
        </div>
      </el-form-item>
    </el-card>

    <!-- Group 8: Default HTTPS Certificate -->
    <el-card shadow="never" class="mb-20">
      <template #header>默认HTTPS证书</template>
      <el-alert title="此证书用于节点未匹配到站点证书时的默认HTTPS响应" type="info" :closable="false" class="mb-10"/>
      <el-form-item label="证书内容 (PEM)">
        <el-input v-model="form.cert_content" type="textarea" :rows="6" placeholder="-----BEGIN CERTIFICATE-----..." />
      </el-form-item>
      <el-form-item label="私钥内容 (PEM)">
        <el-input v-model="form.key_content" type="textarea" :rows="6" placeholder="-----BEGIN PRIVATE KEY-----..." />
      </el-form-item>
      <el-alert title="保存后需要重启Master服务生效" type="warning" show-icon class="mt-10" />
    </el-card>

    <el-form-item>
      <el-button type="primary" @click="save">保存</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument } from '@element-plus/icons-vue'

const props = defineProps({
  configItems: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['saved'])

// Key Mapping
const keyMap = {
  master_client_ip_header: 'master_client_ip_header',
  record_repair_enable: 'record-repair-enable',
  dns_rs_protect: 'dns_rs_protect',
  max_site_stream_sync_one_time: 'max_site_stream_sync_one_time',
  sync_site_config_scope: 'sync-site-config-scope',
  res_rank_size: 'res_rank_size',
  http_proxy: 'http_proxy',
  api_key_status: 'api_key_status',
  tcp_traffic_factor: 'tcp_traffic_factor',
  cert_content: 'https_cert',
  key_content: 'https_key',
  
  // New fields
  package_expire_close_site: 'package_expire_close_site',
  traffic_excceed_close_site: 'traffic_excceed_close_site', // note double 'c' in dump "excceed"
  package_allow_upgrade: 'package_allow_upgrade',
  package_allow_downgrade: 'package_allow_downgrade',
  node_health_check: 'node_health_check',
  node_max_failed: 'node_max_failed',
  auto_upgrade_agent: 'auto_upgrade_agent',
  delete_config_delayed: 'delete_config_delayed'
}

const form = ref({
  master_client_ip_header: 'X-Real-IP',
  record_repair_enable: 0,
  dns_rs_protect: '',
  max_site_stream_sync_one_time: 1000,
  sync_site_config_scope: 'group',
  res_rank_size: 100,
  http_proxy: '',
  api_key_status: '0',
  tcp_traffic_factor: 1.1,
  cert_content: '',
  key_content: '',
  
  package_expire_close_site: '1',
  traffic_excceed_close_site: '0',
  package_allow_upgrade: '0',
  package_allow_downgrade: '0',
  node_health_check: '1',
  node_max_failed: 2,
  auto_upgrade_agent: '0',
  delete_config_delayed: ''
})

const apiKeyInfo = ref({
  api_key: '',
  api_secret: '',
  api_ip: ''
})

watch(() => props.configItems, (items) => {
  if (!items) return

  const reverseMap = {}
  Object.keys(keyMap).forEach(k => reverseMap[keyMap[k]] = k)

  items.forEach(item => {
    const modelKey = reverseMap[item.name]
    if (modelKey) {
       let val = item.value
       
       // Handling Numerics
       if (['max_site_stream_sync_one_time', 'res_rank_size', 'record_repair_enable', 'node_max_failed'].includes(modelKey)) {
           val = parseInt(val) || 0
       } else if (modelKey === 'tcp_traffic_factor') {
           val = parseFloat(val) || 1.1
       }
       
       form.value[modelKey] = val
       
       // If API Key Status is '1', fetch key info
       if (modelKey === 'api_key_status' && val === '1') {
           fetchApiKey()
       }
    }
  })
}, { immediate: true, deep: true })

const fetchApiKey = () => {
    request.get('/api_key').then(res => {
        if(res.code === 0 && res.data) {
            apiKeyInfo.value = res.data
        }
    })
}

const handleApiKeyStatusChange = (val) => {
    if (val === '1') {
        fetchApiKey()
    }
}

const resetSecret = () => {
    ElMessageBox.confirm('确定要重置密钥吗？旧的密钥将即刻失效。', '警告', {
        confirmButtonText: '确定重置',
        cancelButtonText: '取消',
        type: 'warning'
    }).then(() => {
        request.post('/api_key/reset').then(res => {
             if(res.code === 0 && res.data) {
                 apiKeyInfo.value.api_secret = res.data.api_secret
                 ElMessage.success('密钥已重置')
             }
        })
    })
}

const copy = (text) => {
    if (!text) return
    navigator.clipboard.writeText(text).then(() => {
        ElMessage.success('已复制')
    }).catch(() => {
        ElMessage.error('复制失败')
    })
}

const save = async () => {
  const items = []
  Object.keys(form.value).forEach(modelKey => {
    const backendKey = keyMap[modelKey]
    if (backendKey) {
        items.push({
            name: backendKey,
            value: String(form.value[modelKey])
        })
    }
  })

  // 1. Save Config Items (including api_key_status)
  try {
      await request.post('/config_items', {
        type: 'system',
        scope_name: 'global',
        items: items
      })
      
      // 2. Save API Key IP whitelist if status is enabled
      if (form.value.api_key_status === '1') {
          await request.put('/api_key', {
              api_ip: apiKeyInfo.value.api_ip
          })
      }
      
      ElMessage.success('保存成功')
      emit('saved')
  } catch(e) {
      ElMessage.error('保存失败: ' + (e.msg || '未知错误'))
  }
}
</script>

<style scoped>
.mb-20 { margin-bottom: 20px; }
.mt-10 { margin-top: 10px; }
.ml-10 { margin-left: 10px; }
.pl-20 { padding-left: 20px; }
.text-gray-500 { color: #888; }
.line-height-1\.5 { line-height: 1.5; }
.form-helper {
    font-size: 12px;
    color: #909399;
    margin-top: 5px;
}
</style>
