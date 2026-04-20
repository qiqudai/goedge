# cnn.net 父版本兼容补齐可开工文档

目标：把当前 `src/Cnn.*` 补齐到可替代父版本 `../api + ../agent + ../web/admin` 的前后端与 agent 完整功能。

范围只列“还能开工的缺口”，不重复已经覆盖的模块。

## 1. 实施顺序

1. P0 先补数据面硬缺口：健康检查、TLS/HTTPS、缓存任务一致性、`deploy_cert` 任务链。
2. P1 再补安全链语义：ACL/WAF/CC 挑战、Cookie、地区封禁。
3. P2 最后收前端与控制面体验差异：套餐升降配、任务展示、页面配置联调。

---

## 2. P0 必做

### P0.1 健康检查执行链补齐

现状：
- `EdgeConfig` 已能编译健康检查字段。
- `ProxyConfigValidator` 已校验相关参数。
- 文档明确仍缺“主动健康检查细节”。

主要文件：
- [src/Cnn.Agent/Proxy/EdgeConfigToYarpCompiler.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Proxy/EdgeConfigToYarpCompiler.cs)
- [src/Cnn.Agent/Proxy/ProxyConfigValidator.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Proxy/ProxyConfigValidator.cs)
- [src/Cnn.Agent/Proxy/EdgeProxyRuntime.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Proxy/EdgeProxyRuntime.cs)
- [src/Cnn.Agent/Program.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Program.cs)

对照来源：
- [docs/CNN_NET_REMAINING_DESIGN_FULL.md](/Users/fake/code/goedge/cdn-system/cnn.net/docs/CNN_NET_REMAINING_DESIGN_FULL.md)
- [../agent/http_config.go](/Users/fake/code/goedge/cdn-system/agent/http_config.go)

要做：
- 让主动健康检查真正注册到 YARP 运行时，而不是只写 metadata。
- 被动健康检查摘除与恢复策略要按配置生效。
- cluster 不健康时的可用目标选择要和配置一致。

验收：
- 配坏一个 upstream 后，请求能自动摘除故障目标。
- 恢复后能按 `reactivation/interval/timeout/policy` 自动回切。
- 配置热更新后无需重启 Agent。

---

### P0.2 TLS / SNI / HTTPS 细粒度策略补齐

现状：
- 已有证书热加载、SNI、TLS policy store。
- 文档仍标注“未完整落地到 Kestrel/YARP”。

主要文件：
- [src/Cnn.Agent/Security/TlsCertificateStore.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/TlsCertificateStore.cs)
- [src/Cnn.Agent/Security/TlsRuntimePolicyStore.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/TlsRuntimePolicyStore.cs)
- [src/Cnn.Agent/Program.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Program.cs)
- [src/Cnn.Api/Services/Agent/EdgeConfigService.Build.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Agent/EdgeConfigService.Build.cs)
- [src/Cnn.Api/Services/Agent/EdgeConfigService.SiteSettings.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Agent/EdgeConfigService.SiteSettings.cs)

对照来源：
- [../agent/http_config.go](/Users/fake/code/goedge/cdn-system/agent/http_config.go)
- [docs/CNN_NET_REMAINING_DESIGN_FULL.md](/Users/fake/code/goedge/cdn-system/cnn.net/docs/CNN_NET_REMAINING_DESIGN_FULL.md)

要做：
- 校对 `https_hsts/http2/http3/ocsp/ssl_protocols/ssl_ciphers` 的编译与应用是否按站点语义完整生效。
- 明确当前“全局汇总模式”与旧版“域名维度策略”的差异，能补则补，不能补则先做兼容降级说明。
- 校对 fallback 证书、域名证书和热更新淘汰逻辑。

验收：
- 多证书、多 host 下 SNI 返回正确证书。
- 改 TLS 配置后新连接使用新策略。
- 无证书/坏证书时稳定回落 fallback，不阻断主链路。

---

### P0.3 缓存 key 与 purge/preheat/clear_cache 口径统一

现状：
- Agent 已有 `CacheKeyBuilder`。
- 文档明确记录 purge 与 cache key 可能不一致。

主要文件：
- [src/Cnn.Agent/Cache/CacheKeyBuilder.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Cache/CacheKeyBuilder.cs)
- [src/Cnn.Agent/Cache/CachePolicyResolver.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Cache/CachePolicyResolver.cs)
- [src/Cnn.Agent/Cache/CacheRuntimeStore.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Cache/CacheRuntimeStore.cs)
- [src/Cnn.Agent/Ws/AgentWsClient.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Ws/AgentWsClient.cs)
- [src/Cnn.Api/Pages/Website/Purge.razor](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Pages/Website/Purge.razor)

对照来源：
- [docs/cdn_readme.md](/Users/fake/code/goedge/cdn-system/cnn.net/docs/cdn_readme.md)
- [../agent/tasks.go](/Users/fake/code/goedge/cdn-system/agent/tasks.go)

