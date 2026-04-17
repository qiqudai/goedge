<template>
  <el-dialog
    v-model="visible"
    title="批量修改网站设置"
    width="1050px"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="batch-tip"
      title="仅会修改勾选的设置项，未勾选项保持原值不变。"
    />

    <el-collapse v-model="activeSection" accordion>
      <el-collapse-item title="基本设置" name="basic">
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.basic.userPackageId">套餐设置</el-checkbox>
            <el-select
              v-model="form.basic.userPackageId"
              placeholder="请选择套餐"
              filterable
              clearable
              :loading="loading.userPackages"
              :disabled="!selected.basic.userPackageId"
              style="width: 360px"
            >
              <el-option
                v-for="pkg in userPackages"
                :key="pkg.id"
                :label="pkg.name || pkg.user_plan_name || ('套餐 ' + pkg.id)"
                :value="pkg.id"
              />
            </el-select>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.basic.groupIds">所属分组</el-checkbox>
            <el-select
              v-model="form.basic.groupIds"
              placeholder="请选择分组"
              clearable
              multiple
              filterable
              :loading="loading.siteGroups"
              :disabled="!selected.basic.groupIds"
              style="width: 360px"
            >
              <el-option
                v-for="group in siteGroups"
                :key="group.id"
                :label="group.name"
                :value="group.id"
              />
            </el-select>
          </div>
        </el-form>
        <div class="section-footer">
          <el-button
            type="primary"
            :disabled="!hasSectionSelection('basic') || !hasIds"
            :loading="isSubmitting('basic')"
            @click="submitSection('basic')"
          >
            批量修改
          </el-button>
        </div>
      </el-collapse-item>

      <el-collapse-item title="HTTP 设置" name="http">
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.http.enable">HTTP 开关</el-checkbox>
            <el-switch v-model="form.http.enable" :disabled="!selected.http.enable" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.http.ports">监听端口</el-checkbox>
            <el-input
              v-model="form.http.ports"
              placeholder="80 8080"
              :disabled="!selected.http.ports"
              style="width: 360px"
            />
          </div>
        </el-form>
        <div class="section-footer">
          <el-button
            type="primary"
            :disabled="!hasSectionSelection('http') || !hasIds"
            :loading="isSubmitting('http')"
            @click="submitSection('http')"
          >
            批量修改
          </el-button>
        </div>
      </el-collapse-item>

      <el-collapse-item title="HTTPS 设置" name="https">
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.https.enable">HTTPS 开关</el-checkbox>
            <el-switch v-model="form.https.enable" :disabled="!selected.https.enable" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.https.certId">证书选择</el-checkbox>
            <el-select
              v-model="form.https.certId"
              placeholder="请选择证书"
              filterable
              clearable
              :loading="loading.certs"
              :disabled="!selected.https.certId"
              style="width: 360px"
            >
              <el-option
                v-for="cert in certList"
                :key="cert.id"
                :label="cert.name"
                :value="cert.id"
              />
            </el-select>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.https.listenPorts">监听端口</el-checkbox>
            <el-input
              v-model="form.https.listenPorts"
              placeholder="443 8443"
              :disabled="!selected.https.listenPorts"
              style="width: 360px"
            />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.https.force">强制 HTTPS</el-checkbox>
            <div class="inline-field">
              <el-switch v-model="form.https.force" :disabled="!selected.https.force" />
              <el-input
                v-model="form.https.forcePort"
                placeholder="443"
                :disabled="!selected.https.force || !form.https.force"
                style="width: 120px"
              >
                <template #append>端口</template>
              </el-input>
            </div>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.https.hsts">HSTS</el-checkbox>
            <el-switch v-model="form.https.hsts" :disabled="!selected.https.hsts" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.https.http2">HTTP2</el-checkbox>
            <el-switch v-model="form.https.http2" :disabled="!selected.https.http2" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.https.http3">HTTP3</el-checkbox>
            <el-switch v-model="form.https.http3" :disabled="!selected.https.http3" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.https.ocsp">OCSP 装订</el-checkbox>
            <el-switch v-model="form.https.ocsp" :disabled="!selected.https.ocsp" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.https.ssl">SSL 配置</el-checkbox>
            <el-radio-group v-model="form.https.sslPolicy" :disabled="!selected.https.ssl">
              <el-radio value="compat">兼容</el-radio>
              <el-radio value="modern">现代</el-radio>
              <el-radio value="custom">自定义</el-radio>
            </el-radio-group>
          </div>
          <div class="batch-row" v-if="form.https.sslPolicy === 'custom'">
            <span class="batch-label">协议</span>
            <el-input
              v-model="form.https.sslProtocols"
              placeholder="如：TLSv1.2 TLSv1.3"
              type="textarea"
              :rows="2"
              :disabled="!selected.https.ssl"
              style="width: 360px"
            />
          </div>
          <div class="batch-row" v-if="form.https.sslPolicy === 'custom'">
            <span class="batch-label">加密套件</span>
            <el-input
              v-model="form.https.sslCiphers"
              placeholder="如：ECDHE..."
              type="textarea"
              :rows="2"
              :disabled="!selected.https.ssl"
              style="width: 360px"
            />
          </div>
        </el-form>
        <div class="section-footer">
          <el-button
            type="primary"
            :disabled="!hasSectionSelection('https') || !hasIds"
            :loading="isSubmitting('https')"
            @click="submitSection('https')"
          >
            批量修改
          </el-button>
        </div>
      </el-collapse-item>

      <el-collapse-item title="回源设置" name="origin">
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.origin.protocol">回源协议</el-checkbox>
            <el-radio-group v-model="form.origin.protocol" :disabled="!selected.origin.protocol">
              <el-radio value="http">HTTP</el-radio>
              <el-radio value="https">HTTPS</el-radio>
              <el-radio value="follow">跟随协议</el-radio>
              <el-radio value="follow_port">跟随端口和协议</el-radio>
            </el-radio-group>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.origin.httpPort">HTTP 回源端口</el-checkbox>
            <el-input
              v-model="form.origin.httpPort"
              placeholder="80"
              :disabled="!selected.origin.httpPort"
              style="width: 200px"
            />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.origin.httpsPort">HTTPS 回源端口</el-checkbox>
            <el-input
              v-model="form.origin.httpsPort"
              placeholder="443"
              :disabled="!selected.origin.httpsPort"
              style="width: 200px"
            />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.origin.host">回源主机</el-checkbox>
            <div class="inline-field">
              <el-radio-group v-model="form.origin.host" :disabled="!selected.origin.host">
                <el-radio value="follow">自动跟随</el-radio>
                <el-radio value="domain">网站域名</el-radio>
                <el-radio value="custom">自定义</el-radio>
              </el-radio-group>
              <el-input
                v-if="form.origin.host === 'custom'"
                v-model="form.origin.hostValue"
                placeholder="如：origin.example.com"
                :disabled="!selected.origin.host"
                style="width: 220px"
              />
            </div>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.origin.timeout">回源超时</el-checkbox>
            <el-input
              v-model.number="form.origin.timeout"
              placeholder="60"
              :disabled="!selected.origin.timeout"
              style="width: 160px"
            >
              <template #append>秒</template>
            </el-input>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.origin.connTimeout">连接超时</el-checkbox>
            <el-input
              v-model.number="form.origin.connTimeout"
              placeholder="10"
              :disabled="!selected.origin.connTimeout"
              style="width: 160px"
            >
              <template #append>秒</template>
            </el-input>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.origin.balanceWay">负载方式</el-checkbox>
            <el-select
              v-model="form.origin.balanceWay"
              placeholder="请选择"
              :disabled="!selected.origin.balanceWay"
              style="width: 200px"
            >
              <el-option label="IP 哈希" value="ip_hash" />
              <el-option label="轮循" value="rr" />
              <el-option label="URL 哈希" value="url_hash" />
              <el-option label="最少连接" value="least_conn" />
              <el-option label="随机" value="random" />
            </el-select>
          </div>

          <div class="divider"></div>
          <div class="section-title">源站列表</div>
          <el-table :data="form.origin.list" border size="small" style="margin-bottom: 12px;">
            <el-table-column prop="address" label="源地址">
              <template #default="{ row }">
                <el-input
                  v-model="row.address"
                  placeholder="IP 或域名"
                  size="small"
                  :disabled="!selected.origin.list"
                />
              </template>
            </el-table-column>
            <el-table-column prop="weight" label="权重" width="120">
              <template #default="{ row }">
                <el-input v-model="row.weight" size="small" :disabled="!selected.origin.list" />
              </template>
            </el-table-column>
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-switch v-model="row.enable" size="small" :disabled="!selected.origin.list" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ $index }">
                <el-button
                  link
                  type="danger"
                  size="small"
                  :disabled="!selected.origin.list"
                  @click="removeOrigin($index)"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button size="small" type="primary" :disabled="!selected.origin.list" @click="addOrigin">
            新增源站
          </el-button>

          <el-divider />
          <div class="section-title">条件源站</div>
          <el-table :data="form.origin.conditions" border size="small" style="margin-bottom: 12px;">
            <el-table-column label="匹配项" width="160">
              <template #default="{ row }">
                <el-select
                  v-model="row.item"
                  size="small"
                  placeholder="请选择"
                  :disabled="!selected.origin.conditions"
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
                    placeholder="请求头名称"
                    :disabled="!selected.origin.conditions"
                  />
                  <el-input
                    v-else-if="isOriginStatItem(row.item)"
                    v-model="row.seconds"
                    size="small"
                    placeholder="统计秒数"
                    :disabled="!selected.origin.conditions"
                  />
                  <el-input
                    v-else
                    v-model="row.value"
                    size="small"
                    :placeholder="getOriginConditionPlaceholder(row)"
                    :disabled="!selected.origin.conditions"
                  />
                  <el-select
                    v-if="!isOriginStatItem(row.item)"
                    v-model="row.operator"
                    size="small"
                    placeholder="匹配方式"
                    style="width: 140px;"
                    :disabled="!selected.origin.conditions"
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
                  :disabled="!selected.origin.conditions"
                />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ $index }">
                <el-button
                  link
                  type="danger"
                  size="small"
                  :disabled="!selected.origin.conditions"
                  @click="removeConditionOrigin($index)"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button size="small" type="primary" :disabled="!selected.origin.conditions" @click="addConditionOrigin">
            新增条件源站
          </el-button>
        </el-form>
        <div class="section-footer">
          <el-button
            type="primary"
            :disabled="!hasSectionSelection('origin') || !hasIds"
            :loading="isSubmitting('origin')"
            @click="submitSection('origin')"
          >
            批量修改
          </el-button>
        </div>
      </el-collapse-item>

      <el-collapse-item title="缓存设置" name="cache">
        <div class="toolbar-row">
          <el-checkbox v-model="selected.cache.rules">缓存规则</el-checkbox>
          <el-button size="small" type="primary" :disabled="!selected.cache.rules" @click="openCacheRuleDialog()">新增规则</el-button>
          <el-button size="small" type="danger" :disabled="!selected.cache.rules || !selectedCacheRows.length" @click="removeCacheRulesBatch">删除</el-button>
          <el-select
            v-model="cachePreset"
            placeholder="快速添加缓存"
            size="small"
            style="width: 160px;"
            :disabled="!selected.cache.rules"
            @change="applyCachePreset"
          >
            <el-option label="首页缓存" value="index" />
            <el-option label="全站缓存" value="all" />
            <el-option label="静态资源缓存" value="static" />
            <el-option label="视频资源" value="video" />
            <el-option label="WordPress 缓存" value="wordpress" />
          </el-select>
        </div>
        <el-table :data="form.cache.rules" border size="small" @selection-change="handleCacheSelection">
          <el-table-column type="selection" width="50" :selectable="() => selected.cache.rules" />
          <el-table-column label="类型" min-width="120">
            <template #default="{ row }">{{ cacheTypeLabelMap[row.type] || row.type }}</template>
          </el-table-column>
          <el-table-column label="内容" min-width="240" prop="value" />
          <el-table-column label="TTL(秒)" width="120" prop="ttl" />
          <el-table-column label="操作" width="140">
            <template #default="{ row, $index }">
              <el-button link type="primary" size="small" :disabled="!selected.cache.rules" @click="openCacheRuleDialog(row, $index)">编辑</el-button>
              <el-button link type="danger" size="small" :disabled="!selected.cache.rules" @click="removeCacheRule($index)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="section-footer">
          <el-button
            type="primary"
            :disabled="!hasSectionSelection('cache') || !hasIds"
            :loading="isSubmitting('cache')"
            @click="submitSection('cache')"
          >
            批量修改
          </el-button>
        </div>
      </el-collapse-item>

      <el-collapse-item title="安全设置" name="security">
        <div class="section-title">CC 防护</div>
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.security.defaultRule">默认规则</el-checkbox>
            <el-radio-group v-model="computedMode" :disabled="!selected.security.defaultRule">
              <el-radio v-for="rule in systemRules" :key="rule.id" :value="rule.id">{{ rule.name }}</el-radio>
              <el-radio value="custom">自定义</el-radio>
            </el-radio-group>
            <el-select
              v-if="computedMode === 'custom'"
              v-model="form.security.cc.mode"
              placeholder="请选择自定义规则"
              size="small"
              style="width: 180px;"
              :disabled="!selected.security.defaultRule"
            >
              <el-option
                v-for="rule in userRules"
                :key="rule.id"
                :label="rule.name"
                :value="rule.id"
              />
            </el-select>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.security.autoSwitch">自动防护</el-checkbox>
            <div class="inline-field">
              <el-switch v-model="form.security.cc.autoSwitch.enable" :disabled="!selected.security.autoSwitch" />
              <span v-if="form.security.cc.autoSwitch.enable" class="inline-text">
                当 QPS 超过
                <el-select
                  v-model="qpsSelection"
                  size="small"
                  style="width: 100px"
                  :disabled="!selected.security.autoSwitch"
                  @change="handleQpsChange"
                >
                  <el-option label="20" :value="20" />
                  <el-option label="50" :value="50" />
                  <el-option label="200" :value="200" />
                  <el-option label="自定义" value="custom" />
                </el-select>
                <el-input
                  v-if="qpsSelection === 'custom'"
                  v-model.number="form.security.cc.autoSwitch.qps"
                  size="small"
                  style="width: 80px"
                  :disabled="!selected.security.autoSwitch"
                />
                时，切换到
                <el-select
                  v-model="form.security.cc.autoSwitch.rule"
                  size="small"
                  style="width: 120px"
                  :disabled="!selected.security.autoSwitch"
                >
                  <el-option label="关闭" value="close" />
                  <el-option label="宽松" value="lenient" />
                  <el-option label="普通" value="normal" />
                  <el-option label="严格" value="strict" />
                  <el-option label="JS验证" value="js" />
                  <el-option label="验证码" value="captcha" />
                </el-select>
              </span>
            </div>
          </div>
        </el-form>

        <div class="divider"></div>
        <div class="section-title">自定义规则</div>
        <div class="toolbar-row">
          <el-checkbox v-model="selected.security.customRules">规则列表</el-checkbox>
          <el-button size="small" type="primary" :disabled="!selected.security.customRules" @click="openRuleDialog('create')">新增规则</el-button>
          <el-button size="small" :disabled="!selected.security.customRules" @click="toggleAllRules(true)">启用所有规则</el-button>
          <el-button size="small" :disabled="!selected.security.customRules" @click="toggleAllRules(false)">关闭所有规则</el-button>
        </div>
        <el-table :data="form.security.cc.customRules" border size="small" style="margin-bottom: 10px;">
          <el-table-column label="匹配条件" min-width="200">
            <template #default="{ row }">
              <div v-for="(m, idx) in row.matchers" :key="idx" style="font-size: 12px;">
                {{ getMatcherText(m) }}
              </div>
              <div v-if="!row.matchers || !row.matchers.length">匹配所有请求</div>
            </template>
          </el-table-column>
          <el-table-column label="执行过滤" width="120">
            <template #default="{ row }">{{ getActionText(row.action) }}</template>
          </el-table-column>
          <el-table-column label="匹配模式" width="120">
            <template #default="{ row }">{{ row.breakMatch ? '停止匹配' : '继续下一条' }}</template>
          </el-table-column>
          <el-table-column prop="remark" label="备注" />
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-switch v-model="row.on" size="small" :disabled="!selected.security.customRules" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" align="center">
            <template #default="{ $index }">
              <el-button link type="primary" size="small" :disabled="!selected.security.customRules" @click="editRule($index)">编辑</el-button>
              <el-button link type="danger" size="small" :disabled="!selected.security.customRules" @click="deleteRule($index)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="divider"></div>
        <div class="section-title">搜索引擎爬虫</div>
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.security.crawlers">爬虫策略</el-checkbox>
            <el-radio-group v-model="form.security.crawlers.action" :disabled="!selected.security.crawlers">
              <el-radio value="none">不设置</el-radio>
              <el-radio value="allow">放行</el-radio>
              <el-radio value="block">拦截</el-radio>
            </el-radio-group>
          </div>
        </el-form>

        <div class="divider"></div>
        <div class="section-title">黑白名单时间</div>
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.security.blackTime">黑名单时间</el-checkbox>
            <div class="inline-field">
              <el-radio-group v-model="form.security.ip.blackTimeCustom" :disabled="!selected.security.blackTime">
                <el-radio :value="false">系统默认(3600秒)</el-radio>
                <el-radio :value="true">自定义</el-radio>
              </el-radio-group>
              <el-input
                v-if="form.security.ip.blackTimeCustom"
                v-model.number="form.security.ip.blackTime"
                size="small"
                style="width: 120px"
                :disabled="!selected.security.blackTime"
              >
                <template #append>秒</template>
              </el-input>
            </div>
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.security.whiteTime">白名单时间</el-checkbox>
            <div class="inline-field">
              <el-radio-group v-model="form.security.ip.whiteTimeCustom" :disabled="!selected.security.whiteTime">
                <el-radio :value="false">系统默认(21600秒)</el-radio>
                <el-radio :value="true">自定义</el-radio>
              </el-radio-group>
              <el-input
                v-if="form.security.ip.whiteTimeCustom"
                v-model.number="form.security.ip.whiteTime"
                size="small"
                style="width: 120px"
                :disabled="!selected.security.whiteTime"
              >
                <template #append>秒</template>
              </el-input>
            </div>
          </div>
        </el-form>

        <div class="divider"></div>
        <div class="section-title">黑白名单</div>
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.security.blackList">黑名单</el-checkbox>
            <el-input
              v-model="form.security.ip.black"
              type="textarea"
              :rows="3"
              placeholder="一行一个 IP"
              :disabled="!selected.security.blackList"
              style="width: 360px"
            />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.security.whiteList">白名单</el-checkbox>
            <el-input
              v-model="form.security.ip.white"
              type="textarea"
              :rows="3"
              placeholder="一行一个 IP"
              :disabled="!selected.security.whiteList"
              style="width: 360px"
            />
          </div>
        </el-form>

        <div class="divider"></div>
        <div class="section-title">Cookie 域名</div>
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.security.cookie">Cookie 设置</el-checkbox>
            <div class="inline-field">
              <el-switch v-model="form.security.cookie.enable" :disabled="!selected.security.cookie" />
              <el-input
                v-if="form.security.cookie.enable"
                v-model="form.security.cookie.domain"
                placeholder="如：abc.com"
                :disabled="!selected.security.cookie"
                style="width: 200px"
              />
            </div>
          </div>
        </el-form>

        <div class="divider"></div>
        <div class="section-title">屏蔽设置</div>
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.security.blockProxy">屏蔽透明代理</el-checkbox>
            <el-switch v-model="form.security.block.transparentProxy" :disabled="!selected.security.blockProxy" />
          </div>
        </el-form>

        <div class="divider"></div>
        <div class="section-title">区域屏蔽</div>
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.security.regionBlock">区域选择</el-checkbox>
            <div :class="{ 'disabled-block': !selected.security.regionBlock }">
              <CountrySelector v-model="form.security.regions" />
            </div>
          </div>
        </el-form>

        <div class="section-footer">
          <el-button
            type="primary"
            :disabled="!hasSectionSelection('security') || !hasIds"
            :loading="isSubmitting('security')"
            @click="submitSection('security')"
          >
            批量修改
          </el-button>
        </div>
      </el-collapse-item>

      <el-collapse-item title="访问控制" name="access">
        <el-form label-width="140px" class="section-form">
          <div class="batch-row">
            <el-checkbox v-model="selected.access.acl">ACL 规则</el-checkbox>
            <el-select
              v-model="form.access.acl"
              placeholder="请选择 ACL"
              clearable
              :loading="loading.acls"
              :disabled="!selected.access.acl"
              style="width: 360px"
            >
              <el-option v-for="item in aclList" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </div>

          <div class="divider"></div>
          <div class="section-title">防盗链设置</div>
          <div class="batch-row">
            <el-checkbox v-model="selected.access.hotlink">防盗链</el-checkbox>
            <el-switch v-model="form.access.hotlink.enable" :disabled="!selected.access.hotlink" />
          </div>
          <div class="batch-row" v-if="form.access.hotlink.enable">
            <span class="batch-label">范围</span>
            <el-radio-group v-model="form.access.hotlink.scope" :disabled="!selected.access.hotlink">
              <el-radio value="all">整站</el-radio>
              <el-radio value="suffix">后缀</el-radio>
              <el-radio value="dir">目录</el-radio>
              <el-radio value="path">单个路径</el-radio>
            </el-radio-group>
            <el-input
              v-if="form.access.hotlink.scope !== 'all'"
              v-model="form.access.hotlink.value"
              :placeholder="getHotlinkPlaceholder(form.access.hotlink.scope)"
              :disabled="!selected.access.hotlink"
              style="width: 260px"
            />
          </div>
          <div class="batch-row" v-if="form.access.hotlink.enable">
            <span class="batch-label">允许空来源</span>
            <el-radio-group v-model="form.access.hotlink.allowEmpty" :disabled="!selected.access.hotlink">
              <el-radio :value="true">允许</el-radio>
              <el-radio :value="false">不允许</el-radio>
            </el-radio-group>
          </div>
          <div class="batch-row" v-if="form.access.hotlink.enable">
            <span class="batch-label">额外域名</span>
            <el-input
              v-model="form.access.hotlink.domains"
              placeholder="多个域名空格分隔"
              :disabled="!selected.access.hotlink"
              style="width: 360px"
            />
          </div>

          <div class="divider"></div>
          <div class="section-title">跨域访问设置</div>
          <div class="batch-row">
            <el-checkbox v-model="selected.access.cors">跨域（CORS）</el-checkbox>
            <el-switch v-model="form.access.cors.enable" :disabled="!selected.access.cors" />
          </div>
          <div class="batch-row" v-if="form.access.cors.enable">
            <span class="batch-label">允许来源</span>
            <el-input v-model="form.access.cors.allowOrigin" :disabled="!selected.access.cors" style="width: 360px" />
          </div>
          <div class="batch-row" v-if="form.access.cors.enable">
            <span class="batch-label">允许方法</span>
            <el-input v-model="form.access.cors.allowMethods" :disabled="!selected.access.cors" style="width: 360px" />
          </div>
          <div class="batch-row" v-if="form.access.cors.enable">
            <span class="batch-label">允许请求头</span>
            <el-input v-model="form.access.cors.allowHeaders" :disabled="!selected.access.cors" style="width: 360px" />
          </div>
          <div class="batch-row" v-if="form.access.cors.enable">
            <span class="batch-label">暴露响应头</span>
            <el-input v-model="form.access.cors.exposeHeaders" :disabled="!selected.access.cors" style="width: 360px" />
          </div>
          <div class="batch-row" v-if="form.access.cors.enable">
            <span class="batch-label">允许携带凭证</span>
            <el-radio-group v-model="form.access.cors.allowCredentials" :disabled="!selected.access.cors">
              <el-radio :value="true">允许</el-radio>
              <el-radio :value="false">不允许</el-radio>
            </el-radio-group>
          </div>
          <div class="batch-row" v-if="form.access.cors.enable">
            <span class="batch-label">最大缓存时间</span>
            <el-input v-model="form.access.cors.maxAge" :disabled="!selected.access.cors" style="width: 200px" />
          </div>
        </el-form>
        <div class="section-footer">
          <el-button
            type="primary"
            :disabled="!hasSectionSelection('access') || !hasIds"
            :loading="isSubmitting('access')"
            @click="submitSection('access')"
          >
            批量修改
          </el-button>
        </div>
      </el-collapse-item>

      <el-collapse-item title="高级设置" name="advanced">
        <el-form label-width="140px" class="section-form">
          <div class="section-title">上传大小限制</div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.uploadLimit">大小限制</el-checkbox>
            <div class="inline-field">
              <el-radio-group v-model="form.advanced.uploadLimitMode" :disabled="!selected.advanced.uploadLimit">
                <el-radio value="none">不限制</el-radio>
                <el-radio value="custom">自定义</el-radio>
              </el-radio-group>
              <el-input
                v-if="form.advanced.uploadLimitMode === 'custom'"
                v-model.number="form.advanced.uploadLimitValue"
                placeholder="102400"
                :disabled="!selected.advanced.uploadLimit"
                style="width: 160px"
              >
                <template #append>KB</template>
              </el-input>
            </div>
          </div>

          <div class="divider"></div>
          <div class="section-title">压缩设置</div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.gzip">Gzip 压缩</el-checkbox>
            <el-switch v-model="form.advanced.gzip" :disabled="!selected.advanced.gzip" />
          </div>

          <div class="divider"></div>
          <div class="section-title">WebSocket 设置</div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.websocket">WebSocket</el-checkbox>
            <el-switch v-model="form.advanced.websocket" :disabled="!selected.advanced.websocket" />
          </div>

          <div class="divider"></div>
          <div class="section-title">搜索引擎回源</div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.searchEngineOrigin">搜索引擎回源</el-checkbox>
            <div class="inline-field">
              <el-switch v-model="form.advanced.searchEngineOrigin" :disabled="!selected.advanced.searchEngineOrigin" />
              <el-input
                v-if="form.advanced.searchEngineOrigin"
                v-model="form.advanced.searchEngineOriginIp"
                placeholder="源 IP"
                :disabled="!selected.advanced.searchEngineOrigin"
                style="width: 200px"
              />
            </div>
          </div>

          <div class="divider"></div>
          <div class="section-title">URL 转向</div>
          <div class="toolbar-row">
            <el-checkbox v-model="selected.advanced.urlRedirects">URL 转向</el-checkbox>
            <el-button size="small" type="primary" :disabled="!selected.advanced.urlRedirects" @click="openRedirectDialog()">新增转向</el-button>
          </div>
          <el-table :data="form.advanced.urlRedirects" border size="small" style="margin-bottom: 12px;">
            <el-table-column label="域名端口" prop="domain" />
            <el-table-column label="匹配" prop="match" />
            <el-table-column label="转向到" prop="redirect" />
            <el-table-column label="响应码" prop="code" width="100" />
            <el-table-column label="操作" width="140">
              <template #default="{ row, $index }">
                <el-button link type="primary" size="small" :disabled="!selected.advanced.urlRedirects" @click="openRedirectDialog(row, $index)">编辑</el-button>
                <el-button link type="danger" size="small" :disabled="!selected.advanced.urlRedirects" @click="removeRedirect($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="divider"></div>
          <div class="section-title">源站请求头</div>
          <div class="toolbar-row">
            <el-checkbox v-model="selected.advanced.reqHeaders">源站请求头</el-checkbox>
            <el-button size="small" type="primary" :disabled="!selected.advanced.reqHeaders" @click="openHeaderDialog('req')">新增请求头</el-button>
          </div>
          <el-table :data="form.advanced.reqHeaders" border size="small" style="margin-bottom: 12px;">
            <el-table-column label="名称" prop="name" />
            <el-table-column label="值" prop="value" />
            <el-table-column label="操作" width="140">
              <template #default="{ row, $index }">
                <el-button link type="primary" size="small" :disabled="!selected.advanced.reqHeaders" @click="openHeaderDialog('req', row, $index)">编辑</el-button>
                <el-button link type="danger" size="small" :disabled="!selected.advanced.reqHeaders" @click="removeHeader('req', $index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="divider"></div>
          <div class="section-title">CDN 响应头</div>
          <div class="toolbar-row">
            <el-checkbox v-model="selected.advanced.resHeaders">CDN 响应头</el-checkbox>
            <el-button size="small" type="primary" :disabled="!selected.advanced.resHeaders" @click="openHeaderDialog('res')">新增响应头</el-button>
          </div>
          <el-table :data="form.advanced.resHeaders" border size="small" style="margin-bottom: 12px;">
            <el-table-column label="名称" prop="name" />
            <el-table-column label="值" prop="value" />
            <el-table-column label="操作" width="140">
              <template #default="{ row, $index }">
                <el-button link type="primary" size="small" :disabled="!selected.advanced.resHeaders" @click="openHeaderDialog('res', row, $index)">编辑</el-button>
                <el-button link type="danger" size="small" :disabled="!selected.advanced.resHeaders" @click="removeHeader('res', $index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="divider"></div>
          <div class="section-title">访问日志</div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.logRequestHeader">记录请求头</el-checkbox>
            <el-switch v-model="form.advanced.logRequestHeader" :disabled="!selected.advanced.logRequestHeader" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.logResponseHeader">记录响应头</el-checkbox>
            <el-switch v-model="form.advanced.logResponseHeader" :disabled="!selected.advanced.logResponseHeader" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.logRequestBody">记录请求体</el-checkbox>
            <el-switch v-model="form.advanced.logRequestBody" :disabled="!selected.advanced.logRequestBody" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.logRequestBodySize">请求体大小限制</el-checkbox>
            <el-input
              v-model.number="form.advanced.logRequestBodySizeLimit"
              placeholder="16"
              :disabled="!selected.advanced.logRequestBodySize"
              style="width: 160px"
            >
              <template #append>KB</template>
            </el-input>
          </div>

          <div class="divider"></div>
          <div class="section-title">其它</div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.originCert">源站证书</el-checkbox>
            <el-switch v-model="form.advanced.originCert" :disabled="!selected.advanced.originCert" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.realtimeIdentify">数据实时鉴别</el-checkbox>
            <el-switch v-model="form.advanced.realtimeIdentify" :disabled="!selected.advanced.realtimeIdentify" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.realtimeSend">数据实时发送</el-checkbox>
            <el-switch v-model="form.advanced.realtimeSend" :disabled="!selected.advanced.realtimeSend" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.defaultSite">默认站点</el-checkbox>
            <el-switch v-model="form.advanced.defaultSite" :disabled="!selected.advanced.defaultSite" />
          </div>
          <div class="batch-row">
            <el-checkbox v-model="selected.advanced.l2Config">L2 配置</el-checkbox>
            <el-radio-group v-model="form.advanced.l2Config" :disabled="!selected.advanced.l2Config">
              <el-radio value="current">当前套餐配置</el-radio>
              <el-radio value="none">不配置 L2</el-radio>
              <el-radio value="custom">自定义 L2 配置</el-radio>
            </el-radio-group>
          </div>
        </el-form>
        <div class="section-footer">
          <el-button
            type="primary"
            :disabled="!hasSectionSelection('advanced') || !hasIds"
            :loading="isSubmitting('advanced')"
            @click="submitSection('advanced')"
          >
            批量修改
          </el-button>
        </div>
      </el-collapse-item>
    </el-collapse>

    <CacheRuleDialog
      v-model:visible="cacheRuleDialogVisible"
      :rule="editingCacheRule"
      @save="handleCacheRuleSave"
    />

    <HeaderRuleDialog
      v-model:visible="headerDialogVisible"
      :rule="editingHeaderRule"
      :type="headerRuleType"
      @save="handleHeaderSave"
    />

    <RedirectRuleDialog
      v-model="redirectDialogVisible"
      :rule="editingRedirectRule"
      @submit="handleRedirectSave"
    />

    <el-dialog
      v-model="ruleDialog.visible"
      :title="ruleDialog.mode === 'create' ? '新增规则' : '编辑规则'"
      width="700px"
      append-to-body
      :close-on-click-modal="false"
    >
      <el-form label-width="100px">
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
                <el-select v-model="newMatcher.operator" placeholder="操作符" size="small" style="width: 120px; margin: 0 5px;">
                  <el-option v-for="op in operators" :key="op.value" :label="op.label" :value="op.value" />
                </el-select>
                <el-input v-model="newMatcher.value" placeholder="输入匹配值" size="small" style="width: 200px;" />
              </template>
              <el-button type="primary" size="small" link style="margin-left: 5px;" @click="addMatcher">添加</el-button>
            </div>
          </div>
          <div class="form-helper">多个匹配条件为且关系</div>
        </el-form-item>

        <el-form-item label="执行过滤">
          <div class="action-grid">
            <el-radio-group v-model="ruleForm.action">
              <el-radio v-for="act in actions" :key="act.value" :value="act.value" style="margin-right: 15px; margin-bottom: 5px;">
                {{ act.label }}
              </el-radio>
            </el-radio-group>
          </div>
          <div v-if="ruleForm.action === 'limit_rate'" class="rule-params">
            <el-form-item label="在" label-width="60px" style="margin-bottom: 5px;">
              <el-input v-model.number="ruleForm.actionParams.seconds" size="small"><template #append>秒内</template></el-input>
            </el-form-item>
            <el-form-item label="限制总请求" label-width="100px" style="margin-bottom: 5px;">
              <el-input v-model.number="ruleForm.actionParams.requests" size="small"><template #append>次</template></el-input>
            </el-form-item>
            <el-form-item label="限制同URL" label-width="100px" style="margin-bottom: 0;">
              <el-input v-model.number="ruleForm.actionParams.urlRequests" size="small"><template #append>次</template></el-input>
            </el-form-item>
          </div>
          <div v-if="isVerificationAction(ruleForm.action)" class="rule-params">
            <el-form-item label="是否拉黑" label-width="80px">
              <el-radio-group v-model="ruleForm.actionParams.blockOnFail">
                <el-radio :value="true">拉黑</el-radio>
                <el-radio :value="false">不拉黑</el-radio>
              </el-radio-group>
            </el-form-item>
          </div>
        </el-form-item>

        <el-form-item label="匹配模式">
          <el-radio-group v-model="ruleForm.breakMatch">
            <el-radio :value="false">继续下一条规则</el-radio>
            <el-radio :value="true">停止匹配</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="备注">
          <el-input v-model="ruleForm.remark" placeholder="请输入备注" />
        </el-form-item>

        <el-form-item label="状态">
          <el-switch v-model="ruleForm.on" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="small" @click="ruleDialog.visible = false">取消</el-button>
        <el-button size="small" type="primary" @click="saveRule">确定</el-button>
      </template>
    </el-dialog>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import CacheRuleDialog from '@/components/CacheRuleDialog.vue'
