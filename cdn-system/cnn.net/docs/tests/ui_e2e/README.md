# UI E2E Evidence

This directory stores browser-level evidence artifacts for admin UI regression work:

- screenshots such as `*_controls_YYYY-MM-DD.png`
- structured logs such as `*_controls_report_YYYY-MM-DD.json`
- gap inventories such as `remaining_test_components_2026-04-21.md`

The primary runners live in [`/.runtime/browser_regression`](/Users/fake/code/goedge/cdn-system/cnn.net/.runtime/browser_regression) so the evidence files here stay lightweight and easy to diff.

## Browser Evidence Scripts

Current scripts in [`/.runtime/browser_regression`](/Users/fake/code/goedge/cdn-system/cnn.net/.runtime/browser_regression):

- `manage_controls_check.mjs`
- `list_controls_check.mjs`
- `rules_controls_check.mjs`
- `certs_controls_check.mjs`
- `dnsapi_controls_check.mjs`
- `groups_controls_check.mjs`
- `purge_controls_check.mjs`
- `access_logs_controls_check.mjs`
- `block_logs_controls_check.mjs`

## Run

From [`/.runtime/browser_regression`](/Users/fake/code/goedge/cdn-system/cnn.net/.runtime/browser_regression):

```bash
npm install
node manage_controls_check.mjs
node list_controls_check.mjs
node rules_controls_check.mjs
node certs_controls_check.mjs
node dnsapi_controls_check.mjs
node groups_controls_check.mjs
node purge_controls_check.mjs
node access_logs_controls_check.mjs
node block_logs_controls_check.mjs
```

## Env

- `BASE_URL` default: `http://127.0.0.1:5035`
- `ADMIN_USER` default: `cnn_ai_admin`
- `ADMIN_PASS` default: `admin123`
- `ADMIN_TOKEN` optional: reuse an existing admin token instead of logging in
- `REPORT_DATE` default: `2026-04-21`
- `DNSAPI_USER_KEYWORD` optional: user keyword for the DNS API browser evidence seed flow

## Notes

- `tests/Cnn.Api.Tests/RemainingWebsiteComponentsInteractionTests.cs` covers the component-level regressions for the remaining website modules.
- The remaining work tracked in this folder is specifically browser / E2E evidence, not more duplicate bUnit assertions.
