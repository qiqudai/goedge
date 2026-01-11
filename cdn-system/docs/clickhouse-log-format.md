# ClickHouse 节点日志/指标上报说明

本说明用于对接节点 Lua/WAF 日志与 ClickHouse 写入格式。

## 1. 访问日志（Access Logs）
接口：`POST /api/v1/agent/logs/access`

请求头：
- `Authorization: Bearer <agent_token>`
- `Content-Type: application/json`

请求体：
```json
{
  "node_id": "WIN-CCB276277K5",
  "node_ip": "172.20.9.140",
  "lines": [
    "{\"time_iso8601\":\"2025-12-27T12:30:00+08:00\",\"remote_addr\":\"1.2.3.4\",\"host\":\"www.example.com\",\"request\":\"GET /index.html HTTP/1.1\",\"status\":200,\"body_bytes_sent\":1024,\"request_time\":0.12,\"upstream_addr\":\"10.0.0.2:80\",\"upstream_response_time\":\"0.11\",\"upstream_cache_status\":\"HIT\",\"http_referer\":\"-\",\"http_user_agent\":\"Mozilla/5.0\",\"scheme\":\"http\",\"ssl_protocol\":\"\",\"ssl_cipher\":\"\"}"
  ]
}
```

说明：
- `lines` 为 Nginx JSON 日志原始行数组（字符串，不要再次 JSON 编码）。
- 插入 ClickHouse 表：`node_access_logs`。
- `request` 字段会解析出 `method` 与 `uri`。

## 2. 监控指标（Metrics）
接口：`POST /api/v1/agent/logs/metrics`

请求体：
```json
{
  "node_id": "WIN-CCB276277K5",
  "node_ip": "172.20.9.140",
  "content": "# HELP edge_requests_total Total requests\n# TYPE edge_requests_total counter\nedge_requests_total{host=\"www.example.com\"} 1234\nedge_bytes_total{host=\"www.example.com\"} 12345678\n"
}
```

说明：
- `content` 为 Prometheus 文本格式（可直接抓取 `http://127.0.0.1:9100/metrics`）。
- 插入 ClickHouse 表：`node_metrics`。

## 3. 节点事件/攻防日志（Events/WAF）
接口：`POST /api/v1/agent/logs/events`

请求体：
```json
{
  "node_id": "WIN-CCB276277K5",
  "node_ip": "172.20.9.140",
  "type": "waf_block",
  "payloads": [
    "{\"time\":\"2025-12-27 12:31:00\",\"ip\":\"1.2.3.4\",\"host\":\"www.example.com\",\"uri\":\"/login\",\"rule\":\"cc\",\"action\":\"block\",\"reason\":\"rate_limit\"}"
  ]
}
```

说明：
- `type` 用于区分事件类型（如 `waf_block`、`cc_block`、`acl_deny` 等）。
- `payloads` 是字符串数组，每条可以是 JSON 字符串或纯文本。
- 插入 ClickHouse 表：`node_events`。

## 4. 访问日志查询参数（CK）
接口：`GET /api/v1/admin/logs/access`

新增支持：
- `domain_mode=fuzzy`（域名模糊匹配）
- `port`（按端口筛选）
- `node_ip`、`scheme`、`cache_status`、`referer`、`user_agent`、`upstream_addr`
- `ssl_protocol`、`ssl_cipher`
- `status_min`、`status_max`
- `keyword`（host/uri/remote_addr 模糊）

示例：
```
/api/v1/admin/logs/access?domain=example.com&domain_mode=fuzzy&status_min=400&status_max=499&page=1&pageSize=20
```
