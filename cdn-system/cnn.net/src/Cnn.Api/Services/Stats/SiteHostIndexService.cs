using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Stats;

public sealed class SiteHostIndex
{
    public HostFilter Filter { get; } = new();
    public Dictionary<long, HostFilter> SiteFilters { get; } = new();
}

public interface ISiteHostIndexService
{
    Task<SiteHostIndex> LoadAsync(long userId, CancellationToken cancellationToken);
}

public sealed class SiteHostIndexService : ISiteHostIndexService
{
    private readonly ISqlSugarClient _db;

    public SiteHostIndexService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<SiteHostIndex> LoadAsync(long userId, CancellationToken cancellationToken)
    {
        var query = _db.Queryable<Site>().Select(s => new { s.Id, s.Uid, s.Domain });
        if (userId > 0)
        {
            query = query.Where(s => s.Uid == userId);
        }

        var sites = await query.ToListAsync();
        var index = new SiteHostIndex();
        var seenExact = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var seenWildcard = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        foreach (var site in sites)
        {
            foreach (var raw in DomainParser.ParseDomains(site.Domain))
            {
                var (exact, wildcard) = SplitHostPattern(raw);
                if (string.IsNullOrWhiteSpace(exact) && string.IsNullOrWhiteSpace(wildcard))
                {
                    continue;
                }

                if (!string.IsNullOrWhiteSpace(exact))
                {
                    if (seenExact.Add(exact))
                    {
                        index.Filter.Exact.Add(exact);
                    }
                    AddSiteFilter(index.SiteFilters, site.Id, exact, string.Empty);
                    continue;
                }

                if (!string.IsNullOrWhiteSpace(wildcard))
                {
                    if (seenWildcard.Add(wildcard))
                    {
                        index.Filter.Wildcards.Add(wildcard);
                    }
                    AddSiteFilter(index.SiteFilters, site.Id, string.Empty, wildcard);
                }
            }
        }

        index.Filter.Exact.Sort(StringComparer.OrdinalIgnoreCase);
        index.Filter.Wildcards.Sort(StringComparer.OrdinalIgnoreCase);
        return index;
    }

    private static void AddSiteFilter(Dictionary<long, HostFilter> map, long siteId, string exact, string wildcard)
    {
        if (siteId <= 0)
        {
            return;
        }

        if (!map.TryGetValue(siteId, out var filter))
        {
            filter = new HostFilter();
            map[siteId] = filter;
        }

        if (!string.IsNullOrWhiteSpace(exact))
        {
            filter.Exact.Add(exact);
        }

        if (!string.IsNullOrWhiteSpace(wildcard))
        {
            filter.Wildcards.Add(wildcard);
        }
    }

    private static (string Exact, string Wildcard) SplitHostPattern(string raw)
    {
        var host = raw?.Trim().ToLowerInvariant() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(host))
        {
            return (string.Empty, string.Empty);
        }

        host = host.Replace("http://", string.Empty).Replace("https://", string.Empty);
        var slash = host.IndexOf('/');
        if (slash >= 0)
        {
            host = host[..slash];
        }

        host = host.TrimEnd('.');
        if (host.Contains('*'))
        {
            host = host.TrimStart('*', '.');
            var colon = host.IndexOf(':');
            if (colon >= 0)
            {
                host = host[..colon];
            }
            host = host.TrimEnd('.');
            return (string.Empty, host.Trim());
        }

        var portIndex = host.IndexOf(':');
        if (portIndex >= 0)
        {
            host = host[..portIndex];
        }

        host = host.TrimEnd('.');
        return (host.Trim(), string.Empty);
    }
}
