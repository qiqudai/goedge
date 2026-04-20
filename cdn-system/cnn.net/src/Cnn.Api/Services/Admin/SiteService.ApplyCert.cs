using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed partial class SiteService
{
    public async Task<ServiceResult<SiteApplyCertResult>> ApplyCertAsync(
        SiteApplyCertRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        request ??= new SiteApplyCertRequest();
        if (request.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<SiteApplyCertResult>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        var ids = request.Ids.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return ServiceResult<SiteApplyCertResult>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<SiteApplyCertResult>.Fail(ErrorCodes.PermissionDenied);
            }

            var allowed = await FilterSiteIdsForUserAsync(ids, userId.Value);
            if (allowed.Count != ids.Count)
            {
                return ServiceResult<SiteApplyCertResult>.Fail(ErrorCodes.PermissionDenied);
            }
            ids = allowed;
        }

        var sites = await _db.Queryable<Site>().Where(s => ids.Contains(s.Id)).ToListAsync();
        if (sites.Count == 0)
        {
            return ServiceResult<SiteApplyCertResult>.Fail(ErrorCodes.NotFound, "not_found");
        }

        var createdIds = new List<long>();
        var skipped = new List<SiteApplyCertSkipItem>();
        var appliedSiteIds = new List<long>();

        foreach (var site in sites)
        {
            var domains = DomainParser.ParseDomains(site.Domain).Where(d => !string.IsNullOrWhiteSpace(d)).ToList();
            if (domains.Count == 0)
            {
                return ServiceResult<SiteApplyCertResult>.Fail(ErrorCodes.MissingParam, "domain_name_required");
            }

            var settings = await LoadSiteSettingsAsync(site.Id);
            if (IsSiteHttpsOn(site, settings))
            {
                skipped.Add(new SiteApplyCertSkipItem
                {
                    SiteId = site.Id,
                    Domain = domains[0],
                    Reason = "site_https_enabled"
                });
                continue;
            }

            var (certType, dnsApi) = await ResolveCertDefaultsAsync(site.Uid ?? 0, cancellationToken);
            if (HasWildcardDomain(domains))
            {
                if (dnsApi <= 0)
                {
                    skipped.Add(new SiteApplyCertSkipItem
                    {
                        SiteId = site.Id,
                        Domain = domains[0],
                        Reason = "dnsapi_required_for_wildcard"
                    });
                    continue;
                }
            }
            else
            {
                dnsApi = 0;
            }

            if (!await EnsureNoExistingCertAsync(site.Uid ?? 0, domains))
            {
                return ServiceResult<SiteApplyCertResult>.Fail(ErrorCodes.AlreadyExists, "already_exists");
            }

            var createRequest = new CertCreateRequest
            {
                UserId = site.Uid ?? 0,
                Type = certType,
                Domain = string.Join(',', domains),
                DnsApi = dnsApi,
                AutoRenew = true,
                Description = $"site_id:{site.Id}"
            };

            var created = await _certService.CreateAsync(createRequest, site.Uid ?? 0, true, cancellationToken);
            if (!created.Success || created.Data == null)
            {
                return ServiceResult<SiteApplyCertResult>.Fail(created.ErrorCode, created.MessageKey);
            }

            var certId = created.Data.Id;
            var httpsCfg = GetSubMap(settings, "https");
            httpsCfg["enable"] = true;
            httpsCfg["certificate_id"] = certId;

            if (string.IsNullOrWhiteSpace(site.HttpsListen))
            {
                site.HttpsListen = EncodeStringList(new[] { "443" });
            }

            site.UpdateAt = DateTime.Now;
            await _db.Updateable(site).ExecuteCommandAsync();
            await SaveSiteSettingsAsync(site.Id, settings);

            createdIds.Add(certId);
            appliedSiteIds.Add(site.Id);
        }

        if (appliedSiteIds.Count > 0)
        {
            await _configVersionService.BumpAsync("site", appliedSiteIds.Select(id => (long)id).ToList(), cancellationToken);
        }

        return ServiceResult<SiteApplyCertResult>.Ok(new SiteApplyCertResult(createdIds, skipped));
    }

    private static bool IsSiteHttpsOn(Site site, Dictionary<string, object?> settings)
    {
        var httpsOn = !string.IsNullOrWhiteSpace(site.HttpsListen);
        if (settings.TryGetValue("https", out var httpsRaw) && httpsRaw is Dictionary<string, object?> httpsCfg)
        {
            if (httpsCfg.TryGetValue("enable", out var enable))
            {
                httpsOn = ParseBool(enable, httpsOn);
            }
        }
        return httpsOn;
    }

    private static bool HasWildcardDomain(IReadOnlyList<string> domains)
    {
        foreach (var domain in domains)
        {
            if (domain.Trim().StartsWith("*.", StringComparison.Ordinal))
            {
                return true;
            }
        }
        return false;
    }

    private async Task<(string CertType, int DnsApi)> ResolveCertDefaultsAsync(long userId, CancellationToken cancellationToken)
    {
        var result = await _certService.GetDefaultSettingsAsync(userId, true, cancellationToken);
        if (result.Success && result.Data != null)
        {
            var type = NormalizeCertType(result.Data.Type);
            return (type, result.Data.DnsApi);
        }

        return ("letsencrypt", 0);
    }

    private static string NormalizeCertType(string? raw)
    {
        var type = raw?.Trim().ToLowerInvariant() ?? string.Empty;
        return string.IsNullOrWhiteSpace(type) ? "letsencrypt" : type;
    }

    private async Task<bool> EnsureNoExistingCertAsync(long userId, IReadOnlyList<string> domains)
    {
        if (userId <= 0 || domains.Count == 0)
        {
            return true;
        }

        foreach (var domain in domains)
        {
            var trimmed = domain.Trim();
            if (string.IsNullOrWhiteSpace(trimmed))
            {
                continue;
            }

            var exists = await _db.Queryable<Cert>()
                .Where(c => c.Uid == (int)userId && SqlFunc.Contains(c.Domain, trimmed))
                .AnyAsync();
            if (exists)
            {
                return false;
            }
        }

        return true;
    }
}
