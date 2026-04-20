# 代码框架设计 V1（高复用、低耦合）

> 适用：`cnn.net` 后续全部开发任务。

## 1. 总体分层（强制）

1. `Core`（纯领域）
- 只放接口、领域模型、策略抽象。
- 禁止依赖 ASP.NET / YARP / SQL / 文件系统。

2. `Application`（用例编排）
- 组合 Core 接口实现业务流程。
- 只依赖 `Core` 抽象，不直接依赖基础设施实现。

3. `Infrastructure`（技术实现）
- 提供文件、网络、日志、缓存、插件、YARP 适配实现。
- 通过 DI 注入到 Application。

4. `Host`（Program 装配）
- 仅负责 DI 注册、中间件顺序、配置绑定。
- 禁止写业务逻辑。

## 2. 权限分级框架设计

### 2.1 权限模型

- `PrincipalType`: `system | admin | user | node | plugin`
- `PermissionLevel`: `read | write | execute | manage`
- `ResourceScope`: `global | tenant | node | site | stream | task | log`

### 2.2 核心接口（必须）

```csharp
public interface IAccessContext
{
    string PrincipalId { get; }
    string PrincipalType { get; }
    IReadOnlyDictionary<string, string> Claims { get; }
}

public interface IPermissionPolicy
{
    bool IsAllowed(IAccessContext context, string action, string scope, string resourceId);
}

public interface IAuthorizationService
{
    void Demand(IAccessContext context, string action, string scope, string resourceId);
    bool Check(IAccessContext context, string action, string scope, string resourceId);
}
```

### 2.3 授权执行规则

1. 默认拒绝（Deny by default）。
2. 必须显式策略放行。
3. `node` 只能访问自身 `node_id` 资源。
4. `plugin` 只允许 `execute` 指定扩展点，禁止管理权限。
5. 所有拒绝必须写审计日志（含 principal/action/resource/traceId）。

### 2.4 策略组织（低耦合）

- 使用 `PolicyRegistry` 注册策略：
  - `NodePolicy`
  - `AdminPolicy`
  - `UserPolicy`
  - `PluginPolicy`
- Application 层只调用 `IAuthorizationService`，不关心具体策略实现。

## 3. 日志读写框架设计

### 3.1 日志写入（高吞吐）

采用 `Channel<LogEvent>` 异步写入管道：

- `ILogEventWriter`：业务写入入口
- `ILogPipeline`：内存队列 + 批量 flush
- `ILogSink`：目标存储（file/clickhouse/console）

```csharp
public interface ILogEventWriter
{
    bool TryWrite(LogEvent evt);
}

public interface ILogSink
{
    ValueTask WriteBatchAsync(IReadOnlyList<LogEvent> events, CancellationToken ct);
}
```

### 3.2 日志读取（可扩展）

- `ILogQueryService` 统一查询抽象。
- 支持 `from/to/level/module/traceId/nodeId` 过滤。
- 具体实现：
  - `FileLogQueryService`
  - `ClickHouseLogQueryService`

### 3.3 LogEvent 统一结构

```csharp
public sealed record LogEvent(
    DateTimeOffset Timestamp,
    string Level,
    string Module,
    string Message,
    string TraceId,
    IReadOnlyDictionary<string, object?> Fields);
```

### 3.4 日志规范

1. 结构化 JSON，禁止拼接长字符串。
2. 敏感字段必须脱敏：`Authorization/Cookie/Token/Key`。
3. 每条关键日志必须带 `trace_id`。
4. 模块名固定：`routing/security/cache/tls/plugin/ws/task`。
5. 生产默认 `Information`，`Debug/Trace` 需开关 + TTL。

## 4. 复用与解耦约束

1. 跨模块调用必须走接口，不直接 new 具体实现。
2. 规则引擎、YARP 编译器、TLS 选择器互相只通过 DTO/接口通信。
3. 中间件只依赖 Facade 服务，不依赖底层实现细节。
4. 任何“工具类静态方法”不得持有全局状态。

## 5. 配置管理约束

1. 配置对象与运行时快照分离：
- `RawConfig`（原始）
- `RuntimeSnapshot`（编译后）
2. 请求路径只读 `RuntimeSnapshot`。
3. 更新路径只写 `RawConfig` + 原子替换快照。
4. 更新失败保持旧快照。

## 6. 错误处理与返回语义

统一结果对象：

```csharp
public sealed record Result<T>(bool Success, string Code, string Message, T? Data);
```

- `Code` 必须机器可判断：`ok/skipped/invalid/conflict/fail`。
- 异常仅用于不可恢复错误；可预期分支返回 `Result`。

## 7. 可测试性要求

1. 所有核心服务必须可通过接口 mock。
2. 每个服务至少覆盖：成功、失败、并发、边界四类测试。
3. 日志与权限策略都要有单元测试。

## 8. 命名与目录约定

- `Abstractions`：接口
- `Services`：用例服务
- `Adapters`：基础设施适配
- `Models`：领域/DTO
- `Pipelines`：中间件/处理链

## 9. 立即落地（后续开发默认执行）

1. 新增权限抽象：`IAccessContext / IAuthorizationService / IPermissionPolicy`。
2. 新增日志抽象：`ILogEventWriter / ILogSink / ILogQueryService`。
3. 在 `Program.cs` 仅注入这些抽象，不在业务代码里直接使用具体 logger/sink。

