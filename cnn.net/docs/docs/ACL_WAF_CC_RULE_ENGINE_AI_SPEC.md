# AI 实施规范：ACL/WAF/CC 规则引擎（无歧义版）

> 目标：在 `Cnn.Agent` 实现请求前置安全链，兼容 `EdgeConfigDto` 语义，并可在高并发下稳定运行。

---

## 0. 执行边界

### 0.1 允许修改
- `src/Cnn.Agent/Program.cs`
- `src/Cnn.Agent/Config/EdgeConfigStore.cs`
- `src/Cnn.Agent/Ws/AgentWsClient.cs`（仅接入刷新，不改 WS 协议）
- `src/Cnn.Agent/Security/*.cs`（新建）
- `src/Cnn.Agent/Cnn.Agent.csproj`（仅必要依赖）

### 0.2 禁止事项
- `MUST NOT` 改动 `src/Cnn.Api/*`。
- `MUST NOT` 改变现有缓存功能语义。
- `MUST NOT` 在代理后再执行安全链。

---

## 1. Definition of Done

1. 请求在进入 YARP 前执行安全决策。
2. 支持 ACL 黑白名单与默认动作。
3. 支持 CC 规则基本动作：`allow/block/limit_rate`。
4. 支持 WAF 基础匹配：`sql_injection/xss/scanner`（可配置开关）。
5. 命中后返回对应状态码与错误页键映射。
6. 规则热更新无需重启进程。
7. 至少 10 个单测 + 4 个集成测试通过。

---

## 2. 必须新增的文件与接口

### 2.1 新增文件
- `src/Cnn.Agent/Security/SecurityMiddleware.cs`
- `src/Cnn.Agent/Security/SecurityDecision.cs`
- `src/Cnn.Agent/Security/SecurityDecisionService.cs`
- `src/Cnn.Agent/Security/AclMatcher.cs`
- `src/Cnn.Agent/Security/WafMatcher.cs`
- `src/Cnn.Agent/Security/CcEngine.cs`
- `src/Cnn.Agent/Security/IpCidrTrie.cs`
- `src/Cnn.Agent/Security/SecurityConfigSnapshot.cs`

### 2.2 必须接口签名

```csharp
namespace Cnn.Agent.Security;

public interface ISecurityDecisionService
{
    SecurityDecision Evaluate(HttpContext context);
    void Reload(Cnn.Api.Contracts.Agent.EdgeConfigDto config);
}
```

```csharp
namespace Cnn.Agent.Security;

public sealed record SecurityDecision(
    bool Allowed,
    int StatusCode,
    string? ErrorPageKey,
    string? RuleType,
    string? RuleId,
    string? Reason);
```

---

## 3. Program.cs 强制改造

1. `MUST` 注册：
   - `ISecurityDecisionService`
   - `SecurityMiddleware`
2. `MUST` 在 `app.UseOutputCache()` 前挂载安全中间件。
3. `MUST NOT` 改 ACME 路由行为。

中间件顺序 `MUST` 为：
1. node enabled gate
2. security middleware
3. output cache
4. reverse proxy

---

## 4. 规则语义（必须）

### 4.1 ACL
- 输入来源：`domain.acl_default_action` + `domain.acl_rules` + `black_ips` + `white_ips`。
- 优先级：白名单 > 黑名单 > ACL rules > default action。
- default action 缺省：`allow`。

### 4.2 WAF
- 输入来源：`config.waf`。
- 若 `waf.enable=false` 则跳过。
- 最小匹配能力：
  - `sql_injection`
  - `xss`
  - `scanner`
- 匹配命中默认返回 403（可配置覆盖）。

### 4.3 CC
- 输入来源：`cc_rules/cc_matchers/cc_filters` + `domain.cc_rule_id`。
- 最小动作：
  - `allow`：放行
  - `block`：返回 403
  - `limit_rate`：返回 429
- 计数维度：`ip + host + uri`（最小实现）。
- 计数窗口：按 filter 的 `within_second`。

---

## 5. 性能与并发规则

1. Evaluate 路径 `MUST` 无 JSON 反序列化。
2. 规则索引 `MUST` 在 Reload 阶段构建。
3. CC 计数器 `MUST` 分片并发（至少 16 shards）。
4. 单请求安全决策预算：P95 < 2ms。
5. Regex 匹配 `MUST` 可超时或可控。

---

## 6. 错误页与返回码

默认映射（可配置覆盖）：
- ACL deny -> 403 (`ip`)
- WAF block -> 403 (`403`)
- CC block -> 403 (`403`)
- CC limit_rate -> 429 (`conn_limit`)

`ErrorPageKey` 仅输出 key，不在中间件拼 HTML。

---

## 7. 热更新规则

1. `AgentWsClient` 在 Apply 成功后 `MUST` 调用 `ISecurityDecisionService.Reload(config)`。
2. Reload 失败 `MUST NOT` 影响当前生效规则。
3. Reload 必须采用双缓冲快照替换。

---

## 8. 必须测试清单

单元测试至少 10 条：
1. WhiteList_ShouldAllow
2. BlackList_ShouldDeny
3. AclRule_Deny_ShouldBlock
4. AclDefault_Deny_ShouldBlock
5. WafDisabled_ShouldSkip
6. WafSqlInjection_ShouldBlock
7. CcAllow_ShouldPass
8. CcBlock_ShouldBlock
9. CcLimitRate_Should429
10. ReloadFail_ShouldKeepOldSnapshot

集成测试至少 4 条：
1. Middleware_Order_ShouldBeBeforeProxy
2. EdgeConfigReload_ShouldTakeEffect
3. ConcurrentRequests_ShouldStable
4. RuleHit_ShouldEmitStructuredLog

---

## 9. 验收命令

```bash
rg -n "UseMiddleware<SecurityMiddleware>|ISecurityDecisionService|Reload\(" src/Cnn.Agent
```

```bash
rg -n "class SecurityMiddleware|class CcEngine|class WafMatcher|class AclMatcher" src/Cnn.Agent/Security
```

---

## 10. 交付格式

AI 最终输出必须包含：
1. 改动文件列表
2. 规则优先级证明（日志或测试）
3. 热更新失败不影响线上的证明
4. 测试结果摘要
