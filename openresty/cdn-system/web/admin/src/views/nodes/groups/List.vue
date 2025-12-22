<template>
  <div class="app-container">
    <div class="filter-container">
      <el-button type="primary" :icon="Plus" @click="handleCreate">新增分组</el-button>
      <el-button @click="handleBatchDelete">删除</el-button>
      
      <div class="right-actions" style="float: right;">
        <el-select v-model="listQuery.region" placeholder="区域: 默认" style="width: 120px; margin-right: 10px;">
          <el-option label="默认" value="default" />
          <el-option label="中国" value="china" />
        </el-select>
        <el-input v-model="listQuery.keyword" placeholder="填名称或解析值搜索" style="width: 200px;" class="filter-item" @keyup.enter="handleFilter" />
        <el-button type="text" @click="listQuery.keyword = ''; handleFilter()">清除</el-button>
      </div>
    </div>

    <el-table :data="list" v-loading="listLoading" border fit highlight-current-row style="width: 100%; margin-top: 20px;">
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column label="ID" prop="id" width="80" align="center" />
      <el-table-column label="名称" min-width="150">
        <template #default="{row}">
          <span class="link-type" @click="handleUpdate(row)">{{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column label="区域" prop="region" width="100" align="center" />
      <el-table-column label="解析值" min-width="200">
        <template #default="{row}">
          <div>{{ row.resolution }} <span style="color: grey;">(IPv4: {{ row.ipv4_resolution }})</span></div>
        </template>
      </el-table-column>
      <el-table-column label="统计" min-width="250">
        <template #default="{row}">
          节点数(1个) 网站数(2个) 独立站(0个)
        </template>
      </el-table-column>
      <el-table-column label="L2配置" width="100" align="center">
        <template #default="{row}">
           <!-- Placeholder -->
        </template>
      </el-table-column>
      <el-table-column label="排序" prop="sort_order" width="80" align="center" />
      <el-table-column label="操作" align="center" width="200" class-name="small-padding fixed-width">
        <template #default="{row}">
          <el-button type="text" size="small" @click="handleResolution(row)">配置解析</el-button>
          <el-button type="text" size="small" @click="handleUpdate(row)">编辑</el-button>
          <el-button type="text" size="small" style="color: red;" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div style="margin-top: 20px; text-align: right;">
        <el-pagination
          v-show="total>0"
          v-model:current-page="listQuery.page"
          v-model:page-size="listQuery.limit"
          :page-sizes="[10, 20, 30, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="getList"
          @current-change="getList"
        />
    </div>

    <!-- Edit Dialog -->
    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="600px">
      <el-form ref="dataForm" :model="temp" label-position="right" label-width="100px" style="width: 100%; padding-right: 20px;">
        <el-form-item label="名称:" prop="name">
          <el-input v-model="temp.name" placeholder="分组名称" />
        </el-form-item>
        <el-form-item label="区域:" prop="region">
             <el-select v-model="temp.region" placeholder="选择区域" style="width: 100%;">
                <el-option label="默认" value="默认" />
                <el-option label="中国" value="中国" />
                <el-option label="海外" value="海外" />
             </el-select>
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
            <el-input v-model="temp.sort_order" />
        </el-form-item>
        
        <el-form-item label="L2配置:" prop="l2_config">
           <el-select v-model="temp.l2_config" placeholder="请选择">
               <el-option label="默认配置" value="default" />
           </el-select>
        </el-form-item>
        
        <el-form-item label="备用IP切换:">
            <el-radio-group v-model="temp.spare_ip_switch">
                <el-radio :label="1">有主IP下线时</el-radio>
                <el-radio :label="2">在线IP数少于备用IP数时</el-radio>
                <el-radio :label="3">间隔切换</el-radio>
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
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const list = ref([])
const total = ref(0)
const listLoading = ref(true)
const listQuery = reactive({
  page: 1,
  limit: 20,
  keyword: '',
  region: ''
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
  region: '默认',
  resolution: '',
  ipv4_resolution: '',
  remark: '',
  sort_order: 100,
  l2_config: '',
  spare_ip_switch: 1
})

const getList = () => {
    listLoading.value = true
    // Mock API
    request({
        url: '/node-groups',
        method: 'get',
        params: listQuery
    }).then(response => {
        list.value = response.data.list
        total.value = response.data.total || list.value.length
        listLoading.value = false
    }).catch(() => {
        // Fallback for development if API fails
        list.value = [
            { id: 1, name: 'test', region: '默认', resolution: 'fz2c9mru', ipv4_resolution: 'xqkt5je9', sort_order: 100, spare_ip_switch: 1 }
        ]
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
  temp.resolution = ''
  temp.ipv4_resolution = ''
  temp.remark = ''
  temp.sort_order = 100
  temp.l2_config = ''
  temp.spare_ip_switch = 1
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
        getList()
    })
}

const updateData = () => {
    request({
        url: `/node-groups/${temp.id}`,
        method: 'put',
        data: temp
    }).then(() => {
        dialogFormVisible.value = false
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

const handleBatchDelete = () => {}

const router = useRouter() // Initialize router

const handleResolution = (row) => {
   router.push({ name: 'NodeGroupResolution', params: { id: row.id } })
}

onMounted(() => {
  getList()
})
</script>

<style scoped>
.link-type {
    color: #409EFF;
    cursor: pointer;
}
</style>
