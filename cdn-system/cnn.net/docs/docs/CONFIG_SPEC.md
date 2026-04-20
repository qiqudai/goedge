# CDN Configuration Spec (Draft)

## 1. 默认配置与覆盖规则

1) 系统默认：管理员配置（global scope）
- 仅用于用户未配置或为空时的兜底默认值。
- 存储于 `config` 表，`scope_name=global`、`scope_id=0`。

2) 用户默认：用户级配置
- 用户创建站点/转发时默认使用用户默认配置。
- 用户默认缺失时回退系统默认。
- 存储于 `config` 表，`scope_name=user`、`scope_id=<uid>`。

3) 站点/转发自定义
- 站点/转发级配置覆盖用户默认与系统默认。

优先级：站点/转发自定义 > 用户默认 > 系统默认

## 2. 配置同步策略（目标）

目标：配置变更后尽快同步到边缘节点，避免节点长期不一致。

建议：版本号 + Redis 通知 + 节点定期拉取。

- 配置版本：每类资源（site/stream/cc/acl 等）都生成版本号，汇总为节点级版本。
- 通知：配置变更写入 Redis Pub/Sub（`config:changed`），节点收到后拉取最新版本。
- 兜底：节点定期拉取（如 60s），对比版本不一致则全量拉取。

## 3. 数据范围（涉及全量同步）

### 核心资源
- `user`：用户基础信息、登录态、账号状态等。
- `package`/`user_package`：套餐定义/用户购买套餐实例。
- `site`：网站 L7 CDN 配置。
- `stream`：TCP/UDP 转发配置。
- `cert`/`dnsapi`：证书与 DNS API。
- `acl`/`cc_rule`/`cc_match`/`cc_filter`：安全规则体系。

### 配置存储
- `config`：系统/默认配置存储，使用 `type + scope` 组织。

## 4. 需要全量清单

- 前端每个页面/选项 -> 对应数据表字段/配置 key 的精确映射。
- 配置继承逻辑必须在后端代码/文档中标注清楚。
- 配置同步策略必须与现有 Go 逻辑保持一致。

## 5. db.sql 结构目录（核心表）

- task：任务/异步作业
- site_conf_cache：站点配置缓存
- user：用户
- login_log：登录日志
- region：区域
- node：节点
- node_monitor_log：节点监控日志
- node_group：节点组/线路组
- line：节点线路明细
- op_log：操作日志
- dnsapi：DNS API
- cert：证书
- acl：ACL 规则
- cc_rule/cc_match/cc_filter：CC 规则
- package/package_group/merge_package_group：套餐体系
- package_up/user_package/user_package_up：套餐流量/带宽上调
- config：系统/默认配置键值
- site/site_group/merge_site_group：站点与分组
- stream/stream_group/merge_stream_group：转发与分组
- order：订单
- tlock：事务锁
- res_count：资源统计
- captcha：验证码
- api_key：API 密钥
- message/message_read/message_sub/message_send：消息系统
- ip_switch_log：线路切换日志
- lets_account：证书账户
- cname_domains：CNAME 域名库

## 6. 关键配置项（config 表 type 分类）

### site_default_config（站点默认）
- http_listen-port
- https_listen-port
- https_listen-hsts
- https_listen-http2
- https_listen-force_ssl_enable
- https_listen-ssl_protocols
- https_listen-ssl_ciphers
- https_listen-ssl_prefer_server_ciphers
- balance_way
- cc_default_rule
- gzip_enable
- gzip_types
- websocket_enable
- backend_protocol
- backend_http_port
- backend_https_port
- proxy_timeout
- range
- proxy_cache
- proxy_http_version
- post_size_limit
- proxy_ssl_protocols
- ups_keepalive
- ups_keepalive_conn
- ups_keepalive_timeout

### stream_default_config（转发默认）
- listen_protocol
- balance_way
- proxy_protocol

### cert_default_config
- cert_default_type

### site / stream / site_stream（资源配额）
- related-config-min-limit
- related-config-max-times-limit
- black-ip-limit
- white-ip-limit
- max-domain-persite-limit
- listen-default-http-80
- clean_url
- clean_dir
- pre_cache_url
- pre_cache_timeout
- ip-unlock-max-limit
- ip-unlock-max-per-limit
- cc-rule-max-limit
- acl-max-limit
- download-access-log-limit
- download-access-log-tmp-dir
- download-access-log-retain
- custom-port-not-allow
- custom-port-allow

### nginx_config / openresty_config / error_page
- nginx-config-file
- openresty-config
- error-page

