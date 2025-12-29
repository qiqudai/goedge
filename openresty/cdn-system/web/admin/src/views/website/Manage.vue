<template>
  <div class="site-manage">
    <el-page-header v-if="site" @back="goBack" class="page-header" content="网站配置" style="margin-bottom: 16px;">
      <template #title>
        <span>{{ site.domains?.[0] || site.domain_raw || '网站' }}</span>
      </template>
      <template #content>
        <span>ID {{ site.id }} · {{ siteSettings.basic.cname }}</span>
      </template>
    </el-page-header>

    <el-card class="page-card" v-loading="loading">
      <el-tabs v-model="activeTab" class="manage-tabs" type="border-card">
        <el-tab-pane label="基本配置" name="basic">
          <div class="section-title">基本设置</div>
          <el-form label-width="120px" class="config-form">
            <el-form-item label="状态">
               <span class="status-dot" :class="{ active: siteSettings.basic.status }"></span>
               {{ siteSettings.basic.status ? '正常' : '已停用' }}
            </el-form-item>
            <el-form-item label="CNAME">
               {{ siteSettings.basic.cname }}
            </el-form-item>
            <el-form-item label="企业到期">
               {{ siteSettings.basic.expireTime || '-' }}
            </el-form-item>
             <el-form-item label="创建时间">
               {{ siteSettings.basic.createdAt }}
            </el-form-item>
            <el-form-item label="更新时间">
               {{ siteSettings.basic.updatedAt }}
            </el-form-item>

            <div class="divider"></div>

            <div class="section-title">基本设置</div>
             <el-form-item label="套餐">
               <el-select v-model="siteSettings.basic.planName" disabled placeholder="请选择套餐">
                   <el-option value="请选择套餐" label="请选择套餐" />
               </el-select>
               <div class="form-helper">变更套餐不会导致CNAME地址变动，只会应用新的套餐权益</div>
            </el-form-item>
             <el-form-item label="所属分组">
               <el-select v-model="siteSettings.basic.groupName" placeholder="请选择">
                 <!-- TODO: Load groups -->
               </el-select>
               <div class="form-helper">网站的分组标识，方便为了分类和管理</div>
            </el-form-item>
             <el-form-item label="地区">
               <el-input v-model="siteSettings.basic.regionName" disabled />
               <div class="form-helper">本个域名分配的地区，中文域名会自动转为Punycode。 <a href="#" style="color: #409eff">查看节点</a></div>
            </el-form-item>

            <div class="divider"></div>

            <div class="section-title">HTTP设置</div>
            <el-form-item label="开关">
              <el-switch v-model="siteSettings.basic.httpEnable" />
              <div class="form-helper">如果关闭，网站将完全拒绝HTTP访问</div>
            </el-form-item>
             <el-form-item label="监听端口">
              <el-input v-model="siteSettings.basic.httpPorts" />
              <div class="form-helper">多个端口空格分隔。如需兼容http://www.example.com和http://www.example.com:888访问，则填80 888</div>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="回源设置" name="origin">
          <div class="section-title">源站列表</div>
          <el-table :data="siteSettings.origin.list" border size="small" style="margin-bottom: 12px;">
            <el-table-column prop="address" label="源地址">
              <template #default="{ row }">
                <el-input v-model="row.address" placeholder="IP 或域名" size="small" />
              </template>
            </el-table-column>
            <el-table-column prop="weight" label="权重" width="120">
              <template #default="{ row }">
                <el-input v-model="row.weight" size="small" />
              </template>
            </el-table-column>
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-switch v-model="row.enable" active-text="启用" inactive-text="停用" size="small" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ $index }">
                <el-button link type="danger" size="small" @click="removeOrigin($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button size="small" type="primary" @click="addOrigin">新增源站</el-button>

          <el-divider />
          <div class="section-title">条件源站</div>
          <el-table :data="siteSettings.origin.conditions" border size="small" style="margin-bottom: 12px;">
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
                  />
                  <el-input
                    v-else-if="isOriginStatItem(row.item)"
                    v-model="row.seconds"
                    size="small"
                    placeholder="统计秒数"
                  />
                  <el-input
                    v-else
                    v-model="row.value"
                    size="small"
                    :placeholder="getOriginConditionPlaceholder(row)"
                  />
                  <el-select
                    v-if="!isOriginStatItem(row.item)"
                    v-model="row.operator"
                    size="small"
                    placeholder="匹配方式"
                    style="width: 140px;"
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
                <el-input v-model="row.origin" placeholder="源站地址，多个用 | 分隔" size="small" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ $index }">
                <el-button link type="danger" size="small" @click="removeConditionOrigin($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button size="small" type="primary" @click="addConditionOrigin">新增条件源站</el-button>

          <el-divider />
          <div class="section-title">回源健康检查</div>
          <el-form label-width="150px" class="config-form">
            <el-form-item label="启用健康检查">
              <el-switch v-model="siteSettings.origin.healthCheckEnabled" />
            </el-form-item>
            <el-form-item label="检查地址">
              <el-input v-model="siteSettings.origin.healthCheckHost" placeholder="域名或 IP" />
            </el-form-item>
            <el-form-item label="检查路径">
              <el-input v-model="siteSettings.origin.healthCheckPath" placeholder="/" />
            </el-form-item>
            <el-form-item label="有效状态码">
              <el-input v-model="siteSettings.origin.healthCheckStatus" placeholder="200 301 302" />
            </el-form-item>
            <el-form-item label="检测间隔(秒)">
              <el-input v-model="siteSettings.origin.healthCheckInterval" placeholder="60" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="HTTPS配置" name="https">
           <div class="section-title">HTTPS证书</div>
           <el-form label-width="120px" class="config-form">
             <el-form-item label="开关">
                <el-switch v-model="siteSettings.https.enable" />
             </el-form-item>
              <template v-if="siteSettings.https.enable">
                 <el-form-item label="证书选择">
                    <el-select v-model="siteSettings.https.certId" placeholder="请选择证书" style="width: 100%">
                         <el-option v-for="cert in certList" :key="cert.id" :label="cert.name" :value="cert.id">
                            <span style="float: left">{{ cert.name }}</span>
                            <span style="float: right; color: #8492a6; font-size: 13px">{{ cert.domains }}</span>
                         </el-option>
                    </el-select>
                    <div class="form-helper" v-if="siteSettings.https.certId">
                        <span class="status-dot active"></span> 有效期剩余 {{ getCertDays(siteSettings.https.certId) }} 天
                    </div>
                    <div class="form-helper" v-else>请选择或上传证书</div>
                 </el-form-item>
                 <el-form-item label="监听端口">
                    <el-input v-model="siteSettings.https.listenPorts" placeholder="443" />
                    <div class="form-helper">多个端口空格分隔。如果需要https://www.example.com和https://www.example.com:8433访问，则填443 8433</div>
                 </el-form-item>
                 
                 <div class="divider"></div>
                 
                 <div class="section-title">强制HTTPS</div>
                 <el-form-item label="开关">
                    <el-switch v-model="siteSettings.https.force" />
                    <div class="form-helper">开启后，访问http将会301跳转到https</div>
                 </el-form-item>
                 <el-form-item label="跳转端口" v-if="siteSettings.https.force">
                    <el-select v-model="siteSettings.https.forcePort" placeholder="443">
                        <el-option label="443" value="443" />
                         <!-- TODO: Dynamic ports if needed -->
                    </el-select>
                    <div class="form-helper">如果https监听有多个端口，可以择其一个跳转</div>
                 </el-form-item>

                 <div class="divider"></div>

                 <div class="section-title">HSTS</div>
                 <el-form-item label="开关">
                    <el-switch v-model="siteSettings.https.hsts" />
                    <div class="form-helper">开启后，访问使用浏览器访问http时，将不用请求服务器直接转向https，这可以减少http会话劫持风险</div>
                 </el-form-item>

                 <div class="divider"></div>

                 <div class="section-title">HTTP2设置</div>
                 <el-form-item label="开关">
                    <el-switch v-model="siteSettings.https.http2" />
                    <div class="form-helper">HTTP2.0协议是HTTP1.1协议的升级版本，在Web数据交互性能上具备更多的优势，开启前您需要先配置HTTPS证书。</div>
                 </el-form-item>

                 <div class="divider"></div>

                 <div class="section-title">OCSP Stapling</div>
                 <el-form-item label="开关">
                    <el-switch v-model="siteSettings.https.ocsp" />
                    <div class="form-helper">OCSP Stapling功能可实现由CDN预先缓存在线证书验证结果并下发给客户端，无需浏览器直接向CA站点查询证书状态，从而减少用户验证时间。</div>
                 </el-form-item>

                 <div class="divider"></div>

                 <div class="section-title">HTTP3设置</div>
                 <el-form-item label="开关">
                    <el-switch v-model="siteSettings.https.http3" />
                 </el-form-item>

                 <div class="divider"></div>

                 <div class="section-title">SSL配置</div>
                 <el-form-item label="SSL配置">
                     <el-radio-group v-model="siteSettings.https.sslPolicy">
                         <el-radio value="compat">兼容旧浏览器（安全性降低）</el-radio>
                         <el-radio value="modern">兼容大部分浏览器（更安全）</el-radio>
                         <el-radio value="custom">自定义</el-radio>
                     </el-radio-group>
                 </el-form-item>
              </template>
           </el-form>
        </el-tab-pane>

        <el-tab-pane label="安全设置" name="security">
           <div class="section-title">CC 防护</div>
           <el-form label-width="120px" class="config-form">
              <el-form-item label="默认规则">
                 <el-radio-group v-model="siteSettings.security.cc.mode">
                     <el-radio :value="10002">关闭</el-radio>
                     <el-radio :value="10003">宽松</el-radio>
                     <el-radio :value="10004">普通</el-radio>
                     <el-radio :value="10005">严格</el-radio>
                     <el-radio :value="10006">JS验证</el-radio>
                     <el-radio :value="10008">验证码</el-radio>
                     <!-- <el-radio :value="10009">自定义</el-radio> -->
                 </el-radio-group>
                 <div class="form-helper">不同模式对应不同的防御级别</div>
              </el-form-item>
              <el-form-item label="自动切换">
                  <div style="display: flex; align-items: center; gap: 10px;">
                      <el-switch v-model="siteSettings.security.cc.autoSwitch.enable" />
                      <span v-if="siteSettings.security.cc.autoSwitch.enable" style="font-size: 13px;">
                          当QPS超过 <el-input v-model="siteSettings.security.cc.autoSwitch.qps" size="small" style="width: 80px" /> 时，
                          自动切换到 <el-select v-model="siteSettings.security.cc.autoSwitch.rule" size="small" style="width: 100px;">
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
              
              <div class="divider"></div>
              
              <div class="section-title">自定义规则</div>
              <el-table :data="siteSettings.security.customRules" border size="small" style="margin-bottom: 12px;">
                 <el-table-column label="规则名称" prop="name" />
                 <el-table-column label="匹配条件" prop="condition" />
                 <el-table-column label="执行动作" prop="action" />
                 <el-table-column label="操作" width="100">
                    <template #default="{ $index }">
                        <el-button link type="danger" size="small">删除</el-button>
                    </template>
                 </el-table-column>
              </el-table>
              <el-button size="small" type="primary">新增规则</el-button>

              <div class="divider"></div>
              
              <div class="section-title">黑白名单</div>
              <el-form-item label="IP黑名单">
                  <el-input type="textarea" v-model="siteSettings.security.ip.black" :rows="3" placeholder="一行一个IP" />
              </el-form-item>
              <el-form-item label="IP白名单">
                   <el-input type="textarea" v-model="siteSettings.security.ip.white" :rows="3" placeholder="一行一个IP" />
              </el-form-item>

               <div class="divider"></div>

               <div class="section-title">UA黑白名单</div>
               <el-form-item label="UA黑名单">
                   <el-input type="textarea" v-model="siteSettings.security.ua.black" :rows="3" placeholder="一行一个UA keyword" />
               </el-form-item>
               <el-form-item label="UA白名单">
                    <el-input type="textarea" v-model="siteSettings.security.ua.white" :rows="3" placeholder="一行一个UA keyword" />
               </el-form-item>
               
               <div class="divider"></div>
               
               <div class="section-title">Cookie设置</div>
               <el-form-item label="开关">
                  <el-switch v-model="siteSettings.security.cookie.enable" />
               </el-form-item>
               <el-form-item label="作用域" v-if="siteSettings.security.cookie.enable">
                   <el-input v-model="siteSettings.security.cookie.domain" placeholder="留空则默认为当前域名" />
               </el-form-item>

               <div class="divider"></div>
               
               <div class="section-title">区域屏蔽</div>
               <el-form-item label="区域选择">
                   <CountrySelector v-model="siteSettings.security.regions" />
               </el-form-item>

           </el-form>
        </el-tab-pane>
        <el-tab-pane label="缓存设置" name="cache">
          <div class="toolbar-row" style="margin-bottom: 12px;">
            <el-button type="primary" size="small" @click="openCacheRuleDialog('create')">新增规则</el-button>
            <el-select
              v-model="cacheQuickPreset"
              placeholder="快速添加缓存"
              size="small"
              style="width: 150px; margin-left: 12px;"
              @change="applyCachePreset"
            >
              <el-option label="首页缓存" value="index" />
              <el-option label="全站缓存" value="all" />
              <el-option label="静态资源缓存" value="static" />
              <el-option label="视频资源" value="video" />
              <el-option label="Wordpress 缓存" value="wordpress" />
            </el-select>
          </div>
          <el-table :data="siteSettings.cache.rules" border size="small">
            <el-table-column label="类型" min-width="120">
              <template #default="{ row }">{{ cacheTypeLabel(row.type) }}</template>
            </el-table-column>
            <el-table-column label="内容" min-width="240" prop="value" />
            <el-table-column label="TTL(秒)" width="120" prop="ttl" />
            <el-table-column label="操作" width="140">
              <template #default="{ row, $index }">
                <el-button link type="primary" size="small" @click="openCacheRuleDialog('edit', row, $index)">编辑</el-button>
                <el-button link type="danger" size="small" @click="removeCacheRule($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="访问控制" name="access">
          <el-form label-width="150px" class="config-form">
            <div class="section-title">ACL设置</div>
            <el-form-item label="ACL选择">
              <el-select v-model="siteSettings.access.acl" placeholder="请选择" style="width: 100%" clearable>
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
              <el-switch v-model="siteSettings.access.hotlink.enable" />
            </el-form-item>
            <template v-if="siteSettings.access.hotlink.enable">
              <el-form-item label="防盗链范围">
                <div style="display: flex; align-items: center; gap: 10px; flex-wrap: wrap;">
                  <el-radio-group v-model="siteSettings.access.hotlink.scope">
                    <el-radio value="all">整站</el-radio>
                    <el-radio value="suffix">后缀</el-radio>
                    <el-radio value="dir">目录</el-radio>
                    <el-radio value="path">单个路径</el-radio>
                  </el-radio-group>
                  <el-input
                    v-if="siteSettings.access.hotlink.scope !== 'all'"
                    v-model="siteSettings.access.hotlink.value"
                    style="width: 300px;"
                    :placeholder="getHotlinkPlaceholder()"
                  />
                </div>
              </el-form-item>
              <el-form-item label="允许空来源">
                <el-radio-group v-model="siteSettings.access.hotlink.allowEmpty">
                  <el-radio :value="true">允许</el-radio>
                  <el-radio :value="false">不允许</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="额外允许域名">
                <el-input v-model="siteSettings.access.hotlink.domains" placeholder="请输入除当前网站域名之外的域名 多个域名空格分隔" />
              </el-form-item>
            </template>

            <div class="divider"></div>

            <div class="section-title">跨域访问设置</div>
            <el-form-item label="开关">
              <el-switch v-model="siteSettings.access.cors.enable" />
            </el-form-item>
            <template v-if="siteSettings.access.cors.enable">
              <div class="cors-more-toggle" @click="corsExpanded = !corsExpanded">
                <span>{{ corsExpanded ? '▼ 收起更多设置' : '▶ 查看更多设置' }}</span>
              </div>
              
              <div v-show="corsExpanded">
                <el-form-item label="allow_origin">
                  <el-input v-model="siteSettings.access.cors.allowOrigin" />
                </el-form-item>
                <el-form-item label="allow_methods">
                  <el-input v-model="siteSettings.access.cors.allowMethods" />
                </el-form-item>
                <el-form-item label="allow_headers">
                  <el-input v-model="siteSettings.access.cors.allowHeaders" />
                </el-form-item>
                <el-form-item label="expose_headers">
                  <el-input v-model="siteSettings.access.cors.exposeHeaders" />
                </el-form-item>
                <el-form-item label="allow_credentials">
                  <el-radio-group v-model="siteSettings.access.cors.allowCredentials">
                    <el-radio :value="true">允许</el-radio>
                    <el-radio :value="false">不允许</el-radio>
                  </el-radio-group>
                </el-form-item>
                <el-form-item label="max_age">
                  <el-input v-model="siteSettings.access.cors.maxAge" />
                </el-form-item>
              </div>
            </template>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="高级设置" name="advanced">
           <el-form label-width="150px" class="config-form">
               <div class="section-title">压缩设置</div>
               <el-form-item label="Gzip压缩">
                   <el-switch v-model="siteSettings.advanced.gzip" />
               </el-form-item>

               <div class="divider"></div>

               <div class="section-title">Websocket设置</div>
               <el-form-item label="Websocket">
                   <el-switch v-model="siteSettings.advanced.websocket" />
               </el-form-item>

               <div class="divider"></div>

               <div class="section-title">搜索引擎回源配置</div>
               <el-form-item label="开关">
                   <el-switch v-model="siteSettings.advanced.searchEngineOrigin" />
               </el-form-item>

               <div class="divider"></div>

               <div class="section-title">URL重写设置</div>
               <el-button type="primary" size="small" style="margin-bottom: 12px;" @click="openRewriteDialog('create')">新增重写</el-button>
               <el-table :data="siteSettings.advanced.urlRewrites" border size="small">
                   <el-table-column label="匹配URI" prop="match" />
                   <el-table-column label="重写到" prop="replace" />
                   <el-table-column label="代码" prop="code" width="80" />
                   <el-table-column label="操作" width="100">
                       <template #default="{ $index }">
                           <el-button link type="danger" size="small" @click="removeRewrite($index)">删除</el-button>
                       </template>
                   </el-table-column>
               </el-table>

               <div class="divider"></div>
               
               <div class="section-title">公共请求头设置</div>
               <el-button type="primary" size="small" style="margin-bottom: 12px;" @click="openHeaderDialog('req', 'create')">新增请求头</el-button>
                <el-table :data="siteSettings.advanced.reqHeaders" border size="small">
                   <el-table-column label="名称" prop="name" />
                   <el-table-column label="值" prop="value" />
                   <el-table-column label="操作" width="100">
                       <template #default="{ $index }">
                           <el-button link type="danger" size="small" @click="removeHeader('req', $index)">删除</el-button>
                       </template>
                   </el-table-column>
               </el-table>

               <div class="divider"></div>

               <div class="section-title">CDN响应头设置</div>
               <el-button type="primary" size="small" style="margin-bottom: 12px;" @click="openHeaderDialog('res', 'create')">新增响应头</el-button>
               <el-table :data="siteSettings.advanced.resHeaders" border size="small">
                   <el-table-column label="名称" prop="name" />
                   <el-table-column label="值" prop="value" />
                   <el-table-column label="操作" width="100">
                       <template #default="{ $index }">
                           <el-button link type="danger" size="small" @click="removeHeader('res', $index)">删除</el-button>
                       </template>
                   </el-table-column>
               </el-table>

               <div class="divider"></div>
               
               <div class="section-title">其它</div>
               <el-form-item label="源站证书">
                   <el-switch v-model="siteSettings.advanced.originCert" />
                   <div class="form-helper">用于回源连接（HTTPS）验证源站证书</div>
               </el-form-item>
               <el-form-item label="数据实时鉴别">
                   <el-switch v-model="siteSettings.advanced.realtimeIdentify" />
               </el-form-item>
               <el-form-item label="数据实时发送">
                   <el-switch v-model="siteSettings.advanced.realtimeSend" />
               </el-form-item>

           </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog
      v-model="cacheRuleDialog.visible"
      :title="cacheRuleDialog.mode === 'edit' ? '编辑缓存规则' : '新增缓存规则'"
      width="520px"
    >
      <el-form label-width="120px">
        <el-form-item label="类型">
          <el-select v-model="cacheRuleForm.type">
            <el-option label="首页" value="index" />
            <el-option label="全站" value="all" />
            <el-option label="目录" value="dir" />
            <el-option label="后缀" value="suffix" />
            <el-option label="路径" value="path" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="cacheRuleForm.value" placeholder="支持正则或路径" />
        </el-form-item>
        <el-form-item label="TTL">
          <el-input v-model="cacheRuleForm.ttl" placeholder="单位：秒" />
        </el-form-item>
        <el-form-item label="忽略参数">
          <el-switch v-model="cacheRuleForm.ignore_query" />
        </el-form-item>
        <el-form-item label="强制缓存">
          <el-switch v-model="cacheRuleForm.force_cache" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="small" @click="cacheRuleDialog.visible = false">取消</el-button>
        <el-button size="small" type="primary" @click="saveCacheRule">保存规则</el-button>
      </template>
    </el-dialog>

    <!-- URL Rewrite Dialog -->
    <el-dialog v-model="rewriteDialog.visible" title="新增转向" width="500px">
        <el-form label-width="100px">
            <el-form-item label="匹配URI">
                <el-input v-model="rewriteForm.match" placeholder="(.*)" />
            </el-form-item>
            <el-form-item label="重写到">
                <el-input v-model="rewriteForm.replace" placeholder="https://www.baidu.com$1" />
            </el-form-item>
            <el-form-item label="响应码">
                 <el-select v-model="rewriteForm.code">
                     <el-option value="301" label="301 (永久移动)" />
                     <el-option value="302" label="302 (临时移动)" />
                     <el-option value="307" label="307 (临时重定向)" />
                     <!-- <el-option value="internal" label="内部" /> -->
                 </el-select>
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="rewriteDialog.visible = false">取消</el-button>
            <el-button type="primary" @click="saveRewrite">确定</el-button>
        </template>
    </el-dialog>

    <!-- Header Dialog -->
     <el-dialog v-model="headerDialog.visible" :title="headerDialog.type === 'req' ? '新增请求头' : '新增响应头'" width="500px">
        <el-form label-width="100px">
            <el-form-item label="名称">
                <el-input v-model="headerForm.name" placeholder="Header-Name" />
            </el-form-item>
            <el-form-item label="值">
                <el-input v-model="headerForm.value" placeholder="Value" />
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="headerDialog.visible = false">取消</el-button>
            <el-button type="primary" @click="saveHeader">确定</el-button>
        </template>
    </el-dialog>
  </div>