要做：
- 定义唯一 cache key 规则，缓存命中、刷新 URL、刷新目录、预热都走同一套 builder。
- 补齐 `ignore_query/ignore_args/query_ignore_list/cache_key` 兼容映射。
- 校对 `clear_cache` 是否与旧版语义一致为“全量清空本站缓存”还是“全目录清空”。

验收：
- 同一 URL 的缓存文件、预热定位、purge 定位完全一致。
- 开启忽略参数后仍能正确清理。
- 批量 `clear_cache` 不误删其他站点缓存。

---

### P0.4 `deploy_cert` 任务链补齐

现状：
- `issue_cert` 已有。
- 文档明确写了 `deploy_cert` 当前无执行链路。

主要文件：
- [src/Cnn.Api/Services/Admin/CertService.Tasks.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Admin/CertService.Tasks.cs)
- [src/Cnn.Api/Services/Agent/AgentTaskService.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Agent/AgentTaskService.cs)
- [src/Cnn.Api/Services/Agent/AgentTaskAckService.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Agent/AgentTaskAckService.cs)
- [src/Cnn.Agent/Ws/AgentWsClient.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Ws/AgentWsClient.cs)

对照来源：
- [docs/cdn_readme.md](/Users/fake/code/goedge/cdn-system/cnn.net/docs/cdn_readme.md)
- [../agent/tasks.go](/Users/fake/code/goedge/cdn-system/agent/tasks.go)

要做：
- API 侧补 `deploy_cert` 任务创建与投递。
- Agent 侧补接收、落地证书、回 ACK、失败重试。
- 任务状态机保持和 Go 版一致：`waiting/running/retrying/success/fail/pending`。

验收：
- 证书签发后能自动触发部署任务。
- Agent 部署成功后证书立即可用。
- 失败能回写任务错误并支持重试。

---

## 3. P1 必做

### P1.1 ACL / WAF / 地区封禁完整等价

主要文件：
- [src/Cnn.Agent/Security/SecurityDecisionService.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/SecurityDecisionService.cs)
- [src/Cnn.Agent/Security/WafMatcher.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/WafMatcher.cs)
- [src/Cnn.Agent/Security/SiteSecurityMiddleware.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/SiteSecurityMiddleware.cs)

对照来源：
- [../agent/assets/lua/access.lua](/Users/fake/code/goedge/cdn-system/agent/assets/lua/access.lua)
- [../agent/assets/lua/access_guard.lua](/Users/fake/code/goedge/cdn-system/agent/assets/lua/access_guard.lua)
- [../agent/assets/conf/nginx.conf](/Users/fake/code/goedge/cdn-system/agent/assets/conf/nginx.conf)

要做：
- 对齐黑白名单、默认动作、区域封禁、空 UA、黑白 UA、黑白 URL。
- 校对错误页 key、返回码、日志类型。
- 补齐 `X-Geo-Country/CF-IPCountry/X-Country-Code` 以外的旧链路兼容逻辑。

验收：
- 旧配置导入后拦截结果与 Go 版一致。
- 命中规则时 block log / access log 字段齐全。

---

### P1.2 CC 挑战动作补齐

现状：
- `captcha/slide/click` 目前在 `CcEngine` 里只是返回 429。

主要文件：
- [src/Cnn.Agent/Security/CcEngine.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/CcEngine.cs)
- [src/Cnn.Agent/Security/SiteSecurityMiddleware.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/SiteSecurityMiddleware.cs)
- [src/Cnn.Api/Services/Agent/EdgeConfigService.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Agent/EdgeConfigService.cs)

对照来源：
- [../agent/assets/lua/access_guard.lua](/Users/fake/code/goedge/cdn-system/agent/assets/lua/access_guard.lua)
- [../agent/assets/lua/guard.lua](/Users/fake/code/goedge/cdn-system/agent/assets/lua/guard.lua)

要做：
- 至少恢复旧版 challenge 语义，不要只退化为简单封禁。
- 若暂时不做完整前端验证页，也要把 `captcha/slide/click` 区分处理。
- 守卫 TTL、放行 TTL、封禁 TTL 与旧配置一致。

验收：
- `block/captcha/slide/click/limit_rate` 能被明确区分。
- 调试日志能看到命中的 challenge 类型。

---

### P1.3 Cookie 策略真正执行

现状：
- 配置、DTO、页面都已存在。
- Agent 侧没有看到对应执行逻辑。

主要文件：
- [src/Cnn.Api/Services/Agent/EdgeConfigService.SiteSettings.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Agent/EdgeConfigService.SiteSettings.cs)
- [src/Cnn.Agent/Security/SiteSecurityMiddleware.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/SiteSecurityMiddleware.cs)
- [src/Cnn.Agent/Proxy/EdgeConfigToYarpCompiler.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Proxy/EdgeConfigToYarpCompiler.cs)

要做：
- 明确旧版 Cookie 域名策略语义。
- 响应 `Set-Cookie` 改写或过滤按配置生效。
- 与缓存策略配合，避免误缓存带 Cookie 响应。

