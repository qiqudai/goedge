# cnn.net 剩余功能全量补齐设计文档（实施版）

> 目标：将 `cnn.net` 补齐为可替代父项目 `cdn-system`（Go + OpenResty + Lua）的 .NET CDN 系统。  
> 范围：L7/L4 数据面、规则引擎、动态扩展、控制面一致性、稳定性、容量与性能、运维与长期演进。  
> 读者：架构、后端、边缘、SRE、测试、运维。

---

## 1. 背景与目标

### 1.1 已有能力（当前代码现状）
- 控制面基础已具备：`/ws/agent`、`edge_config` 下发、`task_dispatch/task_ack`、日志上报。
- API 管理面覆盖面较高：Admin/User/Agent 多数控制器与服务已经存在。
- Agent 已切换为 ASP.NET + YARP + OutputCache，不再依赖 OpenResty 进程启动。

### 1.2 关键缺口（必须补齐）
- YARP 已支持基于 `EdgeConfig` 的动态路由/集群热更新，且已落地 `ip_hash`（ClientIpHash）策略；仍需继续补齐主动健康检查细节。
- ACL/CC/WAF/Hotlink/CORS/Cookie/区域封禁等仍缺少完整 C# 数据面执行链。
- 证书热更新、SNI、HTTPS 细粒度策略未完整落地到 Kestrel/YARP。
- Lua 规则体系尚未等价迁移到 C# 规则引擎。
- 动态扩展（复杂规则动态 DLL）已具备基础安全可控机制（签名/哈希校验、目录隔离、尺寸上限、白名单、熔断与拒绝审计）；后续仍可增强进程级沙箱隔离。

### 1.3 总目标
- 功能目标：达到父项目可用功能的 100% 等价或可替代语义。
- 可用性目标：单节点 99.95%，集群 99.99%。
- 性能目标：
  - 控制面配置生效 P95 < 5s（站点规模 <= 2 万域名）。
  - 数据面额外处理开销 P95 < 5ms（不含回源）。
  - 规则链在 95% 请求路径中不产生额外 GC 压力峰值。
- 安全目标：动态扩展具备签名校验、沙箱隔离、故障熔断、快速回滚。

---

## 2. 设计原则

- 必须保持现有语义兼容（字段、默认值、优先级、状态机）。
- 必须支持灰度与回滚（节点级、区域级、站点级）。
- 必须让“配置生效路径”原子化（文件写入 + 内存切换 + 代理热更）。
- 必须先保证稳定，再做高阶优化。
- 必须“可观测优先”：每个功能都有指标、日志、追踪与告警。
- 必须“性能预算先行”：每条中间件链有 CPU/内存预算上限。

---

## 3. 总体架构（补齐后）

### 3.1 控制面
- Cnn.Api：配置生成、任务编排、节点状态、版本管理。
- WS 主通道：`edge_config`、`cache_config`、`task_dispatch`、`heartbeat_ack`。
- HTTP 兜底：`/api/v1/agent/config`、`/api/v1/agent/tasks`。

### 3.2 数据面（Agent）
- Kestrel + YARP：域名路由、回源代理、LB、HTTPS。
- 中间件链：
  1) Node 状态门禁
  2) 安全链（ACL/WAF/CC/地域/UA 等）
  3) 缓存决策链
  4) 代理执行
  5) 响应头与观测补充
- 规则运行时：基于 `EdgeConfigStore.Current` 的无锁读快照。

### 3.3 扩展层
- 内置规则引擎（默认）
- 插件规则引擎（可选）：动态 DLL + 签名 + 沙箱 + 熔断

---

## 4. 分层与模块边界

