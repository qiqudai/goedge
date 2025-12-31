<template>
  <div class="notify-config">
    <el-form label-width="140px">
      <!-- Global Time Setting -->
      <div class="section-top">
         <el-form-item label="通知时间段">
           <el-radio-group v-model="configs.notification_period">
             <el-radio label="all">全天</el-radio>
             <el-radio label="custom">自定义</el-radio>
           </el-radio-group>
           <el-input 
             v-if="configs.notification_period === 'custom'" 
             v-model="configs.notification_period_custom" 
             placeholder="8-22"
             style="width: 100px; margin-left: 10px;"
            />
         </el-form-item>
      </div>

      <!-- 1. Traffic Exceeded -->
      <notify-item-config
        v-model="configs.notify_traffic_exceed_info"
        title="流量已超限通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}']"
      />

      <!-- 2. Traffic Low -->
      <notify-item-config
        v-model="configs.notify_traffic_low_info"
        title="流量即将超限通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}', '{{traffic_remain}}']"
      >
        <template #extra>
           <el-form-item label="剩余流量不足 (GB)">
             <el-input-number v-model="configs.notify_traffic_low_info.remain_traffic" :min="1" controls-position="right" style="width: 100%" />
           </el-form-item>
        </template>
      </notify-item-config>

      <!-- 3. Package Expire -->
      <notify-item-config
        v-model="configs.notify_package_expire_info"
        title="套餐过期通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}']"
      />

      <!-- 4. Package Expiring (Soon) -->
      <notify-item-config
        v-model="configs.notify_package_expiring_info"
        title="套餐即将过期通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}', '{{remain_days}}']"
      >
        <template #extra>
          <el-form-item label="剩余时间不足 (天)">
             <el-input-number v-model="configs.notify_package_expiring_info.days" :min="1" controls-position="right" style="width: 100%" />
          </el-form-item>
        </template>
      </notify-item-config>

      <!-- 5. CC Switch -->
      <notify-item-config
        v-model="configs.notify_cc_switch_info"
        title="网站CC规则自动切换通知"
        :variables="['{{username}}', '{{domain}}', '{{curr_qps}}', '{{qps_limit}}', '{{rule_name}}']"
      />

      <!-- 6. Bandwidth Exceed -->
      <notify-item-config
        v-model="configs.notify_bandwidth_exceed_info"
        title="套餐带宽超限通知"
        :variables="['{{username}}', '{{package_id}}', '{{package_name}}']"
      />

      <!-- 7. Cert Expire -->
      <notify-item-config
        v-model="configs.notify_cert_expire_info"
        title="证书已过期通知"
        :variables="['{{username}}', '{{cert_id}}', '{{cert_name}}', '{{domain}}']"
      />

      <!-- 8. Cert Expiring -->
      <notify-item-config
        v-model="configs.notify_cert_expiring_info"
        title="证书即将过期通知"
        :variables="['{{username}}', '{{cert_id}}', '{{cert_name}}', '{{domain}}', '{{remain_days}}']"
      >
        <template #extra>
          <el-form-item label="剩余时间不足 (天)">
             <el-input-number v-model="configs.notify_cert_expiring_info.days" :min="1" controls-position="right" style="width: 100%" />
          </el-form-item>
        </template>
      </notify-item-config>

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
  notification_period: 'all',
  notification_period_custom: '8-22',
  notify_traffic_exceed_info: {},
  notify_traffic_low_info: {},
  notify_package_expire_info: {},
  notify_package_expiring_info: {},
  notify_cc_switch_info: {},
  notify_bandwidth_exceed_info: {},
  notify_cert_expire_info: {},
  notify_cert_expiring_info: {}
})

// Keys mapping for JSON configs
const jsonKeys = [
  'notify_traffic_exceed_info',
  'notify_traffic_low_info',
  'notify_package_expire_info',
  'notify_package_expiring_info',
  'notify_cc_switch_info',
  'notify_bandwidth_exceed_info',
  'notify_cert_expire_info',
  'notify_cert_expiring_info'
]

// Keys for simple string configs
const simpleKeys = ['notification_period', 'notification_period_custom']

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
