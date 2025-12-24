<template>
  <div class="app-container">
    <div class="filter-container" style="margin-bottom: 20px;">
      <el-input v-model="listQuery.keyword" :placeholder="t.nodeKeyword" style="width: 200px;" class="filter-item" @keyup.enter="handleFilter" />
      <el-button class="filter-item" type="primary" :icon="Search" @click="handleFilter">
        {{ t.search }}
      </el-button>
      <el-button class="filter-item" style="margin-left: 10px;" type="success" :icon="Plus" @click="handleCreate">
        {{ t.addNode }}
      </el-button>

      <el-button-group style="margin-left: 20px;">
        <el-button type="success" plain :disabled="!selectedRows.length" @click="handleBatch('start')">{{ t.enableSelected }}</el-button>
        <el-button type="warning" plain :disabled="!selectedRows.length" @click="handleBatch('stop')">{{ t.disableSelected }}</el-button>
        <el-button type="danger" plain :disabled="!selectedRows.length" @click="handleBatch('delete')">{{ t.deleteSelected }}</el-button>
      </el-button-group>
    </div>

    <el-table
      v-loading="listLoading"
      :data="list"
      border
      fit
      highlight-current-row
      style="width: 100%;"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="55" align="center" />

      <el-table-column label="ID" prop="id" sortable="custom" align="center" width="80">
        <template #default="scope">{{ scope.row.id }}</template>
      </el-table-column>

      <el-table-column :label="t.name" min-width="120px">
        <template #default="{ row }">
          <div style="font-weight: bold; cursor: pointer;" class="link-type" @click="handleUpdate(row)">{{ row.name }}</div>
          <div style="margin-top: 5px;">
            <el-tag v-if="row.type === 1" type="primary" size="small" effect="plain" disable-transitions>{{ t.l1Edge }}</el-tag>
            <el-tag v-else-if="row.type === 2" type="warning" size="small" effect="plain" disable-transitions>{{ t.l2Middle }}</el-tag>
            <el-tag v-if="row.install_status === 1" type="success" size="small" effect="plain" style="margin-left: 5px;">{{ t.installed }}</el-tag>
          </div>
        </template>
      </el-table-column>

      <el-table-column :label="t.group" width="100px" align="center">
        <template #default="{ row }">
          {{ row.group_id ? t.groupPrefix + row.group_id : t.groupDefault }}
        </template>
      </el-table-column>

      <el-table-column :label="t.nodeIp" min-width="120px">
        <template #default="{ row }">
          <div>{{ row.ip }}</div>
          <div v-if="row.sub_ips && row.sub_ips.length > 0" style="color: #909399; font-size: 12px;">
            +{{ row.sub_ips.length }} {{ t.fromIp }}
          </div>
        </template>
      </el-table-column>

      <el-table-column :label="t.monitor" width="80px" align="center">
        <template #default>
          <el-icon color="#67C23A"><Monitor /></el-icon>
        </template>
      </el-table-column>

      <el-table-column :label="t.bandwidth" width="100px" align="center">
        <template #default="{ row }">
          {{ row.bw_limit || '-' }}
        </template>
      </el-table-column>

      <el-table-column :label="t.monthlyTraffic" width="120px" align="center">
        <template #default="{ row }">
          <div><el-icon><Top /></el-icon> {{ row.up_traffic || 0 }} KB</div>
          <div><el-icon><Bottom /></el-icon> {{ row.down_traffic || 0 }} KB</div>
        </template>
      </el-table-column>

      <el-table-column :label="t.status" align="center" width="80">
        <template #default="{ row }">
          <el-switch v-model="row.enable" :active-value="true" :inactive-value="false" @change="handleStatusChange(row)" />
        </template>
      </el-table-column>

      <el-table-column :label="t.remark" prop="remark" min-width="100px" show-overflow-tooltip />

      <el-table-column :label="t.sort" prop="sort_order" width="80" align="center" sortable />

      <el-table-column :label="t.action" align="center" width="180" class-name="small-padding fixed-width">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="handleUpdate(row)">{{ t.setting }}</el-button>
          <el-dropdown trigger="click" style="margin-left: 10px;">
            <span class="el-dropdown-link" style="color: #409EFF; font-size: 12px; cursor: pointer;">
              {{ t.more }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleBatch('delete', [row])">{{ t.delete }}</el-dropdown-item>
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

    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="600px">
      <el-tabs v-model="activeTab" type="card">
        <el-tab-pane :label="t.basicSettings" name="basic">
          <el-form ref="dataForm" :model="temp" label-position="right" label-width="100px" style="margin-top: 20px;">
            <el-form-item :label="t.name" prop="name">
              <el-input v-model="temp.name" :placeholder="t.name" />
            </el-form-item>
            <el-form-item :label="t.remark" prop="remark">
              <el-input v-model="temp.remark" :placeholder="t.remarkPlaceholder" />
            </el-form-item>
            <el-form-item :label="t.sort" prop="sort_order">
              <el-input v-model.number="temp.sort_order" placeholder="100" />
            </el-form-item>
            <el-form-item label="IP" prop="ip">
              <el-input v-model="temp.ip" :placeholder="t.publicIp" />
              <div style="font-size: 12px; color: #909399;" v-text="t.ipNote"></div>
            </el-form-item>
            <el-form-item :label="t.nodeType" prop="type">
              <el-radio-group v-model="temp.type">
                <el-radio :label="1">{{ t.l1EdgeNode }}</el-radio>
                <el-radio :label="2">{{ t.l2MiddleNode }}</el-radio>
              </el-radio-group>
              <div style="font-size: 12px; color: #909399; line-height: 1.5;">
                <div v-text="t.l1Desc"></div>
                <div v-text="t.l2Desc"></div>
              </div>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t.nodeSettings" name="settings">
          <el-form :model="temp" label-position="right" label-width="100px" style="margin-top: 20px;">
            <el-form-item :label="t.cacheDir" prop="cache_dir">
              <el-input v-model="temp.cache_dir" placeholder="/data/nginx/cache/" />
            </el-form-item>
            <el-form-item :label="t.cacheLimit" prop="cache_limit">
              <el-input v-model.number="temp.cache_limit" placeholder="100">
                <template #append>GB</template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t.logDir" prop="log_dir">
              <el-input v-model="temp.log_dir" placeholder="/usr/local/openresty/nginx/logs/" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t.autoDisable" name="disable">
          <el-form :model="temp" label-position="right" label-width="110px" style="margin-top: 20px;">
            <el-form-item :label="t.autoDisableEnable">
              <el-switch v-model="temp.check_on" />
            </el-form-item>
            <el-form-item :label="t.bwLimit">
              <el-input v-model="bwLimitValue" placeholder="100">
                <template #append>
                  <el-select v-model="bwLimitUnit" style="width: 90px;">
                    <el-option label="Mbps" value="Mbps" />
                    <el-option label="Gbps" value="Gbps" />
                  </el-select>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t.checkProtocol">
              <el-select v-model="temp.check_protocol" style="width: 200px;">
                <el-option label="HTTP" value="http" />
                <el-option label="HTTPS" value="https" />
                <el-option label="TCP" value="tcp" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t.checkPort">
              <el-input v-model.number="temp.check_port" placeholder="80" />
            </el-form-item>
            <el-form-item :label="t.checkHost">
              <el-input v-model="temp.check_host" placeholder="example.com" />
            </el-form-item>
            <el-form-item :label="t.checkPath">
              <el-input v-model="temp.check_path" placeholder="/" />
            </el-form-item>
            <el-form-item :label="t.checkTimeout">
              <el-input v-model.number="temp.check_timeout" placeholder="5">
                <template #append>{{ t.seconds }}</template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t.checkAction">
              <el-select v-model="temp.check_action" style="width: 200px;">
                <el-option :label="t.disableAction" value="disable" />
                <el-option :label="t.switchAction" value="switch" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="temp.check_action === 'switch'" :label="t.checkNodeGroup">
              <el-input v-model="temp.check_node_group" placeholder="1" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t.addSubIp" name="sub_ips">
          <el-form style="margin-top: 20px;">
            <el-form-item :label="t.subIp" label-width="80px">
              <el-input
                v-model="tempIPs"
                :rows="5"
                type="textarea"
                :placeholder="t.oneLineOneIp"
              />
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">{{ t.cancel }}</el-button>
          <el-button type="primary" @click="dialogStatus === 'create' ? createData() : updateData()">
            {{ t.confirm }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search, Plus, Top, Bottom, ArrowDown, Monitor } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const t = {
  nodeKeyword: '\u8282\u70b9\u540d\u79f0 / IP',
  search: '\u641c\u7d22',
  addNode: '\u6dfb\u52a0\u8282\u70b9',
  enableSelected: '\u542f\u7528\u9009\u4e2d',
  disableSelected: '\u505c\u7528\u9009\u4e2d',
  deleteSelected: '\u5220\u9664\u9009\u4e2d',
  name: '\u540d\u79f0',
  group: '\u7ec4',
  groupPrefix: '\u5206\u7ec4',
  groupDefault: '\u9ed8\u8ba4',
  nodeIp: '\u8282\u70b9IP',
  fromIp: '\u4eceIP',
  monitor: '\u76d1\u63a7',
  bandwidth: '\u5e26\u5bbd',
  monthlyTraffic: '\u6708\u6d41\u91cf',
  status: '\u72b6\u6001',
  remark: '\u5907\u6ce8',
  sort: '\u6392\u5e8f',
  action: '\u64cd\u4f5c',
  setting: '\u8bbe\u7f6e',
  more: '\u66f4\u591a',
  delete: '\u5220\u9664',
  basicSettings: '\u57fa\u672c\u8bbe\u7f6e',
  nodeSettings: '\u8282\u70b9\u8bbe\u7f6e',
  autoDisable: '\u81ea\u52a8\u7981\u7528',
  autoDisableEnable: '\u81ea\u52a8\u7981\u7528',
  checkProtocol: '\u76d1\u6d4b\u534f\u8bae',
  checkTimeout: '\u76d1\u6d4b\u8d85\u65f6',
  checkPort: '\u76d1\u6d4b\u7aef\u53e3',
  checkHost: '\u76d1\u6d4b\u4e3b\u673a',
  checkPath: '\u76d1\u6d4b\u8def\u5f84',
  checkAction: '\u5f02\u5e38\u52a8\u4f5c',
  checkNodeGroup: '\u5207\u6362\u5206\u7ec4',
  disableAction: '\u7981\u7528\u8282\u70b9',
  switchAction: '\u5207\u6362\u5206\u7ec4',
  bwLimit: '\u5e26\u5bbd\u9650\u5236',
  seconds: '\u79d2',
  noConfig: '\u6682\u65e0\u914d\u7f6e',
  addSubIp: '\u6dfb\u52a0\u5b50IP',
  subIp: '\u5b50IP',
  oneLineOneIp: '\u4e00\u884c\u4e00\u4e2aIP',
  cancel: '\u53d6\u6d88',
  confirm: '\u786e\u8ba4',
  editNode: '\u7f16\u8f91\u8282\u70b9',
  createNode: '\u521b\u5efa\u65b0\u8282\u70b9',
  remarkPlaceholder: '\u8bf7\u8f93\u5165\u5907\u6ce8',
  publicIp: '\u516c\u7f51 IP',
  ipNote: '\u5982\u679c\u65b0\u65e7IP\u662f\u4e0d\u540c\u8282\u70b9\uff0c\u8bf7\u4f7f\u7528\u5f85\u521d\u59cb\u5316\u91cc\u7684\u66ff\u6362\u8282\u70b9\u529f\u80fd\u3002',
  nodeType: '\u7c7b\u578b',
  l1Edge: 'L1 \u8fb9\u7f18',
  l2Middle: 'L2 \u4e2d\u95f4',
  installed: '\u5df2\u5b89\u88c5',
  l1EdgeNode: 'L1\u8fb9\u7f18\u8282\u70b9',
  l2MiddleNode: 'L2\u4e2d\u95f4\u8282\u70b9',
  l1Desc: 'L1\u8fb9\u7f18\u8282\u70b9\u662f\u7528\u6237\u5b9e\u9645\u8bbf\u95ee\u7684\u8282\u70b9;',
  l2Desc: 'L2\u4e2d\u95f4\u8282\u70b9\u662fL1\u4e0e\u6e90\u670d\u52a1\u5668\u4e4b\u95f4\u7684\u8282\u70b9\uff0c\u7528\u4e8e\u6c47\u805aL1\u8282\u70b9\u8bf7\u6c42\uff0c\u63d0\u9ad8\u7f13\u5b58\u547d\u4e2d\u7387\uff0c\u6216\u4f18\u5316\u56de\u6e90\u7ebf\u8def\u3002',
  cacheDir: '\u7f13\u5b58\u76ee\u5f55',
  cacheLimit: '\u7f13\u5b58\u4e0a\u9650',
  logDir: '\u65e5\u5fd7\u76ee\u5f55',
  createSuccess: '\u521b\u5efa\u6210\u529f',
  updateSuccess: '\u66f4\u65b0\u6210\u529f',
  promptTitle: '\u63d0\u793a',
  batchConfirmPrefix: '\u786e\u5b9a\u8981\u6267\u884c',
  batchConfirmSuffix: '\u64cd\u4f5c\u5417\uff1f'
}

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
  update: t.editNode,
  create: t.createNode
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
  log_dir: '/usr/local/openresty/nginx/logs',
  host: '',
  port: 80,
  http_proxy: '',
  is_mgmt: false,
  check_on: false,
  check_protocol: 'http',
  check_timeout: 5,
  check_port: 80,
  check_host: '',
  check_path: '/',
  check_node_group: '',
  check_action: 'disable',
  bw_limit: ''
})

