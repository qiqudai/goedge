using System.Text.Json;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Stats;

public sealed record AccessBucket(
    DateTime Bucket,
    ulong Requests,
    ulong Bytes,
    ulong HitCount,
    ulong OriginBytes,
    ulong Status4xx,
    ulong Status5xx,
    ulong BlockedIps
);

public sealed record AccessTotals(ulong Requests, ulong Bytes, ulong BlockedIps);

public sealed class BucketSeries
{
    public List<string> XAxis { get; } = new();
    public List<ulong> Requests { get; } = new();
    public List<ulong> Bytes { get; } = new();
    public List<ulong> HitCount { get; } = new();
    public List<ulong> OriginBytes { get; } = new();
    public List<ulong> Status4xx { get; } = new();
    public List<ulong> Status5xx { get; } = new();
    public List<ulong> BlockedIps { get; } = new();
}

public interface IAccessStatsService
{
    Task<IReadOnlyList<AccessBucket>> QueryBucketsAsync(StatsRange range, HostFilter hostFilter, CancellationToken cancellationToken);
    Task<AccessTotals> QueryTotalsAsync(DateTime start, DateTime end, HostFilter hostFilter, CancellationToken cancellationToken);
    BucketSeries BuildSeries(StatsRange range, IReadOnlyList<AccessBucket> buckets);
    IReadOnlyList<int> BlockedStatusCodes { get; }
}

public sealed class AccessStatsService : IAccessStatsService
{
    private static readonly IReadOnlyList<int> BlockedCodes = new[] { 403, 418, 429, 451, 410 };
    private const string TimeLayout = "yyyy-MM-dd HH:mm:ss";

    private readonly IConfiguration _configuration;

    public AccessStatsService(IConfiguration configuration)
    {
        _configuration = configuration;
    }

    public IReadOnlyList<int> BlockedStatusCodes => BlockedCodes;

    public async Task<IReadOnlyList<AccessBucket>> QueryBucketsAsync(
        StatsRange range,
        HostFilter hostFilter,
        CancellationToken cancellationToken)
    {
        if (range.Start == DateTime.MinValue || range.End == DateTime.MinValue || range.End < range.Start || range.Bucket <= TimeSpan.Zero)
        {
            return Array.Empty<AccessBucket>();
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return Array.Empty<AccessBucket>();
        }

        var queryWindow = await AccessLogQueryWindowResolver.ResolveAsync(cfg, range.Start, range.End, DateTime.Now, cancellationToken);
        var bucketExpr = BuildBucketExpression(range.Bucket);
        var conditions = new List<string>
        {
            $"ts >= toDateTime('{queryWindow.Start:yyyy-MM-dd HH:mm:ss}') AND ts <= toDateTime('{queryWindow.End:yyyy-MM-dd HH:mm:ss}')"
        };
        var hostClause = hostFilter.BuildHttpCondition();
        if (!string.IsNullOrWhiteSpace(hostClause))
        {
            conditions.Add(hostClause);
        }

        var where = string.Join(" AND ", conditions);
        var blocked = string.Join(",", BlockedCodes);
        var query =
            $"SELECT {bucketExpr} AS bucket," +
            " count() AS requests," +
            " sum(\"bytes\") AS out_bytes," +
            " countIf(upstream_cache_status = 'HIT') AS hit_count," +
            " sumIf(\"bytes\", upstream_cache_status != 'HIT') AS origin_bytes," +
            " countIf(status >= 400 AND status < 500) AS status_4xx," +
            " countIf(status >= 500 AND status < 600) AS status_5xx," +
            $" uniqExactIf(remote_addr, status IN ({blocked})) AS blocked_ips" +
            $" FROM node_access_logs WHERE {where} GROUP BY bucket ORDER BY bucket FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return Array.Empty<AccessBucket>();
        }

