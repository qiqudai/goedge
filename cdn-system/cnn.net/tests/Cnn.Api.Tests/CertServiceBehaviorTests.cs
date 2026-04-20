using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using Microsoft.Extensions.Configuration;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class CertServiceBehaviorTests
{
    private const int ExistingUserA = 2;
    private const int ExistingUserB = 3;

    [Fact]
    public async global::System.Threading.Tasks.Task BatchProgressAsync_AdminRequest_ReturnsAggregatedStates()
    {
        using var scope = new RealMySqlTestScope();
        var marker = "cert-batch-admin-" + Guid.NewGuid().ToString("N");
        var now = DateTime.Now;
        var batchId = (int)(DateTimeOffset.UtcNow.ToUnixTimeSeconds() % int.MaxValue);

        var failCertId = await scope.Db.Insertable(new Cert
        {
            Uid = ExistingUserA,
            Name = marker,
            Type = "letsencrypt",
            Domain = $"{marker}.example.com",
            AutoRenew = false,
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        }).ExecuteReturnIdentityAsync();

        await scope.Db.Insertable(new TaskEntity
        {
            Pid = batchId,
            Type = "issue_cert",
            State = "success",
            Ret = string.Empty,
            Data = "{\"items\":[{\"cert_id\":0}]}",
            CreateAt = now,
            Enable = true
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new TaskEntity
        {
            Pid = batchId,
            Type = "issue_cert",
            State = "fail",
            Ret = "dns verify failed",
            Data = "{\"items\":[{\"cert_id\":" + failCertId + "}]}",
            CreateAt = now,
            Enable = true
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new TaskEntity
        {
            Pid = batchId,
            Type = "issue_cert",
            State = "running",
            Data = "{\"items\":[{\"cert_id\":0}]}",
            CreateAt = now,
            Enable = true
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new TaskEntity
        {
            Pid = batchId,
            Type = "issue_cert",
            State = "retrying",
            Data = "{\"items\":[{\"cert_id\":0}]}",
            CreateAt = now,
            Enable = true
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new TaskEntity
        {
            Pid = batchId,
            Type = "issue_cert",
            State = "waiting",
            Data = "{\"items\":[{\"cert_id\":0}]}",
            CreateAt = now,
            Enable = true
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new TaskEntity
        {
            Pid = batchId,
            Type = "deploy_cert",
            State = "success",
            Data = "{}",
            CreateAt = now,
            Enable = true
        }).ExecuteCommandAsync();

        var sut = CreateSut(scope.Db);
        var result = await sut.BatchProgressAsync(batchId.ToString(), null, true, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotNull(result.Data);
        Assert.Equal(5, result.Data!.Total);
        Assert.Equal(1, result.Data.Success);
        Assert.Equal(1, result.Data.Fail);
        Assert.Equal(2, result.Data.Running);
        Assert.Equal(1, result.Data.Pending);
        Assert.Equal(2, result.Data.Done);
        Assert.Equal(40, result.Data.Percent);
        Assert.Single(result.Data.FailItems);
        Assert.Contains(marker, result.Data.FailItems[0].Domain ?? string.Empty, StringComparison.Ordinal);
        Assert.Equal("dns verify failed", result.Data.FailItems[0].Reason);
    }

    [Fact]
    public async global::System.Threading.Tasks.Task BatchProgressAsync_UserRequest_OnlyReturnsOwnedTasks()
    {
        using var scope = new RealMySqlTestScope();
        var markerA = "cert-batch-user-a-" + Guid.NewGuid().ToString("N");
        var markerB = "cert-batch-user-b-" + Guid.NewGuid().ToString("N");
        var now = DateTime.Now;
        var batchId = (int)(DateTimeOffset.UtcNow.ToUnixTimeSeconds() % int.MaxValue);

        var userACertId = await scope.Db.Insertable(new Cert
        {
            Uid = ExistingUserA,
            Name = markerA,
            Type = "letsencrypt",
            Domain = $"{markerA}.example.com",
            AutoRenew = false,
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        }).ExecuteReturnIdentityAsync();

        var userBCertId = await scope.Db.Insertable(new Cert
        {
            Uid = ExistingUserB,
            Name = markerB,
            Type = "letsencrypt",
            Domain = $"{markerB}.example.com",
            AutoRenew = false,
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        }).ExecuteReturnIdentityAsync();

        await scope.Db.Insertable(new TaskEntity
        {
            Pid = batchId,
            Type = "issue_cert",
            State = "fail",
            Ret = "owned-fail",
            Data = "{\"items\":[{\"cert_id\":" + userACertId + "}]}",
            CreateAt = now,
            Enable = true
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new TaskEntity
        {
            Pid = batchId,
            Type = "issue_cert",
            State = "success",
            Ret = string.Empty,
            Data = "{\"items\":[{\"cert_id\":" + userBCertId + "}]}",
            CreateAt = now,
            Enable = true
        }).ExecuteCommandAsync();

        var sut = CreateSut(scope.Db);
        var result = await sut.BatchProgressAsync(batchId.ToString(), ExistingUserA, false, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotNull(result.Data);
        Assert.Equal(1, result.Data!.Total);
        Assert.Equal(0, result.Data.Success);
        Assert.Equal(1, result.Data.Fail);
        Assert.Equal(0, result.Data.Running);
        Assert.Equal(0, result.Data.Pending);
        Assert.Equal(1, result.Data.Done);
        Assert.Equal(100, result.Data.Percent);
        Assert.Single(result.Data.FailItems);
        Assert.Contains(markerA, result.Data.FailItems[0].Domain ?? string.Empty, StringComparison.Ordinal);
        Assert.DoesNotContain(result.Data.FailItems, x => (x.Domain ?? string.Empty).Contains(markerB, StringComparison.Ordinal));
    }

    private static CertService CreateSut(ISqlSugarClient db)
    {
        return new CertService(
            db,
            new NoopCryptoService(),
            new NoopConfigVersionService(),
            new ConfigurationBuilder().Build(),
            new NoopResourceActionRequestService());
    }

    private sealed class NoopCryptoService : ICryptoService
    {
        public string? Encrypt(string plain) => plain;
        public string? Decrypt(string cipherText) => cipherText;
    }

    private sealed class NoopConfigVersionService : IConfigVersionService
    {
        public global::System.Threading.Tasks.Task<long> BumpAsync(string resource, IReadOnlyList<long> ids, CancellationToken cancellationToken) =>
            global::System.Threading.Tasks.Task.FromResult(0L);
    }

    private sealed class NoopResourceActionRequestService : IResourceActionRequestService
    {
        public global::System.Threading.Tasks.Task<ServiceResult<TaskRequestResult>> RequestAsync(RequestActionCommand command, CancellationToken cancellationToken) =>
            global::System.Threading.Tasks.Task.FromResult(ServiceResult<TaskRequestResult>.Fail(ErrorCodes.InternalError));
    }
}
