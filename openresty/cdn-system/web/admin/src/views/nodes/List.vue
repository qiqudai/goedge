<template>
  <div class="app-container">
    <!-- Filter Toolbar -->
    <div class="filter-container" style="margin-bottom: 20px;">
      <el-input v-model="listQuery.keyword" placeholder="节点名称 / IP" style="width: 200px;" class="filter-item" @keyup.enter="handleFilter" />
      <el-button class="filter-item" type="primary" :icon="Search" @click="handleFilter">
        搜索
      </el-button>
      <el-button class="filter-item" style="margin-left: 10px;" type="success" :icon="Plus" @click="handleCreate">
        添加节点
      </el-button>
      
      <!-- Batch Actions -->
      <el-button-group style="margin-left: 20px;">
        <el-button type="success" plain :disabled="!selectedRows.length" @click="handleBatch('start')">启用选中</el-button>
        <el-button type="warning" plain :disabled="!selectedRows.length" @click="handleBatch('stop')">停用选中</el-button>
        <el-button type="danger" plain :disabled="!selectedRows.length" @click="handleBatch('delete')">删除选中</el-button>
      </el-button-group>
    </div>

    <!-- Data Table -->
    <el-table
      v-loading="listLoading"
      :data="list"
      border
      fit
      highlight-current-row
      style="width: 100%;"
      @selection-change="handleSelectionChange">
      
      <el-table-column type="selection" width="55" align="center" />
      
      <el-table-column label="ID" prop="id" sortable="custom" align="center" width="80">
        <template #default="scope">{{ scope.row.id }}</template>
      </el-table-column>

      <el-table-column label="名称" min-width="120px">
        <template #default="{row}">
           <div style="font-weight: bold;cursor: pointer;" class="link-type" @click="handleUpdate(row)">{{ row.name }}</div>
           <div style="margin-top: 5px;">
               <el-tag v-if="row.type === 1" type="primary" size="small" effect="plain" disable-transitions>L1 边缘</el-tag>
               <el-tag v-else-if="row.type === 2" type="warning" size="small" effect="plain" disable-transitions>L2 中间</el-tag>
               <el-tag v-if="row.install_status === 1" type="success" size="small" effect="plain" style="margin-left: 5px;">已安装</el-tag>
           </div>
        </template>
      </el-table-column>
      
      <el-table-column label="组" width="100px" align="center">
          <template #default="{row}">
              {{ row.group_id ? '分组' + row.group_id : '默认' }}
          </template>
      </el-table-column>

      <el-table-column label="节点IP" min-width="120px">
        <template #default="{row}">
          <div>{{ row.ip }}</div>
          <div v-if="row.sub_ips && row.sub_ips.length > 0" style="color: #909399; font-size: 12px;">
             +{{ row.sub_ips.length }} 从IP
          </div>
        </template>
      </el-table-column>
      
      <el-table-column label="监控" width="80px" align="center">
          <template #default>
              <el-icon color="#67C23A"><Monitor /></el-icon>
          </template>
      </el-table-column>

      <el-table-column label="带宽" width="100px" align="center">
          <template #default>
              0 Mbps
          </template>
      </el-table-column>

      <el-table-column label="月流量" width="120px" align="center">
          <template #default="{row}">
            <div><el-icon><Top /></el-icon> {{ row.up_traffic || 0 }} KB</div>
            <div><el-icon><Bottom /></el-icon> {{ row.down_traffic || 0 }} KB</div>
          </template>
      </el-table-column>

      <el-table-column label="状态" align="center" width="80">
        <template #default="{row}">
          <el-switch v-model="row.status" :active-value="1" :inactive-value="0" disabled />
        </template>
      </el-table-column>
      
      <el-table-column label="备注" prop="remark" min-width="100px" show-overflow-tooltip />

      <el-table-column label="排序" prop="sort_order" width="80" align="center" sortable />

      <el-table-column label="操作" align="center" width="180" class-name="small-padding fixed-width">
        <template #default="{row}">

          <el-button link type="primary" size="small" @click="handleUpdate(row)">设置</el-button>
          <el-dropdown trigger="click" style="margin-left: 10px;">
              <span class="el-dropdown-link" style="color: #409EFF; font-size: 12px; cursor: pointer;">
                更多<el-icon class="el-icon--right"><arrow-down /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="handleBatch('delete', [row])">删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <div style="margin-top: 20px; text-align: right;">
        <el-pagination
          v-if="total > 0"
          v-model:current-page="listQuery.page"
          v-model:page-size="listQuery.pageSize"
          :page-sizes="[10, 20, 30, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="getList"
          @current-change="getList"
        />
    </div>

    <!-- Dialog: Add/Edit Node -->
    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="600px">
      <el-tabs v-model="activeTab" type="card">
        <!-- Tab 1: Basic Settings -->
        <el-tab-pane label="基本设置" name="basic">
            <el-form ref="dataForm" :model="temp" label-position="right" label-width="100px" style="margin-top: 20px;">
                <el-form-item label="名称" prop="name">
                  <el-input v-model="temp.name" placeholder="节点名称" />
                </el-form-item>
                <el-form-item label="备注" prop="remark">
                  <el-input v-model="temp.remark" placeholder="请输入备注" />
                </el-form-item>
                <el-form-item label="排序" prop="sort_order">
                  <el-input v-model.number="temp.sort_order" placeholder="100" />
                </el-form-item>
                <el-form-item label="IP" prop="ip">
                  <el-input v-model="temp.ip" placeholder="公网 IP" />
                  <div style="font-size: 12px; color: #909399;">如果新旧IP是不同节点，请使用待初始化里的替换节点功能</div>
                </el-form-item>
                <el-form-item label="类型" prop="type">
                    <el-radio-group v-model="temp.type">
                        <el-radio :label="1">L1边缘节点</el-radio>
                        <el-radio :label="2">L2中间节点</el-radio>
                    </el-radio-group>
                    <div style="font-size: 12px; color: #909399; line-height: 1.5;">
                        <div>L1边缘节点是用户实际访问的节点;</div>
                        <div>L2中间节点是L1与源服务器之间的节点，用于汇聚L1节点请求，提高缓存命中率，或者优化回源线路</div>
                    </div>
                </el-form-item>
            </el-form>
        </el-tab-pane>

        <!-- Tab 2: Node Settings -->
        <el-tab-pane label="节点设置" name="settings">
            <el-form :model="temp" label-position="right" label-width="100px" style="margin-top: 20px;">
                <el-form-item label="缓存目录" prop="cache_dir">
                  <el-input v-model="temp.cache_dir" placeholder="/data/nginx/cache/" />
                </el-form-item>
                <el-form-item label="缓存上限" prop="cache_limit">
                  <el-input v-model.number="temp.cache_limit" placeholder="100">
                      <template #append>GB</template>
                  </el-input>
                </el-form-item>
                <el-form-item label="日志目录" prop="log_dir">
                  <el-input v-model="temp.log_dir" placeholder="/usr/local/openresty/nginx/logs/" />
                </el-form-item>
            </el-form>
        </el-tab-pane>

        <!-- Tab 3: Auto Disable -->
        <el-tab-pane label="自动禁用" name="disable">
            <div style="padding: 20px; text-align: center; color: #909399;">暂无配置</div>
        </el-tab-pane>

        <!-- Tab 4: Sub IPs -->
        <el-tab-pane label="添加子IP" name="sub_ips">
             <el-form style="margin-top: 20px;">
                 <el-form-item label="子IP" label-width="80px">
                    <el-input
                        v-model="tempIPs"
                        :rows="5"
                        type="textarea"
                        placeholder="一行一个"
                    />
                 </el-form-item>
             </el-form>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">取消</el-button>
          <el-button type="primary" @click="dialogStatus==='create'?createData():updateData()">
            确认
          </el-button>
        </div>
      </template>
    </el-dialog>

  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search, Plus, InfoFilled, Top, Bottom, ArrowDown, Monitor } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const total = ref(0)
