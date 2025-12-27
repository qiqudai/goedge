# User Features Sync

## Scope
- User-side features that must be reflected on edge nodes: sites, site groups, certs, ACL, CC, purge/preheat tasks, stream forwarding.

## API
- User endpoints for CRUD operations and task submission.
- Admin endpoints for syncing configs to nodes.

## Node/Sync
- Convert user settings into Nginx/OpenResty config.
- Apply per-domain server/upstream config generation for performance.
- Use API-based sync and reload safely.
