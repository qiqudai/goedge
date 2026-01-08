---
name: cdnfly-admin-frontend-rule
description: Enforce CDNFly admin frontend rules and db.sql constraints when building or modifying web admin UI and CRUD flows in this project. Use for frontend pages in cdn-system/web/admin, especially user-related forms/lists, role-based visibility, and any feature that depends on the existing database schema (no schema changes).
---

# Cdnfly Admin Frontend Rule

## Overview

Apply strict frontend rules for CDNFly admin UI work. Always rely on the existing schema in `references/db.sql` and never add or expand tables/fields unless explicitly requested by the user. Use the admin-home demo as the UI reference: https://demo.cdnfly.cn/dashboard/admin-home

## Core Rules

1) Always determine the current role (admin vs user) before shaping UI behavior.
2) User role:
   - All CRUD actions target the current logged-in user only.
   - Never show the "user selection" control.
3) Admin role:
   - Show the "user selection" control.
   - Allow choosing/modifying the user via dropdown based on user info.
4) Package/plan selector:
   - Default select the first available plan; do not require the user to reselect.
5) Action buttons:
   - All text action buttons must use size `normal`.
6) Global config autosave:
   - Global config pages must auto-save per field; no manual save button.
   - Input/textarea fields save on blur; switches/selects/radios save on change.
   - On blur, if the value is unchanged or empty, do not save.
7) Date/pagination UI:
   - All date pickers and pagination controls must use Chinese UI text.
   - Pagination controls must always be left-aligned.
   - Date pickers must include time selection and be clearable.
8) Node sync requirement:
   - Any API changes for the following must trigger config sync to nodes:
     - Node management
     - Node group / line group management
     - Line management
     - Base package management
     - User package management
9) No direct node sync:
   - The following do not sync directly to nodes:
     - Region management
     - Admin user management
     - API key management
10) Foreign key inserts:
   - Never write `0` into FK fields. Prefer creating the referenced row first and using its ID. If the FK column is nullable, omit it (NULL) and update later. If the FK column is NOT NULL, you must create the referenced row first (no NULL/0 placeholders).
11) DB-driven sync rules:
   - Use `references/db.sql` + current code to decide which data changes must sync to nodes. If DB shows a resource affects node config but code does not create a sync task/job, record it as a gap and fix it (add `BumpConfigVersion` or create the missing task/job).
12) Task vs job:
   - Task = top-level async operation (audit, retry, progress). Use it whenever the action is long-running or must be tracked.
   - Job = per-node execution linked to a task. Use Task+Job when the same task must run on multiple nodes or needs per-node state/ACKs.
   - Never create jobs without a parent task. Create the task first, then attach jobs to `task_id`.
13) Task-id fields on data tables:
   - If a row has `task_id`/`issue_task_id`/`cname_task_id`, create the task first and write the FK immediately. If you must insert the row first, leave the FK NULL (only if nullable) and update after task creation.

## DB-Driven Sync Map (from references/db.sql + current code)

### Must sync to nodes on change
- `config` (global/system config items: `global_config`, `nginx_config`, `site_default_config`, `user_package_config`, `error_page`, etc.)
- `node`, `node_group`, `line`
- `package`, `user_package`
- `site`, `stream`
- `acl`, `cc_rule`, `cc_match`, `cc_filter`
- `cert`

If the code path updates any of the above without creating a sync task (`config_sync`) or a task/job dispatch, treat as a bug and fix.

### Task required but not necessarily a config sync
- Use a task for long-running or retryable operations (examples in code: `issue_cert`, `deploy_cert`, `site_create`, `refresh_url`, `refresh_dir`, `preheat`, backup/cleanup).
- These tasks may later trigger sync if they update config-bearing tables (e.g., cert issued -> `cert` update -> `BumpConfigVersion`).

## Foreign Keys (references/db.sql)

- `login_log.uid` -> `user.id`
- `node.region_id` -> `region.id`
- `node_group.region_id` -> `region.id`
- `line.node_group_id` -> `node_group.id`
- `line.node_id` -> `node.id`
- `line.node_ip_id` -> `node.id`
- `line.task_id` -> `task.id`
- `op_log.uid` -> `user.id`
- `dnsapi.uid` -> `user.id`
- `cert.uid` -> `user.id`
- `cert.dnsapi` -> `dnsapi.id`
- `cert.task_id` -> `task.id`
- `cert.issue_task_id` -> `task.id`
- `acl.uid` -> `user.id`
- `acl.task_id` -> `task.id`
- `cc_rule.uid` -> `user.id`
- `cc_rule.task_id` -> `task.id`
- `cc_match.uid` -> `user.id`
- `cc_match.task_id` -> `task.id`
- `cc_filter.uid` -> `user.id`
- `cc_filter.task_id` -> `task.id`
- `package.region_id` -> `region.id`
- `package.node_group_id` -> `node_group.id`
- `package.backup_node_group` -> `node_group.id`
- `merge_package_group.package_id` -> `package.id`
- `merge_package_group.package_group_id` -> `package_group.id`
- `user_package.uid` -> `user.id`
- `user_package.package` -> `package.id`
- `user_package.region_id` -> `region.id`
- `user_package.node_group_id` -> `node_group.id`
- `user_package.backup_node_group` -> `node_group.id`
- `user_package.task_id` -> `task.id`
- `user_package_up.uid` -> `user.id`
- `user_package_up.package_up` -> `package_up.id`
- `user_package_up.user_package` -> `user_package.id`
- `config.task_id` -> `task.id`
- `site.uid` -> `user.id`
- `site.user_package` -> `user_package.id`
- `site.acl` -> `acl.id`
- `site.task_id` -> `task.id`
- `site.region_id` -> `region.id`
- `site.cname_task_id` -> `task.id`
- `site.backup_node_group` -> `node_group.id`
- `site.node_group_id` -> `node_group.id`
- `stream.uid` -> `user.id`
- `stream.user_package` -> `user_package.id`
- `stream.task_id` -> `task.id`
- `stream.region_id` -> `region.id`
- `stream.cname_task_id` -> `task.id`
- `stream.backup_node_group` -> `node_group.id`
- `stream.node_group_id` -> `node_group.id`
- `site_group.uid` -> `user.id`
- `merge_site_group.site_id` -> `site.id`
- `merge_site_group.group_id` -> `site_group.id`
- `stream_group.uid` -> `user.id`
- `merge_stream_group.stream_id` -> `stream.id`
- `merge_stream_group.group_id` -> `stream_group.id`
- `order.uid` -> `user.id`
- `job.uid` -> `user.id`
- `job.task_id` -> `task.id`
- `api_key.uid` -> `user.id`
- `message_read.msg_id` -> `message.id`
- `message_read.uid` -> `user.id`
- `message_sub.uid` -> `user.id`

## Usage Checklist

- Confirm which role the page is operating under.
- Hide or show the user selector based on role (never show for user role).
- Bind CRUD operations to current user when role is user.
- Use `references/db.sql` as the only schema source unless the user explicitly asks for schema changes.
- Match UI layout and styling to the admin-home demo.
- Ensure APIs for node-related config changes trigger node sync logic.

## References

- `references/db.sql`: authoritative database schema for all frontend features.
