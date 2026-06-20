# AI 实施规范：调试日志 / 人工调试开关 / 审计（无歧义版）

> 目标：实现“可控、可审计、可自动失效”的调试能力，避免线上常驻 debug 风险。

---

## 0. 执行边界

### 0.1 允许修改
- `src/Cnn.Agent/Program.cs`
- `src/Cnn.Agent/Ws/AgentWsClient.cs`
- `src/Cnn.Agent/Debug/*.cs`（新建）
- `src/Cnn.Agent/Security/*.cs`（仅接入 debug session）

### 0.2 禁止事项
- `MUST NOT` 默认开启 debug。
- `MUST NOT` 记录明文敏感头（Authorization/Cookie/Token）。

---

## 1. Definition of Done

1. 支持全局 debug 开关与模块级开关。
2. 支持请求级人工调试（token 方式）。
3. 开关支持 TTL 自动失效。
4. 调试行为有审计日志。
5. 支持采样率和每秒事件上限。
6. 至少 8 个测试通过。

---

## 2. 必须新增文件与接口

- `src/Cnn.Agent/Debug/DebugSwitchStore.cs`
- `src/Cnn.Agent/Debug/DebugSessionService.cs`
- `src/Cnn.Agent/Debug/DebugAuditLogger.cs`
- `src/Cnn.Agent/Debug/DebugOptions.cs`
- `src/Cnn.Agent/Debug/DebugLogSanitizer.cs`

接口：

```csharp
namespace Cnn.Agent.Debug;

public interface IDebugSessionService
{
    bool IsEnabled(string module, HttpContext context);
    string? GetSessionId(HttpContext context);
    void Update(DebugOptions options, TimeSpan ttl);
}
```

---

## 3. 开关规则（MUST）

1. 默认值：`enabled=false`。
2. 模块开关：`routing/cache/security/tls/plugin/ws/task`。
3. 请求级调试：
   - Header `X-Debug-Token`
   - 仅在 `internal_ip_only=true` 时内网生效
4. TTL 到期后自动回到默认关闭。

---

## 4. 审计与脱敏规则

1. 每次开关变更必须写审计事件：
   - who
   - when
   - module
   - ttl
2. 调试日志字段必须脱敏：
   - Authorization -> `***`
   - Cookie -> `***`
   - Token -> `***`
3. Body 默认不打印，仅白名单字段可打印。

---

## 5. 限流与采样

1. `sample_rate` 默认 0.01。
2. `max_events_per_sec` 默认 200。
3. 超限后降级摘要日志，不得继续全量详细日志。

---

## 6. 必须测试清单

1. DebugDisabled_Default_ShouldFalse
2. ModuleEnabled_ShouldTrue
3. TokenDebug_InternalIp_ShouldTrue
4. TokenDebug_ExternalIp_ShouldFalse
5. TtlExpire_ShouldAutoDisable
6. SensitiveHeaders_ShouldMask
7. MaxEventsPerSec_ShouldThrottle
8. AuditLog_OnSwitchUpdate_ShouldEmit

---

## 7. 验收命令

```bash
rg -n "DebugSession|DebugSwitch|Audit|Sanitizer|X-Debug-Token" src/Cnn.Agent
```

---

## 8. 交付格式

AI 输出必须包含：
1. 开关矩阵
2. 脱敏规则证明
3. TTL 自动失效证明
4. 测试结果摘要