</template>
<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import CountrySelector from '@/components/CountrySelector.vue'
import request from '@/utils/request'
import { debounce } from 'lodash-es'

const route = useRoute()
const router = useRouter()
const activeTab = ref('basic')
const loading = ref(false)
const cacheQuickPreset = ref('')
const certList = ref([])
const site = ref(null)
const sslProtocolOptions = ['SSLv2', 'SSLv3', 'TLSv1', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3']
const defaultSslProtocols = ['TLSv1', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3']
const defaultSslCiphers = [
  'ECDHE-ECDSA-AES128-GCM-SHA256',
  'ECDHE-RSA-AES128-GCM-SHA256',
  'ECDHE-ECDSA-AES256-GCM-SHA384',
  'ECDHE-RSA-AES256-GCM-SHA384',
  'ECDHE-ECDSA-CHACHA20-POLY1305',
  'ECDHE-RSA-CHACHA20-POLY1305'
].join(':')


// Initialize with new structure
const siteSettings = reactive({
  basic: {
    planName: '-',
    groupName: '-', 
    nodeGroupName: '-',
    domain: '',
    cname: '',
    status: true,
    createdAt: '-',
    updatedAt: '-',
    httpEnable: true,
    httpPorts: '80',
    regionName: 'Global',
    expireTime: '-' // Added
  },
  origin: {
    list: [],
    conditions: [],
    protocol: 'follow', // http, https, follow, follow_port
    host: 'follow', // custom, follow
    hostValue: '',
    timeout: 60,
    connTimeout: 10,
    balanceWay: 'rr',
    healthCheckEnabled: true, // Keep
    healthCheckHost: '',
    healthCheckPath: '/',
    healthCheckStatus: '200 301 302',
    healthCheckInterval: 60
  },
  https: {
    enable: false,
    listenPorts: '443',
    certId: null,
    force: false,
    forcePort: '443',
    hsts: false,
    http2: false,
    http3: false,
    ocsp: false,
    sslPolicy: 'compat'
  },
  security: {
    cc: {
      mode: 10002,
      autoSwitch: {
        enable: false,
        qps: 200,
        rule: 'close'
      }
    },
    customRules: [],
    ip: { white: '', black: '' },
    ua: { white: '', black: '' },
    cookie: { enable: false, domain: '' },
    regions: []
  },
  cache: { rules: [] },
  access: {
    acl: '',
    hotlink: {
      enable: false,
      scope: 'all',
      value: '',
      allowEmpty: false,
      domains: ''
    },
    cors: {
      enable: false,
      allowOrigin: '*',
      allowMethods: '*',
      allowHeaders: '*',
      exposeHeaders: '*',
      allowCredentials: true,
      maxAge: 1728000
    }
  },
  advanced: {
      gzip: { enable: false, level: 1, minLength: '1k' }, // Updated structure if needed, or keep simple bool if screenshot implies simple switch. Screenshot shows simple switch.
      // Let's stick to simple props matching the UI first, allow complex if backend data requires it.
      // Screenshot: Gzip switch, Websocket switch, SearchEngine switch
      // URL Rewrite list
      // Req Header list
      // Resp Header list
      // Others: Origin Cert switch, Realtime Data switch
      
      gzip: false,
      websocket: false,
      searchEngineOrigin: false, // 搜索引擎回源
      
      urlRewrites: [], // List of { match, replace, code, ... }
      reqHeaders: [], // List of { name, value, op }
      resHeaders: [], // List of { name, value, op }
      
      originCert: false, // 源站证书
      realtimeIdentify: false, // 数据实时鉴别
      realtimeSend: false // 数据实时发送
  }
})



const isSaving = ref(false)

const triggerSave = debounce(() => {
    saveSettings()
}, 1000)

// Watch settings deeper for auto-save
watch(siteSettings, (newVal) => {
    // Special logic for Security Auto Switch
    if (activeTab.value === 'security') {
        const autoSwitch = newVal.security.cc.autoSwitch
        if (autoSwitch.enable && (!autoSwitch.qps || !autoSwitch.rule)) {
            // Don't save if invalid
            return
        }
    }
    triggerSave()
}, { deep: true })

const cacheRuleDialog = reactive({ visible: false, mode: 'create', index: -1 })
const rewriteDialog = reactive({ visible: false, index: -1 })
const headerDialog = reactive({ visible: false, type: 'req', index: -1 }) // type: req | res
const corsExpanded = ref(false)

const cacheRuleForm = reactive({ type: 'index', value: '', ttl: '86400', ignore_query: false, force_cache: false })
const rewriteForm = reactive({ match: '', replace: '', code: '301' })
const headerForm = reactive({ name: '', value: '' })

const siteId = computed(() => parseInt(route.query.site_id || route.params.site_id || 0, 10))

const ccRules = ref([
  { label: '关闭', value: 10002 },
  { label: '宽松', value: 10003 },
  { label: 'JS 验证', value: 10004 },
  { label: '5 秒盾', value: 10005 },
  { label: '点击验证', value: 10006 },
  { label: '滑块验证', value: 10007 },
  { label: '验证码', value: 10008 }
])
const cacheTypeLabelMap = { index: '首页', all: '全站', dir: '目录', suffix: '后缀', path: '路径' }
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
  { label: '用户 IP', value: 'client_ip' },
  { label: '域名', value: 'domain' },
  { label: '请求头', value: 'header' },
  { label: '请求方法', value: 'method' },
  { label: 'HTTP 版本', value: 'http_version' },
  { label: '独立 UA 数量', value: 'ua_count' },
  { label: '404 状态码数量', value: 'status_404' }
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
  { label: '在 IP 段', value: 'in_ip' },
  { label: '不在 IP 段', value: 'not_in_ip' }
]

function createDefaultSettings() {
  return {
    basic: {
      planName: '-',
      groupName: '-',
      nodeGroupName: '-',
      domain: '',
      cname: '-',
      status: true,
      createdAt: '-',
      updatedAt: '-'
    },
    origin: {
      list: [],
      conditions: [],
      healthCheckEnabled: true,
      healthCheckHost: '',
      healthCheckPath: '/',
      healthCheckStatus: '200 301 302',
      healthCheckInterval: 60
    },
    https: {
      enabled: true,
      port: '443',
      certId: null,
      force: false,
      redirectPort: '443',
      hsts: false,
      http2: false,
      http3: false,
      ocspStapling: false,
      sslProfile: 'compat',
      sslProtocols: [...defaultSslProtocols],
      sslCiphers: defaultSslCiphers,
      sslPreferServerCiphers: true
    },
    cache: {
      rules: []
    },
    security: {
      defaultRule: 10002,
      autoSwitch: false,
      bot: 'none',
      blacklist: '',
      whitelist: '',
      blackTimeMode: 'system',
      blackTimeCustom: '',
      whiteTimeMode: 'system',
      whiteTimeCustom: '',
      shieldProxy: false,
      regionMode: 'none',
      regionCustom: []
    },
    access: {
      acl: '',
      hotlink: {
        enable: false,
        scope: 'whole',
        allowEmpty: true,
        domains: ''
      },
      cors: {
        enable: false,
        allowOrigin: '*',
        allowMethods: '*',
        allowHeaders: '*',
        exposeHeaders: '*',
        allowCredentials: true,
        maxAge: 1728000
      }
    },
    advanced: {
      gzip: true,
      websocket: false,
      ipv6: false,
      logRequestHeader: false,
      logResponseHeader: false,
      logRequestBody: false,
      bodyLimit: '16',
      realtimeReturn: false,
      realtimeSend: false,
      acmeBacksource: false
    }
  }
}

function cacheTypeLabel(type) {
  return cacheTypeLabelMap[type] || type
}

function parseBool(value, fallback = false) {
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') {
    const v = value.trim().toLowerCase()
    return v === '1' || v === 'true' || v === 'on'
  }
  if (typeof value === 'number') return value !== 0
  return fallback
}

