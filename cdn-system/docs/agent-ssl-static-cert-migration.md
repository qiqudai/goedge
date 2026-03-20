# Agent SSL Static Certificate Migration

Date: 2026-03-20

## This change

- Removed runtime TLS certificate switching via `ssl_certificate_by_lua_block`.
- Removed `agent/assets/lua/ssl_manager.lua`.
- Removed `agent/edge-node/lua/ssl_manager.lua`.
- Added per-domain certificate persistence under `edge-node/cert/sites/`.
- Changed nginx vhost generation to reference static certificate files per domain.
- Kept fallback certificate only for default servers or domains without a usable certificate.

## Runtime behavior after the change

1. Agent receives config payload.
2. Agent writes each domain certificate/key to:
   - `edge-node/cert/sites/<domain>.pem`
   - `edge-node/cert/sites/<domain>.key`
3. Agent generates nginx config with:
   - `ssl_certificate <domain pem>`
   - `ssl_certificate_key <domain key>`
4. Agent also generates static nginx directives for:
   - fixed request headers
   - fixed response headers
   - static CORS
   - proxied-location ACL rules
   - unconditional and equivalently-renderable rewrite/redirect rules
5. Agent reloads nginx.

This removes certificate selection from the TLS handshake Lua path.

## Why this was changed

The previous design depended on OpenResty Lua to swap certificates during the handshake. That design is flexible, but it is harder to debug and more fragile at runtime. A broken Lua runtime path can make HTTPS fail even when the correct certificate data exists in `cdn_config.json`.

Static certificate files plus nginx reload make the failure surface much smaller:

- certificate file exists or not
- generated nginx config is correct or not
- nginx reload succeeds or not

## Validation status

- Static inspection completed:
  - `ssl_certificate_by_lua_block` removed from generated domain server config path.
  - `ssl_manager.lua` removed from bundled assets and runtime template tree.
  - runtime asset restore will not bring `ssl_manager.lua` back because it restores from current `agent/assets/lua`.
- Automated Go test/build in the current local environment is blocked:
  - `go test ./...` under `agent/` fails with `go: no such tool "asm"`.
  - Local Go installation is incomplete because `C:\Program Files\Go\pkg\tool\windows_amd64` does not contain `asm.exe`.

## Lua migration assessment

### Can be replaced 100% by nginx config with no functional loss

1. Dynamic TLS certificate selection
   - Current result: migrated.
   - Replacement: static `ssl_certificate` and `ssl_certificate_key` per server block.
   - Side effect: requires nginx reload when certificate data changes.
   - Functional loss: none.

2. Static response header injection
   - Current result: already generated in nginx config.
   - Scope: domain-level fixed response headers.
   - Replacement: `add_header`, `more_set_headers` if present, or generated nginx directives.
   - Side effect: none if headers are static and do not depend on upstream/runtime state.
   - Functional loss: none for fixed-value headers.

3. Static request header forwarding
   - Current result: already generated in nginx config.
   - Scope: domain-level fixed upstream request headers.
   - Replacement: `proxy_set_header`.
   - Side effect: none for constant mappings.
   - Functional loss: none for fixed-value headers.

4. CORS with fixed policy
   - Current result: migrated to generated nginx config for proxied locations.
   - Scope: allow-origin/methods/headers/expose/max-age that do not depend on runtime conditions.
   - Replacement: generated `add_header` and `if ($request_method = OPTIONS)` / dedicated location handling.
   - Side effect: nginx config becomes larger.
   - Functional loss: none for static CORS policies.

5. HSTS, HTTP/2, HTTP/3, gzip, ssl protocol/cipher tuning
   - Replacement: native nginx directives.
   - Side effect: none.
   - Functional loss: none.

6. Simple URL redirect and rewrite rules
   - Current result: unconditional and equivalently-renderable rules migrated.
   - Scope: path regex/path match rules that do not depend on geo, UA, header, IP metadata, or backend state.
   - Replacement: `return`, `rewrite`, `map`.
   - Side effect: regex-heavy config may become harder to maintain.
   - Functional loss: none for simple deterministic rules.

7. Static connection/body/rate/cache directive settings
   - Scope: `limit_conn`, body size, proxy timeout, cache zone usage, gzip, websocket switches.
   - Replacement: native nginx directives.
   - Side effect: none.
   - Functional loss: none.

### Can be replaced mostly by nginx config, but not 100% without side effects

1. Hotlink protection
   - Replacement: `valid_referers` + `if`.
   - Side effect: current Lua supports richer scope matching and custom domain lists in one path.
   - Functional loss: possible for complex scope combinations.

2. ACL allow/deny rules
   - Current result: migrated for proxied locations.
   - Replacement: `allow` / `deny`.
   - Side effect: dynamic rule generation expands config size.
   - Functional loss: none for the current exact-IP proxied-location ACL behavior.

3. Simple cache bypass decisions
   - Replacement: `map`, `proxy_no_cache`, `proxy_cache_bypass`.
   - Side effect: complex per-rule logic gets harder to express.
   - Functional loss: possible when rules depend on richer matching than nginx maps comfortably support.

4. Simple geo blocking
   - Replacement: nginx geo modules or external geo database integration.
   - Side effect: depends on module availability and database format.
   - Functional loss: likely if current Lua IP lookup behavior must remain identical.

### Should stay in Lua or another programmable layer if behavior must not change

1. WAF logic
   - Reason: request inspection, multi-rule evaluation, dynamic block decisions, custom challenge flow.
   - Side effect if moved: major capability loss unless replaced with another full WAF engine.

2. Anti-CC / guard / captcha / browser challenge
   - Reason: stateful verification, token/cookie workflows, request-rate state, custom challenge pages.
   - Side effect if moved: functional reduction is unavoidable with pure config.

3. Dynamic upstream selection based on request metadata
   - Reason: current logic can use geo, headers, URI, IP, method, and conditional origin rules.
   - Side effect if moved: config complexity explodes and matching fidelity drops.

4. Runtime metrics aggregation and structured request/response log capture
   - Reason: current logic writes metrics and serialized headers/body data through Lua.
   - Side effect if moved: pure nginx logging can cover part of it, but not the same structure or control.

5. Edge compute / programmable request handling
   - Reason: by definition this is code execution logic.
   - Side effect if moved: functional loss.

## Recommended next migration targets

If the goal is to reduce Lua surface area further without changing behavior materially, the safest next targets are:

1. Validate the migrated ACL/CORS/rewrite paths on a Linux runtime with nginx reload
2. Trim any remaining duplicated Lua branches after runtime verification
3. Revisit hotlink protection only if a narrower rule model is acceptable

These are the highest-confidence migrations with the lowest operational risk.
