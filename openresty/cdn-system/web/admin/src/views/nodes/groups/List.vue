<template>
  <div class="app-container">
    <div class="filter-container">
      <el-button type="primary" :icon="Plus" @click="handleCreate">新增分组</el-button>
      <el-button :disabled="selectedIds.length === 0" @click="handleBatchDelete">删除</el-button>

      <div class="right-actions">
        <el-select v-model="listQuery.region_id" placeholder="所有区域" style="width: 160px; margin-right: 10px;">
          <el-option label="所有区域" :value="0" />
          <el-option v-for="region in regions" :key="region.id" :label="region.name" :value="region.id" />
        </el-select>
        <el-input v-model="listQuery.keyword" placeholder="填名称或解析值搜索" style="width: 220px;" class="filter-item" @keyup.enter="handleFilter">
          <template #append>
            <el-button :icon="Search" @click="handleFilter" />
          </template>
        </el-input>
        <el-button link @click="resetFilter">清除</el-button>
      </div>
    </div>

    <AppTable
      :data="list"
      :loading="listLoading"
      border
      fit
      highlight-current-row
      style="width: 100%; margin-top: 20px;"
      v-model:current-page="listQuery.page"
      v-model:page-size="listQuery.limit"

      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="getList"
      @current-change="getList"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column label="ID" prop="id" width="80" align="center" />
      <el-table-column label="名称" min-width="150">
        <template #default="{row}">
          <el-button type="primary" link @click="handleUpdate(row)">{{ row.name }}</el-button>
        </template>
      </el-table-column>
      <el-table-column label="区域" min-width="120" align="center">
        <template #default="{row}">
          {{ regionNameMap[row.region_id || 0] || '默认' }}
        </template>
      </el-table-column>
      <el-table-column label="解析值" min-width="220">
        <template #default="{row}">
          <div>
            {{ row.resolution }}
            <span class="muted-text">(IPv4: {{ row.ipv4_resolution || '-' }})</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="统计" min-width="200" align="center">
        <template #default="{row}">
          节点数({{ row.node_count || 0 }})
          网站数({{ row.site_count || 0 }})
          转发数({{ row.forward_count || 0 }})
        </template>
      </el-table-column>
      <el-table-column label="L2配置" width="100" align="center">
        <template #default="{row}">
           {{ row.l2_config === 'default' || !row.l2_config ? '默认配置' : row.l2_config }}
        </template>
      </el-table-column>
      <el-table-column label="排序" prop="sort_order" width="80" align="center" />
      <el-table-column label="操作" align="center" width="200" class-name="small-padding fixed-width">
        <template #default="{row}">
          <el-button link type="primary" @click="handleResolution(row)">配置解析</el-button>
          <el-button link type="primary" @click="handleUpdate(row)">编辑</el-button>
          <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </AppTable>

    <!-- Edit Dialog -->
    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="600px">
      <el-form ref="dataForm" :model="temp" label-position="right" label-width="100px" style="width: 100%; padding-right: 20px;">
        <el-form-item label="名称:" prop="name">
          <el-input v-model="temp.name" placeholder="分组名称" />
        </el-form-item>
        <el-form-item label="区域ID:" prop="region_id">
             <el-input v-model.number="temp.region_id" placeholder="区域ID (0为默认)" />
        </el-form-item>
        <el-form-item label="解析值:" prop="resolution">
          <el-input v-model="temp.resolution" placeholder="CNAME解析值" />
        </el-form-item>
        <el-form-item label="IPv4解析值:" prop="ipv4_resolution">
          <el-input v-model="temp.ipv4_resolution" />
        </el-form-item>
        <el-form-item label="备注:" prop="remark">
            <el-input v-model="temp.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item label="排序:" prop="sort_order">
            <el-input v-model="temp.sort_order" placeholder="数字越小越靠前" />
        </el-form-item>
        
        <el-form-item label="L2配置:" prop="l2_config">
           <el-select v-model="temp.l2_config" placeholder="请选择">
               <el-option label="默认配置" value="default" />
           </el-select>
        </el-form-item>
        
        <el-form-item label="备用IP切换:">
            <el-radio-group v-model="temp.spare_ip_switch">
                <el-radio value="1">有主IP下线时</el-radio>
                <el-radio value="2">在线IP数少于备用IP数时</el-radio>
                <el-radio value="3">间隔切换</el-radio>
            </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">取消</el-button>
          <el-button type="primary" @click="dialogStatus==='create'?createData():updateData()">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const list = ref([])
