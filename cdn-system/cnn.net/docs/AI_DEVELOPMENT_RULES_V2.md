# CNN.NET AI Development Rules v2

## 1. Scope and Goals
- Target: `cnn.net` should replicate `../cdn-system` core behavior on pure .NET runtime.
- Reverse proxy runtime: built-in ASP.NET + YARP, no OpenResty dependency.
- Rule system: all Lua logic must become configuration-driven C# rules.
- Dynamic extensions: complex rules may be loaded via signed C# DLL plugins.
- Primary constraint: design for AI coding agents (deterministic, explicit, low ambiguity).

## 2. Non-Negotiable Engineering Rules
- All runtime config changes must support hot-reload without process restart.
- Every write operation must be idempotent by `version` or `request_id`.
- No module can call another module's storage directly; call interface/service only.
- No hidden global state; all runtime states must be DI-managed singleton services.
- On parse/validation failure: keep last-good runtime snapshot, do not clear active routes.
- Any fallback mode must be observable in API/metrics/logs.

## 3. Code Style for AI Agents
- C# 12, nullable enabled, explicit DTO fields.
- Namespaces map 1:1 to folder hierarchy.
- Service contracts first: create `I*` before implementation.
- Pure functions preferred for compile/transform stages.
- Side effects only in `Runtime`, `Store`, `Writer`, `Client` classes.
- For task handlers:
  - parse payload
  - validate payload
  - execute service
  - return normalized ack (`success|fail|ignored`)
- Error text must be short and machine-parsable (`invalid payload`, `version conflict`, `apply failed`).

## 4. Permission Hierarchy Framework (Low Coupling)

### 4.1 Canonical role levels
- `root`: internal full control (system migration/recovery).
- `admin`: platform admin control.
- `operator`: operations team (runtime changes, no billing/user super-admin actions).
- `user`: tenant/user-level operations.
- `agent`: edge-node identity only.

### 4.2 Permission model
- Use policy key strings, not hardcoded path checks in controllers.
- Format: `domain:action[:scope]`
  - examples: `site:read`, `site:update`, `cert:issue`, `node:config:write`
- Maintain a policy catalog and minimum role per policy.
- Middleware resolves principal role + requested policy, then checks `role_level >= policy_level`.

### 4.3 Enforcement layering
- L1: auth middleware (identity integrity)
- L2: policy middleware (role/permission)
- L3: resource guard in service layer (tenant ownership/package limits)

### 4.4 Required audit fields
- `trace_id`, `user_id`, `role`, `policy`, `resource_id`, `result`, `error_code`, `duration_ms`.

## 5. Logging Read/Write Framework

### 5.1 Log channels
- `access`: request/response traffic logs
- `security`: ACL/WAF/CC decisions
- `system`: runtime state and lifecycle logs
- `debug`: temporary deep diagnostics
- `manual_debug`: operator/AI injected debug annotations

### 5.2 Log writer contract
- Single abstraction: `ILogWriter.Write(channel, eventName, payload, context)`.
- Writer implementation can fan out to file + ship queue.
- All writes must be non-blocking to request path where possible.

### 5.3 Log switch system (hot)
- Central runtime switch store (`IDebugSwitchStore`) with persisted snapshot.
- Key examples:
  - `ship_access_logs`
  - `ship_stream_logs`
  - `ship_metrics`
  - `manual_debug_log`
  - `runtime_verbose`
- Switch updates are task-driven and auditable.

### 5.4 Manual debug logs
- Endpoint/task writes structured JSON lines (`.jsonl`).
- Minimum fields: `timestamp`, `category`, `message`, `actor`, `data`.
- Intended for temporary diagnosis; can be disabled by switch.

## 6. Runtime Safety and Performance Baseline
- Keep last-good snapshot in memory for proxy/rule runtime.
- Protect apply path with lock/semaphore; single writer, multi reader.
- Use bounded queue for async shipping to avoid memory blowup.
- Backpressure strategy on ship failure: retry with cap, then drop-oldest with counter.
- Use periodic metrics emission for:
  - apply latency
  - active route/cluster count
  - config version
  - drop/retry counters

## 7. AI Execution Workflow (Deterministic)
1. Implement contracts and DTOs.
2. Implement in-memory runtime and validator.
3. Implement persistence adapter.
4. Wire task handler to runtime.
5. Add fallback and last-good behavior.
6. Add logs/metrics and switch control.
7. Add tests: parser, validator, apply idempotency, rollback behavior.

