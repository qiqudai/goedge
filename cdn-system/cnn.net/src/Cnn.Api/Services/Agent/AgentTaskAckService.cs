using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Agent;

public interface IAgentTaskAckService
{
    Task HandleAsync(TaskAckMessage message, CancellationToken cancellationToken);
}

public sealed class AgentTaskAckService : IAgentTaskAckService
{
    private const int MaxAttempts = 3;
    private static readonly int[] RetryMinutes = { 5, 10, 20, 30, 60, 60, 60 };
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly ISystemConfigService? _systemConfigService;

    public AgentTaskAckService(ISqlSugarClient db, ISystemConfigService? systemConfigService = null)
    {
        _db = db;
        _systemConfigService = systemConfigService;
    }

    public async Task HandleAsync(TaskAckMessage message, CancellationToken cancellationToken)
    {
        if (message.TaskId <= 0)
        {
            return;
        }

        var task = await _db.Queryable<TaskEntity>().Where(t => t.Id == message.TaskId).FirstAsync();
        if (task == null)
        {
            return;
        }

        var now = DateTime.Now;
        var status = (message.Status ?? string.Empty).Trim().ToLowerInvariant();
        var baseRet = !string.IsNullOrWhiteSpace(message.Error)
            ? message.Error!.Trim()
            : message.Ret?.Trim() ?? string.Empty;
        var ret = BuildRetWithDiagnostics(baseRet, message);

        if (string.Equals(task.Type, "deploy_cert", StringComparison.OrdinalIgnoreCase) &&
            !string.IsNullOrWhiteSpace(task.TargetsJson) &&
            message.NodeId.HasValue &&
            message.NodeId.Value > 0 &&
            status is "success" or "ignored" or "fail")
        {
            var deployPolicy = await DeployCertCompletionPolicy.ResolvePolicyAsync(_systemConfigService, cancellationToken);
            await HandleDeployCertTargetAckAsync(task, message.NodeId.Value, status, ret, now, deployPolicy, cancellationToken);
            return;
        }

        if (status is "success" or "ignored")
        {
            await _db.Updateable<TaskEntity>()
                .SetColumns(t => new TaskEntity { State = "success", Ret = ret, EndAt = now })
                .Where(t => t.Id == task.Id)
                .ExecuteCommandAsync();
            return;
        }

        if (status == "fail")
        {
            var nextErrTimes = (task.ErrTimes ?? 0) + 1;
            if (nextErrTimes >= MaxAttempts)
            {
                await _db.Updateable<TaskEntity>()
                    .SetColumns(t => new TaskEntity { State = "fail", Ret = ret, EndAt = now, ErrTimes = nextErrTimes })
                    .Where(t => t.Id == task.Id)
                    .ExecuteCommandAsync();
            }
            else
            {
                var delay = nextErrTimes - 1 < RetryMinutes.Length ? RetryMinutes[nextErrTimes - 1] : 60;
                await _db.Updateable<TaskEntity>()
                    .SetColumns(t => new TaskEntity
                    {
                        State = "retrying",
                        Ret = ret,
                        RetryAt = now.AddMinutes(delay),
                        ErrTimes = nextErrTimes
                    })
                    .Where(t => t.Id == task.Id)
                    .ExecuteCommandAsync();
            }

            if (string.Equals(task.Type, "issue_cert", StringComparison.OrdinalIgnoreCase))
            {
                await _db.Updateable<Cert>()
                    .SetColumns(c => new Cert { State = "fail", Ret = ret, UpdateAt = now })
                    .Where(c => c.IssueTaskId == task.Id)
                    .ExecuteCommandAsync();
            }
        }
    }

    private async Task HandleDeployCertTargetAckAsync(
        TaskEntity task,
        long nodeId,
        string status,
        string ret,
        DateTime now,
        string deployPolicy,
        CancellationToken cancellationToken)
    {
        var targets = ParseTargets(task.TargetsJson);
        var nodeKey = nodeId.ToString();
        if (!targets.Nodes.TryGetValue(nodeKey, out var target) || target == null)
        {
            target = new TaskTarget();
            targets.Nodes[nodeKey] = target;
        }

        target.LastAt = DateTimeOffset.Now.ToUnixTimeSeconds();
        target.Ret = string.IsNullOrWhiteSpace(ret) ? null : ret;

        if (status is "success" or "ignored")
        {
            target.State = TaskTargetState.Success;
            target.RetryAt = 0;
        }
        else
        {
            var tries = Math.Max(1, target.Tries);
            var maxAttempts = messageMaxAttemptsOrDefault(taskMaxAttempts: MaxAttempts, target: target);
            if (tries >= maxAttempts)
            {
                target.State = TaskTargetState.FailedFinal;
                target.RetryAt = 0;
            }
            else
            {
                var delay = tries - 1 < RetryMinutes.Length ? RetryMinutes[tries - 1] : 60;
                target.State = TaskTargetState.Waiting;
                target.RetryAt = DateTimeOffset.Now.AddMinutes(delay).ToUnixTimeSeconds();
            }
        }

        targets.EnsureCounts();
        var allowPartialFailures = DeployCertCompletionPolicy.IsAllowPartial(deployPolicy);
        var allSettled = targets.Total > 0 && targets.Pending == 0;
        var nextState = "running";
        DateTime? endAt = null;
        DateTime? retryAt = null;

        if (!allowPartialFailures && targets.Fail > 0)
        {
            nextState = "fail";
            endAt = now;
        }
        else if (targets.Total > 0 && targets.Success >= targets.Total)
        {
            nextState = "success";
            endAt = now;
        }
        else if (allowPartialFailures && allSettled)
        {
            if (targets.Success > 0)
            {
                nextState = "success";
                endAt = now;
            }
            else if (targets.Fail > 0)
            {
                nextState = "fail";
                endAt = now;
            }
        }
        else if (targets.Pending > 0 &&
                 targets.Nodes.Values.All(t => string.Equals(t.State, TaskTargetState.Waiting, StringComparison.OrdinalIgnoreCase)))
        {
            nextState = "retrying";
            var minRetryAt = targets.Nodes.Values
                .Where(t => string.Equals(t.State, TaskTargetState.Waiting, StringComparison.OrdinalIgnoreCase) && t.RetryAt > 0)
                .Select(t => DateTimeOffset.FromUnixTimeSeconds(t.RetryAt).LocalDateTime)
                .DefaultIfEmpty()
                .Min();
            if (minRetryAt != default)
            {
                retryAt = minRetryAt;
            }
        }

        var summarizedRet = $"node:{nodeId} {ret}".Trim();
        var finalRet = BuildDeployRetSummary(nextState, targets, allowPartialFailures, deployPolicy, summarizedRet);
        await _db.Updateable<TaskEntity>()
            .SetColumns(t => new TaskEntity
            {
                TargetsJson = targets.Marshal(),
                State = nextState,
                Ret = finalRet,
                EndAt = endAt,
                RetryAt = retryAt
            })
            .Where(t => t.Id == task.Id)
            .ExecuteCommandAsync();
    }

