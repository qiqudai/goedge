# 权限分级框架 AI 实施规范（Permission Hierarchy Framework AI Spec）

## 1. 目标与范围
- 目标：在 `Cnn.Api` 与 `Cnn.Agent` 建立统一的权限分级框架，替代“按路径硬编码角色判断”的做法。
- 范围：鉴权身份、策略匹配、资源域校验、审计日志、性能边界。
- 不在范围：UI 权限显示（仅后端强制授权）。

## 2. 角色与级别
- 角色：`root`、`admin`、`operator`、`user`、`agent`、`plugin`。
- 级别（由高到低）：`root > admin > operator > user > agent > plugin`。
- 默认策略：`deny by default`。

## 3. 权限点模型
- 权限点命名：`domain:action[:scope]`
- 示例：
  - `site:read`
  - `site:update`
  - `node:config:write`
  - `task:dispatch`
  - `log:read:security`

## 4. 核心接口（必须实现）
```csharp
public interface IAccessContext
{
    long? UserId { get; }
    string Role { get; }
    string? NodeId { get; }
    string? TraceId { get; }
    IReadOnlyDictionary<string, string> Claims { get; }
}

public interface IPermissionCatalog
{
    bool TryGet(string permission, out PermissionRule rule);
    IReadOnlyCollection<PermissionRule> ListAll();
}

public interface IAuthorizationService
{
    AuthorizationDecision Check(IAccessContext context, string permission, string? resourceId = null);
    void Demand(IAccessContext context, string permission, string? resourceId = null);
}
```

## 5. 资源域约束
- `agent`：只能访问自身节点资源（`node_id` 一致）。
- `user`：只能访问所属租户资源。
- `operator`：可操作运行期资源，不可操作财务/全局高危配置。
- `plugin`：仅允许指定扩展点执行，禁止任何管理权限。

## 6. 中间件与服务分层
- L1 身份认证：Token/Agent 认证。
- L2 权限判断：基于 `permission` 与角色级别。
- L3 资源归属：服务层再次校验资源 owner（防越权）。

## 7. 审计日志规范
- 每次授权检查必须可审计（采样可配）。
- 字段：
  - `trace_id`
  - `principal_role`
  - `principal_id`
  - `permission`
  - `resource_id`
  - `decision` (`allow/deny`)
  - `reason`
  - `duration_us`

## 8. 性能与稳定性约束
- 单次 `Check()` 目标：P99 < 100us（内存命中）。
- 权限目录必须为只读快照结构，热更新用原子替换。
- 禁止每请求访问数据库做角色-权限映射。

## 9. AI 实施步骤（顺序固定）
1. 新建 `Abstractions/Authz`：接口与模型。
2. 新建 `Infrastructure/Authz`：内存目录实现。
3. 新建 `Services/Authz`：`AuthorizationService`。
4. 在 `ApiAuthMiddleware` 增加上下文构建。
5. 在 Controller/Service 入口使用 `Demand()`。
6. 增加审计 sink。
7. 增加测试（见第 10 节）。

## 10. 测试矩阵
- 单元：
  - 角色级别比较
  - deny-by-default
  - 资源域约束
  - catalog 热更新原子可见性
- 集成：
  - 管理员访问用户接口
  - 用户越权访问他人资源
  - agent 访问非自身节点资源
- 压测：
  - 5k RPS 授权检查 CPU 占比

## 11. Definition of Done
- 所有核心 API 从“路径角色判断”迁移到权限点模型。
- 权限拒绝统一返回语义（`permission_denied`）。
- 审计日志可检索且字段完整。
- 单元/集成测试通过，压测满足约束。
