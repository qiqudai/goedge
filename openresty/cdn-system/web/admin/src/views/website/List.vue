<template>
  <div class="app-container">
    <el-tabs v-model="activeTopTab" class="site-tabs" @tab-click="handleTopTab">
      <el-tab-pane label="网站列表" name="list" />
      <el-tab-pane label="默认设置" name="default" />
      <el-tab-pane label="DNS API" name="dns" />
      <el-tab-pane label="解析检测" name="resolve" />
    </el-tabs>

    <div v-if="activeTopTab === 'list'" class="filter-container">
      <div class="filter-left">
        <el-button type="primary" @click="openCreateDialog">添加网站</el-button>
        <el-button :disabled="!selectedRows.length" @click="openBatchEdit">批量修改</el-button>
        <el-button :disabled="!selectedRows.length" @click="handleApplyCert">申请证书</el-button>
        <el-dropdown trigger="click">
          <el-button>
            更多操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="handleBatchAction('enable')">启用</el-dropdown-item>
              <el-dropdown-item @click="handleBatchAction('disable')">禁用</el-dropdown-item>
              <el-dropdown-item @click="handleBatchAction('delete')">删除</el-dropdown-item>
              <el-dropdown-item @click="handleBatchAction('unlock')">解除黑名单</el-dropdown-item>
              <el-dropdown-item @click="handleBatchAction('clear_cache')">清空缓存</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        
        <el-divider direction="vertical" style="height: 32px; margin: 0 12px; border-color: #dcdfe6;" />

        <el-select v-model="listQuery.searchField" class="filter-item" style="width: 140px;">
          <el-option label="全字段" value="all" />
          <el-option label="域名" value="domain" />
          <el-option label="多域名" value="multi_domain" />
          <el-option label="源IP" value="origin" />
          <el-option label="网站分组" value="group" />
          <el-option label="网站ID" value="site_id" />
          <el-option label="CNAME" value="cname" />
          <el-option label="网站套餐" value="package" />
          <el-option label="HTTP监听端口" value="http_port" />
          <el-option label="HTTPS监听端口" value="https_port" />
        </el-select>
        <el-input
          v-model="listQuery.keyword"
          placeholder="输入域名, 模糊搜索"
          style="width: 200px;"
          class="filter-item"
          @keyup.enter="handleFilter"
        />
        <el-button type="primary" class="filter-item" @click="handleFilter">查询</el-button>
        <el-button class="filter-item" @click="handleExport">导出</el-button>
        <el-button link class="filter-item" @click="advancedVisible = true">高级搜索</el-button>
      </div>
    </div>

    <div v-if="activeTags.length" class="filter-tags">
      <el-tag v-for="tag in activeTags" :key="tag.key" closable @close="removeTag(tag.key)">
        {{ tag.label }}
      </el-tag>
      <el-button link class="filter-tags-clear" @click="clearFilters">清除</el-button>
    </div>

    <el-table
      v-if="activeTopTab === 'list'"
      v-loading="listLoading"
      :data="list"
      border
      fit
      highlight-current-row
      style="width: 100%;"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="domain_display" label="域名" min-width="220" show-overflow-tooltip />
      <el-table-column prop="listen_ports" label="监听端口" width="110" />
      <el-table-column prop="origin_display" label="源站" min-width="200" show-overflow-tooltip />
      <el-table-column prop="cname" label="CNAME" min-width="200" show-overflow-tooltip />
      <el-table-column label="HTTPS" width="80" align="center">
        <template #default="{ row }">
          <el-icon v-if="row.https" color="#67C23A"><CircleCheckFilled /></el-icon>
          <el-icon v-else color="#C0C4CC"><CircleCloseFilled /></el-icon>
        </template>
      </el-table-column>
      <el-table-column prop="group_name" label="分组" width="120" />
      <el-table-column prop="node_group_name" label="区域(线路组)" min-width="140" show-overflow-tooltip />
      <el-table-column label="状态" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status ? 'success' : 'info'">{{ row.status ? '正常' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="添加时间" width="180" />
      <el-table-column label="操作" width="150" align="center">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="handleManage(row)">管理</el-button>
          <el-dropdown trigger="click">
            <span class="link-more">
              更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleRowAction('enable', row)">启用</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('disable', row)">禁用</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('delete', row)">删除</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('unlock', row)">解除黑名单</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('clear_cache', row)">清空缓存</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="activeTopTab === 'list'" class="pagination-container">
      <el-pagination
        v-model:current-page="listQuery.page"
        v-model:page-size="listQuery.pageSize"
        :page-sizes="[10, 20, 30, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="handleFilter"
        @current-change="handleFilter"
      />
    </div>

    <div v-if="activeTopTab === 'default'" class="default-section">
      <el-card>
        <div class="default-toolbar">
          <el-button type="primary" size="normal" @click="openDefaultDialog()">新增设置</el-button>
          <el-button size="normal" :disabled="!selectedDefaults.length" @click="removeDefaultBatch">删除</el-button>
        </div>
        <el-table
          :data="defaultRows"
          v-loading="defaultLoading"
          border
          style="width: 100%;"
          @selection-change="handleDefaultSelection">
          <el-table-column type="selection" width="55" align="center" />
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column v-if="isAdmin" prop="user_name" label="用户" min-width="140" />
          <el-table-column prop="label" label="设置项" min-width="220" />
          <el-table-column prop="value" label="设置值" min-width="220" />
          <el-table-column label="生效范围" width="120">
            <template #default="{ row }">
              {{ row.scopeLabel }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" align="center">
            <template #default="{ row }">
              <el-button link type="primary" size="normal" @click="openDefaultDialog(row)">编辑</el-button>
              <el-button link type="danger" size="normal" @click="removeDefault(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <ResolvePage v-if="activeTopTab === 'resolve'" :hide-tabs="true" />

    <div v-if="activeTopTab === 'dns'" class="dnsapi-section">
      <div class="dnsapi-toolbar">
        <el-button type="primary" @click="openDnsapiDialog">新增DNS API</el-button>
        <el-button :disabled="!selectedDnsapi.length" @click="removeDnsapiBatch">删除</el-button>
      </div>
      <el-table v-loading="dnsapiLoading" :data="dnsapiList" border style="width: 100%;" @selection-change="handleDnsapiSelection">
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="uid" label="用户" width="120" />
        <el-table-column prop="name" label="名称" min-width="160" show-overflow-tooltip />
        <el-table-column prop="type" label="类型" width="140">
          <template #default="{ row }">
            {{ formatDnsType(row.type) }}
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="140" align="center">
          <template #default="{ row }">
            <el-button link type="primary" size="normal" @click="openDnsapiEdit(row)">编辑</el-button>
            <el-button link type="danger" size="normal" @click="removeDnsapi(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Add Group Dialog -->
    <el-dialog v-model="createGroupVisible" title="添加分组" width="400px">
      <el-form :model="createGroupForm" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="createGroupForm.name" placeholder="请输入分组名称" @keyup.enter="submitCreateGroup" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="createGroupForm.remark" placeholder="请输入备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createGroupVisible = false">取消</el-button>
        <el-button type="primary" @click="submitCreateGroup">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="createVisible" width="620px" title="添加网站">
      <el-tabs v-model="createTab" type="card">
        <el-tab-pane label="单个" name="single">
          <el-form :model="createForm" label-width="90px">
            <el-form-item v-if="isAdmin" label="所属用户">
              <el-select
                v-model.number="createForm.user_id"
                filterable
                remote
                clearable
                placeholder="搜索用户 (默认管理员)"
                :remote-method="loadUsers"
                :loading="userLoading">
                <el-option v-for="u in userOptions" :key="u.id" :label="formatUserLabel(u)" :value="u.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="网站套餐">
              <el-select v-model.number="createForm.user_package_id" clearable placeholder="选择套餐 (可选)" style="width: 100%;">
                <el-option v-for="p in packageOptions" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="网站域名">
              <el-input v-model="createForm.domains_input" placeholder="www.abc.com www.abc.com:8080 abc.com:80" />
            </el-form-item>
            <el-form-item label="源站地址">
              <el-input v-model="createForm.backends_input" placeholder="1.1.1.1或1.2.3.4:8080或www.abc.com:80" />
            </el-form-item>
            <el-form-item label="加速类型">
              <el-radio-group v-model="createForm.site_type">
                <el-radio label="website">网页加速(常用网站元素缓存)</el-radio>
                <el-radio label="api">API加速(无缓存快速回源)</el-radio>
                <el-radio label="download">大文件下载加速(压缩+分片回源)</el-radio>
                <el-radio label="custom">自定义(手动配置)</el-radio>
              </el-radio-group>
            </el-form-item>
            <div class="expand-more" @click="createMore = !createMore">
              <span>展开更多</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <div v-if="createMore" class="extra-fields">
              <el-form-item label="所属分组">
                <div style="display: flex; gap: 8px; width: 100%;">
                  <el-select v-model.number="createForm.group_id" clearable placeholder="网站分组, 可不选" style="flex: 1;">
                    <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
                  </el-select>
                  <el-button :icon="Plus" circle @click="openCreateGroupDialog" />
                </div>
              </el-form-item>
              <el-form-item label="DNS API">
                <el-select v-model.number="createForm.dns_provider_id" clearable placeholder="自动添加记录, 可不选" style="width: 100%;">
                  <el-option v-for="d in dnsOptions" :key="d.id" :label="d.name" :value="d.id" />
                </el-select>
              </el-form-item>
            </div>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="批量" name="batch">
          <el-form :model="batchForm" label-width="90px">
            <el-form-item v-if="isAdmin" label="所属用户">
              <el-select
                v-model.number="batchForm.user_id"
                filterable
                remote
                clearable
                placeholder="搜索用户 (默认管理员)"
                :remote-method="loadUsers"
                :loading="userLoading">
                <el-option v-for="u in userOptions" :key="u.id" :label="formatUserLabel(u)" :value="u.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="网站套餐">
              <el-select v-model.number="batchForm.user_package_id" clearable placeholder="选择套餐 (可选)" style="width: 100%;">
                <el-option v-for="p in packageOptions" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="网站数据">
              <el-input
                v-model="batchForm.data"
                type="textarea"
                rows="5"
                placeholder="数据格式以key=value的方式，一行一个网站配置&#10;domain=www.abc.com,abc.com|ip=1.1.1.1&#10;domain=www.qq.com|ip=1.1.1.1,2.2.2.2"
              />
              <div class="help-text">
                domain是网站域名，ip源站地址，配置项以 | 分隔。
                <el-link type="primary" :underline="false">了解更多</el-link>
              </div>
            </el-form-item>
            <el-form-item label="忽略错误">
              <el-switch v-model="batchForm.ignore_error" />
              <span class="help-text">有网站添加出错时，不中断，继续添加下一条。</span>
            </el-form-item>
            <div class="expand-more" @click="batchMore = !batchMore">
              <span>展开更多</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <div v-if="batchMore" class="extra-fields">
              <el-form-item label="所属分组">
                <div style="display: flex; gap: 8px; width: 100%;">
                  <el-select v-model="batchForm.group_id" clearable placeholder="网站分组, 可不选" style="flex: 1;">
                    <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
                  </el-select>
                  <el-button :icon="Plus" circle @click="openCreateGroupDialog" />
                </div>
              </el-form-item>
              <el-form-item label="DNS API">
                <el-select v-model="batchForm.dns_provider_id" clearable placeholder="自动添加记录, 可不选" style="width: 100%;">
                  <el-option v-for="d in dnsOptions" :key="d.id" :label="d.name" :value="d.id" />
                </el-select>
              </el-form-item>
            </div>
          </el-form>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateSubmit">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="dnsapiDialogVisible" title="DNS API" width="720px" class="dnsapi-dialog">
      <el-form :model="dnsapiForm" label-width="140px">
        <el-form-item label="名称">
          <el-input v-model="dnsapiForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="dnsapiForm.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item label="DNS">
          <el-select v-model="dnsapiForm.type" placeholder="请选择" style="width: 100%;" @change="resetDnsapiAuth">
            <el-option v-for="t in dnsapiTypes" :key="t.type" :label="t.name" :value="t.type" />
          </el-select>
        </el-form-item>
        <el-form-item label="验证信息" v-if="currentDnsapiType">
          <div class="dnsapi-fields">
            <div v-for="field in currentDnsapiType.fields" :key="field" class="dnsapi-field-row">
              <div class="dnsapi-field-label">{{ dnsapiFieldLabel(dnsapiForm.type, field) }}</div>
              <el-input v-model="dnsapiForm.credentials[field]" />
            </div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dnsapiDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitDnsapi">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="defaultDialogVisible" :title="defaultDialogTitle" width="520px">
      <el-form :model="defaultForm" label-width="90px">
        <el-form-item v-if="isAdmin" label="用户">
          <el-select
            v-model.number="defaultForm.user_id"
            filterable
            remote
            clearable
            placeholder="输入ID、邮箱、用户名、手机号搜索"
            :remote-method="loadUsers"
            :loading="userLoading">
            <el-option v-for="u in userOptions" :key="u.id" :label="formatUserLabel(u)" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="设置项">
          <el-select
            v-model="defaultForm.name"
            placeholder="请选择"
            :disabled="isAdmin && !defaultForm.user_id"
            style="width: 100%;">
            <el-option v-for="opt in defaultOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="设置值">
          <el-input-number
            v-if="defaultOptionType === 'number'"
            v-model="defaultForm.value"
            :min="0"
            controls-position="right"
            style="width: 100%;"
          />
          <el-switch
            v-else-if="defaultOptionType === 'bool'"
            v-model="defaultForm.boolValue"
          />
          <el-select
            v-else-if="defaultOptionType === 'select'"
            v-model="defaultForm.selectValue"
            placeholder="请选择"
            style="width: 100%;">
            <el-option v-for="opt in defaultOptionChoices" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
          <el-select
            v-else-if="defaultOptionType === 'multi'"
            v-model="defaultForm.multiValue"
            multiple
            collapse-tags
            placeholder="请选择"
            style="width: 100%;">
            <el-option v-for="opt in defaultOptionChoices" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
          <el-input
            v-else-if="defaultOptionType === 'text'"
            v-model="defaultForm.textValue"
            placeholder="请输入设置值"
          />
          <el-input
            v-else-if="defaultOptionType === 'lines'"
            v-model="defaultForm.textValue"
            type="textarea"
            rows="4"
            placeholder="一行一个"
          />
          <div v-else-if="defaultOptionType === 'headers'" class="header-list">
            <el-button type="primary" size="normal" @click="addDefaultHeader">新增请求头</el-button>
            <el-table :data="defaultForm.headers" border size="small" style="margin-top: 8px;">
              <el-table-column label="名称" min-width="160">
                <template #default="{ row }">
                  <el-input v-model="row.name" placeholder="Header 名称" />
                </template>
              </el-table-column>
              <el-table-column label="值" min-width="200">
                <template #default="{ row }">
                  <el-input v-model="row.value" placeholder="Header 值" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ $index }">
                  <el-button link type="danger" size="normal" @click="removeDefaultHeader($index)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
          <div v-else-if="defaultOptionType === 'region'" class="region-setting">
            <el-radio-group v-model="defaultForm.region_mode">
              <el-radio label="none">不设置</el-radio>
              <el-radio label="custom">自定义</el-radio>
            </el-radio-group>
            <CountrySelector
              v-if="defaultForm.region_mode === 'custom'"
              v-model="defaultForm.region_custom"
              style="margin-top: 8px;"
            />
          </div>
        </el-form-item>
        <el-form-item label="生效范围">
          <el-radio-group v-model="defaultForm.scope">
            <el-radio label="global">全局</el-radio>
            <el-radio label="group">网站分组</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="defaultForm.scope === 'group'" label="网站分组">
          <el-select
            v-model.number="defaultForm.group_id"
            placeholder="请选择分组"
            :disabled="isAdmin && !defaultForm.user_id"
            style="width: 100%;">
            <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="normal" @click="defaultDialogVisible = false">取消</el-button>
        <el-button type="primary" size="normal" @click="submitDefault">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="batchEditVisible" title="批量修改网站" width="720px">
      <div class="batch-header">正在修改的网站: {{ selectedIdsText }}</div>
      <div class="batch-dialog-body">
        <el-form label-width="90px">
          <el-collapse v-model="batchCollapse">
            <el-collapse-item title="基本设置" name="basic">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.group_id">所属分组</el-checkbox>
                <el-select v-model="batchEditForm.group_id" clearable placeholder="请选择" style="width: 70%;">
                  <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
                </el-select>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.dns_provider_id">DNS API</el-checkbox>
                <el-select v-model="batchEditForm.dns_provider_id" clearable placeholder="请选择" style="width: 70%;">
                  <el-option v-for="d in dnsOptions" :key="d.id" :label="d.name" :value="d.id" />
                </el-select>
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="HTTP设置" name="http">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.http_enable">状态设置</el-checkbox>
                <el-radio-group v-model="batchEditForm.http_enable">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.http_listen">监听端口</el-checkbox>
                <el-input v-model="batchEditForm.http_listen" placeholder="多个端口空格分隔" style="width: 70%;" />
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="HTTPS设置" name="https">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.https_enable">证书设置</el-checkbox>
                <el-radio-group v-model="batchEditForm.https_enable">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.https_listen">监听端口</el-checkbox>
                <el-input v-model="batchEditForm.https_listen" placeholder="多个端口空格分隔" style="width: 70%;" />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.force_https">强制HTTPS</el-checkbox>
                <el-radio-group v-model="batchEditForm.force_https">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.https_redirect_port">HTTPS转向端口</el-checkbox>
                <el-input v-model="batchEditForm.https_redirect_port" placeholder="请输入转向到的端口" style="width: 70%;" />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.hsts">HSTS</el-checkbox>
                <el-radio-group v-model="batchEditForm.hsts">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.http2">HTTP2</el-checkbox>
                <el-radio-group v-model="batchEditForm.http2">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.ocsp_stapling">OCSP Stapling</el-checkbox>
                <el-radio-group v-model="batchEditForm.ocsp_stapling">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.http3">HTTP3</el-checkbox>
                <el-radio-group v-model="batchEditForm.http3">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.ssl_profile">SSL配置</el-checkbox>
                <el-radio-group v-model="batchEditForm.ssl_profile">
                  <el-radio label="compat">兼容旧浏览器</el-radio>
                  <el-radio label="modern">兼容大部分浏览器</el-radio>
                  <el-radio label="custom">自定义</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="源站设置" name="origin">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.origin_list">源站列表</el-checkbox>
                <el-button size="small" type="primary" @click="addOrigin">新增源站信息</el-button>
              </div>
              <el-table v-if="batchEditForm.origin_list.length" :data="batchEditForm.origin_list" border size="small">
                <el-table-column label="源地址">
                  <template #default="{ row }">
                    <el-input v-model="row.address" placeholder="IP或域名" />
                  </template>
                </el-table-column>
                <el-table-column label="权重" width="100">
                  <template #default="{ row }">
                    <el-input v-model="row.weight" placeholder="权重" />
                  </template>
                </el-table-column>
                <el-table-column label="状态" width="120">
                  <template #default="{ row }">
                    <el-switch v-model="row.enable" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeOrigin($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="batch-row" style="margin-top: 16px;">
                <el-checkbox v-model="batchEditChecks.origin_conditions">条件源站</el-checkbox>
                <el-button size="small" type="primary" @click="addConditionOrigin">新增条件源站</el-button>
              </div>
              <el-table v-if="batchEditForm.origin_conditions.length" :data="batchEditForm.origin_conditions" border size="small">
                <el-table-column label="条件" min-width="360">
                  <template #default="{ row }">
                    <div class="condition-origin-row">
                      <el-select v-model="row.item" placeholder="请选择" size="small" style="width: 160px;" @change="handleOriginConditionChange(row)">
                        <el-option v-for="opt in originConditionItems" :key="opt.value" :label="opt.label" :value="opt.value" />
                      </el-select>
                      <el-input
                        v-if="isOriginHeaderItem(row.item)"
                        v-model="row.header"
                        placeholder="请求头名称，如: user-agent"
                        size="small"
                        style="width: 180px;"
                      />
                      <el-input
                        v-else-if="isOriginStatItem(row.item)"
                        v-model="row.seconds"
                        placeholder="统计秒数"
                        size="small"
                        style="width: 120px;"
                      />
                      <el-select v-if="!isOriginStatItem(row.item)" v-model="row.operator" placeholder="请选择" size="small" style="width: 150px;">
                        <el-option v-for="opt in originConditionOperators" :key="opt.value" :label="opt.label" :value="opt.value" />
                      </el-select>
                      <el-select v-else v-model="row.operator" size="small" disabled style="width: 120px;">
                        <el-option label="大于" value="gt" />
                      </el-select>
                      <el-input
                        v-model="row.value"
                        :placeholder="getOriginConditionPlaceholder(row)"
                        type="textarea"
                        :rows="1"
                        size="small"
                        style="min-width: 220px;"
                      />
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="源站" min-width="220">
                  <template #default="{ row }">
                    <el-input v-model="row.origin" placeholder="源站地址，多个用|分隔" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeConditionOrigin($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="batch-row" style="margin-top: 16px;">
                <el-checkbox v-model="batchEditChecks.balance_way">负载方式</el-checkbox>
                <el-radio-group v-model="batchEditForm.balance_way">
                  <el-radio label="rr">轮循</el-radio>
                  <el-radio label="ip_hash">定源</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.origin_health_check">源站健康检查</el-checkbox>
                <el-radio-group v-model="batchEditForm.origin_health_check">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="回源设置" name="backsource">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.backsource_protocol">回源协议</el-checkbox>
                <el-radio-group v-model="batchEditForm.backsource_protocol">
                  <el-radio label="http">HTTP</el-radio>
                  <el-radio label="https">HTTPS</el-radio>
                  <el-radio label="follow">跟随协议</el-radio>
                  <el-radio label="follow_with_port">跟随端口和协议</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.backsource_http_port">HTTP回源端口</el-checkbox>
                <el-input v-model="batchEditForm.backsource_http_port" placeholder="请输入HTTP回源端口" style="width: 70%;" />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.backsource_https_port">HTTPS回源端口</el-checkbox>
                <el-input v-model="batchEditForm.backsource_https_port" placeholder="请输入HTTPS回源端口" style="width: 70%;" />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.backsource_host">回源域名</el-checkbox>
                <el-radio-group v-model="batchEditForm.backsource_host_mode">
                  <el-radio label="domain">访问域名</el-radio>
                  <el-radio label="domain_port">访问域名+访问端口</el-radio>
                  <el-radio label="custom">自定义</el-radio>
                </el-radio-group>
                <el-input
                  v-if="batchEditForm.backsource_host_mode === 'custom'"
                  v-model="batchEditForm.backsource_host_custom"
                  placeholder="输入回源域名"
                  style="width: 50%;"
                />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.backsource_timeout">回源超时</el-checkbox>
                <el-input v-model="batchEditForm.backsource_timeout" placeholder="请输入回源超时" style="width: 70%;">
                  <template #append>秒</template>
                </el-input>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.connect_timeout">连接超时</el-checkbox>
                <el-input v-model="batchEditForm.connect_timeout" placeholder="请输入连接超时" style="width: 70%;">
                  <template #append>秒</template>
                </el-input>
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="缓存设置" name="cache">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.cache_enable">CDN缓存</el-checkbox>
                <el-switch v-model="batchEditForm.cache_enable" />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.cache_preset">缓存规则</el-checkbox>
                <el-select v-model="batchEditForm.cache_preset" placeholder="快速设置缓存" style="width: 70%;">
                  <el-option v-for="opt in cachePresets" :key="opt.value" :label="opt.label" :value="opt.value" />
                </el-select>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.cache_rules">缓存规则列表</el-checkbox>
                <el-button size="small" type="primary" @click="addCacheRule">新增规则</el-button>
              </div>
              <el-table v-if="batchEditForm.cache_rules.length" :data="batchEditForm.cache_rules" border size="small">
                <el-table-column label="类型" width="160">
                  <template #default="{ row }">
                    <el-select v-model="row.type" placeholder="请选择" style="width: 100%;">
                      <el-option v-for="opt in cacheRuleTypes" :key="opt.value" :label="opt.label" :value="opt.value" />
                    </el-select>
                  </template>
                </el-table-column>
                <el-table-column label="内容">
                  <template #default="{ row }">
                    <el-input
                      v-if="cacheRuleNeedsValue(row.type)"
                      v-model="row.value"
                      :placeholder="cacheRulePlaceholder(row.type)"
                    />
                    <span v-else class="help-text">无需填写</span>
                  </template>
                </el-table-column>
                <el-table-column label="有效期" width="140">
                  <template #default="{ row }">
                    <el-input v-model="row.ttl" placeholder="秒" />
                  </template>
                </el-table-column>
                <el-table-column label="忽略参数" width="100">
                  <template #default="{ row }">
                    <el-switch v-model="row.ignore_args" />
                  </template>
                </el-table-column>
                <el-table-column label="强制缓存" width="100">
                  <template #default="{ row }">
                    <el-switch v-model="row.force_cache" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeCacheRule($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="安全设置" name="security">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_default_rule">默认防护</el-checkbox>
                <el-select v-model="batchEditForm.security_default_rule" placeholder="请选择" style="width: 70%;">
                  <el-option v-for="opt in securityRules" :key="opt.value" :label="opt.label" :value="opt.value" />
                </el-select>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_auto_switch">自动切换</el-checkbox>
                <el-radio-group v-model="batchEditForm.security_auto_switch">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_custom_rules">自定义规则</el-checkbox>
                <el-button size="small" type="primary" @click="addSecurityRule">新增规则</el-button>
              </div>
              <el-table v-if="batchEditForm.security_custom_rules.length" :data="batchEditForm.security_custom_rules" border size="small">
                <el-table-column label="匹配条件">
                  <template #default="{ row }">
                    <el-input v-model="row.match" placeholder="匹配条件" />
                  </template>
                </el-table-column>
                <el-table-column label="执行过滤">
                  <template #default="{ row }">
                    <el-input v-model="row.action" placeholder="过滤动作" />
                  </template>
                </el-table-column>
                <el-table-column label="匹配模式">
                  <template #default="{ row }">
                    <el-input v-model="row.mode" placeholder="匹配模式" />
                  </template>
                </el-table-column>
                <el-table-column label="备注">
                  <template #default="{ row }">
                    <el-input v-model="row.remark" placeholder="备注" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeSecurityRule($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <div class="batch-row" style="margin-top: 16px;">
                <el-checkbox v-model="batchEditChecks.security_black_time">黑名单时间</el-checkbox>
                <el-radio-group v-model="batchEditForm.security_black_time_mode">
                  <el-radio label="system">系统默认</el-radio>
                  <el-radio label="custom">自定义</el-radio>
                </el-radio-group>
                <el-input
                  v-if="batchEditForm.security_black_time_mode === 'custom'"
                  v-model="batchEditForm.security_black_time_custom"
                  placeholder="输入黑名单时间"
                  style="width: 50%;"
                />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_white_time">白名单时间</el-checkbox>
                <el-radio-group v-model="batchEditForm.security_white_time_mode">
                  <el-radio label="system">系统默认</el-radio>
                  <el-radio label="custom">自定义</el-radio>
                </el-radio-group>
                <el-input
                  v-if="batchEditForm.security_white_time_mode === 'custom'"
                  v-model="batchEditForm.security_white_time_custom"
                  placeholder="输入白名单时间"
                  style="width: 50%;"
                />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_bot">搜索引擎爬虫</el-checkbox>
                <el-radio-group v-model="batchEditForm.security_bot">
                  <el-radio label="none">不设置</el-radio>
                  <el-radio label="allow">放行</el-radio>
                  <el-radio label="deny">拦截</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_blacklist">黑名单</el-checkbox>
                <el-input v-model="batchEditForm.security_blacklist" type="textarea" rows="3" placeholder="一行一个" style="width: 70%;" />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_whitelist">白名单</el-checkbox>
                <el-input v-model="batchEditForm.security_whitelist" type="textarea" rows="3" placeholder="一行一个" style="width: 70%;" />
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_shield_proxy">屏蔽透明代理</el-checkbox>
                <el-radio-group v-model="batchEditForm.security_shield_proxy">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.security_region_block">区域屏蔽</el-checkbox>
                <el-radio-group v-model="batchEditForm.security_region_mode">
                  <el-radio label="none">不设置</el-radio>
                  <el-radio label="overseas_without_hk">国外(不包括港澳台)</el-radio>
                  <el-radio label="overseas_with_hk">国外(包括港澳台)</el-radio>
                  <el-radio label="china_with_hk">中国(包括港澳台)</el-radio>
                  <el-radio label="china_without_hk">中国(不包括港澳台)</el-radio>
                  <el-radio label="custom">自定义</el-radio>
                </el-radio-group>
              </div>
              <country-selector
                v-if="batchEditForm.security_region_mode === 'custom'"
                v-model="batchEditForm.security_region_custom"
              />
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="访问控制" name="access">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.access_acl">ACL设置</el-checkbox>
                <el-select v-model="batchEditForm.access_acl" placeholder="请选择" style="width: 70%;">
                  <el-option label="不设置" value="none" />
                  <el-option label="仅白名单" value="whitelist" />
                  <el-option label="仅黑名单" value="blacklist" />
                </el-select>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.access_hotlink">防盗链</el-checkbox>
                <el-radio-group v-model="batchEditForm.access_hotlink">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.access_cors">跨域访问</el-checkbox>
                <el-radio-group v-model="batchEditForm.access_cors">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>

            <el-collapse-item title="高级设置" name="advanced">
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_ipv6">IPv6开启</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_ipv6">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_gzip">Gzip压缩</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_gzip">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_websocket">Websocket</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_websocket">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_search_origin">搜索引擎回源</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_search_origin">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>

              <div class="batch-row" style="margin-top: 10px;">
                <el-checkbox v-model="batchEditChecks.adv_error_pages">自定义错误页</el-checkbox>
                <el-button size="small" type="primary" @click="addErrorPage">新增页面</el-button>
              </div>
              <el-table v-if="batchEditForm.adv_error_pages.length" :data="batchEditForm.adv_error_pages" border size="small">
                <el-table-column label="状态码" width="120">
                  <template #default="{ row }">
                    <el-input v-model="row.code" placeholder="如 404" />
                  </template>
                </el-table-column>
                <el-table-column label="内容/地址">
                  <template #default="{ row }">
                    <el-input v-model="row.content" placeholder="HTML或URL" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeErrorPage($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="batch-row" style="margin-top: 16px;">
                <el-checkbox v-model="batchEditChecks.adv_url_redirect">URL转向</el-checkbox>
                <el-button size="small" type="primary" @click="addUrlRedirect">新增转向</el-button>
              </div>
              <el-table v-if="batchEditForm.adv_url_redirects.length" :data="batchEditForm.adv_url_redirects" border size="small">
                <el-table-column label="域名端口">
                  <template #default="{ row }">
                    <el-input v-model="row.host" placeholder="域名或端口" />
                  </template>
                </el-table-column>
                <el-table-column label="匹配">
                  <template #default="{ row }">
                    <el-input v-model="row.match" placeholder="匹配规则" />
                  </template>
                </el-table-column>
                <el-table-column label="转向到">
                  <template #default="{ row }">
                    <el-input v-model="row.target" placeholder="目标地址" />
                  </template>
                </el-table-column>
                <el-table-column label="响应码" width="120">
                  <template #default="{ row }">
                    <el-input v-model="row.code" placeholder="301/302" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeUrlRedirect($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="batch-row" style="margin-top: 16px;">
                <el-checkbox v-model="batchEditChecks.adv_origin_headers">源站请求头</el-checkbox>
                <el-button size="small" type="primary" @click="addOriginHeader">新增请求头</el-button>
              </div>
              <el-table v-if="batchEditForm.adv_origin_headers.length" :data="batchEditForm.adv_origin_headers" border size="small">
                <el-table-column label="名称">
                  <template #default="{ row }">
                    <el-input v-model="row.name" placeholder="Header 名称" />
                  </template>
                </el-table-column>
                <el-table-column label="值">
                  <template #default="{ row }">
                    <el-input v-model="row.value" placeholder="Header 值" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeOriginHeader($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="batch-row" style="margin-top: 16px;">
                <el-checkbox v-model="batchEditChecks.adv_cdn_headers">CDN响应头</el-checkbox>
                <el-button size="small" type="primary" @click="addCdnHeader">新增响应头</el-button>
              </div>
              <el-table v-if="batchEditForm.adv_cdn_headers.length" :data="batchEditForm.adv_cdn_headers" border size="small">
                <el-table-column label="名称">
                  <template #default="{ row }">
                    <el-input v-model="row.name" placeholder="Header 名称" />
                  </template>
                </el-table-column>
                <el-table-column label="值">
                  <template #default="{ row }">
                    <el-input v-model="row.value" placeholder="Header 值" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="removeCdnHeader($index)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="batch-row" style="margin-top: 16px;">
                <el-checkbox v-model="batchEditChecks.adv_acme_backsource">acme请求回源</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_acme_backsource">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_realtime_return">数据实时返回</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_realtime_return">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_realtime_send">数据实时发送</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_realtime_send">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_log_request_header">记录请求头</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_log_request_header">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_log_response_header">记录响应头</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_log_response_header">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_log_request_body">记录请求体</el-checkbox>
                <el-radio-group v-model="batchEditForm.adv_log_request_body">
                  <el-radio :label="true">开启</el-radio>
                  <el-radio :label="false">关闭</el-radio>
                </el-radio-group>
              </div>
              <div class="batch-row">
                <el-checkbox v-model="batchEditChecks.adv_body_limit">请求体大小限制</el-checkbox>
                <el-input v-model="batchEditForm.adv_body_limit" placeholder="请输入请求体大小限制" style="width: 70%;">
                  <template #append>KB</template>
                </el-input>
              </div>
              <div class="batch-action">
                <el-button type="primary" @click="submitBatchEdit">批量修改</el-button>
              </div>
            </el-collapse-item>
          </el-collapse>
        </el-form>
      </div>
    </el-dialog>

    <el-dialog v-model="advancedVisible" title="高级搜索" width="520px">
      <el-form :model="advancedForm" label-width="90px">
        <el-form-item label="分组">
          <el-select v-model="advancedForm.group_id" multiple clearable placeholder="请选择" style="width: 100%;">
            <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="advancedForm.status" clearable placeholder="请选择" style="width: 100%;">
            <el-option label="正常" value="enabled" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="HTTPS">
          <el-select v-model="advancedForm.https" clearable placeholder="请选择" style="width: 100%;">
            <el-option label="开启" value="1" />
            <el-option label="关闭" value="0" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="advancedVisible = false">取消</el-button>
        <el-button type="primary" @click="applyAdvancedFilter">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, CircleCheckFilled, CircleCloseFilled, Plus } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { useRouter } from 'vue-router'
import CountrySelector from '@/components/CountrySelector.vue'
import ResolvePage from './Resolve.vue'

const router = useRouter()
const isAdmin = ref((localStorage.getItem('role') || 'user') === 'admin')
const activeTopTab = ref('list')
const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const selectedRows = ref([])

const listQuery = reactive({
  page: 1,
  pageSize: 10,
  keyword: '',
  searchField: 'all'
})

const advancedVisible = ref(false)
const advancedForm = reactive({
  group_id: [],
  status: '',
  https: ''
})

const createVisible = ref(false)
const createTab = ref('single')
const createMore = ref(false)
const batchMore = ref(false)
const createForm = reactive({
  user_id: undefined,
  user_package_id: undefined,
  group_id: undefined,
  dns_provider_id: undefined,
  site_type: 'website',
  domains_input: '',
  backends_input: ''
})
const batchForm = reactive({
  user_id: undefined,
  user_package_id: undefined,
  group_id: undefined,
  dns_provider_id: undefined,
  data: '',
  ignore_error: false
})

const dnsapiList = ref([])
const dnsapiLoading = ref(false)
const selectedDnsapi = ref([])
const dnsapiTypes = ref([])
const dnsapiDialogVisible = ref(false)
const dnsapiForm = reactive({
  id: 0,
  name: '',
  remark: '',
  type: '',
  credentials: {}
})

const defaultList = ref([])
const defaultLoading = ref(false)
const selectedDefaults = ref([])
const defaultDialogVisible = ref(false)
const defaultDialogMode = ref('create')
const defaultEditName = ref('')
const defaultEditScopeName = ref('')
const defaultEditScopeID = ref(0)
const defaultForm = reactive({
  user_id: undefined,
  name: '',
  value: 0,
  boolValue: false,
  selectValue: '',
  multiValue: [],
  textValue: '',
  headers: [],
  region_mode: 'none',
  region_custom: [],
  scope: 'global',
  group_id: 0
})

const batchEditVisible = ref(false)
const batchCollapse = ref(['basic'])
const batchEditForm = reactive({
  group_id: undefined,
  dns_provider_id: undefined,
  http_enable: true,
  http_listen: '',
  https_enable: true,
  https_listen: '',
  force_https: false,
  https_redirect_port: '',
  hsts: false,
  http2: true,
  ocsp_stapling: false,
  http3: false,
  ssl_profile: 'compat',
  origin_list: [],
  origin_conditions: [],
  balance_way: 'rr',
  origin_health_check: true,
  backsource_protocol: 'http',
  backsource_http_port: '',
  backsource_https_port: '',
  backsource_host_mode: 'domain',
  backsource_host_custom: '',
  backsource_timeout: '',
  connect_timeout: '',
  cache_enable: true,
  cache_preset: '',
  cache_rules: [],
  security_default_rule: 0,
  security_auto_switch: false,
  security_custom_rules: [],
  security_black_time_mode: 'system',
  security_black_time_custom: '',
  security_white_time_mode: 'system',
  security_white_time_custom: '',
  security_bot: 'none',
  security_blacklist: '',
  security_whitelist: '',
  security_shield_proxy: false,
  security_region_mode: 'none',
  security_region_custom: [],
  access_acl: '',
  access_hotlink: false,
  access_cors: false,
  adv_ipv6: false,
  adv_gzip: false,
  adv_websocket: false,
  adv_search_origin: false,
  adv_error_pages: [],
  adv_url_redirects: [],
  adv_origin_headers: [],
  adv_cdn_headers: [],
  adv_acme_backsource: false,
  adv_realtime_return: false,
  adv_realtime_send: false,
  adv_log_request_header: false,
  adv_log_response_header: false,
  adv_log_request_body: false,
  adv_body_limit: ''
})
const batchEditChecks = reactive({
  group_id: false,
  dns_provider_id: false,
  http_enable: false,
  http_listen: false,
  https_enable: false,
  https_listen: false,
  force_https: false,
  https_redirect_port: false,
  hsts: false,
  http2: false,
  ocsp_stapling: false,
  http3: false,
  ssl_profile: false,
  origin_list: false,
  origin_conditions: false,
  balance_way: false,
  origin_health_check: false,
  backsource_protocol: false,
  backsource_http_port: false,
  backsource_https_port: false,
  backsource_host: false,
  backsource_timeout: false,
  connect_timeout: false,
  cache_enable: false,
  cache_preset: false,
  cache_rules: false,
  security_default_rule: false,
  security_auto_switch: false,
  security_custom_rules: false,
  security_black_time: false,
  security_white_time: false,
  security_bot: false,
  security_blacklist: false,
  security_whitelist: false,
  security_shield_proxy: false,
  security_region_block: false,
  access_acl: false,
  access_hotlink: false,
  access_cors: false,
  adv_ipv6: false,
  adv_gzip: false,
  adv_websocket: false,
  adv_search_origin: false,
  adv_error_pages: false,
  adv_url_redirect: false,
  adv_origin_headers: false,
  adv_cdn_headers: false,
  adv_acme_backsource: false,
  adv_realtime_return: false,
  adv_realtime_send: false,
  adv_log_request_header: false,
  adv_log_response_header: false,
  adv_log_request_body: false,
  adv_body_limit: false
})

const groupOptions = ref([])
const dnsOptions = ref([])
const userOptions = ref([])
const packageOptions = ref([])
const userLoading = ref(false)

  const loadUsers = (query) => {
    if (query !== '') {
      userLoading.value = true
      request.get('/users', { params: { keyword: query, pageSize: 20 } }).then(res => {
        userOptions.value = res.data?.list || res.list || []
      userLoading.value = false
    }).catch(() => {
      userLoading.value = false
    })
    } else {
      userOptions.value = []
    }
  }

  const ensureUserOption = (userId, userName) => {
    if (!userId) return
    const exists = userOptions.value.some(item => item.id === userId)
    if (!exists) {
      userOptions.value = [{ id: userId, name: userName || String(userId) }, ...userOptions.value]
    }
  }

const loadPackages = (userId) => {
  const params = {}
  if (userId) params.user_id = userId
  request.get('/user_packages', { params }).then(res => {
    packageOptions.value = res.data?.list || res.list || []
    const first = packageOptions.value[0]
    if (first && !createForm.user_package_id) {
      createForm.user_package_id = first.id
    }
    if (first && !batchForm.user_package_id) {
      batchForm.user_package_id = first.id
    }
  })
}

const cachePresets = [
  { label: '首页缓存', value: 'home' },
  { label: '全站缓存', value: 'all' },
  { label: '静态资源缓存', value: 'static' },
  { label: '视频文件缓存', value: 'video' },
  { label: 'Wordpress缓存', value: 'wordpress' }
]

const cacheRuleTypes = [
  { label: '首页', value: 'home' },
  { label: '全站', value: 'all' },
  { label: '目录', value: 'dir' },
  { label: '后缀', value: 'suffix' },
  { label: '单个路径', value: 'path' }
]

const cacheRuleNeedsValue = (type) => ['dir', 'suffix', 'path'].includes(type)

const cacheRulePlaceholder = (type) => {
  switch (type) {
    case 'dir':
      return '例如: /images/'
    case 'suffix':
      return '例如: .jpg .png'
    case 'path':
      return '例如: /index.html'
    default:
      return ''
  }
}

const originConditionItems = [
  { label: '请求URI', value: 'uri' },
  { label: '请求URI(不带参数)', value: 'uri_no_args' },
  { label: '节点国家代码', value: 'node_country' },
  { label: '节点运营商', value: 'node_isp' },
  { label: '节点省份', value: 'node_province' },
  { label: '节点城市', value: 'node_city' },
  { label: '客户端国家代码', value: 'client_country' },
  { label: '客户端运营商', value: 'client_isp' },
  { label: '客户端省份', value: 'client_province' },
  { label: '客户端城市', value: 'client_city' },
  { label: '用户IP', value: 'client_ip' },
  { label: '域名', value: 'domain' },
  { label: '请求头', value: 'header' },
  { label: '请求方法', value: 'method' },
  { label: 'HTTP版本', value: 'http_version' },
  { label: '独立UA数量', value: 'ua_count' },
  { label: '404状态码数量', value: 'status_404' }
]

const originConditionOperators = [
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

const isOriginHeaderItem = item => item === 'header'
const isOriginStatItem = item => item === 'ua_count' || item === 'status_404'

const getOriginConditionPlaceholder = row => {
  switch (row.item) {
    case 'http_version':
      return '输入匹配值,一行一个,如:\nHTTP/1.0\nHTTP/1.1\nHTTP/2.0'
    case 'method':
      return '输入匹配值,一行一个,如:\nGET\nPOST'
    case 'client_ip':
      return '输入匹配值,一行一个,如:\n1.1.1.1\n2.2.2.2'
    case 'domain':
      return '输入匹配值,一行一个,如:\nexample.com\nwww.example.com'
    case 'uri':
    case 'uri_no_args':
      return '输入匹配值,一行一个,如:\n/index.html\n/api/v1/'
    case 'node_country':
    case 'client_country':
      return '输入匹配值,一行一个,如:\nCN\nUS'
    case 'node_isp':
    case 'client_isp':
      return '输入匹配值,一行一个,如:\n电信\n联通'
    case 'node_province':
    case 'client_province':
      return '输入匹配值,一行一个,如:\n广东\n浙江'
    case 'node_city':
    case 'client_city':
      return '输入匹配值,一行一个,如:\n深圳\n杭州'
    case 'ua_count':
    case 'status_404':
      return '输入次数'
    case 'header':
      return '输入匹配值,一行一个'
    default:
      return '输入匹配值,一行一个'
  }
}

const handleOriginConditionChange = (row) => {
  if (isOriginStatItem(row.item)) {
    row.operator = 'gt'
    row.seconds = row.seconds || '10'
  } else if (!row.operator) {
    row.operator = 'eq'
  }
  if (isOriginHeaderItem(row.item) && !row.header) {
    row.header = ''
  }
}

const securityRules = [
  { label: '不设置', value: 0 },
  { label: '基础防护', value: 1 },
  { label: '增强防护', value: 2 }
]

const activeTags = computed(() => {
  const tags = []
  if (listQuery.keyword) {
    tags.push({ key: 'keyword', label: `${labelForSearchField(listQuery.searchField)}: ${listQuery.keyword}` })
  }
  if (advancedForm.group_id && advancedForm.group_id.length) {
    const groupNames = advancedForm.group_id
      .map(id => groupOptions.value.find(g => g.id === id)?.name)
      .filter(Boolean)
      .join(', ')
    tags.push({ key: 'group_id', label: `分组: ${groupNames}` })
  }
  if (advancedForm.status) tags.push({ key: 'status', label: `状态: ${advancedForm.status === 'enabled' ? '正常' : '停用'}` })
  if (advancedForm.https !== '') tags.push({ key: 'https', label: `HTTPS: ${advancedForm.https === '1' ? '开启' : '关闭'}` })
  return tags
})

const selectedIdsText = computed(() => selectedRows.value.map(row => row.id).join(','))

const currentDnsapiType = computed(() => dnsapiTypes.value.find(t => t.type === dnsapiForm.type))

const defaultDialogTitle = computed(() => (defaultDialogMode.value === 'edit' ? '编辑设置' : '新增设置'))
const defaultOption = computed(() => defaultOptions.find(opt => opt.value === defaultForm.name))
const defaultOptionType = computed(() => defaultOption.value?.type || 'number')
const ccRuleOptions = ref([{ label: '不设置', value: '0' }])
const getDefaultOptionChoices = (option) => {
  if (!option) return []
  if (option.choicesKey === 'cc_rules') return ccRuleOptions.value
  if (option.choicesKey === 'dns_providers') {
    return dnsOptions.value.map(item => ({ label: item.name, value: String(item.id) }))
  }
  return option.choices || []
}
const defaultOptionChoices = computed(() => getDefaultOptionChoices(defaultOption.value))

const defaultOptions = [
  { label: '默认CC规则', value: 'cc_default_rule', type: 'select', choicesKey: 'cc_rules' },
  { label: '黑名单时间', value: 'security_black_time', type: 'number' },
  { label: '白名单时间', value: 'security_white_time', type: 'number' },
  { label: '搜索引擎爬虫', value: 'security_bot', type: 'select', choices: [
    { label: '不设置', value: 'none' },
    { label: '放行', value: 'allow' },
    { label: '拦截', value: 'deny' }
  ] },
  { label: '黑名单IP', value: 'black_ip', type: 'lines' },
  { label: '白名单IP', value: 'white_ip', type: 'lines' },
  { label: '屏蔽透明代理', value: 'security_shield_proxy', type: 'bool' },
  { label: '区域屏蔽', value: 'block_region', type: 'region' },
  { label: 'DNS API(解析)', value: 'dns_provider_id', type: 'select', choicesKey: 'dns_providers' },
  { label: 'HTTP监听端口', value: 'http_listen-port', type: 'number' },
  { label: 'HTTPS监听端口', value: 'https_listen-port', type: 'number' },
  { label: '强制HTTPS', value: 'https_listen-force_ssl_enable', type: 'bool' },
  { label: '开启HSTS', value: 'https_listen-hsts', type: 'bool' },
  { label: '开启HTTP2', value: 'https_listen-http2', type: 'bool' },
  { label: '开启HTTP3', value: 'https_listen-http3', type: 'bool' },
  { label: 'ssl_protocols', value: 'https_listen-ssl_protocols', type: 'multi', choices: [
    { label: 'SSLv2', value: 'SSLv2' },
    { label: 'SSLv3', value: 'SSLv3' },
    { label: 'TLSv1', value: 'TLSv1' },
    { label: 'TLSv1.1', value: 'TLSv1.1' },
    { label: 'TLSv1.2', value: 'TLSv1.2' },
    { label: 'TLSv1.3', value: 'TLSv1.3' }
  ] },
  { label: 'ssl_ciphers', value: 'https_listen-ssl_ciphers', type: 'text' },
  { label: 'ssl_prefer_server_ciphers', value: 'https_listen-ssl_prefer_server_ciphers', type: 'select', choices: [
    { label: 'On', value: 'on' },
    { label: 'Off', value: 'off' }
  ] },
  { label: 'ocsp_stapling', value: 'https_listen-ocsp_stapling', type: 'bool' },
  { label: '回源协议', value: 'backend_protocol', type: 'select', choices: [
    { label: 'HTTP', value: 'http' },
    { label: 'HTTPS', value: 'https' },
    { label: '跟随协议', value: 'follow' }
  ] },
  { label: '回源HTTP端口', value: 'backend_http_port', type: 'number' },
  { label: '回源HTTPS端口', value: 'backend_https_port', type: 'number' },
  { label: '回源超时', value: 'proxy_timeout', type: 'number' },
  { label: '开启IPv6', value: 'ipv6_enable', type: 'bool' },
  { label: '开启Gzip', value: 'gzip_enable', type: 'bool' },
  { label: '开启Websocket', value: 'websocket_enable', type: 'bool' },
  { label: '上传文件大小限制', value: 'post_size_limit', type: 'number' },
  { label: '数据实时发送', value: 'realtime_send', type: 'bool' },
  { label: '数据实时返回', value: 'realtime_return', type: 'bool' },
  { label: '源站请求头', value: 'origin_headers', type: 'headers' },
  { label: '回源负载方式', value: 'balance_way', type: 'select', choices: [
    { label: '轮询', value: 'rr' },
    { label: '定源', value: 'hash' }
  ] }
]

const defaultRows = computed(() => defaultList.value.map((item, index) => {
  const option = defaultOptions.find(opt => opt.value === item.name)
  const label = option ? option.label : item.name
  let scopeLabel = item.scope_name === 'group' ? '分组' : '全局'
  if (item.scope_name === 'group' && item.group_name) {
    scopeLabel = `分组(${item.group_name})`
  }
  return {
    id: index + 1,
    name: item.name,
    label,
    value: formatDefaultValue(item.name, item.value),
    rawValue: item.value,
    enable: item.enable,
    scopeLabel,
      user_id: item.user_id,
      user_name: item.user_name || '',
      scope_name: item.scope_name,
      scope_id: item.scope_id,
      group_name: item.group_name || ''
  }
}))

const dnsapiFieldLabel = (type, field) => {
  const mapping = {
    aliyun: { id: 'AccessKey ID', secret: 'AccessKey Secret' },
    huawei: { id: 'Access Key ID', secret: 'Secret Access Key' },
    dnsla: { id: 'API ID', secret: 'API Key' },
    dnspod: { id: 'ID', token: 'Token' },
    dnspod_intl: { id: 'ID', token: 'Token' },
    cloudflare: { email: 'Email', key: 'API Key' },
    godaddy: { key: 'Key', secret: 'Secret' }
  }
  if (mapping[type] && mapping[type][field]) {
    return mapping[type][field]
  }
  return field.replace(/_/g, ' ').toUpperCase()
}

const formatDnsType = type => {
  const t = dnsapiTypes.value.find(item => item.type === type)
  return t ? t.name : type
}

const confirmRemove = (message, onConfirm) => {
  ElMessageBox.confirm(message, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    onConfirm()
  })
}

const loadDnsapiList = () => {
  dnsapiLoading.value = true
  request.get('/dnsapi').then(res => {
    dnsapiList.value = res.data?.list || res.list || []
    dnsapiLoading.value = false
  }).catch(() => {
    dnsapiLoading.value = false
  })
}

const loadDnsapiTypes = () => {
  request.get('/dnsapi/types').then(res => {
    dnsapiTypes.value = res.data?.types || res.list || []
  })
}

const loadCcRuleOptions = () => {
  const params = { page: 1, pageSize: 200, name: '', status: '' }
  request.get('/rules/cc/groups', { params }).then(res => {
    const list = res.data?.list || res.list || []
    ccRuleOptions.value = [{ label: '不设置', value: '0' }]
      .concat(list.map(item => ({
        label: item.name || `规则${item.id}`,
        value: String(item.id)
      })))
  })
}

const resetDnsapiAuth = () => {
  dnsapiForm.credentials = {}
}

const openDnsapiDialog = () => {
  dnsapiForm.id = 0
  dnsapiForm.name = ''
  dnsapiForm.remark = ''
  dnsapiForm.type = ''
  dnsapiForm.credentials = {}
  dnsapiDialogVisible.value = true
}

const openDnsapiEdit = row => {
  dnsapiForm.id = row.id
  dnsapiForm.name = row.name
  dnsapiForm.remark = row.remark || ''
  dnsapiForm.type = row.type
  dnsapiForm.credentials = row.auth ? JSON.parse(row.auth) : {}
  dnsapiDialogVisible.value = true
}

const submitDnsapi = () => {
  if (!dnsapiForm.name || !dnsapiForm.type) {
    ElMessage.error('请填写名称和类型')
    return
  }
  const payload = {
    name: dnsapiForm.name,
    remark: dnsapiForm.remark,
    type: dnsapiForm.type,
    auth: JSON.stringify(dnsapiForm.credentials || {}),
  }
  if (dnsapiForm.id) {
    request.put(`/dnsapi/${dnsapiForm.id}`, payload).then(() => {
      dnsapiDialogVisible.value = false
      loadDnsapiList()
    })
    return
  }
  request.post('/dnsapi', payload).then(() => {
    dnsapiDialogVisible.value = false
    loadDnsapiList()
  })
}

const removeDnsapi = row => {
  confirmRemove('确认删除该DNS API?', () => {
    request.delete(`/dnsapi/${row.id}`).then(() => {
      loadDnsapiList()
    })
  })
}

const handleDnsapiSelection = rows => {
  selectedDnsapi.value = rows
}

const removeDnsapiBatch = () => {
  if (!selectedDnsapi.value.length) return
  const ids = selectedDnsapi.value.map(row => row.id)
  confirmRemove('确认删除选中的DNS API?', () => {
    Promise.all(ids.map(id => request.delete(`/dnsapi/${id}`))).then(() => {
      loadDnsapiList()
    })
  })
}

const loadSiteDefaults = (userId) => {
  defaultLoading.value = true
  const params = {}
  if (userId) params.user_id = userId
  request.get('/site_defaults', { params }).then(res => {
    defaultList.value = res.data?.list || res.list || []
  }).finally(() => {
    defaultLoading.value = false
  })
}

const openDefaultDialog = (row) => {
  const hasRow = row && typeof row === 'object' && Object.prototype.hasOwnProperty.call(row, 'name')
  if (hasRow) {
    defaultDialogMode.value = 'edit'
    defaultEditName.value = row.name
    defaultEditScopeName.value = row.scope_name || 'global'
    defaultEditScopeID.value = Number(row.scope_id || 0)
    defaultForm.user_id = isAdmin.value ? row.user_id : undefined
    if (isAdmin.value) {
      ensureUserOption(row.user_id, row.user_name)
    }
    defaultForm.name = row.name
    hydrateDefaultForm(row.name, row.rawValue)
    defaultForm.scope = row.scope_name === 'group' ? 'group' : 'global'
    defaultForm.group_id = row.scope_name === 'group' ? Number(row.scope_id || 0) : 0
  } else {
    defaultDialogMode.value = 'create'
    defaultEditName.value = ''
    defaultEditScopeName.value = ''
    defaultEditScopeID.value = 0
    defaultForm.user_id = isAdmin.value ? undefined : undefined
    resetDefaultForm()
  }
  defaultDialogVisible.value = true
}

const submitDefault = () => {
  if (!defaultForm.name) {
    ElMessage.warning('请输入设置项')
    return
  }
  if (isAdmin.value && !defaultForm.user_id) {
    ElMessage.warning('请选择用户')
    return
  }
  if (defaultForm.scope === 'group' && !defaultForm.group_id) {
    ElMessage.warning('请选择网站分组')
    return
  }
  const finalValue = buildDefaultValue()
  const payload = {
    name: defaultForm.name,
    value: finalValue,
    scope_name: defaultForm.scope
  }
  if (defaultForm.scope === 'group') {
    payload.scope_id = defaultForm.group_id
  } else {
    payload.scope_id = defaultForm.user_id
  }
  if (defaultDialogMode.value === 'edit') {
    payload.old_scope_name = defaultEditScopeName.value
    payload.old_scope_id = defaultEditScopeID.value
  }
  if (isAdmin.value) payload.user_id = defaultForm.user_id
  const targetName = defaultDialogMode.value === 'edit'
    ? (defaultEditName.value || defaultForm.name)
    : defaultForm.name
  const requestCall = defaultDialogMode.value === 'edit'
    ? request.put(`/site_defaults/${encodeURIComponent(targetName)}`, payload)
    : request.post('/site_defaults', payload)
  requestCall.then(() => {
    defaultDialogVisible.value = false
    loadSiteDefaults()
  })
}

const removeDefault = (row) => {
  confirmRemove('确认删除该默认设置?', () => {
      const params = { scope_name: row.scope_name, scope_id: row.scope_id }
      if (isAdmin.value && row.user_id) params.user_id = row.user_id
      request.delete(`/site_defaults/${encodeURIComponent(row.name)}`, { params }).then(() => {
        loadSiteDefaults()
      })
    })
  }

const handleDefaultSelection = rows => {
  selectedDefaults.value = rows
}

const removeDefaultBatch = () => {
  if (!selectedDefaults.value.length) return
  confirmRemove('确认删除选中的默认设置?', () => {
      const tasks = selectedDefaults.value.map(row => {
        const params = { scope_name: row.scope_name, scope_id: row.scope_id }
        if (isAdmin.value && row.user_id) params.user_id = row.user_id
        return request.delete(`/site_defaults/${encodeURIComponent(row.name)}`, { params })
      })
      Promise.all(tasks).then(() => {
        selectedDefaults.value = []
        loadSiteDefaults()
      })
    })
  }

const resetDefaultForm = () => {
  defaultForm.name = ''
  defaultForm.value = 0
  defaultForm.boolValue = false
  defaultForm.selectValue = ''
  defaultForm.multiValue = []
  defaultForm.textValue = ''
  defaultForm.headers = []
  defaultForm.region_mode = 'none'
  defaultForm.region_custom = []
  defaultForm.scope = 'global'
  defaultForm.group_id = 0
}

const hydrateDefaultForm = (name, value) => {
  resetDefaultForm()
  defaultForm.name = name
  const option = defaultOptions.find(opt => opt.value === name)
  const type = option?.type || 'number'
  if (type === 'number') {
    defaultForm.value = Number(value) || 0
  } else if (type === 'bool') {
    defaultForm.boolValue = value === '1' || value === 'true' || value === 'on'
  } else if (type === 'select') {
    defaultForm.selectValue = value !== undefined && value !== null ? String(value) : ''
  } else if (type === 'multi') {
    defaultForm.multiValue = (value || '').split(/\s+/).filter(Boolean)
  } else if (type === 'lines') {
    defaultForm.textValue = value || ''
  } else if (type === 'headers') {
    try {
      defaultForm.headers = JSON.parse(value || '[]')
    } catch (e) {
      defaultForm.headers = []
    }
  } else if (type === 'region') {
    const raw = String(value || '')
    if (!raw || raw === 'none') {
      defaultForm.region_mode = 'none'
      defaultForm.region_custom = []
    } else {
      defaultForm.region_mode = 'custom'
      defaultForm.region_custom = raw.split(',').map(item => item.trim()).filter(Boolean)
    }
  } else {
    defaultForm.textValue = value || ''
  }
}

const buildDefaultValue = () => {
  const option = defaultOptions.find(opt => opt.value === defaultForm.name)
  const type = option?.type || 'number'
  if (type === 'number') {
    return String(defaultForm.value ?? 0)
  }
  if (type === 'bool') {
    return defaultForm.boolValue ? '1' : '0'
  }
  if (type === 'select') {
    return String(defaultForm.selectValue || '')
  }
  if (type === 'multi') {
    return (defaultForm.multiValue || []).join(' ')
  }
  if (type === 'lines') {
    return String(defaultForm.textValue || '')
  }
  if (type === 'headers') {
    return JSON.stringify((defaultForm.headers || []).filter(item => item.name))
  }
  if (type === 'region') {
    return defaultForm.region_mode === 'custom'
      ? (defaultForm.region_custom || []).join(',')
      : 'none'
  }
  return String(defaultForm.textValue || '')
}

const formatDefaultValue = (name, value) => {
  const option = defaultOptions.find(opt => opt.value === name)
  const type = option?.type || 'number'
  if (type === 'bool') {
    return value === '1' || value === 'true' || value === 'on' ? '是' : '否'
  }
  if (type === 'select') {
    const val = value !== undefined && value !== null ? String(value) : ''
    const choices = getDefaultOptionChoices(option)
    const match = choices.find(opt => String(opt.value) === val)
    return match ? match.label : val
  }
  if (type === 'multi') {
    return value || ''
  }
  if (type === 'lines') {
    return value || ''
  }
  if (type === 'headers') {
    try {
      const items = JSON.parse(value || '[]')
      return items.map(item => `${item.name}: ${item.value}`).join('; ')
    } catch (e) {
      return value
    }
  }
  if (type === 'region') {
    const raw = String(value || '')
    if (!raw || raw === 'none') return '不设置'
    return raw
  }
  return value
}

const addDefaultHeader = () => {
  defaultForm.headers.push({ name: '', value: '' })
}

const removeDefaultHeader = (index) => {
  defaultForm.headers.splice(index, 1)
}

const handleTopTab = tab => {
  if (tab.paneName === 'default') {
    loadSiteDefaults()
  } else if (tab.paneName === 'dns') {
    loadDnsapiList()
    loadDnsapiTypes()
  }
}

const formatUserLabel = (user) => {
  if (!user) return ''
  const name = user.name || user.email || user.phone || '用户'
  return `${name} (id: ${user.id})`
}

const fetchList = () => {
  listLoading.value = true
  request.get('/sites', {
    params: {
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      keyword: listQuery.keyword,
      search_field: listQuery.searchField,
      group_id: advancedForm.group_id?.length ? advancedForm.group_id.join(',') : undefined,
      status: advancedForm.status || undefined,
      https: advancedForm.https !== '' ? advancedForm.https : undefined
    }
  }).then(res => {
    list.value = res.list || res.data || []
    total.value = res.total || 0
    listLoading.value = false
  }).catch(() => {
    listLoading.value = false
  })
}

const handleFilter = () => {
  listQuery.page = 1
  fetchList()
}

const handleSelectionChange = rows => {
  selectedRows.value = rows
}

const openCreateDialog = () => {
  createVisible.value = true
  createTab.value = 'single'
}

const openBatchEdit = () => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择网站')
    return
  }
  batchEditVisible.value = true
}

const handleCreateSubmit = () => {
  if (createTab.value === 'single') {
    const payload = {
      user_id: createForm.user_id || undefined,
      user_package_id: createForm.user_package_id || undefined,
      group_id: createForm.group_id || undefined,
      dns_provider_id: createForm.dns_provider_id || undefined,
      site_type: createForm.site_type,
      domains_input: createForm.domains_input,
      backends_input: createForm.backends_input
    }
    console.log('[DEBUG] Create Site Payload:', payload)
    request.post('/sites', payload).then(() => {
      ElMessage.success('创建成功')
      createVisible.value = false
      fetchList()
    }).catch(err => {
      console.error('[DEBUG] Create Error:', err)
      const msg = err.response?.data?.error || err.message || '请求失败'
      ElMessageBox.alert(msg, '错误提示', { type: 'error' })
    })
  } else {
    request.post('/sites/batch', batchForm).then(res => {
      ElMessage.success(res.message || '批量创建完成')
      createVisible.value = false
      fetchList()
    })
  }
}

const buildSettingsPayload = () => {
  const settings = {}
  if (batchEditChecks.force_https || batchEditChecks.https_redirect_port || batchEditChecks.hsts || batchEditChecks.http2 ||
      batchEditChecks.ocsp_stapling || batchEditChecks.http3 || batchEditChecks.ssl_profile) {
    settings.https = {
      force: batchEditForm.force_https,
      redirect_port: batchEditForm.https_redirect_port,
      hsts: batchEditForm.hsts,
      http2: batchEditForm.http2,
      ocsp_stapling: batchEditForm.ocsp_stapling,
      http3: batchEditForm.http3,
      ssl_profile: batchEditForm.ssl_profile
    }
  }
  if (batchEditChecks.origin_list || batchEditChecks.origin_conditions || batchEditChecks.origin_health_check) {
    settings.origin = {
      list: batchEditForm.origin_list,
      conditions: batchEditForm.origin_conditions,
      health_check: batchEditForm.origin_health_check
    }
  }
  if (batchEditChecks.backsource_protocol || batchEditChecks.backsource_http_port || batchEditChecks.backsource_https_port ||
      batchEditChecks.backsource_host || batchEditChecks.backsource_timeout || batchEditChecks.connect_timeout) {
    settings.backsource = {
      protocol: batchEditForm.backsource_protocol,
      http_port: batchEditForm.backsource_http_port,
      https_port: batchEditForm.backsource_https_port,
      host_mode: batchEditForm.backsource_host_mode,
      host_custom: batchEditForm.backsource_host_custom,
      timeout: batchEditForm.backsource_timeout,
      connect_timeout: batchEditForm.connect_timeout
    }
  }
  if (batchEditChecks.cache_enable || batchEditChecks.cache_preset || batchEditChecks.cache_rules) {
    settings.cache = {
      enable: batchEditForm.cache_enable,
      preset: batchEditForm.cache_preset,
      rules: batchEditForm.cache_rules.map(normalizeCacheRule)
    }
  }
  if (batchEditChecks.security_default_rule || batchEditChecks.security_auto_switch || batchEditChecks.security_custom_rules ||
      batchEditChecks.security_black_time || batchEditChecks.security_white_time || batchEditChecks.security_bot ||
      batchEditChecks.security_blacklist || batchEditChecks.security_whitelist || batchEditChecks.security_shield_proxy ||
      batchEditChecks.security_region_block) {
    settings.security = {
      default_rule: batchEditForm.security_default_rule,
      auto_switch: batchEditForm.security_auto_switch,
      custom_rules: batchEditForm.security_custom_rules,
      black_time_mode: batchEditForm.security_black_time_mode,
      black_time_custom: batchEditForm.security_black_time_custom,
      white_time_mode: batchEditForm.security_white_time_mode,
      white_time_custom: batchEditForm.security_white_time_custom,
      bot: batchEditForm.security_bot,
      blacklist: splitLines(batchEditForm.security_blacklist),
      whitelist: splitLines(batchEditForm.security_whitelist),
      shield_proxy: batchEditForm.security_shield_proxy,
      region_mode: batchEditForm.security_region_mode,
      region_custom: batchEditForm.security_region_custom
    }
  }
  if (batchEditChecks.access_acl || batchEditChecks.access_hotlink || batchEditChecks.access_cors) {
    settings.access = {
      acl: batchEditForm.access_acl,
      hotlink: batchEditForm.access_hotlink,
      cors: batchEditForm.access_cors
    }
  }
  if (batchEditChecks.adv_ipv6 || batchEditChecks.adv_gzip || batchEditChecks.adv_websocket || batchEditChecks.adv_search_origin ||
      batchEditChecks.adv_error_pages || batchEditChecks.adv_url_redirect || batchEditChecks.adv_origin_headers ||
      batchEditChecks.adv_cdn_headers || batchEditChecks.adv_acme_backsource || batchEditChecks.adv_realtime_return ||
      batchEditChecks.adv_realtime_send || batchEditChecks.adv_log_request_header || batchEditChecks.adv_log_response_header ||
      batchEditChecks.adv_log_request_body || batchEditChecks.adv_body_limit) {
    settings.advanced = {
      ipv6: batchEditForm.adv_ipv6,
      gzip: batchEditForm.adv_gzip,
      websocket: batchEditForm.adv_websocket,
      search_origin: batchEditForm.adv_search_origin,
      error_pages: batchEditForm.adv_error_pages,
      url_redirects: batchEditForm.adv_url_redirects,
      origin_headers: batchEditForm.adv_origin_headers,
      cdn_headers: batchEditForm.adv_cdn_headers,
      acme_backsource: batchEditForm.adv_acme_backsource,
      realtime_return: batchEditForm.adv_realtime_return,
      realtime_send: batchEditForm.adv_realtime_send,
      log_request_header: batchEditForm.adv_log_request_header,
      log_response_header: batchEditForm.adv_log_response_header,
      log_request_body: batchEditForm.adv_log_request_body,
      body_limit: batchEditForm.adv_body_limit
    }
  }
  return settings
}

const submitBatchEdit = () => {
  const ids = selectedRows.value.map(row => row.id)
  if (!ids.length) {
    ElMessage.warning('请选择网站')
    return
  }
  const payload = { ids }
  if (batchEditChecks.group_id) payload.group_id = batchEditForm.group_id || 0
  if (batchEditChecks.dns_provider_id) payload.dns_provider_id = batchEditForm.dns_provider_id || 0
  if (batchEditChecks.http_enable || batchEditChecks.http_listen) {
    payload.http_listen = batchEditForm.http_enable ? splitFields(batchEditForm.http_listen) : []
  }
  if (batchEditChecks.https_enable || batchEditChecks.https_listen) {
    payload.https_listen = batchEditForm.https_enable ? splitFields(batchEditForm.https_listen) : []
  }
  if (batchEditChecks.balance_way) payload.balance_way = batchEditForm.balance_way
  if (batchEditChecks.backsource_protocol) payload.backend_protocol = batchEditForm.backsource_protocol
  if (batchEditChecks.security_default_rule) payload.cc_default_rule = batchEditForm.security_default_rule
  if (batchEditChecks.security_blacklist) payload.black_ip = batchEditForm.security_blacklist
  if (batchEditChecks.security_whitelist) payload.white_ip = batchEditForm.security_whitelist
  if (batchEditChecks.security_region_block) {
    payload.block_region = batchEditForm.security_region_mode === 'custom'
      ? batchEditForm.security_region_custom.join(',')
      : batchEditForm.security_region_mode
  }
  const settings = buildSettingsPayload()
  if (Object.keys(settings).length) {
    payload.settings = settings
  }
  request.post('/sites/batch_update', payload).then(() => {
    ElMessage.success('批量修改完成')
    batchEditVisible.value = false
    fetchList()
  })
}

const handleBatchAction = action => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择网站')
    return
  }
  const ids = selectedRows.value.map(row => row.id)
  ElMessageBox.confirm(`确定执行${action}操作?`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.post('/sites/batch_action', { action, ids }).then(res => {
      ElMessage.success(res.message || '操作成功')
      fetchList()
    })
  })
}

const handleRowAction = (action, row) => {
  selectedRows.value = [row]
  handleBatchAction(action)
}

const handleApplyCert = () => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择网站')
    return
  }
  const ids = selectedRows.value.map(row => row.id)
  request.post('/sites/apply_cert', { ids }).then(res => {
    ElMessage.success(res.message || '已提交证书申请')
    fetchList()
  })
}

