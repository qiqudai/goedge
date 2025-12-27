<template>
  <div class="app-container">
    <el-tabs v-model="pageTab" class="node-tabs" @tab-click="handleTabChange">
      <el-tab-pane label="节点列表" name="list" />
      <el-tab-pane label="待初始化" name="pending" />
      <el-tab-pane label="区域管理" name="region" />
    </el-tabs>

    <div class="filter-container node-actions">
      <el-button type="primary" size="normal" @click="handleCreate">{{ t.installNode }}</el-button>
      <el-button size="normal" :disabled="!selectedRows.length" @click="handleBatch('stop')">{{ t.disableNode }}</el-button>
      <el-button size="normal" :disabled="!selectedRows.length" @click="handleBatch('start')">{{ t.enableNode }}</el-button>
      <el-button size="normal" @click="handleRefresh">{{ t.refresh }}</el-button>
      <el-dropdown trigger="click">
        <el-button size="normal">
          {{ t.moreAction }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item :disabled="!selectedRows.length" @click="handleBatch('delete')">{{ t.deleteSelected }}</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <div class="filter-container node-filters">
      <el-select v-model="listQuery.region_id" placeholder="All Regions" class="filter-item" style="width: 150px;">
        <el-option v-for="item in regionOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-select v-model="listQuery.status" placeholder="All Status" class="filter-item" style="width: 150px;">
        <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-select v-model="listQuery.node_type" placeholder="All Types" class="filter-item" style="width: 150px;">
        <el-option v-for="item in typeOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-input v-model="listQuery.keyword" :placeholder="t.nodeKeyword" class="filter-item" style="width: 240px;" @keyup.enter="handleFilter">
        <template #suffix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button type="primary" size="normal" class="filter-item" @click="handleFilter">{{ t.search }}</el-button>
      <el-button type="text" size="normal" class="filter-item" @click="resetFilters">{{ t.reset }}</el-button>
    </div>

    <AppTable
      :loading="listLoading"
      :data="list"
      v-model:current-page="listQuery.page"
      v-model:page-size="listQuery.pageSize"
      persist-key="list"
      :page-sizes="[10, 20, 30, 50]"
      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="getList"
      @current-change="getList"
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

      <el-table-column :label="t.name" min-width="140px">
        <template #default="{ row }">
          <div class="node-name" @click="handleUpdate(row)">{{ row.name }}</div>
        </template>
      </el-table-column>

      <el-table-column label="区域" min-width="140px">
        <template #default="{ row }">
          <span>{{ row.region_name || '默认' }}</span>
          <el-link type="primary" :underline="false" class="node-group-link" @click="goToNodeGroups">
            {{ t.lineGroup }}({{ row.group_count || 1 }}{{ t.groupCountUnit }})
          </el-link>
        </template>
      </el-table-column>

      <el-table-column :label="t.nodeIp" min-width="150px">
        <template #default="{ row }">
          <div>{{ row.ip }}</div>
          <div v-if="row.sub_ips && row.sub_ips.length > 0" style="color: #909399; font-size: 12px;">
            <el-popover placement="right" trigger="click" width="260">
              <div class="sub-ip-list">
                <div v-for="item in row.sub_ips" :key="item.ip || item">{{ item.ip || item }}</div>
              </div>
              <template #reference>
                <el-button link type="primary" size="normal">+{{ row.sub_ips.length }} {{ t.fromIp }}</el-button>
              </template>
            </el-popover>
          </div>
        </template>
      </el-table-column>

      <el-table-column :label="t.monitor" min-width="120px" align="center">
        <template #default="{ row }">
          <span class="monitor-protocol">{{ formatMonitorProtocol(row) }}</span>
          <el-link type="primary" :underline="false" @click="openMonitorLogs(row)">{{ t.monitorLog }}</el-link>
        </template>
      </el-table-column>

      <el-table-column :label="t.bandwidth" min-width="140px" align="center">
        <template #default="{ row }">
          <span class="clickable-text" @click="goToRealtimeMonitor(row)">{{ formatBandwidth(row) }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="t.monthlyTraffic" min-width="120px" align="center">
        <template #default="{ row }">
          <div class="clickable-text" @click="goToRealtimeMonitor(row)">
            <div>{{ formatMonthlyTraffic(row).up }}</div>
            <div>{{ formatMonthlyTraffic(row).down }}</div>
          </div>
        </template>
      </el-table-column>

      <el-table-column :label="t.status" align="center" width="90">
        <template #default="{ row }">
          <div class="node-status">
            <span :class="['status-dot', isNodeOnline(row) ? 'status-ok' : 'status-stop']"></span>
            <span>{{ isNodeOnline(row) ? t.statusOnline : t.statusOffline }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="启用" align="center" width="90">
        <template #default="{ row }">
          <el-switch v-model="row.enable" :active-value="true" :inactive-value="false" @change="handleStatusChange(row)" />
        </template>
      </el-table-column>

      <el-table-column :label="t.remark" prop="remark" min-width="100px" show-overflow-tooltip />

      <el-table-column :label="t.sort" prop="sort_order" width="80" align="center" sortable />

      <el-table-column :label="t.action" align="center" width="160" class-name="small-padding fixed-width">
        <template #default="{ row }">
          <div class="action-row">
            <el-button link type="primary" size="normal" @click="handleUpdate(row)">{{ t.manage }}</el-button>
            <el-dropdown trigger="click">
              <el-button link type="primary" size="normal">
                {{ t.more }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="handleBatch('delete', [row])">{{ t.delete }}</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>
      </el-table-column>
    </AppTable>


    <el-dialog title="监控日志" v-model="monitorDialogVisible" width="680px">
      <el-form :inline="true" class="monitor-form">
        <el-form-item label="日志查看">
          <el-select v-model="monitorQuery.type" style="width: 200px;">
            <el-option v-for="item in monitorTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="Time Range">
          <el-date-picker
            v-model="monitorQuery.timeRange"
            type="datetimerange"
            range-separator="To"
            start-placeholder="Start"
            end-placeholder="End"
            value-format="YYYY-MM-DD HH:mm:ss"
            clearable
            style="width: 260px;"
          />
        </el-form-item>
        <el-form-item label="Time Range">
          <el-select v-model="monitorQuery.group" style="width: 180px;">
            <el-option v-for="item in monitorGroupOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <AppTable
        :data="monitorLogs"
        v-model:current-page="monitorQuery.page"
        v-model:page-size="monitorQuery.pageSize"
        persist-key="monitor"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        :total="monitorTotal"
        @current-change="loadMonitorLogs"
        @size-change="loadMonitorLogs"
        border
      >
        <el-table-column prop="checked_at" label="Checked At" min-width="140" />
        <el-table-column prop="fail_count" label="失败个数" width="100" align="center" />
        <el-table-column prop="total_count" label="总检测点" width="100" align="center" />
      </AppTable>
    </el-dialog>

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
          <el-button size="normal" @click="dialogFormVisible = false">{{ t.cancel }}</el-button>
          <el-button size="normal" type="primary" @click="dialogStatus === 'create' ? createData() : updateData()">
            {{ t.confirm }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, ArrowDown } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const t = {
  nodeKeyword: '\u8282\u70b9\u540d\u79f0 / IP',
  search: '\u641c\u7d22',
  reset: '\u6e05\u9664',
  refresh: '\u5237\u65b0',
  installNode: '\u5b89\u88c5\u8282\u70b9',
  enableNode: '\u542f\u7528\u8282\u70b9',
  disableNode: '\u7981\u7528\u8282\u70b9',
  moreAction: '\u66f4\u591a\u64cd\u4f5c',
  deleteSelected: '\u5220\u9664\u9009\u4e2d',
  name: '\u540d\u79f0',
  lineGroup: '\u7ebf\u8def\u7ec4',
  groupCountUnit: '\u4e2a',
  nodeIp: '\u8282\u70b9IP',
  fromIp: '\u4eceIP',
  monitor: '\u76d1\u63a7',
  monitorLog: '\u65e5\u5fd7',
  bandwidth: '\u5e26\u5bbd',
  monthlyTraffic: '\u6708\u6d41\u91cf',
  status: '\u72b6\u6001',
  statusOnline: '\u5728\u7ebf',
  statusOffline: '\u4e0d\u5728\u7ebf',
  remark: '\u5907\u6ce8',
  sort: '\u6392\u5e8f',
  action: '\u64cd\u4f5c',
  manage: '\u7ba1\u7406',
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
const pageTab = ref('list')
const listQuery = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  region_id: '',
  status: '',
  node_type: ''
})
const regionOptions = ref([{ label: '\u6240\u6709\u533a\u57df', value: '' }])
const statusOptions = ref([
  { label: '\u6240\u6709\u72b6\u6001', value: '' },
  { label: '\u6b63\u5e38', value: 'enabled' },
  { label: '\u7981\u7528', value: 'disabled' }
])
const typeOptions = ref([
  { label: '\u6240\u6709\u7c7b\u578b', value: '' },
  { label: 'L1\u8fb9\u7f18', value: 1 },
  { label: 'L2\u4e2d\u95f4', value: 2 }
])

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
const monitorDialogVisible = ref(false)
const monitorLogs = ref([])
const monitorTotal = ref(0)
const monitorQuery = reactive({
  type: 'availability',
  timeRange: [],
  group: 'all',
  page: 1,
  pageSize: 10
})
const monitorTypeOptions = [
  { label: '\u53ef\u7528\u6027\u76d1\u63a7\u65e5\u5fd7', value: 'availability' }
]
const monitorGroupOptions = [
  { label: '\u6240\u6709\u76d1\u63a7\u7ec4', value: 'all' }
]

const getList = () => {
  listLoading.value = true
  request({
    url: '/nodes',
    method: 'get',
    params: {
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      keyword: listQuery.keyword,
      region_id: listQuery.region_id,
      status: listQuery.status,
      node_type: listQuery.node_type,
      tab: pageTab.value
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

const handleRefresh = () => {
  getList()
}

const resetFilters = () => {
  listQuery.region_id = ''
  listQuery.status = ''
  listQuery.node_type = ''
  listQuery.keyword = ''
  handleFilter()
}

const handleTabChange = () => {
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

const goToNodeGroups = () => {
  router.push('/node/groups')
}

const goToRealtimeMonitor = () => {
  router.push('/node/realtime')
}

const pickNumber = (row, keys) => {
  for (const key of keys) {
    const value = row[key]
    if (value !== undefined && value !== null && value !== '') {
      const num = Number(value)
      if (!Number.isNaN(num)) {
        return num
      }
    }
  }
  return 0
}

const formatSpeed = (value) => {
  const num = Number(value)
  if (!num) {
    return '0 Kbps'
  }
  const units = ['Kbps', 'Mbps', 'Gbps', 'Tbps']
  let size = num
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }
  const display = size >= 10 ? size.toFixed(1) : size.toFixed(2)
  return `${display} ${units[unitIndex]}`
}

const formatMonitorProtocol = (row) => {
  if (row.check_protocol) {
    return row.check_protocol
  }
  return 'tcp'
}

const formatBandwidth = (row) => {
  const up = pickNumber(row, ['bandwidth_in', 'bw_in', 'in_speed'])
  const down = pickNumber(row, ['bandwidth_out', 'bw_out', 'out_speed'])
  return `${formatSpeed(up)} �?${formatSpeed(down)} ↓`
}

const formatMonthlyTraffic = (row) => {
  const up = pickNumber(row, ['month_traffic_in', 'month_in', 'up_traffic', 'traffic_in'])
  const down = pickNumber(row, ['month_traffic_out', 'month_out', 'down_traffic', 'traffic_out'])
  return {
    up: formatSpeed(up),
    down: formatSpeed(down)
  }
}

const isNodeOnline = (row) => {
  if (typeof row.online === 'boolean') {
    return row.online
  }
  if (typeof row.is_online === 'boolean') {
    return row.is_online
  }
  if (typeof row.status === 'string') {
    return row.status === 'online'
  }
  if (typeof row.state === 'string') {
    return row.state === 'online'
  }
  return false
}

const buildMonitorLogs = () => {
  const rows = []
  const now = new Date()
  for (let i = 0; i < monitorQuery.pageSize; i += 1) {
    const at = new Date(now.getTime() - i * 60 * 1000)
    rows.push({
      checked_at: `${at.getMonth() + 1}-${String(at.getDate()).padStart(2, '0')} ${String(at.getHours()).padStart(2, '0')}:${String(at.getMinutes()).padStart(2, '0')}:${String(at.getSeconds()).padStart(2, '0')}`,
      fail_count: 0,
      total_count: 6
    })
  }
  monitorLogs.value = rows
  monitorTotal.value = 20
}

const loadMonitorLogs = () => {
  buildMonitorLogs()
}

const openMonitorLogs = () => {
  monitorQuery.page = 1
  loadMonitorLogs()
  monitorDialogVisible.value = true
}



onMounted(() => {
  getList()
})
</script>

<style scoped>
.node-tabs {
  margin-bottom: 12px;
}

.node-actions,
.node-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}

.node-name {
  font-weight: 600;
  cursor: pointer;
  color: #409eff;
}

.node-group-link {
  margin-left: 6px;
  font-size: 12px;
}

.monitor-protocol {
  color: #409eff;
  margin-right: 6px;
}

.clickable-text {
  color: #409eff;
  cursor: pointer;
}

.node-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.action-row {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  display: inline-block;
}

.status-ok {
  background: #67c23a;
}

.status-stop {
  background: #f56c6c;
}

.sub-ip-list {
  max-height: 220px;
  overflow-y: auto;
  line-height: 1.6;
  color: #606266;
}
</style>







