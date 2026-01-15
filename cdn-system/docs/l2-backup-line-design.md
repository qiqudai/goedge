# L2 回源节点与备用线路设计

## 目标
- 当套餐/站点配置开启 L2 层代理时：L1 缓存未命中先请求 L2；L2 未命中再回源站。
- L2 节点同时支持网站与 TCP/UDP 转发，L1/L2 使用同一套 agent 逻辑，配置决定节点是 L1 还是 L2。
- L2 不可用时自动回退：可选 L2 节点都不可用则直连源站。

## L2 回源节点
### 角色与配置
- 节点层级：`models.Node.Level`（1=边缘/L1，2=L2）。
- 同一套 agent：节点角色由配置确定（L1/L2），逻辑共用。
- 节点组：`models.NodeGroup` 中 `L2Config` 存储在 `backup_switch_policy` 的 JSON 字段中。
- 套餐/实例开关：`package.l2_origin` -> `user_package.l2_origin`。
- 站点高级配置：`settings.l2_config`（`current`/`none`/`custom`），`current` 跟随套餐开关，`custom` 强制开启（仍受线路组配置影响）。
- 线路组 L2 配置：`models.NodeGroup` 的 `backup_switch_policy.l2_config`（`none` 禁用，其它为启用）。
- 区域 L2 检测端口：区域配置 `region_meta`（config 表）中的 `l2_check_port`，默认 80。
  - 前端入口：节点列表新增区域时设置（`nodes/list/RegionList.vue`）。

### 节点发现与同步
- Agent 获取 L2 节点列表：`GET /api/v1/agent/l2/nodes`。
  - 逻辑：读取 L1 节点所在 `line` 的 `node_group_id`，查询相同组内 `level=2` 且启用的节点作为 L2 节点。
- WebSocket 同步：`l2_nodes_request` / `l2_nodes_response` / `l2_heartbeat`。
- L1 agent 需要在节点/线路变更后及时刷新 L2 列表（WS 推送或周期拉取），保证配置变更能快速生效。

### 回源链路
#### 网站/HTTP
- L1 缓存命中：直接响应，`X-Cache-Status: HIT from L1:{id}`。
- L1 缓存未命中且开启 L2：
  1) L1 选择可用 L2 节点并发起请求。
  2) L2 缓存命中：直接返回，L1 返回结果。
  3) L2 缓存未命中：L2 回源站，结果返回 L1，再返回用户。
- L2 未开启或无可用 L2：L1 直接回源站。

#### TCP/UDP 转发
- L1 在 OpenResty stream 转发层使用 L2 作为中转。
- L2 执行二次转发到真实源站，链路为 L1 → L2 → 源站。
- L2 目标列表与网站回源共用，同组 L2 先作为主转发，源站作为备份回源。

### L2 健康检测与切换
- 每个 L1 节点每 10 秒探测每个 L2 节点连通性。
- 默认 TCP 探测 L2 的 80 端口，端口可在区域 `l2_check_port` 中修改。
- 连续 3 次不通：该 L1 视该 L2 下线；连续 3 次正常：恢复使用。
- 监控程序将 L2 节点禁用时，L1 自动剔除该 L2；若全部 L2 不可用则直连源站。

### L2 使用判定
- 响应头：
  - `X-Cache-Status: HIT from L1:7` 表示 L1 命中。
  - `X-Cache-Status: HIT from L2:8, MISS from L1:7` 表示 L1 未命中但 L2 命中。
  - `Via: L1:7, L2:8` 表示请求链路经过的节点。
- 后台访问日志：L2 IP 列不为空表示连接了 L2 节点。

### L2 连接 IP 选择
- L1 使用 CDN 后台节点管理中 L2 节点填写的 IP。
- 若 L2 节点存在多个 IP，使用第一个 IP 作为连接 IP。

## 备用线路/备用节点组
### 现有数据模型
- 站点层：`models.Site.BackupNodeGroupID`、`EnableBackupGroup`。
- 用户套餐：`models.UserPackage.BackupNodeGroup`、`EnableBackup`。
- 套餐计划：`models.Plan.BackupGroup`。
- 线路明细：`models.Line.IsBackup`、`IsBackupDefaultLine` 等。
- 节点组切换策略：`NodeGroup.BackupSwitchType`（前端字段 `spare_ip_switch`，值 1/2/3）。

### 生效逻辑
- `services/package_dns_sync.go` 中 `resolveSiteGroups`：
  - 主线路组优先级：站点 > 用户套餐 > 计划。
  - 备用线路组生效条件：`EnableBackupGroup` 为 true。
  - 备用组优先级：站点 > 用户套餐 > 计划。

### 自动切换（备用线路组）
- 备用线路用于主线路组整体不可用后的自动切换。
- API 自动检测主线路组全部掉线后，切换套餐 CNAME 指向备用线路组。
- 只修改套餐 CNAME 指向，不修改线路组内的 CNAME/IP。
- 常用于高防场景：主线路被攻击时切换到高防线路。
- 主线路确认连续 10 次连接正常后，自动恢复套餐 CNAME 指向主线路组。

### 线路备份能力（节点/线路维度）
- 节点组解析页可：批量设置“备用 IP”、设置“备用默认解析”。
- 切换策略（节点组级）：
  - `1`：有主 IP 下线时切到备用。
  - `2`：在线主 IP 数少于备用 IP 数时切换。
  - `3`：按间隔切换（轮转）。

## 监控与告警
- 监控选项包含：`backup_ip`、`backup_default_line`、`backup_group`。
- 建议：当备用线路触发时打标并记录到备份日志（已有备份日志通道 `/logs/backup`）。

## 验收/测试
- 套餐 L2 开关、站点 L2 配置（`settings.l2_config`）与线路组 L2 配置能下发到 agent，L1 在缓存未命中时走 L2。
- L1 agent 能及时同步 L2 节点变更（WS/拉取），并按 10 秒探测逻辑上下线。
- L2 全部不可用时自动回源源站，不再使用 L2。
- 网站回源可通过 `X-Cache-Status`/`Via` 及访问日志验证是否使用 L2。
- 备用线路组可在主线路组全部掉线后自动切换、恢复。

## 注意事项
- 网站缓存链路与 TCP/UDP 转发链路使用同一组 L2 节点，但实现层分别在 HTTP 与 stream 转发层。
- 备用线路组与主线路组不可重复（前端已有校验）。
