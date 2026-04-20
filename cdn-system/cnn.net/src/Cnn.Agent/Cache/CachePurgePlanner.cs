using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Cache;

public static class CachePurgePlanner
{
    public static string? BuildDirectoryPrefix(Uri uri)
    {
        if (uri == null || string.IsNullOrWhiteSpace(uri.Host))
        {
            return null;
        }

        var host = uri.Host.Trim().ToLowerInvariant();
        var path = uri.AbsolutePath?.Trim() ?? "/";
        if (path == "/" || path.Length == 0)
        {
            return host;
        }

        var normalized = path.TrimStart('/');
        if (normalized.Length == 0)
        {
            return host;
        }

        return $"{host}/{normalized.TrimEnd('/')}";
    }

    public static IReadOnlySet<string> ResolveHostsForSiteIds(EdgeConfigDto? config, IReadOnlyCollection<long> siteIds)
    {
        var result = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        if (config == null || siteIds == null || siteIds.Count == 0)
        {
            return result;
        }

        foreach (var domain in config.Domains)
        {
            if (!TryExtractSiteIdFromUpstreamKey(domain.UpstreamKey, out var siteId) || !siteIds.Contains(siteId))
            {
                continue;
            }

            foreach (var host in SplitDomainHosts(domain.Name))
            {
                result.Add(host);
            }
        }

        return result;
    }

    public static bool TryExtractSiteIdFromUpstreamKey(string? upstreamKey, out long siteId)
    {
        siteId = 0;
        if (string.IsNullOrWhiteSpace(upstreamKey))
        {
            return false;
        }

        const string prefix = "upstream_";
        var value = upstreamKey.Trim();
        if (!value.StartsWith(prefix, StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        return long.TryParse(value[prefix.Length..], out siteId) && siteId > 0;
    }

    private static IEnumerable<string> SplitDomainHosts(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            yield break;
        }

        var parts = raw.Split([',', ';', '\n', '\r', '\t', ' '], StringSplitOptions.RemoveEmptyEntries);
        foreach (var part in parts)
        {
            var host = part.Trim().TrimEnd('.').ToLowerInvariant();
            if (!string.IsNullOrWhiteSpace(host))
            {
                yield return host;
            }
        }
    }
}
