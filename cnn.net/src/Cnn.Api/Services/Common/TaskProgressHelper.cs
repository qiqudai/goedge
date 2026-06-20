using System.Text.Json;
using System.Text.Json.Serialization;

namespace Cnn.Api.Services.Common;

internal static class TaskProgressHelper
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    public static bool HasNode(string? raw, string nodeId)
    {
        if (string.IsNullOrWhiteSpace(nodeId) || string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        if (!TryParseProgress(raw, out var progress))
        {
            return false;
        }

        return progress.TryGetValue(nodeId, out var state) && !string.Equals(state, "fail", StringComparison.OrdinalIgnoreCase);
    }

    public static string UpdateProgress(string? raw, string nodeId, string state)
    {
        if (string.IsNullOrWhiteSpace(nodeId))
        {
            return raw ?? string.Empty;
        }

        var progress = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        if (!string.IsNullOrWhiteSpace(raw))
        {
            TryParseProgress(raw, out progress);
        }

        progress[nodeId] = state;
        return JsonSerializer.Serialize(progress, JsonOptions);
    }

    public static string DeriveState(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return "running";
        }

        if (!TryParseProgress(raw, out var progress) || progress.Count == 0)
        {
            return "running";
        }

        foreach (var item in progress.Values)
        {
            if (!string.Equals(item, "done", StringComparison.OrdinalIgnoreCase))
            {
                return "running";
            }
        }

        return "done";
    }

    public static string AppendLog(string? raw, string nodeId, string state, string message, int attempt)
    {
        var now = DateTime.Now.ToString("yyyy-MM-dd HH:mm:ss");
        var entry = new TaskLogEntry
        {
            Time = now,
            NodeId = nodeId,
            State = state,
            Message = message,
            Attempt = attempt
        };

        var logs = new List<TaskLogEntry>();
        if (!string.IsNullOrWhiteSpace(raw))
        {
            try
            {
                var parsed = JsonSerializer.Deserialize<List<TaskLogEntry>>(raw, JsonOptions);
                if (parsed != null)
                {
                    logs.AddRange(parsed);
                }
            }
            catch
            {
                logs.Add(new TaskLogEntry
                {
                    Time = now,
                    NodeId = string.Empty,
                    State = "legacy",
                    Message = raw,
                    Attempt = 0
                });
            }
        }

        logs.Add(entry);
        return JsonSerializer.Serialize(logs, JsonOptions);
    }

    private static bool TryParseProgress(string raw, out Dictionary<string, string> progress)
    {
        progress = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        try
        {
            var parsed = JsonSerializer.Deserialize<Dictionary<string, string>>(raw, JsonOptions);
            if (parsed == null)
            {
                return false;
            }

            foreach (var pair in parsed)
            {
                if (string.IsNullOrWhiteSpace(pair.Key))
                {
                    continue;
                }
                progress[pair.Key] = pair.Value ?? string.Empty;
            }

            return progress.Count > 0;
        }
        catch
        {
            return false;
        }
    }

    private sealed class TaskLogEntry
    {
        [JsonPropertyName("time")]
        public string? Time { get; set; }

        [JsonPropertyName("node_id")]
        public string? NodeId { get; set; }

        [JsonPropertyName("state")]
        public string? State { get; set; }

        [JsonPropertyName("message")]
        public string? Message { get; set; }

        [JsonPropertyName("attempt")]
        public int Attempt { get; set; }
    }
}
