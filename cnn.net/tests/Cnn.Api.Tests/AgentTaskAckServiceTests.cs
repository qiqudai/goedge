using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Cnn.Common.Contracts;
using SqlSugar;
using System.Text.Json;
using SystemTask = System.Threading.Tasks.Task;
using TaskEntity = Cnn.Domain.Entities.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class AgentTaskAckServiceTests
{
    [Fact]
    public async SystemTask HandleAsync_DeployCertTargets_MultiNodeAcks_ReachSuccessAfterAllNodesDone()
    {
        using var scope = new RealMySqlTestScope();
        var targets = TaskTargets.Create(new[] { 11L, 12L }).Marshal();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-cert-targets-success-" + Guid.NewGuid().ToString("N"),
            State = "running",
            Enable = true,
            TargetsJson = targets,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = new AgentTaskAckService(scope.Db);
        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.DeployCert,
            NodeId = 11,
            Status = "success",
            Ret = "{\"cert_id\":201,\"applied_domains\":1}",
            RetCode = "OK",
            Attempt = 1,
            MaxAttempts = 3
        }, CancellationToken.None);

        var first = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("running", first.State);
        Assert.NotNull(first.TargetsJson);
        Assert.Contains("\"11\"", first.TargetsJson);
        Assert.Contains("\"state\":\"success\"", first.TargetsJson);

        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.DeployCert,
            NodeId = 12,
            Status = "success",
            Ret = "{\"cert_id\":201,\"applied_domains\":1}",
            RetCode = "OK",
            Attempt = 1,
            MaxAttempts = 3
        }, CancellationToken.None);

        var done = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("success", done.State);
        Assert.NotNull(done.EndAt);
    }

    [Fact]
    public async SystemTask HandleAsync_DeployCertFail_FirstAttempt_TransitionsToRetryingWithDiagnostics()
    {
        using var scope = new RealMySqlTestScope();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-cert-first-fail-" + Guid.NewGuid().ToString("N"),
            State = "running",
            Enable = true,
            ErrTimes = 0,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = new AgentTaskAckService(scope.Db);
        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.DeployCert,
            Status = "fail",
            Error = "no matching domains on this node",
            RetCode = "NO_MATCHING_DOMAINS",
            ErrorType = "domain_mismatch",
            IsRetryable = false,
            Attempt = 1,
            MaxAttempts = 3
        }, CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("retrying", task.State);
        Assert.Equal(1, task.ErrTimes);
        Assert.NotNull(task.RetryAt);
        Assert.Contains("\"ret_code\":\"NO_MATCHING_DOMAINS\"", task.Ret);
        Assert.Contains("\"error_type\":\"domain_mismatch\"", task.Ret);
        Assert.Contains("\"attempt\":1", task.Ret);
    }

    [Fact]
    public async SystemTask HandleAsync_DeployCertFail_ReachesMaxAttempts_TransitionsToFail()
    {
        using var scope = new RealMySqlTestScope();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-cert-final-fail-" + Guid.NewGuid().ToString("N"),
            State = "retrying",
            Enable = true,
            ErrTimes = 2,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = new AgentTaskAckService(scope.Db);
        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.DeployCert,
            Status = "fail",
            Error = "runtime write failed",
            RetCode = "DEPLOY_RUNTIME_ERROR",
            ErrorType = "runtime",
            IsRetryable = true,
            Attempt = 3,
            MaxAttempts = 3,
            NextBackoffMs = 1200000
        }, CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("fail", task.State);
        Assert.Equal(3, task.ErrTimes);
        Assert.NotNull(task.EndAt);
        Assert.Contains("\"ret_code\":\"DEPLOY_RUNTIME_ERROR\"", task.Ret);
        Assert.Contains("\"is_retryable\":true", task.Ret);
        Assert.Contains("\"max_attempts\":3", task.Ret);
    }

    [Fact]
    public async SystemTask HandleAsync_DeployCertFailThenSuccess_CompletesEndToEnd()
    {
        using var scope = new RealMySqlTestScope();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-cert-retry-success-" + Guid.NewGuid().ToString("N"),
            State = "running",
            Enable = true,
            ErrTimes = 0,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = new AgentTaskAckService(scope.Db);
        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.DeployCert,
            Status = "fail",
            Error = "runtime write failed",
            RetCode = "DEPLOY_RUNTIME_ERROR",
            ErrorType = "runtime",
            IsRetryable = true,
            Attempt = 1,
            MaxAttempts = 3,
            NextBackoffMs = 300000
        }, CancellationToken.None);

        var afterFail = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("retrying", afterFail.State);
        Assert.Equal(1, afterFail.ErrTimes);

        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.DeployCert,
            Status = "success",
            Ret = "{\"cert_id\":101,\"applied_domains\":2}",
            RetCode = "OK",
            Attempt = 2,
            MaxAttempts = 3
        }, CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("success", task.State);
        Assert.NotNull(task.EndAt);
        Assert.Contains("\"ret_code\":\"OK\"", task.Ret);
        Assert.Contains("\"attempt\":2", task.Ret);
    }

    [Fact]
    public async SystemTask HandleAsync_DeployCertTargets_StrictPolicy_PartialPermanentFailureTransitionsToFail()
    {
        using var scope = new RealMySqlTestScope();
        var targets = new TaskTargets
        {
            Nodes = new Dictionary<string, TaskTarget>(StringComparer.Ordinal)
            {
                ["11"] = new() { State = TaskTargetState.Success, Tries = 1 },
                ["12"] = new() { State = TaskTargetState.Running, Tries = 3 }
            }
        }.Marshal();

        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-cert-strict-partial-fail-" + Guid.NewGuid().ToString("N"),
            State = "running",
            Enable = true,
            TargetsJson = targets,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = new AgentTaskAckService(scope.Db, new SystemConfigService(scope.Db));
        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.DeployCert,
            NodeId = 12,
            Status = "fail",
            Error = "write cert failed",
            RetCode = "DEPLOY_RUNTIME_ERROR",
            MaxAttempts = 3
        }, CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("fail", task.State);
        Assert.NotNull(task.EndAt);
        Assert.Contains("strict policy", task.Ret ?? string.Empty, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public async SystemTask HandleAsync_DeployCertTargets_TolerantPolicy_PartialPermanentFailureTransitionsToSuccess()
    {
        using var scope = new RealMySqlTestScope();
        await scope.Db.Insertable(new Cnn.Domain.Entities.Config
        {
            Name = DeployCertCompletionPolicy.ConfigKey,
            Value = DeployCertCompletionPolicy.AllowPartialFailures,
            Type = "system",
            ScopeName = "global",
            ScopeId = 0,
            Enable = true,
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();

        var targets = new TaskTargets
        {
            Nodes = new Dictionary<string, TaskTarget>(StringComparer.Ordinal)
            {
                ["11"] = new() { State = TaskTargetState.Success, Tries = 1 },
                ["12"] = new() { State = TaskTargetState.Running, Tries = 3 }
            }
        }.Marshal();

        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-cert-tolerant-partial-success-" + Guid.NewGuid().ToString("N"),
            State = "running",
            Enable = true,
            TargetsJson = targets,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = new AgentTaskAckService(scope.Db, new SystemConfigService(scope.Db));
        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.DeployCert,
            NodeId = 12,
            Status = "fail",
            Error = "write cert failed",
            RetCode = "DEPLOY_RUNTIME_ERROR",
            MaxAttempts = 3
        }, CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("success", task.State);
        Assert.NotNull(task.EndAt);
        Assert.Contains("partial success", task.Ret ?? string.Empty, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public async SystemTask HandleAsync_ConfigSyncAck_WritesValidatedStreamAuditFields()
    {
        using var scope = new RealMySqlTestScope();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.ConfigSync,
            Name = "config-sync-stream-audit-valid-" + Guid.NewGuid().ToString("N"),
            State = "running",
            Enable = true,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = new AgentTaskAckService(scope.Db);
        var applied = JsonSerializer.Deserialize<JsonElement>(
            """
            {
              "old_version": 100,
              "new_version": 101,
              "stream": {
                "received": 3,
                "applied": 2,
                "skipped": 1,
                "skip_reasons": ["compile_errors"]
              }
            }
            """);

        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.ConfigSync,
            Status = "success",
            Ret = "ok",
            Applied = applied
        }, CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("success", task.State);
        Assert.Contains("\"streams_received\":3", task.Ret ?? string.Empty);
        Assert.Contains("\"streams_applied\":2", task.Ret ?? string.Empty);
        Assert.Contains("\"streams_skipped\":1", task.Ret ?? string.Empty);
        Assert.Contains("\"streams_reason\":\"compile_errors\"", task.Ret ?? string.Empty);
        Assert.Contains("\"streams_audit_valid\":true", task.Ret ?? string.Empty);
    }

    [Fact]
    public async SystemTask HandleAsync_ConfigSyncAck_MarksInvalidWhenAuditFieldsMissingOrOutOfRange()
    {
        using var scope = new RealMySqlTestScope();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.ConfigSync,
            Name = "config-sync-stream-audit-invalid-" + Guid.NewGuid().ToString("N"),
            State = "running",
            Enable = true,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = new AgentTaskAckService(scope.Db);
        var applied = JsonSerializer.Deserialize<JsonElement>(
            """
            {
              "stream": {
                "received": 1,
                "applied": 2
              }
            }
            """);

        await sut.HandleAsync(new TaskAckMessage
        {
            TaskId = taskId,
            TaskType = AgentTaskTypes.ConfigSync,
            Status = "success",
            Ret = "ok",
            Applied = applied
        }, CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("success", task.State);
        Assert.Contains("\"streams_audit_valid\":false", task.Ret ?? string.Empty);
        Assert.Contains("\"streams_reason\":\"invalid_stream_audit\"", task.Ret ?? string.Empty);
        Assert.Contains("missing_skipped", task.Ret ?? string.Empty);
    }
}
