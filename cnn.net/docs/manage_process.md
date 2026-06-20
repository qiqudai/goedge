# Manage Page Test Items

This checklist enumerates every interactive control on the site manage page.
Group assignment is excluded from testing per requirement.

## Global
- [x] Page header back button (SiteHeader back)
- [x] Tab switch: Basic, Origin, HTTPS, Security, Cache, Access, Advanced

## Basic Tab (BasicConfig.vue)
- [x] Status switch (siteSettings.basic.status)
- [x] CNAME display (site.cname) read-only
- [x] Plan expire time display (site.expireTime) read-only
- [x] Created time display (site.createdAt) read-only
- [x] Updated time display (site.updatedAt) read-only
- [x] User package select (siteSettings.basic.userPackageId)
- [x] Domain input (siteSettings.basic.domain)
- [x] HTTP enable switch (siteSettings.basic.httpEnable)
- [x] HTTP listen ports input (siteSettings.basic.httpPorts)
- [x] Origin list add row
- [x] Origin list remove row
- [x] Origin list address input (originList[].address)
- [x] Origin list weight input (originList[].weight)
- [x] Origin list enable switch (originList[].enable)
- [x] Condition origin list add row
- [x] Condition origin list remove row
- [x] Condition origin match item select (originConditions[].item)
- [x] Condition origin match operator select (originConditions[].operator)
- [x] Condition origin match value input (originConditions[].value)
- [x] Condition origin header name input (originConditions[].header, when item=header)
- [x] Condition origin target input (originConditions[].origin)

## Origin Tab (OriginConfig.vue)
- [x] Origin protocol radio (origin.protocol: http/https/follow/follow_port)
- [x] HTTP origin port input (origin.httpPort, when protocol http/follow)
- [x] HTTPS origin port input (origin.httpsPort, when protocol https/follow)
- [x] Origin host mode radio (origin.host: follow/domain/custom)
- [x] Origin host custom value input (origin.hostValue, when custom)
- [x] Origin timeout input (origin.timeout)
- [x] Origin connect timeout input (origin.connTimeout)

## HTTPS Tab (HttpsConfig.vue)
- [x] HTTPS enable switch (https.enable)
- [x] Certificate select dropdown (https.certId)
- [x] Apply certificate button (applyCert)
- [x] HTTPS listen ports input (https.listenPorts, when enabled + cert selected)
- [x] Force HTTPS switch (https.force)
- [x] Force HTTPS port select (https.forcePort, when force on)
- [x] HSTS switch (https.hsts)
- [x] HTTP2 switch (https.http2)
- [x] OCSP switch (https.ocsp)
- [x] HTTP3 switch (https.http3)
- [x] SSL policy radio (https.sslPolicy: compat/modern/custom)
- [x] SSL ciphers textarea (https.sslCiphers, when custom)
- [x] SSL protocols textarea (https.sslProtocols, when custom)

## Security Tab (SecurityConfig.vue)
- [x] CC default rule radio (security.cc.mode system rules)
- [x] CC custom rule select (security.cc.mode user rule, when custom)
- [x] CC auto-switch enable (security.cc.autoSwitch.enable)
- [x] CC auto-switch QPS select (security.cc.autoSwitch.qps)
- [x] CC auto-switch QPS custom input (security.cc.autoSwitch.qps, when custom)
- [x] CC auto-switch target rule select (security.cc.autoSwitch.rule)
- [x] CC custom rules add button
- [x] CC custom rules enable-all button
- [x] CC custom rules disable-all button
- [x] CC custom rules row status switch (customRules[].on)
- [x] CC custom rules row edit button
- [x] CC custom rules row delete button
- [x] CC rule dialog: matcher add (key/operator/value)
- [x] CC rule dialog: matcher remove
- [x] CC rule dialog: action select (allow/block/limit_rate/invisible/5s/click/click_simple/slide/slide_simple/captcha/rotate/302/url_auth)
- [x] CC rule dialog: limit_rate seconds input
- [x] CC rule dialog: limit_rate requests input
- [x] CC rule dialog: limit_rate urlRequests input
- [x] CC rule dialog: verification blockOnFail radio (for verification actions)
- [x] CC rule dialog: breakMatch radio
- [x] CC rule dialog: remark input
- [x] CC rule dialog: status switch
- [x] CC rule dialog: cancel/confirm
- [x] Crawler policy radio (security.crawlers.action: none/allow/block)
- [x] IP blacklist time mode radio (security.ip.blackTimeCustom)
- [x] IP blacklist time input (security.ip.blackTime, when custom)
- [x] IP whitelist time mode radio (security.ip.whiteTimeCustom)
- [x] IP whitelist time input (security.ip.whiteTime, when custom)
- [x] IP blacklist textarea (security.ip.black)
- [x] IP whitelist textarea (security.ip.white)
- [x] Cookie domain enable (security.cookie.enable)
- [x] Cookie domain input (security.cookie.domain, when enabled)
- [x] Block transparent proxy switch (security.block.transparentProxy)
- [x] Region block selector (security.regions)