验收：
- 开关 Cookie 域名策略后，浏览器收到的 Cookie 域正确变化。

---

## 4. P2 收口

### P2.1 网站配置页联调收口

主要文件：
- [src/Cnn.Api/Pages/Website/Manage.razor](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Pages/Website/Manage.razor)
- [src/Cnn.Api/Pages/Website/Rules.razor](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Pages/Website/Rules.razor)
- [src/Cnn.Api/Pages/System/Tasks.razor](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Pages/System/Tasks.razor)
- [src/Cnn.Api/Pages/Website/Purge.razor](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Pages/Website/Purge.razor)

对照来源：
- [../web/admin/src/views/website/Manage.vue](/Users/fake/code/goedge/cdn-system/web/admin/src/views/website/Manage.vue)
- [../web/admin/src/views/website/Rules.vue](/Users/fake/code/goedge/cdn-system/web/admin/src/views/website/Rules.vue)
- [../web/admin/src/views/website/Purge.vue](/Users/fake/code/goedge/cdn-system/web/admin/src/views/website/Purge.vue)

要做：
- 页面上已有配置项，要逐个验证是否能真实生效。
- 页面上若已有字段但后端暂不支持，要隐藏或标记，避免假功能。

验收：
- 页面可配项与 Agent/后端真实能力一致，不出现“能保存但不生效”。

---

### P2.2 套餐页升降配/升级包补齐

现状：
- `Plans/My` 目前只做了切换套餐。
- 旧版文档写明“升级包/更换套餐”有更完整语义。

主要文件：
- [src/Cnn.Api/Pages/Plans/My.razor](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Pages/Plans/My.razor)
- [src/Cnn.Api/Pages/Plans/Sold.razor](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Pages/Plans/Sold.razor)
- [src/Cnn.Api/Services/Common/UserPackageService.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Common/UserPackageService.cs)
- [src/Cnn.Api/Services/Common/UserPackageSyncService.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Common/UserPackageSyncService.cs)
- [src/Cnn.Api/Controllers/User/UserPackagesController.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Controllers/User/UserPackagesController.cs)
- [src/Cnn.Api/Controllers/Admin/UserPlansController.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Controllers/Admin/UserPlansController.cs)

对照来源：
- [docs/cdn_readme.md](/Users/fake/code/goedge/cdn-system/cnn.net/docs/cdn_readme.md)
- [../web/admin/src/views/packages/My.vue](/Users/fake/code/goedge/cdn-system/web/admin/src/views/packages/My.vue)
- [../web/admin/src/views/plans/Sold.vue](/Users/fake/code/goedge/cdn-system/web/admin/src/views/plans/Sold.vue)

要做：
- 补“已购升级包”展示与实际数据。
- 补“更换套餐/升降配”完整交互与价格、约束、任务同步。
- 保证变更后会触发 `package_sync`。

验收：
- 用户端与管理员端都能完整完成套餐调整。
- 调整后 Agent 套餐限制即时同步生效。

---

## 5. 推荐开发批次

### 批次 A
- P0.1 健康检查
- P0.2 TLS / HTTPS

### 批次 B
- P0.3 缓存 key 与任务统一
- P0.4 `deploy_cert`

### 批次 C
- P1.1 ACL/WAF/区域
- P1.2 CC challenge
- P1.3 Cookie

### 批次 D
- P2.1 页面联调
- P2.2 套餐页补齐

---

## 6. 每项任务统一交付要求

- 必须补单测，优先放在：
  - [tests/Cnn.Agent.Tests](/Users/fake/code/goedge/cdn-system/cnn.net/tests/Cnn.Agent.Tests)
  - [tests/Cnn.Api.Tests](/Users/fake/code/goedge/cdn-system/cnn.net/tests/Cnn.Api.Tests)
- 必须补最少一条端到端验证路径。
- 不允许页面保留“可配置但不生效”的假入口。
- 所有新增行为优先兼容旧字段，不先改 schema。

---

## 7. 最小开工清单

如果现在就开始，建议先从这 8 个文件下手：

1. [src/Cnn.Agent/Proxy/EdgeConfigToYarpCompiler.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Proxy/EdgeConfigToYarpCompiler.cs)
2. [src/Cnn.Agent/Program.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Program.cs)
3. [src/Cnn.Agent/Cache/CacheKeyBuilder.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Cache/CacheKeyBuilder.cs)
4. [src/Cnn.Agent/Ws/AgentWsClient.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Ws/AgentWsClient.cs)
5. [src/Cnn.Agent/Security/CcEngine.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/CcEngine.cs)
6. [src/Cnn.Agent/Security/SiteSecurityMiddleware.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Agent/Security/SiteSecurityMiddleware.cs)
7. [src/Cnn.Api/Services/Admin/CertService.Tasks.cs](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Services/Admin/CertService.Tasks.cs)
8. [src/Cnn.Api/Pages/Plans/My.razor](/Users/fake/code/goedge/cdn-system/cnn.net/src/Cnn.Api/Pages/Plans/My.razor)
