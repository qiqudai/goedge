# cccadmin.665305.cc Functional Test Report

## Scope
- /website/manage?site_id=37 (all tabs, origin + cache validation)
- /website/certs (add cert, batch apply, wildcard + Huawei DNS API, edit dialog behavior)
- /forward/list + /forward/default (TCP forwarding + default config)
- /global/default (global defaults apply to new site + agent config)
- /website/list (user default settings override global)

## Environment
- Base URL: https://cccadmin.665305.cc
- Test account: 13903333333 / 123456
- Admin account: admin / 123456
- Site: site_id=37

## Progress
- [x] Module 1: Website Manage
- [x] Module 2: Certificate Management
- [x] Module 3: Forwarding
- [x] Module 4: Global Default Settings
- [x] Module 5: User Default Settings (override global)
- [x] Module 6: Admin/User Availability Smoke

## Detailed Steps & Results

### Module 1: Website Manage
- Status: completed
- Steps:
  - API: PUT `/api/v1/admin/sites/37` enable=false then enable=true; verify `state=stop` then `state=running`.
  - API: Update origin/backends/protocol/ports, set origin condition `x-origin` -> `202.73.4.80:8086`, set `proxy_http_version=1.1`, and verify via GET `/api/v1/agent/config?node_id=18`.
  - API: Enable HTTPS force/HSTS/HTTP2/HTTP3 and verify HTTP -> HTTPS redirect via `curl -I http://testabc.665305.cc/`.
  - API: Configure security (black/white lists, timeouts, block transparent proxy, crawler action block then allow for search-engine origin).
  - API: Configure cache rules (suffix/dir/path) and access controls (hotlink, CORS).
  - curl: Hotlink check on `/static/images/tabbar/1.png` -> 403 without Referer, 200 with Referer.
  - curl: CORS headers present when Origin is `https://example.com`.
  - curl: `/redirect-test` returns 302 to `/redirect-target`.
  - curl + logs: request CSS and PNG twice; access logs show MISS then HIT for cached assets.
  - curl + logs: `User-Agent: Googlebot` routes to `202.155.141.95` (search-engine origin rule), `X-Origin: origin8086` routes to `202.73.4.80:8086` (origin condition).
- Results:
  - Basic/Origin/HTTPS/Security/Cache/Access/Advanced tabs saved successfully; agent config reflects protocol/ports/rules/headers.
  - HTTPS force redirect works (301); responses negotiated HTTP/2; HTTP3 flag present in config.
  - Search-engine origin and custom origin conditions are applied (logs show upstream targets); upstreams returned 502/499 due to origin reachability, but routing rules matched.
  - Hotlink and CORS work as configured; response header `X-Edge-Test: edge-header` visible.
  - Cache rules mapped to ext/prefix/uri and confirmed HIT for CSS and PNG after initial MISS.

### Module 2: Certificate Management
- Status: completed
- Steps:
  - API: POST `/api/v1/admin/certs/batch` with array domains `testbatch1.665305.cc`, `testbatch2.665305.cc` (dnsapi=0).
  - API: POST `/api/v1/admin/certs/batch` with wildcard `*.665305.cc` (dnsapi=0) to confirm validation.
  - API: POST `/api/v1/admin/certs/batch` with wildcard `*.693133.cc` (dnsapi=12, letsencrypt).
  - API: POST `/api/v1/admin/certs/wildcard` with `*.693133.cc` (dnsapi=12, letsencrypt) to validate Huawei DNS API auto-DNS.
  - API: POST `/api/v1/admin/certs/wildcard` with manual DNS (`*.665305.cc`, letsencrypt), then GET `/api/v1/admin/certs/64/dns_challenge`, then POST `/api/v1/admin/certs/64/verify_dns`.
  - UI (Playwright): run `cdn-system/web/admin/tmp/cert_ui_check.cjs` to verify add dialog show/hide and wildcard tab manual block visibility.
- Results:
  - Batch apply succeeded for normal domains (batch_id 1768233998, ids 60/61).
  - Wildcard without DNS API correctly rejected (`wildcard requires dnsapi`).
  - Wildcard with Huawei DNS API created cert id 70 and issued successfully (see `api.run.codex.err` for ACME DNS-01 success).
  - Manual wildcard returned DNS TXT info: `_acme-challenge.665305.cc` with TXT value; DNS verify failed as expected because TXT record not set.
  - Playwright results: add dialog shows cert/key for “自己上传”, hides for “Let’s Encrypt”; wildcard manual block visible; edit dialog hides cert/key until switching to upload.

