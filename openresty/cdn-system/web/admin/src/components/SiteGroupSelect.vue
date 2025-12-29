<template>
  <div style="display: flex; gap: 8px; width: 100%;">
    <el-select
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
      clearable
      :placeholder="placeholder || '网站分组, 可不选'"
      style="flex: 1;"
    >
      <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
    </el-select>
    <el-button :icon="Plus" circle @click="openCreateGroupDialog" />

    <!-- Add Group Dialog -->
    <el-dialog v-model="createGroupVisible" title="添加分组" width="400px" append-to-body>
      <el-form :model="createGroupForm" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="createGroupForm.name" placeholder="请输入分组名称" @keyup.enter="submitCreateGroup" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="createGroupForm.remark" placeholder="请输入备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createGroupVisible = false">取消</el-button>
        <el-button type="primary" @click="submitCreateGroup">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const props = defineProps({
  modelValue: {
    type: [Number, String],
    default: null
  },
  placeholder: String
})

const emit = defineEmits(['update:modelValue'])

const groupOptions = ref([])
const createGroupVisible = ref(false)
const createGroupForm = reactive({
  name: '',
  remark: ''
})

function loadGroups() {
  request.get('/site_groups').then(res => {
    groupOptions.value = res.data?.list || res.list || []
  })
}

function openCreateGroupDialog() {
  createGroupForm.name = ''
  createGroupForm.remark = ''
  createGroupVisible.value = true
}

function submitCreateGroup() {
  if (!createGroupForm.name) {
    ElMessage.warning('请输入分组名称')
    return
  }
  request.post('/site_groups', createGroupForm).then(() => {
    ElMessage.success('创建成功')
    createGroupVisible.value = false
    loadGroups() // Refresh list
  })
}

onMounted(() => {
  loadGroups()
})
</script>
