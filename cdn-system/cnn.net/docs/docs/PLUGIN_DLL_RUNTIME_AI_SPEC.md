# AI 实施规范：动态 DLL 规则插件运行时（无歧义版）

> 目标：为复杂规则提供可热插拔扩展，同时保证安全、稳定、可回滚。

---

## 0. 执行边界

### 0.1 允许修改
- `src/Cnn.Agent/Program.cs`
- `src/Cnn.Agent/Security/*.cs`
- `src/Cnn.Agent/Plugin/*.cs`（新建）
- `src/Cnn.Agent/Config/AgentRuntimePaths.cs`

### 0.2 禁止事项
- `MUST NOT` 允许未签名插件在生产启用。
- `MUST NOT` 让插件直接访问数据库连接对象。

---

## 1. Definition of Done

1. 插件可从指定目录加载并参与规则决策。
2. 插件支持启用/禁用与热切换。
3. 插件执行超时可控，超时后自动熔断。
4. 插件异常不会影响主链路可用性。
5. 至少 8 个单元测试通过。

---

## 2. 必须新增的文件与接口

- `src/Cnn.Agent/Plugin/IRulePlugin.cs`
- `src/Cnn.Agent/Plugin/PluginHost.cs`
- `src/Cnn.Agent/Plugin/PluginLoadContext.cs`
- `src/Cnn.Agent/Plugin/PluginManifest.cs`
- `src/Cnn.Agent/Plugin/PluginDecision.cs`
- `src/Cnn.Agent/Plugin/PluginCircuitBreaker.cs`

接口：

```csharp
namespace Cnn.Agent.Plugin;

public interface IRulePlugin
{
    string Name { get; }
    string Version { get; }
    ValueTask InitializeAsync(IReadOnlyDictionary<string, string> settings, CancellationToken ct);
    ValueTask<PluginDecision> EvaluateAsync(HttpContext context, CancellationToken ct);
    ValueTask DisposeAsync();
}
```

```csharp
namespace Cnn.Agent.Plugin;

public sealed record PluginDecision(bool Handled, bool Allowed, int StatusCode, string? Reason);
```

---

## 3. 安全规则（MUST）

1. 生产环境默认 `plugins.enabled=false`。
2. 启用时 `MUST` 验证 manifest：
   - `name`
   - `version`
   - `sha256`
   - `entry_type`
3. 若 `plugins.require_signature=true`，必须校验签名通过。
4. 加载失败时不得影响主链路。

---

## 4. 运行时规则

1. 插件执行顺序：内置规则后、最终响应前。
2. 单插件超时默认 5ms（可配）。
3. 连续失败达到阈值触发熔断，熔断期间跳过执行。
4. 熔断恢复采用半开探测。

---

## 5. 与 SecurityPipeline 集成

1. `SecurityDecisionService` 必须调用 `PluginHost.Evaluate()`。
2. 若插件返回 `Handled=true`，按插件决策终止后续规则。
3. 若插件异常/超时，记录 warning 并继续内置规则或默认策略。

---

## 6. 配置项（必须支持）

```json
{
  "plugins": {
    "enabled": false,
    "directory": "edge-node/plugins",
    "require_signature": true,
    "eval_timeout_ms": 5,
    "breaker": {
      "fail_threshold": 20,
      "window_seconds": 60,
      "open_seconds": 120
    }
  }
}
```

---

## 7. 必须测试清单

1. Load_ValidManifest_ShouldSuccess
2. Load_BadHash_ShouldFail
3. Evaluate_HandledBlock_ShouldReturnPluginDecision
4. Evaluate_Timeout_ShouldOpenBreaker
5. Evaluate_Exception_ShouldNotBreakMainFlow
6. Breaker_Open_ShouldSkipPlugin
7. Breaker_HalfOpen_ShouldProbe
8. Reload_NewPluginVersion_ShouldSwap

---

## 8. 验收命令

```bash
rg -n "IRulePlugin|PluginHost|PluginCircuitBreaker|EvaluateAsync" src/Cnn.Agent
```

---

## 9. 交付格式

AI 输出必须包含：
1. 安全护栏清单（签名/哈希/超时/熔断）
2. 主链路不受影响的证明
3. 测试结果摘要
