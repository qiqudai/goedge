using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed class ForwardDefaultService : IForwardDefaultService
{
    private const string ForwardDefaultKey = "forward_default_settings";
    private const string SystemType = "system";
    private const string DefaultScope = "global";
    private const string StreamDefaultType = "stream_default_config";

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public ForwardDefaultService(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public async Task<ServiceResult<ForwardDefaultListResult>> ListAsync(long? userId, bool isAdmin, CancellationToken cancellationToken)
    {
        if (!isAdmin || userId is > 0)
        {
            var targetUserId = userId ?? 0;
            if (targetUserId <= 0)
            {
                return ServiceResult<ForwardDefaultListResult>.Fail(ErrorCodes.PermissionDenied);
            }

            var items = await LoadUserDefaultItemsAsync(targetUserId);
            return ServiceResult<ForwardDefaultListResult>.Ok(new ForwardDefaultListResult(items));
        }

        var entities = await LoadAdminDefaultItemEntitiesAsync();
        var list = entities.Select(item => item.ToDto()).ToList();
        if (list.Count > 0)
        {
            var groupMap = await LoadForwardGroupMapAsync(list);
            foreach (var item in list)
            {
                item.IdStr = item.Id.ToString();
                if (item.GroupId != 0 && groupMap.TryGetValue(item.GroupId, out var name))
                {
                    item.GroupName = name;
                }
            }
        }

        return ServiceResult<ForwardDefaultListResult>.Ok(new ForwardDefaultListResult(list));
    }

    public async Task<ServiceResult<bool>> CreateAsync(ForwardDefaultCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(request.Key))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "forward_default_key_required");
        }

        if (!isAdmin || userId is > 0)
        {
            var targetUserId = userId ?? 0;
            if (targetUserId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }

            var value = EncodeForwardDefaultValue(request.Value);
            await UpsertUserConfigItemAsync(targetUserId, request.Key.Trim(), value);
            await _configVersionService.BumpAsync("config_item", new[] { targetUserId }, cancellationToken);
            return ServiceResult<bool>.Ok(true);
        }

        var items = await LoadAdminDefaultItemEntitiesAsync();
        items.Add(new ForwardDefaultItem
        {
            Id = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds(),
            Key = request.Key.Trim(),
            Value = request.Value?.ValueKind == JsonValueKind.Undefined ? null : request.Value,
            Scope = request.Scope?.Trim(),
            GroupId = request.GroupId
        });

        await SaveAdminDefaultItemsAsync(items);
        await _configVersionService.BumpAsync("forward_default", Array.Empty<long>(), cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(ForwardDefaultDeleteRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken)
    {
        if (!isAdmin || userId is > 0)
        {
            var targetUserId = userId ?? 0;
            if (targetUserId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }

            var key = request.IdStr?.Trim();
            if (string.IsNullOrWhiteSpace(key))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "forward_default_id_required");
            }

            await _db.Deleteable<Config>()
                .Where(c => c.Type == StreamDefaultType && c.ScopeName == "user" && c.ScopeId == (int)targetUserId && c.Name == key)
                .ExecuteCommandAsync();
            await _configVersionService.BumpAsync("config_item", new[] { targetUserId }, cancellationToken);
            return ServiceResult<bool>.Ok(true);
        }

        var id = request.Id;
        if (id == 0 && !string.IsNullOrWhiteSpace(request.IdStr) && long.TryParse(request.IdStr, out var parsed))
        {
            id = parsed;
        }

        if (id == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "forward_default_id_required");
        }

        var items = await LoadAdminDefaultItemEntitiesAsync();
        var next = items.Where(item => item.Id != id).ToList();
        await SaveAdminDefaultItemsAsync(next);
        await _configVersionService.BumpAsync("forward_default", Array.Empty<long>(), cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private async Task<List<ForwardDefaultItem>> LoadAdminDefaultItemEntitiesAsync()
    {
        var cfg = await _db.Queryable<Config>()
            .Where(c => c.Name == ForwardDefaultKey && c.Type == SystemType)
            .FirstAsync();
        if (cfg == null || string.IsNullOrWhiteSpace(cfg.Value))
        {
            return new List<ForwardDefaultItem>();
        }

        try
        {
            var items = JsonSerializer.Deserialize<List<ForwardDefaultItem>>(cfg.Value, JsonOptions);
            return items ?? new List<ForwardDefaultItem>();
        }
        catch
        {
            return new List<ForwardDefaultItem>();
        }
    }

    private async Task SaveAdminDefaultItemsAsync(List<ForwardDefaultItem> items)
    {
        var payload = JsonSerializer.Serialize(items, JsonOptions);
        var existing = await _db.Queryable<Config>()
            .Where(c => c.Name == ForwardDefaultKey && c.Type == SystemType)
            .FirstAsync();

        var now = DateTime.Now;
        if (existing == null)
        {
            var record = new Config
            {
                Name = ForwardDefaultKey,
                Type = SystemType,
                ScopeId = 0,
                ScopeName = DefaultScope,
                Value = payload,
                Enable = true,
                CreateAt = now,
                UpdateAt = now
            };
            await _db.Insertable(record).ExecuteCommandAsync();
            return;
        }

        await _db.Updateable<Config>()
            .SetColumns(c => new Config { Value = payload, UpdateAt = now, Enable = true })
            .Where(c => c.Name == ForwardDefaultKey && c.Type == SystemType)
            .ExecuteCommandAsync();
    }

    private async Task<List<ForwardDefaultItemDto>> LoadUserDefaultItemsAsync(long userId)
    {
        var items = await _db.Queryable<Config>()
            .Where(c => c.Type == StreamDefaultType && c.ScopeName == "user" && c.ScopeId == (int)userId)
            .ToListAsync();

        var result = new List<ForwardDefaultItemDto>();
        foreach (var item in items)
        {
            if (string.IsNullOrWhiteSpace(item.Name))
            {
                continue;
            }
            result.Add(new ForwardDefaultItemDto
            {
                Id = 0,
                IdStr = item.Name,
                Key = item.Name,
                Value = ParseForwardDefaultValue(item.Name, item.Value ?? string.Empty),
                Scope = DefaultScope,
                GroupId = 0
            });
        }
        return result;
    }

    private async Task<Dictionary<long, string>> LoadForwardGroupMapAsync(IReadOnlyList<ForwardDefaultItemDto> items)
    {
        var ids = items.Select(item => item.GroupId).Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return new Dictionary<long, string>();
        }

        var groups = await _db.Queryable<StreamGroup>().Where(g => ids.Contains(g.Id)).ToListAsync();
        var map = new Dictionary<long, string>();
        foreach (var group in groups)
        {
            map[group.Id] = group.Name ?? string.Empty;
        }
        return map;
    }

    private async Task UpsertUserConfigItemAsync(long userId, string key, string value)
    {
        var existing = await _db.Queryable<Config>()
            .Where(c => c.Type == StreamDefaultType && c.ScopeName == "user" && c.ScopeId == (int)userId && c.Name == key)
            .FirstAsync();

        var now = DateTime.Now;
        if (existing == null)
        {
            var record = new Config
            {
                Name = key,
                Value = value,
                Type = StreamDefaultType,
                ScopeName = "user",
                ScopeId = (int)userId,
                Enable = true,
                CreateAt = now,
                UpdateAt = now
            };
            await _db.Insertable(record).ExecuteCommandAsync();
            return;
        }

        await _db.Updateable<Config>()
            .SetColumns(c => new Config { Value = value, Enable = true, UpdateAt = now })
            .Where(c => c.Type == StreamDefaultType && c.ScopeName == "user" && c.ScopeId == (int)userId && c.Name == key)
            .ExecuteCommandAsync();
    }

    private static object ParseForwardDefaultValue(string key, string raw)
    {
        var trimmed = raw.Trim();
        switch (key.Trim())
        {
            case "proxy_protocol":
                return ParseBool(trimmed, false);
            case "listen_protocol":
            case "balance_way":
                return trimmed;
            default:
                if (string.IsNullOrWhiteSpace(trimmed))
                {
                    return string.Empty;
                }
                var lower = trimmed.ToLowerInvariant();
                if (lower == "true" || lower == "false")
                {
                    return ParseBool(trimmed, false);
                }
                return trimmed;
        }
    }

    private static string EncodeForwardDefaultValue(JsonElement? value)
    {
        if (value == null || value.Value.ValueKind == JsonValueKind.Undefined)
        {
            return string.Empty;
        }

        var element = value.Value;
        return element.ValueKind switch
        {
            JsonValueKind.String => element.GetString() ?? string.Empty,
            JsonValueKind.True => "true",
            JsonValueKind.False => "false",
            JsonValueKind.Number => EncodeNumber(element),
            JsonValueKind.Null => string.Empty,
            _ => JsonSerializer.Serialize(element, JsonOptions)
        };
    }

    private static string EncodeNumber(JsonElement value)
    {
        if (value.TryGetInt64(out var parsed))
        {
            return parsed.ToString();
        }
        if (value.TryGetDouble(out var dbl))
        {
            return ((long)dbl).ToString();
        }
        return value.ToString();
    }

    private static bool ParseBool(string? raw, bool fallback)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return fallback;
        }
        var normalized = raw.Trim().ToLowerInvariant();
        return normalized is "1" or "true" or "yes" or "on";
    }

    private sealed class ForwardDefaultItem
    {
        [JsonPropertyName("id")]
        public long Id { get; set; }

        [JsonPropertyName("key")]
        public string? Key { get; set; }

        [JsonPropertyName("value")]
        public object? Value { get; set; }

        [JsonPropertyName("scope")]
        public string? Scope { get; set; }

        [JsonPropertyName("group_id")]
        public long GroupId { get; set; }

        public ForwardDefaultItemDto ToDto()
        {
            return new ForwardDefaultItemDto
            {
                Id = Id,
                Key = Key,
                Value = Value,
                Scope = Scope,
                GroupId = GroupId
            };
        }
    }
}
