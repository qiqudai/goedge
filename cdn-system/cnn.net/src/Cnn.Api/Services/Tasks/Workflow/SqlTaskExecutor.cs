using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class SqlTaskExecutor : ITaskExecutor
{
    private readonly ISqlSugarClient _db;
    private readonly ITaskHandlerRegistry _registry;

    public SqlTaskExecutor(ISqlSugarClient db, ITaskHandlerRegistry registry)
    {
        _db = db;
        _registry = registry;
    }

    public async Task ExecuteAsync(long taskId, CancellationToken cancellationToken)
    {
        if (taskId <= 0)
        {
            throw new ArgumentOutOfRangeException(nameof(taskId));
        }

        var task = await _db.Queryable<TaskEntity>().Where(t => t.Id == taskId).FirstAsync();
        if (task == null)
        {
            throw new InvalidOperationException($"Task {taskId} was not found.");
        }

        var claimed = await _db.Updateable<TaskEntity>()
            .SetColumns(t => new TaskEntity
            {
                State = "running",
                StartAt = DateTime.Now
            })
            .Where(t => t.Id == taskId && t.Enable == true && (t.State == "waiting" || t.State == "retrying" || t.State == null))
            .ExecuteCommandAsync();
        if (claimed <= 0)
        {
            return;
        }

        var handler = _registry.Resolve(task.Type ?? string.Empty);

        try
        {
            await handler.HandleAsync(taskId, task.Data ?? "{}", cancellationToken);

            await _db.Updateable<TaskEntity>()
                .SetColumns(t => new TaskEntity
                {
                    State = "success",
                    EndAt = DateTime.Now,
                    Progress = "100"
                })
                .Where(t => t.Id == taskId)
                .ExecuteCommandAsync();
        }
        catch (Exception ex)
        {
            await _db.Updateable<TaskEntity>()
                .SetColumns(t => new TaskEntity
                {
                    State = "failed",
                    EndAt = DateTime.Now,
                    Ret = ex.Message
                })
                .Where(t => t.Id == taskId)
                .ExecuteCommandAsync();

            throw;
        }
    }
}
