using Cnn.Common.Contracts;
using Cnn.Api.Cache;
using Cnn.Infrastructure.Db;
using SqlSugar;
using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Agent;

namespace Cnn.Api.Services.Admin;

public sealed class SiteCacheApplicationService : ISiteCacheApplicationService
{
    private readonly ISqlSugarClient _db;
    private readonly JsonSerializerOptions _jsonOptions;
    private readonly IAgentConnectionManager _connections;

    public SiteCacheApplicationService(ISqlSugarClient db, IAgentConnectionManager connections)
    {
        _db = db;
        _connections = connections;
        _jsonOptions = new JsonSerializerOptions(JsonSerializerDefaults.Web)
        {
            PropertyNameCaseInsensitive = true,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        };
    }

    public async Task<ServiceResult<SiteCacheViewDto>> GetAsync(int siteId, CancellationToken cancellationToken)
    {
        var site = await _db.Queryable<Cnn.Domain.Entities.Site>().Where(s => s.Id == siteId).FirstAsync();
        if (site == null)
        {
            return ServiceResult<SiteCacheViewDto>.Fail(ErrorCodes.NotFound);
        }

        var settings = await LoadSiteSettingsAsync(siteId);
        var cacheResult = TryLoadCacheFromSettings(settings);
        var raw = cacheResult.Raw ?? site.ProxyCache;
        var config = cacheResult.Config ?? CacheConfigCompiler.DeserializeConfig(site.ProxyCache);

        return ServiceResult<SiteCacheViewDto>.Ok(new SiteCacheViewDto
        {
            Raw = raw,
            Config = config
        });
    }

    public async Task<ServiceResult<SiteCacheSaveResultDto>> SaveAsync(int siteId, CacheConfigDto input, bool compile, CancellationToken cancellationToken)
    {
        var site = await _db.Queryable<Cnn.Domain.Entities.Site>().Where(s => s.Id == siteId).FirstAsync();
        if (site == null)
        {
            return ServiceResult<SiteCacheSaveResultDto>.Fail(ErrorCodes.NotFound);
        }

        var raw = await SaveSiteCacheAsync(siteId, input);

        CacheSiteConfigDto? compiled = null;
        if (compile)
        {
            compiled = CacheConfigCompiler.Compile(site, input);
            await SaveCompiledAsync(compiled);
            await BroadcastCacheConfigAsync(compiled);
        }

        return ServiceResult<SiteCacheSaveResultDto>.Ok(new SiteCacheSaveResultDto
        {
            Raw = raw,
            Compiled = compiled
        });
    }

    public async Task<ServiceResult<CacheSiteConfigDto>> CompileAsync(int siteId, CancellationToken cancellationToken)
    {
        var site = await _db.Queryable<Cnn.Domain.Entities.Site>().Where(s => s.Id == siteId).FirstAsync();
        if (site == null)
        {
            return ServiceResult<CacheSiteConfigDto>.Fail(ErrorCodes.NotFound);
        }

        var settings = await LoadSiteSettingsAsync(siteId);
        var cacheResult = TryLoadCacheFromSettings(settings);
        var config = cacheResult.Config ?? CacheConfigCompiler.DeserializeConfig(site.ProxyCache);
        var compiled = CacheConfigCompiler.Compile(site, config);
        
        await SaveCompiledAsync(compiled);
        await BroadcastCacheConfigAsync(compiled);

        return ServiceResult<CacheSiteConfigDto>.Ok(compiled);
    }

    private async Task SaveCompiledAsync(CacheSiteConfigDto compiled)
    {
        var data = CacheConfigCompiler.Serialize(compiled);
        var md5 = CacheConfigCompiler.ComputeMd5(data);
        await _db.Ado.BeginTranAsync();
        try
        {
            await _db.Deleteable<Cnn.Domain.Entities.SiteConfCache>()
                .Where(c => c.SiteId == compiled.SiteId)
                .ExecuteCommandAsync();

            await _db.Insertable(new Cnn.Domain.Entities.SiteConfCache
            {
                SiteId = compiled.SiteId,
                Version = compiled.Version,
                Data = data,
                TemplMd5 = md5
            }).ExecuteCommandAsync();

            await _db.Ado.CommitTranAsync();
        }
        catch
        {
            await _db.Ado.RollbackTranAsync();
            throw;
        }
    }

