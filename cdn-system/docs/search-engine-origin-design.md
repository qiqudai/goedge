# 搜索引擎回源(L回源)设计方案

## 目标与范围
- 允许站点开启“搜索引擎回源”，当请求来自搜索引擎蜘蛛时，走指定回源 IP。
- 统一维护蜘蛛 IP 列表，并同步到 agent 侧用于匹配。
- 确保“回源 IP”不对非授权用户暴露，避免源站 IP 泄露风险。

## 现状梳理
- 站点高级设置已有字段：`search_engine_origin`、`search_engine_origin_ip`。
- 后端在构建站点配置时，会根据设置拼接回源条件：
  - 优先使用蜘蛛 IP 白名单（`client_ip ip_range`）。
  - 无白名单时回退到 UA 关键词匹配。
- 白名单文件格式：`spider_ip_allowlist.json`，按搜索引擎分类存储 IP/CIDR/前缀。

## 核心设计
### 1) 配置模型
- 站点级配置（每站点）：
  - `search_engine_origin`：布尔开关。
  - `search_engine_origin_ip`：回源 IP（仅支持 IPv4/IPv6 的单 IP）。
- 全局蜘蛛 IP 允许列表（系统级）：
  - 存储于 `spider_ip_allowlist.json`，支持三类格式：
    - 单 IP：`1.2.3.4`
    - CIDR：`1.2.3.0/24`
    - 前缀：`1.2.3`（自动归一化为 `/24`）

### 2) 数据流与同步
1. 管理员在控制台维护蜘蛛 IP 列表（或上传 JSON）。
2. API 写入 `spider_ip_allowlist.json` 并触发配置版本更新（例如 `spider_ip_allowlist` 版本号）。
3. Agent 侧订阅配置版本：
   - 变更后拉取最新允许列表并落盘（`edge-node/conf/spider_ip_allowlist.json`）。
   - 运行时可基于文件 mtime 热加载，避免频繁重启。
4. 站点配置构建：
   - 若开启搜索引擎回源，且回源 IP 有效，则追加 origin condition：
     - `item=client_ip` + `operator=ip_range` + `value=<allowlist>` + `origin=<origin_ip>`。
   - 若允许列表为空，则回退到 UA 匹配（`header=user-agent` 包含规则）。

### 3) 权限与安全
- 蜘蛛 IP 列表：仅管理员可读写。
- 回源 IP：
  - 仅站点所有者/管理员可读写。
  - 列表接口返回时需脱敏（如仅在详情接口返回），避免批量泄露。
- 日志与审计：
  - 业务日志、操作日志、API 响应禁止返回回源 IP。
  - UI 提示“回源 IP 属于敏感信息”，避免误分享。

### 4) 防泄露策略
- 服务端：
  - 日志记录中删除/脱敏 `search_engine_origin_ip`。
  - 日志 API 对普通用户不返回任何源站 IP 信息。
- Agent 侧：
  - 配置文件权限限制（只读、root/agent 用户可读）。
  - 下发只包含必要的 IP 列表，不返回给控制台前端。

### 5) 失败与回退
- 未配置白名单：
  - 回退到 UA 匹配；若 UA 规则关闭或无效，则不触发回源。
- 回源 IP 不合法或为空：
  - 自动忽略回源条件，继续走默认回源。
- 允许列表加载失败：
  - 使用缓存版本，或者退回到 UA 匹配。

## API 设计建议
- 管理员 API：
  - `GET /api/v1/admin/spider_ips`：读取当前允许列表。
  - `PUT /api/v1/admin/spider_ips`：提交 JSON 全量更新。
- Agent 同步：
  - `GET /api/v1/agent/spider_ips`：基于版本号增量拉取。
  - 返回结构包含 `version` 与 `list`，便于热加载和缓存。

## UI 设计建议
- 系统管理新增“蜘蛛 IP 白名单”页面：
  - 支持编辑/导入 JSON、校验格式、显示更新时间。
- 站点高级设置：
  - 开关 + 回源 IP 输入，提示“敏感信息”。

## 测试计划
- 单元测试：
  - 解析 JSON（IP/CIDR/前缀）与匹配逻辑。
- 接口测试：
  - 允许列表更新、版本变更、回源条件注入。
- E2E：
  - 启用搜索引擎回源 -> 回源条件生效。
  - 普通用户列表接口不暴露回源 IP。

## 风险与备注
- 白名单需定期更新，建议加“最后更新时间”与可选自动抓取。
- 若误将真实源站 IP 暴露在日志或列表 API，将直接导致源站被绕过攻击。