## 8. Current Implementation Status (2026-04-01)
- Dynamic YARP runtime: in progress and already wired to WS config apply path.
- Debug switch + manual debug log framework: implemented in `Cnn.Agent` runtime.
- API-side permission-policy: upgraded to method+path resolver and unified `agent` permission checks in middleware, continuing to expand finer-grained policy coverage.
- Log pipeline convergence: channel-aware backpressure/drop accounting + local query/retention worker implemented, and API-side `node_events` query surface is now available via admin event log endpoint.
- Permission granularity: log routes are split to dedicated permissions (`log:read:security` / `log:read:user`) with resolver and catalog tests.
- Plugin safety baseline: dynamic DLL loading now enforces hash/signature verification, plugin root-directory isolation, max assembly size limit, allowlist-based admission, and lifecycle audit events (`plugin_loaded` / `plugin_rejected`).

## 9. Hard Prohibitions (Must Reject in Review)

### 9.1 Forbidden code patterns
- No cross-module direct DB access (especially controller -> db or module A -> module B storage table).
- No mutable static/global runtime state that bypasses DI lifecycle.
- No "catch (Exception) { }" silent swallow; every catch must either rethrow or return machine-parsable error + structured log.
- No hidden fallback that changes behavior without audit/metric.
- No magic protocol strings scattered in handlers (state, policy, error code must be centralized constants/catalog).
- No runtime behavior depending on undocumented implicit defaults.
- No mixed responsibilities in one class (controller + domain logic + persistence in the same file).
- No write path without idempotency key/version gate.
- No feature merge without at least one deterministic automated test for the primary success path and one failure path.
- No bypassing permission checks by route-level shortcut.

### 9.2 Forbidden architectural drift
- No new feature that introduces a second control plane for the same concern.
- No parallel "temporary" runtime models that diverge from canonical DTO/contracts.
- No plugin capability that can modify critical runtime path without signature and allowlist checks.
- No new queue/buffer without explicit bound and overflow policy.

## 10. Design Compliance Rules (Must Follow)

### 10.1 Contract and layering
- Contract-first: define/update DTO + interface before implementation.
- Keep boundaries strict:
  - Endpoint: validation + auth + mapping only.
  - Service: orchestration + policy decisions.
  - Runtime/Store: side effects and state transitions.
- State machine changes require:
  - explicit transition table,
  - terminal-state definition,
  - retry/timeout rule,
  - observability fields (`ret`, `retry_at`, counters).

### 10.2 Observability and safety
- Every fallback/retry/degradation path must expose:
  - API-visible state,
  - metric counter,
  - structured log with `trace_id`.
- Last-good snapshot is mandatory for dynamic runtime apply.
- Hot-reload apply must be atomic from caller perspective (all-or-last-good).

### 10.3 Reviewable determinism
- All new config options must declare:
  - key name,
  - value range/default,
  - invalid input behavior.
- Error responses must be short and stable for automation parsing.
- Time-related fields in APIs must use explicit absolute formats (ISO or unix seconds), no ambiguous locale text.

## 11. Anti-Entropy Constraints (Prevent Requirement Bloat)

### 11.1 Change admission checklist
- Any new "major" capability must ship with:
  - one-page scope boundary (what is explicitly out-of-scope),
  - migration/rollback strategy,
  - compatibility impact statement,
  - test matrix delta.
- If not provided, feature should not enter implementation.

### 11.2 Complexity budgets
- Per incremental feature PR:
  - max 1 new runtime state machine (unless approved by ADR),
  - max 1 new persistent schema concept,
  - max 1 new background worker.
- Reuse existing channel/task/transport before creating a new one.
- Prefer extension points over forks:
  - policy catalog extension,
  - plugin capability extension,
  - DTO version extension.

### 11.3 Ownership and documentation gate
- Each new module must define:
  - owner,
  - SLO/health signal,
  - oncall troubleshooting entry.
- Code without matching doc updates (contract/config/ops note) is incomplete.

### 11.4 DoD additions
- Definition of Done for production-ready changes:
  - architecture constraints pass,
  - test coverage added (success + failure + stale/replay when applicable),
  - logs/metrics/audit fields validated,
  - rollback path verified,
  - docs + project skill updated.
