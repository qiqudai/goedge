---
name: cdn-feature-design
description: CDN feature design and implementation guide for this project. Use when modeling admin/user features, API flows, node sync, and OpenResty/Nginx behavior for CDNFly-style functionality without changing db.sql.
---

# CDN Feature Design

Use this skill when translating CDNFly documentation into concrete frontend, API, sync, and node behavior for this repo.

## Core constraints

- Use the existing schema in `cdn-system/db.sql` only. Do not add tables/fields unless the user explicitly requests it.
- Prefer high-performance Nginx/OpenResty config over Lua; use Lua only where unavoidable.
- Config changes that impact nodes must be synchronized through API (no Redis-based sync).
- Role-aware UI: admin vs user behavior is different; never show a user selector for normal users.

## Coding Conventions (Observed)

- Prefer explicit JSON structs for API payloads and store raw JSON in `settings`/`data` columns when schema is flexible.
- Use `BumpConfigVersion(<resource>, ids)` after any config change that must reach nodes.
- Keep agent writes atomic (`fsutil.WriteJSONAtomic`) and read them on agent boot to restore memory state.
- Nginx config changes are generated from JSON and then reloaded (`nginx -s reload`); avoid runtime Lua checks for hard restrictions.
- When adding Lua modules, keep them under `cdn-system/agent/assets/lua` and ensure Nginx `lua_package_path` can resolve them.

## API Patterns (Observed)

- API is split by role: `/api/v1/admin/*`, `/api/v1/user/*`, `/api/v1/agent/*`, plus `/ws/agent` for WebSocket.
- Admin/user APIs generally mirror each other (list/get/create/update/delete), with user scope restricted by `user_id`.
- Agent pulls config from `/api/v1/agent/config` and tasks from `/api/v1/agent/tasks`; agent also uploads logs/metrics.
- WebSocket is used for `node_sync` and task notifications, but data payload is still pulled via HTTP.

## Agent API Responsibilities

- `/api/v1/agent/config`: returns `EdgeConfig` (domains, upstreams, WAF, resources, error pages, CC rules, Nginx).
- `/api/v1/agent/tasks`: returns tasks, including `config_sync` and `package_sync`.
- `/api/v1/agent/tasks/:id/finish`: task completion callback.
- `/api/v1/agent/logs/access` and `/api/v1/agent/logs/metrics`: upstream data to ClickHouse/metrics.
- `/api/v1/agent/l2/nodes`: L2 node metadata.
- `/ws/agent`: realtime control channel for `node_sync`/`heartbeat` and config change notifications.

## Redundant / Alias APIs (Observed)

- `/api/v1/admin/nodes/batch_action` is an alias of `/api/v1/admin/nodes/batch`.
- Many user/admin endpoints are duplicates in role namespace (e.g., `/api/v1/admin/rules/acl` vs `/api/v1/user/rules/acl`).

## API Surface (High-level)

- **Config**: `/api/v1/admin/global_config`, `/api/v1/admin/config_items` (global configs and templates).
- **Sites**: `/api/v1/admin/sites`, `/api/v1/user/sites` (CRUD + settings).
- **Rules**: `/api/v1/admin/rules/acl`, `/api/v1/admin/rules/cc/*` and user equivalents.
- **Packages**: `/api/v1/admin/user_packages`, `/api/v1/admin/packages`, `/api/v1/admin/plans`, `/api/v1/admin/user_plans`.
- **Logs/Stats**: `/api/v1/admin/logs/*`, `/api/v1/admin/stats/*` and user equivalents.
- **Nodes**: `/api/v1/admin/nodes`, `/api/v1/admin/node-groups`, `/api/v1/admin/regions`.

## Frontend API Usage (Observed in web/admin)

Use this list to identify APIs that are actually called by the frontend. Base URL is set in `cdn-system/web/admin/src/utils/request.js`, defaulting to `/api/v1/admin` and switching to `/api/v1/user` based on `localStorage.role`. The only call using `/api/v1` directly is `/acls`.

Regenerate quickly:
```bash
python - <<'PY'
import re
from pathlib import Path
root = Path('cdn-system/web/admin/src')
pattern = re.compile(r"request\.(?:get|post|put|delete|patch)\(\s*([`\"'])(.+?)\1", re.S)
endpoints = set()
for path in list(root.rglob('*.vue')) + list(root.rglob('*.js')):
    text = path.read_text(encoding='utf-8', errors='ignore')
    endpoints |= {m.group(2) for m in pattern.finditer(text)}
