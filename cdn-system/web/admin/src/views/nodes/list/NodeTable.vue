<template>
  <div class="filter-container node-actions">
    <el-button type="primary" @click="$emit('create')">{{ NODE_T.installNode }}</el-button>
    <el-button :disabled="!selectedRows.length" @click="$emit('batch', 'stop')">{{ NODE_T.disableNode }}</el-button>
    <el-button :disabled="!selectedRows.length" @click="$emit('batch', 'start')">{{ NODE_T.enableNode }}</el-button>
    <el-button @click="$emit('refresh')">{{ NODE_T.refresh }}</el-button>
    <el-dropdown trigger="click" @command="(c) => $emit('batch', c)">
      <el-button>
        {{ NODE_T.moreAction }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="delete" :disabled="!selectedRows.length">{{ NODE_T.deleteSelected }}</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>

  <div class="filter-container node-filters">
    <el-select v-model="query.region_id" :placeholder="NODE_T.allRegions" class="filter-item" style="width: 150px;">
      <el-option :label="NODE_T.allRegions" value="" />
      <el-option v-for="item in regions" :key="item.id" :label="item.name" :value="item.id" />
    </el-select>
    <el-select v-model="query.status" :placeholder="NODE_T.allStatus" class="filter-item" style="width: 150px;">
      <el-option :label="NODE_T.allStatus" value="" />
      <el-option v-for="item in STATUS_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
    </el-select>
    <el-select v-model="query.node_type" :placeholder="NODE_T.allTypes" class="filter-item" style="width: 150px;">
      <el-option :label="NODE_T.allTypes" value="" />
      <el-option v-for="item in TYPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
    </el-select>
    <el-input v-model="query.keyword" :placeholder="NODE_T.nodeKeyword" class="filter-item" style="width: 240px;" @keyup.enter="handleSearch">
      <template #suffix><el-icon><Search /></el-icon></template>
    </el-input>
    <el-button type="primary" class="filter-item" @click="handleSearch">{{ NODE_T.search }}</el-button>
    <el-button link type="primary" class="filter-item" @click="resetFilters">{{ NODE_T.reset }}</el-button>
  </div>

  <AppTable
    ref="tableRef"
    :loading="loading"
    :data="list"
    v-model:current-page="query.page"
    v-model:page-size="query.pageSize"
    storage-key="node-list-table"
    persist-key="node-list-table"
    :total="total"
    border
    fit
    highlight-current-row
    style="width: 100%;"
    @selection-change="(rows) => $emit('selection-change', rows)"
    @current-change="handleSearch"
    @size-change="handleSearch"
  >
    <el-table-column type="selection" width="55" align="center" />
    <el-table-column label="ID" prop="id" sortable="custom" align="center" width="80" />
    <el-table-column :label="NODE_T.name" min-width="140px">
      <template #default="{ row }">
        <div class="node-name-link" @click="$emit('edit', row)">{{ row.name }}</div>
      </template>
    </el-table-column>
    <el-table-column :label="NODE_T.region" min-width="140px">
      <template #default="{ row }">
        <span>{{ row.region_name || NODE_T.defaultRegion }}</span>
        <el-link type="primary" underline="never" class="node-group-link" @click="$emit('go-groups', row)">
          {{ NODE_T.lineGroup }}({{ row.group_count || 1 }}{{ NODE_T.groupCountUnit }})
        </el-link>
      </template>
    </el-table-column>
    <el-table-column :label="NODE_T.nodeIp" min-width="150px">
      <template #default="{ row }">
        <div>{{ row.ip }}</div>
        <div v-if="row.sub_ips?.length" class="sub-ips">
          <el-popover placement="right" trigger="click" width="260">
            <template #reference><el-button link type="primary">+{{ row.sub_ips.length }} {{ NODE_T.fromIp }}</el-button></template>
            <div class="ip-list">
               <div v-for="ip in row.sub_ips" :key="ip.ip || ip">{{ ip.ip || ip }}</div>
            </div>
          </el-popover>
        </div>
      </template>
    </el-table-column>
    <el-table-column :label="NODE_T.monitor" min-width="120px" align="center">
      <template #default="{ row }">
        <span class="monitor-protocol">{{ row.check_protocol?.toUpperCase() || 'HTTP' }}</span>
        <el-link type="primary" underline="never" @click="$emit('monitor-logs', row)">{{ NODE_T.monitorLog }}</el-link>
      </template>
    </el-table-column>
    <el-table-column :label="NODE_T.bandwidth" min-width="140px" align="center">
      <template #default="{ row }">
         <span class="clickable-text" @click="$emit('go-monitor', row)">{{ row.bandwidth_display || '0bps' }}</span>
      </template>
    </el-table-column>
    <el-table-column :label="NODE_T.status" align="center" width="90">
       <template #default="{ row }">
          <div class="node-status-wrap">
             <span :class="['status-dot', row.status_class]"></span>
             <span>{{ row.status_text }}</span>
          </div>
       </template>
    </el-table-column>
    <el-table-column :label="NODE_T.installStatus" align="center" min-width="120">
      <template #default="{ row }">
        <el-popover
          v-if="row.install_status === 'failed' && row.install_error"
          placement="right"
          trigger="hover"
          width="320"
        >
          <template #reference>
            <span class="install-status failed">{{ resolveInstallStatus(row.install_status) }}</span>
          </template>
          <div class="install-error">{{ row.install_error }}</div>
        </el-popover>
        <span v-else :class="['install-status', row.install_status]">
          <el-icon v-if="row.install_status === 'running'" class="install-loading"><Loading /></el-icon>
          {{ resolveInstallStatus(row.install_status) }}
          <span v-if="row.install_status === 'running' && Number.isFinite(row.install_progress)" class="install-progress">
            {{ row.install_progress }}%
          </span>
        </span>
      </template>
    </el-table-column>
    <el-table-column :label="NODE_T.enableLabel" align="center" width="90">
      <template #default="{ row }">
        <el-switch v-model="row.enable" @change="$emit('status-change', row)" />
      </template>
    </el-table-column>
    <el-table-column :label="NODE_T.remark" prop="remark" min-width="100px" show-overflow-tooltip />
    <el-table-column :label="NODE_T.sort" prop="sort_order" width="80" align="center" sortable />
    <el-table-column :label="NODE_T.action" align="center" width="160" fixed="right">
      <template #default="{ row }">
        <div style="display: flex; justify-content: center; gap: 8px;">
          <el-button link type="primary" @click="$emit('edit', row)">{{ NODE_T.manage }}</el-button>
          <el-dropdown trigger="click" @command="(c) => $emit('row-action', c, row)">
            <span class="link-more">{{ NODE_T.more }}<el-icon><ArrowDown /></el-icon></span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="install" :disabled="!row.ssh_user">{{ NODE_T.reinstall }}</el-dropdown-item>
                <el-dropdown-item command="delete">{{ NODE_T.delete }}</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </template>
    </el-table-column>
  </AppTable>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { Search, ArrowDown, Loading } from '@element-plus/icons-vue'
import { NODE_T, STATUS_OPTIONS, TYPE_OPTIONS } from './constants'

const props = defineProps({
  list: Array,
  total: Number,
  loading: Boolean,
  selectedRows: Array,
  regions: Array
})

const emit = defineEmits(['search', 'create', 'batch', 'refresh', 'selection-change', 'edit', 'go-groups', 'monitor-logs', 'go-monitor', 'status-change', 'row-action'])

const query = reactive({ page: 1, pageSize: 20, keyword: '', region_id: '', status: '', node_type: '' })
const tableRef = ref(null)

const handleSearch = () => emit('search', query)
const resetFilters = () => {
  Object.assign(query, { region_id: '', status: '', node_type: '', keyword: '' })
  handleSearch()
}
const resolveInstallStatus = (status) => {
  const normalized = (status || '').toLowerCase()
  const map = {
    idle: NODE_T.installStatusIdle,
    running: NODE_T.installStatusRunning,
    success: NODE_T.installStatusSuccess,
    failed: NODE_T.installStatusFailed
  }
  return map[normalized] || NODE_T.installStatusIdle
}
</script>

<style scoped>
.node-name-link { color: #409eff; cursor: pointer; font-weight: 500; }
.node-name-link:hover { text-decoration: underline; }
.node-group-link { margin-left: 8px; font-size: 12px; }
.monitor-protocol { margin-right: 8px; color: #909399; font-size: 11px; }
.status-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 5px; }
.status-dot.online { background-color: #67c23a; }
.status-dot.offline { background-color: #f56c6c; }
.status-dot.disabled { background-color: #909399; }
.clickable-text { color: #409eff; cursor: pointer; }
.link-more { color: #409eff; cursor: pointer; font-size: 12px; margin-left: 10px; }
.sub-ips { margin-top: 4px; }
.ip-list { max-height: 200px; overflow-y: auto; line-height: 2; }
.install-status { font-size: 12px; color: #909399; display: inline-flex; align-items: center; gap: 4px; }
.install-status.running { color: #e6a23c; }
.install-status.success { color: #67c23a; }
.install-status.failed { color: #f56c6c; cursor: pointer; }
.install-loading { animation: install-spin 1s linear infinite; }
.install-progress { font-variant-numeric: tabular-nums; }
.install-error { white-space: pre-wrap; word-break: break-all; }
@keyframes install-spin { to { transform: rotate(360deg); } }
</style>
