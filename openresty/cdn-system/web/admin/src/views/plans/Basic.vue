<template>
  <div class="app-container">
    <h2>基础套餐</h2>

    <div class="filter-container">
      <el-button type="primary" @click="handleCreate">添加套餐</el-button>
    </div>

    <AppTable
      :data="list"
      :loading="loading"
      border
      style="width: 100%; margin-top: 20px;"
      persist-key="plans-basic"
    >
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="group" label="分组" />
      <el-table-column prop="region" label="区域" />
      <el-table-column prop="price_monthly" label="月付" width="100">
        <template #default="{ row }">{{ row.price_monthly }}</template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'">{{ row.status ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="scope">
          <el-button size="" @click="handleEdit(scope.row)">管理</el-button>
          <el-button size="" type="success" @click="handleAssign(scope.row)">分配</el-button>
          <el-button size="" type="danger" @click="handleDelete(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </AppTable>

    <el-dialog
      :title="dialogStatus === 'create' ? '添加套餐' : '编辑套餐'"
      v-model="dialogVisible"
      width="900px"
      top="5vh"
    >
      <el-form ref="dataForm" :model="temp" label-width="120px" size="">
        <el-row>
          <el-col :span="12">
            <el-form-item label="名称">
              <el-input v-model="temp.name" placeholder="请输入套餐名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="描述">
              <el-input v-model="temp.desc" placeholder="套餐备注" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="8">
            <el-form-item label="套餐分组">
              <el-select v-model="temp.group" placeholder="默认">
                <el-option label="默认" value="default" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="区域">
              <el-select v-model="temp.region" placeholder="默认">
                <el-option label="默认" :value="0" />
                <el-option
                  v-for="item in regionOptions"
                  :key="item.id"
                  :label="item.name"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="线路分组">
              <el-select v-model="temp.line_group" placeholder="请选择">
                <el-option label="默认" :value="0" />
                <el-option
                  v-for="item in nodeGroupOptions"
                  :key="item.id"
                  :label="item.name"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="8">
            <el-form-item label="备用分组">
              <el-select v-model="temp.backup_group" placeholder="请选择">
                <el-option label="默认" :value="0" />
                <el-option
                  v-for="item in nodeGroupOptions"
                  :key="item.id"
                  :label="item.name"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">资源限制</el-divider>
        <el-row>
          <el-col :span="8">
            <el-form-item label="月流量">
              <el-input v-model="temp.traffic_limit" placeholder="不限" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="带宽">
              <el-input v-model="temp.bandwidth_limit" placeholder="不限">
                <template #append>Mbps</template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="连接数">
              <el-input v-model="temp.connection_limit" placeholder="不限" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="8">
            <el-form-item label="四层端口数">
              <el-input v-model="temp.stream_port" placeholder="不限" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="域名数">
              <el-input v-model="temp.domain_limit" placeholder="不限" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="8">
            <el-form-item label="网站非标端口数">
              <el-input v-model="temp.http_port" placeholder="不限" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="自定义CC规则">
              <el-switch v-model="temp.custom_cc_rules" active-text="允许" inactive-text="禁止" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Websocket">
              <el-switch v-model="temp.websocket" active-text="允许" inactive-text="禁止" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">定价</el-divider>
        <el-row>
          <el-col :span="8">
            <el-form-item label="月付">
              <el-input v-model="temp.price_monthly">
                <template #append>元</template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="季度付">
              <el-input v-model="temp.price_quarterly">
                <template #append>元</template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="年付">
              <el-input v-model="temp.price_yearly">
                <template #append>元</template>
              </el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">CNAME设置</el-divider>
        <el-row>
          <el-col :span="12">
            <el-form-item label="主机名">
              <el-input v-model="temp.cname_hostname2" placeholder="留空则使用一级域名" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="CNAME域名">
              <el-select v-model="temp.cname_domain" placeholder="cdnfly.com">
                <el-option label="cdnfly.com" value="cdnfly.com" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="CNAME模式">
              <el-select v-model="temp.cname_mode" placeholder="默认">
                <el-option label="默认" value="default" />
                <el-option label="站点生成" value="site" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">购买限制</el-divider>
        <el-row>
          <el-col :span="12">
            <el-form-item label="单用户购买数量">
              <el-input v-model="temp.buy_num_limit" placeholder="不限" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="有效期">
              <el-date-picker
                v-model="temp.expire"
                type="datetime"
                placeholder="留空则无限制"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                clearable
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="实名认证">
              <el-switch v-model="temp.id_verify" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="套餐时间少于N天才能续费">
              <el-input v-model="temp.before_exp_days_renew" placeholder="不限">
                <template #append>天</template>
              </el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">其它</el-divider>
        <el-row>
          <el-col :span="8">
            <el-form-item label="分配给用户">
              <el-input v-model="temp.owner" placeholder="输入用户ID, 多个逗号分隔" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="排序">
              <el-input v-model="temp.sort_order" placeholder="默认100, 数字小的靠前" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="状态">
              <el-switch v-model="temp.status" active-text="启用" inactive-text="禁用" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="源IP限制">
              <el-input type="textarea" v-model="temp.backend_ip_limit" placeholder="一行一个IP或IP段" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveData">确定</el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="assignVisible" title="分配套餐" width="520px">
      <el-form :model="assignForm" label-width="90px">
        <el-form-item label="套餐名称">
          <el-input v-model="assignForm.plan_name" disabled />
        </el-form-item>
        <el-form-item label="用户选择">
          <el-select
            v-model.number="assignForm.user_id"
            filterable
            remote
            clearable
            placeholder="输入ID、邮箱、用户名、手机号搜索"
            :remote-method="loadUsers"
            :loading="userLoading"
            style="width: 100%;"
          >
            <el-option v-for="u in userOptions" :key="u.id" :label="formatUserLabel(u)" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="时长">
          <el-radio-group v-model="assignForm.duration_mode">
            <el-radio value="1">1个月</el-radio>
            <el-radio value="3">3个月</el-radio>
            <el-radio value="12">12个月</el-radio>
            <el-radio value="custom">自定义</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker
            v-model="assignForm.end_at"
            type="datetime"
            placeholder="请选择"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            clearable
            style="width: 100%;"
            :disabled="assignForm.duration_mode !== 'custom'"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="" @click="assignVisible = false">取消</el-button>
        <el-button size="" type="primary" :loading="assignSaving" @click="submitAssign">确定</el-button>
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
const assignVisible = ref(false)
const assignSaving = ref(false)
const userOptions = ref([])
const userLoading = ref(false)
const assignForm = ref({
    plan_id: null,
    plan_name: '',
    user_id: null,
    duration_mode: '1',
    end_at: ''
})
const regionOptions = ref([])
const nodeGroupOptions = ref([])
const temp = ref({
    status: true
})

