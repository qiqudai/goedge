<template>
  <div class="filter-container">
    <div class="filter-left">
      <el-button type="primary" @click="$emit('create')">添加转发</el-button>
      <el-button :disabled="!selectedRows.length" @click="$emit('batch-edit')">批量修改</el-button>
      <el-dropdown trigger="click" @command="(c) => $emit('batch-action', c)">
        <el-button>
          更多操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="enable">启用</el-dropdown-item>
            <el-dropdown-item command="disable">禁用</el-dropdown-item>
            <el-dropdown-item command="delete">删除</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <div class="filter-right">
      <el-select v-model="query.searchField" style="width: 120px;">
        <el-option label="全字段" value="all" />
        <el-option label="监听端口" value="listen" />
        <el-option label="源站" value="origin" />
        <el-option label="CNAME" value="cname" />
        <el-option label="用户" value="user" />
      </el-select>
      <el-input
        v-model="query.keyword"
        placeholder="输入关键词"
        style="width: 200px;"
        @keyup.enter="handleSearch"
      />
      <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
      <el-button link @click="$emit('advanced')">高级搜索</el-button>
    </div>
  </div>

  <AppTable
    ref="tableRef"
    v-loading="loading"
    :data="list"
    border
    fit
    storage-key="forward-list-table"
    persist-key="forward-list-table"
    :show-pagination="false"
    @selection-change="(rows) => $emit('selection-change', rows)"
  >
    <el-table-column type="selection" width="55" align="center" />
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="user_name" label="用户" width="120" />
    <el-table-column prop="listen_ports" label="监听端口" width="120" />
    <el-table-column prop="origin_display" label="源站" min-width="200" />
    <el-table-column prop="user_package_name" label="套餐" min-width="140" />
    <el-table-column prop="group_name" label="分组" width="120" />
    <el-table-column prop="node_group_name" label="区域(线路组)" min-width="140" />
    <el-table-column prop="cname" label="CNAME" min-width="200" />
    <el-table-column label="状态" width="90" align="center">
      <template #default="{ row }">
        <el-tag :type="row.status ? 'success' : 'info'">{{ row.status ? '正常' : '停用' }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="created_at" label="时间" width="180" />
    <el-table-column label="操作" width="140" align="center" fixed="right">
      <template #default="{ row }">
        <el-button link type="primary" @click="$emit('edit', row)">管理</el-button>
        <el-dropdown trigger="click" @command="(c) => $emit('row-action', c, row)">
          <span class="link-more">更多<el-icon><ArrowDown /></el-icon></span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="enable">启用</el-dropdown-item>
              <el-dropdown-item command="disable">禁用</el-dropdown-item>
              <el-dropdown-item command="delete">删除</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </template>
    </el-table-column>
  </AppTable>

  <div class="pagination-container">
    <el-pagination
      v-model:current-page="query.page"
      v-model:page-size="query.pageSize"
      layout="total, sizes, prev, pager, next"
      :total="total"
      @size-change="handleSearch"
      @current-change="handleSearch"
    />
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { Search, ArrowDown } from '@element-plus/icons-vue'

const props = defineProps({
  list: Array,
  total: Number,
  loading: Boolean,
  selectedRows: Array
})

const emit = defineEmits(['search', 'create', 'batch-edit', 'batch-action', 'selection-change', 'edit', 'row-action', 'advanced'])

const query = reactive({ page: 1, pageSize: 10, keyword: '', searchField: 'listen' })
const tableRef = ref(null)

const handleSearch = () => emit('search', query)
</script>

<style scoped>
.filter-container { display: flex; justify-content: space-between; margin-bottom: 20px; flex-wrap: wrap; gap: 10px; }
.filter-left, .filter-right { display: flex; gap: 10px; align-items: center; }
.link-more { color: #409eff; cursor: pointer; font-size: 12px; margin-left: 10px; }
.pagination-container { margin-top: 20px; text-align: right; }
</style>
