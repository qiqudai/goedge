using System.Collections.Concurrent;
using Cnn.Common.Contracts;
using Microsoft.Extensions.Options;

namespace Cnn.Agent.Cache;

public sealed class CacheRuntimeStore
{
    private readonly CachePolicyResolver _globalResolver;
    private readonly ConcurrentDictionary<string, CachePolicyResolver> _resolverByHost = new(StringComparer.OrdinalIgnoreCase);

    public CacheRuntimeStore(IOptions<CacheOptions> options)
    {
        _globalResolver = new CachePolicyResolver(Options.Create(options.Value ?? new CacheOptions()));
    }

    public CacheDecision Resolve(HttpContext context)
    {
        var host = context.Request.Host.Host ?? string.Empty;
        if (!string.IsNullOrWhiteSpace(host) && _resolverByHost.TryGetValue(host, out var resolver))
        {
            return resolver.Resolve(context);
        }

        return _globalResolver.Resolve(context);
    }

    public void UpsertSiteConfig(CacheSiteConfigDto config)
    {
        if (config == null)
        {
            return;
        }

        var hosts = ResolveHosts(config);
        if (hosts.Count == 0)
        {
            return;
        }

        var options = BuildOptions(config);
        var resolver = new CachePolicyResolver(Options.Create(options));

        foreach (var host in hosts)
        {
            _resolverByHost[host] = resolver;
        }
    }

    private static List<string> ResolveHosts(CacheSiteConfigDto config)
    {
        var hosts = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        if (config.Hosts != null)
        {
            foreach (var host in config.Hosts)
            {
                if (!string.IsNullOrWhiteSpace(host))
                {
                    hosts.Add(host.Trim());
                }
            }
        }

        if (config.Rules != null)
        {
            foreach (var rule in config.Rules)
            {
                if (!string.IsNullOrWhiteSpace(rule.Host))
                {
                    hosts.Add(rule.Host.Trim());
                }
            }
        }

        return hosts.ToList();
    }

    private static CacheOptions BuildOptions(CacheSiteConfigDto config)
    {
        var options = new CacheOptions
        {
            Profiles = new Dictionary<string, CacheProfileOptions>(StringComparer.OrdinalIgnoreCase),
            Rules = new List<CacheRuleOptions>()
        };

        if (config.Profiles != null)
        {
            foreach (var pair in config.Profiles)
            {
                if (string.IsNullOrWhiteSpace(pair.Key) || pair.Value == null)
                {
                    continue;
                }

                options.Profiles[pair.Key] = new CacheProfileOptions
                {
                    Ttl = pair.Value.Ttl,
                    IgnoreQuery = pair.Value.IgnoreQuery,
                    ForceCache = pair.Value.ForceCache,
                    QueryIgnoreList = pair.Value.QueryIgnoreList
                };
            }
        }

        if (config.Rules != null)
        {
            foreach (var rule in config.Rules)
            {
                options.Rules.Add(new CacheRuleOptions
                {
                    Host = rule.Host,
                    PathPrefix = rule.PathPrefix,
                    PathRegex = rule.PathRegex,
                    Profile = rule.Profile
                });
            }
        }

        return options;
    }
}
