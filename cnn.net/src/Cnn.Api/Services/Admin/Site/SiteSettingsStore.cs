using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Domain.Entities;
using Cnn.Infrastructure.Db;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed class SiteSettingsStore : ISiteSettingsStore
{
    // ── Storage protocol constants ────────────────────────────────────────
    private const string SettingsType  = "site_settings";
    private const string SettingsScope = "site";
    private const string SettingsName  = "settings";

    private const string MetaType  = "site_meta";
    private const string MetaScope = "site";
    private const string MetaName  = "site_type";

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    private readonly ISqlSugarClient _db;

    public SiteSettingsStore(ISqlSugarClient db)
    {
        _db = db;
    }

    // ── Settings ──────────────────────────────────────────────────────────

    public async Task<Dictionary<string, object?>> LoadSettingsAsync(long siteId, CancellationToken cancellationToken = default)
    {
        var map = await LoadSettingsMapAsync(new[] { siteId }, cancellationToken);
        return map.TryGetValue(siteId, out var settings) ? settings : new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
    }

    public async Task<Dictionary<long, Dictionary<string, object?>>> LoadSettingsMapAsync(
        IReadOnlyList<long> siteIds,
        CancellationToken cancellationToken = default)
    {
        var result = new Dictionary<long, Dictionary<string, object?>>();
        if (siteIds == null || siteIds.Count == 0) return result;

        var ids = siteIds.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0) return result;

        var rows = await _db.Queryable<Config>()
            .Where(c => c.Type == SettingsType && c.ScopeName == SettingsScope && c.ScopeId.HasValue && ids.Contains(c.ScopeId.Value))
            .ToListAsync();

        if (rows.Count == 0) return result;

        foreach (var group in rows.GroupBy(r => r.ScopeId!.Value))
        {
            // Prefer the canonical name, fall back to newest row
            var candidate = group
                .Where(r => string.Equals(r.Name, SettingsName, StringComparison.OrdinalIgnoreCase) ||
                            string.Equals(r.Name, SettingsType, StringComparison.OrdinalIgnoreCase))
                .OrderByDescending(r => r.UpdateAt ?? r.CreateAt)
                .FirstOrDefault();

            candidate ??= group.OrderByDescending(r => r.UpdateAt ?? r.CreateAt).FirstOrDefault();
            if (candidate == null || string.IsNullOrWhiteSpace(candidate.Value)) continue;

            var settings = DeserializeSettings(candidate.Value);
            if (settings.Count > 0)
            {
                result[group.Key] = settings;
            }
        }

        return result;
    }

    public async Task SaveSettingsAsync(long siteId, Dictionary<string, object?> settings, CancellationToken cancellationToken = default)
    {
        var raw = JsonSerializer.Serialize(settings, JsonOptions);
        var now = DateTime.Now;

        var existing = await _db.Queryable<Config>()
            .Where(c => c.Type == SettingsType && c.ScopeName == SettingsScope && c.ScopeId == siteId && c.Name == SettingsName)
            .FirstAsync();

        if (existing == null)
        {
            await _db.Insertable(new Config
            {
                Name     = SettingsName,
                Type     = SettingsType,
                ScopeName = SettingsScope,
                ScopeId  = (int)siteId,
                Value    = raw,
                Enable   = true,
                CreateAt = now,
                UpdateAt = now
            }).ExecuteCommandAsync();
        }
        else
        {
            await _db.Updateable<Config>()
                .SetColumns(c => new Config { Value = raw, UpdateAt = now })
                .Where(c => c.Type == SettingsType && c.ScopeName == SettingsScope && c.ScopeId == siteId && c.Name == SettingsName)
                .ExecuteCommandAsync();
        }
    }

    // ── Site type meta ────────────────────────────────────────────────────

    public async Task<Dictionary<long, string>> LoadSiteTypeMapAsync(
        IReadOnlyList<long> siteIds,
        CancellationToken cancellationToken = default)
    {
        var result = new Dictionary<long, string>();
        if (siteIds == null || siteIds.Count == 0) return result;

        var ids = siteIds.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0) return result;

        var rows = await _db.Queryable<Config>()
            .Where(c => c.Type == MetaType && c.ScopeName == MetaScope && c.ScopeId.HasValue && ids.Contains(c.ScopeId.Value))
            .ToListAsync();

        foreach (var row in rows)
        {
            if (row.ScopeId is null or <= 0) continue;

            if (!string.Equals(row.Name, MetaName, StringComparison.OrdinalIgnoreCase) &&
                !string.Equals(row.Type, MetaType, StringComparison.OrdinalIgnoreCase)) continue;

            var value = row.Value?.Trim() ?? string.Empty;
            if (string.IsNullOrWhiteSpace(value)) continue;

            result[row.ScopeId.Value] = value;
        }

        return result;
    }

    public async Task SaveSiteTypeAsync(long siteId, string siteType, CancellationToken cancellationToken = default)
    {
        if (string.IsNullOrWhiteSpace(siteType)) return;

        var now = DateTime.Now;
        var existing = await _db.Queryable<Config>()
            .Where(c => c.Type == MetaType && c.ScopeName == MetaScope && c.ScopeId == siteId && c.Name == MetaName)
            .FirstAsync();

        if (existing == null)
        {
            await _db.Insertable(new Config
            {
                Name      = MetaName,
                Type      = MetaType,
                ScopeName = MetaScope,
                ScopeId   = (int)siteId,
                Value     = siteType.Trim(),
                Enable    = true,
                CreateAt  = now,
                UpdateAt  = now
            }).ExecuteCommandAsync();
        }
        else
        {
            await _db.Updateable<Config>()
                .SetColumns(c => new Config { Value = siteType.Trim(), UpdateAt = now })
                .Where(c => c.Type == MetaType && c.ScopeName == MetaScope && c.ScopeId == siteId && c.Name == MetaName)
                .ExecuteCommandAsync();
        }
    }

    // ── Private helpers ───────────────────────────────────────────────────

    private static Dictionary<string, object?> DeserializeSettings(string raw)
    {
        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
                return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);

            var result = JsonSerializer.Deserialize<Dictionary<string, object?>>(raw, JsonOptions);
            return result ?? new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        }
        catch
        {
            return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        }
    }
}