const tempIPs = ref('')
const bwLimitValue = ref('')
const bwLimitUnit = ref('Mbps')

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
  temp.host = ''
  temp.port = 80
  temp.http_proxy = ''
  temp.is_mgmt = false
  temp.check_on = false
  temp.check_protocol = 'http'
  temp.check_timeout = 5
  temp.check_port = 80
  temp.check_host = ''
  temp.check_path = '/'
  temp.check_node_group = ''
  temp.check_action = 'disable'
  temp.bw_limit = ''
  activeTab.value = 'basic'
  tempIPs.value = ''
  bwLimitValue.value = ''
  bwLimitUnit.value = 'Mbps'
}

const syncBwLimitFromTemp = () => {
  if (!temp.bw_limit) {
    bwLimitValue.value = ''
    bwLimitUnit.value = 'Mbps'
    return
  }
  const match = temp.bw_limit.match(/^(\d+(?:\.\d+)?)\s*(Mbps|Gbps)$/i)
  if (match) {
    bwLimitValue.value = match[1]
    bwLimitUnit.value = match[2]
  } else {
    bwLimitValue.value = temp.bw_limit
    bwLimitUnit.value = 'Mbps'
  }
}

const applyBwLimitToTemp = () => {
  const value = String(bwLimitValue.value || '').trim()
  if (!value) {
    temp.bw_limit = ''
    return
  }
  temp.bw_limit = `${value}${bwLimitUnit.value}`
}

