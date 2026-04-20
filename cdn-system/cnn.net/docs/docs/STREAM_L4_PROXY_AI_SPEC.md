# AI 实施规范：L4 TCP/UDP 转发（无歧义版）

> 目标：补齐 `EdgeConfig.streams` 数据面能力，支持动态监听、转发、健康与限流。

---

## 0. 执行边界

### 0.1 允许修改
- `src/Cnn.Agent/Program.cs`
- `src/Cnn.Agent/Ws/AgentWsClient.cs`
- `src/Cnn.Agent/Stream/*.cs`（新建）

### 0.2 禁止事项
- `MUST NOT` 与 HTTP YARP 共享端口。
- `MUST NOT` 在热更新时中断无关端口监听。

---

## 1. Definition of Done

1. 支持按配置动态开启/关闭 TCP 监听。
2. 支持上游目标池与基础负载（round robin）。
3. 支持连接超时与空闲超时。
4. 支持热更新：仅重载受影响 stream。
5. 至少 8 个单测 + 2 个压力测试脚本。

---

## 2. 必须新增文件与接口

- `src/Cnn.Agent/Stream/StreamProxyHost.cs`
- `src/Cnn.Agent/Stream/StreamRuntime.cs`
- `src/Cnn.Agent/Stream/StreamListener.cs`
- `src/Cnn.Agent/Stream/StreamRouteCompiler.cs`
- `src/Cnn.Agent/Stream/StreamSession.cs`

接口：

```csharp
namespace Cnn.Agent.Stream;

public interface IStreamRuntime
{
    StreamApplyResult Apply(Cnn.Api.Contracts.Agent.EdgeConfigDto config);
    IReadOnlyCollection<StreamListenerState> GetStates();
}
```

---

## 3. 行为规则

1. `streams` 为空时，不启动任何 L4 监听。
2. 同一 `listen_ip:listen_port:protocol` 不允许重复。
3. 上游为空时该监听不生效并记录错误。
4. 热更新时按 diff：
   - 新增 -> 启动 listener
   - 删除 -> 优雅停止 listener
   - 修改 -> 重建该 listener

---

## 4. 性能与稳定性规则

1. `MUST` 使用 `SocketAsyncEventArgs` 或等效异步模型。
2. `MUST` 使用 buffer pool，禁止每连接分配大块内存。
3. 默认限制：
   - `max_conns_per_listener`
   - `idle_timeout_seconds`
   - `connect_timeout_ms`
4. 必须暴露当前活跃连接数指标。

---

## 5. 最小配置结构（必须支持）

```json
{
  "streams": [
    {
      "id": 1001,
      "protocol": "tcp",
      "listen_ip": "0.0.0.0",
      "listen_port": 3306,
      "upstream_key": "stream_up_1001",
      "balance": "round_robin"
    }
  ]
}
```

---

## 6. 必须测试清单

1. Apply_EmptyStreams_ShouldStopAll
2. Apply_NewStream_ShouldStartListener
3. Apply_DeleteStream_ShouldStopListener
4. Apply_ModifiedStream_ShouldRestartOne
5. DuplicateListen_ShouldReject
6. NoUpstream_ShouldSkipListener
7. ConnectionIdleTimeout_ShouldClose
8. ActiveConnectionsMetric_ShouldUpdate

压力脚本：
1. 10k 并发连接 10 分钟稳定脚本
2. 热更新期间连接存活脚本

---

## 7. 验收命令

```bash
rg -n "IStreamRuntime|StreamListener|SocketAsyncEventArgs|Apply\(" src/Cnn.Agent
```

---

## 8. 交付格式

AI 输出必须包含：
1. diff 重载策略说明
2. 连接生命周期说明
3. 压测结果摘要
