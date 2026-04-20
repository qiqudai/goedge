# PR Summary
- What changed:
- Why:
- Scope boundary (explicitly out of scope):

## Required Project Rules
- [ ] I reviewed and followed:
  - `docs/PROJECT_SKILL.md`
  - `docs/AI_DEVELOPMENT_RULES_V2.md`

## Architecture & Design Compliance
- [ ] Contract-first respected (DTO/interface/state transition defined before implementation).
- [ ] Layering respected (endpoint vs service vs runtime/store responsibilities are not mixed).
- [ ] No cross-module direct storage access introduced.
- [ ] No hidden fallback/degradation path without observability.
- [ ] Runtime apply path keeps last-good behavior (when applicable).

## Hard Prohibitions Self-Check
- [ ] No silent catch/error swallowing.
- [ ] No new mutable global/static runtime state bypassing DI.
- [ ] No unbounded queue/cache/buffer introduced.
- [ ] No permission bypass shortcut added.
- [ ] No scattered new magic strings for state/policy/error (catalog/constant used).

## Requirement Growth Control
- [ ] Complexity budget respected for this PR:
  - at most one new persistent concept,
  - at most one new runtime loop/worker,
  - at most one new public API group.
- [ ] Reuse-first applied (existing channels/policies/observability extended before adding new rails).
- [ ] If this is a major capability: scope boundary + rollback + compatibility note are included.

## Testing & Verification
- [ ] Added/updated deterministic tests for success path.
- [ ] Added/updated deterministic tests for failure path.
- [ ] Added stale/replay/idempotency test where relevant.
- [ ] Local verification commands and results are included below.

### Verification Commands
```bash
# paste executed commands here
```

### Verification Results
```text
# paste concise results here
```

## Observability & Ops
- [ ] Logs/metrics/audit fields cover retry/fallback/error path changes.
- [ ] Operator-facing behavior changes are documented.
- [ ] Rollback path is defined and feasible.

## Schema / Config / Breaking Changes
- Schema change:
- Config key change (name/default/range/invalid behavior):
- Breaking change:

## Exception (ADR Required)
- [ ] No exception.
- [ ] Exception used and ADR attached (reason/risk/rollback/removal deadline/owner).
