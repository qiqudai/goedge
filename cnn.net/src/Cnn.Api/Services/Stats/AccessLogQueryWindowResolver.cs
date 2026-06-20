using System.Globalization;
using System.Text.Json;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Stats;

public sealed record AccessLogQueryWindow(DateTime Start, DateTime End, TimeSpan BucketDisplayShift)
{
    public static AccessLogQueryWindow Unshifted(DateTime start, DateTime end)
    {
        return new AccessLogQueryWindow(start, end, TimeSpan.Zero);
    }
}

public static class AccessLogQueryWindowResolver
{
    private static readonly TimeSpan MaxRealtimeRange = TimeSpan.FromHours(2);
    private static readonly TimeSpan RealtimeEndTolerance = TimeSpan.FromMinutes(15);
    private static readonly TimeSpan MinUsefulSkew = TimeSpan.FromSeconds(90);
    private static readonly TimeSpan MaxSupportedSkew = TimeSpan.FromHours(14);
    private static readonly TimeSpan CacheTtl = TimeSpan.FromMinutes(1);
    private static readonly SemaphoreSlim CacheLock = new(1, 1);

    private static DateTime _cachedAtUtc = DateTime.MinValue;
    private static TimeSpan _cachedSkew = TimeSpan.Zero;

    public static async Task<AccessLogQueryWindow> ResolveAsync(
        IConfiguration configuration,
        DateTime start,
        DateTime end,
        CancellationToken cancellationToken)
    {
        return await ResolveAsync(
            ClickHouseHttpHelper.ResolveConfig(configuration),
            start,
            end,
            DateTime.Now,
            cancellationToken);
    }

    public static async Task<AccessLogQueryWindow> ResolveAsync(
        ClickHouseHttpConfig? config,
        DateTime start,
        DateTime end,
        DateTime now,
        CancellationToken cancellationToken)
    {
        if (config == null || !ShouldCheckSkew(start, end, now))
        {
            return AccessLogQueryWindow.Unshifted(start, end);
        }

        var skew = await GetCachedSkewAsync(config, cancellationToken);
        return AdjustForSkew(start, end, now, skew);
    }

    public static AccessLogQueryWindow AdjustForSkew(DateTime start, DateTime end, DateTime now, TimeSpan skew)
    {
        if (!ShouldCheckSkew(start, end, now) || skew == TimeSpan.Zero)
        {
            return AccessLogQueryWindow.Unshifted(start, end);
        }

        return new AccessLogQueryWindow(start.Add(skew), end.Add(skew), -skew);
    }

    public static TimeSpan NormalizeSkew(long maxLogUnixSeconds, long clickHouseNowUnixSeconds)
    {
        if (maxLogUnixSeconds <= 0 || clickHouseNowUnixSeconds <= 0)
        {
            return TimeSpan.Zero;
        }

        var skew = TimeSpan.FromSeconds(maxLogUnixSeconds - clickHouseNowUnixSeconds);
        var abs = skew.Duration();
        if (abs < MinUsefulSkew || abs > MaxSupportedSkew)
        {
            return TimeSpan.Zero;
        }

        return skew;
    }

    public static void ResetCacheForTests()
    {
        _cachedAtUtc = DateTime.MinValue;
        _cachedSkew = TimeSpan.Zero;
    }

    private static async Task<TimeSpan> GetCachedSkewAsync(ClickHouseHttpConfig config, CancellationToken cancellationToken)
    {
        var nowUtc = DateTime.UtcNow;
        if (nowUtc - _cachedAtUtc < CacheTtl)
        {
            return _cachedSkew;
        }

        await CacheLock.WaitAsync(cancellationToken);
        try
        {
            nowUtc = DateTime.UtcNow;
            if (nowUtc - _cachedAtUtc < CacheTtl)
            {
                return _cachedSkew;
            }

            _cachedSkew = await DetectSkewAsync(config, cancellationToken);
            _cachedAtUtc = nowUtc;
            return _cachedSkew;
        }
        finally
        {
            CacheLock.Release();
        }
    }

    private static async Task<TimeSpan> DetectSkewAsync(ClickHouseHttpConfig config, CancellationToken cancellationToken)
    {
        const string query = "SELECT toUnixTimestamp(max(ts)) AS max_ts, toUnixTimestamp(now()) AS now_ts FROM node_access_logs FORMAT JSONEachRow";

        try
        {
            var rows = await ClickHouseHttpHelper.QueryRowsAsync(config, query, cancellationToken);
            if (rows == null || rows.Length == 0 || string.IsNullOrWhiteSpace(rows[0]))
            {
                return TimeSpan.Zero;
            }

            using var doc = JsonDocument.Parse(rows[0]);
            var root = doc.RootElement;
            return NormalizeSkew(ReadInt64(root, "max_ts"), ReadInt64(root, "now_ts"));
        }
        catch
        {
            return TimeSpan.Zero;
        }
    }

    private static bool ShouldCheckSkew(DateTime start, DateTime end, DateTime now)
    {
        if (start == DateTime.MinValue || end == DateTime.MinValue || end < start)
        {
            return false;
        }

        if (end - start > MaxRealtimeRange)
        {
            return false;
        }

        return (end - now).Duration() <= RealtimeEndTolerance;
    }

    private static long ReadInt64(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt64(out var number))
        {
            return number;
        }

        if (value.ValueKind == JsonValueKind.String && long.TryParse(value.GetString(), NumberStyles.Integer, CultureInfo.InvariantCulture, out var parsed))
        {
            return parsed;
        }

        return 0;
    }
}
