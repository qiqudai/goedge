<template>
  <div class="security-config" @focusin="cacheInputValue">
    <!-- CC Protection -->
    <div class="section-title">CC 防护</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="默认规则">
        <el-radio-group v-model="computedMode">
          <el-radio 
            v-for="rule in systemRules" 
            :key="rule.id" 
            :value="rule.id"
          >
            {{ rule.name }}
          </el-radio>
          <el-radio value="custom">自定义</el-radio>
        </el-radio-group>
        
        <el-select 
          v-if="computedMode === 'custom' && userRules.length > 0"
          v-model="securitySettings.cc.mode"
          placeholder="请选择自定义规则"
          size="small"
          style="margin-left: 10px; width: 160px;"
          @change="handleSave"
        >
          <el-option
            v-for="rule in userRules"
            :key="rule.id"
            :label="rule.name"
            :value="rule.id"
          />
        </el-select>
        <span v-if="computedMode === 'custom' && userRules.length === 0" style="margin-left: 10px; color: #909399; font-size: 13px;">
          (暂无自定义规则)
        </span>
        <div class="form-helper">不同模式对应不同的防御级别</div>
      </el-form-item>
      
      <el-form-item label="自动切换">
        <div style="display: flex; align-items: center; gap: 10px;">
          <el-switch v-model="securitySettings.cc.autoSwitch.enable" @change="handleSave" />
          <span v-if="securitySettings.cc.autoSwitch.enable" style="font-size: 13px; display: flex; align-items: center; gap: 5px;">
            当QPS超过 
            <el-select 
              v-model="qpsSelection" 
              size="small" 
              style="width: 100px" 
              @change="handleQpsChange"
            >
              <el-option label="20" :value="20" />
              <el-option label="50" :value="50" />
              <el-option label="200" :value="200" />
              <el-option label="自定义" value="custom" />
            </el-select>
            <el-input 
              v-if="qpsSelection === 'custom'"
              v-model="securitySettings.cc.autoSwitch.qps" 
              size="small" 
              style="width: 80px" 
              @blur="handleBlurSave" 
            /> 
            时，自动切换到 
            <el-select v-model="securitySettings.cc.autoSwitch.rule" size="small" style="width: 100px;" @change="handleSave">
              <el-option label="关闭" value="close" />
              <el-option label="宽松" value="lenient" />
              <el-option label="普通" value="normal" />
              <el-option label="严格" value="strict" />
              <el-option label="JS验证" value="js" />
              <el-option label="验证码" value="captcha" />
            </el-select>
          </span>
        </div>
      </el-form-item>
    </el-form>

    <div class="divider"></div>

    <!-- Custom Rules -->
    <div class="section-title">自定义规则</div>
    <div style="margin-bottom: 15px; padding-left: 20px;">
      <el-button type="primary" size="small" @click="openRuleDialog('create')">新增规则</el-button>
    </div>

    <el-table :data="securitySettings.cc.customRules" border style="width: 100%; margin-bottom: 10px;">
      <el-table-column label="匹配条件" min-width="200">
        <template #default="{ row }">
          <div v-for="(m, idx) in row.matchers" :key="idx" style="font-size: 12px;">
            {{ getMatcherText(m) }}
          </div>
          <div v-if="!row.matchers || !row.matchers.length">匹配所有请求</div>
        </template>
      </el-table-column>
      <el-table-column label="执行过滤" width="120">
        <template #default="{ row }">
          {{ getActionText(row.action) }}
        </template>
      </el-table-column>
      <el-table-column label="匹配模式" width="120">
        <template #default="{ row }">
          {{ row.breakMatch ? '停止匹配' : '继续下一条' }}
        </template>
      </el-table-column>
      <el-table-column prop="remart" label="备注" />
      <el-table-column label="操作" width="150" align="center">
        <template #default="{ $index }">
          <div class="rule-actions">
            <el-button class="rule-actions__edit" size="small" @click="editRule($index)">
              <el-icon><EditPen /></el-icon>
            </el-button>
            <div class="rule-actions__sort">
              <el-button
                size="small"
                :disabled="$index === 0"
                @click="moveRule($index, -1)"
              >
                <el-icon><ArrowUp /></el-icon>
              </el-button>
              <el-button
                size="small"
                :disabled="$index === securitySettings.cc.customRules.length - 1"
                @click="moveRule($index, 1)"
              >
                <el-icon><ArrowDown /></el-icon>
              </el-button>
            </div>
            <el-button class="rule-actions__delete" size="small" @click="deleteRule($index)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div class="form-helper" style="padding-left: 20px;">
      <div>1. 自定义规则优先匹配，之后才是上面的默认防护;</div>
      <div>2. 像API请求的放行可以使用此处的自定义规则;</div>
      <div>3. 规则是从上到下匹配，可使用上下按钮调整顺序。</div>
    </div>

    <div class="divider"></div>

    <!-- Rule Dialog -->
    <el-dialog 
      v-model="ruleDialog.visible" 
      :title="ruleDialog.mode === 'create' ? '新增规则' : '编辑规则'" 
      width="700px"
      append-to-body
      :close-on-click-modal="false"
    >
      <el-form label-width="100px">
        <!-- Matchers -->
        <el-form-item label="匹配条件">
          <div class="matcher-config">
            <div v-for="(m, idx) in ruleForm.matchers" :key="idx" class="matcher-row">
              <span class="matcher-text">{{ getMatcherText(m) }}</span>
              <el-button link type="danger" size="small" @click="removeMatcher(idx)">删除</el-button>
            </div>
            
            <div class="matcher-add">
               <el-select v-model="newMatcher.key" placeholder="选择匹配项" size="small" style="width: 140px">
                 <el-option v-for="item in matchItems" :key="item.value" :label="item.label" :value="item.value" />
               </el-select>
               
               <template v-if="newMatcher.key !== 'all'">
                 <el-select v-model="newMatcher.operator" placeholder="操作符" size="small" style="width: 100px; margin: 0 5px;">
                   <el-option v-for="op in operators" :key="op.value" :label="op.label" :value="op.value" />
                 </el-select>
                 <el-input v-model="newMatcher.value" placeholder="输入匹配值" size="small" style="width: 200px;" />
               </template>
               
               <el-button type="primary" size="small" link style="margin-left: 5px;" @click="addMatcher">添加</el-button>
            </div>
          </div>
          <div class="form-helper">多个匹配条件的关系为且，即所有条件都满足时才执行下面的过滤</div>
        </el-form-item>

        <!-- Action -->
        <el-form-item label="执行过滤">
          <div class="action-grid">
            <el-radio-group v-model="ruleForm.action">
              <el-radio v-for="act in actions" :key="act.value" :value="act.value" style="margin-right: 15px; margin-bottom: 5px;">
                {{ act.label }}
              </el-radio>
            </el-radio-group>
          </div>
          
          <div v-if="ruleForm.action === 'block'" class="form-helper">
            匹配后将 IP 加入节点黑名单，时长取自下方「黑白名单时间 → 黑名单时间」（默认 3600 秒）。
          </div>

          <!-- Action Params: Rate Limit -->
          <div v-if="ruleForm.action === 'limit_rate'" class="rule-params">
            <el-form-item label="在" label-width="60px" style="margin-bottom: 5px;">
               <el-input v-model.number="ruleForm.actionParams.seconds" size="small" ><template #append>秒内</template></el-input>
            </el-form-item>
            <el-form-item label="限制总请求" label-width="100px" style="margin-bottom: 5px;">
               <el-input v-model.number="ruleForm.actionParams.requests" size="small" ><template #append>次</template></el-input>
            </el-form-item>
            <el-form-item label="限制同URL" label-width="100px" style="margin-bottom: 0;">
               <el-input v-model.number="ruleForm.actionParams.urlRequests" size="small" ><template #append>次</template></el-input>
            </el-form-item>
          </div>

          <!-- Action Params: Block Logic for verification -->
          <div v-if="isVerificationAction(ruleForm.action)" style="margin-top: 10px;">
            <el-form-item label="是否拉黑" label-width="80px">
              <el-radio-group v-model="ruleForm.actionParams.blockOnFail">
                <el-radio :value="true">拉黑</el-radio>
                <el-radio :value="false">不拉黑</el-radio>
              </el-radio-group>
              <div class="form-helper" v-if="ruleForm.actionParams.blockOnFail">
                选择拉黑时，在一定时间内多次验证不通过，则显示拉黑页面，或者加到ipset全局拉黑；
              </div>
              <div class="form-helper" v-else>
                选择不拉黑时，可以无限次验证，直到验证通过。
              </div>
            </el-form-item>
          </div>
        </el-form-item>
        
        <el-form-item label="匹配模式">
          <el-radio-group v-model="ruleForm.breakMatch">
            <el-radio :value="false">继续下一条规则</el-radio>
            <el-radio :value="true">停止匹配</el-radio>
          </el-radio-group>
          <div class="form-helper">模式为继续下一条规则时，执行当前过滤后，仍然继续下一条规则匹配</div>
        </el-form-item>

        <el-form-item label="备注">
          <el-input v-model="ruleForm.remark" placeholder="请输入备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="small" @click="ruleDialog.visible = false">取消</el-button>
        <el-button size="small" type="primary" @click="saveRule">确定</el-button>
      </template>
    </el-dialog>

    <div class="divider"></div>

    <!-- Crawlers -->
    <div class="section-title">搜索引擎爬虫</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="爬虫策略">
        <el-radio-group v-model="securitySettings.crawlers.action" @change="handleSave">
          <el-radio value="none">不设置</el-radio>
          <el-radio value="allow">放行</el-radio>
          <el-radio value="block">拦截</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <div class="divider"></div>
    
    <!-- IP List Timeouts -->
    <div class="section-title">黑白名单时间</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="黑名单时间">
        <div style="display: flex; gap: 20px;">
          <el-radio-group v-model="securitySettings.ip.blackTimeCustom" @change="handleTimeTypeChange('black', $event)">
            <el-radio :value="false">系统默认(3600秒)</el-radio>
            <el-radio :value="true">自定义</el-radio>
          </el-radio-group>
          <el-input 
            v-if="securitySettings.ip.blackTimeCustom" 
            v-model.number="securitySettings.ip.blackTime" 
            size="small" 
            style="width: 100px" 
            @blur="handleBlurSave"
          >
            <template #append>秒</template>
          </el-input>
        </div>
      </el-form-item>
      <el-form-item label="白名单时间">
        <div style="display: flex; gap: 20px;">
          <el-radio-group v-model="securitySettings.ip.whiteTimeCustom" @change="handleTimeTypeChange('white', $event)">
            <el-radio :value="false">系统默认(21600秒)</el-radio>
            <el-radio :value="true">自定义</el-radio>
          </el-radio-group>
          <el-input 
            v-if="securitySettings.ip.whiteTimeCustom" 
            v-model.number="securitySettings.ip.whiteTime" 
            size="small" 
            style="width: 100px" 
            @blur="handleBlurSave"
          >
            <template #append>秒</template>
          </el-input>
        </div>
        <div class="form-helper">客户端防cc(如点击，滑动等)验证通过后，距离下一次再次需要验证的时长。</div>
      </el-form-item>
    </el-form>

    <div class="divider"></div>
       
    <!-- IP Lists (Content only, logic simplified as requested to remove UA controls but keep IP lists?) 
         Wait, Request F: "Remove UA Black/White List controls". 
         Existing code had IP Black/White list. I should keep IP lists but remove UA lists.
    -->
    <div class="section-title">黑白名单</div>
    <el-form label-width="120px">
      <el-form-item label="IP黑名单">
        <el-input type="textarea" v-model="securitySettings.ip.black" :rows="3" placeholder="一行一个IP" @blur="handleBlurSave" />
      </el-form-item>
      <el-form-item label="IP白名单">
        <el-input type="textarea" v-model="securitySettings.ip.white" :rows="3" placeholder="一行一个IP" @blur="handleBlurSave" />
        <div class="form-helper">白名单 IP 将跳过 CC 防护、WAF、站点黑名单、区域屏蔽和防盗链；保存后自动同步到节点。</div>
      </el-form-item>
    </el-form>
      
    <div class="divider"></div>
    
    <!-- Cookie Domain -->
    <div class="section-title">设置Cookie域名</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="Cookie域名">
        <div style="display: items-center; gap: 10px;">
           <el-switch v-model="securitySettings.cookie.enable" @change="handleSave" />
           <el-input v-if="securitySettings.cookie.enable" v-model="securitySettings.cookie.domain" placeholder="例如: abc.com" style="width: 200px" @blur="handleBlurSave" />
        </div>
        <div class="form-helper">
          当主站(www.abc.com)引用图片站(img.abc.com)的资源时，如果两个域名都开启了防御，可以设置Cookie域名为abc.com。这样当访问者通过主站验证后，Cookie将在所有子域名间共享，从而图片站也会自动获得验证状态，确保资源正常加载。
        </div>
      </el-form-item>
    </el-form>

    <div class="divider"></div>
    
    <!-- Block Settings -->
    <div class="section-title">屏蔽设置</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="屏蔽透明代理">
        <el-switch v-model="securitySettings.block.transparentProxy" @change="handleSave" />
        <div class="form-helper">透明代理即网上免费公开的代理，带有x-forwarded-for请求头的</div>
      </el-form-item>
    </el-form>
    
    <div class="divider"></div>
      
    <div class="section-title">区域屏蔽</div>
    <el-form label-width="120px">
      <el-form-item label="区域选择">
        <CountrySelector v-model="securitySettings.regions" @change="handleSave" />
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, reactive, computed } from 'vue'
import { ArrowDown, ArrowUp, Delete, EditPen } from '@element-plus/icons-vue'
import CountrySelector from '@/components/CountrySelector.vue'
import { useSiteSettings } from '@/composables/useSiteSettings'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import { cacheInputValue, shouldSkipBlurSave } from '@/utils/saveGuard'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  }
})

