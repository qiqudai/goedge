using System.Text.Json;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public sealed class RegionMetaService
{
    private const string RegionMetaKey = "region_meta";

    private readonly ISqlSugarClient _db;

    public RegionMetaService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<Dictionary<string, RegionMeta>> LoadAsync()
    {
        var item = await _db.Queryable<Config>()
            .Where(c => c.Name == RegionMetaKey && c.Type == "system" && c.ScopeName == "global" && c.ScopeId == 0)
            .FirstAsync();
        if (item == null || string.IsNullOrWhiteSpace(item.Value))
        {
            return new Dictionary<string, RegionMeta>();
        }

        try
        {
            var map = JsonSerializer.Deserialize<Dictionary<string, RegionMeta>>(item.Value);
            return map ?? new Dictionary<string, RegionMeta>();
        }
        catch
        {
            return new Dictionary<string, RegionMeta>();
        }
    }

    public async Task<bool> SaveAsync(Dictionary<string, RegionMeta> metaMap)
    {
        var json = JsonSerializer.Serialize(metaMap);
        var now = DateTime.Now;
        var existing = await _db.Queryable<Config>()
            .Where(c => c.Name == RegionMetaKey && c.Type == "system" && c.ScopeName == "global" && c.ScopeId == 0)
            .FirstAsync();

        if (existing == null)
        {
            var created = new Config
            {
                Name = RegionMetaKey,
                Value = json,
                Type = "system",
                ScopeId = 0,
                ScopeName = "global",
                Enable = true,
                CreateAt = now,
                UpdateAt = now
            };
            return await _db.Insertable(created).ExecuteCommandAsync() > 0;
        }

        return await _db.Updateable<Config>()
            .SetColumns(c => new Config { Value = json, UpdateAt = now })
            .Where(c => c.Name == RegionMetaKey && c.Type == "system" && c.ScopeName == "global" && c.ScopeId == 0)
            .ExecuteCommandAsync() > 0;
    }

    public static int ResolveL2CheckPort(Dictionary<string, RegionMeta> metaMap, long? regionId)
    {
        if (regionId == null || regionId <= 0)
        {
            return 80;
        }

        var key = regionId.Value.ToString();
        if (metaMap.TryGetValue(key, out var meta) && meta.L2CheckPort > 0)
        {
            return meta.L2CheckPort;
        }

        return 80;
    }

    public static int ResolveSortOrder(Dictionary<string, RegionMeta> metaMap, long? regionId)
    {
        if (regionId == null || regionId <= 0)
        {
            return 100;
        }

        var key = regionId.Value.ToString();
        if (metaMap.TryGetValue(key, out var meta) && meta.SortOrder > 0)
        {
            return meta.SortOrder;
        }

        return 100;
    }
}

public sealed class RegionMeta
{
    public int L2CheckPort { get; set; }

    public int SortOrder { get; set; }
}