### 4.1 Agent 模块拆分
- `RuntimeConfig`：配置持久化、版本对比、原子切换、回滚。
- `ProxyConfigProvider`：把 `EdgeConfig` 转为 YARP `Routes/Clusters`。
- `TlsManager`：证书/SNI/HTTPS 策略、热更新与缓存。
- `SecurityPipeline`：ACL/WAF/CC/Hotlink/CORS/Cookie 等。
- `CachePipeline`：缓存 profile/规则、purge/preheat 统一 key。
- `PluginHost`：外部规则插件生命周期管理。
- `Telemetry`：指标、日志、Tracing、告警上下文。

### 4.2 API 模块补齐
- `EdgeConfigService`：继续作为唯一配置编译入口。
- `ConfigVersionService`：统一版本 bump 策略。
- `TaskService`：强一致任务状态机与重试策略。
- `AgentNodeService`：节点在线状态、L2 健康、同步动作。

---

## 5. EdgeConfig 统一语义与限制

### 5.1 版本策略
- `version` 必须单调递增（建议：逻辑版本 + hash 双因子）。
- Agent 仅接受更高版本；允许强制重放 `force=true`。
- 每次生效写 `applied_version`（本地文件 + 内存）。

### 5.2 原子生效
1. 校验 schema（字段合法、边界值、引用完整）。
2. 生成运行时对象（路由表、规则索引、证书缓存）。
3. 预热（regex 编译、CIDR 树构建、域名索引构建）。
4. 交换引用（Interlocked/Volatile 原子替换）。
5. 成功后写 `current`，失败回退 `last_good`。
- 已补充 Agent 侧代理运行时字段校验：`origin_protocol/origin_http_port/origin_https_port/proxy_*_timeout/proxy_http_version` 不合法会在应用前直接拒绝，避免坏配置进入数据面。

### 5.3 限制建议
- 单节点域名上限（初始）：2 万（可调）。
- 单域 ACL 规则上限：500。
- 单域 CC 匹配上限：500。
- 单条正则最大长度：4KB。
- 配置 JSON 最大大小：20MB（超过改分片或压缩下发）。

---

## 6. YARP 动态路由补齐设计

### 6.1 关键实现点
- 实现 `IProxyConfigProvider`，从 `EdgeConfigStore.Current` 动态构建路由。
- `RouteId` 规范：`{domain}:{port}:{protocol}`。
- `ClusterId` 与 `upstream_key` 对齐。
- 在配置切换时触发 YARP `IChangeToken`，无重启热生效。

### 6.2 约束与技巧
- 路由构建必须 O(n)；禁止嵌套高复杂度查找。
- 预建 `Dictionary<string, DomainRoutePlan>`，请求路径只做 O(1) 查询。
- host 比较统一小写并规整 punycode。
- 避免每请求动态解析 JSON，全部转成强类型 runtime 对象。

### 6.3 负载均衡语义映射
- `ip_hash` -> ConsistentHash policy（自定义）
- `least_conn` -> LeastRequests
- `round_robin` -> RoundRobin
- `random` -> PowerOfTwoChoices 或 Random
- 已实现加权轮询：当 `round_robin` 且上游 `targets.weight` 非均匀时自动切换 `WeightedRoundRobin`，`weight` 在数据面真实参与目标选择。

### 6.4 失败切换
- 上游失败重试遵循 `proxy_next_upstream` 语义。
- 对每个 cluster 维护失败滑动窗口，触发临时摘除。
- 与健康检查协同：主动探测 + 被动熔断。
- 已实现域名级 `proxy_read_timeout/proxy_send_timeout/proxy_http_version/proxy_ssl_protocols/upstream_keepalive_conn/origin_protocol(origin_*_port)` 到 YARP `HttpRequest/HttpClient/Destination` 的编译映射，并纳入 cluster 去重键，避免不同域名误复用同一上游配置。
- 已实现自定义 `IForwarderHttpClientFactory`，把 `proxy_connect_timeout/upstream_keepalive_timeout` 元数据转为 `SocketsHttpHandler.ConnectTimeout/PooledConnectionIdleTimeout`，连接行为可随配置热更新生效。

---

