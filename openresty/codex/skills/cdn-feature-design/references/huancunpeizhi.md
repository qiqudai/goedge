# Cache Configuration

## Scope
- Define cache rules by homepage, full site, directory, file suffix, or path.
- Specify TTLs and cache behaviors (no-cache, priority, ignore query, etc.).

## UI
- Rule list with type, match content, TTL, and enable flag.
- Add/Edit dialog supports rule type + value input where required.

## API
- CRUD for cache rules.
- Validate TTL numeric values and rule formats.

## Node/Sync
- Sync cache rules to nodes and render Nginx cache directives.
- Use native `proxy_cache` + `map`/`location` where possible.
