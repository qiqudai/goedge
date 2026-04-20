# Remaining Gaps Execution Checklist (2026-04-20)

This checklist tracks the two known open gaps after API/script regression pass.

## Open gaps

1. Browser-level full-page regression is not closed.
2. Real online `Cnn.Agent` task consume + ACK evidence is not closed.

---

## Gap A: Browser-level full-page regression

### Current status
- API + script layer is strongly covered.
- In-repo browser E2E under `docs/tests/ui_e2e` currently has only `README.md` (no committed spec files).

### Required completion criteria
- Every key admin/user page can be opened in browser and complete at least one write/readback flow.
- For each page group, capture pass/fail evidence (trace/screenshot/log).

### Minimal execution plan
1. Start API:
   - `./scripts/start_local_api_mysql.sh`
2. Run baseline live API regression first:
   - `./scripts/verify_live_mysql_extended.sh`
3. Execute browser walkthrough (manual or Playwright) for page groups:
   - Dashboard / Stats / Logs
   - Sites / Site defaults / Batch update
   - Certs / DNS / Forward / Node / Plans
   - Global config pages
4. For each group, verify:
   - page load success
   - save success
   - refresh after save keeps value
5. Save evidence into `docs/tests/ui_e2e/` (without `node_modules` or raw cache artifacts).

### Pass gate
- No high-severity page flow failures.
- All critical settings pages have at least one verified write/readback path.

---

## Gap B: Real agent online consume + ACK

### Current status
- Service-level ACK behavior has tests (`tests/Cnn.Api.Tests/AgentTaskAckServiceTests.cs`).
- Missing item is real runtime proof with live websocket session and ACK round-trip.

### New runnable verifier
- Added script: `scripts/verify_agent_ws_ack.sh`
- Purpose: verify admin dispatch -> live node websocket -> ACK received.

### How to run
1. Ensure API is up:
   - `./scripts/start_local_api_mysql.sh`
2. Ensure at least one real agent is online on `/ws/agent`.
3. Run:
   - `./scripts/verify_agent_ws_ack.sh`

Optional environment overrides:
- `BASE_URL` (default `http://127.0.0.1:5035`)
- `ADMIN_USER` / `ADMIN_PASS`
- `NODE_ID` (if omitted, script auto-picks an online node)
- `TASK_TYPE` (default `config_sync`)
- `WAIT_SECONDS` (default `10`)
- `PAYLOAD_JSON` (default `{}`; sent as task payload string)

### Pass gate
- Dispatch API returns `code=200`.
- `data.connected=true`.
- `data.state` is not `timeout`.

---

## Final closure condition
Both gaps are considered closed only when:
1. Browser full-page regression evidence is recorded.
2. `verify_agent_ws_ack.sh` passes against a real online agent node.