function splitLines(value) {
  if (!value) return []
  return String(value)
    .split(/\\r?\\n/)
    .map(item => item.trim())
    .filter(Boolean)
}

function loadSite() {
  if (!siteId.value) {
    ElMessage.warning('缺少 site_id')
    router.push({ path: '/website/list' })
    return
  }
  loading.value = true
  request
    .get(`/sites/${siteId.value}`)
    .then(res => {
      const data = res.data?.site || res.site || res.data || res
      if (!data || !data.id) {
        ElMessage.error('站点信息载入失败')
        return
      }
      site.value = data
      applySiteData(data)
    })
    .finally(() => {
      loading.value = false
    })
}

const aclList = ref([])

function loadAcls() {
  request.get('/acls').then(res => {
    aclList.value = res.data?.list || res.list || []
  })
}

function loadCerts() {
  request.get('/certs').then(res => {
    certList.value = res.data?.list || res.list || []
  })
}

function normalizeOriginCondition(item) {
  if (!item) return null
  return {
    item: item.item || 'uri',
    operator: item.operator || 'eq',
    value: item.value || '',
    origin: item.origin || '',
    header: item.header || '',
    seconds: item.seconds || ''
  }
}


function applySiteData(data) {
  site.value = data
  const settings = data.settings || {}

  // Basic
  siteSettings.basic.planName = data.user_package_id ? `套餐ID ${data.user_package_id}` : '商业版(飞扬)'
  siteSettings.basic.groupName = data.group_id ? `分组ID ${data.group_id}` : ''
  siteSettings.basic.nodeGroupName = data.node_group_id ? `集群ID ${data.node_group_id}` : ''
  siteSettings.basic.domain = (data.domains || []).join('\n')
  siteSettings.basic.cname = computeCname(data)
  siteSettings.basic.status = parseBool(data.enable, true)
  siteSettings.basic.createdAt = formatDate(data.create_at)
  siteSettings.basic.updatedAt = formatDate(data.update_at)
  siteSettings.basic.httpEnable = !!(data.http_listen && data.http_listen.length)
  siteSettings.basic.httpPorts = (data.http_listen || []).join(' ')
  siteSettings.basic.regionName = 'Global' // TODO

  // Origin
  siteSettings.origin.list = (data.backends || []).map(b => ({
      address: b,
      weight: 1,
      enable: true
  }))
  siteSettings.origin.protocol = data.backend_protocol || 'follow'
  siteSettings.origin.host = 'follow'
  if (settings.origin_host) {
      siteSettings.origin.host = settings.origin_host === 'follow' ? 'follow' : 'custom'
      siteSettings.origin.hostValue = settings.origin_host === 'follow' ? '' : settings.origin_host
  }
  siteSettings.origin.timeout = settings.origin_timeout || 60
  siteSettings.origin.connTimeout = settings.origin_conn_timeout || 10
  siteSettings.origin.balanceWay = data.balance_way || 'rr'

  // HTTPS
  siteSettings.https.enable = !!(data.https_listen && data.https_listen.length)
  siteSettings.https.listenPorts = (data.https_listen || []).join(' ')
  siteSettings.https.certId = data.cert_id || null
  siteSettings.https.force = parseBool(settings.force_https, false)
  siteSettings.https.forcePort = settings.force_https_port || '443'
  siteSettings.https.hsts = parseBool(settings.hsts, false)
  siteSettings.https.http2 = parseBool(settings.http2, false)
  siteSettings.https.http3 = parseBool(settings.http3, false)
  siteSettings.https.ocsp = parseBool(settings.ocsp_stapling, false)
  siteSettings.https.sslPolicy = settings.ssl_policy || 'compat'

  // Security
  siteSettings.security.cc.mode = data.cc_default_rule || 10002
  // Parse auto switch from JSON or dedicated fields if any
  if (settings.cc_auto_switch) {
      try {
          siteSettings.security.cc.autoSwitch = typeof settings.cc_auto_switch === 'string' 
            ? JSON.parse(settings.cc_auto_switch) 
            : settings.cc_auto_switch
      } catch(e) {}
  }
  
  // Lists
  siteSettings.security.ip.black = (settings.ip_black || []).join('\n')
  siteSettings.security.ip.white = (settings.ip_white || []).join('\n')
  siteSettings.security.ua.black = (settings.ua_black || []).join('\n')
  siteSettings.security.ua.white = (settings.ua_white || []).join('\n')
  
  // Cookie
  if (settings.cookie_secure) {
       // Assuming it's stored here
       siteSettings.security.cookie = settings.cookie_secure
  }
  
  // Region
  if (settings.region_block) {
       siteSettings.security.regions = settings.region_block
  }
  
  // Access
  if (settings.access) {
    if (settings.access.acl) siteSettings.access.acl = settings.access.acl
    if (settings.access.hotlink) {
      Object.assign(siteSettings.access.hotlink, settings.access.hotlink)
    }
    if (settings.access.cors) {
      Object.assign(siteSettings.access.cors, settings.access.cors)
    }
  }
  
  if (aclList.value.length === 0) {
    loadAcls()
  }
  if (certList.value.length === 0) {
      loadCerts()
  }
}

