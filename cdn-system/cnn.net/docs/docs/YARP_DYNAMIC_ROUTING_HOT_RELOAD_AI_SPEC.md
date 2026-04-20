# AI 实施规范：YARP 动态路由与热更新（无歧义版）

> 文档类型：AI 编码执行规范（不是概念说明）  
> 目标：让任意代码生成 AI 按本文直接修改 `cnn.net` 并通过验收。  
> 约束词：`MUST`=必须，`MUST NOT`=禁止，`SHOULD`=建议。

---

## 0. 执行边界

### 0.1 允许修改文件
- `src/Cnn.Agent/Program.cs`
- `src/Cnn.Agent/Ws/AgentWsClient.cs`
- `src/Cnn.Agent/Config/EdgeConfigStore.cs`
- `src/Cnn.Agent/Config/AgentRuntimePaths.cs`（仅当必须新增路径）
- `src/Cnn.Agent/Cnn.Agent.csproj`（仅当必须新增包）
- `src/Cnn.Agent/Proxy/*.cs`（新建）

### 0.2 禁止事项
- `MUST NOT` 改动 `src/Cnn.Api/*` 行为。
- `MUST NOT` 删除现有 WS 协议字段。
- `MUST NOT` 改变现有缓存逻辑语义（`CacheRuntimeStore` 相关）。

---

## 1. 结果定义（Definition of Done）

以下 8 条全部满足才算完成：

1. `Program.cs` 不再调用 `LoadFromConfig("ReverseProxy")`。
2. 系统注册自定义 `IProxyConfigProvider`。
3. `AgentWsClient.ApplyConfigPayloadAsync()` 成功时会触发 YARP 配置热更新。
4. 热更新失败时，线上旧配置继续生效（无中断）。
5. 同版本配置重复下发返回 `skipped`。
6. 新版本配置生效后可在内存读取到 `current version`。
7. 至少具备 1 个“最小兜底路由”（配置为空时返回 503）。
8. 新增最少 6 个单元测试（见第 7 节）。

---

## 2. 必须新增的类与命名（精确）

### 2.1 命名空间
- 所有新类 `MUST` 在命名空间 `Cnn.Agent.Proxy`。

### 2.2 新建文件列表（精确）
- `src/Cnn.Agent/Proxy/DynamicProxyConfigProvider.cs`
- `src/Cnn.Agent/Proxy/DynamicProxyConfig.cs`
- `src/Cnn.Agent/Proxy/ProxySnapshot.cs`
- `src/Cnn.Agent/Proxy/EdgeConfigToYarpCompiler.cs`
- `src/Cnn.Agent/Proxy/ProxyConfigValidator.cs`
- `src/Cnn.Agent/Proxy/EdgeProxyRuntime.cs`
- `src/Cnn.Agent/Proxy/ProxyApplyResult.cs`

### 2.3 必须公开的接口与签名（精确）

```csharp
namespace Cnn.Agent.Proxy;

public interface IEdgeProxyRuntime
{
    ProxyApplyResult TryApply(Cnn.Api.Contracts.Agent.EdgeConfigDto config, bool force = false);
    ProxySnapshot GetCurrent();
    ProxySnapshot? GetLastGood();
}
```

```csharp
namespace Cnn.Agent.Proxy;

public sealed class DynamicProxyConfigProvider : Yarp.ReverseProxy.Configuration.IProxyConfigProvider
{
    public Yarp.ReverseProxy.Configuration.IProxyConfig GetConfig();
    public ProxyApplyResult TryUpdate(ProxySnapshot snapshot);
}
```

```csharp
namespace Cnn.Agent.Proxy;

public sealed record ProxyApplyResult(bool Success, string Status, string? Error, long Version);
```

---

## 3. Program.cs 改造规则（逐条）

### 3.1 DI 注册
- `MUST` 删除：
  - `builder.Services.AddReverseProxy().LoadFromConfig(builder.Configuration.GetSection("ReverseProxy"));`
- `MUST` 增加：
  - `builder.Services.AddSingleton<DynamicProxyConfigProvider>();`
  - `builder.Services.AddSingleton<IProxyConfigProvider>(sp => sp.GetRequiredService<DynamicProxyConfigProvider>());`
  - `builder.Services.AddSingleton<IEdgeProxyRuntime, EdgeProxyRuntime>();`
  - `builder.Services.AddSingleton<EdgeConfigToYarpCompiler>();`
  - `builder.Services.AddSingleton<ProxyConfigValidator>();`
  - `builder.Services.AddReverseProxy();`

### 3.2 管道
- `app.MapReverseProxy().CacheOutput();` 保持不变。
- `MUST NOT` 变更现有 ACME 路由行为。

---

## 4. AgentWsClient 改造规则（逐条）

