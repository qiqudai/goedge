# GoEdge CDN 系统 API & Agent 全量说明（源码级整理版）


> 生成目标：覆盖 **API 与 Agent 的结构、每个接口功能、返回数据结构、所有功能点**。本文以 `E:/cdn/goedge/cdn-system` 当前源码为准


---



## 0. 总览



### 0.1 模块划分

- **API（控制面）**：`api/`，提供管理端/用户/Agent 接口与后台任务
- **Agent（边缘节点）**：`agent/`，负责拉取配置、生成 Nginx/OpenResty 配置、日志指标上报、任务执行与升级
- **运行时资源**：Agent 启动时在 `WorkDir/edge-node` 生成完整运行时目录（3.1）


### 0.2 入口

- API 入口：`api/main.go`

- Agent 入口：`agent/main.go`



### 0.3 认证与权限
- **Admin/User API**：`Authorization: Bearer <token>`，JWT `utils.GenerateTokenWithExpiry` 生成；`AuthRequired(role)` 校验角色
  - 请求成功时可能返回 `X-Auth-Token` 做滑动续期
- **Agent API**：`Authorization: Bearer <agent_token>`
  - **全局 Token**：`config.App.AgentToken` 或环境变量 `APP_AGENT_TOKEN`
  - **节点 Token**：`node.token`（数据库）
- **WS**：Agent API 通过 `/ws/agent` 建立 WebSocket，先发送 `agent_hello` 完成鉴权与注册


---



## 1. Agent 运行结构与行为


### 1.1 Agent 启动参数与配置
- CLI 参数：
  - `-config`：配置文件路径（默认 `agent.json`）
  - `-api`：API BaseURL

  - `-token`：节点 Token

  - `-node-id`：节点 ID（字符串，通常是数值）

  - `-debug`：调试日志
  - `-version`：输出版本
- `agent.json`（示例结构）字段
  - `api`、`token`、`node_id`、`debug`

  - `work_dir`：工作目录（固定为可执行文件所在目录，忽略配置）
  - `reset_resources`：启动前清空 WorkDir/edge-node

  - `bootstrap_sync`：启动时拉取配置

  - `bootstrap_start`：bootstrap 后启动 Nginx

> 约定：WorkDir 永远等于可执行文件所在目录（API/Agent 一致），其他运行时/配置/数据路径均相对该目录解析。



### 1.2 运行时目录结构（WorkDir=可执行文件所在目录，运行时根=WorkDir/edge-node）
- `conf/`

  - `cdn_config.json`：完整 EdgeConfig

  - `cdn_config.json.bak`：备份
  - `resources.json`：Global resources

  - `error_pages.json` + `error_pages/*.html`

  - `default_config.json`

  - `cc_rules.json`/`cc_matchers.json`/`cc_filters.json`

  - `l2_status.json`

  - `dynamic/`
    - `http.conf` / `http_global.conf`

    - `stream.conf` / `stream_global.conf`

    - `main.conf` / `events.conf`

- `logs/`：访问日志/stream 日志/pid

- `cache/`：缓存
- `cert/`：证书与 `acme/`

- `packages/`：用户套餐配置包 `*.json`

- `lua/`：运行时 Lua 脚本

- `openresty/`：OpenResty 运行环境



### 1.3 启动流程（agent/main.go）
1. 解析参数 `agent.json`
2. 若未指定 NodeID，则默认使用主机名
3. `initEnvironment()`
   - 创建目录
   - 解包 `assets`（OpenResty、Lua、conf、cert 等）
   - 修补 `nginx.conf` 路径（cache、ip2region）
   - 生成 fallback 证书占位
4. `bootstrapSyncAndStart()`
   - `/api/v1/agent/config?node_id=...` 拉取配置（不 reload）
   - 根据 `bootstrap_start` 启动 Nginx
5. 启动 WS、日志指标上报、日志清理、L2 健康检查


### 1.4 WS 协议（Agent <-> API）
**握手**
- Agent 发送 `agent_hello`
  ```json

  {"kind":"agent_hello","node_id":"<string>","token":"<token>","agent_version":"1.0.3","capabilities":["sync_package","acl_publish","cc_publish"]}

  ```

- API 验证 token -> 绑定 `node_id`

**C# 重构（当前实现）**
- Agent 发送 `hello`：
  ```json
  {"kind":"hello","node_id":"<string>","token":"<token>","version":"1.0"}
  ```
- API 返回 `ack`：`{"kind":"ack","connection_id":"...","heartbeat":30}`
- API 下发配置：
  - `edge_config`：`{"kind":"edge_config","data":{...EdgeConfig...}}`
  - `cache_config`：`{"kind":"cache_config","data":{...CacheSiteConfig...}}`
- 心跳：`ping/pong`

**消息类型（现有 Go）**（双向）
- `heartbeat` / `heartbeat_ack`

- `task_dispatch` / `task_ack`

- `node_sync`

- `agent_logs_access` / `agent_logs_stream` / `agent_logs_metrics` / `agent_logs_events`

- `l2_nodes_request` / `l2_nodes_response`

- `l2_heartbeat`

- `cert_issued`



### 1.5 Agent 任务类型（WS 下发）
- `refresh_url`：按 URL 清理缓存

- `refresh_dir`：清空缓存目
- `clear_cache`：清`cache` 目录

- `preheat`：预URL（对本机 127.0.0.1 发请求）

- `issue_cert`：ACME 证书签发（含 rate limit 回报）
- `config_sync`：强制应用配置并 reload

- `package_sync`：同步用户套餐配置到 `packages/*.json`

- `agent_upgrade`：下载并升级 agent + edge-node 资源



### 1.6 Nginx/OpenResty 配置生成（Agent）
- `http_config.go`
**缓存规则（现状 + C# 重构建议）**
- 现状数据源：
  - 站点级设置：`config(type=site_settings,scope_name=site,scope_id=siteId,name=settings).cache`（`enable/ttl/rules`）
  - 默认值：`site_defaults` -> `config(type=site_default_config,name=proxy_cache)`
- 规则字段（前端/存储）：
  - `type/value` 或 `rule/uri/prefix/ext`
  - `ttl/ignore_query/ignore_args/force_cache/no_cache/enable/priority/cache_key`
- 规则解析（API）：
  - `type` 映射：`suffix` -> `ext`，`dir` -> `prefix`，`path` -> `uri`，`all` -> `prefix=/`，`index` -> `uri=/`
  - `value` 支持空格/`|` 分隔；`suffix` 自动去掉 `*`/`.` 前缀
  - `ignore_query` 与 `ignore_args` 视为同义，最终落在 `ignore_args`
  - `normalizeCacheRulesRaw`：按 location 去重（倒序保留最后一条）
- 规则落地（Agent -> Nginx）：
  - 按 `priority` 降序生成 location；与保留路径冲突（`/_guard`、`/.well-known`）则跳过
  - `rule` 直接作为 Nginx location（`= /x`、`^~ /p`、`~*` 等）；仅 `/path` 自动补 `^~ /path`
  - `enable=false` 或 `no_cache=true` -> `proxy_no_cache 1; proxy_cache_bypass 1;`
  - `force_cache=true` -> `proxy_ignore_headers Cache-Control Expires;`
  - `ttl>0` -> `proxy_cache_valid 200 302 <ttl>;`
  - `cache_key` 非空优先；否则：
    - `ignore_args=true` -> `proxy_cache_key $host$uri`
    - 默认 -> `proxy_cache_key $host$uri$is_args$args`
- 现网缺口（Go）：
  - purge URL 仍按 `host+uri+args` 定位缓存文件（与 `ignore_args/cache_key` 不一致）
  - Lua `cache.resolve` 仅设置 `cache_bypass/cache_ttl`，当前 Nginx 未使用该变量
  - 前端字段 `enable_range/ignore_vary/skip_conditions` 未落地
- C# 重构落地（兼容并更简）：
  - `query_mode`: `all(默认)/ignore/include/exclude` + `query_keys`
  - `cache_key` 保留为高级覆盖（优先级最高）
  - 统一 purge/预热 与 cache key 计算口径（包含忽略参数逻辑）
  - 对 `query_keys` 做排序/规范化，保证 key 稳定
- `stream_config.go`：
  - L4 转发（TCP/UDP）、ProxyProtocol、conn_limit、L2 优先/回源策略
- `main.conf` / `events.conf`：
  - worker_processes、worker_connections、rlimit、shutdown_timeout
- `http_global.conf` / `stream_global.conf`：
  - resolver、proxy_cache_path 等全局指令
---



## 2. EdgeConfig（API 下发 Agent）


### 2.1 EdgeConfig 顶层结构

- `version` / `node_id` / `node_level`

- `domains[]` / `upstreams[]` / `streams[]`

- `waf` / `resources` / `error_pages` / `default_config`

- `cc_rules` / `cc_matchers` / `cc_filters`

- `nginx`

- `fallback_cert_data` / `fallback_key_data`



### 2.2 EdgeDomain 核心字段

- 站点：`name`, `upstream_key`, `status`, `http_listen`, `https_listen`

- L2：`use_l2`, `l2_upstream_key`, `l2_http_port`, `l2_https_port`

- SSL：`ssl_cert_data`, `ssl_key_data`, `https_force`, `https_hsts`, `https_http2`, `https_http3`

- 代理：`proxy_connect_timeout`, `proxy_read_timeout`, `proxy_send_timeout`

- 缓存：`cache.enable`, `cache.default_ttl`, `cache.rules`

- 访问控制：`acl_default_action`, `acl_rules`, `black_ips`, `white_ips`, `region_block`

- CC：`cc_rule_id`

- 防盗/ CORS / Cookie

- 连接限速：`conn_limit`, `limit_rate`, `body_limit`

- Gzip / Websocket / Range



### 2.3 EdgeStream

- `listen_ports`, `targets`, `use_listen_port`, `balance_way`, `proxy_protocol`, `proxy_connect_timeout`, `proxy_timeout`, `conn_limit`

**现有 Go 行为**
- 来源：`stream` 表（enable=true + node_group_id in 当前节点组），字段 `listen/backend/acl/balance_way/proxy_protocol`
- 解析：
  - `listen`：支持 JSON 数组或空格/逗号分隔
  - `backend`：支持 JSON 数组 `{address,weight,enable}` 或字符串列表，禁用/空地址忽略
  - `acl.origin`：读取 `connect_timeout/proxy_timeout/conn_limit`
- 默认值：
  - `connect_timeout` 为空 → `10s`
  - `proxy_timeout` 为空 → `60s`
  - `balance_way` 为空 → 从默认配置补齐（见 2.3.1）
  - `proxy_protocol`：默认配置存在时直接覆盖
- L2 逻辑：
  - 仅 `node.level=1` 时生效
  - `resolveL2Enabled("current", groupL2Config, user_package.l2_origin)` 为 true 且存在 L2 节点时启用
  - 启用后：L2 节点作为主目标（`weight=1, enable=true, node_id`），源站作为 `backup`
  - 启用后 `use_listen_port=true`（Agent 在上游地址无端口时补 listen 端口）

**C# 重构要求**
- 以上解析/默认值/L2 规则完全一致
- `balance_way` 不做映射，保持原值（仅 trim）
- `proxy_protocol` 默认值与 Go 一致：默认存在时覆盖 `stream.proxy_protocol`

### 2.3.1 站点/转发默认配置（site_default_config / stream_default_config）

**现有 Go 行为**
- `GetSiteDefaultMap(userID)` / `GetStreamDefaultMap(userID)`：读取 `config` 表对应 `type` 的默认配置（全局 + 用户覆盖）
- `ApplySiteDefaults/ApplyForwardDefaults`：将默认值写入 `settings`，作为后续 `EdgeConfig` 生成输入
  - `stream_default_config` 之外，还会合并 `config(type=system,name=forward_default_settings)` 的全局默认项（仅 `scope=global`）

**C# 重构实现（当前落地）**
- 读取 `config` 表：
  - `type=site_default_config`：全局（`scope_name=global, scope_id=0`）+ 用户覆盖（`scope_name=user, scope_id=uid`）合并
  - `type=stream_default_config`：全局（`scope_name=global, scope_id=0`）+ 用户覆盖
  - `type=system,name=forward_default_settings`：合并为全局默认（覆盖同名项）
- 仅在站点字段为空时注入默认值（不回写 DB）：
  - `http_listen-port` / `https_listen-port` -> `http_listen` / `https_listen`
  - `backend_protocol` / `backend_http_port` / `backend_https_port`
  - `balance_way` / `proxy_timeout`
  - `gzip_enable` / `gzip_types` / `websocket_enable` / `range`
  - `proxy_http_version` / `proxy_ssl_protocols` / `post_size_limit`
  - `ups_keepalive` / `ups_keepalive_conn` / `ups_keepalive_timeout`
  - `cc_default_rule` / `proxy_cache`
  - `https_listen-hsts` / `https_listen-http2` / `https_listen-force_ssl_enable`
  - `https_listen-ssl_protocols` / `https_listen-ssl_ciphers` / `https_listen-ssl_prefer_server_ciphers`
- `stream_default_config` 目前用于补齐：
  - `balance_way`（stream.balance_way 为空时）
  - `proxy_protocol`（默认存在时直接覆盖 stream.proxy_protocol）

**缺失字段策略**
- 站点 settings 统一存放于 `config` 表：
  - `type=site_settings`、`scope_name=site`、`scope_id=siteId`、`name=settings`（Value=JSON）
- 若缺少对应 JSON key：按旧字段/默认值兜底（如 `proxy_cache/hotlink/cors/advanced/backsource`）



### 2.4 GlobalConfig / WAF / Resources

**数据存储与下发**
- 存储位置：`config` 表（SysConfig），`name=global_config`、`type=system`、`scope_name=global`、`scope_id=0`
- GET `/global_config`：读取 SysConfig.Value(JSON)；不存在时写入默认配置；解析失败时回退默认配置
- POST `/global_config`：覆盖保存整个 GlobalConfig；触发 `BumpConfigVersion("global_config")` 并创建 `config_sync` Task（进入任务派发）
- 节点下发：
  - **现有 Go**：`ConfigService.GenerateConfigForNode` -> `EdgeConfig.WAF/Resources/ErrorPages/DefaultConfig`
  - **C# 重构**：`EdgeConfigService.GenerateAsync` 生成 `EdgeConfig`，通过 `/ws/agent` 下发 `edge_config`，并提供 `/api/v1/agent/config?node_id=...` 兜底拉取
- 注意：`GlobalConfig.Nginx` 为历史字段；当前节点 Nginx 配置实际来自 `config_items(type=nginx_config,name=nginx-config-file)`

**WAF（字段与行为）**
- `enable`：全局开关
- `default_block_action`：`ipset|disconnect|page`
- `auto_ipset_enable` / `auto_ipset_threshold`（req/s）：自动升级 IPSet
- `block_page_rate_limit_enable` / `block_page_rate_limit`（次/60s）`block_page_traffic_free`
- `blacklist_timeout` / `temp_whitelist_timeout`（秒）
- `temp_whitelist_limit_total` / `temp_whitelist_limit_url`（秒窗口）
- `whitelist_ips` / `blacklist_ips`：多行文本，支持 CIDR
- `prevent_tls_handshake` / `block_unbound_domain` / `disable_ping`
- `default_page_protection` / `default_page_protection_threshold`
- `secret_key`：Guard/JS 验证等通信密钥
- `node_log_clean_strategy`：`none|log_only|log_cache`
- `cc_rule_auto_switch`
- `anti_cc_type`：`slide|click|5s|rotate|slide_simple` 
- `anti_cc_image_source`（system/custom）`anti_cc_image_custom_url`
- `anti_cc_debug`
- `well_known_protection_threshold`（次/60s）：`.well-known` 防护阈值
- `resource_protection_enable/threshold/block_timeout` + `resource_protection_rules[{duration,max_requests}]`
- 兼容字段：`mode/policy/cc/access_control/syntactic`（Lua WAF 仍可读取；C# 重构建议统一到上面字段）

**ErrorPages**
- `error_pages`：`map[string]HTML`，键：`400/403/502/504/traffic_limit/site_locked/domain_invalid/conn_limit/timeout/ip`
- 默认加载：从 `config_items(type=error_page,name=error-page)` 读取 JSON（`p400/p403/p502/p504/p513/p514/host_not_found/p515/p512/access_ip_not_allow`），缺失时使用内置模板
- 下发后落盘到节点 `conf/error_pages.json`，供 Lua/模板渲染

**Resources（资源限制与运行时参数）**
- `website`：`min_limit/max_limit_multiplier/max_blacklist_ips/max_whitelist_ips/daily_url_purge_limit/daily_dir_purge_limit/daily_preload_limit/daily_unlock_ip_limit/unlock_ip_batch_limit/max_cc_rules_per_group/max_acl_rules/daily_log_download_limit/log_storage_dir/log_storage_hours/max_domains_per_site/default_listen_80`
- `forward`：`disabled_ports/min_limit/max_limit_multiplier/max_acl_rules`
- `public`：`disabled_custom_ports/allowed_custom_ports`
- 运行时使用：agent 读取 `default_listen_80/log_storage_dir/log_storage_hours/disabled_ports/disabled_custom_ports`；其余字段主要用于后台约束/套餐逻辑（当前代码未统一强校验）

**DefaultConfig（模板默认值）**
- `default_config.website/api/download`：`cache_enable/cache_ttl/gzip/waf_enable/ssl_ciphers`
- 用途：新站点创建或模板应用时补充默认值（`DefaultConfigService.ApplySiteTemplateDefaults`）

**缓存规则**
- 详述见 1.6 中的 "缓存规则（现状 + C# 重构建议）"，此处不重复。

**C# 重构要求**
- GET `/global_config`：记录不存在时创建默认值；JSON 解析失败返回默认值（与现有行为一致）
- POST `/global_config`：全量覆盖保存并触发 `BumpConfigVersion("global_config")`
- ErrorPages 必须从 `config_items(type=error_page,name=error-page)` 读取并落盘
- `GlobalConfig.Nginx` 仅作为历史字段保留，实际 Nginx 配置取 `config_items(type=nginx_config,name=nginx-config-file)`
- EdgeConfig 下发必须包含 `waf/resources/error_pages/default_config/cc_rules/cc_matchers/cc_filters`，ErrorPages 做 canonical key 归一化（`p400` -> `400` 等）
- 统一返回结构 + i18n
---



## 3. API：路由与接口（按分组）


> 统一返回结构：`{"code":200,"message":"Success","data":...}`
> 业务错误统一`code` 表示；HTTP 状态码仅作为传输层，可保持 200
> message 根据前端语言自动本地化（优先 `Accept-Language`/`lang`，未传用系统默认语言）


### 3.1 Public



- 返回结构（统一）：`{"code":200,"message":"Success","data":...}`，message 本地化（Accept-Language/lang）
#### GET /health

- 功能：负载均衡健康检查
- 返回：`{ status, node }`

**现有行为（Go 代码）**
- 路由：`/health`
- 返回：HTTP 200，JSON `{"status": i18n.T("status.ok"), "node": i18n.T("server-1")}`
- 无鉴权、无统一包装（直接 JSON）

**C# 重构要求**
- 必须同时提供 `/health` 与 `/api/health`，两者返回一致（兼容 LB 与现有 C# 路由）
- 统一返回结构 `{code,message,data,trace_id}`，`data` 包含 `status/node`
- `status/node` 需要按 `Accept-Language/lang` 本地化（沿用 `status.ok`、`server-1`）
- 仍保持公开路径不鉴权



#### GET /.well-known/acme-challenge/:token

- 功能：ACME HTTP-01 challenge 文件服务（Agent 与 API 均提供，实际由 Agent 对外服务）

**现有行为（Go 代码）**
- 路由：`/.well-known/acme-challenge/:token`
- `token` 为空直接 404
- 读取内存 `services.AcmeTokens`（由 `/api/v1/agent/acme/tokens` 写入）
- 命中返回纯文本 `text/plain`（body 为 token value）；未命中 404
- 无鉴权

**C# 重构要求**
- 路由保持标准路径；禁止统一 JSON 包装（否则 ACME 验证失败）
- 继续使用内存 Token Store（与 Agent 接口共享）
- 命中时 `Content-Type=text/plain`；空或不存在返回 404



#### POST /api/v1/login | /api/v1/admin/login | /api/v1/user/login

- 请求：`{username,password,password_hash?,captcha?,captcha_type?}`

- 功能：用户/管理员登录（支持用户名或邮箱），可触发验证码验证与限流
- 返回：`{ token, role, uid, name }`

- 可能返回：`rate_limited`, `rate_cooldown`

**现有行为（Go 代码）**
- 支持用户名或邮箱登录（`name/email`）
- 密码：
  - `password_hash=sha256` 时要求 `password` 为 64 位 hex
  - 未声明但看似 SHA256 时视为已哈希
- 兼容旧密码：若数据库密码非 bcrypt，登录成功后自动升级为 bcrypt
- 角色：`type=1` -> `admin`，否则 `user`
- 启用校验：`enable=false` 拒绝登录
- Host 限制：
  - 读取 `limit_admin_login_domain` / `limit_user_login_domain`
  - 若配置包含点：仅允许精确 Host
  - 若不包含点：按前缀 + `bind-master-host` 组合匹配
  - Host 来源优先 `X-Forwarded-Host`，否则 `Request.Host`
- 限流：`login|ip|username`，5 次/5 分钟，触发 10 分钟冷却
  - 返回 `rate_limited=true`，`rate_cooldown` 为剩余秒数
- 验证码：
  - 受 `allow-enable-email-captcha-login` / `allow-enable-sms-captcha-login` 控制
  - 优先使用请求 `captcha_type`，其次用户 `login_captcha`，再回退 email/sms
  - email/sms 缺失对应联系方式直接报错
  - 验证失败返回 `Invalid captcha`
- 会话 TTL：`login_session_valid_time`（秒），空/非法时默认 24 小时
- 记录登录日志：`login_log(uid, ip, success, post_content, create_at)`

**C# 重构要求**
- 所有校验、限流、Host 限制、密码升级与 TTL 规则保持一致
- 登录接口需返回 `rate_limited/rate_cooldown`（统一响应结构内）
- `master_client_ip_header` 优先作为客户端 IP 读取来源
- 统一返回结构 `{code,message,data,trace_id}` + i18n



#### POST /api/v1/login/captcha | /api/v1/admin/login/captcha | /api/v1/user/login/captcha

- 请求：`{username,type?}`

- 功能：发送登录验证码（邮件）

- 返回：`{"code":200,"message":"Success","data":null}` 或错误码

**现有行为（Go 代码）**
- 限流：`captcha|ip|username`，3 次/10 分钟，触发 30 分钟冷却
- 仅支持 email；sms 直接返回未配置
- 邮件模板取 `config(type=system,name=email_captcha_templ)` JSON（`title`/`data`）
  - 支持变量：`{{captcha}}/{{code}}/{{username}}`
- 保存验证码到 `captcha` 表（TTL=5 分钟）
- 非法用户/禁用用户直接返回错误

**C# 重构要求**
- 与 Go 保持相同限流、模板、验证码存储与 TTL
- 返回 `rate_limited/rate_cooldown`（统一结构内）
- 统一返回结构 `{code,message,data,trace_id}` + i18n


#### POST /api/v1/register | /api/v1/user/register

- 请求：`{username,password,password_hash?,email?,phone?}`

- 功能：注册（由系统配置 `allow_register` 控制）
- 返回：`{"code":200,"message":"Success","data":null}`

**现有行为（Go 代码）**
- `allow_register=false` 直接拒绝
- 校验用户名/密码；`password_hash=sha256` 需满足 SHA256 格式
- `name/email` 任一重复即拒绝
- 新用户：`enable=true`、`type=2`、`group_id=0`、`created_at=now`、密码为 bcrypt

**C# 重构要求**
- 注册校验规则与字段保持一致（不新增字段）
- 统一返回结构 `{code,message,data,trace_id}` + i18n



#### GET /api/v1/system_info

- 功能：系统信息
- 返回：系统信息 JSON

**现有行为（Go 代码）**
- 数据来源：系统配置 `system_info`（`config` 表，`type=system`、`scope_name=global`、`scope_id=0`）
- `system_info` 为 JSON，解析失败则返回空结构
- 常用字段（与前端 BasicConfig 对齐）：
  - `sys_name`
  - `user_console_title`
  - `admin_console_title`
  - `favicon_file`
  - `logo_file`
  - `login_ad_file`
  - `footer_link`
  - `footer_copyright`
- 额外注入三项布尔开关：
  - `enable_email_login` <- `allow-enable-email-captcha-login`
  - `enable_sms_login` <- `allow-enable-sms-captcha-login`
  - `allow_register` <- `allow_register`
- 公共与管理端同一逻辑（仅路由不同）

**C# 重构要求**
- 使用 `SystemInfoService` + `SystemConfigService` 读取/合并字段，保持字段名与默认行为一致
- 统一返回结构 `{code,message,data,trace_id}` + i18n



---



### 3.2 /api/v1（公共）ACL 列表


- 返回结构（统一）：`{"code":200,"message":"Success","data":...}`，message 本地化（Accept-Language/lang）
#### GET /api/v1/acls

- 功能：ACL 列表（管理员或用户视角过滤）

- 查询：`name`, `status(on/off)`

- 返回：`{"code":200,"message":"Success","data":{list,total}}`

**现有行为（Go 代码）**
- 用户端：基于当前登录 `uid` 过滤（不接受前端传入的 `user_id` 覆盖）
- 管理端：不默认过滤 `uid`，可通过 `user_id` 查询特定用户
- `name`：`LIKE %keyword%` 模糊查询
- `status`：`on/off` 映射 `enable=true/false`
- 返回列表字段：`id/user_id/uid/user{name,id}/name/des/default_action/enable/create_time`
  - `create_time` 格式：`YYYY-MM-DD HH:mm:ss`
  - `user_id=0` 表示系统规则（`user.username` 为空）

**C# 重构要求**
- 列表过滤规则/字段保持一致（`name/status/user_id` 与 `create_time` 格式）
- 用户端必须以登录身份为准；未登录返回 `user_id_required`
- 统一返回结构 `{code,message,data,trace_id}` + i18n



---



### 3.3 Admin /api/v1/admin


- 返回结构（统一）：`{"code":200,"message":"Success","data":...}`，message 本地化（Accept-Language/lang）
#### 节点（Node）
- **GET /nodes**：节点列表（region、安装状态等）
- **GET /nodes/:id/monitor_logs**：节点监控日志
- **POST /nodes**：创建节点
- **POST /nodes/:id/install**：通过 SSH 自动安装 agent

- **PUT /nodes/:id**：更新节点
- **PUT /nodes/:id/status**：启停节点
- **DELETE /nodes/:id**：删除节点
- **POST /nodes/batch**：批量操作（启停/删除等）

- **POST /nodes/batch_action**：同上（兼容路径）

**现有行为（Go 代码）**
- 列表：仅返回主节点（`pid=0`）；支持 `keyword/region_id/status/node_type/page/pageSize`
  - `status=enabled/disabled` 映射 `enable=true/false`
  - `node_type` 映射 `level`（仅当 >0）
  - `keyword` 若为数字：`id` 精确匹配；否则 `name/ip` 模糊匹配（`lower(name)`）
  - 额外填充：`sub_ips`（`pid IN parentIDs` 的子节点）、`line_count`（`lines` 分组）、`region_name`、安装进度（`FetchInstallProgress`）、在线状态（`IsNodeOnline(30s)`）
- `PUT /nodes/:id/status`：
  - 仅更新 `enable` + `config_task`（`sync_enable/sync_disable`）
  - 子节点同步 `enable`
  - 写 IP 切换日志并触发 `SyncPackageCnameForNodes(add/delete)`
- 创建节点：
  - 必填 `name/ip`；`region_id=0` 视为 `NULL`
  - `work_dir` 固定为可执行文件所在目录（由安装位置决定，不接受外部传入）
  - `token` 优先使用全局 `App.AgentToken`；否则生成随机 Token
  - `enable=false` 也会被强制改为 `true`（当前实现不允许新建禁用）
  - `auto_install=true`：`install_status=running` 并异步执行安装（`InstallNodeAgent`）
  - `sub_ips` 会转为子节点（`pid=parentID`），仅复制基础字段（无 token）
  - 创建成功后：若 `enable=true` 触发 `SyncPackageCnameForNodes(add)`
- 更新节点：
  - 若存在 `lines` 绑定（`node_id` 或 `node_ip_id`），禁止变更 `region_id`
  - `enable` 变更会写 `config_task`
  - `ssh_password/ssh_key` 仅在有值时更新
  - 子节点全部重建（`replaceSubIPs`）
  - 更新后触发 `SyncPackageCnameForNodes(resync)`
- 删除节点：
  - 先写 IP 切换日志、同步 `SyncPackageCnameForNodes(delete)`
  - 事务内删除：`lines`（`node_id`/`node_ip_id` in ids）→ 子节点 → 主节点
- 批量操作：
  - `start/stop`：更新 `enable` + `config_task`，并同步子节点
  - `delete`：删除 `lines`（仅 `node_id IN ids`）、子节点、主节点
  - 注意：批量删除不触发 `SyncPackageCnameForNodes`（与单删不一致）
- 监控日志：按 `node_id + type + timeRange` 过滤，按 `event_id + create_at` 分组，返回 `checked_at/fail_count/total_count`

**C# 重构要求**
- 列表查询/过滤/分页规则与字段回填必须一致（`pid=0`、`SubIPs/LineCount/RegionName/InstallProgress/Online`）
- `work_dir` 固定为可执行文件所在目录（由安装位置决定）；新建节点即使传 `enable=false` 也要落为 `true`
- `region_id` 变更限制必须保留（存在 `lines` 绑定即阻断）
- `config_task` 语义保持（启用/停用触发 `sync_enable/sync_disable`）
- `sub_ips` 处理必须与 `replaceSubIPs` 一致（删除后重建）
- 统一返回 `{code,message,data}` + i18n；保留单删/批量删在 DNS 同步上的差异（如需改动需显式说明）


#### 线路组（NodeGroup）
- **GET /node-groups**：线路组列表

- **POST /node-groups**：创建
- **PUT /node-groups/:id**：更新
- **DELETE /node-groups/:id**：删除
- **GET /node-groups/:id/resolution**：解析配置
- **POST /node-groups/:id/resolution/assign**：批量分配解析线路
- **POST /node-groups/:id/resolution/action**：线路解析动作

**现有行为（Go 代码）**
- 列表：
  - 先 `ensureNodeGroupCnameDomainColumn()`，缺列则自动补 `cname_domain`
  - `keyword` 支持 `id/name/cname_hostname/des`，`region_id` 精确过滤
  - 分页 `page/limit`，默认 1/20
  - 读取后 `applyNodeGroupPolicy()`：从 `backup_switch_policy` JSON 回填 `ipv4_resolution/l2_config/sort_order`
  - 额外统计：`node_count`（`lines` distinct node_id）、`site_count`、`forward_count`
- 创建：
  - `region_id=0` 视为 `NULL`
  - `cname_domain` 为空时自动取首条 `cname_domains`（按 id 升序），并校验域名合法且存在
  - `cname_hostname` 归一化：若等于 `cname_domain` 变为 `@`；若包含后缀则去后缀
  - `cname_hostname` 为空则生成 8 位唯一 token
  - `ipv4_resolution` 为空自动生成 8 位 token
  - `backup_switch_policy` 存储 `ipv4_resolution/l2_config/sort_order`
  - 创建后 `BumpConfigVersion("node_group")`
- 更新：
  - 读取原记录并 `applyNodeGroupPolicy`
  - `cname_domain` 未提供则沿用旧值；仍需合法且存在
  - `cname_hostname/ipv4_resolution` 为空则沿用旧值
  - 更新 `backup_switch_type/backup_switch_policy` + `update_at`
  - 更新后 `BumpConfigVersion("node_group")`
- 删除：
  - 若 `lines` 有绑定 → 禁止删除
  - 若 `packages/user_packages` 使用（`node_group_id` 或 `backup_node_group`）→ 禁止删除
  - 删除后 `BumpConfigVersion("node_group")`
- `GET /resolution`：
  - `line_id` 为空默认 `default`；`line_id=all` 表示查询全部
  - `assigned`：按 `line` 表构建，字段含 `node_is_on/is_on/is_backup/weight/sort_order/online(90s)` 等
  - `available`：仅 `enable=true` 且存在 `region_id` 的节点；若组配置 `region_id` 则必须匹配；排除已被其他组占用或当前已分配 IP
- `POST /resolution/assign`：
  - 仅允许启用节点；节点必须配置 `region_id` 且合法
  - 若组设 `region_id`，则节点必须同区域
  - 禁止节点在多个线路组重复分配（冲突检查）
  - 创建 `lines`（`line_id` 默认 `default`，`weight=1`，`enable=true`）
  - 创建后写 IP 切换日志 + `BumpConfigVersion("line")`
  - 同步 DNS：`dns.SyncLineRecords(add)` + `SyncPackageCnameForLineChange(add)`
  - DNS 同步失败仍返回 HTTP 200，但 `code=1`
- `POST /resolution/action`：
  - `action` 支持：`enable/disable/delete/set_backup/unset_backup/set_backup_default/unset_backup_default/set_weight/set_sort`
  - `delete` 支持延迟删除（`ResolveDeleteConfigDelay` + `QueueLineConfigDeletion`）
  - `set_sort` 实际更新节点 `sort`（非 line）
  - 操作后写 IP 切换日志 + `BumpConfigVersion("line")`
  - DNS 同步：`enable=add`、`disable/delete=delete`、`set_weight/set_sort=resync`

**C# 重构要求**
- 线路组 `cname_domain` 字段必须保留自动补列逻辑（兼容老库）
- `cname_domain`/`cname_hostname` 规范化与生成规则必须与 Go 一致
- `ipv4_resolution/l2_config/sort_order` 继续存放在 `backup_switch_policy` JSON
- 解析/分配/动作的校验与 DNS/CNAME 同步必须保持一致
- `set_sort` 更新的是节点 `sort`（非 line），不可误改
- 统一返回 `{code,message,data}` + i18n，保持 DNS 同步失败返回语义（HTTP 200 + code != 200）


#### 区域（Region）
- **GET /regions**：列表
- **POST /regions**：创建
- **PUT /regions/:id**：更新
- **DELETE /regions/:id**：删除

**现有行为（Go 代码）**
- 列表：
  - 读取 `regions` 表（`id asc`），再叠加 `RegionMeta`（`L2CheckPort/SortOrder`）
  - `L2CheckPort` 默认 80；`SortOrder` 默认 100
  - `RegionMeta` 存储在独立配置（`services.LoadRegionMetaMap`）
- 创建：
  - 必填 `name`；`remark` 可空
  - `L2CheckPort` 默认 80，`SortOrder` 默认 100
  - 先写入 `regions`，再写 `RegionMetaMap`
  - 保存 Meta 失败时直接返回错误（不回滚 region）
- 更新：
  - 必填 `name`
  - 更新 `regions` 后写入 `RegionMetaMap`（无事务）
- 删除：
  - 若 `nodes.region_id` 仍有引用 → 禁止删除
  - 删除 `regions` 后移除 `RegionMetaMap`（失败不回滚）

**C# 重构要求**
- `L2CheckPort/SortOrder` 仍需放在独立配置（不加表字段），默认值保持
- 删除区域前必须检查节点引用
- 写 Meta 的非事务行为需保持（如改动必须写明）
- 统一返回 `{code,message,data}` + i18n


#### DNS / CNAME

- **GET /dns/providers**：DNS 账号列表
- **GET /dns/providers/types**：DNS Provider 类型
- **POST /dns/providers**：创建
- **DELETE /dns/providers/:id**：删除
- **GET /dns/test**：测试 DNS
- **POST /dns/records/fix**：修复记录
- **POST /dns/records/cleanup**：清理无效记录

**现有行为（Go 代码）**
- `dns/test`：取最新一条已绑定 DNS Provider 的 CNAME 域名进行校验；无配置直接返回错误
- `dns/records/fix`：调用 `services.RepairDNSRecords()`，若返回错误列表则 `code=1`
- `dns/records/cleanup`：调用 `services.CleanupInvalidDNSRecords()`，若返回错误列表则 `code=1`

**C# 重构要求**
- 维持 `dns/test/fix/cleanup` 的错误处理语义（HTTP 200 + code 1）
- 其余 Provider/CNAME 逻辑详见后续章节，避免重复实现

#### CNAME 域名

- **GET /cname_domains**：列表
- **POST /cname_domains**：创建
- **PUT /cname_domains/:id**：更新
- **DELETE /cname_domains/:id**：删除

**说明**
- 具体校验、引用检查与联动规则详见 `CNAME /cname_domains` 小节

**现有行为（Go 代码）**
- 仅管理员侧接口（挂在 `/api/v1/admin`），逻辑与 `CNAME /cname_domains` 小节完全一致
- 列表无分页，返回 `data.list`
- 创建/更新会做域名规范化 + 合法性校验 + DNS Provider 存在性校验
- 更新后触发 `ResyncDNSForCnameDomains`（新旧域名都会同步）
- 删除前必须做引用检查；有引用则返回冲突

**C# 重构要求**
- 保持与 `CNAME /cname_domains` 小节一致的校验/引用检查/同步行为
- 统一返回结构 `{code,message,data}` + i18n

#### DNS Provider /dns/providers

**现有行为（Go 代码）**
**作用**
- 管理 DNS 账号（DNSAPI），用于自动创建/更新/清理 CNAME/A 记录等。

**数据模型：DNSAPI**
- `id`：int64
- `uid`：int64，所属用户
- `name`：string，账号名称
- `remark`：string，备注（数据库字段 `des`）
- `type`：string，供应商类型
- `auth`：string，JSON 字符串，存储凭证

**接口行为**
- `GET /dns/providers`
  - Admin：可用 `user_id` 过滤
  - User：仅返回当前用户的 DNS 账号
  - 排序：`id desc`
- `GET /dns/providers/types`
  - 返回支持的类型与所需字段：
    - `aliyun`：`access_key_id`、`access_key_secret`
    - `huawei`：`id`、`secret`
    - `dnsla`：`id`、`secret`
    - `dnspod`：`id`、`token`
    - `dnspod_intl`：`secret_id`、`secret_key`
    - `51dns`：`id`、`secret`
    - `cloudflare`：`email`、`api_key`
    - `godaddy`：`key`、`secret`
- `POST /dns/providers`
  - body：`{user_id?, name, type, credentials}`
  - 校验：
    - `name`/`type` 必填
    - `dnspod_intl` 要求 `credentials` 中包含 `secret_id`/`secret_key`
    - 统一通过 `dns.GetProvider(type, credentials)` 校验凭证
  - `user_id` 为空时默认当前登录用户
- `DELETE /dns/providers/:id`
  - 若存在 `cname_domains.dns_provider_id` 引用，禁止删除
- `GET /dns/test`
  - 取最近一条 `cname_domains` 且 `dns_provider_id <> 0` 的记录进行测试
  - 若未配置 CNAME 或 DNS Provider，返回错误
  - 调用 provider `GetRecords(domain)` 验证连通性与权限
- `POST /dns/records/fix`
  - 修复线路 A 记录与站点 CNAME 记录
- `POST /dns/records/cleanup`
  - 清理线路域名下的无效记录

**依赖关系**
- CNAME 域名必须绑定 DNS Provider，DNS 自动化才能工作
- 删除 DNS Provider 会影响所有绑定的 CNAME 域名

**C# 重构要求**
- DNS Provider 必须校验凭证（沿用 `dns.GetProvider(type, credentials)` 语义）
- 删除 DNS Provider 前必须检查 `cname_domains.dns_provider_id` 引用
- `dns/test` 必须选取可用 CNAME 域名进行校验（无配置时返回明确错误）
- 统一返回结构 + i18n

#### CNAME /cname_domains

**作用**
- 维护可用的 CNAME 域名列表，并绑定 DNS Provider；该列表用于站点/转发/线路组/套餐/用户套餐/计划的 CNAME 生成与 DNS 同步

**数据模型：CnameDomain**
- `id`：int64
- `domain`：string，唯一，经过规范化与校验
- `dns_provider_id`：int64，必填，DNS 账号（DNSAPI）
- `note`：string，备注
- `created_at` / `updated_at`

**域名规范化（normalizeDomainInput）**
- 去前后空格、转小写
- 去掉 `http://` / `https://` 前缀
- 截断 `/`、`?`、`#` 之后内容
- 去掉端口（`:` 之后）
- 去掉末尾 `.`

**域名校验（isValidDomain）**
- 总长度 `<= 253`，至少 2 段
- 单段长度 1~63
- 仅允许 `a-z`、`0-9`、`-`
- 单段不能以 `-` 开头或结尾

**接口行为**
- `GET /cname_domains`：返回 `data.list[]`，无分页总数
- `POST /cname_domains`：必填 `domain`、`dns_provider_id`，且 `dns_provider_id` 必须存在
- `PUT /cname_domains/:id`：同创建校验
- `DELETE /cname_domains/:id`：记录不存在 => `40401`；被引用 => `40901`；引用校验失败 => `50001`

**现有行为（Go 代码）**
- 每次请求都会先 `ensureCnameTable()`：若表不存在自动创建；若缺 `dns_provider_id` 列则补列
- `domain` 统一 `normalizeDomainInput`：去协议/路径/查询/片段/端口/末尾点，转小写再校验
- `domain` 全局唯一（`UNIQUE KEY idx_cname_domains_domain`）
- 创建/更新必须校验 DNS Provider 存在；`dns_provider_id=0` 直接拒绝
- 删除前执行“引用检查范围”中的所有表查询；只要有引用则禁止删除
- 返回格式目前为 `{code,msg,data}`，未统一 `{code,message,data}`（重构时必须统一）

**C# 重构要求（强制）**
- 使用 SqlSugar 实体映射 `cname_domains`（保持字段名 `domain/dns_provider_id/note/create_at/update_at` 与唯一索引）
- 复用同样的 `normalizeDomainInput` + `isValidDomain` 规则，保证与现有行为一致
- 删除时必须检查引用范围（Site/Forward/NodeGroup/Package/UserPackage/Plan），有引用直接返回冲突码
- 所有响应使用统一结构 `{code,message,data}`，`message` 通过 i18n 输出
- 创建/更新必须写 `create_at/update_at`（与现有表字段一致），不得引入新字段

**引用检查范围（删除）**
- `sites.cname_domain`
- `forwards.cname_domain`
- `node_groups.cname_domain`
- `packages.cname_domain`
- `user_packages.cname_domain`
- `plans.cname_domain`
- 注：仅对存在该列的表执行检查（`HasColumn`）

**与 DNS Provider 的关系**
- 删除 DNS Provider 前必须保证没有 CNAME 域名引用（`dns_controller` 中阻止删除）

**联动与默认行为（跨模块）**
- NodeGroup
  - 创建/更新通过 `resolveNodeGroupCnameDomain` 校验域名必须存在于 CNAME 列表
  - 未指定时自动选取首条 CNAME 域名（按 ID 升序）；若未配置则报错 `cname domains not configured`
  - `cname_hostname` 为空时自动生成 8 位 token，并保证唯一
  - `cname_hostname` == `cname_domain` 时归一化为 `@`
- Site
  - 创建时若用户套餐 `cname_mode=package` 且 `cname_hostname` 存在，使用套餐主机名 + 套餐域名
  - 否则使用套餐域名（若空则默认 `cdn.node.com`）并以站点首域名拼接生成 CNAME
  - 更新时按 `computeSiteCnameHostname` 重新计算；`cname_domain`/`cname_mode`/`cname_hostname` 变化会触发 `ResyncSiteCnameForSite`
- Forward
  - `cname_mode=package` 时使用用户套餐 CNAME；否则若 CNAME 为空会生成唯一主机名；域名默认 `cdn.node.com`
- Package / Plan / UserPackage
  - 存储 `cname_domain` / `cname_hostname` / `cname_hostname2` / `cname_mode`
  - 当前代码未对 `cname_domain` 做存在性校验（建议 C# 重写时统一校验）
  - 用户套餐更新后触发 `SyncUserPackage`，并可能影响站点/转发 CNAME 生成
#### 监控 /monitor_config

**接口**
- `GET /monitor_config`：获取节点监控配置
- `POST /monitor_config`：更新节点监控配置

**数据模型（NodeMonitorConfig）**
- `notification_period`：string，通知时间段（默认 `8-22`）
- `notify_method`：string，通知方式（默认 `email sms`）
- `notify_msg_type`：string，消息类型集合（默认 `node_ip_dns bandwidth monitor backup_ip backup_default_line backup_group`）
- `email`：string，通知邮箱
- `phone`：string，通知手机号
- `bw_exceed_times`：int，带宽超限次数阈值（默认 2）
- `auto_switch_enable`：bool，是否启用自动切换（默认 false）
- `auto_switch_threshold`：int，自动切换阈值百分比（1~100，默认 90）
- `auto_switch_duration`：int，阈值持续秒数（默认 30）
- `auto_switch_recover`：int，恢复阈值持续秒数（最小 300，默认 300）
- `auto_switch_min_weight`：int，自动切换最小权重（最小 1，默认 1）
- `monitor_api`：string，监控 API（默认空）
- `interval`：int，探测间隔秒（默认 30）
- `failed_times`：int，失败次数阈值（默认 3）
- `failed_rate`：string，失败率阈值（默认 `50`）

**存储与默认值**
- 存储表：`config`
- `name`：`node_monitor_config`
- `type`：`system`
- `value`：JSON 序列化后的 NodeMonitorConfig
- 不存在记录时返回默认值，并在更新时创建记录（`enable=true`，`task_id` 允许 NULL）

**规范化规则（normalizeNodeMonitorConfig）**
- `auto_switch_threshold`：<=0 或 >100 => 90
- `auto_switch_duration`：<=0 => 30
- `auto_switch_recover`：<300 => 300
- `auto_switch_min_weight`：<=0 => 1

**接口行为**
- `GET /monitor_config`
  - 若 `config` 记录不存在：返回默认配置
  - 若存在：反序列化 JSON 并规范化后返回
- `POST /monitor_config`
  - body：NodeMonitorConfig JSON
  - 若记录不存在：创建新记录
  - 若存在：更新 `value` 与 `update_at`

**错误与返回**
- JSON 解析失败：`400` + `Invalid JSON`
- DB 读写失败：`500` + `Database Create Error` / `Database Save Error`
- 成功：`{code:0, msg:"Monitor Config Updated"}` / `GET` 返回 `{code:0, data:cfg}`

**现有行为（Go 代码）**
- 仅使用 `config(name=node_monitor_config,type=system)` 存储，未区分用户
- `notify_msg_type` 为**空格分隔字符串**，前端通过分隔/合并维护
- 仅对 `auto_switch_*` 做规范化，其余字段不做强校验
- 返回结构仍为 `code=0` / `msg`（未统一）

**C# 重构要求**
- SqlSugar 实体映射 `config`，保持 `name/type/value` 语义不变
- GET 不存在时返回默认值；POST Upsert 并保持 `update_at` 写入
- `notify_msg_type` 继续使用空格分隔协议（前后端兼容）
- 统一返回结构 `{code,message,data}` + i18n

#### 日志 /logs

**通用参数**
- 分页：`page`（默认 1）、`pageSize`（默认 20）
- 时间区间：
  - `timeRange[]` / `timeRange`（两个值，格式 `YYYY-MM-DD HH:mm:ss`）
  - `start_time`/`end_time`（支持同格式或 Unix 秒时间戳）

**GET /logs/login**（管理员）
- 来源表：`login_log` 左联 `user`
- 过滤：`keyword`（匹配 `user.name` 或 `login_log.ip`）
- 时间：`login_log.create_at`
- 返回字段：`id`、`user_id`、`username`、`ip`、`success`、`post_content`、`created_at`

**GET /logs/operation**（管理员）
- 来源表：`op_log` 左联 `user`
- 过滤：`keyword`（匹配 `user.name` / `op_log.action` / `op_log.content` / `op_log.ip`）
- 时间：`op_log.create_at`
- 返回字段：`id`、`user_id`、`type`、`action`、`content`、`diff`、`ip`、`process`、`description`、`created_at`、`username`
- 说明：`description` 取 `op_log.content`

**GET /user/logs/operation**（用户）
- 仅返回当前用户（`uid = userID`）
- 过滤：`keyword`（匹配 `action` / `content` / `ip`）
- 返回字段同 `OpLogRow`，但不联表用户名（`username` 可能为空）

**GET /logs/backup**（管理员）
- 来源表：`task`，`type = backup`
- 过滤：
  - `keyword` 若命中状态：`1/success/ok/done` => `state=done`；`0/fail/failed/error` => `state=fail`
  - 否则模糊匹配 `state` / `ret`
- 时间：`create_at`
- 返回字段：`id`、`created_at`、`finished_at`（`end_at` 优先，否则 `start_at`）、`status`（done=1 else 0）、`result`（`ret`）
- 计数异常时返回空列表与 total=0（code=0）

**GET /logs/mail**（管理员）
- 来源表：`message`
- 过滤：`keyword`（匹配 `title`/`content`，若是数字则匹配 `id` 或 `uid`）
- 时间：`create_at`
- 返回字段：`message_id`、`user_id`（`receive`）、`subject`（`title`）、`medium`（Email/SMS）、`fails`（固定 0）、`status`（EmailIsSent 或 PhoneIsSent => 1）、`reason`（空）、`created_at`

**GET /logs/access**（管理员/用户）
- 数据源：ClickHouse `node_access_logs`（未启用 ClickHouse 时直接返回空列表）
- 用户模式：根据用户站点 HostFilter 过滤；无有效过滤时返回空列表
- 可选过滤：
  - `domain` + `domain_mode`（`fuzzy` => `host LIKE`，否则 `host =`）
  - `client_ip`（`remote_addr`）
  - `uri` + `uri_mode`（`exact` => `uri =`，否则 `uri LIKE`）
  - `method`、`status`、`status_min`、`status_max`
  - `node_id`、`node_ip`、`port`（`host LIKE %:port`）
  - `scheme`、`cache_status`、`referer`、`user_agent`、`ssl_protocol`、`ssl_cipher`、`keyword`（host/uri/remote_addr 模糊）
- 时间：`ts`（使用通用时间参数）
- 返回字段：`timestamp`、`node_id`、`node_ip`、`remote_addr`、`host`、`method`、`uri`、`status`、`bytes`、`request_time`、`upstream_addr`、`upstream_response_time`、`upstream_cache_status`、`http_referer`、`http_user_agent`、`scheme`、`ssl_protocol`、`ssl_cipher`
- 脱敏逻辑：
  - 管理端：若 `remote_addr` 非蜘蛛 IP，则 `upstream_addr` 置空
  - 用户端：`upstream_addr` 与 `node_ip` 始终置空

**GET /logs/block/current**（管理员）
- 当前封禁 IP 列表（默认范围 `7d`）
- 参数：`page`/`pageSize`（默认 10，最大 200）、`type`（`ip`/`site_id`）、`keyword`、`range`/`time_range`
- 返回字段：`id`、`site_id`、`domain`、`ip`、`location`、`filter`（HTTP_状态）、`block_time`、`release_time`（固定 `PERMANENT`）

**GET /logs/block/stats**（管理员）
- 封禁统计（默认范围 `7d`）
- 参数：`page`/`pageSize`、`range`/`time_range`
- 返回字段：`site_id`、`domain`、`count`

**GET /logs/block/history**（管理员）
- 封禁历史（默认范围 `7d` 或 `start_time`/`end_time` 自定义）
- 参数：`page`/`pageSize`、`type`（`ip`/`site_id`/`time_range`）、`keyword`、`start_time`、`end_time`
- 返回字段：`id`、`site_id`、`domain`、`ip`、`location`、`filter`（HTTP_状态）、`block_time`、`is_manual`（固定 `false`）

**公共说明**
- Block 日志使用 HostFilter：用户仅可见自身站点，管理员可见全部
- `location` 由 IP 归属地解析（国家-省份），无结果返回 `-`

**现有行为（Go 代码）**
- 登录/操作/备份/邮件日志来自 MySQL；访问/封禁日志来自 ClickHouse
- ClickHouse 未启用时：访问与封禁日志直接返回空列表（`code=0`）
- 管理端访问日志对非蜘蛛 IP 隐藏 `upstream_addr`，用户端同时隐藏 `upstream_addr` 与 `node_ip`

**C# 重构要求**
- 过滤/分页/时间范围语义与字段保持完全一致（含 `keyword` 的字段映射）
- HostFilter 与脱敏规则保持现有行为不变
- ClickHouse 未启用时返回空结构（不报错）
- 统一返回结构 `{code,message,data}` + i18n

#### 消息 /messages

**现有行为（Go 代码）**
**数据表**
- `message`：消息主体
- `message_read`：用户已读标记
- `message_sub`：用户订阅偏好

**默认消息类型**
- `package-expire`（套餐到期）
- `traffic-exceed`（流量超限）
- `connection-exceed`（连接超限）
- `bandwidth-exceed`（带宽超限）
- `cc-switch`（CC 规则开关）
- `cert-expire`（证书到期）
- `refresh_url`（URL 刷新）
- `refresh_dir`（目录刷新）
- `preheat`（预热）

**类型名称本地化（typeLabel）**
- `package-expire` -> `message.package_expire`
- `traffic-exceed` -> `message.traffic_exceed`
- `connection-exceed` -> `message.conn_exceed`
- `bandwidth-exceed` -> `message.bandwidth_exceed`
- `cc-switch` -> `message.rule_switch`
- `cert-expire` -> `message.cert_expire`
- `refresh_url` -> `message.refresh_url`
- `refresh_dir` -> `message.refresh_dir`
- `preheat` -> `message.preheat`
- 兜底 -> `message.other`

**标题规范（normalizeMessageTitle）**
- `title` 为空：使用类型默认标题（`typeLabel`，本地化）
- `title` 仅包含 ASCII：仍使用类型默认标题（多语言兼容）
- `title` 含非 ASCII：保留原始标题

**GET /messages**（管理员）
- 路径：`/api/v1/admin/messages`
- 参数：`page`（默认 1）、`pageSize`（默认 20）、`type`、`keyword`
- 过滤逻辑：
  - `type` 精确匹配
  - `keyword` 模糊匹配 `title/content`；若 `keyword` 可转为数字，则额外 **OR** `site_id = keyword`
- 返回字段（`messageRow`）：
  - `id`、`type`、`type_label`、`title`、`content`、`phone`（`phone_content`）、`site_id`
  - `created_at`（字符串 `YYYY-MM-DD HH:mm:ss`）
  - `is_read` 固定 `false`
- **注意（现有缺陷）**：`keyword` 为数字时使用 `OR site_id = ?`，会绕开已叠加的 `type` 过滤（GORM `Or` 语义）。重写时需用分组条件确保 `type` 仍生效。

**GET /user/messages**（用户）
- 路径：`/api/v1/user/messages`
- 仅返回 `receive = 当前用户`
- 过滤参数同管理员
- `is_read` 依据 `message_read` 是否存在
- **注意（现有缺陷）**：`keyword` 为数字时同样存在 `OR site_id = ?` 绕开 `receive/type` 的问题。重写必须修正为 `(receive=uid AND type=?) AND (title/content LIKE ? OR site_id=?)`。

**GET /user/messages/unread**
- 路径：`/api/v1/user/messages/unread`
- 返回：
  - `count`：未读数量
  - `latest`：最新未读消息（无则空对象）
- 读取逻辑：`message` LEFT JOIN `message_read`，`m.receive = uid AND r.msg_id IS NULL`

**POST /user/messages/:id/read**
- 路径：`/api/v1/user/messages/:id/read`
- 行为：写入 `message_read(uid,msg_id,create_at)`
- **注意（现有缺陷）**：未校验消息是否属于该用户；重复写入不会报错。重写需补充归属校验与幂等处理。

**GET /user/message_sub**
- 路径：`/api/v1/user/message_sub`
- 返回用户订阅列表
- 若无订阅：返回默认类型列表，`phone=true`、`email=true`
- 字段：`msg_type`、`name`（本地化标题）、`phone`、`email`

**PUT /user/message_sub**
- 路径：`/api/v1/user/message_sub`
- body：`{list:[{msg_type, phone, email}]}`
- 行为：事务内先删除旧订阅，再写入新订阅（全量覆盖）

**错误与返回**
- 现有实现：`code=0` 表示成功；异常使用 `code=400/500` + `msg`
- **重写要求（统一响应）**：`{code:200,message:"...",data:...}`；`message` 根据 `Accept-Language` 自动本地化

**C# 重构要求**
- 修复 `keyword` 数字触发的 `OR site_id` 绕过过滤问题（必须保留 `type/receive` 条件）
- `messages/:id/read` 必须校验消息归属并保证幂等（重复读不报错）
- 所有消息相关接口统一返回结构 + i18n

#### 统计 /stats

**现有行为（Go 代码）**
**通用规则**
- 依赖 ClickHouse（未启用时返回空列表或零值结构）
- 时间范围（`services.ResolveStatsRange`）
  - `range` / `time_range`：`today` / `yesterday` / `7d` / `30d` / `last_month` / `10min` / `1h` / `custom`
  - `custom` 需要 `start_time`/`end_time` 或 `timeRange[]`/`timeRange`（格式 `YYYY-MM-DD HH:mm:ss`）
  - 默认 `30min`
- 用户请求自动套用 HostFilter（`services.LoadHostFilter(uid)`），HostFilter 为空时直接返回空结果

**GET /stats/ranking**（管理员/用户）
- 路径：
  - 管理员：`/api/v1/admin/stats/ranking`
  - 用户：`/api/v1/user/stats/ranking`
- 参数：
  - `type`：`domain` / `url` / `ip` / `referer` / `country` / `province` / `latency`
  - `keyword`：模糊匹配（不同类型匹配字段不同）
  - `range` / `time_range` + `start_time`/`end_time`
- 榜单大小：`res_rank_size`（系统配置，默认 100；读取 `config(type=system, scope=global)`）
- 返回 `list[]`：
  - 非 `latency`：`rank`、`item`、`request_count`、`out_traffic`、`origin_traffic`（格式化字符串）
  - `latency`：`rank`、`item`（`host+uri`）、`request_count`、`avg_time`、`max_time`、`min_time`、`p95_time`（秒，保留 3 位小数）
- 类型映射：
  - `domain`：`host`
  - `url`：`host + uri`
  - `ip`：`remote_addr`
  - `referer`：`http_referer`（空值归一为 `-`）
  - `country/province`：`LookupIPRegion` 归属地统计

**GET /stats/basic**（管理员/用户）
- 路径：`/api/v1/*/stats/basic`
- 返回：
  - `x_axis`：时间刻度
  - `bandwidth`：Mbps（按 bucket）
  - `traffic`：MB（按 bucket）
  - `qps`：请求数 / bucket 秒数

**GET /stats/quality**（管理员/用户）
- 路径：`/api/v1/*/stats/quality`
- 返回：
  - `x_axis`
  - `hit_rate`：`HIT / requests * 100`
  - `status_4xx` / `status_5xx`

**GET /stats/origin**（管理员/用户）
- 路径：`/api/v1/*/stats/origin`
- 返回：
  - `x_axis`
  - `origin_bandwidth`：回源带宽 Mbps
  - `origin_traffic`：回源流量 MB

**GET /stats/node_traffic**（管理员）
- 路径：`/api/v1/admin/stats/node_traffic`
- 参数：`window` = `1d` / `7d` / `30d` / `custom`
- 返回：`x_axis`、`in_traffic`、`out_traffic`
- C# 实现：基于 ClickHouse `node_metrics` 统计真实流量

**GET /stats/node_ranking**（管理员）
- 路径：`/api/v1/admin/stats/node_ranking`
- 参数：`metric` = `bandwidth` / `connection` / `load` / `disk`，`window` = `1m` / `5m` / `30m` / `1h`
- 返回：`list[]`（`rank`、`node`、`nic`、`out`、`in`，带单位字符串）
- C# 实现：基于 ClickHouse `node_metrics` 统计真实排行

**GET /stats/node_metrics**（管理员）
- 路径：`/api/v1/admin/stats/node_metrics`
- 参数：
  - `metric` = `bandwidth` / `connection` / `load` / `disk`
  - `window` = `1h` / `6h` / `12h` / `custom`
  - `custom` 需要 `start_time`/`end_time`
- 返回：`list[]`（`time`、`value`）
- C# 实现：基于 ClickHouse `node_metrics` 统计真实指标

**GET /usage**（用户）
- 路径：`/api/v1/user/usage`
- 参数：`range`（默认 `today`）
- 返回：
  - `x_axis` / `values` / `list[{time,value}]`
  - `total` / `avg` / `peak`
  - `unit`：总量 >= 1GB 时为 `GB`，否则 `MB`

**返回结构**
- 现有实现：`code=0` + `data`
- **重写要求（统一响应）**：`{code:200,message:"...",data:...}`；`message` 根据 `Accept-Language` 自动本地化

**C# 重构要求**
- 时间范围解析与 `ResolveStatsRange` 逻辑保持一致（range/时间参数优先级不变）
- HostFilter 规则必须复用现有实现（用户无有效 HostFilter 时返回空结果）
- 统计口径与单位保持一致（Mbps/MB/秒/毫秒）
- 节点相关统计（node_traffic/node_ranking/node_metrics）必须替换为真实数据源
- 统一返回结构 `{code,message,data}` + i18n

#### Dashboard

**现有行为（Go 代码）**
**GET /dashboard**（管理员/用户）
- 路径：
  - 管理员：`/api/v1/admin/dashboard`
  - 用户：`/api/v1/user/dashboard`
- 聚合返回字段：
  - `user`：用户信息
  - `stats`：概览统计（格式化字符串）
  - `charts`：趋势曲线
  - `top_domains` / `top_urls` / `top_ips` / `top_countries`
  - `announcements`：公告列表
  - `package`：当前用户套餐使用情况（仅用户）
  - `resources`：资源数量统计（域名/转发/证书/套餐）
  - `ops`：运营概览（仅管理员）
  - `system_status`：系统状态（节点/ClickHouse/Agent）
  - `license`：授权信息（Go 固定返回占位结构）
    - `total_nodes`：节点总数（主节点数）
    - `current_nodes`：在线节点数
    - `expire_at`：固定 `"-"`（当前无真实授权过期时间）

**时间范围参数**
- `overview_range`：概览统计范围（默认 `today`）
- `chart_range`：趋势范围（默认 `today`）
- `ops_range`：运营统计范围（默认 `7d`）
- 若未传 `overview_range/chart_range/ops_range`，会回退到 `range` 参数

**HostFilter 行为**
- 仅用户请求会加载 HostFilter
- 用户且 HostFilter 为空：`stats/charts/top_*` 返回空结构

**user 字段**
- `role`：`admin` / `user`
- `username` / `id`
- `level`：固定 `V0`
- `auth_state`：`dashboard.auth_verified` / `dashboard.auth_unverified`
- `last_login` / `login_ip`：来自 `login_log` 最近一次成功登录
- `avatar`：当前为空字符串

**stats（概览）**
- 字段：
  - `bandwidth_peak`：峰值带宽（格式化字符串，Mbps）
  - `requests`：请求数（格式化字符串）
  - `traffic`：流量（格式化字符串）
  - `blocked_ips`：封禁 IP 数（格式化字符串）
- 数据来源：ClickHouse 访问统计

**charts（趋势）**
- 字段：`x_axis`、`bandwidth`（Mbps）、`requests`、`traffic`（MB）、`blocked`
- 数据来源：ClickHouse bucket 系列

**Top 列表**
- `top_domains` / `top_urls` / `top_ips` / `top_countries`
- 固定使用最近 `30min` 统计
- 每类 Top 10：`name`、`count`、`traffic`

**C# 重构要求**
- 字段名与结构保持不变（`user/stats/charts/top_* /announcements/package/resources/ops/system_status/license`）
- 时间范围参数优先级保持一致（`overview_range/chart_range/ops_range` 优先，其次 `range`）
- Top 列表固定 `30min`，每类 10 条；公告固定 5 条（仅 `is_show=true`）
- 用户 HostFilter 为空时 `stats/charts/top_*` 返回空结构（不报错）
- 统一返回结构 `{code,message,data}` + i18n

**announcements**
- 来源表：`message`（`type = announcement` 且 `is_show = true`）
- 返回：`id`、`title`、`time`（`YYYY-MM-DD`）

**package**
- 仅用户返回
- 取最新未过期套餐（`user_package`，`is_expired = false`）
- 计算区间：`start_at` -> 当前；`start_at` 无效时默认最近 24h
- `percent` = `used/limit`（最大 100）
- `desc`：`"x.xx GB / x.xx GB"` 或 `"x.xx GB used"`

**resources**
- `domains`：聚合 `site.domain` 去重（去协议/端口/通配）
- `forward`：`forward` 表计数
- `certs`：`cert` 表计数
- `packages`：`user_package` 表计数

**ops（管理员）**
- `summary.users`：新用户数（`user.type <> 1`）
- `summary.packages`：订单 `type in purchase/renew` 且 `state in paid/success/done`
- `summary.recharge`：充值金额（分 -> 元字符串）

**system_status**
- `master`：固定 `true`
- `elastic`：ClickHouse 是否启用
- `agent`：是否所有主节点在线（90s 内心跳）
- `agent_total` / `agent_online`
- `checked_at`：当前时间

**license**
- `total_nodes` / `current_nodes`
- `expire_at`：当前固定 `-`

**返回结构**
- 现有实现：`code=0` + `data`
- **重写要求（统一响应）**：`{code:200,message:\"...\",data:...}`；`message` 根据 `Accept-Language` 自动本地化

#### Global Config

**现有行为（Go 代码）**
**存储与读取**
- 存储表：`config`（SysConfig）
  - `name=global_config`、`type=system`、`scope_name=global`、`scope_id=0`
  - `value` 为 `GlobalConfig` JSON
- `GET /api/v1/admin/global_config`
  - 不存在则写入默认配置（`getDefaultConfig()`）
  - JSON 解析失败时返回默认配置（不会回写）
  - `error_pages` 为空时自动回填默认值

**更新**
- `POST /api/v1/admin/global_config`
  - body：完整 `GlobalConfig`
  - Upsert 到 `config`
  - `BumpConfigVersion("global_config", [])`
  - C# 重构中 `BumpConfigVersion` 会创建 `config_sync` Task，依赖任务派发下发最新配置

**C# 重构要求**
- `GlobalConfigService.GetAsync`：缺失时写入默认值；JSON 解析失败返回默认值；`error_pages` 为空时回填默认值
- `GlobalConfigService.UpdateAsync`：全量覆盖保存 + `BumpConfigVersion("global_config")`
- API 统一返回结构 `{code,message,data,trace_id}` + i18n

**GlobalConfig 结构**
- `waf`（WAFConfig）
  - `enable`=true
  - `default_block_action`=`disconnect`
  - `auto_ipset_enable`=true
  - `auto_ipset_threshold`=200
  - `block_page_rate_limit_enable`=true
  - `block_page_rate_limit`=200
  - `block_page_traffic_free`=false
  - `blacklist_timeout`=3600
  - `temp_whitelist_timeout`=21600
  - `temp_whitelist_limit_total`=400
  - `temp_whitelist_limit_url`=50
  - `whitelist_ips`=`""`
  - `blacklist_ips`=`""`
  - `prevent_tls_handshake`=true
  - `block_unbound_domain`=true
  - `disable_ping`=false
  - `default_page_protection`=`auto`
  - `default_page_protection_threshold`=100
  - `secret_key`=`KPS1CC6oGp`
  - `node_log_clean_strategy`=`none`
  - `cc_rule_auto_switch`=false
  - `anti_cc_image_source`=`system`
  - `anti_cc_image_custom_url`=`""`
  - `anti_cc_type`=`slide`
  - `anti_cc_debug`=false
  - `well_known_protection_threshold`=600
  - `resource_protection_enable`=false
  - `resource_protection_threshold`=50
  - `resource_protection_block_timeout`=3600
  - `resource_protection_rules` 默认 `[{"duration":120,"max_requests":20}]`
  - 兼容字段（未在默认值显式赋值）：`mode`、`policy`、`cc`、`access_control`、`syntactic`
- `nginx`（NginxConfig）
  - `worker_processes`=`auto`
  - `worker_connections`=51200
  - `worker_rlimit_nofile`=51200
  - `worker_shutdown_timeout`=`60s`
  - `log_directory`=`/usr/local/openresty/nginx/logs/`
  - `keepalive_timeout`=60
  - `gzip`=true
  - `custom_snippet`=`""`
- `default_config`（DefaultSiteConfig）
  - `website`：`cache_enable=true`、`cache_ttl=86400`、`gzip=true`、`waf_enable=true`、`ssl_ciphers=<长串>`
  - `api`：`cache_enable=false`、`cache_ttl=0`、`gzip=true`、`waf_enable=true`
  - `download`：`cache_enable=false`、`cache_ttl=0`、`gzip=false`、`waf_enable=true`
- `resources`（GlobalResourceConfig）
  - `website`：
    - `min_limit`=1000
    - `max_limit_multiplier`=200
    - `max_blacklist_ips`=50
    - `max_whitelist_ips`=50
    - `daily_url_purge_limit`=2000
    - `daily_dir_purge_limit`=500
    - `daily_preload_limit`=2000
    - `daily_unlock_ip_limit`=1000
    - `unlock_ip_batch_limit`=50
    - `max_cc_rules_per_group`=5
    - `max_acl_rules`=5
    - `daily_log_download_limit`=10
    - `log_storage_dir`=`/data/download-temp/`
    - `log_storage_hours`=12
    - `max_domains_per_site`=100
    - `default_listen_80`=true
  - `forward`：
    - `disabled_ports`=`80 443`
    - `min_limit`=1000
    - `max_limit_multiplier`=200
    - `max_acl_rules`=10
  - `public`：
    - `disabled_custom_ports`=`22`
    - `allowed_custom_ports`=`1-65535`
- `error_pages`（map[code]html）
  - 通过 `config_items(type=error_page,name=error-page,scope=global)` 读取 JSON
  - 关键字映射：
    - `400` -> `p400`
    - `403` -> `p403`
    - `502` -> `p502`
    - `504` -> `p504`
    - `traffic_limit` -> `p513`
    - `site_locked` -> `p514`
    - `domain_invalid` -> `host_not_found`
    - `conn_limit` -> `p515`
    - `timeout` -> `p512`
    - `ip` -> `access_ip_not_allow`

**注意**
- `GlobalConfig.Nginx` 为历史字段；节点 Nginx 实际配置来自 `config_items(type=nginx_config,name=nginx-config-file)`

**返回结构**
- 现有实现：成功 `code=0`，错误 `code=400/500` + `msg`
- **重写要求（统一响应）**：`{code:200,message:\"...\",data:...}`；`message` 根据 `Accept-Language` 自动本地化

#### ConfigItem

**数据模型**
- 表：`config`
- 字段：`name`、`value`、`type`、`scope_name`、`scope_id`、`enable`、`create_at`、`update_at`

**GET /config_items（管理员）**
- 路径：`/api/v1/admin/config_items`
- 参数：`type`（可选），`scope_name`、`scope_id`（可选）
  - 同时提供 `scope_name + scope_id` 才做 scope 过滤
  - 任一缺省时不做 scope 过滤（兼容现有 Go 行为）
- 返回：统一结构 `{code,message,data,trace_id}`，`data` 为配置项数组

**POST /config_items（管理员）**
- 路径：`/api/v1/admin/config_items`
- body：
  - `type`
  - `scope_name`
  - `scope_id`（可选，缺省视为 `0`）
  - `items[]`：`{name,value,enable?}`
- 行为：事务内逐项 Upsert（不存在则创建，存在则更新）
- `enable` 为空时默认 `true`
- 成功后：`BumpConfigVersion(\"config_item\", [])`

**GET /config_items（用户）**
- 路径：`/api/v1/user/config_items`
- 参数：`type`（可选）
- 作用域：`scope_name=user`、`scope_id=当前用户`
- 返回：统一结构 `{code,message,data,trace_id}`

**POST /config_items（用户）**
- 路径：`/api/v1/user/config_items`
- body：同管理员，但服务端强制 `scope_name=user`、`scope_id=当前用户`
- 成功后：`BumpConfigVersion(\"config_item\", [uid])`

**常用 type / name**
- `nginx_config`：`nginx-config-file`（JSON，Nginx 主配置）
- `cert_default_config`：`cert_default_type` / `cert_default_dnsapi_type` / `cert_default_dnsapi_data`
- `stream_default_config`：`listen_protocol` / `balance_way` / `proxy_protocol`
- `site_default_config`：通过 `/site_defaults` 维护（本质写入 config 表）
- `error_page`：`error-page`（JSON，错误页模板集合）
- `system`：系统配置键值（如 `res_rank_size`）

**system 常见键（系统设置/通知/登录）**
- 登录/注册：`allow_register`、`login_session_valid_time`、`allow-enable-email-captcha-login`、`allow-enable-sms-captcha-login`
- 登录域名限制：`bind-master-host`、`limit_admin_login_domain`、`limit_user_login_domain`
- 客户端 IP：`master_client_ip_header`
- 登录验证码模板：`email_captcha_templ`（C# 当前仅使用该模板；`phone_captcha_templ`/`sms_config` 为预留配置）
- 维护与域名：`maintain`、`admin_domain`、`user_domain`
- 套餐策略：`package_allow_upgrade`、`package_allow_downgrade`、`package_expire_close_site`、`traffic_excceed_close_site`
- 通知：`notification-period`、`notify-method`、`traffic-exceed-notify`、`traffic-exceeding-notify`、`package-expire-notify`、`package-expiring-notify`、`cert-expire-notify`、`cert-expiring-notify`、`cc-switch-notify`、`bandwidth-exceed-notify`、`conn-exceed-notify`
- 监控通知：`node_monitor_config`

**现有行为（Go 代码）**
- 管理端 GET 仅按 `type` 过滤，**不做 scope 过滤**（即使传了 `scope_name`）
- 管理端 GET 返回：`list` + `debug_type` + `debug_scope_name` + `count`
- 管理端 POST 要求 `type/scope_name`；`scope_id` 为空视为 `0`
- 用户端 GET 固定 `scope_name=user`、`scope_id=uid`
- 用户端 POST 强制 `scope_name=user`、`scope_id=uid`

**C# 重构要求**
- admin 端允许缺省 scope 参数（仅当 `scope_name+scope_id` 同时存在时过滤）
- Upsert 事务化，`enable` 缺省为 `true`
- 更新后严格触发 `BumpConfigVersion`（scope 与现有一致）
- 统一返回结构 + i18n

**返回结构**
- **重写要求（统一响应）**：`{code:200,message:"...",data:[...]}`

#### 系统设置（管理端）配置键映射（web/admin/src/views/settings）

> 说明：均通过 `POST /api/v1/admin/config_items` 保存，`type=system`、`scope_name=global`。以下映射以现有前端组件为准。

**1) BasicConfig.vue**
- `system_info`（JSON）：`sys_name/user_console_title/admin_console_title/footer_link/footer_copyright/favicon_file/logo_file/login_ad_file`
- `bind-master-host`：绑定主控域名（登录域名限制/节点通信使用）

**2) PackageConfig.vue**
- `package_expire_close_site`
- `traffic_excceed_close_site`
- `package_allow_upgrade`
- `package_allow_downgrade`

**3) MaintenanceConfig.vue**
- `maintain`（JSON）：`{enable,msg}`
- `auto_upgrade_agent`

**4) CleaningConfig.vue（键映射）**
- `clean_cache_days` -> `keep-task-log-days`
- `clean_login_log_days` -> `keep-login-log-days`
- `clean_op_log_days` -> `keep-op-log-days`
- `clean_site_log_days` -> `keep-access-log-days`
- `clean_node_monitor_days` -> `keep-node-log-days`
- `clean_traffic_days` -> `keep-traffic-history-days`
- `clean_node_traffic_days` -> `keep-node-traffic-days`（前端使用的键，若库中缺失需补齐）
- `clean_blacklist_days` -> `keep-blacklist-days`（前端使用的键，若库中缺失需补齐）
- `backup_frequency` -> `backup_rate`
- `backup_retention` -> `backup_keep_days`
- `backup_dir` -> `backup_dir`

**5) UserConfig.vue**
- `login_session_valid_time`
- `limit_user_login_domain`
- `limit_admin_login_domain`
- `allow-enable-email-captcha-login`
- `allow-enable-sms-captcha-login`
- `allow_register`
- `register_success_templ`（JSON：`{title,data}`）
- `forget_password_templ`（JSON：`{title,data}`）
- `email_captcha_templ`（JSON：`{title,data}`）
- `phone_captcha_templ`（短信模板字符串）

**6) NotifyConfig.vue**
- `notification-period`（`all`/`custom`）
- `notification-period-custom`（如 `8-22`）
- JSON 配置项（模板内变量以前端组件为准）：
  - `traffic-exceed-notify`
  - `traffic-exceeding-notify`（含 `remain_traffic`/`less` 等阈值字段）
  - `package-expire-notify`
  - `package-expiring-notify`（含 `days` 阈值）
  - `cc-switch-notify`
  - `bandwidth-exceed-notify`
  - `conn-exceed-notify`
  - `cert-expire-notify`
  - `cert-expiring-notify`（含 `days` 阈值）
  - `account-auth2-notify`

**7) OtherConfig.vue**
- `master_client_ip_header`
- `record-repair-enable`
- `dns_rs_protect`
- `max_site_stream_sync_one_time`
- `sync-site-config-scope`（`region`/`group`）
- `res_rank_size`
- `http_proxy`
- `api_key_status`（开启时会额外调用 `/api_key` 与 `/api_key/reset`）
- `tcp_traffic_factor`
- `https_cert` / `https_key`
- `node_health_check`
- `node_max_failed`
- `auto_upgrade_agent`
- `delete_config_delayed`

#### Packages（Agent 升级包）

- GET /packages：AgentPackage 列表

- POST /packages：上传包（multipart/form-data）
- POST /packages/grayscale：灰度比例
- POST /packages/stable：设为稳定版
- GET /packages/nodes：节点版本与状态
- POST /packages/upgrade：同步升级
- GET /packages/upgrade/status：升级进度

**现有行为（Go 代码）**
- 列表来源 `services.ListAgentPackages()`，返回 `version/status/gray_percent/upload_time/filename/size/sha256`
- 上传包：`version` 必填且仅允许 `[a-zA-Z0-9._-]`；文件仅允许 `.zip`/`.tar.gz`
- 文件保存为 `agent_<version>.<ext>`，写入包目录；计算 `sha256` 并 Upsert
- `grayscale/stable` 仅更新包状态，未做额外校验
- 节点列表：`current_version` 来自 `agent_version` 配置；`latest_version` 来自 `ResolveLatestAgentVersion`
- 升级任务：按 `node_ids/group_ids/region_ids` 合并目标，创建 `agent_upgrade` Task 并触发派发
- 升级状态：读取 Task Targets，若 `progress=0` 且 `ret` 可解析则以 `ret.progress/message` 覆盖
- 下载包：按 `version` 定位文件并直接附件下载

**C# 重构要求**
- 版本校验/文件扩展名/文件命名与现有行为一致（避免升级包不兼容）
- `agent_upgrade` payload 字段（`version/file_name/sha256/download_url`）保持不变
- UpgradeStatus 的 `progress/message` 解析逻辑保持一致
- 返回结构统一 `{code,message,data}` + i18n


#### Plan & UserPlan

- GET /plans /plans/:id

- POST /plans

- PUT /plans/:id

- DELETE /plans/:id

- GET /user_plans

- POST /user_plans/assign

- PUT /user_plans/:id

- DELETE /user_plans

**现有行为（Go 代码）**
- Plan 实体直接使用 `package` 表（List 依 `sort asc, id desc`）
- Create/Update 基于 JSON map 按需更新字段；`name/region/line_group` 必填且需合法
- `backup_group` 不允许等于 `line_group`；非 0 时必须存在
- Delete 直接删除 `package`（无引用检查）
- AssignUserPlan：复制 Package 到 `user_package`，`end_at` 由 `duration_months` 或 `end_at` 决定（必须未来时间）
- `record_id` 为空时自动生成并回写；Assign/Update 触发 `SyncUserPackage`
- ListUserPlans：`status` 由 `end_at` 判断（`active/expired`）
- Plan 详情返回字段包含：`http_port/stream_port/cname_domain/cname_hostname2/cname_mode/buy_num_limit/backend_ip_limit/id_verify/before_exp_days_renew/expire/owner`
- Create/Update 字段映射：
  - `desc` -> `des`，`line_group` -> `node_group_id`，`backup_group` -> `backup_node_group`
  - 资源字段：`traffic_limit/bandwidth_limit/connection_limit/domain_limit`
  - 功能字段：`custom_cc_rules/websocket/l2_origin`
  - 价格字段：`price_monthly/price_quarterly/price_yearly`
  - 状态字段：`status` -> `enable`，排序字段 `sort_order`
- AssignUserPlan 生成 `user_package`：
  - `record_id` 为空时生成 8 位随机字符串（最多重试 5 次，需唯一）
  - `end_at` 未传时按 `duration_months`（缺省 1 个月）计算；若已过期则以当前时间为起点
  - `enable_backup` 默认为 `false`
- ListUserPlans：`package_name` 优先取 `package` 表名称，缺失则回退 `user_package.name`；`start_at` 为空时回退 `created_at`
- UpdateUserPlan（管理员）：
  - 支持更新 `name/end_at/region_id/node_group_id/backup_group_id/enable_backup_group`
  - `backup_group_id` 变更时若未显式传 `enable_backup_group`，则自动设置为 `backup_group_id>0`
  - `main_domain_limit` 可独立更新
  - `cname_domain/cname_hostname/cname_mode` 可更新；若 `cname_mode` 变化，先将「现有站点中 `cname_mode` 为空」的记录写回为**旧模式**，避免已落地站点被新模式影响
  - 更新后触发 `SyncUserPackage`
- DeleteUserPlans（管理员批量）：仅删除 `user_package`（不做引用检查）

**C# 重构要求**
- Plan 仍使用 `package` 表语义与字段名（不新增字段）
- 继续执行 `region/line_group/backup_group` 校验与约束
- `AssignUserPlan/UpdateUserPlan` 必须触发 `SyncUserPackage`
- `record_id` 生成与回写逻辑保持一致
- 返回结构统一 `{code,message,data}` + i18n
- 与前端一致的字段：`traffic_limit/bandwidth_limit/connection_limit/domain_limit/http_port/stream_port/custom_cc_rules/websocket/l2_origin/cname_*`



#### Finance

- GET /orders：Order 列表

- POST /recharge：充值

**现有行为（Go 代码）**
- 管理员订单：`keyword` 匹配 `mch_order_no/des`，数字关键字会追加 `OR uid=?`
- 用户订单：仅返回自身 `uid`，可按 `type` 过滤
- 金额存储分（int64），返回时转为元（两位小数）
- 管理员充值：事务内更新 `user.balance` 并创建 `order`（`pay_type=manual,state=paid`）
- 用户充值：创建 `order`（`pay_type=online,state=pending`），不直接加余额

**C# 重构要求**
- 金额换算与展示格式保持一致（分 -> 元，两位小数）
- 管理员充值必须事务化（余额与订单一致）
- 用户充值仅创建待支付订单（不直接更新余额）
- 返回结构统一 `{code,message,data}` + i18n


#### Announcements

- GET /announcements

- POST /announcements

- PUT /announcements/:id

- DELETE /announcements/:id

**现有行为（Go 代码）**
- 实体使用 `message` 表（`type=announcement`）
- 列表支持 `keyword` 模糊匹配 `title/content`
- 返回字段：`id/title/content/is_show/is_red/is_bold/created_at`
- Create/Update/ Delete 直接操作 `message` 表，无额外校验

**C# 重构要求**
- 保持 `type=announcement` 与字段语义不变
- 列表排序按 `id desc`；`created_at` 输出 `YYYY-MM-DD HH:mm:ss`
- 统一返回结构 `{code,message,data}` + i18n


#### Upload

- POST /upload/image：上传图片

**现有行为（Go 代码）**
- 仅允许图片后缀：`.jpg/.jpeg/.png/.ico/.gif/.webp`
- 保存到 `uploads/images`，文件名为 `UnixNano + ext`
- 返回相对 URL：`/uploads/images/<filename>`

**C# 重构要求**
- 文件类型校验与路径保持一致
- 返回 URL 结构不变
- 统一返回结构 `{code,message,data}` + i18n

#### API Key

- GET /api_key：APIKey

- PUT /api_key：更新
- POST /api_key/reset：重置 secret

（Admin/User 同时存在：`/api/v1/admin/api_key`、`/api/v1/user/api_key`）

**现有行为（Go 代码）**
- `ensureAPIKey(uid)`：不存在则创建（`api_key` 16 位、`api_secret` 30 位随机 hex）
- `PUT /api_key` 仅更新 `api_ip`
- `POST /api_key/reset` 仅更新 `api_secret`

**C# 重构要求**
- 生成规则保持一致（长度与字符集）
- `api_ip`/`api_secret` 更新字段保持不变
- Admin/User 共用同一逻辑（基于当前登录 uid）
- 统一返回结构 `{code,message,data}` + i18n


#### 用户资料 / Profile

- GET /profile：获取用户资料
- PUT /profile：更新用户资料
- PUT /password：修改密码

**现有行为（Go 代码）**
- `GET /api/v1/user/profile`：返回 `id/name/email/phone/qq/balance/cert_name/cert_no/cert_verified/white_ip/login_captcha/create_at`
- `PUT /api/v1/user/profile`：更新 `email/phone/qq/cert_name/cert_no/white_ip/login_captcha`
- `PUT /api/v1/user/password`：
  - `current/next` 必填
  - `password_hash=sha256` 时要求 `current/next` 为 64 位 hex；否则返回参数非法
  - 未显式声明但两者看似 64 位 hex 时视为已哈希
  - 校验旧密码后写入新密码（存储为 bcrypt）

**C# 重构要求**
- 字段范围与校验逻辑保持一致
- 密码处理与登录复用同一哈希/校验规则
- 统一返回结构 `{code,message,data}` + i18n


#### WS Dispatch

- POST /ws/dispatch

  - Body：`{node_id,task_type,payload,wait_seconds}`

  - 返回：`{"code":200,"message":"Success","data":{node_id,connected,task_id,state,error}}`

**现有行为（Go 代码）**
- 仅用于测试：若节点未连接返回 `connected=false`
- 发送 `task_dispatch` 后等待 Ack（默认 5s），超时返回 `state=timeout`
- 不落库 Task；仅通过 WS 流转

**C# 重构要求**
- 仅作测试通道，不写入任务表
- 等待 Ack 超时逻辑保持一致（默认 5 秒）
- 统一返回结构 `{code,message,data}` + i18n


#### Domain 管理

- GET /domains：管理员域名列表（Domain 模型）

**现有行为（Go 代码）**
- 管理员列表：`GET /api/v1/admin/domains`，支持 `page/pageSize/keyword`，`keyword` 按 `name` 模糊匹配
- 用户列表：`GET /api/v1/user/domains`，仅返回自身域名
- 预加载 `Origins` 关联
- 创建域名（用户）：`POST /api/v1/user/domains`
  - `name` 必填且对用户唯一
  - 默认 `cname = name + ".cdn.node.com"`，`status = 1`
  - 事务内写入域名 + 回源列表，回源缺省值：`port=80`、`weight=1`、`protocol=http`
- 获取配置：`GET /api/v1/user/domains/:id/config` 返回 `domain/origins/https_on=true/cache_rules(示例)`
- 未提供更新/删除接口

**C# 重构要求**
- 域名与回源的事务落库与默认值保持一致
- `config` 端点结构保持一致（包含示例字段，避免前端依赖断裂）
- 管理员/用户权限边界保持一致
- 统一返回结构 `{code,message,data}` + i18n


#### 用户管理

- GET /users：User 列表

- PUT /users/:id/status：启用/禁用
- PUT /users/:id：更新
- DELETE /users/:id：删除
- POST /users/:id/purge/reset：清空用户清理额度
- POST /users/:id/impersonate：模拟登录

**现有行为（Go 代码）**
- 列表支持 `keyword` 模糊匹配 `name/email/phone/qq/des`，数字关键字追加 `OR id=?`
- 启停：`status=1` => `enable=true`，其余为 `false`
- 删除：直接删除 `user` 记录（无级联清理）
- 清理额度：写 `config(name=purge_usage,type=user,scope=user)`，重置 `refresh_url/refresh_dir/preheat`
- 模拟登录：仅允许 `enable=true` 的用户，生成 JWT（TTL 取 `ResolveLoginSessionTTL`）
- 更新：`AutoMigrate` 后批量更新字段；`password` 写入前会哈希

**C# 重构要求**
- 维持字段更新范围与权限边界；密码必须哈希存储
- `impersonate` 仅允许启用用户并复用现有 JWT 过期策略
- 清理额度写入结构保持一致（同 key/type/scope）
- 支持 `GET /users/:id`（用于前端精确回填）
- 兼容 `size` 作为 `pageSize` 的别名
- 用户表字段以 `db.sql` 为准（不扩展额外列）
- 统一返回结构 `{code,message,data}` + i18n


#### Site

- GET /sites

- POST /sites

- POST /sites/batch

- GET /sites/batch/:id/progress

- POST /sites/batch_update

- POST /sites/batch_action

- POST /sites/apply_cert

- GET /sites/export

- GET /sites/resolve

- GET /sites/:id

- PUT /sites/:id

- GET /domain_usage

**现有行为（Go 代码）**
- 列表：管理员/用户分流，使用统一查询与 `buildSiteListItems`
- 创建：默认填充 `user_package_id`/`dns_provider_id`（若未传）；校验域名数量与唯一性
- CNAME 生成：优先套餐 `cname_mode=package`，否则用套餐 CNAME 域名 + 站点首域名生成主机名
- 应用默认配置：`GetSiteDefaultMapWithGroup` + 全局模板（按 `site_type`）
- 更新：校验用户套餐归属、域名配额、分组权限；`settings` 规范化；事务内更新站点与分组关系
- 批量动作：`enable/disable/delete/clear_cache`；删除前必须先禁用站点
- 删除：同步 DNS 清理记录；删除站点分组关系
- 证书应用：`apply_cert` 支持批量创建证书（含通配符校验与默认 DNSAPI）
- 任何变更触发 `BumpConfigVersion("site")`；CNAME/DNS 变更触发同步

**C# 重构要求**
- 创建站点必须绑定套餐；默认 DNS/CNAME 取现有逻辑
- 删除前必须禁用站点（与现有行为一致）
- 更新/批量操作必须触发配置版本更新与 DNS 同步
- 统一返回结构 `{code,message,data}` + i18n



#### SiteGroup

- GET /site_groups

- POST /site_groups

- PUT /site_groups/:id

- DELETE /site_groups/:id

**现有行为（Go 代码）**
- 列表支持 `page/pageSize/keyword/user_id`，用户端仅可见自身
- 创建：`user_id` 为空时取当前用户（管理员为自身用户）
- 更新：用户端需校验归属；仅更新 `name/remark`（表无 `created_at/updated_at`，Go 写入会被忽略）
- 删除：先删 `merge_site_group` 再删 `site_group`

**C# 重构要求**
- 归属与权限校验保持一致
- 删除顺序与关系清理保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### SiteDefault

- GET /site_defaults

- POST /site_defaults

- PUT /site_defaults/:name

- DELETE /site_defaults/:name

**现有行为（Go 代码）**
- 存储为 `config(type=site_default_config)`，支持 `global/group/user` 三类 scope
- scope 输出：`scope_name=user` 在列表中展示为 `global`
- 列表支持 `scope_name/scope_id` 过滤；管理员无 `user_id` 时返回所有 scope（`scope_id!=0`）
- 支持批量 `data` 写入（`name -> value`），写入时统一 Upsert
- 用户端写入：`scope_name=global` 且 `scope_id=0` 时强制使用 `uid`
- group scope 必须校验分组归属（`scope_id` 为分组 ID）
- 删除按 `name/scope` 定位删除；`global` 等价匹配 `global/user`

**C# 重构要求**
- scope 解析与返回结构保持一致（含 group/user 的展示字段）
- Upsert 与批量更新逻辑保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### UserPackage

- GET /user_packages

**现有行为（Go 代码）**
- 列表：管理员可按 `user_id` 过滤；用户仅自身
- 附加字段：`ipv6/http3_enabled` 从 `config_items(type=user_package_config)` 读取
- `status` 基于 `end_at` 计算（`active/expired`）
- 更新：字段按需更新（含资源限制/CNAME/价格）；更新后触发 `SyncUserPackage`
- 续费：按 `period/months` 延长 `end_at`，并触发 `SyncUserPackage`
- 切换套餐：遵循 `package_allow_upgrade/package_allow_downgrade`；更新后 `SyncUserPackage` + 重新计算站点 CNAME
- 更新字段范围：
  - 资源：`traffic/bandwidth/connection/domain/main_domain_limit/http_port/stream_port`
  - 功能：`custom_cc_rule/websocket`
  - 分组：`region_id/node_group_id/backup_node_group`（用户端仅允许传 >0）
  - 价格：`month_price/quarter_price/year_price`（仅管理员可改）
  - CNAME：`cname_hostname/cname_domain/cname_mode`（用户端为空不覆盖）
  - 过期：`end_at` 支持 `time.DateTime` 或 `YYYY-MM-DD HH:mm:ss`
- 布尔扩展项：
  - `ipv6/http3_enabled` 使用 `config_items(type=user_package_config,scope_name=user_package)` 存储
- 续费逻辑：
  - `period=month/quarter/year` 或 `months`
  - 基准时间为 `max(now,end_at)`，结果写回 `end_at`
- 切换套餐：
  - 升/降级判断：优先按价格（月价/季价/年价折算），否则按资源分值
  - `package_allow_upgrade/package_allow_downgrade` 来自系统配置
  - 更新后执行 `resyncSitesForUserPackage`（仅在站点/套餐 `cname_mode != package` 时触发线路组 CNAME resync）
- 列表返回字段包含：
  - `UserPackage` 原字段（含 `cname_domain/cname_hostname/cname_mode/record_id`）
  - 计算字段：`ipv6`、`status`
  - 字符串字段做 `TrimSpace`，避免空白显示问题

**C# 重构要求**
- `user_package_config` 布尔项保持兼容（ipv6/http3）
- 更新/续费/切换后必须触发 `SyncUserPackage`
- CNAME 变更需复用现有规则（避免影响已落地站点）
- 统一返回结构 `{code,message,data}` + i18n
- 返回字段结构与前端依赖保持一致（`status/ipv6/http3_enabled/cname_*` 必须可用）



#### Cert

- GET /certs

- POST /certs

- PUT /certs/:id

- DELETE /certs/:id

- POST /certs/batch

- POST /certs/wildcard

- POST /certs/batch_action

- POST /certs/reissue

- GET /certs/:id/dns_challenge

- POST /certs/:id/verify_dns

- GET /certs/:id/download

- GET /certs/default_settings

- POST /certs/default_settings

**现有行为（Go 代码）**
- 列表：管理员/用户分流（`queryCerts`）
- 上传/更新：支持上传证书并解析有效期/域名；`type=upload` 时可更新证书内容
- 删除：必须先禁用；否则拒绝
- 批量操作：启用/禁用/自动续期开关/删除；禁用前会检查站点引用
- 证书变更触发 `BumpConfigVersion("cert")`
- 支持 DNS challenge 流程与默认 DNSAPI 设置；证书状态写入 `cert.state/cert.ret`
- DNS-01 判定：`dnsapi>0` 或域名含通配符
- 单个创建（`POST /certs`）非 `upload` 类型会创建 `issue_cert` 任务并进入签发流程
- 证书类型规范化：`upload/self -> upload`，`letsencrypt/zerossl/buypass/google`
- `POST /certs/batch`：
  - `type` 必填；`domains` 必填
  - 含通配符域名必须指定 `dnsapi`
  - 批量创建后触发 `issue_cert` 任务（异步签发）
- `GET /certs/:id/dns_challenge`：从 `cert.ret` 解析 DNS challenge；解析失败返回 `data=null`
- `POST /certs/:id/verify_dns`：校验 TXT 记录存在（`DNS TXT record not found`）
- `POST /certs/reissue`：要求 `acme_email` 已配置，否则拒绝
- `GET /certs/default_settings`：读取 `config(type=system,name=cert_default_settings,scope_name=global/user)`，不存在则创建默认 `{type:system,dnsapi:0}`

**C# 重构要求**
- 删除/禁用前的引用校验保持一致
- 证书状态与任务流转保持一致（`cert.state/cert.ret` + `task.state/task.ret`），HTTP-01 由 Agent 执行、DNS-01 由 API 本地执行
- DNSAPI 自动写 TXT 需对接 Provider；未接入时按 `dns_pending` + `verify_dns` 手工流程
- `default_settings` 行为保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### DNSAPI

- GET /dnsapi

- POST /dnsapi

- PUT /dnsapi/:id

- DELETE /dnsapi/:id

- GET /dnsapi/types

**现有行为（Go 代码）**
- 列表：支持 `keyword` 模糊匹配 `name/type`；用户端仅自身并隐藏他人 `auth`
- 创建/更新/删除：用户端需校验归属
- 类型列表固定返回 `type/name/fields`

**C# 重构要求**
- `auth` 的脱敏与归属校验保持一致
- 类型列表字段顺序与内容保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### Forward

- GET /forwards

- POST /forwards

- PUT /forwards/:id

- POST /forwards/batch

- POST /forwards/batch_update

- POST /forwards/batch_action

**现有行为（Go 代码）**
- 列表：管理员/用户分流，统一查询与 `buildForwardListItems`
- 创建：解析端口/源站；应用默认配置；生成 CNAME；写分组关系
- 更新：重新计算端口/源站/节点组；更新分组关系；同步 CNAME
- 批量创建：支持 `data` 多行；可 `ignore_error`
- 批量更新：按勾选字段批量覆盖；更新后 `BumpConfigVersion`
- 批量动作：启用/禁用/删除（删除时清理分组关系）
- 变更后触发 `BumpConfigVersion("forward")`，并同步 DNS 记录

**C# 重构要求**
- 端口/源站解析、默认配置与 CNAME 生成保持一致
- 批量行为与错误处理语义保持一致（`ignore_error`）
- 统一返回结构 `{code,message,data}` + i18n
- 端口与源站解析兼容 `JSON 数组` + `空格/逗号/分号/换行` 输入；源站支持 `addr|weight|enable` 结构
- `backend_port` 由源站列表提取端口；未指定时从默认配置回填
- `stream_default_config` + `forward_default_settings` 合并后应用，`proxy_protocol` 直接覆盖 `stream.proxy_protocol`
- CNAME 生成：
  - `cname_mode=package` 或套餐继承：优先 `user_package.cname_hostname`/`cname_hostname2`/`record_id` 作为 host
  - 自动模式：生成 8 位唯一 token，避免与 `site`/`stream` 现有 CNAME 冲突
  - 默认 CNAME 域名：`cdn.node.com`（未配置时）
- 列表构建时若 CNAME 为空会即时生成并回写
- 创建/更新/批量创建后：`BumpConfigVersion("forward")` + `ForwardCnameSyncService.SyncAsync` 同步 DNS
- 分组关系存 `merge_stream_group`，更新时先清后建



#### ForwardGroup

- GET /forward_groups

- POST /forward_groups

- PUT /forward_groups

- DELETE /forward_groups

**现有行为（Go 代码）**
- 列表可按 `user_id` 过滤；用户端仅自身
- 创建/更新/删除后触发 `BumpConfigVersion("forward_group")`
- 删除不检查引用（仅删除自身）

**C# 重构要求**
- 权限与字段更新逻辑保持一致
- 统一返回结构 `{code,message,data}` + i18n
- 删除接口接收 `body:{id}`，不做引用检查



#### ForwardDefault

- GET /forward_defaults

- POST /forward_defaults

- DELETE /forward_defaults

**现有行为（Go 代码）**
- 管理员：使用 `config(type=system,name=forward_default_settings,scope_name=global,scope_id=0)` 存储默认项
- 用户：使用 `config_items(type=stream_default_config, scope=user)` 存储覆盖项
- `id` 对管理员为时间戳生成；用户端以 `name` 作为删除键
- 变更后触发 `BumpConfigVersion("forward_default")` 或 `config_item`

**C# 重构要求**
- 管理员/用户存储路径区分保持一致
- `id/id_str` 的兼容处理保持一致
- 统一返回结构 `{code,message,data}` + i18n
- 用户端 `type=stream_default_config, scope_name=user, scope_id=user_id` 写入
- 管理端 `config(type=system,name=forward_default_settings,scope_name=global,scope_id=0)` 写入
- `proxy_protocol/balance_way/listen_protocol` 解析/序列化保持一致



#### Forward 监控

- GET /forward/traffic

- GET /forward/ranking

**现有行为（Go 代码）**
- 统计范围：`range=1h/6h/24h`，按分钟桶聚合
- 用户端限制端口范围（仅自身 forward 端口）
- `keyword` 支持 `port` 或 `port/protocol` 过滤
- 返回 `bandwidth/traffic`（GB/Mbps），榜单 `connections/traffic`

**C# 重构要求**
- 口径与单位保持一致（GB/Mbps）
- 用户端端口过滤保持一致
- 统一返回结构 `{code,message,data}` + i18n
- ClickHouse HTTP DSN 从配置读取（`ClickHouse:Dsn|ClickHouse:HttpDsn`），无配置则返回空列表
- 查询表 `node_stream_logs`，按 `server_port/protocol` 过滤与聚合
- `tcp_traffic_factor` 读取 `config(type=system)` 作为倍率



#### Task（清理/预热）
- GET /tasks：分页列表
- POST /tasks：创建
  - Body：`{type,urls,user_id?}`

- GET /tasks/usage：当日限已用/剩余

- GET /tasks/:id：任务详情
- POST /tasks/:id/resubmit：重试

**现有行为（Go 代码）**
- 任务创建/派发/回写详见第 13 章
- `refresh_url/refresh_dir/clear_cache/preheat` 行为一致，用户受 `purge_usage` 限制

**C# 重构要求**
- 严格遵循第 13/17 章任务状态机与回写规则
- 统一返回结构 `{code,message,data}` + i18n


#### 规则（CC/ACL）
- CC Groups `/rules/cc/groups` CRUD

- CC Matchers `/rules/cc/matchers` CRUD

- CC Filters `/rules/cc/filters` CRUD

- ACL `/rules/acl` CRUD（结构见 ACLController）

**现有行为（Go 代码）**
- CC 规则分为**系统规则**与**用户规则**：
  - 系统规则：`internal=true` 或 `uid=0`
  - 用户规则：`internal=false` 且 `uid>0`
- 用户端创建/更新/删除 CC 规则前必须具备套餐权限：
  - 权限来源：`user_package.custom_cc_rule=true`
  - 校验规则：存在未过期的套餐（`end_at` 为空或大于当前时间）
  - 未开启时返回 `forbidden` + `custom cc rule not enabled`
- 用户端列表/详情可见：
  - 系统规则（`uid=0`）
  - 自己创建的用户规则（`uid=当前用户`）
- 管理端创建规则时：
  - `type=system` → `internal=true` 且 `uid=0`
  - `type=user` → `internal=false` 且 `uid=可选`（未传则为 0）
- 规则以 JSON 结构存储（见 ACL/CC 模型定义）

**CC Groups（cc_rule）**
- List：
  - 过滤：`name` 模糊匹配；`status=on/off` → `enable=true/false`
  - 排序：`sort asc, id desc`
  - 返回字段：`id/user_id/uid/user{name,id}/name/is_system/is_on/is_show/status/sort_order/create_time`
- Get：
  - 返回字段：`id/name/remark/user_id/is_system/type/is_on/is_visible/sort_order/rules/visible_users`
  - `data` 解析：`{rules:[...],visible_users:[...]}`；解析失败返回空数组
- Create：
  - `name` 必填
  - `enable` 默认 `true`（不接受前端传入的 on/off）
  - `data` 写入 `{rules,visible_users}`；`is_visible` 写入 `is_show`
  - 创建后触发 `BumpConfigVersion("cc_rule")`
- Update：
  - 用户端必须为 owner（`uid` 一致），否则 `forbidden`
  - 管理端可切换 `type`（system/user）
  - 更新 `name/remark/data/is_show/sort_order`，不修改 `enable`
  - 更新后触发 `BumpConfigVersion("cc_rule")`
- Delete：
  - **Go 未提供删除接口**（前端已有入口）

**CC Matchers（cc_match）**
- List：
  - 过滤：`name` 模糊匹配；`status=on/off`
  - 排序：`id desc`
  - 返回字段：`id/user_id/uid/user{name,id}/name/is_system/is_on/status/create_time`
- Get：
  - 返回字段：`id/name/remark/is_system/type/is_on/rules`
  - `data` 解析：`{rules:[{item,operator,value}]}`；解析失败返回空数组
- Create/Update：
  - `name` 必填
  - `data` 写入 `{rules}`；`is_on` -> `enable`
  - 用户端必须为 owner
  - 触发 `BumpConfigVersion("cc_match")`
- Delete：
  - **Go 未提供删除接口**（前端已有入口）

**CC Filters（cc_filter）**
- List：
  - 过滤：`name` 模糊匹配；`status=on/off`
  - 排序：`id desc`
  - 返回字段：`id/user_id/uid/user{name,id}/name/is_system/type/is_on/status/create_time`
- Get：
  - 返回字段：`id/type/name/remark/enable/action/match_mode/blacklist/within_second/max_req/max_req_per_uri/auth`
  - `extra` 解析：`{match_mode, blacklist, auth}`，缺失字段返回 `null`
- Create：
  - `name` 必填
  - `action` 写入 `type` 字段
  - `extra` 写入 `{match_mode, blacklist, auth}`（`auth` 为空则不写）
  - 触发 `BumpConfigVersion("cc_filter")`
- Update：
  - 用户端必须为 owner
  - 管理端仅切换 `internal`（不重写 `uid`）
  - 更新 `name/remark/type/within_second/max_req/max_req_per_uri/extra/enable`
  - 触发 `BumpConfigVersion("cc_filter")`
- Delete：
  - 用户端必须为 owner
  - 触发 `BumpConfigVersion("cc_filter")`
- ACL 读取：先解析 `{rules,default_deny_status,default_redirect_url}`；失败则回退为 `[]rules`（默认 `deny_status=0`）
- ACL Create：
  - 用户端 `uid` 来自登录态（为空返回 `user_id is required`）
  - `name` 必填；`default_action` 为空默认 `allow`
  - `data` 统一存 `{rules,default_deny_status,default_redirect_url}`
  - 创建后触发 `BumpConfigVersion("acl")`
- ACL Update：
  - 用户端必须为资源 owner，否则 `forbidden`
  - `default_action` 为空默认 `allow`；更新 `name/des/enable/data`
  - 更新后触发 `BumpConfigVersion("acl")`
- ACL Delete：
  - 用户端必须为资源 owner，否则 `forbidden`
  - 删除后触发 `BumpConfigVersion("acl")`
- CRUD 触发 `BumpConfigVersion`，并影响 EdgeConfig 下发

**C# 重构要求**
- CC 规则权限：
  - 用户端 create/update/delete 必须检查 `custom_cc_rule`（同 Go）
  - 未开启时返回 `permission_denied` + `custom_cc_rule_not_enabled`
- CC Groups：
  - `data` 必须存 `{rules,visible_users}`（snake_case）
  - `enable` 默认 `true`，更新不修改 `enable`
  - 返回补充 `created_at`（Unix seconds）以兼容现有前端 `created_at` 显示
  - **已实现 Delete 接口**（C# 管理端/用户端均已提供）
- CC Matchers：
  - `data` 必须存 `{rules}`（snake_case）
  - 返回补充 `created_at`（Unix seconds）以兼容现有前端 `created_at` 显示
  - **已实现 Delete 接口**（C# 管理端/用户端均已提供）
- CC Filters：
  - `extra` 必须存 `{match_mode,blacklist,auth}`（snake_case）
  - `action` ↔ `type` 字段映射保持一致
  - 返回字段应同时包含 `type` 与 `action`（兼容前端）
- 规则结构与字段名保持一致（不新增/不重命名）
- ACL `data` 必须使用 `default_deny_status/default_redirect_url`（snake_case）落库，确保与历史数据兼容
- 兼容旧数据：若 `data` 仅为 `[]rules` 仍可解析
- 用户端 `user_id` 以登录态为准；未登录统一 `user_id_required`
- 任何规则变更必须触发配置同步
- EdgeConfig 下发：
  - 规则来源：`cc_rule/cc_match/cc_filter` 仅 `enable=true`
  - `cc_rule.data` 同时兼容 `[]` 与 `{rules:[]}` 两种格式
  - `matcher/matcher_id` 与 `filter1/filter_id` 兼容，`state` 缺失默认 `true`
- 统一返回结构 `{code,message,data,trace_id}` + i18n


---



### 9.4 User /api/v1/user


> 大部分与 Admin 接口一致，区别在于权限校验和默认作用域（只允许操作自身资源）


- Domains：GET/POST/GET config

- ConfigItems：GET/POST

- Profile：GET/PUT；Password：PUT

- Recharge：POST /recharge

- Orders：GET /orders

- Dashboard：GET /dashboard

- Logs：GET /logs/operation / logs/access

- Messages：GET /messages / messages/unread / messages/:id/read

- Message subscriptions：GET/PUT /message_sub

- APIKey：GET/PUT/RESET

- Sites / Certs / Tasks / Plans / UserPackages / SiteGroups / SiteDefaults：同 admin 路由

- DNS Providers & DNSAPI：仅允许操作自身资源
- CC/ACL 规则：同 admin

- Stats/Usage：/stats/basic /stats/quality /stats/origin /stats/ranking + /usage

- Forward：同 admin



---



### 9.5 Agent /api/v1/agent


#### POST /agent/heartbeat

- Body：`{node_id?,timestamp,status}`

- 返回：`{status:"pong",sync_action}`

**现有行为（Go 代码）**
- `node_id` 解析顺序：认证 `nodeID` → body `node_id`（可为 ID/name/host）→ `ClientIP` 匹配 `nodes.ip`
- 若解析到节点：`MarkNodeOnline` + 写入监控日志 `WriteNodeMonitorLog(nodeID,"heartbeat",true,clientIP)`
- 若节点 `config_task` 为 `sync_enable/sync_disable` 则返回 `sync_action=enable/disable`
- 即使解析失败也会返回 `status=pong`（无错误）

**C# 重构要求**
- 节点解析顺序与 `sync_action` 逻辑保持一致
- 监控日志必须写入 `heartbeat`
- 统一返回结构 `{code,message,data}` + i18n（`status.pong`）



#### POST /agent/node/sync

- Body：`{node_id?,action,success}`

- 返回：`{status}`

**现有行为（Go 代码）**
- `node_id` 解析：优先认证 `nodeID`，否则 body `node_id`（支持 name/id，仅主节点 `pid=0`）
- `action` 仅允许 `enable/disable`
- 写入监控日志 `WriteNodeMonitorLog(nodeID,"sync",success,clientIP)`
- `success=false`：返回 `status=ignored`，不更新数据库
- `success=true`：清空 `nodes.config_task` 并更新 `update_at`

**C# 重构要求**
- `action` 校验与 `config_task` 清理逻辑保持一致
- 返回 `status.ok/ignored` 语义一致
- 统一返回结构 `{code,message,data}` + i18n



#### GET /agent/config

- Query：`node_id`, `version?`

- 返回：`{code,message,data:EdgeConfig}`
**现有行为（Go 代码）**
- `node_id` 优先取认证 `nodeID`，否则 query `node_id`
- 缺失 `node_id` 返回 400；找不到节点返回 404；生成失败返回 500
- 若传 `version` 且等于配置 `Version`，返回 `304 Not Modified`
- 返回体为 **原始 EdgeConfig JSON**（无统一包装）

**C# 重构要求**
- 保持 `node_id` 解析与错误码语义一致
- 不使用 `304`，统一返回结构并由 Agent 侧比较版本



#### GET /agent/upgrade

- 返回：`{"code":200,"message":"Success","data":{api_version,node_version,auto_upgrade,need_upgrade,should_upgrade}}`

**现有行为（Go 代码）**
- `node_id` 解析失败返回 400（`node_id is required`）
- `api_version`：API 侧内置 agent 版本（`ReadAgentBinaryVersion`）
- `node_version`：节点配置 `agent_version`
- `auto_upgrade`：系统配置 `auto_upgrade_agent`
- `need_upgrade`：`CompareVersion(api_version,node_version) > 0`
- `should_upgrade`：`need_upgrade && auto_upgrade`

**C# 重构要求**
- 字段名与判定逻辑保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### GET /agent/upgrade/package

- Query：`version`

- 返回：下载文件流

**现有行为（Go 代码）**
- `version` 为空返回 400
- `GetAgentPackage(version)` 不存在返回 404
- 目录解析失败返回 500；文件不存在返回 404
- 以附件形式返回文件（`FileAttachment`）

**C# 重构要求**
- 版本校验与文件名/扩展名策略保持一致（避免升级包不兼容）
- 保持附件下载语义（不包 JSON）



#### GET /agent/tasks

- 说明：C# 已实现 HTTP 任务拉取（断线/冷启动兜底），WS `task_dispatch` 仍为主通道
- 返回：`{code,message,data:{tasks:[Task]}}`

**现有行为（Go 代码）**
- 拉取 `enable=true` 且 `state in (waiting,running)` 且 `retry_at <= now` 的任务，按 `id asc`，最多 100 条
- `issue_cert` 任务：若 `res` 指定 `target_node_id`，仅目标节点可领取
- 过滤已处理节点：若 `progress[node_id]` 存在且不为 `fail`，则跳过
- 节点领取后：将任务状态置 `running`，写入 `start_at` 与 `progress`
- `issue_cert` 任务领取时：对应证书 `state` 从 `waiting` 更新为 `issuing`

**C# 重构要求**
- 若保留 HTTP 拉取，以上过滤/更新逻辑必须一致
- 默认以 WS `task_dispatch` 为主，但保留兼容路径
- 统一返回结构 `{code,message,data}` + i18n



#### POST /agent/tasks/:id/finish

- 说明：C# 已实现 HTTP 回写（断线兜底），WS `task_ack` 为主
- Body：`{state,ret}`

- 返回：`{code,message,data:true}`

**现有行为（Go 代码）**
- `state` 为空默认 `done`
- `issue_cert` 且 `ret` 含 `429`：标记节点限流 15 分钟
- 更新任务 `progress`（按 `node_id` 标记状态）
- 追加 `ret` 日志（`{time,node_id,state,message,attempt}`）
- `fail`：`err_times+1`，最多 3 次；未达上限则 `retry_at=now+err_times*30s` 并回到 `waiting`
- 非 `fail`：`deriveTaskState`（全部节点 done 才结束），完成时写 `end_at`
- 任务完成/失败会触发消息通知（`CreateUserMessage`）

**C# 重构要求**
- 失败重试、`retry_at` 与 `ret` 日志格式保持一致
- 任务完成通知规则保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### GET /agent/l2/nodes

- 返回：`{nodes:[{id,ip,port,check_protocol,check_port,check_host,check_path,check_timeout}]}`

**现有行为（Go 代码）**
- 仅 L1 节点（`level=1`）可获取；非 L1 返回空列表
- 先根据自身所在线路组，查找同组 L2 节点（`level=2` 且 `enable=true`）
- 默认 `check_protocol=tcp`
- `check_port` 为空时使用区域 `L2CheckPort`（RegionMeta）

**C# 重构要求**
- L1/L2 过滤与默认值逻辑保持一致
- 返回字段结构不变
- 统一返回结构 `{code,message,data}` + i18n



#### POST /agent/l2/heartbeat

- Body：`{nodes:[...]}`

**现有行为（Go 代码）**
- `nodes` 为空直接返回 `status=ok`
- 对每个节点执行 `MarkNodeOnline` 并写入监控日志 `l2_beat`

**C# 重构要求**
- 保持在线更新与日志写入逻辑一致
- 统一返回结构 `{code,message,data}` + i18n


#### POST /agent/certs/issued

- Body：`{cert_id,cert,key,issue_task_id,rate_limited?,rate_cooldown?}`

**现有行为（Go 代码）**
- `cert_id` 必填；`cert/key` 必填
- 若 `rate_limited=true`，按 `rate_cooldown`（秒，默认 10 分钟）标记节点限流
- 解析证书有效期失败直接拒绝
- 更新证书时对私钥执行加密存储（加密失败会降级存明文）
- 调用 `UpdateIssuedCert` 写入证书与私钥并绑定 `issue_task_id`

**C# 重构要求**
- 证书有效期解析、限流标记、加密策略保持一致
- 更新证书时必须与任务状态联动一致
- 统一返回结构 `{code,message,data}` + i18n



#### POST /agent/acme/tokens

- Body：`{token,value,ttl}`

**现有行为（Go 代码）**
- `token/value` 必填，`ttl` 秒级（默认 15 分钟）
- 写入内存 `AcmeTokens`，供 `/.well-known/acme-challenge/:token` 读取

**C# 重构要求**
- TTL 默认值与存储策略保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### DELETE /agent/acme/tokens/:token

**现有行为（Go 代码）**
- `token` 为空返回 400
- 从 `AcmeTokens` 删除，立即生效

**C# 重构要求**
- 删除逻辑保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### POST /agent/logs/access

- Body：`{node_id,node_ip,lines[]}`

**现有行为（Go 代码）**
- `node_id` 为空时取认证 `nodeID`
- `node_ip` 为空时使用 `ClientIP`
- 调用 `InsertAccessLogs(node_id,node_ip,lines)` 写入 ClickHouse

**C# 重构要求**
- NodeID/IP 补齐逻辑保持一致
- ClickHouse 写入失败不可导致 Agent 崩溃（返回结构统一）
- 统一返回结构 `{code,message,data}` + i18n



#### POST /agent/logs/metrics

- Body：`{node_id,node_ip,content}`

**现有行为（Go 代码）**
- `node_id` 为空时取认证 `nodeID`
- `node_ip` 为空时使用 `ClientIP`
- 调用 `InsertMetrics(node_id,node_ip,content)` 写入 ClickHouse

**C# 重构要求**
- NodeID/IP 补齐逻辑保持一致
- 统一返回结构 `{code,message,data}` + i18n



#### POST /agent/logs/events

- Body：`{node_id,node_ip,type,payloads[]}`

**现有行为（Go 代码）**
- `node_id` 为空时取认证 `nodeID`
- `node_ip` 为空时使用 `ClientIP`
- `type` 为空默认 `event`
- 调用 `InsertEventLogs(node_id,node_ip,type,payloads)` 写入 ClickHouse

**C# 重构要求**
- `type` 默认值与写入逻辑保持一致
- 统一返回结构 `{code,message,data}` + i18n


---



## 10. 关键数据结构（请响应复用）


### 10.1 ACL

- ACLData：`{rules:[{conditions:[{item,operator,value}],action,deny_status,redirect_url}],default_deny_status,default_redirect_url}`



### 10.2 Task WS

- TaskDispatchMsg：`{kind,msg_id,task:{task_id,task_type,task_name,payload}}`

- TaskAckMsg：`{kind,msg_id,node_id,task_id,task_type,status,applied,ret,error}`



### 10.3 L2

- L2Node：`{id,ip,port,check_protocol,check_port,check_host,check_path,check_timeout}`



---



（已根据当前源码尽可能完整展开；如需精确字段可直接对api/models controllers 定义。）



---



## 11. Task 与功能关系图（必读）



### 11.1 任务来源与类型总表



> 任务是“功能执行的异步化载体”。不同功能会生成不同 `task.Type` 或通过 WS 直接派发


**A. 通用任务（由 API 下发，Agent 执行）**

- `refresh_url`：清理单 URL 缓存
  - 入口：`POST /api/v1/admin/tasks` / `POST /api/v1/user/tasks`
  - 执行者：Agent（`processTask`）
  - 影响：清理缓存文件（按 URL hash 定位）

- `refresh_dir`：清理目录（实现上为清空 cache 目录）
  - 入口同上
  - 影响：删除整站 cache 目录

- `preheat`：预热 URL（从本机回源）
  - 入口同上
  - 影响：本机对 `127.0.0.1` 发起 GET 请求

- `config_sync`：配置同步
  - 入口：任务派发系统（WS dispatch），或某些更新后触发 `TriggerDispatchPending`
  - 执行者：Agent（`applyConfigPayload` + reload）
  - 影响：刷新 Nginx 配置 & 动态文件

- `package_sync`：套餐同步
  - 入口：`services.UserPackageService` 变更后产生任务
  - 执行者：Agent（写入 `packages/*.json`）

- `issue_cert`：证书签发
  - 入口：证书相关控制器（创建签发任务）
  - 执行者：Agent（ACME）→ 回传证书

- `agent_upgrade`：Agent 升级
  - 入口：后台升级流程（PackageController / services）
  - 执行者：Agent（下载、解压、更新资源）


**B. API 内部任务（Worker 执行，不下发 Agent）**

- `DNS_PLATFORM_CNAME_UPSERT` / `DNS_USER_CNAME_UPSERT`
  - 入口：DNS 自动配置（平台用户 CNAME）
  - 执行者：DNS Worker（`services/dns_task_worker.go`）

- `site_create`
  - 入口：站点批量创建 / 后台任务
  - 执行者：SiteCreateWorker（`services/site_task_service.go`）


### 11.2 任务生命周期（统一模型）


1) **创建**  

   - `TaskController.Create` / 其他服务创建 `models.Task`  

   - 默认 `state=waiting`, `enable=true`



2) **派发**  

   - WS 在线：API `task_dispatch` 主动派发  

   - HTTP 拉取仅作为可选扩展（C# 默认不依赖）  

   - 任务支持 **并发派发** + **重试机制**



3) **执行**  
   - Agent `processTask` 执行并上报进度/结果  
   - API 更新 `task.progress`, `task.ret`, `task.state`



4) **结束**  

   - `state=done` `state=fail`  

  - 失败会写`retry_at`，最多 3 次重试


### 11.3 任务与核心功能关系（链路）


**站点/配置相关**

- 站点新增/更新 `BumpConfigVersion` 触发配置变更  

- Agent 获取`EdgeConfig` 写入 `cdn_config.json` reload Nginx  

- 影响：域名配置、缓存规则、ACL/CC、WAF、CORS、SSL、L2 


**证书相关**

- API 创建签发任务 (`issue_cert`)  

- Agent ACME 签发 回传 `/agent/certs/issued` WS `cert_issued`  

- API 更新证书状态/version  

- 证书变更 -> 配置版本变更 -> Agent reload



**套餐相关**

- 套餐创建/变更 `package_sync` 任务  

- Agent 写入 `packages/*.json`  

- 配置生成时使用套餐限制（带宽/连接/域名/L2 等）



**DNS 相关**

- CNAME/解析变更 DNS Task（平台/用户触发）

- DNS Worker 执行 更新 site 记录 ID  

- 节点/线路变更 SyncLineRecords + CNAME 同步



**清理/预热**

- 用户/管理员发起任务

- Task Agent 执行 回写结果  

- 影响缓存和回源行为


**节点启停**

- API 设定 `node.config_task = sync_enable/sync_disable`  

- WS heartbeat_ack 下发Agent  

- Agent 调用 start/stop Nginx  

- 回报 `node_sync`



### 11.4 任务依赖点（建议开发者关注）



- **配置同步依赖** 
  - 站点、证书、ACL/CC、全局配置、资源限制都会影响 `EdgeConfig`

- **DNS 依赖** 
  - NodeGroup/Line 变更、Package CNAME 变更需触发 DNS Sync  

- **状态同步依赖** 
  - Agent Online/Offline 节点状态、监控日志、自动切换逻辑



---



## 12. 开发者应如何梳理功能（建议方法论）


### 12.1 五步法（推荐）


1) **入口识别**  

   - 找到路由入口（`routers/setup.go`）

   - 标记 controller 方法进入具体业务逻辑



2) **模型与状态**  

   - 定位 `models` 对应数据结构  

   - 找出状态字段：`state/enable/version/err_times/retry_at` 


3) **业务链路**  

   - 判断是否触发：`BumpConfigVersion`、`TriggerDispatchPending`、DNS Worker  

   - 识别“同步路径”与“后台任务路径”


4) **Agent 侧影响**  

   - 是否进入 `EdgeConfig` 

   - 是否触发 Nginx 配置生成 / reload 

   - 是否需要写入 `packages` / `resources` / `error_pages` 等本地文件？



5) **验证与回写**  

- 状态回写字段（task.state/task.ret、node.config_task、cert.state/cert.ret 等）

   - 日志或监控是否需要补充（op_log / node_monitor_log / ClickHouse 等）


### 12.2 功能梳理速查


| 功能 | 入口 | 关键服务 | 是否触发 EdgeConfig | 是否任务 | Agent 影响 |

|------|------|----------|--------------------|---------|-----------|

| 站点管理 | SiteController | config_service | | | reload |

| 证书签发 | CertController | cert_issue_service | | | reload |

| 套餐同步 | UserPackageController | user_package_service | | | 本地 packages |

| DNS 解析 | DnsController/DNSAPI | dns_task_worker | | | |

| 清理/预热 | TaskController | task_dispatch | | | 清缓回源 |

| 节点启停 | NodeController | agent_ws | | | 启停 Nginx |



### 12.3 建议的排查顺序


1) **出问题先看是否任务执行成功（Task 状态/ret）**

2) **再看是否已生成最新 EdgeConfig**（版本号是否变化）

3) **再看 Agent 是否 reload**（日志 + nginx.conf）

4) **最后确认日志上报或 DNS 同步是否完成**



---



（新增：任务关系与开发梳理方法）



---



## 13. Task 追踪表（创建/派发/执行/回写）


> 目的：给开发者一个“从入口到执行”的完整追踪视图，避免遗漏任务与功能的耦合


### 13.1 Task 状态与字段约定



**核心字段**（`models.Task`）：

- `Type`：任务类型
- `Data`：任务 payload（多为 JSON 或 URL 列表）
- `Res`：任务元信息（如 `user_id` / `target_node_id`）
- `TargetsJSON`：多节点任务状态（`TaskTargets` JSON）
- `State`：状态字符串（注意存在多种值，见下）
- `Progress`：单任务进度字典（旧版 targets）
- `ErrTimes` / `RetryAt`：重试控制
- `Ret`：结果或错误



**状态值（实际代码中出现）**
- `waiting` / `running` / `retrying` / `success` / `fail` / `done` / `pending`

- 注意
  - `TaskController` 使用 `waiting/running/done/fail` 语义
  - `dns_task_worker` 使用 `pending/retrying/success/fail`
  - `cert_issue_service` 使用 `waiting/running/retrying/success/fail`


**TargetsJSON 语义**
- 用于“多节点任务”并行派发与回写（`TaskTargets` 结构）
- `AgentWSController.handleTargetsTaskAck` 会按节点更新重试与最终状态


### 13.2 任务类型逐项追踪（创建/派发/执行/回写）


#### A. Agent 执行

**现有行为（Go 代码）**
- 本节任务由 Agent 实际执行（WS/HTTP 派发后在节点本机生效）
- 任务结果通过 `task.progress/task.ret/task.state` 回写，并触发后续消息通知
- 任务输入结构与副作用必须与现网保持一致


**1) refresh_url**

- 创建入口：`TaskController.Create` / `TaskController.Resubmit`

- Data：多 URL（字符串拼接，\n 分隔）
- Res：`{"user_id":<uid>}`（用于权限与消息）
- Targets：无（按节点拉取/派发）
- 派发：`TriggerDispatchPending()` WS/HTTP

- 执行：Agent `processTask` `purgeURLs()`

- 回写：`task.ret` + `task.progress` + `task.state`

- 关联功能
  - URL 校验与域名限制（只能清理已配置域名）

  - 用户每日限额统计（`purge_usage`）


**2) refresh_dir**

- 创建入口：同 `refresh_url`

- Data：URL 列表（接口层要求，但 Agent 实际忽略内容）
- 执行：Agent 直接清空 `WorkDir/cache`

- 影响：全站缓存失效


**3) preheat**

- 创建入口：同 `refresh_url`

- Data：URL 列表

- 执行：Agent 访问 `127.0.0.1:<port>`，带 `Host` 头，触发回源

- 影响：缓存预热


**4) clear_cache**

- 创建入口：`SiteController.AdminBatchAction`（action=clear_cache）
- Data：`{"action":"clear_cache","site_ids":[...]}`

- 执行：Agent `clearCacheDir(cache)`（实际清空全部缓存，忽略 site_ids）
- 注意：这是全局级操作


**5) config_sync**

- 创建入口：`BumpConfigVersion` `NotifyConfigChanged` `createConfigSyncTask`

- Data：`ConfigChange{version,resource,ids,timestamp}`

- Targets
  - 默认 targets（全节点轮询派发）
  - `resource=site/forward`，按 scope 解析出节点 `TargetsJSON`

- 派发前重写 payload
  - `AgentWSController.dispatchPendingTasksForNode` 会将 Data 替换为 `EdgeConfig` JSON（最新配置）

- 执行：Agent `applyConfigPayloadWithOptions` 生成动态配置 + reload

- 关联功能：所有配置变更的最终落地


**6) package_sync**（i18n 名称）
- 创建入口：`UserPackageService.SyncUserPackage`

- Type：`i18n.T("agent.task_sync_package")`（Agent 同时兼容 `package_sync`）
- Data：`{"packages":[{"package_id", "version", "config": AgentPackageConfig}]}`

- Targets：`TargetsJSON` = 关联节点列表（线路组/备线路组）
- 执行：Agent 写入 `packages/<package_id>.json` + 更新内存

- 回写：`applied` JSON 列表（updated/skipped）


**7) issue_cert（Agent 路径）**

- 创建入口：`CertService.CreateIssueTasksAsync`（HTTP-01 类证书）

- Data：`IssueCertTaskPayload{ca,ca_dir_url,email,items:[{cert_id,domains}]}`

- Res：`{"target_node_id":<node_id>}`，用于只派发给指定节点
- 派发：API WS `task_dispatch` → Agent（仅命中目标节点）
- 执行：Agent 本地 ACME HTTP-01，写入 `/.well-known/acme-challenge/:token`（内存存储）
- 回写：Agent HTTP `POST /api/v1/agent/certs/issued` + WS `task_ack`，API 更新 `cert` 并 `BumpConfigVersion("cert")`



**8) agent_upgrade**

- 创建入口：`PackageController.SyncVersion`

- Data：`{version,file_name,sha256,download_url}`

- Targets：`TargetsJSON` 指定节点列表

- 执行：Agent 下载/解压，更新 `edge-node` 资源；Linux 下替换二进制并重启
- 回写
  - WS progress（`task_ack` status=progress）
  - 最终返`{version,restart}`

**C# 重构要求**
- 任务类型、Data/Res/Targets/Ret/Progress 的字段结构必须保持一致
- Agent 侧执行顺序与副作用必须与 Go 逻辑一致（尤其是清缓存/预热/证书签发/升级/配置落地）
- 任务完成/失败的回写与消息通知规则必须一致



---



#### B. API Worker 执行类（不下 Agent）

**现有行为（Go 代码）**
- 本节任务由 API 进程本地 Worker 执行（不下发 Agent）
- 任务状态由 Worker 直接写回数据库（`task.state/ret/progress`）


**9) site_create**

- 创建入口：`SiteController.AdminBatchCreate` `services.CreateSiteCreateTask`

- Data：`SiteCreatePayload{user_id,user_package_id,dns_provider_id,node_group_id,group_id,domain,backends}`

- 执行：`services.StartSiteCreateWorker` 本地执行

- 回写：创Site `BumpConfigVersion("site")` DNS 同步



**10) DNS_PLATFORM_CNAME_UPSERT / DNS_USER_CNAME_UPSERT**

- 预期入口：`services.CreateDNSTask`（目前未发现业务调用）
- 执行：`dns_task_worker`

- 注意：Worker 查询 `state=pending/retrying`，但 CreateDNSTask 设置 `state=waiting`，属**潜在历史逻辑/待确认**



**11) issue_cert（API 本地路径）**

- 创建入口：`CertService.CreateIssueTasksAsync`（DNS-01 类证书）
- 执行：API 本地 ACME DNS-01
- 状态：
  - `cert.state=dns_pending`，`cert.ret` 写入 DNS challenge 信息
  - TXT 生效后继续签发，成功转 `ready`，失败转 `fail`
- 回写：Task 状态、Cert 状态 + `BumpConfigVersion("cert")`



**12) backup（记录型任务）**

- 创建入口：`cleanup_worker.runBackup` `recordBackupTask`

- 执行：数据库备份任务本身由 worker 执行，Task 仅记录结果


**13) deploy_cert（当前无执行链路）**

- 创建入口：`CreateDeployTask`

- 说明：当前 Agent/Worker 未实现执行逻辑，仅为保留任务（C# 重构保持记录型任务）

**C# 重构要求**
- Worker 侧任务的状态流转必须与 Go 保持一致（`waiting/running/retrying/success/fail/pending` 等）
- 若保留 `dns_task_worker`，需统一 `state` 含义（避免 `waiting` 与 `pending` 混用）
- 记录型任务（如 `backup/deploy_cert`）仍需持久化 Task 与日志，保证可追踪



---



### 13.3 任务回写的核心路径（代码位置）


- **Agent WS 回写**：`Cnn.Api/Program.cs` + `AgentTaskAckService`

  - `task_ack` 更新 `task.state`, `task.ret`, `task.retry_at`

- **Agent HTTP 回写**：`POST /api/v1/agent/tasks/:id/finish`（C# 已实现，可作为断线兜底）

- **Worker 回写**：`CertIssueProcessor` / `UserPackageExpirationWorker` / `UserPackageTrafficWorker`



---



### 13.4 典型关联图（功能 Task 结果）


- 站点更新 `BumpConfigVersion` `config_sync` Agent reload

- 证书申请 `issue_cert` cert 状态变更 -> `config_sync` -> HTTPS 生效

- 套餐调整 `package_sync` Agent packages 更新 限额/功能立即生效

- 节点升级 `agent_upgrade` Agent 替换二进制并重启

- 批量清理 `clear_cache` Agent 清空 cache



---



### 13.5 开发者建议（任务追踪清单）


1) 找到“任务创建点”（controller/service），确认 `Type/Data/Res/TargetsJSON`
2) 找到“派发逻辑”（WS dispatch / Worker），确认执行路径
3) 确认执行端（Agent / API Worker）的实现与回写字段
4) 检Task 状态字符串是否被完整处理（避免 `pending` vs `waiting` 不一致）
5) 任何会影响 EdgeConfig 的任务，都要确保最终触发 `BumpConfigVersion` 或配置下发


---



（新增：任务与功能逐项追踪表）



---



## 14. C# 重写方案：YARP + WS（设计蓝图）



> 目的：在保持现有业务语义的前提下，以 C# 重写 API Agent，数据面使用 **YARP**（HTTP/HTTPS 反向代理），控制面使用 **WS**（Server <-> Node 长连接）。本节作为“架构实现一体化”规范，兼顾 CTO、架构师与实现人员视角，避免歧义


### 14.1 目标与非目标（必须明确）



**目标**

- **功能等价**：覆盖当前 Go 实现的所有功能点（配置同步、任务系统、日志指标、L2、DNS、证书、套餐、站点、ACL/CC/WAF、清理/预热等）
- **可迁移**：支持灰度升级与双栈共存；确保旧 Agent 不被立即淘汰
- **可观测**：统一日志、指标、链路追踪，提供可回溯的任务执行轨迹
- **可扩展**：配置模型版本化、插件式 WAF/缓存/鉴权


**非目标**

- 不在第一阶段替换业务数据库模型与表结构（除非必要）
- 不强行在第一阶段实现完全等价 HTTP/3/WAF 高阶能力（可分阶段）


### 14.2 关键技术定义


- **YARP（数据面 L7）**：承载 HTTP/HTTPS 反向代理、路由、Header 改写、负载均衡、健康检查
- **控制面协作**：WS 长连接（Server <-> Node），HTTP 拉取作为兜底




### 14.3 组件划分（系统上下文）


**A. Control Plane（API）**

- Config Service：生成 EdgeConfigV2

- Task Service：任务编排/分发/回写

- Logs Service：日志指标入库（ClickHouse）
- DNS Service：解析同步与修复任务

- Cert Service：证书签发/续期/下发



**B. Data Plane（Agent）**

- Proxy Host（YARP）：HTTP/HTTPS 代理

- Config Applier：配置落地 + 运行时热更新

- Task Executor：任务执行与回报

- Telemetry：日志指标/事件上报



### 14.4 配置模型（EdgeConfigV2）


> **严格 schema** 定义为准，禁止“隐式字段”。每个字段必须带：类型、必填、默认值、取值范围、影响范围


**顶层结构（示例）**

```json

{

  "version": 123,

  "node_id": "1001",

  "node_level": 1,

  "domains": [ ... ],

  "upstreams": [ ... ],

  "streams": [ ... ],

  "waf": { ... },

  "resources": { ... },

  "error_pages": { ... },

  "default_config": { ... },

  "cc_rules": { ... },

  "cc_matchers": { ... },

  "cc_filters": { ... },

  "nginx": { ... },

  "fallback_cert_data": "...",

  "fallback_key_data": "..."

}

```

**C# 重构落地（当前实现范围）**
- 已实现：`waf/resources/error_pages/default_config/cc_rules/cc_matchers/cc_filters/fallback_cert_data/fallback_key_data/domains/upstreams/streams/nginx`
- CC 解析规则：`cc_rule.data` 支持 `[]` 或 `{rules:[]}`，字段兼容 `matcher/matcher_id`、`filter1/filter_id`、`state` 默认 `true`

**配置落地策略（必须明确）**

- **原子性**：配置写入 + 内存更新 + YARP 热更必须是原子步骤
- **失败回滚**：若热更新失败，必须回滚到上一次配置版本
- **版本一致性**：Agent 只接受比当前更新的 `version`


### 14.5 控制面协议设计（WS 主通道）
**连接入口**
- `GET /ws/agent`（WebSocket）
**消息类型**
- `hello`：节点注册与鉴权
- `ack`：服务端确认 + 心跳间隔
- `ping/pong`：保持连接
- `edge_config`：下发 EdgeConfig
- `cache_config`：下发缓存配置
- `task_dispatch`：任务下发
- `task_ack`：任务状态回报
**兜底机制（可选）**
- `GET /api/v1/agent/config`：配置拉取（断线或冷启动）
- `GET /api/v1/agent/tasks`：任务拉取（断线恢复，C# 已实现）

### 14.6 Task 体系重构（强一致模型）



**统一 Task DTO**

```json

{

  "task_id": 123,

  "type": "refresh_url",

  "payload": "...",

  "targets": [1001,1002],

  "timeout": 600,

  "retry": {"max":3, "backoff":"exp"}

}

```



**强制规范**

- Task 状态只允许：`waiting | running | success | fail | retrying`

- Task 进度必须包含 `percent` + `message`

- Task 回写必须包含 `node_id` + `task_id` + `state`



### 14.7 数据面设计（YARP）
**核心能力映射**
- 路由与域名映射：`Domains` / `Upstreams` -> YARP `Routes` + `Clusters`
- 负载均衡策略 YARP `LoadBalancingPolicy`
- Header/ResponseHeader YARP `RequestHeaders` / `ResponseHeaders`
- HTTPS 终止 Kestrel + 证书热更新（证书缓存）
- 缓存：独立缓存中间件（可用 Redis + 本地缓存策略）
- WAF/CC/ACL：ASP.NET Middleware + 规则引擎（配置来源 `EdgeConfigStore.Current`）

#### 14.7.1 缓存设计（四类 Profile，默认不忽略参数）

**现有行为（Go 代码）**
- 站点缓存默认 **不忽略参数**（`ignore_args=false`）
- 兼容旧字段：`ignore_query/ignore_args/query_ignore_list/cache_key`
- 缓存规则最终落地为 Nginx `proxy_cache_key` 与 `proxy_cache_valid`

**C# 重构要求**
- 必须保持默认不忽略参数的语义
- `query_mode` 与旧字段映射规则保持一致
- 不新增数据库字段（仍用 `site_settings/site.proxy_cache/site_conf_cache`）

**设计目标**
- 默认 **不忽略 Query**（与现网行为一致）
- 兼容旧系统语义（站点级配置、节点级落地、可灰度迁移）
- 不新增数据库字段（沿用 `config(type=site_settings,name=settings)` / `site.proxy_cache` / `site_conf_cache.data` / `node.cache_dir`）
- 兼容旧字段：`ignore_query`、`ignore_args`、`query_ignore_list`、`cache_key`

**缓存类型（固定 4 类）**
- Home（首页）
- Site（全站）
- Static（静态资源）
- Video（视频）

**每类固定参数（最小闭环）**
- `ttl`：缓存有效期（秒）
- `force_cache`：是否强制缓存（默认 `false`）
- `query_mode`：参数参与策略（默认 `all`）
  - `all`：包含全部 Query
  - `ignore`：忽略全部 Query
  - `include`：仅包含 `query_keys`
  - `exclude`：排除 `query_keys`
- `query_keys`：参数名列表（支持前缀通配，如 `utm_*`）
- `cache_key`（可选高级）：直接指定 key 模板（优先级最高）

**兼容旧字段映射**
- `ignore_query=true` / `ignore_args=true` -> `query_mode=ignore`
- `query_ignore_list` 非空 -> `query_mode=exclude` + `query_keys=query_ignore_list`
- 同时存在 `query_mode` 与旧字段时：`query_mode` 优先
- `cache_key` 存在时：优先于 `query_mode`

**默认值（建议）**
- Home：`ttl=86400`（1 天），`query_mode=all`，`force_cache=false`
- Site：`ttl=259200`（3 天），`query_mode=all`，`force_cache=false`
- Static：`ttl=604800`（7 天），`query_mode=ignore`，`force_cache=false`
- Video：`ttl=2592000`（30 天），`query_mode=all`，`force_cache=false`
#### 14.7.2 规则优先级（唯一命中，避免歧义）
**现有行为（Go 代码）**
- 规则按 `priority` 降序；同优先级按解析顺序
- 规则冲突时仅命中一条（避免多条 location 同时生效）

**C# 重构要求**
- 优先级与“唯一命中”策略必须保持一致
- 规则排序与冲突处理保持可预测性
1) **显式规则**（Host + Path 前缀/正则）
2) **静态资源识别**（扩展名 / Content-Type）
3) **首页识别**（`/` / `/index*`）
4) **全站兜底**（Site）
- 若规则携带 `priority`：按 `priority` 降序优先，数值相同再按上述顺序
#### 14.7.3 Cache Key 生成（默认包含参数）
**现有行为（Go 代码）**
- `cache_key` 存在时直接使用
- 否则：`ignore_args=true` -> `host+uri`；默认 `host+uri+args`
- 当前 purge 仍按 `host+uri+args` 定位（与 `ignore_args/cache_key` 可能不一致）

**C# 重构要求**
- 新 key 生成需兼容旧行为，并补齐 `query_mode` 的可预期规则
- 必须与 purge/预热使用的 key 规则保持一致（避免清不掉缓存）
- 若 `cache_key` 有值：按模板生成（支持变量 `{host}` `{path}` `{query}` `{query_hash}`）
- 否则根据 `query_mode` 生成规范化 Query：
  - `all`：包含全部参数
  - `ignore`：忽略全部参数（空 Query）
  - `include`：仅保留 `query_keys`
  - `exclude`：排除 `query_keys`
- `query_keys` 匹配：大小写不敏感；支持前缀通配 `utm_*`
- `normalized_query` 规则：解码 -> 过滤 -> key/value 排序 -> 重新编码
- 落盘 key：
  - `www/<host>/<path>`
  - 若 Query 非空追加 `__q=<hash>`（建议稳定算法，如 xxHash64）

**示例**
- `www.a.com/a.png?x=1&y=2`
  - 默认：`www/a.com/a.png__q=<hash(x=1&y=2)>`
  - 忽略参数：`www/a.com/a.png`
- `www.a.com/v.mp4?token=abc&utm_source=ad`
  - `query_mode=exclude` + `query_keys=["utm_*"]`：`www/a.com/v.mp4__q=<hash(token=abc)>`
#### 14.7.4 强制缓存（force_cache）的边界
**现有行为（Go 代码）**
- `force_cache=true` 会忽略源站 `Cache-Control/Expires`
- 但 `no_cache/disable` 仍会绕过缓存

**C# 重构要求**
- 强制缓存边界保持一致（非缓存响应仍需跳过）
- `force_cache=true`：忽略源站 `Cache-Control: no-store/no-cache` 与 `Expires`
- 仍然不缓存：`4xx/5xx`、非 `GET/HEAD`、`Set-Cookie` 响应
- 视频允许缓存 `206 Partial Content`
#### 14.7.5 配置落地（不改表）
**现有行为（Go 代码）**
- 站点缓存设置存于 `config(type=site_settings,name=settings)` 与 `site.proxy_cache`
- 编译结果写入 `site_conf_cache.data`
- 节点落地依赖 `node.cache_dir/max_cache_size`

**C# 重构要求**
- 继续沿用现有表字段，不新增字段
- 编译与落地位置保持一致
**站点级配置入口**
- `site.proxy_cache` 存放 JSON（Profiles + Rules + query 策略）
**编译后分发**
- `site_conf_cache.data` 存放编译后的规则（Agent 直接使用）
**节点级落地**
- `node.cache_dir`：Agent 缓存根目录
- `node.max_cache_size`：Agent 缓存容量控制
#### 14.7.6 Agent 执行顺序（落盘路径固定）
**现有行为（Go 代码）**
- 规则命中后生成缓存 key 并映射到固定磁盘路径
- `ignore_args` 与 `cache_key` 影响落盘路径

**C# 重构要求**
- 执行顺序与落盘路径必须保持一致
1) 匹配规则 -> 得到 Profile
2) 计算 `ttl/force_cache/query_mode/query_keys/cache_key`
3) 生成缓存 key -> 映射落盘路径
   - `www/<host>/<path>`
   - `www/<host>/<path>__q=<hash>`（Query 非空）
4) 写入本地缓存（磁盘为主，可选 Redis）
#### 14.7.7 配置示例（站点级 JSON）
**现有行为（Go 代码）**
- 站点级缓存配置以 JSON 形式存储并下发

**C# 重构要求**
- 示例字段与含义保持一致，避免前端/配置误解
```json
{
  "profiles": {
    "Home":   { "ttl": 86400, "force_cache": false, "query_mode": "all", "query_keys": [] },
    "Site":   { "ttl": 259200, "force_cache": false, "query_mode": "all", "query_keys": [] },
    "Static": { "ttl": 604800, "force_cache": false, "query_mode": "ignore", "query_keys": [] },
    "Video":  { "ttl": 2592000, "force_cache": false, "query_mode": "exclude", "query_keys": ["utm_*", "session"] }
  },
  "rules": [
    { "host": "example.com", "path_prefix": "/static/", "profile": "Static" },
    { "host": "example.com", "path_regex": "^/video/", "profile": "Video" },
    { "host": "example.com", "path_prefix": "/", "profile": "Site" }
  ]
}
```

#### 14.7.8 缓存模块 C# 重构任务拆分（与缓存设置同级）
**现有行为（Go 代码）**
- 任务拆分基于现有缓存规则/任务/下发链路设计（保证迁移兼容）

**C# 重构要求**
- 按清单逐项落地，不新增字段，保持兼容性优先
**目标**
- 与现有功能行为一致（优先兼容），并明确“参数策略/强制缓存/落盘路径”的唯一口径
- 不新增数据库字段（沿用 `config(type=site_settings,name=settings)` / `site.proxy_cache` / `site_conf_cache` / `config(type=site_default_config,name=proxy_cache)` / `node.cache_dir` / `node.max_cache_size`）
- 所有返回使用统一结构 `{code,message,data}`，`message` i18n（Accept-Language/lang）

**范围**
- 站点级缓存设置（单站编辑 + 批量编辑）
- 缓存规则解析/兼容/编译（规则 -> 缓存 profile -> 运行时策略）
- 网关/节点缓存引擎（YARP + OutputCaching + 磁盘落盘路径）
- purge/预热/清缓存任务与 Task 体系的关联与回写

**任务清单（按依赖顺序）**
1) **数据与实体（SqlSugar）**
   - 表与字段映射：
     - `site`：`settings`(json)、`proxy_cache`(legacy)、`version`
     - `config`：`type=site_default_config`，`name=proxy_cache`
     - `site_conf_cache`：`site_id/version/data`
     - `node`：`cache_dir/max_cache_size`
     - `task`：`type=refresh_url/refresh_dir/pre_cache_url/clear_cache`（详 Task 章节）
   - 产出：`SiteEntity/ConfigItemEntity/SiteConfCacheEntity/NodeEntity/TaskEntity`
   - 统一 JSON 读写（`settings` 反序列化/合并）

2) **DTO/契约（输入输出结构）**
   - `CacheRuleDto`（前端输入）
     - `type/value/ttl/ignore_query/force_cache/no_cache/enable/priority/cache_key/enable_range/ignore_vary/skip_conditions`
   - `CacheConfigDto`：`rules: CacheRuleDto[]`
   - `CacheProfile`（重构内核）：`ttl/force_cache/query_mode/query_keys/cache_key`
   - `CacheRuleNormalized`（编译后）：`rule/uri/prefix/ext/ttl/force_cache/no_cache/enable/priority/ignore_args/cache_key`
   - 兼容字段映射：
     - `ignore_query/ignore_args` -> `query_mode=ignore`
     - `cache_key` 优先于 `query_mode`
     - `type` -> `rule/uri/prefix/ext`

3) **规则解析与校验**
   - 规则合法性：
     - `ttl` 必须为正整数（允许 0 表示不设 TTL）
     - `type=suffix/dir/path` 必须提供 `value`
     - `rule/uri/prefix/ext` 直写时需校验格式（`uri/prefix` 必须 `/` 开头）
   - 兼容解析：
     - `value` 支持空格/`|` 分隔
     - `suffix` 自动去掉 `*`/`.` 前缀
   - 标准化输出：
     - `type=all` -> `prefix=/`
     - `type=index` -> `uri=/`
     - `rule` 若为 `/path` 视为 `^~ /path`

4) **站点设置读写**
   - 读取：`config(type=site_settings,name=settings).cache.rules` -> 反序列化 `CacheRuleDto[]`
   - 写入：保存到 `config(type=site_settings,name=settings).cache`（保留原字段与顺序，避免回写抖动）
   - 默认值：站点未配置时读取 `config(type=site_default_config,name=proxy_cache)` 作为默认规则
   - 批量修改：仅覆盖被勾选的字段（与 `BatchSettingsDialog.vue` 行为一致）

5) **规则编译与下发**
   - 输入：`CacheRuleDto[] + CacheProfile + Site/Package/Global defaults`
   - 输出：
     - `site_conf_cache.data`（编译后的 EdgeConfig/CacheConfig）
     - 生成 `config_sync` Task，触发节点更新
   - 排序策略：`priority` 降序；冲突规则跳过；默认 `/` 兜底

6) **缓存 Key 生成器（网关/节点共用）**
   - `cache_key` 有值：直接按模板渲染（支持 `{host}{path}{query}{query_hash}`）
   - 否则按 `query_mode` 归一 Query：
     - `all/ignore/include/exclude` + `query_keys`（大小写不敏感，支持 `utm_*`）
   - 落盘路径：
     - `www/<host>/<path>`
     - Query 非空：追加 `__q=<hash>`（稳定 hash）

7) **YARP + OutputCaching 落地**
   - 统一缓存策略：
     - 默认仅缓存 `GET/HEAD`，响应码 `200/302`
     - `force_cache=true`：忽略 `Cache-Control/Expires`
     - `enable_range=true`：允许 `206`（视频场景）
   - 自定义 `IOutputCacheKeyProvider`：
     - 承载 `query_mode/query_keys/cache_key/ignore_vary`
   - 自定义 `IOutputCacheStore`：
     - 落盘 `www/<host>/<path>`（满足路径规则）
     - 支持 TTL、并发锁、过期清理

8) **purge/预热/清缓存任务**
   - `refresh_url`：按“同一 cache key 生成规则”定位并删除文件  
     - `ignore_query=ignore` 时：需同时删除“无参数版本”与 Query 版本
   - `refresh_dir`：支持按目录前缀清理（不得退化为全站清空）
   - `clear_cache`：全量清空（保持与现有行为兼容）
   - 任务回写：必须更新 `task.state/progress/ret`

9) **日志与观测**
   - 保留 `upstream_cache_status` 语义（HIT/MISS/BYPASS）
   - 统计口径 ClickHouse 字段保持一致

10) **前端/Blazor 对齐**
   - 规则结构必须兼容 `CacheConfig.vue` / `BatchSettingsDialog.vue`
   - 快捷模板：`index/all/static/video/wordpress` 不改行为
   - 新增字段展示顺序：`TTL` -> `忽略参数` -> `强制缓存` -> `高级 cache_key`

**验收标准**
- 同一站点同一规则，Go 版本与 C# 版本的缓存 key/落盘路径一致
- purge/clear_cache 行为与现有任务兼容（且不误删）
- 前端可无改动直接读写缓存规则，批量设置按勾选字段生效
#### 14.7.9 缓存模块 C# 接口与伪代码级流程（无歧义版）
**现有行为（Go 代码）**
- 现网缓存逻辑由 Nginx/OpenResty + Lua 配置驱动，本节为等价迁移的流程化描述

**C# 重构要求**
- 伪代码与接口必须可直接落地实现（保持与 Go 行为一致）
**说明**
- 本节只覆盖“缓存模块”可落地的最小闭环接口与流程；API 路由保持与第 3 章一致
- 所有接口统一返回 `{code,message,data}`，`message` i18n 生成
##### 14.7.9.1 领域接口（建议拆分）
- `ICacheSettingsService`
  - `GetSiteCacheSettings(siteId, userId, role)`：读取站点缓存规则（含默认补齐）
  - `SaveSiteCacheSettings(siteId, userId, role, CacheConfigDto dto)`：保存并触发编译/下发
  - `BatchUpdateSiteCacheSettings(siteIds, CacheBatchDto dto, selectedFlags)`：批量覆盖（只改勾选字段）
- `ICacheRuleParser`
  - `Parse(CacheRuleDto dto) -> IEnumerable<CacheRuleNormalized>`
  - `Validate(CacheRuleDto dto) -> ValidationResult`
- `ICacheCompiler`
  - `Compile(siteId, CacheRuleDto[] rules, defaults) -> CacheCompiledConfig`
  - `Persist(siteId, compiled)`：写`site_conf_cache`
- `ICacheKeyService`
  - `BuildKey(CacheProfile profile, HttpRequest req) -> CacheKeyResult`
- `ICacheStore`（OutputCacheStore + Disk）
  - `Get(key)` / `Set(key, payload, ttl)` / `Remove(key)` / `RemoveByPrefix(prefix)` / `ClearAll()`
- `ICacheTaskService`
  - `CreatePurgeUrlTask(siteId, urls)` / `CreatePurgeDirTask(siteId, dirs)`
  - `CreateClearCacheTask(siteId)` / `CreatePreheatTask(siteId, urls)`

##### 14.7.9.2 保存与编译流程（伪代码）
```pseudo
SaveSiteCacheSettings(siteId, dto):
  site = SiteRepo.Get(siteId)
  require site exists & permission ok

  rules = dto.rules ?? []
  validate each rule

  // 持久化到 config(type=site_settings,name=settings).cache
  settings.cache.rules = rules
  SiteRepo.Update(site)

  // 编译后缓存配置
  compiled = CacheCompiler.Compile(siteId, rules, defaultsFromConfig)
  CacheCompiler.Persist(siteId, compiled)  // site_conf_cache

  // 配置同步任务
  TaskService.CreateConfigSyncTask(siteId)
  return OK
```

##### 14.7.9.3 规则解析与归一化（伪代码）
```pseudo
Parse(rule):
  if rule.rule/uri/prefix/ext provided:
     yield normalizeDirect(rule)
  else:
     values = split(rule.value, " " or "|")
     for each v in values:
        map type -> ext/prefix/uri
        yield normalized

normalizeDirect(rule):
  if rule.rule like "/path" -> "^~ /path"
  if rule.uri not startsWith "/" -> invalid
  if rule.prefix not startsWith "/" -> invalid
  if rule.ext startsWith "." -> trim "."
  return CacheRuleNormalized(...)
```

##### 14.7.9.4 请求命中与缓存决策（伪代码）
```pseudo
OnRequest(req):
  rule = MatchRule(req.host, req.path, req.ext)  // priority desc, first hit
  if rule == null: rule = DefaultSiteRule("/")

  if rule.enable == false or rule.no_cache == true:
     bypass cache

  profile = ResolveProfile(rule, defaults)
  if req.method not in GET/HEAD: bypass cache

  key = CacheKeyService.BuildKey(profile, req)
  CacheMiddleware.TryServe(key, profile)
```

##### 14.7.9.5 CacheKey 生成（伪代码）
```pseudo
BuildKey(profile, req):
  if profile.cache_key not empty:
     return render(profile.cache_key, {host,path,query,query_hash})

  query = NormalizeQuery(req.query, profile.query_mode, profile.query_keys)
  base = "www/" + req.host + req.path
  if query empty:
     return base
  else:
     return base + "__q=" + Hash(query)
```

##### 14.7.9.6 OutputCaching + Disk Store 落地（伪代码）
```pseudo
OutputCachePolicy:
  if profile.force_cache: ignore response Cache-Control/Expires
  ttl = profile.ttl
  if ttl <= 0:
     use upstream cache headers (if any)
  else:
     set absolute expiration ttl
  if response has Set-Cookie or status >= 400: bypass cache

DiskCacheStore.Set(key, payload, ttl):
  path = Root + "/" + key
  ensure dir exists
  write payload + meta(expireAt, headers) atomically

DiskCacheStore.Get(key):
  load meta; if expired -> remove and miss
  return payload
```

##### 14.7.9.7 purge/清理/预热（伪代码）
```pseudo
PurgeURL(url):
  req = parse(url)
  key = CacheKeyService.BuildKey(profile, req)
  store.Remove(key)
  if profile.query_mode == ignore:
     remove base path without query hash

PurgeDir(prefix):
  store.RemoveByPrefix("www/" + host + prefix)

ClearCache():
  store.ClearAll()
```

##### 14.7.9.8 必须保持的兼容点
- `ignore_query/ignore_args` `cache_key` 的优先级规则不变
- purge cache key 必须使用同一生成逻辑（避免“清不掉”）
- 默认快捷模板（index/all/static/video/wordpress）行为一致
### 14.8 控制面设计（WS Server + Node）

**核心约束**

- 必须完成鉴权：`node_id` + `token`
- 连接存活：心跳间隔可配置（建议 30s）
- 任务派发：Server 推`task`，Node 回报 `report`
- 状态回写：Task 状态变更必须持久化并可重放


**最小协议**

- `hello` / `ack` / `ping` / `pong`

- `task` / `report`



### 14.9 迁移与兼容策略（强制执行）


1) **双栈运行**：新 API 与旧 API 并行，Agent 可选择接入 

2) **配置双写**：新 API 生成 EdgeConfigV2，同时可转换为旧 EdgeConfig（过渡期） 

3) **渐进替换**：先迁移日志/任务，再迁移配置，再迁移数据面 

4) **回滚机制**：支持“一键切回旧 Agent”


### 14.10 文档易读且无歧义的写法（强规范）



**必须遵循**

- 使用 **RFC 2119** 风格词汇：`MUST/SHOULD/MAY`（文档内统一中文：必应该/可以）
- 所有接口字段必须给出：

  - 类型、必填、默认值、取值范围、示例
- 所有状态字段必须给出：

- 状态机图（文本/图示 + 迁移条件）
- 所有重要流程必须给出：

  - **顺序性**（可用文本流程图或步骤清单）



**文档结构模板**（建议统一）
1) 背景与目标 

2) 术语 

3) 架构图（逻辑/部署/时序） 

4) 数据模型（JSON Schema/Proto） 

5) API 定义（路由/字段 + 返回码）  

6) 任务/事件流（状态机） 

7) 失败与回滚策略 

8) 观测与运维 

9) 迁移与兼容 

10) Open Issues



### 14.11 CTO / 架构/ 实现人员职责划分



**CTO 视角**

- 明确阶段目标与业务优先级（必须落地）

- 决策迁移与回滚策略（必须落地）


**架构师视角**

- 给出组件边界与协议规范（必须落地）
- 给出扩展点设计（插件式）



**实现人员视角**

- 代码必须严格符合规范（字段状态回写一致）

- 任何新增字段必须更新 Schema 与示例


---



（新增：C# + YARP + WS 重写设计与文档规范）



---



## 15. 全功能流程文档（C# 重写版，按业务链路顺序）



> 本节是“实现级流程说明”。每个流程包含：入口、前置校验、数据更新、任务/配置触发、Agent 影响、失败回写。写法以“开发者无需猜测”为目标


缓存规则校验与默认值补齐（见 2.4 缓存规则与 14.7 缓存设计）
---



### 15.6 证书申请/签发



**入口**

- `POST /api/v1/*/certs` / `POST /certs/wildcard` / `POST /certs/reissue`



**流程（两条路径）**

A) **Agent 签发**（HTTP-01 或指定节点）

1) API 创建 `issue_cert` Task，指`TargetNodeID`
2) WS 派发到节点
3) Agent 执行 ACME 回传 `cert_issued`
4) API 更新 cert + `BumpConfigVersion("cert")`


B) **API 本地签发**（DNS challenge）
1) API 本地 ACME 签发

2) 更新 cert 状态
3) `BumpConfigVersion("cert")`



**Agent 影响**

- 更新证书配置后触发 `config_sync`，Agent reload HTTPS 证书


---



### 15.7 套餐同步（package_sync）


**入口**

- `UserPackageService.SyncUserPackage`（套餐变更/续费/过期）


**流程**

1) 读取 `user_package`，并对 `version` 做自增（`NULL` 视为 0，写回为 1）
2) 读取系统配置 `package_expire_close_site`（默认 true）
   - 若 `trigger=expire` 或 `EndAt < now` 且开关开启，则 `status="expired"`
   - 否则 `status="active"`
3) 构建 `AgentPackageConfig`（写入 Task.data）
   - 顶层：`package_id/uid/version/status/region_id/node_group_id/backup_node_group/enable_backup`
   - `cname`：`domain/hostname/hostname2/mode/record_id`
   - `limits`：`traffic/bandwidth/connection/domain`
   - `features`：`http_port/stream_port/websocket/custom_cc_rule/l2_origin`
   - `time`：`start_at/end_at`（`YYYY-MM-DD HH:mm:ss`）
4) 计算目标节点
   - `group_ids = node_group_id + (enable_backup_group ? backup_node_group : 0)`
   - 使用 `line.node_group_id` 找到关联 `node_id/node_ip_id`，再过滤 `node.enable=true`
5) 创建 Task
   - `type = i18n.T("agent.task_sync_package")`（中文为“套餐同步”，Agent 兼容 `package_sync`）
   - `name = i18n.T("task.sync_package_prefix") + package_id`
   - `data = { packages: [ {package_id, version, config} ] }`
   - `targets_json = TaskTargets`（nodes->state=waiting）
   - `state = waiting`；若 `targets.total=0` 则 `state=done` 且 `end_at=now`
6) WS/轮询派发后，Agent 写入 `packages/<id>.json`


**回写**

- Agent 返回 `applied` 状态（updated/skipped）


#### 15.7.1 套餐到期检测

**现有行为（Go 代码）**
- **周期**：每小时
- **前置**：`package_expire_close_site=true`
- **条件**：`EndAt < now` 且 `is_expired=false/NULL`
- **处理**：
  - 更新 `is_expired=true`
  - 发送 `message`（type=package-expire）
  - 触发 `SyncUserPackage(id, "expire")`

**C# 重构要求**
- 保持定时频率、触发条件与消息类型一致
- `SyncUserPackage` 必须在到期后触发并同步到 Agent


#### 15.7.2 流量超限控制

**现有行为（Go 代码）**
- **周期**：每 10 分钟
- **前置**：ClickHouse HTTP DSN 存在 + `traffic_excceed_close_site=true`
- **参数**：`tcp_traffic_factor`（倍率，默认 1）
- **统计**：
  - 取 `user_package.traffic>0` 且未过期的套餐
  - 收集 `site.domain`（合并去重）
  - ClickHouse 查询 `node_access_logs`（`sum(bytes)`）
  - `used_bytes *= tcp_traffic_factor`
- **超限处理**：
  - `site.state` 从 `""/NULL/running` -> `traffic_limit`
  - `BumpConfigVersion("site", ids)`
  - 发送 `message`（type=traffic-exceed）
- **恢复处理**：
  - `site.state` 从 `traffic_limit` -> `running`
  - `BumpConfigVersion("site", ids)`

**C# 重构要求**
- ClickHouse 查询口径与倍数计算保持一致
- 超限/恢复的站点状态与 `BumpConfigVersion` 行为保持一致


---



### 15.8 节点启停



**入口**

- `PUT /api/v1/admin/nodes/:id/status`



**流程**

1) DB 更新 `node.enable` `node.config_task`（sync_enable/sync_disable）
2) Agent heartbeat_ack 触发 `applyNodeSync`
3) Agent 执行 start/stop Nginx
4) 回传 `node_sync`，API 清空 `config_task`


---



### 15.9 节点升级（agent_upgrade）


**入口**

- Admin: `POST /api/v1/admin/packages/upgrade`



**流程**

1) 创建 Task `agent_upgrade`，Targets=节点
2) Agent 下载/解压，替换 runtime，重启
3) Task 回写 progress 与最终结果


---



### 15.10 清理/预热任务



**入口**

- `POST /api/v1/*/tasks`（refresh_url/refresh_dir/preheat）


**流程**

1) 校验 URL & 域名归属
2) 扣减当天配额
3) 创建 Task，WS 派发
4) Agent 执行并回写


---



### 15.11 DNS 同步



**入口**

- 站点创建/更新 `SyncUserDNSRecords`

- 线路/节点变更 `SyncLineRecords` + `SyncPackageCnameForLineChange`



**流程**

- 同步 CNAME 记录，更`platform_dns_record_id` / `user_dns_record_id`


---



### 15.12 L2 健康与回源


**入口**

- Agent 周期性：`requestL2Nodes` + `checkL2Nodes`



**流程**

1) API 返回 L2 列表（同线路组、level=2）
2) Agent 探测存活，写`l2_status.json`
3) stream.conf 根据健康状态切L2/源站


---



（新增：C# 重写下的完整流程文档）


---



## 16. EdgeConfigV2 JSON Schema + 示例（C# 重写）


> 本节为“唯一真实来源”。任何字段变更必须先更新 Schema 与示例


### 16.1 JSON Schema（核心骨架）



```json

{

  "$schema": "https://json-schema.org/draft/2020-12/schema",

  "title": "EdgeConfigV2",

  "type": "object",

  "required": ["version", "node_id", "node_level", "domains", "upstreams"],

  "properties": {

    "version": {"type": "integer", "minimum": 1},

    "node_id": {"type": "string", "minLength": 1},

    "node_level": {"type": "integer", "enum": [1, 2]},

    "domains": {"type": "array", "items": {"$ref": "#/definitions/EdgeDomain"}},

    "upstreams": {"type": "array", "items": {"$ref": "#/definitions/EdgeUpstream"}},

    "streams": {"type": "array", "items": {"$ref": "#/definitions/EdgeStream"}},

    "waf": {"$ref": "#/definitions/WAFConfig"},

    "resources": {"$ref": "#/definitions/GlobalResourceConfig"},

    "error_pages": {"type": "object", "additionalProperties": {"type": "string"}},

    "default_config": {"$ref": "#/definitions/DefaultSiteConfig"},

    "cc_rules": {"type": "object", "additionalProperties": {"type": "array", "items": {"$ref": "#/definitions/EdgeCCRuleItem"}}},

    "cc_matchers": {"type": "object", "additionalProperties": {"$ref": "#/definitions/EdgeCCMatcher"}},

    "cc_filters": {"type": "object", "additionalProperties": {"$ref": "#/definitions/EdgeCCFilter"}},

    "nginx": {"$ref": "#/definitions/EdgeNginxConfig"},

    "fallback_cert_data": {"type": "string"},

    "fallback_key_data": {"type": "string"}

  },

  "definitions": {

    "EdgeDomain": {

      "type": "object",

      "required": ["name", "upstream_key"],

      "properties": {

        "name": {"type": "string"},

        "upstream_key": {"type": "string"},

        "l2_upstream_key": {"type": "string"},

        "use_l2": {"type": "boolean"},

        "l2_http_port": {"type": "string"},

        "l2_https_port": {"type": "string"},

        "load_balance_policy": {"type": "string", "enum": ["round_robin", "random", "ip_hash"]},

        "headers": {"type": "object", "additionalProperties": {"type": "string"}},

        "response_headers": {"type": "object", "additionalProperties": {"type": "string"}},

        "hotlink": {"$ref": "#/definitions/EdgeHotlinkConfig"},

        "cors": {"$ref": "#/definitions/EdgeCorsConfig"},

        "cookie": {"$ref": "#/definitions/EdgeCookieConfig"},

        "block_transparent_proxy": {"type": "boolean"},

        "crawler_action": {"type": "string"},

        "guard_pass_ttl": {"type": "integer"},

        "guard_block_ttl": {"type": "integer"},

        "url_redirects": {"type": "array", "items": {"type": "object"}},

        "origin_conditions": {"type": "array", "items": {"type": "object"}},

        "status": {"type": "string", "enum": ["active", "locked", "expired", "traffic_limit", "conn_limit"]},

        "conn_limit": {"type": "integer"},

        "ssl_cert_data": {"type": "string"},

        "ssl_key_data": {"type": "string"},

        "ssl_cert_path": {"type": "string"},

        "ssl_key_path": {"type": "string"},

        "acl_default_action": {"type": "string", "enum": ["allow", "deny"]},

        "acl_rules": {"type": "array", "items": {"$ref": "#/definitions/EdgeACLRule"}},

        "black_ips": {"type": "array", "items": {"type": "string"}},

        "white_ips": {"type": "array", "items": {"type": "string"}},

        "region_block": {"type": "array", "items": {"type": "string"}},

        "cc_rule_id": {"type": "integer"},

        "origin_protocol": {"type": "string", "enum": ["http", "https", "follow", "follow_port"]},

        "origin_http_port": {"type": "string"},

        "origin_https_port": {"type": "string"},

        "cache": {"$ref": "#/definitions/EdgeCacheConfig"},

        "http_listen": {"type": "array", "items": {"type": "string"}},

        "https_listen": {"type": "array", "items": {"type": "string"}},

        "https_force": {"type": "boolean"},

        "https_redirect_port": {"type": "string"},

        "https_hsts": {"type": "boolean"},

        "https_http2": {"type": "boolean"},

        "https_http3": {"type": "boolean"},

        "https_ocsp": {"type": "boolean"},

        "https_ssl_protocols": {"type": "string"},

        "https_ssl_ciphers": {"type": "string"},

        "https_ssl_prefer_server_ciphers": {"type": "string"},

        "proxy_connect_timeout": {"type": "string"},

        "proxy_read_timeout": {"type": "string"},

        "proxy_send_timeout": {"type": "string"},

        "proxy_http_version": {"type": "string", "enum": ["1.0", "1.1"]},

        "proxy_ssl_protocols": {"type": "string"},

        "enable_gzip": {"type": "boolean"},

        "gzip_types": {"type": "string"},

        "enable_websocket": {"type": "boolean"},

        "enable_range": {"type": "boolean"},

        "body_limit": {"type": "integer"},

        "limit_rate": {"type": "integer"},

        "upstream_keepalive": {"type": "boolean"},

        "upstream_keepalive_conn": {"type": "integer"},

        "upstream_keepalive_timeout": {"type": "integer"}

      }

    },

    "EdgeUpstream": {

      "type": "object",

      "required": ["id", "targets"],

      "properties": {

        "id": {"type": "string"},

        "targets": {"type": "array", "items": {"$ref": "#/definitions/EdgeUpstreamTarget"}}

      }

    },

    "EdgeUpstreamTarget": {

      "type": "object",

      "required": ["addr"],

      "properties": {

        "addr": {"type": "string"},

        "weight": {"type": "integer", "minimum": 1},

        "node_id": {"type": "integer"}

      }

    },

    "EdgeStream": {

      "type": "object",

      "required": ["id", "listen_ports", "targets"],

      "properties": {

        "id": {"type": "integer"},

        "listen_ports": {"type": "array", "items": {"type": "string"}},

        "targets": {"type": "array", "items": {"$ref": "#/definitions/EdgeStreamTarget"}},

        "use_listen_port": {"type": "boolean"},

        "balance_way": {"type": "string", "enum": ["ip_hash", "least_conn", "round_robin"]},

        "proxy_protocol": {"type": "boolean"},

        "proxy_connect_timeout": {"type": "string"},

        "proxy_timeout": {"type": "string"},

        "conn_limit": {"type": "integer"}

      }

    },

    "EdgeStreamTarget": {

      "type": "object",

      "required": ["addr"],

      "properties": {

        "addr": {"type": "string"},

        "weight": {"type": "integer"},

        "enable": {"type": "boolean"},

        "node_id": {"type": "integer"},

        "backup": {"type": "boolean"}

      }

    },

    "EdgeCacheConfig": {

      "type": "object",

      "required": ["enable"],

      "properties": {

        "enable": {"type": "boolean"},

        "default_ttl": {"type": "integer"},

        "rules": {"type": "array", "items": {"$ref": "#/definitions/EdgeCacheRule"}}

      }

    },

    "EdgeCacheRule": {

      "type": "object",

      "properties": {

        "rule": {"type": "string"},

        "ext": {"type": "string"},

        "uri": {"type": "string"},

        "prefix": {"type": "string"},

        "ttl": {"type": "integer"},

        "enable": {"type": "boolean"},

        "no_cache": {"type": "boolean"},

        "force_cache": {"type": "boolean"},

        "priority": {"type": "integer"},

        "ignore_args": {"type": "boolean"},

        "cache_key": {"type": "string"}

      }

    },

    "EdgeHotlinkConfig": {

      "type": "object",

      "properties": {

        "enable": {"type": "boolean"},

        "scope": {"type": "string"},

        "value": {"type": "string"},

        "allow_empty": {"type": "boolean"},

        "domains": {"type": "array", "items": {"type": "string"}}

      }

    },

    "EdgeCorsConfig": {

      "type": "object",

      "properties": {

        "enable": {"type": "boolean"},

        "allow_origin": {"type": "string"},

        "allow_methods": {"type": "string"},

        "allow_headers": {"type": "string"},

        "expose_headers": {"type": "string"},

        "allow_credentials": {"type": "boolean"},

        "max_age": {"type": "string"}

      }

    },

    "EdgeCookieConfig": {

      "type": "object",

      "properties": {

        "enable": {"type": "boolean"},

        "domain": {"type": "string"}

      }

    },

    "EdgeACLRule": {

      "type": "object",

      "properties": {

        "ip": {"type": "string"},

        "action": {"type": "string", "enum": ["allow", "deny"]}

      }

    },

    "EdgeCCRuleItem": {

      "type": "object",

      "properties": {

        "matcher_id": {"type": "integer"},

        "filter_id": {"type": "integer"},

        "action": {"type": "string"},

        "enabled": {"type": "boolean"}

      }

    },

    "EdgeCCMatcher": {

      "type": "object",

      "properties": {

        "id": {"type": "integer"},

        "data": {"type": "string"}

      }

    },

    "EdgeCCFilter": {

      "type": "object",

      "properties": {

        "id": {"type": "integer"},

        "type": {"type": "string"},

        "within_second": {"type": "integer"},

        "max_req": {"type": "integer"},

        "max_req_per_uri": {"type": "integer"},

        "extra": {"type": "string"}

      }

    },

    "WAFConfig": {

      "type": "object",

      "properties": {

        "enable": {"type": "boolean"},

        "block_unbound_domain": {"type": "boolean"}

      }

    },

    "DefaultSiteConfig": {

      "type": "object",

      "properties": {

        "website": {"$ref": "#/definitions/SiteTemplate"},

        "api": {"$ref": "#/definitions/SiteTemplate"},

        "download": {"$ref": "#/definitions/SiteTemplate"}

      }

    },

    "SiteTemplate": {

      "type": "object",

      "properties": {

        "cache_enable": {"type": "boolean"},

        "cache_ttl": {"type": "integer"},

        "gzip": {"type": "boolean"},

        "waf_enable": {"type": "boolean"},

        "ssl_ciphers": {"type": "string"}

      }

    },

    "GlobalResourceConfig": {

      "type": "object",

      "properties": {

        "website": {"$ref": "#/definitions/WebsiteResourceConfig"},

        "forward": {"$ref": "#/definitions/ForwardResourceConfig"},

        "public": {"$ref": "#/definitions/PublicResourceConfig"}

      }

    },

    "WebsiteResourceConfig": {

      "type": "object",

      "properties": {

        "log_storage_dir": {"type": "string"},

        "log_storage_hours": {"type": "integer"},

        "default_listen_80": {"type": "boolean"}

      }

    },

    "ForwardResourceConfig": {

      "type": "object",

      "properties": {

        "disabled_ports": {"type": "string"}

      }

    },

    "PublicResourceConfig": {

      "type": "object",

      "properties": {

        "disabled_custom_ports": {"type": "string"}

      }

    },

    "EdgeNginxConfig": {

      "type": "object",

      "properties": {

        "logs_dir": {"type": "string"},

        "worker_processes": {"type": "string"},

        "worker_connections": {"type": "integer"},

        "worker_rlimit_nofile": {"type": "integer"},

        "worker_shutdown_timeout": {"type": "string"},

        "resolver": {"type": "string"},

        "resolver_timeout": {"type": "string"},

        "http": {"type": "object"},

        "stream": {"type": "object"}

      }

    }

  }

}

```



### 16.2 示例（最小可运行配置）


```json

{

  "version": 1001,

  "node_id": "1",

  "node_level": 1,

  "domains": [

    {

      "name": "www.example.com",

      "upstream_key": "upstream_1",

      "http_listen": ["80"],

      "https_listen": ["443"],

      "https_force": true,

      "cache": {"enable": true, "default_ttl": 120}

    }

  ],

  "upstreams": [

    {

      "id": "upstream_1",

      "targets": [

        {"addr": "10.0.0.1:80", "weight": 1}

      ]

    }

  ],

  "streams": [],

  "waf": {"enable": false, "block_unbound_domain": false},

  "resources": {

    "website": {"log_storage_dir": "logs", "log_storage_hours": 12, "default_listen_80": true},

    "forward": {"disabled_ports": ""},

    "public": {"disabled_custom_ports": ""}

  },

  "error_pages": {},

  "default_config": {},

  "cc_rules": {},

  "cc_matchers": {},

  "cc_filters": {},

  "nginx": {"logs_dir": "logs"}

}

```



---



## 17. Task 状态机与约束规范（C# 重写）


> 本节为“强一致约束”。C# 实现必须以此为准，不允许出现新状态或未定义状态


### 17.1 状态机定义



**状态集合（唯一允许）**

- `waiting` / `running` / `success` / `fail` / `retrying`



**状态转移（文本图）**

```

waiting -> running -> success

waiting -> running -> fail

running -> retrying -> waiting（延迟重试）

retrying -> waiting

retrying -> fail (超过 maxRetries)

```



**约束规则**

- `waiting` 必须拥有：`create_at`

- `running` 必须拥有：`start_at`

- `success/fail` 必须拥有：`end_at`

- `retrying` 必须设置：`retry_at`

- 每次失败必须递增 `err_times`



### 17.2 Progress 回写规范



**统一 Progress 格式**

```json

{ "progress": 0-100, "message": "..." }

```



**回写约束**

- `progress` 只能递增（不得回退）
- 最终 `success` progress 必须 100



### 17.3 多节点任务（TargetsJSON）规范


**TaskTargets**

- `nodes`: `{ node_id: {state,tries,retry_at,ret,progress,message,last_at} }`

- `total/success/fail/pending` 必须在每次更新后重算



**节点回写规则**

- 任一节点进入 `failed_final` 如果所有节点已完成，则 Task 进入 `fail`

- 任一节点 `success` 但仍有其他节点 `waiting/running` 时 Task 保持 `running`



### 17.4 错误码与回写字段规范



- 所有 Task 执行失败必须回写
  - `task.ret`：错误字符串（不可为空）

  - `task.state = fail`

- 若失败可重试
  - `task.state = retrying`

- `retry_at` 设定为下一次执行时间


---



（新增：EdgeConfigV2 Schema + Task 状态机规范）


---



## 18. API C# 模块映射表（无歧义对照）



> 本表是“功能迁移总索引”。任何旧接口都必须能在 C# 新架构中找到唯一归属


### 18.1 控制面（API）映射


| 旧模块/Controller | 旧接口范围 | C# 新模块 | 关键 Service | 任务/配置影响 |

|---|---|---|---|---|

| AuthController | /login /register /login/captcha | AuthModule | AuthService, CaptchaService | |

| SystemInfoController | /system_info | SystemModule | SystemInfoService | |

| GlobalConfigController | /global_config | ConfigModule | GlobalConfigService | BumpConfigVersion(global) config_sync |

| ConfigItemController | /config_items | ConfigModule | ConfigItemService | 可能触发 config_sync |

| NodeController | /nodes* | NodeModule | NodeService, NodeInstallService | 节点启停 config_task / WS sync |

| NodeGroupController | /node-groups* | LineModule | LineGroupService | BumpConfigVersion(line/group) |

| RegionController | /regions* | RegionModule | RegionService | 影响 L2 检测参数 |

| SiteController | /sites* /domain_usage | SiteModule | SiteService, CnameService | BumpConfigVersion(site) + DNS 同步 |

| SiteGroupController | /site_groups* | SiteModule | SiteGroupService | 影响站点归属与配置 |

| SiteDefaultController | /site_defaults* | SiteModule | SiteDefaultService | 影响默认配置 |

| CertController | /certs* | CertModule | CertService, AcmeService | issue_cert / config_sync |

| ACLController | /rules/acl* | PolicyModule | ACLService | config_sync |

| RuleController (CC) | /rules/cc/* | PolicyModule | CCRuleService | config_sync |

| TaskController | /tasks* | TaskModule | TaskService, DispatchService | refresh/preheat/clear_cache |

| PackageController | /packages* | PackageModule | AgentPackageService | agent_upgrade 任务 |

| UserPackageController | /user_packages* | PackageModule | UserPackageService | package_sync 任务 |

| PlanController | /plans* /user_plans* | PackageModule | PlanService | 间接影响套餐 |

| DNSController | /dns* | DNSModule | DNSService | DNS task worker |

| DNSAPIController | /dnsapi* | DNSModule | DNSAPIService | DNS task worker |

| CnameController | /cname_domains* | DNSModule | CnameDomainService | DNS 同步 |

| LogController | /logs/* | TelemetryModule | LogQueryService | 只读 |

| StatController | /stats/* /usage | TelemetryModule | StatService | 只读 |

| AgentController | /agent/* | AgentGateway | AgentConfigService | config_sync / heartbeat |

| AgentWSController | /ws/agent | AgentGateway | AgentDispatchHub | 任务派发、心跳 |

| AgentLogController | /agent/logs/* | TelemetryModule | LogIngestService | ClickHouse |

| FinanceController | /orders /recharge | BillingModule | BillingService | 可能影响套餐 |

| MessageController | /messages* | NotificationModule | MessageService | 只读/已读 |

| UploadController | /upload/* | MediaModule | UploadService | 上传文件 |

| ForwardController | /forwards* | StreamModule | ForwardService | config_sync/streams |

| ForwardGroupController | /forward_groups* | StreamModule | ForwardGroupService | config_sync |

| ForwardDefaultController | /forward_defaults* | StreamModule | ForwardDefaultService | config_sync |

| ForwardMonitorController | /forward/traffic | TelemetryModule | ForwardStatService | 只读 |



### 18.2 Agent 接口映射

| 旧接口 | C# 通道 | 新服务 | 说明 |
|---|---|---|---|
| /agent/heartbeat | WS 心跳 | AgentHeartbeatService | 单连接心跳 |
| /agent/config | WS 配置推送 + HTTP fallback(可选) | AgentConfigService | C# 默认仅推送 |
| /agent/tasks | WS 任务派发 + HTTP fallback(可选) | AgentTaskService | C# 默认仅 WS |
| /agent/l2/nodes | WS 请求/响应 | AgentL2Service | L2 节点列表 |
| /agent/logs/* | WS Telemetry | AgentTelemetryService | 日志/指标/事件 |
| WS /ws/agent | WS 主通道 | AgentDispatchHub | 保留 WS，禁止弃用 |

### 18.3 管理后台实时推送（Blazor Server + SignalR）

**目标**
- 管理后台页面无需刷新即可同步状态（节点/任务/安装进度/告警/消息）
- 事件统一由 API 层发布，Blazor Server 自动刷新 UI

**通道与鉴权**
- Hub：`/ws/admin`（SignalR）
- 认证：复用后台 JWT（或 Cookie），禁止匿名连接
- 分组：`admin` / `user:{uid}` / `tenant:{tid}`（若多租户）

**事件清单（最小闭环）**
- `node.status.changed`：节点在线/离线/启停变化  
  - payload：`{node_id, online, enable, checked_at}`
- `node.install.progress`：安装进度  
  - payload：`{node_id, stage, percent, current_bytes, total_bytes, message}`
- `task.state.changed`：任务状态变更  
  - payload：`{task_id, type, state, progress, ret, updated_at}`
- `config.version.bumped`：配置版本变更  
  - payload：`{type, ids, version, bumped_at}`
- `message.new`：系统消息/公告  
  - payload：`{id, title, time, level}`

**触发源**
- Agent WS 事件（心跳/日志/任务回执）→ 转换为后台事件推送
- API 写库或任务状态变更 → 直接推送

**C# 重构要求**
- SignalR 事件必须幂等（以 `task_id/node_id` 为主键）
- 断线自动重连（前端）+ 服务端保留最近 N 条事件（可选）
- 前端所有状态更新 **必须** 能由事件驱动（不依赖手动刷新）



---



## 19. C# 项目结构与实现边界（强约束）



### 19.1 项目结构（项目层面合并）

```
/src
  /Cnn.Api                     # API + Blazor Server（管理后台）+ Domain + Infrastructure
    /Controllers               # API 层（DTO/鉴权/校验）
    /Domain                    # 领域模型/值对象/业务规则
    /Infrastructure            # SqlSugar/文件系统/外部服务
    /Application               # 可选：应用服务层（编排 Domain/Infra）
    /Contracts                 # DTO / OpenAPI 契约
    /Schema                    # JSON Schema
    /Pages                     # Blazor Pages
    /Shared                    # Blazor 共享组件
    /Services                  # Blazor 前端服务
    /Data                      # Blazor 数据模型
    /wwwroot                   # 静态资源

  /Cnn.Agent                   # 节点/网关

/tests
  /ContractTests               # 协议一致性测试
  /IntegrationTests            # 端到端测试
```

### 19.2 分层规则（必须遵守）



- **Controller 不允许直接访问 DB**
- **Application 层必须调用 Domain Service**
- **Domain 层禁止依赖 Infrastructure**
- **所有 DTO 必须来自 Cnn.Api/Contracts**
- **配置 Schema 必须来自 Cnn.Api/Schema**


### 19.3 代码歧义避免规范



- 所有“状态”必须用枚举/常量集中定义
- 所有“字符串字段”必须统一正则校验（域名/端口/URL/IP）
- 所有“配置写入”必须使用原子写与版本校验


---



## 20. 协议规范与文档生成机制（避免歧义）


### 20.1 API 响应统一格式（强制）



```json

{

  "code": 200,

  "message": "ok",

  "data": {},

  "trace_id": "..."

}

```



- `code=200` 表示成功，其他为业务错误
- `trace_id` 必须全链路贯通（API/Agent/Task）


### 20.2 错误码体系（强制）


- 业务错误码必须集中在 `ErrorCodes.cs`
- 必须包含：`code`、`http_status`、`message`


### 20.3 版本与兼容规则


- REST：`/api/v2/...`

- WS 协议：消息结构必须版本化，不允许破坏性变更
- Schema：`schema_version` 必填，变更必须记录


### 20.4 文档生成



- REST 文档：OpenAPI 自动生成（Swagger）
- WS 文档：消息结构 + 手工补充语义文档
- 配置文档：JSON Schema + 示例自动生成
- CI 中强制校验：

  - Schema 与代码一致
  - 示例 Schema 校验通过



---



（新增：映射表 + 项目结构 + 协议规范）


---



## 21. 全量功能重写逻辑（逐模块标准化流程）


> 统一格式：入口 / 校验 / 持久化 / 触发与副作用 / 返回。开发者按模板实现，不允许省略步骤


### 21.1 Auth / Register

- 入口：`/api/v2/login`、`/api/v2/register`、`/api/v2/login/captcha`

- 校验：账号存在且启用、密码校验、验证码/限流

- 持久化：登录日志、验证码记录

- 副作用：签发 JWT / 刷新 token（滑动续期）

- 返回：`{token, role, uid, name}` 或错误码

**C# 重构要求**
- 兼容 `username` 为 `name` 或 `email`；`password` 支持明文与 hash（`password_hash=sha256` 或自动识别）
- 保持登录限流与验证码逻辑（email/sms），并记录 `login_log`
- JWT 过期策略与滑动续期保持一致（必要时回写 `X-Auth-Token`）
- 统一返回结构 `{code,message,data}` + i18n



### 21.2 用户管理（Admin）
- 入口：`/api/v2/admin/users/*`

- 校验：管理员权限

- 持久化：`user` 表更新
- 副作用：封禁/启用影响登录；重置清理额度
- 返回：标准响应

**C# 重构要求**
- 严格区分 Admin/User 权限边界（User 端只允许操作自身资源）
- 保持字段一致（不新增/不重命名），变更只更新必要字段
- 启用/禁用必须可追溯（写操作日志）
- 统一返回结构 + i18n


### 21.3 节点管理

- 入口：`/api/v2/admin/nodes/*`

- 校验：节点信息完整性（IP/SSH/Region）
- 持久化：`node` 表、子 IP 节点

- 副作用：

  - 更新 `config_task` 触发启停

  - DNS CNAME 同步

  - 自动安装：创建安装进度与触发 SSH 安装

- 返回：节点详情列表

**C# 重构要求**
- 保持 `config_task` 语义与节点启停流程（Agent 心跳触发 `applyNodeSync`）
- 维持节点 Token/状态字段与现有表一致（不新增字段）
- 节点变更需触发 DNS 同步与配置版本变更
- 统一返回结构 + i18n



### 21.4 线路与线路分配

- 入口：`/api/v2/admin/node-groups/*`

- 校验：
  - 节点归属/区域一致
  - 节点不重复归属
- 持久化：`node_group` / `line`

- 副作用：

  - DNS 解析记录同步

  - `BumpConfigVersion(line/group)`

- 返回：线路分配结果

**C# 重构要求**
- 线路组与解析配置必须与 CNAME 域名列表联动校验
- 线路变更必须触发 DNS 同步与配置版本变更
- 保持分配规则与前端行为一致（禁止重复分配）
- 统一返回结构 + i18n


### 21.5 站点管理

- 入口：`/api/v2/*/sites/*`

- 校验：域名配置/所属关系
- 持久化：`site` + `merge_site_group`

- 副作用：

  - `BumpConfigVersion(site)`

  - DNS CNAME 同步

  - HTTPS 证书联动

- 返回：站点配置

**C# 重构要求**
- 创建站点必须绑定套餐；CNAME/区域/线路组默认取套餐配置
- 删除站点前必须先禁用站点（禁止直接删除启用态站点）
- 套餐变更后，已落地的 CNAME/区域/线路组不应自动变化（保持现有行为）
- 站点变更触发 `config_sync` 与 DNS 同步；返回结构统一 + i18n


### 21.6 证书管理

- 入口：`/api/v2/*/certs/*`

- 校验：域名合法、DNS API 可用、证书冲突
- 持久化：`cert` 
- 副作用：

  - 创建 `issue_cert` Task

  - `BumpConfigVersion(cert)`

- 返回：证书状态

**C# 重构要求**
- 证书签发支持 Agent 路径与 API 本地路径（DNS challenge），保持现有任务/状态流转
- 回写必须更新 `task.state/task.ret` 与配置版本（触发 `config_sync`）
- 保留 `rate_limited` / `rate_cooldown` 等字段语义
- 统一返回结构 + i18n


### 21.7 ACL / CC / WAF 规则

- 入口：`/api/v2/*/rules/*`

- 校验：
  - 用户权限

  - 规则字段合法

- 持久化：`acl` / `cc_rule` / `cc_match` / `cc_filter`

- 副作用：`BumpConfigVersion(rule)` `config_sync`

- 返回：规则对象

**C# 重构要求**
- 规则字段必须与现有 JSON 结构一致（不新增/不重命名）
- 规则优先级与排序保持一致（命中即停止）
- 任何规则变更必须触发配置同步
- 统一返回结构 + i18n


### 21.8 套餐与用户套餐

**入口**
- 套餐（Plan/Package）
  - 管理端：`GET/POST/PUT/DELETE /api/v1/admin/plans`
  - 详情：`GET /api/v1/admin/plans/{id}`
  - 用户端：`GET /api/v1/user/plans` / `GET /api/v1/user/plans/{id}`
- 用户套餐（UserPackage）
  - 列表：`GET /api/v1/admin/user_packages` / `GET /api/v1/user/user_packages`
  - 更新（用户）：`PUT /api/v1/user/user_packages/{id}`
  - 续费（用户）：`POST /api/v1/user/user_packages/{id}/renew`
  - 切换套餐（用户）：`POST /api/v1/user/user_packages/{id}/switch`
  - 分配/更新/删除（管理端）：`POST /api/v1/admin/user_plans/assign` / `PUT /api/v1/admin/user_plans/{id}` / `DELETE /api/v1/admin/user_plans`


**数据落地**
- `package`：套餐定义（产品维度）
- `user_package`：用户购买实例（运行时维度）
- `config`：`user_package_config`（ipv6/http3_enabled 等开关）


**通用校验**
- `region_id` 必须存在
- `node_group_id` 必须存在
- `backup_group_id != node_group_id`，且非 0 时必须存在
- `end_at` 必须是未来时间（Assign/Update）
- `package_allow_upgrade` / `package_allow_downgrade` 控制切换


**Plan 创建/更新**
- 字段映射：
  - `traffic_limit/bandwidth_limit/connection_limit/domain_limit`
  - `http_port/stream_port`
  - `custom_cc_rules/websocket/l2_origin`
  - `expire/buy_num_limit/backend_ip_limit/id_verify/before_exp_days_renew`
  - `cname_domain/cname_hostname2/cname_mode`
- `backup_group` 变更时再次校验与线路组冲突


**用户套餐列表**
- `status` 由 `EndAt < now` 派生（active/expired）
- `record_id` 为空时自动生成并写回
- `ipv6/http3_enabled` 读取自 `config`（type=user_package_config）
- 返回包含 `version/is_expired`（运行时字段）


**用户套餐更新（用户/管理员差异）**
- 用户侧：`region/node_group/backup_group` 仅允许传 >0 的值
- 管理端：允许写 0（清空）
- 价格字段仅管理端可更新
- `cname_*`：用户侧仅可非空更新；管理端可写空
- `ipv6/http3_enabled` 写入 `config`
- 触发 `package_sync`


**续费**
- `months` 优先；否则 `period=month/quarter/year`
- `end_at = max(now, end_at) + months`
- 触发 `package_sync`


**切换套餐**
- 评分策略：先价格（月/季/年），否则资源分（traffic+bandwidth+connection+domain）
- `package_allow_upgrade` / `package_allow_downgrade` 控制开关
- 替换字段：`name/package/region/node_group/backup_group/traffic/bandwidth/connection/domain/custom_cc_rule/websocket/l2_origin/price_*`
- 触发 `package_sync` + 站点 CNAME 重算


**分配用户套餐**
- 从 `package` 复制字段到 `user_package`
- `start_at=now`，`end_at` 根据 `duration_months` 或 `end_at`
- 自动生成 `record_id`
- 触发 `package_sync`


**更新用户套餐（管理端）**
- `cname_mode` 变更时：若站点 `cname_mode` 为空则补旧值
- 触发 `package_sync`


**副作用**
- `package_sync` Task（见 15.7）
- `user_package.version` / `user_package.is_expired` 为运行时字段（db.sql 无列，启动时补列）


**C# 重构要求**
- 保持字段/限制字段一致（不新增字段）
- `package_sync` 必须按节点范围下发并写入 `packages/*.json`
- 套餐变更只影响后续策略，不强制重算已落地站点 CNAME/区域/线路组
- `user_package_config` 与 `config` 表一致
- 统一返回结构 + i18n


### 21.9 清理/预热任务

- 入口：`/api/v2/*/tasks`

- 校验：域名归属、配额
- 持久化：`task` 
- 副作用：Agent 执行清理/预热

- 返回：task_id

**C# 重构要求**
- 任务状态机必须遵循第 17 章约束（包含多节点任务回写）
- `refresh_url/refresh_dir/clear_cache/preheat` 与现有行为完全一致
- 任务执行回写必须更新 `task.state/progress/ret`
- 统一返回结构 + i18n



### 21.10 DNS CNAME

- 入口：`/api/v2/admin/dns*` / `dnsapi` / `cname_domains`

- 校验：DNS provider 凭证

- 持久化：`dnsapi` / `cname_domains`

- 副作用：DNS 任务执行

- 返回：记录状态

**C# 重构要求**
- CNAME 域名删除必须做引用检查（Site/Forward/NodeGroup/Package/UserPackage/Plan）
- 域名规范化与校验规则必须与 Go 一致
- DNS Provider 删除必须阻止存在引用的 CNAME 域名
- 统一返回结构 + i18n


### 21.11 日志与统计
- 入口：`/api/v2/*/logs`、`/api/v2/*/stats`

- 校验：权限
- 持久化：ClickHouse 查询（只读）

- 副作用：无
- 返回：列表/聚合统计结构

**C# 重构要求**
- ClickHouse 字段与查询口径保持一致（HostFilter 规则不变）
- 未启用 ClickHouse 时返回空结构（保持现有行为）
- 不新增数据库字段；仅查询聚合
- 统一返回结构 + i18n


### 21.12 Agent 网关

- 入口：`/api/v2/agent/*`（WS 主通道 + HTTP 兜底）

- 校验：Agent token

- 持久化：

  - Node 状态心跳
  - Task ack/progress

  - 日志/指标

- 副作用：配置下发、任务派发

**C# 重构要求**
- 采用 WS 长连接作为主通道（握手/心跳/任务派发/回写）
- 保留 HTTP 拉取作为兜底（断线恢复）
- 任务回写必须包含 `node_id` + `task_id` + `state`
- 统一返回结构 + i18n


---



## 22. 文档无歧义写作清单（强制检查表）


### 22.1 必须具备的内容
- [x] 接口路径 + 方法

- [x] 参数类型/必填/默认值/示例

- [x] 返回码与错误结构
- [x] 状态字段状态机

- [x] 副作用（配置/任务/日志）


### 22.2 禁止出现的描述
- “可能”、“大概”、“需要自行判断”
- 未定义字段
- 未描述的默认值


### 22.3 一致性规范
- 字段命名全程一致（同字段不得出现多种命名）

- 状态名必须统一（不得混用 waiting/pending）
- 返回结构必须统一（`code/message/data/trace_id`）


---



## 23. 验收标准（开发与文档同步）


### 23.1 功能验收

- API 功能 100% 可在新系统执行
- 关键链路（站证书/配置/任务）端到端通过



### 23.2 文档验收

- Schema 校验通过

- 示例可用（Schema 验证成功）
- 每个功能都有流程与状态机



### 23.3 测试验收

- Contract Tests：接口入出参一致
- Integration Tests：完整流程通过

- 性能指标达标（并发吞吐/延迟）


---



（新增：全量功能流程模板 + 无歧义写作清单 + 验收标准）


## 24. C# 重写错误码字典（统一语义，禁止私有含义）



> 目标：所有 API/Agent/WS 均引用同一错误码字典，避免“同码不同义”


### 24.1 统一返回结构与本地化规则

- 成功响应示例：`{"code":200,"message":"Success","data":...,"trace_id":"..."}`
- 错误响应示例：`{"code":4xxxx/5xxxx,"message":"...","data":null,"trace_id":"...","error":{"field":"...","reason":"...","detail":"..."}}`
- `message` 必须根据前端语言自动本地化（优先 `Accept-Language`，其次 `lang`）
- 当前语言范围：`zh-CN` / `en-US`（未命中时回退默认语言）
- `trace_id` 必须贯穿 API / Agent / Task / 日志链路（可用于全链路追踪）

### 24.2 错误码建议分段（可直接落地）

**统一规则**
- `code` 与 HTTP 状态码解耦；前端只依赖 `code`
- HTTP 可保留语义（401/403/500），但**所有业务错误必须有明确 code**

**基础错误码（所有模块通用）**

| code | http | message | 中文语义 | 典型场景 |
|---|---|---|---|---|
| 200 | 200 | success | 成功 | - |
| 40001 | 400 | invalid_param | 参数非法 | 格式错误/范围错误 |
| 40002 | 400 | missing_param | 缺少必填 | 必填字段为空 |
| 40101 | 401 | auth_invalid | 认证失败 | token 无效/缺失 |
| 40102 | 401 | auth_expired | 认证过期 | token 过期 |
| 40301 | 403 | permission_denied | 权限不足 | 非管理员访问 admin |
| 40401 | 404 | not_found | 资源不存在 | site/node/cert 不存在 |
| 40901 | 409 | already_exists | 资源已存在 | 重复创建 |
| 40902 | 409 | in_use | 资源被引用 | CNAME/DNSAPI 被占用 |
| 40903 | 409 | state_conflict | 状态冲突 | 禁用状态操作/版本冲突 |
| 41201 | 412 | precondition_failed | 前置条件不足 | 未满足依赖关系 |
| 42901 | 429 | rate_limited | 限流 | 频率超限 |
| 42902 | 429 | quota_exceeded | 配额耗尽 | 日配额不足 |
| 50001 | 500 | internal_error | 内部错误 | 未分类服务异常 |
| 50002 | 500 | db_error | 数据库错误 | 读写失败 |
| 50003 | 500 | config_error | 配置错误 | 配置解析失败 |
| 50201 | 502 | external_provider_error | 外部系统错误 | DNS/ACME/支付失败 |
| 50301 | 503 | service_unavailable | 服务不可用 | 维护/依赖不可用 |
| 50302 | 503 | task_queue_full | 任务队列拥塞 | 批量任务峰值 |
| 50401 | 504 | timeout | 超时 | 上游/任务超时 |

**Agent / WS 专用**
| code | http | message | 中文语义 | 典型场景 |
|---|---|---|---|---|
| 60001 | 503 | agent_offline | 节点离线 | 下发失败 |
| 60002 | 401 | agent_auth_failed | Agent 鉴权失败 | token 错误 |
| 60003 | 400 | agent_version_mismatch | 版本不匹配 | 需要升级 |
| 60004 | 409 | agent_task_reject | 任务拒绝 | 节点忙/条件不满足 |
| 61001 | 503 | ws_not_connected | WS 未连接 | 需要重连 |

> 说明：必须提供 `ErrorCodes.cs`（或等价常量类）+ `error_codes.md`，并在 CI 中校验：任何新增 code 必须文档同步


---



## 25. C# 重写协议草案（OpenAPI + WS 无歧义模板）



### 25.1 OpenAPI 结构要求（REST）
- 版本：`OpenAPI 3.1`

- 统一组件：`components/schemas/CommonResponse`、`components/schemas/ErrorResponse`

- 必须声明 `trace_id` 与 `error` 结构

- 所有路由必须显式声明 `Auth`（Bearer / AgentToken）


**示例（节选）**

```yaml

paths:

  /api/v2/sites:

    post:

      summary: 创建站点

      security: [{ bearerAuth: [] }]

      requestBody:

        required: true

        content:

          application/json:

            schema:

              $ref: '#/components/schemas/SiteCreateReq'

      responses:

        '200':

          description: OK

          content:

            application/json:

              schema:

                $ref: '#/components/schemas/CommonResponse'

```



### 25.2 WS 消息结构（Agent/Task/Config）
- 版本：`ws.v1`（可选 `schema_version`）
- 任务消息：`{kind,msg_id,task{task_id,task_type,task_name,payload}}` / Ack：`{kind,msg_id,task_id,status,error?,ret?}`
- 配置消息：`{kind,data}`（`edge_config`/`cache_config`）
- 必须支持双向 WS（实时任务/心跳/配置推送）；HTTP 兜底仅作为可选扩展

**示例（节选）**

```json
{"kind":"hello","node_id":"1001","token":"***","version":"1.0"}
{"kind":"ping","ts":1700000000}
{"kind":"task_dispatch","msg_id":"task-123-1001","task":{"task_id":123,"task_type":"issue_cert","task_name":"Issue Cert 123","payload":"{...}"}}
{"kind":"task_ack","msg_id":"task-123-1001","task_id":123,"status":"success","ret":""}
{"kind":"edge_config","data":{...}}
```

> 说明：WS 消息必须附带字段级语义说明（文档与注释同步）


---



## 26. YARP 路由/集群映射规范（EdgeConfigV2 -> ProxyConfig）


### 26.1 映射规则（强制一致）

- **Site -> Route**：每个站点域名生成一Route；`RouteId = site_id:domain`
- **Backend/Origin -> Cluster**：每个后端组生成 Cluster；`ClusterId = site_id:backend_id`
- **Origin -> Destination**：源站列表映射到 `Destinations`；权重映射到 `LoadBalancing`
- **TLS/SNI**：证书由 `ICertificateSelector` 动态选择；SNI 绑定 `domain`
- **HTTP/HTTPS 强制**：在 Route 上加 Redirect 规则（或 Middleware）
- **ACL/CC/WAF**：进YARP 前执`SecurityPipeline`（禁止在 Proxy 后处理）
- **Cache**：使用自研响应缓存中间件 + 磁盘缓存（替Nginx 缓存语义）


### 26.2 动态配置更新
- 监听 `config_version`，变化即生成`IProxyConfig` 快照
- 更新必须原子替换，旧配置继续服务已建立连接
- 配置生成过程必须可回滚（生成失败时保留旧版本）


**示例（节选）**

```json

{

  "Routes": [

    {

      "RouteId": "site:1001:example.com",

      "Match": { "Hosts": ["example.com"] },

      "ClusterId": "cluster:1001:backend:1"

    }

  ],

  "Clusters": {

    "cluster:1001:backend:1": {

      "Destinations": {

        "origin-1": { "Address": "http://10.0.0.1:8080" },

        "origin-2": { "Address": "http://10.0.0.2:8080" }

      },

      "LoadBalancingPolicy": "LeastRequests"

    }

  }

}

```



---



## 28. C# DTO/校验规范（避免歧义的强制规则）


### 28.1 字段命名与类
- JSON 字段命名必须与现 API 一致（snake_case）
- `id`/`*_id` 统一`long`，时间字段统一Unix 秒或 RFC3339（必须明确并全局统一）
- 所有 `status/state` 字段必须映射到枚举并提供字典表


### 28.2 统一校验策略

- 任何校验规则必须**单一位置** 定义（`ValidationSpecs`），禁止散落
- 校验规则必须可序列化输出到文档（CI 校验）
- 典型校验项必须包含：

  - 域名格式/去重

  - 端口范围

- 权限与归属
  - 资源配额



### 28.3 DTO 与数据库的映射约束
- DTO 字段必须与数据库字段保持一一映射；任何差异需在文档中声明原因
- JSON 字段（如 `settings`/`rules`）必须配 Schema（见 16 节）


---



## 29. C# 重写实施顺序（避免遗漏与返工）


1) **接口对照冻结**：以第 9 节接口清单为单一事实来源，逐条映射 C# Controller/Endpoint
2) **模型 Schema 固化**：完成 `EdgeConfigV2`/规则 Schema，启用 CI 校验
3) **任务系统复刻**：按第 13/17 节实现 Task 生命周期与回写
4) **Agent 通信落地**：先实现 PullConfig/Heartbeat，再扩展 Task/Log/Metric 流
5) **YARP 数据面接入**：从站点/域名最小集开始，逐步补齐 ACL/CC/WAF/Cache
6) **全链路压测**：对比旧系统性能与稳定性指标


---



（新增：错误码字典 + OpenAPI/WS 模板 + YARP 映射 + DTO/校验规范 + 实施顺序）


## 30. OpenAPI 3.1 全量细化版（逐接口 Request/Response DTO）


> 本节以源码路由为单一事实来源，按分组列出**每个接口**的路径、鉴权、参数与 DTO 绑定。DTO 字段详见 30.3 Schema 字段表


### 30.1 OpenAPI 3.1 骨架（统一规范）


```yaml

openapi: 3.1.0

info:

  title: GoEdge CDN API

  version: v1

servers:

  - url: https://{host}

security:

  - bearerAuth: []

components:

  securitySchemes:

    bearerAuth:

      type: http

      scheme: bearer

      bearerFormat: JWT

    agentAuth:

      type: apiKey

      in: header

      name: Authorization

  schemas:

    CommonResponse:

      type: object

      required: [code, msg, data, trace_id]

      properties:

        code: { type: integer }

        msg: { type: string }

        data: { type: object }

        trace_id: { type: string }

    ErrorResponse:

      allOf:

        - $ref: '#/components/schemas/CommonResponse'

        - type: object

          properties:

            error:

              type: object

              properties:

                field: { type: string }

                reason: { type: string }

                detail: { type: string }

```



### 30.2 路由-DTO 对照表（自动抽取 + 人工补充）


> 说明：Body 若为空表示无 requestBody；Path/Query 为控制器内显式读取的参数名；实际字段结构请对30.3 DTO/Model


| 方法 | 路径 | Auth | Path 参数 | Query 参数 | Body DTO | Response |

|---|---|---|---|---|---|---|

| GET | /.well-known/acme-challenge/:token | None | token | - | - | CommonResponse |

| GET | /api/v1/acls | None | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| DELETE | /api/v1/admin/announcements/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/announcements/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/announcements | Bearer(admin) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/admin/announcements | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/api_key/reset | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/api_key | Bearer(admin) | - | - | - | CommonResponse |

| PUT | /api/v1/admin/api_key | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/certs/:id/dns_challenge | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/certs/:id/download | Bearer(admin) | id | domain,keyword,page,pageSize,search_field,user_id | - | CommonResponse |

| POST | /api/v1/admin/certs/:id/verify_dns | Bearer(admin) | id | - | - | CommonResponse |

| DELETE | /api/v1/admin/certs/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/certs/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/certs/batch/:id/progress | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/certs/batch | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/certs/batch | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/certs/batch_action | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/certs/default_settings | Bearer(admin) | - | user_id | - | CommonResponse |

| POST | /api/v1/admin/certs/default_settings | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/certs/reissue | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/certs/wildcard | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/certs | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/certs | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/cname_domains/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/cname_domains/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/cname_domains | Bearer(admin) | - | keyword,page,pageSize | - | CommonResponse |

| POST | /api/v1/admin/cname_domains | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/config_items | Bearer(admin) | - | type,scope_name,scope_id | - | CommonResponse |

| POST | /api/v1/admin/config_items | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/dashboard | Bearer(admin) | - | range | - | CommonResponse |

| DELETE | /api/v1/admin/dns/providers/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/dns/providers/types | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/dns/providers | Bearer(admin) | - | user_id | - | CommonResponse |

| POST | /api/v1/admin/dns/providers | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/dns/records/cleanup | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/dns/records/fix | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/dns/test | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/dnsapi/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/dnsapi/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/dnsapi/types | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/dnsapi | Bearer(admin) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/admin/dnsapi | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/domain_usage | Bearer(admin) | - | user_id,user_package_id | - | CommonResponse |

| GET | /api/v1/admin/domains | Bearer(admin) | - | keyword,page,pageSize | - | CommonResponse |

| GET | /api/v1/admin/forward/ranking | Bearer(admin) | - | range | - | CommonResponse |

| GET | /api/v1/admin/forward/traffic | Bearer(admin) | - | keyword,range | - | CommonResponse |

| DELETE | /api/v1/admin/forward_defaults | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/forward_defaults | Bearer(admin) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/admin/forward_defaults | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/forward_groups | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/forward_groups | Bearer(admin) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/admin/forward_groups | Bearer(admin) | - | - | - | CommonResponse |

| PUT | /api/v1/admin/forward_groups | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/forwards/:id | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/forwards/batch | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/forwards/batch_action | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/forwards/batch_update | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/forwards | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/forwards | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/global_config | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/global_config | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/login/captcha | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/login | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/logs/access | Bearer(admin) | - | cache_status,client_ip,domain,domain_mode,end_time,keyword,method,node_id,node_ip,page,pageSize,port,referer,scheme,ssl_cipher,ssl_protocol,start_time,status,status_max,status_min,uri,uri_mode,user_agent | - | CommonResponse |

| GET | /api/v1/admin/logs/backup | Bearer(admin) | - | keyword,page,pageSize | - | CommonResponse |

| GET | /api/v1/admin/logs/block/current | Bearer(admin) | - | keyword,type | - | CommonResponse |

| GET | /api/v1/admin/logs/block/history | Bearer(admin) | - | end_time,keyword,page,pageSize,range,start_time,time_range,type | - | CommonResponse |

| GET | /api/v1/admin/logs/block/stats | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/logs/login | Bearer(admin) | - | keyword,page,pageSize | - | CommonResponse |

| GET | /api/v1/admin/logs/mail | Bearer(admin) | - | keyword,page,pageSize | - | CommonResponse |

| GET | /api/v1/admin/logs/operation | Bearer(admin) | - | keyword,page,pageSize | - | CommonResponse |

| GET | /api/v1/admin/messages | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/monitor_config | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/monitor_config | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/node-groups/:id/resolution/action | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/node-groups/:id/resolution/assign | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/node-groups/:id/resolution | Bearer(admin) | id | line_id | - | CommonResponse |

| DELETE | /api/v1/admin/node-groups/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/node-groups/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/node-groups | Bearer(admin) | - | keyword,limit,page,region_id | - | CommonResponse |

| POST | /api/v1/admin/node-groups | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/nodes/:id/install | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/nodes/:id/monitor_logs | Bearer(admin) | id | end,page,pageSize,start,type | - | CommonResponse |

| PUT | /api/v1/admin/nodes/:id/status | Bearer(admin) | id | - | - | CommonResponse |

| DELETE | /api/v1/admin/nodes/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/nodes/:id | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/nodes/batch | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/nodes/batch_action | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/nodes | Bearer(admin) | - | version | - | CommonResponse |

| POST | /api/v1/admin/nodes | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/orders | Bearer(admin) | - | keyword,page,pageSize | - | CommonResponse |

| POST | /api/v1/admin/packages/grayscale | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/packages/nodes | Bearer(admin) | - | version | - | CommonResponse |

| POST | /api/v1/admin/packages/stable | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/packages/upgrade/status | Bearer(admin) | - | task_id | - | CommonResponse |

| POST | /api/v1/admin/packages/upgrade | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/packages | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/packages | Bearer(admin) | - | - | DTO.uploadVersionReq | CommonResponse |

| DELETE | /api/v1/admin/plans/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/plans/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/plans/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/plans | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/plans | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/recharge | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/regions/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/regions/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/regions | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/regions | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/rules/acl/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/rules/acl/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/rules/acl/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/rules/acl | Bearer(admin) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/admin/rules/acl | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/rules/cc/filters/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/rules/cc/filters/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/rules/cc/filters/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/rules/cc/filters | Bearer(admin) | - | name,status | - | CommonResponse |

| POST | /api/v1/admin/rules/cc/filters | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/rules/cc/groups/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/rules/cc/groups/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/rules/cc/groups | Bearer(admin) | - | name,status | - | CommonResponse |

| POST | /api/v1/admin/rules/cc/groups | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/rules/cc/matchers/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/rules/cc/matchers/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/rules/cc/matchers | Bearer(admin) | - | name,status | - | CommonResponse |

| POST | /api/v1/admin/rules/cc/matchers | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/site_defaults/:name | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/site_defaults/:name | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/site_defaults | Bearer(admin) | - | scope_name,scope_id,user_id | - | CommonResponse |

| POST | /api/v1/admin/site_defaults | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/site_groups/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/site_groups/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/site_groups | Bearer(admin) | - | keyword,page,pageSize,user_id | - | CommonResponse |

| POST | /api/v1/admin/site_groups | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/sites/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/sites/:id | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/sites/apply_cert | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/sites/batch/:id/progress | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/sites/batch | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/sites/batch_action | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/sites/batch_update | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/sites/export | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/sites/resolve | Bearer(admin) | - | domain | - | CommonResponse |

| GET | /api/v1/admin/sites | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/sites | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/stats/basic | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/stats/node_metrics | Bearer(admin) | - | end_time,metric,range,start_time,time_range,window | - | CommonResponse |

| GET | /api/v1/admin/stats/node_ranking | Bearer(admin) | - | metric,window | - | CommonResponse |

| GET | /api/v1/admin/stats/node_traffic | Bearer(admin) | - | end_time,exclude_nic,node_id,start_time,window | - | CommonResponse |

| GET | /api/v1/admin/stats/origin | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/stats/quality | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/stats/ranking | Bearer(admin) | - | keyword,type | - | CommonResponse |

| GET | /api/v1/admin/system_info | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/tasks/:id/resubmit | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/tasks/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/tasks/usage | Bearer(admin) | - | user_id | - | CommonResponse |

| GET | /api/v1/admin/tasks | Bearer(admin) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/admin/tasks | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/upload/image | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/user_packages | Bearer(admin) | - | user_id | - | CommonResponse |

| PUT | /api/v1/admin/user_plans/:id | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/user_plans/assign | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/admin/user_plans | Bearer(admin) | - | - | - | CommonResponse |

| GET | /api/v1/admin/user_plans | Bearer(admin) | - | - | - | CommonResponse |

| POST | /api/v1/admin/users/:id/impersonate | Bearer(admin) | id | - | - | CommonResponse |

| POST | /api/v1/admin/users/:id/purge/reset | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/users/:id/status | Bearer(admin) | id | - | - | CommonResponse |

| DELETE | /api/v1/admin/users/:id | Bearer(admin) | id | - | - | CommonResponse |

| PUT | /api/v1/admin/users/:id | Bearer(admin) | id | - | - | CommonResponse |

| GET | /api/v1/admin/users | Bearer(admin) | - | keyword,page,pageSize | - | CommonResponse |

| POST | /api/v1/admin/ws/dispatch | Bearer(admin) | - | - | - | CommonResponse |

| DELETE | /api/v1/agent/acme/tokens/:token | AgentToken | token | - | - | CommonResponse |

| POST | /api/v1/agent/acme/tokens | AgentToken | - | - | - | CommonResponse |

| POST | /api/v1/agent/certs/issued | AgentToken | - | - | - | CommonResponse |

| GET | /api/v1/agent/config | AgentToken | id | - | - | CommonResponse |

| POST | /api/v1/agent/heartbeat | AgentToken | - | - | DTO.struct | CommonResponse |

| POST | /api/v1/agent/l2/heartbeat | AgentToken | - | - | - | CommonResponse |

| GET | /api/v1/agent/l2/nodes | AgentToken | - | - | - | CommonResponse |

| POST | /api/v1/agent/logs/access | AgentToken | - | - | - | CommonResponse |

| POST | /api/v1/agent/logs/events | AgentToken | - | - | - | CommonResponse |

| POST | /api/v1/agent/logs/metrics | AgentToken | - | - | - | CommonResponse |

| POST | /api/v1/agent/node/sync | AgentToken | - | node_id | - | CommonResponse |

| POST | /api/v1/agent/tasks/:id/finish (可选) | AgentToken | id | node_id | - | CommonResponse |

| GET | /api/v1/agent/tasks (可选) | AgentToken | - | node_id | - | CommonResponse |

| GET | /api/v1/agent/upgrade/package | AgentToken | - | version | - | CommonResponse |

| GET | /api/v1/agent/upgrade | AgentToken | - | - | - | CommonResponse |

| POST | /api/v1/login/captcha | None | - | - | - | CommonResponse |

| POST | /api/v1/login | None | - | - | - | CommonResponse |

| POST | /api/v1/register | None | - | - | - | CommonResponse |

| GET | /api/v1/system_info | None | id | - | - | CommonResponse |

| POST | /api/v1/user/api_key/reset | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/api_key | Bearer(user) | - | - | - | CommonResponse |

| PUT | /api/v1/user/api_key | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/certs/:id/dns_challenge | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/certs/:id/download | Bearer(user) | id | domain,keyword,page,pageSize,search_field,user_id | - | CommonResponse |

| POST | /api/v1/user/certs/:id/verify_dns | Bearer(user) | id | - | - | CommonResponse |

| DELETE | /api/v1/user/certs/:id | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/certs/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/certs/batch/:id/progress | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/certs/batch | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/certs/batch | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/certs/batch_action | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/certs/default_settings | Bearer(user) | - | user_id | - | CommonResponse |

| POST | /api/v1/user/certs/default_settings | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/certs/reissue | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/certs/wildcard | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/certs | Bearer(user) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/user/certs | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/config_items | Bearer(user) | - | type | - | CommonResponse |

| POST | /api/v1/user/config_items | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/dashboard | Bearer(user) | - | range | - | CommonResponse |

| GET | /api/v1/user/dns/providers/types | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/dns/providers | Bearer(user) | - | user_id | - | CommonResponse |

| GET | /api/v1/user/dnsapi/types | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/dnsapi | Bearer(user) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| GET | /api/v1/user/domain_usage | Bearer(user) | - | user_id,user_package_id | - | CommonResponse |

| GET | /api/v1/user/domains/:id/config | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/domains | Bearer(user) | - | keyword,page,pageSize | - | CommonResponse |

| POST | /api/v1/user/domains | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/forward/ranking | Bearer(user) | - | range | - | CommonResponse |

| GET | /api/v1/user/forward/traffic | Bearer(user) | - | keyword,range | - | CommonResponse |

| DELETE | /api/v1/user/forward_defaults | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/forward_defaults | Bearer(user) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/user/forward_defaults | Bearer(user) | - | - | - | CommonResponse |

| DELETE | /api/v1/user/forward_groups | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/forward_groups | Bearer(user) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/user/forward_groups | Bearer(user) | - | - | - | CommonResponse |

| PUT | /api/v1/user/forward_groups | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/forwards/:id | Bearer(user) | id | - | - | CommonResponse |

| POST | /api/v1/user/forwards/batch | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/forwards/batch_action | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/forwards/batch_update | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/forwards | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/forwards | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/login/captcha | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/login | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/logs/access | Bearer(user) | - | cache_status,client_ip,domain,domain_mode,end_time,keyword,method,node_id,node_ip,page,pageSize,port,referer,scheme,ssl_cipher,ssl_protocol,start_time,status,status_max,status_min,uri,uri_mode,user_agent | - | CommonResponse |

| GET | /api/v1/user/logs/block/current | Bearer(user) | - | keyword,type | - | CommonResponse |

| GET | /api/v1/user/logs/block/history | Bearer(user) | - | end_time,keyword,page,pageSize,range,start_time,time_range,type | - | CommonResponse |

| GET | /api/v1/user/logs/block/stats | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/logs/operation | Bearer(user) | - | keyword,page,pageSize | - | CommonResponse |

| GET | /api/v1/user/message_sub | Bearer(user) | - | - | - | CommonResponse |

| PUT | /api/v1/user/message_sub | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/messages/:id/read | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/messages/unread | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/messages | Bearer(user) | - | keyword,page,pageSize,type | - | CommonResponse |

| GET | /api/v1/user/orders | Bearer(user) | - | keyword,page,pageSize,type | - | CommonResponse |

| PUT | /api/v1/user/password | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/plans | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/profile | Bearer(user) | - | - | - | CommonResponse |

| PUT | /api/v1/user/profile | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/recharge | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/register | Bearer(user) | - | - | - | CommonResponse |

| DELETE | /api/v1/user/rules/acl/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/rules/acl/:id | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/rules/acl/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/rules/acl | Bearer(user) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/user/rules/acl | Bearer(user) | - | - | - | CommonResponse |

| DELETE | /api/v1/user/rules/cc/filters/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/rules/cc/filters/:id | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/rules/cc/filters/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/rules/cc/filters | Bearer(user) | - | name,status | - | CommonResponse |

| POST | /api/v1/user/rules/cc/filters | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/rules/cc/groups/:id | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/rules/cc/groups/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/rules/cc/groups | Bearer(user) | - | name,status | - | CommonResponse |

| POST | /api/v1/user/rules/cc/groups | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/rules/cc/matchers/:id | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/rules/cc/matchers/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/rules/cc/matchers | Bearer(user) | - | name,status | - | CommonResponse |

| POST | /api/v1/user/rules/cc/matchers | Bearer(user) | - | - | - | CommonResponse |

| DELETE | /api/v1/user/site_defaults/:name | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/site_defaults/:name | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/site_defaults | Bearer(user) | - | scope_name,scope_id,user_id | - | CommonResponse |

| POST | /api/v1/user/site_defaults | Bearer(user) | - | - | - | CommonResponse |

| DELETE | /api/v1/user/site_groups/:id | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/site_groups/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/site_groups | Bearer(user) | - | keyword,page,pageSize,user_id | - | CommonResponse |

| POST | /api/v1/user/site_groups | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/sites/:id | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/sites/:id | Bearer(user) | id | - | - | CommonResponse |

| POST | /api/v1/user/sites/apply_cert | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/sites/batch/:id/progress | Bearer(user) | id | - | - | CommonResponse |

| POST | /api/v1/user/sites/batch | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/sites/batch_action | Bearer(user) | - | - | - | CommonResponse |

| POST | /api/v1/user/sites/batch_update | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/sites/export | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/sites/resolve | Bearer(user) | - | domain | - | CommonResponse |

| GET | /api/v1/user/sites | Bearer(user) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/user/sites | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/stats/basic | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/stats/origin | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/stats/quality | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/stats/ranking | Bearer(user) | - | keyword,type | - | CommonResponse |

| POST | /api/v1/user/tasks/:id/resubmit | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/tasks/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/tasks/usage | Bearer(user) | - | user_id | - | CommonResponse |

| GET | /api/v1/user/tasks | Bearer(user) | - | keyword,page,pageSize,type,user_id | - | CommonResponse |

| POST | /api/v1/user/tasks | Bearer(user) | - | - | - | CommonResponse |

| GET | /api/v1/user/usage | Bearer(user) | - | range | - | CommonResponse |

| POST | /api/v1/user/user_packages/:id/renew | Bearer(user) | id | - | - | CommonResponse |

| POST | /api/v1/user/user_packages/:id/switch | Bearer(user) | id | - | - | CommonResponse |

| PUT | /api/v1/user/user_packages/:id | Bearer(user) | id | - | - | CommonResponse |

| GET | /api/v1/user/user_packages | Bearer(user) | - | user_id | - | CommonResponse |

| GET | /health | None | - | - | - | CommonResponse |

| GET | /health | None | - | - | - | CommonResponse |

| GET | /ws/agent | None | - | - | - | CommonResponse |



### 30.3 Schema 字段表（从源码自动抽取）



> 规则：字json tag '-' 表示不输出；json tag omitempty Go 指针类型T）视为可选
> Optional 列：Y=可选，N=必填


#### Model.ACL
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐
- `work_dir` 仅作字段保留，运行时固定为可执行文件所在目录，不接受外部输入覆盖




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | uid | N |

| Name | string | name | N |

| Description | string | des | N |

| DefaultAction | string | default_action | N |

| Data | string | data | N |

| Enable | bool | enable | N |

| TaskID | int64 | task_id | N |

| Version | int | version | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.APIKey
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| APIKey | string | api_key | N |

| APISecret | string | api_secret | N |

| APIIP | string | api_ip | N |



#### Model.AccessControl
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| BlackIP | []string | black_ip | N |

| WhiteIP | []string | white_ip | N |

| BlackUA | []string | black_ua | N |

| WhiteUA | []string | white_ua | N |

| BlackURL | []string | black_url | N |

| WhiteURL | []string | white_url | N |

| RegionBlock | []string | region_block | N |

| BlockEmptyUA | bool | block_empty_ua | N |



#### Model.AgentCnameInfo
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Domain | string | domain | N |

| Hostname | string | hostname | N |

| Hostname2 | string | hostname2 | N |

| Mode | string | mode | N |

| RecordID | string | record_id | N |



#### Model.AgentFeatures
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| HTTPPort | int | http_port | N |

| StreamPort | int | stream_port | N |

| Websocket | bool | websocket | N |

| CustomCCRule | bool | custom_cc_rule | N |

| L2Origin | bool | l2_origin | N |



#### Model.AgentLimits
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Traffic | int32 | traffic | N |

| Bandwidth | string | bandwidth | N |

| Connection | int32 | connection | N |

| Domain | int32 | domain | N |



#### Model.AgentPackageConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| PackageID | int32 | package_id | N |

| UID | int32 | uid | N |

| Version | int | version | N |

| Status | string | status | N |

| RegionID | int32 | region_id | N |

| NodeGroupID | int32 | node_group_id | N |

| BackupNodeGroup | int32 | backup_node_group | N |

| EnableBackup | int | enable_backup_group | N |

| Cname | AgentCnameInfo | cname | N |

| Limits | AgentLimits | limits | N |

| Features | AgentFeatures | features | N |

| Time | AgentTime | time | N |



#### Model.AgentTime
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| StartAt | string | start_at | N |

| EndAt | string | end_at | N |



#### Model.CCConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Enable | bool | enable | N |

| Threshold | int | threshold | N |

| Action | string | action | N |

| BlockTimeout | int | block_timeout | N |

| EmergencyMode | bool | emergency_mode | N |

| SlideCount | int | slide_count | N |



#### Model.CCFilter
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | uid | N |

| Name | string | name | N |

| Description | string | des | N |

| Type | string | type | N |

| WithinSecond | int | within_second | N |

| MaxReq | int | max_req | N |

| MaxReqPerUri | int | max_req_per_uri | N |

| Extra | string | extra | N |

| Internal | bool | internal | N |

| Enable | bool | enable | N |

| TaskID | int64 | task_id | N |

| Version | int | version | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.CCMatch
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | uid | N |

| Name | string | name | N |

| Description | string | des | N |

| Data | string | data | N |

| Internal | bool | internal | N |

| Enable | bool | enable | N |

| TaskID | int64 | task_id | N |

| Version | int | version | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.CCRule
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Sort | int | sort | N |

| UserID | int64 | uid | N |

| Name | string | name | N |

| Description | string | des | N |

| Data | string | data | N |

| Internal | bool | internal | N |

| Enable | bool | enable | N |

| IsShow | bool | is_show | N |

| TaskID | int64 | task_id | N |

| Version | int | version | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.Captcha
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Email | string | email | N |

| Phone | string | phone | N |

| Code | string | captcha | N |

| IP | string | ip | N |

| CreatedAt | time.Time | create_at | N |



#### Model.Cert
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int | id | N |

| UserID | int | uid | N |

| Name | string | name | N |

| Description | string | des | N |

| Type | string | type | N |

| Domain | string | domain | N |

| DNSAPI | *int | dnsapi | Y |

| Cert | string | cert | N |

| Key | string | key | N |

| StartTime | *time.Time | start_time | Y |

| ExpireTime | *time.Time | expire_time | Y |

| AutoRenew | bool | auto_renew | N |

| CreateAt | time.Time | create_at | N |

| UpdateAt | time.Time | update_at | N |

| Enable | bool | enable | N |

| TaskID | int64 | task_id | N |

| IssueTaskID | int64 | issue_task_id | N |

| State | string | state | N |

| Ret | string | ret | N |

| Version | int | version | N |



#### Model.CnameDomain
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Domain | string | domain | N |

| DNSProviderID | int64 | dns_provider_id | N |

| Note | string | note | N |

| CreatedAt | time.Time | created_at | N |

| UpdatedAt | time.Time | updated_at | N |



#### Model.ConfigItem
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Name | string | name | N |

| Value | string | value | N |

| Type | string | type | N |

| ScopeID | int64 | scope_id | N |

| ScopeName | string | scope_name | N |

| Enable | bool | enable | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.DNSAPI
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | uid | N |

| Name | string | name | N |

| Remark | string | remark | N |

| Type | string | type | N |

| Auth | string | auth | N |



#### Model.DNSProvider
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Name | string | name | N |

| Type | string | type | N |

| Credentials | string | credentials | N |

| CreatedAt | time.Time | created_at | N |



#### Model.DefaultSiteConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Website | SiteTemplate | website | N |

| API | SiteTemplate | api | N |

| Download | SiteTemplate | download | N |



#### Model.Domain
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| Name | string | name | N |

| Cname | string | cname | N |

| Status | int | status | N |

| Origins | []DomainOrigin | origins | N |

| CreatedAt | time.Time | created_at | N |

| UpdatedAt | time.Time | updated_at | N |



#### Model.DomainOrigin
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| DomainID | int64 | domain_id | N |

| Addr | string | addr | N |

| Port | int | port | N |

| Weight | int | weight | N |

| Protocol | string | protocol | N |

| CreatedAt | time.Time | created_at | N |

| UpdatedAt | time.Time | updated_at | N |



#### Model.EdgeACLRule
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| IP | string | ip | N |

| Action | string | action | N |



#### Model.EdgeCCFilter
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Type | string | type | N |

| WithinSecond | int | within_second | N |

| MaxReq | int | max_req | N |

| MaxReqPerURI | int | max_req_per_uri | N |

| Extra | string | extra,omitempty | Y |



#### Model.EdgeCCMatcher
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Data | string | data | N |



#### Model.EdgeCCRuleItem
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| MatcherID | int64 | matcher_id,omitempty | Y |

| FilterID | int64 | filter_id,omitempty | Y |

| Action | string | action,omitempty | Y |

| Enabled | bool | enabled | N |



#### Model.EdgeCacheConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Enable | bool | enable | N |

| DefaultTTL | int | default_ttl,omitempty | Y |

| Rules | []EdgeCacheRule | rules,omitempty | Y |



#### Model.EdgeCacheRule
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Rule | string | rule,omitempty | Y |

| Ext | string | ext,omitempty | Y |

| URI | string | uri,omitempty | Y |

| Prefix | string | prefix,omitempty | Y |

| TTL | int | ttl,omitempty | Y |

| Enable | *bool | enable,omitempty | Y |

| NoCache | bool | no_cache,omitempty | Y |

| ForceCache | bool | force_cache,omitempty | Y |

| Priority | int | priority,omitempty | Y |

| IgnoreArgs | bool | ignore_args,omitempty | Y |

| CacheKey | string | cache_key,omitempty | Y |



#### Model.EdgeConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Version | int64 | version | N |

| NodeID | string | node_id,omitempty | Y |

| NodeLevel | int | node_level,omitempty | Y |

| Domains | []EdgeDomain | domains | N |

| Upstreams | []EdgeUpstream | upstreams | N |

| WAF | *WAFConfig | waf,omitempty | Y |

| Resources | *GlobalResourceConfig | resources,omitempty | Y |

| ErrorPages | map[string]string | error_pages,omitempty | Y |

| DefaultConfig | *DefaultSiteConfig | default_config,omitempty | Y |

| CCRules | map[int64][]EdgeCCRuleItem | cc_rules,omitempty | Y |

| CCMatchers | map[int64]EdgeCCMatcher | cc_matchers,omitempty | Y |

| CCFilters | map[int64]EdgeCCFilter | cc_filters,omitempty | Y |

| Streams | []EdgeStream | streams,omitempty | Y |

| Nginx | *EdgeNginxConfig | nginx,omitempty | Y |

| FallbackCertData | string | fallback_cert_data,omitempty | Y |

| FallbackKeyData | string | fallback_key_data,omitempty | Y |



#### Model.EdgeCookieConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Enable | bool | enable | N |

| Domain | string | domain,omitempty | Y |



#### Model.EdgeCorsConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Enable | bool | enable | N |

| AllowOrigin | string | allow_origin,omitempty | Y |

| AllowMethods | string | allow_methods,omitempty | Y |

| AllowHeaders | string | allow_headers,omitempty | Y |

| ExposeHeaders | string | expose_headers,omitempty | Y |

| AllowCredentials | bool | allow_credentials,omitempty | Y |

| MaxAge | string | max_age,omitempty | Y |



#### Model.EdgeDomain
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Name | string | name | N |

| UpstreamKey | string | upstream_key | N |

| L2UpstreamKey | string | l2_upstream_key,omitempty | Y |

| UseL2 | bool | use_l2,omitempty | Y |

| L2HTTPPort | string | l2_http_port,omitempty | Y |

| L2HTTPSPort | string | l2_https_port,omitempty | Y |

| LoadBalancePolicy | string | load_balance_policy,omitempty | Y |

| Headers | map[string]string | headers,omitempty | Y |

| ResponseHeaders | map[string]string | response_headers,omitempty | Y |

| Hotlink | *EdgeHotlinkConfig | hotlink,omitempty | Y |

| CORS | *EdgeCorsConfig | cors,omitempty | Y |

| Cookie | *EdgeCookieConfig | cookie,omitempty | Y |

| BlockTransparentProxy | bool | block_transparent_proxy,omitempty | Y |

| CrawlerAction | string | crawler_action,omitempty | Y |

| GuardPassTTL | int | guard_pass_ttl,omitempty | Y |

| GuardBlockTTL | int | guard_block_ttl,omitempty | Y |

| URLRedirects | []map[string]interface{} | url_redirects,omitempty | Y |

| OriginConditions | []map[string]interface{} | origin_conditions,omitempty | Y |

| Status | string | status,omitempty | Y |

| ConnLimit | int | conn_limit,omitempty | Y |

| SSLCertData | string | ssl_cert_data,omitempty | Y |

| SSLKeyData | string | ssl_key_data,omitempty | Y |

| SSLCertPath | string | ssl_cert_path,omitempty | Y |

| SSLKeyPath | string | ssl_key_path,omitempty | Y |

| ACLDefaultAction | string | acl_default_action,omitempty | Y |

| ACLRules | []EdgeACLRule | acl_rules,omitempty | Y |

| BlackIPs | []string | black_ips,omitempty | Y |

| WhiteIPs | []string | white_ips,omitempty | Y |

| RegionBlock | []string | region_block,omitempty | Y |

| CCRuleID | int64 | cc_rule_id,omitempty | Y |

| OriginProtocol | string | origin_protocol,omitempty | Y |

| OriginHTTPPort | string | origin_http_port,omitempty | Y |

| OriginHTTPSPort | string | origin_https_port,omitempty | Y |

| Cache | *EdgeCacheConfig | cache,omitempty | Y |

| HttpListen | []string | http_listen,omitempty | Y |

| HttpsListen | []string | https_listen,omitempty | Y |

| HTTPSForce | bool | https_force,omitempty | Y |

| HTTPSRedirectPort | string | https_redirect_port,omitempty | Y |

| HTTPSHSTS | bool | https_hsts,omitempty | Y |

| HTTPSHTTP2 | bool | https_http2,omitempty | Y |

| HTTPSOCSP | bool | https_ocsp,omitempty | Y |

| HTTPSHTTP3 | bool | https_http3,omitempty | Y |

| HTTPSSSLProtocols | string | https_ssl_protocols,omitempty | Y |

| HTTPSSSLCiphers | string | https_ssl_ciphers,omitempty | Y |

| HTTPSSSLPreferServerCiphers | string | https_ssl_prefer_server_ciphers,omitempty | Y |

| ProxyConnectTimeout | string | proxy_connect_timeout,omitempty | Y |

| ProxyReadTimeout | string | proxy_read_timeout,omitempty | Y |

| ProxySendTimeout | string | proxy_send_timeout,omitempty | Y |

| ProxyHTTPVersion | string | proxy_http_version,omitempty | Y |

| ProxySSLProtocols | string | proxy_ssl_protocols,omitempty | Y |

| EnableGzip | bool | enable_gzip,omitempty | Y |

| GzipTypes | string | gzip_types,omitempty | Y |

| EnableWebsocket | bool | enable_websocket,omitempty | Y |

| EnableRange | bool | enable_range,omitempty | Y |

| BodyLimit | int64 | body_limit,omitempty | Y |

| LimitRate | int64 | limit_rate,omitempty | Y |

| UpstreamKeepalive | bool | upstream_keepalive,omitempty | Y |

| UpstreamKeepaliveConn | int | upstream_keepalive_conn,omitempty | Y |

| UpstreamKeepaliveTimeout | int | upstream_keepalive_timeout,omitempty | Y |



#### Model.EdgeHotlinkConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Enable | bool | enable | N |

| Scope | string | scope,omitempty | Y |

| Value | string | value,omitempty | Y |

| AllowEmpty | bool | allow_empty,omitempty | Y |

| Domains | []string | domains,omitempty | Y |



#### Model.EdgeNginxConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| LogsDir | string | logs_dir,omitempty | Y |

| WorkerProcesses | string | worker_processes,omitempty | Y |

| WorkerConnections | int | worker_connections,omitempty | Y |

| WorkerRlimitNofile | int | worker_rlimit_nofile,omitempty | Y |

| WorkerShutdownTimeout | string | worker_shutdown_timeout,omitempty | Y |

| Resolver | string | resolver,omitempty | Y |

| ResolverTimeout | string | resolver_timeout,omitempty | Y |

| HTTP | map[string]interface{} | http,omitempty | Y |

| Stream | map[string]interface{} | stream,omitempty | Y |



#### Model.EdgeStream
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| ListenPorts | []string | listen_ports | N |

| Targets | []EdgeStreamTarget | targets | N |

| UseListenPort | bool | use_listen_port,omitempty | Y |

| BalanceWay | string | balance_way,omitempty | Y |

| ProxyProtocol | bool | proxy_protocol,omitempty | Y |

| ProxyConnectTimeout | string | proxy_connect_timeout,omitempty | Y |

| ProxyTimeout | string | proxy_timeout,omitempty | Y |

| ConnLimit | int | conn_limit,omitempty | Y |



#### Model.EdgeStreamTarget
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Addr | string | addr | N |

| Weight | int | weight | N |

| Enable | bool | enable | N |

| NodeID | int64 | node_id,omitempty | Y |

| Backup | bool | backup,omitempty | Y |



#### Model.EdgeUpstream
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | string | id | N |

| Targets | []EdgeUpstreamTarget | targets | N |



#### Model.EdgeUpstreamTarget
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Addr | string | addr | N |

| Weight | int | weight | N |

| NodeID | int64 | node_id,omitempty | Y |



#### Model.Forward
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | uid | N |

| UserPackageID | int64 | user_package_id | N |

| RegionID | int64 | region_id | N |

| NodeGroupID | int64 | node_group_id | N |

| BackupNodeGroup | int64 | backup_node_group | N |

| EnableBackupGroup | bool | enable_backup_group | N |

| Enable | bool | enable | N |

| State | string | state | N |

| Remark | string | remark | N |

| CnameDomain | string | cname_domain | N |

| CnameHostname2 | string | cname_hostname2 | N |

| CnameMode | string | cname_mode | N |

| Cname | string | cname | N |

| ListenPorts | []string | listen_ports | N |

| Origins | []ForwardOrigin | origins | N |

| BackendPort | string | backend_port | N |

| BalanceWay | string | balance_way | N |

| ProxyProtocol | bool | proxy_protocol | N |

| ConnLimit | string | conn_limit | N |

| Settings | map[string]interface{} | settings | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.ForwardGroup
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | uid | N |

| Name | string | name | N |

| Remark | string | remark | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.ForwardGroupRelation
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| ForwardID | int64 | forward_id | N |

| GroupID | int64 | group_id | N |



#### Model.ForwardOrigin
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Address | string | address | N |

| Weight | int | weight | N |

| Enable | bool | enable | N |



#### Model.ForwardResourceConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| DisabledPorts | string | disabled_ports | N |

| MinLimit | int | min_limit | N |

| MaxLimitMultiplier | int | max_limit_multiplier | N |

| MaxACLRules | int | max_acl_rules | N |



#### Model.GlobalConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| WAF | WAFConfig | waf | N |

| Nginx | NginxConfig | nginx | N |

| DefaultConfig | DefaultSiteConfig | default_config | N |

| ErrorPages | map[string]string | error_pages | N |

| Resources | GlobalResourceConfig | resources | N |



#### Model.GlobalResourceConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Website | WebsiteResourceConfig | website | N |

| Forward | ForwardResourceConfig | forward | N |

| Public | PublicResourceConfig | public | N |



#### Model.IPSwitchLog
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| CreatedAt | time.Time | create_at | N |

| Type | string | type | N |

| NodeGroupID | int64 | node_group_id | N |

| NodeID | int64 | node_id | N |

| LineID | int64 | line_id | N |

| IP | string | ip | N |

| Action | string | action | N |

| EmailNeedSend | bool | email_need_send | N |

| EmailIsSent | bool | email_is_sent | N |

| EmailFailTimes | int | email_fail_times | N |

| EmailRet | string | email_ret | N |

| EmailTime | *time.Time | email_time | Y |

| EmailSendState | string | email_send_state | N |

| PhoneNeedSend | bool | phone_need_send | N |

| PhoneIsSent | bool | phone_is_sent | N |

| PhoneFailTimes | int | phone_fail_times | N |

| PhoneRet | string | phone_ret | N |

| PhoneTime | *time.Time | phone_time | Y |

| PhoneSendState | string | phone_send_state | N |

| Content | string | content | N |



#### Model.Line
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| NodeGroupID | int64 | node_group_id | N |

| NodeID | int64 | node_id | N |

| NodeIPID | int64 | node_ip_id | N |

| LineID | string | line_id | N |

| LineName | string | line_name | N |

| Weight | string | weight | N |

| RecordID | string | record_id | N |

| TaskID | *int64 | task_id | Y |

| Enable | bool | enable | N |

| IsBackup | bool | is_backup | N |

| EnableBackup | bool | enable_backup | N |

| IsBackupDefaultLine | bool | is_backup_default_line | N |

| EnableBackupDefaultLine | bool | enable_backup_default_line | N |

| SwitchAt | *time.Time | switch_at | Y |

| DisableBy | string | disable_by | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.LineDeleteQueue
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| NodeID | int64 | node_id | N |

| NodeGroupID | int64 | node_group_id | N |

| LineID | string | line_id | N |

| LineName | string | line_name | N |

| DeleteAt | time.Time | delete_at | N |

| CreatedAt | time.Time | create_at | N |



#### Model.Message
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Type | string | type | N |

| PubUser | int64 | pub_user | N |

| Receive | int64 | receive | N |

| Title | string | title | N |

| Content | string | content | N |

| PhoneContent | string | phone_content | N |

| EventID | string | event_id | N |

| UserPackageID | int64 | user_package_id | N |

| SiteID | int64 | site_id | N |

| IsShow | bool | is_show | N |

| IsRed | bool | is_red | N |

| IsBold | bool | is_bold | N |

| IsExternal | bool | is_external | N |

| IsPopup | bool | is_popup | N |

| EmailNeedSend | bool | email_need_send | N |

| PhoneNeedSend | bool | phone_need_send | N |

| EmailIsSent | bool | email_is_sent | N |

| PhoneIsSent | bool | phone_is_sent | N |

| URL | string | url | N |

| Sort | int | sort | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.MessageRead
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| UserID | int64 | user_id | N |

| MessageID | int64 | msg_id | N |

| CreatedAt | time.Time | create_at | N |



#### Model.MessageSub
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| UserID | int64 | user_id | N |

| MsgType | string | msg_type | N |

| Phone | bool | phone | N |

| Email | bool | email | N |



#### Model.NginxConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| WorkerProcesses | string | worker_processes | N |

| WorkerConnections | int | worker_connections | N |

| WorkerRlimitNofile | int | worker_rlimit_nofile | N |

| WorkerShutdownTimeout | string | worker_shutdown_timeout | N |

| LogDirectory | string | log_directory | N |

| KeepaliveTimeout | int | keepalive_timeout | N |

| Gzip | bool | gzip | N |

| CustomSnippet | string | custom_snippet | N |



#### Model.Node
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| PID | int64 | pid | N |

| GroupID | int64 | group_id | N |

| RegionID | *int64 | region_id | Y |

| Name | string | name | N |

| Remark | string | remark | N |

| IP | string | ip | N |

| Token | string | token | N |

| Host | string | host | N |

| Port | int | port | N |

| HttpProxy | string | http_proxy | N |

| IsMgmt | bool | is_mgmt | N |

| Enable | bool | enable | N |

| DisableBy | string | disable_by | N |

| ConfigTask | string | config_task | N |

| RegionName | string | region_name | N |

| CheckOn | bool | check_on | N |

| CheckProtocol | string | check_protocol | N |

| CheckTimeout | int | check_timeout | N |

| CheckPort | int | check_port | N |

| CheckHost | string | check_host | N |

| CheckPath | string | check_path | N |

| CheckNodeGroup | string | check_node_group | N |

| CheckAction | string | check_action | N |

| BwLimit | string | bw_limit | N |

| Online | bool | online | N |

| LineCount | int64 | line_count | N |

| Level | int | type | N |

| Sort | int | sort_order | N |

| CacheDir | string | cache_dir | N |

| MaxCacheSize | int | cache_limit | N |

| LogDir | string | log_dir | N |

| SSHHost | string | ssh_host | N |

| SSHPort | int | ssh_port | N |

| SSHUser | string | ssh_user | N |

| SSHAuthType | string | ssh_auth_type | N |

| WorkDir | string | work_dir | N |

| AutoInstall | bool | auto_install | N |

| InstallStatus | string | install_status | N |

| InstallError | string | install_error | N |

| InstallAt | *time.Time | install_at | Y |

| InstallStage | string | install_stage | N |

| InstallProgress | int | install_progress | N |

| InstallProgressBytes | int64 | install_progress_bytes | N |

| InstallProgressTotal | int64 | install_progress_total | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |

| SubIPs | []NodeSubIP | sub_ips,omitempty | Y |



#### Model.NodeGroup
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| RegionID | *int64 | region_id | Y |

| Name | string | name | N |

| CnameHostname | string | resolution | N |

| CnameDomain | string | cname_domain | N |

| Ipv4Resolution | string | ipv4_resolution | N |

| Description | string | remark | N |

| SortOrder | int | sort_order | N |

| L2Config | string | l2_config | N |

| BackupSwitchType | string | spare_ip_switch | N |

| BackupSwitchPolicy | string | backup_switch_policy | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.NodeMonitorLog
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| CreateAt | time.Time | create_at | N |

| Type | string | type | N |

| EventID | string | event_id | N |

| IP | string | ip | N |

| Success | string | success | N |

| NodeID | int64 | node_id | N |



#### Model.NodeSubIP
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| IP | string | ip | N |



#### Model.Order
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| Type | string | type | N |

| Description | string | des | N |

| Data | string | data | N |

| CreatedAt | time.Time | create_at | N |

| PaidAt | time.Time | pay_at | N |

| Amount | int64 | amount | N |

| PayType | string | pay_type | N |

| MerchantOrder | string | mch_order_no | N |

| TransactionID | string | transaction_id | N |

| State | string | state | N |



#### Model.Origin
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Addr | string | addr | N |

| Port | int | port | N |

| Weight | int | weight | N |

| Protocol | string | protocol | N |



#### Model.Package
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Name | string | name | N |

| Description | string | des | N |

| RegionID | int64 | region_id | N |

| NodeGroupID | int64 | node_group_id | N |

| BackupNode | int64 | backup_node_group | N |

| CnameDomain | string | cname_domain | N |

| CnameHost2 | string | cname_hostname2 | N |

| CnameMode | string | cname_mode | N |

| MonthPrice | int64 | month_price | N |

| QuarterPrice | int64 | quarter_price | N |

| YearPrice | int64 | year_price | N |

| Traffic | int64 | traffic | N |

| Bandwidth | string | bandwidth | N |

| Connection | int64 | connection | N |

| DomainLimit | int64 | domain | N |

| HttpPort | int64 | http_port | N |

| StreamPort | int64 | stream_port | N |

| ExpireAt | *time.Time | expire | Y |

| BuyNumLimit | int64 | buy_num_limit | N |

| BackendIPLimit | string | backend_ip_limit | N |

| IDVerify | bool | id_verify | N |

| BeforeExpDaysRenew | int64 | before_exp_days_renew | N |

| Websocket | bool | websocket | N |

| CustomCCRule | bool | custom_cc_rule | N |

| L2Origin | bool | l2_origin | N |

| Sort | int | sort | N |

| Owner | string | owner | N |

| Enable | bool | enable | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.Plan
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | uint | id | N |

| Name | string | name | N |

| Desc | string | desc | N |

| Group | string | group | N |

| Region | string | region | N |

| LineGroup | string | line_group | N |

| BackupGroup | string | backup_group | N |

| TrafficLimit | string | traffic_limit | N |

| BandwidthLimit | string | bandwidth_limit | N |

| ConnectionLimit | string | connection_limit | N |

| L4PortLimit | string | l4_port_limit | N |

| DomainLimit | string | domain_limit | N |

| MainDomainLimit | string | main_domain_limit | N |

| NonStandardPortCount | string | non_standard_port_count | N |

| CustomCCRules | bool | custom_cc_rules | N |

| Websocket | bool | websocket | N |

| HTTP3 | bool | http3 | N |

| L2Origin | bool | l2_origin | N |

| CCProtection | string | cc_protection | N |

| DDOSProtection | string | ddos_protection | N |

| PriceMonthly | float64 | price_monthly | N |

| PriceQuarterly | float64 | price_quarterly | N |

| PriceYearly | float64 | price_yearly | N |

| CNAMEHostname | string | cname_hostname | N |

| CNAMEDomain | string | cname_domain | N |

| CNAMEMode | string | cname_mode | N |

| SingleUserLimit | int | single_user_limit | N |

| Validity | string | validity | N |

| RealNameAuth | bool | real_name_auth | N |

| RenewalDelay | int | renewal_delay | N |

| AssignedUser | string | assigned_user | N |

| SortOrder | int | sort_order | N |

| Status | bool | status | N |

| SourceIPLimit | string | source_ip_limit | N |

| CreatedAt | time.Time | created_at | N |

| UpdatedAt | time.Time | updated_at | N |



#### Model.PublicResourceConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| DisabledCustomPorts | string | disabled_custom_ports | N |

| AllowedCustomPorts | string | allowed_custom_ports | N |



#### Model.Region
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Name | string | name | N |

| Desc | string | des | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.ResourceRule
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Duration | int | duration | N |

| MaxRequests | int | max_requests | N |



#### Model.Site
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | uid | N |

| UserPackageID | int64 | user_package_id | N |

| RegionID | int64 | region_id | N |

| NodeGroupID | int64 | node_group_id | N |

| BackupNodeGroupID | int64 | backup_node_group_id | N |

| EnableBackupGroup | bool | enable_backup_group | N |

| DNSProviderID | int64 | dns_provider_id | N |

| PlatformDNSRecordID | string | platform_dns_record_id | N |

| UserDNSRecordID | string | user_dns_record_id | N |

| CnameDomain | string | cname_domain | N |

| CnameHostname | string | cname_hostname | N |

| CnameHostname2 | string | cname_hostname_2 | N |

| CnameMode | string | cname_mode | N |

| Domains | []string | domains | N |

| HttpListen | []string | http_listen | N |

| HttpsListen | []string | https_listen | N |

| CertID | int64 | cert_id | N |

| BackendProtocol | string | backend_protocol | N |

| BalanceWay | string | balance_way | N |

| Backends | []string | backends | N |

| CcDefaultRule | int64 | cc_default_rule | N |

| Settings | map[string]interface{} | settings | N |

| State | string | state | N |

| Enable | bool | enable | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |

| GroupID | int64 | group_id | N |

| GroupIDs | []int64 | group_ids | N |



#### Model.SiteConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| Domain | string | domain | N |

| Origins | []Origin | origins | N |

| SSLEnable | bool | ssl_enable | N |

| CertID | int64 | cert_id | N |



#### Model.SiteGroup
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | uid | N |

| Name | string | name | N |

| Remark | string | remark | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### Model.SiteGroupRelation
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| SiteID | int64 | site_id | N |

| GroupID | int64 | group_id | N |



#### Model.SiteTemplate
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| CacheEnable | bool | cache_enable | N |

| CacheTTL | int | cache_ttl | N |

| Gzip | bool | gzip | N |

| WAFEnable | bool | waf_enable | N |

| SSLCiphers | string | ssl_ciphers | N |



#### Model.SyntacticWAF
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| SQLInjection | bool | sql_injection | N |

| XSS | bool | xss | N |

| Scanner | bool | scanner | N |



#### Model.SysConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Name | string | name | N |

| Value | string | value | N |

| Type | string | type | N |

| ScopeID | int | scope_id | N |

| ScopeName | string | scope_name | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |

| Enable | bool | enable | N |

| TaskID | *int64 | task_id | Y |



#### Model.Task
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| PID | int64 | pid | N |

| Pry | int | pry | N |

| Name | string | name | N |

| Type | string | type | N |

| Res | string | res | N |

| Data | string | data | N |

| TargetsJSON | string | targets_json | N |

| Depend | string | depend | N |

| CreateAt | time.Time | create_at | N |

| StartAt | *time.Time | start_at | Y |

| EndAt | *time.Time | end_at | Y |

| Ret | string | ret | N |

| Enable | bool | enable | N |

| State | string | state | N |

| ErrTimes | int | err_times | N |

| RetryAt | *time.Time | retry_at | Y |

| Progress | string | progress | N |



#### Model.User
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Email | string | email | N |

| Name | string | name | N |

| Description | string | des | N |

| Phone | string | phone | N |

| QQ | string | qq | N |

| CertID | string | cert_id | N |

| CertName | string | cert_name | N |

| CertNo | string | cert_no | N |

| CertVerified | bool | cert_verified | N |

| WhiteIP | string | white_ip | N |

| LoginCaptcha | string | login_captcha | N |

| Balance | int64 | balance | N |

| Freeze | int64 | freeze | N |

| Enable | bool | enable | N |

| Type | int | type | N |

| CreatedAt | time.Time | create_at | N |



#### Model.UserLoginLog
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| IP | string | ip | N |

| Success | bool | success | N |

| PostContent | string | post_content | N |

| CreatedAt | time.Time | created_at | N |



#### Model.UserOperationLog
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| Type | string | type | N |

| Action | string | action | N |

| Content | string | content | N |

| Diff | string | diff | N |

| IP | string | ip | N |

| Process | string | process | N |

| CreatedAt | time.Time | created_at | N |



#### Model.UserPackage
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int32 | uid | N |

| Name | string | name | N |

| PackageID | int32 | package_id | N |

| RegionID | int64 | region_id | N |

| NodeGroupID | int64 | node_group_id | N |

| BackupNodeGroup | int64 | backup_node_group | N |

| EnableBackup | bool | enable_backup_group | N |

| CnameDomain | string | cname_domain | N |

| CnameHostname2 | string | cname_hostname2 | N |

| CnameHostname | string | cname_hostname | N |

| CnameMode | string | cname_mode | N |

| RecordID | string | record_id | N |

| Traffic | int32 | traffic | N |

| Bandwidth | string | bandwidth | N |

| Connection | int32 | connection | N |

| DomainLimit | int32 | domain | N |

| MainDomainLimit | int32 | main_domain_limit | N |

| HTTPPortLimit | int32 | http_port | N |

| StreamPortLimit | int32 | stream_port | N |

| CustomCCRule | bool | custom_cc_rule | N |

| Websocket | bool | websocket | N |

| L2Origin | bool | l2_origin | N |

| MonthPrice | int64 | month_price | N |

| QuarterPrice | int64 | quarter_price | N |

| YearPrice | int64 | year_price | N |

| StartAt | time.Time | start_at | N |

| EndAt | time.Time | end_at | N |

| CreatedAt | time.Time | create_at | N |

| TaskID | *int64 | task_id | Y |

| Version | int | version | N |

| IsExpired | bool | is_expired | N |



#### Model.UserPlan
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | uint | id | N |

| UserID | uint | user_id | N |

| PlanID | uint | plan_id | N |

| PlanName | string | plan_name | N |

| ExpireAt | time.Time | expire_at | N |

| Status | string | status | N |

| CreatedAt | time.Time | created_at | N |

| UpdatedAt | time.Time | updated_at | N |



#### Model.WAFConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Enable | bool | enable | N |

| DefaultBlockAction | string | default_block_action | N |

| AutoIPSetEnable | bool | auto_ipset_enable | N |

| AutoIPSetThreshold | int | auto_ipset_threshold | N |

| BlockPageRateLimitEnable | bool | block_page_rate_limit_enable | N |

| BlockPageRateLimit | int | block_page_rate_limit | N |

| BlockPageTrafficFree | bool | block_page_traffic_free | N |

| BlacklistTimeout | int | blacklist_timeout | N |

| TempWhitelistTimeout | int | temp_whitelist_timeout | N |

| TempWhitelistLimitTotal | int | temp_whitelist_limit_total | N |

| TempWhitelistLimitURL | int | temp_whitelist_limit_url | N |

| WhitelistIPs | string | whitelist_ips | N |

| BlacklistIPs | string | blacklist_ips | N |

| PreventTLSHandshake | bool | prevent_tls_handshake | N |

| BlockUnboundDomain | bool | block_unbound_domain | N |

| DisablePing | bool | disable_ping | N |

| DefaultPageProtection | string | default_page_protection | N |

| DefaultPageProtectionThreshold | int | default_page_protection_threshold | N |

| SecretKey | string | secret_key | N |

| NodeLogCleanStrategy | string | node_log_clean_strategy | N |

| CCRuleAutoSwitch | bool | cc_rule_auto_switch | N |

| AntiCCImageSource | string | anti_cc_image_source | N |

| AntiCCImageCustomURL | string | anti_cc_image_custom_url | N |

| AntiCCType | string | anti_cc_type | N |

| AntiCCDebug | bool | anti_cc_debug | N |

| WellKnownProtectionThreshold | int | well_known_protection_threshold | N |

| ResourceProtectionEnable | bool | resource_protection_enable | N |

| ResourceProtectionThreshold | int | resource_protection_threshold | N |

| ResourceProtectionBlockTimeout | int | resource_protection_block_timeout | N |

| ResourceProtectionRules | []ResourceRule | resource_protection_rules | N |

| Mode | string | mode | N |

| Policy | string | policy | N |

| CC | CCConfig | cc | N |

| AccessControl | AccessControl | access_control | N |

| Syntactic | SyntacticWAF | syntactic | N |



#### Model.WebsiteResourceConfig
**现有行为(Go 代码)**
- 字段定义即为现有行为，JSON tag/Optional 规则按表格执行

**C# 重构要求**
- SqlSugar 实体字段、JSON 名与可空性必须与表格一致
- 不新增字段，保持与 db.sql 对齐




| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| MinLimit | int | min_limit | N |

| MaxLimitMultiplier | int | max_limit_multiplier | N |

| MaxBlacklistIPs | int | max_blacklist_ips | N |

| MaxWhitelistIPs | int | max_whitelist_ips | N |

| DailyURLPurgeLimit | int | daily_url_purge_limit | N |

| DailyDirPurgeLimit | int | daily_dir_purge_limit | N |

| DailyPreloadLimit | int | daily_preload_limit | N |

| DailyUnlockIPLimit | int | daily_unlock_ip_limit | N |

| UnlockIPBatchLimit | int | unlock_ip_batch_limit | N |

| MaxCCRulesPerGroup | int | max_cc_rules_per_group | N |

| MaxACLRules | int | max_acl_rules | N |

| DailyLogDownloadLimit | int | daily_log_download_limit | N |

| LogStorageDir | string | log_storage_dir | N |

| LogStorageHours | int | log_storage_hours | N |

| MaxDomainsPerSite | int | max_domains_per_site | N |

| DefaultListen80 | bool | default_listen_80 | N |



#### DTO.ACLCondition
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义
- `work_dir` 输入忽略，服务端固定为可执行文件所在目录


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Item | string | item | N |

| Operator | string | operator | N |

| Value | string | value | N |



#### DTO.ACLData
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Rules | []ACLRule | rules | N |

| DefaultDenyStatus | int | default_deny_status | N |

| DefaultRedirectURL | string | default_redirect_url | N |



#### DTO.ACLRule
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Conditions | []ACLCondition | conditions | N |

| Action | string | action | N |

| DenyStatus | int | deny_status | N |

| RedirectURL | string | redirect_url | N |



#### DTO.AccessLogRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Timestamp | time.Time | timestamp | N |

| NodeID | string | node_id | N |

| NodeIP | string | node_ip | N |

| RemoteAddr | string | remote_addr | N |

| Host | string | host | N |

| Method | string | method | N |

| URI | string | uri | N |

| Status | int | status | N |

| Bytes | uint64 | bytes | N |

| RequestTime | float64 | request_time | N |

| UpstreamAddr | string | upstream_addr | N |

| UpstreamResponseTime | float64 | upstream_response_time | N |

| UpstreamCacheStatus | string | upstream_cache_status | N |

| HTTPReferer | string | http_referer | N |

| HTTPUserAgent | string | http_user_agent | N |

| Scheme | string | scheme | N |

| SSLProtocol | string | ssl_protocol | N |

| SSLCipher | string | ssl_cipher | N |



#### DTO.AccessLogsMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| NodeID | string | node_id | N |

| NodeIP | string | node_ip | N |

| Lines | []string | lines | N |

| MsgID | string | msg_id,omitempty | Y |



#### DTO.AgentController
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ConfigSvc | *services.ConfigService | (no json tag) | Y |



#### DTO.AgentHelloMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| NodeID | string | node_id | N |

| Token | string | token | N |

| AgentVersion | string | agent_version | N |

| Capabilities | []string | capabilities | N |



#### DTO.AgentWSController
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| upgrader | websocket.Upgrader | (no json tag) | N |



#### DTO.BackupLogRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| CreatedAt | time.Time | created_at | N |

| FinishedAt | *time.Time | finished_at | Y |

| Status | int | status | N |

| Result | string | result | N |



#### DTO.CertDetail
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int | id | N |

| Uid | int | uid | N |

| Name | string | name | N |

| Description | string | des | N |

| Type | string | type | N |

| Domain | string | domain | N |

| DNSAPI | int | dnsapi | N |

| Cert | string | cert | N |

| Key | string | key | N |

| StartTime | *time.Time | start_time | Y |

| ExpireTime | *time.Time | expire_time | Y |

| AutoRenew | bool | auto_renew | N |

| CreateAt | time.Time | create_at | N |

| UpdateAt | time.Time | update_at | N |

| Enable | bool | enable | N |

| TaskID | int64 | task_id | N |

| State | string | state | N |

| Ret | string | ret | N |

| Version | int | version | N |

| UserName | string | user_name,omitempty | Y |

| IssueTaskRet | string | issue_task_ret,omitempty | Y |

| IssueTaskState | string | issue_task_state,omitempty | Y |

| IssueTaskRetryAt | *time.Time | issue_task_retry_at,omitempty | Y |

| IssueTaskErrTimes | int | issue_task_err_times,omitempty | Y |



#### DTO.CertIssuedMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| CertID | int64 | cert_id | N |

| CertPEM | string | cert | N |

| KeyPEM | string | key | N |

| IssueTaskID | int64 | issue_task_id | N |

| RateLimited | bool | rate_limited | N |

| RateCooldown | int | rate_cooldown | N |



#### DTO.EventsMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| NodeID | string | node_id | N |

| NodeIP | string | node_ip | N |

| Type | string | type | N |

| Payloads | []string | payloads | N |

| MsgID | string | msg_id,omitempty | Y |



#### DTO.FailItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Domain | string | domain | N |

| Reason | string | reason | N |



#### DTO.HeartbeatAckMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| SyncAction | string | sync_action | N |



#### DTO.HeartbeatMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| Timestamp | int64 | timestamp | N |

| Status | string | status | N |



#### DTO.L2HeartbeatMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| Nodes | []int64 | nodes | N |



#### DTO.L2NodesRequestMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| MsgID | string | msg_id | N |



#### DTO.L2NodesResponseMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| MsgID | string | msg_id | N |

| Nodes | []l2NodeInfo | nodes | N |



#### DTO.LoginCaptchaRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Username | string | username | N |

| Type | string | type | N |



#### DTO.LoginLogRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| Username | string | username | N |

| IP | string | ip | N |

| Success | bool | success | N |

| PostContent | string | post_content | N |

| CreatedAt | time.Time | created_at | N |



#### DTO.LoginRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Username | string | username | N |

| Password | string | password | N |

| Hash | string | password_hash | N |

| Captcha | string | captcha | N |

| Type | string | captcha_type | N |



#### DTO.MailLogRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | message_id | N |

| UserID | int64 | user_id | N |

| Title | string | subject | N |

| Medium | string | medium | N |

| Fails | int | fails | N |

| Status | int | status | N |

| Reason | string | reason | N |

| CreatedAt | time.Time | created_at | N |



#### DTO.MetricsMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| NodeID | string | node_id | N |

| NodeIP | string | node_ip | N |

| Content | string | content | N |

| MsgID | string | msg_id,omitempty | Y |



#### DTO.NodeController
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| NodeService | *services.NodeService | (no json tag) | Y |



#### DTO.NodeMonitorConfig
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| NotificationPeriod | string | notification_period | N |

| NotifyMethod | string | notify_method | N |

| NotifyMsgType | string | notify_msg_type | N |

| Email | string | email | N |

| Phone | string | phone | N |

| BwExceedTimes | int | bw_exceed_times | N |

| AutoSwitchEnable | bool | auto_switch_enable | N |

| AutoSwitchThreshold | int | auto_switch_threshold | N |

| AutoSwitchDuration | int | auto_switch_duration | N |

| AutoSwitchRecover | int | auto_switch_recover | N |

| AutoSwitchMinWeight | int | auto_switch_min_weight | N |

| MonitorAPI | string | monitor_api | N |

| Interval | int | interval | N |

| FailedTimes | int | failed_times | N |

| FailedRate | string | failed_rate | N |



#### DTO.NodeSyncMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| Action | string | action | N |

| Success | bool | success | N |



#### DTO.OpLogRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| Type | string | type | N |

| Action | string | action | N |

| Content | string | content | N |

| Diff | string | diff | N |

| IP | string | ip | N |

| Process | string | process | N |

| CreatedAt | time.Time | created_at | N |

| Username | string | username | N |

| Description | string | description | N |



#### DTO.RankItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Rank | int | rank | N |

| Item | string | item | N |

| RequestCount | int | request_count | N |

| OutTraffic | string | out_traffic | N |

| OriginTraffic | string | origin_traffic | N |



#### DTO.RegisterRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Username | string | username | N |

| Password | string | password | N |

| Hash | string | password_hash | N |

| Email | string | email | N |

| Phone | string | phone | N |



#### DTO.SystemInfoPayload
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| SysName | string | sys_name | N |

| UserConsoleTitle | string | user_console_title | N |

| AdminConsoleTitle | string | admin_console_title | N |

| FooterLink | string | footer_link | N |

| FooterCopyright | string | footer_copyright | N |

| FaviconFile | string | favicon_file | N |

| LogoFile | string | logo_file | N |

| LoginAdFile | string | login_ad_file | N |

| EnableEmailLogin | bool | enable_email_login | N |

| EnableSMSLogin | bool | enable_sms_login | N |

| AllowRegister | bool | allow_register | N |



#### DTO.TaskAckMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| MsgID | string | msg_id | N |

| NodeID | int64 | node_id,omitempty | Y |

| TaskID | int64 | task_id | N |

| TaskType | string | task_type,omitempty | Y |

| Status | string | status | N |

| Applied | json.RawMessage | applied | N |

| Error | string | error | N |

| Ret | string | ret,omitempty | Y |



#### DTO.TaskDispatchMsg
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |

| MsgID | string | msg_id | N |

| Task | TaskPayload | task | N |



#### DTO.TaskPayload
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| TaskID | int64 | task_id | N |

| TaskType | string | task_type | N |

| TaskName | string | task_name | N |

| Payload | string | payload,omitempty | Y |



#### DTO.WSDispatchRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| NodeID | int64 | node_id | N |

| TaskType | string | task_type | N |

| Payload | string | payload | N |

| WaitSeconds | int | wait_seconds | N |



#### DTO.WSDispatchResponse
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| NodeID | int64 | node_id | N |

| Connected | bool | connected | N |

| TaskID | int64 | task_id | N |

| State | string | state,omitempty | Y |

| Error | string | error,omitempty | Y |



#### DTO.WSMsgHeader
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Kind | string | kind | N |



#### DTO.aclPayload
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Name | string | name | N |

| Description | string | des | N |

| DefaultAction | string | default_action | N |

| Enable | bool | enable | N |

| Rules | []ACLRule | rules | N |

| UserID | int64 | user_id | N |

| DefaultDenyStatus | int | default_deny_status | N |

| DefaultRedirectURL | string | default_redirect_url | N |



#### DTO.acmeTokenRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Token | string | token | N |

| Value | string | value | N |

| TTL | int64 | ttl | N |



#### DTO.adminOrderRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| Amount | float64 | amount | N |

| Status | int | status | N |

| CreatedAt | string | created_at | N |

| PayType | string | pay_type | N |

| OrderNo | string | order_no | N |

| Type | string | type | N |

| Remark | string | remark | N |



#### DTO.announcementRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Title | string | title | N |

| Content | string | content | N |

| IsShow | bool | is_show | N |

| IsRed | bool | is_red | N |

| IsBold | bool | is_bold | N |

| CreatedAt | string | created_at | N |



#### DTO.applyCertSkipItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| SiteID | int64 | site_id | N |

| Domain | string | domain | N |

| Reason | string | reason | N |



#### DTO.backupTaskRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | (no json tag) | N |

| CreateAt | time.Time | (no json tag) | N |

| StartAt | *time.Time | (no json tag) | Y |

| EndAt | *time.Time | (no json tag) | Y |

| State | string | (no json tag) | N |

| Ret | string | (no json tag) | N |



#### DTO.batchSiteItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Domains | []string | (no json tag) | N |

| Backends | []string | (no json tag) | N |



#### DTO.certDefaultSettings
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Type | string | type | N |

| DNSAPI | int | dnsapi | N |



#### DTO.certListResult
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Certs | []CertDetail | (no json tag) | N |

| Total | int64 | (no json tag) | N |



#### DTO.configItemPayload
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Name | string | name | N |

| Value | string | value | N |

| Enable | *bool | enable | Y |



#### DTO.configItemUpsertRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Type | string | type | N |

| ScopeName | string | scope_name | N |

| ScopeID | int64 | scope_id | N |

| Items | []configItemPayload | items | N |



#### DTO.dispatchRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| nodeID | int64 | (no json tag) | N |

| task | models.Task | (no json tag) | N |

| payload | string | (no json tag) | N |

| onError | func(error) | (no json tag) | N |



#### DTO.forwardDefaultItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| IDStr | string | id_str,omitempty | Y |

| Key | string | key | N |

| Value | interface{} | value | N |

| Scope | string | scope | N |

| GroupID | int64 | group_id | N |

| GroupName | string | group_name | N |



#### DTO.forwardListItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| UserName | string | user_name | N |

| ListenPorts | string | listen_ports | N |

| OriginDisplay | string | origin_display | N |

| Origin | string | origin | N |

| UserPackageID | int64 | user_package_id | N |

| UserPackageName | string | user_package_name | N |

| GroupID | int64 | group_id | N |

| GroupIDs | []int64 | group_ids | N |

| GroupName | string | group_name | N |

| NodeGroupID | int64 | node_group_id | N |

| NodeGroupName | string | node_group_name | N |

| CNAME | string | cname | N |

| Status | bool | status | N |

| Remark | string | remark | N |

| CreatedAt | time.Time | created_at | N |



#### DTO.forwardQueryResult
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Forwards | []models.Forward | (no json tag) | N |

| Total | int64 | (no json tag) | N |



#### DTO.forwardRankingItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Port | string | port | N |

| Connections | uint64 | connections | N |

| Traffic | string | traffic | N |



#### DTO.issueTaskMeta
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| TargetNodeID | int64 | target_node_id | N |



#### DTO.knownDomainSet
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Exact | map[string]struct{} | (no json tag) | N |

| Host | map[string]struct{} | (no json tag) | N |



#### DTO.l2NodeInfo
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| IP | string | ip | N |

| Port | int | port | N |

| CheckProtocol | string | check_protocol | N |

| CheckPort | int | check_port | N |

| CheckHost | string | check_host | N |

| CheckPath | string | check_path | N |

| CheckTimeout | int | check_timeout | N |



#### DTO.lineActionRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Action | string | action | N |

| IDs | []int64 | ids | N |

| Value | string | value | N |



#### DTO.lineAssignRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| LineID | string | line_id | N |

| LineName | string | line_name | N |

| Items | []lineIPItem | items | N |



#### DTO.lineAssignedItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| NodeID | int64 | node_id | N |

| NodeIPID | int64 | node_ip_id | N |

| LineID | string | line_id | N |

| LineName | string | line_name | N |

| Name | string | name | N |

| IP | string | ip | N |

| Online | bool | online | N |

| IsOn | bool | is_on | N |

| NodeIsOn | bool | node_is_on | N |

| IsBackup | bool | is_backup | N |

| IsBackupDefaultLine | bool | is_backup_default_line | N |

| Weight | string | weight | N |

| SortOrder | int | sort_order | N |



#### DTO.lineIPItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| NodeID | int64 | node_id | N |

| NodeIPID | int64 | node_ip_id | N |

| Name | string | name | N |

| IP | string | ip | N |

| Online | bool | online | N |



#### DTO.logSummary
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| CheckedAt | time.Time | checked_at | N |

| FailCount | int64 | fail_count | N |

| TotalCount | int64 | total_count | N |



#### DTO.messageRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Type | string | type | N |

| TypeLabel | string | type_label | N |

| Title | string | title | N |

| Content | string | content | N |

| Phone | string | phone | N |

| SiteID | int64 | site_id | N |

| CreatedAt | string | created_at | N |

| IsRead | bool | is_read | N |



#### DTO.metricPoint
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Time | string | time | N |

| Value | float64 | value | N |



#### DTO.nodeGroupCount
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| NodeGroupID | int64 | (no json tag) | N |

| Count | int64 | (no json tag) | N |



#### DTO.nodeGroupPolicy
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Ipv4Resolution | string | ipv4_resolution | N |

| L2Config | string | l2_config | N |

| SortOrder | int | sort_order | N |



#### DTO.nodeGroupView
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| NodeCount | int64 | node_count | N |

| SiteCount | int64 | site_count | N |

| ForwardCount | int64 | forward_count | N |



#### DTO.nodeRankItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Rank | int | rank | N |

| Node | string | node | N |

| NIC | string | nic | N |

| Out | string | out | N |

| In | string | in | N |



#### DTO.nodeRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| RegionID | *int64 | region_id | Y |

| Name | string | name | N |

| Remark | string | remark | N |

| IP | string | ip | N |

| Host | string | host | N |

| Port | int | port | N |

| HttpProxy | string | http_proxy | N |

| IsMgmt | bool | is_mgmt | N |

| Enable | bool | enable | N |

| CheckOn | bool | check_on | N |

| CheckProtocol | string | check_protocol | N |

| CheckTimeout | int | check_timeout | N |

| CheckPort | int | check_port | N |

| CheckHost | string | check_host | N |

| CheckPath | string | check_path | N |

| CheckNodeGroup | string | check_node_group | N |

| CheckAction | string | check_action | N |

| BwLimit | string | bw_limit | N |

| Level | int | type | N |

| Sort | int | sort_order | N |

| CacheDir | string | cache_dir | N |

| MaxCacheSize | int | cache_limit | N |

| LogDir | string | log_dir | N |

| SSHHost | string | ssh_host | N |

| SSHPort | int | ssh_port | N |

| SSHUser | string | ssh_user | N |

| SSHAuthType | string | ssh_auth_type | N |

| SSHPassword | string | ssh_password | N |

| SSHKey | string | ssh_key | N |

| WorkDir | string | work_dir | N |

| AutoInstall | bool | auto_install | N |

| SubIPs | []models.NodeSubIP | sub_ips | N |



#### DTO.packageLineKey
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | string | (no json tag) | N |

| Name | string | (no json tag) | N |



#### DTO.planDetail
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| HTTPPort | int64 | http_port | N |

| StreamPort | int64 | stream_port | N |

| CnameDomain | string | cname_domain | N |

| CnameHostname2 | string | cname_hostname2 | N |

| CnameMode | string | cname_mode | N |

| BuyNumLimit | int64 | buy_num_limit | N |

| BackendIPLimit | string | backend_ip_limit | N |

| IDVerify | bool | id_verify | N |

| BeforeExpDaysRenew | int64 | before_exp_days_renew | N |

| ExpireAt | *time.Time | expire | Y |

| Owner | string | owner | N |



#### DTO.planItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Name | string | name | N |

| Desc | string | desc | N |

| Group | string | group | N |

| Region | int64 | region | N |

| LineGroup | int64 | line_group | N |

| BackupGroup | int64 | backup_group | N |

| TrafficLimit | int64 | traffic_limit | N |

| BandwidthLimit | string | bandwidth_limit | N |

| ConnectionLimit | int64 | connection_limit | N |

| DomainLimit | int64 | domain_limit | N |

| CustomCCRules | bool | custom_cc_rules | N |

| Websocket | bool | websocket | N |

| L2Origin | bool | l2_origin | N |

| PriceMonthly | int64 | price_monthly | N |

| PriceQuarterly | int64 | price_quarterly | N |

| PriceYearly | int64 | price_yearly | N |

| SortOrder | int | sort_order | N |

| Status | bool | status | N |



#### DTO.profileResponse
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Name | string | name | N |

| Email | string | email | N |

| Phone | string | phone | N |

| QQ | string | qq | N |

| Balance | int64 | balance | N |

| CertName | string | cert_name | N |

| CertNo | string | cert_no | N |

| CertVerified | bool | cert_verified | N |

| WhiteIP | string | white_ip | N |

| LoginCaptcha | string | login_captcha | N |

| CreatedAt | string | create_at | N |



#### DTO.purgeLimit
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| RefreshURL | int | refresh_url | N |

| RefreshDir | int | refresh_dir | N |

| Preheat | int | preheat | N |



#### DTO.purgeTaskMeta
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| UserID | int64 | user_id | N |



#### DTO.purgeUsage
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Date | string | date | N |

| RefreshURL | int | refresh_url | N |

| RefreshDir | int | refresh_dir | N |

| Preheat | int | preheat | N |



#### DTO.regionView
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Name | string | name | N |

| Remark | string | remark | N |

| L2CheckPort | int | l2_check_port | N |

| SortOrder | int | sort_order | N |

| CreatedAt | time.Time | create_at | N |

| UpdatedAt | time.Time | update_at | N |



#### DTO.siteGroupMeta
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | (no json tag) | N |

| UserID | int64 | (no json tag) | N |

| Name | string | (no json tag) | N |



#### DTO.siteListItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| UserName | string | user_name | N |

| Domains | []string | domains | N |

| DomainDisplay | string | domain_display | N |

| ListenPorts | string | listen_ports | N |

| HttpListen | []string | http_listen | N |

| HttpsListen | []string | https_listen | N |

| OriginDisplay | string | origin_display | N |

| CNAME | string | cname | N |

| Backends | []string | backends | N |

| HTTPS | bool | https | N |

| CertID | int64 | cert_id | N |

| UserPackageID | int64 | user_package_id | N |

| UserPackageName | string | user_package_name | N |

| DNSProviderID | int64 | dns_provider_id | N |

| GroupID | int64 | group_id | N |

| GroupIDs | []int64 | group_ids | N |

| GroupName | string | group_name | N |

| NodeGroupID | int64 | node_group_id | N |

| NodeGroupName | string | node_group_name | N |

| RegionID | int64 | region_id | N |

| RegionName | string | region_name | N |

| Status | bool | status | N |

| State | string | state | N |

| Settings | map[string]interface{} | settings | N |

| ExpireTime | string | expire_time | N |

| CreatedAt | time.Time | created_at | N |

| UpdatedAt | time.Time | updated_at | N |



#### DTO.siteQueryResult
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Sites | []models.Site | (no json tag) | N |

| Total | int64 | (no json tag) | N |



#### DTO.taskInfo
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Ret | string | (no json tag) | N |

| State | string | (no json tag) | N |

| RetryAt | *time.Time | (no json tag) | Y |

| ErrTimes | int | (no json tag) | N |



#### DTO.taskListItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| PID | int64 | pid | N |

| Pry | int | pry | N |

| Name | string | name | N |

| Type | string | type | N |

| Depend | string | depend | N |

| CreateAt | time.Time | create_at | N |

| StartAt | *time.Time | start_at | Y |

| EndAt | *time.Time | end_at | Y |

| State | string | state | N |

| ErrTimes | int | err_times | N |

| Progress | string | progress | N |



#### DTO.taskLogEntry
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Time | string | time | N |

| NodeID | string | node_id | N |

| State | string | state | N |

| Message | string | message | N |

| Attempt | int | attempt | N |



#### DTO.taskMeta
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| UserID | int64 | user_id | N |



#### DTO.taskProgressPayload
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Progress | int | progress | N |

| Message | string | message | N |



#### DTO.updatePasswordRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Current | string | current | N |

| Next | string | next | N |

| Hash | string | password_hash | N |



#### DTO.updateProfileRequest
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Email | string | email | N |

| Phone | string | phone | N |

| QQ | string | qq | N |

| CertName | string | cert_name | N |

| CertNo | string | cert_no | N |

| WhiteIP | string | white_ip | N |

| LoginCaptcha | string | login_captcha | N |



#### DTO.uploadVersionReq
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Version | string | version | N |



#### DTO.usagePoint
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| Time | string | time | N |

| Value | float64 | value | N |



#### DTO.userOrderRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| Type | string | type | N |

| TypeLabel | string | type_label | N |

| Remark | string | remark | N |

| Price | string | price | N |

| Pay | string | pay | N |

| More | string | more | N |

| PayType | string | pay_type | N |

| OrderNo | string | order_no | N |

| CreatedAt | string | created_at | N |

| Paid | bool | paid | N |



#### DTO.userPackageRow
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| IPv6 | bool | ipv6 | N |

| Status | string | status | N |

| CnameDomain | string | cname_domain | N |

| CnameHostname | string | cname_hostname | N |

| CnameMode | string | cname_mode | N |

| RecordID | string | record_id | N |



#### DTO.userPlanItem
**现有行为(Go 代码)**
- DTO 结构以当前定义为准，字段命名/类型保持一致

**C# 重构要求**
- C# DTO 字段名与类型必须与表格一致
- 不新增字段，避免前端歧义


| Field | Type | JSON | Optional(Y/N) |

|---|---|---|---|

| ID | int64 | id | N |

| UserID | int64 | user_id | N |

| UserName | string | user_name | N |

| PackageID | int64 | package_id | N |

| PackageName | string | package_name | N |

| PlanName | string | plan_name | N |

| RecordID | string | record_id | N |

| RegionID | int64 | region_id | N |

| NodeGroupID | int64 | node_group_id | N |

| BackupGroupID | int64 | backup_group_id | N |

| EnableBackup | bool | enable_backup_group | N |

| Traffic | int64 | traffic | N |

| Bandwidth | string | bandwidth | N |

| Connection | int64 | connection | N |

| DomainLimit | int64 | domain | N |

| MainDomainLimit | int64 | main_domain_limit | N |

| HTTPPort | int64 | http_port | N |

| StreamPort | int64 | stream_port | N |

| CustomCCRule | bool | custom_cc_rule | N |

| Websocket | bool | websocket | N |

| CnameDomain | string | cname_domain | N |

| CnameHostname | string | cname_hostname | N |

| CnameHostname2 | string | cname_hostname2 | N |

| CnameMode | string | cname_mode | N |

| StartAt | time.Time | start_at | N |

| EndAt | time.Time | end_at | N |

| Status | string | status | N |

| CreatedAt | time.Time | created_at | N |





### 30.4 关键接口返回 data 结构示例（Task / Logs / Stats）


> 说明：以下仅展示 `data` 内部结构与字段语义，外层统一`{code,message,data,trace_id}`（详见 24/30.1）


#### 30.4.1 Task 任务

**现有行为（Go 代码）**
- 以下示例即现有返回结构
- 列表接口顶层重复返回 `list/total/page` 用于兼容旧前端

**C# 重构要求**
- 字段命名、时间格式与顶层重复字段必须保持一致
- 外层统一 `{code,message,data,trace_id}` 不变



**GET /api/v1/*/tasks（列表）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {

        "id": 1,

        "pid": 0,

        "pry": 0,

        "name": "",

        "type": "refresh_url|refresh_dir|preheat|config_sync|...",

        "depend": "",

        "create_at": "2024-01-01T10:00:00Z",

        "start_at": "2024-01-01T10:00:10Z",

        "end_at": "2024-01-01T10:02:00Z",

        "state": "waiting|running|done|fail",

        "err_times": 0,

        "progress": ""

      }

    ],

    "total": 1,

    "page": 1

  },

  "list": ["..."],

  "total": 1,

  "page": 1

}

```

> 说明：顶层重复返`list/total/page` 为兼容旧前端，C# 重写需保留


**GET /api/v1/*/tasks/:id（详情）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "id": 1,

    "pid": 0,

    "pry": 0,

    "name": "",

    "type": "refresh_url|refresh_dir|preheat|config_sync|...",

    "depend": "",

    "create_at": "2024-01-01T10:00:00Z",

    "start_at": "2024-01-01T10:00:10Z",

    "end_at": "2024-01-01T10:02:00Z",

    "state": "waiting|running|done|fail",

    "err_times": 0,

    "progress": "",

    "ret": ""

  }

}

```



**GET /api/v1/*/tasks/usage（配额）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "limits": { "refresh_url": 2000, "refresh_dir": 500, "preheat": 2000 },

    "used": { "date": "2024-01-01", "refresh_url": 3, "refresh_dir": 1, "preheat": 0 },

    "remaining": { "refresh_url": 1997, "refresh_dir": 499, "preheat": 2000 }

  }

}

```



**POST /api/v1/*/tasks（创建） / POST /api/v1/*/tasks/:id/resubmit（重提）**

```json

{"code":200,"message":"Success","data":null}

```



#### 30.4.2 Logs 日志

**现有行为（Go 代码）**
- 以下示例即现有日志接口的返回结构（字段名与类型需保持一致）

**C# 重构要求**
- 保持字段名、时间格式与 list/total 结构不变
- 外层统一 `{code,message,data,trace_id}` 不变



**GET /api/v1/admin/logs/login（登录日志）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {

        "id": 1,

        "user_id": 100,

        "username": "admin",

        "ip": "1.2.3.4",

        "success": true,

        "post_content": "{...}",

        "created_at": "2024-01-01T10:00:00Z"

      }

    ],

    "total": 1

  }

}

```



**GET /api/v1/admin/logs/operation（操作日志）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {

        "id": 1,

        "user_id": 100,

        "type": "admin|user",

        "action": "site.update",

        "content": "...",

        "diff": "...",

        "ip": "1.2.3.4",

        "process": "api",

        "created_at": "2024-01-01T10:00:00Z",

        "username": "admin",

        "description": "..."

      }

    ],

    "total": 1

  }

}

```



**GET /api/v1/*/logs/access（访问日志，ClickHouse）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {

        "timestamp": "2024-01-01T10:00:00Z",

        "node_id": "1001",

        "node_ip": "10.0.0.1",

        "remote_addr": "8.8.8.8",

        "host": "example.com",

        "method": "GET",

        "uri": "/index.html",

        "status": 200,

        "bytes": 1234,

        "request_time": 0.123,

        "upstream_addr": "1.1.1.1:80",

        "upstream_response_time": 0.045,

        "upstream_cache_status": "HIT|MISS|BYPASS",

        "http_referer": "-",

        "http_user_agent": "Mozilla/...",

        "scheme": "https",

        "ssl_protocol": "TLSv1.3",

        "ssl_cipher": "TLS_AES_128_GCM_SHA256"

      }

    ],

    "total": 1

  }

}

```

> 用户请求`node_ip` `upstream_addr` 会被清空返回


**GET /api/v1/admin/logs/backup（备份任务日志）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {

        "id": 1,

        "created_at": "2024-01-01T10:00:00Z",

        "finished_at": "2024-01-01T10:10:00Z",

        "status": 1,

        "result": "..."

      }

    ],

    "total": 1

  }

}

```



**GET /api/v1/admin/logs/mail（邮件/短信日志）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {

        "message_id": 1,

        "user_id": 100,

        "subject": "title",

        "medium": "Email|SMS",

        "fails": 0,

        "status": 1,

        "reason": "",

        "created_at": "2024-01-01T10:00:00Z"

      }

    ],

    "total": 1

  }

}

```



**GET /api/v1/*/logs/block/current（封禁当前）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {"id": 1, "site_id": 10, "domain": "a.com", "ip": "1.1.1.1", "location": "CN-BJ", "filter": "HTTP_403", "block_time": "2024-01-01 10:00:00", "release_time": "PERMANENT"}

    ],

    "total": 1

  }

}

```



**GET /api/v1/*/logs/block/stats（封禁统计）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {"site_id": 10, "domain": "a.com", "count": 100}

    ],

    "total": 1

  }

}

```



**GET /api/v1/*/logs/block/history（封禁历史）**

```json

{

  "code": 200,
  "message": "Success",

  "data": {

    "list": [

      {"id": 1, "site_id": 10, "domain": "a.com", "ip": "1.1.1.1", "location": "CN-BJ", "filter": "HTTP_403", "block_time": "2024-01-01 10:00:00", "is_manual": false}

    ],

    "total": 1

  }

}

```



#### 30.4.3 Stats 统计

**现有行为（Go 代码）**
- 以下示例即现有统计接口的 `data` 结构

**C# 重构要求**
- 保持字段命名、单位与数组结构不变
- 外层统一 `{code,message,data,trace_id}` 不变



**GET /api/v1/*/stats/basic（带宽/流量/QPS）**

```json

{"code":200,"message":"Success","data": { "x_axis": ["10:00"], "bandwidth": [12.3], "traffic": [45.6], "qps": [123.4] } }

```



**GET /api/v1/*/stats/quality（命中率/4xx/5xx）**

```json

{"code":200,"message":"Success","data": { "x_axis": ["10:00"], "hit_rate": [98.2], "status_4xx": [3], "status_5xx": [1] } }

```



**GET /api/v1/*/stats/origin（回源带宽/流量）**

```json

{"code":200,"message":"Success","data": { "x_axis": ["10:00"], "origin_bandwidth": [1.2], "origin_traffic": [3.4] } }

```



**GET /api/v1/*/stats/ranking（访问排行）**

- 常规类型（domain/url/ip/referer/country/province）：

```json

{"code":200,"message":"Success","data": { "list": [ {"rank":1,"item":"example.com","request_count":1000,"out_traffic":"12.3MB","origin_traffic":"4.5MB"} ] } }

```

- 延迟排行（type=latency）：

```json

{"code":200,"message":"Success","data": { "list": [ {"rank":1,"item":"example.com/index","request_count":100,"avg_time":0.123,"max_time":0.456,"min_time":0.012,"p95_time":0.345} ] } }

```



**GET /api/v1/admin/stats/node_traffic（节点进/出流量）**

```json

{"code":200,"message":"Success","data": { "x_axis": ["2024-01-01"], "in_traffic": [12.3], "out_traffic": [45.6] } }

```



**GET /api/v1/admin/stats/node_ranking（节点排行）**

```json

{"code":200,"message":"Success","data": { "list": [ {"rank":1,"node":"node-1","nic":"eth0","out":"120 Mbps","in":"30 Mbps"} ] } }

```



**GET /api/v1/admin/stats/node_metrics（节点指标时序）**

```json

{"code":200,"message":"Success","data": { "list": [ {"time":"10:00","value":12.3} ] } }

```



**GET /api/v1/user/usage（套餐用量）**

```json

{"code":200,"message":"Success","data": { "x_axis":["10:00"], "values":[1.2], "list":[{"time":"10:00","value":1.2}], "total":12.3, "avg":1.0, "peak":2.1, "unit":"MB" } }

```






---

## 附录：前端页面元素与权限（基于 web/admin/src）

> 规则：以现有前端 Vue 页面代码为准；列出页面内的筛选项、表格列、操作按钮与弹窗字段，并区分 Admin/User 可见性

### A. 节点管理（Admin）

#### A1. 节点列表 `/node/list`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/nodes/List.vue` + `web/admin/src/views/nodes/list/NodeTable.vue`
- 列表自动轮询：存在 `install_status=running` 的节点时，每 5 秒刷新一次列表

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**顶部操作**
- 安装节点（主按钮）
- 批量：禁用节点(`stop`) / 启用节点(`start`)
- 刷新
- 更多操作（下拉：删除选中）
- 以上批量按钮在未选中行时禁用

**筛选条**
- 区域：`region_id`
- 状态：`status`（正常/禁用）
- 类型：`node_type`（L1边缘/L2中间）
- 关键字：节点名称 / IP
- 操作：搜索 / 清除

**表格**
- 选择
- ID
- 名称（点击进入编辑）
- 区域（显示区域名 + 线路组数量，点击跳转线路分组）
- 节点 IP（含 IP 展示；子 IP 以弹窗列表显示）
- 监控（显示协议 + 监控日志入口）
- 带宽（点击跳转实时监控）
- 状态（在线/离线/禁用，带色点）
- 安装状态（未安装/安装中/安装成功/安装失败；安装中显示进度%，失败 hover 显示错误详情）
- 开启（开关）
- 备注
- 排序
- 操作：管理（编辑），更多（重新安装、删除）

**行级动作**
- 重新安装：仅当存在 `ssh_user` 时可
- 删除：二次确认
- 启停：切换 `enable` 状态（失败会回滚开关）
- 带宽列跳转路由：`/nodes/monitor?node_id=<id>`（前端当前写死该路径）

#### A2. 节点编辑弹窗（新增/编辑）
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/nodes/list/NodeEditDialog.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**Tab：基本设置**
- 名称
- 区域（若节点已分配线路则锁定；锁定依据 `line_count>0`）
- 备注
- 排序
- IP
- 类型（L1/L2）
- SSH 认证：端口 / 用户 / 认证方式（密码/私钥）
- 工作目录（固定为可执行文件所在目录，只读）
- 提示：运行时根目录始终为 `WorkDir/edge-node`
- 自动安装（开关）
- 说明：新增时若区域为空，自动选择列表第一个区域

**Tab：节点设置**
- 缓存目录
- 缓存上限（GB）
- 日志目录
- 带宽限制（Mbps）

**Tab：子 IP**
- IP 文本域（每行一 IP）

**提交行为**
- 新增：`auto_install=true` 时使用带超时的创建请求；若返回 `install_error` 弹出警告
- 编辑：保存成功关闭弹窗并刷新列表

#### A3. 区域管理 `/node/region`（节点列表内 Tab）
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/nodes/list/RegionList.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**顶部操作**
- 新增区域
- 删除（批量）

**表格**
- 选择
- ID
- 名称
- 备注
- L2 检测端口
- 排序
- 添加时间
- 操作：编辑/删除

**区域编辑弹窗**
- 名称
- 备注
- L2 检测端口
- 排序
- 批量删除：前端逐个调用删除接口

#### A4. 节点监控日志弹窗
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/nodes/list/MonitorLogDialog.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**筛选条**
- 日志类型：可用性监控日志（下拉仅此一项）
- 时间范围（时间区间选择）

**表格**
- 检测时间
- 失败个数
- 总检测点数

#### A5. 线路分组 `/node/groups`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/nodes/groups/List.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**顶部操作**
- 新增分组
- 删除（批量）
- 过滤：区域、关键字（名称或解析值）
- 清除筛选

**表格**
- 选择
- ID
- 名称（点击编辑）
- 区域（无值显示“默认”）
- 解析值（展示主解析 + IPv4 解析值）
- 统计：节点数 / 网站 / 转发
- L2 配置（默认/自定义）
- 排序
- 操作：配置解析/编辑/删除

**分组编辑弹窗**
- 名称
- 区域
- 解析值（留空自动生成）
- CNAME 域名（下拉选择）
- IPv4 解析值（留空自动生成）
- 备注
- 排序
- L2 配置（默认配置）
- 备用 IP 切换策略（有 IP 下线 / 在线 IP 数少于备用 IP / 间隔切换）

#### A6. 分组解析配置 `/node/groups/:id/resolution`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/nodes/groups/Resolution.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**顶部区域**
- 返回按钮
- 区域选择
- 分组选择

**左侧：未设置的 IP**
- 操作：批量添加 / 批量备用
- 搜索：IP/名称
- 表格列：名称 / IP / 状态（在线/不在线）

**右侧：已分配解析**
- 线路选择（级联：全部/默认/电信/联通/移动/其他运营商(铁通/广电/教育网)/境内(各省)/境外/搜索引擎(百度/谷歌/有道/必应/搜狗/奇虎/搜索引擎)/线路分组/自定义线路）
- 提示：当前线路为“全部”时，新增节点会对所有线路生效
- 操作：启用 / 禁用 / 删除 / 更多操作
- 更多操作：设置权重 / 备用 IP / 取消备用 IP / 备用默认解析 / 取消备用默认解析 / 修改排序
- 搜索：IP/名称
- 表格列：ID / 线路 / 名称 / IP / 备用 IP（图标或“否”） / 状态（禁用/节点已禁用/节点离线/启用） / 权重 / 排序

**限制与提示**
- 禁止添加离线或禁用节点
- 删除前要求先禁用节点
- 自动刷新（3 秒轮询）

#### A7. DNS 设置 `/node/dns`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/dns/Index.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**Tab：DNS 配置**
- DNS Provider（下拉选择）
- 动态凭证字段（根据提供商类型）
- TTL
- 开启 IP 权重（开关）
- DNS 错误显示
- 操作：保存 / 记录修复 / 清除 CDN 无关解析
- Tab 切换到 DNS 配置时会触发 DNS 测试并刷新错误信息

**Tab：CNAME 域名**
- 操作：添加域 / 批量删除（前端提示暂未开放）
- 搜索：域名关键字
- 表格列：ID / 域名 / DNS Provider / 备注 / 操作（编辑/删除）
- 新增/编辑弹窗：域名 / DNS Provider / 备注

#### A8. 监控配置 `/node/monitor`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/settings/Monitor.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**表单字段**
- 通知时间段（`notification_period`，默认 `8-22`）
- 通知方式（`notify_method`：邮件/短信/邮件+短信）
- 通知类型（多选）：节点 IP 变化 / 带宽超限 / 备用 IP / 默认线路备份 / 节点组备份
- 邮箱 / 手机
- 带宽超限次数
- 高负载自动降权（开关）
- 高负载恢复时间（秒）
- 监控 API
- 检测间隔（秒）
- 失败次数
- 失败率（%）
- 保存配置

**数据组织**
- `notify_msg_type` 在前端以空格拼接保存（多选值 -> 字符串）

#### A9. 实时监控 `/node/realtime`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/nodes/RealtimeMonitor.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**Tab：资源排行**
- 指标：带宽 / 连接 / 负载 / 硬盘
- 时间：1m / 5m / 30m / 1h
- 操作：刷新
- 表格列：排行 / 节点 / 网卡 / 出站带宽 / 入站带宽

**Tab：监控指标**
- 指标：带宽 / 连接 / 负载 / 硬盘
- 时间：1h / 6h / 12h / 自定义时间范围
- 操作：刷新
- 表格列：时间 / 值

**Tab：节点流量**
- 类型：出站流量 / 入站流量（多选）
- 时间：1d / 7d / 30d / 自定义时间范围
- 节点选择（全部节点/指定节点）
- 排除网卡（输入）
- 操作：刷新
- 展示：折线图（ECharts）


### B. 全局配置（Admin）

#### B1. 防火墙配置 `/global/firewall`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/global/Firewall.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**Tab：基础防护 & 拉黑策略**
- 全局 WAF 开启
- 默认拉黑方式：IPSet / 断开连接 / 拦截页面
- 自动 IPSet：开关 + 触发阈值（次/秒）
- 拉黑页频率限制：开关 + 阈值（次/60 秒）
- 拉黑页不计流量
- 黑名单封禁时长（秒）
- 临时白名单时长（秒）
- 临时白名单自动加入条件（5 秒内）：总请求数限制 / URL 请求限制

**Tab：安全控制 & 名单**
- 白名单 IP（多行，CIDR）
- 黑名单 IP（多行，CIDR）
- 防止 TLS 握手攻击（开关）
- 禁止未绑定域名访问（开关）
- 禁止 PING（开关）

**Tab：CC 防护 & 验证**
- 默认页防护模式：强制开启 / 自动开启
- 自动开启阈值（请求/秒）
- CC 验证方式：滑动验证 / 点击验证 / 5 秒盾 / 图片旋转 / 简单滑动
- 验证图片来源：系统默认 / 自定义 URL
- 图片 URL（自定义时生效）
- 开启调试日志
- CC 规则自动切换

**Tab：高级防护 & 系统**
- 通讯密钥（SecretKey）
- 节点日志清理策略：不清理 / 仅清理日志 / 清理日志+缓存
- `.well-known` 防护阈值（60 秒内 404 次数）
- 内置资源防护：开启 / 阈值（QPS）/ 拉黑时长（秒）
- 限流规则表：统计时长（秒） / 最大请求数 / 添加 / 删除

**保存行为**
- 输入blur 保存，开change 即保存；Tab 切换重新拉取

#### B2. Nginx 配置 `/global/nginx`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/global/Nginx.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**工作进程**
- `worker_processes`（auto/数字）
- `worker_connections`
- `worker_rlimit_nofile`
- `worker_shutdown_timeout`

**路径**
- 日志目录：`log_directory`

**其他**
- `keepalive_timeout`（秒）
- `gzip` 开关
- `custom_snippet`（HTTP block 自定义片段）

**保存/读取**
- 读取 `config_items(type=nginx_config)` `nginx-config-file` JSON
- 保存时覆盖 JSON（worker/日志/http.keepalive_timeout/gzip/custom_snippet）

#### B3. 资源限制配置 `/global/resources`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/global/Resources.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**Tab：Website**
- 相关配置限制不低于（min_limit）
- 相关配置限制的最大倍数（max_limit_multiplier）
- 黑名单 IP 数量限制 / 白名单 IP 数量限制
- 日清 URL 缓存次数 / 日清目录缓存次数 / 日预热 URL 次数
- 日解锁 IP 次数 / 每次解锁 IP 个数
- 单个 CC 规则数量 / 单个 ACL 规则数量
- 每天允许下载日志次数
- 日志文件存放目录 / 日志文件存放时长（小时）
- 单个站点最大域名数限制
- 默认监听 80 端口（开关）

**Tab：Forward**
- 禁用端口（空格分隔，`80 443`）
- 相关配置限制不低于（min_limit）
- 相关配置限制的最大倍数（max_limit_multiplier）
- ACL 规则数量限制

**Tab：Public**
- 禁用的自定义端口（空格分隔）
- 允许的自定义端口（范围或列表）

**保存行为**
- 输入blur 保存；Tab 切换重新拉取

#### B4. 默认配置 `/global/default`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/global/DefaultConfig.vue` + `web/admin/src/views/global/components/*`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin

**顶层 Tab**
- 全局配置（子 Tab：网站/转发/证书）
- 缓存配置（子 Tab：网站 / API / 下载）

**B4.1 全局配置-网站（SiteConfig）**
- HTTP：监听端口（`http_listen-port`）
- HTTPS：监听端口 / HSTS / HTTP2 / HTTP3 / 强制 HTTPS / SSL 协议 / SSL 加密套件 / 优先服务端加密套件 / OCSP 装订
- 回源设置：协议（http/https/follow/follow_port）/ 回源 HTTP 端口 / 回源 HTTPS 端口 / 回源超时 / 连接超时 / 回源 SSL 协议
- 缓存规则
  - 表格列：类型 / 内容 / 有效期 / 忽略参数 / 强制缓存 / 操作
  - 快速添加：首页 / 全站 / 静态资源 / 视频 / WordPress
  - 新增/编辑规则字段：类型（index/all/dir/suffix/path）/ 内容 / 有效期（天/时/分/秒）/ 忽略参数 / 强制缓存
  - 更多设置：分片回源（enable_range）/ 忽略 Vary（ignore_vary）/ 不缓存条件（skip_conditions）
  - 不缓存条件：匹配项（请求URI/请求URI不带参数/客户IP/请求协议/请求参数/域名/自定义）+ 匹配值
  - 规则匹配自上而下，命中即停止；可拖拽排序（界面提示）
- 源站请求头：表格列（名称/值/操作），新增/编辑（名称、值）
- 访问日志：记录请求头 / 记录响应头 / 记录请求体 / 请求体大小限制（KB）
- 其它：负载方式（rr/ip_hash）/ 默认 CC 规则（下拉默认项：关闭/宽松/JS验证/5秒盾/点击验证/滑块验证/验证码/旋转图片/点击验证(简单)/滑块验证(简单)/临时白名单；若 `/rules/cc/groups` 有数据则覆盖）/ 搜索引擎爬虫动作（不设置/放行/拦截）
- 高级：Gzip 开关 + 类型 / WebSocket 开关 / 屏蔽透明代理 / 数据实时返回 / 数据实时发送 / IPv6 开关

**B4.2 全局配置-转发（StreamConfig）**
- 监听协议（tcp/udp）
- 负载方式（rr/ip_hash）
- Proxy Protocol（开关）

**B4.3 全局配置-证书（CertConfig）**
- 默认证书类型：ZeroSSL / Let's Encrypt / BuyPass / Google CA
- DNS 接口：选择 DNS Provider
- 动态字段：不同 provider 的认证字段（名称自动翻译）
- 更改 DNS 接口会清空认证字段并立即保存

**B4.4 缓存配置（CacheConfig）**
- 网站 / API / 下载 三类模板
- 每类字段：`cache_enable` / `cache_ttl` / `gzip` / `waf_enable`
- 数据来源：`/global_config.default_config`

**保存行为**
- SiteConfig 使用 `/site_defaults`（`type=site_default_config`）
- StreamConfig/CertConfig 使用 `/config_items`（`type=stream_default_config` / `cert_default_config`）
- CacheConfig 使用 `/global_config`
- change/blur 触发保存；有 debounce（约 300ms）

#### B5. 错误页面 `/global/errors`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/global/ErrorPages.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


**可见角色**：Admin
- 左侧 Tab：400/403/502/504/traffic_limit/site_locked/domain_invalid/conn_limit/timeout/ip
- 每个 Tab：HTML 文本编辑 + 预览
- blur 保存，空值或未变化不触发保存；缺失时自动补默认模板

#### B6. 套餐/计划
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/plans/Basic.vue` + `web/admin/src/views/plans/Sold.vue` + `web/admin/src/views/packages/My.vue` + `web/admin/src/views/packages/Usage.vue`

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致


##### B6.1 基础套餐 `/plans/basic`

**可见角色**：Admin

**顶部操作**
- 添加套餐

**表格列**
- ID / 名称 / 分组 / 区域 / 月付 / 状态（启用/禁用）
- 操作：管理 / 分配 / 删除

**新增/编辑弹窗字段（/plans）**
- 基本：`name/desc/group(default)/region/line_group/backup_group`
- 资源限制：`traffic_limit/bandwidth_limit/connection_limit/stream_port/domain_limit/http_port`
- 功能：`custom_cc_rules/websocket/l2_origin`
- 定价：`price_monthly/price_quarterly/price_yearly`
- CNAME：`cname_hostname2/cname_domain/cname_mode(domain|package)`
- 购买限制：`buy_num_limit/expire/id_verify/before_exp_days_renew`
- 其它：`owner/sort_order/status/backend_ip_limit`

**前端校验**
- `region`/`line_group` 必填
- `backup_group` 不能与 `line_group` 相同
- `cname_domain` 必选（前端硬性校验）

**新增默认值**
- `status=true`，`websocket=true`，`custom_cc_rules=true`，`l2_origin=false`，`domain_limit=100`

**分配弹窗（/user_plans/assign）**
- 字段：`plan_id/plan_name/user_id/duration_mode(1/3/12/custom)/end_at`
- 前端规则：`user_id` 必选；`duration_mode=custom` 时必须选择 `end_at`

##### B6.2 已售套餐 `/plans/sold`

**可见角色**：Admin

**顶部操作**
- 同步数据
- 删除（批量）

**筛选**
- `keywordType`：`user_id/user_name/plan_name`
- `keyword`：前端在本地过滤列表（非服务端）

**表格列**
- 选择 / ID / 用户 / 基础套餐 / 套餐名称 / 解析值(record_id) / 购买时间 / 到期时间
- 调试列：`cname_domain` / `cname_mode`
- 操作：详情 / 编辑 / 升降配

**详情弹窗**
- Tab：使用情况（流量/域名/HTTP端口/转发端口 总额/已用/剩余）
- Tab：套餐详情（名称/流量/带宽/连接/域名/HTTP端口/转发端口/自定义CC/WebSocket/到期时间）
- 已购升级包表（目前前端展示为空表）

**升降配弹窗**
- Tab：升级包（表格，前端显示“暂无数据”）
- Tab：更换套餐（选择套餐 -> 确定）
- 现状：提交更换套餐按钮为“暂未实现”

**编辑弹窗（/user_plans/:id）**
- 分组：`region_id/node_group_id/backup_group_id`
- 资源：`traffic/bandwidth/connection/domain/main_domain_limit/http_port/stream_port`
- 功能：`custom_cc_rule/websocket/http3_enabled`
- CNAME：`cname_hostname/cname_domain/cname_mode`
- 续费价格：`price_monthly/price_quarterly/price_yearly`
- 到期时间：`end_at`

**批量删除**
- 通过 `/user_plans`（DELETE + ids）

##### B6.3 我的套餐 `/plans/my`

**可见角色**：User

**表格列**
- ID / 套餐名称 / 到期时间 / 状态
- 操作：详情 / 续费 / 升降配 / 编辑

**详情弹窗**
- Tab：使用情况（流量/域名/主域名/HTTP端口/转发端口 总额/已用/剩余）
- Tab：套餐详情（名称/流量/带宽/连接/域名/HTTP端口/转发端口/自定义CC/WebSocket/IPv6/到期时间）
- 已购升级包表（前端展示为空表）

**续费**
- `/user_packages/:id/renew`，字段：`period`（month/quarter/year）
- 价格为只读显示（根据当前套餐价格）

**升降配**
- `/user_packages/:id/switch`，字段：`package_id`
- 仅“更换套餐”Tab可操作；升级包Tab前端为空

**编辑**
- `/user_packages/:id`，字段：`name/ipv6`
- IPv6 提示：仅对“该套餐生成 CNAME 的网站”有效

##### B6.4 用量查询 `/plans/usage`

**可见角色**：User

**交互**
- 时间范围：`today/yesterday/7days/30days`（按钮组）
- 刷新：链接按钮
- 展示：汇总卡片（Total/Peak/Average）、折线图、明细表格
- 接口：`GET /usage?range=...`
- 数据结构：`x_axis/values/list/total/avg/peak/unit`

### C. 网站管理（Admin/User）

#### C1. 网站列表 `/website/list`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/List.vue` + `web/admin/src/views/website/list/SiteTable.vue`
- 顶部操作：添加网站 / 批量修改 / 申请证书 / 更多操作（启用/禁用/删除/解除黑名单/清空缓存；Admin 额外：CNAME域名 / CNAME模式 / 线路分组）
- 高级搜索弹窗：状态（正常/停用）
- 搜索区：搜索字段（全部/域名/源站IP/CNAME） + 关键字 + 查询 + 导出 + 高级搜索
- 列表列：选择 / ID / 用户(Admin) / 域名 / 监听端口 / 源站 / CNAME / HTTPS / 套餐 / 区域 / 线路组 / 分组 / 状态 / 添加时间 / 操作
- 行操作：管理（跳转 `/website/manage?site_id=...`） / 更多（启用/禁用/删除）
- 清空缓存：弹二次确认，调用批量 action，若返回 task_id 打开任务详情对话框
- 批量 CNAME 域名变更后：自动执行解析检测，弹出检测结果表（域名/期望CNAME/解析CNAME/IP/状态/错误）

**C# 重构要求**
- Blazor 重写需保持筛选项/表格列/按钮/弹窗字段与 Admin/User 差异一致
- 清空缓存必须创建任务并与任务监控联动（保持 task_id 弹窗行为）

**可见角色**：Admin/User（User 无“用户列”，无 CNAME 域名/模式/线路分组批量操作）

#### C2. 新增/编辑网站弹窗
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/list/SiteEditDialog.vue`
- 模式：新增时支持“单个添加/批量添加”两 Tab；编辑时仅单个
- 单个模式字段：
  - 用户(Admin)：可搜索用户，选择后自动拉取该用户套餐/DNS接口/分组
  - 网站域名（每行一个；支持泛域名 `*.example.com`）
  - 域名用量提示（/domain_usage）：总域名/主域名配额与当前用量；超限禁止提交
  - 源站地址（每行一个，如 `1.1.1.1` 或 `1.1.1.1:8080`）
  - 网站套餐（可为空；默认自动选择第一个套餐）
  - 加速类型：网页/ API / 下载
  - 展开更多：网站分组（多选）/ DNS 接口 / 备注
- 批量模式字段：
  - 用户(Admin) / 网站套餐
  - 简单模式：域名批量输入 + 源站（所有域名共用）
  - 高级模式：数据区（`domain=...|ip=...` 每行一条）
  - 选项：忽略错误
  - 网站分组 / DNS 接口
- 校验：
  - 域名必须符合域名或 IP 格式
  - 源站必填

**C# 重构要求**
- 域名批量解析逻辑需与前端格式一致（`domain=...|ip=...`）
- 超配额时禁止创建（与前端一致）

#### C3. 批量修改弹窗（CNAME/线路）
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/list/BatchEditDialog.vue`
- 模式：
  - CNAME 域名：下拉选择（来自 `/cname_domains`）
  - CNAME 模式：按网站生成 / 按套餐生成
  - 线路分组：区域、线路分组、备用线路分组（区域变化联动过滤）
- 提交：`POST /sites/batch_update`，携带 ids 与对应字段

**C# 重构要求**
- 备用线路组选择后需设置 `enable_backup_group=true` 与前端一致

#### C4. 批量设置弹窗
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/list/BatchSettingsDialog.vue`
- 说明：**仅勾选的设置项会生效**；每个折叠项独立“批量修改”
- 基本设置：套餐 / 站点分组（多选）
- HTTP 设置：开关 / 监听端口
- HTTPS 设置：
  - 开关 / 证书选择 / 监听端口
  - 强制 HTTPS + 跳转端口
  - HSTS / HTTP2 / HTTP3 / OCSP
  - SSL 配置（兼容/现代/自定义；自定义含协议+加密套件）
- 回源设置：
  - 协议（http/https/follow/follow_port）
  - HTTP/HTTPS 回源端口
  - 回源 Host（follow/domain/custom + hostValue）
  - 回源超时 / 连接超时
  - 负载方式（ip_hash/rr/url_hash/least_conn/random）
  - 源站列表（address/weight/enable）
  - 条件源站（匹配项/运算符/值/源站）
    - 匹配项：请求URI/请求URI不带参数/节点国家/节点ISP/节点省/节点市/客户国家/客户ISP/客户省/客户市/客户IP/域名/请求头/请求方法/HTTP版本
    - 运算符：等于/不等于/包含/不包含/前缀/后缀/正则/正则不匹配/存在/不存在/IP段内/IP段外
- 缓存设置：规则表（类型/内容/TTL/操作）+ 快速添加（首页/全站/静态资源/视频/WordPress）
  - 规则编辑弹窗：类型（index/all/dir/suffix/path）/ 内容 / 有效期(秒/分/时/天) / 忽略参数 / 强制缓存
  - 高级：分片回源 / 忽略Vary / 不缓存条件（请求URI/请求URI不带参数/客户IP/协议/请求参数/域名/自定义）
- 安全设置：
  - CC 默认规则（系统规则/自定义规则）+ 自动防护（QPS 阈值 + 切换规则）
  - 自定义规则（匹配器+动作+模式+备注+启用）+ 规则弹窗
    - 匹配项：全部/IP/域名/URI/URI(无参)/Header/独立UA数/404数量/方法/UA/Referer/国家/AS/省/市/ISP/HTTP版本/Accept-Language
    - 运算符：等于/不等于/包含/不包含/前缀/后缀/正则/正则不匹配/存在/不存在/IP段内/IP段外
    - 动作：放行/拉黑/请求频率/无感验证/5秒盾/点击/点击(简)/滑块/滑块(简)/验证码/旋转图/302/URL鉴权
  - 搜索引擎爬虫策略（不设置/放行/拦截）
  - 黑白名单时间（默认或自定义秒数）
  - IP 黑白名单（多行）
  - Cookie 域名
  - 屏蔽透明代理
  - 区域屏蔽（国家/地区选择）
- 访问控制：
  - ACL 规则选择
  - 防盗链（范围/值/允许空来源/额外域名）
  - CORS（允许来源/方法/请求头/响应头/凭证/缓存时间）
- 高级设置：
  - 上传大小限制（不限制/自定义KB）
  - Gzip / WebSocket
  - 搜索引擎回源 + 回源IP
  - URL 重定向（域名端口/匹配/跳转/状态码）
  - 源站请求头（Name/Value）
  - CDN 响应头（Name/Value）
  - 访问日志：记录请求头/响应头/请求体/请求体大小限制
  - 其它：源站证书/数据实时识别/数据实时发送/默认站点/L2配置（当前套餐/不配置/自定义）

**C# 重构要求**
- 所有勾选项必须仅覆盖被选字段；未勾选字段保持原值
- 批量提交仍使用 `/sites/batch_update`，结构与前端 payload 完全一致

#### C5. 网站默认设置 `/website/list?tab=default`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/list/DefaultSettings.vue`
- 列表列：设置项 / 设置值 / 范围（全局/分组）/ 操作（编辑/删除）
- 新增/编辑弹窗：
  - 设置项（下拉）：  
    - 默认CC规则/黑名单时间/白名单时间/搜索引擎爬虫  
    - 黑名单IP/白名单IP/屏蔽透明代理/区域屏蔽  
    - DNS接口（解析）  
    - HTTP/HTTPS 监听端口/强制HTTPS/HSTS/HTTP2/HTTP3  
    - SSL 协议/SSL 加密套件/优先服务端加密套件/OCSP  
    - 回源协议/回源HTTP端口/回源HTTPS端口/回源超时  
    - IPv6/Gzip/WebSocket/上传大小限制  
    - 数据实时发送/数据实时返回  
    - 源站请求头（Headers）  
    - 回源负载方式
  - 适用范围：全局 / 分组（需选分组）
  - 用户(Admin 必选)
  - 设置值类型：bool/number/select/multi/lines/region/headers/text
- 删除：逐条删除；批量删除逐条调用

**C# 重构要求**
- 默认设置保存逻辑保持 scope 与 user 关系一致（scope=global 时 scope_id=用户）

#### C6. DNS 接口（网站侧）
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/components/DnsApiTab.vue`
- 顶部操作：新增 DNS 接口 / 批量删除
- 列表列：ID / 名称 / DNS / 备注 / 操作
- 新增/编辑弹窗字段：名称/备注/DNS 类型/认证字段（随 provider 动态变化）
- DNS 类型来源：`/dns/providers/types` + `/dnsapi/types`

#### C7. 解析检测
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/Resolve.vue`
- 操作：开始检测 / 域名搜索
- 列表列：ID / 网站ID / 域名 / CNAME / 解析状态 / DNS接口 / 任务状态
- 解析状态：检测中/正常/异常/未检测（异常 hover 显示解析结果）
- 注意：`dns_api` 与 `task_status` 当前为前端占位字段

#### C8. 网站管理页 `/website/manage`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/Manage.vue` + `web/admin/src/components/manage/*`
- Tab：基本配置 / 回源配置 / HTTPS / 安全 / 缓存 / 访问控制 / 高级设置
- 保存策略：开关 change 或输入框 blur 即保存（自动去重缓存规则/headers）

**C# 重构要求**
- 保持“自动保存”行为，不做额外确认弹窗

**基本配置**
- 站点状态、CNAME、套餐到期、创建/更新时间（只读）
- 套餐选择（提示：更换套餐不改变已落地 CNAME/线路组）
- 站点分组（多选）
- 域名（空格分隔，含 Punycode 提示）
- HTTP 开关 / 监听端口
- 源站列表（地址/权重/启用/删除/新增）
- 条件源站（匹配项/运算符/值/源站）

**回源配置**
- 回源协议 / 回源端口 / 回源Host（follow/domain/custom）
- 回源超时 / 连接超时

**HTTPS 配置**
- 开关 / 证书选择 / 申请证书
- 监听端口 / 强制HTTPS（跳转端口）/ HSTS / HTTP2 / HTTP3 / OCSP
- SSL 策略（兼容/现代/自定义；自定义协议/加密套件）

**安全配置**
- CC 默认规则 / 自动防护（QPS阈值+切换规则）
- 自定义规则（匹配器/动作/模式/备注/启用）
- 搜索引擎爬虫策略
- 黑白名单时间 / IP 黑白名单
- Cookie 域名 / 屏蔽透明代理 / 区域屏蔽

**缓存配置**
- 规则表 + 快速添加（首页/全站/静态/视频/WordPress）
- 规则字段同“批量设置 > 缓存规则”

**访问控制**
- ACL 选择
- 防盗链（范围/值/允许空来源/额外域名）
- CORS（允许来源/方法/请求头/响应头/凭证/缓存时长）

**高级设置**
- 上传大小限制 / Gzip / WebSocket
- 搜索引擎回源 + IP
- URL 重定向（规则列表）
- 源站请求头 / CDN 响应头
- 访问日志（请求头/响应头/请求体/大小限制）
- 源站证书 / 数据实时识别 / 数据实时发送 / 默认站点 / L2配置

#### C9. 网站分组 `/website/groups`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/Groups.vue`
- 操作：添加分组 / 批量删除
- 搜索：分组名称关键字
- 列表列：ID / 分组名称 / 备注 / 操作（编辑/删除）

#### C10. 证书管理 `/website/certs`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/Certs.vue` + `web/admin/src/views/website/CertEditPopup.vue`
- Tab：证书列表 / 默认设置 / DNS 接口

**证书列表**
- 顶部操作：添加证书 / 重新申请 / 更多操作（启用/禁用/开启续签/关闭续签/删除/强制禁用/下载）
- 搜索：字段（名称/域名/类型/全部）+ 关键字
- 列表列：ID / 用户(Admin) / 名称 / 类型 / 域名 / 创建时间 / 到期时间 / 自动续签 / 状态 / 失败原因 / 操作
- 状态显示：DNS验证中/待签发/签发中/已签发/失败；失败可查看错误详情
- 批量下载：逐条下载

**默认设置**
- Admin：先选用户，再配置证书类型 + DNS 接口
- User：仅配置证书类型 + DNS 接口
- 说明：DNS 接口仅用于证书申请（DNS 验证），与 CNAME 解析无关

**证书编辑弹窗**
- 单个证书：
  - 用户(Admin) / 名称 / 备注 / 类型（上传/ZeroSSL/Let’s Encrypt/BuyPass/Google）
  - 上传类型：证书内容 + 私钥内容
  - 非上传类型：域名（空格分隔）+ DNS 接口（0=HTTP验证）
- 批量申请：
  - 用户(Admin) / 类型 / 域名（每行一个）/ DNS 接口（0=HTTP验证）
  - 含泛域名必须选择 DNS 接口
- 泛证书申请：
  - 用户(Admin) / 类型 / 域名（必须 `*.` 开头）/ DNS 接口
  - DNS=0 时显示 TXT 验证信息 + 验证按钮

#### C11. 规则管理 `/website/rules`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/Rules.vue` + `rules/*`
- Tab：CC 规则 / ACL 规则

**CC 规则**
- 规则组：
  - 列表列：ID / 用户(Admin) / 名称 / 系统规则 / 显示 / 状态 / 排序 / 创建时间 / 操作
  - 规则组编辑：类型(系统/用户)/用户(Admin)/名称/备注/规则列表/是否显示/指定显示用户/排序
  - 规则列表：匹配器 / 过滤器1 / 过滤器2 / 动作（拉黑/记录）/ 模式（继续/停止）/ 启用 / 排序
- 匹配器：
  - 列表列：ID / 用户(Admin) / 名称 / 系统规则 / 状态 / 创建时间 / 操作
  - 规则项：匹配项 + 操作符 + 匹配值；支持上下移动
- 过滤器：
  - 列表列：ID / 用户(Admin) / 名称 / 系统规则 / 类型 / 状态 / 创建时间 / 操作
  - 类型：请求频率/无感验证/5秒盾/点击/点击(简)/滑块/滑块(简)/验证码/旋转图/302/URL鉴权

**ACL 规则**
- 列表列：ID / 用户(Admin) / 名称 / 备注 / 状态 / 操作
- 编辑字段：用户(Admin)/名称/备注/启用/默认行为(允许/拒绝)
- 默认拒绝可选 403 或 URL 跳转
- 规则列表：匹配条件（多条件且关系）+ 行为（允许/拒绝）

#### C12. 刷新预热 `/website/purge`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/Purge.vue`
- Tab：刷新预热 / 操作记录
- 刷新预热：类型（刷新URL/刷新目录/预热）+ URL 列表 + 提交
- 显示每日限额与剩余次数（`/tasks/usage`）
- 操作记录：列表筛选 + 重新提交（单条/批量）

#### C13. 统计 `/website/statistics`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/Statistics.vue`
- Tab：基础数据（带宽/流量/QPS）/ 质量监控（命中率/4xx/5xx）/ 回源监控 / 数据排行
- 数据排行：类型（域名/URL/耗时/IP/国家/省份/来源）+ 时间范围（10min/30min/1h/custom）+ 关键字

#### C14. 访问日志 `/website/access_logs`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/AccessLogs.vue`
- 基础筛选：域名/关键字/下载/高级搜索
- 高级筛选：时间范围/域名匹配/客户IP/URI+匹配方式/方法/状态码/状态码范围/端口/节点ID/节点IP/协议/缓存状态/来源/UA/SSL协议/SSL套件
- 列表列：时间/域名/协议/方法/URI/状态码/客户IP/字节/耗时/回源耗时/回源地址/缓存状态/来源/UA/节点ID/节点IP/SSL协议/SSL套件

#### C15. 封禁日志 `/website/block_logs`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/BlockLogs.vue`
- Tab：当前封禁 / 统计 / 历史记录
- 当前封禁：批量解封/解封网站/导出当前（前端占位）+ 类型(IP/网站ID)搜索
- 历史记录：按 IP/网站ID/时间范围筛选

#### C16. 网站监控 `/website/monitor`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/website/Monitor.vue`
- 状态：页面显示“开发中”


### D. 转发管理（Admin/User）

#### D1. 转发列表 `/forward/list`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/forward/List.vue` + `web/admin/src/views/forward/list/ForwardTable.vue`
- 顶部操作：添加转发 / 批量修改 / 更多操作（启用/禁用/删除）
- 搜索：字段（全部/监听端口/源站/CNAME/用户）+ 关键字 + 高级搜索（状态）
- 列表列：选择 / ID / 用户 / 监听端口 / 源站 / 套餐 / 分组 / 区域(线路组) / CNAME / 状态 / 时间 / 操作

#### D2. 转发新增/编辑弹窗
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/forward/list/ForwardEditDialog.vue`
- 模式：单个 / 批量（新增时）
- 单个字段：用户(Admin)/用户套餐/监听端口/源站地址:端口
- 批量字段：数据（`监听端口|IP|回源端口` 每行）+ 忽略错误
- 展开更多：所属分组（多选）/ 备注

#### D3. 转发分组 `/forward/groups`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/forward/Groups.vue`
- 操作：添加分组 / 批量删除
- 列表列：ID / 分组名称 / 备注 / 操作（编辑/删除）

#### D4. 转发默认设置 `/forward/default`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/forward/Default.vue`
- 设置项：开启 proxy_protocol / 监听协议 / 负载方式
- 生效范围：全局 / 转发分组
- 列表列：设置项 / 设置值 / 生效范围 / 分组 / 操作（删除）

#### D5. 转发实时监控 `/forward/monitor`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/forward/Monitor.vue`
- Tab：带宽流量 / 端口排行
- 带宽流量：端口检索 + 时间范围(1h/6h/24h) + 刷新；图表（带宽/流量）
- 端口排行：端口/连接数/累计流量


### E. 系统管理（Admin）

#### E1. 系统公告 `/system/announcements`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/system/Announcements.vue`
- 列表列：ID / 标题 / 创建时间 / 状态 / 操作
- 新增/编辑：标题/内容/是否显示/红色/加粗

#### E2. 系统消息 `/system/messages`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/system/Messages.vue`
- 筛选：类型（套餐到期/流量超限/连接数超限/带宽超限/规则切换/证书到期）+ 关键字
- 列表列：ID / 类型 / 标题 / 网站ID / 创建时间 / 操作（详情）
- 详情：标题 + 邮件内容 + 手机内容

#### E3. 系统日志 `/system/logs`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/system/Logs.vue`
- Tab：登录日志 / 操作日志 / 备份日志 / 发信日志
- 时间范围：今天/近7天/近30天/自定义
- 登录日志列：用户ID / IP / 地理位置 / 时间 / 状态
- 操作日志列：用户ID / 类别 / 对象 / 动作 / 内容 / IP / 地理位置 / 时间
- 备份日志列：备份时间 / 完成时间 / 状态 / 结果
- 发信日志列：用户ID / 消息ID / 标题 / 媒介 / 失败次数 / 状态 / 原因 / 发送时间

#### E4. 系统任务 `/system/tasks`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/system/Tasks.vue`
- 筛选：类型 + 关键字
- 列表列：ID/优先级/名称/类型/资源ID/依赖/开始时间/耗时/状态/失败次数
- 操作：重试

#### E5. 版本与升级 `/system/upgrade`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/system/Upgrade.vue`
- 版本管理：上传版本包、设置稳定版、灰度比例
- 同步升级：选择版本 + 区域选择 + 节点列表升级；轮询升级状态


### F. 系统设置（Admin）

#### F1. 系统配置 `/settings/system`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/settings/System.vue` + `components/*`
- Tab：系统配置 / 数据清理 / 用户相关 / 通知配置 / 其它配置

**系统配置**
- 基本信息（系统名称/控制台标题/底部信息/图标上传/绑定主机）
- 套餐控制（过期关闭/流量超限关闭/允许升降级）
- 维护升级（维护模式+公告/Agent 自动升级）

**数据清理**
- 清理周期：缓存解封/登录日志/操作日志/访问日志/节点监控/流量历史/黑名单
- 备份设置：频率/保留/目录

**用户相关**
- Session 生命周期
- 登录域名限制（用户端/管理员端）
- 登录方式（邮箱/短信）
- 注册开关
- 邮件模板：注册成功/找回密码/邮箱验证码
- 短信模板：验证码模板ID/模板内容

**通知配置**
- 通知时间段（全天/自定义）
- 事件模板：流量超限/流量预警/套餐到期/到期预警/CC切换/带宽超限/连接数超限/证书到期/证书预警/二次验证
- 每事件字段：开关/方式/连续次数/间隔时间/模板（邮件/短信）

**其它配置**
- Master IP Header / 记录修复 / DNS 记录保护
- 同步范围 / 每次同步最大站点数 / 资源排行大小
- HTTP 代理
- API Key 开关/密钥/白名单/重置
- TCP 流量系数
- 默认 HTTPS 证书内容/私钥

#### F2. 节点监控配置 `/settings/monitor`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/settings/Monitor.vue`
- 通知时间段/通知方式/通知类型/邮箱/手机
- 带宽超限次数/高负载自动切换/恢复时间/监控API/检测间隔/失败次数/失败率

#### F3. 旧版全局配置 `/settings/global`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/settings/Global.vue`
- 核心设置：worker_processes/worker_connections/shutdown_timeout
- WAF 与防火墙：拦截模式/黑名单时长/CC阈值/动作
- HTTPS 与协议：HTTP2/HTTP3/HSTS/SSL加密套件


### G. 用户管理（Admin）

#### G1. 用户列表 `/users/list`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/users/List.vue`
- 搜索：用户名/邮箱/手机号
- 列表列：ID / 用户名 / 邮箱 / 手机 / QQ / 余额 / 状态 / 备注 / 操作
- 操作：编辑 / 切换登录 / 重置刷新次数 / 删除

#### G2. 用户编辑弹窗
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/users/UserEditPopup.vue`
- Tab：基础信息 / 实名信息 / 登录安全
- 基础信息：邮箱/用户名/备注/手机/QQ/密码/分组/启用
- 实名信息：姓名/身份证/公司/信用代码 + 二次验证设置
- 登录安全：登录验证码方式 + 登录白名单 IP


### H. 账户中心（User）

#### H1. 个人资料 `/account/profile`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/account/Profile.vue`
- 基本信息：用户名/余额/QQ(可编辑)/注册时间/密码修改
- 实名认证：个人/企业
- 绑定信息：手机/邮箱（验证码按钮为占位）
- 安全设置：IP 白名单/登录验证码方式

#### H2. API Key `/account/api_key`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/account/ApiKey.vue`
- 显示 API Key/Secret + IP 白名单；支持重置 Secret

#### H3. 账单 `/account/bills`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/account/Bills.vue`
- 列表列：ID/类型/备注/金额/实付/更多/支付方式/订单号/创建时间/已付款

#### H4. 充值 `/account/recharge`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/account/Recharge.vue`
- 字段：金额/备注

#### H5. 消息 `/account/messages`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/account/Messages.vue`
- 列表列：类型/标题/状态/网站ID/创建时间/操作（详情）
- 详情：邮件内容/短信内容 + 置为已读

#### H6. 消息订阅 `/account/subscribe`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/account/Subscribe.vue`
- 列表列：消息类型 + 手机/邮件勾选

#### H7. 操作日志 `/account/logs`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/account/Logs.vue`
- 列表列：ID/操作/内容/IP/时间


### I. 财务管理（Admin）

#### I1. 订单与充值 `/finance/orders`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/finance/Orders.vue`
- 列表列：订单ID/用户ID/金额/状态/支付方式/订单号/类型/备注/创建时间
- 手动充值：用户ID/金额/备注


### J. 仪表盘 `/dashboard`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/dashboard/index.vue`
- Admin：运营数据 + 网络概览 + 监控趋势 + TOP10 + 系统状态 + 授权信息
- User：用户信息卡 + 系统公告 + 套餐流量 + 资源统计


### K. 旧日志页面（占位）

#### K1. 登录日志 `/logs/login`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/logs/Login.vue`
- 状态：空页面（仅空表格）

#### K2. 操作日志 `/logs/operation`
**现有行为(前端 Vue)**
- 数据源：`web/admin/src/views/logs/Operation.vue`
- 列表列：ID/用户/动作/描述/IP/时间