const handleCreate = () => {
  resetTemp()
  syncBwLimitFromTemp()
  dialogStatus.value = 'create'
  dialogFormVisible.value = true
}

const createData = () => {
  applyBwLimitToTemp()
  const ipLines = tempIPs.value.split('\n').filter(i => i.trim() !== '')
  const subIPs = ipLines.map(ip => ({ ip: ip.trim() }))
  const payload = {
    ...temp,
    sub_ips: subIPs
  }
  request({
    url: '/nodes',
    method: 'post',
    data: payload
  }).then(() => {
    dialogFormVisible.value = false
    ElMessage.success(t.createSuccess)
    getList()
  })
}

const handleUpdate = row => {
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
  temp.host = row.host || ''
  temp.port = row.port || 80
  temp.http_proxy = row.http_proxy || ''
  temp.is_mgmt = row.is_mgmt || false
  temp.check_on = row.check_on || false
  temp.check_protocol = row.check_protocol || 'http'
  temp.check_timeout = row.check_timeout || 5
  temp.check_port = row.check_port || 80
  temp.check_host = row.check_host || ''
  temp.check_path = row.check_path || '/'
  temp.check_node_group = row.check_node_group || ''
  temp.check_action = row.check_action || 'disable'
  temp.bw_limit = row.bw_limit || ''

  tempIPs.value = row.sub_ips ? row.sub_ips.map(i => i.ip).join('\n') : ''

  syncBwLimitFromTemp()
  dialogStatus.value = 'update'
  dialogFormVisible.value = true
  activeTab.value = 'basic'
}

