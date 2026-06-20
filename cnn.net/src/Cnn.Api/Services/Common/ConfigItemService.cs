using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface IConfigItemService
{
    Task<ServiceResult<IReadOnlyList<ConfigItemDto>>> ListAsync(string? type, string? scopeName, int? scopeId, CancellationToken cancellationToken);
    Task<ServiceResult<IReadOnlyList<ConfigItemDto>>> ListUserAsync(long userId, string? type, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpsertAsync(ConfigItemUpsertRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpsertUserAsync(long userId, ConfigItemUpsertRequest request, CancellationToken cancellationToken);
}

public sealed class ConfigItemService : IConfigItemService
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public ConfigItemService(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public async Task<ServiceResult<IReadOnlyList<ConfigItemDto>>> ListAsync(
        string? type,
        string? scopeName,
        int? scopeId,
        CancellationToken cancellationToken)
    {
        var query = _db.Queryable<Config>();

        if (!string.IsNullOrWhiteSpace(type))
        {
            query = query.Where(c => c.Type == type);
        }

        if (!string.IsNullOrWhiteSpace(scopeName) && scopeId.HasValue)
        {
            query = query.Where(c => c.ScopeName == scopeName && c.ScopeId == scopeId.Value);
        }

        var list = await query.OrderBy("name asc").ToListAsync();
        var result = list.Select(MapItem).ToList();
        return ServiceResult<IReadOnlyList<ConfigItemDto>>.Ok(result);
    }

    public async Task<ServiceResult<IReadOnlyList<ConfigItemDto>>> ListUserAsync(
        long userId,
        string? type,
        CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<IReadOnlyList<ConfigItemDto>>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var query = _db.Queryable<Config>()
            .Where(c => c.ScopeName == "user" && c.ScopeId == (int)userId);

        if (!string.IsNullOrWhiteSpace(type))
        {
            query = query.Where(c => c.Type == type);
        }

        var list = await query.OrderBy("name asc").ToListAsync();
        var result = list.Select(MapItem).ToList();
        return ServiceResult<IReadOnlyList<ConfigItemDto>>.Ok(result);
    }

    public async Task<ServiceResult<bool>> UpsertAsync(ConfigItemUpsertRequest request, CancellationToken cancellationToken)
    {
        if (request == null || string.IsNullOrWhiteSpace(request.Type) || string.IsNullOrWhiteSpace(request.ScopeName))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var items = NormalizeItems(request.Items);
        if (items.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var scopeId = request.ScopeId.GetValueOrDefault();
        await UpsertInternalAsync(request.Type.Trim(), request.ScopeName.Trim(), scopeId, items);
        await _configVersionService.BumpAsync("config_item", Array.Empty<long>(), cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpsertUserAsync(long userId, ConfigItemUpsertRequest request, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        if (request == null || string.IsNullOrWhiteSpace(request.Type))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var items = NormalizeItems(request.Items);
        if (items.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        await UpsertInternalAsync(request.Type.Trim(), "user", (int)userId, items);
        await _configVersionService.BumpAsync("config_item", new[] { userId }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private async Task UpsertInternalAsync(
        string type,
        string scopeName,
        int scopeId,
        IReadOnlyList<ConfigItemPayloadDto> items)
    {
        var now = DateTime.Now;
        _db.Ado.BeginTran();
        try
        {
            foreach (var item in items)
            {
                var name = item.Name?.Trim();
                if (string.IsNullOrWhiteSpace(name))
                {
                    continue;
                }

                var enable = item.Enable ?? true;
                var value = item.Value ?? string.Empty;

                var existing = await _db.Queryable<Config>()
                    .Where(c => c.Type == type && c.ScopeName == scopeName && c.ScopeId == scopeId && c.Name == name)
                    .FirstAsync();

                if (existing == null)
                {
                    var created = new Config
                    {
                        Name = name,
                        Value = value,
                        Type = type,
                        ScopeName = scopeName,
                        ScopeId = scopeId,
                        Enable = enable,
                        CreateAt = now,
                        UpdateAt = now
                    };
                    await _db.Insertable(created).ExecuteCommandAsync();
                }
                else
                {
                    await _db.Updateable<Config>()
                        .SetColumns(c => new Config
                        {
                            Value = value,
                            Enable = enable,
                            UpdateAt = now
                        })
                        .Where(c => c.Type == type && c.ScopeName == scopeName && c.ScopeId == scopeId && c.Name == name)
                        .ExecuteCommandAsync();
                }
            }

            _db.Ado.CommitTran();
        }
        catch
        {
            _db.Ado.RollbackTran();
            throw;
        }
    }

    private static List<ConfigItemPayloadDto> NormalizeItems(IEnumerable<ConfigItemPayloadDto>? items)
    {
        if (items == null)
        {
            return new List<ConfigItemPayloadDto>();
        }

        return items
            .Where(item => item != null && !string.IsNullOrWhiteSpace(item.Name))
            .Select(item => new ConfigItemPayloadDto
            {
                Name = item.Name?.Trim(),
                Value = item.Value,
                Enable = item.Enable
            })
            .ToList();
    }

    private static ConfigItemDto MapItem(Config item)
    {
        return new ConfigItemDto
        {
            Name = item.Name,
            Value = item.Value,
            Type = item.Type,
            ScopeName = item.ScopeName,
            ScopeId = item.ScopeId,
            Enable = item.Enable
        };
    }
}