print('\n'.join(sorted(endpoints)))
PY
```

Endpoints (prefix with `/api/v1/admin` or `/api/v1/user` unless noted):
```
/acls  (baseURL=/api/v1)
/announcements
/announcements/${form.id}
/announcements/${row.id}
/api_key
/api_key/reset
/certs
/certs/${props.certId}
/certs/${row.id}/download
/certs/batch
/certs/batch_action
/certs/default_settings
/certs/reissue
/cname_domains
/cname_domains/${cnameForm.id}
/cname_domains/${row.id}
/config_items
/dashboard
/dns/providers
/dns/providers/${currentProviderId.value}
/dns/providers/types
/dnsapi
/dnsapi/${dnsapiForm.id}
/dnsapi/${form.id}
/dnsapi/${id}
/dnsapi/${row.id}
/dnsapi/types
/domain_usage
/forward_defaults
/forward_groups
/forwards
/forwards/${form.id}
/forwards/batch
/forwards/batch_action
/global_config
/login  (public: /api/v1/login)
/logs/block/current
/logs/block/history
/logs/block/stats
/logs/operation
/message_sub
/messages
/messages/${row.id}/read
/monitor_config
/node-groups
/nodes
/nodes/${form.id}
/nodes/${props.nodeId}/monitor_logs
/nodes/${row.id}
/nodes/${row.id}/status
/nodes/batch_action
/orders
/packages
/packages/grayscale
/password
/plans
/plans/${row.id}
/profile
/recharge
/regions
/regions/${form.id}
/regions/${id}
/regions/${row.id}
/rules/acl
/rules/acl/${form.id}
/rules/acl/${row.id}
/rules/cc/filters
/rules/cc/filters/${form.id}
/rules/cc/filters/${row.id}
/rules/cc/groups
/rules/cc/groups/${form.id}
/rules/cc/groups/${row.id}
/rules/cc/matchers
/rules/cc/matchers/${form.id}
/rules/cc/matchers/${row.id}
/site_defaults
/site_defaults/${encodeURIComponent(editScope.originalName)}
/site_defaults/${encodeURIComponent(row.name)}
/site_groups
/site_groups/${editingId.value}
/site_groups/${row.id}
/sites
/sites/${form.id}
/sites/${siteId.value}
/sites/batch
/sites/batch_action
/sites/batch_update
/sites/resolve
/stats/basic
/stats/node_traffic
/stats/origin
/stats/quality
/stats/ranking
/tasks
/tasks/${row.id}/resubmit
/tasks/usage
/usage
/user_packages
/user_packages/${current.value.id}
/user_packages/${current.value.id}/renew
/user_packages/${current.value.id}/switch
/user_plans
/user_plans/${editForm.value.id}
/user_plans/assign
/users
/users/${form.id}
/users/${form.user_id}
/users/${row.id}/impersonate
/users/${row.id}/purge/reset
```

Route -> component map (from `request.*` usage in `web/admin`):
```
/acls -> cdn-system/web/admin/src/composables/useSiteSettings.js
/announcements -> cdn-system/web/admin/src/views/system/Announcements.vue
/announcements/${form.id} -> cdn-system/web/admin/src/views/system/Announcements.vue
/announcements/${row.id} -> cdn-system/web/admin/src/views/system/Announcements.vue
/api_key -> cdn-system/web/admin/src/views/account/ApiKey.vue, cdn-system/web/admin/src/views/settings/components/OtherConfig.vue
/api_key/reset -> cdn-system/web/admin/src/views/account/ApiKey.vue, cdn-system/web/admin/src/views/settings/components/OtherConfig.vue
/certs -> cdn-system/web/admin/src/composables/useSiteSettings.js, cdn-system/web/admin/src/views/website/CertEditPopup.vue, cdn-system/web/admin/src/views/website/Certs.vue
/certs/${props.certId} -> cdn-system/web/admin/src/views/website/CertEditPopup.vue
/certs/${row.id}/download -> cdn-system/web/admin/src/views/website/Certs.vue
/certs/batch -> cdn-system/web/admin/src/views/website/CertEditPopup.vue
/certs/batch_action -> cdn-system/web/admin/src/views/website/Certs.vue
/certs/default_settings -> cdn-system/web/admin/src/views/website/Certs.vue
/certs/reissue -> cdn-system/web/admin/src/views/website/Certs.vue
/cname_domains -> cdn-system/web/admin/src/views/dns/Index.vue, cdn-system/web/admin/src/views/plans/Basic.vue, cdn-system/web/admin/src/views/plans/Sold.vue
/cname_domains/${cnameForm.id} -> cdn-system/web/admin/src/views/dns/Index.vue
/cname_domains/${row.id} -> cdn-system/web/admin/src/views/dns/Index.vue
/config_items -> cdn-system/web/admin/src/views/global/components/CertConfig.vue, cdn-system/web/admin/src/views/global/components/StreamConfig.vue, cdn-system/web/admin/src/views/settings/System.vue, cdn-system/web/admin/src/views/settings/components/BasicConfig.vue, cdn-system/web/admin/src/views/settings/components/CleaningConfig.vue, cdn-system/web/admin/src/views/settings/components/MaintenanceConfig.vue, cdn-system/web/admin/src/views/settings/components/NotifyConfig.vue, cdn-system/web/admin/src/views/settings/components/OtherConfig.vue, cdn-system/web/admin/src/views/settings/components/PackageConfig.vue, cdn-system/web/admin/src/views/settings/components/UserConfig.vue
/dashboard -> cdn-system/web/admin/src/views/dashboard/Index.vue
/dns/providers -> cdn-system/web/admin/src/views/dns/Index.vue
/dns/providers/${currentProviderId.value} -> cdn-system/web/admin/src/views/dns/Index.vue
/dns/providers/types -> cdn-system/web/admin/src/views/dns/Index.vue
/dnsapi -> cdn-system/web/admin/src/views/website/CertEditPopup.vue, cdn-system/web/admin/src/views/website/Certs.vue, cdn-system/web/admin/src/views/website/list/DefaultSettings.vue, cdn-system/web/admin/src/views/website/list/DnsApiList.vue, cdn-system/web/admin/src/views/website/list/SiteEditDialog.vue
/dnsapi/${dnsapiForm.id} -> cdn-system/web/admin/src/views/website/Certs.vue
/dnsapi/${form.id} -> cdn-system/web/admin/src/views/website/list/DnsApiList.vue
/dnsapi/${id} -> cdn-system/web/admin/src/views/website/Certs.vue
/dnsapi/${row.id} -> cdn-system/web/admin/src/views/website/Certs.vue, cdn-system/web/admin/src/views/website/list/DnsApiList.vue
/dnsapi/types -> cdn-system/web/admin/src/views/website/Certs.vue, cdn-system/web/admin/src/views/website/list/DnsApiList.vue
/domain_usage -> cdn-system/web/admin/src/views/website/List.vue, cdn-system/web/admin/src/views/website/list/SiteEditDialog.vue
/forward_defaults -> cdn-system/web/admin/src/views/forward/Default.vue
/forward_groups -> cdn-system/web/admin/src/views/forward/Default.vue, cdn-system/web/admin/src/views/forward/Groups.vue
/forwards -> cdn-system/web/admin/src/views/forward/List.vue, cdn-system/web/admin/src/views/forward/list/ForwardEditDialog.vue
/forwards/${form.id} -> cdn-system/web/admin/src/views/forward/list/ForwardEditDialog.vue
/forwards/batch -> cdn-system/web/admin/src/views/forward/list/ForwardEditDialog.vue
/forwards/batch_action -> cdn-system/web/admin/src/views/forward/List.vue
/global_config -> cdn-system/web/admin/src/views/global/ErrorPages.vue, cdn-system/web/admin/src/views/global/Firewall.vue, cdn-system/web/admin/src/views/global/Resources.vue, cdn-system/web/admin/src/views/global/components/CacheConfig.vue, cdn-system/web/admin/src/views/settings/Global.vue
/login -> cdn-system/web/admin/src/views/Login.vue
/logs/block/current -> cdn-system/web/admin/src/views/website/BlockLogs.vue
/logs/block/history -> cdn-system/web/admin/src/views/website/BlockLogs.vue
/logs/block/stats -> cdn-system/web/admin/src/views/website/BlockLogs.vue
/logs/operation -> cdn-system/web/admin/src/views/account/Logs.vue, cdn-system/web/admin/src/views/logs/Operation.vue
/message_sub -> cdn-system/web/admin/src/views/account/Subscribe.vue
/messages -> cdn-system/web/admin/src/views/account/Messages.vue, cdn-system/web/admin/src/views/system/Messages.vue
/messages/${row.id}/read -> cdn-system/web/admin/src/views/account/Messages.vue
/monitor_config -> cdn-system/web/admin/src/views/settings/Monitor.vue
/node-groups -> cdn-system/web/admin/src/views/plans/Basic.vue, cdn-system/web/admin/src/views/plans/Sold.vue, cdn-system/web/admin/src/views/website/list/BatchEditDialog.vue, cdn-system/web/admin/src/views/website/list/DefaultSettings.vue
/nodes -> cdn-system/web/admin/src/views/nodes/List.vue, cdn-system/web/admin/src/views/nodes/RealtimeMonitor.vue, cdn-system/web/admin/src/views/nodes/list/NodeEditDialog.vue
/nodes/${form.id} -> cdn-system/web/admin/src/views/nodes/list/NodeEditDialog.vue
/nodes/${props.nodeId}/monitor_logs -> cdn-system/web/admin/src/views/nodes/list/MonitorLogDialog.vue
/nodes/${row.id} -> cdn-system/web/admin/src/views/nodes/List.vue
/nodes/${row.id}/status -> cdn-system/web/admin/src/views/nodes/List.vue
/nodes/batch_action -> cdn-system/web/admin/src/views/nodes/List.vue
/orders -> cdn-system/web/admin/src/views/account/Bills.vue, cdn-system/web/admin/src/views/finance/Orders.vue
/packages -> cdn-system/web/admin/src/views/system/Upgrade.vue
/packages/grayscale -> cdn-system/web/admin/src/views/system/Upgrade.vue
/password -> cdn-system/web/admin/src/views/account/Profile.vue
/plans -> cdn-system/web/admin/src/views/packages/My.vue, cdn-system/web/admin/src/views/plans/Basic.vue, cdn-system/web/admin/src/views/plans/Sold.vue
/plans/${row.id} -> cdn-system/web/admin/src/views/plans/Basic.vue
/profile -> cdn-system/web/admin/src/views/account/Profile.vue, cdn-system/web/admin/src/views/account/Recharge.vue
/recharge -> cdn-system/web/admin/src/views/account/Recharge.vue, cdn-system/web/admin/src/views/finance/Orders.vue
/regions -> cdn-system/web/admin/src/views/nodes/List.vue, cdn-system/web/admin/src/views/nodes/list/RegionList.vue, cdn-system/web/admin/src/views/plans/Basic.vue, cdn-system/web/admin/src/views/plans/Sold.vue, cdn-system/web/admin/src/views/website/list/BatchEditDialog.vue
/regions/${form.id} -> cdn-system/web/admin/src/views/nodes/list/RegionList.vue
/regions/${id} -> cdn-system/web/admin/src/views/nodes/list/RegionList.vue
/regions/${row.id} -> cdn-system/web/admin/src/views/nodes/list/RegionList.vue
/rules/acl -> cdn-system/web/admin/src/views/website/rules/AclList.vue
/rules/acl/${form.id} -> cdn-system/web/admin/src/views/website/rules/AclList.vue
/rules/acl/${row.id} -> cdn-system/web/admin/src/views/website/rules/AclList.vue
/rules/cc/filters -> cdn-system/web/admin/src/views/website/rules/cc/FilterList.vue, cdn-system/web/admin/src/views/website/rules/cc/GroupList.vue
/rules/cc/filters/${form.id} -> cdn-system/web/admin/src/views/website/rules/cc/FilterList.vue
/rules/cc/filters/${row.id} -> cdn-system/web/admin/src/views/website/rules/cc/FilterList.vue
/rules/cc/groups -> cdn-system/web/admin/src/components/manage/SecurityConfig.vue, cdn-system/web/admin/src/composables/useGlobalConfig.js, cdn-system/web/admin/src/views/global/components/SiteConfig.vue, cdn-system/web/admin/src/views/website/list/DefaultSettings.vue, cdn-system/web/admin/src/views/website/rules/cc/GroupList.vue
/rules/cc/groups/${form.id} -> cdn-system/web/admin/src/views/website/rules/cc/GroupList.vue
/rules/cc/groups/${row.id} -> cdn-system/web/admin/src/views/website/rules/cc/GroupList.vue
/rules/cc/matchers -> cdn-system/web/admin/src/views/website/rules/cc/GroupList.vue, cdn-system/web/admin/src/views/website/rules/cc/MatcherList.vue
/rules/cc/matchers/${form.id} -> cdn-system/web/admin/src/views/website/rules/cc/MatcherList.vue
/rules/cc/matchers/${row.id} -> cdn-system/web/admin/src/views/website/rules/cc/MatcherList.vue
/site_defaults -> cdn-system/web/admin/src/composables/useGlobalConfig.js, cdn-system/web/admin/src/views/global/components/SiteConfig.vue, cdn-system/web/admin/src/views/website/list/DefaultSettings.vue
/site_defaults/${encodeURIComponent(editScope.originalName)} -> cdn-system/web/admin/src/views/website/list/DefaultSettings.vue
/site_defaults/${encodeURIComponent(row.name)} -> cdn-system/web/admin/src/views/website/list/DefaultSettings.vue
/site_groups -> cdn-system/web/admin/src/views/website/Groups.vue, cdn-system/web/admin/src/views/website/list/DefaultSettings.vue
/site_groups/${editingId.value} -> cdn-system/web/admin/src/views/website/Groups.vue
/site_groups/${row.id} -> cdn-system/web/admin/src/views/website/Groups.vue
/sites -> cdn-system/web/admin/src/views/website/List.vue, cdn-system/web/admin/src/views/website/Resolve.vue, cdn-system/web/admin/src/views/website/list/SiteEditDialog.vue
/sites/${form.id} -> cdn-system/web/admin/src/views/website/list/SiteEditDialog.vue
/sites/${siteId.value} -> cdn-system/web/admin/src/composables/useSiteSettings.js
/sites/batch -> cdn-system/web/admin/src/views/website/list/SiteEditDialog.vue
/sites/batch_action -> cdn-system/web/admin/src/views/website/List.vue
/sites/batch_update -> cdn-system/web/admin/src/views/website/list/BatchEditDialog.vue
/sites/resolve -> cdn-system/web/admin/src/views/website/Resolve.vue
/stats/basic -> cdn-system/web/admin/src/views/website/Statistics.vue
/stats/node_traffic -> cdn-system/web/admin/src/views/nodes/RealtimeMonitor.vue
/stats/origin -> cdn-system/web/admin/src/views/website/Statistics.vue
/stats/quality -> cdn-system/web/admin/src/views/website/Statistics.vue
/stats/ranking -> cdn-system/web/admin/src/views/website/Statistics.vue
/tasks -> cdn-system/web/admin/src/views/system/Tasks.vue, cdn-system/web/admin/src/views/website/Purge.vue
/tasks/${row.id}/resubmit -> cdn-system/web/admin/src/views/system/Tasks.vue, cdn-system/web/admin/src/views/website/Purge.vue
/tasks/usage -> cdn-system/web/admin/src/views/website/Purge.vue
/usage -> cdn-system/web/admin/src/views/packages/Usage.vue
/user_packages -> cdn-system/web/admin/src/composables/useSiteSettings.js, cdn-system/web/admin/src/views/forward/list/ForwardEditDialog.vue, cdn-system/web/admin/src/views/packages/My.vue, cdn-system/web/admin/src/views/website/List.vue, cdn-system/web/admin/src/views/website/Rules.vue, cdn-system/web/admin/src/views/website/list/SiteEditDialog.vue
/user_packages/${current.value.id} -> cdn-system/web/admin/src/views/packages/My.vue
/user_packages/${current.value.id}/renew -> cdn-system/web/admin/src/views/packages/My.vue
/user_packages/${current.value.id}/switch -> cdn-system/web/admin/src/views/packages/My.vue
/user_plans -> cdn-system/web/admin/src/views/plans/Sold.vue
/user_plans/${editForm.value.id} -> cdn-system/web/admin/src/views/plans/Sold.vue
/user_plans/assign -> cdn-system/web/admin/src/views/plans/Basic.vue
/users -> cdn-system/web/admin/src/views/forward/list/ForwardEditDialog.vue, cdn-system/web/admin/src/views/plans/Basic.vue, cdn-system/web/admin/src/views/website/CertEditPopup.vue, cdn-system/web/admin/src/views/website/Certs.vue, cdn-system/web/admin/src/views/website/list/DefaultSettings.vue, cdn-system/web/admin/src/views/website/list/SiteEditDialog.vue, cdn-system/web/admin/src/views/website/rules/AclList.vue, cdn-system/web/admin/src/views/website/rules/cc/FilterList.vue, cdn-system/web/admin/src/views/website/rules/cc/GroupList.vue, cdn-system/web/admin/src/views/website/rules/cc/MatcherList.vue
/users/${form.id} -> cdn-system/web/admin/src/views/users/UserEditPopup.vue
/users/${form.user_id} -> cdn-system/web/admin/src/views/website/rules/cc/MatcherList.vue
/users/${row.id}/impersonate -> cdn-system/web/admin/src/views/users/List.vue
/users/${row.id}/purge/reset -> cdn-system/web/admin/src/views/users/List.vue
```

Direct URL usages (not via `request.*`):
```
/api/v1/admin/upload/image -> cdn-system/web/admin/src/views/settings/components/BasicConfig.vue
/api/v1/admin/packages -> cdn-system/web/admin/src/views/system/Upgrade.vue
```

Method coverage gaps (path used, but method not called via `request.*`; note direct URL usages above):
```
Admin:
DELETE /api/v1/admin/certs/:id
DELETE /api/v1/admin/user_plans
DELETE /api/v1/admin/users/:id
GET /api/v1/admin/rules/cc/filters/:id
POST /api/v1/admin/node-groups
POST /api/v1/admin/plans
PUT /api/v1/admin/plans/:id
User:
DELETE /api/v1/user/certs/:id
GET /api/v1/user/rules/cc/filters/:id
Shared:
```

Frontend-unused API list (path not referenced by frontend at all):
```
Admin:
DELETE /api/v1/admin/node-groups/:id
GET /api/v1/admin/certs/batch/:id/progress
GET /api/v1/admin/domains
GET /api/v1/admin/logs/access
GET /api/v1/admin/logs/backup
GET /api/v1/admin/logs/login
GET /api/v1/admin/logs/mail
GET /api/v1/admin/node-groups/:id/resolution
GET /api/v1/admin/sites/batch/:id/progress
GET /api/v1/admin/sites/export
GET /api/v1/admin/system_info
POST /api/v1/admin/forwards/batch_update
POST /api/v1/admin/node-groups/:id/resolution/action
POST /api/v1/admin/node-groups/:id/resolution/assign
POST /api/v1/admin/nodes/batch
POST /api/v1/admin/sites/apply_cert
POST /api/v1/admin/system_info
POST /api/v1/admin/ws/dispatch
PUT /api/v1/admin/node-groups/:id
PUT /api/v1/admin/users/:id/status
User:
GET /api/v1/user/certs/batch/:id/progress
GET /api/v1/user/domains
GET /api/v1/user/domains/:id/config
GET /api/v1/user/logs/access
POST /api/v1/user/domains
POST /api/v1/user/forwards/batch_update
Shared:
```

Removable candidates (risk noted; based on frontend-only evidence):
- Low risk (test/unused by UI):
  - `/api/v1/admin/ws/dispatch`
  - `/api/v1/admin/certs/batch/:id/progress`
  - `/api/v1/user/certs/batch/:id/progress`
- Medium risk (not in UI, could be used by ops/scripts/external clients):
  - `/api/v1/admin/domains`
  - `/api/v1/admin/logs/access`
  - `/api/v1/admin/logs/backup`
  - `/api/v1/admin/logs/login`
  - `/api/v1/admin/logs/mail`
  - `/api/v1/admin/system_info` (GET/POST)
  - `/api/v1/admin/forwards/batch_update`
  - `/api/v1/admin/nodes/batch` (alias of batch_action)
  - `/api/v1/admin/sites/batch/:id/progress`
  - `/api/v1/admin/sites/export`
  - `/api/v1/admin/sites/apply_cert`
  - `/api/v1/admin/node-groups/:id/resolution`
  - `/api/v1/admin/node-groups/:id/resolution/assign`
  - `/api/v1/admin/node-groups/:id/resolution/action`
  - `/api/v1/admin/node-groups/:id` (PUT/DELETE)
  - `/api/v1/admin/users/:id/status`
  - `/api/v1/user/domains` (GET/POST)
  - `/api/v1/user/domains/:id/config`
  - `/api/v1/user/logs/access`
  - `/api/v1/user/forwards/batch_update`

Page -> API map by module (from `request.*` usage):
```
[Login.vue]
cdn-system/web/admin/src/views/Login.vue:
  - /login
