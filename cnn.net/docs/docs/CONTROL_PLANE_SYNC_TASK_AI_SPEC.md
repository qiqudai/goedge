# AI 实施规范：控制面同步与任务一致性（无歧义版）

> 目标：确保 Agent 的配置同步、任务执行、状态回写在断线/重连/重复消息场景下保持一致且幂等。

---

## 0. 执行边界

### 0.1 允许修改
- `src/Cnn.Agent/Ws/AgentWsClient.cs`
- `src/Cnn.Agent/Config/*.cs`
- `src/Cnn.Agent/Sync/*.cs`（新建）
- `src/Cnn.Agent/Tasks/*.cs`（新建）

### 0.2 禁止事项
- `MUST NOT` 修改现有 WS 消息字段命名。
- `MUST NOT` 删除现有任务类型兼容分支。

---

## 1. Definition of Done

1. 配置应用记录 `last_applied_version`。
2. 同版本配置重复下发幂等 `skipped`。
3. `task_ack` 重复发送不产生重复副作用。
4. WS 断线恢复后可自动补发 pending node_sync/task_ack。
5. 至少 10 个单元测试 + 3 个集成测试通过。

---

## 2. 必须新增文件与接口

- `src/Cnn.Agent/Sync/SyncStateStore.cs`
- `src/Cnn.Agent/Sync/ConfigVersionTracker.cs`
- `src/Cnn.Agent/Tasks/TaskIdempotencyStore.cs`
- `src/Cnn.Agent/Tasks/TaskAckOutbox.cs`

接口：

```csharp
namespace Cnn.Agent.Sync;

public interface IConfigVersionTracker
{
    long ReadAppliedVersion();
    bool ShouldApply(long newVersion, bool force);
    void MarkApplied(long version);
}
```

```csharp
namespace Cnn.Agent.Tasks;

public interface ITaskIdempotencyStore
{
    bool TryBegin(long taskId, string taskType, string payloadHash);
    void MarkDone(long taskId, string resultHash);
    bool IsDone(long taskId, out string? resultHash);
}
```

---

## 3. ApplyConfigPayloadAsync 强制规则

1. `MUST` 先判断 `ShouldApply(version, force)`。
2. `MUST` 在成功应用后 `MarkApplied(version)`。
3. `MUST` 对失败应用记录 `last_apply_error`（含 version 与 traceId）。
4. `MUST` 在返回值区分：`ok/skipped/fail`。

---

## 4. Task 幂等规则

1. 任务执行前 `TryBegin(taskId, type, payloadHash)`。
2. 若返回 false 且已完成，直接回放历史 `task_ack`。
3. 若返回 false 且执行中，返回 `ignored` 或 `running` 状态。
4. 执行成功后 `MarkDone` 并落 outbox。

---

## 5. Outbox 与断线恢复

1. `task_ack` 与 `node_sync` 必须写 outbox 后发送。
2. 发送失败保留 outbox，下一次 heartbeat 周期重试。
3. 每条 outbox 记录包含：
   - kind
   - payload
   - attempts
   - last_error
   - created_at
4. attempts 超阈值必须告警。

---

## 6. 必须测试清单

1. Config_SameVersion_ShouldSkip
2. Config_ForceTrue_ShouldApply
3. Config_ApplySuccess_ShouldMarkApplied
4. Task_FirstRun_ShouldExecute
5. Task_DuplicateDone_ShouldReplayAck
6. Task_DuplicateRunning_ShouldNotReexecute
7. Outbox_SendFail_ShouldRetry
8. Outbox_Reconnect_ShouldDrain
9. NodeSync_Pending_ShouldRetryOnHeartbeat
10. Tracker_PersistedVersion_ShouldSurviveRestart

集成测试：
1. WS_Disconnect_Reconnect_ShouldRecoverAcks
2. RepeatedEdgeConfig_ShouldBeIdempotent
3. DuplicateTaskDispatch_ShouldNoDuplicateSideEffect

---

## 7. 验收命令

```bash
rg -n "last_applied_version|ShouldApply|TaskIdempotency|Outbox|MarkApplied" src/Cnn.Agent
```

---

## 8. 交付格式

AI 输出必须包含：
1. 幂等键设计说明
2. 断线恢复时序说明
3. 测试结果摘要
