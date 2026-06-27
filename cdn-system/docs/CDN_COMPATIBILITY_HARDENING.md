# CDN Compatibility Hardening

This document defines edge compatibility rules for abnormal origins. The goal is
to keep client-facing HTTP/HTTPS behavior valid even when the origin emits
protocol-specific, conflicting, or unsafe headers.

## Principles

- Preserve business semantics by default.
- Normalize connection-level protocol details at the CDN boundary.
- Never cache personalized or ambiguous responses unless explicitly configured.
- Keep WebSocket `101 Switching Protocols` behavior separate from normal HTTP
  responses.
- Prefer edge-side protocol normalization over origin-specific hacks.
- Do not hand-edit generated node config in production; fixes must come from the
  agent package or control-plane config.

## Priority 1: Protocol Correctness

These fixes are safe because they remove hop-by-hop transport metadata that
should not be forwarded by a proxy as end-to-end response metadata.

| Case | Example | Risk | CDN behavior | Status |
| --- | --- | --- | --- | --- |
| Hop-by-hop response headers on normal responses | `Upgrade`, `Connection`, `Keep-Alive`, `Proxy-Connection`, `TE`, `Trailer`, `Transfer-Encoding` | HTTP/2/HTTP/3 protocol errors, hangs, malformed responses | Strip on every non-`101` response | Implemented |
| WebSocket upgrade | `101 Switching Protocols`, `Upgrade: websocket` | WebSocket breakage if stripped | Preserve required upgrade headers when status is `101` | Implemented |
| HTTP/2 forbidden headers | `Connection`, `Upgrade`, `Transfer-Encoding` | Browser/curl `PROTOCOL_ERROR` | Ensure client-facing HTTP/2 never sees them on non-`101` responses | Implemented |
| Origin `h2c` advertisement | `Upgrade: h2,h2c` | HTTPS over HTTP/2 disconnects | Strip as hop-by-hop metadata | Implemented |
| Client `Upgrade` leakage to origin | `Upgrade: h2,h2c` on a WebSocket-enabled CDN site | Origin enters wrong upgrade path | Forward upgrade headers only when the client asks for `websocket`; clear normal request hop-by-hop headers | Implemented |
| `Expect: 100-continue` origin leakage | Large uploads, strict origins | Upload stalls or inconsistent origin behavior | Let edge own client expectation state and do not forward `Expect` by default | Implemented |
| Origin `Alt-Svc` | `Alt-Svc: h3=":443"` | Client may bypass CDN or try unsupported protocol | Hide origin `Alt-Svc`; CDN owns client protocol advertisement | Existing |
| Origin HSTS | `Strict-Transport-Security` | Browser may force HTTPS before CDN cert/config is ready | CDN owns HSTS; hide origin HSTS when CDN HSTS is enabled | Existing |

## Priority 2: Scheme And Routing Consistency

| Case | Example | Risk | CDN behavior | Status |
| --- | --- | --- | --- | --- |
| Wrong forwarded scheme | HTTPS client, HTTP origin, `X-Forwarded-Proto: http` | Redirect loops, bad absolute URLs, insecure cookies | Forward external client scheme | Existing |
| Wrong Host/SNI on HTTPS origin | `Host` differs from `proxy_ssl_name` | Origin cert failure or wrong vhost | Host and SNI must be configurable; default SNI follows origin host config | Existing |
| Location leaks wrong scheme | CDN HTTPS -> origin redirects `http://...` | HTTPS downgrade or loop | Rewrite absolute `http://` redirects to `https://` on TLS edge listeners | Existing |
| HTTP to HTTPS redirect loops | CDN HTTPS -> origin HTTP -> origin redirects HTTPS repeatedly | User sees redirect loop | Use correct `X-Forwarded-Proto`; automatic loop breaking is skipped because it may hide real application redirects | Partially implemented / skipped |

## Priority 3: Cache Safety

| Case | Example | Risk | CDN behavior | Status |
| --- | --- | --- | --- | --- |
| `Set-Cookie` with public cache | `Cache-Control: public` and `Set-Cookie` | User data leakage | Do not store responses that set cookies | Implemented |
| Sensitive request auth | `Authorization: Bearer ...` | Private API response cached under shared key | Bypass and do not store cache when `Authorization` is present | Implemented |
| Sensitive vary headers | `Vary: Cookie`, `Vary: Authorization`, `Vary: *` | Cache key explosion or private cache leak | Treat as uncacheable | Implemented |
| Conflicting cache directives | `public, no-store` | Incorrect cache persistence | Conservative directive wins: `no-store`, `no-cache`, `private` | Implemented |
| Error/redirect caching | `302`, `403`, `500` | Sticky login redirects or stale errors | Cache only configured safe statuses; default map denies unlisted upstream statuses | Existing |
| `206` / Range mismatch | Invalid `Content-Range` or multipart range | Broken video/download and cache pollution | Do not add automatic validation yet; range caching behavior remains explicit/configured | Skipped: can break video/download sites |

## Priority 4: Body And Encoding Safety

