<template>
  <div class="notify-config">
    <el-form label-width="140px">
      <!-- Global Time Setting -->
      <div class="section-top">
         <el-form-item label="通知时间段">
           <el-radio-group v-model="configs['notification-period']">
             <el-radio label="all">全天</el-radio>
             <el-radio label="custom">自定义</el-radio>
           </el-radio-group>
           <el-input 
             v-if="configs['notification-period'] === 'custom'" 
             v-model="configs['notification-period-custom']" 
             placeholder="8-22"
             style="width: 100px; margin-left: 10px;"
            />
         </el-form-item>
      </div>

      <!-- 1. Traffic Exceeded -->
      <notify-item-config
        v-model="configs['traffic-exceed-notify']"
        title="流量已超限通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}']"
      />

      <!-- 2. Traffic Low -->
      <notify-item-config
        v-model="configs['traffic-exceeding-notify']"
        title="流量即将超限通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}', '{{traffic_remain}}']"
      >
        <template #extra>
           <el-form-item label="剩余流量不足 (GB)">
             <el-input-number v-model="configs['traffic-exceeding-notify'].remain_traffic" :min="1" controls-position="right" style="width: 100%" />
           </el-form-item>
        </template>
      </notify-item-config>

      <!-- 3. Package Expire -->
      <notify-item-config
        v-model="configs['package-expire-notify']"
        title="套餐过期通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}']"
      />

      <!-- 4. Package Expiring (Soon) -->
      <notify-item-config
        v-model="configs['package-expiring-notify']"
        title="套餐即将过期通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}', '{{remain_days}}']"
      >
        <template #extra>
          <el-form-item label="剩余时间不足 (天)">
             <el-input-number v-model="configs['package-expiring-notify'].days" :min="1" controls-position="right" style="width: 100%" />
          </el-form-item>
        </template>
      </notify-item-config>

      <!-- 5. CC Switch -->
      <notify-item-config
        v-model="configs['cc-switch-notify']"
        title="网站CC规则自动切换通知"
        :variables="['{{username}}', '{{domain}}', '{{curr_qps}}', '{{qps_limit}}', '{{rule_name}}']"
      />

      <!-- 6. Bandwidth Exceed -->
      <notify-item-config
        v-model="configs['bandwidth-exceed-notify']"
        title="套餐带宽超限通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}']"
      />

      <!-- 7. Conn Exceed (New) -->
      <notify-item-config
        v-model="configs['conn-exceed-notify']"
        title="连接数超限通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}']"
      />

      <!-- 8. Cert Expire -->
      <notify-item-config
        v-model="configs['cert-expire-notify']"
        title="证书已过期通知"
        :variables="['{{username}}', '{{cert_id}}', '{{cert_name}}', '{{domain}}']"
      />

      <!-- 9. Cert Expiring -->
      <notify-item-config
        v-model="configs['cert-expiring-notify']"
        title="证书即将过期通知"
        :variables="['{{username}}', '{{cert_id}}', '{{cert_name}}', '{{domain}}', '{{remain_days}}']"
      >
        <template #extra>
          <el-form-item label="剩余时间不足 (天)">
             <el-input-number v-model="configs['cert-expiring-notify'].days" :min="1" controls-position="right" style="width: 100%" />
          </el-form-item>
        </template>
      </notify-item-config>

      <!-- 10. Account Auth2 (New) -->
      <notify-item-config
        v-model="configs['account-auth2-notify']"
        title="二次验证通知"
        :variables="['{{username}}', '{{ip}}', '{{time}}', '{{code}}']"
      />

      <el-form-item label-width="140px">
        <el-button type="primary" @click="save">保存所有配置</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import NotifyItemConfig from './NotifyItemConfig.vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const props = defineProps({
  configItems: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['saved'])

// We use a local reactive object to store the parsed JSON objects
const configs = ref({
  'notification-period': 'all',
  'notification-period-custom': '8-22',
  'traffic-exceed-notify': {},
  'traffic-exceeding-notify': {},
  'package-expire-notify': {},
  'package-expiring-notify': {},
  'cc-switch-notify': {},
  'bandwidth-exceed-notify': {},
  'cert-expire-notify': {},
  'cert-expiring-notify': {},
  'conn-exceed-notify': {},
  'account-auth2-notify': {}
})

// Keys mapping for JSON configs
const jsonKeys = [
  'traffic-exceed-notify',
  'traffic-exceeding-notify',
  'package-expire-notify',
  'package-expiring-notify',
  'cc-switch-notify',
  'bandwidth-exceed-notify',
  'cert-expire-notify',
  'cert-expiring-notify',
  'conn-exceed-notify',
  'account-auth2-notify'
]

// Keys for simple string configs
const simpleKeys = ['notification-period', 'notification-period-custom']

// Helper to safely parse JSON
const parseJSON = (str) => {
  try {
    return str ? JSON.parse(str) : {}
  } catch (e) {
    return {}
  }
}

// Watch for props change to initialize
watch(() => props.configItems, (items) => {
  if (!items) return

  jsonKeys.forEach(key => {
    const item = items.find(i => i.name === key)
    if (item && item.value) {
      configs.value[key] = parseJSON(item.value)
    }
  })
  
  simpleKeys.forEach(key => {
    const item = items.find(i => i.name === key)
    if (item) configs.value[key] = item.value
  })

}, { immediate: true, deep: true })

const save = () => {
  const items = []
  
  jsonKeys.forEach(key => {
    items.push({
      name: key,
      value: JSON.stringify(configs.value[key])
    })
  })
  
  simpleKeys.forEach(key => {
      items.push({
          name: key,
          value: configs.value[key]
      })
  })

  // We are saving these as individual keys in 'system' type 'global' scope
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
.notify-config {
  padding: 10px;
}
.section-top {
    border-bottom: 1px solid #ebeef5;
    margin-bottom: 20px;
    padding-bottom: 20px;
}
</style>
