# Node Groups / Line Groups

## Scope
- Region (L1) is a large grouping: each node belongs to exactly one region.
- Line group (L2) is a small grouping: a node can belong to multiple line groups.
- Sites should inherit a default line group at creation, but can change later.

## UI
- Region list with add/edit/delete.
- Line group list with region assignment and L2 settings.
- Resolution config view to assign nodes to line groups with primary/backup roles.

## API
- CRUD for regions and line groups.
- Endpoints to assign nodes to line groups and set backup/weight/sort.

## Node/Sync
- Sync line group changes to nodes.
- L1 should periodically check L2 availability (agent-side health checks).
- Backup node switching modes must be enforced by node runtime.
