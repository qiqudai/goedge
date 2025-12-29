<template>
  <el-dialog v-model="visible" :title="item.id ? NODE_T.editNode : NODE_T.createNode" width="600px">
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane :label="NODE_T.basicSettings" name="basic">
        <el-form :model="form" label-width="100px" style="margin-top: 20px;">
          <el-form-item :label="NODE_T.name"><el-input v-model="form.name" /></el-form-item>
          <el-form-item :label="NODE_T.remark"><el-input v-model="form.remark" type="textarea" /></el-form-item>
          <el-form-item :label="NODE_T.sort"><el-input v-model.number="form.sort_order" /></el-form-item>
          <el-form-item label="IP"><el-input v-model="form.ip" /></el-form-item>
          <el-form-item :label="NODE_T.nodeType">
            <el-radio-group v-model="form.type">
              <el-radio :value="1">{{ NODE_T.l1EdgeNode }}</el-radio>
              <el-radio :value="2">{{ NODE_T.l2MiddleNode }}</el-radio>
            </el-radio-group>
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
import { ref, reactive, watch } from 'vue'
import { NODE_T } from './constants'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const props = defineProps({
  item: { type: Object, default: () => ({ id: 0 }) },
  modelValue: Boolean
})
const emit = defineEmits(['update:modelValue', 'success'])

const visible = ref(false)
const activeTab = ref('basic')
const form = reactive({ id: 0, name: '', remark: '', sort_order: 100, ip: '', type: 1, cache_dir: '', cache_limit: 0, log_dir: '', sub_ips: [] })
const subIpsText = ref('')

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    Object.assign(form, { ...props.item })
    subIpsText.value = (props.item.sub_ips || []).map(i => i.ip || i).join('\n')
  }
})
watch(visible, (val) => emit('update:modelValue', val))

const handleSubmit = async () => {
    form.sub_ips = subIpsText.value.split('\n').filter(i => i.trim()).map(ip => ({ ip: ip.trim() }))
    if (form.id) await request.put(`/nodes/${form.id}`, form)
    else await request.post('/nodes', form)
    ElMessage.success('保存成功')
    visible.value = false
    emit('success')
}
</script>
