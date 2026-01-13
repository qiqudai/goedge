# cccadmin Dashboard Data Report

## Scope
- /dashboard data sources and ranges (admin + user)
- /website/monitor ranking data used by TOP10 (30 min)
- /website/logs/block current/stats/history real data mapping

## Data Inventory
### Dashboard (User)
- User card: `user` table (name/id/cert_verified) + `login_log` last success IP/time; role from auth context.
- Network overview: ClickHouse `node_access_logs` totals within `overview_range`.
  - bandwidth_peak = max per-bucket bytes -> Mbps.
  - requests = total request count.
  - traffic = sum(bytes).
  - blocked_ips = uniq blocked IPs (status 403/418/429/451/410).
- Monitoring trend: ClickHouse buckets within `chart_range`.
  - bandwidth/traffic/requests/blocked per bucket.
- TOP10 (30 min): ClickHouse rankings for domain/url/ip/country (same logic as `/website/monitor`).
- Announcements: `message` rows with `type=announcement` and `is_show=1` (latest 5).
- Package traffic: `user_package` (latest active) + ClickHouse totals across package sites (from package start to now).
- Resource counts: unique domains from `site.domain`, plus forward/cert/user_package counts scoped to user.

### Dashboard (Admin)
- Ops summary (`ops_range`):
  - users = `user.type != 1` created in range.
  - packages = paid orders with `type in (purchase, renew)` in range.
  - recharge = sum paid orders with `type=recharge` in range.
- System status: master=true, elastic=ClickHouse enabled, agent=all enabled nodes online (90s TTL), counts included.
- License: total/current nodes derived from node counts (no license server).

### Website Monitor
- `/stats/ranking` uses ClickHouse rankings; admin sees all, user limited to own sites via host filter.

### Block Logs
- `/logs/block/current`, `/logs/block/stats`, `/logs/block/history` use ClickHouse `node_access_logs` with blocked status codes.
- Filters: `type=ip/site_id/time_range`; history supports `start_time/end_time`.
- Location resolves via ip2region xdb.

## Range Parameters
- `/dashboard`: `overview_range`, `chart_range`, `ops_range` (today/yesterday/7d/30d/last_month).
- `/stats/ranking`: `time_range` (10min/30min/1h/custom).
- Block logs: `time_range` for current/stats; `start_time/end_time` for history (default 7d).

## Tests
- `go test ./...` (pass)
- `npm run build` (pass)
- `node scripts/sync-wwwroot.cjs` (pass)

## Assumptions / Notes
- Block logs are inferred from access log status codes; filter label uses `HTTP_<status>`, release_time is `PERMANENT`, and history `is_manual=false` without WAF event data.
- Package usage uses ClickHouse totals from package start time; if start is empty/invalid, defaults to last 24 hours.
- Agent status considers all enabled nodes online within 90 seconds; adjust TTL if heartbeat interval differs.
- License data has no source; dashboard uses node counts and `expire_at` set to `-`.
