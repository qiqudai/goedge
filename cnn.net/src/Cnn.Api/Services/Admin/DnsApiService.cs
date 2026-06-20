using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using System.Text.Json;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public interface IDnsApiService
{
    Task<ServiceResult<DnsApiListResult>> ListAsync(DnsApiListQuery query, long? userId, bool isUserRequest, CancellationToken cancellationToken);
    Task<ServiceResult<DnsApiItemDto>> CreateAsync(DnsApiCreateRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, DnsApiUpdateRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isUserRequest, CancellationToken cancellationToken);
    Task<ServiceResult<DnsApiTypesResult>> GetTypesAsync(CancellationToken cancellationToken);
}

public sealed class DnsApiService : IDnsApiService
{
    private static readonly IReadOnlyList<DnsApiTypeItem> Types = new List<DnsApiTypeItem>
    {
        new("cloudflare", "Cloudflare", new[] { "email", "api_key" }),
        new("aliyun", "Aliyun", new[] { "access_key_id", "access_key_secret" }),
        new("dnspod", "DNSPod.cn", new[] { "id", "token" }),
        new("dnspod_intl", "DNSPod.com", new[] { "secret_id", "secret_key" }),
        new("godaddy", "GoDaddy", new[] { "api_key", "api_secret" }),
        new("namecom", "Name.com", new[] { "username", "api_token" }),
        new("namecheap", "Namecheap", new[] { "user", "api_key", "ip" }),
        new("cloudns", "ClouDNS", new[] { "auth_id", "auth_password" }),
        new("namesilo", "Namesilo", new[] { "api_key" }),
        new("jdcloud", "JDCloud", new[] { "access_key", "secret_key" }),
        new("dnsla", "DNS.LA", new[] { "api_id", "api_pass" }),
        new("51dns", "51DNS", new[] { "app_id", "app_secret" }),
        new("huawei", "Huawei Cloud", new[] { "access_key_id", "secret_access_key" })
    };

    private readonly ISqlSugarClient _db;

    public DnsApiService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<DnsApiListResult>> ListAsync(
        DnsApiListQuery query,
        long? userId,
        bool isUserRequest,
        CancellationToken cancellationToken)
    {
        query ??= new DnsApiListQuery();
        var uid = userId.GetValueOrDefault();
        if (isUserRequest && uid <= 0)
        {
            return ServiceResult<DnsApiListResult>.Fail(ErrorCodes.PermissionDenied);
        }

        var q = _db.Queryable<Dnsapi>();
        if (isUserRequest)
        {
            q = q.Where(d => d.Uid == (int)uid);
        }
        else if (query.UserId is > 0)
        {
            q = q.Where(d => d.Uid == (int)query.UserId.Value);
        }
        if (!string.IsNullOrWhiteSpace(query.Keyword))
        {
            var keyword = query.Keyword.Trim();
            q = q.Where(d => SqlFunc.Contains(d.Name, keyword) || SqlFunc.Contains(d.Type, keyword));
        }

        var page = Math.Max(1, query.Page);
        var pageSize = query.PageSize <= 0 ? 1000 : query.PageSize;
        pageSize = Math.Min(1000, pageSize);

        var total = await q.CountAsync();
        var list = await q.OrderBy(d => d.Id, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);
        var items = list.Select(item =>
        {
            var auth = item.Auth;
            if (isUserRequest && uid > 0 && item.Uid != (int)uid)
            {
                auth = string.Empty;
            }

            return new DnsApiItemDto
            {
                Id = item.Id,
                UserId = item.Uid ?? 0,
                Name = item.Name,
                Remark = item.Des,
                Type = item.Type,
                Auth = auth
            };
        }).ToList();

        return ServiceResult<DnsApiListResult>.Ok(new DnsApiListResult(items, total));
    }

    public async Task<ServiceResult<DnsApiItemDto>> CreateAsync(
        DnsApiCreateRequest request,
        long? userId,
        bool isUserRequest,
        CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<DnsApiItemDto>.Fail(ErrorCodes.InvalidParam);
        }

        var name = request.Name?.Trim();
        var type = request.Type?.Trim();
        if (string.IsNullOrWhiteSpace(name))
        {
            return ServiceResult<DnsApiItemDto>.Fail(ErrorCodes.MissingParam, "dnsapi_name_required");
        }
        if (string.IsNullOrWhiteSpace(type))
        {
            return ServiceResult<DnsApiItemDto>.Fail(ErrorCodes.MissingParam, "dnsapi_type_required");
        }

        var uid = request.UserId;
        if (uid <= 0)
        {
            uid = userId ?? 0;
        }
        if (isUserRequest)
        {
            uid = userId ?? 0;
        }
        if (isUserRequest && uid <= 0)
        {
            return ServiceResult<DnsApiItemDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var auth = NormalizeAuth(type, request.Auth, request.Data);

        var item = new Dnsapi
        {
            Uid = uid > 0 ? (int)uid : 0,
            Name = name,
            Des = request.Remark,
            Type = type,
            Auth = auth
        };

        var id = await _db.Insertable(item).ExecuteReturnIdentityAsync();
        if (id <= 0)
        {
            return ServiceResult<DnsApiItemDto>.Fail(ErrorCodes.DbError, "db_create_error");
        }

        item.Id = id;
        return ServiceResult<DnsApiItemDto>.Ok(new DnsApiItemDto
        {
            Id = item.Id,
            UserId = item.Uid ?? 0,
            Name = item.Name,
            Remark = item.Des,
            Type = item.Type,
            Auth = item.Auth
        });
    }

