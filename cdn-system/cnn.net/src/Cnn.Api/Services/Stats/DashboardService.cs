using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Services.Common;
using Microsoft.Extensions.Configuration;
using Cnn.Domain.Entities;
using SqlSugar;
using Stream = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Stats;

public interface IDashboardService
{
    Task<ServiceResult<DashboardResultDto>> GetAsync(
        AccessScope scope,
        string? overviewRange,
        string? chartRange,
        string? opsRange,
        string? rangeFallback,
        string language,
        CancellationToken cancellationToken);
}

public sealed class DashboardService : IDashboardService
{
    private const string AnnouncementType = "announcement";
    private const string DashboardTimeLayout = "yyyy-MM-dd HH:mm:ss";

    private readonly ISqlSugarClient _db;
    private readonly IAccessStatsService _accessStatsService;
    private readonly IRankingService _rankingService;
    private readonly ISiteHostIndexService _hostIndexService;
    private readonly IHostFilterResolver _hostFilterResolver;
    private readonly INodeStatusService _nodeStatusService;
    private readonly IMessageLocalizer _localizer;
    private readonly IConfiguration _configuration;

    public DashboardService(
        ISqlSugarClient db,
        IAccessStatsService accessStatsService,
        IRankingService rankingService,
        ISiteHostIndexService hostIndexService,
        IHostFilterResolver hostFilterResolver,
        INodeStatusService nodeStatusService,
        IMessageLocalizer localizer,
        IConfiguration configuration)
    {
        _db = db;
        _accessStatsService = accessStatsService;
        _rankingService = rankingService;
        _hostIndexService = hostIndexService;
        _hostFilterResolver = hostFilterResolver;
        _nodeStatusService = nodeStatusService;
        _localizer = localizer;
        _configuration = configuration;
    }

    public async Task<ServiceResult<DashboardResultDto>> GetAsync(
        AccessScope scope,
        string? overviewRange,
        string? chartRange,
        string? opsRange,
        string? rangeFallback,
        string language,
        CancellationToken cancellationToken)
    {
        var now = DateTime.Now;
        var role = scope.IsAdmin ? "admin" : "user";
        var hostFilter = await _hostFilterResolver.ResolveAsync(scope, cancellationToken);

        var overviewRng = StatsRangeResolver.Resolve(FirstNonEmpty(overviewRange, rangeFallback, "today"), null, null, now);
        var chartRng = StatsRangeResolver.Resolve(FirstNonEmpty(chartRange, rangeFallback, "today"), null, null, now);
        var opsRng = StatsRangeResolver.Resolve(FirstNonEmpty(opsRange, "7d"), null, null, now);

        var overview = EmptyOverview();
        var charts = EmptyCharts();
        var topDomains = new List<DashboardTopItemDto>();
        var topUrls = new List<DashboardTopItemDto>();
        var topIps = new List<DashboardTopItemDto>();
        var topCountries = new List<DashboardTopItemDto>();

        if (scope.IsAdmin || !hostFilter.Empty)
        {
            var topRange = StatsRangeResolver.Resolve("30min", null, null, now);
            overview = await BuildOverviewAsync(overviewRng, hostFilter, cancellationToken);
            charts = await BuildChartsAsync(chartRng, hostFilter, cancellationToken);
            topDomains = BuildTopList(await _rankingService.QueryAccessRankingAsync("domain", topRange.Start, topRange.End, hostFilter, null, 10, cancellationToken));
            topUrls = BuildTopList(await _rankingService.QueryAccessRankingAsync("url", topRange.Start, topRange.End, hostFilter, null, 10, cancellationToken));
            topIps = BuildTopList(await _rankingService.QueryAccessRankingAsync("ip", topRange.Start, topRange.End, hostFilter, null, 10, cancellationToken));
            topCountries = BuildTopList(await _rankingService.QueryRegionRankingAsync("country", topRange.Start, topRange.End, hostFilter, null, 10, cancellationToken));
        }

        var announcements = await LoadAnnouncementsAsync(5, cancellationToken);
        var packageInfo = await LoadPackageInfoAsync(scope, cancellationToken);
        var resources = await LoadResourcesAsync(scope, cancellationToken);
        var ops = scope.IsAdmin ? await LoadOpsSummaryAsync(opsRng, cancellationToken) : new DashboardOpsDto();
        var (systemStatus, license) = await LoadSystemStatusAsync(cancellationToken);

        var userInfo = await LoadUserInfoAsync(scope, role, language, cancellationToken);

        var result = new DashboardResultDto
        {
            User = userInfo,
            Stats = overview,
            Charts = charts,
            TopDomains = topDomains,
            TopUrls = topUrls,
            TopIps = topIps,
            TopCountries = topCountries,
            Announcements = announcements,
            Package = packageInfo,
            Resources = resources,
            Ops = ops,
            SystemStatus = systemStatus,
            License = license
        };

        return ServiceResult<DashboardResultDto>.Ok(result);
    }

