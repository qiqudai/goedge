using System.Text.Json;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class SqlTaskCommandFactory : ITaskCommandFactory
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;

    public SqlTaskCommandFactory(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<TaskRequestResult> CreateAsync(CreateTaskCommand command, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(command.TaskType))
        {
            throw new ArgumentException("TaskType is required.", nameof(command));
        }

        if (!string.IsNullOrWhiteSpace(command.DedupeKey))
        {
            var existing = await _db.Queryable<TaskEntity>()
                .Where(t => t.Depend == command.DedupeKey && t.Type == command.TaskType && t.Enable == true)
                .Where(t => t.State == "waiting" || t.State == "running" || t.State == "retrying")
                .OrderBy(t => t.Id, OrderByType.Desc)
                .FirstAsync();

            if (existing != null)
            {
                return new TaskRequestResult
                {
                    TaskId = existing.Id,
                    TaskNo = BuildTaskNo(existing.Id),
                    State = existing.State ?? "waiting"
                };
            }
        }

        var meta = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        if (command.OwnerUserId.HasValue && command.OwnerUserId.Value > 0)
        {
            meta["owner_user_id"] = command.OwnerUserId.Value;
        }

        if (command.OperatorUserId.HasValue && command.OperatorUserId.Value > 0)
        {
            meta["operator_user_id"] = command.OperatorUserId.Value;
        }

        if (!string.IsNullOrWhiteSpace(command.ResourceType))
        {
            meta["resource_type"] = command.ResourceType;
        }

        if (command.ResourceId.HasValue && command.ResourceId.Value > 0)
        {
            meta["resource_id"] = command.ResourceId.Value;
        }

        var entity = new TaskEntity
        {
            Name = command.TaskType,
            Type = command.TaskType,
            State = "waiting",
            Enable = true,
            CreateAt = DateTime.Now,
            RetryAt = command.ScheduledAt,
            Data = command.PayloadJson,
            Res = JsonSerializer.Serialize(meta, JsonOptions),
            Depend = command.DedupeKey,
            Progress = "0"
        };

        var taskId = await _db.Insertable(entity).ExecuteReturnBigIdentityAsync();
        if (taskId <= 0)
        {
            throw new InvalidOperationException("Failed to create task.");
        }

        return new TaskRequestResult
        {
            TaskId = taskId,
            TaskNo = BuildTaskNo(taskId),
            State = "waiting"
        };
    }

    private static string BuildTaskNo(long taskId)
    {
        return $"task-{taskId}";
    }
}
