using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Cnn.Common.Contracts;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class TaskDispatchWorker : BackgroundService
{
    private readonly IServiceScopeFactory _scopeFactory;
    private readonly ILogger<TaskDispatchWorker> _logger;

    public TaskDispatchWorker(IServiceScopeFactory scopeFactory, ILogger<TaskDispatchWorker> logger)
    {
        _scopeFactory = scopeFactory;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(2));
        while (await timer.WaitForNextTickAsync(stoppingToken))
        {
            try
            {
                await ProcessAsync(stoppingToken);
            }
            catch (OperationCanceledException)
            {
                // ignored
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "Task dispatch worker tick failed");
            }
        }
    }

    private async Task ProcessAsync(CancellationToken cancellationToken)
    {
        while (!cancellationToken.IsCancellationRequested)
        {
            using var scope = _scopeFactory.CreateScope();
            var db = scope.ServiceProvider.GetRequiredService<ISqlSugarClient>();
            var executor = scope.ServiceProvider.GetRequiredService<ITaskExecutor>();
            var registry = scope.ServiceProvider.GetRequiredService<ITaskHandlerRegistry>();

            await RepairLegacyConfigSyncTasksAsync(db, cancellationToken);

            var task = await db.Queryable<TaskEntity>()
                .Where(t => t.Enable == true)
                .Where(t => t.State == "waiting" || t.State == "retrying")
                .Where(t => t.Type != AgentTaskTypes.IssueCert)
                .Where(t => t.Type != AgentTaskTypes.DeployCert)
                .Where(t => t.Type != AgentTaskTypes.RefreshUrl)
                .Where(t => t.Type != AgentTaskTypes.RefreshDir)
                .Where(t => t.Type != AgentTaskTypes.ClearCache)
                .Where(t => t.Type != AgentTaskTypes.Preheat)
                .Where(t => t.Type != AgentTaskTypes.ConfigSync)
                .Where(t => t.Type != AgentTaskTypes.AgentUpgrade)
                .Where(t => t.Type != AgentTaskTypes.DebugSwitch)
                .Where(t => t.Type != AgentTaskTypes.DebugLogSwitch)
                .Where(t => t.Type != AgentTaskTypes.ManualDebugLog)
                .Where(t => t.Type != AgentTaskTypes.DebugLogWrite)
                .Where(t => t.Type != "sync_package")
                .Where(t => t.Type != "package_sync")
                .OrderBy(t => t.Id, OrderByType.Asc)
                .FirstAsync();

            if (task == null)
            {
                return;
            }

            if (!registry.TryResolve(task.Type ?? string.Empty, out _))
            {
                await db.Updateable<TaskEntity>()
                    .SetColumns(t => new TaskEntity
                    {
                        State = "failed",
                        EndAt = DateTime.Now,
                        Ret = $"No task handler registered for task type '{task.Type}'."
                    })
                    .Where(t => t.Id == task.Id)
                    .ExecuteCommandAsync();
                continue;
            }

            try
            {
                await executor.ExecuteAsync(task.Id, cancellationToken);
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "Task dispatch worker failed for task {TaskId}", task.Id);
            }
        }
    }

    private static async Task RepairLegacyConfigSyncTasksAsync(ISqlSugarClient db, CancellationToken cancellationToken)
    {
        const string legacyType = "config_sync";
        var legacyTasks = await db.Queryable<TaskEntity>()
            .Where(t => t.Enable == true && t.Type == legacyType)
            .Where(t => t.State == "waiting" || t.State == "retrying" || t.State == "running" ||
                        (t.State == "failed" && t.Ret != null && t.Ret.Contains("No task handler registered")))
            .OrderBy(t => t.Id, OrderByType.Asc)
            .Take(20)
            .ToListAsync();

        if (legacyTasks.Count == 0)
        {
            return;
        }

        foreach (var legacyTask in legacyTasks)
        {
            var shouldRequeue = string.Equals(legacyTask.State, "failed", StringComparison.OrdinalIgnoreCase);
            if (shouldRequeue)
            {
                await db.Updateable<TaskEntity>()
                    .SetColumns(t => new TaskEntity
                    {
                        Type = AsyncTaskTypes.ConfigSync,
                        State = "waiting",
                        Ret = null,
                        StartAt = null,
                        EndAt = null
                    })
                    .Where(t => t.Id == legacyTask.Id)
                    .ExecuteCommandAsync();
                continue;
            }

            await db.Updateable<TaskEntity>()
                .SetColumns(t => new TaskEntity
                {
                    Type = AsyncTaskTypes.ConfigSync
                })
                .Where(t => t.Id == legacyTask.Id)
                .ExecuteCommandAsync();
        }
    }
}
