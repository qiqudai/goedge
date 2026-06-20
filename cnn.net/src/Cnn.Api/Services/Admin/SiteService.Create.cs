using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed partial class SiteService
{
    public async Task<ServiceResult<SiteDetailDto>> CreateAsync(
        SiteCreateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        request ??= new SiteCreateRequest();

        var targetUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (targetUserId <= 0)
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var userPackageId = request.UserPackageId;
        if (userPackageId <= 0)
        {
            userPackageId = await ResolveDefaultUserPackageIdAsync(targetUserId);
        }
        if (userPackageId <= 0)
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.NotFound, "package_not_found");
        }

        if (!isAdmin && !await EnsureUserPackageOwnershipAsync(targetUserId, userPackageId))
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.PermissionDenied);
        }

        var domains = ResolveDomains(request.Domains, request.DomainsInput);
        if (domains.Count == 0)
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.MissingParam, "domain_name_required");
        }

        var limitOk = await CheckDomainLimitAsync(targetUserId, userPackageId, domains, null);
        if (!limitOk.Success)
        {
            return ServiceResult<SiteDetailDto>.Fail(limitOk.ErrorCode, limitOk.MessageKey);
        }

        if (await DomainExistsAsync(domains[0]))
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.AlreadyExists, "domain_exists");
        }

        var backends = ResolveDomains(request.Backends, request.BackendsInput);
        var nodeGroupId = await ResolveNodeGroupFromPackageAsync(userPackageId, request.NodeGroupId);
        var now = DateTime.Now;

        var site = new Site
        {
            Uid = (int)targetUserId,
            UserPackage = (int)userPackageId,
            DnsProviderId = request.DnsProviderId > 0 ? (int?)request.DnsProviderId : null,
            NodeGroupId = nodeGroupId > 0 ? (int?)nodeGroupId : null,
            Domain = EncodeStringList(domains),
            Backend = EncodeStringList(backends),
            HttpListen = EncodeStringList(new[] { "80" }),
            State = "running",
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        };

        var siteType = (request.SiteType ?? string.Empty).Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(siteType))
        {
            siteType = "website";
        }

        var settings = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase)
        {
            ["site_type"] = siteType
        };

        UserPackage? pkg = null;
        if (userPackageId > 0)
        {
            pkg = await _db.Queryable<UserPackage>().Where(p => p.Id == userPackageId).FirstAsync();
        }

        if (pkg != null)
        {
            if ((site.NodeGroupId ?? 0) == 0 && (pkg.NodeGroupId ?? 0) > 0)
            {
                site.NodeGroupId = pkg.NodeGroupId;
            }
            if ((site.RegionId ?? 0) == 0 && (pkg.RegionId ?? 0) > 0)
            {
                site.RegionId = pkg.RegionId;
            }
            if (!(site.EnableBackupGroup ?? false) && (pkg.EnableBackupGroup ?? false) && (pkg.BackupNodeGroup ?? 0) > 0)
            {
                site.EnableBackupGroup = true;
                site.BackupNodeGroup = pkg.BackupNodeGroup;
            }
        }

        if ((site.RegionId ?? 0) == 0)
        {
            site.RegionId = await ResolveRegionFromPackageAsync(userPackageId, site.NodeGroupId ?? 0);
        }

        if (pkg != null)
        {
            ApplySiteCname(site, pkg, domains);
        }

        if ((site.DnsProviderId ?? 0) == 0)
        {
            var dnsProviderId = await ResolveDefaultDnsProviderIdAsync(targetUserId, cancellationToken);
            if (dnsProviderId > 0)
            {
                site.DnsProviderId = (int)dnsProviderId;
            }
        }

        var globalDefaults = await LoadGlobalDefaultConfigAsync();
        ApplySiteTemplateDefaultsByType(settings, siteType, globalDefaults);

        var defaults = await LoadSiteDefaultMapWithGroupAsync(targetUserId, request.GroupId);
        ApplySiteDefaults(site, settings, defaults);
        EnsureSitePersistenceDefaults(site);
        settings = SiteSettingsNormalizer.Normalize(settings);

        var groupIds = ResolveGroupIds(request.GroupIds, request.GroupId);

        var tran = await _db.Ado.UseTranAsync(async () =>
        {
            var id = await _db.Insertable(site).ExecuteReturnIdentityAsync();
            if (id <= 0)
            {
                throw new InvalidOperationException("db_create_error");
            }
            site.Id = id;

            if (groupIds.Count > 0)
            {
                var relations = groupIds
                    .Where(gid => gid > 0)
                    .Distinct()
                    .Select(gid => new MergeSiteGroup { SiteId = id, GroupId = (int)gid })
                    .ToList();
                if (relations.Count > 0)
                {
                    await _db.Insertable(relations).ExecuteCommandAsync();
                }
            }

            await SaveSiteSettingsAsync(site.Id, settings);
            await UpsertSiteTypeMetaAsync(site.Id, siteType);
        });

        if (!tran.IsSuccess || site.Id <= 0)
        {
            var key = string.Equals(tran.ErrorMessage, "db_create_error", StringComparison.Ordinal)
                ? "db_create_error"
                : "db_save_error";
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.DbError, key);
        }

        await _configVersionService.BumpAsync("site", new[] { (long)site.Id }, cancellationToken);

        await RefreshSiteCnameHostnameAsync(site, null, null);
        await TrySyncUserDnsRecordsAsync(null, site);
        await ResyncSiteCnameForSiteAsync(site);

        var items = await BuildSiteListItemsAsync(new List<Site> { site });
        var detail = items.Count > 0 ? ToDetailDto(items[0]) : new SiteDetailDto { Id = site.Id };
        return ServiceResult<SiteDetailDto>.Ok(detail);
    }

    private static long ResolveUserId(long requestUserId, long? userId, bool isAdmin)
    {
        if (!isAdmin)
        {
            return userId ?? 0;
        }

        if (requestUserId > 0)
        {
            return requestUserId;
        }

        return userId ?? 0;
    }

    private async Task<long> ResolveDefaultUserPackageIdAsync(long userId)
    {
        UserPackage? pack = null;
        if (userId > 0)
        {
            pack = await _db.Queryable<UserPackage>()
                .Where(p => p.Uid == userId)
                .OrderBy(p => p.Id, OrderByType.Asc)
                .FirstAsync();
        }

        pack ??= await _db.Queryable<UserPackage>()
            .OrderBy(p => p.Id, OrderByType.Asc)
            .FirstAsync();

        return pack?.Id ?? 0;
    }

    private async Task<bool> EnsureUserPackageOwnershipAsync(long userId, long userPackageId)
    {
        if (userId <= 0 || userPackageId <= 0)
        {
            return false;
        }

        return await _db.Queryable<UserPackage>()
            .AnyAsync(p => p.Id == userPackageId && p.Uid == (int)userId);
    }

    private static List<string> ResolveDomains(IReadOnlyList<string>? domains, string? input)
    {
        if (domains != null && domains.Count > 0)
        {
            return NormalizeStringList(domains);
        }

        if (!string.IsNullOrWhiteSpace(input))
        {
            return NormalizeStringList(SplitFields(input));
        }

        return new List<string>();
    }

    private static List<long> ResolveGroupIds(IReadOnlyList<long>? groupIds, long groupId)
    {
        if (groupIds != null && groupIds.Count > 0)
        {
            return groupIds.Where(id => id > 0).Distinct().ToList();
        }
        if (groupId > 0)
        {
            return new List<long> { groupId };
        }
        return new List<long>();
    }

    private async Task<long> ResolveNodeGroupFromPackageAsync(long userPackageId, long nodeGroupId)
    {
        if (nodeGroupId > 0)
        {
            return nodeGroupId;
        }
        if (userPackageId <= 0)
        {
            return 0;
        }

        var pkg = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == userPackageId)
            .Select(p => new { p.NodeGroupId })
            .FirstAsync();

        return pkg?.NodeGroupId ?? 0;
    }

    private async Task<int> ResolveRegionFromPackageAsync(long userPackageId, long nodeGroupId)
    {
        if (userPackageId > 0)
        {
            var pkg = await _db.Queryable<UserPackage>()
                .Where(p => p.Id == userPackageId)
                .Select(p => new { p.RegionId })
                .FirstAsync();
            if (pkg?.RegionId is > 0)
            {
                return pkg.RegionId.Value;
            }
        }

        if (nodeGroupId > 0)
        {
            var group = await _db.Queryable<NodeGroup>()
                .Where(g => g.Id == nodeGroupId)
                .Select(g => new { g.RegionId })
                .FirstAsync();
            if (group?.RegionId is > 0)
            {
                return group.RegionId.Value;
            }
        }

        return 0;
    }

    private async Task<long> ResolveDefaultDnsProviderIdAsync(long userId, CancellationToken cancellationToken)
    {
        var result = await _certService.GetDefaultSettingsAsync(userId, true, cancellationToken);
        if (result.Success && result.Data != null && result.Data.DnsApi > 0)
        {
            return result.Data.DnsApi;
        }

        return 0;
    }

    private async Task<ServiceResult<bool>> CheckDomainLimitAsync(long userId, long userPackageId, IReadOnlyList<string> newDomains, long? excludeSiteId)
    {
        if (userId <= 0 || userPackageId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        var limits = await LoadDomainLimitsAsync(userPackageId);
        if (limits == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "package_not_found");
        }

        if (limits.Value.DomainLimit <= 0 && limits.Value.MainDomainLimit <= 0)
        {
            return ServiceResult<bool>.Ok(true);
        }

        var sets = await LoadUserDomainSetsAsync(userId, excludeSiteId);
        AddDomains(sets.DomainSet, sets.MainSet, newDomains);

        if (limits.Value.DomainLimit > 0 && sets.DomainSet.Count > limits.Value.DomainLimit)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.QuotaExceeded, "quota_exceeded");
        }
        if (limits.Value.MainDomainLimit > 0 && sets.MainSet.Count > limits.Value.MainDomainLimit)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.QuotaExceeded, "quota_exceeded");
        }

        return ServiceResult<bool>.Ok(true);
    }

    private async Task<bool> DomainExistsAsync(string domain)
    {
        if (string.IsNullOrWhiteSpace(domain))
        {
            return false;
        }

        return await _db.Queryable<Site>()
            .Where(s => SqlFunc.Contains(s.Domain, domain))
            .AnyAsync();
    }

    private static void ApplySiteCname(Site site, UserPackage pkg, IReadOnlyList<string> domains)
    {
        var pkgMode = (pkg.CnameMode ?? string.Empty).Trim();
        var pkgDomain = (pkg.CnameDomain ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(pkgDomain))
        {
            pkgDomain = DefaultCnameDomain;
        }

        if (string.Equals(pkgMode, "package", StringComparison.OrdinalIgnoreCase) && !string.IsNullOrWhiteSpace(pkg.CnameHostname))
        {
            site.CnameHostname = pkg.CnameHostname.Trim();
            if (!string.IsNullOrWhiteSpace(pkg.CnameDomain))
            {
                site.CnameHostname += "." + pkg.CnameDomain.Trim();
            }
            return;
        }

        site.CnameDomain = pkgDomain;
        if (domains.Count > 0)
        {
            site.CnameHostname = domains[0] + "." + pkgDomain;
        }
    }
}
