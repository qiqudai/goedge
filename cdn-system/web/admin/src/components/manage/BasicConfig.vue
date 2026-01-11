<template>
  <div class="basic-config" @focusin="cacheInputValue">
    <div class="section-title">基本设置</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="状态">
        <el-switch 
          v-model="localSettings.status" 
          active-text="正常" 
          inactive-text="已停用"
          @change="handleSave"
        />
      </el-form-item>
      <el-form-item label="CNAME">
        {{ basicCname || '-' }}
      </el-form-item>
      <el-form-item label="套餐到期">
        {{ basicExpireTime || '-' }}
      </el-form-item>
      <el-form-item label="创建时间">
        {{ basicCreatedAt || '-' }}
      </el-form-item>
      <el-form-item label="更新时间">
        {{ basicUpdatedAt || '-' }}
      </el-form-item>
    </el-form>

    <div class="divider"></div>

    <div class="section-title">基本设置</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="套餐" style="width: 520px">
        <el-select 
          v-model="localSettings.userPackageId" 
          placeholder="请选择套餐" 
          clearable
          @change="handlePackageChange"
        >
          <el-option 
            v-for="pkg in userPackages" 
            :key="pkg.id" 
            :label="pkg.name || pkg.user_plan_name || ('套餐 ' + pkg.id)" 
            :value="pkg.id" 
          />
        </el-select>
        <div class="form-helper">变更套餐不会导致CNAME地址变动，只会应用新的套餐权益</div>
      </el-form-item>
      
      <el-form-item label="所属分组" style="width: 520px">
        <SiteGroupSelect 
          v-model="localSettings.groupIds"
          :user-id="siteUserId"
          multiple
          :key="`manage-${siteUserId || 'self'}`"
          @change="handleGroupChange"
        />
        <div class="form-helper">网站的分组标识，方便为了分类和管理</div>
      </el-form-item>
      
      <el-form-item label="域名" style="width: 520px">
        <el-input
          v-model="localSettings.domain"
          @input="updateSettings"
          @blur="handleBlurSave"
        />
        <div class="form-helper">
          多个域名以空格分隔，中文域名及其它IDN域名需要转成Punycode，
          <a href="https://tool.cccyun.cc/punycode/" target="_blank" style="color: #409eff">
            转换工具
          </a>
        </div>
      </el-form-item>
    </el-form>

    <div class="divider"></div>

    <div class="section-title">HTTP设置</div>
    <el-form label-width="150px" class="config-form">
      <el-form-item label="开关">
        <el-switch 
          v-model="localSettings.httpEnable" 
          @change="handleSave"
        />
        <div class="form-helper">如果关闭，网站将完全拒绝HTTP访问</div>
      </el-form-item>
      
      <el-form-item label="监听端口" style="width: 520px">
        <el-input
          v-model="localSettings.httpPorts"
          @input="updateSettings"
          @blur="handleBlurSave"
        />
        <div class="form-helper">
          多个端口空格分隔。如需兼容http://www.example.com和http://www.example.com:888访问，则填80 888
        </div>
      </el-form-item>
    </el-form>

    <div class="divider"></div>

    <!-- 源站列表 -->
    <div class="section-title">源站列表</div>
    <el-table 
      :data="localSettings.originList || []" 
      border 
      size="small" 
      style="margin-bottom: 12px;"
    >
      <el-table-column prop="address" label="源地址">
        <template #default="{ row }">
          <el-input
            v-model="row.address"
            placeholder="IP 或域名"
            size="small"
            @blur="handleBlurSave"
          />
        </template>
      </el-table-column>
      <el-table-column prop="weight" label="权重" width="120">
        <template #default="{ row }">
          <el-input v-model="row.weight" size="small" @blur="handleBlurSave" />
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-switch 
            v-model="row.enable" 
            active-text="启用" 
            inactive-text="停用" 
            size="small"
            @change="handleSave"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="80">
        <template #default="{ $index }">
          <el-button 
            link 
            type="danger" 
            size="small" 
            @click="removeOrigin($index)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-button size="small" type="primary" @click="addOrigin">
      新增源站
    </el-button>

    <el-divider />

    <!-- 条件源站 -->
    <div class="section-title">条件源站</div>
    <el-table 
      :data="localSettings.originConditions || []" 
      border 
      size="small" 
      style="margin-bottom: 12px;"
    >
      <el-table-column label="匹配项" width="180">
        <template #default="{ row }">
          <el-select
            v-model="row.item"
            size="small"
            placeholder="请选择"
            @change="handleOriginConditionChange(row)"
          >
            <el-option
              v-for="opt in originConditionItems"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </template>
      </el-table-column>
      
      <el-table-column label="条件" min-width="260">
        <template #default="{ row }">
          <div class="condition-origin-row">
            <el-input
              v-if="isOriginHeaderItem(row.item)"
              v-model="row.header"
              size="small"
              placeholder="请求头名称，如 user-agent"
              @blur="handleBlurSave"
            />
            <el-input
              v-else-if="isOriginStatItem(row.item)"
              v-model="row.seconds"
              size="small"
              placeholder="统计秒数"
              @blur="handleBlurSave"
            />
            <el-input
              v-else
              v-model="row.value"
              size="small"
              :placeholder="getOriginConditionPlaceholder(row)"
              @blur="handleBlurSave"
            />
            <el-select
              v-if="!isOriginStatItem(row.item)"
              v-model="row.operator"
              size="small"
              placeholder="匹配方式"
              style="width: 140px;"
              @change="handleSave"
            >
              <el-option
                v-for="opt in originConditionOperators"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
          </div>
        </template>
      </el-table-column>
      
      <el-table-column label="源站" min-width="220">
        <template #default="{ row }">
          <el-input
            v-model="row.origin"
            placeholder="源站地址，多个用 | 分隔"
            size="small"
            @blur="handleBlurSave"
          />
        </template>
      </el-table-column>
      
      <el-table-column label="操作" width="100">
        <template #default="{ $index }">
          <el-button 
            link 
            type="danger" 
            size="small" 
            @click="removeConditionOrigin($index)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-button size="small" type="primary" @click="addConditionOrigin">
      新增条件源站
    </el-button>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import SiteGroupSelect from '@/components/SiteGroupSelect.vue'
