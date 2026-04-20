using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class SiteService
{
    public async Task<ServiceResult<bool>> BatchUpdateAsync(
        SiteBatchUpdateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        request ??= new SiteBatchUpdateRequest();
        if (request.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        var ids = request.Ids.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }

            var allowed = await FilterSiteIdsForUserAsync(ids, userId.Value);
            if (allowed.Count != ids.Count)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            ids = allowed;
        }

        if (!isAdmin && request.UserPackageId is > 0)
        {
            if (!await EnsureUserPackageOwnershipAsync(userId ?? 0, request.UserPackageId.Value))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
        }

        if (!isAdmin && (request.GroupIds != null || request.GroupId != null))
        {
            var groupIds = ResolveGroupIds(request.GroupIds, request.GroupId ?? 0);
            if (groupIds.Count > 0)
            {
                var allowedGroups = await FilterSiteGroupIdsForUserAsync(groupIds, userId ?? 0);
                if (allowedGroups.Count != groupIds.Count)
                {
                    return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
                }
                request.GroupIds = allowedGroups;
                request.GroupId = allowedGroups.Count > 0 ? allowedGroups[0] : 0;
            }
        }

        if (request.CnameDomain != null)
        {
            var normalized = DomainHelper.NormalizeDomainInput(request.CnameDomain);
            if (string.IsNullOrWhiteSpace(normalized) || !DomainHelper.IsValidDomain(normalized))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_domain");
            }

            var cname = await _db.Queryable<CnameDomains>().Where(c => c.Domain == normalized).FirstAsync();
            if (cname == null)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "cname_domain_not_found");
            }

            request.CnameDomain = normalized;
        }

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
            request.Settings = SiteSettingsNormalizer.Normalize(request.Settings);
        }

        var blacklistFromInput = !string.IsNullOrWhiteSpace(request.BlackIp)
            ? SplitFields(request.BlackIp!)
            : new List<string>();
        var whitelistFromInput = !string.IsNullOrWhiteSpace(request.WhiteIp)
            ? SplitFields(request.WhiteIp!)
            : new List<string>();

        await _db.Ado.UseTranAsync(async () =>
        {
            var sites = await _db.Queryable<Site>().Where(s => ids.Contains(s.Id)).ToListAsync();
            foreach (var site in sites)
            {
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
                if (!string.IsNullOrWhiteSpace(request.BalanceWay))
                {
                    site.BalanceWay = request.BalanceWay.Trim();
                }
                if (!string.IsNullOrWhiteSpace(request.BackendProtocol))
                {
                    site.BackendProtocol = request.BackendProtocol.Trim();
                }
                if (request.Backends != null)
                {
                    site.Backend = EncodeStringList(request.Backends);
                }
                if (request.CcDefaultRule.HasValue)
                {
                    site.CcDefaultRule = (int)request.CcDefaultRule.Value;
                    if (request.Settings != null)
                    {
                        SetCcDefaultRuleInSettings(request.Settings, request.CcDefaultRule.Value);
                    }
                }
                else if (hasCcDefault)
                {
                    site.CcDefaultRule = (int)ccDefaultFromSettings;
                }
                if (request.BlackIp != null)
                {
                    site.BlackIp = EncodeStringList(blacklistFromInput);
                }
                else if (hasBlacklist)
                {
                    site.BlackIp = EncodeStringList(blacklistFromSettings);
                }
                if (request.WhiteIp != null)
                {
                    site.WhiteIp = EncodeStringList(whitelistFromInput);
                }
                else if (hasWhitelist)
                {
                    site.WhiteIp = EncodeStringList(whitelistFromSettings);
                }
                if (request.BlockRegion != null)
                {
                    site.BlockRegion = request.BlockRegion;
                }
                if (request.RegionId.HasValue)
                {
                    site.RegionId = (int)request.RegionId.Value;
                }
                if (request.NodeGroupId.HasValue)
                {
                    site.NodeGroupId = (int)request.NodeGroupId.Value;
                }
                if (request.BackupNodeGroupId.HasValue)
                {
                    site.BackupNodeGroup = (int)request.BackupNodeGroupId.Value;
                }
                if (request.EnableBackupGroup.HasValue)
                {
                    site.EnableBackupGroup = request.EnableBackupGroup.Value;
                }

                if (request.CnameDomain != null)
                {
                    site.CnameDomain = request.CnameDomain;
                }
                if (request.CnameMode != null)
                {
                    site.CnameMode = request.CnameMode;
                }

                if (request.CnameDomain != null || request.CnameMode != null)
                {
                    var pkgId = request.UserPackageId is > 0 ? request.UserPackageId.Value : site.UserPackage ?? 0;
                    if (pkgId > 0)
                    {
                        var pkg = await _db.Queryable<UserPackage>().Where(p => p.Id == pkgId).FirstAsync();
                        if (pkg != null)
                        {
                            site.CnameHostname = ComputeSiteCnameHostname(site, pkg, request.CnameMode, request.CnameDomain);
                        }
                    }
                }

                site.UpdateAt = DateTime.Now;

                await _db.Updateable(site).ExecuteCommandAsync();

                if (request.Settings != null)
                {
                    var existing = await LoadSiteSettingsAsync(site.Id);
                    var merged = MergeSettingsMaps(existing, request.Settings);
                    merged = SiteSettingsNormalizer.Normalize(merged);
                    await SaveSiteSettingsAsync(site.Id, merged);
                }
            }

            if (request.GroupIds != null || request.GroupId != null)
            {
                await _db.Deleteable<MergeSiteGroup>().Where(r => r.SiteId.HasValue && ids.Contains(r.SiteId.Value)).ExecuteCommandAsync();

                var groupIds = ResolveGroupIds(request.GroupIds, request.GroupId ?? 0);
                if (groupIds.Count > 0)
                {
                    var relations = new List<MergeSiteGroup>();
                    foreach (var siteId in ids)
                    {
                        foreach (var gid in groupIds)
                        {
                            if (gid <= 0)
                            {
                                continue;
                            }
                            relations.Add(new MergeSiteGroup { SiteId = (int)siteId, GroupId = (int)gid });
                        }
                    }
                    if (relations.Count > 0)
                    {
                        await _db.Insertable(relations).ExecuteCommandAsync();
                    }
                }
            }
        });

        await _configVersionService.BumpAsync("site", ids, cancellationToken);

        var needResync = request.UserPackageId != null || request.CnameDomain != null || request.CnameMode != null ||
                         request.NodeGroupId != null || request.BackupNodeGroupId != null || request.EnableBackupGroup != null;
        if (needResync)
        {
            var sites = await _db.Queryable<Site>().Where(s => ids.Contains(s.Id)).ToListAsync();
            foreach (var site in sites)
            {
                await ResyncSiteCnameForSiteAsync(site);
            }
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<SiteBatchActionResult>> BatchActionAsync(
        SiteBatchActionRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        request ??= new SiteBatchActionRequest();
        if (request.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<SiteBatchActionResult>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        var ids = request.Ids.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return ServiceResult<SiteBatchActionResult>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<SiteBatchActionResult>.Fail(ErrorCodes.PermissionDenied);
            }

            var allowed = await FilterSiteIdsForUserAsync(ids, userId.Value);
            if (allowed.Count != ids.Count)
            {
                return ServiceResult<SiteBatchActionResult>.Fail(ErrorCodes.PermissionDenied);
            }
            ids = allowed;
        }

        var action = request.Action?.Trim().ToLowerInvariant() ?? string.Empty;
        switch (action)
        {
            case "enable":
            {
                var taskResult = await _resourceActionRequestService.RequestAsync(
                    SiteActionCommandFactory.CreateStatusChange(ids, true, userId, userId),
                    cancellationToken);
                if (!taskResult.Success)
                {
                    return ServiceResult<SiteBatchActionResult>.Fail(taskResult.ErrorCode, taskResult.MessageKey);
                }

                return ServiceResult<SiteBatchActionResult>.Ok(new SiteBatchActionResult(taskResult.Data!.TaskId));
            }
            case "disable":
            {
                var taskResult = await _resourceActionRequestService.RequestAsync(
                    SiteActionCommandFactory.CreateStatusChange(ids, false, userId, userId),
                    cancellationToken);
                if (!taskResult.Success)
                {
                    return ServiceResult<SiteBatchActionResult>.Fail(taskResult.ErrorCode, taskResult.MessageKey);
                }

                return ServiceResult<SiteBatchActionResult>.Ok(new SiteBatchActionResult(taskResult.Data!.TaskId));
            }
            case "delete":
            {
                var sites = await _db.Queryable<Site>().Where(s => ids.Contains(s.Id)).ToListAsync();
                foreach (var site in sites)
                {
                    if (site.Enable == true)
                    {
                        return ServiceResult<SiteBatchActionResult>.Fail(ErrorCodes.PreconditionFailed, "precondition_failed");
                    }
                }

                var taskResult = await _resourceActionRequestService.RequestAsync(
                    SiteActionCommandFactory.CreateDelete(ids, userId, userId),
                    cancellationToken);
                if (!taskResult.Success)
                {
                    return ServiceResult<SiteBatchActionResult>.Fail(taskResult.ErrorCode, taskResult.MessageKey);
                }

                return ServiceResult<SiteBatchActionResult>.Ok(new SiteBatchActionResult(taskResult.Data!.TaskId));
            }
            case "unlock":
                break;
            case "clear_cache":
            {
                var payload = new { action = "clear_cache", site_ids = ids };
                var task = new TaskEntity
                {
                    Type = "clear_cache",
                    Name = "Clear Cache",
                    Data = JsonSerializer.Serialize(payload, JsonOptions),
                    Res = userId is > 0 ? JsonSerializer.Serialize(new { user_id = userId.Value }, JsonOptions) : string.Empty,
                    State = "waiting",
                    Enable = true,
                    CreateAt = DateTime.Now
                };

                var taskId = await _db.Insertable(task).ExecuteReturnIdentityAsync();
                return ServiceResult<SiteBatchActionResult>.Ok(new SiteBatchActionResult(taskId));
            }
            default:
                return ServiceResult<SiteBatchActionResult>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        if (action != "unlock")
        {
            await _configVersionService.BumpAsync("site", ids, cancellationToken);
        }

        return ServiceResult<SiteBatchActionResult>.Ok(new SiteBatchActionResult(0));
    }

    private async Task<List<long>> FilterSiteIdsForUserAsync(IReadOnlyList<long> ids, long userId)
    {
        if (ids.Count == 0 || userId <= 0)
        {
            return new List<long>();
        }

        return await _db.Queryable<Site>()
            .Where(s => ids.Contains(s.Id) && s.Uid == (int)userId)
            .Select(s => (long)s.Id)
            .ToListAsync();
    }
}