const emit = defineEmits(['update:modelValue'])
const { saveSettings } = useSiteSettings()

// Local settings sync
const localSettings = ref(props.modelValue) // Not strictly used as we bind directly to securitySettings in template via useSiteSettings hook's returned object? 
// Wait, useSiteSettings returns `siteSettings` which is global state. 
// But here the component receives `modelValue`.
// In `Manage.vue`, `<SecurityConfig v-model="siteSettings.security" />`.
// So `props.modelValue` refers to `siteSettings.security`.
// WE SHOULD USE `props.modelValue` directly or a computed wrapping it.
// BUT `useSiteSettings` is single instance. Ideally we use that directly to avoid prop drilling if we are already inside the scope.
// However, the component is designed with v-model. 
// Let's stick to `securitySettings` from `useSiteSettings` if strictly needed, or just map `props.modelValue` to `securitySettings` ref.
// Actually, to avoid confusion, let's use the provided `siteSettings` from composable as the source of truth if `props.modelValue` is not reliable or sync issues.
// But valid approach: use `siteSettings` from `useSiteSettings` directly since this is a specific management component.
// The existing code did: `const securitySettings = localSettings`.
// I will use `siteSettings.security` directly from the hook to ensure full reactivity.

const { siteSettings } = useSiteSettings()
const securitySettings = siteSettings.security // Helper ref

