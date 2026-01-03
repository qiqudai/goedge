# Agent 数据同步计划

## 一、目标
将以下管理端配置与套餐信息统一迁移到 WebSocket 通道，由 agent 接收、反序列化、更新内存、再写到本地文件，同时保持原有日志/指标的单向上报策略，满足全球配置、资源数量、错误页、套餐、持久化等整体需求。计划按数据类型逐一列出字段、持久化策略和待补充事项。

## 二、同步清单（按照页面分类）

### 1. `全局防火墙`（`/global/firewall` → `GlobalConfig.waf`）
**传输载体**：`EdgeConfig.WAF`（由 ConfigService 打包后通过 WS 发送）  
**字段**：
- 基础/拉黑：`enable`、`default_block_action`、`auto_ipset_enable`、`auto_ipset_threshold`、`block_page_rate_limit_enable`、`block_page_rate_limit`、`block_page_traffic_free`、`blacklist_timeout`、`temp_whitelist_timeout`、`temp_whitelist_limit_total`、`temp_whitelist_limit_url`  
- 名单：`whitelist_ips`、`blacklist_ips`  
- 系统安全/CC：`prevent_tls_handshake`、`block_unbound_domain`、`disable_ping`、`default_page_protection`、`default_page_protection_threshold`、`anti_cc_type`、`anti_cc_image_source`、`anti_cc_image_custom_url`、`anti_cc_debug`、`cc_rule_auto_switch`  
- 系统/资源保护：`secret_key`、`node_log_clean_strategy`、`well_known_protection_threshold`、`resource_protection_enable`、`resource_protection_threshold`、`resource_protection_block_timeout`、`resource_protection_rules[] {duration,max_requests}`  
- 兼容字段（避免后端逻辑漏掉）：`mode`、`policy`、`cc.*`（`enable`、`threshold`、`action`、`block_timeout`、`emergency_mode`、`slide_count`）、`access_control.*`（黑白名单、UA/URL、区域、`block_empty_ua`），其中 `access_control.region_block` 等价于 `block_region`（如 CN、HK、JP、VN 等）；`syntactic`（`sql_injection`、`xss`、`scanner`）
**agent 持久化策略**：
1. 接收到 `heartbeat`/`node_sync` 中附带的 WAF payload 后先反序列化成 `models.WAFConfig`（确保所有字段存在）；  
2. 反序列化成功再写入 `/etc/cdn/global_waf.json`（或 configurable path），再更新内存配置；  
3. 如果写盘失败，记录错误并保持旧配置，不触发节点同步；  
4. 变更时触发 Lua ACL/CC 规则重载。

### 2. `全局 Nginx`（`/global/nginx` → `config_items(type=nginx_config,name=nginx-config-file)`）
**传输载体**：`EdgeConfig.Nginx`（`loadNginxConfig` 读取并加入 WS payload）  
**字段**：
- 顶层：`worker_processes`、`worker_connections`、`worker_rlimit_nofile`、`worker_shutdown_timeout`、`logs_dir`、`resolver`、`resolver_timeout`  
- `http` map：`proxy_cache_dir`、`proxy_cache_max_size`、`proxy_cache_keys_zone_size`、`proxy_cache_methods`、`custom_snippet`，以及映射的 HTTP directives（`proxy_request_buffering`、`proxy_buffering`、`proxy_http_version`、`proxy_next_upstream`、`proxy_max_temp_file_size`、`proxy_connect_timeout`、`proxy_send_timeout`、`proxy_read_timeout`、`client_max_body_size`、`large_client_header_buffers`、`gzip`、`keepalive_timeout`、`keepalive_requests`、`gzip_comp_level`、`gzip_http_version`、`gzip_min_length`、`gzip_vary`、`server_tokens`、`log_not_found`、`default_type`、`server`）  
- `stream` map：`proxy_connect_timeout`、`proxy_timeout`  
**agent 持久化策略**：
1. 接到配置后反序列化到 `models.EdgeNginxConfig`（保留 `http`/`stream` map）；  
2. 写入 `/etc/cdn/global_nginx.json`，同时让 `generateDynamicConfigs` 参考该结构重新生成 `conf/dynamic/*`；  
3. 保留 JSON 原文用于 diff/version 检测，配置修改后通过 WebSocket 通知 agent 重载。

