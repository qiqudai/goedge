<template>
  <div class="app-container">
    <div class="header-bar">
      <el-button @click="goBack">返回</el-button>
      <div class="header-fields">
        <div class="field">
          <span class="label">区域:</span>
          <el-select v-model="selectedRegionId" placeholder="请选择" style="width: 160px" @change="handleRegionChange">
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

    <div class="split-container">
      <el-card class="panel left">
        <template #header>
          <div class="panel-title">未设置的IP</div>
        </template>
        <div class="panel-actions">
          <el-button type="primary" @click="handleAssign(false)">批量添加</el-button>
          <el-button @click="handleAssign(true)">批量备用</el-button>
          <div class="search-box">
            <el-input
              v-model="leftSearchValue"
              placeholder="输入IP或名称搜索"
              clearable
              style="width: 200px"
              @keyup.enter="handleLeftSearch"
              @clear="handleLeftSearch"
            />
            <el-button @click="handleLeftSearch">
              <el-icon><Search /></el-icon>
            </el-button>
          </div>
        </div>
        <AppTable
          :data="filteredAvailable"
          :loading="loading"
          border

          layout="total, sizes, prev, pager, next"
          :show-pagination="false"
          :page-sizes="[10000]"
          :persist-key="availablePersistKey"
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
          <div class="line-bar">
            <span class="label">当前线路:</span>
            <el-cascader
              v-model="currentLineId"
              :options="lineOptions"
              :props="lineProps"
              style="width: 300px"
              popper-class="long-cascader-dropdown"
              @change="handleLineChange"
            />
            <span class="line-tip">当前线路为“全部”时，新增节点会对所有线路生效。</span>
          </div>

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
          <div class="search-box">
            <el-input
              v-model="rightKeyword"
              placeholder="输入IP或名称搜索"
              clearable
              style="width: 200px"
              @keyup.enter="handleRightSearch"
              @clear="handleRightSearch"
            />
            <el-button @click="handleRightSearch">
              <el-icon><Search /></el-icon>
            </el-button>
          </div>
        </div>
        <AppTable
          :data="filteredAssigned"
          :loading="loading"
          border
          height="420"

          layout="total, sizes, prev, pager, next"
          :show-pagination="false"
          :page-sizes="[10000]"
          :persist-key="assignedPersistKey"
          @selection-change="handleRightSelection"
        >
          <el-table-column type="selection" width="48" align="center" />
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="line_name" label="线路" min-width="100" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="ip" label="IP" min-width="140" />
          <el-table-column label="备用IP" width="100" align="center">
            <template #default="{ row }">
              <el-icon v-if="row.is_backup" style="font-size: 16px; color: rgb(25, 190, 107); vertical-align: middle;"><CircleCheckFilled /></el-icon>
              <span v-else>否</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110" align="center">
            <template #default="{ row }">
              <span :class="['status-dot', (!row.is_on || !row.node_is_on || !row.online) ? 'status-stop' : 'status-ok']"></span>
              <span v-if="!row.is_on">禁用</span>
              <span v-else-if="!row.node_is_on">节点已禁用</span>
              <span v-else-if="!row.online">节点离线</span>
              <span v-else>启用</span>
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
import { ArrowDown, Search, CircleCheckFilled } from '@element-plus/icons-vue'
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

const allAvailable = ref([])
const allAssigned = ref([])
const leftKeyword = ref('')
const leftSearchValue = ref('')
const rightKeyword = ref('')
const leftSelected = ref([])
const rightSelected = ref([])

const lineProps = {
  emitPath: false,
  expandTrigger: 'hover',
  value: 'value',
  label: 'label',
  children: 'children'
}

const lineOptions = [
  { label: '全部', value: 'all' },
  { label: '默认', value: 'default' },
  { label: '电信', value: 'telecom' },
  { label: '联通', value: 'unicom' },
  { label: '移动', value: 'mobile' },
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
    value: 'group_china',
    children: [
      { label: '全部', value: 'china' },
      { label: '安徽', value: 'anhui' },
      { label: '北京', value: 'beijing' },
      { label: '重庆', value: 'chongqing' },
      { label: '福建', value: 'fujian' },
      { label: '甘肃', value: 'gansu' },
      { label: '广东', value: 'guangdong' },
      { label: '广西', value: 'guangxi' },
      { label: '贵州', value: 'guizhou' },
      { label: '海南', value: 'hainan' },
      { label: '河北', value: 'hebei' },
      { label: '黑龙江', value: 'heilongjiang' },
      { label: '河南', value: 'henan' },
      { label: '湖北', value: 'hubei' },
      { label: '湖南', value: 'hunan' },
      { label: '江苏', value: 'jiangsu' },
      { label: '江西', value: 'jiangxi' },
      { label: '吉林', value: 'jilin' },
      { label: '辽宁', value: 'liaoning' },
      { label: '内蒙古', value: 'neimenggu' },
      { label: '宁夏', value: 'ningxia' },
      { label: '青海', value: 'qinghai' },
      { label: '陕西', value: 'shaanxi' },
      { label: '山东', value: 'shandong' },
      { label: '上海', value: 'shanghai' },
      { label: '山西', value: 'shanxi' },
      { label: '四川', value: 'sichuan' },
      { label: '天津', value: 'tianjin' },
      { label: '西藏', value: 'xizang' },
      { label: '新疆', value: 'xinjiang' },
      { label: '云南', value: 'yunnan' },
      { label: '浙江', value: 'zhejiang' }
    ]
  },
  { label: '境外', value: 'global' },
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

