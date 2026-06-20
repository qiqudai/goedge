using System.Reflection;
using Cnn.Api.Services.Tasks.Workflow;
using SqlSugar;
using SystemTask = System.Threading.Tasks.Task;
using TaskEntity = Cnn.Domain.Entities.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class TaskDispatchWorkerTests
{
    [Fact]
    public async SystemTask RepairLegacyConfigSyncTasksAsync_RequeuesKnownFailedLegacyTask()
    {
        using var scope = new RealMySqlTestScope();
        await DisableExistingLegacyTasksAsync(scope.Db);
        var marker = "legacy-config-sync-failed-" + Guid.NewGuid().ToString("N");
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = "config_sync",
            Name = marker,
            State = "failed",
            Enable = true,
            Ret = "No task handler registered for task type 'config_sync'.",
            StartAt = DateTime.Now.AddMinutes(-1),
            EndAt = DateTime.Now.AddMinutes(-1),
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        await InvokeRepairAsync(scope.Db);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal(AsyncTaskTypes.ConfigSync, task.Type, ignoreCase: true);
        Assert.Equal("waiting", task.State);
        Assert.Null(task.Ret);
        Assert.Null(task.StartAt);
        Assert.Null(task.EndAt);
    }

    [Fact]
    public async SystemTask RepairLegacyConfigSyncTasksAsync_OnlyRenamesWaitingLegacyTask()
    {
        using var scope = new RealMySqlTestScope();
        await DisableExistingLegacyTasksAsync(scope.Db);
        var marker = "legacy-config-sync-waiting-" + Guid.NewGuid().ToString("N");
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = "config_sync",
            Name = marker,
            State = "running",
            Enable = true,
            Ret = "still running",
            StartAt = DateTime.Now.AddMinutes(-2),
            EndAt = null,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        await InvokeRepairAsync(scope.Db);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal(AsyncTaskTypes.ConfigSync, task.Type, ignoreCase: true);
        Assert.Equal("running", task.State);
        Assert.Equal("still running", task.Ret);
        Assert.NotNull(task.StartAt);
    }

    [Fact]
    public async SystemTask RepairLegacyConfigSyncTasksAsync_DoesNotTouchUnrelatedFailedTask()
    {
        using var scope = new RealMySqlTestScope();
        await DisableExistingLegacyTasksAsync(scope.Db);
        var marker = "legacy-config-sync-otherfail-" + Guid.NewGuid().ToString("N");
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = "config_sync",
            Name = marker,
            State = "failed",
            Enable = true,
            Ret = "actual business failure",
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        await InvokeRepairAsync(scope.Db);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("config_sync", task.Type);
        Assert.Equal("failed", task.State);
        Assert.Equal("actual business failure", task.Ret);
    }

    private static async SystemTask DisableExistingLegacyTasksAsync(ISqlSugarClient db)
    {
        await db.Updateable<TaskEntity>()
            .SetColumns(t => new TaskEntity { Enable = false })
            .Where(t => t.Type == "config_sync")
            .ExecuteCommandAsync();
    }

    private static async SystemTask InvokeRepairAsync(ISqlSugarClient db)
    {
        var method = typeof(TaskDispatchWorker).GetMethod(
            "RepairLegacyConfigSyncTasksAsync",
            BindingFlags.NonPublic | BindingFlags.Static);

        Assert.NotNull(method);
        var task = (global::System.Threading.Tasks.Task)method!.Invoke(null, new object?[] { db, CancellationToken.None })!;
        await task;
    }
}
