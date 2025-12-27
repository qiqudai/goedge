# Certificate Management

## Scope
- Issue, upload, renew, and distribute certificates from master to nodes.
- HTTP-01 challenge via `/.well-known/acme-challenge/`.

## UI
- Cert list with status and expiry.
- Create/upload and batch issue dialogs.
- Default settings for user cert issuance.

## API
- Issue/reissue endpoints, download endpoint, default settings endpoints.
- Agent endpoint to report issued certs back to master.

## Node/Sync
- Master stores certs and pushes to nodes.
- Nodes serve ACME HTTP-01 challenge or proxy to master.