// System Rules
const systemRules = ref([])
const userRules = ref([])
const fetchSystemRules = async () => {
  try {
    const { data } = await request.get('/rules/cc/groups')
    if (data.list) {
      systemRules.value = data.list.filter(item => item.is_system)
      userRules.value = data.list.filter(item => !item.is_system)
    }
  } catch (err) {
    console.error('Failed to fetch system rules', err)
  }
}

const computedMode = computed({
  get: () => {
    // If the current mode ID exists in systemRules, return it (selects that radio)
    const currentId = securitySettings.cc.mode
    if (systemRules.value.some(r => r.id === currentId)) {
      return currentId
    }
    // Otherwise it's custom (or invalid/legacy)
    return 'custom'
  },
  set: (val) => {
    if (val === 'custom') {
      // If switching to custom, default to first user rule if current isn't one
      if (userRules.value.length > 0) {
        // Only change if not currently pointing to a valid user rule
        const currentIsUser = userRules.value.some(r => r.id === securitySettings.cc.mode)
        if (!currentIsUser) {
           securitySettings.cc.mode = userRules.value[0].id
           handleSave()
        }
      } else {
        // No user rules? We can't really set a valid ID. Maybe keep current or prompt?
        // Logic: allow selecting 'custom' radio even if empty, UI shows "No rules"
      }
    } else {
      // System rule ID selected
      securitySettings.cc.mode = val
      handleSave()
    }
  }
})

