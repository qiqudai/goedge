# YARP 动态路由与配置热更详细设计（实施级）

> 适用范围：`cnn.net/src/Cnn.Agent`  
> 目标：将当前静态 `LoadFromConfig("ReverseProxy")` 改为基于 `EdgeConfig` 的动态路由/集群热更新，并保证高并发下稳定、可回滚、可观测。

---

## 1. 目标与非目标

### 1.1 目标
- 基于 `EdgeConfigDto` 动态生成 YARP `Routes` 与 `Clusters`。
- 配置热更新必须无进程重启、无监听中断。
- 新配置切换必须原子化，失败自动回滚到 `last_good`。
- 保持与现有业务语义兼容：域名、上游、LB、Header、HTTPS Force。
- 支持 2 万域名规模下 P95 配置生效时延 < 5 秒。

### 1.2 非目标
- 本文不覆盖 ACL/WAF/CC 实现细节（仅定义前置挂载点）。
- 本文不覆盖 L4（TCP/UDP）代理。

---

## 2. 现状与问题

### 2.1 当前实现
- Agent 在启动时使用静态配置：
  - `builder.Services.AddReverseProxy().LoadFromConfig(builder.Configuration.GetSection("ReverseProxy"));`
- 这意味着 `edge_config` 下发到 Agent 后，无法驱动 YARP 路由动态变化。

### 2.2 风险
- 节点配置与控制面不一致，导致路由漂移。
- 新增域名、源站变更、策略切换不能即时生效。
- 无统一回滚点，故障处理依赖重启或手工改配置。

---

## 3. 总体方案

### 3.1 架构变更
- 引入 `IProxyConfigProvider` 自定义实现 `DynamicProxyConfigProvider`。
- 在收到 `edge_config` 并通过校验后，编译成 `ProxySnapshot`：
  - `IReadOnlyList<RouteConfig>`
  - `IReadOnlyList<ClusterConfig>`
  - `RoutingIndex`（自定义辅助索引）
  - `Meta`（version/hash/created_at）
- 使用 `CancellationChangeToken` 发布变更，YARP 自动切换到新快照。

### 3.2 原子切换策略
1. 接收新配置（WS 或 HTTP 兜底）。
2. 校验 + 编译（内存阶段，不触网，不替换当前配置）。
3. 若成功：单次原子交换 `currentSnapshot`。
4. 触发 ChangeToken。
5. 写入 `last_good_snapshot` 与 `applied_version`。
6. 失败：维持旧快照并记录错误。

### 3.3 双缓冲模型
- `current`: 线上生效版本。
- `staging`: 编译中的候选版本。
- `last_good`: 最近一次成功版本（可快速回滚）。

---

## 4. 模块与类设计

### 4.1 新增模块
- `Proxy/DynamicProxyConfigProvider.cs`
- `Proxy/ProxySnapshot.cs`
- `Proxy/EdgeConfigToYarpCompiler.cs`
- `Proxy/ProxyConfigValidator.cs`
- `Proxy/ProxyRollbackStore.cs`
- `Proxy/RouteTransformBuilder.cs`

### 4.2 关键接口

```csharp
public interface IEdgeProxyRuntime
{
    ProxyApplyResult TryApply(EdgeConfigDto config, bool force = false);
    ProxySnapshot GetCurrent();
    ProxySnapshot? GetLastGood();
    ProxyRollbackResult Rollback(long targetVersion);
}
```

```csharp
public sealed class DynamicProxyConfigProvider : IProxyConfigProvider
{
    public IProxyConfig GetConfig();
    public ProxyApplyResult TrySwap(ProxySnapshot next);
}
```

### 4.3 核心数据结构

```csharp
public sealed record ProxySnapshot(
    long Version,
    string Hash,
    DateTimeOffset CreatedAt,
    IReadOnlyList<RouteConfig> Routes,
    IReadOnlyList<ClusterConfig> Clusters,
    RoutingIndex Index,
    SnapshotMetrics Metrics);
```

```csharp
public sealed record SnapshotMetrics(
    int DomainCount,
    int RouteCount,
    int ClusterCount,
    int DestinationCount,
    TimeSpan CompileCost,
    long ManagedBytes);
```

---

## 5. EdgeConfig 到 YARP 的映射规则

### 5.1 Route 生成
- 每个 `domain` 生成一条或多条 Route（按 http/https 监听拆分）。
- RouteId 规则：`route:{domain}:{listen_port}:{protocol}`。
- Match：
  - `Hosts = [domain]`
  - `Path = /{**catch-all}`
- ClusterId：来自 `domain.upstream_key`。

