# Base Package Management

## Scope
- Create and manage base plans with quotas: traffic, bandwidth, connections, domains, ports.
- Control feature flags: CC rule support, websocket, HTTP3, CNAME settings.

## UI
- Plan list with enabled status and basic limits.
- Plan editor for numeric limits, feature toggles, and CNAME modes.

## API
- CRUD for base packages.
- Validate numeric fields, date fields, and defaults.

## Node/Sync
- Changes that affect runtime behavior must sync to nodes.
- Plan settings are used when creating user packages and sites.
