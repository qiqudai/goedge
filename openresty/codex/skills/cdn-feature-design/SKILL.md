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
