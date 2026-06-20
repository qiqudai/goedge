using System.Text.Json;
using System.Text.Json.Serialization;
using Task = System.Threading.Tasks.Task;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class AclService : IAclService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public AclService(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public async Task<ServiceResult<AclListResult>> ListAsync(AclListQuery query, CancellationToken cancellationToken)
    {
        var q = _db.Queryable<Acl>();
        if (query.UserId is > 0)
        {
            q = q.Where(a => a.Uid == query.UserId);
        }

        if (!string.IsNullOrWhiteSpace(query.Name))
        {
            var keyword = query.Name.Trim();
            q = q.Where(a => a.Name!.Contains(keyword));
        }

        if (!string.IsNullOrWhiteSpace(query.Status))
        {
            var status = query.Status.Trim().ToLowerInvariant();
            if (status is "on" or "enabled")
            {
                q = q.Where(a => a.Enable == true);
            }
            else if (status is "off" or "disabled")
            {
                q = q.Where(a => a.Enable == false);
            }
        }

        var total = await q.CountAsync();
        var items = await q.OrderBy(a => a.Id, OrderByType.Desc).ToListAsync();
        var userMap = await LoadUserNamesAsync(items.Select(i => i.Uid).Where(id => id is > 0).Distinct().ToList());

        var list = items.Select(item =>
        {
            var uid = item.Uid ?? 0;
            userMap.TryGetValue(uid, out var username);
            return new AclListItem
            {
                Id = item.Id,
                UserId = uid,
                Uid = uid,
                User = uid > 0 ? new AclUserInfo(username, uid) : null,
                Name = item.Name,
                Description = item.Des,
                DefaultAction = item.DefaultAction,
                Enable = item.Enable,
                CreateTime = item.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss")
            };
        }).ToList();

        return ServiceResult<AclListResult>.Ok(new AclListResult(list, total));
    }

    public async Task<ServiceResult<AclDetailDto>> GetAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<AclDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        var item = await _db.Queryable<Acl>().Where(a => a.Id == id).FirstAsync();
        if (item == null)
        {
            return ServiceResult<AclDetailDto>.Fail(ErrorCodes.NotFound);
        }

        var (rules, denyStatus, redirectUrl) = ParseAclData(item.Data);

        var detail = new AclDetailDto
        {
            Id = item.Id,
            UserId = item.Uid ?? 0,
            Name = item.Name,
            Description = item.Des,
            DefaultAction = item.DefaultAction,
            Enable = item.Enable,
            Rules = rules,
            DefaultDenyStatus = denyStatus,
            DefaultRedirectUrl = redirectUrl
        };

        return ServiceResult<AclDetailDto>.Ok(detail);
    }

    public async Task<ServiceResult<AclDetailDto>> CreateAsync(AclUpsertRequest request, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(request.Name))
        {
            return ServiceResult<AclDetailDto>.Fail(ErrorCodes.MissingParam);
        }

        var defaultAction = string.IsNullOrWhiteSpace(request.DefaultAction) ? "allow" : request.DefaultAction!.Trim();
        var data = BuildAclData(request.Rules, request.DefaultDenyStatus, request.DefaultRedirectUrl);

        var now = DateTime.Now;
        var item = new Acl
        {
            Uid = request.UserId is > 0 ? (int?)request.UserId.Value : null,
            Name = request.Name.Trim(),
            Des = request.Description,
            DefaultAction = defaultAction,
            Enable = request.Enable ?? false,
            Data = data,
            CreateAt = now,
            UpdateAt = now
        };

        var id = await _db.Insertable(item).ExecuteReturnIdentityAsync();
        item.Id = id;

        await _configVersionService.BumpAsync("acl", new[] { (long)id }, cancellationToken);

        var (rules, denyStatus, redirectUrl) = ParseAclData(item.Data);
        var detail = new AclDetailDto
        {
            Id = item.Id,
            UserId = item.Uid ?? 0,
            Name = item.Name,
            Description = item.Des,
            DefaultAction = item.DefaultAction,
            Enable = item.Enable,
            Rules = rules,
            DefaultDenyStatus = denyStatus,
            DefaultRedirectUrl = redirectUrl
        };

        return ServiceResult<AclDetailDto>.Ok(detail);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long id, AclUpsertRequest request, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var existing = await _db.Queryable<Acl>().Where(a => a.Id == id).FirstAsync();
        if (existing == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        var defaultAction = string.IsNullOrWhiteSpace(request.DefaultAction) ? "allow" : request.DefaultAction!.Trim();
        var data = BuildAclData(request.Rules, request.DefaultDenyStatus, request.DefaultRedirectUrl);
        var newUid = request.UserId.HasValue && request.UserId.Value > 0 ? (int?)request.UserId.Value : existing.Uid;
        var newName = request.Name?.Trim();

        var now = DateTime.Now;
        await _db.Updateable<Acl>()
            .SetColumns(a => new Acl
            {
                Uid = newUid,
                Name = newName,
                Des = request.Description,
                DefaultAction = defaultAction,
                Enable = request.Enable,
                Data = data,
                UpdateAt = now
            })
            .Where(a => a.Id == id)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("acl", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var exists = await _db.Queryable<Acl>().Where(a => a.Id == id).AnyAsync();
        if (!exists)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        await _db.Deleteable<Acl>().Where(a => a.Id == id).ExecuteCommandAsync();
        await _configVersionService.BumpAsync("acl", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private static string BuildAclData(IReadOnlyList<AclRuleDto>? rules, int defaultDenyStatus, string? defaultRedirectUrl)
    {
        var payload = new AclDataPayload
        {
            Rules = rules?.ToList() ?? new List<AclRuleDto>(),
            DefaultDenyStatus = defaultDenyStatus,
            DefaultRedirectUrl = defaultRedirectUrl ?? string.Empty
        };

        return JsonSerializer.Serialize(payload, JsonOptions);
    }

    private static (IReadOnlyList<AclRuleDto> Rules, int DefaultDenyStatus, string DefaultRedirectUrl) ParseAclData(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return (Array.Empty<AclRuleDto>(), 0, string.Empty);
        }

        try
        {
            var payload = JsonSerializer.Deserialize<AclDataPayload>(raw, JsonOptions);
            if (payload != null)
            {
                return (payload.Rules ?? new List<AclRuleDto>(), payload.DefaultDenyStatus, payload.DefaultRedirectUrl ?? string.Empty);
            }
        }
        catch
        {
            // ignore
        }

        try
        {
            var rules = JsonSerializer.Deserialize<List<AclRuleDto>>(raw, JsonOptions);
            if (rules != null)
            {
                return (rules, 0, string.Empty);
            }
        }
        catch
        {
            // ignore
        }

        return (Array.Empty<AclRuleDto>(), 0, string.Empty);
    }

    private async Task<Dictionary<long, string>> LoadUserNamesAsync(IReadOnlyList<int?> ids)
    {
        var map = new Dictionary<long, string>();
        var list = ids.Where(id => id is > 0).Select(id => id!.Value).Distinct().ToList();
        if (list.Count == 0)
        {
            return map;
        }

        var rows = await _db.Queryable<User>()
            .Where(u => list.Contains(u.Id))
            .Select(u => new { u.Id, u.Name })
            .ToListAsync();

        foreach (var row in rows)
        {
            map[row.Id] = row.Name ?? string.Empty;
        }

        return map;
    }

    private sealed class AclDataPayload
    {
        [JsonPropertyName("rules")]
        public List<AclRuleDto> Rules { get; set; } = new();

        [JsonPropertyName("default_deny_status")]
        public int DefaultDenyStatus { get; set; }

        [JsonPropertyName("default_redirect_url")]
        public string DefaultRedirectUrl { get; set; } = string.Empty;
    }
}