function computeCname(data) {
  if (data.cname_hostname) return data.cname_hostname
  if (data.domains?.length && data.cname_domain) {
    return `${data.domains[0]}.${data.cname_domain}`
  }
  return '-'
}

function formatDate(value) {
  if (!value) return '-'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return '-'
  return parsed.toLocaleString()
}

function toggleSiteStatus(enabled) {
  if (!siteId.value) return
  const action = enabled ? 'enable' : 'disable'
  request
    .post('/sites/batch_action', { action, ids: [siteId.value] })
    .then(() => {
      ElMessage.success('状态已更新')
      loadSite()
    })
}

function addOrigin() {
  siteSettings.origin.list.push({ address: '', weight: '10', enable: true })
}

function removeOrigin(index) {
  siteSettings.origin.list.splice(index, 1)
}

function addConditionOrigin() {
  siteSettings.origin.conditions.push({
    item: 'uri',
    operator: 'eq',
    value: '',
    origin: '',
    header: '',
    seconds: ''
  })
}

function removeConditionOrigin(index) {
  siteSettings.origin.conditions.splice(index, 1)
}

function handleRegionModeChange() {
  if (siteSettings.security.regionMode !== 'custom') {
    siteSettings.security.regionCustom = []
  }
}

function isOriginHeaderItem(item) {
  return item === 'header'
}

