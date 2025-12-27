# Purge / Preheat

## Scope
- Submit purge (URL/dir) and preheat tasks.
- Track limits and remaining quota; allow resubmission.

## UI
- Task submission form with URL list and type.
- Task record list with status and resubmit action.

## API
- Create task, list tasks, query usage/limits.
- Validate domain existence and prevent wildcard purge unless supported.

## Node/Sync
- Sync purge/preheat tasks to nodes for execution.
