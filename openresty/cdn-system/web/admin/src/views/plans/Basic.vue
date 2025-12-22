<template>
  <div class="app-container">
    <h2>基础套餐</h2>
    
    <div class="filter-container">
       <el-button type="primary" @click="handleCreate">添加套餐</el-button>
    </div>

    <!-- Package List -->
    <el-table :data="list" v-loading="loading" border style="width: 100%; margin-top: 20px;">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="group" label="分组" />
      <el-table-column prop="region" label="区域" />
      <el-table-column prop="price_monthly" label="月付" width="100">
          <template #default="{row}">{{ row.price_monthly }} 元</template>
      </el-table-column>
       <el-table-column prop="status" label="状态" width="100">
         <template #default="{row}">
             <el-tag :type="row.status ? 'success' : 'info'">{{ row.status ? '启用' : '禁用' }}</el-tag>
         </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="scope">
          <el-button size="small" @click="handleEdit(scope.row)">管理</el-button>
          <el-button size="small" type="success" @click="handleAssign(scope.row)">分配</el-button>
          <el-button size="small" type="danger" @click="handleDelete(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create/Edit Dialog -->
    <el-dialog :title="dialogStatus === 'create' ? '添加套餐' : '编辑套餐'" v-model="dialogVisible" width="900px" top="5vh">
       <el-form ref="dataForm" :model="temp" label-width="120px" size="small">
           
           <!-- Basic Info -->
           <el-row>
               <el-col :span="12"><el-form-item label="名称"><el-input v-model="temp.name" placeholder="请输入套餐名称" /></el-form-item></el-col>
               <el-col :span="12"><el-form-item label="描述"><el-input v-model="temp.desc" placeholder="套餐备注" /></el-form-item></el-col>
           </el-row>
           <el-row>
               <el-col :span="8"><el-form-item label="套餐分组"><el-select v-model="temp.group" placeholder="默认"><el-option label="默认" value="default"/></el-select></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="区域"><el-select v-model="temp.region" placeholder="默认"><el-option label="默认" value="default"/></el-select></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="线路分组"><el-select v-model="temp.line_group" placeholder="请选择"><el-option label="test" value="test"/></el-select></el-form-item></el-col>
           </el-row>
            <el-row>
               <el-col :span="8"><el-form-item label="备用分组"><el-select v-model="temp.backup_group" placeholder="请选择"></el-select></el-form-item></el-col>
           </el-row>

           <el-divider content-position="left">资源限制</el-divider>
            <el-row>
               <el-col :span="8">
                   <el-form-item label="月流量">
                       <el-input v-model="temp.traffic_limit" placeholder="不限">
                            <template #append><el-switch v-model="temp.traffic_unlimited" active-text="不限" inactive-text="" /></template>
                       </el-input>
                   </el-form-item>
               </el-col>
                <el-col :span="8">
                   <el-form-item label="带宽">
                        <el-input v-model="temp.bandwidth_limit" placeholder="不限"><template #append>Mbps</template></el-input>
                   </el-form-item>
               </el-col>
                <el-col :span="8">
                   <el-form-item label="连接数">
                       <el-input v-model="temp.connection_limit" placeholder="不限"></el-input>
                   </el-form-item>
               </el-col>
           </el-row>
           <!-- More rows for ports, domains, etc matching screenshot -->
           <el-row>
               <el-col :span="8"><el-form-item label="四层端口数"><el-input v-model="temp.l4_port_limit" placeholder="不限" /></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="域名数"><el-input v-model="temp.domain_limit" placeholder="不限" /></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="主域名数"><el-input v-model="temp.main_domain_limit" placeholder="不限" /></el-form-item></el-col>
           </el-row>

            <el-row>
               <el-col :span="8"><el-form-item label="网站非标端口数"><el-input v-model="temp.non_standard_port_limit" placeholder="不限" /></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="自定义CC规则"><el-switch v-model="temp.custom_cc_rules" active-text="允许" inactive-text="禁止" /></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="Websocket"><el-switch v-model="temp.websocket" active-text="允许" inactive-text="禁止" /></el-form-item></el-col>
           </el-row>

            <el-row>
               <el-col :span="8"><el-form-item label="HTTP3"><el-switch v-model="temp.http3" active-text="支持" inactive-text="禁止" /></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="L2节点回源"><el-switch v-model="temp.l2_origin" active-text="禁止" inactive-text="允许" /></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="CC防护"><el-input v-model="temp.cc_protection" placeholder="如填写支持" /></el-form-item></el-col>
           </el-row>
            <el-row>
               <el-col :span="8"><el-form-item label="DDOS防护"><el-input v-model="temp.ddos_protection" placeholder="如填写100G" /></el-form-item></el-col>
           </el-row>

           <el-divider content-position="left">定价</el-divider>
            <el-row>
               <el-col :span="8"><el-form-item label="月付"><el-input v-model="temp.price_monthly"><template #append>元</template></el-input></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="季度付"><el-input v-model="temp.price_quarterly"><template #append>元</template></el-input></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="年付"><el-input v-model="temp.price_yearly"><template #append>元</template></el-input></el-form-item></el-col>
           </el-row>

           <el-divider content-position="left">CNAME设置</el-divider>
            <el-row>
               <el-col :span="12"><el-form-item label="主机名"><el-input v-model="temp.cname_hostname" placeholder="留空则使用一级域名" /></el-form-item></el-col>
               <el-col :span="12">
                   <el-form-item label="CNAME域名">
                       <el-select v-model="temp.cname_domain" placeholder="cdnfly.com">
                            <el-option label="cdnfly.com" value="cdnfly.com"/>
                       </el-select>
                   </el-form-item>
               </el-col>
           </el-row>

           <el-divider content-position="left">购买限制</el-divider>
            <el-row>
               <el-col :span="12"><el-form-item label="单用户购买数量"><el-input v-model="temp.single_user_limit" placeholder="不限"><template #append><el-switch v-model="temp.single_user_unlimited" /></template></el-input></el-form-item></el-col>
               <el-col :span="12"><el-form-item label="有效至"><el-date-picker v-model="temp.validity" type="date" placeholder="留空则无限制" /></el-form-item></el-col>
           </el-row>
            <el-row>
               <el-col :span="12"><el-form-item label="实名认证"><el-switch v-model="temp.real_name_auth" /></el-form-item></el-col>
               <el-col :span="12"><el-form-item label="套餐时间少于N天才能续费"><el-input v-model="temp.renewal_delay" placeholder="不限"><template #append>天</template></el-input></el-form-item></el-col>
           </el-row>


           <el-divider content-position="left">其它</el-divider>
            <el-row>
               <el-col :span="8"><el-form-item label="分配给用户"><el-input v-model="temp.assigned_user" placeholder="输入用户ID, 多个逗号分隔" /></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="排序"><el-input v-model="temp.sort_order" placeholder="默认100, 数字小的靠前" /></el-form-item></el-col>
               <el-col :span="8"><el-form-item label="状态"><el-switch v-model="temp.status" active-text="启用" inactive-text="禁用" /></el-form-item></el-col>
           </el-row>
           <el-row>
               <el-col :span="12"><el-form-item label="源IP限制"><el-input type="textarea" v-model="temp.source_ip_limit" placeholder="一行一个IP或IP段" /></el-form-item></el-col>
           </el-row>


       </el-form>
       <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveData">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogStatus = ref('create')
