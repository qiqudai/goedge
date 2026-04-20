using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Agent;

public interface IAgentTaskService
{
    Task<ServiceResult<AgentTaskListResult>> ListAsync(string? nodeId, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> FinishAsync(long taskId, string? nodeId, AgentTaskFinishRequest request, CancellationToken cancellationToken);
}

public sealed class AgentTaskService : IAgentTaskService
{
    private const int MaxRetries = 3;
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly INodeRateLimitService _nodeRateLimit;
    private readonly ISystemConfigService _systemConfigService;
    private readonly IMessageLocalizer _localizer;

    public AgentTaskService(
        ISqlSugarClient db,
        INodeRateLimitService nodeRateLimit,
        ISystemConfigService systemConfigService,
        IMessageLocalizer localizer)
    {
        _db = db;
        _nodeRateLimit = nodeRateLimit;
        _systemConfigService = systemConfigService;
        _localizer = localizer;
    }

    public async Task<ServiceResult<AgentTaskListResult>> ListAsync(string? nodeId, CancellationToken cancellationToken)
    {
        var now = DateTime.Now;
        var tasks = await _db.Queryable<TaskEntity>()
            .Where(t => t.Enable == true && (t.State == "waiting" || t.State == "running" || t.State == "retrying"))
            .Where(t => t.RetryAt == null || t.RetryAt <= now)
            .OrderBy(t => t.Id, OrderByType.Asc)
            .Take(100)
            .ToListAsync();

        var filtered = new List<TaskEntity>();
        foreach (var task in tasks)
        {
            if (task.RetryAt.HasValue && task.RetryAt.Value > now)
            {
                continue;
            }

            if (string.Equals(task.Type, "issue_cert", StringComparison.OrdinalIgnoreCase) && !string.IsNullOrWhiteSpace(nodeId))
            {
                var target = ParseIssueTaskTarget(task.Res);
                if (!string.IsNullOrWhiteSpace(target) && !string.Equals(target, nodeId, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
            }

            if (string.IsNullOrWhiteSpace(nodeId) || !TaskProgressHelper.HasNode(task.Progress, nodeId))
            {
                filtered.Add(task);
            }
        }

        if (filtered.Count > 0 && !string.IsNullOrWhiteSpace(nodeId))
        {
            foreach (var task in filtered)
            {
                var startAt = task.StartAt ?? now;
                var progress = TaskProgressHelper.UpdateProgress(task.Progress, nodeId, "running");
                await _db.Updateable<TaskEntity>()
                    .SetColumns(t => new TaskEntity
                    {
                        State = "running",
                        StartAt = startAt,
                        Progress = progress
                    })
                    .Where(t => t.Id == task.Id)
                    .ExecuteCommandAsync();

                if (string.Equals(task.Type, "issue_cert", StringComparison.OrdinalIgnoreCase))
                {
                    await _db.Updateable<Cert>()
                        .SetColumns(c => new Cert { State = "issuing", UpdateAt = now })
                        .Where(c => c.IssueTaskId == task.Id && c.State == "waiting")
                        .ExecuteCommandAsync();
                }
            }
        }

        var list = filtered.Select(BuildAgentTaskDto).ToList();
        return ServiceResult<AgentTaskListResult>.Ok(new AgentTaskListResult(list));
    }

    public async Task<ServiceResult<bool>> FinishAsync(long taskId, string? nodeId, AgentTaskFinishRequest request, CancellationToken cancellationToken)
    {
        if (taskId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "task_id_required");
        }

        var task = await _db.Queryable<TaskEntity>().Where(t => t.Id == taskId).FirstAsync();
        if (task == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "task_not_found");
        }

        var state = request?.State?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(state))
        {
            state = "done";
        }

        var retMessage = request?.Ret?.Trim() ?? string.Empty;
        if (string.Equals(task.Type, "issue_cert", StringComparison.OrdinalIgnoreCase) &&
            retMessage.Contains("429", StringComparison.OrdinalIgnoreCase) &&
            long.TryParse(nodeId, out var nodeIdValue))
        {
            _nodeRateLimit.MarkLimited(nodeIdValue, TimeSpan.FromMinutes(15));
        }

        var now = DateTime.Now;
        var progress = TaskProgressHelper.UpdateProgress(task.Progress, nodeId ?? string.Empty, state);
        var retLog = TaskProgressHelper.AppendLog(task.Ret, nodeId ?? string.Empty, state, retMessage, task.ErrTimes ?? 0);

        var nextState = task.State ?? string.Empty;
        var nextErrTimes = task.ErrTimes ?? 0;
        DateTime? retryAt = task.RetryAt;
        DateTime? endAt = task.EndAt;

