<template>
  <div class="app-container">
    <el-tabs v-model="activeTopTab" class="site-tabs" @tab-click="handleTopTab">
      <el-tab-pane label="转发列表" name="list" />
      <el-tab-pane label="分组设置" name="groups" />
      <el-tab-pane label="默认设置" name="default" />
      <el-tab-pane label="实时监控" name="monitor" />
    </el-tabs>
    <div class="filter-container">
      <div class="filter-left">
        <el-button type="primary" @click="openCreateDialog">添加转发</el-button>
        <el-button :disabled="!selectedRows.length" @click="openBatchEdit">批量修改</el-button>
        <el-dropdown trigger="click">
          <el-button>
            更多操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="handleBatchAction('enable')">启用</el-dropdown-item>
              <el-dropdown-item @click="handleBatchAction('disable')">禁用</el-dropdown-item>
              <el-dropdown-item @click="handleBatchAction('delete')">删除</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>

      <div class="filter-right">
        <el-select v-model="listQuery.searchField" class="filter-item" style="width: 120px;">
          <el-option label="全字段" value="all" />
          <el-option label="监听端口" value="listen" />
          <el-option label="源站" value="origin" />
          <el-option label="CNAME" value="cname" />
          <el-option label="用户" value="user" />
        </el-select>
        <el-input
          v-model="listQuery.keyword"
          placeholder="输入监听端口"
          style="width: 260px;"
          class="filter-item"
          @keyup.enter="handleFilter"
        />
        <el-button type="primary" class="filter-item" @click="handleFilter">查询</el-button>
        <el-button link class="filter-item" @click="advancedVisible = true">高级搜索</el-button>
      </div>
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
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="user_name" label="用户" width="120" />
      <el-table-column prop="listen_ports" label="监听端口" width="120" />
      <el-table-column prop="origin_display" label="源站" min-width="200" show-overflow-tooltip />
      <el-table-column prop="user_package_name" label="套餐" min-width="140" show-overflow-tooltip />
      <el-table-column prop="group_name" label="分组" width="120" />
      <el-table-column prop="node_group_name" label="区域(线路组)" min-width="140" show-overflow-tooltip />
      <el-table-column prop="cname" label="CNAME" min-width="200" show-overflow-tooltip />
      <el-table-column label="状态" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'">{{ row.status ? '正常' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
      <el-table-column prop="created_at" label="添加时间" width="180" />
      <el-table-column label="操作" width="140" align="center">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="openEdit(row)">管理</el-button>
          <el-dropdown trigger="click">
            <span class="link-more">
              更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleRowAction('enable', row)">启用</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('disable', row)">禁用</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('delete', row)">删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-container">
      <el-pagination
        v-model:current-page="listQuery.page"
        v-model:page-size="listQuery.pageSize"

        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="handleFilter"
        @current-change="handleFilter"
      />
    </div>

    <el-dialog v-model="createVisible" width="620px" title="添加转发">
      <el-tabs v-model="createTab" type="card">
        <el-tab-pane label="单个" name="single">
          <el-form :model="createForm" label-width="90px">
            <el-form-item label="用户选择">
              <el-select
                v-model="createForm.user_id"
                filterable
                remote
                clearable
                reserve-keyword
                placeholder="输入ID、邮箱、用户名、手机号搜索"
                :remote-method="searchUsers"
                :loading="userLoading"
                style="width: 100%;"
                @change="loadUserPackages"
              >
                <el-option v-for="u in userOptions" :key="u.id" :label="`${u.name} (${u.id})`" :value="u.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="用户套餐">
              <el-select v-model="createForm.user_package_id" clearable placeholder="请选择" style="width: 100%;">
                <el-option v-for="p in userPackageOptions" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="监听端口">
              <el-input v-model="createForm.listen_ports_input" placeholder="88, 99/udp或88/tcp, 多个端口空格分隔" />
            </el-form-item>
            <el-form-item label="源站地址端口">
              <el-input v-model="createForm.origin_input" placeholder="1.1.1.1:99或www.abc.com:99" />
            </el-form-item>
            <div class="expand-more" @click="createMore = !createMore">
              <span>展开更多</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <div v-if="createMore" class="extra-fields">
              <el-form-item label="所属分组">
                <el-select v-model="createForm.group_id" clearable placeholder="转发分组, 可不选" style="width: 100%;">
                  <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="备注">
                <el-input v-model="createForm.remark" placeholder="输入备注信息" />
              </el-form-item>
            </div>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="批量" name="batch">
          <el-form :model="batchForm" label-width="90px">
            <el-form-item label="用户选择">
              <el-select
                v-model="batchForm.user_id"
                filterable
                remote
                clearable
                reserve-keyword
                placeholder="输入ID、邮箱、用户名、手机号搜索"
                :remote-method="searchUsers"
                :loading="userLoading"
                style="width: 100%;"
                @change="loadUserPackages"
              >
                <el-option v-for="u in userOptions" :key="u.id" :label="`${u.name} (${u.id})`" :value="u.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="用户套餐">
              <el-select v-model="batchForm.user_package_id" clearable placeholder="请选择" style="width: 100%;">
                <el-option v-for="p in userPackageOptions" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="转发数据">
              <el-input
                v-model="batchForm.data"
                type="textarea"
                rows="5"
                placeholder="格式为: 监听端口|IP|回源端口&#10;88|1.2.3.4|8080&#10;77|6.6.8.8|8080"
              />
            </el-form-item>
            <el-form-item label="忽略错误">
              <el-switch v-model="batchForm.ignore_error" />
              <span class="help-text">有转发添加出错时，不中断，继续添加下一条。</span>
            </el-form-item>
            <div class="expand-more" @click="batchMore = !batchMore">
              <span>展开更多</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <div v-if="batchMore" class="extra-fields">
              <el-form-item label="所属分组">
                <el-select v-model="batchForm.group_id" clearable placeholder="转发分组, 可不选" style="width: 100%;">
                  <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="备注">
                <el-input v-model="batchForm.remark" placeholder="输入备注信息" />
              </el-form-item>
            </div>
          </el-form>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateSubmit">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="batchEditVisible" title="批量修改转发" width="720px">
      <div class="batch-header">正在修改的转发: {{ selectedIdsText }}</div>
      <div class="batch-dialog-body">
        <el-form label-width="90px">
          <el-collapse v-model="batchCollapse">
            <el-collapse-item title="基本设置" name="basic">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.user_package_id">套餐</el-checkbox>
                <el-select v-model="batchEditForm.user_package_id" clearable placeholder="请选择" style="width: 70%;">
                  <el-option v-for="p in userPackageOptions" :key="p.id" :label="p.name" :value="p.id" />
                </el-select>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.group_id">所属分组</el-checkbox>
                <el-select v-model="batchEditForm.group_id" clearable placeholder="请选择" style="width: 70%;">
                  <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
                </el-select>
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="源站设置" name="origin">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.balance_way">负载方式</el-checkbox>
                <el-radio-group v-model="batchEditForm.balance_way">
                  <el-radio value="rr">轮循</el-radio>
                  <el-radio value="ip_hash">定源</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.backsource_port">回源端口</el-checkbox>
                <el-input v-model="batchEditForm.backsource_port" placeholder="请输入回源端口" style="width: 70%;" />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.proxy_protocol">Proxy Protocol</el-checkbox>
                <el-radio-group v-model="batchEditForm.proxy_protocol">
                  <el-radio :value="true">开启</el-radio>
                  <el-radio :value="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.origins">源站列表</el-checkbox>
                <el-button size="small" type="primary" @click="openOriginDialog">新增源站信息</el-button>
              </div>
              <el-table v-if="batchEditForm.origins.length" :data="batchEditForm.origins" border size="small">
                <el-table-column label="源地址">
                  <template #default="{ row }">
                    <el-input v-model="row.address" placeholder="IP或域名" />
                  </template>
                </el-table-column>
                <el-table-column label="权重" width="100">
                  <template #default="{ row }">
                    <el-input v-model="row.weight" />
                  </template>
                </el-table-column>
                <el-table-column label="状态" width="120">
                  <template #default="{ row }">
                    <el-switch v-model="row.enable" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeOrigin($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="访问控制" name="access">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.acl_default">ACL默认行为</el-checkbox>
                <el-radio-group v-model="batchEditForm.acl_default">
                  <el-radio value="allow">允许</el-radio>
                  <el-radio value="deny">拒绝</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.acl_rules">ACL规则</el-checkbox>
                <el-button size="small" type="primary" @click="addAclRule">新增规则</el-button>
              </div>
              <el-table v-if="batchEditForm.acl_rules.length" :data="batchEditForm.acl_rules" border size="small">
                <el-table-column label="IP">
                  <template #default="{ row }">
                    <el-input v-model="row.ip" placeholder="IP" />
                  </template>
                </el-table-column>
                <el-table-column label="行为" width="120">
                  <template #default="{ row }">
                    <el-select v-model="row.action" placeholder="行为" style="width: 100%;">
                      <el-option label="允许" value="allow" />
                      <el-option label="拒绝" value="deny" />
                    </el-select>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeAclRule($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="batch-row" style="margin-top: 16px;">
                <el-checkbox v-model="batchEditChecks.region_block">区域屏蔽</el-checkbox>
                <el-radio-group v-model="batchEditForm.region_mode">
                  <el-radio value="none">不设置</el-radio>
                  <el-radio value="overseas_without_hk">国外(不包括港澳台)</el-radio>
                  <el-radio value="overseas_with_hk">国外(包括港澳台)</el-radio>
                  <el-radio value="china_with_hk">中国(包括港澳台)</el-radio>
                  <el-radio value="china_without_hk">中国(不包括港澳台)</el-radio>
                  <el-radio value="custom">自定义</el-radio>
                </el-radio-group>
              </div>
              <country-selector v-if="batchEditForm.region_mode === 'custom'" v-model="batchEditForm.region_custom" />

              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.ipv6">IPv6开启</el-checkbox>
                <el-radio-group v-model="batchEditForm.ipv6">
                  <el-radio :value="true">开启</el-radio>
                  <el-radio :value="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>
          </el-collapse>
        </el-form>
      </div>
    </el-dialog>

    <el-dialog v-model="originDialogVisible" title="新增源站信息" width="520px">
      <el-form :model="originForm" label-width="80px">
        <el-form-item label="源地址">
          <el-input v-model="originForm.address" placeholder="请输入ip或域名" />
        </el-form-item>
        <el-form-item label="权重">
          <el-input v-model="originForm.weight" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="originForm.enable" style="width: 100%;">
            <el-option label="上线" :value="true" />
            <el-option label="下线" :value="false" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="originDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmOrigin">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="advancedVisible" title="高级搜索" width="520px">
      <el-form :model="advancedForm" label-width="90px">
        <el-form-item label="用户">
          <el-select
            v-model="advancedForm.user_id"
            filterable
            remote
            clearable
            reserve-keyword
            placeholder="输入ID、邮箱、用户名、手机号搜索"
            :remote-method="searchUsers"
            :loading="userLoading"
            style="width: 100%;"
          >
            <el-option v-for="u in userOptions" :key="u.id" :label="`${u.name} (${u.id})`" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="套餐">
          <el-select v-model="advancedForm.user_package_id" clearable placeholder="请选择" style="width: 100%;">
            <el-option v-for="p in userPackageOptions" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="分组">
          <el-select v-model="advancedForm.group_id" clearable placeholder="请选择" style="width: 100%;">
            <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="advancedVisible = false">取消</el-button>
        <el-button type="primary" @click="applyAdvancedFilter">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown } from '@element-plus/icons-vue'
import request from '@/utils/request'
import CountrySelector from '@/components/CountrySelector.vue'

const router = useRouter()
const activeTopTab = ref('list')
const handleTopTab = tab => {
  const name = typeof tab === 'string' ? tab : tab?.paneName
  const map = {
    list: '/forward/list',
    groups: '/forward/groups',
    default: '/forward/default',
    monitor: '/forward/monitor'
  }
  const path = map[name]
  if (path) {
    router.push(path)
  }
}
const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const selectedRows = ref([])

const listQuery = reactive({
  page: 1,
  pageSize: 10,
  keyword: '',
  searchField: 'listen'
})

const advancedVisible = ref(false)
const advancedForm = reactive({
  user_id: '',
  user_package_id: '',
  group_id: ''
})

const createVisible = ref(false)
const createTab = ref('single')
const createMore = ref(false)
const batchMore = ref(false)

const createForm = reactive({
  user_id: '',
  user_package_id: '',
  group_id: '',
  listen_ports_input: '',
  origin_input: '',
  remark: ''
})

const batchForm = reactive({
  user_id: '',
  user_package_id: '',
  group_id: '',
  data: '',
  ignore_error: false,
  remark: ''
})

const batchEditVisible = ref(false)
const batchCollapse = ref(['basic'])
const batchEditForm = reactive({
  user_package_id: '',
  group_id: '',
  balance_way: 'rr',
  backsource_port: '',
  proxy_protocol: true,
  origins: [],
  acl_default: 'allow',
  acl_rules: [],
  region_mode: 'none',
  region_custom: [],
  ipv6: false
})
const batchEditChecks = reactive({
  user_package_id: false,
  group_id: false,
  balance_way: false,
  backsource_port: false,
  proxy_protocol: false,
  origins: false,
  acl_default: false,
  acl_rules: false,
  region_block: false,
  ipv6: false
})

const originDialogVisible = ref(false)
const originForm = reactive({
  address: '',
  weight: '1',
  enable: true
})

const userOptions = ref([])
const userLoading = ref(false)
const userPackageOptions = ref([])
const groupOptions = ref([])

const selectedIdsText = computed(() => selectedRows.value.map(row => row.id).join(','))

const fetchList = () => {
  listLoading.value = true
  request.get('/forwards', {
    params: {
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      keyword: listQuery.keyword,
      search_field: listQuery.searchField,
      user_id: advancedForm.user_id || undefined,
      user_package_id: advancedForm.user_package_id || undefined,
      group_id: advancedForm.group_id || undefined
    }
  }).then(res => {
    list.value = res.list || res.data || []
    total.value = res.total || 0
    listLoading.value = false
  }).catch(() => {
    listLoading.value = false
  })
}

const handleFilter = () => {
  listQuery.page = 1
  fetchList()
}

const handleSelectionChange = rows => {
  selectedRows.value = rows
}

const openCreateDialog = () => {
  createVisible.value = true
  createTab.value = 'single'
}

const openBatchEdit = () => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择转发')
    return
  }
  batchEditVisible.value = true
}

