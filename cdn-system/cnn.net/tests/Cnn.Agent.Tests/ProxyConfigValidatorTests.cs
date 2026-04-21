using Cnn.Agent.Proxy;
using Cnn.Common.Contracts.Agent;
using Xunit;

namespace Cnn.Agent.Tests;

public sealed class ProxyConfigValidatorTests
{
    [Fact]
    public void Validate_AllowsActiveHealthPathWithQuery()
    {
        var validator = new ProxyConfigValidator();
        var config = BuildConfig(new EdgeDomainDto
        {
            Name = "example.com",
            UpstreamKey = "upstream-1",
            UpstreamActiveHealthCheck = true,
            UpstreamActiveHealthCheckPath = "healthz/ping?probe=1"
        });

        var result = validator.Validate(config);

        Assert.True(result.Success);
    }

    [Fact]
    public void Validate_RejectsInvalidActiveHealthPath()
    {
        var validator = new ProxyConfigValidator();
        var config = BuildConfig(new EdgeDomainDto
        {
            Name = "example.com",
            UpstreamKey = "upstream-1",
            UpstreamActiveHealthCheck = true,
            UpstreamActiveHealthCheckPath = "https://evil.example/health"
        });

        var result = validator.Validate(config);

        Assert.False(result.Success);
        Assert.Contains("upstream_active_health_check_path", result.Error ?? string.Empty);
    }

    [Fact]
    public void Validate_RejectsInvalidActiveHealthThreshold()
    {
        var validator = new ProxyConfigValidator();
        var config = BuildConfig(new EdgeDomainDto
        {
            Name = "example.com",
            UpstreamKey = "upstream-1",
            UpstreamActiveHealthCheck = true,
            UpstreamActiveHealthCheckThreshold = 0
        });

        var result = validator.Validate(config);

        Assert.False(result.Success);
        Assert.Contains("upstream_active_health_check_threshold", result.Error ?? string.Empty);
    }

    [Fact]
    public void Validate_RejectsInvalidPassiveHealthRateLimit()
    {
        var validator = new ProxyConfigValidator();
        var config = BuildConfig(new EdgeDomainDto
        {
            Name = "example.com",
            UpstreamKey = "upstream-1",
            UpstreamPassiveHealthCheck = true,
            UpstreamPassiveHealthCheckRateLimit = 1.2
        });

        var result = validator.Validate(config);

        Assert.False(result.Success);
        Assert.Contains("upstream_passive_health_check_rate_limit", result.Error ?? string.Empty);
    }

    [Fact]
    public void Validate_AllowsNegativeVersionFromGoHash()
    {
        var validator = new ProxyConfigValidator();
        var config = BuildConfig(new EdgeDomainDto
        {
            Name = "example.com",
            UpstreamKey = "upstream-1"
        });
        config.Version = -4092667256324154400;

        var result = validator.Validate(config);

        Assert.True(result.Success);
    }

    [Fact]
    public void Validate_AllowsFollowOriginProtocol()
    {
        var validator = new ProxyConfigValidator();
        var config = BuildConfig(new EdgeDomainDto
        {
            Name = "example.com",
            UpstreamKey = "upstream-1",
            OriginProtocol = "follow"
        });

        var result = validator.Validate(config);

        Assert.True(result.Success);
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