### 5.2 Cluster 生成
- 每个 `upstream` 生成一个 Cluster。
- ClusterId 规则：`cluster:{upstream_id}`。
- Destinations：按 `targets` 映射。
- Address 规则：
  - 若 target 已带 scheme，直接使用。
  - 若无 scheme，按 `origin_protocol/http/https port` 推导。

### 5.3 负载均衡策略映射
- `ip_hash` -> `Hash`（自定义 policy provider）。
- `least_conn` -> `LeastRequests`。
- `round_robin` -> `RoundRobin`。
- `random` -> `PowerOfTwoChoices`。
- 未知值 -> `RoundRobin` 并打 warning。

### 5.4 Header 映射
- 请求头和响应头必须在编译阶段转为 Transform。
- Header 覆盖冲突策略：域名级 > 全局默认。
- Hop-by-hop 头（Connection/Keep-Alive/Upgrade 等）不允许用户透传。

### 5.5 HTTPS 强制跳转
- `https_force=true` 时，生成 HTTP Route 的 redirect transform。
- 保留 `https_redirect_port` 语义。

---

## 6. 校验与失败处理

### 6.1 校验分层
- Schema 校验：字段类型、必填、长度、范围。
- 语义校验：
  - `domain -> upstream_key` 必须可解析。
  - listen 端口必须合法。
  - destination 地址必须合法 URI。
- 冲突校验：重复 RouteId、冲突 Host/Port。

### 6.2 校验失败策略
- 任何失败都不得替换 current。
- 错误写入 `proxy_apply_error.log`（结构化）。
- 上报控制面（task_ack 或 node_sync error）。

### 6.3 部分失败策略
- 默认禁止部分生效（确保一致性）。
- 允许开发环境 `allow_partial_apply=true`（生产必须 false）。

---

## 7. 并发与锁模型

### 7.1 读写模型
- 请求路径只读 `IProxyConfig`，无锁。
- 配置更新路径使用单写锁（`SemaphoreSlim(1,1)`）。
- 快照切换采用 `Volatile.Write` + ChangeToken 触发。

### 7.2 竞争处理
- 当多次配置同时到达，按 `version` 严格序。
- 小版本覆盖大版本禁止。
- 相同版本重复下发直接 `skipped`。

### 7.3 连接行为
- YARP 切换后新请求走新配置。
- 旧连接保持到自然结束（不强制断连）。

---

## 8. 持久化与回滚

### 8.1 文件布局
- `edge-node/conf/cdn_config.json`：原始配置。
- `edge-node/conf/proxy_snapshot.current.json`：当前快照摘要。
- `edge-node/conf/proxy_snapshot.last_good.json`：最后可用快照摘要。
- `edge-node/conf/proxy_snapshot.history/{version}.json`：历史摘要（保留 N 份）。

### 8.2 回滚触发
- 自动：连续 N 次 apply 失败。
- 手动：`task_dispatch` 下发 `proxy_rollback`。

### 8.3 回滚原则
- 仅回滚到已验证成功版本。
- 回滚动作必须审计并上报。

---

## 9. 启动与热更新流程

### 9.1 启动流程
1. 读取 `cdn_config.json`。
2. 尝试编译并加载为 current。
3. 失败时尝试 `last_good`。
4. 若都失败，加载最小安全兜底（503 路由）并告警。

### 9.2 热更新流程（WS）
1. `AgentWsClient` 接收 `edge_config`。
2. 写盘备份（已存在逻辑）。
3. 调用 `IEdgeProxyRuntime.TryApply`。
4. 返回 `ok/skipped/fail` 给控制面。

### 9.3 兜底拉取流程（HTTP）
- 周期拉取比对 `version`。
- 当 WS 断线恢复后主动对账一次。

---

## 10. 观测与调试

### 10.1 指标
- `proxy_apply_total{status}`
- `proxy_apply_duration_ms`
- `proxy_snapshot_version`
- `proxy_routes_count`
- `proxy_clusters_count`
- `proxy_compile_alloc_bytes`

### 10.2 日志
- 关键事件：apply start/success/fail/rollback。
- 必含字段：`trace_id/version/hash/domain_count/route_count/cluster_count/cost_ms/error`。

### 10.3 调试开关
- `debug.routing=true` 时输出：
  - 单域名映射结果
  - 负载策略解析结果
  - 冲突检测详情
- 开关必须 TTL 自动失效（参考总文档第 25 节）。

---

## 11. 性能预算与容量规划

### 11.1 编译预算
- 2 万域名目标：
  - 编译耗时 P95 < 3000ms
  - 额外分配内存 < 1.5GB 峰值
- 单次 Apply CPU 时间不超过 1 core * 3s。

