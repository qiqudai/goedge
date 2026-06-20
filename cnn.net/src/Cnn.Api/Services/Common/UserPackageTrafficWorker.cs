using System.Globalization;
using Cnn.Api.Services.Stats;
using Cnn.Domain.Entities;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public sealed class UserPackageTrafficWorker : BackgroundService
{
    private const int HostChunkSize = 200;
    private static readonly TimeSpan Interval = TimeSpan.FromMinutes(10);

    private readonly IServiceScopeFactory _scopeFactory;
    private readonly ILogger<UserPackageTrafficWorker> _logger;
    private readonly IConfiguration _configuration;

    public UserPackageTrafficWorker(
        IServiceScopeFactory scopeFactory,
        ILogger<UserPackageTrafficWorker> logger,
        IConfiguration configuration)
    {
        _scopeFactory = scopeFactory;
        _logger = logger;
        _configuration = configuration;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await CheckTrafficAsync(stoppingToken);
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "User package traffic worker failed");
            }

            try
            {
                await Task.Delay(Interval, stoppingToken);
            }
            catch (TaskCanceledException)
            {
                return;
            }
        }
    }

    private async Task CheckTrafficAsync(CancellationToken cancellationToken)
    {
        var httpCfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (httpCfg == null)
        {
            return;
        }

        using var scope = _scopeFactory.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<ISqlSugarClient>();
        var systemConfig = scope.ServiceProvider.GetRequiredService<ISystemConfigService>();
        var configVersion = scope.ServiceProvider.GetRequiredService<IConfigVersionService>();

        var cfg = await systemConfig.LoadSystemConfigAsync(cancellationToken);
        if (cfg.TryGetValue("traffic_excceed_close_site", out var enabledRaw) && !systemConfig.ParseBoolFlag(enabledRaw))
        {
            return;
        }

        var trafficFactor = ParseDouble(cfg.TryGetValue("tcp_traffic_factor", out var factorRaw) ? factorRaw : null, 1d);
        if (trafficFactor <= 0)
        {
            trafficFactor = 1d;
        }

        var packages = await db.Queryable<UserPackage>()
            .Where(p => p.Traffic > 0 && (p.IsExpired == null || p.IsExpired == false))
            .ToListAsync();
        if (packages.Count == 0)
        {
            return;
        }

        var now = DateTime.Now;
        var packageIds = packages
            .Where(p => !p.EndAt.HasValue || p.EndAt.Value >= now)
            .Select(p => p.Id)
            .Where(id => id > 0)
            .ToList();
        if (packageIds.Count == 0)
        {
            return;
        }

        var sites = await db.Queryable<Site>()
            .Where(s => s.UserPackage.HasValue && packageIds.Contains(s.UserPackage.Value))
            .ToListAsync();

        var domainMap = new Dictionary<long, HashSet<string>>();
        foreach (var site in sites)
        {
            if (!site.UserPackage.HasValue || site.UserPackage.Value <= 0)
            {
                continue;
            }

            var domains = DomainParser.ParseDomains(site.Domain);
            if (domains.Count == 0)
            {
                continue;
            }

            if (!domainMap.TryGetValue(site.UserPackage.Value, out var set))
            {
                set = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
                domainMap[site.UserPackage.Value] = set;
            }

            foreach (var raw in domains)
            {
                var host = DomainParser.NormalizeDomain(raw);
                if (!string.IsNullOrWhiteSpace(host))
                {
                    set.Add(host);
                }
            }
        }

        foreach (var pkg in packages)
        {
            if (pkg.EndAt.HasValue && pkg.EndAt.Value < now)
            {
                continue;
            }

            if (!domainMap.TryGetValue(pkg.Id, out var hostSet) || hostSet.Count == 0)
            {
                continue;
            }

            var hosts = hostSet.ToList();
            var startAt = pkg.StartAt ?? now.AddDays(-1);
            if (startAt > now)
            {
                startAt = now.AddDays(-1);
            }

            var usedBytes = await SumTrafficBytesByHostsAsync(httpCfg, hosts, startAt, now, cancellationToken);
            if (trafficFactor != 1d)
            {
                usedBytes = (ulong)(usedBytes * trafficFactor);
            }

            var usedGb = usedBytes / (1024d * 1024d * 1024d);
            var limitGb = (double)(pkg.Traffic ?? 0);
            if (limitGb <= 0)
            {
                continue;
            }

            if (usedGb >= limitGb)
            {
                var siteIds = await ApplyTrafficLimitAsync(db, pkg.Id, configVersion, cancellationToken);
                if (siteIds.Count > 0)
                {
                    var userId = pkg.Uid ?? 0;
                    if (userId > 0)
                    {
                        var title = "Traffic limit exceeded";
                        var content = $"Package {pkg.Name} exceeded traffic ({usedGb:F2}GB/{limitGb:F2}GB). {siteIds.Count} site(s) have been limited.";
                        await NotificationHelper.CreateUserMessageAsync(db, systemConfig, userId, "traffic-exceed", title, content, pkg.Id, 0, cancellationToken);
                    }
                }
            }
            else
            {
                await ClearTrafficLimitAsync(db, pkg.Id, configVersion, cancellationToken);
            }
        }
    }

    private static async Task<ulong> SumTrafficBytesByHostsAsync(
        ClickHouseHttpConfig cfg,
        IReadOnlyList<string> hosts,
        DateTime start,
        DateTime end,
        CancellationToken cancellationToken)
    {
        if (hosts.Count == 0)
        {
            return 0;
        }

        ulong total = 0;
        for (var i = 0; i < hosts.Count; i += HostChunkSize)
        {
            var count = Math.Min(HostChunkSize, hosts.Count - i);
            var chunk = hosts.Skip(i).Take(count)
                .Select(ClickHouseHttpHelper.QuoteString)
                .ToList();
            if (chunk.Count == 0)
            {
                continue;
            }

            var query = $"SELECT sum(\"bytes\") FROM node_access_logs WHERE ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}') " +
                        $"AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}') AND host IN ({string.Join(",", chunk)})";
            var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
            if (rows == null || rows.Length == 0)
            {
                continue;
            }

            if (ulong.TryParse(rows[0].Trim(), out var sum))
            {
                total += sum;
            }
        }

        return total;
    }

    private static async Task<List<long>> ApplyTrafficLimitAsync(
        ISqlSugarClient db,
        int packageId,
        IConfigVersionService configVersion,
        CancellationToken cancellationToken)
    {
        var siteIds = await db.Queryable<Site>()
            .Where(s => s.UserPackage == packageId &&
                        s.Enable == true &&
                        (s.State == null || s.State == string.Empty || s.State == "running"))
            .Select(s => (long)s.Id)
            .ToListAsync();

        if (siteIds.Count == 0)
        {
            return siteIds;
        }

        await db.Updateable<Site>()
            .SetColumns(s => new Site { State = "traffic_limit" })
            .Where(s => siteIds.Contains(s.Id))
            .ExecuteCommandAsync();

        await configVersion.BumpAsync("site", siteIds, cancellationToken);
        return siteIds;
    }

    private static async Task ClearTrafficLimitAsync(
        ISqlSugarClient db,
        int packageId,
        IConfigVersionService configVersion,
        CancellationToken cancellationToken)
    {
        var siteIds = await db.Queryable<Site>()
            .Where(s => s.UserPackage == packageId && s.Enable == true && s.State == "traffic_limit")
            .Select(s => (long)s.Id)
            .ToListAsync();

        if (siteIds.Count == 0)
        {
            return;
        }

        await db.Updateable<Site>()
            .SetColumns(s => new Site { State = "running" })
            .Where(s => siteIds.Contains(s.Id))
            .ExecuteCommandAsync();

        await configVersion.BumpAsync("site", siteIds, cancellationToken);
    }

    private static double ParseDouble(string? raw, double fallback)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return fallback;
        }

        if (double.TryParse(raw.Trim(), NumberStyles.Any, CultureInfo.InvariantCulture, out var parsed))
        {
            return parsed;
        }

        return fallback;
    }
}
