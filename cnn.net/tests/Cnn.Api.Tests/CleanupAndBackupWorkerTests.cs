using System.Reflection;
using Cnn.Api.Services.Common;
using SqlSugar;
using SystemTask = System.Threading.Tasks.Task;
using TaskEntity = Cnn.Domain.Entities.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class CleanupAndBackupWorkerTests
{
    [Fact]
    public async SystemTask CleanupTasksByDaysAsync_DeletesOnlyUnreferencedOldTasks()
    {
        using var scope = new RealMySqlTestScope();
        var oldTime = DateTime.Now.AddYears(-50);
        var keepDays = 365 * 40;
        var marker = "cleanup-test-" + Guid.NewGuid().ToString("N");

        await scope.Db.Insertable(new TaskEntity
        {
            Type = marker + "-free",
            State = "success",
            Enable = true,
            CreateAt = oldTime
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new TaskEntity
        {
            Type = marker + "-ref",
            State = "success",
            Enable = true,
            CreateAt = oldTime
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new TaskEntity
        {
            Type = marker + "-new",
            State = "success",
            Enable = true,
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();

        var tasks = await scope.Db.Queryable<TaskEntity>().OrderBy(t => t.Id, OrderByType.Asc).ToListAsync();
        var freeTask = tasks.Single(t => t.Type == marker + "-free");
        var refTask = tasks.Single(t => t.Type == marker + "-ref");
        var newTask = tasks.Single(t => t.Type == marker + "-new");
        await scope.Db.Ado.ExecuteCommandAsync(
            "INSERT INTO cert (task_id, create_at, enable) VALUES (@task_id, @create_at, @enable)",
            new { task_id = refTask.Id, create_at = DateTime.Now, enable = 1 });

        var method = typeof(CleanupAndBackupWorker).GetMethod(
            "CleanupTasksByDaysAsync",
            BindingFlags.NonPublic | BindingFlags.Static);

        Assert.NotNull(method);
        var task = (System.Threading.Tasks.Task)method!.Invoke(null, new object?[] { scope.Db, keepDays, CancellationToken.None })!;
        await task;

        var remaining = await scope.Db.Queryable<TaskEntity>()
            .Where(t => t.Id == freeTask.Id || t.Id == refTask.Id || t.Id == newTask.Id)
            .OrderBy(t => t.Id, OrderByType.Asc)
            .ToListAsync();
        Assert.DoesNotContain(remaining, t => t.Id == freeTask.Id);
        Assert.Contains(remaining, t => t.Id == refTask.Id);
        Assert.Contains(remaining, t => t.Id == newTask.Id);
    }
}
