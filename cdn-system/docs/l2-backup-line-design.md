# L2 回源节点与备用线路设计（基于现有代码与配置）

## 目的
- 说明当前系统中 L2 节点与备用线路（备用分组/备用 IP/默认线路备份）的数据模型、配置入口与生效路径。
- 标明已实现与待补齐的环节，避免误解。

## L2 回源节点
### 现有数据模型
- 节点层级：`models.Node.Level`（1=边缘/L1，2=L2）。
- 节点组：`models.NodeGroup` 中 `L2Config` 存储在 `backup_switch_policy` 的 JSON 字段中。
- 站点高级配置：`settings.l2_config`（`current`/`none`/`custom`）。
- 套餐字段：`models.Plan.L2Origin`（当前仅字段保留，代码中未直接使用）。

### 现有接口与同步流程
- Agent 获取 L2 节点列表：`GET /api/v1/agent/l2/nodes`。
  - 逻辑：
    1) 读取 L1 节点所在 `line` 的 `node_group_id`。
    2) 查询相同组内 `level=2` 且启用的节点作为 L2 节点。
- Agent 上报 L2 节点在线心跳：`POST /api/v1/agent/l2/heartbeat`。
- WebSocket 同步：`l2_nodes_request` / `l2_nodes_response` / `l2_heartbeat`。
- Agent 侧定时 L2 健康检测：`L2_CHECK_INT`（默认 30s），使用 `check_protocol/check_port/check_host/check_path` 探活。

### 现状结论
- 已具备 L2 节点发现、健康检测与心跳上报。
- L2 配置项已在节点组与站点设置中存在。
- 目前配置下发与 Nginx 上游是否真正使用 L2，代码中未看到明确的回源链路拼装逻辑（需补齐时要新增 upstream 配置与切换策略）。

### 建议落地步骤（待补齐）
1) 在配置生成服务中生成 L2 upstream（按节点组输出 L2 节点列表）。
2) L1 回源改为优先走 L2 upstream；L2 再回源到源站。
3) `l2_config` 规则：
   - `none`：绕过 L2，直接源站。
   - `current`：走套餐/节点组默认 L2。
   - `custom`：站点配置指定 L2 组（需新增字段/映射）。
4) 失败回退策略：L2 不可用时直连源站或切备用 L2 组。

## 备用线路/备用节点组
### 现有数据模型
- 站点层：`models.Site.BackupNodeGroupID`、`EnableBackupGroup`。
- 用户套餐：`models.UserPackage.BackupNodeGroup`、`EnableBackup`。
- 套餐计划：`models.Plan.BackupGroup`。
- 线路明细：`models.Line.IsBackup`、`IsBackupDefaultLine` 等。
- 节点组切换策略：`NodeGroup.BackupSwitchType`（前端字段 `spare_ip_switch`，值 1/2/3）。

### 现有生效逻辑
- `services/package_dns_sync.go` 中 `resolveSiteGroups`：
  - 主线路组优先级：站点 > 用户套餐 > 计划。
  - 备用线路组生效条件：`EnableBackupGroup` 为 true。
  - 备用组优先级：站点 > 用户套餐 > 计划。
- 当启用备用线路组时，会触发相关解析/配置同步流程。

### 线路备份能力（节点/线路维度）
- 节点组解析页可：
  - 批量设置“备用 IP”。
  - 设置“备用默认解析”。
- 切换策略（节点组级）：
  - `1`：有主 IP 下线时切到备用。
  - `2`：在线主 IP 数少于备用 IP 数时切换。
  - `3`：按间隔切换（轮转）。

### 现状结论
- 备用线路组/备用 IP 已有完整字段与控制入口。
- 站点/套餐/计划的备用组优先级已在同步服务中实现。
- 备用 IP 切换策略依赖节点组 `spare_ip_switch`，并存于 `backup_switch_policy`。

## 配置入口（前端）
- 节点组列表：`nodes/groups/List.vue`（L2 配置、备用 IP 切换策略）。
- 节点组解析：`nodes/groups/Resolution.vue`（备用 IP、备用默认解析）。
- 网站批量设置：`website/list/BatchEditDialog.vue`（备用线路组）。
- 套餐配置：`plans/Basic.vue`、`plans/Sold.vue`（备用分组）。
- 网站高级设置：`manage/AdvancedConfig.vue`（L2 配置）。

## 监控与告警
- 监控选项包含：`backup_ip`、`backup_default_line`、`backup_group`。
- 建议：当备用线路触发时打标并记录到备份日志（已有备份日志通道 `/logs/backup`，可扩展）。

## 风险与注意事项
- L2 回源尚未看到完整回源链路落地，需确认配置下发与模板是否接入。
- 备用线路组与主线路组不可重复（前端已有校验）。
- 备用 IP 切换策略需与线路健康探测保持一致，避免频繁抖动。

## 后续建议
- 补齐 L2 upstream 生成与回源切换逻辑。
- 为 L2/备用线路增加状态面板与回源命中统计。
- 对备用切换引入速率限制或抖动保护（避免频繁切换）。
