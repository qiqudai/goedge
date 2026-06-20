# AI 实施规范：CK 日志落库 + Agent/API 交互与配置同步测试（无歧义版）

> 文档日期：2026-04-01  
> 目标读者：代码生成型 AI（可直接按文档实施）  
> 文档定位：补齐当前测试缺口，不改协议语义，只增强可测性与验证覆盖。

---

## 0. 现状与缺口

当前代码已具备以下实现：
- API 侧 WS 通道：`/ws/agent`，支持 `agent_hello`、`heartbeat`、`node_sync`、`task_ack`、`agent_logs_*`。
- Agent 侧 WS 客户端：`AgentWsClient`，支持心跳、日志上报、配置应用、task ack/outbox 重试。
- CK 写入服务：`AgentLogService` + `ClickHouseHttpHelper`，支持 `node_access_logs` / `node_stream_logs` / `node_metrics` / `node_events`。

当前测试通过（`120/120`）但存在两个关键空白：
1. **未覆盖 CK 真实写入链路**（仅有实现，无落库断言测试）。
2. **未覆盖 agent/api WS 端到端交互链路**（现有测试偏状态/幂等单元，不是完整消息往返）。

---

## 1. 本次实施目标

### 1.1 必达目标
1. 验证 `agent_logs_access|stream|metrics|events` 被 API 接收后可写入 CK（至少合同级 + 可选真实 CK 级）。
2. 验证 Agent 与控制面 WS 的关键往返：
   - `heartbeat -> heartbeat_ack(sync_action) -> node_sync`
   - `task_dispatch -> task_ack`
   - `edge_config` 下发与版本应用行为。
3. 保证测试可重复、可在 CI 执行、失败可定位。

### 1.2 非目标
- 不修改现有 WS 字段名/字段语义。
- 不重写业务流程，不引入协议破坏性变更。
- 不要求一次性把所有 API 页面 E2E 接入本规范（前端单独维护）。

---

## 2. 强制约束（MUST / MUST NOT）

### 2.1 MUST
- 必须新增“可单独执行”的测试分组：`CK`、`WS_SYNC`。
- 必须提供失败诊断信息（至少包含：消息 kind、task_id/msg_id、node_id、trace 或错误文本）。
- 必须保证不依赖人工操作（命令可直接执行）。
- 必须在文档末尾输出“验收命令清单”并可直接复制运行。

### 2.2 MUST NOT
- 不得修改以下协议 key：
  - `kind`
  - `task_id`
  - `task_type`
  - `msg_id`
  - `sync_action`
  - `node_sync`
  - `task_ack`
  - `agent_logs_access|agent_logs_stream|agent_logs_metrics|agent_logs_events`
- 不得把测试逻辑塞进生产逻辑分支（禁止 `if DEBUG` 方式污染业务代码）。
- 不得让测试默认依赖外部常驻 MySQL/ClickHouse（必须提供自包含路径）。

---

## 3. 分层测试策略（固定三层）

## 3.1 L1：合同级（默认必跑）
- 目标：在不依赖 Docker 的环境验证请求拼装、字段映射、发送行为。
- 方法：
  - API 侧：用本地 fake HTTP sink（模拟 CK HTTP 接口）验证 `INSERT ... FORMAT JSONEachRow` 的请求路径、鉴权头、body 行数与关键字段。
  - Agent 侧：用 fake WS server 验证消息序列和关键字段。

## 3.2 L2：组件集成级（默认必跑）
- 目标：验证消息处理器在真实对象组合下的行为正确（不需要全量进程启动）。
- 方法：
  - API：将 `/ws/agent` 消息处理逻辑抽离为可测试组件，注入 fake service 断言副作用。
  - Agent：构造 `AgentWsClient` 依赖的 fake/stub 组件，跑真实 Receive/Send 循环。

## 3.3 L3：真实依赖级（可选门禁，建议 nightly）
- 目标：验证真实 ClickHouse 落库。
- 方法：
  - 启用条件：`CNN_ENABLE_CK_IT=1`。
  - 启动方式：Testcontainers 或 docker compose（二选一，推荐 Testcontainers）。
  - 若环境不支持 Docker，测试应 `Skip`，并输出明确原因。

---

## 4. 代码结构与文件规划（无歧义）

