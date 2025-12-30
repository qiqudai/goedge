<template>
  <div style="display: flex; gap: 8px; width: 100%;">
    <el-select
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
      :multiple="multiple"
      clearable
      :placeholder="placeholder || (type === 'forward' ? '转发分组, 可不选' : '网站分组, 可不选')"
      style="flex: 1;"
    >
      <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
    </el-select>
    <el-button :icon="Plus" circle @click="openCreateGroupDialog" />

    <!-- Add Group Dialog -->
    <el-dialog v-model="createGroupVisible" :title="'添加' + (type === 'forward' ? '转发' : '网站') + '分组'" width="400px" append-to-body>
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
import { ref, onMounted, reactive, watch, computed } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const props = defineProps({
  modelValue: {
    type: [Number, String, Array],
    default: null
  },
  placeholder: String,
  type: {
    type: String,
    default: 'site' // 'site' or 'forward'
  },
  userId: {
    type: [Number, String],
    default: 0
  },
  multiple: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue'])

const groupOptions = ref([])
const createGroupVisible = ref(false)
const createGroupForm = reactive({
  name: '',
  remark: ''
})

const apiPath = computed(() => {
  return props.type === 'forward' ? '/forward_groups' : '/site_groups'
})

function loadGroups() {
  const params = {}
  if (props.userId) {
    params.user_id = props.userId
  }
  
  request.get(apiPath.value, { params }).then(res => {
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
  
  const payload = {
    ...createGroupForm,
    user_id: props.userId || 0
  }
  
  request.post(apiPath.value, payload).then(() => {
    ElMessage.success('创建成功')
    createGroupVisible.value = false
    loadGroups() // Refresh list
  })
}

watch([() => props.type, () => props.userId], () => {
  loadGroups()
})

onMounted(() => {
  loadGroups()
})
</script>