const openEdit = row => {
  selectedRows.value = [row]
  batchEditVisible.value = true
}

const handleCreateSubmit = () => {
  if (createTab.value === 'single') {
    request.post('/forwards', {
      user_id: createForm.user_id,
      user_package_id: createForm.user_package_id,
      group_id: createForm.group_id,
      listen_ports_input: createForm.listen_ports_input,
      origin_input: createForm.origin_input,
      remark: createForm.remark
    }).then(() => {
      ElMessage.success('创建成功')
      createVisible.value = false
      fetchList()
    })
  } else {
    request.post('/forwards/batch', batchForm).then(res => {
      ElMessage.success(res.message || '批量创建完成')
      createVisible.value = false
      fetchList()
    })
  }
}

const submitBatchEdit = () => {
  const ids = selectedRows.value.map(row => row.id)
  if (!ids.length) {
    ElMessage.warning('请选择转发')
    return
  }
  const payload = { ids }
  if (batchEditChecks.user_package_id) payload.user_package_id = batchEditForm.user_package_id || 0
  if (batchEditChecks.group_id) payload.group_id = batchEditForm.group_id || 0

  const settings = {}
  if (batchEditChecks.balance_way || batchEditChecks.backsource_port || batchEditChecks.proxy_protocol || batchEditChecks.origins) {
    settings.origin = {
      balance_way: batchEditForm.balance_way,
      backsource_port: batchEditForm.backsource_port,
      proxy_protocol: batchEditForm.proxy_protocol,
      origins: batchEditForm.origins
    }
  }
  if (batchEditChecks.acl_default || batchEditChecks.acl_rules || batchEditChecks.region_block || batchEditChecks.ipv6) {
    settings.access = {
      acl_default: batchEditForm.acl_default,
      acl_rules: batchEditForm.acl_rules,
      region_mode: batchEditForm.region_mode,
      region_custom: batchEditForm.region_custom,
      ipv6: batchEditForm.ipv6
    }
  }
  if (Object.keys(settings).length) {
    payload.settings = settings
  }

  request.post('/forwards/batch_update', payload).then(() => {
    ElMessage.success('批量修改完成')
    batchEditVisible.value = false
    fetchList()
  })
}