    private async Task<DashboardUserDto> LoadUserInfoAsync(AccessScope scope, string role, string language, CancellationToken cancellationToken)
    {
        var info = new DashboardUserDto
        {
            Role = role
        };

        if (!scope.HasUserId)
        {
            return info;
        }

        var user = await _db.Queryable<User>().Where(u => u.Id == scope.UserId).FirstAsync();
        if (user == null)
        {
            return info;
        }

        var (lastLoginAt, lastLoginIp) = await LoadLastLoginAsync(user.Id, cancellationToken);
        var authState = user.CertVerified == true
            ? _localizer.Translate("dashboard.auth_verified", language)
            : _localizer.Translate("dashboard.auth_unverified", language);

        info.Username = user.Name;
        info.Id = user.Id;
        info.Level = "V0";
        info.AuthState = authState;
        info.LastLogin = lastLoginAt;
        info.LoginIp = lastLoginIp;
        info.Avatar = string.Empty;
        return info;
    }

    private async Task<(string LastLogin, string LastIp)> LoadLastLoginAsync(long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ("-", "-");
        }

        var row = await _db.Queryable<LoginLog>()
            .Where(l => l.Uid == userId && l.Success == true)
            .OrderBy(l => l.Id, OrderByType.Desc)
            .Select(l => new { l.Ip, l.CreateAt })
            .FirstAsync();

        if (row == null || row.CreateAt == null)
        {
            return ("-", "-");
        }

