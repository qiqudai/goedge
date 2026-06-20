using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed class SiteDefaultService : ISiteDefaultService
{
    private const string ConfigType = "site_default_config";
    private const string ScopeGlobal = "global";
    private const string ScopeUser = "user";
    private const string ScopeGroup = "group";

    private readonly ISqlSugarClient _db;

    public SiteDefaultService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<SiteDefaultListResult>> ListAsync(
        SiteDefaultListQuery query,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new SiteDefaultListQuery();
        var scopeName = (query.ScopeName ?? string.Empty).Trim();
        var scopeId = query.ScopeId.GetValueOrDefault();
        var queryUserId = query.UserId.GetValueOrDefault();

        long effectiveUserId = 0;
        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<SiteDefaultListResult>.Fail(ErrorCodes.PermissionDenied);
            }
            effectiveUserId = userId.Value;
        }
        else if (queryUserId > 0)
        {
            effectiveUserId = queryUserId;
        }

        if (!string.IsNullOrWhiteSpace(scopeName) || scopeId > 0)
        {
            if (string.IsNullOrWhiteSpace(scopeName))
            {
                scopeName = ScopeGlobal;
            }

            var items = await _db.Queryable<Config>()
                .Where(c => c.Type == ConfigType && c.ScopeName == scopeName && c.ScopeId == scopeId)
                .OrderBy(c => c.Name)
                .ToListAsync();

            var list = items.Select(item => new SiteDefaultItemDto
            {
                Name = item.Name,
                Value = item.Value,
                Type = item.Type,
                ScopeId = item.ScopeId ?? 0,
                ScopeName = item.ScopeName,
                Enable = item.Enable ?? false,
                UserId = 0,
                UserName = string.Empty,
                GroupName = string.Empty
            }).ToList();

            return ServiceResult<SiteDefaultListResult>.Ok(new SiteDefaultListResult(list));
        }

        if (isAdmin && effectiveUserId == 0)
        {
            var items = await _db.Queryable<Config>()
                .Where(c => c.Type == ConfigType &&
                            (c.ScopeName == ScopeGlobal || c.ScopeName == ScopeGroup || c.ScopeName == ScopeUser) &&
                            c.ScopeId != 0)
                .OrderBy("scope_name asc, scope_id asc, name asc")
                .ToListAsync();

            var groupMap = await LoadGroupMapAsync(items);
            var userMap = await LoadUserNameMapAsync(items, groupMap);

            var list = new List<SiteDefaultItemDto>(items.Count);
            foreach (var item in items)
            {
                var outputScope = NormalizeOutputScopeName(item.ScopeName);
                var groupName = string.Empty;
                var ownerId = 0L;
                if (item.ScopeName == ScopeGroup)
                {
                    if (groupMap.TryGetValue(item.ScopeId ?? 0, out var group))
                    {
                        ownerId = group.UserId;
                        groupName = group.Name ?? string.Empty;
                    }
                }
                else
                {
                    ownerId = item.ScopeId ?? 0;
                }

                list.Add(new SiteDefaultItemDto
                {
                    Name = item.Name,
                    Value = item.Value,
                    Type = item.Type,
                    ScopeId = item.ScopeId ?? 0,
                    ScopeName = outputScope,
                    Enable = item.Enable ?? false,
                    UserId = ownerId,
                    UserName = userMap.TryGetValue(ownerId, out var name) ? name : string.Empty,
                    GroupName = groupName
                });
            }

            return ServiceResult<SiteDefaultListResult>.Ok(new SiteDefaultListResult(list));
        }

        if (effectiveUserId <= 0)
        {
            return ServiceResult<SiteDefaultListResult>.Ok(new SiteDefaultListResult(Array.Empty<SiteDefaultItemDto>()));
        }

        var groupIds = await _db.Queryable<SiteGroup>()
            .Where(g => g.Uid == (int)effectiveUserId)
            .Select(g => (long)g.Id)
            .ToListAsync();

        ISugarQueryable<Config> queryable;
        if (groupIds.Count > 0)
        {
            queryable = _db.Queryable<Config>()
                .Where(c => c.Type == ConfigType &&
                            (((c.ScopeName == ScopeGlobal || c.ScopeName == ScopeUser) && c.ScopeId == effectiveUserId) ||
                             (c.ScopeName == ScopeGroup && groupIds.Contains(c.ScopeId ?? 0))));
        }
        else
        {
            queryable = _db.Queryable<Config>()
                .Where(c => c.Type == ConfigType && (c.ScopeName == ScopeGlobal || c.ScopeName == ScopeUser) && c.ScopeId == effectiveUserId);
        }

        var userItems = await queryable.OrderBy("scope_name asc, scope_id asc, name asc").ToListAsync();
        var groupMapByUser = await LoadGroupMapByIdsAsync(groupIds);

        var userList = new List<SiteDefaultItemDto>(userItems.Count);
        foreach (var item in userItems)
        {
            var outputScope = NormalizeOutputScopeName(item.ScopeName);
            var groupName = string.Empty;
            if (item.ScopeName == ScopeGroup && groupMapByUser.TryGetValue(item.ScopeId ?? 0, out var group))
            {
                groupName = group.Name ?? string.Empty;
            }

            userList.Add(new SiteDefaultItemDto
            {
                Name = item.Name,
                Value = item.Value,
                Type = item.Type,
                ScopeId = item.ScopeId ?? 0,
                ScopeName = outputScope,
                Enable = item.Enable ?? false,
                UserId = effectiveUserId,
                UserName = string.Empty,
                GroupName = groupName
            });
        }

        return ServiceResult<SiteDefaultListResult>.Ok(new SiteDefaultListResult(userList));
    }

    public async Task<ServiceResult<bool>> CreateAsync(
        SiteDefaultCreateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        request ??= new SiteDefaultCreateRequest();
        var scopeName = NormalizeScopeName(request.ScopeName);
        var scopeId = request.ScopeId.GetValueOrDefault();

        var effectiveUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (!isAdmin && effectiveUserId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "user_id_required");
        }

        if (!isAdmin && scopeName == ScopeGlobal && scopeId == 0 && effectiveUserId > 0)
        {
            scopeId = effectiveUserId;
        }

        if (scopeName == ScopeGroup)
        {
            if (scopeId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "missing_param");
            }

            if (effectiveUserId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "user_id_required");
            }

            if (!await EnsureGroupOwnerAsync(scopeId, effectiveUserId))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_param");
            }
        }

        if (request.Data != null && request.Data.Count > 0)
        {
            foreach (var (key, value) in request.Data)
            {
                var name = (key ?? string.Empty).Trim();
                if (string.IsNullOrWhiteSpace(name))
                {
                    continue;
                }

                var finalValue = ConvertValue(value);
                await UpsertConfigAsync(name, finalValue, scopeName, scopeId);
            }

            return ServiceResult<bool>.Ok(true);
        }

        var itemName = (request.Name ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(itemName))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "name_required");
        }

        await UpsertConfigAsync(itemName, request.Value ?? string.Empty, scopeName, scopeId);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(
        string name,
        SiteDefaultUpdateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        name = (name ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(name))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "name_required");
        }

        request ??= new SiteDefaultUpdateRequest();
        var effectiveUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (!isAdmin && effectiveUserId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "user_id_required");
        }

        var scopeName = NormalizeScopeName(request.ScopeName);
        var scopeId = request.ScopeId.GetValueOrDefault();
        if (!isAdmin && scopeName == ScopeGlobal && scopeId == 0 && effectiveUserId > 0)
        {
            scopeId = effectiveUserId;
        }

        if (scopeName == ScopeGroup)
        {
            if (scopeId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "missing_param");
            }

            if (effectiveUserId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "user_id_required");
            }

            if (!await EnsureGroupOwnerAsync(scopeId, effectiveUserId))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_param");
            }
        }

        var lookupScopeName = string.IsNullOrWhiteSpace(request.OldScopeName)
            ? scopeName
            : NormalizeScopeName(request.OldScopeName);
        var lookupScopeId = request.OldScopeId.GetValueOrDefault(scopeId);

        var q = _db.Updateable<Config>().Where(c => c.Name == name && c.Type == ConfigType && c.ScopeId == lookupScopeId);
        if (lookupScopeName == ScopeGlobal)
        {
            q = q.Where(c => c.ScopeName == ScopeGlobal || c.ScopeName == ScopeUser);
        }
        else
        {
            q = q.Where(c => c.ScopeName == lookupScopeName);
        }

        await q.SetColumns(c => new Config
            {
                Value = request.Value ?? string.Empty,
                Enable = true,
                UpdateAt = DateTime.Now,
                ScopeName = scopeName,
                ScopeId = (int)scopeId
            })
            .ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(
        string name,
        string? scopeName,
        long? scopeId,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        name = (name ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(name))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "name_required");
        }

        var effectiveUserId = userId ?? 0;
        if (!isAdmin && effectiveUserId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "user_id_required");
        }

        var finalScopeName = NormalizeScopeName(scopeName);
        var finalScopeId = scopeId.GetValueOrDefault();
        if (!isAdmin && finalScopeName == ScopeGlobal && finalScopeId == 0 && effectiveUserId > 0)
        {
            finalScopeId = effectiveUserId;
        }

        if (finalScopeName == ScopeGroup)
        {
            if (finalScopeId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "missing_param");
            }

            if (effectiveUserId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "user_id_required");
            }

            if (!await EnsureGroupOwnerAsync(finalScopeId, effectiveUserId))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_param");
            }
        }

        var q = _db.Deleteable<Config>().Where(c => c.Name == name && c.Type == ConfigType && c.ScopeId == finalScopeId);
        if (finalScopeName == ScopeGlobal)
        {
            q = q.Where(c => c.ScopeName == ScopeGlobal || c.ScopeName == ScopeUser);
        }
        else
        {
            q = q.Where(c => c.ScopeName == finalScopeName);
        }

        await q.ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    private static long ResolveUserId(long? requestUserId, long? userId, bool isAdmin)
    {
        if (!isAdmin)
        {
            return userId ?? 0;
        }

        if (requestUserId is > 0)
        {
            return requestUserId.Value;
        }

        return userId ?? 0;
    }

    private static string NormalizeScopeName(string? scopeName)
    {
        var name = (scopeName ?? string.Empty).Trim();
        return string.IsNullOrWhiteSpace(name) ? ScopeGlobal : name;
    }

    private static string NormalizeOutputScopeName(string? scopeName)
    {
        var name = (scopeName ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(name) || name.Equals(ScopeUser, StringComparison.OrdinalIgnoreCase))
        {
            return ScopeGlobal;
        }

        return name;
    }

    private static string ConvertValue(object? value)
    {
        if (value == null)
        {
            return string.Empty;
        }

        return value switch
        {
            string s => s,
            bool b => b ? "1" : "0",
            double d => d.ToString("G"),
            float f => f.ToString("G"),
            decimal m => m.ToString("G"),
            int i => i.ToString(),
            long l => l.ToString(),
            _ => value.ToString() ?? string.Empty
        };
    }

    private async Task UpsertConfigAsync(string name, string value, string scopeName, long scopeId)
    {
        var now = DateTime.Now;
        var scopeIdValue = (int)scopeId;
        var q = _db.Queryable<Config>()
            .Where(c => c.Name == name && c.Type == ConfigType && c.ScopeId == scopeIdValue);

        if (scopeName == ScopeGlobal)
        {
            q = q.Where(c => c.ScopeName == ScopeGlobal || c.ScopeName == ScopeUser);
        }
        else
        {
            q = q.Where(c => c.ScopeName == scopeName);
        }

        var existingItems = await q.ToListAsync();
        if (existingItems.Count > 0)
        {
            var targetItem = existingItems.FirstOrDefault(c =>
                    string.Equals(c.ScopeName, scopeName, StringComparison.OrdinalIgnoreCase))
                ?? existingItems[0];

            var update = _db.Updateable<Config>()
                .SetColumns(c => new Config
                {
                    Value = value,
                    Enable = true,
                    ScopeName = scopeName,
                    ScopeId = scopeIdValue,
                    UpdateAt = now
                })
                .Where(c => c.Name == name &&
                            c.Type == ConfigType &&
                            c.ScopeId == scopeIdValue &&
                            c.ScopeName == targetItem.ScopeName);

            await update.ExecuteCommandAsync();

            if (scopeName == ScopeGlobal && existingItems.Count > 1)
            {
                foreach (var stale in existingItems)
                {
                    if (string.Equals(stale.ScopeName, scopeName, StringComparison.OrdinalIgnoreCase))
                    {
                        continue;
                    }

                    await _db.Deleteable<Config>()
                        .Where(c => c.Name == name &&
                                    c.Type == ConfigType &&
                                    c.ScopeId == scopeIdValue &&
                                    c.ScopeName == stale.ScopeName)
                        .ExecuteCommandAsync();
                }
            }

            return;
        }

        var record = new Config
        {
            Name = name,
            Value = value,
            Type = ConfigType,
            ScopeId = scopeIdValue,
            ScopeName = scopeName,
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        };
        await _db.Insertable(record).ExecuteCommandAsync();
    }

    private async Task<bool> EnsureGroupOwnerAsync(long groupId, long userId)
    {
        if (groupId <= 0 || userId <= 0)
        {
            return false;
        }

        return await _db.Queryable<SiteGroup>()
            .Where(g => g.Id == groupId && g.Uid == (int)userId)
            .AnyAsync();
    }


    private sealed class SiteGroupMeta
    {
        public long UserId { get; set; }
        public string? Name { get; set; }
    }

    private async Task<Dictionary<long, SiteGroupMeta>> LoadGroupMapAsync(IReadOnlyList<Config> items)
    {
        var groupIds = new HashSet<long>();
        foreach (var item in items)
        {
            if (item.ScopeName == ScopeGroup && item.ScopeId is > 0)
            {
                groupIds.Add(item.ScopeId.Value);
            }
        }

        return await LoadGroupMapByIdsAsync(groupIds.ToList());
    }

    private async Task<Dictionary<long, SiteGroupMeta>> LoadGroupMapByIdsAsync(IReadOnlyList<long> groupIds)
    {
        var result = new Dictionary<long, SiteGroupMeta>();
        if (groupIds.Count == 0)
        {
            return result;
        }

        var groups = await _db.Queryable<SiteGroup>().Where(g => groupIds.Contains(g.Id)).ToListAsync();
        foreach (var group in groups)
        {
            result[group.Id] = new SiteGroupMeta
            {
                UserId = group.Uid ?? 0,
                Name = group.Name
            };
        }

        return result;
    }

    private async Task<Dictionary<long, string>> LoadUserNameMapAsync(
        IReadOnlyList<Config> items,
        IReadOnlyDictionary<long, SiteGroupMeta> groupMap)
    {
        var userIds = new HashSet<long>();
        foreach (var item in items)
        {
            long ownerId = 0;
            if (item.ScopeName == ScopeGlobal || item.ScopeName == ScopeUser)
            {
                ownerId = item.ScopeId ?? 0;
            }
            else if (item.ScopeName == ScopeGroup)
            {
                if (groupMap.TryGetValue(item.ScopeId ?? 0, out var group))
                {
                    ownerId = group.UserId;
                }
            }

            if (ownerId > 0)
            {
                userIds.Add(ownerId);
            }
        }

        if (userIds.Count == 0)
        {
            return new Dictionary<long, string>();
        }

        var users = await _db.Queryable<User>().Where(u => userIds.Contains(u.Id)).ToListAsync();
        return users.ToDictionary(u => (long)u.Id, u => u.Name ?? string.Empty);
    }

}