// Auto Switch QPS Logic
const qpsSelection = ref(200)

// Watch for initial load to set qpsSelection
watch(() => securitySettings.cc.autoSwitch.qps, (val) => {
  if ([20, 50, 200].includes(val)) {
    qpsSelection.value = val
  } else {
    qpsSelection.value = 'custom'
  }
}, { immediate: true })

const handleQpsChange = (val) => {
  if (val !== 'custom') {
    securitySettings.cc.autoSwitch.qps = val
    saveSettings(true)
  }
}

// Time type change
const handleTimeTypeChange = (type, isCustom) => {
  if (type === 'black') {
    if (!isCustom) {
      securitySettings.ip.blackTime = 3600
    }
    saveSettings(true)
  } else {
    if (!isCustom) {
      securitySettings.ip.whiteTime = 21600
    }
    saveSettings(true)
  }
}

const handleSave = () => {
  saveSettings(true)
}

const handleBlurSave = (event) => {
  if (shouldSkipBlurSave(event, { skipEmpty: true })) {
    return
  }
  handleSave()
}

// --- Custom Rules Logic ---

const matchItems = [
  { label: '匹配所有请求', value: 'all' },
  { label: 'IP地址', value: 'ip' },
  { label: '域名', value: 'domain' },
  { label: '请求URI', value: 'uri' },
  { label: '请求URI(不带参数)', value: 'uri_no_args' },
  { label: '请求头', value: 'header' },
  { label: '独立UA数量', value: 'ua_count' },
  { label: '404状态码数量', value: '404_count' },
  { label: '请求方法', value: 'method' },
  { label: '浏览器UA', value: 'ua' },
  { label: '请求来源', value: 'referer' },
  { label: '国家代码', value: 'country' },
  { label: 'AS号码', value: 'as' },
  { label: '省份', value: 'province' },
  { label: '城市', value: 'city' },
  { label: '运营商', value: 'isp' },
  { label: 'HTTP版本', value: 'http_version' },
  { label: '请求头accept_language', value: 'header_accept_language' }
]