    public async Task<ServiceResult<bool>> UpdateAsync(
        long id,
        DnsApiUpdateRequest request,
        long? userId,
        bool isUserRequest,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "dnsapi_id_required");
        }
        if (isUserRequest)
        {
            var uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            var owns = await _db.Queryable<Dnsapi>().AnyAsync(d => d.Id == id && d.Uid == (int)uid);
            if (!owns)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
        }

        var updates = new Dnsapi
        {
            Name = request.Name?.Trim(),
            Des = request.Remark,
            Type = request.Type?.Trim(),
            Auth = NormalizeAuth(request.Type, request.Auth, request.Data)
        };

        var rows = await _db.Updateable(updates)
            .UpdateColumns(d => new { d.Name, d.Des, d.Type, d.Auth })
            .Where(d => d.Id == id)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(
        long id,
        long? userId,
        bool isUserRequest,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "dnsapi_id_required");
        }

        if (isUserRequest)
        {
            var uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            var owns = await _db.Queryable<Dnsapi>().AnyAsync(d => d.Id == id && d.Uid == (int)uid);
            if (!owns)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
        }

        var rows = await _db.Deleteable<Dnsapi>().Where(d => d.Id == id).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public Task<ServiceResult<DnsApiTypesResult>> GetTypesAsync(CancellationToken cancellationToken)
    {
        return Task.FromResult(ServiceResult<DnsApiTypesResult>.Ok(new DnsApiTypesResult(Types)));
    }

    private static string? NormalizeAuth(string? type, string? auth, JsonElement? data)
    {
        var normalizedType = type?.Trim();
        if (!string.IsNullOrWhiteSpace(auth))
        {
            var normalizedAuth = NormalizeDnsPodIntlAuth(normalizedType, auth);
            return string.IsNullOrWhiteSpace(normalizedAuth) ? auth : normalizedAuth;
        }

        if (data.HasValue &&
            data.Value.ValueKind != JsonValueKind.Undefined &&
            data.Value.ValueKind != JsonValueKind.Null)
        {
            var raw = data.Value.GetRawText();
            var normalizedData = NormalizeDnsPodIntlAuth(normalizedType, raw);
            return string.IsNullOrWhiteSpace(normalizedData) ? raw : normalizedData;
        }

        return auth;
    }

    private static string? NormalizeDnsPodIntlAuth(string? type, string? auth)
    {
        if (!string.Equals(type, "dnspod_intl", StringComparison.OrdinalIgnoreCase) ||
            string.IsNullOrWhiteSpace(auth))
        {
            return auth;
        }

        try
        {
            using var doc = JsonDocument.Parse(auth);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                return auth;
            }

            var root = doc.RootElement;
            var secretId = GetCredential(root, "secret_id");
            var secretKey = GetCredential(root, "secret_key");
            if (string.IsNullOrWhiteSpace(secretId))
            {
                secretId = GetCredential(root, "id");
            }
            if (string.IsNullOrWhiteSpace(secretKey))
            {
                secretKey = GetCredential(root, "token");
            }

            if (string.IsNullOrWhiteSpace(secretId) || string.IsNullOrWhiteSpace(secretKey))
            {
                return auth;
            }

            var payload = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase)
            {
                ["secret_id"] = secretId.Trim(),
                ["secret_key"] = secretKey.Trim()
            };

            if (root.TryGetProperty("ttl", out var ttlElement))
            {
                payload["ttl"] = ttlElement.ValueKind switch
                {
                    JsonValueKind.Number when ttlElement.TryGetInt32(out var ttl) => ttl,
                    JsonValueKind.String => ttlElement.GetString(),
                    _ => ttlElement.ToString()
                };
            }

            if (root.TryGetProperty("ip_weight", out var ipWeight))
            {
                payload["ip_weight"] = ipWeight.ValueKind switch
                {
                    JsonValueKind.True => true,
                    JsonValueKind.False => false,
                    JsonValueKind.String when bool.TryParse(ipWeight.GetString(), out var parsed) => parsed,
                    _ => ipWeight.ToString()
                };
            }

            if (root.TryGetProperty("apiType", out var apiType) && apiType.ValueKind == JsonValueKind.String)
            {
                payload["apiType"] = apiType.GetString();
            }

            if (root.TryGetProperty("region", out var region) && region.ValueKind == JsonValueKind.String)
            {
                payload["region"] = region.GetString();
            }

            return JsonSerializer.Serialize(payload);
        }
        catch
        {
            return auth;
        }
    }

    private static string? GetCredential(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return null;
        }

        if (value.ValueKind == JsonValueKind.String)
        {
            return value.GetString();
        }

        return value.ToString();
    }
}
