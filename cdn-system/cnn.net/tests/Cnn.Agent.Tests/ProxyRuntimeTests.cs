using Cnn.Agent.Proxy;
using Cnn.Common.Contracts.Agent;
using Microsoft.Extensions.Logging.Abstractions;
using Yarp.ReverseProxy.Health;
using Xunit;

namespace Cnn.Agent.Tests;

public sealed class ProxyRuntimeTests
{
    [Fact]
    public void TryApply_UpdatesHealthCheckConfigAcrossVersions()
    {
        var runtime = CreateRuntime();

        var v1 = BuildConfig(
            version: 1,
            activePath: "health/v1?probe=fast",
            activeInterval: "10s",
            activeTimeout: "2s",
            activeThreshold: 2,
            passiveEnabled: true,
            passiveReactivation: "20s",
            passiveRateLimit: 0.35,
            availablePolicy: "healthy_or_panic");

        var applyV1 = runtime.TryApply(v1);
        Assert.True(applyV1.Success);

        var snapshotV1 = runtime.GetCurrent();
        Assert.Equal(1, snapshotV1.Version);
        var clusterV1 = Assert.Single(snapshotV1.Clusters);
        Assert.Equal("/health/v1", clusterV1.HealthCheck!.Active!.Path);
        Assert.Equal("2", clusterV1.Metadata![ConsecutiveFailuresHealthPolicyOptions.ThresholdMetadataName]);
        Assert.Equal("probe=fast", clusterV1.HealthCheck.Active.Query);
        Assert.Equal("0.35", clusterV1.Metadata![TransportFailureRateHealthPolicyOptions.FailureRateLimitMetadataName]);
        Assert.Equal(TimeSpan.FromSeconds(20), clusterV1.HealthCheck.Passive!.ReactivationPeriod);

        var v2 = BuildConfig(
            version: 2,
            activePath: "/health/v2?probe=deep",
            activeInterval: "15s",
            activeTimeout: "4s",
            activeThreshold: 5,
            passiveEnabled: true,
            passiveReactivation: "45s",
            passiveRateLimit: 0.55,
            availablePolicy: "healthy_and_unknown");

        var applyV2 = runtime.TryApply(v2);
        Assert.True(applyV2.Success);

        var snapshotV2 = runtime.GetCurrent();
        Assert.Equal(2, snapshotV2.Version);
        var clusterV2 = Assert.Single(snapshotV2.Clusters);
        Assert.Equal("/health/v2", clusterV2.HealthCheck!.Active!.Path);
        Assert.Equal("probe=deep", clusterV2.HealthCheck.Active.Query);
        Assert.Equal("5", clusterV2.Metadata![ConsecutiveFailuresHealthPolicyOptions.ThresholdMetadataName]);
        Assert.Equal("0.55", clusterV2.Metadata![TransportFailureRateHealthPolicyOptions.FailureRateLimitMetadataName]);
        Assert.Equal(TimeSpan.FromSeconds(45), clusterV2.HealthCheck.Passive!.ReactivationPeriod);
        Assert.Equal("HealthyAndUnknown", clusterV2.HealthCheck.AvailableDestinationsPolicy);
    }

    [Fact]
    public void TryApply_SkipsStaleVersionAndKeepsCurrentSnapshot()
    {
        var runtime = CreateRuntime();

        var v2 = BuildConfig(version: 2, activePath: "/health/v2");
        var applyV2 = runtime.TryApply(v2);
        Assert.True(applyV2.Success);

        var staleV1 = BuildConfig(version: 1, activePath: "/health/stale");
        var applyStale = runtime.TryApply(staleV1);

        Assert.True(applyStale.Success);
        Assert.Equal("skipped", applyStale.Status);

        var current = runtime.GetCurrent();
        Assert.Equal(2, current.Version);
        Assert.Equal("/health/v2", Assert.Single(current.Clusters).HealthCheck!.Active!.Path);
    }

    [Fact]
    public void TryApply_RejectsInvalidConfigWithoutMutatingCurrent()
    {
        var runtime = CreateRuntime();

        var valid = BuildConfig(version: 1, activePath: "/health/ok");
        var ok = runtime.TryApply(valid);
        Assert.True(ok.Success);

        var invalid = BuildConfig(version: 2, activePath: "https://bad.example/health");
        var failed = runtime.TryApply(invalid);

        Assert.False(failed.Success);
        Assert.Contains("upstream_active_health_check_path", failed.Error ?? string.Empty);

        var current = runtime.GetCurrent();
        Assert.Equal(1, current.Version);
        Assert.Equal("/health/ok", Assert.Single(current.Clusters).HealthCheck!.Active!.Path);
    }

    private static EdgeProxyRuntime CreateRuntime()
    {
        return new EdgeProxyRuntime(
            new DynamicProxyConfigProvider(),
            new EdgeConfigToYarpCompiler(),
            new ProxyConfigValidator(),
            NullLogger<EdgeProxyRuntime>.Instance);
    }

    private static EdgeConfigDto BuildConfig(
        long version,
        string activePath,
        string activeInterval = "10s",
        string activeTimeout = "3s",
        int? activeThreshold = null,
        bool passiveEnabled = true,
        string passiveReactivation = "30s",
        double? passiveRateLimit = null,
        string availablePolicy = "healthy_or_panic")
    {
        return new EdgeConfigDto
        {
            Version = version,
            Domains =
            [
                new EdgeDomainDto
                {
                    Name = "example.com",
                    UpstreamKey = "upstream-1",
                    UpstreamActiveHealthCheck = true,
                    UpstreamActiveHealthCheckPath = activePath,
                    UpstreamActiveHealthCheckInterval = activeInterval,
                    UpstreamActiveHealthCheckTimeout = activeTimeout,
                    UpstreamActiveHealthCheckThreshold = activeThreshold,
                    UpstreamPassiveHealthCheck = passiveEnabled,
                    UpstreamPassiveHealthCheckReactivation = passiveReactivation,
                    UpstreamPassiveHealthCheckRateLimit = passiveRateLimit,
                    UpstreamAvailableDestinationsPolicy = availablePolicy
                }
            ],
            Upstreams =
            [
                new EdgeUpstreamDto
                {
                    Id = "upstream-1",
                    Targets =
                    [
                        new EdgeUpstreamTargetDto { Addr = "127.0.0.1:8080", Weight = 100 },
                        new EdgeUpstreamTargetDto { Addr = "127.0.0.1:8081", Weight = 100 }
                    ]
                }
            ]
        };
    }
}
