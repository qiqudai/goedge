<template>
  <div class="filter-container">
    <div class="filter-left">
      <el-button type="primary" @click="handleAction('create')">添加网站</el-button>
      <el-button :disabled="!selectedRows.length" @click="handleAction('batch-edit')">批量修改</el-button>
      <el-button :disabled="!selectedRows.length" @click="handleAction('apply-cert')">申请证书</el-button>
      <el-dropdown trigger="click" @command="(c) => handleAction('batch-' + c)">
        <el-button>
          更多操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="enable">启用</el-dropdown-item>
            <el-dropdown-item command="disable">禁用</el-dropdown-item>
            <el-dropdown-item command="delete">删除</el-dropdown-item>
            <el-dropdown-item command="unlock">解除黑名单</el-dropdown-item>
            <el-dropdown-item command="clear_cache">清空缓存</el-dropdown-item>
            <el-dropdown-item v-if="isAdmin" divided command="cname-domain">CNAME域名</el-dropdown-item>
            <el-dropdown-item v-if="isAdmin" command="cname-mode">CNAME模式</el-dropdown-item>
            <el-dropdown-item v-if="isAdmin" command="node-group">线路分组</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      
      <el-divider direction="vertical" style="height: 32px; margin: 0 12px;" />

      <el-select v-model="query.searchField" style="width: 120px;">
        <el-option label="全字段" value="all" />
        <el-option label="域名" value="domain" />
        <el-option label="源IP" value="origin" />
        <el-option label="CNAME" value="cname" />
      </el-select>
      <el-input
        v-model="query.keyword"
        placeholder="输入关键字"
        style="width: 200px;"
        @keyup.enter="handleSearch"
      />
      <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
      <el-button @click="$emit('export')">导出</el-button>
      <el-button link @click="$emit('advanced')">高级搜索</el-button>
    </div>
  </div>

  <AppTable
    ref="tableRef"
    v-loading="loading"
    :data="list"
    border
    fit
    storage-key="website-site-list"
    :total="total"
    v-model:current-page="query.page"
    v-model:page-size="query.pageSize"
    layout="total, sizes, prev, pager, next"
    @size-change="handleSearch"
    @current-change="handleSearch"
    @selection-change="(rows) => $emit('selection-change', rows)"
  >
    <el-table-column type="selection" width="55" align="center" />
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column label="域名" min-width="200">
      <template #default="{ row }">
        <span class="clickable-text" @click="$emit('manage', row)">
           {{ row.domain_display || (row.domains && row.domains[0]) || '-' }}
        </span>
        <el-icon class="copy-icon" @click.stop="copyText(row.domain_display || (row.domains && row.domains[0]))"><CopyDocument /></el-icon>
      </template>
    </el-table-column>
    <el-table-column prop="listen_ports" label="监听端口" width="120" />
    <el-table-column label="源站" min-width="180">
      <template #default="{ row }">
        <span>{{ row.origin_display }}</span>
        <el-icon class="copy-icon" @click.stop="copyText(row.origin_display)"><CopyDocument /></el-icon>
      </template>
    </el-table-column>
    <el-table-column prop="cname" label="CNAME" min-width="180">
       <template #default="{ row }">
        <span>{{ row.cname }}</span>
        <el-icon class="copy-icon" @click.stop="copyText(row.cname)"><CopyDocument /></el-icon>
      </template>
    </el-table-column>
    <el-table-column label="HTTPS" width="80" align="center">
      <template #default="{ row }">
        <el-tag v-if="row.https" type="success" size="small">开启</el-tag>
        <el-tag v-else type="info" size="small">关闭</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="user_package_name" label="套餐" min-width="100" />
    <el-table-column prop="region_name" label="区域" min-width="100" />
    <el-table-column prop="node_group_name" label="线路组" min-width="100" />
    <el-table-column prop="group_name" label="分组" min-width="100" />
    <el-table-column label="状态" width="80" align="center">
      <template #default="{ row }">
        <el-tag v-if="row.status" type="success" size="small">正常</el-tag>
        <el-tag v-else type="info" size="small">停用</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="添加时间" width="160">
      <template #default="{ row }">
        {{ row.created_at ? row.created_at.replace('T', ' ').substring(0, 19) : '-' }}
      </template>
    </el-table-column>
    <el-table-column label="操作" width="150" align="center">
      <template #default="{ row }">
        <div style="display: flex; justify-content: center; gap: 8px;">
          <el-button link type="primary" @click="$emit('manage', row)">管理</el-button>
          <el-dropdown trigger="click" @command="(c) => handleAction('row-' + c, row)">
             <span class="link-more">更多<el-icon><ArrowDown /></el-icon></span>
             <template #dropdown>
               <el-dropdown-menu>
                 <el-dropdown-item v-if="!row.status" command="enable">启用</el-dropdown-item>
                 <el-dropdown-item v-else command="disable">禁用</el-dropdown-item>
                 <el-dropdown-item command="delete">删除</el-dropdown-item>
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
import { Search, ArrowDown, CopyDocument } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  list: Array,
  total: Number,
  loading: Boolean,
  selectedRows: Array,
  isAdmin: Boolean
})

const emit = defineEmits(['search', 'action', 'selection-change', 'manage', 'export', 'advanced'])

const tableRef = ref(null)

const query = reactive({
  page: 1,
  pageSize: 10,
  keyword: '',
  searchField: 'all'
})

const handleSearch = () => emit('search', query)
const handleAction = (type, data) => emit('action', type, data)

const copyText = async (text) => {
  if (!text || text === '-') return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success({ message: '复制成功', duration: 1500 })
  } catch (e) {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped>
.filter-container { margin-bottom: 20px; }
.filter-left { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.clickable-text { color: #409eff; cursor: pointer; }
.clickable-text:hover { text-decoration: underline; }
.link-more { color: #409eff; cursor: pointer; font-size: 12px; margin-left: 10px; }
.pagination-container { margin-top: 20px; text-align: right; }
.copy-icon { margin-left: 5px; cursor: pointer; color: #909399; vertical-align: middle; }
.copy-icon:hover { color: #409eff; }
.copyable-text { cursor: pointer; }
.copyable-text:hover { color: #409eff; }
</style>
