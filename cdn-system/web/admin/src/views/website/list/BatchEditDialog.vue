<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="500px"
    @closed="handleClosed"
    :close-on-click-modal="false"
  >
  <el-form :model="form" label-width="120px" ref="formRef">
      
      <!-- CNAME Domain Mode -->
      <template v-if="mode === 'cname-domain'">
        <el-form-item label="CNAME 根域名">
          <el-select
            v-model="form.cname_hostname"
            placeholder="请选择CNAME根域名"
            style="width: 100%"
            filterable
            :loading="cnameDomainsLoading"
            clearable
          >
            <el-option v-for="d in cnameDomains" :key="d.id || d.domain" :label="d.note ? `${d.domain}（${d.note}）` : d.domain" :value="d.domain" />
          </el-select>
        </el-form-item>
      </template>

      <!-- CNAME Generation Mode -->
      <template v-if="mode === 'cname-mode'">
        <el-form-item label="模式">
          <el-select v-model="form.cname_mode" style="width: 100%">
            <el-option label="按网站生成" value="custom" />
            <el-option label="按套餐生成" value="package" />
          </el-select>
        </el-form-item>
      </template>

      <!-- Node Group Mode -->
      <template v-if="mode === 'node-group'">
        <el-form-item label="区域">
          <el-select v-model="form.region_id" placeholder="请选择区域" style="width: 100%" clearable @change="handleRegionChange">
            <el-option v-for="r in regions" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="线路分组">
          <el-select v-model="form.node_group_id" placeholder="请选择线路分组" style="width: 100%" clearable>
            <el-option v-for="n in filteredNodeGroups" :key="n.id" :label="n.name" :value="n.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="备用线路组">
          <el-select v-model="form.backup_node_group_id" placeholder="请选择备用线路组" style="width: 100%" clearable>
             <el-option v-for="n in nodeGroups" :key="n.id" :label="n.name" :value="n.id" />
          </el-select>
        </el-form-item>
      </template>

    </el-form>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const props = defineProps({
  modelValue: Boolean,
  mode: String,
  ids: Array
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = ref(false)
const submitting = ref(false)
const formRef = ref(null)

const regions = ref([])
const nodeGroups = ref([])
const cnameDomains = ref([])
const cnameDomainsLoading = ref(false)

const form = reactive({
  cname_hostname: '',
  cname_mode: 'custom',
  region_id: '',
  node_group_id: '',
  backup_node_group_id: ''
})

const title = computed(() => {
  switch (props.mode) {
    case 'cname-domain': return '修改CNAME根域名'
    case 'cname-mode': return '修改CNAME生成模式'
    case 'node-group': return '修改线路分组'
    default: return '批量修改'
  }
})

const filteredNodeGroups = computed(() => {
  if (!form.region_id) return nodeGroups.value
  return nodeGroups.value.filter(n => n.region_id === form.region_id)
})

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    if (props.mode === 'node-group') {
      loadDependencies()
    }
    if (props.mode === 'cname-domain') {
      loadCnameDomains()
    }
  }
})

watch(() => visible.value, (val) => {
  emit('update:modelValue', val)
})

const handleClosed = () => {
    form.cname_hostname = ''
    form.cname_mode = 'custom'
    form.region_id = ''
    form.node_group_id = ''
    form.backup_node_group_id = ''
}

const loadDependencies = async () => {
  try {
    const [rRes, nRes] = await Promise.all([
      request.get('/regions', { params: { pageSize: 100 } }),
      request.get('/node-groups', { params: { pageSize: 100 } })
    ])
    regions.value = rRes.data?.list || rRes.list || []
    nodeGroups.value = nRes.data?.list || nRes.list || []
  } catch (e) {
    console.error(e)
  }
}

const loadCnameDomains = async () => {
  if (cnameDomainsLoading.value) return
  cnameDomainsLoading.value = true
  try {
    const res = await request.get('/cname_domains', { skipLoading: true })
    cnameDomains.value = res.data?.list || res.list || []
  } catch (e) {
    console.error(e)
  } finally {
    cnameDomainsLoading.value = false
  }
}

const handleRegionChange = () => {
  form.node_group_id = ''
}

const handleSubmit = async () => {
  if (!props.ids || props.ids.length === 0) {
    ElMessage.error('没有选择任何站点')
    return
  }
  submitting.value = true
  try {
    const payload = { ids: props.ids }
    
    if (props.mode === 'cname-domain') {
      if (!form.cname_hostname) {
        ElMessage.error('请选择CNAME根域名')
        return
      }
      payload.cname_hostname = form.cname_hostname
    } else if (props.mode === 'cname-mode') {
      payload.cname_mode = form.cname_mode
    } else if (props.mode === 'node-group') {
      payload.region_id = form.region_id ? Number(form.region_id) : 0
      payload.node_group_id = form.node_group_id ? Number(form.node_group_id) : 0
      payload.backup_node_group_id = form.backup_node_group_id ? Number(form.backup_node_group_id) : 0
      if (payload.backup_node_group_id > 0) {
          payload.enable_backup_group = true
      }
    }

    await request.post('/sites/batch_update', payload)
    ElMessage.success('批量修改成功')

    const successPayload = { mode: props.mode, ids: props.ids }
    if (props.mode === 'cname-domain') {
      successPayload.cname_hostname = form.cname_hostname
    } else if (props.mode === 'cname-mode') {
      successPayload.cname_mode = form.cname_mode
    } else if (props.mode === 'node-group') {
      successPayload.region_id = form.region_id
      successPayload.node_group_id = form.node_group_id
      successPayload.backup_node_group_id = form.backup_node_group_id
    }

    emit('success', successPayload)
    visible.value = false
  } catch (e) {
    console.error(e)
  } finally {
    submitting.value = false
  }
}
</script>