## 7. HTTPS/TLS 补齐设计

### 7.1 证书策略
- 数据来源：
  - 站点证书（domain 绑定）
  - fallback 证书（全局兜底）
- 内存缓存：`ConcurrentDictionary<string, X509Certificate2>`。
- 热更新策略：按版本增量替换，不中断连接。

### 7.2 SNI 回调
- 使用 Kestrel SNI 回调按 Host 取证书。
- 证书不存在时回退 fallback。
- 证书加载失败不可阻断主链路，降级 fallback 并告警。

### 7.3 HTTPS 参数落地
- `https_force/hsts/http2/http3/ssl_protocols/ssl_ciphers` 必须支持。
- 初期建议：HTTP/3 分阶段，先灰度到低流量节点。
- 已实现 `https_ssl_protocols/https_ocsp/https_ssl_ciphers` 的 Agent 运行时编译与 Kestrel TLS 回调应用（当前为单监听器全局策略汇总模式）。

### 7.4 性能注意
- `X509Certificate2` 对象避免反复加载与释放。
- OCSP、证书链校验应异步预热，避免请求路径阻塞。
- 已实现证书热更新增量复用与旧证书淘汰释放，降低长期运行的内存/句柄累积风险。

---

## 8. 安全链（ACL/WAF/CC）补齐设计

### 8.1 执行顺序（建议固定）
1. 白名单短路放行
2. 黑名单与 ACL
3. 地域封禁
4. WAF 语法/特征检测
5. CC 限速与挑战动作
6. 业务状态限制（套餐到期、连接超限、站点锁定）

### 8.2 ACL
- 支持 IP/CIDR/条件组合。
- 数据结构：
  - 精确 IP：HashSet
  - CIDR：前缀树（Radix Trie）
- 默认动作：`allow/deny/block`，需兼容旧字段。

### 8.3 WAF
- 规则分层：静态规则 + 配置规则 + 插件规则。
- 语义兼容字段：`mode/policy/cc/access_control/syntactic`。
- 字符串匹配尽量预编译（Aho-Corasick/Regex cache）。

### 8.4 CC
- 维度：IP、URI、Host、UA（按规则定义）。
- 存储：本地内存计数 + 可选 Redis（集群一致模式）。
- 算法：令牌桶 + 滑动窗口二选一，规则可配置。
- 动作：`allow/block/limit_rate/challenge` 等兼容映射。

### 8.5 关键限制
- 单请求规则评估预算：< 2ms（P95）。
- 正则总数上限（单域）：1000。
- 防止 ReDoS：正则超时 + 编译期审计 + 白名单表达式。

### 8.6 高峰技巧
- 只构建一次规则索引，热更新替换引用。
- 计数结构分片（shard）降低锁竞争。
- 热 key（高频 IP）使用局部缓存 + 批量回写。

---

## 9. Hotlink/CORS/Cookie/Header 等站点功能补齐

### 9.1 Hotlink
- 校验 Referer/Origin，支持空 Referer 策略。
- 对静态资源优先执行，避免动态路径误伤。

### 9.2 CORS
- 预检请求单独路径处理（OPTIONS 快速返回）。
- `Access-Control-*` 由配置严格输出，禁止宽松默认。

### 9.3 Cookie 与响应头
- 支持追加、覆盖、删除策略。
- 安全头（HSTS/XFO/CSP）需可配但提供推荐模板。

### 9.4 请求头透传
- 明确 hop-by-hop headers 黑名单，避免协议问题。
- 已落地站点级请求控制：`enable_websocket=false` 时拦截 Upgrade 请求；`body_limit` 超限返回 `413` 并尝试下发运行时最大请求体限制。
- 已落地 `limit_rate` 响应限速（字节/秒）写流包装器，避免单请求长响应挤占出口带宽。

---

## 10. 缓存体系完善

