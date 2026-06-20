# AGENT ´úÂëÓŻ挝袂嵥

> Ŀ±꣺Ôڲ桓ıäҵ务语义的ǰÌáÏ£档团渲蒙稍肷⒓跎僦ظ绰߼⒉鸱ִ笪ļþ²¢ͳһ reload Á÷³̣¬ÌáÉý¿Éά»¤ÐÔÓëÎȶㄐԡ£

## 1. Êä³öÎȶ¨»¯£¨P0）

**ģ块**：HTTP ÅäÖÃÉú³ɣ¨`config.go` ÄÚÏà¹غ¯Êý£©

**问题**：map 无序迭代导致 `http.conf`/错误ҳָ令/headers ˳Ðò²»Îȶ¨£¬´¥·¢²»±ØҪ的 reload。

**¸Ķ¯µã**：
- 对 `error_pages`、`domain.Headers`、`domain.ResponseHeaders` ͳһʹ用排序后的 key ˳序дÈ롣

**新增/变更函数**：
- `sortedStringKeys(m map[string]string) []string`
- `writeErrorPageDirectives` ÄڸÄΪ按 key 排序
- `writeProxyBlock` 内 header д入改Ϊ按 key 排序

**α代码**：
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

**风险**£ºµͣ¨½öÊä³ö˳Ðò±仯）。

---

## 2. Server Éú³ÉÂ߼ϲ¢£¨P1）

**ģ块**：HTTP server Éú³ɣ¨`writeDefaultHTTPServer`/`writeDefaultHTTPSServer`/`writeHTTPSRedirectServer`/`writeHTTPServer`）

**问题**：server ¼¶±ðÖظ´дÈ루listen、server_name、错误ҳָÁ¶࣬ά»¤³ɱ靖ߡ£

**¸Ķ¯µã**：
- ºϲ¢Ĭ认 HTTP/HTTPS Ϊ单һ函数 `writeDefaultServer`。
- 将错误ҳÏà¹صÄ server 级ָÁ`sub_filter_types`）集中在 `writeErrorPageServerDirectives`。

**新增/变更函数**：
- `writeDefaultServer(b *strings.Builder, port string, tls bool, errorPages map[string]string, errorPageDir string, status int)`
- `writeErrorPageServerDirectives(b *strings.Builder, pages map[string]string)`£¨ÒѼ尤룩
- ɾ除/Ì滻 `writeDefaultHTTPServer`、`writeDefaultHTTPSServer` 调用

**α代码**：
```
func writeDefaultServer(..., tls bool, ...) {
  writeServerBegin(listen, tls, default)
  writeErrorPageServerDirectives(...)
  writeErrorPageDirectives(...)
  writeReturn(status)
}
```

**风险**£ºÖеͣ¨server 生成·¾¶ºϲⅲ瓒ԱÈÊä³ö£©¡£

---

## 3. Proxy ¿éÂ߼鸱֣¨P2）

**ģ块**£º·´´úÅäÖÃÉú³ɣ¨`writeProxyBlock`）

**问题**£ºµ¥º¯ÊýÄÚÂ߼ぁ⒛岩Ը´ÓÃÓëÀ©չ。

**¸Ķ¯µã**：
- 拆Ϊ¡°¹̶¨¿é + 可ѡ块 + cache ¿顱。

**新增/变更函数**：
- `writeProxyBase(b *strings.Builder)`
- `writeProxyTimeouts(b *strings.Builder, domain edgeDomain)`
- `writeProxyHeaders(b *strings.Builder, headers map[string]string, responseHeaders map[string]string)`
- `writeProxyWebsocket(b *strings.Builder, domain edgeDomain)`
- `writeProxySSL(b *strings.Builder, domain edgeDomain)`
- `writeProxyBlock` ֻ负责组װ调用

**α代码**：
```
writeProxyBase(b)
writeProxyHeaders(b, domain.Headers, domain.ResponseHeaders)
writeProxyWebsocket(b, domain)
writeProxyTimeouts(b, domain)
writeProxySSL(b, domain)
applyCacheDirectives(b, ...)
```

**风险**£ºÖУ¨Ðèȷ±£Êä³ö²»±䣬尤其是ָ令˳Ð򣩡£

---

## 4. Îļ鸱֣¨P3）

**ģ块**：`agent/config.go` 过大

**问题**£ºµ¥ÎļþְÔð¹ý¶࣬定λÀ§Äѡ£

**¸Ķ¯µã**：
- 将 HTTP/Stream 配置生成与 sanitizers ²ð·ֵ蕉懒⑽ļþ¡£

**新增/ǨÒÆÎļþ**：
- `agent/http_config.go`：`writeHTTPConfig`、`writeHTTPGlobalConfig`、server Éú³ɡ¢error pages
- `agent/stream_config.go`：stream 相关生成与 L2 ˢ新
- `agent/nginx_sanitize.go`：`sanitize*` / `quoteNginxValue` 等

**Êý¾ݽṹ**：保持ԭ有 `edgeConfig` / `edgeDomain` / `edgeStream` ²»±䣬仅ǨÒƺ¯Êý¡£

**风险**£ºÖУ¨ǨÒƿÉÄÜÒÅ©导入/ÒýÓã©¡£

---

## 5. Reload 流程ͳһ（P4）

**ģ块**：配置Ӧ用与 reload（`applyConfigPayload*`, `executeReload`, `refreshStreamConfigForL2Status`）

**问题**：不ͬ入口 reload ·径分ɢ£¬Ò׳öÏÖÐÐΪ不һÖ¡£

**¸Ķ¯µã**：
- ͳһΪ `applyConfigPayloadWithOptionsAndReload(payload, skipReload)` 负责ȫ部 reload Èë¿ڡ£
- L2 ˢÐµ÷ÓÃͳһ走ͬһ个 reload ִÐк¯Êý¡£

**新增/变更函数**：
- `reloadNginxWithRollback()`£¨Äڲ¿·âװ reload + rollback Â߼­£©
- `executeReload()` 改Ϊ调用ͳһÈë¿ڣㄈ粢延性蚋从ã©

**α代码**：
```
if skipReload { return }
if err := reloadNginxWithRollback(); err != nil { return err }
```

**风险**£ºÖУ¨reload 行Ϊ变更ӰÏìÏßÉϣ©

---

## ²âÊԼƻ®£¨WSL）

1. `go test ./...`£¨Èô»·¾³¾߱¸ Go）
2. 启动 agent，触发һ次 config_sync
3. 检查 `http.conf` Êä³öÎȶㄐԣ¨ǰ后 diff Îޱ仯）
4. 验֤错误ҳÌ滻仍生Ч

```
# ʾ例（WSL）
cd /mnt/e/cdn/goedge/cdn-system/agent
GO111MODULE=on go test ./...
```