const temp = ref({
    status: true,
    traffic_unlimited: true, // Helper for UI
    single_user_unlimited: true // Helper for UI
})

const fetchList = () => {
    loading.value = true
    request.get('/plans').then(res => {
        list.value = res.data.list || []
    }).finally(() => {
        loading.value = false
    })
}

const handleCreate = () => {
    dialogStatus.value = 'create'
    temp.value = { status: true, traffic_unlimited: true, single_user_unlimited: true }
    dialogVisible.value = true
}

const handleEdit = (row) => {
    dialogStatus.value = 'update'
    temp.value = { ...row }
    dialogVisible.value = true
}

const handleDelete = (row) => {
    ElMessageBox.confirm('确认删除该套餐吗?', '提示', {
        type: 'warning'
    }).then(() => {
        request.delete(`/plans/${row.id}`).then(() => {
            ElMessage.success('删除成功')
            fetchList()
        })
    })
}

const saveData = () => {
    const method = dialogStatus.value === 'create' ? 'post' : 'put'
    const url = dialogStatus.value === 'create' ? '/plans' : `/plans/${temp.value.id}`
    
    // Process UI helpers back to data models if needed
    // e.g. mapping traffic_unlimited toggle to traffic_limit value

    request({
        url: url,
        method: method,
        data: temp.value
    }).then(() => {
        ElMessage.success('操作成功')
        dialogVisible.value = false
        fetchList()
    })
}

const handleAssign = (row) => {
    ElMessageBox.prompt('请输入用户ID', '分配套餐', {
        confirmButtonText: '分配',
        cancelButtonText: '取消',
    }).then(({ value }) => {
        request.post('/user_plans/assign', { plan_id: row.id, user_id: parseInt(value) }).then(() => {
             ElMessage.success('分配成功')
        })
    })
}

onMounted(() => {
    fetchList()
})
</script>