    private async Task<Dictionary<string, JsonElement>?> LoadSiteSettingsAsync(int siteId)
    {
        var rows = await _db.Queryable<Cnn.Domain.Entities.Config>()
            .Where(c => c.Type == SettingsConstants.SiteSettingsType && c.ScopeName == SettingsConstants.SiteSettingsScope && c.ScopeId == siteId)
            .ToListAsync();

        if (rows.Count == 0) return null;

        var selected = rows
            .Where(r => string.Equals(r.Name, SettingsConstants.SiteSettingsName, StringComparison.OrdinalIgnoreCase) ||
                        string.Equals(r.Name, SettingsConstants.SiteSettingsType, StringComparison.OrdinalIgnoreCase))
            .OrderByDescending(r => r.UpdateAt ?? r.CreateAt)
            .FirstOrDefault();

        selected ??= rows.OrderByDescending(r => r.UpdateAt ?? r.CreateAt).FirstOrDefault();
        if (selected == null || string.IsNullOrWhiteSpace(selected.Value)) return null;

        try
        {
            using var doc = JsonDocument.Parse(selected.Value);
            if (doc.RootElement.ValueKind != JsonValueKind.Object) return null;

            var map = new Dictionary<string, JsonElement>(StringComparer.OrdinalIgnoreCase);
            foreach (var prop in doc.RootElement.EnumerateObject())
            {
                map[prop.Name] = prop.Value.Clone();
            }

            return map.Count == 0 ? null : map;
        }
        catch
        {
            return null;
        }
    }

    private (CacheConfigDto? Config, string? Raw) TryLoadCacheFromSettings(Dictionary<string, JsonElement>? settings)
    {
        if (settings == null || !TryGetEntry(settings, "cache", out var cacheElement))
        {
            return (null, null);
        }

        if (cacheElement.ValueKind == JsonValueKind.Null || cacheElement.ValueKind == JsonValueKind.Undefined)
        {
            return (null, null);
        }

        var raw = cacheElement.GetRawText();
        try
        {
            var config = JsonSerializer.Deserialize<CacheConfigDto>(raw, _jsonOptions);
            return (config, raw);
        }
        catch
        {
            return (null, raw);
        }
    }

    private async Task<string> SaveSiteCacheAsync(int siteId, CacheConfigDto input)
    {
        var settings = await LoadSiteSettingsAsync(siteId) ?? new Dictionary<string, JsonElement>(StringComparer.OrdinalIgnoreCase);
        var cacheElement = JsonSerializer.SerializeToElement(input, _jsonOptions).Clone();
        settings["cache"] = cacheElement;

        var raw = JsonSerializer.Serialize(input, _jsonOptions);
        var settingsRaw = JsonSerializer.Serialize(settings, _jsonOptions);

        var now = DateTime.Now;
        var existing = await _db.Queryable<Cnn.Domain.Entities.Config>()
            .Where(c => c.Type == SettingsConstants.SiteSettingsType && c.ScopeName == SettingsConstants.SiteSettingsScope && c.ScopeId == siteId && c.Name == SettingsConstants.SiteSettingsName)
            .FirstAsync();

        if (existing == null)
        {
            await _db.Insertable(new Cnn.Domain.Entities.Config
            {
                Name = SettingsConstants.SiteSettingsName,
                Type = SettingsConstants.SiteSettingsType,
                ScopeName = SettingsConstants.SiteSettingsScope,
                ScopeId = siteId,
                Value = settingsRaw,
                Enable = true,
                CreateAt = now,
                UpdateAt = now
            }).ExecuteCommandAsync();
        }
        else
        {
            await _db.Updateable<Cnn.Domain.Entities.Config>()
                .SetColumns(c => new Cnn.Domain.Entities.Config { Value = settingsRaw, UpdateAt = now })
                .Where(c => c.Type == SettingsConstants.SiteSettingsType && c.ScopeName == SettingsConstants.SiteSettingsScope && c.ScopeId == siteId && c.Name == SettingsConstants.SiteSettingsName)
                .ExecuteCommandAsync();
        }

        return raw;
    }

    private bool TryGetEntry(Dictionary<string, JsonElement> entry, string key, out JsonElement value)
    {
        if (entry.TryGetValue(key, out value)) return true;

        foreach (var pair in entry)
        {
            if (string.Equals(pair.Key, key, StringComparison.OrdinalIgnoreCase))
            {
                value = pair.Value;
                return true;
            }
        }

        value = default;
        return false;
    }

    private async Task BroadcastCacheConfigAsync(CacheSiteConfigDto compiled)
    {
        var nodeIds = _connections.GetConnectedNodeIds();
        if (nodeIds.Count == 0) return;

        var payload = new { kind = AgentMessageKinds.CacheConfig, data = compiled };
        foreach (var nodeId in nodeIds)
        {
            if (_connections.TryGetSocket(nodeId, out var socket) && socket.State == System.Net.WebSockets.WebSocketState.Open)
            {
                await _connections.SendAsync(socket, payload, CancellationToken.None);
            }
        }
    }
}
