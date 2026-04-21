using Cnn.Api.Services.Admin;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using System.Text.Json;
using SystemTask = System.Threading.Tasks.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class DnsApiServiceOwnershipTests
{
    private const int ExistingUserA = 2;
    private const int ExistingUserB = 3;

    [Fact]
    public async SystemTask ListAsync_UserRequest_ReturnsOnlyOwnedItems()
    {
        using var scope = new RealMySqlTestScope();
        await scope.Db.Insertable(new Dnsapi
        {
            Uid = ExistingUserA,
            Name = "user-owned",
            Type = "cloudflare",
            Auth = "owned-auth"
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new Dnsapi
        {
            Uid = ExistingUserB,
            Name = "other-owned",
            Type = "aliyun",
            Auth = "other-auth"
        }).ExecuteCommandAsync();

        var sut = new DnsApiService(scope.Db);
        var result = await sut.ListAsync(new DnsApiListQuery(), ExistingUserA, true, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotEmpty(result.Data!.List);
        Assert.True(result.Data.Total >= result.Data.List.Count);
        Assert.All(result.Data.List, item => Assert.Equal(ExistingUserA, item.UserId));
        Assert.Contains(result.Data.List, item => item.Name == "user-owned" && item.Auth == "owned-auth");
        Assert.DoesNotContain(result.Data.List, item => item.Name == "other-owned");
    }

    [Fact]
    public async SystemTask ListAsync_AdminRequest_WithUserIdFiltersTargetUser()
    {
        using var scope = new RealMySqlTestScope();
        await scope.Db.Insertable(new Dnsapi
        {
            Uid = ExistingUserA,
            Name = "admin-target",
            Type = "cloudflare",
            Auth = "target-auth"
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new Dnsapi
        {
            Uid = ExistingUserB,
            Name = "admin-other",
            Type = "aliyun",
            Auth = "other-auth"
        }).ExecuteCommandAsync();

        var sut = new DnsApiService(scope.Db);
        var result = await sut.ListAsync(new DnsApiListQuery { UserId = ExistingUserA }, null, false, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotEmpty(result.Data!.List);
        Assert.True(result.Data.Total >= result.Data.List.Count);
        Assert.All(result.Data.List, item => Assert.Equal(ExistingUserA, item.UserId));
        Assert.Contains(result.Data.List, item => item.Name == "admin-target" && item.Auth == "target-auth");
        Assert.DoesNotContain(result.Data.List, item => item.Name == "admin-other");
    }

    [Fact]
    public async SystemTask ListAsync_WithPagination_ReturnsPagedDataAndTotal()
    {
        using var scope = new RealMySqlTestScope();
        var marker = "dnsapi-page-" + Guid.NewGuid().ToString("N");

        await scope.Db.Insertable(new Dnsapi
        {
            Uid = ExistingUserA,
            Name = marker + "-1",
            Type = "cloudflare",
            Auth = "a1"
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new Dnsapi
        {
            Uid = ExistingUserA,
            Name = marker + "-2",
            Type = "cloudflare",
            Auth = "a2"
        }).ExecuteCommandAsync();
        await scope.Db.Insertable(new Dnsapi
        {
            Uid = ExistingUserA,
            Name = marker + "-3",
            Type = "cloudflare",
            Auth = "a3"
        }).ExecuteCommandAsync();

        var sut = new DnsApiService(scope.Db);
        var result = await sut.ListAsync(new DnsApiListQuery
        {
            UserId = ExistingUserA,
            Keyword = marker,
            Page = 2,
            PageSize = 1
        }, null, false, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotNull(result.Data);
        Assert.Equal(3, result.Data!.Total);
        Assert.Single(result.Data.List);
        Assert.Contains(marker, result.Data.List[0].Name ?? string.Empty, StringComparison.Ordinal);
    }

    [Fact]
    public async SystemTask CreateAsync_DnsPodIntl_NormalizesLegacyIdTokenCredentials()
    {
        using var scope = new RealMySqlTestScope();
        var sut = new DnsApiService(scope.Db);

        var result = await sut.CreateAsync(new DnsApiCreateRequest
        {
            UserId = ExistingUserA,
            Name = "dnspod-intl-test",
            Type = "dnspod_intl",
            Data = JsonDocument.Parse("{\"id\":\"legacy-id\",\"token\":\"legacy-token\",\"ttl\":600}").RootElement
        }, null, false, CancellationToken.None);

        Assert.True(result.Success);
        Assert.NotNull(result.Data);
        Assert.False(string.IsNullOrWhiteSpace(result.Data!.Auth));

        using var authDoc = JsonDocument.Parse(result.Data.Auth!);
        Assert.Equal("legacy-id", authDoc.RootElement.GetProperty("secret_id").GetString());
        Assert.Equal("legacy-token", authDoc.RootElement.GetProperty("secret_key").GetString());
        Assert.Equal(600, authDoc.RootElement.GetProperty("ttl").GetInt32());
    }
}