const handleBatchAction = action => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择转发')
    return
  }
  const ids = selectedRows.value.map(row => row.id)
  ElMessageBox.confirm(`确定执行${action}操作?`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.post('/forwards/batch_action', { action, ids }).then(res => {
      ElMessage.success(res.message || '操作成功')
      fetchList()
    })
  })
}

const handleRowAction = (action, row) => {
  selectedRows.value = [row]
  handleBatchAction(action)
}

const searchUsers = keyword => {
  if (!keyword) {
    userOptions.value = []
    return
  }
  userLoading.value = true
  request.get('/users', { params: { keyword, pageSize: 20 } }).then(res => {
    userOptions.value = res.data?.list || res.list || []
    userLoading.value = false
  }).catch(() => {
    userLoading.value = false
  })
}

const loadUserPackages = userId => {
  const params = userId ? { user_id: userId } : {}
  request.get('/user_packages', { params }).then(res => {
    userPackageOptions.value = res.data?.list || res.list || []
  })
}

const loadGroups = () => {
  request.get('/forward_groups').then(res => {
    groupOptions.value = res.data?.list || res.list || []
  })
}

const openOriginDialog = () => {
  originForm.address = ''
  originForm.weight = '1'
  originForm.enable = true
  originDialogVisible.value = true
}