const fetchList = () => {
    loading.value = true
    request.get('/plans').then(res => {
        list.value = res.data.list || []
    }).finally(() => {
        loading.value = false
    })
}

const fetchRegions = () => {
    request.get('/regions').then(res => {
        regionOptions.value = res.data.list || []
    })
}

const fetchNodeGroups = () => {
    request.get('/node-groups').then(res => {
        nodeGroupOptions.value = res.data.list || []
    })
}

const handleCreate = () => {
    dialogStatus.value = 'create'
    temp.value = { status: true }
    dialogVisible.value = true
}

const handleEdit = (row) => {
    dialogStatus.value = 'update'
    request.get(`/plans/${row.id}`).then(res => {
        temp.value = { ...res.data }
        dialogVisible.value = true
    })
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
    assignForm.value = {
        plan_id: row.id,
        plan_name: row.name || '',
        user_id: null,
        duration_mode: '1',
        end_at: ''
    }
    userOptions.value = []
    assignVisible.value = true
}

const loadUsers = (query) => {
    if (!query) {
        userOptions.value = []
        return
    }
    userLoading.value = true
    request.get('/users', { params: { keyword: query, pageSize: 20 } }).then(res => {
        userOptions.value = res.data?.list || res.list || []
    }).finally(() => {
        userLoading.value = false
    })
}

const formatUserLabel = (u) => {
    if (!u) return ''
    const name = u.name || u.username || '-'
    return `${name} (id: ${u.id})`
}

const submitAssign = () => {
    if (!assignForm.value.user_id) {
        ElMessage.error('请选择用户')
        return
    }
    const payload = {
        plan_id: assignForm.value.plan_id,
        user_id: assignForm.value.user_id
    }
    if (assignForm.value.duration_mode === 'custom') {
        if (!assignForm.value.end_at) {
            ElMessage.error('请选择到期时间')
            return
        }
        payload.end_at = assignForm.value.end_at
    } else {
        payload.duration_months = parseInt(assignForm.value.duration_mode, 10)
    }
    assignSaving.value = true
    request.post('/user_plans/assign', payload).then(() => {
        ElMessage.success('分配成功')
        assignVisible.value = false
    }).finally(() => {
        assignSaving.value = false
    })
}

onMounted(() => {
    fetchList()
    fetchRegions()
    fetchNodeGroups()
})
</script>
