using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class CcMatcherService : ICcMatcherService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public CcMatcherService(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public async Task<ServiceResult<CcListResult<CcMatcherListItem>>> ListAsync(CcListQuery query, long? userId, bool userScope, CancellationToken cancellationToken)
    {
        var q = _db.Queryable<CcMatch>();
        if (userScope && userId is > 0)
        {
            var uid = (int)userId.Value;
            q = q.Where(m => m.Uid == uid || m.Uid == 0);
        }

        if (!string.IsNullOrWhiteSpace(query.Name))
        {
            var keyword = query.Name.Trim();
            q = q.Where(m => m.Name!.Contains(keyword));
        }

        if (!string.IsNullOrWhiteSpace(query.Status))
        {
            var status = query.Status.Trim().ToLowerInvariant();
            if (status == "on")
            {
                q = q.Where(m => m.Enable == true);
            }
            else if (status == "off")
            {
                q = q.Where(m => m.Enable == false);
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
            return new CcMatcherListItem
            {
                Id = item.Id,
                UserId = uid,
                Uid = uid,
                User = uid > 0 ? new CcUserInfo(username, uid) : null,
                Name = item.Name,
                IsSystem = isSystem,
                IsOn = item.Enable,
                Status = "normal",
                CreateTime = FormatTime(item.CreateAt),
                CreatedAt = ToUnixSeconds(item.CreateAt)
            };
        }).ToList();

        return ServiceResult<CcListResult<CcMatcherListItem>>.Ok(new CcListResult<CcMatcherListItem>(list, total));
    }

    public async Task<ServiceResult<CcMatcherDetailDto>> GetAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<CcMatcherDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        var item = await _db.Queryable<CcMatch>().Where(m => m.Id == id).FirstAsync();
        if (item == null)
        {
            return ServiceResult<CcMatcherDetailDto>.Fail(ErrorCodes.NotFound);
        }

        var rules = ParseData(item.Data);
        var uid = item.Uid ?? 0;
        var isSystem = item.Internal == true || uid == 0;

        var detail = new CcMatcherDetailDto
        {
            Id = item.Id,
            Name = item.Name,
            Remark = item.Des,
            UserId = uid,
            Uid = uid,
            Internal = item.Internal,
            IsSystem = isSystem,
            IsOn = item.Enable,
            Type = MapRuleType(uid, item.Internal),
            Rules = rules
        };

        return ServiceResult<CcMatcherDetailDto>.Ok(detail);
    }

    public async Task<ServiceResult<bool>> CreateAsync(CcMatcherUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken)
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

        var data = BuildData(request.Rules);
        var now = DateTime.Now;
        var item = new CcMatch
        {
            Uid = uid,
            Name = request.Name.Trim(),
            Des = request.Remark,
            Data = data,
            Enable = request.IsOn,
            Internal = internalRule,
            CreateAt = now,
            UpdateAt = now
        };

        var id = await _db.Insertable(item).ExecuteReturnIdentityAsync();
        item.Id = id;
        await _configVersionService.BumpAsync("cc_match", new[] { (long)id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long id, CcMatcherUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var item = await _db.Queryable<CcMatch>().Where(m => m.Id == id).FirstAsync();
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
        else
        {
            if (string.Equals(request.Type, "system", StringComparison.OrdinalIgnoreCase))
            {
                item.Internal = true;
                item.Uid = 0;
            }
            else
            {
                item.Internal = false;
                if (request.UserId is > 0)
                {
                    item.Uid = (int)request.UserId.Value;
                }
            }
        }

        var data = BuildData(request.Rules);
        item.Name = request.Name?.Trim();
        item.Des = request.Remark;
        item.Data = data;
        item.Enable = request.IsOn;
        item.UpdateAt = DateTime.Now;

        await _db.Updateable<CcMatch>()
            .SetColumns(m => new CcMatch
            {
                Uid = item.Uid,
                Name = item.Name,
                Des = item.Des,
                Data = item.Data,
                Enable = item.Enable,
                Internal = item.Internal,
                UpdateAt = item.UpdateAt
            })
            .Where(m => m.Id == id)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("cc_match", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isUserRequest, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var item = await _db.Queryable<CcMatch>().Where(m => m.Id == id).FirstAsync();
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

        await _db.Deleteable<CcMatch>().Where(m => m.Id == id).ExecuteCommandAsync();
        await _configVersionService.BumpAsync("cc_match", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private static string BuildData(IReadOnlyList<JsonElement>? rules)
    {
        var payload = new CcMatcherPayload
        {
            Rules = rules?.ToList() ?? new List<JsonElement>()
        };
        return JsonSerializer.Serialize(payload, JsonOptions);
    }

    private static IReadOnlyList<JsonElement> ParseData(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return Array.Empty<JsonElement>();
        }

        try
        {
            var payload = JsonSerializer.Deserialize<CcMatcherPayload>(raw, JsonOptions);
            if (payload != null)
            {
                return payload.Rules ?? new List<JsonElement>();
            }
        }
        catch
        {
            // ignore
        }

        return Array.Empty<JsonElement>();
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

    private sealed class CcMatcherPayload
    {
        [JsonPropertyName("rules")]
        public List<JsonElement>? Rules { get; set; }
    }
}