import HeaderRuleDialog from '@/components/HeaderRuleDialog.vue'
import RedirectRuleDialog from '@/components/RedirectRuleDialog.vue'
import CountrySelector from '@/components/CountrySelector.vue'
import {
  normalizeOriginCondition,
  handleOriginConditionChange,
  isOriginHeaderItem,
  isOriginStatItem,
  getOriginConditionPlaceholder,
  splitStr,
  parsePortList,
  getHotlinkPlaceholder,
  normalizeCacheRule,
  getCachePreset
} from '@/utils/siteHelpers'
import { originConditionItems, originConditionOperators, cacheTypeLabelMap } from '@/constants/origin'

const props = defineProps({
  modelValue: Boolean,
  ids: Array
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = ref(false)
const activeSection = ref('basic')
const submittingSection = ref('')

const hasIds = computed(() => Array.isArray(props.ids) && props.ids.length > 0)

const defaultForm = () => ({
  basic: {
    userPackageId: null,
    groupIds: []
  },
  http: {
    enable: true,
    ports: '80'
  },
  https: {
    enable: false,
    certId: null,
    listenPorts: '443',
    force: false,
    forcePort: '443',
    hsts: false,
    http2: false,
    http3: false,
    ocsp: false,
    sslPolicy: 'compat',
    sslCiphers: '',
    sslProtocols: ''
  },
  origin: {
    protocol: 'follow',
    httpPort: '80',
    httpsPort: '443',
    host: 'follow',
    hostValue: '',
    timeout: 60,
    connTimeout: 10,
    balanceWay: '',
    list: [],
    conditions: []
  },
  cache: {
    rules: []
  },
  security: {
    cc: {
      mode: 10002,
      autoSwitch: {
        enable: false,
        qps: 200,
        rule: 'close'
      },
      customRules: []
    },
    crawlers: {
      action: 'none'
    },
    ip: {
      black: '',
      white: '',
      blackTime: 3600,
      blackTimeCustom: false,
      whiteTime: 21600,
      whiteTimeCustom: false
    },
    cookie: {
      enable: false,
      domain: ''
    },
    block: {
      transparentProxy: false
    },
    regions: []
  },
  access: {
    acl: '',
    hotlink: {
      enable: false,
      scope: 'all',
      value: '',
      allowEmpty: true,
      domains: ''
    },
    cors: {
      enable: false,
      allowOrigin: '*',
      allowMethods: '*',
      allowHeaders: '*',
      exposeHeaders: '*',
      allowCredentials: false,
      maxAge: 1728000
    }
  },
  advanced: {
    uploadLimitMode: 'none',
    uploadLimitValue: 102400,
    gzip: false,
    websocket: false,
    searchEngineOrigin: false,
    searchEngineOriginIp: '',
    urlRedirects: [],
    reqHeaders: [],
    resHeaders: [],
    logRequestHeader: false,
    logResponseHeader: false,
    logRequestBody: false,
    logRequestBodySizeLimit: 16,
    originCert: false,
    realtimeIdentify: false,
    realtimeSend: false,
    defaultSite: false,
    l2Config: 'current'
  }
})

const defaultSelected = () => ({
  basic: {
    userPackageId: false,
    groupIds: false
  },
  http: {
    enable: false,
    ports: false
  },
  https: {
    enable: false,
    certId: false,
    listenPorts: false,
    force: false,
    hsts: false,
    http2: false,
    http3: false,
    ocsp: false,
    ssl: false
  },
  origin: {
    protocol: false,
    httpPort: false,
    httpsPort: false,
    host: false,
    timeout: false,
    connTimeout: false,
    balanceWay: false,
    list: false,
    conditions: false
  },
  cache: {
    rules: false
  },
  security: {
    defaultRule: false,
    autoSwitch: false,
    customRules: false,
    crawlers: false,
    blackList: false,
    whiteList: false,
    blackTime: false,
    whiteTime: false,
    cookie: false,
    blockProxy: false,
    regionBlock: false
  },
  access: {
    acl: false,
    hotlink: false,
    cors: false
  },
  advanced: {
    uploadLimit: false,
    gzip: false,
    websocket: false,
    searchEngineOrigin: false,
    urlRedirects: false,
    reqHeaders: false,
    resHeaders: false,
    logRequestHeader: false,
    logResponseHeader: false,
    logRequestBody: false,
    logRequestBodySize: false,
    originCert: false,
    realtimeIdentify: false,
    realtimeSend: false,
    defaultSite: false,
    l2Config: false
  }
})

const form = reactive(defaultForm())
const selected = reactive(defaultSelected())

const loading = reactive({
  userPackages: false,
  siteGroups: false,
  certs: false,
  acls: false,
  ccRules: false
})

const userPackages = ref([])
const siteGroups = ref([])
const certList = ref([])
const aclList = ref([])
const systemRules = ref([])
const userRules = ref([])

const cachePreset = ref('')
const selectedCacheRows = ref([])

const cacheRuleDialogVisible = ref(false)
const editingCacheRule = ref(null)
const editingCacheIndex = ref(-1)

const headerDialogVisible = ref(false)
const headerRuleType = ref('req')
const editingHeaderRule = ref(null)
const editingHeaderIndex = ref(-1)

const redirectDialogVisible = ref(false)
const editingRedirectRule = ref(null)
const editingRedirectIndex = ref(-1)

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

const computedMode = computed({
  get: () => {
    const currentId = form.security.cc.mode
    if (systemRules.value.some(r => r.id === currentId)) {
      return currentId
    }
    return 'custom'
  },
  set: (val) => {
    if (val === 'custom') {
      if (userRules.value.length > 0) {
        const currentIsUser = userRules.value.some(r => r.id === form.security.cc.mode)
        if (!currentIsUser) {
          form.security.cc.mode = userRules.value[0].id
        }
      }
    } else {
      form.security.cc.mode = val
    }
  }
})

const qpsSelection = ref(200)
watch(
  () => form.security.cc.autoSwitch.qps,
  (val) => {
    if ([20, 50, 200].includes(val)) {
      qpsSelection.value = val
    } else {
      qpsSelection.value = 'custom'
    }
  },
  { immediate: true }
)

const handleQpsChange = (val) => {
  if (val !== 'custom') {
    form.security.cc.autoSwitch.qps = val
  }
}

watch(
  () => form.security.ip.blackTimeCustom,
  (val) => {
    if (!val) {
      form.security.ip.blackTime = 3600
    }
  }
)

watch(
  () => form.security.ip.whiteTimeCustom,
  (val) => {
    if (!val) {
      form.security.ip.whiteTime = 21600
    }
  }
)

watch(
  () => props.modelValue,
  (val) => {
    visible.value = val
    if (val) {
      resetForm()
      loadDependencies()
    }
  }
)

watch(
  () => visible.value,
  (val) => {
    emit('update:modelValue', val)
  }
)

const handleClosed = () => {
  resetForm()
}

const resetForm = () => {
  const freshForm = defaultForm()
  Object.keys(freshForm).forEach((key) => {
    form[key] = freshForm[key]
  })
  const freshSelected = defaultSelected()
  Object.keys(freshSelected).forEach((key) => {
    selected[key] = freshSelected[key]
  })
  activeSection.value = 'basic'
  cachePreset.value = ''
  selectedCacheRows.value = []
}

const hasSectionSelection = (section) => {
  const group = selected[section]
  if (!group) return false
  return Object.values(group).some(Boolean)
}

const isSubmitting = (section) => submittingSection.value === section

const loadDependencies = async () => {
  await Promise.all([
    loadUserPackages(),
    loadSiteGroups(),
    loadCerts(),
    loadAcls(),
    loadCcRules()
  ])
}

const loadUserPackages = async () => {
  if (loading.userPackages) return
  loading.userPackages = true
  try {
    const res = await request.get('/user_packages', { params: { pageSize: 1000 } })
    userPackages.value = res.data?.list || res.list || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.userPackages = false
  }
}

const loadSiteGroups = async () => {
  if (loading.siteGroups) return
  loading.siteGroups = true
  try {
    const res = await request.get('/site_groups', { params: { pageSize: 1000 } })
    siteGroups.value = res.data?.list || res.list || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.siteGroups = false
  }
}

const loadCerts = async () => {
  if (loading.certs) return
  loading.certs = true
  try {
    const res = await request.get('/certs', { params: { pageSize: 1000 } })
    certList.value = res.data?.list || res.list || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.certs = false
  }
}

const loadAcls = async () => {
  if (loading.acls) return
  loading.acls = true
  try {
    const res = await request.get('/acls', { baseURL: '/api/v1' })
    aclList.value = res.data?.list || res.list || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.acls = false
  }
}

const loadCcRules = async () => {
  if (loading.ccRules) return
  loading.ccRules = true
  try {
    const res = await request.get('/rules/cc/groups')
    const list = res.data?.list || res.list || []
    systemRules.value = list.filter(item => item.is_system)
    userRules.value = list.filter(item => !item.is_system)
  } catch (e) {
    console.error(e)
  } finally {
    loading.ccRules = false
  }
}

const setSetting = (settings, path, value) => {
  let current = settings
  for (let i = 0; i < path.length - 1; i += 1) {
    const key = path[i]
    if (!current[key] || typeof current[key] !== 'object' || Array.isArray(current[key])) {
      current[key] = {}
    }
    current = current[key]
  }
  current[path[path.length - 1]] = value
}

const buildSectionPayload = (section) => {
  const payload = { ids: props.ids }
  const settings = {}

  if (section === 'basic') {
    if (selected.basic.userPackageId) {
      if (!form.basic.userPackageId) {
        ElMessage.warning('请选择套餐')
        return null
      }
      payload.user_package_id = form.basic.userPackageId
    }
    if (selected.basic.groupIds) {
      payload.group_ids = form.basic.groupIds || []
    }
  }

  if (section === 'http') {
    if (selected.http.enable) {
      settings.http_enable = !!form.http.enable
    }
    if (selected.http.ports) {
      payload.http_listen = parsePortList(form.http.ports)
    }
  }

  if (section === 'https') {
    if (selected.https.enable) {
      setSetting(settings, ['https', 'enable'], !!form.https.enable)
    }
    if (selected.https.certId) {
      payload.cert_id = form.https.certId || 0
    }
    if (selected.https.listenPorts) {
      payload.https_listen = parsePortList(form.https.listenPorts)
      setSetting(settings, ['https', 'listen_port'], form.https.listenPorts || '')
    }
    if (selected.https.force) {
      setSetting(settings, ['https', 'force'], !!form.https.force)
      setSetting(settings, ['https', 'redirect_port'], form.https.forcePort || '443')
    }
    if (selected.https.hsts) {
      setSetting(settings, ['https', 'hsts'], !!form.https.hsts)
    }
    if (selected.https.http2) {
      setSetting(settings, ['https', 'http2'], !!form.https.http2)
    }
    if (selected.https.http3) {
      setSetting(settings, ['https', 'http3'], !!form.https.http3)
    }
    if (selected.https.ocsp) {
      setSetting(settings, ['https', 'ocsp_stapling'], !!form.https.ocsp)
    }
    if (selected.https.ssl) {
      setSetting(settings, ['https', 'ssl_profile'], form.https.sslPolicy)
      setSetting(settings, ['https', 'ssl_protocols'], form.https.sslProtocols || '')
      setSetting(settings, ['https', 'ssl_ciphers'], form.https.sslCiphers || '')
    }
  }

  if (section === 'origin') {
    if (selected.origin.protocol) {
      payload.backend_protocol = form.origin.protocol
      settings.backend_protocol = form.origin.protocol
    }
    if (selected.origin.httpPort) {
      settings.origin_http_port = form.origin.httpPort || '80'
    }
    if (selected.origin.httpsPort) {
      settings.origin_https_port = form.origin.httpsPort || '443'
    }
    if (selected.origin.host) {
      if (form.origin.host === 'custom' && !form.origin.hostValue) {
        ElMessage.warning('请输入自定义回源 HOST')
        return null
      }
      const hostValue = form.origin.host === 'custom' ? form.origin.hostValue : form.origin.host
      settings.origin_host = hostValue
    }
    if (selected.origin.timeout) {
      settings.origin_timeout = Number(form.origin.timeout) || 60
    }
    if (selected.origin.connTimeout) {
      setSetting(settings, ['origin', 'connTimeout'], Number(form.origin.connTimeout) || 10)
    }
    if (selected.origin.balanceWay) {
      payload.balance_way = form.origin.balanceWay || ''
    }
    if (selected.origin.list) {
      const list = (form.origin.list || []).map(item => ({
        address: item.address || '',
        weight: parseInt(item.weight || 10, 10),
        enable: item.enable !== false
      }))
      setSetting(settings, ['origin', 'list'], list)
      payload.backends = list.map(item => item.address).filter(Boolean)
    }
    if (selected.origin.conditions) {
      const conditions = (form.origin.conditions || [])
        .map(item => normalizeOriginCondition(item))
        .filter(Boolean)
      setSetting(settings, ['origin', 'conditions'], conditions)
    }
  }

  if (section === 'cache') {
    if (selected.cache.rules) {
      const rules = (form.cache.rules || []).map(rule => normalizeCacheRule(rule)).filter(Boolean)
      setSetting(settings, ['cache', 'rules'], rules)
    }
  }

  if (section === 'security') {
    if (selected.security.defaultRule) {
      setSetting(settings, ['security', 'default_rule'], form.security.cc.mode)
    }
    if (selected.security.autoSwitch) {
      const autoSwitch = form.security.cc.autoSwitch
      const autoValue = autoSwitch.enable ? JSON.stringify(autoSwitch) : ''
      setSetting(settings, ['security', 'auto_switch'], autoValue)
    }
    if (selected.security.customRules) {
      setSetting(settings, ['security', 'custom_rules'], form.security.cc.customRules || [])
    }
    if (selected.security.crawlers) {
      setSetting(settings, ['security', 'crawlers_action'], form.security.crawlers.action)
    }
    if (selected.security.blackList) {
      setSetting(settings, ['security', 'blacklist'], splitStr(form.security.ip.black))
    }
    if (selected.security.whiteList) {
      setSetting(settings, ['security', 'whitelist'], splitStr(form.security.ip.white))
    }
    if (selected.security.blackTime) {
      setSetting(settings, ['security', 'ip_black_timeout'], Number(form.security.ip.blackTime) || 3600)
    }
    if (selected.security.whiteTime) {
      setSetting(settings, ['security', 'ip_white_timeout'], Number(form.security.ip.whiteTime) || 21600)
    }
    if (selected.security.cookie) {
      setSetting(settings, ['security', 'cookie'], {
        enable: !!form.security.cookie.enable,
        domain: form.security.cookie.domain || ''
      })
    }
    if (selected.security.blockProxy) {
      setSetting(settings, ['security', 'block_transparent_proxy'], !!form.security.block.transparentProxy)
    }
    if (selected.security.regionBlock) {
      setSetting(settings, ['security', 'region_block'], form.security.regions || [])
    }
  }

  if (section === 'access') {
    if (selected.access.acl) {
      setSetting(settings, ['access', 'acl'], form.access.acl || '')
    }
    if (selected.access.hotlink) {
      setSetting(settings, ['access', 'hotlink'], {
        enable: !!form.access.hotlink.enable,
        scope: form.access.hotlink.scope,
        value: form.access.hotlink.value || '',
        allowEmpty: !!form.access.hotlink.allowEmpty,
        domains: form.access.hotlink.domains || ''
      })
    }
    if (selected.access.cors) {
      setSetting(settings, ['access', 'cors'], {
        enable: !!form.access.cors.enable,
        allowOrigin: form.access.cors.allowOrigin || '*',
        allowMethods: form.access.cors.allowMethods || '*',
        allowHeaders: form.access.cors.allowHeaders || '*',
        exposeHeaders: form.access.cors.exposeHeaders || '*',
        allowCredentials: !!form.access.cors.allowCredentials,
        maxAge: form.access.cors.maxAge || 0
      })
    }
  }

  if (section === 'advanced') {
    if (selected.advanced.uploadLimit) {
      const uploadLimit = form.advanced.uploadLimitMode === 'none'
        ? 0
        : Number(form.advanced.uploadLimitValue || 0)
      setSetting(settings, ['advanced', 'body_limit'], uploadLimit)
      setSetting(settings, ['advanced', 'body_limit_unit'], 'kb')
    }
    if (selected.advanced.gzip) {
      settings.gzip = !!form.advanced.gzip
    }
    if (selected.advanced.websocket) {
      settings.websocket = !!form.advanced.websocket
    }
    if (selected.advanced.searchEngineOrigin) {
      settings.search_engine_origin = !!form.advanced.searchEngineOrigin
      settings.search_engine_origin_ip = form.advanced.searchEngineOriginIp || ''
    }
    if (selected.advanced.urlRedirects) {
      settings.url_redirects = form.advanced.urlRedirects || []
    }
    if (selected.advanced.reqHeaders) {
      settings.req_headers = form.advanced.reqHeaders || []
    }
    if (selected.advanced.resHeaders) {
      settings.res_headers = form.advanced.resHeaders || []
    }
    if (selected.advanced.logRequestHeader) {
      settings.log_request_header = !!form.advanced.logRequestHeader
    }
    if (selected.advanced.logResponseHeader) {
      settings.log_response_header = !!form.advanced.logResponseHeader
    }
    if (selected.advanced.logRequestBody) {
      settings.log_request_body = !!form.advanced.logRequestBody
    }
    if (selected.advanced.logRequestBodySize) {
      settings.log_request_body_size_limit = Number(form.advanced.logRequestBodySizeLimit || 16)
    }
    if (selected.advanced.originCert) {
      settings.origin_cert = !!form.advanced.originCert
    }
    if (selected.advanced.realtimeIdentify) {
      settings.realtime_identify = !!form.advanced.realtimeIdentify
    }
    if (selected.advanced.realtimeSend) {
      settings.realtime_send = !!form.advanced.realtimeSend
    }
    if (selected.advanced.defaultSite) {
      settings.default_site = !!form.advanced.defaultSite
    }
    if (selected.advanced.l2Config) {
      settings.l2_config = form.advanced.l2Config
    }
  }

  if (Object.keys(settings).length > 0) {
    payload.settings = settings
  }

  if (!payload.settings && Object.keys(payload).length === 1) {
    ElMessage.warning('请勾选要修改的项')
    return null
  }
  return payload
}

const submitSection = async (section) => {
  if (!hasIds.value) {
    ElMessage.warning('请先选择站点')
    return
  }
  if (!hasSectionSelection(section)) {
    ElMessage.warning('请勾选要修改的项')
    return
  }
  const payload = buildSectionPayload(section)
  if (!payload) return

  submittingSection.value = section
  try {
    await request.post('/sites/batch_update', payload)
    ElMessage.success('批量修改成功')
    emit('success', { section })
  } catch (e) {
    console.error(e)
  } finally {
    submittingSection.value = ''
  }
}

const addOrigin = () => {
  form.origin.list.push({ address: '', weight: '10', enable: true })
}

const removeOrigin = (index) => {
  form.origin.list.splice(index, 1)
}

const addConditionOrigin = () => {
  form.origin.conditions.push({
    item: 'uri',
    operator: 'eq',
    value: '',
    origin: '',
    header: '',
    seconds: ''
  })
}

const removeConditionOrigin = (index) => {
  form.origin.conditions.splice(index, 1)
}

const openCacheRuleDialog = (rule = null, index = -1) => {
  editingCacheRule.value = rule
  editingCacheIndex.value = index
  cacheRuleDialogVisible.value = true
}

const handleCacheRuleSave = (rule) => {
  const normalized = normalizeCacheRule(rule)
  if (!normalized) return
  if (editingCacheIndex.value >= 0) {
    form.cache.rules.splice(editingCacheIndex.value, 1, normalized)
  } else {
    form.cache.rules.push(normalized)
  }
  editingCacheRule.value = null
  editingCacheIndex.value = -1
}

const removeCacheRule = (index) => {
  form.cache.rules.splice(index, 1)
}

const handleCacheSelection = (rows) => {
  selectedCacheRows.value = rows
}

const removeCacheRulesBatch = () => {
  if (!selectedCacheRows.value.length) return
  const toRemove = new Set(selectedCacheRows.value)
  form.cache.rules = form.cache.rules.filter(rule => !toRemove.has(rule))
  selectedCacheRows.value = []
}

const applyCachePreset = (val) => {
  if (!val) return
  const preset = getCachePreset(val)
  if (preset) {
    const rule = normalizeCacheRule(preset)
    if (rule) {
      form.cache.rules.push(rule)
    }
  }
  cachePreset.value = ''
}

const openHeaderDialog = (type, rule = null, index = -1) => {
  headerRuleType.value = type
  editingHeaderRule.value = rule
  editingHeaderIndex.value = index
  headerDialogVisible.value = true
}

const handleHeaderSave = (rule) => {
  const list = headerRuleType.value === 'req' ? form.advanced.reqHeaders : form.advanced.resHeaders
  if (editingHeaderIndex.value >= 0) {
    list.splice(editingHeaderIndex.value, 1, rule)
  } else {
    list.push(rule)
  }
  editingHeaderRule.value = null
  editingHeaderIndex.value = -1
}

const removeHeader = (type, index) => {
  const list = type === 'req' ? form.advanced.reqHeaders : form.advanced.resHeaders
  list.splice(index, 1)
}

const openRedirectDialog = (rule = null, index = -1) => {
  editingRedirectRule.value = rule
  editingRedirectIndex.value = index
  redirectDialogVisible.value = true
}

const handleRedirectSave = (ruleData) => {
  if (editingRedirectIndex.value >= 0) {
    form.advanced.urlRedirects.splice(editingRedirectIndex.value, 1, {
      ...form.advanced.urlRedirects[editingRedirectIndex.value],
      ...ruleData
    })
  } else {
    form.advanced.urlRedirects.push({
      domain: '',
      ...ruleData
    })
  }
  editingRedirectRule.value = null
  editingRedirectIndex.value = -1
}

const removeRedirect = (index) => {
  form.advanced.urlRedirects.splice(index, 1)
}

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
    const rule = form.security.cc.customRules[index]
    Object.assign(ruleForm, JSON.parse(JSON.stringify(rule)))
    if (!ruleForm.actionParams) {
      ruleForm.actionParams = { seconds: 10, requests: 10, urlRequests: 10, blockOnFail: true }
    }
  }
}