[account]
cdn-system/web/admin/src/views/account/ApiKey.vue:
  - /api_key
  - /api_key/reset
cdn-system/web/admin/src/views/account/Bills.vue:
  - /orders
cdn-system/web/admin/src/views/account/Logs.vue:
  - /logs/operation
cdn-system/web/admin/src/views/account/Messages.vue:
  - /messages
  - /messages/${row.id}/read
cdn-system/web/admin/src/views/account/Profile.vue:
  - /password
  - /profile
cdn-system/web/admin/src/views/account/Recharge.vue:
  - /profile
  - /recharge
cdn-system/web/admin/src/views/account/Subscribe.vue:
  - /message_sub
[dashboard]
cdn-system/web/admin/src/views/dashboard/Index.vue:
  - /dashboard
[dns]
cdn-system/web/admin/src/views/dns/Index.vue:
  - /cname_domains
  - /cname_domains/${cnameForm.id}
  - /cname_domains/${row.id}
  - /dns/providers
  - /dns/providers/${currentProviderId.value}
  - /dns/providers/types
[finance]
cdn-system/web/admin/src/views/finance/Orders.vue:
  - /orders
  - /recharge
[forward]
cdn-system/web/admin/src/views/forward/Default.vue:
  - /forward_defaults
  - /forward_groups
