using System.Globalization;
using System.Text.Json;

namespace Cnn.Api.Pages;

internal static class TaskTargetsJsonParser
{
    public static TaskTargetSummary ParseSummary(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return TaskTargetSummary.Empty;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            var root = doc.RootElement;
            return new TaskTargetSummary(
                Total: GetInt(root, "total"),
                Success: GetInt(root, "success"),
                Fail: GetInt(root, "fail"),
                Pending: GetInt(root, "pending"));
        }
        catch
        {
            return TaskTargetSummary.Empty;
        }
    }

    public static List<TaskTargetNodeItem> ParseNodes(string? raw)
    {
        var result = new List<TaskTargetNodeItem>();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return result;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (!doc.RootElement.TryGetProperty("nodes", out var nodes) || nodes.ValueKind != JsonValueKind.Object)
            {
                return result;
            }

            foreach (var entry in nodes.EnumerateObject())
            {
                var obj = entry.Value;
                result.Add(new TaskTargetNodeItem(
                    NodeId: entry.Name,
                    State: GetString(obj, "state") ?? "waiting",
                    Tries: GetInt(obj, "tries"),
                    RetryAt: GetLong(obj, "retry_at"),
                    LastAt: GetLong(obj, "last_at"),
                    Progress: GetNullableInt(obj, "progress"),
                    Ret: GetString(obj, "ret")));
            }
        }
        catch
        {
            return result;
        }

        return result
            .OrderBy(static x => x.NodeId, StringComparer.Ordinal)
            .ToList();
    }

    public static string BuildFailedSummary(IReadOnlyList<TaskTargetNodeItem> nodes)
    {
        if (nodes == null || nodes.Count == 0)
        {
            return string.Empty;
        }

        var failed = nodes
            .Where(static n => string.Equals(n.State, "failed_final", StringComparison.OrdinalIgnoreCase))
            .Select(static n =>
            {
                var reason = string.IsNullOrWhiteSpace(n.Ret) ? "-" : n.Ret!.Trim();
                if (reason.Length > 60)
                {
                    reason = reason[..60] + "...";
                }

                return $"{n.NodeId}:{reason}";
            })
            .Take(4)
            .ToList();

        return failed.Count == 0 ? string.Empty : string.Join(" | ", failed);
    }

    private static string? GetString(JsonElement obj, string name)
    {
        if (!obj.TryGetProperty(name, out var value))
        {
            return null;
        }

        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString(),
            JsonValueKind.Number => value.GetRawText(),
            JsonValueKind.True => "true",
            JsonValueKind.False => "false",
            _ => null
        };
    }

    private static int GetInt(JsonElement obj, string name)
    {
        if (!obj.TryGetProperty(name, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var numeric))
        {
            return numeric;
        }

        if (value.ValueKind == JsonValueKind.String
            && int.TryParse(value.GetString(), NumberStyles.Integer, CultureInfo.InvariantCulture, out var parsed))
        {
            return parsed;
        }

        return 0;
    }

    private static int? GetNullableInt(JsonElement obj, string name)
    {
        if (!obj.TryGetProperty(name, out var value))
        {
            return null;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var numeric))
        {
            return numeric;
        }

        if (value.ValueKind == JsonValueKind.String
            && int.TryParse(value.GetString(), NumberStyles.Integer, CultureInfo.InvariantCulture, out var parsed))
        {
            return parsed;
        }

        return null;
    }

    private static long? GetLong(JsonElement obj, string name)
    {
        if (!obj.TryGetProperty(name, out var value))
        {
            return null;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt64(out var numeric))
        {
            return numeric;
        }

        if (value.ValueKind == JsonValueKind.String
            && long.TryParse(value.GetString(), NumberStyles.Integer, CultureInfo.InvariantCulture, out var parsed))
        {
            return parsed;
        }

        return null;
    }
}

internal sealed record TaskTargetSummary(int Total, int Success, int Fail, int Pending)
{
    public static readonly TaskTargetSummary Empty = new(0, 0, 0, 0);
}

internal sealed record TaskTargetNodeItem(
    string NodeId,
    string State,
    int Tries,
    long? RetryAt,
    long? LastAt,
    int? Progress,
    string? Ret);
