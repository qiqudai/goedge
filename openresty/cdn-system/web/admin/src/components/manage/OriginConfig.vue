<template>
  <div class="origin-config">
    <!-- 回源协议与端口 -->
    <div class="section-title">回源协议与端口</div>
    <el-form label-width="150px" class="config-form">
      <el-form-item label="回源协议">
        <el-radio-group v-model="originSettings.protocol">
          <el-radio value="http">HTTP</el-radio>
          <el-radio value="https">HTTPS</el-radio>
          <el-radio value="follow">跟随协议</el-radio>
          <el-radio value="follow_port">跟随端口和协议</el-radio>
        </el-radio-group>
        <div class="form-helper">
          1. 当选择HTTP，即节点与源的连接使用HTTP协议；<br/>
          2. 当选择HTTPS时，节点使用HTTPS连接；<br/>
          3. 当选择跟随协议时，当用户使用HTTP访问你在cdn上的网站时，节点也使用HTTP连接源，用户使用HTTPS访问时，节点也使用HTTPS连接源；<br/>
          4. 当选择跟随端口和协议时，即用户访问的协议和端口，节点也使用同样的协议和端口与源连接，一般用于当监听多个端口时，也希望以同样的访问端口回源
        </div>
      </el-form-item>
      
      <el-form-item 
        label="HTTP回源端口" 
        v-if="['http', 'follow'].includes(originSettings.protocol)" 
        style="width: 520px"
      >
        <el-input v-model="originSettings.httpPort" />
        <div class="form-helper">当节点与源使用HTTP连接时所使用的端口</div>
      </el-form-item>
      
      <el-form-item 
        label="HTTPS回源端口" 
        v-if="['https', 'follow'].includes(originSettings.protocol)" 
        style="width: 520px"
      >
        <el-input v-model="originSettings.httpsPort" />
        <div class="form-helper">当节点与源使用HTTPS连接时所使用的端口</div>
      </el-form-item>
    </el-form>

    <div class="divider"></div>

    <!-- 回源超时 -->
    <div class="section-title">回源超时</div>
    <el-form label-width="150px" class="config-form">
      <el-form-item label="回源超时" style="width: 320px">
        <el-input v-model="originSettings.timeout">
          <template #append>秒</template>
        </el-input>
      </el-form-item>
      <el-form-item label="连接超时" style="width: 320px">
        <el-input style="width: 320px" v-model="originSettings.connTimeout">
          <template #append>秒</template>
        </el-input>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  }
})

const emit = defineEmits(['update:modelValue'])

const originSettings = computed({
  get: () => props.modelValue || {},
  set: (val) => emit('update:modelValue', val)
})
</script>

<style scoped>
.origin-config {
  padding: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 20px;
  padding-left: 10px;
  border-left: 3px solid #409eff;
}

.divider {
  height: 1px;
  background-color: #ebeef5;
  margin: 24px 0;
}

.form-helper {
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
  margin-top: 6px;
}
</style>