### 3. `资源限制配置`（`/global/resources` → `GlobalConfig.resources`）
**传输载体**：新增 `EdgeConfig.Resources`（目前需拓展 ConfigService 及 agent payload）  
**字段**：
- `website`：`min_limit`、`max_limit_multiplier`、`max_blacklist_ips`、`max_whitelist_ips`、`daily_url_purge_limit`、`daily_dir_purge_limit`、`daily_preload_limit`、`daily_unlock_ip_limit`、`unlock_ip_batch_limit`、`max_cc_rules_per_group`、`max_acl_rules`、`daily_log_download_limit`、`log_storage_dir`、`log_storage_hours`、`max_domains_per_site`、`default_listen_80`
- `website`：`min_limit`、`max_limit_multiplier`、`max_blacklist_ips`、`max_whitelist_ips`、`daily_url_purge_limit`、`daily_dir_purge_limit`、`daily_preload_limit`、`daily_unlock_ip_limit`、`unlock_ip_batch_limit`、`max_cc_rules_per_group`、`max_acl_rules`、`daily_log_download_limit`、`log_storage_dir`、`log_storage_hours`、`max_domains_per_site`、`default_listen_80`（`true` 表示 agent 默认创建的网站将监听 80 端口）
- `forward`：`disabled_ports`、`min_limit`、`max_limit_multiplier`、`max_acl_rules`  
- `public`：`disabled_custom_ports`、`allowed_custom_ports`  
**agent 持久化策略**：
1. 反序列化成 `models.GlobalResourceConfig` 后写 `/etc/cdn/resources.json`；  
2. 维护 `resources` 的内存副本供限流/日志等模块查询；  
3. 需要配合节点分组/地区再次校验（例如 `default_listen_80` 影响端口监听）。

### 4. `错误页配置`（`/global/errors` → `GlobalConfig.error_pages`）
**字段**：代码=HTML 内容，包含 `400`、`403`、`502`、`504`、`traffic_limit`、`site_locked`、`domain_invalid`、`conn_limit`、`timeout`（套餐到期）、`ip`（限制IP访问）（前端 `ErrorPages.vue` 编辑）  
**状态码映射建议**（用于 Nginx `error_page`）：  
- `traffic_limit` → 509  
- `site_locked` → 451  
- `domain_invalid` → 404  
- `conn_limit` → 429（配合 `limit_conn_status 429`）  
- `timeout` → 410（套餐到期）  
- `ip` → 418（IP 限制访问）  
**agent 持久化策略**：  
1. 接收 `map[string]string` → 写 `/etc/cdn/error_pages.json`；  
2. 将内容写成 Nginx 可引用的 HTML/JSON 文件或由 Lua 直接读取；  
3. 保留内存拷贝供 runtime 生效；  
4. 若字段缺失或 HTML 非法，返回错误给 WS 心跳 ack 让 master 重试。

### 5. `已售套餐`（`/plans/sold` 管理套餐配置）
**传输载体**：已有 `services.UserPackageService.SyncUserPackage` 生成 `AgentPackageConfig` 的任务/Job，通过 WS `job_dispatch` 下发。  
**字段**（与 `models.AgentPackageConfig` 对齐）：
- `PackageID`、`UID`、`Version`、`Status`（active/expired/deleted）  
- 节点分配：`RegionID`、`NodeGroupID`、`BackupNodeGroup`、`EnableBackup`  
- CNAME：`Cname.Domain`、`Cname.Hostname`、`Cname.Hostname2`、`Cname.Mode`、`Cname.RecordID`  
- 限制：`Limits.Traffic`、`Limits.Bandwidth`、`Limits.Connection`、`Limits.Domain`  
- 功能：`Features.HTTPPort`、`Features.StreamPort`、`Features.Websocket`、`Features.CustomCCRule`  
- 时间：`Time.StartAt`、`Time.EndAt`  
- 附加布尔值（来自 `user_package_config` 配置表）：`ipv6`、`http3_enabled`  
**agent 持久化策略**：  
1. Task payload 反序列化成 `AgentPackageConfig`；  
2. 写 `/etc/cdn/packages/{package_id}.json`，并更新内存 map；  
3. 若写入失败，记录日志并不更新 task 状态，等待重试；  
4. 需在 agent 启动时加载所有 package 文件补全缓存。