### system（系统配置）
- keep-login-log-days / keep-op-log-days / keep-task-log-days
- keep-access-log-days / keep-node-log-days / keep-traffic-history-days
- backup_rate / backup_keep_days / backup_dir
- max_site_stream_sync_one_time
- login_session_valid_time
- dns_config
- admin_domain / user_domain
- allow_register
- smtp / sms_config
- register_success_templ / forget_password_templ / email_captcha_templ
- register_require / user_agreement
- phone_captcha_templ
- alipay_id_auth
- dns_rs_protect
- node_health_check / node_max_failed
- record_repair / record_sync / record-repair-enable
- package_expire_close_site / traffic_excceed_close_site
- package_allow_upgrade / package_allow_downgrade
- system_info / auth_code / maintain / master_client_ip_header
- recharge
- auto_upgrade_agent
- delete_config_delayed
- notification-period
- traffic-exceed-notify / traffic-exceeding-notify
- package-expire-notify / package-expiring-notify
- cert-expire-notify / cert-expiring-notify
- cc-switch-notify
- bandwidth-exceed-notify / conn-exceed-notify
- notify-method
- sync-site-config-scope
- node_monitor_config
- https_cert / https_key

## 7. 站点默认应用规则（旧系统行为）

创建站点时默认值来源：
- 用户默认存在：`config(type=site_default_config,scope_name=user,scope_id=uid)`
- 用户默认缺失：`config(type=site_default_config,scope_name=global,scope_id=0)`

应用字段（映射）：
- http_listen-port -> site.http_listen
- https_listen-port -> site.https_listen
- balance_way -> site.balance_way
- backend_protocol -> site.backend_protocol
- cc_default_rule -> site.cc_default_rule
- 其余默认配置写入 `config(type=site_settings,scope_name=site,scope_id=siteId,name=settings)`（按 https/backsource/cache/advanced 分类）

## 8. 转发默认应用规则（旧系统行为）

创建转发时默认值来源：
- 用户默认存在：`config(type=stream_default_config,scope_name=user,scope_id=uid)`
- 用户默认缺失：`config(type=stream_default_config,scope_name=global,scope_id=0)`

应用字段（映射）：
- listen_protocol -> forward.settings.listen_protocol
- balance_way -> forward.settings.origin.balance_way
- proxy_protocol -> forward.settings.origin.proxy_protocol


## 9. 配置同步（config_items / config_version）

**现有行为（Go）**
- admin GET `/config_items`：允许 `type` 过滤；`scope_name/scope_id` 逻辑存在，历史上可能返回所有 scope。
- admin POST `/config_items`：`type/scope_name/scope_id` upsert；`name/value/enable`；完成后触发 `BumpConfigVersion("config_item")` 并创建 `task(type=config_sync)`。
- user GET `/config_items`：固定 `scope_name=user, scope_id=uid`。
- user POST `/config_items`：强制 `scope_name=user, scope_id=uid`；完成后 bump `config_version`（`resource=config_item, ids=[uid]`）。
- 版本号存储：`config(name=edge_config_version,type=system,scope=global)`。

**C# 重构要求**
- admin 必须显式传 `scope_name/scope_id`，禁止全量 scope 返回。
- 统一返回结构 `{code,message,data,trace_id}` + i18n。
- 写入 config_items 必须触发 `config_sync`，保持与 Go 行为一致。
- admin/user 路径与权限严格隔离。
- 必须实现 admin/user GET/POST `/config_items`，scope 强校验 + config_sync。

**参考（Go）**
- `api/controllers/config_item_controller.go`
- `api/services/sync_service.go`


## 10. 站点关键字段映射（站点/域名）

说明：settings 指 `config(type=site_settings,scope_name=site,scope_id=siteId,name=settings).value` 的 JSON。

### 10.1 基础
- 套餐 -> `site.user_package`
- 合并站点组 -> `merge_site_group`
- DNS API -> `site.dns_provider_id`

### 10.2 HTTP
- 监听端口 -> `site.http_listen`

### 10.3 HTTPS
- 监听端口 -> `site.https_listen`
- 强制 HTTPS -> `settings.https.force`
- 跳转端口 -> `settings.https.redirect_port`
- HSTS -> `settings.https.hsts`
- HTTP/2 -> `settings.https.http2`
- OCSP -> `settings.https.ocsp_stapling`
- HTTP/3 -> `settings.https.http3`
- SSL Profile -> `settings.https.ssl_profile`
- 协议/套件 -> `settings.https.ssl_protocols` / `settings.https.ssl_ciphers`
- PreferServerCiphers -> `settings.https.ssl_prefer_server_ciphers`

### 10.4 源站
- 源站列表 -> `settings.origin.list`
- 回源条件 -> `settings.origin.conditions`
- 健康检查 -> `settings.origin.health_check`
- 负载均衡 -> `site.balance_way`

