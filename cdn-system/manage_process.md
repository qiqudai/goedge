# Manage Page Test Items

This checklist enumerates every interactive control on the site manage page.
Group assignment is excluded from testing per requirement.

## Global
- [ ] Page header back button (SiteHeader back)
- [ ] Tab switch: Basic, Origin, HTTPS, Security, Cache, Access, Advanced

## Basic Tab (BasicConfig.vue)
- [x] Status switch (siteSettings.basic.status)
- [ ] CNAME display (site.cname) read-only
- [ ] Plan expire time display (site.expireTime) read-only
- [ ] Created time display (site.createdAt) read-only
- [ ] Updated time display (site.updatedAt) read-only
- [ ] User package select (siteSettings.basic.userPackageId)
- [x] Domain input (siteSettings.basic.domain)
- [x] HTTP enable switch (siteSettings.basic.httpEnable)
- [x] HTTP listen ports input (siteSettings.basic.httpPorts)
- [x] Origin list add row
- [x] Origin list remove row
- [x] Origin list address input (originList[].address)
- [ ] Origin list weight input (originList[].weight)
- [ ] Origin list enable switch (originList[].enable)
- [x] Condition origin list add row
- [x] Condition origin list remove row
- [x] Condition origin match item select (originConditions[].item)
- [ ] Condition origin match operator select (originConditions[].operator)
- [ ] Condition origin match value input (originConditions[].value)
- [ ] Condition origin header name input (originConditions[].header, when item=header)
- [x] Condition origin target input (originConditions[].origin)

## Origin Tab (OriginConfig.vue)
- [x] Origin protocol radio (origin.protocol: http/https/follow/follow_port)
- [x] HTTP origin port input (origin.httpPort, when protocol http/follow)
- [ ] HTTPS origin port input (origin.httpsPort, when protocol https/follow)
- [x] Origin host mode radio (origin.host: follow/domain/custom)
- [x] Origin host custom value input (origin.hostValue, when custom)
- [x] Origin timeout input (origin.timeout)
- [x] Origin connect timeout input (origin.connTimeout)

## HTTPS Tab (HttpsConfig.vue)
- [ ] HTTPS enable switch (https.enable)
- [ ] Certificate select dropdown (https.certId)
- [ ] Apply certificate button (applyCert)
- [ ] HTTPS listen ports input (https.listenPorts, when enabled + cert selected)
- [ ] Force HTTPS switch (https.force)
- [ ] Force HTTPS port select (https.forcePort, when force on)
- [ ] HSTS switch (https.hsts)
- [ ] HTTP2 switch (https.http2)
- [ ] OCSP switch (https.ocsp)
- [ ] HTTP3 switch (https.http3)
- [ ] SSL policy radio (https.sslPolicy: compat/modern/custom)
- [ ] SSL ciphers textarea (https.sslCiphers, when custom)
- [ ] SSL protocols textarea (https.sslProtocols, when custom)

## Security Tab (SecurityConfig.vue)
- [ ] CC default rule radio (security.cc.mode system rules)
- [ ] CC custom rule select (security.cc.mode user rule, when custom)
- [ ] CC auto-switch enable (security.cc.autoSwitch.enable)
- [ ] CC auto-switch QPS select (security.cc.autoSwitch.qps)
- [ ] CC auto-switch QPS custom input (security.cc.autoSwitch.qps, when custom)
- [ ] CC auto-switch target rule select (security.cc.autoSwitch.rule)
- [ ] CC custom rules add button
- [ ] CC custom rules enable-all button
- [ ] CC custom rules disable-all button
- [ ] CC custom rules row status switch (customRules[].on)
- [ ] CC custom rules row edit button
- [ ] CC custom rules row delete button
- [ ] CC rule dialog: matcher add (key/operator/value)
- [ ] CC rule dialog: matcher remove
- [ ] CC rule dialog: action select (allow/block/limit_rate/invisible/5s/click/click_simple/slide/slide_simple/captcha/rotate/302/url_auth)
- [ ] CC rule dialog: limit_rate seconds input
- [ ] CC rule dialog: limit_rate requests input
- [ ] CC rule dialog: limit_rate urlRequests input
- [ ] CC rule dialog: verification blockOnFail radio (for verification actions)
- [ ] CC rule dialog: breakMatch radio
- [ ] CC rule dialog: remark input
- [ ] CC rule dialog: status switch
- [ ] CC rule dialog: cancel/confirm
- [ ] Crawler policy radio (security.crawlers.action: none/allow/block)
- [ ] IP blacklist time mode radio (security.ip.blackTimeCustom)
- [ ] IP blacklist time input (security.ip.blackTime, when custom)
- [ ] IP whitelist time mode radio (security.ip.whiteTimeCustom)
- [ ] IP whitelist time input (security.ip.whiteTime, when custom)
- [ ] IP blacklist textarea (security.ip.black)
- [ ] IP whitelist textarea (security.ip.white)
- [ ] Cookie domain enable (security.cookie.enable)
- [ ] Cookie domain input (security.cookie.domain, when enabled)
- [ ] Block transparent proxy switch (security.block.transparentProxy)
- [ ] Region block selector (security.regions)

