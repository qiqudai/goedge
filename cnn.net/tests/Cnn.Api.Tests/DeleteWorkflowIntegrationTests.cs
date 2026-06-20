using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Common.Contracts;
using Cnn.Domain.Entities;
using SystemTask = System.Threading.Tasks.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class DeleteWorkflowIntegrationTests
{
    private const int ExistingUserId = 3;

    [Fact]
    public async SystemTask SiteDeletionGuard_DeniesEnabledSite()
    {
        using var scope = new RealMySqlTestScope();
        var site = new Site
        {
            Uid = ExistingUserId,
            Domain = "[\"guard-enabled.example.com\"]",
            Enable = true,
            State = "running",
            CreateAt = DateTime.Now
        };
        var siteId = await scope.Db.Insertable(site).ExecuteReturnIdentityAsync();

        var guard = new SiteDeletionGuard(scope.Db);
        var result = await guard.CheckAsync(siteId, CancellationToken.None);

        Assert.False(result.CanDelete);
        Assert.Equal("SITE_MUST_DISABLE_FIRST", result.ErrorCode);
    }

    [Fact]
    public async SystemTask ResourceDeleteRequestService_ReturnsInUseWhenPreviewDenied()
    {
        using var scope = new RealMySqlTestScope();
        var siteId = await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserId,
            Domain = "[\"delete-deny.example.com\"]",
            Enable = true,
            State = "running",
            CreateAt = DateTime.Now
        }).ExecuteReturnIdentityAsync();

        var preview = new DeletionPreviewService(new DeletionGuardRegistry(new IDeletionGuard[]
        {
            new SiteDeletionGuard(scope.Db)
        }));
        var taskFactory = new RecordingTaskCommandFactory();
        var sut = new ResourceDeleteRequestService(preview, taskFactory);

        var result = await sut.RequestDeleteAsync(
            DeleteRequestCommandFactory.Create(ResourceTypes.Site, siteId, 12, 34),
            CancellationToken.None);

        Assert.False(result.Success);
        Assert.Equal(ErrorCodes.InUse, result.ErrorCode);
        Assert.NotNull(result.Data);
        Assert.False(result.Data!.Queued);
        Assert.Equal("SITE_MUST_DISABLE_FIRST", result.Data.ErrorCode);
        Assert.Null(taskFactory.LastCommand);
    }

    [Fact]
    public async SystemTask ResourceDeleteRequestService_QueuesDeleteTaskWhenPreviewAllows()
    {
        using var scope = new RealMySqlTestScope();
        var siteId = await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserId,
            Domain = "[\"delete-allow.example.com\"]",
            Enable = false,
            State = "stop",
            CreateAt = DateTime.Now
        }).ExecuteReturnIdentityAsync();

        var preview = new DeletionPreviewService(new DeletionGuardRegistry(new IDeletionGuard[]
        {
            new SiteDeletionGuard(scope.Db)
        }));
        var taskFactory = new RecordingTaskCommandFactory();
        var sut = new ResourceDeleteRequestService(preview, taskFactory);

        var result = await sut.RequestDeleteAsync(
            DeleteRequestCommandFactory.Create(ResourceTypes.Site, siteId, 56, 78),
            CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotNull(result.Data);
        Assert.True(result.Data!.Queued);
        Assert.NotNull(taskFactory.LastCommand);
        Assert.Equal(AsyncTaskTypes.SiteDelete, taskFactory.LastCommand!.TaskType);
        Assert.Equal(siteId, taskFactory.LastCommand.ResourceId);
        Assert.Equal(56, taskFactory.LastCommand.OwnerUserId);
        Assert.Equal(78, taskFactory.LastCommand.OperatorUserId);
        Assert.Contains($"\"resource_id\":{siteId}", taskFactory.LastCommand.PayloadJson);
    }

    private sealed class RecordingTaskCommandFactory : ITaskCommandFactory
    {
        public CreateTaskCommand? LastCommand { get; private set; }

        public global::System.Threading.Tasks.Task<TaskRequestResult> CreateAsync(CreateTaskCommand command, CancellationToken cancellationToken)
        {
            LastCommand = command;
            return global::System.Threading.Tasks.Task.FromResult(new TaskRequestResult
            {
                TaskId = 4242,
                TaskNo = "task-4242",
                State = "waiting"
            });
        }
    }
}
