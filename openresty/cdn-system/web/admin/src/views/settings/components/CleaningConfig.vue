<template>
  <el-form label-width="180px">
    <el-card shadow="never" class="mb-20">
      <template #header>数据清理</template>
      <el-form-item label="清缓存/解锁IP记录">
        <el-input v-model.number="form.clean_cache_days"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="登录记录">
        <el-input v-model.number="form.clean_login_log_days"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="操作记录">
        <el-input v-model.number="form.clean_op_log_days"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="网站访问日志(ES)">
        <el-input v-model.number="form.clean_site_log_days"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="节点监控数据">
        <el-input v-model.number="form.clean_node_monitor_days"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="流量带宽历史">
        <el-input v-model.number="form.clean_traffic_days"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="节点流量历史">
        <el-input v-model.number="form.clean_node_traffic_days"><template #append>天</template></el-input>
      </el-form-item>

      <el-form-item label="IP黑名单清理">
        <el-input v-model.number="form.clean_blacklist_days"><template #append>天</template></el-input>
      </el-form-item>

      <el-divider>数据备份</el-divider>
      <el-form-item label="备份频率">
        <el-input v-model.number="form.backup_frequency"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="保留天数">
        <el-input v-model.number="form.backup_retention"><template #append>天</template></el-input>
      </el-form-item>
      <el-form-item label="备份目录">
        <el-input v-model="form.backup_dir" />
        <div class="mt-10 text-gray-500">
          推荐使用阿里云备份本地数据库文件, <a href="https://help.aliyun.com/document_detail/461008.html" target="_blank" class="text-blue-500">查看文档</a>
        </div>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="save">保存</el-button>
      </el-form-item>
    </el-card>
  </el-form>
</template>

<script setup>
import { ref, watch } from 'vue'
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
  const infoItem = items.find(i => i.name === 'system_info')
  if (infoItem && infoItem.value) {
    try {
      const parsed = JSON.parse(infoItem.value)
      // Only merge parsing fields relevant to this component
      const keys = Object.keys(form.value)
      keys.forEach(k => {
        if (parsed[k] !== undefined) form.value[k] = parsed[k]
      })
    } catch (e) { /* ignore */ }
  }
}, { immediate: true, deep: true })

const save = () => {
  let fullInfo = {}
  const infoItem = props.configItems.find(i => i.name === 'system_info')
  if (infoItem && infoItem.value) {
    try {
      fullInfo = JSON.parse(infoItem.value)
    } catch (e) { /* ignore */ }
  }

  const items = []
  items.push({
    name: 'system_info',
    value: JSON.stringify({ ...fullInfo, ...form.value })
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
.mt-10 { margin-top: 10px; }
.text-gray-500 { color: #888; }
.text-blue-500 { color: #409eff; }
</style>
