using System.Net;
using Cnn.Common.Contracts;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface IDomainUsageService
{
    Task<long> FindDefaultUserPackageIdAsync(long userId);
    Task<ServiceResult<DomainUsageDto>> GetUsageAsync(long userId, long userPackageId, CancellationToken cancellationToken);
}

public sealed class DomainUsageService : IDomainUsageService
{
    private readonly ISqlSugarClient _db;

    public DomainUsageService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<DomainUsageDto>> GetUsageAsync(long userId, long userPackageId, CancellationToken cancellationToken)
    {
        if (userId <= 0 || userPackageId <= 0)
        {
            return ServiceResult<DomainUsageDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var limits = await LoadDomainLimitsAsync(userPackageId);
        if (limits == null)
        {
            return ServiceResult<DomainUsageDto>.Fail(ErrorCodes.NotFound);
        }

        var domainSet = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var mainSet = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var sites = await _db.Queryable<Site>().Where(s => s.Uid == userId).ToListAsync();
        foreach (var site in sites)
        {
            foreach (var domain in DomainParser.ParseDomains(site.Domain))
            {
                var normalized = DomainParser.NormalizeDomain(domain);
                if (string.IsNullOrWhiteSpace(normalized))
                {
                    continue;
                }
                domainSet.Add(normalized);
                var mainKey = MainDomainKey(normalized);
                if (!string.IsNullOrWhiteSpace(mainKey))
                {
                    mainSet.Add(mainKey);
                }
            }
        }

        var (domainLimit, mainDomainLimit) = limits.Value;
        var usage = new DomainUsageDto
        {
            DomainLimit = domainLimit,
            MainDomainLimit = mainDomainLimit,
            TotalDomains = domainSet.Count,
            TotalMainDomains = mainSet.Count
        };

        if (domainLimit > 0 && usage.TotalDomains > domainLimit)
        {
            usage.Exceeded = true;
        }
        else if (mainDomainLimit > 0 && usage.TotalMainDomains > mainDomainLimit)
        {
            usage.Exceeded = true;
        }

        return ServiceResult<DomainUsageDto>.Ok(usage);
    }

    private async Task<(int DomainLimit, int MainDomainLimit)?> LoadDomainLimitsAsync(long userPackageId)
    {
        var pack = await _db.Queryable<UserPackage>().Where(p => p.Id == userPackageId).FirstAsync();
        if (pack == null)
        {
            return null;
        }

        var totalLimit = pack.Domain ?? 0;
        var mainLimit = pack.MainDomainLimit ?? 0;

        if (mainLimit <= 0)
        {
            var cfg = await _db.Queryable<Config>()
                .Where(c => c.Type == SettingsConstants.UserPackageConfigType && c.ScopeName == SettingsConstants.UserPackageScope && c.ScopeId == userPackageId && c.Name == SettingsConstants.MainDomainLimitName)
                .FirstAsync();
            if (cfg != null)
            {
                mainLimit = ParseIntConfig(cfg.Value);
            }
        }

        return (totalLimit, mainLimit);
    }

    private static string NormalizeDomain(string? input)
    {
        return DomainParser.NormalizeDomain(input);
    }

    private static string MainDomainKey(string domain)
    {
        if (string.IsNullOrWhiteSpace(domain))
        {
            return string.Empty;
        }

        if (IPAddress.TryParse(domain, out _))
        {
            return domain;
        }

        var parts = domain.Split('.', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length < 2)
        {
            return domain;
        }

        return $"{parts[^2]}.{parts[^1]}";
    }

    private static int ParseIntConfig(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return 0;
        }
        return int.TryParse(value.Trim(), out var parsed) ? parsed : 0;
    }

    public async Task<long> FindDefaultUserPackageIdAsync(long userId)
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
}
