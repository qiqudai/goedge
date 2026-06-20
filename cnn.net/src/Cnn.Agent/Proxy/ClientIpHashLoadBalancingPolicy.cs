using System.Net;
using System.Text;
using Yarp.ReverseProxy.LoadBalancing;
using Yarp.ReverseProxy.Model;

namespace Cnn.Agent.Proxy;

public sealed class ClientIpHashLoadBalancingPolicy : ILoadBalancingPolicy
{
    public const string PolicyName = "ClientIpHash";

    public string Name => PolicyName;

    public DestinationState? PickDestination(
        HttpContext context,
        ClusterState cluster,
        IReadOnlyList<DestinationState> availableDestinations)
    {
        if (availableDestinations == null || availableDestinations.Count == 0)
        {
            return null;
        }

        if (availableDestinations.Count == 1)
        {
            return availableDestinations[0];
        }

        var hash = ComputeStableHash(context, cluster);
        var index = (int)(hash % (uint)availableDestinations.Count);
        return availableDestinations[index];
    }

    private static uint ComputeStableHash(HttpContext context, ClusterState cluster)
    {
        var clientKey = ResolveClientKey(context);
        var clusterId = cluster.ClusterId ?? "cluster";
        var raw = $"{clusterId}|{clientKey}";
        return Fnv1a32(raw);
    }

    private static string ResolveClientKey(HttpContext context)
    {
        if (context.Request.Headers.TryGetValue("X-Forwarded-For", out var xffValues))
        {
            var first = xffValues.ToString().Split(',', 2, StringSplitOptions.TrimEntries)[0];
            if (TryNormalizeIp(first, out var xffIp))
            {
                return xffIp;
            }
        }

        if (context.Connection.RemoteIpAddress != null)
        {
            return NormalizeIp(context.Connection.RemoteIpAddress);
        }

        if (!string.IsNullOrWhiteSpace(context.TraceIdentifier))
        {
            return context.TraceIdentifier;
        }

        return "unknown";
    }

    private static bool TryNormalizeIp(string? raw, out string normalized)
    {
        normalized = string.Empty;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        if (!IPAddress.TryParse(raw.Trim(), out var ip))
        {
            return false;
        }

        normalized = NormalizeIp(ip);
        return normalized.Length > 0;
    }

    private static string NormalizeIp(IPAddress ip)
    {
        if (ip.IsIPv4MappedToIPv6)
        {
            ip = ip.MapToIPv4();
        }

        return ip.ToString();
    }

    private static uint Fnv1a32(string input)
    {
        const uint offset = 2166136261;
        const uint prime = 16777619;

        var hash = offset;
        var bytes = Encoding.UTF8.GetBytes(input);
        foreach (var b in bytes)
        {
            hash ^= b;
            hash *= prime;
        }

        return hash == 0 ? 1u : hash;
    }
}
