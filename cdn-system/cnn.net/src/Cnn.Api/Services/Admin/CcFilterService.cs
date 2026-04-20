using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class CcFilterService : ICcFilterService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public CcFilterService(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public async Task<ServiceResult<CcListResult<CcFilterListItem>>> ListAsync(CcListQuery query, long? userId, bool userScope, CancellationToken cancellationToken)
    {
        var q = _db.Queryable<CcFilter>();
        if (userScope && userId is > 0)
        {
            var uid = (int)userId.Value;
            q = q.Where(f => f.Uid == uid || f.Uid == 0);
        }

        if (!string.IsNullOrWhiteSpace(query.Name))
        {
            var keyword = query.Name.Trim();
            q = q.Where(f => f.Name!.Contains(keyword));
        }

        if (!string.IsNullOrWhiteSpace(query.Status))
        {
            var status = query.Status.Trim().ToLowerInvariant();
            if (status == "on")
            {
                q = q.Where(f => f.Enable == true);
            }
            else if (status == "off")
            {
                q = q.Where(f => f.Enable == false);
            }
        }

        var total = await q.CountAsync();
        var items = await q.OrderBy("id desc").ToListAsync();
        var userMap = await LoadUserNamesAsync(items.Select(i => i.Uid).Where(id => id is > 0).Distinct().ToList());

        var list = items.Select(item =>
        {
            var uid = item.Uid ?? 0;
            userMap.TryGetValue(uid, out var username);
            var isSystem = item.Internal == true || uid == 0;
            return new CcFilterListItem
            {
                Id = item.Id,
                UserId = uid,
                Uid = uid,
                User = uid > 0 ? new CcUserInfo(username, uid) : null,
                Name = item.Name,
                IsSystem = isSystem,
                Type = item.Type,
                Action = item.Type,
                Status = "normal",
                IsOn = item.Enable,
                CreateTime = FormatTime(item.CreateAt),
                CreatedAt = ToUnixSeconds(item.CreateAt)
            };
        }).ToList();

        return ServiceResult<CcListResult<CcFilterListItem>>.Ok(new CcListResult<CcFilterListItem>(list, total));
    }

    public async Task<ServiceResult<CcFilterDetailDto>> GetAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<CcFilterDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        var item = await _db.Queryable<CcFilter>().Where(f => f.Id == id).FirstAsync();
        if (item == null)
        {
            return ServiceResult<CcFilterDetailDto>.Fail(ErrorCodes.NotFound);
        }

        var (matchMode, blacklist, auth) = ParseExtra(item.Extra);
        var uid = item.Uid ?? 0;
        var isSystem = item.Internal == true || uid == 0;
        var detail = new CcFilterDetailDto
        {
            Id = item.Id,
            UserId = uid,
            Uid = uid,
            Internal = item.Internal,
            IsSystem = isSystem,
            Type = MapRuleType(uid, item.Internal),
            Name = item.Name,
            Remark = item.Des,
            Enable = item.Enable,
            Action = item.Type,
            MatchMode = matchMode,
            Blacklist = blacklist,
            WithinSecond = item.WithinSecond,
            MaxReq = item.MaxReq,
            MaxReqPerUri = item.MaxReqPerUri,
            Auth = auth
        };

        return ServiceResult<CcFilterDetailDto>.Ok(detail);
    }

    public async Task<ServiceResult<bool>> CreateAsync(CcFilterUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(request.Name))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var uid = 0;
        var internalRule = false;
        if (isUserRequest)
        {
            if (userId is null or <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            uid = (int)userId.Value;
            request.Type = "user";
        }
        else
        {
            if (string.Equals(request.Type, "system", StringComparison.OrdinalIgnoreCase))
            {
                internalRule = true;
                uid = 0;
            }
            else
            {
                internalRule = false;
                if (request.UserId is > 0)
                {
                    uid = (int)request.UserId.Value;
                }
            }
        }

        var extra = BuildExtra(request.MatchMode, request.Blacklist, request.Auth);
        var now = DateTime.Now;
        var item = new CcFilter
        {
            Uid = uid,
            Name = request.Name.Trim(),
            Des = request.Remark,
            Type = request.Action,
            WithinSecond = request.WithinSecond,
            MaxReq = request.MaxReq,
            MaxReqPerUri = request.MaxReqPerUri,
            Extra = extra,
            Internal = internalRule,
            Enable = request.Enable,
            CreateAt = now,
            UpdateAt = now
        };

        var id = await _db.Insertable(item).ExecuteReturnIdentityAsync();
        item.Id = id;
        await _configVersionService.BumpAsync("cc_filter", new[] { (long)id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long id, CcFilterUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var item = await _db.Queryable<CcFilter>().Where(f => f.Id == id).FirstAsync();
        if (item == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        if (isUserRequest)
        {
            if (userId is null or <= 0 || item.Uid != (int)userId.Value)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            request.Type = "user";
        }

        var extra = BuildExtra(request.MatchMode, request.Blacklist, request.Auth);
        item.Name = request.Name?.Trim();
        item.Des = request.Remark;
        item.Type = request.Action;
        item.WithinSecond = request.WithinSecond;
        item.MaxReq = request.MaxReq;
        item.MaxReqPerUri = request.MaxReqPerUri;
        item.Extra = extra;
        item.Enable = request.Enable;
        if (!isUserRequest)
        {
            item.Internal = string.Equals(request.Type, "system", StringComparison.OrdinalIgnoreCase);
        }
        item.UpdateAt = DateTime.Now;

        await _db.Updateable<CcFilter>()
            .SetColumns(f => new CcFilter
            {
                Name = item.Name,
                Des = item.Des,
                Type = item.Type,
                WithinSecond = item.WithinSecond,
                MaxReq = item.MaxReq,
                MaxReqPerUri = item.MaxReqPerUri,
                Extra = item.Extra,
                Enable = item.Enable,
                Internal = item.Internal,
                UpdateAt = item.UpdateAt
            })
            .Where(f => f.Id == id)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("cc_filter", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isUserRequest, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var item = await _db.Queryable<CcFilter>().Where(f => f.Id == id).FirstAsync();
        if (item == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        if (isUserRequest)
        {
            if (userId is null or <= 0 || item.Uid != (int)userId.Value)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
        }

        await _db.Deleteable<CcFilter>().Where(f => f.Id == id).ExecuteCommandAsync();
        await _configVersionService.BumpAsync("cc_filter", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private static string BuildExtra(string? matchMode, bool blacklist, JsonElement? auth)
    {
        var extra = new Dictionary<string, object?>
        {
            ["match_mode"] = matchMode,
            ["blacklist"] = blacklist
        };

        if (auth.HasValue && auth.Value.ValueKind != JsonValueKind.Undefined && auth.Value.ValueKind != JsonValueKind.Null)
        {
            extra["auth"] = auth.Value;
        }

        return JsonSerializer.Serialize(extra, JsonOptions);
    }

    private static (string? MatchMode, bool? Blacklist, JsonElement? Auth) ParseExtra(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return (null, null, null);
        }

        try
        {
            var payload = JsonSerializer.Deserialize<CcFilterExtraPayload>(raw, JsonOptions);
            if (payload != null)
            {
                var auth = payload.Auth.HasValue ? payload.Auth : null;
                return (payload.MatchMode, payload.Blacklist, auth);
            }
        }
        catch
        {
            // ignore
        }

        return (null, null, null);
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

    private static string MapRuleType(int userId, bool? internalRule)
    {
        if (userId == 0 || internalRule == true)
        {
            return "system";
        }

        return "user";
    }

    private static string? FormatTime(DateTime? time)
    {
        return time?.ToString("yyyy-MM-dd HH:mm:ss");
    }

    private static long? ToUnixSeconds(DateTime? time)
    {
        if (!time.HasValue)
        {
            return null;
        }

        return new DateTimeOffset(time.Value).ToUnixTimeSeconds();
    }

    private sealed class CcFilterExtraPayload
    {
        [JsonPropertyName("match_mode")]
        public string? MatchMode { get; set; }

        [JsonPropertyName("blacklist")]
        public bool? Blacklist { get; set; }

        [JsonPropertyName("auth")]
        public JsonElement? Auth { get; set; }
    }
}
