using System.Globalization;
using Yarp.ReverseProxy;
using Yarp.ReverseProxy.Configuration;
using Yarp.ReverseProxy.Health;
using Yarp.ReverseProxy.Model;

namespace Cnn.Agent.Proxy;

public sealed class ProxyHealthReportBuilder
{
    public ProxyHealthReport Build(ProxySnapshot snapshot, IEnumerable<ClusterState> runtimeClusters, bool activeProbeInitialized)
    {
        var configuredById = (snapshot.Clusters ?? [])
            .ToDictionary(cluster => cluster.ClusterId, StringComparer.OrdinalIgnoreCase);

        var reports = new List<ProxyHealthClusterReport>();
        foreach (var runtimeCluster in runtimeClusters.OrderBy(x => x.ClusterId, StringComparer.OrdinalIgnoreCase))
        {
            configuredById.TryGetValue(runtimeCluster.ClusterId, out var configuredCluster);
            var clusterConfig = runtimeCluster.Model?.Config ?? configuredCluster;
            var metadata = clusterConfig?.Metadata;

            var destinationConfigById = clusterConfig?.Destinations ?? new Dictionary<string, DestinationConfig>(StringComparer.OrdinalIgnoreCase);
            var destinationReports = new List<ProxyHealthDestinationReport>();
            foreach (var entry in runtimeCluster.Destinations.OrderBy(kv => kv.Key, StringComparer.OrdinalIgnoreCase))
            {
                var destinationId = entry.Key;
                var destination = entry.Value;
                destinationConfigById.TryGetValue(destinationId, out var configuredDestination);

                var modelConfig = destination.Model?.Config;
                var active = destination.Health.Active;
                var passive = destination.Health.Passive;
                var isAvailable = active != DestinationHealth.Unhealthy && passive != DestinationHealth.Unhealthy;

                destinationReports.Add(new ProxyHealthDestinationReport(
                    DestinationId: destinationId,
                    NodeId: TryGetNodeId(modelConfig?.Metadata) ?? TryGetNodeId(configuredDestination?.Metadata),
                    Address: modelConfig?.Address ?? configuredDestination?.Address,
                    Active: active.ToString(),
                    Passive: passive.ToString(),
                    Status: DescribeStatus(active, passive),
                    IsAvailable: isAvailable,
                    ConcurrentRequests: destination.ConcurrentRequestCount));
            }

            var availableCount = destinationReports.Count(x => x.IsAvailable);
            reports.Add(new ProxyHealthClusterReport(
                ClusterId: runtimeCluster.ClusterId,
                UpstreamKey: TryGetString(metadata, "upstream_key"),
                LoadBalancingPolicy: clusterConfig?.LoadBalancingPolicy,
                ActiveProbePath: clusterConfig?.HealthCheck?.Active?.Path,
                ActiveProbeQuery: clusterConfig?.HealthCheck?.Active?.Query,
                ActiveProbeIntervalMs: ToMs(clusterConfig?.HealthCheck?.Active?.Interval),
                ActiveProbeTimeoutMs: ToMs(clusterConfig?.HealthCheck?.Active?.Timeout),
                ActiveUnhealthyThreshold: TryGetInt(metadata, ConsecutiveFailuresHealthPolicyOptions.ThresholdMetadataName),
                PassiveReactivationMs: ToMs(clusterConfig?.HealthCheck?.Passive?.ReactivationPeriod),
                PassiveFailureRateLimit: TryGetDouble(metadata, TransportFailureRateHealthPolicyOptions.FailureRateLimitMetadataName),
                AvailableDestinationsPolicy: clusterConfig?.HealthCheck?.AvailableDestinationsPolicy,
                DestinationCount: destinationReports.Count,
                AvailableDestinationCount: availableCount,
                UnavailableDestinationCount: destinationReports.Count - availableCount,
                Destinations: destinationReports));
        }

        return new ProxyHealthReport(
            SnapshotVersion: snapshot.Version,
            SnapshotHash: snapshot.Hash,
            IsFallbackMode: snapshot.IsFallbackMode,
            ActiveProbeInitialized: activeProbeInitialized,
            ClusterCount: reports.Count,
            Clusters: reports);
    }

    private static int? ToMs(TimeSpan? value)
    {
        return value.HasValue ? (int)Math.Round(value.Value.TotalMilliseconds) : null;
    }

    private static string DescribeStatus(DestinationHealth active, DestinationHealth passive)
    {
        if (active == DestinationHealth.Unhealthy && passive == DestinationHealth.Unhealthy)
        {
            return "unhealthy(active+passive)";
        }

        if (active == DestinationHealth.Unhealthy)
        {
            return "unhealthy(active)";
        }

        if (passive == DestinationHealth.Unhealthy)
        {
            return "unhealthy(passive)";
        }

        if (active == DestinationHealth.Healthy || passive == DestinationHealth.Healthy)
        {
            return "healthy";
        }

        return "unknown";
    }

    private static int? TryGetInt(IReadOnlyDictionary<string, string>? metadata, string key)
    {
        var raw = TryGetString(metadata, key);
        if (raw == null)
        {
            return null;
        }

        return int.TryParse(raw, NumberStyles.Integer, CultureInfo.InvariantCulture, out var value)
            ? value
            : null;
    }

    private static double? TryGetDouble(IReadOnlyDictionary<string, string>? metadata, string key)
    {
        var raw = TryGetString(metadata, key);
        if (raw == null)
        {
            return null;
        }

        return double.TryParse(raw, NumberStyles.Float, CultureInfo.InvariantCulture, out var value)
            ? value
            : null;
    }

    private static long? TryGetNodeId(IReadOnlyDictionary<string, string>? metadata)
    {
        var raw = TryGetString(metadata, "node_id");
        if (raw == null)
        {
            return null;
        }

        return long.TryParse(raw, NumberStyles.Integer, CultureInfo.InvariantCulture, out var value)
            ? value
            : null;
    }

    private static string? TryGetString(IReadOnlyDictionary<string, string>? metadata, string key)
    {
        if (metadata == null)
        {
            return null;
        }

        return metadata.TryGetValue(key, out var value) && !string.IsNullOrWhiteSpace(value)
            ? value
            : null;
    }
}

public sealed record ProxyHealthReport(
    long SnapshotVersion,
    string SnapshotHash,
    bool IsFallbackMode,
    bool ActiveProbeInitialized,
    int ClusterCount,
    IReadOnlyList<ProxyHealthClusterReport> Clusters);

public sealed record ProxyHealthClusterReport(
    string ClusterId,
    string? UpstreamKey,
    string? LoadBalancingPolicy,
    string? ActiveProbePath,
    string? ActiveProbeQuery,
    int? ActiveProbeIntervalMs,
    int? ActiveProbeTimeoutMs,
    int? ActiveUnhealthyThreshold,
    int? PassiveReactivationMs,
    double? PassiveFailureRateLimit,
    string? AvailableDestinationsPolicy,
    int DestinationCount,
    int AvailableDestinationCount,
    int UnavailableDestinationCount,
    IReadOnlyList<ProxyHealthDestinationReport> Destinations);

public sealed record ProxyHealthDestinationReport(
    string DestinationId,
    long? NodeId,
    string? Address,
    string Active,
    string Passive,
    string Status,
    bool IsAvailable,
    int ConcurrentRequests);