        var time = row.CreateAt.Value.ToString(DashboardTimeLayout);
        var ip = row.Ip?.Trim() ?? "-";
        return (time, ip);
    }

    private async Task<DashboardOverviewDto> BuildOverviewAsync(StatsRange range, HostFilter hostFilter, CancellationToken cancellationToken)
    {
        var totals = await _accessStatsService.QueryTotalsAsync(range.Start, range.End, hostFilter, cancellationToken);
        var buckets = await _accessStatsService.QueryBucketsAsync(range, hostFilter, cancellationToken);
        var series = _accessStatsService.BuildSeries(range, buckets);

        var peakMbps = 0.0;
        for (var i = 0; i < series.Bytes.Count; i++)
        {
            var val = StatsFormat.BytesToMbps(series.Bytes[i], range.Bucket);
            if (val > peakMbps)
            {
                peakMbps = val;
            }
        }

        return new DashboardOverviewDto
        {
            BandwidthPeak = StatsFormat.FormatBandwidth(StatsFormat.RoundFloat(peakMbps, 2)),
            NodeBandwidthPeak = "-",
            Requests = StatsFormat.FormatCount(totals.Requests),
            Traffic = StatsFormat.FormatBytes(totals.Bytes),
            BlockedIps = StatsFormat.FormatCount(totals.BlockedIps)
        };
    }

    private async Task<DashboardChartDto> BuildChartsAsync(StatsRange range, HostFilter hostFilter, CancellationToken cancellationToken)
    {
        var buckets = await _accessStatsService.QueryBucketsAsync(range, hostFilter, cancellationToken);
        var series = _accessStatsService.BuildSeries(range, buckets);

        var bandwidth = new List<double>();
        var traffic = new List<double>();
        var requests = new List<double>();
        var blocked = new List<double>();
        for (var i = 0; i < series.Bytes.Count; i++)
        {
            bandwidth.Add(StatsFormat.RoundFloat(StatsFormat.BytesToMbps(series.Bytes[i], range.Bucket), 2));
            traffic.Add(StatsFormat.RoundFloat(StatsFormat.BytesToMB(series.Bytes[i]), 2));
            requests.Add(series.Requests[i]);
            blocked.Add(series.BlockedIps[i]);
        }

        return new DashboardChartDto
        {
            XAxis = series.XAxis,
            Bandwidth = bandwidth,
            Requests = requests,
            Traffic = traffic,
            Blocked = blocked
        };
    }

    private static DashboardOverviewDto EmptyOverview()
    {
        return new DashboardOverviewDto
        {
            BandwidthPeak = "-",
            NodeBandwidthPeak = "-",
            Requests = "0",
            Traffic = "0 B",
            BlockedIps = "0"
        };
    }

    private static DashboardChartDto EmptyCharts()
    {
        return new DashboardChartDto
        {
            XAxis = Array.Empty<string>(),
            Bandwidth = Array.Empty<double>(),
            Requests = Array.Empty<double>(),
            Traffic = Array.Empty<double>(),
            Blocked = Array.Empty<double>()
        };
    }

    private static List<DashboardTopItemDto> BuildTopList(IReadOnlyList<RankItem> items)
    {
        if (items.Count == 0)
        {
            return new List<DashboardTopItemDto>();
        }

        var list = new List<DashboardTopItemDto>();
        foreach (var item in items)
        {
            var name = string.IsNullOrWhiteSpace(item.Item) ? "-" : item.Item.Trim();
            list.Add(new DashboardTopItemDto
            {
                Name = name,
                Count = item.RequestCount,
                Traffic = StatsFormat.FormatBytes(item.OutBytes)
            });
        }

        return list;
    }

    private async Task<List<DashboardAnnouncementDto>> LoadAnnouncementsAsync(int limit, CancellationToken cancellationToken)
    {
        if (limit <= 0)
        {
            limit = 5;
        }

        var items = await _db.Queryable<Message>()
            .Where(m => m.Type == AnnouncementType && m.IsShow == true)
            .OrderBy(m => m.Id, OrderByType.Desc)
            .Take(limit)
            .ToListAsync();

        var list = new List<DashboardAnnouncementDto>();
        foreach (var item in items)
        {
            list.Add(new DashboardAnnouncementDto
            {
                Id = item.Id,
                Title = item.Title,
                Time = item.CreateAt?.ToString("yyyy-MM-dd")
            });
        }

        return list;
    }

    private async Task<DashboardPackageDto?> LoadPackageInfoAsync(AccessScope scope, CancellationToken cancellationToken)
    {
        if (!scope.HasUserId)
        {
            return new DashboardPackageDto();
        }

        var now = DateTime.Now;
        var pkg = await _db.Queryable<UserPackage>()
            .Where(p => p.Uid == scope.UserId && (p.EndAt == null || p.EndAt >= now))
            .OrderBy(p => p.Id, OrderByType.Desc)
            .FirstAsync();

        if (pkg == null)
        {
            return new DashboardPackageDto();
        }

        var hostFilter = await BuildPackageHostFilterAsync(scope.UserId, pkg.Id, cancellationToken);
        if (hostFilter.Empty)
        {
            return new DashboardPackageDto
            {
                Name = pkg.Name,
                Desc = string.Empty,
                Percent = 0
            };
        }

        var start = pkg.StartAt ?? now.AddDays(-1);
        if (start > now)
        {
            start = now.AddDays(-1);
        }

        var totals = await _accessStatsService.QueryTotalsAsync(start, now, hostFilter, cancellationToken);
        var usedGb = totals.Bytes / (1024.0 * 1024.0 * 1024.0);
        var limitGb = pkg.Traffic ?? 0;
        var percent = 0;
        var desc = $"{usedGb:F2} GB used";
        if (limitGb > 0)
        {
            percent = (int)StatsFormat.RoundFloat(usedGb / limitGb * 100, 0);
            if (percent > 100)
            {
                percent = 100;
            }
            desc = $"{usedGb:F2} GB / {limitGb:F2} GB";
        }

        return new DashboardPackageDto
        {
            Name = pkg.Name,
            Desc = desc,
            Percent = percent
        };
    }

    private async Task<HostFilter> BuildPackageHostFilterAsync(long userId, long packageId, CancellationToken cancellationToken)
    {
        if (userId <= 0 || packageId <= 0)
        {
            return new HostFilter();
        }

        var siteIds = await _db.Queryable<Site>()
            .Where(s => s.Uid == userId && s.UserPackage == packageId)
            .Select(s => s.Id)
            .ToListAsync();

        if (siteIds.Count == 0)
        {
            return new HostFilter();
        }

        var index = await _hostIndexService.LoadAsync(userId, cancellationToken);
        var filter = new HostFilter();
        var seenExact = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var seenWildcard = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var siteId in siteIds)
        {
            if (!index.SiteFilters.TryGetValue(siteId, out var siteFilter))
            {
                continue;
            }

            foreach (var host in siteFilter.Exact)
            {
                if (seenExact.Add(host))
                {
                    filter.Exact.Add(host);
                }
            }

            foreach (var suffix in siteFilter.Wildcards)
            {
                if (seenWildcard.Add(suffix))
                {
                    filter.Wildcards.Add(suffix);
                }
            }
        }

        return filter;
    }

    private async Task<DashboardResourceDto> LoadResourcesAsync(AccessScope scope, CancellationToken cancellationToken)
    {
        var siteQuery = _db.Queryable<Site>().Select(s => new Site { Id = s.Id, Domain = s.Domain });
        if (!scope.IsAdmin && scope.HasUserId)
        {
            siteQuery = siteQuery.Where(s => s.Uid == scope.UserId);
        }

        var sites = await siteQuery.ToListAsync();
        var domainCount = CountUniqueDomains(sites);

        var forwardQuery = _db.Queryable<Stream>();
        if (!scope.IsAdmin && scope.HasUserId)
        {
            forwardQuery = forwardQuery.Where(f => f.Uid == scope.UserId);
        }
        var forwardCount = await forwardQuery.CountAsync();

        var certQuery = _db.Queryable<Cert>();
        if (!scope.IsAdmin && scope.HasUserId)
        {
            certQuery = certQuery.Where(c => c.Uid == scope.UserId);
        }
        var certCount = await certQuery.CountAsync();

        var packageQuery = _db.Queryable<UserPackage>();
        if (!scope.IsAdmin && scope.HasUserId)
        {
            packageQuery = packageQuery.Where(p => p.Uid == scope.UserId);
        }
        var packageCount = await packageQuery.CountAsync();

        return new DashboardResourceDto
        {
            Domains = domainCount,
            Forward = forwardCount,
            Certs = certCount,
            Packages = packageCount
        };
    }

    private static long CountUniqueDomains(IEnumerable<Site> sites)
    {
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var site in sites)
        {
            foreach (var domain in DomainParser.ParseDomains(site.Domain))
            {
                var host = NormalizeDashboardDomain(domain);
                if (string.IsNullOrWhiteSpace(host))
                {
                    continue;
                }
                seen.Add(host);
            }
        }

        return seen.Count;
    }

    private static string NormalizeDashboardDomain(string input)
    {
        var host = input?.Trim().ToLowerInvariant() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        host = host.Replace("http://", string.Empty).Replace("https://", string.Empty);
        var idx = host.IndexOfAny(new[] { '/', '#', '?' });
        if (idx >= 0)
        {
            host = host[..idx];
        }

        var colon = host.IndexOf(':');
        if (colon >= 0)
        {
            host = host[..colon];
        }

        host = host.TrimStart('*', '.').TrimEnd('.');
        return host;
    }

    private async Task<DashboardOpsDto> LoadOpsSummaryAsync(StatsRange range, CancellationToken cancellationToken)
    {
        var userCount = await _db.Queryable<User>()
            .Where(u => u.Type != 1 && u.CreateAt >= range.Start && u.CreateAt <= range.End)
            .CountAsync();

        var packageCount = await _db.Queryable<Order>()
            .Where(o => o.Type != null && (SqlFunc.ToLower(o.Type) == "purchase" || SqlFunc.ToLower(o.Type) == "renew"))
            .Where(o => o.State != null && (SqlFunc.ToLower(o.State) == "paid" || SqlFunc.ToLower(o.State) == "success" || SqlFunc.ToLower(o.State) == "done"))
            .Where(o => o.CreateAt >= range.Start && o.CreateAt <= range.End)
            .CountAsync();

        var rechargeSum = await _db.Queryable<Order>()
            .Where(o => o.Type != null && SqlFunc.ToLower(o.Type) == "recharge")
            .Where(o => o.State != null && (SqlFunc.ToLower(o.State) == "paid" || SqlFunc.ToLower(o.State) == "success" || SqlFunc.ToLower(o.State) == "done"))
            .Where(o => o.CreateAt >= range.Start && o.CreateAt <= range.End)
            .Select(o => SqlFunc.AggregateSum(o.Amount))
            .FirstAsync();

        var rechargeText = ((rechargeSum ?? 0) / 100.0).ToString("F2");

        return new DashboardOpsDto
        {
            Summary = new DashboardOpsSummaryDto
            {
                Users = userCount,
                Packages = packageCount,
                Recharge = rechargeText
            }
        };
    }

    private async Task<(DashboardSystemStatusDto Status, DashboardLicenseDto License)> LoadSystemStatusAsync(CancellationToken cancellationToken)
    {
        var totalNodes = await _db.Queryable<Node>().Where(n => n.Pid == 0).CountAsync();
        var enabledNodes = await _db.Queryable<Node>()
            .Where(n => n.Pid == 0 && n.Enable == true)
            .Select(n => n.Id)
            .ToListAsync();

        var onlineNodes = 0;
        foreach (var id in enabledNodes)
        {
            if (_nodeStatusService.IsOnline(id, TimeSpan.FromSeconds(90)))
            {
                onlineNodes++;
            }
        }

        var elastic = ClickHouseHttpHelper.ResolveConfig(_configuration) != null;
        var status = new DashboardSystemStatusDto
        {
            Master = true,
            Elastic = elastic,
            Agent = totalNodes > 0 && onlineNodes == totalNodes,
            AgentTotal = totalNodes,
            AgentOnline = onlineNodes,
            CheckedAt = DateTime.Now.ToString(DashboardTimeLayout)
        };

        var license = new DashboardLicenseDto
        {
            TotalNodes = totalNodes,
            CurrentNodes = onlineNodes,
            ExpireAt = "-"
        };

        return (status, license);
    }



    private static string FirstNonEmpty(params string?[] values)
    {
        foreach (var value in values)
        {
            if (!string.IsNullOrWhiteSpace(value))
            {
                return value.Trim();
            }
        }

        return string.Empty;
    }
}

