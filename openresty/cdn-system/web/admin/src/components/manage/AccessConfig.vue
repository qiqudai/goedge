<template>
  <div class="access-config">
    <el-form label-width="150px" class="config-form">
      <div class="section-title">ACL设置</div>
      <el-form-item label="ACL选择" style="width: 520px">
        <el-select v-model="accessSettings.acl" placeholder="请选择" style="width: 100%" clearable>
          <el-option
            v-for="item in aclList"
            :key="item.id"
            :label="item.name"
            :value="item.id"
          />
        </el-select>
        <div class="form-helper">需要到左侧菜单规则管理里创建好ACL，再在这里选择应用</div>
      </el-form-item>

      <div class="divider"></div>
      
      <div class="section-title">防盗链设置</div>
      <el-form-item label="开关">
        <el-switch v-model="accessSettings.hotlink.enable" />
      </el-form-item>
      <template v-if="accessSettings.hotlink.enable">
        <el-form-item label="防盗链范围">
          <div style="display: flex; align-items: center; gap: 10px; flex-wrap: wrap;">
            <el-radio-group v-model="accessSettings.hotlink.scope">
              <el-radio value="all">整站</el-radio>
              <el-radio value="suffix">后缀</el-radio>
              <el-radio value="dir">目录</el-radio>
              <el-radio value="path">单个路径</el-radio>
            </el-radio-group>
            <el-input
              v-if="accessSettings.hotlink.scope !== 'all'"
              v-model="accessSettings.hotlink.value"
              style="width: 300px;"
              :placeholder="getHotlinkPlaceholder()"
            />
          </div>
        </el-form-item>
        <el-form-item label="允许空来源">
          <el-radio-group v-model="accessSettings.hotlink.allowEmpty">
            <el-radio :value="true">允许</el-radio>
            <el-radio :value="false">不允许</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="额外允许域名" style="width: 520px">
          <el-input v-model="accessSettings.hotlink.domains" placeholder="请输入除当前网站域名之外的域名 多个域名空格分隔" />
        </el-form-item>
      </template>

      <div class="divider"></div>

      <div class="section-title">跨域访问设置</div>
      <el-form-item label="开关">
        <el-switch v-model="accessSettings.cors.enable" />
      </el-form-item>
      <template v-if="accessSettings.cors.enable">
        <div class="cors-more-toggle" @click="corsExpanded = !corsExpanded">
          <span>{{ corsExpanded ? '▼ 收起更多设置' : '▶ 查看更多设置' }}</span>
        </div>
        
        <div v-show="corsExpanded">
          <el-form-item label="allow_origin" style="width: 520px">
            <el-input v-model="accessSettings.cors.allowOrigin" />
          </el-form-item>
          <el-form-item label="allow_methods" style="width: 520px">
            <el-input v-model="accessSettings.cors.allowMethods" />
          </el-form-item>
          <el-form-item label="allow_headers" style="width: 520px">
            <el-input v-model="accessSettings.cors.allowHeaders" />
          </el-form-item>
          <el-form-item label="expose_headers" style="width: 520px">
            <el-input v-model="accessSettings.cors.exposeHeaders" />
          </el-form-item>
          <el-form-item label="allow_credentials" style="width: 520px">
            <el-radio-group v-model="accessSettings.cors.allowCredentials">
              <el-radio :value="true">允许</el-radio>
              <el-radio :value="false">不允许</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="max_age" style="width: 520px">
            <el-input v-model="accessSettings.cors.maxAge" />
          </el-form-item>
        </div>
      </template>
    </el-form>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { getHotlinkPlaceholder } from '@/utils/siteHelpers'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  },
  aclList: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

const accessSettings = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const corsExpanded = ref(false)
const aclList = computed(() => props.aclList)

// 方法在模板中直接使用 getHotlinkPlaceholder
</script>

<style scoped>
.access-config {
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

.cors-more-toggle {
  cursor: pointer;
  color: #606266;
  font-size: 14px;
  margin-bottom: 20px;
  margin-left: 150px;
  display: flex;
  align-items: center;
  background: #f5f7fa;
  padding: 10px 15px;
  border-radius: 4px;
  transition: all 0.3s;
}

.cors-more-toggle:hover {
  background: #edf2f7;
  color: #409eff;
}
</style>