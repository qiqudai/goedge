<template>
  <div class="app-container">
    <el-card shadow="never" class="recharge-card">
      <div class="balance-row">
        <div class="label">余额</div>
        <div class="value">{{ balanceText }}</div>
      </div>
      <el-form :model="form" label-width="100px">
        <el-form-item label="充值金额">
          <el-input v-model.number="form.amount" placeholder="请输入充值金额" type="number" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="submitRecharge">提交充值</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const form = reactive({
  amount: 0,
  remark: ''
})

const balance = ref(0)
const balanceText = computed(() => `${(balance.value / 100).toFixed(2)}元`)

const loadProfile = () => {
  request.get('/profile').then(res => {
    balance.value = res.data?.balance || 0
  })
}

const submitRecharge = () => {
  if (!form.amount || form.amount <= 0) {
    ElMessage.warning('\u8bf7\u8f93\u5165\u5145\u503c\u91d1\u989d')
    return
  }
  request.post('/recharge', form).then(() => {
    ElMessage.success('\u5145\u503c\u63d0\u4ea4\u6210\u529f')
    form.amount = 0
    form.remark = ''
  })
}

onMounted(() => loadProfile())
</script>

<style scoped>
.recharge-card {
  max-width: 520px;
}
.balance-row {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
  font-size: 16px;
}
.label {
  color: #606266;
}
.value {
  font-weight: 600;
}
</style>