const listLoading = ref(true)
const selectedRows = ref([])
const listQuery = reactive({
  page: 1,
  pageSize: 20,
  keyword: ''
})

const dialogFormVisible = ref(false)
const dialogStatus = ref('')
const activeTab = ref('basic')
const textMap = {
  update: '编辑节点',
  create: '创建新节点'
}

const temp = reactive({
  id: undefined,
  name: '',
  group_id: 0,
  ip: '',
  remark: '',
  sort_order: 100,
  type: 1,
  cache_dir: '/data/nginx/cache',
  cache_limit: 0,
  log_dir: '/usr/local/nginx/logs',
  ssh_host: '',
  ssh_port: 22,
  ssh_user: 'root',
  ssh_password: ''
})

const tempIPs = ref('')
const currentEditingNode = ref(null)

const getList = () => {
  listLoading.value = true
  request({
    url: '/nodes',
    method: 'get',
    params: {
        page: listQuery.page,
        pageSize: listQuery.pageSize,
        keyword: listQuery.keyword
    }
  }).then(response => {
    if (response.data) {
        list.value = response.data.list
        total.value = response.data.total
    }
    listLoading.value = false
  }).catch(() => {
    listLoading.value = false
  })
}

const handleFilter = () => {
    listQuery.page = 1
    getList()
}