function isOriginStatItem(item) {
  return item === 'ua_count' || item === 'status_404'
}

function getOriginConditionPlaceholder(row) {
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
    case 'ua_count':
    case 'status_404':
      return '输入次数'
    case 'header':
      return '输入请求头名称'
    default:
      return '输入匹配值，一行一个'
  }
}

function handleOriginConditionChange(row) {
  if (!row) return
  if (isOriginStatItem(row.item)) {
    row.operator = 'gt'
    row.seconds = row.seconds || '10'
  } else if (!row.operator) {
    row.operator = 'eq'
  }
}

function openCacheRuleDialog(mode, rule, index) {
  cacheRuleDialog.mode = mode
  cacheRuleDialog.index = index ?? -1
  if (rule) {
    Object.assign(cacheRuleForm, { ...rule })
  } else {
    Object.assign(cacheRuleForm, { type: 'index', value: '', ttl: '86400', ignore_query: false, force_cache: false })
  }
  cacheRuleDialog.visible = true
}

function saveCacheRule() {
  const rule = normalizeCacheRule({
    type: cacheRuleForm.type,
    value: cacheRuleForm.value,
    ttl: cacheRuleForm.ttl,
    ignore_query: cacheRuleForm.ignore_query,
    force_cache: cacheRuleForm.force_cache
  })
  if (!rule) return
  if (cacheRuleDialog.mode === 'edit' && cacheRuleDialog.index >= 0) {
    siteSettings.cache.rules.splice(cacheRuleDialog.index, 1, rule)
  } else {
    siteSettings.cache.rules.push(rule)
  }
  cacheRuleDialog.visible = false
}