| Case | Example | Risk | CDN behavior | Status |
| --- | --- | --- | --- | --- |
| `Content-Length` + `Transfer-Encoding` conflict | Both present | Response truncation/hang/request smuggling class issues | Let edge normalize transport; strip invalid hop-by-hop TE | Priority 1 |
| Wrong `Content-Encoding` | Header says gzip but body is plain | Browser decode failure | Do not auto rewrite encoding headers | Skipped: cannot prove body encoding in header filter without buffering/latency risk |
| Range with dynamic compression | `Range` + gzip | Corrupt partial content | Keep existing explicit range/gzip behavior | Skipped: auto disabling gzip can change normal download performance |
| `204` / `304` with body metadata | `Content-Length`, `Transfer-Encoding` | Client waits or protocol error | Strip hop-by-hop `Transfer-Encoding`; keep `Content-Length` untouched | Partially implemented: stripping `Content-Length` on `304` can affect caches |

## Priority 5: Browser And Application Compatibility

| Case | Example | Risk | CDN behavior | Status |
| --- | --- | --- | --- | --- |
| Duplicate/conflicting CORS | `Access-Control-Allow-Origin: *` plus explicit origin | Browser blocks API | CDN or origin should own CORS; only configured CDN CORS emits CDN headers | Existing / skipped: automatic origin CORS removal can break APIs |
| Broken OPTIONS preflight | Origin returns `404/405` | Browser blocks API | Optional CDN-managed preflight per site config | Existing when CORS is enabled |
| Cookie attribute mismatch | `SameSite=None` without `Secure` | Login/iframe/payment failures | Do not rewrite by default | Skipped: cookie rewrite can break auth/payment flows |
| Mixed content | HTTPS page references HTTP scripts/XHR | Browser blocks resources | Prefer origin fix | Skipped: HTML/content rewriting is high risk |

## Priority 6: Operational Edge Cases

| Case | Example | Risk | CDN behavior | Status |
| --- | --- | --- | --- | --- |
| `103 Early Hints` | preload links before final response | Unsupported client/proxy behavior | No automatic forwarding changes | Skipped: needs end-to-end protocol support validation |
| `100 Continue` | Large uploads with `Expect` | Upload stalls | Clear `Expect` to origin so edge owns expectation handling | Implemented |
| Oversized response headers | Many cookies | 502 or client refusal | Keep existing buffer configuration controls | Skipped: increasing buffers globally raises memory footprint |
| Illegal header names/values | control chars, invalid names | HTTP/2 disconnect | Nginx rejects invalid upstream headers; no lossy rewrite | Skipped: silent rewrite can hide origin bugs |
| Non-standard status codes | `444`, `499` from origin | Client incompatibility | Do not auto map statuses | Skipped: mapping can change application semantics |
| HTTP/2 request concurrency and CC | Many static resources multiplexed | False-positive rate limiting | Keep minimum connection-limit floor | Existing |

## Implementation Scope

Implemented safe defaults:

1. Strip response hop-by-hop headers on all non-`101` responses.
2. Preserve `101 Switching Protocols` for WebSocket.
3. Only forward client upgrade headers to the origin when `Upgrade` is exactly
   `websocket`.
4. Clear request-side `Expect`, `Keep-Alive`, `Proxy-Connection`, `TE`,
   `Trailer`, and non-WebSocket `Upgrade` before proxying.
5. Hide upstream `Alt-Svc` on TLS edge listeners.
6. Let CDN own HSTS when CDN HSTS is enabled.
7. Rewrite absolute `http://` redirects to `https://` on TLS edge listeners.
8. Do not cache responses tied to `Set-Cookie`, `Authorization`,
   `Cache-Control: no-store/no-cache/private`, or `Vary: */Cookie/Authorization`.
9. Keep existing HTTP/2 multiplexing guard by enforcing a minimum per-IP
   connection limit.

Skipped because automatic changes can affect normal sites:

- CORS conflict rewriting.
- Cookie attribute rewriting.
- HTML/mixed-content rewriting.
- Automatic range/compression behavior changes.
- Forced redirect-loop suppression.
- Non-standard status-code mapping.
- Global response buffer enlargement.
- `Content-Length` stripping for `304`.

## Test Requirements

Before production deployment:

- `go test ./...` under `agent`.
- `go test ./...` under `api`.
- Docker compatibility test for HTTPS/HTTP2 with an origin that emits
  `Upgrade: h2,h2c` and `Connection: Upgrade`.
- Unit tests for sanitized WebSocket upgrade forwarding and cache safety maps.
- Existing Docker acceptance tests via `scripts/test_acceptance.sh`.
- Manual verification on a staging/known node:
  - HTTP/1.1 still works.
  - HTTPS/HTTP2 no longer fails with `PROTOCOL_ERROR`.
  - WebSocket `101` still upgrades.

## Production Deployment Gate

Do not update production until all of the following are true:

- The generated agent package includes both `assets/lua/response_headers.lua`
  and `edge-node/lua/response_headers.lua`.
- Docker compatibility and full acceptance tests pass.
- The rollback package/version is available.
- The change has been pushed through the agent package API, not by manually
  editing generated node config.
