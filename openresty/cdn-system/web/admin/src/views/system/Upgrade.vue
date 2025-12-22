<template>
  <div class="app-container">
     <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>边缘节点版本管理</span>
          <el-button style="float: right; padding: 3px 0" text>检查更新</el-button>
        </div>
      </template>
      
      <div style="margin-bottom: 20px;">
        <el-upload
            class="upload-demo"
            action="/api/v1/admin/packages"
            :limit="1"
            :on-success="handleUploadSuccess"
        >
            <el-button type="primary">上传新版本 (tar.gz)</el-button>
            <template #tip>
            <div class="el-upload__tip">
                上传 edge-node 二进制文件或压缩包。
            </div>
            </template>
        </el-upload>
      </div>

      <el-table :data="list" style="width: 100%">
        <el-table-column prop="version" label="版本号" width="180" />
        <el-table-column prop="status" label="状态" width="180">
             <template #default="{row}">
                 <el-tag :type="row.status === 'stable' ? 'success' : 'warning'">{{ row.status === 'stable' ? '稳定版' : row.status }}</el-tag>
             </template>
        </el-table-column>
        <el-table-column prop="gray_percent" label="灰度比例 %">
            <template #default="{row}">
                 <div v-if="row.status === 'gray'">
                     <el-slider v-model="row.gray_percent" :step="10" show-input @change="handleGrayChange(row)" />
                 </div>
                 <span v-else>-</span>
            </template>
        </el-table-column>
        <el-table-column prop="upload_time" label="上传时间" />
        <el-table-column label="操作">
            <template #default="{row}">
                <el-button size="small" type="success" v-if="row.status !== 'stable'" @click="promoteToStable(row)">设为稳定版</el-button>
            </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const list = ref([])

const getList = () => {
    request.get('/packages').then(res => {
         list.value = res.data.list
    })
}

const handleUploadSuccess = () => {
    ElMessage.success('上传成功')
    getList()
}

const handleGrayChange = (row) => {
    request.post('/packages/grayscale', { version: row.version, percent: row.gray_percent })
        .then(() => ElMessage.success('灰度比例已更新'))
}

const promoteToStable = (row) => {
     ElMessage.success('已将版本 ' + row.version + ' 设为稳定版')
     // Call api to set stable
}

onMounted(() => getList())
</script>