const handleExport = () => {
  request.get('/sites/export', {
    params: {
      keyword: listQuery.keyword,
      search_field: listQuery.searchField,
      page: 1,
      pageSize: 10000
    },
    responseType: 'blob'
  }).then(res => {
    const blob = new Blob([res], { type: 'text/csv' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'sites.csv'
    link.click()
    window.URL.revokeObjectURL(url)
  })
}

const handleManage = row => {
  router.push({ path: '/website/rules', query: { site_id: row.id } })
}

const applyAdvancedFilter = () => {
  advancedVisible.value = false
  handleFilter()
}

const removeTag = key => {
  if (key === 'keyword') {
    listQuery.keyword = ''
  } else if (key in advancedForm) {
    advancedForm[key] = ''
  }
  handleFilter()
}

const clearFilters = () => {
  listQuery.keyword = ''
  advancedForm.group_id = []
  advancedForm.status = ''
  advancedForm.https = ''
  handleFilter()
}

const loadGroups = (userId) => {
  const params = { page: 1, pageSize: 200 }
  if (userId) params.user_id = userId
  request.get('/site_groups', { params }).then(res => {
    groupOptions.value = res.data?.list || res.list || []
  })
}

const loadDnsProviders = () => {
  request.get('/dns/providers').then(res => {
    const list = res.data?.list || res.list || []
    dnsOptions.value = list
    if (!createForm.dns_provider_id) {
    }
  })
}

const splitFields = value => {
  return value
    .split(/[\s,]+/)
    .map(item => item.trim())
    .filter(Boolean)
}

const splitLines = value => {
  return value
    .split(/\r?\n/)
    .map(item => item.trim())
    .filter(Boolean)
}

const addOrigin = () => {
  batchEditForm.origin_list.push({ address: '', weight: '', enable: true })
}

const removeOrigin = index => {
  confirmRemove('确认删除该源站?', () => {
    batchEditForm.origin_list.splice(index, 1)
  })
}

const addConditionOrigin = () => {
  batchEditForm.origin_conditions.push({
    item: 'uri_no_args',
    operator: 'eq',
    value: '',
    header: '',
    seconds: '',
    origin: ''
  })
}

const removeConditionOrigin = index => {
  confirmRemove('确认删除该条件源站?', () => {
    batchEditForm.origin_conditions.splice(index, 1)
  })
}

const addCacheRule = () => {
  batchEditForm.cache_rules.push({
    type: 'home',
    value: '',
    ttl: '',
    ignore_args: false,
    force_cache: false
  })
}

const removeCacheRule = index => {
  confirmRemove('确认删除该缓存规则?', () => {
    batchEditForm.cache_rules.splice(index, 1)
  })
}

const normalizeCacheRule = (rule) => {
  let type = rule.type
  let value = rule.value || ''
  if (!type) {
    const raw = (rule.rule || '').trim()
    if (raw === 'home' || raw === 'all') {
      type = raw
    } else if (raw.startsWith('dir:')) {
      type = 'dir'
      value = raw.slice(4)
    } else if (raw.startsWith('suffix:')) {
      type = 'suffix'
      value = raw.slice(7)
    } else if (raw.startsWith('path:')) {
      type = 'path'
      value = raw.slice(5)
    } else if (raw !== '') {
      type = 'path'
      value = raw
    } else {
      type = 'home'
    }
  }
  let ruleText = ''
  switch (type) {
    case 'dir':
      ruleText = `dir:${value}`
      break
    case 'suffix':
      ruleText = `suffix:${value}`
      break
    case 'path':
      ruleText = `path:${value}`
      break
    case 'home':
    case 'all':
      ruleText = type
      break
    default:
      ruleText = type
      break
  }
  return {
    rule: ruleText,
    type,
    value,
    ttl: rule.ttl,
    ignore_args: rule.ignore_args,
    force_cache: rule.force_cache
  }
}

const addSecurityRule = () => {
  batchEditForm.security_custom_rules.push({ match: '', action: '', mode: '', remark: '' })
}

const removeSecurityRule = index => {
  confirmRemove('确认删除该安全规则?', () => {
    batchEditForm.security_custom_rules.splice(index, 1)
  })
}

const addErrorPage = () => {
  batchEditForm.adv_error_pages.push({ code: '', content: '' })
}

const removeErrorPage = index => {
  confirmRemove('确认删除该错误页?', () => {
    batchEditForm.adv_error_pages.splice(index, 1)
  })
}

const addUrlRedirect = () => {
  batchEditForm.adv_url_redirects.push({ host: '', match: '', target: '', code: '' })
}

const removeUrlRedirect = index => {
  confirmRemove('确认删除该跳转规则?', () => {
    batchEditForm.adv_url_redirects.splice(index, 1)
  })
}

const addOriginHeader = () => {
  batchEditForm.adv_origin_headers.push({ name: '', value: '' })
}

const removeOriginHeader = index => {
  confirmRemove('确认删除该回源头?', () => {
    batchEditForm.adv_origin_headers.splice(index, 1)
  })
}

const addCdnHeader = () => {
  batchEditForm.adv_cdn_headers.push({ name: '', value: '' })
}

const removeCdnHeader = index => {
  confirmRemove('确认删除该响应头?', () => {
    batchEditForm.adv_cdn_headers.splice(index, 1)
  })
}

const labelForSearchField = value => {
  const mapping = {
    all: '全字段',
    domain: '域名',
    multi_domain: '多域名',
    origin: '源IP',
    group: '网站分组',
    site_id: '网站ID',
    cname: 'CNAME',
    package: '网站套餐',
    http_port: 'HTTP端口',
    https_port: 'HTTPS端口'
  }
  return mapping[value] || '搜索'
}

watch(
  () => createForm.user_id,
  (val) => {
    if (!isAdmin.value) return
    createForm.user_package_id = undefined
    packageOptions.value = []
    if (val) {
      loadPackages(val)
      loadGroups(val)
    }
  }
)

watch(
  () => batchForm.user_id,
  (val) => {
    if (!isAdmin.value) return
    batchForm.user_package_id = undefined
    packageOptions.value = []
    if (val) {
      loadPackages(val)
      loadGroups(val)
    }
  }
)

watch(
  () => defaultForm.user_id,
  (val) => {
    if (!isAdmin.value) return
    defaultForm.group_id = 0
    if (val) loadGroups(val)
  }
)

onMounted(() => {
  fetchList()
  loadGroups()
  loadDnsProviders()
  loadCcRuleOptions()
  if (!isAdmin.value) {
    loadPackages()
  }
  // Pre-load some users or wait for search
  // loadUsers('')
})

const createGroupVisible = ref(false)
const createGroupForm = reactive({ name: '', remark: '' })

const openCreateGroupDialog = () => {
  createGroupForm.name = ''
  createGroupForm.remark = ''
  createGroupVisible.value = true
}

const submitCreateGroup = () => {
  if (!createGroupForm.name) {
    ElMessage.warning('请输入分组名称')
    return
  }
  request.post('/site_groups', { name: createGroupForm.name, remark: createGroupForm.remark }).then(() => {
    ElMessage.success('创建成功')
    createGroupVisible.value = false
    loadGroups()
  })
}
</script>

<style scoped>
.filter-container {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.condition-origin-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.filter-left,
.filter-right {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.filter-tags {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}
.filter-tags-clear {
  margin-left: auto;
}
.dnsapi-section {
  margin-top: 16px;
}
.dnsapi-toolbar {
  margin-bottom: 12px;
  display: flex;
  gap: 8px;
}
.default-section {
  margin-top: 16px;
}
.default-toolbar {
  margin-bottom: 12px;
  display: flex;
  gap: 8px;
}
.default-empty {
  color: #909399;
  text-align: center;
  padding: 20px 0;
}
.header-list {
  width: 100%;
}
.dnsapi-fields {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.dnsapi-field-row {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 12px;
}
.dnsapi-field-label {
  width: 220px;
  text-align: right;
  color: var(--el-text-color-regular);
  white-space: nowrap;
}
.dnsapi-field-row .el-input {
  flex: 1 1 auto;
}
.dnsapi-dialog .el-dialog__header {
  background: var(--el-color-primary-light-9);
  border-bottom: 1px solid var(--el-border-color);
  margin-right: 0;
  padding: 14px 20px;
}
.dnsapi-dialog .el-dialog__title {
  color: var(--el-color-primary);
  font-weight: 600;
}
.dnsapi-dialog .el-dialog__body {
  padding-top: 20px;
}
.dnsapi-dialog .el-form-item__label {
  white-space: nowrap;
}
.dnsapi-dialog .dnsapi-fields .el-form-item__label {
  width: 220px;
}
.pagination-container {
  margin-top: 16px;
  text-align: right;
}
.expand-more {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #409eff;
  cursor: pointer;
  margin: 6px 0 10px;
}
.extra-fields {
  padding-top: 4px;
}
.help-text {
  font-size: 12px;
  color: #909399;
  margin-top: 6px;
}
.batch-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.batch-header {
  margin-bottom: 12px;
  color: #606266;
  font-size: 12px;
}
.batch-action {
  display: flex;
  justify-content: center;
  margin-top: 12px;
}
.batch-dialog-body {
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 8px;
}
.link-more {
  color: #409eff;
  cursor: pointer;
  font-size: 12px;
  margin-left: 8px;
}
</style>



