<template>
  <div class="app-container">
    <div class="header-bar">
      <el-button @click="goBack">返回</el-button>
      <div class="header-fields">
        <div class="field">
          <span class="label">区域:</span>
          <el-select v-model="selectedRegionId" placeholder="请选择" style="width: 160px" @change="handleRegionChange">
            <el-option label="默认" :value="0" />
            <el-option v-for="region in regions" :key="region.id" :label="region.name" :value="region.id" />
          </el-select>
        </div>
        <div class="field">
          <span class="label">分组:</span>
          <el-select v-model="selectedGroupId" placeholder="请选择" style="width: 200px" @change="handleGroupChange">
            <el-option v-for="group in filteredGroups" :key="group.id" :label="group.name" :value="group.id" />
          </el-select>
        </div>
      </div>
    </div>

    <div class="line-bar">
      <span class="label">当前线路:</span>
      <el-cascader
        v-model="currentLineId"
        :options="lineOptions"
        :props="lineProps"
        style="width: 300px"
        @change="handleLineChange"
      />
      <span class="line-tip">当前线路为“全部”时，新增节点会对所有线路生效。</span>
    </div>

    <div class="split-container">
      <el-card class="panel left">
        <template #header>
          <div class="panel-title">未设置的IP</div>
        </template>
        <div class="panel-actions">
          <el-button type="primary" @click="handleAssign">批量添加</el-button>
          <el-input v-model="leftKeyword" placeholder="输入IP或名称搜索" clearable style="width: 220px" />
          <el-button link @click="leftKeyword = ''">清除</el-button>
        </div>
        <AppTable
          :data="filteredAvailable"
          :loading="loading"
          border
          height="420"

          layout="total, sizes, prev, pager, next"
          :persist-key="`node-group-available-${selectedGroupId}`"
          @selection-change="handleLeftSelection"
        >
          <el-table-column type="selection" width="48" align="center" />
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="ip" label="IP" min-width="140" />
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <span :class="['status-dot', row.online ? 'status-ok' : 'status-stop']"></span>
              <span>{{ row.online ? '在线' : '不在线' }}</span>
            </template>
          </el-table-column>
        </AppTable>
      </el-card>

      <el-card class="panel right">
        <template #header>
          <div class="panel-title">已设置的IP，当前线路：{{ currentLineLabel }}</div>
        </template>
        <div class="panel-actions">
          <el-button @click="handleAction('enable')">启用</el-button>
          <el-button @click="handleAction('disable')">禁用</el-button>
          <el-button @click="handleDelete">删除</el-button>
          <el-dropdown @command="handleMoreAction">
            <el-button>
              更多操作
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="set_backup">备用IP</el-dropdown-item>
                <el-dropdown-item command="unset_backup">取消备用IP</el-dropdown-item>
                <el-dropdown-item command="set_weight">设置权重</el-dropdown-item>
                <el-dropdown-item command="set_backup_default">备用默认解析</el-dropdown-item>
                <el-dropdown-item command="unset_backup_default">取消备用默认解析</el-dropdown-item>
                <el-dropdown-item command="set_sort">修改排序</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-input v-model="rightKeyword" placeholder="输入IP或名称搜索" clearable style="width: 220px" />
          <el-button link @click="rightKeyword = ''">清除</el-button>
        </div>
        <AppTable
          :data="filteredAssigned"
          :loading="loading"
          border
          height="420"

          layout="total, sizes, prev, pager, next"
          :persist-key="`node-group-assigned-${selectedGroupId}-${currentLineId}`"
          @selection-change="handleRightSelection"
        >
          <el-table-column type="selection" width="48" align="center" />
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="ip" label="IP" min-width="140" />
          <el-table-column label="备用IP" width="100" align="center">
            <template #default="{ row }">
              {{ row.is_backup ? '是' : '否' }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110" align="center">
            <template #default="{ row }">
              <span :class="['status-dot', row.enabled ? 'status-ok' : 'status-stop']"></span>
              <span>{{ row.enabled ? '启用' : '禁用' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="weight" label="权重" width="90" align="center" />
          <el-table-column prop="sort_order" label="排序" width="90" align="center" />
        </AppTable>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDown } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'
import AppTable from '@/components/AppTable.vue'

const route = useRoute()
const router = useRouter()

const regions = ref([])
const groups = ref([])
const selectedRegionId = ref(0)
const selectedGroupId = ref(0)
const currentLineId = ref('default')
const loading = ref(false)

const availableNodes = ref([])
const assignedNodes = ref([])
const leftKeyword = ref('')
const rightKeyword = ref('')
const leftSelected = ref([])
const rightSelected = ref([])

const lineProps = {
  emitPath: false,
  value: 'value',
  label: 'label',
  children: 'children'
}

const lineOptions = [
  { label: '全部', value: 'all' },
  { label: '默认', value: 'default' },
  {
    label: '电信',
    value: 'telecom',
    children: [{ label: '电信', value: 'telecom' }]
  },
  {
    label: '联通',
    value: 'unicom',
    children: [{ label: '联通', value: 'unicom' }]
  },
  {
    label: '移动',
    value: 'mobile',
    children: [{ label: '移动', value: 'mobile' }]
  },
  {
    label: '其他运营商',
    value: 'other',
    children: [
      { label: '铁通', value: 'tie-tong' },
      { label: '广电', value: 'broadcast' },
      { label: '教育网', value: 'edu' }
    ]
  },
  {
    label: '境内',
    value: 'china',
    children: [{ label: '境内', value: 'china' }]
  },
  {
    label: '全球',
    value: 'global',
    children: [{ label: '境外', value: 'global' }]
  },
  {
    label: '搜索引擎',
    value: 'search',
    children: [
      { label: '百度', value: 'baidu' },
      { label: '谷歌', value: 'google' },
      { label: '有道', value: 'youdao' },
      { label: '必应', value: 'bing' },
      { label: '搜狗', value: 'sogou' },
      { label: '奇虎', value: 'qh360' },
      { label: '搜索引擎', value: 'search' }
    ]
  },
  { label: '线路分组', value: 'line_group' },
  { label: '自定义线路', value: 'custom' }
]

const lineLabelMap = computed(() => {
  const map = {}
  const walk = (items) => {
    items.forEach(item => {
      map[item.value] = item.label
      if (item.children) {
        walk(item.children)
      }
    })
  }
  walk(lineOptions)
  return map
})

const currentLineLabel = computed(() => lineLabelMap.value[currentLineId.value] || currentLineId.value)

const filteredGroups = computed(() => {
  if (!selectedRegionId.value) {
    return groups.value
  }
  return groups.value.filter(group => Number(group.region_id || 0) === Number(selectedRegionId.value))
})

const filteredAvailable = computed(() => {
  if (!leftKeyword.value) {
    return availableNodes.value
  }
  const keyword = leftKeyword.value.trim()
  return availableNodes.value.filter(item => item.name.includes(keyword) || item.ip.includes(keyword))
})

const filteredAssigned = computed(() => {
  if (!rightKeyword.value) {
    return assignedNodes.value
  }
  const keyword = rightKeyword.value.trim()
  return assignedNodes.value.filter(item => item.name.includes(keyword) || item.ip.includes(keyword))
})

const goBack = () => {
  router.push('/node/groups')
}

const loadRegions = () => {
  request({ url: '/regions', method: 'get' }).then(res => {
    regions.value = res.data.list || []
  })
}

const loadGroups = () => {
  request({
    url: '/node-groups',
    method: 'get',
    params: { page: 1, limit: 200 }
  }).then(res => {
    groups.value = res.data.list || []
    if (!selectedGroupId.value && groups.value.length > 0) {
      selectedGroupId.value = Number(route.params.id || groups.value[0].id)
    }
  })
}

const loadResolution = () => {
  if (!selectedGroupId.value) {
    return
  }
  loading.value = true
  request({
    url: `/node-groups/${selectedGroupId.value}/resolution`,
    method: 'get',
    params: { line_id: currentLineId.value }
  }).then(res => {
    const payload = res.data || {}
    const group = payload.group || {}
    selectedRegionId.value = Number(group.region_id || 0)
    availableNodes.value = payload.available || []
    assignedNodes.value = payload.assigned || []
  }).finally(() => {
    loading.value = false
  })
}

const handleRegionChange = () => {
  if (filteredGroups.value.length === 0) {
    selectedGroupId.value = 0
    return
  }
  if (!filteredGroups.value.find(item => item.id === selectedGroupId.value)) {
    selectedGroupId.value = filteredGroups.value[0].id
  }
  if (selectedGroupId.value) {
    router.push({ name: 'NodeGroupResolution', params: { id: selectedGroupId.value } })
  }
}

const handleGroupChange = () => {
  if (!selectedGroupId.value) {
    return
  }
  router.push({ name: 'NodeGroupResolution', params: { id: selectedGroupId.value } })
}

const handleLineChange = () => {
  loadResolution()
}

const handleLeftSelection = (rows) => {
  leftSelected.value = rows
}

const handleRightSelection = (rows) => {
  rightSelected.value = rows
}

const handleAssign = () => {
  if (!selectedGroupId.value) {
    return
  }
  if (leftSelected.value.length === 0) {
    ElMessage.warning('请选择要添加的节点')
    return
  }
  const items = leftSelected.value.map(item => ({
    node_id: item.node_id,
    node_ip_id: item.node_ip_id,
    name: item.name,
    ip: item.ip
  }))
  request({
    url: `/node-groups/${selectedGroupId.value}/resolution/assign`,
    method: 'post',
    data: {
      line_id: currentLineId.value,
      line_name: currentLineLabel.value,
      items
    }
  }).then(() => {
    ElMessage.success('添加成功')
    leftSelected.value = []
    loadResolution()
  })
}

const handleAction = (action, value = '') => {
  if (rightSelected.value.length === 0) {
    ElMessage.warning('请选择要操作的节点')
    return
  }
  request({
    url: `/node-groups/${selectedGroupId.value}/resolution/action`,
    method: 'post',
    data: {
      action,
      ids: rightSelected.value.map(item => item.id),
      value
    }
  }).then(() => {
    ElMessage.success('操作成功')
    rightSelected.value = []
    loadResolution()
  })
}

const handleDelete = () => {
  if (rightSelected.value.length === 0) {
    ElMessage.warning('请选择要删除的节点')
    return
  }
  ElMessageBox.confirm('确认删除选中的节点?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    handleAction('delete')
  })
}

const handleMoreAction = (command) => {
  if (command === 'set_weight') {
    ElMessageBox.prompt('请输入权重', '设置权重', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputPattern: /^[0-9]+$/,
      inputErrorMessage: '请输入数字'
    }).then(({ value }) => {
      handleAction('set_weight', value)
    })
    return
  }
  if (command === 'set_sort') {
    ElMessageBox.prompt('请输入排序值', '修改排序', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputPattern: /^[0-9]+$/,
      inputErrorMessage: '请输入数字'
    }).then(({ value }) => {
      handleAction('set_sort', value)
    })
    return
  }
  handleAction(command)
}

onMounted(() => {
  selectedGroupId.value = Number(route.params.id || 0)
  loadRegions()
  loadGroups()
  loadResolution()
})

watch(
  () => route.params.id,
  (val) => {
    const parsed = Number(val || 0)
    if (parsed && parsed !== selectedGroupId.value) {
      selectedGroupId.value = parsed
      loadResolution()
    }
  }
)
</script>

<style scoped>
.header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.header-fields {
  display: flex;
  align-items: center;
  gap: 16px;
}
.field {
  display: flex;
  align-items: center;
  gap: 8px;
}
.label {
  color: #606266;
}
.line-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.line-tip {
  color: #909399;
  font-size: 12px;
}
.split-container {
  display: flex;
  gap: 16px;
}
.panel {
  flex: 1;
}
.panel-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.panel-title {
  font-weight: 600;
}
.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
}
.status-ok {
  background: #67c23a;
}
.status-stop {
  background: #f56c6c;
}
</style>
