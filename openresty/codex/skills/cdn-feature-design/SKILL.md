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
