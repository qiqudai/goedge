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
        <el-form-item label="支付方式">
          <el-select v-model="form.pay_type" style="width: 100%">
            <el-option label="USDT-TRC20" value="usdt_trc20" />
            <el-option label="其他(待扩展)" value="online" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="submitRecharge">提交充值</el-button>
        </el-form-item>
      </el-form>

      <el-alert
        v-if="payInfoText"
        style="margin-top: 16px"
        type="success"
        :closable="false"
        :title="payInfoText"
      />

      <el-descriptions
        v-if="lastOrder"
        style="margin-top: 16px"
        title="最新订单"
        :column="1"
        border
      >
        <el-descriptions-item label="订单号">{{ lastOrder.order_no }}</el-descriptions-item>
        <el-descriptions-item label="支付方式">{{ lastOrder.pay_type }}</el-descriptions-item>
        <el-descriptions-item label="收款地址">{{ lastOrder.wallet || '-' }}</el-descriptions-item>
        <el-descriptions-item label="预计支付币额">{{ lastOrder.expected_amount || '-' }}</el-descriptions-item>
        <el-descriptions-item label="汇率">{{ lastOrder.exchange_rate || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const form = reactive({
  amount: 0,
  remark: '',
  pay_type: 'usdt_trc20'
})

const balance = ref(0)
const lastOrder = ref(null)

const balanceText = computed(() => `${(balance.value / 100).toFixed(2)} 元`)
const payInfoText = computed(() => {
  if (!lastOrder.value) return ''
  if (!lastOrder.value.wallet) return '订单已创建，请等待支付渠道处理'
  return `订单已创建，请向地址 ${lastOrder.value.wallet} 支付 ${lastOrder.value.expected_amount || '-'}`
})

const loadProfile = () => {
  request.get('/profile').then(res => {
    balance.value = res.data?.balance || 0
  })
}

const submitRecharge = () => {
  if (!form.amount || form.amount <= 0) {
    ElMessage.warning('请输入充值金额')
    return
  }
  request.post('/recharge', form).then(res => {
    const payload = res?.data || {}
    const payInfo = payload.pay_info || {}
    lastOrder.value = {
      order_no: payload.order_no,
      pay_type: payload.pay_type || form.pay_type,
      wallet: payInfo.wallet,
      expected_amount: payInfo.expected_amount,
      exchange_rate: payInfo.exchange_rate
    }
    ElMessage.success('充值订单创建成功')
    form.amount = 0
    form.remark = ''
  })
}

onMounted(() => loadProfile())
</script>

<style scoped>
.recharge-card {
  max-width: 640px;
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

