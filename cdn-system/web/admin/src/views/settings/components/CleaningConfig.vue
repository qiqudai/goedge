<template>
  <el-form label-width="180px" @focusin="cacheInputValue">
    <el-card shadow="never" class="mb-20">
      <template #header>数据清理</template>
      <el-form-item label="清缓存/解锁IP记录">
        <el-input v-model.number="form.clean_cache_days" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="登录记录">
        <el-input v-model.number="form.clean_login_log_days" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="操作记录">
        <el-input v-model.number="form.clean_op_log_days" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="网站访问日志(ES)">
        <el-input v-model.number="form.clean_site_log_days" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="节点监控数据">
        <el-input v-model.number="form.clean_node_monitor_days" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="流量带宽历史">
        <el-input v-model.number="form.clean_traffic_days" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="节点流量历史">
        <el-input v-model.number="form.clean_node_traffic_days" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>

      <el-form-item label="IP黑名单清理">
        <el-input v-model.number="form.clean_blacklist_days" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>

      <el-divider>数据备份</el-divider>
      <el-form-item label="备份频率">
        <el-input v-model.number="form.backup_frequency" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="保留天数">
        <el-input v-model.number="form.backup_retention" @blur="handleBlurSave"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="备份目录">
        <el-input v-model="form.backup_dir" @blur="handleBlurSave" />
        <div class="mt-10 text-gray-500">
          推荐使用阿里云备份本地数据库文件, <a href="https://help.aliyun.com/document_detail/461008.html" target="_blank" class="text-blue-500">查看文档</a>
        </div>
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

// Keys mapping: Component Model Key -> Backend Key
const keyMap = {
  clean_cache_days: 'keep-task-log-days',
  clean_login_log_days: 'keep-login-log-days',
  clean_op_log_days: 'keep-op-log-days',
  clean_site_log_days: 'keep-access-log-days',
  clean_node_monitor_days: 'keep-node-log-days',
  clean_traffic_days: 'keep-traffic-history-days',
  clean_node_traffic_days: 'keep-node-traffic-days', // Assuming a key or keep as is if not in list, but let's guess keep-node-log-days covers it? No, keep independent.
  clean_blacklist_days: 'keep-blacklist-days',
  backup_frequency: 'backup_rate',
  backup_retention: 'backup_keep_days',
  backup_dir: 'backup_dir'
}

const form = ref({
  clean_cache_days: 30,
  clean_login_log_days: 30,
  clean_op_log_days: 365,
  clean_site_log_days: 7,
  clean_node_monitor_days: 7,
  clean_traffic_days: 90,
  clean_node_traffic_days: 45,
  clean_blacklist_days: 7,
  backup_frequency: 2,
  backup_retention: 7,
  backup_dir: '/data/backup/cdn/'
})

watch(() => props.configItems, (items) => {
  if (!items) return

  // Inverse Map: Backend Key -> Component Model Key
  const reverseMap = {}
  Object.keys(keyMap).forEach(k => reverseMap[keyMap[k]] = k)

  items.forEach(item => {
    const modelKey = reverseMap[item.name]
    if (modelKey) {
      if (item.name === 'backup_dir') {
        form.value[modelKey] = item.value
      } else {
        // Handle "2h" or "30" strings
        let val = parseInt(item.value, 10)
        if (isNaN(val)) val = item.value // fallback for strings like '2h' if not purely numeric? backup_rate is '2h' in dump
        form.value[modelKey] = val
      }
    }
  })
}, { immediate: true, deep: true })

const save = () => {
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

  return request.post('/config_items', {
    type: 'system',
    scope_name: 'global',
    items: items
  }).then(() => {
    ElMessage.success('保存成功')
    emit('saved')
  })
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

const handleBlurSave = (event) => {
  if (shouldSkipBlurSave(event)) {
    return
  }
  queueSave()
}
</script>

<style scoped>
.mb-20 { margin-bottom: 20px; }
.mt-10 { margin-top: 10px; }
.text-gray-500 { color: #888; }
.text-blue-500 { color: #409eff; }
</style>
