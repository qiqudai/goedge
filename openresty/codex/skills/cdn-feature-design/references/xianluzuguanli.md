# Line Group Management

## Scope
- Manage line groups that map to ISP/geo routing (default, telecom, unicom, mobile, overseas, search engine).
- Define backup switch logic and line behavior.

## UI
- Line group list with resolution values, L2 settings, and sort.
- Configure resolution: assign nodes, set backup roles, weights, and sort.

## API
- CRUD for line groups.
- Endpoints for node assignment and line-specific metadata.

## Node/Sync
- Sync line group changes to nodes for DNS/route behavior.
- Backup IP switching modes must be applied at runtime.