const addMatcher = () => {
  if (!newMatcher.key) return
  if (newMatcher.key !== 'all' && !newMatcher.value && !['exists', 'not_exists'].includes(newMatcher.operator)) {
    ElMessage.warning('请输入匹配值')
    return
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
  if (ruleDialog.mode === 'create') {
    form.security.cc.customRules.push(newRule)
  } else {
    form.security.cc.customRules[ruleDialog.index] = newRule
  }
  ruleDialog.visible = false
}

const deleteRule = (idx) => {
  form.security.cc.customRules.splice(idx, 1)
}

const editRule = (idx) => {
  openRuleDialog('update', idx)
}

const toggleAllRules = (enable) => {
  form.security.cc.customRules.forEach(r => {
    r.on = enable
  })
}

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
  if (props.modelValue) {
    resetForm()
    loadDependencies()
  }
})
</script>

<style scoped>
.batch-tip {
  margin-bottom: 12px;
}

.section-form {
  padding: 8px 4px 16px;
}

.batch-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.batch-label {
  min-width: 80px;
  color: #606266;
}

.inline-field {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.inline-text {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin: 12px 0;
  padding-left: 10px;
  border-left: 3px solid #409eff;
}

.section-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.toolbar-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.divider {
  height: 1px;
  background-color: #ebeef5;
  margin: 20px 0;
}

.condition-origin-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.matcher-config {
  border: 1px solid #dcdfe6;
  padding: 10px;
  border-radius: 4px;
}

.matcher-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 5px;
  background: #f5f7fa;
  padding: 5px 10px;
  border-radius: 4px;
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
  background: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
}

.disabled-block {
  pointer-events: none;
  opacity: 0.6;
}

.form-helper {
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
  margin-top: 6px;
}
</style>
