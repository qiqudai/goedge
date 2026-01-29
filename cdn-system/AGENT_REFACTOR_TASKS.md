# AGENT 代码优化任务清单

> 目标：在不改变业务语义的前提下，降低配置生成噪声、减少重复逻辑、拆分大文件并统一 reload 流程，提升可维护性与稳定性。

## 1. 输出稳定化（P0）

**模块**：HTTP 配置生成（`config.go` 内相关函数）

**问题**：map 无序迭代导致 `http.conf`/错误页指令/headers 顺序不稳定，触发不必要的 reload。

**改动点**：
- 对 `error_pages`、`domain.Headers`、`domain.ResponseHeaders` 统一使用排序后的 key 顺序写入。

**新增/变更函数**：
- `sortedStringKeys(m map[string]string) []string`
- `writeErrorPageDirectives` 内改为按 key 排序
- `writeProxyBlock` 内 header 写入改为按 key 排序

**伪代码**：
```
keys := sortedStringKeys(pages)
for _, key := range keys {
  status := errorPageStatusForKey(key)
  if status == 0 { continue }
  // write error_page + location
}

hdrKeys := sortedStringKeys(domain.Headers)
for _, k := range hdrKeys {
  v := domain.Headers[k]
  // sanitize + write
}
```

**风险**：低（仅输出顺序变化）。

---

## 2. Server 生成逻辑合并（P1）

**模块**：HTTP server 生成（`writeDefaultHTTPServer`/`writeDefaultHTTPSServer`/`writeHTTPSRedirectServer`/`writeHTTPServer`）

**问题**：server 级别重复写入（listen、server_name、错误页指令）多，维护成本高。

**改动点**：
- 合并默认 HTTP/HTTPS 为单一函数 `writeDefaultServer`。
- 将错误页相关的 server 级指令（`sub_filter_types`）集中在 `writeErrorPageServerDirectives`。

**新增/变更函数**：
- `writeDefaultServer(b *strings.Builder, port string, tls bool, errorPages map[string]string, errorPageDir string, status int)`
- `writeErrorPageServerDirectives(b *strings.Builder, pages map[string]string)`（已加入）
- 删除/替换 `writeDefaultHTTPServer`、`writeDefaultHTTPSServer` 调用

**伪代码**：
```
func writeDefaultServer(..., tls bool, ...) {
  writeServerBegin(listen, tls, default)
  writeErrorPageServerDirectives(...)
  writeErrorPageDirectives(...)
  writeReturn(status)
}
```

**风险**：中低（server 生成路径合并，需对比输出）。

---

## 3. Proxy 块逻辑拆分（P2）

**模块**：反代配置生成（`writeProxyBlock`）

**问题**：单函数内逻辑过长、难以复用与扩展。

**改动点**：
- 拆为“固定块 + 可选块 + cache 块”。

**新增/变更函数**：
- `writeProxyBase(b *strings.Builder)`
- `writeProxyTimeouts(b *strings.Builder, domain edgeDomain)`
- `writeProxyHeaders(b *strings.Builder, headers map[string]string, responseHeaders map[string]string)`
- `writeProxyWebsocket(b *strings.Builder, domain edgeDomain)`
- `writeProxySSL(b *strings.Builder, domain edgeDomain)`
- `writeProxyBlock` 只负责组装调用

**伪代码**：
```
writeProxyBase(b)
writeProxyHeaders(b, domain.Headers, domain.ResponseHeaders)
writeProxyWebsocket(b, domain)
writeProxyTimeouts(b, domain)
writeProxySSL(b, domain)
applyCacheDirectives(b, ...)
```

**风险**：中（需确保输出不变，尤其是指令顺序）。

---

## 4. 文件拆分（P3）

**模块**：`agent/config.go` 过大

**问题**：单文件职责过多，定位困难。

**改动点**：
- 将 HTTP/Stream 配置生成与 sanitizers 拆分到独立文件。

**新增/迁移文件**：
- `agent/http_config.go`：`writeHTTPConfig`、`writeHTTPGlobalConfig`、server 生成、error pages
- `agent/stream_config.go`：stream 相关生成与 L2 刷新
- `agent/nginx_sanitize.go`：`sanitize*` / `quoteNginxValue` 等

**数据结构**：保持原有 `edgeConfig` / `edgeDomain` / `edgeStream` 不变，仅迁移函数。

**风险**：中（迁移可能遗漏导入/引用）。

---

## 5. Reload 流程统一（P4）

**模块**：配置应用与 reload（`applyConfigPayload*`, `executeReload`, `refreshStreamConfigForL2Status`）

**问题**：不同入口 reload 路径分散，易出现行为不一致。

**改动点**：
- 统一为 `applyConfigPayloadWithOptionsAndReload(payload, skipReload)` 负责全部 reload 入口。
- L2 刷新调用统一走同一个 reload 执行函数。

**新增/变更函数**：
- `reloadNginxWithRollback()`（内部封装 reload + rollback 逻辑）
- `executeReload()` 改为调用统一入口（若已有则复用）

**伪代码**：
```
if skipReload { return }
if err := reloadNginxWithRollback(); err != nil { return err }
```

**风险**：中（reload 行为变更影响线上）

---

## 测试计划（WSL）

1. `go test ./...`（若环境具备 Go）
2. 启动 agent，触发一次 config_sync
3. 检查 `http.conf` 输出稳定性（前后 diff 无变化）
4. 验证错误页替换仍生效

```
# 示例（WSL）
cd /mnt/e/cdn/goedge/cdn-system/agent
GO111MODULE=on go test ./...
```
