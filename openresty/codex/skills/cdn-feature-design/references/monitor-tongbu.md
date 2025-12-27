# Monitoring Sync

## Scope
- Collect HTTP and stream metrics from nodes.
- Provide realtime stats, top lists, black IP lists, and access logs.

## Data sources
- Nginx/OpenResty access logs (HTTP and stream).
- Local blocking lists (ipset/iptables) if used.

## API
- Node pushes metrics/log batches to master via authenticated endpoints.
- Master provides query endpoints for dashboards.

## Node/Sync
- Prefer API-based sync (HTTPS + token/HMAC or mTLS).
- Avoid Redis for cross-node sync.