### 10.1 统一 key 规则
- `cache/purge/preheat` 必须使用同一 key builder。
- 支持 `query_mode` + 兼容旧字段映射。

### 10.2 存储层
- L1：本地文件缓存（当前已有）
- L2（可选）：Redis 元数据索引（清理与统计）
- 清理策略：LRU + TTL + 磁盘水位线

### 10.3 峰值瓶颈
- 磁盘 IOPS：大量小文件写入导致抖动。
- 解决：
  - 分桶目录（hash 前缀）
  - 异步批量写 meta
  - 独立缓存盘（NVMe）

### 10.4 限制建议
- 单节点缓存目录 inode 预估必须提前规划。
- 清理线程 CPU 占比上限 15%。

---

## 11. L4 Stream（TCP/UDP）补齐

### 11.1 目标
- 覆盖父项目 `streams` 能力。
- 支持端口监听、上游集群、健康检查、限速。

### 11.2 实现方案
- .NET 内实现独立 Stream Proxy Service（与 YARP 解耦）。
- 按配置动态绑定端口（注意端口冲突检测与最小权限）。
- 使用 `SocketAsyncEventArgs` + 池化 buffer。

### 11.3 风险与限制
- 高并发连接下 GC 与 socket 资源泄漏风险高。
- 必须引入连接上限、慢连接淘汰、空闲超时。

---

## 12. 动态 DLL 扩展（复杂规则）

### 12.1 目标
- 对复杂、变化快、业务定制规则提供“热插拔”。
- 不污染主进程稳定性。

### 12.2 插件接口规范（建议）
- `IRulePlugin`
  - `string Name`
  - `string Version`
  - `Task Initialize(PluginContext ctx)`
  - `RuleDecision Evaluate(RequestContext ctx)`
  - `Task Dispose()`

### 12.3 安全机制
- 必须签名校验（证书白名单）。
- 必须插件清单（版本、hash、依赖）。
- 必须隔离加载上下文（`AssemblyLoadContext`）。
- 必须超时控制与熔断（单次执行超时、错误率熔断）。
- 建议进程级隔离（高风险插件走 sidecar RPC）。

### 12.4 性能机制
- 只允许纯计算型快速逻辑进入请求链。
- 插件不得直接做网络 I/O；如需外部数据必须异步预拉取。
- 规则执行预算：P95 < 1ms。

### 12.5 回滚机制
- 插件新版本先灰度节点。
- 任一错误率超阈值自动回退到上一版本。
- 插件失效时主链路使用默认安全策略继续服务。

---

## 13. 控制面一致性与任务系统

### 13.1 任务状态机
- `waiting -> running -> success|fail`
- `fail -> retrying -> running`
- 每次状态跃迁必须持久化并记录时间戳。

### 13.2 幂等与重试
- `task_id + node_id` 唯一键。
- ACK 重复上报必须幂等。
- 重试采用指数退避，带最大重试次数与死信标记。

### 13.3 配置同步一致性
- 下发前写版本，生效后 ack 应回带版本。
- Server 保存节点 `last_applied_version`。
- 连续不一致触发全量重发并告警。

---

## 14. 观测体系（必须先落地）

### 14.1 指标
- 请求：QPS、P50/P95/P99、4xx/5xx、回源错误率。
- 规则：ACL 命中率、WAF 命中率、CC 命中率、挑战成功率。
- 缓存：命中率、写入延迟、清理耗时、磁盘使用率。
- 连接：活跃连接、连接建立失败率、L4 并发。
- 控制面：配置应用时延、任务积压、WS 重连频率。

### 14.2 日志
- 访问日志：结构化 JSON。
- 安全日志：命中规则 ID、动作、原因。
- 配置日志：版本、耗时、成功/失败、回滚原因。
- 插件日志：加载/卸载/异常/熔断状态。

### 14.3 Trace
- 网关入口到回源全链路 trace id。
- 控制面配置下发与数据面生效关联 trace。

---

## 15. 性能与峰值容量设计