cdn-system/web/admin/src/views/forward/Groups.vue:
  - /forward_groups
cdn-system/web/admin/src/views/forward/List.vue:
  - /forwards
  - /forwards/batch_action
cdn-system/web/admin/src/views/forward/list/ForwardEditDialog.vue:
  - /forwards
  - /forwards/${form.id}
  - /forwards/batch
  - /user_packages
  - /users
[global]
cdn-system/web/admin/src/views/global/ErrorPages.vue:
  - /global_config
cdn-system/web/admin/src/views/global/Firewall.vue:
  - /global_config
cdn-system/web/admin/src/views/global/Resources.vue:
  - /global_config
cdn-system/web/admin/src/views/global/components/CacheConfig.vue:
  - /global_config
cdn-system/web/admin/src/views/global/components/CertConfig.vue:
  - /config_items
cdn-system/web/admin/src/views/global/components/SiteConfig.vue:
  - /rules/cc/groups
  - /site_defaults
cdn-system/web/admin/src/views/global/components/StreamConfig.vue:
  - /config_items
[logs]
cdn-system/web/admin/src/views/logs/Operation.vue:
  - /logs/operation
[nodes]
cdn-system/web/admin/src/views/nodes/List.vue:
  - /nodes
  - /nodes/${row.id}
  - /nodes/${row.id}/status
  - /nodes/batch_action
  - /regions
cdn-system/web/admin/src/views/nodes/RealtimeMonitor.vue:
  - /nodes
  - /stats/node_traffic
cdn-system/web/admin/src/views/nodes/list/MonitorLogDialog.vue:
  - /nodes/${props.nodeId}/monitor_logs
cdn-system/web/admin/src/views/nodes/list/NodeEditDialog.vue:
  - /nodes
  - /nodes/${form.id}
cdn-system/web/admin/src/views/nodes/list/RegionList.vue:
  - /regions
  - /regions/${form.id}
  - /regions/${id}
  - /regions/${row.id}
[packages]
cdn-system/web/admin/src/views/packages/My.vue:
  - /plans
  - /user_packages
  - /user_packages/${current.value.id}
  - /user_packages/${current.value.id}/renew
  - /user_packages/${current.value.id}/switch
cdn-system/web/admin/src/views/packages/Usage.vue:
  - /usage
[plans]
cdn-system/web/admin/src/views/plans/Basic.vue:
  - /cname_domains
  - /node-groups
  - /plans
  - /plans/${row.id}
  - /regions
  - /user_plans/assign
  - /users
cdn-system/web/admin/src/views/plans/Sold.vue:
  - /cname_domains
  - /node-groups
  - /plans
  - /regions
  - /user_plans
  - /user_plans/${editForm.value.id}
[settings]
cdn-system/web/admin/src/views/settings/Global.vue:
  - /global_config
cdn-system/web/admin/src/views/settings/Monitor.vue:
  - /monitor_config
cdn-system/web/admin/src/views/settings/System.vue:
  - /config_items
