<template>
  <el-form label-width="120px" @focusin="cacheInputValue">
    <el-card shadow="never" class="mb-20">
      <template #header>基本信息</template>
      <el-form-item label="系统名称">
         <el-input v-model="systemInfo.sys_name" placeholder="CDN 4.0" @blur="handleBlurSave" />
      </el-form-item>

      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="Favicon">
             <el-upload
               class="avatar-uploader"
               :action="uploadUrl"
               :headers="uploadHeaders"
               :show-file-list="false"
               :on-success="(res) => handleUploadSuccess(res, 'favicon_file')"
               accept=".ico,.png"
             >
               <img v-if="systemInfo.favicon_file" :src="systemInfo.favicon_file" class="avatar" style="width: 32px; height: 32px; object-fit: contain;" />
               <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
             </el-upload>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="Logo">
             <el-upload
               class="avatar-uploader"
               :action="uploadUrl"
               :headers="uploadHeaders"
               :show-file-list="false"
               :on-success="(res) => handleUploadSuccess(res, 'logo_file')"
               accept=".png,.jpg,.jpeg,.svg"
             >
               <img v-if="systemInfo.logo_file" :src="systemInfo.logo_file" class="avatar" style="height: 40px; object-fit: contain;" />
               <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
             </el-upload>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="登录页广告">
             <el-upload
               class="avatar-uploader"
               :action="uploadUrl"
               :headers="uploadHeaders"
               :show-file-list="false"
               :on-success="(res) => handleUploadSuccess(res, 'login_ad_file')"
               accept=".jpg,.jpeg,.png"
             >
               <img v-if="systemInfo.login_ad_file" :src="systemInfo.login_ad_file" class="avatar" style="height: 60px; object-fit: contain;" />
               <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
             </el-upload>
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item label="普通用户标题">
         <el-input v-model="systemInfo.user_console_title" placeholder="CDN用户控制台" @blur="handleBlurSave" />
      </el-form-item>
      <el-form-item label="管理员标题">
         <el-input v-model="systemInfo.admin_console_title" placeholder="CDN管理员控制台" @blur="handleBlurSave" />
      </el-form-item>
      <el-form-item label="底部链接">
        <el-input type="textarea" :rows="3" v-model="systemInfo.footer_link" placeholder="名称|URL (换行分隔)" @blur="handleBlurSave" />
      </el-form-item>
      <el-form-item label="底部版权">
        <el-input type="textarea" :rows="2" v-model="systemInfo.footer_copyright" placeholder="" @blur="handleBlurSave" />
      </el-form-item>
      <el-form-item label="Master Host">
        <el-input v-model="bindMasterHost" placeholder="" @blur="handleBlurSave" />
        <div class="form-helper">绑定主节点Host，用于节点通信。</div>
      </el-form-item>
    </el-card>
  </el-form>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { cacheInputValue, shouldSkipBlurSave } from '@/utils/saveGuard'

const props = defineProps({
  configItems: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['saved'])

const uploadUrl = '/api/v1/admin/upload/image'
const uploadHeaders = {
    Authorization: 'Bearer ' + localStorage.getItem('admin_token')
}

const systemInfo = ref({
  sys_name: '',
  user_console_title: '',
  admin_console_title: '',
  footer_link: '',
  footer_copyright: '',
  favicon_file: '',
  logo_file: '',
  login_ad_file: ''
})
const bindMasterHost = ref('')

const handleUploadSuccess = (res, field) => {
    if (res.code === 0) {
        systemInfo.value[field] = res.url
        ElMessage.success('上传成功')
        handleSave()
    } else {
        ElMessage.error(res.msg || '上传失败')
    }
}

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
  
  // system_info: Merge with existing to preserve other fields (cleaning, user, etc.)
  let fullInfo = {}
  const infoItem = props.configItems.find(i => i.name === 'system_info')
  if (infoItem && infoItem.value) {
    try {
      fullInfo = JSON.parse(infoItem.value)
    } catch (e) { /* ignore */ }
  }
  
  items.push({
    name: 'system_info',
    value: JSON.stringify({ ...fullInfo, ...systemInfo.value })
  })

  // bind-master-host
  items.push({
    name: 'bind-master-host',
    value: bindMasterHost.value
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

const handleSave = () => {
  queueSave()
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
.form-helper { color: #999; font-size: 12px; margin-top: 5px; }

.avatar-uploader :deep(.el-upload) {
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: var(--el-transition-duration-fast);
}
.avatar-uploader :deep(.el-upload:hover) {
  border-color: var(--el-color-primary);
}
.avatar-uploader-icon {
  font-size: 28px;
  color: #8c939d;
  width: 100px;
  height: 100px;
  text-align: center;
  line-height: 100px;
  display: flex;
  justify-content: center;
  align-items: center;
}
.avatar {
  display: block;
}
</style>
