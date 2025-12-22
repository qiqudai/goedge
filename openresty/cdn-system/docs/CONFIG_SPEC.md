# CDN Configuration Spec (Draft)

## 1. Defaulting & Override Rules

1) 系统默认（管理员配置，global scope）
- 只用于用户注册后的默认值和“无用户默认”时的兜底。
- 存储在 `config` 表（scope_name = global, scope_id = 0）。

2) 用户默认（用户配置）
- 用户创建站点/转发时优先使用用户默认配置。
- 仅当用户默认缺失时，回退到系统默认。
- 建议存储在 `config` 表（scope_name = user, scope_id = <uid>）。

3) 站点/转发级配置
- 站点/转发上的显式配置最高优先级。

优先级：站点/转发覆盖 > 用户默认 > 系统默认

## 2. 配置同步策略（目标）

目标：中心配置能快速增量同步到所有节点，减少节点负担。

推荐：增量版本号 + Redis 通知，节点拉取差量；保底全量拉取。

- 配置版本：每个资源（site/stream/cc/acl 等）生成版本号，合并成节点级版本。
- 发布：中心写入 Redis Pub/Sub 通知（`config:changed`），节点收到后仅拉取更新版本。
- 保底：节点定时拉取（例如 60s），比较版本不一致则全量同步。

## 3. 数据表与配置用途（持续补全）

### 核心资源
- `user`：用户主体信息、认证、余额、白名单、登录安全等。
- `package`/`user_package`：套餐定义 / 用户已购套餐实例。
- `site`：网站（L7 CDN）主配置。
- `stream`：四层转发主配置。
- `cert`/`dnsapi`：证书与 DNS API。
- `acl`/`cc_rule`/`cc_match`/`cc_filter`：安全规则体系。

### 配置中心
- `config`：系统/默认配置存储（多 type + scope）。

## 4. 待补全清单

- 前端每个页面/选项 -> 对应数据表字段/配置 key 的精确映射。
- 配置继承逻辑在后端创建/更新流程中的落地代码。
- 配置同步具体实现：Redis 通知 + 节点拉取差量协议。