## 4.1 API 测试文件（新增）
- `tests/Cnn.Api.Tests/Integration/ClickHouse/FakeClickHouseServer.cs`
- `tests/Cnn.Api.Tests/Integration/ClickHouse/AgentLogServiceContractTests.cs`
- `tests/Cnn.Api.Tests/Integration/ClickHouse/AgentLogServiceRealClickHouseTests.cs`（可选启用）
- `tests/Cnn.Api.Tests/Integration/Ws/AgentWsMessageProcessorTests.cs`

## 4.2 Agent 测试文件（新增）
- `tests/Cnn.Agent.Tests/Ws/FakeControlPlaneWsServer.cs`
- `tests/Cnn.Agent.Tests/Ws/AgentWsClientSyncFlowTests.cs`
- `tests/Cnn.Agent.Tests/Ws/AgentWsClientOutboxReplayTests.cs`

## 4.3 生产代码（最小重构，允许新增）
- `src/Cnn.Api/Ws/AgentWsMessageProcessor.cs`（从 `Program.cs` 抽消息处理逻辑）
- `src/Cnn.Api/Ws/AgentWsSessionContext.cs`（消息处理上下文 DTO）

说明：
- `Program.cs` 只保留 WS 连接、收发循环和依赖组装；业务判断下沉到 `AgentWsMessageProcessor`。
- 这是为测试而做的低耦合重构，不改变外部协议。

---

## 5. 详细实施板块（按顺序执行）

## 5.1 板块 A：可测性重构（API WS）
### A.1 任务
1. 从 `Program.cs` 抽离消息分发逻辑：
   - 输入：`json string + session context`
   - 输出：`outbound frames`（可为空）
2. 提供可替换依赖：
   - `IAgentLogService`
   - `IAgentNodeService`
   - `IAgentTaskAckService`
   - `IAgentAckWaiter`
   - 其他当前 WS 处理已用服务

### A.2 DoD
- `Program.cs` 功能不变，协议不变。
- 抽离后的处理器可在单元测试中直接调用。

---

## 5.2 板块 B：CK 合同级测试（默认必跑）
### B.1 用例
1. `InsertAccessLogsAsync_ShouldSendJsonEachRowToNodeAccessLogs`
2. `InsertStreamLogsAsync_ShouldSendJsonEachRowToNodeStreamLogs`
3. `InsertMetricsAsync_ShouldSendJsonEachRowToNodeMetrics`
4. `InsertEventLogsAsync_ShouldSendJsonEachRowToNodeEvents`
5. `ClickHouseConfigInvalid_ShouldReturnZeroAndNotSend`

### B.2 断言
- 请求 URL 包含正确 query（`INSERT INTO <table> FORMAT JSONEachRow`）。
- Basic Auth 头存在且正确（若配置含 user/pass）。
- body 行数与输入有效行数一致。
- 关键字段映射正确：`node_id`、`node_ip`、`ts`、`raw/payload`。
- 错误/无配置时返回 `0` 且不发送请求。

### B.3 DoD
- 上述 5 个用例全部通过。

---

## 5.3 板块 C：API WS 消息处理测试（默认必跑）
### C.1 用例
1. `HeartbeatMessage_ShouldReturnHeartbeatAckWithSyncAction`
2. `TaskAckMessage_ShouldInvokeAckServiceAndAckWaiter`
3. `NodeSyncMessage_ShouldInvokeAgentNodeService`
4. `AgentLogsAccessMessage_ShouldInvokeInsertAccessLogs`
5. `AgentLogsMetricsMessage_ShouldInvokeInsertMetrics`
6. `InvalidJsonOrUnknownKind_ShouldBeIgnoredWithoutCrash`

### C.2 断言
- `kind=heartbeat` 时生成 `heartbeat_ack` 帧，且含 `sync_action`。
- `kind=task_ack` 时调用 ack 服务与 waiter。
- 各 `agent_logs_*` 对应调用正确的 `IAgentLogService` 方法。
- 非法消息不抛未处理异常。

### C.3 DoD
- 上述 6 个用例通过，覆盖所有日志 kind。

---

