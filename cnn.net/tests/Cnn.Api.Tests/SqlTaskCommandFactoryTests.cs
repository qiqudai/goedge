using Cnn.Api.Services.Tasks.Workflow;
using SqlSugar;
using SystemTask = System.Threading.Tasks.Task;
using TaskEntity = Cnn.Domain.Entities.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class SqlTaskCommandFactoryTests
{
    [Fact]
    public async SystemTask CreateAsync_ReusesWaitingTaskWithSameDedupeKey()
    {
        using var scope = new RealMySqlTestScope();
        var key = "test:site-delete:waiting:" + Guid.NewGuid().ToString("N");
        await scope.Db.Insertable(new TaskEntity
        {
            Type = AsyncTaskTypes.SiteDelete,
            Depend = key,
            State = "waiting",
            Enable = true,
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();

        var existing = await scope.Db.Queryable<TaskEntity>()
            .Where(t => t.Depend == key)
            .SingleAsync();
        var sut = new SqlTaskCommandFactory(scope.Db);

        var result = await sut.CreateAsync(new CreateTaskCommand
        {
            TaskType = AsyncTaskTypes.SiteDelete,
            DedupeKey = key,
            PayloadJson = """{"resource_id":88}"""
        }, CancellationToken.None);

        Assert.Equal(existing.Id, result.TaskId);
        Assert.Equal("waiting", result.State);
        Assert.Equal(1, await scope.Db.Queryable<TaskEntity>().Where(t => t.Depend == key).CountAsync());
    }

    [Fact]
    public async SystemTask CreateAsync_CreatesNewTaskWhenLatestMatchingTaskFailed()
    {
        using var scope = new RealMySqlTestScope();
        var key = "test:site-delete:failed:" + Guid.NewGuid().ToString("N");
        await scope.Db.Insertable(new TaskEntity
        {
            Type = AsyncTaskTypes.SiteDelete,
            Depend = key,
            State = "failed",
            Enable = true,
            CreateAt = DateTime.Now.AddMinutes(-1),
            Ret = "old failure"
        }).ExecuteCommandAsync();

        var failed = await scope.Db.Queryable<TaskEntity>()
            .Where(t => t.Depend == key)
            .SingleAsync();
        var sut = new SqlTaskCommandFactory(scope.Db);

        var result = await sut.CreateAsync(new CreateTaskCommand
        {
            TaskType = AsyncTaskTypes.SiteDelete,
            DedupeKey = key,
            PayloadJson = """{"resource_id":89}"""
        }, CancellationToken.None);

        Assert.NotEqual(failed.Id, result.TaskId);
        Assert.Equal(2, await scope.Db.Queryable<TaskEntity>().Where(t => t.Depend == key).CountAsync());
        var latest = await scope.Db.Queryable<TaskEntity>()
            .Where(t => t.Depend == key)
            .OrderBy(t => t.Id, OrderByType.Desc)
            .FirstAsync();
        Assert.Equal("waiting", latest!.State);
        Assert.Equal(AsyncTaskTypes.SiteDelete, latest.Type);
        Assert.Equal(key, latest.Depend);
    }
}