cdn-system/web/admin/src/views/settings/components/BasicConfig.vue:
  - /config_items
cdn-system/web/admin/src/views/settings/components/CleaningConfig.vue:
  - /config_items
cdn-system/web/admin/src/views/settings/components/MaintenanceConfig.vue:
  - /config_items
cdn-system/web/admin/src/views/settings/components/NotifyConfig.vue:
  - /config_items
cdn-system/web/admin/src/views/settings/components/OtherConfig.vue:
  - /api_key
  - /api_key/reset
  - /config_items
cdn-system/web/admin/src/views/settings/components/PackageConfig.vue:
  - /config_items
cdn-system/web/admin/src/views/settings/components/UserConfig.vue:
  - /config_items
[system]
cdn-system/web/admin/src/views/system/Announcements.vue:
  - /announcements
  - /announcements/${form.id}
  - /announcements/${row.id}
cdn-system/web/admin/src/views/system/Messages.vue:
  - /messages
cdn-system/web/admin/src/views/system/Tasks.vue:
  - /tasks
  - /tasks/${row.id}/resubmit
cdn-system/web/admin/src/views/system/Upgrade.vue:
  - /packages
  - /packages/grayscale
[users]
cdn-system/web/admin/src/views/users/List.vue:
  - /users/${row.id}/impersonate
  - /users/${row.id}/purge/reset
cdn-system/web/admin/src/views/users/UserEditPopup.vue:
  - /users/${form.id}
[website]
cdn-system/web/admin/src/views/website/BlockLogs.vue:
  - /logs/block/current
  - /logs/block/history
  - /logs/block/stats
cdn-system/web/admin/src/views/website/CertEditPopup.vue:
  - /certs
  - /certs/${props.certId}
  - /certs/batch
  - /dnsapi
  - /users
cdn-system/web/admin/src/views/website/Certs.vue:
  - /certs
  - /certs/${row.id}/download
  - /certs/batch_action
  - /certs/default_settings
  - /certs/reissue
  - /dnsapi
  - /dnsapi/${dnsapiForm.id}
  - /dnsapi/${id}
  - /dnsapi/${row.id}
  - /dnsapi/types
  - /users
cdn-system/web/admin/src/views/website/Groups.vue:
  - /site_groups
  - /site_groups/${editingId.value}
  - /site_groups/${row.id}
cdn-system/web/admin/src/views/website/List.vue:
  - /domain_usage
  - /sites
  - /sites/batch_action
  - /user_packages
cdn-system/web/admin/src/views/website/Purge.vue:
  - /tasks
  - /tasks/${row.id}/resubmit
  - /tasks/usage
cdn-system/web/admin/src/views/website/Resolve.vue:
  - /sites
  - /sites/resolve
cdn-system/web/admin/src/views/website/Rules.vue:
  - /user_packages
cdn-system/web/admin/src/views/website/Statistics.vue:
  - /stats/basic
  - /stats/origin
  - /stats/quality
  - /stats/ranking
cdn-system/web/admin/src/views/website/list/BatchEditDialog.vue:
  - /node-groups
  - /regions
  - /sites/batch_update
cdn-system/web/admin/src/views/website/list/DefaultSettings.vue:
  - /dnsapi
  - /node-groups
  - /rules/cc/groups
  - /site_defaults
  - /site_defaults/${encodeURIComponent(editScope.originalName)}
  - /site_defaults/${encodeURIComponent(row.name)}
  - /site_groups
  - /users
cdn-system/web/admin/src/views/website/list/DnsApiList.vue:
  - /dnsapi
  - /dnsapi/${form.id}
  - /dnsapi/${row.id}
  - /dnsapi/types
cdn-system/web/admin/src/views/website/list/SiteEditDialog.vue:
  - /dnsapi
  - /domain_usage
  - /sites
  - /sites/${form.id}
  - /sites/batch
  - /user_packages
  - /users
cdn-system/web/admin/src/views/website/rules/AclList.vue:
  - /rules/acl
  - /rules/acl/${form.id}
  - /rules/acl/${row.id}
  - /users
cdn-system/web/admin/src/views/website/rules/cc/FilterList.vue:
  - /rules/cc/filters
  - /rules/cc/filters/${form.id}
  - /rules/cc/filters/${row.id}
  - /users
cdn-system/web/admin/src/views/website/rules/cc/GroupList.vue:
  - /rules/cc/filters
  - /rules/cc/groups
  - /rules/cc/groups/${form.id}
  - /rules/cc/groups/${row.id}
  - /rules/cc/matchers
  - /users
cdn-system/web/admin/src/views/website/rules/cc/MatcherList.vue:
  - /rules/cc/matchers
  - /rules/cc/matchers/${form.id}
  - /rules/cc/matchers/${row.id}
  - /users
  - /users/${form.user_id}
[shared/composables]
cdn-system/web/admin/src/components/manage/SecurityConfig.vue:
  - /rules/cc/groups
cdn-system/web/admin/src/composables/useGlobalConfig.js:
  - /rules/cc/groups
  - /site_defaults
cdn-system/web/admin/src/composables/useSiteSettings.js:
  - /acls
  - /certs
  - /sites/${siteId.value}
  - /user_packages