        if (state == "fail")
        {
            nextErrTimes += 1;
            retLog = TaskProgressHelper.AppendLog(retLog, nodeId ?? string.Empty, "retry", $"retry {nextErrTimes}/{MaxRetries}", nextErrTimes);
            if (nextErrTimes >= MaxRetries)
            {
                nextState = "fail";
                endAt = now;
            }
            else
            {
                nextState = "waiting";
                retryAt = now.AddSeconds(nextErrTimes * 30);
            }
        }
        else
        {
            nextState = TaskProgressHelper.DeriveState(progress);
            if (string.Equals(nextState, "done", StringComparison.OrdinalIgnoreCase))
            {
                endAt = now;
            }
        }

        await _db.Updateable<TaskEntity>()
            .SetColumns(t => new TaskEntity
            {
                Ret = retLog,
                Progress = progress,
                State = nextState,
                ErrTimes = nextErrTimes,
                RetryAt = retryAt,
                EndAt = endAt
            })
            .Where(t => t.Id == task.Id)
            .ExecuteCommandAsync();

        if (!string.Equals(task.State, nextState, StringComparison.OrdinalIgnoreCase) &&
            (string.Equals(nextState, "done", StringComparison.OrdinalIgnoreCase) ||
             string.Equals(nextState, "fail", StringComparison.OrdinalIgnoreCase)))
        {
            await NotifyTaskCompletionAsync(task, nextState, retMessage, cancellationToken);
        }

        return ServiceResult<bool>.Ok(true);
    }

    private AgentTaskDto BuildAgentTaskDto(TaskEntity task)
    {
        return new AgentTaskDto
        {
            Id = task.Id,
            Pid = task.Pid,
            Pry = task.Pry,
            Name = task.Name,
            Type = task.Type,
            Res = task.Res,
            Data = task.Data,
            TargetsJson = task.TargetsJson,
            Depend = task.Depend,
            CreateAt = task.CreateAt,
            StartAt = task.StartAt,
            EndAt = task.EndAt,
            Ret = task.Ret,
            Enable = task.Enable,
            State = task.State,
            ErrTimes = task.ErrTimes,
            RetryAt = task.RetryAt,
            Progress = task.Progress
        };
    }

    private async Task NotifyTaskCompletionAsync(TaskEntity task, string state, string ret, CancellationToken cancellationToken)
    {
        var userId = ParseTaskUserId(task.Res);
        if (userId <= 0)
        {
            return;
        }

        var language = _localizer.DefaultLanguage;
        var title = BuildTaskTitle(task.Type, state, language);
        var content = BuildTaskContent(task.Type, state, task.Data, ret, language);

        await NotificationHelper.CreateUserMessageAsync(
            _db,
            _systemConfigService,
            userId,
            task.Type ?? string.Empty,
            title,
            content,
            0,
            0,
            cancellationToken);
    }

    private string BuildTaskTitle(string? taskType, string state, string language)
    {
        var label = taskType ?? string.Empty;
        switch (taskType)
        {
            case "refresh_url":
                label = _localizer.Translate("task.refresh_url", language);
                break;
            case "refresh_dir":
                label = _localizer.Translate("task.refresh_dir", language);
                break;
            case "preheat":
                label = _localizer.Translate("task.preheat", language);
                break;
        }

        var suffix = string.Equals(state, "fail", StringComparison.OrdinalIgnoreCase)
            ? _localizer.Translate("task.failed_suffix", language)
            : _localizer.Translate("task.done_suffix", language);
        return label + suffix;
    }

    private string BuildTaskContent(string? taskType, string state, string? data, string ret, string language)
    {
        var result = string.Equals(state, "fail", StringComparison.OrdinalIgnoreCase)
            ? _localizer.Translate("task.exec_failed", language)
            : _localizer.Translate("task.exec_success", language);

        if (!string.IsNullOrWhiteSpace(ret))
        {
            result += _localizer.Translate("task.reason_prefix", language) + ret;
        }

        if (string.IsNullOrWhiteSpace(data))
        {
            return result;
        }

        return result + _localizer.Translate("task.url_prefix", language) + data;
    }

    private static long ParseTaskUserId(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return 0;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.TryGetProperty("user_id", out var value))
            {
                if (value.ValueKind == JsonValueKind.Number && value.TryGetInt64(out var id))
                {
                    return id;
                }
                if (value.ValueKind == JsonValueKind.String && long.TryParse(value.GetString(), out var parsed))
                {
                    return parsed;
                }
            }
        }
        catch
        {
            return 0;
        }

        return 0;
    }

    private static string ParseIssueTaskTarget(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return string.Empty;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.TryGetProperty("target_node_id", out var value))
            {
                if (value.ValueKind == JsonValueKind.Number && value.TryGetInt64(out var id) && id > 0)
                {
                    return id.ToString();
                }
                if (value.ValueKind == JsonValueKind.String)
                {
                    return value.GetString() ?? string.Empty;
                }
            }
        }
        catch
        {
            return string.Empty;
        }

        return string.Empty;
    }
}