const resetTemp = () => {
    temp.id = undefined
    temp.name = ''
    temp.group_id = 0
    temp.ip = ''
    temp.remark = ''
    temp.sort_order = 100
    temp.type = 1
    temp.cache_dir = ''
    temp.cache_limit = 0
    temp.log_dir = ''
    activeTab.value = 'basic'
    tempIPs.value = ''
}

const handleCreate = () => {
    resetTemp()
    dialogStatus.value = 'create'
    dialogFormVisible.value = true
}

// ... createData ...

const handleUpdate = (row) => {
    temp.id = row.id
    temp.name = row.name
    temp.group_id = row.group_id
    temp.ip = row.ip
    temp.remark = row.remark || ''
    temp.sort_order = row.sort_order || 100
    temp.type = row.type || 1
    temp.cache_dir = row.cache_dir || ''
    temp.cache_limit = row.cache_limit || 0
    temp.log_dir = row.log_dir || ''
    
    // Convert SubIPs to string for textarea
    tempIPs.value = row.sub_ips ? row.sub_ips.map(i => i.ip).join('\n') : ''
    
    dialogStatus.value = 'update'
    dialogFormVisible.value = true
    activeTab.value = 'basic'
}

const updateData = () => {
    // 1. Parse SubIPs
    const ipLines = tempIPs.value.split('\n').filter(i => i.trim() !== '')
    const subIPs = ipLines.map(ip => ({ ip: ip.trim() }))
    
    // 2. Prepare Payload
    const payload = {
        ...temp,
        sub_ips: subIPs
    }

    // 3. Call API
    request({
        url: `/nodes/${temp.id}`,
        method: 'put',
        data: payload
    }).then(() => {
        dialogFormVisible.value = false
        ElMessage.success('更新成功')
        getList()
    })
}

const handleBatch = (action, rows) => {
    const listToProcess = rows || selectedRows.value
    if (listToProcess.length === 0) return
    
    const ids = listToProcess.map(row => row.id)
    
    ElMessageBox.confirm(`确定要执行 ${action} 操作吗？`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
    }).then(() => {
        request({
            url: '/nodes/batch',
            method: 'post',
            data: { action, ids }
        }).then(res => {
             ElMessage.success(res.msg)
             getList()
        })
    })
}

const handleSelectionChange = (val) => {
    selectedRows.value = val
}



onMounted(() => {
  getList()
})
</script>