### 15.1 估算模型（建议）
- 带宽峰值：`B_peak = QPS_peak * AvgRespBytes * 8`
- CPU 粗估：
  - 纯转发：0.3 ~ 0.8 core / 10k RPS
  - 启用完整规则：1.0 ~ 2.5 core / 10k RPS
- 内存：
  - 基础进程：0.5 ~ 1.5 GB
  - 每 1 万域名路由：+0.3 ~ 1.0 GB
  - 规则索引：按规则量线性增长

### 15.2 单节点建议（初始）
- 中小节点：8C/16G，1 x NVMe（缓存盘）
- 主力节点：16C/32G，2 x NVMe（系统/缓存分盘）
- 高峰节点：32C/64G，25G 网卡，独立日志盘

### 15.3 硬件瓶颈优先级
1. 网卡带宽
2. 磁盘 IOPS（缓存）
3. CPU（规则链）
4. 内存（大规模路由与规则）

### 15.4 压测目标
- L7：
  - 10k/50k/100k RPS 分级压测
  - 规则开启/关闭对比
- L4：
  - 10万并发连接稳态 30 分钟
- 配置热更：
  - 2 万域名配置切换 P95 < 5s

---

## 16. 长期稳定运行设计

### 16.1 进程级稳定
- 启用 `Server GC` + GC 指标监控。
- 大对象池化（buffers、regex、序列化对象）。
- 禁止请求路径频繁分配大对象。

### 16.2 故障隔离
- 插件故障不应拖垮主链路。
- 外部依赖失败（DB/Redis/API）需有降级策略。
- 回源不可用时必须触发熔断与快速失败。

### 16.3 灰度策略
- 三层灰度：节点 -> 节点组 -> 全量。
- 强制留后门回滚：一键切回 last_good_config。
- 每次灰度必须带自动验收脚本。

### 16.4 运维策略
- 周期巡检：证书、磁盘、连接、慢请求。
- 变更窗口：高峰期禁止全量配置重载。
- 备份：关键配置与任务状态定时快照。

---

## 17. 安全与合规

- Agent 与 API 通信必须支持 TLS + token 轮换。
- 敏感字段（密钥、证书私钥）必须加密存储与最小暴露。
- 管理接口必须 RBAC + 审计日志 + TraceId。
- 插件包必须来源可信仓、签名校验通过才能启用。

---

## 18. 测试策略

### 18.1 单元测试
- 配置解析、规则匹配、缓存 key、状态机迁移。
- 每个兼容字段都要有回归用例。

### 18.2 集成测试
- WS 下发 -> Agent 生效 -> 请求行为验证。
- 证书热更新、路由热更新不中断。
- 任务重试、断线恢复、幂等 ACK。

### 18.3 压力与稳定性测试
- 72 小时 soak test。
- 峰值压测 + 故障注入（网络抖动、磁盘变慢、上游超时）。

### 18.4 安全测试
- 正则 ReDoS、Header 注入、路径穿越、插件越权。

---

## 19. 分阶段实施路线图（建议 6 阶段）

### 阶段 A：核心动态路由与配置原子生效
- 交付：`IProxyConfigProvider`、EdgeConfig 原子切换、回滚机制。
- 验收：2 万域名热切换成功，流量无中断。

### 阶段 B：HTTPS 与证书体系
- 交付：SNI、fallback、证书热更新、HTTPS 策略落地。
- 验收：证书更新不重启，不丢连接。

### 阶段 C：安全链一期（ACL/WAF 基础）
- 交付：ACL、黑白名单、地域封禁、基础 WAF。
- 验收：规则命中准确率 >= 99.9%。

### 阶段 D：CC 与挑战动作
- 交付：CC 计数引擎、限速/阻断/挑战、日志闭环。
- 验收：压测下误封率与漏封率达标。