const confirmOrigin = () => {
  if (!originForm.address) {
    ElMessage.warning('请输入源地址')
    return
  }
  batchEditForm.origins.push({
    address: originForm.address,
    weight: Number(originForm.weight) || 1,
    enable: originForm.enable
  })
  originDialogVisible.value = false
}

const removeOrigin = index => {
  batchEditForm.origins.splice(index, 1)
}

const addAclRule = () => {
  batchEditForm.acl_rules.push({ ip: '', action: 'allow' })
}

const removeAclRule = index => {
  batchEditForm.acl_rules.splice(index, 1)
}

const applyAdvancedFilter = () => {
  advancedVisible.value = false
  handleFilter()
}

onMounted(() => {
  fetchList()
  loadGroups()
  loadUserPackages()
})
</script>

<style scoped>
.filter-container {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.filter-left,
.filter-right {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.pagination-container {
  margin-top: 16px;
  text-align: right;
}
.expand-more {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #409eff;
  cursor: pointer;
  margin: 6px 0 10px;
}
.extra-fields {
  padding-top: 4px;
}
.help-text {
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
}
.batch-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.batch-header {
  margin-bottom: 12px;
  color: #606266;
  font-size: 12px;
}
.batch-action {
  display: flex;
  justify-content: center;
  margin-top: 12px;
}
.batch-dialog-body {
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 8px;
}
.link-more {
  color: #409eff;
  cursor: pointer;
  font-size: 12px;
  margin-left: 8px;
}
</style>







