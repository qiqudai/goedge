# API Key Management

## Scope
- Issue, rotate, and revoke API keys for users.
- Show key metadata (name, status, created time, last used if available).

## UI
- List API keys; actions: create, disable, delete.
- Display key/secret once at creation if required.

## API
- CRUD endpoints for API keys.
- Validation for ownership and role permissions.

## Security
- Hash secrets at rest when possible.
- Enforce key scopes/permissions in API middleware.
