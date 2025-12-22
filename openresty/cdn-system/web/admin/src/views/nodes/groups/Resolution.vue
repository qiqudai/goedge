<template>
  <div class="app-container">
    <div class="header-actions">
      <el-button @click="$router.push('/node-groups')">← 返回</el-button>
      <div class="title-info">
        <span>区域：默认</span>
        <span style="margin-left: 20px;">分组：{{ groupName }}</span>
      </div>
    </div>

    <div class="split-container">
      <!-- Left: Unset IPs -->
      <el-card class="box-card left-card">
        <template #header>
          <div class="clearfix">
            <span>🔴 未设置的IP</span>
          </div>
        </template>
        <div class="filter-bar">
           <el-input v-model="leftKeyword" placeholder="输入IP进行查找" prefix-icon="Search" clearable />
           <el-button type="text" @click="clearLeft">清空</el-button>
        </div>
        <el-table :data="filteredUnsetIPs" style="width: 100%" height="400">
           <el-table-column type="selection" width="40" />
           <el-table-column prop="name" label="名称" />
           <el-table-column prop="ip" label="IP" />
           <el-table-column prop="status" label="状态">
               <template #default="{row}">
                   <el-tag :type="row.status === 'online' ? 'success' : 'info'">{{ row.status === 'online' ? '在线' : '离线' }}</el-tag>
               </template>
           </el-table-column>
        </el-table>
        <div class="actions">
            <el-button type="primary" @click="addToGroup">批量添加</el-button>
        </div>
      </el-card>

      <!-- Right: Set IPs -->
      <el-card class="box-card right-card">
        <template #header>
          <div class="clearfix">
            <span>🔵 已设置IP，当前线路：默认</span>
          </div>
        </template>
         <div class="filter-bar">
           <!-- Actions for right side -->
           <el-button size="small">启用</el-button>
           <el-button size="small">停用</el-button>
           <el-button size="small">删除</el-button>
           <el-select v-model="resolutionLine" size="small" placeholder="修改线路" style="width: 100px; margin-left: 10px;">
               <el-option label="默认" value="default" />
               <el-option label="联通" value="unicom" />
           </el-select>
           <el-input v-model="rightKeyword" placeholder="输入IP进行查找" prefix-icon="Search" size="small" style="width: 150px; margin-left: 10px;" />
        </div>
        <el-table :data="filteredSetIPs" style="width: 100%" height="400">
            <el-table-column type="selection" width="40" />
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="ip" label="IP" />
            <el-table-column prop="spare_ip" label="备用IP" width="80">
                 <template #default="{row}">
                     {{ row.is_spare ? '是' : '否' }}
                 </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="80">
                 <template #default="{row}">
                     <span :style="{ color: row.enabled ? 'green' : 'red' }">● {{ row.enabled ? '启用' : '停用' }}</span>
                 </template>
            </el-table-column>
            <el-table-column prop="weight" label="权重" width="60" />
            <el-table-column prop="sort_order" label="排序" width="60" />
        </el-table>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const groupName = ref('test') // Should fetch from API based on route.params.id

const leftKeyword = ref('')
const rightKeyword = ref('')
const resolutionLine = ref('')

const unsetIPs = ref([]) // Empty for now as per screenshot "暂无数据"
const setIPs = ref([
    { id: 1, name: 'agent', ip: '156.227.1.72', is_spare: false, enabled: true, weight: 1, sort_order: 100 }
])

const filteredUnsetIPs = computed(() => {
    if (!leftKeyword.value) return unsetIPs.value
    return unsetIPs.value.filter(item => item.name.includes(leftKeyword.value) || item.ip.includes(leftKeyword.value))
})

const filteredSetIPs = computed(() => {
    if (!rightKeyword.value) return setIPs.value
    return setIPs.value.filter(item => item.name.includes(rightKeyword.value) || item.ip.includes(rightKeyword.value))
})

const clearLeft = () => {
    leftKeyword.value = ''
}

const addToGroup = () => {
    // Logic to move IP from unset to set
}

</script>

<style scoped>
.app-container {
    padding: 20px;
}
.header-actions {
    margin-bottom: 20px;
    display: flex;
    align-items: center;
}
.title-info {
    font-size: 16px;
    font-weight: bold;
}
.split-container {
    display: flex;
    gap: 20px;
}
.left-card, .right-card {
    flex: 1;
}
.filter-bar {
    display: flex;
    margin-bottom: 10px;
    gap: 10px;
}
</style>
