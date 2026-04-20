using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using SystemTask = System.Threading.Tasks.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class SiteServiceOwnershipTests
{
    private const int ExistingUserA = 2;
    private const int ExistingUserB = 3;

    [Fact]
    public async SystemTask ListAsync_UserRequest_ReturnsOnlyOwnedSites()
    {
        using var scope = new RealMySqlTestScope();
        var ownedMarker = "site-owned-" + Guid.NewGuid().ToString("N");
        var otherMarker = "site-other-" + Guid.NewGuid().ToString("N");

        await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserA,
            Domain = $"[\"{ownedMarker}.example.com\"]",
            Enable = false,
            State = "stop",
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserB,
            Domain = $"[\"{otherMarker}.example.com\"]",
            Enable = false,
            State = "stop",
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();

        var sut = CreateSut(scope.Db);
        var result = await sut.ListAsync(new SiteListQuery(), ExistingUserA, false, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotNull(result.Data);
        var list = result.Data!.List;
        Assert.Contains(list, item => (item.DomainDisplay ?? string.Empty).Contains(ownedMarker, StringComparison.Ordinal));
        Assert.DoesNotContain(list, item => (item.DomainDisplay ?? string.Empty).Contains(otherMarker, StringComparison.Ordinal));
        Assert.All(list, item => Assert.Equal(ExistingUserA, item.UserId));
    }

    [Fact]
    public async SystemTask ListAsync_AdminRequest_WithUserIdFiltersTargetUserSites()
    {
        using var scope = new RealMySqlTestScope();
        var targetMarker = "site-admin-target-" + Guid.NewGuid().ToString("N");
        var otherMarker = "site-admin-other-" + Guid.NewGuid().ToString("N");

        await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserA,
            Domain = $"[\"{targetMarker}.example.com\"]",
            Enable = false,
            State = "stop",
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserB,
            Domain = $"[\"{otherMarker}.example.com\"]",
            Enable = false,
            State = "stop",
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();

        var sut = CreateSut(scope.Db);
        var result = await sut.ListAsync(new SiteListQuery { UserId = ExistingUserA }, null, true, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotNull(result.Data);
        var list = result.Data!.List;
        Assert.Contains(list, item => (item.DomainDisplay ?? string.Empty).Contains(targetMarker, StringComparison.Ordinal));
        Assert.DoesNotContain(list, item => (item.DomainDisplay ?? string.Empty).Contains(otherMarker, StringComparison.Ordinal));
        Assert.All(list, item => Assert.Equal(ExistingUserA, item.UserId));
    }

    [Fact]
    public async SystemTask ListAsync_WithPagination_ReturnsExpectedSliceAndTotal()
    {
        using var scope = new RealMySqlTestScope();
        var marker = "site-page-" + Guid.NewGuid().ToString("N");

        await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserA,
            Domain = $"[\"{marker}-1.example.com\"]",
            Enable = true,
            State = "run",
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserA,
            Domain = $"[\"{marker}-2.example.com\"]",
            Enable = true,
            State = "run",
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new Site
        {
            Uid = ExistingUserA,
            Domain = $"[\"{marker}-3.example.com\"]",
            Enable = true,
            State = "run",
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();

        var sut = CreateSut(scope.Db);
        var result = await sut.ListAsync(new SiteListQuery
        {
            UserId = ExistingUserA,
            SearchField = "domain",
            Keyword = marker,
            Page = 2,
            PageSize = 1
        }, null, true, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotNull(result.Data);
        Assert.Equal(3, result.Data!.Total);
        Assert.Single(result.Data.List);
        Assert.Contains(marker, result.Data.List[0].DomainDisplay ?? string.Empty, StringComparison.Ordinal);
    }

    private static SiteService CreateSut(ISqlSugarClient db)
    {
        return new SiteService(
            db,
            new NoopConfigVersionService(),
            new NoopDnsSyncService(),
            new NoopSystemConfigService(),
            new NoopCryptoService(),
            new StaticGlobalConfigService(),
            new NoopCertService(),
            new EmptySiteSettingsStore(),
            new NoopResourceActionRequestService());
    }

    private sealed class NoopConfigVersionService : IConfigVersionService
    {
        public global::System.Threading.Tasks.Task<long> BumpAsync(string resource, IReadOnlyList<long> ids, CancellationToken cancellationToken) => global::System.Threading.Tasks.Task.FromResult(0L);
    }

    private sealed class NoopDnsSyncService : IDnsSyncService
    {
        public global::System.Threading.Tasks.Task<bool> SyncUserDnsRecordsAsync(Site? oldSite, Site? newSite) => global::System.Threading.Tasks.Task.FromResult(true);
        public global::System.Threading.Tasks.Task<bool> SyncLineRecordsAsync(long groupId, string lineId, string lineName, string action, IReadOnlyList<long> nodeIds) => global::System.Threading.Tasks.Task.FromResult(true);
        public global::System.Threading.Tasks.Task<bool> SyncPackageCnameForLineChangeAsync(long groupId, string lineId, string lineName, IReadOnlyList<long> nodeIds, string action) => global::System.Threading.Tasks.Task.FromResult(true);
        public global::System.Threading.Tasks.Task<bool> SyncPackageCnameForNodesAsync(IReadOnlyList<long> nodeIds, string action) => global::System.Threading.Tasks.Task.FromResult(true);
        public global::System.Threading.Tasks.Task<bool> SyncPackageLineRecordsAsync(CnameDomains domain, string host, long groupId, string lineId, string lineName, string action, IReadOnlyList<long> nodeIds) => global::System.Threading.Tasks.Task.FromResult(true);
    }

    private sealed class NoopSystemConfigService : ISystemConfigService
    {
        public global::System.Threading.Tasks.Task<Dictionary<string, string>> LoadSystemConfigAsync(CancellationToken cancellationToken) =>
            global::System.Threading.Tasks.Task.FromResult(new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase));

        public bool ParseBoolFlag(string? raw) => false;
    }

    private sealed class NoopCryptoService : ICryptoService
    {
        public string? Encrypt(string plain) => plain;
        public string? Decrypt(string cipherText) => cipherText;
    }

    private sealed class StaticGlobalConfigService : IGlobalConfigService
    {
        public global::System.Threading.Tasks.Task<ServiceResult<GlobalConfigDto>> GetAsync(CancellationToken cancellationToken) =>
            global::System.Threading.Tasks.Task.FromResult(ServiceResult<GlobalConfigDto>.Ok(new GlobalConfigDto()));

        public global::System.Threading.Tasks.Task<ServiceResult<bool>> UpdateAsync(GlobalConfigDto config, CancellationToken cancellationToken) =>
            global::System.Threading.Tasks.Task.FromResult(ServiceResult<bool>.Ok(true));
    }

    private sealed class NoopCertService : ICertService
    {
        public global::System.Threading.Tasks.Task<ServiceResult<CertListResult>> ListAsync(CertListQuery query, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<CertItemDto>> CreateAsync(CertCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<bool>> UpdateAsync(long id, CertUpdateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<CertBatchCreateResult>> BatchCreateAsync(CertBatchCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<CertBatchProgressResult>> BatchProgressAsync(string batchId, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<CertWildcardResult>> WildcardCreateAsync(CertWildcardRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<CertBatchActionResult>> BatchActionAsync(CertBatchActionRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<bool>> ReissueAsync(CertReissueRequest request, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<DnsChallengeInfoDto?>> GetDnsChallengeAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<bool>> VerifyDnsChallengeAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<CertDownloadPayload>> DownloadAsync(long id, long? userId, bool isAdmin, string? domain, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<CertDefaultSettingsDto>> GetDefaultSettingsAsync(long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<bool>> UpdateDefaultSettingsAsync(CertDefaultSettingsRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken) => throw new NotSupportedException();
        public global::System.Threading.Tasks.Task<ServiceResult<bool>> UpdateIssuedCertAsync(Cnn.Common.Contracts.Agent.AgentIssuedCertRequest request, CancellationToken cancellationToken) => throw new NotSupportedException();
    }

    private sealed class EmptySiteSettingsStore : ISiteSettingsStore
    {
        public global::System.Threading.Tasks.Task<Dictionary<string, object?>> LoadSettingsAsync(long siteId, CancellationToken cancellationToken = default) =>
            global::System.Threading.Tasks.Task.FromResult(new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase));

        public global::System.Threading.Tasks.Task<Dictionary<long, Dictionary<string, object?>>> LoadSettingsMapAsync(IReadOnlyList<long> siteIds, CancellationToken cancellationToken = default) =>
            global::System.Threading.Tasks.Task.FromResult(new Dictionary<long, Dictionary<string, object?>>());

        public global::System.Threading.Tasks.Task SaveSettingsAsync(long siteId, Dictionary<string, object?> settings, CancellationToken cancellationToken = default) =>
            global::System.Threading.Tasks.Task.CompletedTask;

        public global::System.Threading.Tasks.Task<Dictionary<long, string>> LoadSiteTypeMapAsync(IReadOnlyList<long> siteIds, CancellationToken cancellationToken = default) =>
            global::System.Threading.Tasks.Task.FromResult(new Dictionary<long, string>());

        public global::System.Threading.Tasks.Task SaveSiteTypeAsync(long siteId, string siteType, CancellationToken cancellationToken = default) =>
            global::System.Threading.Tasks.Task.CompletedTask;
    }

    private sealed class NoopResourceActionRequestService : IResourceActionRequestService
    {
        public global::System.Threading.Tasks.Task<ServiceResult<TaskRequestResult>> RequestAsync(RequestActionCommand command, CancellationToken cancellationToken) =>
            global::System.Threading.Tasks.Task.FromResult(ServiceResult<TaskRequestResult>.Fail(ErrorCodes.InternalError));
    }
}
