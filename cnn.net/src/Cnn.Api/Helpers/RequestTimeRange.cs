using System.Globalization;

namespace Cnn.Api.Helpers;

public static class RequestTimeRange
{
    public static (DateTime? Start, DateTime? End) Resolve(HttpRequest request)
    {
        var (startRaw, endRaw) = ResolveRaw(request);
        return (ParseTimeValue(startRaw), ParseTimeValue(endRaw));
    }

    public static (string? Start, string? End) ResolveRaw(HttpRequest request)
    {
        var query = request.Query;
        var start = query["start_time"].ToString();
        var end = query["end_time"].ToString();

        if (string.IsNullOrWhiteSpace(start) || string.IsNullOrWhiteSpace(end))
        {
            if (query.TryGetValue("timeRange[]", out var values) && values.Count >= 2)
            {
                start = values[0];
                end = values[1];
            }
        }

        if (string.IsNullOrWhiteSpace(start) || string.IsNullOrWhiteSpace(end))
        {
            if (query.TryGetValue("timeRange", out var values) && values.Count >= 2)
            {
                start = values[0];
                end = values[1];
            }
        }

        return (start, end);
    }

    public static DateTime? ParseTimeValue(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        if (long.TryParse(raw, out var seconds) && seconds > 0)
        {
            return DateTimeOffset.FromUnixTimeSeconds(seconds).LocalDateTime;
        }

        if (DateTime.TryParse(raw, CultureInfo.InvariantCulture, DateTimeStyles.AssumeLocal, out var parsed))
        {
            return parsed;
        }

        return null;
    }
}
