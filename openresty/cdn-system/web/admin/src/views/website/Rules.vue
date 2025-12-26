<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane label="CC规则" name="cc">
        <el-tabs v-model="ccActiveTab">
          <el-tab-pane label="规则组" name="groups">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateGroup">新增分组</el-button>
              <el-select v-model="listQuery.status" placeholder="状态" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="禁用" value="off" />
              </el-select>
              <el-input v-model="listQuery.name" placeholder="分组名称，模糊搜索" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchGroups">查询</el-button>
            </div>

            <el-table :data="groupsList" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                <template #default="{row}">{{ row.is_system ? '\u7cfb\u7edf' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统" width="100" align="center">
                <template #default="{row}">
                  <el-tag type="success" v-if="row.is_system" effect="dark" size="small" style="border-radius: 50%; width: 20px; height: 20px; padding: 0;">&nbsp;</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="显示" width="100" align="center">
                <template #default="{row}">
                  <el-tag type="success" v-if="row.is_show" effect="dark" size="small" style="border-radius: 50%; width: 20px; height: 20px; padding: 0;">&nbsp;</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="显示" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '\u542f\u7528' : '\u7981\u7528' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="sort_order" label="排序" width="80" />
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small" @click="handleEditGroup(row)">编辑</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="匹配器" name="matchers">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateMatcher">新增匹配器</el-button>
              <el-select v-model="matcherListQuery.status" placeholder="状态" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="禁用" value="off" />
              </el-select>
              <el-input v-model="matcherListQuery.name" placeholder="匹配器名称" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchMatchers">查询</el-button>
            </div>

            <el-table :data="matchers" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                <template #default="{row}">{{ row.is_system ? '\u7cfb\u7edf' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统" width="100" align="center">
                <template #default="{row}">
                  <el-icon v-if="row.is_system" color="#67C23A"><Select /></el-icon>
                </template>
              </el-table-column>
              <el-table-column label="显示" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '\u542f\u7528' : '\u7981\u7528' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small" @click="handleEditMatcher(row)">编辑</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="过滤器" name="filters">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateFilter">新增过滤器</el-button>
              <el-select v-model="filterListQuery.status" placeholder="状态" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="禁用" value="off" />
              </el-select>
              <el-input v-model="filterListQuery.name" placeholder="过滤器名称" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchFilters">查询</el-button>
            </div>

            <el-table :data="filters" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                <template #default="{row}">{{ row.is_system ? '\u7cfb\u7edf' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统" width="100" align="center">
                <template #default="{row}">
                  <el-icon v-if="row.is_system" color="#67C23A"><Select /></el-icon>
                </template>
              </el-table-column>
              <el-table-column prop="type" label="类型" width="150" />
              <el-table-column label="显示" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '\u542f\u7528' : '\u7981\u7528' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="normal" @click="handleEditFilter(row)">编辑</el-button>
                  <el-button type="danger" link size="normal" @click="deleteFilter(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </el-tab-pane>

      <el-tab-pane label="ACL规则" name="acl">
        <div class="filter-container">
          <el-button type="primary" class="filter-item" @click="openAclDialog()">新增ACL</el-button>
          <el-select v-model="aclQuery.status" placeholder="状态" class="filter-item" style="width: 120px; margin-left:10px;">
            <el-option label="启用" value="on" />
            <el-option label="禁用" value="off" />
          </el-select>
          <el-input v-model="aclQuery.name" placeholder="ACL名称" style="width: 200px; margin-left: 10px;" class="filter-item" />
          <el-button class="filter-item" type="primary" :icon="Search" @click="fetchAcl">查询</el-button>
        </div>

        <el-table :data="aclList" border fit highlight-current-row style="width: 100%">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="名称" min-width="160" />
          <el-table-column prop="default_action" label="默认动作" width="120">
            <template #default="{row}">
              {{ row.default_action === 'deny' ? '\u62d2\u7edd' : '\u5141\u8bb8' }}
            </template>
          </el-table-column>
          <el-table-column prop="enable" label="用户" width="100">
            <template #default="{row}">
              <el-tag :type="row.enable ? 'success' : 'info'">{{ row.enable ? '\u542f\u7528' : '\u7981\u7528' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="create_time" label="创建时间" width="160" />
          <el-table-column label="操作" width="160" align="center">
            <template #default="{row}">
              <el-button link type="primary" size="small" @click="openAclDialog(row)">编辑</el-button>
              <el-button link type="danger" size="small" @click="deleteAcl(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="800px">
      <el-form :model="tempGroup" label-position="right" label-width="100px" style="width: 700px; margin-left:50px;">
        <el-form-item label="类型">
          <el-radio-group v-model="tempGroup.type">
            <el-radio label="system">系统</el-radio>
            <el-radio label="user">用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="tempGroup.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="tempGroup.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item label="规则">
          <el-button type="primary" plain size="small" @click="handleAddRule">新增规则</el-button>
          <el-table :data="tempGroup.rules" border style="width: 100%; margin-top: 10px;" size="small">
            <el-table-column prop="matcher_name" label="匹配器" />
            <el-table-column prop="filter1_name" label="过滤器" />
            <el-table-column prop="action" label="允许" />
            <el-table-column label="拒绝" width="60">
              <template #default="{$index}">
                <el-button link type="danger" @click="removeRule($index)">取消</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">确定</el-button>
          <el-button type="primary" @click="saveGroup">新增</el-button>
        </div>
      </template>

      <el-dialog width="600px" v-model="innerVisible" title="新增规则" append-to-body>
        <el-form :model="tempRule" label-width="100px">
          <el-form-item label="匹配器">
            <el-select v-model="tempRule.matcher_id" placeholder="请选择匹配器" style="width: 100%">
              <el-option v-for="item in matchers" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="过滤器">
            <el-select v-model="tempRule.filter1_id" placeholder="请选择匹配器" style="width: 100%">
              <el-option v-for="item in filters" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="动作">
            <el-select v-model="tempRule.action" placeholder="请选择动作" style="width: 100%">
              <el-option label="阻断" value="block" />
              <el-option label="放行" value="allow" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <div class="dialog-footer">
            <el-button @click="innerVisible = false">删除</el-button>
            <el-button type="primary" @click="confirmAddRule">取消</el-button>
          </div>
        </template>
      </el-dialog>
    </el-dialog>

    <el-dialog title="新增匹配器" v-model="matcherDialogVisible" width="800px">
      <el-form :model="tempMatcher" label-width="80px">
        <el-form-item label="类型">
          <el-radio-group v-model="tempMatcher.type">
            <el-radio label="system">系统</el-radio>
            <el-radio label="user">用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="tempMatcher.name" placeholder="请选择动作" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="tempMatcher.remark" placeholder="请选择动作" />
        </el-form-item>
        <el-form-item label="规则">
          <div style="width: 100%">
            <el-row :gutter="10" style="margin-bottom: 5px; font-weight: bold; color: #606266; font-size:12px;">
              <el-col :span="6">条件项</el-col>
              <el-col :span="4">操作符</el-col>
              <el-col :span="10">匹配值</el-col>
              <el-col :span="4">操作</el-col>
            </el-row>
            <div v-for="(rule, index) in tempMatcher.rules" :key="index" style="margin-bottom: 10px;">
              <el-row :gutter="10">
                <el-col :span="6">
                  <el-select v-model="rule.item" placeholder="请选择" size="small" @change="handleMatcherItemChange(rule)">
                    <el-option v-for="opt in matcherItemOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
                  </el-select>
                  <el-input
                    v-if="isHeaderItem(rule.item)"
                    v-model="rule.header"
                    placeholder="请求头名称，如: user-agent"
                    size="small"
                    style="margin-top: 6px;"
                  />
                  <el-input
                    v-else-if="isStatItem(rule.item)"
                    v-model="rule.seconds"
                    placeholder="统计秒数"
                    size="small"
                    style="margin-top: 6px;"
                  />
                </el-col>
                <el-col :span="4">
                  <el-select v-if="!isStatItem(rule.item)" v-model="rule.operator" placeholder="请选择" size="small">
                    <el-option v-for="opt in getMatcherOperatorOptions(rule.item)" :key="opt.value" :label="opt.label" :value="opt.value" />
                  </el-select>
                  <el-select v-else v-model="rule.operator" size="small" disabled>
                    <el-option label="大于" value="gt" />
                  </el-select>
                </el-col>
                <el-col :span="10">
                  <el-input
                    v-model="rule.value"
                    :placeholder="getMatcherValuePlaceholder(rule)"
                    type="textarea"
                    :rows="1"
                    size="small"
                  />
                </el-col>
                <el-col :span="4">
                  <el-button type="primary" link @click="addMatcherRule">新增</el-button>
                  <el-button type="danger" link @click="removeMatcherRule(index)" v-if="tempMatcher.rules.length > 1">删除</el-button>
                </el-col>
              </el-row>
            </div>
          </div>
          <div style="font-size: 12px; color: #999; margin-top: 5px;">
            同一规则内条件为 AND，不同规则之间为 OR
          </div>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="tempMatcher.is_on" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="matcherDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveMatcher">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="filterDialogVisible" :title="filterDialogTitle" width="820px">
      <el-form :model="tempFilter" label-width="100px">
        <el-form-item label="类型">
          <el-radio-group v-model="tempFilter.type">
            <el-radio label="system">系统</el-radio>
            <el-radio label="user">用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="tempFilter.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="tempFilter.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item label="执行过滤">
          <el-radio-group v-model="tempFilter.action">
            <el-radio v-for="opt in filterActionOptions" :key="opt.value" :label="opt.value">{{ opt.label }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="showMatchMode" label="匹配模式">
          <el-radio-group v-model="tempFilter.match_mode">
            <el-radio label="continue">继续下一条规则</el-radio>
            <el-radio label="stop">停止匹配</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="showBlacklist" label="是否拉黑">
          <el-radio-group v-model="tempFilter.blacklist">
            <el-radio :label="true">拉黑</el-radio>
            <el-radio :label="false">不拉黑</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="tempFilter.action === 'rate'" label="请求频率">
          <div class="rate-fields">
            <el-input v-model="tempFilter.within_second" placeholder="在" style="width: 220px;">
              <template #append>秒内</template>
            </el-input>
            <el-input v-model="tempFilter.max_req" placeholder="限制总请求" style="margin-top: 8px;">
              <template #append>次</template>
            </el-input>
            <el-input v-model="tempFilter.max_req_per_uri" placeholder="限制同URL最大请求" style="margin-top: 8px;">
              <template #append>次</template>
            </el-input>
          </div>
        </el-form-item>
        <el-form-item v-if="tempFilter.action === 'url_auth'" label="鉴权方式">
          <el-select v-model="tempFilter.auth_method" placeholder="鉴权方式" style="width: 100%;">
            <el-option label="鉴权方式A" value="a" />
            <el-option label="鉴权方式B" value="b" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="tempFilter.action === 'url_auth'" label="IP鉴权">
          <el-switch v-model="tempFilter.ip_auth" />
        </el-form-item>
        <el-form-item v-if="tempFilter.action === 'url_auth'" label="密钥(16-32位)">
          <el-input v-model="tempFilter.auth_secret" placeholder="密钥" />
        </el-form-item>
        <el-form-item v-if="tempFilter.action === 'url_auth'" label="签名参数名">
          <el-input v-model="tempFilter.auth_param_sign" placeholder="sign" />
        </el-form-item>
        <el-form-item v-if="tempFilter.action === 'url_auth'" label="时间戳参数名">
          <el-input v-model="tempFilter.auth_param_time" placeholder="t" />
        </el-form-item>
        <el-form-item v-if="tempFilter.action === 'url_auth'" label="最大允许时间偏差">
          <el-input v-model="tempFilter.auth_time_diff" placeholder="180">
            <template #append>秒</template>
          </el-input>
        </el-form-item>
        <el-form-item v-if="tempFilter.action === 'url_auth'" label="签名使用次数(0不限制)">
          <el-input v-model="tempFilter.auth_use_limit" placeholder="10" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="tempFilter.enable" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="filterDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveFilter">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="aclDialogVisible" :title="aclDialogTitle" width="680px">
      <el-form :model="aclForm" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="aclForm.name" placeholder="请选择动作" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="aclForm.des" placeholder="请选择动作" />
        </el-form-item>
        <el-form-item label="默认动作">
          <el-radio-group v-model="aclForm.default_action">
            <el-radio label="allow">允许</el-radio>
            <el-radio label="deny">拒绝</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="aclForm.enable" />
        </el-form-item>
        <el-form-item label="规则">
          <el-button type="primary" plain size="small" @click="addAclRule">新增规则</el-button>
          <el-table :data="aclForm.rules" border style="width: 100%; margin-top: 10px;" size="small">
            <el-table-column label="IP">
              <template #default="{ row }">
                <el-input v-model="row.ip" placeholder="IP?CIDR" />
              </template>
            </el-table-column>
            <el-table-column label="动作" width="140">
              <template #default="{ row }">
                <el-select v-model="row.action" style="width: 100%;">
                  <el-option label="放行" value="allow" />
                  <el-option label="拒绝" value="deny" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ $index }">
                <el-button link type="danger" @click="removeAclRule($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="aclDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAcl">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { Search, ArrowDown, Select } from '@element-plus/icons-vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import request from '@/utils/request'

const activeTab = ref('cc')
const ccActiveTab = ref('groups')
const groupsList = ref([])
const matchers = ref([])
const filters = ref([])
const aclList = ref([])

const listQuery = reactive({
  name: '',
  status: ''
})
const matcherListQuery = reactive({
  name: '',
  status: ''
})
const filterListQuery = reactive({
  name: '',
  status: ''
})
const aclQuery = reactive({
  name: '',
  status: ''
})

const dialogFormVisible = ref(false)
const innerVisible = ref(false)
const matcherDialogVisible = ref(false)
const dialogStatus = ref('')
const textMap = {
  update: '\u7f16\u8f91\u89c4\u5219\u7ec4',
  create: '\u65b0\u589e\u89c4\u5219\u7ec4'
}

const tempGroup = reactive({
  id: undefined,
  type: 'system',
  name: '',
  remark: '',
  rules: []
})

const tempRule = reactive({
  matcher_id: undefined,
  matcher_name: '',
  filter1_id: undefined,
  filter1_name: '',
  action: 'block'
})

const tempMatcher = reactive({
  type: 'system',
  name: '',
  remark: '',
  is_on: true,
  rules: [
    { item: 'ip', operator: 'eq', value: '', header: '', seconds: '' }
  ]
})

const matcherItemOptions = [
  { label: '匹配所有请求', value: 'all' },
  { label: 'IP地址', value: 'ip' },
  { label: '域名', value: 'domain' },
  { label: '请求URI', value: 'uri' },
  { label: '请求URI(不带参数)', value: 'uri_no_args' },
  { label: '请求头', value: 'header' },
  { label: '独立UA数量', value: 'ua_count' },
  { label: '404状态码数量', value: 'status_404' },
  { label: '请求方法', value: 'method' },
  { label: '浏览器UA', value: 'ua' },
  { label: '请求来源', value: 'referer' },
  { label: '国家代码', value: 'country' },
  { label: 'AS号码', value: 'asn' },
  { label: '省份', value: 'province' },
  { label: '城市', value: 'city' },
  { label: '运营商', value: 'isp' },
  { label: 'HTTP版本', value: 'http_version' },
  { label: '请求头accept_language', value: 'accept_language' }
]

const matcherOperatorOptions = [
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
  { label: '在IP段', value: 'in_ip' },
  { label: '不在IP段', value: 'not_in_ip' }
]

const isHeaderItem = item => item === 'header'
const isStatItem = item => item === 'ua_count' || item === 'status_404'

const getMatcherOperatorOptions = () => matcherOperatorOptions

const getMatcherValuePlaceholder = rule => {
  switch (rule.item) {
    case 'http_version':
      return '输入匹配值,一行一个,如:\nHTTP/1.0\nHTTP/1.1\nHTTP/2.0'
    case 'ip':
      return '输入匹配值,一行一个,如:\n1.1.1.1\n2.2.2.2'
    case 'domain':
      return '输入匹配值,一行一个,如:\nexample.com\nwww.example.com'
    case 'uri':
      return '输入匹配值,一行一个,如:\n/index.html\n/api/v1/'
    case 'uri_no_args':
      return '输入匹配值,一行一个,如:\n/index.html\n/api/v1/'
    case 'method':
      return '输入匹配值,一行一个,如:\nGET\nPOST'
    case 'accept_language':
      return '输入匹配值,一行一个,如:\nzh-CN\nen-US'
    case 'ua_count':
      return '输入次数'
    case 'status_404':
      return '输入次数'
    case 'header':
      return '输入匹配值,一行一个'
    default:
      return '输入匹配值,一行一个'
  }
}

const handleMatcherItemChange = (rule) => {
  if (isStatItem(rule.item)) {
    rule.operator = 'gt'
    rule.seconds = rule.seconds || '10'
  } else if (!rule.operator) {
    rule.operator = 'eq'
  }
  if (isHeaderItem(rule.item) && !rule.header) {
    rule.header = ''
  }
}

const filterDialogVisible = ref(false)
const tempFilter = reactive({
  id: 0,
  type: 'system',
  name: '',
  remark: '',
  enable: true,
  action: 'allow',
  match_mode: 'continue',
  blacklist: false,
  within_second: '',
  max_req: '',
  max_req_per_uri: '',
  auth_method: 'a',
  ip_auth: false,
  auth_secret: '',
  auth_param_sign: 'sign',
  auth_param_time: 't',
  auth_time_diff: '180',
  auth_use_limit: '10'
})
const filterDialogTitle = computed(() => (tempFilter.id ? '编辑过滤器' : '新增过滤器'))
const filterActionOptions = [
  { label: '放行', value: 'allow' },
  { label: '拉黑', value: 'block' },
  { label: '请求频率', value: 'rate' },
  { label: '无感验证', value: 'silent' },
  { label: '5秒盾', value: 'shield_5s' },
  { label: '点击验证', value: 'click' },
  { label: '点击(简单)', value: 'click_simple' },
  { label: '滑动验证', value: 'slide' },
  { label: '滑动(简单)', value: 'slide_simple' },
  { label: '验证码', value: 'captcha' },
  { label: '旋转图片', value: 'rotate' },
  { label: '302跳转', value: 'redirect_302' },
  { label: 'URL鉴权', value: 'url_auth' }
]
const showMatchMode = computed(() => !['allow', 'block'].includes(tempFilter.action))
const showBlacklist = computed(() => !['allow', 'block', 'rate'].includes(tempFilter.action))

const aclDialogVisible = ref(false)
const aclForm = reactive({
  id: 0,
  name: '',
  des: '',
  default_action: 'allow',
  enable: true,
  rules: []
})
const aclDialogTitle = computed(() => (aclForm.id ? '\u7f16\u8f91ACL' : '\u65b0\u589eACL'))

const fetchGroups = async () => {
  const { data } = await request.get('/rules/cc/groups', { params: listQuery })
  groupsList.value = data.list || []
}

const fetchMatchers = async () => {
  const { data } = await request.get('/rules/cc/matchers', { params: matcherListQuery })
  matchers.value = data.list || []
}

const fetchFilters = async () => {
  const { data } = await request.get('/rules/cc/filters', { params: filterListQuery })
  filters.value = data.list || []
}

const fetchAcl = async () => {
  const { data } = await request.get('/rules/acl', { params: aclQuery })
  aclList.value = data.list || []
}

const handleCreateGroup = () => {
  dialogStatus.value = 'create'
  dialogFormVisible.value = true
  Object.assign(tempGroup, { id: undefined, type: 'system', name: '', remark: '', rules: [] })
}

const handleEditGroup = async row => {
  dialogStatus.value = 'update'
  dialogFormVisible.value = true
  const { data } = await request.get(`/rules/cc/groups/${row.id}`)
  Object.assign(tempGroup, data)
  // Ensure rules is an array
  if (!tempGroup.rules) tempGroup.rules = []
}

const saveGroup = async () => {
  const payload = { ...tempGroup }
  // Ensure internal boolean is set correctly based on type
  if (payload.type === 'system') {
    // Backend handles this but good to be explicit or leave it to backend
  } 
  
  try {
    if (tempGroup.id) {
      await request.put(`/rules/cc/groups/${tempGroup.id}`, payload)
      ElMessage.success('更新成功')
    } else {
      await request.post('/rules/cc/groups', payload)
      ElMessage.success('创建成功')
    }
    dialogFormVisible.value = false
    fetchGroups()
  } catch (error) {
    // Error handled by request interceptor usually
  }
}

const handleAddRule = async () => {
  if (matchers.value.length === 0) await fetchMatchers()
  if (filters.value.length === 0) await fetchFilters()
  Object.assign(tempRule, {
    matcher_id: undefined,
    matcher_name: '',
    filter1_id: undefined,
    filter1_name: '',
    action: 'block'
  })
  innerVisible.value = true
}

const confirmAddRule = () => {
  const m = matchers.value.find(i => i.id === tempRule.matcher_id)
  if (m) tempRule.matcher_name = m.name
  const f1 = filters.value.find(i => i.id === tempRule.filter1_id)
  if (f1) tempRule.filter1_name = f1.name
  tempGroup.rules.push({ ...tempRule })
  innerVisible.value = false
}

const removeRule = index => {
  tempGroup.rules.splice(index, 1)
}

const handleCreateMatcher = () => {
  matcherDialogVisible.value = true
  Object.assign(tempMatcher, {
    id: undefined,
    type: 'system',
    name: '',
    remark: '',
    is_on: true,
    rules: [{ item: 'ip', operator: 'eq', value: '', header: '', seconds: '' }]
  })
}

const handleEditMatcher = async (row) => {
  matcherDialogVisible.value = true
  const { data } = await request.get(`/rules/cc/matchers/${row.id}`)
  
  const normalizedRules = (data.rules || []).map(rule => ({
    item: rule.item || 'ip',
    operator: rule.operator || (isStatItem(rule.item) ? 'gt' : 'eq'),
    value: rule.value || '',
    header: rule.header || '',
    seconds: rule.seconds || ''
  }))
  Object.assign(tempMatcher, {
    id: data.id,
    type: data.type,
    name: data.name,
    remark: data.remark || '',
    is_on: !!data.is_on,
    rules: normalizedRules.length ? normalizedRules : [{ item: 'ip', operator: 'eq', value: '', header: '', seconds: '' }]
  })
}

const addMatcherRule = () => {
  tempMatcher.rules.push({ item: 'ip', operator: 'eq', value: '', header: '', seconds: '' })
}

const removeMatcherRule = index => {
  tempMatcher.rules.splice(index, 1)
}

const saveMatcher = async () => {
  const payload = { ...tempMatcher }
  try {
    if (tempMatcher.id) {
      await request.put(`/rules/cc/matchers/${tempMatcher.id}`, payload)
      ElMessage.success('更新成功')
    } else {
      await request.post('/rules/cc/matchers', payload)
      ElMessage.success('创建成功')
    }
    matcherDialogVisible.value = false
    fetchMatchers()
  } catch (error) {
    // console.error(error)
  }
}

const handleCreateFilter = () => {
  filterDialogVisible.value = true
  Object.assign(tempFilter, {
    id: 0,
    type: 'system',
    name: '',
    remark: '',
    enable: true,
    action: 'allow',
    match_mode: 'continue',
    blacklist: false,
    within_second: '',
    max_req: '',
    max_req_per_uri: '',
    auth_method: 'a',
    ip_auth: false,
    auth_secret: '',
    auth_param_sign: 'sign',
    auth_param_time: 't',
    auth_time_diff: '180',
    auth_use_limit: '10'
  })
}

const handleEditFilter = async (row) => {
  const { data } = await request.get(`/rules/cc/filters/${row.id}`)
  Object.assign(tempFilter, {
    id: data.id,
    type: data.type,
    name: data.name,
    remark: data.remark || '',
    enable: !!data.enable,
    action: data.action || 'allow',
    match_mode: data.match_mode || 'continue',
    blacklist: !!data.blacklist,
    within_second: data.within_second || '',
    max_req: data.max_req || '',
    max_req_per_uri: data.max_req_per_uri || '',
    auth_method: data.auth?.method || 'a',
    ip_auth: !!data.auth?.ip_auth,
    auth_secret: data.auth?.secret || '',
    auth_param_sign: data.auth?.param_sign || 'sign',
    auth_param_time: data.auth?.param_time || 't',
    auth_time_diff: data.auth?.time_diff || '180',
    auth_use_limit: data.auth?.use_limit || '10'
  })
  filterDialogVisible.value = true
}

const saveFilter = async () => {
  const withinSecond = parseInt(tempFilter.within_second || '0', 10)
  const maxReq = parseInt(tempFilter.max_req || '0', 10)
  const maxReqPerURI = parseInt(tempFilter.max_req_per_uri || '0', 10)
  const payload = {
    type: tempFilter.type,
    name: tempFilter.name,
    remark: tempFilter.remark,
    enable: tempFilter.enable,
    action: tempFilter.action,
    match_mode: tempFilter.match_mode,
    blacklist: tempFilter.blacklist,
    within_second: withinSecond,
    max_req: maxReq,
    max_req_per_uri: maxReqPerURI,
    auth: {
      method: tempFilter.auth_method,
      ip_auth: tempFilter.ip_auth,
      secret: tempFilter.auth_secret,
      param_sign: tempFilter.auth_param_sign,
      param_time: tempFilter.auth_param_time,
      time_diff: tempFilter.auth_time_diff,
      use_limit: tempFilter.auth_use_limit
    }
  }
  if (tempFilter.id) {
    await request.put(`/rules/cc/filters/${tempFilter.id}`, payload)
  } else {
    await request.post('/rules/cc/filters', payload)
  }
  ElMessage.success('保存成功')
  filterDialogVisible.value = false
  fetchFilters()
}

const deleteFilter = row => {
  ElMessageBox.confirm('确定删除过滤器?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.delete(`/rules/cc/filters/${row.id}`).then(() => {
      ElMessage.success('删除成功')
      fetchFilters()
    })
  })
}