## 5.4 板块 D：Agent WS 往返与同步测试（默认必跑）
### D.1 用例
1. `HeartbeatAckEnable_ShouldSendNodeSyncSuccess`
2. `HeartbeatAckDisable_ShouldSendNodeSyncSuccess`
3. `EdgeConfigMessage_ShouldApplyAndMarkVersion`
4. `TaskDispatch_ConfigSync_ShouldSendTaskAckSuccess`
5. `TaskDispatch_Duplicate_ShouldReplayAckOrIgnored`
6. `SendFail_ShouldPersistOutbox_AndReplayAfterReconnect`

### D.2 断言
- 收到 `heartbeat_ack(sync_action=enable|disable)` 后，Agent 发出 `node_sync`，并携带 `action/success`。
- 收到 `edge_config` 后，版本标记逻辑符合 `ConfigVersionTracker` 预期。
- 收到 `task_dispatch` 后会发送 `task_ack`，状态与执行结果一致。
- 断链重连后 pending outbox 被补发并清理。

### D.3 DoD
- 上述 6 个用例通过。

---

## 5.5 板块 E：真实 CK 落库测试（可选门禁）
### E.1 开关
- 仅当 `CNN_ENABLE_CK_IT=1` 时执行。

### E.2 用例
1. `AccessLog_Insert_ShouldBeQueryableFromNodeAccessLogs`
2. `Metrics_Insert_ShouldBeQueryableFromNodeMetrics`

### E.3 断言
- 插入后可查询到对应 `node_id` 和关键字段。
- 行数符合预期。

### E.4 DoD
- 在支持 Docker 的环境中通过；不支持时明确 Skip 原因。

---

## 6. 性能与稳定性门禁（测试实现层）

- 所有新增测试单文件执行时间目标：
  - 合同级：< 5s
  - 组件集成级：< 20s
  - 真实 CK 级：< 120s（含容器启动）
- 失败日志必须包含：
  - 消息 kind
  - task_id/msg_id（如有）
  - node_id
  - 失败阶段（receive/process/send/insert/query）

---

## 7. CI 运行策略（固定）

### 7.1 PR 必跑
- `dotnet test Cnn.sln -v minimal`
- 仅 L1 + L2，不启用真实 CK。

### 7.2 Nightly / 手动门禁
- `CNN_ENABLE_CK_IT=1 dotnet test Cnn.sln -v minimal`

---

## 8. 验收命令（可直接复制）

```bash
cd /Users/fake/code/goedge/cdn-system/cnn.net

# 1) 全量基础测试
~/.dotnet/dotnet test Cnn.sln -v minimal

# 2) 只跑 API 集成测试（建议新增 Trait: Category=Integration）
~/.dotnet/dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj -v minimal

# 3) 只跑 Agent WS 同步测试（建议新增 Trait: Category=WsSync）
~/.dotnet/dotnet test tests/Cnn.Agent.Tests/Cnn.Agent.Tests.csproj -v minimal

# 4) 启用真实 CK 集成（可选）
CNN_ENABLE_CK_IT=1 ~/.dotnet/dotnet test tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj -v minimal
```

---

## 9. 风险清单与处理

1. 风险：API WS 逻辑仍全部留在 `Program.cs`，难测。  
   处理：先做板块 A 抽离，再写测试。

2. 风险：真实 CK 依赖 Docker，CI 环境可能不可用。  
   处理：L3 测试设环境开关，可 Skip，不阻塞 PR。

3. 风险：AgentWsClient 依赖多，测试构造复杂。  
   处理：统一 `TestHostBuilder` 工具类集中构造 fake 依赖。

---

## 10. 最终交付清单（必须全部满足）

1. 新增测试文件已按第 4 节落地。
2. L1 + L2 测试在本地稳定通过（连续 3 次）。
3. 真实 CK 测试至少在 1 次 Docker 环境通过并保存日志。
4. 输出最终报告需包含：
   - 新增测试总数
   - 通过数/失败数
   - 覆盖的消息 kind 列表
   - CK 覆盖表列表（`node_access_logs/node_stream_logs/node_metrics/node_events`）

---

## 11. Definition of Done（本规范）

当且仅当以下全部成立，才算“CK 日志 + agent/api 交互/同步测试已完成”：
1. 本规范第 5 节的板块 A/B/C/D 全部完成并通过。  
2. `dotnet test Cnn.sln -v minimal` 通过。  
3. 至少完成一次 L3 真实 CK 落库验证（或有明确环境限制说明并保留合同级证据）。  
4. 无协议字段变更、无兼容性破坏。  

