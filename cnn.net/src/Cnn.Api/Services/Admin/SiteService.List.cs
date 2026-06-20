using System.Net;
using System.Text;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed partial class SiteService
{
    public async Task<ServiceResult<SiteListResult>> ListAsync(
        SiteListQuery query,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new SiteListQuery();
        if (!isAdmin && (!userId.HasValue || userId <= 0))
        {
            return ServiceResult<SiteListResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var q = _db.Queryable<Site>();
        if (!isAdmin)
        {
            q = q.Where(s => s.Uid == (int)userId!.Value);
        }
        else if (query.UserId is > 0)
        {
            q = q.Where(s => s.Uid == (int)query.UserId.Value);
        }

        if (query.UserPackageId is > 0)
        {
            q = q.Where(s => s.UserPackage == (int)query.UserPackageId.Value);
        }

        if (query.NodeGroupId is > 0)
        {
            q = q.Where(s => s.NodeGroupId == (int)query.NodeGroupId.Value);
        }

        var groupIds = ParseGroupIds(query.GroupId);
        if (groupIds.Count > 0)
        {
            var siteIds = await FindSiteIdsByGroupIdsAsync(groupIds);
            if (siteIds.Count == 0)
            {
                return ServiceResult<SiteListResult>.Ok(new SiteListResult(Array.Empty<SiteListItem>(), 0));
            }
            q = q.Where(s => siteIds.Contains(s.Id));
        }

        var status = query.Status?.Trim().ToLowerInvariant();
        if (!string.IsNullOrWhiteSpace(status))
        {
            if (status is "enabled" or "enable" or "1" or "true")
            {
                q = q.Where(s => s.Enable == true);
            }
            else if (status is "disabled" or "disable" or "0" or "false")
            {
                q = q.Where(s => s.Enable == false);
            }
        }

        var https = query.Https?.Trim().ToLowerInvariant();
        if (!string.IsNullOrWhiteSpace(https))
        {
            if (https is "1" or "true")
            {
                q = q.Where(s => !string.IsNullOrEmpty(s.HttpsListen));
            }
            else if (https is "0" or "false")
            {
                q = q.Where(s => string.IsNullOrEmpty(s.HttpsListen));
            }
        }

        var keyword = query.Keyword?.Trim() ?? string.Empty;
        var searchField = query.SearchField?.Trim().ToLowerInvariant() ?? "all";
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            q = await ApplySearchAsync(q, keyword, searchField);
        }

        var page = query.Page < 1 ? 1 : query.Page;
        var pageSize = query.PageSize < 1 ? 10 : query.PageSize;
        if (query.Size is > 0)
        {
            pageSize = query.Size.Value;
        }

        var total = await q.CountAsync();
        var sites = await q.OrderBy(s => s.Id, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);

        var items = await BuildSiteListItemsAsync(sites);
        return ServiceResult<SiteListResult>.Ok(new SiteListResult(items, total));
    }

    public async Task<ServiceResult<SiteDetailDto>> GetAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        var site = await _db.Queryable<Site>().Where(s => s.Id == id).FirstAsync();
        if (site == null)
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.NotFound, "site_not_found");
        }

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            if (site.Uid != (int)userId.Value)
            {
                return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.PermissionDenied);
            }
        }

        var items = await BuildSiteListItemsAsync(new List<Site> { site });
        var item = items.FirstOrDefault();
        if (item == null)
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.NotFound, "site_not_found");
        }

        return ServiceResult<SiteDetailDto>.Ok(ToDetailDto(item));
    }

    public async Task<ServiceResult<SiteExportResult>> ExportAsync(
        SiteListQuery query,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        var listResult = await ListAsync(query, userId, isAdmin, cancellationToken);
        if (!listResult.Success)
        {
            return ServiceResult<SiteExportResult>.Fail(listResult.ErrorCode, listResult.MessageKey);
        }

        var sb = new StringBuilder();
        sb.AppendLine("ID,User,Domain,Listen,Origin,CNAME,HTTPS,Package,Group,Region,Status,CreatedAt");

        foreach (var item in listResult.Data?.List ?? Array.Empty<SiteListItem>())
        {
            var httpsVal = item.Https ? "yes" : "no";
            var statusVal = item.Status ? "enabled" : "disabled";
            var createdAt = item.CreatedAt?.ToString("yyyy-MM-ddTHH:mm:ss") ?? string.Empty;

            sb.AppendLine(string.Join(',', new[]
            {
                EscapeCsv(item.Id.ToString()),
                EscapeCsv(item.UserName ?? string.Empty),
                EscapeCsv(item.DomainDisplay ?? string.Empty),
                EscapeCsv(item.ListenPorts ?? string.Empty),
                EscapeCsv(item.OriginDisplay ?? string.Empty),
                EscapeCsv(item.Cname ?? string.Empty),
                EscapeCsv(httpsVal),
                EscapeCsv(item.UserPackageName ?? string.Empty),
                EscapeCsv(item.GroupName ?? string.Empty),
                EscapeCsv(item.NodeGroupName ?? string.Empty),
                EscapeCsv(statusVal),
                EscapeCsv(createdAt)
            }));
        }

        return ServiceResult<SiteExportResult>.Ok(new SiteExportResult("sites.csv", sb.ToString()));
    }

    public async Task<ServiceResult<SiteResolveResult>> ResolveAsync(string domain, CancellationToken cancellationToken)
    {
        var input = domain?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(input))
        {
            return ServiceResult<SiteResolveResult>.Fail(ErrorCodes.MissingParam, "domain_name_required");
        }

        var cname = string.Empty;
        var ips = new List<string>();
        try
        {
            var entry = await Dns.GetHostEntryAsync(input);
            cname = entry.HostName?.TrimEnd('.') ?? string.Empty;
        }
        catch
        {
            cname = string.Empty;
        }

        try
        {
            var addresses = await Dns.GetHostAddressesAsync(input);
            foreach (var addr in addresses)
            {
                ips.Add(addr.ToString());
            }
        }
        catch
        {
            ips = new List<string>();
        }

        return ServiceResult<SiteResolveResult>.Ok(new SiteResolveResult
        {
            Domain = input,
            Cname = cname,
            Ips = ips
        });
    }

    private async Task<ISugarQueryable<Site>> ApplySearchAsync(ISugarQueryable<Site> query, string keyword, string searchField)
    {
        var like = $"%{keyword}%";
        switch (searchField)
        {
            case "site_id":
                if (long.TryParse(keyword, out var siteId))
                {
                    return query.Where(s => s.Id == siteId);
                }
                return query.Where(s => false);
            case "domain":
            case "multi_domain":
                return query.Where(s => SqlFunc.Contains(s.Domain, keyword));
            case "origin":
                return query.Where(s => SqlFunc.Contains(s.Backend, keyword));
            case "cname":
                return query.Where(s => SqlFunc.Contains(s.CnameHostname, keyword) || SqlFunc.Contains(s.CnameDomain, keyword));
            case "package":
            {
                var ids = await FindUserPackageIdsByNameAsync(keyword);
                if (ids.Count == 0)
                {
                    return query.Where(s => false);
                }
                return query.Where(s => ids.Contains(s.UserPackage ?? 0));
            }
            case "group":
            {
                var siteIds = await FindSiteIdsByGroupNameAsync(keyword);
                if (siteIds.Count == 0)
                {
                    return query.Where(s => false);
                }
                return query.Where(s => siteIds.Contains(s.Id));
            }
            case "user":
            {
                var userIds = await FindUserIdsByKeywordAsync(keyword);
                if (userIds.Count == 0)
                {
                    return query.Where(s => false);
                }
                return query.Where(s => userIds.Contains(s.Uid ?? 0));
            }
            case "http_port":
                return query.Where(s => SqlFunc.Contains(s.HttpListen, keyword));
            case "https_port":
                return query.Where(s => SqlFunc.Contains(s.HttpsListen, keyword));
            default:
            {
                var userIds = await FindUserIdsByKeywordAsync(keyword);
                var pkgIds = await FindUserPackageIdsByNameAsync(keyword);
                var siteIds = await FindSiteIdsByGroupNameAsync(keyword);

                var cond = Expressionable.Create<Site>();
                cond.Or(s =>
                    SqlFunc.Contains(s.Domain, keyword) ||
                    SqlFunc.Contains(s.Backend, keyword) ||
                    SqlFunc.Contains(s.CnameHostname, keyword) ||
                    SqlFunc.Contains(s.CnameDomain, keyword));

                if (long.TryParse(keyword, out var id))
                {
                    cond.Or(s => s.Id == id);
                }
                if (userIds.Count > 0)
                {
                    cond.Or(s => userIds.Contains(s.Uid ?? 0));
                }
                if (pkgIds.Count > 0)
                {
                    cond.Or(s => pkgIds.Contains(s.UserPackage ?? 0));
                }
                if (siteIds.Count > 0)
                {
                    cond.Or(s => siteIds.Contains(s.Id));
                }

                return query.Where(cond.ToExpression());
            }
        }
    }

    private async Task<IReadOnlyList<SiteListItem>> BuildSiteListItemsAsync(IReadOnlyList<Site> sites)
    {
        if (sites == null || sites.Count == 0)
        {
            return Array.Empty<SiteListItem>();
        }

        var userMap = await LoadUserNameMapAsync(sites);
        var pkgMap = await LoadUserPackageMapAsync(sites);
        var (groupMap, relMap) = await LoadSiteGroupMapAsync(sites);
        var nodeGroupMap = await LoadNodeGroupMapAsync(sites);
        var regionMap = await LoadRegionMapAsync(sites);

        var siteIds = sites.Select(s => (long)s.Id).Where(id => id > 0).ToList();
        var settingsMap = await LoadSiteSettingsMapAsync(siteIds);
        var siteTypeMap = await LoadSiteTypeMetaMapAsync(siteIds);

        var globalDefaults = await LoadGlobalDefaultConfigAsync();
        var defaultCache = new Dictionary<(long, long), Dictionary<string, string>>();
        var scopedCache = new Dictionary<(long, long), Dictionary<string, string>>();

        var items = new List<SiteListItem>(sites.Count);
        foreach (var site in sites)
        {
            var groupIds = relMap.TryGetValue(site.Id, out var gids) ? gids : new List<long>();
            var groupId = groupIds.Count > 0 ? groupIds[0] : 0;

            var settings = settingsMap.TryGetValue(site.Id, out var map)
                ? map
                : new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);

            var siteType = siteTypeMap.TryGetValue(site.Id, out var type) ? type : string.Empty;
            settings = await EnsureSiteSettingsAsync(site, groupId, siteType, settings, globalDefaults, defaultCache, scopedCache);

            var domains = DomainParser.ParseDomains(site.Domain);
            var domainDisplay = domains.Count > 0 ? string.Join(",", domains) : string.Empty;

            var backends = DecodeStringList(site.Backend);
            var originDisplay = backends.Count > 0 ? string.Join(",", backends) : string.Empty;

            var httpPorts = DecodeStringList(site.HttpListen);
            var httpsPorts = DecodeStringList(site.HttpsListen);

            var httpOn = httpPorts.Count > 0 || !string.IsNullOrWhiteSpace(site.HttpListen);
            var httpsOn = httpsPorts.Count > 0 || !string.IsNullOrWhiteSpace(site.HttpsListen);
            if (settings.TryGetValue("https", out var httpsRaw) && httpsRaw is Dictionary<string, object?> httpsCfg)
            {
                if (httpsCfg.TryGetValue("enable", out var enable))
                {
                    httpsOn = ParseBool(enable, httpsOn);
                }
            }

            if (httpOn && httpPorts.Count == 0)
            {
                httpPorts = new List<string> { "80" };
            }
            if (httpsOn && httpsPorts.Count == 0)
            {
                httpsPorts = new List<string> { "443" };
            }

            var listenParts = new List<string>();
            if (httpOn && httpPorts.Count > 0)
            {
                listenParts.Add("HTTP:" + string.Join(",", httpPorts));
            }
            if (httpsOn && httpsPorts.Count > 0)
            {
                listenParts.Add("HTTPS:" + string.Join(",", httpsPorts));
            }
            var listenPorts = listenParts.Count > 0 ? string.Join(" ", listenParts) : string.Empty;

            var cname = string.IsNullOrWhiteSpace(site.CnameHostname) ? "-" : site.CnameHostname.Trim();

            MergeSecurityIpList(settings, "blacklist", site.BlackIp);
            MergeSecurityIpList(settings, "whitelist", site.WhiteIp);

            var pkg = site.UserPackage is > 0 && pkgMap.TryGetValue(site.UserPackage.Value, out var pkgItem)
                ? pkgItem
                : null;

            var item = new SiteListItem
            {
                Id = site.Id,
                UserId = site.Uid ?? 0,
                UserName = userMap.TryGetValue(site.Uid ?? 0, out var userName) ? userName : string.Empty,
                Domains = domains,
                DomainDisplay = domainDisplay,
                ListenPorts = listenPorts,
                HttpListen = httpPorts,
                HttpsListen = httpsPorts,
                OriginDisplay = originDisplay,
                Cname = cname,
                Backends = backends,
                Https = httpsOn,
                CertId = site.CertId ?? 0,
                UserPackageId = site.UserPackage ?? 0,
                UserPackageName = pkg?.Name,
                DnsProviderId = site.DnsProviderId ?? 0,
                GroupId = 0,
                GroupIds = groupIds,
                GroupName = string.Empty,
                NodeGroupId = site.NodeGroupId ?? 0,
                NodeGroupName = nodeGroupMap.TryGetValue(site.NodeGroupId ?? 0, out var groupName) ? groupName : string.Empty,
                RegionId = site.RegionId ?? 0,
                RegionName = regionMap.TryGetValue(site.RegionId ?? 0, out var regionName) ? regionName : string.Empty,
                Status = site.Enable ?? false,
                State = site.State,
                Settings = settings,
                ExpireTime = pkg?.EndAt?.ToString("yyyy-MM-dd"),
                CreatedAt = site.CreateAt,
                UpdatedAt = site.UpdateAt
            };

            if (groupIds.Count > 0)
            {
                item.GroupId = groupIds[0];
                var names = groupIds
                    .Select(gid => groupMap.TryGetValue(gid, out var name) ? name : string.Empty)
                    .Where(name => !string.IsNullOrWhiteSpace(name))
                    .ToList();
                item.GroupName = names.Count > 0 ? string.Join(", ", names) : string.Empty;
            }

            items.Add(item);
        }

        return items;
    }

    private static SiteDetailDto ToDetailDto(SiteListItem item)
    {
        return new SiteDetailDto
        {
            Id = item.Id,
            UserId = item.UserId,
            UserName = item.UserName,
            Domains = item.Domains,
            DomainDisplay = item.DomainDisplay,
            ListenPorts = item.ListenPorts,
            HttpListen = item.HttpListen,
            HttpsListen = item.HttpsListen,
            OriginDisplay = item.OriginDisplay,
            Cname = item.Cname,
            Backends = item.Backends,
            Https = item.Https,
            CertId = item.CertId,
            UserPackageId = item.UserPackageId,
            UserPackageName = item.UserPackageName,
            DnsProviderId = item.DnsProviderId,
            GroupId = item.GroupId,
            GroupIds = item.GroupIds,
            GroupName = item.GroupName,
            NodeGroupId = item.NodeGroupId,
            NodeGroupName = item.NodeGroupName,
            RegionId = item.RegionId,
            RegionName = item.RegionName,
            Status = item.Status,
            State = item.State,
            Settings = item.Settings,
            ExpireTime = item.ExpireTime,
            CreatedAt = item.CreatedAt,
            UpdatedAt = item.UpdatedAt
        };
    }

    private static string EscapeCsv(string input)
    {
        if (string.IsNullOrEmpty(input))
        {
            return string.Empty;
        }

        if (input.Contains(',') || input.Contains('"') || input.Contains('\n') || input.Contains('\r'))
        {
            var escaped = input.Replace("\"", "\"\"");
            return "\"" + escaped + "\"";
        }

        return input;
    }
}
