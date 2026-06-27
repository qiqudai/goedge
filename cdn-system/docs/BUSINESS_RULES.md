# CDN 业务规则手册

> **AI 必读**：修改删除逻辑、配置合并、套餐/默认配置、CC/ACL、节点线路相关代码前，必须先阅读本文档并保持一致。  
> 后端校验实现集中在 `api/services/reference_guard_service.go` 与 `api/services/default_config_service.go`。

---

## 1. 配置层级与优先级

### 1.1 站点运行时配置（边缘生效）

优先级从高到低：

```
站点已保存的设置  >  网站分组默认  >  用户全局默认  >  平台全局默认模板
```

对应函数：

| 层级 | 函数 | 说明 |
|------|------|------|
| 站点自身 | `site.settings` / 实体字段 | 最高优先级，永不被默认覆盖 |
| 网站分组默认 | `GetSiteDefaultMapWithGroup(userID, groupID)` | `scope_name=group` |
| 用户全局默认 | 同上，`scope_name=global, scope_id=user_id` | website/list「范围=全局」 |
| 平台模板 | `GetGlobalDefaultConfig()` + `ApplySiteTemplateDefaultsByType` | 仅建站或空字段时填充 |

**关键约束**：

- 所有默认填充必须使用 `setIfMissing`，不得覆盖站点已有字段。
- 边缘配置生成（`config_service.GenerateConfigForNode`）必须使用 `GetSiteDefaultMapWithGroup(site.UserID, groupID)`，其中 `groupID` 来自 `LoadSiteDefaultGroupMap`（取站点第一个网站分组）。
- `settings/Global.vue` 的 WAF（`blacklist_timeout` 等）是**节点级全局 WAF**，与站点 `security_black_time` / `guard_block_ttl` **不是同一配置链**。

### 1.2 网站默认配置「范围」语义

| UI 显示 | 存储 | 生效对象 |
|---------|------|----------|
| 全局 | `site_default_config`, `scope_name=global`, `scope_id=user_id` | 该账号下所有网站（字段为空时） |
| 分组(xxx) | `site_default_config`, `scope_name=group`, `scope_id=group_id` | 仅该网站分组内站点 |

合并顺序（后者覆盖前者）：`平台(scope_id=0)` → `user legacy` → `用户全局` → `分组`。

### 1.3 套餐与全局配置关系

```
全局系统配置 (SysConfig global_config)
    ↓ 节点级 WAF/资源/Nginx
平台套餐模板 (Package)
    ↓ 用户购买
已购套餐 (UserPackage) ──绑定──> 线路分组(NodeGroup)
    ↓
网站 (Site) ── user_package 字段关联已购套餐
```

| 配置类型 | 存储 | 作用 |
|----------|------|------|
| 全局配置 | `SysConfig` name=`global_config` | 边缘节点 WAF、Nginx、错误页、默认站点模板 |
| 套餐模板 | `Package` | 可售套餐定义、默认线路分组 |
| 已购套餐 | `UserPackage` | 用户实例、到期、线路分组、域名额度 |
| 网站默认 | `ConfigItem` type=`site_default_config` | 建站/空字段默认值 |
| 站点设置 | `Site.settings` JSON | 单站最终业务配置 |

**禁止**：删除仍有网站引用的已购套餐；删除仍有购买记录的套餐模板。

---

## 2. 删除 / 禁用安全规则

> 原则：**先解除引用 → 再禁用 → 最后删除**。所有规则必须**后端强制校验**，前端仅作预检与友好提示。

### 2.1 网站 (Site)

| 操作 | 规则 | 后端 | 前端 |
|------|------|------|------|
| 删除 | 必须先 `enable=false` | `site_admin.AdminBatchAction` case `delete` | `List.vue` 预检 + 提示 |
| 禁用 | 无额外约束 | 同上 | — |

错误码：`Please disable site before delete`

### 2.2 网站分组 (SiteGroup)

| 操作 | 规则 | 后端 | 前端 |
|------|------|------|------|
| 删除 | 分组内**不能有任何网站** | `CountSiteGroupMembers` | `Groups.vue` 确认文案 |

错误码：`site_group.has_members`

**禁止**静默删除分组并解除关联而不报错。

### 2.3 节点 (Node)

| 操作 | 规则 | 后端 | 前端 |
|------|------|------|------|
| 删除 | **不能有任何线路引用**（`Line.node_id` / `node_ip_id`） | `hasLineBindings` | `nodes/List.vue` 预检 `line_count` |

错误码：`node.delete_in_use`

**操作顺序**：先从线路分组移除节点 → 再删节点。

### 2.4 线路 (Line)

| 操作 | 规则 | 后端 | 前端 |
|------|------|------|------|
| 删除 | 必须先 `enable=false` | `HasEnabledLines` | `Resolution.vue` 预检 `is_on` |

错误码：`line.delete_disable_first`

**操作顺序**：禁用线路 → 删除线路 → 才能删节点。

### 2.5 线路分组 (NodeGroup)

| 操作 | 规则 | 后端 |
|------|------|------|
| 删除 | 无线路；无套餐/已购套餐引用；**无 L3 节点 `parent_group_id` 引用** | `DeleteNodeGroup` |