const operators = [
  { label: '等于', value: 'eq' },
  { label: '不等于', value: 'neq' },
  { label: '包含', value: 'contains' },
  { label: '不包含', value: 'not_contains' },
  { label: '前缀匹配', value: 'prefix' },
  { label: '后缀匹配', value: 'suffix' },
  { label: '正则匹配', value: 'regex' },
  { label: '正则不匹配', value: 'not_regex' },
  { label: '存在', value: 'exists' },
  { label: '不存在', value: 'not_exists' },
  { label: '在IP段', value: 'ip_range' },
  { label: '不在IP段', value: 'not_ip_range' }
]

const actions = [
  { label: '放行', value: 'allow' },
  { label: '拉黑', value: 'block' },
  { label: '请求频率', value: 'limit_rate' },
  { label: '无感验证', value: 'invisible' },
  { label: '5秒盾', value: '5s' },
  { label: '点击验证', value: 'click' },
  { label: '点击(简单)', value: 'click_simple' },
  { label: '滑动验证', value: 'slide' },
  { label: '滑动(简单)', value: 'slide_simple' },
  { label: '验证码', value: 'captcha' },
  { label: '旋转图片', value: 'rotate' },
  { label: '302跳转', value: '302' },
  { label: 'URL鉴权', value: 'url_auth' }
]

const ruleDialog = reactive({
  visible: false,
  mode: 'create',
  index: -1
})

const ruleForm = reactive({
  matchers: [],
  action: 'block',
  actionParams: {
    seconds: 10,
    requests: 10,
    urlRequests: 10,
    blockOnFail: true
  },
  breakMatch: false,
  remark: '',
  on: true
})

const newMatcher = reactive({
  key: '',
  operator: 'eq',
  value: ''
})

const openRuleDialog = (mode, index = -1) => {
  ruleDialog.mode = mode
  ruleDialog.index = index
  ruleDialog.visible = true
  
  if (mode === 'create') {
    Object.assign(ruleForm, {
      matchers: [],
      action: 'block',
      actionParams: { seconds: 10, requests: 10, urlRequests: 10, blockOnFail: true },
      breakMatch: false,
      remark: '',
      on: true
    })
  } else {
    // Edit
    const rule = securitySettings.cc.customRules[index]
    Object.assign(ruleForm, JSON.parse(JSON.stringify(rule)))
    if (!ruleForm.actionParams) {
       ruleForm.actionParams = { seconds: 10, requests: 10, urlRequests: 10, blockOnFail: true }
    }
  }
}

const addMatcher = () => {
  if (!newMatcher.key) return
  if (newMatcher.key !== 'all' && !newMatcher.value && !['exists', 'not_exists'].includes(newMatcher.operator)) {
     // Validate value
  }
  
  ruleForm.matchers.push({ ...newMatcher })
  newMatcher.key = ''
  newMatcher.value = ''
  newMatcher.operator = 'eq'
}

const removeMatcher = (idx) => {
  ruleForm.matchers.splice(idx, 1)
}

