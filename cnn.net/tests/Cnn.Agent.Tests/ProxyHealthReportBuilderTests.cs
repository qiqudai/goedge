using Cnn.Agent.Proxy;
using Yarp.ReverseProxy.Configuration;
using Yarp.ReverseProxy.Model;
using Xunit;

namespace Cnn.Agent.Tests;

public sealed class ProxyHealthReportBuilderTests
{
    [Fact]
    public void Build_ExportsClusterPolicyAndDestinationHealth()
    {
        var builder = new ProxyHealthReportBuilder();

        var clusterConfig = new ClusterConfig
        {
            ClusterId = "cluster:upstream-1:roundrobin",
            LoadBalancingPolicy = "RoundRobin",
            Metadata = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
            {
                ["upstream_key"] = "upstream-1",
                ["ConsecutiveFailuresHealthPolicy.Threshold"] = "4",
                ["TransportFailureRateHealthPolicy.RateLimit"] = "0.45"
            },
            HealthCheck = new HealthCheckConfig
            {
                Active = new ActiveHealthCheckConfig
                {
                    Enabled = true,
                    Path = "/healthz",
                    Query = "deep=1",
                    Interval = TimeSpan.FromSeconds(12),
                    Timeout = TimeSpan.FromSeconds(2)
                },
                Passive = new PassiveHealthCheckConfig
                {
                    Enabled = true,
                    ReactivationPeriod = TimeSpan.FromSeconds(35)
                },
                AvailableDestinationsPolicy = "HealthyAndUnknown"
            },
            Destinations = new Dictionary<string, DestinationConfig>(StringComparer.OrdinalIgnoreCase)
            {
                ["dest-0"] = new() { Address = "http://127.0.0.1:8080", Metadata = new Dictionary<string, string>{{"node_id","101"}} },
                ["dest-1"] = new() { Address = "http://127.0.0.1:8081", Metadata = new Dictionary<string, string>{{"node_id","102"}} }
            }
        };

        var snapshot = new ProxySnapshot(
            Version: 12,
            Hash: "abc123",
            CreatedAt: DateTimeOffset.UtcNow,
            Routes: [],
            Clusters: [clusterConfig],
            DomainCount: 1,
            UpstreamCount: 1,
            IsFallbackMode: false);

        var clusterState = new ClusterState(
            clusterConfig.ClusterId,
            new ClusterModel(clusterConfig, new HttpMessageInvoker(new HttpClientHandler())));

        var destination0 = new DestinationState("dest-0", new DestinationModel(clusterConfig.Destinations!["dest-0"]))
        {
            ConcurrentRequestCount = 3
        };
        destination0.Health.Active = DestinationHealth.Healthy;
        destination0.Health.Passive = DestinationHealth.Unknown;

        var destination1 = new DestinationState("dest-1", new DestinationModel(clusterConfig.Destinations!["dest-1"]));
        destination1.Health.Active = DestinationHealth.Unknown;
        destination1.Health.Passive = DestinationHealth.Unhealthy;

        clusterState.Destinations["dest-0"] = destination0;
        clusterState.Destinations["dest-1"] = destination1;

        var report = builder.Build(snapshot, [clusterState], activeProbeInitialized: true);

        Assert.Equal(12, report.SnapshotVersion);
        Assert.True(report.ActiveProbeInitialized);
        var cluster = Assert.Single(report.Clusters);
        Assert.Equal("upstream-1", cluster.UpstreamKey);
        Assert.Equal(4, cluster.ActiveUnhealthyThreshold);
        Assert.Equal(0.45, cluster.PassiveFailureRateLimit);
        Assert.Equal(2, cluster.DestinationCount);
        Assert.Equal(1, cluster.AvailableDestinationCount);
        Assert.Equal(1, cluster.UnavailableDestinationCount);

        var first = Assert.Single(cluster.Destinations, x => x.DestinationId == "dest-0");
        Assert.Equal(101, first.NodeId);
        Assert.Equal("healthy", first.Status);
        Assert.True(first.IsAvailable);
        Assert.Equal(3, first.ConcurrentRequests);

        var second = Assert.Single(cluster.Destinations, x => x.DestinationId == "dest-1");
        Assert.Equal(102, second.NodeId);
        Assert.Equal("unhealthy(passive)", second.Status);
        Assert.False(second.IsAvailable);
    }
}