### 4.1 依赖注入
- `MUST` 在构造函数注入 `IEdgeProxyRuntime proxyRuntime`。
- `MUST` 保存为私有字段 `_proxyRuntime`。

### 4.2 ApplyConfigPayloadAsync 行为
- 在 `config != null` 分支中，当前顺序 `MUST` 变为：
  1. `_edgeConfigStore.Update(config)`
  2. `PersistDynamicConfigAsync(config, cancellationToken)`
  3. `_proxyRuntime.TryApply(config, force)`
- 若第 3 步返回失败：
  - `MUST` 抛出异常或返回 `fail`，并保留当前线上路由不变。
- 返回值规则：
  - `ok`：成功更新且版本变更。
  - `skipped`：版本相同且 `force=false`。
  - `fail`：校验失败或编译失败。

### 4.3 日志
- `MUST` 记录结构化日志字段：
  - `version`
  - `status`
  - `route_count`
  - `cluster_count`
  - `error`

---

## 5. 编译与校验规则（不可自由发挥）

### 5.1 输入
- 输入类型固定：`EdgeConfigDto`。
- 使用字段：
  - `domains`
  - `upstreams`
  - `version`

### 5.2 校验规则（MUST）
1. `version > 0`。
2. 每个 domain 的 `name` 非空。
3. 每个 domain 的 `upstream_key` 必须存在于 upstream 集合。
4. 每个 upstream 至少 1 个 target。
5. 每个 target 的地址可解析为合法 URI（`http://` 或 `https://` 自动补全允许）。

### 5.3 映射规则（MUST）
- RouteId：`route:{host}`（若冲突再加 `:{index}`）。
- ClusterId：`cluster:{upstream.id}`。
- Route Match：
  - `Hosts = [domain.name]`
  - `Path = "/{**catch-all}"`
- Route -> Cluster：使用 `domain.upstream_key`。
- Cluster LB：
  - 空值或未知值 => `RoundRobin`

### 5.4 兜底路由（MUST）
- 当 `domains` 为空时，`MUST` 生成一个兜底 Route + Cluster。
- 兜底行为：返回 503（可通过固定本地 destination 或专用 middleware）。

---

## 6. 线程安全与热更新规则（MUST）

1. `DynamicProxyConfigProvider` 内部当前配置引用必须原子替换。
2. 每次更新必须创建新的 `CancellationTokenSource` 并触发旧 token 取消。
3. `GetConfig()` 不得加写锁。
4. 并发更新必须串行化（`SemaphoreSlim(1,1)`）。
5. 新配置编译失败时 `MUST NOT` 影响当前 `IProxyConfig`。

---

## 7. 最小测试清单（必须实现）

新增测试项目或在现有测试中补齐，至少以下 6 条：

1. `Apply_NewVersion_ShouldSwapConfig`
2. `Apply_SameVersion_WithoutForce_ShouldSkip`
3. `Apply_InvalidDomainUpstream_ShouldFailAndKeepOld`
4. `Compile_EmptyDomains_ShouldBuildFallbackRoute`
5. `ConcurrentApply_ShouldKeepLastValidSnapshot`
6. `GetConfig_AfterUpdate_ShouldExposeNewChangeToken`

每个测试 `MUST` 断言：
- `Success/Status`
- `Version`
- `Route/Cluster` 数量
- 旧配置是否保持

---

## 8. 验收命令（AI 自检）

AI 完成修改后必须执行并检查：

```bash
rg -n "LoadFromConfig\(" src/Cnn.Agent/Program.cs
```
期望：无匹配。

```bash
rg -n "DynamicProxyConfigProvider|IEdgeProxyRuntime|EdgeConfigToYarpCompiler|ProxyConfigValidator" src/Cnn.Agent
```
期望：均有定义与引用。

```bash
rg -n "TryApply\(|TryUpdate\(|IProxyConfigProvider" src/Cnn.Agent
```
期望：`Program.cs`、`AgentWsClient.cs`、`Proxy/*.cs` 均有关联。

---

## 9. 代码生成限制（给 AI）

- `MUST` 使用现有 `net9.0` 与当前 YARP 版本 API。
- `MUST` 优先复用现有 `EdgeConfigStore` 与 `AgentRuntimePaths`。
- `MUST NOT` 增加复杂第三方依赖。
- `MUST` 产出可读、可维护代码，避免反射黑魔法。

---

## 10. 可选增强（本次可不做）

以下能力本轮可以不实现，但预留接口：
- 增量编译（只编译变化域名）。
- Route Transform 插件。
- 规则链与路由链联动 trace。

---

## 11. 提交格式（AI 产出要求）

AI 最终输出必须包含：
1. 改动文件清单。
2. 每个文件核心变更点（1-3 条）。
3. 失败场景如何保证不影响线上流量。
4. 验收命令结果摘要。

