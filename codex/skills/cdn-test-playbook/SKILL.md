---
name: cdn-test-playbook
description: Testing playbook for this CDN project. Use when validating API/WS/agent changes, config sync persistence, nginx reload, WAF/ACL/CC rules, error pages, CK pipelines, or frontend pages that call APIs.
---

# CDN Test Playbook

Use this skill to run the required test steps after any change.

## Always-run checks (backend/agent)

1) **API smoke**
   - Run: `cdn-system/api/scripts/smoke_api.ps1`
   - Goal: core admin API endpoints return 2xx.

2) **WS smoke**
   - Run: `cdn-system/api/scripts/smoke_ws.ps1`
   - Goal: WS dispatch + agent config sync path is healthy.

3) **Error pages smoke**
   - Run: `cdn-system/api/scripts/smoke_error_pages.ps1`
   - Goal: error page endpoints respond with expected status codes.

4) **Agent e2e + persistence + nginx**
   - Run: `cdn-system/tests/agent_e2e/run.ps1`
   - Required flags: `-ApiBase -AdminUser -AdminPass -AgentToken -NodeId`
   - The script validates:
     - Agent endpoints (config/tasks/l2/heartbeat) are 2xx
     - Config is persisted under `edge-node/conf/*`
     - `nginx -t` passes for the generated config
     - No reload errors in `/tmp/agent_e2e.out`

## Feature-specific checks (run when touched)

### Agent config sync / persistence / reload
- Trigger a `config_sync` and confirm:
  - JSON config files updated in `edge-node/conf/`
  - `nginx -t` passes and reload succeeds
- If agent is restarted, ensure persisted configs load and nginx can start.

### Error pages and state pages
- Confirm configured error pages come from config `error_page` JSON.
- Validate `traffic_limit`, `site_locked`, `timeout`, `ip` blocks return their error pages via Nginx.

### WAF / ACL / CC
- Verify allow/deny ACL rules, default deny, and white/black IP behavior.
- Verify CC rules load (`cc_rules.json`, `cc_matchers.json`, `cc_filters.json`) and trigger as expected.

### Region block
- Use `ip2region_test.lua` to confirm IP-to-region mapping for test IPs.
- Validate region blocking in Nginx/Lua with real CN/HK IPs if available.

### ClickHouse metrics/logs pipeline
- Generate real traffic and confirm CK counts increase.
- Validate access log / metrics / event ingestion end-to-end.

## Frontend change requirement

If any frontend page or component is changed:
1) Identify the APIs used by that page.
2) Verify those APIs respond correctly (browser or direct call).
3) Confirm the page can fetch and render the API data without errors.

## Playwright timeout rule

- Use a 30s timeout for Playwright network waits and expectations unless explicitly overridden.
- Keep `web/admin/playwright.config.ts` aligned to `timeout: 30000` and `expect.timeout: 30000`.

## Notes

- Agent must always connect to API; do not use direct DB access from agent.
- Keep test code under `cdn-system/tests/` for easy removal.