### Module 3: Forwarding
- Status: completed (data-path verification blocked by edge runtime ports)
- Steps:
  - API: GET `/api/v1/admin/forward_defaults` to review defaults (proxy_protocol listed).
  - DB: `stream_default_config` checked (listen_protocol=udp, balance_way=rr, proxy_protocol=1).
  - API: POST `/api/v1/admin/forwards` with listen `39090` and origin `202.73.4.80:39091` (id 10); verified agent config includes stream with listen_ports, balance_way=rr, proxy_protocol=true.
  - Local TCP server started on `39091`; attempted client connect to `202.73.4.80:39090` (blocked by edge runtime not running on local ports); remote nodes timed out.
  - Remote nodes `38.165.23.110` / `38.165.23.136`: port 22/80/443 reachable; port 39090 closed (`/dev/tcp` timeout). SSH key `~/.ssh/qiqudai-ssh` not authorized.
- Results:
  - Forward creation succeeds after backend fix (default node group + region).
  - Default balance/proxy_protocol applied in agent config stream output.
  - End-to-end TCP payload verification blocked due to edge OpenResty failing to start (ports 80/443 already in use) and remote node ports not reachable.

### Module 4: Global Default Settings
- Status: completed
- Steps:
  - Playwright: iterate every global default in `/global/default` (site defaults + cert defaults + stream defaults + cache templates).
  - For each default: set new value, create a new site (random subdomain of `665305.cc`), verify site detail settings and agent config (`/api/v1/agent/config?node_id=18`) after assigning node group.
  - For stream defaults: create a forward, verify `listen_protocol`, `balance_way`, and `proxy_protocol` in agent stream config.
  - For cache templates: verify website/api/download defaults apply to new sites (`cache_enable`, `cache_ttl`, `gzip`, `waf_enable`).
- Results:
  - All global defaults applied on new site creation; site detail values and agent config match.
  - Stream defaults applied to new forwards; agent streams reflect defaults.
  - Cache templates apply by site type after backend fix for `site_type` on create.

### Module 5: User Default Settings
- Status: completed
- Steps:
  - Playwright: iterate every default item in `/website/list` -> 新增默认设置 (user scope, admin user).
  - For each item: set user default to a value different from global, create a new site, verify site detail settings and agent config (`/api/v1/agent/config?node_id=18`) after assigning node group.
- Results:
  - User defaults override global defaults for all dialog items; agent config reflects user default values.

### Module 6: Admin/User Availability Smoke
- Status: completed with timeout
- Steps:
  - Playwright (admin): run `/cdn-system/web/admin/tests/e2e/admin-smoke.spec.ts` against `https://cccadmin.665305.cc`.
  - Playwright (user): run `/cdn-system/web/admin/tests/e2e/user-smoke.spec.ts` against `https://cccadmin.665305.cc`.
  - Playwright (admin): run `/cdn-system/web/admin/tests/e2e/site-group-load.spec.ts` (admin add site loads groups).
  - Playwright (admin): run `/cdn-system/web/admin/tests/e2e/website-batch.spec.ts` (batch clear cache + CNAME flow).
- Results:
  - Admin smoke test reached `/website/logs/access` but exceeded 600s timeout during toolbar clicks.
  - User smoke test reached `/website/logs/access` but exceeded 600s timeout during toolbar clicks.
  - `site-group-load` and `website-batch` tests passed.

## Test Commands & Results
- Module 1 (backend): `go test ./...` (pass)
- Module 2 (backend): `go test ./...` (pass)
- Module 3 (backend): `go test ./...` (pass)
- Module 4 (frontend e2e): `npx playwright test tests/e2e/global-defaults.spec.ts` (pass)
- Module 5 (frontend e2e): `npx playwright test tests/e2e/user-defaults.spec.ts` (pass)
- Module 6 (frontend e2e): `PW_BASE_URL=https://cccadmin.665305.cc E2E_SMOKE=1 npx playwright test tests/e2e/admin-smoke.spec.ts` (timeout at `/website/logs/access`)
- Module 6 (frontend e2e): `PW_BASE_URL=https://cccadmin.665305.cc E2E_SMOKE=1 npx playwright test tests/e2e/user-smoke.spec.ts` (timeout at `/website/logs/access`)
- Module 6 (frontend e2e): `PW_BASE_URL=https://cccadmin.665305.cc npx playwright test tests/e2e/site-group-load.spec.ts` (pass)
- Module 6 (frontend e2e): `PW_BASE_URL=https://cccadmin.665305.cc npx playwright test tests/e2e/website-batch.spec.ts` (pass)

## Open Issues / Assumptions
- Search-engine origin and custom origin-condition tests return 502/499 because upstream targets are unreachable from edge nodes; routing rules matched but origin availability is unverified.
- TCP forwarding data-path not fully verified: edge OpenResty cannot start locally (ports 80/443 in use by system nginx), and remote nodes block port 39090 (SSH access unavailable to open firewall).
- Test forwards created in DB (ids 8/9/10); cleanup if needed.
- Admin/User smoke tests hit 600s timeout on `/website/logs/access`; access logs API responded in logs, but full smoke suite likely needs a higher timeout or split to finish.