const updateData = () => {
  applyBwLimitToTemp()
  const ipLines = tempIPs.value.split('\n').filter(i => i.trim() !== '')
  const subIPs = ipLines.map(ip => ({ ip: ip.trim() }))

  const payload = {
    ...temp,
    sub_ips: subIPs
  }

  request({
    url: `/nodes/${temp.id}`,
    method: 'put',
    data: payload
  }).then(() => {
    dialogFormVisible.value = false
    ElMessage.success(t.updateSuccess)
    getList()
  })
}

const handleBatch = (action, rows) => {
  const listToProcess = rows || selectedRows.value
  if (listToProcess.length === 0) return

  const ids = listToProcess.map(row => row.id)

  ElMessageBox.confirm(`${t.batchConfirmPrefix}${action}${t.batchConfirmSuffix}`, t.promptTitle, {
    confirmButtonText: t.confirm,
    cancelButtonText: t.cancel,
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

const handleStatusChange = (row) => {
  const action = row.enable ? 'start' : 'stop'
  request({
    url: '/nodes/batch',
    method: 'post',
    data: { action, ids: [row.id] }
  }).then(res => {
    ElMessage.success(res.msg)
  }).catch(() => {
    row.enable = !row.enable // Revert on failure
  })
}

const handleSelectionChange = val => {
  selectedRows.value = val
}

onMounted(() => {
  getList()
})
</script>