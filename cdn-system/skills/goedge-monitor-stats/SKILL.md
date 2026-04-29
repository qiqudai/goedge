---
name: goedge-monitor-stats
description: Use when maintaining GoEdge website monitor/statistics pages, ClickHouse node_access_logs queries, stats/ranking time ranges, access-log ingestion, or chart buckets; preserves 10/30 minute realtime data and prevents timezone/upload-lag regressions.
---

# GoEdge Monitor Stats

Use this skill for `web/admin/src/views/website/Statistics.vue`, `api/controllers/stat_controller.go`, `api/services/*stats*`, `ranking_service.go`, `ck_time_skew.go`, `cnn.net/src/Cnn.Api/Services/Stats/*`, or ClickHouse `node_access_logs` changes.

## Non-Negotiables

- Website monitor tabs must not return empty data just because `node_access_logs.max(ts)` lags behind API `time.Now()`.
- Keep realtime ranges (`10min`, default `30min`, `1h`) resilient to agent upload delay and node clock skew.
- Do not remove `accessLogQueryWindow` / `adjustAccessLogQueryRangeForSkew` behavior unless replacing it with tests that prove 10/30 minute windows still show latest available data.
- In `cnn.net`, do not remove `AccessLogQueryWindowResolver` behavior unless replacing it with tests that prove short realtime windows still query the latest available `node_access_logs` data.
- For chart bucket queries, if the query range is shifted to match ClickHouse latest logs, shift returned bucket timestamps back by the inverse amount before `BuildBucketSeries`.
- For ranking queries, shift the query window but do not alter row labels because rankings are not bucket-aligned.
- Do not apply skew shifting to historical or long ranges; custom/yesterday/day ranges should stay anchored to the requested time.

## Validation Checklist

Run these before handing off monitor/stat changes:

```bash
cd api
go test ./services -run 'Test.*(Skew|Bucket|Stats|Ranking|AccessLog)'
go test ./services
```

Run `go test ./controllers` too when the controllers package is buildable in the current branch; do not ignore monitor-related controller failures.

For `cnn.net` stats changes, run the .NET tests that pin the same guardrail:

```bash
dotnet test cnn.net/tests/Cnn.Api.Tests/Cnn.Api.Tests.csproj --filter AccessLogQueryWindowResolverTests
```

If ClickHouse is available, also check:

```bash
curl -sS -u default:123 'http://127.0.0.1:8123/?database=cdn_logs' \
  --data-binary "SELECT now() AS ck_now, max(ts) AS max_ts, dateDiff('second', max(ts), now()) AS lag_seconds, count() AS rows FROM node_access_logs FORMAT JSONEachRow"
```

Expected behavior: if latest logs are behind by minutes, `/stats/basic`, `/stats/quality`, `/stats/origin`, and `/stats/ranking?time_range=10min|30min` should still use the latest available log window instead of returning empty data.