function removeCacheRule(index) {
  siteSettings.cache.rules.splice(index, 1)
}

function applyCachePreset(val) {
  if (!val) return
  let preset = null
  switch (val) {
    case 'index':
      preset = { type: 'index', value: '', ttl: '86400' }
      break
    case 'all':
      preset = { type: 'all', value: '', ttl: '259200' }
      break
    case 'static':
      preset = { type: 'suffix', value: 'jpg|jpeg|png|gif|ico|css|js|svg|bmp|webp|woff|woff2', ttl: '604800', ignore_query: true }
      break
    case 'video':
      preset = { type: 'suffix', value: 'mp4|avi|mov|webm|m3u8|ts', ttl: '2592000' }
      break
    case 'wordpress':
      preset = { type: 'all', value: '', ttl: '259200' }
      break
  }
  if (preset) {
    siteSettings.cache.rules.push(normalizeCacheRule(preset))
    ElMessage.success('已添加缓存规则')
  }
  cacheQuickPreset.value = ''
}

function normalizeCacheRule(rule) {
  if (!rule) return null
  const ttl = rule.ttl || '86400'
  return {
    type: rule.type || 'index',
    value: rule.value || '',
    ttl: String(ttl),
    ignore_query: !!rule.ignore_query,
    force_cache: !!rule.force_cache
  }
}

