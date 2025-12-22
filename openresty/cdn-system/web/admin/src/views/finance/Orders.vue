<template>
  <div class="app-container">
    <div class="filter-container">
        <el-input placeholder="订单号 / 用户 ID" style="width: 200px; margin-right: 10px;" />
        <el-button type="primary" @click="handleFilter">搜索</el-button>
        <el-button type="success" @click="dialogVisible = true">人工充值</el-button>
    </div>
    
    <el-table :data="list" border style="width: 100%; margin-top: 20px;">
      <el-table-column prop="id" label="订单号" width="120" />
      <el-table-column prop="user_id" label="用户 ID" width="100" />
      <el-table-column prop="amount" label="金额">
        <template #default="{row}">
            ¥ {{ row.amount.toFixed(2) }}
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态">
         <template #default="{row}">
             <el-tag type="success" v-if="row.status === 1">已支付</el-tag>
         </template>
      </el-table-column>
      <el-table-column prop="created_at" label="时间" />
    </el-table>

    <el-dialog title="人工充值" v-model="dialogVisible" width="400px">
        <el-form :model="form" label-width="100px">
            <el-form-item label="用户 ID">
                <el-input v-model.number="form.user_id" />
            </el-form-item>
            <el-form-item label="金额">
                <el-input v-model.number="form.amount" type="number" />
            </el-form-item>
            <el-form-item label="备注">
                <el-input v-model="form.remark" />
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="dialogVisible = false">取消</el-button>
            <el-button type="primary" @click="handleRecharge">确认</el-button>
        </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const list = ref([])
const dialogVisible = ref(false)
const form = reactive({ user_id: undefined, amount: 0, remark: '' })

const getList = () => {
    request.get('/orders').then(res => list.value = res.data.list)
}

const handleFilter = () => getList()

const handleRecharge = () => {
    request.post('/recharge', form).then(() => {
        ElMessage.success('充值成功')
        dialogVisible.value = false
        getList()
    })
}

onMounted(() => getList())
</script>