const total = ref(0)
const listLoading = ref(true)
const regions = ref([])
const regionNameMap = ref({ 0: '默认' })
const selectedIds = ref([])
const listQuery = reactive({
  page: 1,
  limit: 20,
  keyword: '',
  region_id: 0
})

const dialogFormVisible = ref(false)
const dialogStatus = ref('')
const textMap = {
  update: '编辑线路分组',
  create: '新增线路分组'
}

const temp = reactive({
  id: undefined,
  name: '',
  region_id: 0,
  resolution: '',
  ipv4_resolution: '',
  remark: '',
  sort_order: 100,
  l2_config: '',
  spare_ip_switch: '1'
})

const getList = () => {
    listLoading.value = true
    request({
        url: '/node-groups', // Matches controller: /api/v1/admin/node-groups
        method: 'get',
        params: listQuery
    }).then(response => {
        // Backend returns: { code: 0, data: { list: [], total: 0 } }
        list.value = response.data.list
        total.value = response.data.total
        listLoading.value = false
    }).catch(() => {
        list.value = []
        listLoading.value = false
    })
}

const handleFilter = () => {
  listQuery.page = 1
  getList()
}

const resetFilter = () => {
  listQuery.keyword = ''
  listQuery.region_id = 0
  handleFilter()
}

const resetTemp = () => {
  temp.id = undefined
  temp.name = ''
  temp.region_id = 0
  temp.resolution = ''
  temp.ipv4_resolution = ''
  temp.remark = ''
  temp.sort_order = 100
  temp.l2_config = ''
  temp.spare_ip_switch = '1'
}

const handleCreate = () => {
  resetTemp()
  dialogStatus.value = 'create'
  dialogFormVisible.value = true
}

const createData = () => {
    request({
        url: '/node-groups',
        method: 'post',
        data: temp
    }).then(() => {
        dialogFormVisible.value = false
        ElMessage.success('创建成功')
        getList()
    })
}

const handleUpdate = (row) => {
    temp.id = row.id
    temp.name = row.name
    temp.region_id = row.region_id || 0
    temp.resolution = row.resolution
    temp.ipv4_resolution = row.ipv4_resolution
    temp.remark = row.remark // Model: Description json:"remark"
    temp.sort_order = row.sort_order
    temp.l2_config = row.l2_config
    temp.spare_ip_switch = row.spare_ip_switch // Model: BackupSwitchType json:"spare_ip_switch"
    
    dialogStatus.value = 'update'
    dialogFormVisible.value = true
}

const updateData = () => {
    request({
        url: `/node-groups/${temp.id}`,
        method: 'put',
        data: temp
    }).then(() => {
        dialogFormVisible.value = false
        ElMessage.success('更新成功')
        getList()
    })
}

const handleDelete = (row) => {
    ElMessageBox.confirm('确认删除该分组?', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
    }).then(() => {
        request({
            url: `/node-groups/${row.id}`,
            method: 'delete'
        }).then(() => {
            getList()
            ElMessage.success('删除成功')
        })
    })
}

const handleSelectionChange = (rows) => {
  selectedIds.value = rows.map(row => row.id)
}

const handleBatchDelete = () => {
  if (selectedIds.value.length === 0) {
    return
  }
  ElMessageBox.confirm('确认删除选中的分组?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    Promise.all(
      selectedIds.value.map(id =>
        request({
          url: `/node-groups/${id}`,
          method: 'delete'
        })
      )
    ).then(() => {
      selectedIds.value = []
      getList()
      ElMessage.success('删除成功')
    })
  })
}

const router = useRouter() // Initialize router

const handleResolution = (row) => {
   router.push({ name: 'NodeGroupResolution', params: { id: row.id } })
}

const loadRegions = () => {
  request({
    url: '/regions',
    method: 'get'
  }).then(response => {
    regions.value = response.data.list || []
    const map = { 0: '默认' }
    regions.value.forEach(region => {
      map[region.id] = region.name
    })
    regionNameMap.value = map
  })
}

onMounted(() => {
  loadRegions()
  getList()
})
</script>

<style scoped>
.filter-container {
  display: flex;
  align-items: center;
  gap: 12px;
}
.right-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.muted-text {
  color: #909399;
  margin-left: 6px;
}
.link-type {
    color: #409EFF;
    cursor: pointer;
}
</style>

