<template>
  <div class="filter-container">
    <el-button type="primary" size="small" @click="handleCreate">添加过滤器</el-button>
    <el-dropdown v-if="selection.length > 0" @command="handleBatchCommand" style="margin-left: 10px;">
      <el-button size="small">
        更多操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="delete">批量删除</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    
    <div style="flex-grow: 1;"></div>
    
    <el-input v-model="query.name" placeholder="过滤器名称、编辑搜索" style="width: 250px;" size="small">
      <template #append>
        <el-button :icon="Search" @click="fetchData" />
      </template>
    </el-input>
  </div>

  <AppTable :data="list" :loading="loading" border fit highlight-current-row persist-key="cc-filters" @selection-change="handleSelectionChange">
    <el-table-column type="selection" width="55" />
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column v-if="isAdmin" label="用户" width="120">
      <template #default="{row}">
        <span v-if="row.is_system">系统</span>
        <span v-else>{{ row.user?.username || row.user_id }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="name" label="名称" min-width="150" />
    <el-table-column label="系统规则" width="100" align="center">
      <template #default="{row}">
        <el-icon v-if="row.is_system" color="#67C23A"><Check /></el-icon>
        <el-icon v-else><Close /></el-icon>
      </template>
    </el-table-column>
    <el-table-column prop="type" label="类型" width="150">
       <template #default="{row}">{{ getActionLabel(row.action || row.type) }}</template>
    </el-table-column>
    <el-table-column label="状态" width="80" align="center">
      <template #default="{row}">
        <span :class="row.is_on ? 'text-success' : 'text-danger'">{{ row.is_on ? '正常' : '禁用' }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="create_time" label="创建时间" width="160">
       <template #default="{row}">{{ formatTime(row.create_time) }}</template>
    </el-table-column>
    <el-table-column label="操作" width="120" align="center">
      <template #default="{row}">
        <el-button type="primary" link size="small" @click="handleEdit(row)">管理</el-button>
        <el-dropdown trigger="click" @command="(cmd) => handleCommand(cmd, row)">
          <el-button type="primary" link size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="delete" style="color: #F56C6C;">删除</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </template>
    </el-table-column>
  </AppTable>

  <el-dialog :title="dialogMode === 'create' ? '添加过滤器' : '编辑过滤器'" v-model="dialogVisible" width="600px" :close-on-click-modal="false">
    <el-form :model="form" label-width="120px">
      <el-form-item label="类型" v-if="isAdmin">
        <el-radio-group v-model="form.type">
          <el-radio value="system">系统规则</el-radio>
          <el-radio value="user">用户规则</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="用户" v-if="isAdmin && form.type === 'user'">
        <el-select
          v-model="form.user_id"
          filterable
          remote
          placeholder="请输入ID或账号搜索"
          :remote-method="searchUsers"
          :loading="userLoading"
          style="width: 100%"
        >
          <el-option v-for="item in userOptions" :key="item.id" :label="item.username" :value="item.id">
             <span style="float: left">{{ item.username }}</span>
             <span style="float: right; color: #8492a6; font-size: 13px">ID:{{ item.id }}</span>
          </el-option>
        </el-select>
      </el-form-item>

      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="请输入过滤器名称" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="form.remark" type="textarea" placeholder="请输入备注" />
      </el-form-item>
      
      <el-form-item label="类型" required>
        <el-select v-model="form.action" style="width: 100%">
          <el-option label="请求频率" value="req_rate" />
          <el-option label="无感验证" value="silent_captcha" />
          <el-option label="5秒盾" value="five_seconds" />
          <el-option label="点击验证" value="click_captcha" />
          <el-option label="点击验证(简单)" value="click_captcha_simple" />
          <el-option label="滑动验证" value="slide_captcha" />
          <el-option label="滑动验证(简单)" value="slide_captcha_simple" />
          <el-option label="验证码" value="captcha" />
          <el-option label="旋转图片" value="rotate_captcha" />
          <el-option label="302跳转" value="302" />
          <el-option label="URL鉴权" value="url_auth" />
        </el-select>
      </el-form-item>

      <!-- Request Rate Fields -->
      <template v-if="form.action === 'req_rate'">
        <el-form-item label="统计周期">
          <el-input v-model.number="form.within_second" style="width: 150px">
            <template #append>秒</template>
          </el-input>
        </el-form-item>
        <el-form-item label="允许总请求">
          <el-input v-model.number="form.max_req" style="width: 150px">
            <template #append>次</template>
          </el-input>
        </el-form-item>
        <el-form-item label="允许同URL请求">
          <el-input v-model.number="form.max_req_per_uri" style="width: 150px">
             <template #append>次</template>
          </el-input>
        </el-form-item>
      </template>

      <!-- Challenge Fields (Shared) -->
      <template v-if="['silent_captcha', 'five_seconds', 'click_captcha', 'click_captcha_simple', 'slide_captcha', 'slide_captcha_simple', 'captcha', 'rotate_captcha', '302'].includes(form.action)">
         <el-form-item label="统计周期">
          <el-input v-model.number="form.within_second" style="width: 150px">
            <template #append>秒</template>
          </el-input>
        </el-form-item>
        <el-form-item label="允许验证失败">
          <el-input v-model.number="form.max_req" style="width: 150px">
            <template #append>次</template>
          </el-input>
        </el-form-item>
      </template>

      <!-- URL Auth Fields -->
      <template v-if="form.action === 'url_auth'">
         <el-form-item label="统计周期">
          <el-input v-model.number="form.within_second" style="width: 150px">
            <template #append>秒</template>
          </el-input>
        </el-form-item>
        <el-form-item label="允许验证失败">
          <el-input v-model.number="form.max_req" style="width: 150px">
             <template #append>次</template>
          </el-input>
        </el-form-item>
        <el-form-item label="鉴权方式">
           <el-radio-group v-model="form.auth.method">
             <el-radio value="A">鉴权方式A</el-radio>
             <el-radio value="B">鉴权方式B</el-radio>
           </el-radio-group>
        </el-form-item>
        <el-form-item label="IP鉴权">
           <el-switch v-model="form.auth.ip_auth" />
        </el-form-item>
        <el-form-item label="密钥(16-32位)">
           <el-input v-model="form.auth.key" placeholder="请输入密钥" show-password />
        </el-form-item>
        <el-form-item label="签名参数名">
           <el-input v-model="form.auth.sign_param" placeholder="默认 sign" />
        </el-form-item>
        <el-form-item label="时间戳参数名" v-if="form.auth.method === 'A'">
           <el-input v-model="form.auth.time_param" placeholder="默认 t" />
        </el-form-item>
        <el-form-item label="最大允许时间相差">
           <el-input v-model.number="form.auth.max_time_diff" style="width: 150px">
             <template #append>秒</template>
           </el-input>
        </el-form-item>
        <el-form-item label="允许签名使用次数">
           <el-input v-model.number="form.auth.max_sign_usage" style="width: 150px">
             <template #append>次</template>
           </el-input>
        </el-form-item>
      </template>

      <el-form-item label="启用">
        <el-switch v-model="form.enable" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submitForm">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search, Check, Close, ArrowDown } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const list = ref([])
const loading = ref(false)
const query = reactive({ name: '' })
const dialogVisible = ref(false)
const dialogMode = ref('create')
const isAdmin = ref(localStorage.getItem('role') === 'admin')
const userLoading = ref(false)
const userOptions = ref([])
const selection = ref([])

const form = reactive({ 
  id: 0, 
  name: '', 
  type: 'system', // system or user
  user_id: null,
  remark: '', 
  action: 'req_rate',
  within_second: 10,
  max_req: 0,
  max_req_per_uri: 0,
  auth: {
    method: 'A',
    ip_auth: false,
    key: '',
    sign_param: 'sign',
    time_param: 't',
    max_time_diff: 180,
    max_sign_usage: 10
  },
  enable: true,
  is_system: false 
})

const fetchData = async () => {
  loading.value = true
  try {
    const { data } = await request.get('/rules/cc/filters', { params: query })
    list.value = data.list || []
  } catch (e) {
    // ignore
  } finally {
    loading.value = false
  }
}

const searchUsers = async (keyword) => {
  if (!isAdmin.value) return
  userLoading.value = true
  try {
    const { data } = await request.get('/users', { params: { keyword: keyword, size: 20 } })
    userOptions.value = data.list || []
  } finally {
    userLoading.value = false
  }
}

const handleCreate = () => {
  dialogMode.value = 'create'
  Object.assign(form, { 
    id: 0, 
    name: '', 
    type: isAdmin.value ? 'system' : 'user', 
    user_id: null,
    remark: '', 
    action: 'req_rate',
    within_second: 10,
    max_req: 0,
    max_req_per_uri: 0,
    auth: {
       method: 'A',
       ip_auth: false,
       key: '',
       sign_param: 'sign',
       time_param: 't',
       max_time_diff: 180,
       max_sign_usage: 10
    },
    enable: true
  })
  dialogVisible.value = true
}

const handleEdit = (row) => {
  dialogMode.value = 'update'
  Object.assign(form, row)
  // Fix naming mapping from API if needed. API returns 'action' in 'action' field or 'type'?
  // Backend ListFilters: "action": filter.Type. 
  // In our form we use 'action'. row.action should be correct.
  // Ensure auth object is populated
  if (!form.auth) {
     form.auth = { method: 'A', ip_auth: false, key: '', sign_param: 'sign', time_param: 't', max_time_diff: 180, max_sign_usage: 10 }
  }
  // Setup form.type based on is_system
  form.type = row.is_system ? 'system' : 'user'
  
  if (row.user_id && isAdmin.value) {
     searchUsers('')
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  try {
    if (form.id) {
       await request.put(`/rules/cc/filters/${form.id}`, form)
    } else {
       await request.post('/rules/cc/filters', form)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch (err) {}
}

const handleCommand = (cmd, row) => {
  if (cmd === 'delete') handleDelete(row)
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除过滤器?', '提示', { type: 'warning' }).then(async () => {
    await request.delete(`/rules/cc/filters/${row.id}`)
    ElMessage.success('删除成功')
    fetchData()
  })
}

const handleSelectionChange = (val) => {
  selection.value = val
}

const handleBatchCommand = () => {
  // Implement batch delete if supported by backend
}

const getActionLabel = (act) => {
  const map = {
    req_rate: '请求频率',
    silent_captcha: '无感验证',
    five_seconds: '5秒盾',
    click_captcha: '点击验证',
    click_captcha_simple: '点击验证(简单)',
    slide_captcha: '滑动验证',
    slide_captcha_simple: '滑动验证(简单)',
    captcha: '验证码',
    rotate_captcha: '旋转图片',
    302: '302跳转',
    url_auth: 'URL鉴权'
  }
  return map[act] || act
}

const formatTime = (str) => str // backend returns formatted time string

onMounted(() => {
  fetchData()
  if (isAdmin.value) searchUsers('')
})
</script>

<style scoped>
.filter-container {
  padding-bottom: 15px;
  display: flex;
  align-items: center;
}
.text-success { color: #67C23A; }
.text-danger { color: #F56C6C; }
</style>