### 6. `CK 数据`
保持现有逻辑：agent 继续向 API 推送 access log、metrics、event（三个 ClickHouse 表），不需要下发回 agent，只做统计展示。

## 三、版本与同步控制及应用保障
- 每次 `global_config`、`config_items`、`user_package` 更变时都触发 `services.BumpConfigVersion`，master 通过 `heartbeat_ack.sync_action` 或单独的 `node_sync` 告知 agent 重新拉取/应用。  
- 同步后的配置必须驱动 agent 内部流程：WAF/资源规则变化触发 Lua ACL/CC 重建，Nginx/global 变更必须写入动态 conf 并 `nginx -s reload` (或在生成后调用已有 `executeReload`)，套餐同步写文件后应立即在内存限额/路由筛选中生效。  
- 站点锁定/套餐到期/流量超限等“硬限制”建议由 master 生成 Nginx 配置直接 `return <status>`，命中 `error_page` 输出固定 HTML；解除限制时重新生成配置移除相关 `return`，避免运行时判断的性能损耗。  
- 同步消息需包含 `version` 字段（可用更新时间 Unix）以便 agent 判定是否已经应用并避免重复写盘。  
- 全局配置和套餐统一通过 WS 发送，agent 在完成反序列化并写盘后，再通过 `node_sync` 或 `heartbeat` 上报 `success`，如失败将返回错误以触发 master 重试。

## 四、遗漏事项与补充建议
1. `GlobalConfig.DefaultConfig`（站点默认模板）目前未纳入 WS，但 agent 在生成域名/站点时需要该模板，建议同步并持久化。  
2. `Global CC 规则`（`models.CCRule` 相关）需确保 `EdgeConfig.CCRules`/`CCMatchers`/`CCFilters` 在配置变更后也及时持久化；如果直接写入 agent “Lua ACL”，需要同步版本号。  
3. “资源限制”中的 `log_storage_dir`/`log_storage_hours` 影响日志轮转脚本，需完成 agent 本地脚本与 JSON 之间的绑定。  
4. `Node/Region` 相关 metadata（例如 `default_listen_80` 对应的实际监听 port），如果管理端未来支持按节点细分限额，payload 结构需扩展。  
5. 需要定义统一的持久化根路径（如 `/etc/cdn/`）和格式，便于 agent 启动时依赖文件恢复内存状态。

## 五、下一步任务
1. 为 ConfigService 增加 `EdgeConfig.Resources` 和 `ErrorPages` 字段，保证 WS payload 里完整传输这些配置。  
2. 在 agent WS client 中新增 `heartbeat`/`node_sync` 对应的消息处理，写入 JSON、持久化文件、更新内存。  
3. 在 agent 启动流程中加载 `/etc/cdn/*.json`，并在写入失败时实现回滚/重试战略。  
4. 完善 `UserPackageService.SyncUserPackage` 的 payload 校验与版本控制，保证 `AgentPackageConfig` 一致性。  
5. 撰写测试用例验证：WS config 更新→agent 反序列化→写盘→内存生效→node_sync ack。

请审核以上计划，有没有遗漏或需要微调的字段、路径、版本策略，确认后我可以按此任务拆解代码实现。 

## ��������ִ���߼���������
- ����Դ��agent �ϱ� access log -> ClickHouse `node_access_logs`��ͳ�� `sum(bytes)`���� `host` �� `site.domains` ƥ�䡣
- ��ͨ���㣺`�ײ�����(GB)` vs `sum(bytes)`����ʹ�� `tcp_traffic_factor`��system config_items����������
- ִ�п��أ�`traffic_excceed_close_site`=1 ��ִ�У����򲻸� `site.state`��
- ���Ӧ�ã�
  1) ���� -> `site.state=traffic_limit`��Nginx `return 509` + `error_page` ʹ�� `traffic_limit` ҳ��
  2) �ָ� -> ֻ�ָ���ǰ `state=traffic_limit` ��վ�㣬����״̬�����š�
- ͬ����ÿ�θĶ� `site.state` ����� `BumpConfigVersion("site", siteIDs)`��agent ����һ����ȡ��ִ�� `nginx -s reload`��ȷ����Ч��
