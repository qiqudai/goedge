# DNS 设置（功能理解）

## 功能目标
- 通过 API 对接第三方 DNS 服务商，自动生成网站 CNAME 记录。

## 支持的 DNS 提供商
- aliyun
- dnspod_cn（国内版）
- dns.com
- dns.la
- cloudflare.com
- dnsdun

## 关键能力
- 记录修复：当第三方 API 连接失败导致记录冲突时，可通过“记录修复”按钮恢复。

## 获取 ID 与 Token/Secret
### aliyun
- 进入 AccessKey 管理，创建 AccessKey。
- AccessKey ID 填入 ID，Secret 填入 Token/Secret。

### dnsdun
- 后台“账户设置 -> API 设置”，获取 UID 与 API_KEY。
- UID 填入 ID，API_KEY 填入 Token/Secret。

### dnspod_cn
- 控制台“密钥管理”，创建密钥。
- 对应填入 ID 与 Token/Secret。

### dns_com
- 账户中心 -> API 设置，获取 API Key 与 API Secret。
- API Key 填入 ID，API Secret 填入 Token/Secret。

### cloudflare_com
- 控制台“获取您的 API 令牌”，查看 Global Key。
- Global Key 用作 Token/Secret（按文档提示填写）。

## 更换 DNS 提供商
1) 在“节点管理 -> DNS 设置”更新密钥。
2) 点击“记录修复”触发修复任务。
3) 在“系统管理 -> 后台任务”查看“DNS 记录修复”任务结果。

# DNS 设置（实现思路）

## 模块设计
1) DNS 提供商管理
   - 支持多提供商类型与凭据配置。
   - 凭据加密存储，敏感字段脱敏展示。
2) 自动化记录管理
   - 为站点创建/更新 CNAME 记录。
   - 失败重试与冲突检测。
3) 记录修复任务
   - 后台任务队列执行修复。
   - 任务状态可追踪（运行中/成功/失败/原因）。

## 配置与校验流程
- 保存前对凭据进行连通性校验（可选）。
- 保存后提供“立即修复”入口。
- 提供修复结果回显与历史记录。

## UI/交互建议
- 表单字段随提供商类型动态切换。
- 提示用户到对应平台获取凭据的路径。
- 修复操作需要二次确认，避免频繁刷写。