## Cache Tab (CacheConfig.vue + CacheRuleDialog.vue)
- [x] Cache rule add button
- [x] Cache rule delete selected button
- [x] Cache quick preset select (index/all/static/video/wordpress)
- [x] Cache rules table row selection checkbox
- [x] Cache rules row edit button
- [x] Cache rules row delete button
- [x] Cache rule dialog: type select (index/all/dir/suffix/path)
- [x] Cache rule dialog: value input (dir/suffix/path)
- [x] Cache rule dialog: TTL value input
- [x] Cache rule dialog: TTL unit select (s/m/h/d)
- [x] Cache rule dialog: ignore query switch
- [x] Cache rule dialog: force cache switch
- [x] Cache rule dialog: toggle advanced settings
- [x] Cache rule dialog: enable slice switch
- [x] Cache rule dialog: ignore vary switch
- [x] Cache rule dialog: skip condition add (type + value)
- [x] Cache rule dialog: skip condition remove
- [x] Cache rule dialog: cancel/save

## Access Tab (AccessConfig.vue)
- [x] ACL select (access.acl)
- [x] Hotlink enable (access.hotlink.enable)
- [x] Hotlink scope radio (access.hotlink.scope)
- [x] Hotlink scope value input (access.hotlink.value, when not all)
- [x] Hotlink allow empty radio (access.hotlink.allowEmpty)
- [x] Hotlink extra domains input (access.hotlink.domains)
- [x] CORS enable (access.cors.enable)
- [x] CORS expand/collapse toggle (corsExpanded)
- [x] CORS allow_origin input (access.cors.allowOrigin)
- [x] CORS allow_methods input (access.cors.allowMethods)
- [x] CORS allow_headers input (access.cors.allowHeaders)
- [x] CORS expose_headers input (access.cors.exposeHeaders)
- [x] CORS allow_credentials radio (access.cors.allowCredentials)
- [x] CORS max_age input (access.cors.maxAge)

## Advanced Tab (AdvancedConfig.vue + RedirectRuleDialog.vue + HeaderRuleDialog.vue)
- [x] Upload limit mode radio (advanced.uploadLimitMode)
- [x] Upload limit value input (advanced.uploadLimitValue, when custom)
- [x] Gzip switch (advanced.gzip)
- [x] Websocket switch (advanced.websocket)
- [x] Search engine origin switch (advanced.searchEngineOrigin)
- [x] Search engine origin IP input (advanced.searchEngineOriginIp, when enabled)
- [x] URL redirect add button
- [x] URL redirect row edit button
- [x] URL redirect row delete button
- [x] Redirect dialog: match URI input
- [x] Redirect dialog: redirect target input
- [x] Redirect dialog: response code select (301/302/307/internal)
- [x] Redirect dialog: conditions expand/collapse
- [x] Redirect dialog: condition add select
- [x] Redirect dialog: condition value input
- [x] Redirect dialog: condition remove button
- [x] Redirect dialog: cancel/confirm
- [x] Request header add button
- [x] Request header row edit button
- [x] Request header row delete button
- [x] Response header add button
- [x] Response header row edit button
- [x] Response header row delete button
- [x] Header dialog (req/res): name input
- [x] Header dialog (req/res): value input
- [x] Header dialog (req/res): cancel/confirm
- [x] Access log: log request header switch (advanced.logRequestHeader)
- [x] Access log: log response header switch (advanced.logResponseHeader)
- [x] Access log: log request body switch (advanced.logRequestBody)
- [x] Access log: log request body size limit input (advanced.logRequestBodySizeLimit)
- [x] Origin cert validation switch (advanced.originCert)
- [x] Realtime identify switch (advanced.realtimeIdentify)
- [x] Realtime send switch (advanced.realtimeSend)
- [x] Default site switch (advanced.defaultSite)
- [x] L2 config radio (advanced.l2Config: current/none/custom)

> 说明（2026-04-03）：当前 Blazor 版使用“行内编辑”实现 URL Redirect/Header 编辑，功能等价于原 Dialog 的新增/编辑/删除输入能力；“确认”由页面保存按钮承担，“取消”可通过离开页面或手动回改值实现。
