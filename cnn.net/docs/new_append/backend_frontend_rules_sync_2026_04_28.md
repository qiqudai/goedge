# Backend/Frontend Rule Sync (2026-04-28)

This document summarizes all newly implemented rules and behavior changes that `cnn.net` should replicate.

## 1. User Creation and User Group Integration

## 1.1 Add User required fields
- `username` is required
- `password` is required
- `email` is required

Request should be rejected with a clear validation message if any of the above is missing.

## 1.2 User Group source of truth
- Add User dialog must load group options from User Group API, not hardcoded local options.
- Group options must come from:
  - `GET /api/v1/admin/user_groups`

## 1.3 User Group management API
- `GET /api/v1/admin/user_groups`
- `POST /api/v1/admin/user_groups`
- `DELETE /api/v1/admin/user_groups/:id`

Rules:
- Group name cannot be empty.
- Group name should be unique.
- Deleting a group must be blocked when any user is using it.

## 1.4 Real-name section behavior
- "Real-name Information" is removed from active Add User workflow.
- Backend should not depend on that step for user creation.

---

## 2. Plan Deletion Rules

## 2.1 Why deletion can fail
A base plan cannot be deleted if referenced by sold user plans (`user_package`) or downstream records linked to them.

## 2.2 Implemented guard
Before deleting `plan/package`, backend must:
1. Validate ID.
2. Check references in `user_package`.
3. If referenced, return explicit message:
   - `Plan is referenced by N user package(s). Delete sold plans first.`
4. Clear `merge_package_group` records for that plan.
5. Delete plan only if safe.

This replaces opaque DB foreign-key error behavior.

---

## 3. Blocked IP Dashboard vs List Consistency

## 3.1 Problem pattern
Dashboard blocked IP count had value, but blocked list could be empty.

## 3.2 Root cause
Blocked list query depended on geo fields (`client_country`, `client_province`) that may not exist or be incompatible in some environments.

## 3.3 Rule implemented
For blocked list queries:
- Execute primary query with geo fields.
- If query fails, fallback to a compatible query without geo dependency.
- Return list anyway with location placeholders (`-`) instead of empty list.
- Log fallback activation in backend logs.

Applied to:
- Current blocked list
- Blocked history list

---

## 4. User Deletion Rules (Critical)

## 4.1 Built-in admin protection
- Built-in admin user `ID=1` must not be deletable.
- Return explicit message:
  - `Built-in admin (ID=1) cannot be deleted`

## 4.2 Dependency pre-check before deleting any user
Backend must check related resources first (at least):
- `cert`
- `user_package`
- `site`
- `stream`
- `dnsapi`
- `acl`
- `cc_rule`
- `cc_match`
- `cc_filter`
- `api_key`

If any reference exists:
- Block deletion
- Return explicit blockers and counts, e.g.
  - `User has related resources, delete blocked: cert:2, site:5`

Do not rely on raw foreign-key error bubbling to frontend.

---

## 5. Deployment Rule for Go Binary

- Build target must be Linux x86_64:
  - `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`
- Verify artifact is ELF executable before deploy.
- Replace binary on server only after upload completes.
- Go production service restart remains manual (per operation policy).

---

## 6. Frontend Behavior Alignment

- User page tabs:
  - User List
  - User Group
- Deletion/operation failure messages should display backend explicit reason directly.
- When deletion is blocked by dependency rules, frontend should not mask server message.

---

## 7. Required Outcome in `cnn.net`

If `cnn.net` mirrors these rules, behavior will match the current Go implementation for:
- User add/edit/group flow
- Plan delete safety
- Block log consistency
- User delete safety and admin protection