const saveRule = () => {
  const newRule = JSON.parse(JSON.stringify(ruleForm))
  if (ruleDialog.mode === 'create' && newRule.on === undefined) {
    newRule.on = true
  }
  if (ruleDialog.mode === 'create') {
    if (!securitySettings.cc.customRules) securitySettings.cc.customRules = []
    securitySettings.cc.customRules.push(newRule)
  } else {
    securitySettings.cc.customRules[ruleDialog.index] = newRule
  }
  ruleDialog.visible = false
  handleSave()
}

const deleteRule = (idx) => {
  securitySettings.cc.customRules.splice(idx, 1)
  handleSave()
}

const moveRule = (index, direction) => {
  const rules = securitySettings.cc.customRules
  if (!rules?.length) return
  const target = index + direction
  if (target < 0 || target >= rules.length) return
  const [item] = rules.splice(index, 1)
  rules.splice(target, 0, item)
  handleSave()
}

const editRule = (idx) => {
  openRuleDialog('update', idx)
}

// Helpers
const getMatcherText = (m) => {
  if (m.key === 'all') return '匹配所有请求'
  const k = matchItems.find(i => i.value === m.key)?.label || m.key
  const o = operators.find(i => i.value === m.operator)?.label || m.operator
  return `${k} ${o} ${m.value || ''}`
}

const getActionText = (val) => {
  return actions.find(i => i.value === val)?.label || val
}

const isVerificationAction = (val) => {
  return ['invisible', '5s', 'click', 'click_simple', 'slide', 'slide_simple', 'captcha', 'rotate'].includes(val)
}

onMounted(() => {
  fetchSystemRules()
})
</script>

<style scoped>
.security-config {
  padding: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 20px;
  padding-left: 10px;
  border-left: 3px solid var(--el-color-primary);
}

.divider {
  height: 1px;
  background-color: var(--el-border-color-lighter);
  margin: 24px 0;
}

.form-helper {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  margin-top: 6px;
}

.matcher-config {
  border: 1px solid var(--control-border);
  padding: 10px;
  border-radius: 4px;
  background: var(--control-bg);
  color: var(--control-text);
}

.matcher-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 5px;
  background: var(--control-bg-hover);
  color: var(--control-text);
  padding: 5px 10px;
  border-radius: 4px;
}

.matcher-text {
  color: var(--control-text);
  font-size: 13px;
}

.matcher-add {
  display: flex;
  align-items: center;
  margin-top: 10px;
  flex-wrap: wrap;
  gap: 6px;
}

.action-grid {
  display: flex;
  flex-wrap: wrap;
}

.rule-params {
  margin-top: 10px;
  background: var(--control-bg);
  border: 1px solid var(--control-border);
  padding: 10px;
  border-radius: 4px;
  color: var(--control-text);
}

.rule-actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.rule-actions__edit {
  --el-button-bg-color: var(--el-color-success);
  --el-button-border-color: var(--el-color-success);
  --el-button-hover-bg-color: var(--el-color-success-light-3);
  --el-button-hover-border-color: var(--el-color-success-light-3);
  --el-button-active-bg-color: var(--el-color-success-dark-2);
  --el-button-active-border-color: var(--el-color-success-dark-2);
  color: var(--el-color-white);
  padding: 5px 8px;
}

.rule-actions__sort {
  display: inline-flex;
  border-radius: var(--el-border-radius-base);
  overflow: hidden;
}

.rule-actions__sort .el-button {
  --el-button-bg-color: var(--el-color-primary);
  --el-button-border-color: var(--el-color-primary);
  --el-button-hover-bg-color: var(--el-color-primary-light-3);
  --el-button-hover-border-color: var(--el-color-primary-light-3);
  --el-button-active-bg-color: var(--el-color-primary-dark-2);
  --el-button-active-border-color: var(--el-color-primary-dark-2);
  color: var(--el-color-white);
  margin: 0;
  padding: 5px 8px;
  border-radius: 0;
}

.rule-actions__sort .el-button + .el-button {
  margin-left: 0;
  border-left: 1px solid color-mix(in srgb, var(--el-color-white) 35%, transparent);
}

.rule-actions__delete {
  --el-button-bg-color: var(--el-color-danger);
  --el-button-border-color: var(--el-color-danger);
  --el-button-hover-bg-color: var(--el-color-danger-light-3);
  --el-button-hover-border-color: var(--el-color-danger-light-3);
  --el-button-active-bg-color: var(--el-color-danger-dark-2);
  --el-button-active-border-color: var(--el-color-danger-dark-2);
  color: var(--el-color-white);
  padding: 5px 8px;
}
</style>