### 10.5 回源协议/端口
- 协议 -> `site.backend_protocol` + `settings.backsource.protocol`
- 端口/Host/超时 -> `settings.backsource.{http_port,https_port,host_mode,host_custom,timeout,connect_timeout}`

### 10.6 缓存
- 缓存/规则/TTL -> `settings.cache.{enable,rules,ttl}`

### 10.7 安全/CC
- 默认 CC -> `site.cc_default_rule` + `settings.security.default_rule`
- CC 规则 -> `settings.security.{auto_switch,custom_rules}`
- 黑/白名单时间 -> `settings.security.{black_time_mode,black_time_custom,white_time_mode,white_time_custom}`
- 黑/白名单 -> `site.black_ip` / `site.white_ip` + `settings.security.{blacklist,whitelist}`
- 透明代理 -> `settings.security.block_transparent_proxy`
- 区域屏蔽 -> `site.block_region` + `settings.security.{region_block,region_custom}`

### 10.8 访问控制
- ACL/防盗链/CORS -> `settings.access.{acl,hotlink,cors}`

### 10.9 高级
- IPv6/Gzip/Websocket -> `settings.advanced.{ipv6,gzip,websocket}`
- 搜索引擎回源 -> `settings.search_engine_origin` / `settings.search_engine_origin_ip`
- 错误页 -> `settings.advanced.error_pages`
- URL 重定向 -> `settings.advanced.url_redirects`
- 请求/响应头 -> `settings.advanced.{origin_headers,cdn_headers}`
- ACME/日志/上传 -> `settings.advanced.{acme_backsource,realtime_return,realtime_send,log_request_header,log_response_header,log_request_body,body_limit}`

### 10.10 site_settings 存储规则（config 表）
- 主键：`type=site_settings` + `scope_name=site` + `scope_id=siteId` + `name=settings`
- `value` 为 JSON，缺失字段按 `site` 表字段/默认值补齐。


## 11. global_config / WAF / Resources / EdgeConfig

### 11.1 DefaultConfig（global_config.default_config）
- 网站默认：`global_config.default_config.website`（`cache_enable/cache_ttl/gzip/waf_enable`）
- API 默认：`global_config.default_config.api`
- 下载默认：`global_config.default_config.download`

### 11.2 WAF（global_config.waf）
- 关键字段含义见 `cnn.net/docs/cdn_readme.md` 对应章节

### 11.3 Resources（global_config.resources）
- website/forward/public 资源限制字段见 `cnn.net/docs/cdn_readme.md` 对应章节

### 11.4 L7 站点下发
- EdgeConfig 生成逻辑：`api/services/config_service.go`
- domains/upstreams 映射：`EdgeDomain` / `EdgeUpstream`
- 证书绑定：按 `cert.domain` 匹配 HTTPS 域名
- Lua 入口：`cdn-edge-node/lua/config_loader.lua` / `access.lua` / `ssl_manager.lua`

### 11.5 L4（stream/tcp/udp）
- Nginx stream 配置生成依赖 `EdgeStream` 结构

### 11.6 WAF Lua 行为
- `lua/waf.lua` 根据 `waf` 结构执行（黑/白名单、默认策略）

### 11.7 CC 规则
- `cc_rule/cc_match/cc_filter` 下发到 `EdgeConfig` 交由 `waf.lua/anti_cc.lua` 执行
- API：`/rules/cc/*`

### 11.8 ACL
- `acl_default_action` + `acl_rules` 下发到 `EdgeConfig`
- `access.lua` 负责 IP/CIDR 判断


## Progress Log
- 修复了前台页面的端口配置占位与列表字段：account/finance/Orders/website/Purge/BlockLogs/Rules/Certs/global/DefaultConfig/ErrorPages/system/Messages/Announcements/Tasks/settings/Monitor。
- 替换 DNS 提交器为 DB 存储，修正 DNS 页面中文提示文案。

## Progress Log
- 2025-12-23: Fixed admin API errors for users/status, packages, plans, user_plans assign, forwards/forward_groups, and dns/providers; mapped forward models to stream tables and package/user_package to plans.
- 2025-12-23: Expanded node model/controller to cover node table fields, added sub IP handling via pid children, and updated node list UI with auto-disable and bandwidth limit controls.
- 2025-12-23: Node create/update now treat region_id as nullable to avoid FK errors when not set.
- 2025-12-23: Fixed node model column mapping (pid) and removed non-existent token field to stop insert errors.
- 2025-12-23: Removed node token generation from controller to match DB schema.
- 2025-12-23: Website list UI removes user/package selects; site create/batch now default user to current admin and auto-pick first user_package when not provided.
