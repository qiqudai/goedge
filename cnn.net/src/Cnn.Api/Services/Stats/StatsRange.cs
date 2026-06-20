namespace Cnn.Api.Services.Stats;

public sealed record StatsRange(DateTime Start, DateTime End, TimeSpan Bucket, string LabelFormat);

public static class StatsRangeResolver
{
    private const string StatsTimeLayout = "yyyy-MM-dd HH:mm:ss";

    public static StatsRange Resolve(string? rangeKey, string? startRaw, string? endRaw, DateTime now)
    {
        var key = (rangeKey ?? string.Empty).Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(key))
        {
            key = "30min";
        }

        switch (key)
        {
            case "today":
                {
                    var start = BeginningOfDay(now);
                    return new StatsRange(start, now, TimeSpan.FromHours(1), "HH:00");
                }
            case "yesterday":
                {
                    var start = BeginningOfDay(now).AddDays(-1);
                    var end = EndOfDay(start);
                    return new StatsRange(start, end, TimeSpan.FromHours(1), "HH:00");
                }
            case "7d":
            case "7days":
            case "7-day":
            case "7":
                {
                    var start = BeginningOfDay(now).AddDays(-6);
                    return new StatsRange(start, now, TimeSpan.FromDays(1), "MM-dd");
                }
            case "30d":
            case "30days":
            case "30-day":
            case "30":
                {
                    var start = BeginningOfDay(now).AddDays(-29);
                    return new StatsRange(start, now, TimeSpan.FromDays(1), "MM-dd");
                }
            case "last_month":
                {
                    var start = BeginningOfMonth(now).AddMonths(-1);
                    var end = EndOfMonth(start);
                    return new StatsRange(start, end, TimeSpan.FromDays(1), "MM-dd");
                }
            case "10min":
                return new StatsRange(now.AddMinutes(-10), now, TimeSpan.FromMinutes(1), "HH:mm");
            case "1h":
                return new StatsRange(now.AddHours(-1), now, TimeSpan.FromMinutes(1), "HH:mm");
            case "custom":
                if (TryParseCustomRange(startRaw, endRaw, now, out var customStart, out var customEnd))
                {
                    return BuildCustomRange(customStart, customEnd);
                }
                break;
        }

        return new StatsRange(now.AddMinutes(-30), now, TimeSpan.FromMinutes(1), "HH:mm");
    }

    public static DateTime AlignToBucket(DateTime value, TimeSpan bucket)
    {
        if (bucket >= TimeSpan.FromDays(1))
        {
            return BeginningOfDay(value);
        }

        if (bucket == TimeSpan.Zero)
        {
            return value;
        }

        var ticks = value.Ticks / bucket.Ticks * bucket.Ticks;
        return new DateTime(ticks, value.Kind);
    }

    private static bool TryParseCustomRange(string? startRaw, string? endRaw, DateTime now, out DateTime start, out DateTime end)
    {
        start = DateTime.MinValue;
        end = DateTime.MinValue;
        startRaw = startRaw?.Trim();
        endRaw = endRaw?.Trim();
        if (string.IsNullOrWhiteSpace(startRaw) || string.IsNullOrWhiteSpace(endRaw))
        {
            return false;
        }

        if (!DateTime.TryParseExact(startRaw, StatsTimeLayout, null, System.Globalization.DateTimeStyles.None, out start))
        {
            return false;
        }

        if (!DateTime.TryParseExact(endRaw, StatsTimeLayout, null, System.Globalization.DateTimeStyles.None, out end))
        {
            return false;
        }

        if (end < start)
        {
            return false;
        }

        return true;
    }

    private static StatsRange BuildCustomRange(DateTime start, DateTime end)
    {
        var duration = end - start;
        if (duration <= TimeSpan.FromHours(1))
        {
            return new StatsRange(start, end, TimeSpan.FromMinutes(1), "HH:mm");
        }

        if (duration <= TimeSpan.FromDays(1))
        {
            return new StatsRange(start, end, TimeSpan.FromHours(1), "HH:00");
        }

        return new StatsRange(start, end, TimeSpan.FromDays(1), "MM-dd");
    }

    private static DateTime BeginningOfDay(DateTime value)
    {
        return new DateTime(value.Year, value.Month, value.Day, 0, 0, 0, value.Kind);
    }

    private static DateTime EndOfDay(DateTime value)
    {
        return new DateTime(value.Year, value.Month, value.Day, 23, 59, 59, value.Kind);
    }

    private static DateTime BeginningOfMonth(DateTime value)
    {
        return new DateTime(value.Year, value.Month, 1, 0, 0, 0, value.Kind);
    }

    private static DateTime EndOfMonth(DateTime value)
    {
        var start = BeginningOfMonth(value);
        var next = start.AddMonths(1);
        return next.AddSeconds(-1);
    }
}
