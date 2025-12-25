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
              <el-form-item label="所属用户">
                <el-select
                  v-model.number="createForm.user_id"
                  filterable
                  remote
                  clearable
                  placeholder="搜索用户 (默认管理员)"
                  :remote-method="loadUsers"
                  :loading="userLoading">
                  <el-option v-for="u in userOptions" :key="u.id" :label="u.username" :value="u.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="网站套餐">
                <el-select v-model.number="createForm.user_package_id" clearable placeholder="选择套餐 (可选)" style="width: 100%;">
                  <el-option v-for="p in packageOptions" :key="p.id" :label="p.name" :value="p.id" />
                </el-select>
              </el-form-item>
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
                <el-table-column label="条件">
                  <template #default="{ row }">
                    <el-input v-model="row.condition" placeholder="匹配条件" />
                  </template>
                </el-table-column>
                <el-table-column label="源站">
                  <template #default="{ row }">
                    <el-input v-model="row.origin" placeholder="源站" />
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
                <el-table-column label="规则">
                  <template #default="{ row }">
                    <el-input v-model="row.rule" placeholder="匹配规则" />
                  </template>
                </el-table-column>
                <el-table-column label="有效期" width="120">
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
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, CircleCheckFilled, CircleCloseFilled, Plus } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { useRouter } from 'vue-router'
import CountrySelector from '@/components/CountrySelector.vue'
import ResolvePage from './Resolve.vue'

const router = useRouter()
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
    request.get('/users/list', { params: { keyword: query, pageSize: 20 } }).then(res => {
      userOptions.value = res.data?.list || res.list || []
      userLoading.value = false
    }).catch(() => {
      userLoading.value = false
    })
  } else {
    userOptions.value = []
  }
}

const loadPackages = () => {
  request.get('/user_packages').then(res => {
    packageOptions.value = res.data?.list || res.list || []
  })
}

const cachePresets = [
  { label: '首页缓存', value: 'home' },
  { label: '全站缓存', value: 'all' },
  { label: '静态资源缓存', value: 'static' },
  { label: '视频文件缓存', value: 'video' },
  { label: 'Wordpress缓存', value: 'wordpress' }
]

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

const handleTopTab = tab => {
  if (tab.paneName === 'default') {
    router.push('/global/default')
  } else if (tab.paneName === 'dns') {
    loadDnsapiList()
    loadDnsapiTypes()
  }
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
      rules: batchEditForm.cache_rules
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

const loadGroups = () => {
  request.get('/site_groups').then(res => {
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
  batchEditForm.origin_conditions.push({ condition: '', origin: '' })
}

const removeConditionOrigin = index => {
  confirmRemove('确认删除该条件源站?', () => {
    batchEditForm.origin_conditions.splice(index, 1)
  })
}

const addCacheRule = () => {
  batchEditForm.cache_rules.push({ rule: '', ttl: '', ignore_args: false, force_cache: false })
}

const removeCacheRule = index => {
  confirmRemove('确认删除该缓存规则?', () => {
    batchEditForm.cache_rules.splice(index, 1)
  })
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

onMounted(() => {
  fetchList()
  loadGroups()
  loadDnsProviders()
  loadPackages()
  // Pre-load some users or wait for search
  // loadUsers('') 
})

const createGroupVisible = ref(false)
const createGroupForm = reactive({ name: '' })

const openCreateGroupDialog = () => {
  createGroupForm.name = ''
  createGroupVisible.value = true
}

const submitCreateGroup = () => {
  if (!createGroupForm.name) {
    ElMessage.warning('请输入分组名称')
    return
  }
  request.post('/site_groups', { name: createGroupForm.name }).then(() => {
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