import { originConditionItems, originConditionOperators } from '@/constants/origin'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { cacheInputValue, shouldSkipBlurSave } from '@/utils/saveGuard'

const { saveSettings } = useSiteSettings()

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  },
  site: { 
    type: Object, 
    default: null 
  },
  userPackages: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

// 提取只读属性用于显示
const basicCname = computed(() => props.modelValue?.cname || '-')
const basicExpireTime = computed(() => props.modelValue?.expireTime || '-')
const basicCreatedAt = computed(() => props.modelValue?.createdAt || '-')
const basicUpdatedAt = computed(() => props.modelValue?.updatedAt || '-')
const siteUserId = computed(() => props.site?.user_id || props.site?.uid || 0)

// 本地设置状态
const localSettings = ref({
  status: props.modelValue?.status !== false,
  userPackageId: props.modelValue?.userPackageId || null,
  groupIds: Array.isArray(props.modelValue?.groupIds)
    ? props.modelValue.groupIds
    : (props.modelValue?.groupId ? [props.modelValue.groupId] : []),
  domain: props.modelValue?.domain || '',
  httpEnable: props.modelValue?.httpEnable !== false,
  httpPorts: props.modelValue?.httpPorts || '80',
  originList: JSON.parse(JSON.stringify(props.modelValue?.originList || [])),
  originConditions: JSON.parse(JSON.stringify(props.modelValue?.originConditions || []))
})

let isInternalUpdate = false

// 深度监听本地配置变化并同步到父组件
watch(localSettings, (newVal) => {
  isInternalUpdate = true
  updateSettings()
}, { deep: true })

// 监听props变化，更新本地状态
watch(() => props.modelValue, (newValue) => {
  if (newValue && !isInternalUpdate) {
    localSettings.value = {
      status: newValue.status !== false,
      userPackageId: newValue.userPackageId || null,
      groupIds: Array.isArray(newValue.groupIds)
        ? newValue.groupIds
        : (newValue.groupId ? [newValue.groupId] : []),
      domain: newValue.domain || '',
      httpEnable: newValue.httpEnable !== false,
      httpPorts: newValue.httpPorts || '80',
      originList: JSON.parse(JSON.stringify(newValue.originList || [])),
      originConditions: JSON.parse(JSON.stringify(newValue.originConditions || []))
    }
  }
  isInternalUpdate = false
}, { deep: true })

const updateSettings = () => {
  isInternalUpdate = true
  emit('update:modelValue', {
    ...props.modelValue,
    ...localSettings.value
  })
}

const handleSave = () => {
  updateSettings()
  saveSettings(true)
}

const handleBlurSave = (event) => {
  if (shouldSkipBlurSave(event, { skipEmpty: true })) {
    return
  }
  handleSave()
}

const handlePackageChange = (value) => {
  handleSave()
}

const handleGroupChange = (value) => {
  handleSave()
}

// 源站相关方法
const addOrigin = () => {
  if (!localSettings.value.originList) {
    localSettings.value.originList = []
  }
  localSettings.value.originList.push({ 
    address: '', 
    weight: '10', 
    enable: true 
  })
  handleSave()
}

const removeOrigin = (index) => {
  if (localSettings.value.originList) {
    localSettings.value.originList.splice(index, 1)
    handleSave()
  }
}

const addConditionOrigin = () => {
  if (!localSettings.value.originConditions) {
    localSettings.value.originConditions = []
  }
  localSettings.value.originConditions.push({
    item: 'uri',
    operator: 'eq',
    value: '',
    origin: '',
    header: ''
  })
  handleSave()
}

const removeConditionOrigin = (index) => {
  if (localSettings.value.originConditions) {
    localSettings.value.originConditions.splice(index, 1)
    handleSave()
  }
}

const isOriginHeaderItem = (item) => {
  return item === 'header'
}

const isOriginStatItem = (item) => {
  return false
}

const getOriginConditionPlaceholder = (row) => {
  if (!row) return '输入匹配值，一行一个'
  switch (row.item) {
    case 'http_version':
      return '输入 HTTP/1.0、HTTP/1.1 等'
    case 'method':
      return '输入请求方法，如 GET'
    case 'client_ip':
      return '输入 IP 地址'
    case 'domain':
      return '输入域名，如 example.com'
    case 'uri':
    case 'uri_no_args':
      return '输入路径，如 /index.html'
    case 'node_country':
    case 'client_country':
      return '输入国家代码，如 CN'
    case 'node_isp':
    case 'client_isp':
      return '输入运营商，如 电信'
    case 'node_province':
    case 'client_province':
      return '输入省份，如 广东'
    case 'node_city':
    case 'client_city':
      return '输入城市，如 深圳'

    case 'header':
      return '输入请求头名称'
    default:
      return '输入匹配值，一行一个'
  }
}

const handleOriginConditionChange = (row) => {
  if (!row) return
  if (isOriginStatItem(row.item)) {
    row.operator = 'gt'
    row.seconds = row.seconds || '10'
  } else if (!row.operator) {
    row.operator = 'eq'
  }
  handleSave()
}
</script>

<style scoped>
.basic-config {
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

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #f56c6c;
  margin-right: 6px;
}

.status-dot.active {
  background-color: #67c23a;
}

.condition-origin-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
</style>
