using Yarp.ReverseProxy.Configuration;

namespace Cnn.Agent.Proxy;

public sealed record ProxySnapshot(
    long Version,
    string Hash,
    DateTimeOffset CreatedAt,
    IReadOnlyList<RouteConfig> Routes,
    IReadOnlyList<ClusterConfig> Clusters,
    int DomainCount,
    int UpstreamCount,
    bool IsFallbackMode)
{
    public static ProxySnapshot CreateFallback()
    {
        var route = new RouteConfig
        {
            RouteId = "route:fallback",
            ClusterId = "cluster:fallback",
            Match = new RouteMatch
            {
                Path = "/{**catch-all}"
            }
        };

        var cluster = new ClusterConfig
        {
            ClusterId = "cluster:fallback",
            Destinations = new Dictionary<string, DestinationConfig>(StringComparer.OrdinalIgnoreCase)
            {
                ["fallback-1"] = new DestinationConfig
                {
                    Address = "http://127.0.0.1:9/"
                }
            }
        };

        return new ProxySnapshot(
            Version: 0,
            Hash: "fallback",
            CreatedAt: DateTimeOffset.UtcNow,
            Routes: new[] { route },
            Clusters: new[] { cluster },
            DomainCount: 0,
            UpstreamCount: 0,
            IsFallbackMode: true);
    }
}
