# Agent 任务载荷契约（AI 可直接调用）

## 1. 通用 envelope
```json
{
  "kind": "task_dispatch",
  "msg_id": "uuid",
  "task": {
    "task_id": 123,
    "task_type": "config_sync",
    "task_name": "...",
    "payload": "...json string..."
  }
}
```

## 2. `config_sync`
- payload: `EdgeConfigDto` JSON 字符串
- ack:
  - `success + ret`（JSON，含应用摘要）
  - `fail + error=config apply failed`
  - `applied`（推荐结构）:
```json
{
  "old_version": 100,
  "new_version": 101,
  "force": true,
  "reason": "version_not_newer",
  "stream": {
    "success": true,
    "received": 3,
    "planned": 3,
    "applied": 2,
    "skipped": 1,
    "skip_reasons": ["compile_errors"]
  }
}
```
- API 侧 ACK 审计字段（`task_type=config_sync`）会校验并落库：
  - `streams_received`
  - `streams_applied`
  - `streams_skipped`
  - `streams_reason`

## 3. `debug_switch` / `debug_log_switch`
### 3.1 批量格式
```json
{
  "switches": {
    "ship_access_logs": true,
    "ship_metrics": false,
    "manual_debug_log": true
  }
}
```
### 3.2 单项格式
```json
{
  "key": "ship_stream_logs",
  "enabled": false
}
```
### 3.3 ack ret
```json
{
  "applied": 2,
  "updated": {"ship_metrics": false},
  "current": {"ship_metrics": false, "ship_access_logs": true}
}
```

## 4. `manual_debug_log` / `debug_log_write`
```json
{
  "category": "routing",
  "message": "apply route check",
  "data": {"version": 1024, "host": "a.example.com"}
}
```
- `message` 必填。
- 开关 `manual_debug_log=false` 时写入被忽略但任务可返回 success（幂等友好）。

## 5. 现有任务保留
- `issue_cert`
- `refresh_url`
- `refresh_dir`
- `clear_cache`
- `preheat`
- `agent_upgrade`
- `sync_package`（及兼容别名）

## 6. 错误码建议
- `invalid payload`
- `message is required`
- `config apply failed`
- `permission denied`（API 层）

## 7. 兼容性要求
- 所有任务类型名大小写不敏感。
- payload 允许字段扩展，不应因未知字段失败。
- 解析失败返回 `fail`，不可 panic。