    private static int messageMaxAttemptsOrDefault(int taskMaxAttempts, TaskTarget target)
    {
        return taskMaxAttempts <= 0 ? Math.Max(1, target.Tries) : taskMaxAttempts;
    }

    private static string BuildDeployRetSummary(
        string nextState,
        TaskTargets targets,
        bool allowPartialFailures,
        string deployPolicy,
        string fallback)
    {
        var failedNodes = targets.Nodes
            .Where(static kv => string.Equals(kv.Value?.State, TaskTargetState.FailedFinal, StringComparison.OrdinalIgnoreCase))
            .Select(static kv => kv.Key)
            .OrderBy(static x => x, StringComparer.Ordinal)
            .ToList();
        var failedNodesText = failedNodes.Count == 0 ? "-" : string.Join(",", failedNodes.Take(8));

        if (string.Equals(nextState, "success", StringComparison.OrdinalIgnoreCase) && targets.Fail > 0)
        {
            return $"deploy_cert partial success ({targets.Success}/{targets.Total}), failed={targets.Fail}, failed_nodes={failedNodesText}, policy={deployPolicy}";
        }

        if (string.Equals(nextState, "fail", StringComparison.OrdinalIgnoreCase) && targets.Fail > 0)
        {
            var reason = allowPartialFailures
                ? "deploy_cert failed: all target nodes failed"
                : "deploy_cert failed by strict policy";
            var suffix = string.IsNullOrWhiteSpace(fallback) ? string.Empty : $"; last={fallback}";
            return $"{reason} ({targets.Fail}/{targets.Total}), failed_nodes={failedNodesText}, policy={deployPolicy}{suffix}";
        }

        return fallback;
    }

    private static TaskTargets ParseTargets(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new TaskTargets();
        }

        try
        {
            var targets = JsonSerializer.Deserialize<TaskTargets>(raw, JsonOptions);
            return targets ?? new TaskTargets();
        }
        catch
        {
            return new TaskTargets();
        }
    }

    private static string BuildRetWithDiagnostics(string baseRet, TaskAckMessage message)
    {
        var hasDiagnostics =
            !string.IsNullOrWhiteSpace(message.RetCode) ||
            !string.IsNullOrWhiteSpace(message.ErrorType) ||
            message.IsRetryable.HasValue ||
            message.Attempt.HasValue ||
            message.MaxAttempts.HasValue ||
            message.NextBackoffMs.HasValue;

        if (!hasDiagnostics)
        {
            return baseRet;
        }

        var details = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase)
        {
            ["ret_code"] = message.RetCode,
            ["error_type"] = message.ErrorType,
            ["is_retryable"] = message.IsRetryable,
            ["attempt"] = message.Attempt,
            ["max_attempts"] = message.MaxAttempts,
            ["next_backoff_ms"] = message.NextBackoffMs
        };

        var compact = JsonSerializer.Serialize(details, JsonOptions);
        if (string.IsNullOrWhiteSpace(baseRet))
        {
            return compact;
        }

        return $"{baseRet} | {compact}";
    }
}

public sealed class TaskAckMessage
{
    [JsonPropertyName("kind")]
    public string? Kind { get; set; }

    [JsonPropertyName("msg_id")]
    public string? MsgId { get; set; }

    [JsonPropertyName("node_id")]
    public long? NodeId { get; set; }

    [JsonPropertyName("task_id")]
    public long TaskId { get; set; }

    [JsonPropertyName("task_type")]
    public string? TaskType { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("error")]
    public string? Error { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }

    [JsonPropertyName("ret_code")]
    public string? RetCode { get; set; }

    [JsonPropertyName("error_type")]
    public string? ErrorType { get; set; }

    [JsonPropertyName("is_retryable")]
    public bool? IsRetryable { get; set; }

    [JsonPropertyName("attempt")]
    public int? Attempt { get; set; }

    [JsonPropertyName("max_attempts")]
    public int? MaxAttempts { get; set; }

    [JsonPropertyName("next_backoff_ms")]
    public int? NextBackoffMs { get; set; }

    [JsonPropertyName("applied")]
    public JsonElement? Applied { get; set; }
}