        var list = new List<AccessBucket>();
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            try
            {
                using var doc = JsonDocument.Parse(row);
                var root = doc.RootElement;
                var bucketRaw = ReadString(root, "bucket");
                if (string.IsNullOrWhiteSpace(bucketRaw))
                {
                    continue;
                }

                if (!DateTime.TryParseExact(bucketRaw, TimeLayout, null, System.Globalization.DateTimeStyles.None, out var bucket))
                {
                    continue;
                }

                var displayBucket = bucket.Add(queryWindow.BucketDisplayShift);
                list.Add(new AccessBucket(
                    displayBucket,
                    ReadUInt64(root, "requests"),
                    ReadUInt64(root, "out_bytes"),
                    ReadUInt64(root, "hit_count"),
                    ReadUInt64(root, "origin_bytes"),
                    ReadUInt64(root, "status_4xx"),
                    ReadUInt64(root, "status_5xx"),
                    ReadUInt64(root, "blocked_ips")
                ));
            }
            catch
            {
                // ignore invalid rows
            }
        }

        return list;
    }

    public async Task<AccessTotals> QueryTotalsAsync(DateTime start, DateTime end, HostFilter hostFilter, CancellationToken cancellationToken)
    {
        if (start == DateTime.MinValue || end == DateTime.MinValue || end < start)
        {
            return new AccessTotals(0, 0, 0);
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return new AccessTotals(0, 0, 0);
        }

        var queryWindow = await AccessLogQueryWindowResolver.ResolveAsync(cfg, start, end, DateTime.Now, cancellationToken);
        var conditions = new List<string>
        {
            $"ts >= toDateTime('{queryWindow.Start:yyyy-MM-dd HH:mm:ss}') AND ts <= toDateTime('{queryWindow.End:yyyy-MM-dd HH:mm:ss}')"
        };
        var hostClause = hostFilter.BuildHttpCondition();
        if (!string.IsNullOrWhiteSpace(hostClause))
        {
            conditions.Add(hostClause);
        }

        var where = string.Join(" AND ", conditions);
        var blocked = string.Join(",", BlockedCodes);
        var query =
            "SELECT count() AS requests," +
            " sum(\"bytes\") AS out_bytes," +
            $" uniqExactIf(remote_addr, status IN ({blocked})) AS blocked_ips" +
            $" FROM node_access_logs WHERE {where} FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return new AccessTotals(0, 0, 0);
        }

        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            try
            {
                using var doc = JsonDocument.Parse(row);
                var root = doc.RootElement;
                return new AccessTotals(
                    ReadUInt64(root, "requests"),
                    ReadUInt64(root, "out_bytes"),
                    ReadUInt64(root, "blocked_ips")
                );
            }
            catch
            {
                return new AccessTotals(0, 0, 0);
            }
        }

        return new AccessTotals(0, 0, 0);
    }

    public BucketSeries BuildSeries(StatsRange range, IReadOnlyList<AccessBucket> buckets)
    {
        var series = new BucketSeries();
        if (range.Start == DateTime.MinValue || range.End == DateTime.MinValue || range.End < range.Start || range.Bucket <= TimeSpan.Zero)
        {
            return series;
        }

        var map = new Dictionary<DateTime, AccessBucket>();
        foreach (var bucket in buckets)
        {
            map[StatsRangeResolver.AlignToBucket(bucket.Bucket, range.Bucket)] = bucket;
        }

        var start = StatsRangeResolver.AlignToBucket(range.Start, range.Bucket);
        var end = StatsRangeResolver.AlignToBucket(range.End, range.Bucket);
        for (var cur = start; cur <= end; cur = cur.Add(range.Bucket))
        {
            if (!map.TryGetValue(cur, out var bucket))
            {
                bucket = new AccessBucket(cur, 0, 0, 0, 0, 0, 0, 0);
            }

            series.XAxis.Add(cur.ToString(range.LabelFormat));
            series.Requests.Add(bucket.Requests);
            series.Bytes.Add(bucket.Bytes);
            series.HitCount.Add(bucket.HitCount);
            series.OriginBytes.Add(bucket.OriginBytes);
            series.Status4xx.Add(bucket.Status4xx);
            series.Status5xx.Add(bucket.Status5xx);
            series.BlockedIps.Add(bucket.BlockedIps);
        }

        return series;
    }

    private static string BuildBucketExpression(TimeSpan bucket)
    {
        if (bucket >= TimeSpan.FromDays(1))
        {
            return "toStartOfDay(ts)";
        }

        var seconds = (int)bucket.TotalSeconds;
        if (seconds <= 0)
        {
            seconds = 60;
        }

        return $"toStartOfInterval(ts, INTERVAL {seconds} SECOND)";
    }

    private static string ReadString(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return string.Empty;
        }

        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString() ?? string.Empty,
            _ => value.ToString() ?? string.Empty
        };
    }

    private static ulong ReadUInt64(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number)
        {
            if (value.TryGetUInt64(out var parsed))
            {
                return parsed;
            }
            if (value.TryGetInt64(out var signed) && signed > 0)
            {
                return (ulong)signed;
            }
        }

        if (value.ValueKind == JsonValueKind.String && ulong.TryParse(value.GetString(), out var strValue))
        {
            return strValue;
        }

        return 0;
    }
}