const filteredAssigned = computed(() => {
  let list = allAssigned.value
  if (currentLineId.value !== 'all') {
    list = list.filter(item => (item.line_id || item.line) === currentLineId.value)
  }

  if (!rightKeyword.value) {
    return list
  }
  const keyword = rightKeyword.value.trim()
  return list.filter(item => item.name.includes(keyword) || item.ip.includes(keyword))
})

const filteredAvailable = computed(() => {
  // Universe = allAvailable + unique(allAssigned)
  const universe = [...allAvailable.value]
  const existingNodeIds = new Set(universe.map(u => u.id))
  
  allAssigned.value.forEach(item => {
    // For assigned items, node ID is 'node_id'. If not present, fallback to 'id' (though usually id is Resolution ID)
    // Actually, backend usually sends Resolution object {id, node_id, ...}.
    // We need to map it to a Node-like object for the Left Panel.
    const nodeId = item.node_id
    if (nodeId && !existingNodeIds.has(nodeId)) {
      existingNodeIds.add(nodeId)
      universe.push({
        ...item,
        id: nodeId, // Left panel expects 'id' to be Node ID
        online: true, // Assigned nodes are presumed online or status unknown
        is_on: true // Assigned nodes are presumed enabled (validation shouldn't block based on resolution status)
      })
    }
  })

  // Exclude what is currently in Right Panel
  // Right Panel items: filteredAssigned (but we need the raw list for the current line, ignoring search keyword)
  let currentAssignedList = allAssigned.value
  if (currentLineId.value !== 'all') {
    currentAssignedList = currentAssignedList.filter(item => (item.line_id || item.line) === currentLineId.value)
  }
  const currentAssignedNodeIds = new Set(currentAssignedList.map(item => item.node_id))

  const available = universe.filter(u => !currentAssignedNodeIds.has(u.id))

  if (!leftKeyword.value) {
    return available
  }
  const keyword = leftKeyword.value.trim()
  return available.filter(item => item.name.includes(keyword) || item.ip.includes(keyword))
})

const handleRightSearch = () => {
  // Trigger computed property update if needed, but v-model rightKeyword already does it.
  // This function is mainly for the enter key and button click if we wanted to debounce,
  // but for frontend filtering, reactivity handles it.
  // We can leave it empty or use it to force a refresh if logic was complex.
}

const availablePersistKey = computed(() => `node-group-available-${selectedGroupId.value}`)
const assignedPersistKey = computed(() => `node-group-assigned-${selectedGroupId.value}-${currentLineId.value}`)

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
    allAvailable.value = payload.available || []
    allAssigned.value = payload.assigned || []
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

const handleLeftSearch = () => {
  leftKeyword.value = leftSearchValue.value
}

const handleAssign = (isBackup = false) => {
  if (!selectedGroupId.value) {
    return
  }
  if (leftSelected.value.length === 0) {
    ElMessage.warning('请选择要添加的节点')
    return
  }
  // Check for offline or disabled nodes
  // is_on might be undefined for available nodes, so we only check if it is explicitly false/0
  const invalidNodes = leftSelected.value.filter(item => !item.online || (item.is_on != null && !item.is_on))
  if (invalidNodes.length > 0) {
    ElMessage.warning('禁止添加不在线或已禁用的节点')
    return
  }
  const items = leftSelected.value.map(item => ({
    node_id: item.node_id || item.id,
    node_ip_id: item.node_ip_id,
    name: item.name,
    ip: item.ip,
    is_backup: isBackup,
    line: currentLineLabel.value,
    line_name: currentLineLabel.value
  }))
  request({
    url: `/node-groups/${selectedGroupId.value}/resolution/assign`,
    method: 'post',
    data: {
      line: currentLineLabel.value,
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
      ids: rightSelected.value.map(item => Number(item.id)),
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
  // Check for enabled nodes (require disable first)
  if (rightSelected.value.some(item => item.is_on)) {
    ElMessage.warning('请先禁用节点后再删除')
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
  margin-bottom: 16px;
  gap: 28px;
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
}
.line-tip {
  color: #909399;
  font-size: 12px;
}
.split-container {
  display: flex;
  gap: 16px;
}
.panel.left {
  flex: 0 0 35%;
}
.panel.right {
  flex: 1;
  min-width: 0;
}
.panel-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  gap: 8px;
  margin-bottom: 12px;
}
.search-box {
  display: flex;
  gap: 4px;
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

<style>
.long-cascader-dropdown .el-cascader-menu__wrap {
  height: 400px;
}
</style>
