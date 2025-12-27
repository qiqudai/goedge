# ACL Management

## Scope
- Define allow/deny rules for site access control (IP, UA, URI, referer).
- Apply rules at HTTP edge; changes must sync to nodes.

## UI
- List ACL rules with enable/disable, priority, and remarks.
- Add/Edit: choose rule type (whitelist/blacklist), match fields, and scope (global or site group if supported).

## API
- CRUD endpoints for ACL rules (list, create, update, delete, batch actions).
- Return consistent rule payloads for frontend storage.

## Node/Sync
- Sync ACL rules to nodes and regenerate Nginx config (geo/map/deny/allow).
- Prefer Nginx native directives; avoid Lua unless required.
