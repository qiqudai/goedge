using Cnn.Api.Services.Common;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Domain.Entities;
using SqlSugar;
using SystemTask = System.Threading.Tasks.Task;
using TaskEntity = Cnn.Domain.Entities.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class ConfigVersionServiceTests
{
    [Fact]
    public async SystemTask BumpAsync_CreatesConfigSyncTaskAndInitialVersion()
    {
        using var scope = new RealMySqlTestScope();
        await scope.Db.Deleteable<Config>()
            .Where(c => c.Name == "edge_config_version" && c.Type == "system")
            .ExecuteCommandAsync();
        var sut = new ConfigVersionService(scope.Db);

        var version = await sut.BumpAsync("site", new[] { 88L, 89L }, CancellationToken.None);

        Assert.Equal(1L, version);

        var config = await scope.Db.Queryable<Config>()
            .Where(c => c.Name == "edge_config_version" && c.Type == "system")
            .SingleAsync();
        Assert.NotNull(config);
        Assert.Equal("1", config!.Value);

        var task = await scope.Db.Queryable<TaskEntity>()
            .Where(t => t.Type == AsyncTaskTypes.ConfigSync)
            .OrderBy(t => t.Id, OrderByType.Desc)
            .FirstAsync();
        Assert.Equal(AsyncTaskTypes.ConfigSync, task.Type);
        Assert.Equal("waiting", task.State);
        Assert.Contains("\"Resource\":\"site\"", task.Data);
        Assert.Contains("\"Ids\":[88,89]", task.Data);
    }

    [Fact]
    public async SystemTask BumpAsync_IncrementsExistingVersion()
    {
        using var scope = new RealMySqlTestScope();
        await scope.Db.Deleteable<Config>()
            .Where(c => c.Name == "edge_config_version" && c.Type == "system")
            .ExecuteCommandAsync();
        await scope.Db.Insertable(new Config
        {
            Name = "edge_config_version",
            Type = "system",
            ScopeName = "global",
            ScopeId = 0,
            Value = "41",
            Enable = true,
            CreateAt = DateTime.Now.AddDays(-1),
            UpdateAt = DateTime.Now.AddDays(-1)
        }).ExecuteCommandAsync();

        var sut = new ConfigVersionService(scope.Db);

        var version = await sut.BumpAsync("forward", Array.Empty<long>(), CancellationToken.None);

        Assert.Equal(42L, version);
        var config = await scope.Db.Queryable<Config>()
            .Where(c => c.Name == "edge_config_version" && c.Type == "system")
            .SingleAsync();
        Assert.Equal("42", config!.Value);
    }
}
