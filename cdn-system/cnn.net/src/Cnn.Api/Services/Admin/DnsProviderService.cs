using System.Text.Json;
using Task = System.Threading.Tasks.Task;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Common.Dns;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class DnsProviderService : IDnsProviderService
{
    private static readonly IReadOnlyList<DnsProviderTypeItem> ProviderTypes =
    [
        new("aliyun", "Aliyun", new[] { "access_key_id", "access_key_secret" }),
        new("huawei", "Huawei", new[] { "id", "secret" }),
        new("dnsla", "DNSLA", new[] { "id", "secret" }),
        new("dnspod", "DNSPod", new[] { "id", "token" }),
        new("dnspod_intl", "DNSPod Intl", new[] { "secret_id", "secret_key" }),
        new("51dns", "51DNS", new[] { "id", "secret" }),
        new("cloudflare", "Cloudflare", new[] { "email", "api_key" }),
        new("godaddy", "GoDaddy", new[] { "key", "secret" })
    ];

    private readonly ISqlSugarClient _db;
    private readonly IDnsMaintenanceService _maintenance;

    public DnsProviderService(ISqlSugarClient db, IDnsMaintenanceService maintenance)
    {
        _db = db;
        _maintenance = maintenance;
    }

    public async Task<ServiceResult<DnsProviderListResult>> ListProvidersAsync(DnsProviderListQuery query, CancellationToken cancellationToken)
    {
        var q = _db.Queryable<Dnsapi>();
        if (query.UserId is > 0)
        {
            q = q.Where(p => p.Uid == query.UserId);
        }

        var list = await q.OrderBy(p => p.Id, OrderByType.Desc).ToListAsync();
        var items = list.Select(p => new DnsProviderItem
        {
            Id = p.Id,
            UserId = p.Uid,
            Name = p.Name,
            Remark = p.Des,
            Type = p.Type,
            Auth = p.Auth
        }).ToList();

        return ServiceResult<DnsProviderListResult>.Ok(new DnsProviderListResult(items));
    }

    public Task<ServiceResult<DnsProviderTypesResult>> GetProviderTypesAsync(CancellationToken cancellationToken)
    {
        return Task.FromResult(ServiceResult<DnsProviderTypesResult>.Ok(new DnsProviderTypesResult(ProviderTypes)));
    }

    public async Task<ServiceResult<bool>> CreateProviderAsync(DnsProviderCreateRequest request, CancellationToken cancellationToken)
    {
        var name = request.Name?.Trim();
        var type = request.Type?.Trim();
        var credentials = request.Credentials?.Trim();

        if (string.IsNullOrWhiteSpace(name) || string.IsNullOrWhiteSpace(type))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        if (!TryValidateCredentials(type, credentials, out var messageKey))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, messageKey);
        }

        int? userId = null;
        if (request.UserId is > 0)
        {
            userId = (int)request.UserId.Value;
        }

        var item = new Dnsapi
        {
            Uid = userId,
            Name = name,
            Des = string.Empty,
            Type = type,
            Auth = credentials
        };

        await _db.Insertable(item).ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateProviderAsync(long id, DnsProviderUpdateRequest request, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var name = request.Name?.Trim();
        var type = request.Type?.Trim();
        var credentials = request.Credentials?.Trim();

        if (string.IsNullOrWhiteSpace(name) || string.IsNullOrWhiteSpace(type))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        if (!TryValidateCredentials(type, credentials, out var messageKey))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, messageKey);
        }

        var provider = await _db.Queryable<Dnsapi>().Where(p => p.Id == id).FirstAsync();
        if (provider == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "dns_provider_not_found");
        }

        int? userId = provider.Uid;
        if (request.UserId is > 0)
        {
            userId = (int)request.UserId.Value;
        }

        provider.Uid = userId;
        provider.Name = name;
        provider.Type = type;
        provider.Auth = credentials;

        await _db.Updateable(provider)
            .UpdateColumns(p => new { p.Uid, p.Name, p.Type, p.Auth })
            .Where(p => p.Id == id)
            .ExecuteCommandAsync();

        _ = await _maintenance.ResyncForProviderAsync(id, cancellationToken);

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteProviderAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var used = await _db.Queryable<CnameDomains>().Where(c => c.DnsProviderId == id).AnyAsync();
        if (used)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InUse, "dns_provider_in_use");
        }

        await _db.Deleteable<Dnsapi>().Where(p => p.Id == id).ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<DnsTestResult>> TestAsync(CancellationToken cancellationToken)
    {
        var domain = await _db.Queryable<CnameDomains>()
            .Where(c => c.DnsProviderId != 0)
            .OrderBy(c => c.Id, OrderByType.Desc)
            .FirstAsync();

        if (domain == null || string.IsNullOrWhiteSpace(domain.Domain))
        {
            return ServiceResult<DnsTestResult>.Fail(ErrorCodes.ConfigError, "cname_domains_not_configured");
        }

        if (domain.DnsProviderId == 0)
        {
            return ServiceResult<DnsTestResult>.Fail(ErrorCodes.ConfigError, "dns_provider_not_configured");
        }

        var provider = await _db.Queryable<Dnsapi>().Where(p => p.Id == domain.DnsProviderId).FirstAsync();
        if (provider == null)
        {
            return ServiceResult<DnsTestResult>.Fail(ErrorCodes.ConfigError, "dns_provider_not_configured");
        }

        if (!TryValidateCredentials(provider.Type, provider.Auth, out var messageKey))
        {
            return ServiceResult<DnsTestResult>.Fail(ErrorCodes.ExternalProviderError, messageKey ?? "dns_provider_not_available");
        }

        var dnsProvider = DnsProviderFactory.TryCreate(provider.Type, provider.Auth);
        if (dnsProvider == null)
        {
            return ServiceResult<DnsTestResult>.Fail(ErrorCodes.ExternalProviderError, "dns_provider_not_available");
        }

        try
        {
            await dnsProvider.GetRecordsAsync(domain.Domain);
        }
        catch (Exception ex)
        {
            var message = string.IsNullOrWhiteSpace(ex.Message) ? "dns_provider_not_available" : ex.Message;
            return ServiceResult<DnsTestResult>.Fail(ErrorCodes.ExternalProviderError, message);
        }

        return ServiceResult<DnsTestResult>.Ok(new DnsTestResult("ok"));
    }

    public Task<ServiceResult<DnsFixResult>> FixRecordsAsync(CancellationToken cancellationToken)
    {
        return ExecuteMaintenanceAsync(
            () => _maintenance.RepairRecordsAsync(cancellationToken),
            () => new DnsFixResult("ok"));
    }

    public Task<ServiceResult<DnsCleanupResult>> CleanupRecordsAsync(CancellationToken cancellationToken)
    {
        return ExecuteMaintenanceAsync(
            () => _maintenance.CleanupInvalidRecordsAsync(cancellationToken),
            () => new DnsCleanupResult("ok"));
    }

    private static bool TryValidateCredentials(string? type, string? credentials, out string? messageKey)
    {
        messageKey = null;
        var normalizedType = type?.Trim();
        if (string.IsNullOrWhiteSpace(normalizedType) || string.IsNullOrWhiteSpace(credentials))
        {
            messageKey = "invalid_credentials";
            return false;
        }

        try
        {
            using var doc = JsonDocument.Parse(credentials);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                messageKey = "invalid_credentials_format";
                return false;
            }
            if (string.Equals(normalizedType, "dnspod_intl", StringComparison.OrdinalIgnoreCase))
            {
                if (!doc.RootElement.TryGetProperty("secret_id", out var secretId) ||
                    !doc.RootElement.TryGetProperty("secret_key", out var secretKey) ||
                    string.IsNullOrWhiteSpace(secretId.GetString()) ||
                    string.IsNullOrWhiteSpace(secretKey.GetString()))
                {
                    messageKey = "invalid_credentials_required";
                    return false;
                }
            }

            var provider = DnsProviderFactory.TryCreate(normalizedType, credentials);
            if (provider == null)
            {
                messageKey = "invalid_credentials";
                return false;
            }

            return true;
        }
        catch
        {
            messageKey = "invalid_credentials_format";
            return false;
        }
    }

    private static async Task<ServiceResult<T>> ExecuteMaintenanceAsync<T>(
        Func<Task<IReadOnlyList<string>>> action,
        Func<T> successFactory)
    {
        var errors = await action();
        if (errors.Count > 0)
        {
            return ServiceResult<T>.Fail(ErrorCodes.ExternalProviderError, "dns_sync_failed");
        }

        return ServiceResult<T>.Ok(successFactory());
    }
}
