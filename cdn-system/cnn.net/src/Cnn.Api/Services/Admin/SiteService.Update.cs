using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed partial class SiteService
{
    public async Task<ServiceResult<SiteDetailDto>> UpdateAsync(
        long id,
        SiteUpdateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        request ??= new SiteUpdateRequest();

        var site = await _db.Queryable<Site>().Where(s => s.Id == id).FirstAsync();
        if (site == null)
        {
            return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.NotFound, "not_found");
        }

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.PermissionDenied);
            }
            if (site.Uid != (int)userId.Value)
            {
                return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.PermissionDenied);
            }
        }

        if (!isAdmin && request.UserPackageId is > 0)
        {
            if (!await EnsureUserPackageOwnershipAsync(userId ?? 0, request.UserPackageId.Value))
            {
                return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.PermissionDenied);
            }
        }

        if (!isAdmin && (request.GroupIds != null || request.GroupId != null))
        {
            var groupIds = ResolveGroupIds(request.GroupIds, request.GroupId ?? 0);
            if (groupIds.Count > 0)
            {
                var allowed = await FilterSiteGroupIdsForUserAsync(groupIds, userId ?? 0);
                if (allowed.Count != groupIds.Count)
                {
                    return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.PermissionDenied);
                }
                request.GroupIds = allowed;
                request.GroupId = allowed.Count > 0 ? allowed[0] : 0;
            }
        }

        if (request.Domains != null || request.UserPackageId != null)
        {
            var domainsForCheck = request.Domains != null ? NormalizeStringList(request.Domains) : DomainParser.ParseDomains(site.Domain).ToList();
            var packageId = request.UserPackageId is > 0 ? request.UserPackageId.Value : site.UserPackage ?? 0;
            var limitOk = await CheckDomainLimitAsync(site.Uid ?? 0, packageId, domainsForCheck, site.Id);
            if (!limitOk.Success)
            {
                return ServiceResult<SiteDetailDto>.Fail(limitOk.ErrorCode, limitOk.MessageKey);
            }
        }

        var oldSnapshot = CloneSiteForCompare(site);
        var settings = await LoadSiteSettingsAsync(site.Id);

        var (ccDefaultFromSettings, hasCcDefault) = ExtractCcDefaultRule(request.Settings);
        var (blacklistFromSettings, hasBlacklist) = ExtractSecurityIpList(request.Settings, "blacklist");
        var (whitelistFromSettings, hasWhitelist) = ExtractSecurityIpList(request.Settings, "whitelist");

        if (hasBlacklist)
        {
            SetSecurityIpList(request.Settings!, "blacklist", blacklistFromSettings);
        }
        if (hasWhitelist)
        {
            SetSecurityIpList(request.Settings!, "whitelist", whitelistFromSettings);
        }

        if (request.Settings != null)
        {
            settings = MergeSettingsMaps(settings, request.Settings);
            settings = SiteSettingsNormalizer.Normalize(settings);
        }

        if (request.UserPackageId is > 0)
        {
            site.UserPackage = (int)request.UserPackageId.Value;
        }
        if (request.DnsProviderId.HasValue)
        {
            site.DnsProviderId = request.DnsProviderId > 0 ? (int?)request.DnsProviderId.Value : null;
        }
        if (request.HttpListen != null)
        {
            site.HttpListen = EncodeStringList(request.HttpListen);
        }
        if (request.HttpsListen != null)
        {
            site.HttpsListen = EncodeStringList(request.HttpsListen);
        }
        if (request.Backends != null)
        {
            site.Backend = EncodeStringList(request.Backends);
        }
        if (!string.IsNullOrWhiteSpace(request.BalanceWay))
        {
            site.BalanceWay = request.BalanceWay.Trim();
        }
        if (!string.IsNullOrWhiteSpace(request.BackendProtocol))
        {
            site.BackendProtocol = request.BackendProtocol.Trim();
        }
        if (request.Domains != null)
        {
            var normalized = NormalizeStringList(request.Domains);
            if (normalized.Count == 0)
            {
                return ServiceResult<SiteDetailDto>.Fail(ErrorCodes.MissingParam, "domain_name_required");
            }
            site.Domain = EncodeStringList(normalized);
        }
        if (request.Enable.HasValue)
        {
            site.Enable = request.Enable.Value;
            site.State = request.Enable.Value ? "running" : "stop";
        }
        if (!string.IsNullOrWhiteSpace(request.State))
        {
            var state = request.State.Trim().ToLowerInvariant();
            if (AllowedSiteStates.Contains(state))
            {
                site.State = state;
            }
        }

        if (hasCcDefault)
        {
            site.CcDefaultRule = (int)ccDefaultFromSettings;
        }
        if (hasBlacklist)
        {
            site.BlackIp = EncodeStringList(blacklistFromSettings);
        }
        if (hasWhitelist)
        {
            site.WhiteIp = EncodeStringList(whitelistFromSettings);
        }

        site.UpdateAt = DateTime.Now;

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Updateable(site).ExecuteCommandAsync();

            if (request.Settings != null)
            {
                await SaveSiteSettingsAsync(site.Id, settings);
            }

            if (request.GroupIds != null || request.GroupId != null)
            {
                await _db.Deleteable<MergeSiteGroup>().Where(r => r.SiteId == site.Id).ExecuteCommandAsync();
                var groupIds = ResolveGroupIds(request.GroupIds, request.GroupId ?? 0);
                if (groupIds.Count > 0)
                {
                    var relations = groupIds
                        .Where(gid => gid > 0)
                        .Distinct()
                        .Select(gid => new MergeSiteGroup { SiteId = site.Id, GroupId = (int)gid })
                        .ToList();
                    if (relations.Count > 0)
                    {
                        await _db.Insertable(relations).ExecuteCommandAsync();
                    }
                }
            }
        });

        await _configVersionService.BumpAsync("site", new[] { (long)site.Id }, cancellationToken);

        await RefreshSiteCnameHostnameAsync(site, null, null);
        await TrySyncUserDnsRecordsAsync(oldSnapshot, site);
        if (ShouldResyncSiteCname(oldSnapshot, site))
        {
            await ResyncSiteCnameForSiteAsync(site);
        }

        var items = await BuildSiteListItemsAsync(new List<Site> { site });
        var detail = items.Count > 0 ? ToDetailDto(items[0]) : new SiteDetailDto { Id = site.Id };
        return ServiceResult<SiteDetailDto>.Ok(detail);
    }

    private static readonly HashSet<string> AllowedSiteStates = new(StringComparer.OrdinalIgnoreCase)
    {
        "running",
        "stop",
        "locked",
        "site_locked",
        "traffic_limit",
        "conn_limit",
        "expired",
        "timeout"
    };

    private static Site CloneSiteForCompare(Site site)
    {
        return new Site
        {
            Id = site.Id,
            Uid = site.Uid,
            UserPackage = site.UserPackage,
            NodeGroupId = site.NodeGroupId,
            BackupNodeGroup = site.BackupNodeGroup,
            EnableBackupGroup = site.EnableBackupGroup,
            DnsProviderId = site.DnsProviderId,
            CnameDomain = site.CnameDomain,
            CnameMode = site.CnameMode,
            CnameHostname = site.CnameHostname,
            Domain = site.Domain,
            State = site.State,
            Enable = site.Enable
        };
    }

    private async Task<Dictionary<string, object?>> LoadSiteSettingsAsync(long siteId)
    {
        var map = await LoadSiteSettingsMapAsync(new List<long> { siteId });
        if (map.TryGetValue(siteId, out var settings))
        {
            return settings;
        }

        return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
    }

    private async Task<List<long>> FilterSiteGroupIdsForUserAsync(IReadOnlyList<long> groupIds, long userId)
    {
        if (groupIds.Count == 0 || userId <= 0)
        {
            return new List<long>();
        }

        return await _db.Queryable<SiteGroup>()
            .Where(g => groupIds.Contains(g.Id) && g.Uid == (int)userId)
            .Select(g => (long)g.Id)
            .ToListAsync();
    }
}