### 11.2 请求路径预算
- 路由匹配增加开销 < 0.5ms（P95）。
- 规则未开启场景下与静态 YARP 差距 < 3%。

### 11.3 峰值瓶颈
- 大配置解析导致 Gen2 回收抖动。
- 应对：
  - 使用 `System.Text.Json` SourceGen。
  - 编译阶段对象池化（List/Dictionary 复用）。
  - 控制 history 保留数量，避免内存膨胀。

---

## 12. 安全与防护

- 只接受已鉴权控制面下发。
- 配置文件写入需校验目录穿越。
- URL/Host 需白名单校验，防止恶意注入。
- Debug 日志默认不输出敏感头。

---

## 13. 落地步骤（工程任务分解）

### 13.1 第一批（必须）
1. 新增 `DynamicProxyConfigProvider` 并接入 DI。
2. 新增 `EdgeConfigToYarpCompiler`。
3. 新增 `ProxyConfigValidator`。
4. 修改 `Program.cs`：移除静态 `LoadFromConfig`，改用动态 Provider。
5. `AgentWsClient.ApplyConfigPayloadAsync` 中接入 `TryApply`。

### 13.2 第二批（强烈建议）
1. 快照摘要持久化 + 历史保留。
2. 手动回滚 task。
3. 指标与审计日志。
4. Debug routing 开关。

### 13.3 第三批（优化）
1. 增量编译（只重编译受影响域名/集群）。
2. 并行编译（upstream 与 route 分阶段并行）。
3. Hash policy 自定义实现（ip_hash 语义对齐）。

---

## 14. 验收标准

### 14.1 功能验收
- 新增域名、删除域名、变更上游均可热生效。
- `https_force`、header 改写、LB 策略生效符合预期。
- 配置非法时线上配置不受影响。

### 14.2 稳定性验收
- 10 分钟内持续每 5 秒更新配置，服务无中断。
- 连续 100 次混合配置变更，错误回滚正确。

### 14.3 性能验收
- 50k RPS 压测下，热更新期间错误率不高于基线 +0.05%。
- P99 延迟抖动不超过基线 +10ms。

---

## 15. 测试计划

### 15.1 单测
- 映射测试：domain/upstream/header/lb/https_force。
- 校验测试：缺字段、非法 URI、冲突 Route。
- 版本测试：重复版本、回退版本、强制覆盖。

### 15.2 集成测试
- WS 下发 -> apply -> 请求验证。
- apply fail -> rollback verify。
- 启动恢复（current 丢失但 last_good 存在）。

### 15.3 压测
- 2 万域名配置 + 10k/50k RPS。
- 热更新风暴测试（高频配置变更）。

---

## 16. 代码改造点（当前仓库对应）

- 需改：`src/Cnn.Agent/Program.cs`
- 需改：`src/Cnn.Agent/Ws/AgentWsClient.cs`
- 需增：`src/Cnn.Agent/Proxy/*`
- 需增：`src/Cnn.Agent/Telemetry/*`（可后置）

---

## 17. 风险与回避

1. YARP API 版本差异
- 回避：先锁定 `Yarp.ReverseProxy 2.2.0` 的 `IProxyConfigProvider` 用法并做兼容层。

2. 大规模配置切换导致短暂高 GC
- 回避：限制 apply 并发 + 对象池化 + 增量编译。

3. 路由冲突导致误转发
- 回避：编译期冲突检测必须阻断 apply。

4. 调试日志开太大影响性能
- 回避：采样 + TTL + QPS 限速。

---

## 18. 与后续文档的接口

- 本文输出给 `安全链详细设计` 的接口：
  - `ISecurityDecisionService.Evaluate(HttpContext)`
- 本文输出给 `TLS详细设计` 的接口：
  - `ITlsCertificateSelector.Select(serverName)`
- 本文输出给 `插件详细设计` 的接口：
  - `IRouteTransformPlugin` / `IRulePlugin`

---

## 19. 建议默认配置

```json
{
  "proxy_runtime": {
    "apply": {
      "allow_partial": false,
      "max_apply_concurrency": 1,
      "history_keep": 20,
      "compile_timeout_ms": 10000
    },
    "rollback": {
      "auto_on_consecutive_failures": 3
    },
    "debug": {
      "routing": false,
      "ttl_seconds": 600
    }
  }
}
```

---

## 20. 实施里程碑

- M1（3-5 天）：动态 Provider 与基础映射可跑通。
- M2（5-7 天）：校验、回滚、持久化、WS 集成。
- M3（3-5 天）：压测、优化、发布手册。

> 到 M3 即可支撑“从静态 YARP 迁移到动态 YARP”的生产灰度上线。