function parsePortList(raw) {
  if (!raw) return []
  return raw
    .split(/[\s,]+/)
    .map(item => item.trim())
    .filter(Boolean)
}


function buildSettingsPayload() {
  // Helper for splitting strings
  const splitStr = (str) => (str || '').split(/[\s\n]+/).filter(Boolean)

  return {
    origin: {
      location: siteSettings.origin.protocol === 'follow_port' ? '' : '', // Todo check if needed
      origin_protocol: siteSettings.origin.protocol, // Check key name in backend
      list: siteSettings.origin.list.map(item => ({
        address: item.address,
        weight: item.weight,
        enable: item.enable
      })),
      conditions: siteSettings.origin.conditions.map(item => ({
        ...item,
        seconds: item.seconds ? parseInt(item.seconds) : 0
      })),
      health_check: siteSettings.origin.healthCheckEnabled,
      health_host: siteSettings.origin.healthCheckHost,
      health_path: siteSettings.origin.healthCheckPath,
      health_status: siteSettings.origin.healthCheckStatus,
      health_interval: parseInt(siteSettings.origin.healthCheckInterval)
    },
    https: {
      listen_port: siteSettings.https.listenPorts, // String like "443 8443"
      force: siteSettings.https.force,
      redirect_port: siteSettings.https.forcePort,
      hsts: siteSettings.https.hsts,
      http2: siteSettings.https.http2,
      http3: siteSettings.https.http3,
      ocsp_stapling: siteSettings.https.ocsp,
      ssl_profile: siteSettings.https.sslPolicy,
      
      // These might need specific handling if custom
      ssl_protocols: '', 
      ssl_ciphers: '', 
      ssl_prefer_server_ciphers: true, 
      certificate_id: siteSettings.https.certId
    },
    cache: { 
        rules: siteSettings.cache.rules 
    },
    security: {
      default_rule: siteSettings.security.cc.mode,
      auto_switch: siteSettings.security.cc.autoSwitch.enable ? JSON.stringify(siteSettings.security.cc.autoSwitch) : '', 
      
      blacklist: splitStr(siteSettings.security.ip.black),
      whitelist: splitStr(siteSettings.security.ip.white),
      // UA lists might be separate keys or part of security
       
      shield_proxy: false, // Default or mock
      region_block: siteSettings.security.regions
    },
    // Advanced Settings
    gzip: siteSettings.advanced.gzip, // Simple bool as per new struct
    websocket: siteSettings.advanced.websocket,
    // Provide dedicated advanced object if backend expects it nested, or flat keys
    // Based on previous code: advanced: { gzip: ... }
    // Let's assume flat or specific structure based on prior context.
    // The previous code had `advanced: { gzip: ... }`. Let's assume `settings` object in payload needs these.
    
    // Actually, looking at `saveSettings`, it wraps `buildSettingsPayload` into `settings`.
    // So we should put these INSIDE the returned object here.
    
    url_rewrites: siteSettings.advanced.urlRewrites,
    req_headers: siteSettings.advanced.reqHeaders,
    res_headers: siteSettings.advanced.resHeaders,
    search_engine_origin: siteSettings.advanced.searchEngineOrigin,
    origin_cert: siteSettings.advanced.originCert,
    realtime_identify: siteSettings.advanced.realtimeIdentify,
    realtime_send: siteSettings.advanced.realtimeSend,
    
    access: {
      acl: siteSettings.access.acl,
      hotlink: siteSettings.access.hotlink,
      cors: siteSettings.access.cors
    },
    // Flattened / Specific keys for backend
    origin_host: siteSettings.origin.host === 'custom' ? siteSettings.origin.hostValue : 'follow',
    origin_timeout: siteSettings.origin.timeout,
    backend_protocol: siteSettings.origin.protocol
  }
}




