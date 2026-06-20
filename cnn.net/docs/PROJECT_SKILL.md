# CNN.NET Project Skill (Execution Guardrails)

This document is the project-level skill for human and AI contributors.  
If any lower-level implementation choice conflicts with this file, this file wins.

## 1. Mission
- Keep cnn.net maintainable under requirement growth.
- Prefer explicit contracts and deterministic runtime behavior over local convenience.
- Optimize for safe evolution: small increments, observable state, reversible changes.

## 2. Non-Negotiables
- Contract-first: DTO/interface/state transitions before code path implementation.
- Idempotent writes: every task/apply/write path must be replay-safe.
- Last-good runtime: invalid apply never destroys active serving state.
- Bounded resources: every queue/cache/buffer must define size and overflow behavior.
- Structured observability: error/fallback/retry must be queryable, not only visible in ad-hoc logs.

## 3. Code We Reject
- Cross-module storage coupling (`module A` directly writing `module B` tables).
- Silent catch blocks or error hiding.
- New global mutable singleton state outside DI-managed services.
- Handler-level business orchestration mixed with persistence.
- New magic strings for states/policies/errors when a central catalog exists.
- Unbounded goroutine/thread/task loops without cancel token and backoff cap.
- "Temporary dual path" logic that has no removal deadline.

## 4. Required Design Discipline

### 4.1 Layering
- Endpoint layer:
  - auth + input validation + response mapping only.
- Service layer:
  - orchestration + policy + state transition decisions.
- Runtime/store/adapter layer:
  - side effects and persistence.

### 4.2 State machine rules
- Any state machine change must include:
  - allowed transitions list,
  - terminal-state semantics,
  - retry rules and retry budget,
  - timeout and stale-handling behavior.

### 4.3 Config rules
- Every config key must define:
  - key, type, default, valid range,
  - invalid-value behavior,
  - hot-reload behavior.

## 5. Growth Control (Anti-Bloat)

### 5.1 Scope guard
- Every feature must state "out of scope" explicitly.
- If scope crosses more than one bounded context, split into phases.

### 5.2 Complexity budget per feature
- Prefer 1 vertical slice per PR.
- Avoid adding multiple new abstractions in the same change:
  - at most one new persistent concept,
  - at most one new runtime loop/worker,
  - at most one new public API group.

### 5.3 Reuse-first policy
- Extend existing transport/task channels before creating new ones.
- Extend existing policy catalogs before creating special-case if-else chains.
- Extend existing observability pipeline before adding ad-hoc side channels.

## 6. Merge Gate Checklist
- [ ] Contracts/DTOs updated.
- [ ] Tests include success + failure (+ stale/replay when relevant).
- [ ] Metrics/log/audit fields cover retry/fallback/error paths.
- [ ] Feature can be rolled back safely.
- [ ] Documentation updated:
  - architecture/design note,
  - operator-facing behavior,
  - this project skill when constraints changed.

## 7. PR Review Triggers (Need Explicit Approval)
- Introducing new long-lived background worker.
- New persistent schema/table/column not in approved migration plan.
- Runtime behavior changes that can affect active traffic selection.
- Any bypass of policy middleware or permission resolver.

## 8. Allowed Exceptions
- Exceptions are allowed only with a short ADR note containing:
  - reason,
  - risk,
  - rollback plan,
  - removal deadline.
- No "temporary" exception without owner and deadline.
