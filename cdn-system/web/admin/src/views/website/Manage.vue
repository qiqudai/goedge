<template>
  <div class="site-manage">
    <!-- 网站信息头部 -->
    <SiteHeader v-if="site" :site="site" @back="goBack" />
    
    <el-card class="page-card">
      <el-tabs v-model="activeTab" class="manage-tabs" type="border-card">
        <!-- 基本配置 -->
        <el-tab-pane label="基本配置" name="basic">
          <BasicConfig 
            v-model="siteSettings.basic"
            :site="site"
            :user-packages="userPackages"
          />
        </el-tab-pane>
        
        <!-- 回源设置 -->
        <el-tab-pane label="回源设置" name="origin">
          <OriginConfig v-model="siteSettings.origin" />
        </el-tab-pane>
        
        <!-- HTTPS配置 -->
        <el-tab-pane label="HTTPS配置" name="https">
          <HttpsConfig 
            v-model="siteSettings.https"
            :cert-list="certList"
            @calc-cert-days="calcCertDays"
          />
        </el-tab-pane>
        
        <!-- 安全设置 -->
        <el-tab-pane label="安全设置" name="security">
          <SecurityConfig v-model="siteSettings.security" />
        </el-tab-pane>
        
        <!-- 缓存设置 -->
        <el-tab-pane label="缓存设置" name="cache">
          <CacheConfig 
            v-model="siteSettings.cache"
            @open-rule-dialog="openCacheRuleDialog"
          />
        </el-tab-pane>
        
        <!-- 访问控制 -->
        <el-tab-pane label="访问控制" name="access">
          <AccessConfig 
            v-model="siteSettings.access" 
            :acl-list="aclList"
          />
        </el-tab-pane>
        
        <!-- 高级设置 -->
        <el-tab-pane label="高级设置" name="advanced">
          <AdvancedConfig 
            v-model="siteSettings.advanced"
            @open-header-dialog="openHeaderDialog"
            @remove-header="removeHeader"
          />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 弹窗组件 -->
    <CacheRuleDialog
      v-model:visible="isCacheRuleDialogVisible"
      :rule="editingCacheRule"
      @save="saveCacheRule"
    />

    <HeaderRuleDialog
      v-model:visible="isHeaderDialogVisible"
      :rule="editingHeaderRule"
      :type="headerRuleType"
      @save="saveHeader"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

// 组件导入
import SiteHeader from '@/components/manage/SiteHeader.vue'
import BasicConfig from '@/components/manage/BasicConfig.vue'
import OriginConfig from '@/components/manage/OriginConfig.vue'
import HttpsConfig from '@/components/manage/HttpsConfig.vue'
import SecurityConfig from '@/components/manage/SecurityConfig.vue'
import CacheConfig from '@/components/manage/CacheConfig.vue'
import AccessConfig from '@/components/manage/AccessConfig.vue'
import AdvancedConfig from '@/components/manage/AdvancedConfig.vue'

import CacheRuleDialog from '@/components/CacheRuleDialog.vue'
import HeaderRuleDialog from '@/components/HeaderRuleDialog.vue'

// 组合式API
import { useSiteSettings } from '@/composables/useSiteSettings'
import { getHotlinkPlaceholder, normalizeCacheRule, dedupeCacheRules, dedupeHeaderRules } from '@/utils/siteHelpers'

const route = useRoute()
const router = useRouter()

// 使用状态管理
const {
  site,
  siteSettings,
  loading,
  activeTab,
  certList,
  userPackages,
  aclList,
  loadSite,
  saveSettings,
  loadCerts,
  loadUserPackages,
  loadAcls,
  calcCertDays
} = useSiteSettings()

// 弹窗状态
const isCacheRuleDialogVisible = ref(false)
const editingCacheRule = ref(null)
const isHeaderDialogVisible = ref(false)
const editingHeaderRule = ref(null)
const headerRuleType = ref('req')

// 方法
const goBack = () => {
  router.push({ path: '/website/list' })
}

const openCacheRuleDialog = (rule = null) => {
  editingCacheRule.value = rule
  isCacheRuleDialogVisible.value = true
}

const saveCacheRule = (newRule) => {
  // 使用工具函数处理
  const rule = normalizeCacheRule(newRule)
  if (!rule) return

  const nextRules = [...siteSettings.cache.rules]
  const index = nextRules.findIndex(r => r === editingCacheRule.value)
  if (index > -1) {
    nextRules.splice(index, 1, rule)
  } else {
    nextRules.push(rule)
  }
  const { rules: mergedRules, removed } = dedupeCacheRules(nextRules)
  siteSettings.cache.rules = mergedRules
  if (removed > 0) {
    ElMessage.warning('检测到重复缓存规则，已自动合并')
  }
  saveSettings(true)
}

// 请求头相关方法
const openHeaderDialog = (type, rule = null) => {
  headerRuleType.value = type
  editingHeaderRule.value = rule
  isHeaderDialogVisible.value = true
}

const saveHeader = (newRule) => {
  const list = headerRuleType.value === 'req' 
    ? siteSettings.advanced.reqHeaders
    : siteSettings.advanced.resHeaders

  const nextList = [...list]
  const index = nextList.findIndex(r => r === editingHeaderRule.value)
  if (index > -1) {
    nextList.splice(index, 1, newRule)
  } else {
    nextList.push(newRule)
  }
  const { list: mergedList, removed } = dedupeHeaderRules(nextList)
  if (headerRuleType.value === 'req') {
    siteSettings.advanced.reqHeaders = mergedList
  } else {
    siteSettings.advanced.resHeaders = mergedList
  }
  if (removed > 0) {
    ElMessage.warning('检测到重复 Header，已自动合并')
  }
  saveSettings(true)
}

const removeHeader = (type, index) => {
  const list = type === 'req' 
    ? siteSettings.advanced.reqHeaders 
    : siteSettings.advanced.resHeaders
  list.splice(index, 1)
  saveSettings(true)
}

// 初始化
onMounted(() => {
  loadSite() // 这现在会自动调用内建的 init() 并涵盖 loadCerts 等
})
</script>

<style scoped>
.site-manage {
  padding: 16px;
}

.page-card {
  background: #fff;
}

.manage-tabs {
  margin-bottom: 20px;
}

:deep(.form-helper) {
  display: block;
  width: 100%;
  margin-top: 6px;
  color: #909399;
  font-size: 13px;
  line-height: 1.5;
  clear: both;
}
</style>