const openAclDialog = async row => {
  if (row && row.id) {
    const { data } = await request.get(`/rules/acl/${row.id}`)
    aclForm.id = data.id
    aclForm.name = data.name
    aclForm.des = data.des || ''
    aclForm.default_action = data.default_action || 'allow'
    aclForm.enable = !!data.enable
    aclForm.rules = data.rules || []
  } else {
    aclForm.id = 0
    aclForm.name = ''
    aclForm.des = ''
    aclForm.default_action = 'allow'
    aclForm.enable = true
    aclForm.rules = []
  }
  aclDialogVisible.value = true
}

const saveAcl = async () => {
  const payload = {
    name: aclForm.name,
    des: aclForm.des,
    default_action: aclForm.default_action,
    enable: aclForm.enable,
    rules: aclForm.rules
  }
  if (aclForm.id) {
    await request.put(`/rules/acl/${aclForm.id}`, payload)
  } else {
    await request.post('/rules/acl', payload)
  }
  ElMessage.success('\u4fdd\u5b58\u6210\u529f')
  aclDialogVisible.value = false
  fetchAcl()
}

const deleteAcl = row => {
  ElMessageBox.confirm('\u786e\u5b9a\u5220\u9664ACL\uff1f', '\u63d0\u793a', {
    confirmButtonText: '\u786e\u5b9a',
    cancelButtonText: '\u53d6\u6d88',
    type: 'warning'
  }).then(() => {
    request.delete(`/rules/acl/${row.id}`).then(() => {
      ElMessage.success('\u4fdd\u5b58\u6210\u529f')
      fetchAcl()
    })
  })
}

const addAclRule = () => {
  aclForm.rules.push({ ip: '', action: 'allow' })
}

const removeAclRule = index => {
  aclForm.rules.splice(index, 1)
}

onMounted(() => {
  fetchGroups()
  fetchMatchers()
  fetchFilters()
  fetchAcl()
})
</script>

<style scoped>
.filter-container {
  padding-bottom: 20px;
}
.rate-fields .el-input {
  width: 100%;
}
</style>
