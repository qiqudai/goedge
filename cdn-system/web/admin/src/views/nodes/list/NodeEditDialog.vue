<template>
  <el-dialog v-model="visible" :title="item.id ? NODE_T.editNode : NODE_T.createNode" width="600px">
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane :label="NODE_T.basicSettings" name="basic">
        <el-form :model="form" label-width="100px" style="margin-top: 20px;">
          <el-form-item :label="NODE_T.name"><el-input v-model="form.name" /></el-form-item>
          <el-form-item :label="NODE_T.region">
            <el-select
              v-model="form.region_id"
              :disabled="regionLocked"
              :placeholder="NODE_T.regionPlaceholder"
              clearable
              style="width: 100%;"
            >
              <el-option v-for="region in regions" :key="region.id" :label="region.name" :value="region.id" />
            </el-select>
            <div v-if="regionLocked" class="form-helper">{{ NODE_T.regionLockedHint }}</div>
            <div v-else-if="!regions.length" class="form-helper">{{ NODE_T.regionEmptyHint }}</div>
          </el-form-item>
          <el-form-item :label="NODE_T.remark"><el-input v-model="form.remark" type="textarea" /></el-form-item>
          <el-form-item :label="NODE_T.sort"><el-input v-model.number="form.sort_order" /></el-form-item>
          <el-form-item label="IP"><el-input v-model="form.ip" /></el-form-item>
          <el-form-item :label="NODE_T.nodeType">
            <el-radio-group v-model="form.type">
              <el-radio :value="1">{{ NODE_T.l1EdgeNode }}</el-radio>
              <el-radio :value="2">{{ NODE_T.l2MiddleNode }}</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-divider content-position="left">{{ NODE_T.sshSettings }}</el-divider>
          <el-form-item :label="NODE_T.sshPort"><el-input v-model.number="form.ssh_port" /></el-form-item>
          <el-form-item :label="NODE_T.sshUser"><el-input v-model="form.ssh_user" :placeholder="NODE_T.sshUserPlaceholder" /></el-form-item>
          <el-form-item :label="NODE_T.sshAuthType">
            <el-radio-group v-model="form.ssh_auth_type">
              <el-radio value="password">{{ NODE_T.sshAuthPassword }}</el-radio>
              <el-radio value="key">{{ NODE_T.sshAuthKey }}</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.ssh_auth_type === 'password'" :label="NODE_T.sshPassword">
            <el-input v-model="form.ssh_password" type="password" show-password />
          </el-form-item>
          <el-form-item v-else :label="NODE_T.sshKey">
            <el-input v-model="form.ssh_key" type="textarea" :rows="4" :placeholder="NODE_T.sshKeyPlaceholder" />
          </el-form-item>
          <el-form-item :label="NODE_T.workDir">
            <el-input v-model="form.work_dir" :placeholder="NODE_T.workDirPlaceholder" disabled />
            <div class="form-helper">{{ NODE_T.workDirHint }}</div>
          </el-form-item>
          <el-form-item :label="NODE_T.autoInstall">
            <el-switch v-model="form.auto_install" />
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane :label="NODE_T.nodeSettings" name="settings">
        <el-form :model="form" label-width="100px" style="margin-top: 20px;">
          <el-form-item :label="NODE_T.cacheDir"><el-input v-model="form.cache_dir" /></el-form-item>
          <el-form-item :label="NODE_T.cacheLimit">
            <el-input v-model.number="form.cache_limit"><template #append>GB</template></el-input>
          </el-form-item>
          <el-form-item :label="NODE_T.logDir"><el-input v-model="form.log_dir" /></el-form-item>
          <el-form-item :label="NODE_T.bwLimit">
            <el-input v-model="form.bw_limit" placeholder="1000"><template #append>Mbps</template></el-input>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane :label="NODE_T.addSubIp" name="sub_ips">
        <el-form style="margin-top: 20px;">
          <el-form-item :label="NODE_T.subIp" label-width="80px">
             <el-input v-model="subIpsText" type="textarea" :rows="5" :placeholder="NODE_T.oneLineOneIp" />
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>
    <template #footer>
      <el-button @click="visible = false">{{ NODE_T.cancel }}</el-button>
      <el-button type="primary" @click="handleSubmit">{{ NODE_T.confirm }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, computed } from 'vue'
import { NODE_T } from './constants'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const INSTALL_TIMEOUT = 10000

const props = defineProps({
  item: { type: Object, default: () => ({ id: 0 }) },
  regions: { type: Array, default: () => [] },
  modelValue: Boolean
})
const emit = defineEmits(['update:modelValue', 'success'])

const visible = ref(false)
const activeTab = ref('basic')
const form = reactive({
  id: 0,
  name: '',
  region_id: 0,
  remark: '',
  sort_order: 100,
  ip: '',
  type: 1,
  cache_dir: '',
  cache_limit: 0,
  log_dir: '',
  bw_limit: '',
  ssh_host: '',
  ssh_port: 22,
  ssh_user: '',
  ssh_auth_type: 'password',
  ssh_password: '',
  ssh_key: '',
  work_dir: '/www/node',
  auto_install: true,
  sub_ips: []
})
const subIpsText = ref('')
const originalRegionId = ref(0)
const regionLocked = computed(() => Number(props.item?.line_count || 0) > 0)

const applyItem = (item) => {
  const nextRegionId = Number(item?.region_id || 0)
  Object.assign(form, { id: 0, name: '', region_id: 0, remark: '', sort_order: 100, ip: '', type: 1, cache_dir: '', cache_limit: 0, log_dir: '', bw_limit: '', ssh_host: '', ssh_port: 22, ssh_user: '', ssh_auth_type: 'password', ssh_password: '', ssh_key: '', work_dir: '/www/node', auto_install: true, sub_ips: [] }, item, {
    region_id: nextRegionId,
    work_dir: '/www/node'
  })
  if (!item?.id && !form.region_id && props.regions.length > 0) {
    form.region_id = props.regions[0].id
  }
  originalRegionId.value = Number(form.region_id || 0)
  subIpsText.value = (item?.sub_ips || []).map(i => i.ip || i).join('\n')
}

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    applyItem(props.item)
  }
})
watch(() => props.regions.length, () => {
  if (visible.value && !props.item?.id && !form.region_id && props.regions.length > 0) {
    form.region_id = props.regions[0].id
    originalRegionId.value = Number(form.region_id || 0)
  }
})
watch(visible, (val) => emit('update:modelValue', val))

const handleSubmit = async () => {
  if (regionLocked.value && Number(form.region_id || 0) !== originalRegionId.value) {
    ElMessage.warning(NODE_T.regionLockedHint)
    return
  }
  form.sub_ips = subIpsText.value.split('\n').filter(i => i.trim()).map(ip => ({ ip: ip.trim() }))
  let res
  if (form.id) {
    res = await request.put(`/nodes/${form.id}`, form)
  } else {
    const requestConfig = form.auto_install ? { timeout: INSTALL_TIMEOUT } : undefined
    res = await request.post('/nodes', form, requestConfig)
  }
  ElMessage.success(form.id ? NODE_T.updateSuccess : NODE_T.createSuccess)
  if (!form.id && res?.install_error) {
    ElMessage.warning(`${NODE_T.installFailed}: ${res.install_error}`)
  }
  visible.value = false
  emit('success')
}
</script>