### 阶段 E：L4 Stream + 高级缓存
- 交付：TCP/UDP 代理、缓存清理与容量治理。
- 验收：长稳压测通过。

### 阶段 F：插件体系 + 全量灰度收敛
- 交付：动态 DLL、签名校验、熔断回滚、治理平台。
- 验收：插件故障不影响主链路 SLA。

---

## 20. 功能点限制与关键技巧清单（便于逐步设计）

### 20.1 路由
- 限制：host/path 规则冲突需 deterministic。
- 技巧：统一排序与优先级规则，编译期检测冲突。

### 20.2 规则
- 限制：正则与复杂表达式必须可控。
- 技巧：预编译 + 超时 + 黑名单模式。

### 20.3 缓存
- 限制：purge 与 key 生成不一致会导致“清不掉”。
- 技巧：单一 KeyBuilder，全链路复用。

### 20.4 插件
- 限制：不可信插件会导致稳定性和安全风险。
- 技巧：签名、沙箱、熔断、灰度、回滚五件套必须齐全。

### 20.5 配置
- 限制：超大配置会阻塞更新。
- 技巧：增量下发、分块、预热编译、异步应用。

---

## 21. 上线门禁（Go/No-Go）

必须全部满足才可全量：
- 功能回归通过率 100%。
- 关键路径压测达标（RPS/延迟/错误率）。
- 72h 稳定性测试通过。
- 告警规则、应急预案、回滚演练完成。
- 生产灰度阶段无 P1/P0 事故。

---

## 22. 风险清单与应对

1. 路由热更抖动导致短时 5xx
- 应对：双缓冲配置 + 原子切换 + 失败回退。

2. 安全规则误封
- 应对：灰度发布 + 观测面板 + 一键降级到 monitor 模式。

3. 缓存雪崩
- 应对：TTL 抖动、单飞、分级回源保护。

4. 插件失控
- 应对：执行超时、熔断、自动卸载、回退上一版本。

5. 硬件瓶颈提前触发
- 应对：容量模型 + 自动扩缩容阈值 + 热点迁移。

---

## 23. 交付物清单（后续每阶段必须产出）

- 设计文档（本模板裁剪版）
- 配置 schema 与示例
- 实现 PR 与迁移脚本
- 压测报告
- 回归测试报告
- 灰度与回滚记录
- 运行手册（SRE）

---

## 24. 下一步建议（立即执行）

按优先级启动以下 4 个子文档（每个都可直接进开发）：
1. `YARP 动态路由与配置热更详细设计`
2. `安全链 ACL/WAF/CC 规则引擎详细设计`
3. `TLS/证书热更新与 SNI 详细设计`
4. `插件化规则执行（动态 DLL）详细设计`

> 本文已给出全量约束、关键实现点、性能和稳定性框架。后续按以上 4 个子文档逐项展开，即可直接进入编码与联调。

---

## 25. 调试日志与人工调试开关（新增强制项）

### 25.1 日志分级与用途
- `Error`：必须保留，生产常开。
- `Warning`：必须保留，生产常开。
- `Information`：建议保留核心路径（配置生效、任务状态、节点上下线）。
- `Debug`：默认关闭，仅短时开启。
- `Trace`：默认关闭，仅本地或压测环境。

### 25.2 开关体系（必须支持）
- 全局开关：`debug.enabled`
- 模块开关：
  - `debug.routing`
  - `debug.cache`
  - `debug.security.acl`
  - `debug.security.waf`
  - `debug.security.cc`
  - `debug.tls`
  - `debug.plugin`
  - `debug.ws`
  - `debug.task`
- 请求级开关（人工排障）：
  - 请求头 `X-Debug-Token`
  - 查询参数 `__debug=1`（仅内网白名单可用）