## Cache Tab (CacheConfig.vue + CacheRuleDialog.vue)
- [ ] Cache rule add button
- [ ] Cache rule delete selected button
- [ ] Cache quick preset select (index/all/static/video/wordpress)
- [ ] Cache rules table row selection checkbox
- [ ] Cache rules row edit button
- [ ] Cache rules row delete button
- [ ] Cache rule dialog: type select (index/all/dir/suffix/path)
- [ ] Cache rule dialog: value input (dir/suffix/path)
- [ ] Cache rule dialog: TTL value input
- [ ] Cache rule dialog: TTL unit select (s/m/h/d)
- [ ] Cache rule dialog: ignore query switch
- [ ] Cache rule dialog: force cache switch
- [ ] Cache rule dialog: toggle advanced settings
- [ ] Cache rule dialog: enable slice switch
- [ ] Cache rule dialog: ignore vary switch
- [ ] Cache rule dialog: skip condition add (type + value)
- [ ] Cache rule dialog: skip condition remove
- [ ] Cache rule dialog: cancel/save

## Access Tab (AccessConfig.vue)
- [ ] ACL select (access.acl)
- [ ] Hotlink enable (access.hotlink.enable)
- [ ] Hotlink scope radio (access.hotlink.scope)
- [ ] Hotlink scope value input (access.hotlink.value, when not all)
- [ ] Hotlink allow empty radio (access.hotlink.allowEmpty)
- [ ] Hotlink extra domains input (access.hotlink.domains)
- [ ] CORS enable (access.cors.enable)
- [ ] CORS expand/collapse toggle (corsExpanded)
- [ ] CORS allow_origin input (access.cors.allowOrigin)
- [ ] CORS allow_methods input (access.cors.allowMethods)
- [ ] CORS allow_headers input (access.cors.allowHeaders)
- [ ] CORS expose_headers input (access.cors.exposeHeaders)
- [ ] CORS allow_credentials radio (access.cors.allowCredentials)
- [ ] CORS max_age input (access.cors.maxAge)

## Advanced Tab (AdvancedConfig.vue + RedirectRuleDialog.vue + HeaderRuleDialog.vue)
- [ ] Upload limit mode radio (advanced.uploadLimitMode)
- [ ] Upload limit value input (advanced.uploadLimitValue, when custom)
- [ ] Gzip switch (advanced.gzip)
- [ ] Websocket switch (advanced.websocket)
- [ ] Search engine origin switch (advanced.searchEngineOrigin)
- [ ] Search engine origin IP input (advanced.searchEngineOriginIp, when enabled)
- [ ] URL redirect add button
- [ ] URL redirect row edit button
- [ ] URL redirect row delete button
- [ ] Redirect dialog: match URI input
- [ ] Redirect dialog: redirect target input
- [ ] Redirect dialog: response code select (301/302/307/internal)
- [ ] Redirect dialog: conditions expand/collapse
- [ ] Redirect dialog: condition add select
- [ ] Redirect dialog: condition value input
- [ ] Redirect dialog: condition remove button
- [ ] Redirect dialog: cancel/confirm
- [ ] Request header add button
- [ ] Request header row edit button
- [ ] Request header row delete button
- [ ] Response header add button
- [ ] Response header row edit button
- [ ] Response header row delete button
- [ ] Header dialog (req/res): name input
- [ ] Header dialog (req/res): value input
- [ ] Header dialog (req/res): cancel/confirm
- [ ] Access log: log request header switch (advanced.logRequestHeader)
- [ ] Access log: log response header switch (advanced.logResponseHeader)
- [ ] Access log: log request body switch (advanced.logRequestBody)
- [ ] Access log: log request body size limit input (advanced.logRequestBodySizeLimit)
- [ ] Origin cert validation switch (advanced.originCert)
- [ ] Realtime identify switch (advanced.realtimeIdentify)
- [ ] Realtime send switch (advanced.realtimeSend)
- [ ] Default site switch (advanced.defaultSite)
- [ ] L2 config radio (advanced.l2Config: current/none/custom)
