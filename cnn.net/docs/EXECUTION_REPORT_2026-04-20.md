# Execution Report 2026-04-20

## Scope executed by Codex (no manual user operation required)

1. API startup validation
2. Extended live MySQL regression
3. Real agent websocket dispatch/ACK verification
4. Browser-level full-page regression (Playwright)
5. Browser regression failure fix and re-verification

## Commands and outcomes

### 1) Extended regression
- Command: `./scripts/verify_live_mysql_extended.sh`
- Result: Passed
- Evidence: `[O/15] Extended smoke passed`

### 2) Agent ACK end-to-end
- API run mode: `./scripts/run_local_api_mysql.sh` (foreground)
- Agent run mode:
  - `ASPNETCORE_ENVIRONMENT=Development Api__BaseUrl=http://127.0.0.1:5035 Node__Id=3 Node__Token=token dotnet run --no-launch-profile --project src/Cnn.Agent/Cnn.Agent.csproj`
- Dispatch verification command: `./scripts/verify_agent_ws_ack.sh`
- Result: ACK verified (node online consume+ack confirmed)
  - node_id: `3`
  - state: `fail`
  - error: `config apply failed`

### 3) Browser full-page regression
- Runtime env: `.runtime/browser_regression` (temporary, non-repo)
- Tooling: Playwright (Chromium)
- Route scope: 53 pages (`src/Cnn.Api/Pages` route set)
- Initial run: 52/53 pass, `/tasks` = HTTP 500
- Fix applied: `src/Cnn.Api/Services/TaskHubClient.cs`
  - changed task hub connection base URL resolution to current navigation base URI (same-origin), removing hard dependency on configured `Api:BaseUrl` mismatched port.
- Re-run result: 53/53 pass
- Full report JSON: `docs/tests/ui_e2e/full_pages_report_2026-04-20.json`

## Delivered closure status

- Browser-level full-page regression: Closed (53/53 pass)
- Real agent online consume + ACK evidence: Closed (dispatch->ack observed against live websocket node)
