# 日志读写框架 AI 实施规范（Logging Read/Write Pipeline AI Spec）

## 1. 目标
- 建立统一日志写入与查询框架，替代各模块分散写文件模式。
- 支持调试开关、人工调试日志、异步批量写、可回压。

## 2. 日志通道
- `access`
- `stream_access`
- `security`
- `system`
- `debug`
- `manual_debug`
- `metrics`

## 3. 统一事件模型
```csharp
public sealed record LogEvent(
    DateTimeOffset Timestamp,
    string Channel,
    string Level,
    string Event,
    string TraceId,
    IReadOnlyDictionary<string, object?> Fields);
```

## 4. 写入架构
- `ILogEventWriter`：业务入口。
- `ILogPipeline`：`Channel<LogEvent>` 缓冲队列。
- `ILogSink`：目标写入（file/clickhouse/ws ship）。

### 4.1 回压策略
- 队列满时按通道策略：
  - `access`、`metrics`：可丢弃最旧并计数。
  - `security`、`system`：优先保留，阻塞阈值可配。
  - `manual_debug`：受开关控制。

### 4.2 批量参数（默认）
- `batch_size=512`
- `flush_interval_ms=1000`
- `max_queue=200000`

## 5. 读取架构
- `ILogQueryService`
- 过滤条件：`from/to/channel/level/trace_id/node_id/host/status`
- 输出统一分页结构。

## 6. 调试开关与人工调试日志
- 开关源：`debug_switches.json`（热加载 + 原子替换）。
- 人工调试日志文件：`manual_debug.jsonl`。
- 任务入口：
  - `debug_switch|debug_log_switch`
  - `manual_debug_log|debug_log_write`

## 7. 数据治理
- 脱敏字段：`token`、`authorization`、`cookie`、`password`、`secret`。
- 保留策略：
  - `access` 7~30 天
  - `security` 30~90 天
  - `manual_debug` 7~14 天
- 归档策略：按日滚动 + 压缩。

## 8. 性能边界
- 写入路径不得阻塞主请求链路。
- 单节点日志写入能力目标：20k lines/s（JSONL 本地盘）。
- 磁盘高水位（>85%）时触发降级：
  - 降低 debug 级别写入
  - 提前清理短保留通道

## 9. 硬件建议（单节点）
- CPU：8C 起
- 内存：16GB 起
- 磁盘：NVMe 500GB 起
- 日志高峰写：建议独立分区，避免影响缓存盘

## 10. AI 实施步骤
1. 抽象日志接口与模型。
2. 实现 Channel Pipeline。
3. 实现 File Sink。
4. 挂接 WS shipper。
5. 接入调试开关与人工日志。
6. 增加查询服务与 API。
7. 增加留存/压缩清理作业。

## 11. 验收标准
- 开关可热更新且即时生效。
- 人工日志任务可写入并可查询。
- 10k lines/s 压测下无明显阻塞，丢弃计数可观测。
- 故障场景（磁盘满、sink 异常）有明确降级行为。
