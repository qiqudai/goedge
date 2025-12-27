# L2 Node Configuration

## Scope
- Configure L2 nodes and their health checks.
- L1 nodes should periodically detect L2 availability.

## UI
- L2 node list with IP, status, and add/remove actions.

## API
- CRUD for L2 nodes.
- Health check settings (interval, timeout, mode).

## Node/Sync
- Agent implements periodic L2 health checks and reports status.