function saveSettings() {
  if (!siteId.value) return
  isSaving.value = true
  
  const splitStr = (str) => (str || '').split(/[\s\n]+/).filter(Boolean)

  const payload = {
    ids: [siteId.value],
    settings: buildSettingsPayload(),
    
    // Top level overrides
    enable: siteSettings.basic.status,
    http_listen: siteSettings.basic.httpEnable ? splitStr(siteSettings.basic.httpPorts) : [],
    https_listen: siteSettings.https.enable ? splitStr(siteSettings.https.listenPorts) : [],
    backend_protocol: siteSettings.origin.protocol,
    cert_id: siteSettings.https.certId
  }
  
  if (siteSettings.basic.httpEnable && payload.http_listen.length === 0) {
      payload.http_listen = ['80']
  }
  
  // Clean up
  if (!siteSettings.https.enable) {
      payload.https_listen = []
  }

  request
    .put(`/sites/${siteId.value}`, payload)
    .then(() => {
      ElMessage.success('配置已保存')
      // loadSite() // Optional
    })
    .catch((e) => {
      ElMessage.error('保存失败: ' + (e.message || 'Error'))
    })
    .finally(() => {
      isSaving.value = false
    })
}

// Rewrite Logic
function openRewriteDialog(mode, index) {
  rewriteDialog.visible = true
  rewriteDialog.index = index ?? -1
  if (index >= 0) {
      // Edit
      const item = siteSettings.advanced.urlRewrites[index]
      Object.assign(rewriteForm, item)
  } else {
      // Create
      Object.assign(rewriteForm, { match: '', replace: '', code: '301' })
  }
}

function saveRewrite() {
    if (!rewriteForm.match || !rewriteForm.replace) {
        ElMessage.warning('请填写完整信息')
        return
    }
    const item = { ...rewriteForm }
    if (rewriteDialog.index >= 0) {
        siteSettings.advanced.urlRewrites.splice(rewriteDialog.index, 1, item)
    } else {
        siteSettings.advanced.urlRewrites.push(item)
    }
    rewriteDialog.visible = false
}

function removeRewrite(index) {
    siteSettings.advanced.urlRewrites.splice(index, 1)
}

// Header Logic
function openHeaderDialog(type, mode, index) {
    headerDialog.visible = true
    headerDialog.type = type
    headerDialog.index = index ?? -1 // If passed as generic args, might need adjustment
    // Simplified: always create for now or handle index check carefully if adding edit button
    if (typeof mode === 'number') {
        // Assume it's index if 2nd arg is number, but we called it with ('req', 'create')
        // Let's stick to explicit args from template: openHeaderDialog('req', 'create')
    }
    
    // We didn't pass index in create mode in template: openHeaderDialog('req', 'create')
    // We pass index in remove: removeHeader('req', $index)
    
    Object.assign(headerForm, { name: '', value: '' })
}

function saveHeader() {
    if (!headerForm.name) {
         ElMessage.warning('请填写名称')
         return
    }
    const item = { ...headerForm }
    const list = headerDialog.type === 'req' ? siteSettings.advanced.reqHeaders : siteSettings.advanced.resHeaders
    list.push(item)
    headerDialog.visible = false
}

function removeHeader(type, index) {
    const list = type === 'req' ? siteSettings.advanced.reqHeaders : siteSettings.advanced.resHeaders
    list.splice(index, 1)
}

function getCertDays(certId) {
    if (!certId) return 0
    const cert = certList.value.find(c => c.id === certId)
    if (!cert || !cert.expire_at) return 0
    
    // Parse Go/Standard date format if needed
    // Assuming 2024-01-01 00:00:00
    const now = new Date().getTime()
    const expire = new Date(cert.expire_at.replace(/-/g, '/')).getTime() // Simple replace for compat
    if (isNaN(expire)) return 0
    
    return Math.max(0, Math.floor((expire - now) / (1000 * 60 * 60 * 24)))
}


function goBack() {
  router.push({ path: '/website/list' })
}

function getHotlinkPlaceholder() {
  const scope = siteSettings.access.hotlink.scope
  if (scope === 'suffix') return '请输入后缀，如 png|jpg|gif'
  if (scope === 'dir') return '请输入目录，如 /image/|/static/|/upload/'
  if (scope === 'path') return '请输入路径，如 /index.html'
  return ''
}

onMounted(() => {
  loadAcls()
  loadSite()
  loadCerts()
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
.site-manage-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}
.toolbar-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.mb-16 {
  margin-bottom: 16px;
}
.config-form {
  margin-top: 12px;
}
.section-title {
  font-weight: 600;
  margin: 12px 0 8px;
}
.condition-origin-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.country-selector {
  margin-top: 12px;
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

<style scoped>
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
.save-status {
    position: fixed;
    top: 80px;
    right: 20px;
    z-index: 9999;
}
</style>