- 已实现请求级开关控制项：`allow_header_token`、`allow_query_flag`、`internal_ip_only`（默认仅允许 header token，query flag 默认关闭）。
- 已实现日志管线收敛基座：通道级回压策略（高优先级短时重试 + 低优先级快速丢弃）、按通道丢弃计数汇总日志。
- 已实现本地日志治理基座：统一日志查询服务（`from/to/channel/level/trace_id/node_id/host/status` 过滤 + 分页）与留存清理后台任务（按通道 retention + 磁盘高水位降级清理）。
- 已实现 API 权限策略增强：`path + method` 策略解析，`admin/user/agent` 接口统一走权限点模型校验。
- 已新增 `admin` 事件日志查询接口（`/api/v1/admin/logs/events`），支持按 `event_type/node_id/node_ip/trace_id/host/status/keyword` 与时间范围检索 `node_events`。
- 已新增 `user` 事件日志查询接口（`/api/v1/user/logs/events`），并按用户站点 host 范围进行数据过滤，避免跨租户日志可见性。
- 已将日志接口权限从粗粒度角色拆分为专属权限点：`log:read:security`（平台日志）与 `log:read:user`（用户侧日志路由）。
- 已增强动态插件安全策略：`MaxAssemblyBytes` 尺寸门限、`RestrictAssemblyToPluginDirectory` 目录隔离、`AllowedPluginNames` 白名单准入，并将插件加载成功/拒绝原因写入系统事件日志（`plugin_loaded` / `plugin_rejected`）。

### 25.3 人工调试日志（必须具备）
- 目标：允许工程师“对单个请求”打开详细链路日志，不影响全局性能。
- 必填字段：
  - `trace_id`
  - `request_id`
  - `host`
  - `uri`
  - `matched_route`
  - `matched_rules`（ACL/WAF/CC）
  - `cache_decision`
  - `upstream_target`
  - `latency_ms`
  - `debug_session_id`

### 25.4 开关生效与安全护栏
- 所有 debug 开关必须支持 TTL 自动过期（例如 10 分钟）。
- 生产环境必须要求二次认证（管理员权限 + 审计记录）。
- 调试日志必须脱敏：
  - IP 可部分脱敏（可配置）
  - `Authorization/Cookie/Token` 必须掩码
  - POST Body 默认不打印，除非显式白名单字段
- 调试开关状态必须写审计表（谁、何时、哪台节点、持续多久）。

### 25.5 性能与容量限制
- Debug 模式采样率必须可控：`debug.sample_rate`（默认 0.01）。
- 单节点调试日志写入速率上限：`debug.max_events_per_sec`（默认 200）。
- 超上限后自动降级为摘要日志，避免磁盘与 CPU 打满。

### 25.6 建议配置（可直接落地）

```json
{
  "debug": {
    "enabled": false,
    "ttl_seconds": 600,
    "sample_rate": 0.01,
    "max_events_per_sec": 200,
    "modules": {
      "routing": false,
      "cache": false,
      "security": {
        "acl": false,
        "waf": false,
        "cc": false
      },
      "tls": false,
      "plugin": false,
      "ws": false,
      "task": false
    },
    "request_debug": {
      "allow_header_token": true,
      "allow_query_flag": false,
      "internal_ip_only": true
    }
  }
}
```

### 25.7 日志样例（结构化）

```json
{
  "level": "Debug",
  "trace_id": "a1b2c3",
  "request_id": "req-001",
  "debug_session_id": "dbg-20260401-01",
  "host": "www.example.com",
  "uri": "/api/list?x=1",
  "matched_route": "www.example.com:443:https",
  "matched_rules": ["acl:allowlist", "waf:sql-injection"],
  "cache_decision": "MISS",
  "upstream_target": "10.0.1.20:8080",
  "status": 200,
  "latency_ms": 12
}
```

### 25.8 与上线门禁联动
- 若 `debug.enabled=true` 且持续超过阈值（例如 30 分钟），必须触发告警。
- 发布前检查项必须包含“生产调试开关全关闭”。
- 回滚脚本必须包含“清理调试开关与恢复采样率默认值”。
