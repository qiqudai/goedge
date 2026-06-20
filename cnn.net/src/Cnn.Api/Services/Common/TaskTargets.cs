using System.Text.Json;
using System.Text.Json.Serialization;

namespace Cnn.Api.Services.Common;

public sealed class TaskTargets
{
    [JsonPropertyName("nodes")]
    public Dictionary<string, TaskTarget> Nodes { get; set; } = new(StringComparer.Ordinal);

    [JsonPropertyName("total")]
    public int Total { get; set; }

    [JsonPropertyName("success")]
    public int Success { get; set; }

    [JsonPropertyName("fail")]
    public int Fail { get; set; }

    [JsonPropertyName("pending")]
    public int Pending { get; set; }

    public static TaskTargets Create(IReadOnlyList<long> nodeIds)
    {
        var targets = new TaskTargets();
        if (nodeIds == null || nodeIds.Count == 0)
        {
            targets.EnsureCounts();
            return targets;
        }

        foreach (var id in nodeIds)
        {
            if (id <= 0)
            {
                continue;
            }
            targets.Nodes[id.ToString()] = new TaskTarget { State = TaskTargetState.Waiting };
        }

        targets.EnsureCounts();
        return targets;
    }

    public void EnsureCounts()
    {
        var total = Nodes.Count;
        var success = 0;
        var fail = 0;
        var pending = 0;

        foreach (var target in Nodes.Values)
        {
            switch (target.State)
            {
                case TaskTargetState.Success:
                    success++;
                    break;
                case TaskTargetState.FailedFinal:
                    fail++;
                    break;
                default:
                    pending++;
                    break;
            }
        }

        Total = total;
        Success = success;
        Fail = fail;
        Pending = pending;
    }

    public string Marshal()
    {
        EnsureCounts();
        return JsonSerializer.Serialize(this);
    }
}

public sealed class TaskTarget
{
    [JsonPropertyName("state")]
    public string State { get; set; } = TaskTargetState.Waiting;

    [JsonPropertyName("tries")]
    public int Tries { get; set; }

    [JsonPropertyName("retry_at")]
    public long RetryAt { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }

    [JsonPropertyName("progress")]
    public int? Progress { get; set; }

    [JsonPropertyName("message")]
    public string? Message { get; set; }

    [JsonPropertyName("last_at")]
    public long? LastAt { get; set; }
}

public static class TaskTargetState
{
    public const string Waiting = "waiting";
    public const string Running = "running";
    public const string Success = "success";
    public const string FailedFinal = "failed_final";
}
