using Cnn.Agent.Proxy;
using Cnn.Common.Contracts.Agent;
using Yarp.ReverseProxy.Health;
using Xunit;

namespace Cnn.Agent.Tests;

public sealed class EdgeConfigToYarpCompilerTests
{
    [Fact]
    public void Compile_SplitsActiveHealthCheckPathAndQuery()
    {
        var compiler = new EdgeConfigToYarpCompiler();
        var config = BuildConfig(new EdgeDomainDto
        {
            Name = "example.com",
            UpstreamKey = "upstream-1",
            UpstreamActiveHealthCheck = true,
            UpstreamActiveHealthCheckPath = "healthz/ping?deep=1&mode=full",
            UpstreamActiveHealthCheckInterval = "12s",
            UpstreamActiveHealthCheckTimeout = "2s",
            UpstreamActiveHealthCheckThreshold = 4,
            UpstreamPassiveHealthCheck = true,
            UpstreamPassiveHealthCheckReactivation = "35s",
            UpstreamPassiveHealthCheckRateLimit = 0.45,
            UpstreamAvailableDestinationsPolicy = "healthy_and_unknown"
        });

        var snapshot = compiler.Compile(config);
        var cluster = Assert.Single(snapshot.Clusters);

        Assert.NotNull(cluster.HealthCheck?.Active);
        Assert.Equal("/healthz/ping", cluster.HealthCheck!.Active!.Path);
        Assert.Equal("deep=1&mode=full", cluster.HealthCheck.Active.Query);
        Assert.Equal(TimeSpan.FromSeconds(12), cluster.HealthCheck.Active.Interval);
        Assert.Equal(TimeSpan.FromSeconds(2), cluster.HealthCheck.Active.Timeout);

        Assert.NotNull(cluster.Metadata);
        Assert.Equal("/healthz/ping", cluster.Metadata!["upstream_active_health_check_path"]);
        Assert.Equal("deep=1&mode=full", cluster.Metadata["upstream_active_health_check_query"]);
        Assert.Equal("4", cluster.Metadata["upstream_active_health_check_threshold"]);
        Assert.Equal("4", cluster.Metadata[ConsecutiveFailuresHealthPolicyOptions.ThresholdMetadataName]);
        Assert.Equal("35000", cluster.Metadata["upstream_passive_health_check_reactivation"]);
        Assert.Equal("0.45", cluster.Metadata["upstream_passive_health_check_rate_limit"]);
        Assert.Equal("0.45", cluster.Metadata[TransportFailureRateHealthPolicyOptions.FailureRateLimitMetadataName]);
        Assert.Equal("HealthyAndUnknown", cluster.Metadata["upstream_available_destinations_policy"]);
    }

    [Fact]
    public void Compile_UsesDefaultHealthPathWhenMissing()
    {
        var compiler = new EdgeConfigToYarpCompiler();
        var config = BuildConfig(new EdgeDomainDto
        {
            Name = "example.com",
            UpstreamKey = "upstream-1",
            UpstreamActiveHealthCheck = true
        });

        var snapshot = compiler.Compile(config);
        var cluster = Assert.Single(snapshot.Clusters);

        Assert.NotNull(cluster.HealthCheck?.Active);
        Assert.Equal("/", cluster.HealthCheck!.Active!.Path);
        Assert.Null(cluster.HealthCheck.Active.Query);
        Assert.NotNull(cluster.Metadata);
        Assert.False(cluster.Metadata!.ContainsKey("upstream_active_health_check_query"));
    }

    private static EdgeConfigDto BuildConfig(EdgeDomainDto domain)
    {
        return new EdgeConfigDto
        {
            Version = 1,
            Domains = new List<EdgeDomainDto> { domain },
            Upstreams = new List<EdgeUpstreamDto>
            {
                new()
                {
                    Id = "upstream-1",
                    Targets = new List<EdgeUpstreamTargetDto>
                    {
                        new() { Addr = "127.0.0.1:8080", Weight = 100 }
                    }
                }
            }
        };
    }
}
