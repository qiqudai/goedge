# AI 实施规范：TLS/证书/SNI 热更新（无歧义版）

> 目标：在 `Cnn.Agent` 实现基于域名的证书选择与热更新，保障 HTTPS 稳定与可回滚。

---

## 0. 执行边界

### 0.1 允许修改
- `src/Cnn.Agent/Program.cs`
- `src/Cnn.Agent/Ws/AgentWsClient.cs`
- `src/Cnn.Agent/Config/AgentRuntimePaths.cs`
- `src/Cnn.Agent/Tls/*.cs`（新建）

### 0.2 禁止事项
- `MUST NOT` 改 API 协议。
- `MUST NOT` 在请求路径重复加载证书文件。

---

## 1. Definition of Done

1. HTTPS 请求可按 SNI 选择域名证书。
2. 无域名证书时使用 fallback 证书。
3. 证书更新后无需重启，立即对新连接生效。
4. 证书加载失败不影响现网可用证书。
5. 至少 8 个单元测试通过。

---

## 2. 必须新增的文件与接口

- `src/Cnn.Agent/Tls/TlsCertificateStore.cs`
- `src/Cnn.Agent/Tls/ITlsCertificateSelector.cs`
- `src/Cnn.Agent/Tls/TlsCertificateSelector.cs`
- `src/Cnn.Agent/Tls/TlsReloadResult.cs`

接口：

```csharp
namespace Cnn.Agent.Tls;

public interface ITlsCertificateSelector
{
    X509Certificate2? Select(string? serverName);
    TlsReloadResult Reload(Cnn.Api.Contracts.Agent.EdgeConfigDto config, string certDir);
}
```

---

## 3. Program.cs 强制改造

1. `MUST` 配置 Kestrel HTTPS 回调，使用 `ITlsCertificateSelector.Select(serverName)`。
2. `MUST` 注册 `ITlsCertificateSelector`。
3. `MUST` 保留 HTTP 监听能力。

---

## 4. 证书来源与优先级

1. 域名证书：`domain.ssl_cert_data` + `domain.ssl_key_data` 或 `ssl_cert_path` + `ssl_key_path`。
2. fallback：`fallback_cert_data` + `fallback_key_data`（或 certDir fallback 文件）。
3. 优先级：域名证书 > fallback。

---

## 5. Reload 规则

1. `AgentWsClient.ApplyConfigPayloadAsync()` 成功后 `MUST` 调用 TLS Reload。
2. Reload 采用双缓冲：
   - 先加载新证书集合
   - 加载成功后一次性替换引用
3. 任何域名证书加载失败：
   - 记录 warning
   - 不中断其他证书加载
4. fallback 加载失败：
   - 若旧 fallback 存在，继续用旧 fallback
   - 若不存在，返回 fail 并保留旧集合

---

## 6. 性能与稳定性约束

1. `Select()` 路径 `MUST` O(1) 字典查找。
2. 证书对象不得每请求 new。
3. `Reload` 期间不得阻塞正常请求。
4. 单次 reload 耗时目标：< 1s（1000 证书内）。

---

## 7. 必须测试清单

1. Select_DomainCert_ShouldReturnDomain
2. Select_UnknownDomain_ShouldReturnFallback
3. Reload_NewDomainCert_ShouldTakeEffect
4. Reload_BadDomainCert_ShouldKeepOld
5. Reload_BadFallback_WithOldFallback_ShouldKeepOld
6. Select_NullServerName_ShouldUseFallback
7. Concurrent_Select_And_Reload_ShouldStable
8. CertPath_And_CertData_Priority_ShouldCorrect

---

## 8. 验收命令

```bash
rg -n "ServerCertificate|SNI|ITlsCertificateSelector|Reload\(" src/Cnn.Agent
```

---

## 9. 交付格式

AI 输出必须包含：
1. 证书优先级策略说明
2. reload 失败回退证明
3. 单测结果
