using System.Threading;
using Cnn.Agent.Config;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Security;

public interface IEdgeDomainResolver
{
    bool TryResolve(string host, out EdgeDomainDto domain);
}

public sealed class EdgeDomainResolver : IEdgeDomainResolver
{
    private readonly EdgeConfigStore _edgeConfigStore;
    private readonly object _reloadLock = new();
    private EdgeConfigDto? _lastConfigRef;
    private Dictionary<string, EdgeDomainDto> _exactHosts = new(StringComparer.OrdinalIgnoreCase);
    private List<(string Suffix, EdgeDomainDto Domain)> _wildcardHosts = [];

    public EdgeDomainResolver(EdgeConfigStore edgeConfigStore)
    {
        _edgeConfigStore = edgeConfigStore;
    }

    public bool TryResolve(string host, out EdgeDomainDto domain)
    {
        domain = default!;
        var normalizedHost = NormalizeHost(host);
        if (string.IsNullOrWhiteSpace(normalizedHost))
        {
            return false;
        }

        EnsureCompiled();

        var exact = Volatile.Read(ref _exactHosts);
        if (exact.TryGetValue(normalizedHost, out var matched))
        {
            domain = matched;
            return true;
        }

        var wildcard = Volatile.Read(ref _wildcardHosts);
        foreach (var item in wildcard)
        {
            if (normalizedHost.EndsWith(item.Suffix, StringComparison.OrdinalIgnoreCase))
            {
                domain = item.Domain;
                return true;
            }
        }

        return false;
    }

    private void EnsureCompiled()
    {
        var config = _edgeConfigStore.Current;
        if (config == null)
        {
            return;
        }

        if (ReferenceEquals(config, _lastConfigRef))
        {
            return;
        }

        lock (_reloadLock)
        {
            if (ReferenceEquals(config, _lastConfigRef))
            {
                return;
            }

            var exact = new Dictionary<string, EdgeDomainDto>(StringComparer.OrdinalIgnoreCase);
            var wildcard = new List<(string Suffix, EdgeDomainDto Domain)>();

            foreach (var domain in config.Domains)
            {
                foreach (var host in ExpandHosts(domain.Name))
                {
                    if (host.StartsWith("*.", StringComparison.Ordinal))
                    {
                        var suffix = host.Substring(1);
                        if (suffix.Length > 1)
                        {
                            wildcard.Add((suffix, domain));
                        }
                    }
                    else if (!exact.ContainsKey(host))
                    {
                        exact[host] = domain;
                    }
                }
            }

            Volatile.Write(ref _exactHosts, exact);
            Volatile.Write(ref _wildcardHosts, wildcard);
            _lastConfigRef = config;
        }
    }

    private static IEnumerable<string> ExpandHosts(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            yield break;
        }

        var parts = raw.Split([',', ';', '\n', '\r', '\t', ' '], StringSplitOptions.RemoveEmptyEntries);
        foreach (var part in parts)
        {
            var normalized = NormalizeHost(part);
            if (!string.IsNullOrWhiteSpace(normalized))
            {
                yield return normalized;
            }
        }
    }

    private static string NormalizeHost(string? host)
    {
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        return host.Trim().TrimEnd('.').ToLowerInvariant();
    }
}