错误码：`node_group.has_nodes`, `node_group.has_packages`, `node_group.has_parent_fetch_refs`

**L3 父节点组引用**：`node.level=3` 且 `parent_fetch_mode` 为 `l1`/`l2` 时，`node_config.parent_group_id` 指向父层线路分组。删除该分组前须先解除所有 L3 节点的父层绑定（改为 `origin` 或更换分组）。实现：`CountParentGroupReferences`（`parent_fetch_service.go`）。

### 2.6 证书 (Cert)

| 操作 | 规则 | 后端 | 前端 |
|------|------|------|------|
| 禁用 | 不能被任何站点引用 | `CountSitesReferencingCertIDs` | — |
| 删除 | 必须已禁用 **且** 无站点引用 | 同上 | `Certs.vue` |

检查字段：`site.cert_id` 或 `settings.https.certificate_id`

错误码：`cert.site_ref_disable_first`, `cert.in_use_disable_first`, `cert.delete_site_ref_first`

### 2.7 CC 规则

| 实体 | 删除规则 | 修改规则 | 禁用规则 |
|------|----------|----------|----------|
| CC 规则组 | 不能被站点 `cc_default_rule` 引用；系统内置不可删 | **系统规则仅类型不可改** | 被站点引用时不可禁用 |
| CC 匹配器 | 不能被 `cc_rule.data` 或站点 `custom_rules` 引用；系统内置不可删 | **系统规则仅类型不可改** | 被引用时不可禁用 |
| CC 过滤器 | 不能被 `cc_rule.data`（filter1/filter2）或站点 `custom_rules` 引用；系统内置不可删 | **系统规则仅类型不可改** | 被引用时不可禁用 |

前端约定：
- 列表「类型」列显示：`系统规则` / `用户规则`
- **编辑系统规则时**：仅「类型」单选框禁用，其余字段可编辑保存
- 使用中（`in_use=true`）时，启用开关不可关闭

错误码：`cc_rule.system_type_locked`, `cc_match.system_type_locked`, `cc_filter.system_type_locked`, `cc_rule.in_use_disable`, `cc_match.in_use_disable`, `cc_filter.in_use_disable`

### 2.8 ACL 规则

| 操作 | 规则 | 后端 |
|------|------|------|
| 删除 | 必须已禁用；不能被站点 `settings.access.acl` 引用 | `GuardACLDelete` |

错误码：`acl.in_use_disable_first`, `acl.in_use`

### 2.9 套餐

| 实体 | 删除规则 | 后端 |
|------|----------|------|
| 套餐模板 `Package` | 无已购套餐 `user_package.package` 引用 | `CountUserPackagesReferencingPlan` |
| 已购套餐 `UserPackage` | 无网站 `site.user_package` 引用 | `CountSitesReferencingUserPackages` |

错误码：`plan.in_use`, `user_package.in_use`

---

## 3. 配置同步规则

保存以下配置后必须触发 `BumpConfigVersion` + `config_sync`：

- 站点设置、网站默认、全局防火墙
- CC/ACL 规则
- 证书变更
- IP 解封（额外 `ip_unblock` 快速通道）

Agent 双通道：WebSocket `config_sync` + 定时 `pullConfig`（默认 60s 兜底）。

---

## 4. AI 开发检查清单

修改相关功能前，逐项确认：

- [ ] 删除 API 是否调用 `reference_guard_service` 或等效引用检查？
- [ ] 错误信息是否使用 `common/i18n/messages.json` 中的 key？
- [ ] 前端是否在操作前给出准确预检提示（不替代后端校验）？
- [ ] 站点默认配置是否通过 `GetSiteDefaultMapWithGroup` 合并（含分组）？
- [ ] 是否使用 `setIfMissing` 避免覆盖站点已有配置？
- [ ] 新增删除类接口是否遵循「先禁用 → 解除引用 → 再删除」？

---

## 5. 代码索引

| 模块 | 主要文件 |
|------|----------|
| 引用守卫 | `api/services/reference_guard_service.go` |
| 默认配置合并 | `api/services/default_config_service.go` |
| 边缘配置生成 | `api/services/config_service.go` |
| 站点删除 | `api/controllers/site_admin.go` |
| 网站分组 | `api/controllers/site_group_controller.go` |
| 节点/线路 | `api/controllers/node_controller.go`, `node_group_controller.go` |
| 证书 | `api/controllers/cert_controller.go` |
| CC 规则 | `api/controllers/rule_controller.go` |
| ACL | `api/controllers/acl_controller.go` |
| 套餐 | `api/controllers/plan_controller.go` |
| i18n | `common/i18n/messages.json` |
| 前端网站列表 | `web/admin/src/views/website/List.vue` |
| 前端网站分组 | `web/admin/src/views/website/Groups.vue` |
| 前端证书 | `web/admin/src/views/website/Certs.vue` |
| 前端线路 | `web/admin/src/views/nodes/groups/Resolution.vue` |

---

## 6. 已知限制

- 站点属于多个网站分组时，仅第一个分组 ID 参与默认配置合并。
- `ApplySiteDefaultsScopedOverrides` 仅在站点 settings 为空时强制覆盖 gzip/ssl/cache（建站场景）。
- 全局 WAF 黑名单时长与站点黑名单 TTL 独立，修改时需分别验证。
