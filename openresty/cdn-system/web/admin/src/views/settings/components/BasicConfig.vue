<template>
  <el-form label-width="120px">
    <el-card shadow="never" class="mb-20">
      <template #header>基本信息</template>
      <el-form-item label="系统名称">
        <el-input v-model="systemInfo.sys_name" placeholder="CDN 4.0" />
      </el-form-item>
      <el-form-item label="普通用户标题">
         <el-input v-model="systemInfo.user_console_title" placeholder="CDN用户控制台" />
      </el-form-item>
      <el-form-item label="管理员标题">
         <el-input v-model="systemInfo.admin_console_title" placeholder="CDN管理员控制台" />
      </el-form-item>
      <el-form-item label="底部链接">
        <el-input type="textarea" :rows="3" v-model="systemInfo.footer_link" placeholder="名称|URL (换行分隔)" />
      </el-form-item>
      <el-form-item label="底部版权">
        <el-input type="textarea" :rows="2" v-model="systemInfo.footer_copyright" placeholder="" />
      </el-form-item>
      <el-form-item label="Master Host">
        <el-input v-model="bindMasterHost" placeholder="" />
        <div class="form-helper">绑定主节点Host，用于节点通信。</div>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="save">保存</el-button>
      </el-form-item>
    </el-card>
  </el-form>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const props = defineProps({
  configItems: {
    type: Array,
    default: () => []
  }
})

const systemInfo = ref({
  sys_name: '',
  user_console_title: '',
  admin_console_title: '',
  footer_link: '',
  footer_copyright: ''
})
const bindMasterHost = ref('')

// Initialize data from props
watch(() => props.configItems, (items) => {
  if (!items || items.length === 0) return

  // system_info
  const infoItem = items.find(i => i.name === 'system_info')
  if (infoItem && infoItem.value) {
    try {
      const parsed = JSON.parse(infoItem.value)
      systemInfo.value = { ...systemInfo.value, ...parsed }
    } catch (e) {
      console.error('Failed to parse system_info', e)
    }
  }

  // bind-master-host
  const hostItem = items.find(i => i.name === 'bind-master-host')
  if (hostItem) {
    bindMasterHost.value = hostItem.value
  }
}, { immediate: true, deep: true })

const save = () => {
  const items = []
  
  // system_info
  items.push({
    name: 'system_info',
    value: JSON.stringify(systemInfo.value)
  })

  // bind-master-host
  items.push({
    name: 'bind-master-host',
    value: bindMasterHost.value
  })

  request.post('/config_items', {
    type: 'system',
    scope_name: 'global',
    items: items
  }).then(() => {
    ElMessage.success('保存成功')
  })
}
</script>

<style scoped>
.mb-20 { margin-bottom: 20px; }
.form-helper { color: #999; font-size: 12px; margin-top: 5px; }
</style>