```

## API Surface (Full Route Map from routers/setup.go)

### Public
- `GET /health`: load balancer health check.
- `GET /uploads/*`: static uploads.
- `GET /.well-known/acme-challenge/:token`: ACME HTTP-01.
- `GET /ws/agent`: agent WebSocket.
- `POST /api/v1/login` / `/api/v1/admin/login` / `/api/v1/user/login`: login (same handler).

### Shared (role-filtered)
- `GET /api/v1/acls`: ACL list (filtered by role in controller).

### Admin
- **Nodes**
  - `GET /api/v1/admin/nodes`: list nodes.
  - `POST /api/v1/admin/nodes`: create node.
  - `PUT /api/v1/admin/nodes/:id`: update node.
  - `POST /api/v1/admin/nodes/batch`: batch action.
  - `POST /api/v1/admin/nodes/batch_action`: alias of batch action.
  - `GET /api/v1/admin/node-groups`: list groups.
  - `POST /api/v1/admin/node-groups`: create group.
  - `PUT /api/v1/admin/node-groups/:id`: update group.
  - `DELETE /api/v1/admin/node-groups/:id`: delete group.
  - `GET /api/v1/admin/node-groups/:id/resolution`: list resolution lines.
  - `POST /api/v1/admin/node-groups/:id/resolution/assign`: assign lines.
  - `POST /api/v1/admin/node-groups/:id/resolution/action`: resolution action.
  - `GET /api/v1/admin/regions`: list regions.
  - `POST /api/v1/admin/regions`: create region.
  - `PUT /api/v1/admin/regions/:id`: update region.
  - `DELETE /api/v1/admin/regions/:id`: delete region.
- **DNS**
  - `GET /api/v1/admin/dns/providers`: list providers.
  - `GET /api/v1/admin/dns/providers/types`: list provider types.
  - `POST /api/v1/admin/dns/providers`: create provider.
  - `DELETE /api/v1/admin/dns/providers/:id`: delete provider.
  - `GET /api/v1/admin/dnsapi`: list DNS API.
  - `POST /api/v1/admin/dnsapi`: create DNS API.
  - `PUT /api/v1/admin/dnsapi/:id`: update DNS API.
  - `DELETE /api/v1/admin/dnsapi/:id`: delete DNS API.
  - `GET /api/v1/admin/dnsapi/types`: list DNS API types.
- **CNAME**
  - `GET /api/v1/admin/cname_domains`: list CNAME domains.
  - `POST /api/v1/admin/cname_domains`: create CNAME domain.
  - `PUT /api/v1/admin/cname_domains/:id`: update CNAME domain.
  - `DELETE /api/v1/admin/cname_domains/:id`: delete CNAME domain.
- **Monitor**
  - `GET /api/v1/admin/monitor_config`: get monitor config.
  - `POST /api/v1/admin/monitor_config`: update monitor config.
- **Logs**
  - `GET /api/v1/admin/logs/login`: login logs.
  - `GET /api/v1/admin/logs/operation`: operation logs.
  - `GET /api/v1/admin/logs/access`: access logs.
  - `GET /api/v1/admin/logs/backup`: backup logs.
  - `GET /api/v1/admin/logs/mail`: mail logs.
  - `GET /api/v1/admin/logs/block/current`: WAF/ACL block current.
  - `GET /api/v1/admin/logs/block/stats`: block stats.
  - `GET /api/v1/admin/logs/block/history`: block history.
- **Messages**
  - `GET /api/v1/admin/messages`: admin messages.
- **Stats/Dashboard**
  - `GET /api/v1/admin/stats/ranking`: ranking stats.
  - `GET /api/v1/admin/stats/basic`: basic stats.
  - `GET /api/v1/admin/stats/quality`: quality stats.
  - `GET /api/v1/admin/stats/origin`: origin stats.
  - `GET /api/v1/admin/stats/node_traffic`: node traffic stats.
  - `GET /api/v1/admin/dashboard`: dashboard overview.
- **Global Config**
  - `GET /api/v1/admin/global_config`: get global config.
  - `POST /api/v1/admin/global_config`: update global config.
  - `GET /api/v1/admin/config_items`: list config items.
  - `POST /api/v1/admin/config_items`: upsert config item.
- **Packages/Plans**
  - `GET /api/v1/admin/packages`: list package versions.
  - `POST /api/v1/admin/packages`: upload version.
  - `POST /api/v1/admin/packages/grayscale`: update grayscale.
  - `GET /api/v1/admin/plans`: list plans.
  - `GET /api/v1/admin/plans/:id`: get plan.
  - `POST /api/v1/admin/plans`: create plan.
  - `PUT /api/v1/admin/plans/:id`: update plan.
  - `DELETE /api/v1/admin/plans/:id`: delete plan.
  - `GET /api/v1/admin/user_plans`: list sold plans.
  - `POST /api/v1/admin/user_plans/assign`: assign plan.
  - `PUT /api/v1/admin/user_plans/:id`: update user plan.
  - `DELETE /api/v1/admin/user_plans`: delete user plans.
  - `GET /api/v1/admin/user_packages`: list user packages.
- **Finance**
  - `GET /api/v1/admin/orders`: list orders.
  - `POST /api/v1/admin/recharge`: recharge.
- **Announcements**
  - `GET /api/v1/admin/announcements`: list.
  - `POST /api/v1/admin/announcements`: create.
  - `PUT /api/v1/admin/announcements/:id`: update.
  - `DELETE /api/v1/admin/announcements/:id`: delete.
- **System**
  - `GET /api/v1/admin/system_info`: get system info.
  - `POST /api/v1/admin/system_info`: update system info.
  - `POST /api/v1/admin/upload/image`: upload image.
  - `GET /api/v1/admin/api_key`: get API key.
  - `PUT /api/v1/admin/api_key`: update API key.
  - `POST /api/v1/admin/api_key/reset`: reset API key.
  - `POST /api/v1/admin/ws/dispatch`: dispatch WS test.
- **Users/Domains**
  - `GET /api/v1/admin/domains`: list user domains (admin).
  - `GET /api/v1/admin/users`: list users.
  - `PUT /api/v1/admin/users/:id/status`: toggle status.
  - `PUT /api/v1/admin/users/:id`: update user.
  - `DELETE /api/v1/admin/users/:id`: delete user.
  - `POST /api/v1/admin/users/:id/purge/reset`: reset purge quota.
  - `POST /api/v1/admin/users/:id/impersonate`: impersonate.
- **Sites**
  - `GET /api/v1/admin/sites`: list sites.
  - `POST /api/v1/admin/sites`: create site.
  - `POST /api/v1/admin/sites/batch`: batch create.
  - `GET /api/v1/admin/sites/batch/:id/progress`: batch progress.
  - `POST /api/v1/admin/sites/batch_update`: batch update.
  - `POST /api/v1/admin/sites/batch_action`: batch action.
  - `POST /api/v1/admin/sites/apply_cert`: apply cert.
  - `GET /api/v1/admin/sites/export`: export.
  - `GET /api/v1/admin/sites/resolve`: resolve DNS.
  - `GET /api/v1/admin/sites/:id`: get site.
  - `PUT /api/v1/admin/sites/:id`: update site.
  - `GET /api/v1/admin/domain_usage`: domain usage.
  - `GET /api/v1/admin/site_groups`: list groups.
  - `POST /api/v1/admin/site_groups`: create group.
  - `PUT /api/v1/admin/site_groups/:id`: update group.
  - `DELETE /api/v1/admin/site_groups/:id`: delete group.
  - `GET /api/v1/admin/site_defaults`: list defaults.
  - `POST /api/v1/admin/site_defaults`: create default.
  - `PUT /api/v1/admin/site_defaults/:name`: update default.
  - `DELETE /api/v1/admin/site_defaults/:name`: delete default.
- **Certs**
  - `GET /api/v1/admin/certs`: list.
  - `POST /api/v1/admin/certs`: upload.
  - `PUT /api/v1/admin/certs/:id`: update.
  - `DELETE /api/v1/admin/certs/:id`: delete.
  - `POST /api/v1/admin/certs/batch_action`: batch action.
  - `POST /api/v1/admin/certs/reissue`: reissue.
  - `GET /api/v1/admin/certs/:id/download`: download.
  - `GET /api/v1/admin/certs/default_settings`: get default settings.
  - `POST /api/v1/admin/certs/default_settings`: update default settings.
- **Forward**
  - `GET /api/v1/admin/forwards`: list.
  - `POST /api/v1/admin/forwards`: create.
  - `POST /api/v1/admin/forwards/batch`: batch create.
  - `POST /api/v1/admin/forwards/batch_update`: batch update.
  - `POST /api/v1/admin/forwards/batch_action`: batch action.
  - `GET /api/v1/admin/forward_groups`: list groups.
  - `POST /api/v1/admin/forward_groups`: create group.
  - `PUT /api/v1/admin/forward_groups`: update group.
  - `DELETE /api/v1/admin/forward_groups`: delete group.
  - `GET /api/v1/admin/forward_defaults`: list defaults.
  - `POST /api/v1/admin/forward_defaults`: create default.
  - `DELETE /api/v1/admin/forward_defaults`: delete default.
- **Tasks**
  - `GET /api/v1/admin/tasks`: list tasks.
  - `POST /api/v1/admin/tasks`: create task.
  - `GET /api/v1/admin/tasks/usage`: usage.
  - `POST /api/v1/admin/tasks/:id/resubmit`: resubmit.
- **Rules**
  - `GET /api/v1/admin/rules/cc/groups`: list CC rule groups.
  - `POST /api/v1/admin/rules/cc/groups`: create CC rule group.
  - `PUT /api/v1/admin/rules/cc/groups/:id`: update CC rule group.
  - `GET /api/v1/admin/rules/cc/groups/:id`: get CC rule group.
  - `GET /api/v1/admin/rules/cc/matchers`: list matchers.
  - `GET /api/v1/admin/rules/cc/matchers/:id`: get matcher.
  - `POST /api/v1/admin/rules/cc/matchers`: create matcher.
  - `PUT /api/v1/admin/rules/cc/matchers/:id`: update matcher.
  - `GET /api/v1/admin/rules/cc/filters`: list filters.
  - `GET /api/v1/admin/rules/cc/filters/:id`: get filter.
  - `POST /api/v1/admin/rules/cc/filters`: create filter.
  - `PUT /api/v1/admin/rules/cc/filters/:id`: update filter.
  - `DELETE /api/v1/admin/rules/cc/filters/:id`: delete filter.
  - `GET /api/v1/admin/rules/acl`: list ACL.
  - `GET /api/v1/admin/rules/acl/:id`: get ACL.
  - `POST /api/v1/admin/rules/acl`: create ACL.
  - `PUT /api/v1/admin/rules/acl/:id`: update ACL.
  - `DELETE /api/v1/admin/rules/acl/:id`: delete ACL.

### User
- **Domains/Config**
  - `GET /api/v1/user/domains`: list domains.
  - `POST /api/v1/user/domains`: create domain.
  - `GET /api/v1/user/domains/:id/config`: get domain config.
  - `GET /api/v1/user/config_items`: list config items.
  - `POST /api/v1/user/config_items`: upsert config item.
- **Profile**
  - `GET /api/v1/user/profile`: profile.
  - `PUT /api/v1/user/profile`: update profile.
  - `PUT /api/v1/user/password`: update password.
  - `POST /api/v1/user/recharge`: recharge.
- **Orders/Logs/Messages**
  - `GET /api/v1/user/orders`: list orders.
  - `GET /api/v1/user/logs/operation`: operation logs.
  - `GET /api/v1/user/messages`: list messages.
  - `POST /api/v1/user/messages/:id/read`: mark read.
  - `GET /api/v1/user/message_sub`: get subscriptions.
  - `PUT /api/v1/user/message_sub`: update subscriptions.
- **API Key**
  - `GET /api/v1/user/api_key`: get key.
  - `PUT /api/v1/user/api_key`: update key.
  - `POST /api/v1/user/api_key/reset`: reset secret.
- **Sites**
  - `GET /api/v1/user/sites`: list sites.
  - `POST /api/v1/user/sites`: create site.
  - `GET /api/v1/user/domain_usage`: domain usage.
- **Certs**
  - `GET /api/v1/user/certs`: list.
  - `POST /api/v1/user/certs`: upload.
  - `PUT /api/v1/user/certs/:id`: update.
  - `DELETE /api/v1/user/certs/:id`: delete.
  - `POST /api/v1/user/certs/batch_action`: batch action.
  - `POST /api/v1/user/certs/reissue`: reissue.
  - `GET /api/v1/user/certs/default_settings`: get default settings.
  - `POST /api/v1/user/certs/default_settings`: update default settings.
- **Tasks**
  - `GET /api/v1/user/tasks`: list tasks.
  - `POST /api/v1/user/tasks`: create task.
  - `GET /api/v1/user/tasks/usage`: usage.
  - `POST /api/v1/user/tasks/:id/resubmit`: resubmit.
- **Plans/Packages**
  - `GET /api/v1/user/plans`: list plans.
  - `GET /api/v1/user/user_packages`: list user packages.
  - `PUT /api/v1/user/user_packages/:id`: update package.
  - `POST /api/v1/user/user_packages/:id/renew`: renew.
  - `POST /api/v1/user/user_packages/:id/switch`: switch.
- **Site Groups/Defaults**
  - `GET /api/v1/user/site_groups`: list.
  - `POST /api/v1/user/site_groups`: create.
  - `PUT /api/v1/user/site_groups/:id`: update.
  - `DELETE /api/v1/user/site_groups/:id`: delete.
  - `GET /api/v1/user/site_defaults`: list.
  - `POST /api/v1/user/site_defaults`: create.
  - `PUT /api/v1/user/site_defaults/:name`: update.
  - `DELETE /api/v1/user/site_defaults/:name`: delete.
- **DNS**
  - `GET /api/v1/user/dns/providers`: list providers.
  - `GET /api/v1/user/dns/providers/types`: list provider types.
  - `GET /api/v1/user/dnsapi`: list DNS API.
  - `GET /api/v1/user/dnsapi/types`: list DNS API types.
- **Rules**
  - `GET /api/v1/user/rules/cc/groups`: list CC groups.
  - `POST /api/v1/user/rules/cc/groups`: create CC group.
  - `PUT /api/v1/user/rules/cc/groups/:id`: update CC group.
  - `GET /api/v1/user/rules/cc/groups/:id`: get CC group.
  - `GET /api/v1/user/rules/cc/matchers`: list matchers.
  - `GET /api/v1/user/rules/cc/matchers/:id`: get matcher.
  - `POST /api/v1/user/rules/cc/matchers`: create matcher.
  - `PUT /api/v1/user/rules/cc/matchers/:id`: update matcher.
  - `GET /api/v1/user/rules/cc/filters`: list filters.
  - `GET /api/v1/user/rules/cc/filters/:id`: get filter.
  - `POST /api/v1/user/rules/cc/filters`: create filter.
  - `PUT /api/v1/user/rules/cc/filters/:id`: update filter.
  - `DELETE /api/v1/user/rules/cc/filters/:id`: delete filter.
  - `GET /api/v1/user/rules/acl`: list ACL.
  - `GET /api/v1/user/rules/acl/:id`: get ACL.
  - `POST /api/v1/user/rules/acl`: create ACL.
  - `PUT /api/v1/user/rules/acl/:id`: update ACL.
  - `DELETE /api/v1/user/rules/acl/:id`: delete ACL.
- **Logs/Stats**
  - `GET /api/v1/user/logs/access`: access logs.
  - `GET /api/v1/user/logs/block/current`: block current.
  - `GET /api/v1/user/logs/block/stats`: block stats.
  - `GET /api/v1/user/logs/block/history`: block history.
  - `GET /api/v1/user/stats/basic`: basic stats.
  - `GET /api/v1/user/stats/quality`: quality stats.
  - `GET /api/v1/user/stats/origin`: origin stats.
  - `GET /api/v1/user/stats/ranking`: ranking stats.
  - `GET /api/v1/user/usage`: usage stats.
- **Forward**
  - `GET /api/v1/user/forwards`: list.
  - `POST /api/v1/user/forwards`: create.
  - `POST /api/v1/user/forwards/batch`: batch create.
  - `POST /api/v1/user/forwards/batch_update`: batch update.
  - `POST /api/v1/user/forwards/batch_action`: batch action.
  - `GET /api/v1/user/forward_groups`: list groups.
  - `POST /api/v1/user/forward_groups`: create group.
  - `PUT /api/v1/user/forward_groups`: update group.
  - `DELETE /api/v1/user/forward_groups`: delete group.
  - `GET /api/v1/user/forward_defaults`: list defaults.
  - `POST /api/v1/user/forward_defaults`: create defaults.
  - `DELETE /api/v1/user/forward_defaults`: delete defaults.

### Agent
- `POST /api/v1/agent/heartbeat`: node heartbeat (sync_action hint).
- `POST /api/v1/agent/node/sync`: node sync status report.
- `GET /api/v1/agent/config`: config pull (EdgeConfig).
- `GET /api/v1/agent/tasks`: task list (config_sync/package_sync).
- `POST /api/v1/agent/tasks/:id/finish`: task completion.
- `GET /api/v1/agent/l2/nodes`: L2 nodes info.
- `POST /api/v1/agent/l2/heartbeat`: L2 heartbeat.
- `POST /api/v1/agent/certs/issued`: cert issuance callback.
- `POST /api/v1/agent/logs/access`: access log upload.
- `POST /api/v1/agent/logs/metrics`: metrics upload.
- `POST /api/v1/agent/logs/events`: event upload.

## Redundant / Alias APIs (Full List)

- `POST /api/v1/admin/nodes/batch_action` == `POST /api/v1/admin/nodes/batch`.
- `/api/v1/login`, `/api/v1/admin/login`, `/api/v1/user/login` share the same login handler.
- Many admin/user routes are identical except for role scope (expected redundancy, not removal).

## Gaps / Unknowns (To confirm)

- Some WAF rule behaviors are stubbed in Lua; full “commercial” rule set may need additional modules.
- Redundant endpoints beyond `batch_action` should be audited via routes in `cdn-system/api/routers/setup.go`.

## Workflow

1) Identify the feature area and open the matching reference file under `references/`.
2) Map the feature to: frontend UI, API shape, config sync to nodes, and node-side execution.
3) Favor push/pull config sync mechanisms that can hot-reload or restart safely.
4) Keep UI aligned to `https://demo.cdnfly.cn/dashboard/admin-home`.

## Implementation Architecture

### System Architecture
- **Master (API Server)**: Go + Gin framework, manages configuration, users, sites, and sync tasks. Stores data in MySQL and ClickHouse (for logs/metrics).
- **Edge Nodes (Agent)**: Go program that unpacks embedded OpenResty/Nginx and Lua scripts. Pulls configuration from master via API and reloads Nginx dynamically.
- **Frontend**: Vue.js admin interface for managing the system.

### Key Components
- **Configuration Sync**: API-based pull model, not push. Agents poll for config changes using version numbers. Changes trigger version bump and task creation for sync.
- **Nginx Config Generation**: Dynamic config files generated from JSON payload. Includes upstreams, server blocks, cache rules, Lua integration.
- **Database Schema**: Full schema in `../cdnfly-admin-frontend-rule/references/db.sql`. Uses foreign keys and triggers for data integrity.
- **Authentication**: JWT tokens for admin/user sessions. Agent uses Bearer tokens for API access.
- **Tasks**: Asynchronous task system for purge/preheat/cert issuance. Agents poll tasks and report completion.

### Workflow Details
1. Admin makes changes via frontend -> API saves to DB -> BumpConfigVersion() increments global version.
2. NotifyConfigChanged() creates a "config_sync" task (though currently agents pull directly).
3. Agents poll `/api/v1/agent/config` periodically, compare versions, pull new config if changed.
4. Agent generates Nginx conf files from JSON, runs `nginx -t` test, then `nginx -s reload`.
5. Heartbeat every 3s, logs/metrics ship every 10s/5s.

### Specific Implementations
- **Cache**: Nginx proxy_cache with Lua for custom keys and rules. Rules prioritized by priority field.
- **CC Protection**: Lua scripts handle rate limiting, CAPTCHA challenges, and rule matching.
- **SSL**: ACME HTTP-01 challenge served via Nginx or proxied to master.
- **DNS**: Integration with providers like DNSPod, Cloudflare for automated CNAME/records.
- **Node Groups**: L1 regions (large) and L2 line groups (small) for routing. Nodes assigned to groups with weights/roles.
- **Forwarding**: Stream forwarding for TCP/UDP load balancing.

## References

- `references/anzhuangshuoming.md`: installation flow and system topology.
- `references/DNSshezhi.md`: DNS provider setup and CNAME usage.
- `references/jiedianfenzu.md`: node groups, line groups, and routing behavior.
- `references/CCcanshupeizhi.md`: CC parameter configuration.
- `references/quanjupeizhi.md`: global system settings.
- `references/chongzhishezhi.md`: payment/recharge settings.
- `references/SMTPshezhi.md`: SMTP setup.
- `references/ruhepeizhiL2jiedian.md`: L2 node setup.
- `references/wangzhanbianji.md`: site edit behavior.
- `references/huancunpeizhi.md`: cache settings.
- `references/shuaxinyure.md`: purge/preheat workflow.
- `references/zhengshuguanli.md`: certificate management.
- `references/CCguize.md`: CC rule management.
- `references/ACLguanli.md`: ACL management.
- `references/quyuguanli.md`: region management.
- `references/jiedianguanli-api.md`: node management API and monitoring.
- `references/xianluzuguanli.md`: line group management.
- `references/xianluguanli.md`: line management (ISP/geo).
- `references/jichutaocanguanli.md`: base package management.
- `references/yonghuxiangguan.md`: user management.
- `references/yonghutaocanguanli.md`: user package management.
- `references/apikeyguanli.md`: API key management.
- `references/yonghuduangongneng-tongbu.md`: user-side config sync to nodes.
- `references/monitor-tongbu.md`: monitoring/logs sync.
- `references/user-api-metadata.md`: user API coverage map.

## Recent Updates to Capture

- **Agent sync payload expansion**: `config_service.GenerateConfigForNode` now populates `acl_rules`, `acl_default_action`, `white_ips`, and `region_block` from site/global settings so agents can enforce ACL and geo-blocking in Lua (see `cdn-system/api/services/config_service.go` and `cdn-system/api/models/config_models.go`).
- **Lua runtime enhancements**: Added `resty.ip2region` support inside the agent package (`cdn-system/agent/assets/lua/resty/ip2region/{ip2region,xdb_searcher}.lua`) and wired the Lua access/guard scripts to use `_G.IP_SEARCHER` geo lookups with white/black list + region checks before ACL/CC handling.
- **Geo validation test helper**: Added `ip2region_test.lua` to hit real CN/HK IPs through the packaged `.xdb` so you can confirm region codes and include the list in future rulesets.
- **Agent deployment notes**: Local WSL node now copies the new Lua modules into `/usr/local/goedge/nodes/edge-node/lua/resty/ip2region`, reloads Nginx, and keeps `http2` disabled until the embedded OpenResty build includes the module; ACL changes trigger `config_sync` tasks for rollout.
- **Testing artifacts**: Keep the PowerShell smoke scripts (`cdn-system/api/scripts/smoke_*`) up to date; run them after ACL/region tweaks to confirm API/WS/agent endpoints are healthy, and record real-client hits (logs show `access.json` entries with 403/404 results).
- **WAF shim integration**: Implemented `resty.filter_req` in `cdn-system/agent/assets/lua/resty/filter_req.lua`, reusing IP block/anti-CC/WAF helpers so `cdnfly_wrapper` loads cleanly and WAF/CC checks execute before ACL; copy it to `/usr/local/goedge/nodes/edge-node/lua/resty` during deployment.
- **Site/package sync coverage (tracked in plan)**: `cdn-system/docs/agent-sync-plan.md` is the source of truth for what must be in the WS payload and persisted on agent: WAF/global nginx/resources/error pages, plus package/site-level ACL/CC/limits and hard states (traffic_limit/timeout). Use it to verify `config_sync` tasks include all fields, and confirm agent persistence under `/etc/cdn/*` (or chosen path) before claiming full coverage.
- **Agent persistence paths (current implementation)**: resources -> `WorkDir/conf/resources.json`, error pages -> `WorkDir/conf/error_pages.json` + `WorkDir/conf/error_pages/*.html`, default_config -> `WorkDir/conf/default_config.json`, CC rules -> `WorkDir/conf/cc_rules.json` + matchers/filters, Nginx globals -> `WorkDir/conf/nginx.conf` + `conf/dynamic/*`, packages -> `WorkDir/packages/{package_id}.json`.
