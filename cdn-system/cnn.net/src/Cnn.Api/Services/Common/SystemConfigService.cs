using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public interface ISystemConfigService
{
    Task<Dictionary<string, string>> LoadSystemConfigAsync(CancellationToken cancellationToken);
    bool ParseBoolFlag(string? raw);
}

public sealed class SystemConfigService : ISystemConfigService
{
    private const string SystemType = "system";
    private const string GlobalScopeName = "global";
    private const int GlobalScopeId = 0;
    private static readonly (string Legacy, string Canonical)[] LegacyAliases =
    {
        ("clean_cache_days", "keep-task-log-days"),
        ("clean_login_log_days", "keep-login-log-days"),
        ("clean_op_log_days", "keep-op-log-days"),
        ("clean_site_log_days", "keep-access-log-days"),
        ("clean_access_download_log_days", "keep-access-download-log-days"),
        ("clean_node_monitor_days", "keep-node-log-days"),
        ("clean_traffic_days", "keep-traffic-history-days"),
        ("clean_node_traffic_days", "keep-node-traffic-days"),
        ("clean_blacklist_days", "keep-blacklist-days"),
        ("backup_frequency", "backup_rate"),
        ("backup_retention", "backup_keep_days")
    };

    private readonly ISqlSugarClient _db;

    public SystemConfigService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<Dictionary<string, string>> LoadSystemConfigAsync(CancellationToken cancellationToken)
    {
        var rows = await _db.Queryable<Config>()
            .Where(c => c.Type == SystemType && c.ScopeName == GlobalScopeName && c.ScopeId == GlobalScopeId)
            .ToListAsync();

        var map = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row.Name))
            {
                continue;
            }

            map[row.Name] = row.Value ?? string.Empty;
        }

        ApplyLegacyAliases(map);
        return map;
    }

    public bool ParseBoolFlag(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        var normalized = raw.Trim().ToLowerInvariant();
        return normalized is "1" or "true" or "yes" or "on";
    }

    private static void ApplyLegacyAliases(Dictionary<string, string> map)
    {
        foreach (var (legacy, canonical) in LegacyAliases)
        {
            if (string.IsNullOrWhiteSpace(legacy) || string.IsNullOrWhiteSpace(canonical))
            {
                continue;
            }

            if (map.TryGetValue(canonical, out var current) && !string.IsNullOrWhiteSpace(current))
            {
                continue;
            }

            if (map.TryGetValue(legacy, out var legacyValue) && !string.IsNullOrWhiteSpace(legacyValue))
            {
                map[canonical] = legacyValue;
            }
        }
    }
}
