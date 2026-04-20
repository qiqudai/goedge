using Cnn.Agent.Cache;
using Cnn.Common.Contracts.Agent;
using Xunit;

namespace Cnn.Agent.Tests;

public sealed class CachePurgePlannerTests
{
    [Fact]
    public void BuildDirectoryPrefix_ReturnsHostForRootPath()
    {
        var uri = new Uri("https://Example.COM/");

        var prefix = CachePurgePlanner.BuildDirectoryPrefix(uri);

        Assert.Equal("example.com", prefix);
    }

    [Fact]
    public void BuildDirectoryPrefix_NormalizesPathAndTrimsTrailingSlash()
    {
        var uri = new Uri("https://example.com/static/assets/?v=1");

        var prefix = CachePurgePlanner.BuildDirectoryPrefix(uri);

        Assert.Equal("example.com/static/assets", prefix);
    }

    [Fact]
    public void ResolveHostsForSiteIds_CollectsHostsByUpstreamSiteId()
    {
        var config = new EdgeConfigDto
        {
            Version = 1,
            Domains =
            [
                new EdgeDomainDto
                {
                    Name = "a.example.com, b.example.com",
                    UpstreamKey = "upstream_11"
                },
                new EdgeDomainDto
                {
                    Name = "c.example.com",
                    UpstreamKey = "upstream_22"
                }
            ],
            Upstreams = new List<EdgeUpstreamDto>()
        };

        var hosts = CachePurgePlanner.ResolveHostsForSiteIds(config, new HashSet<long> { 11 });

        Assert.Contains("a.example.com", hosts);
        Assert.Contains("b.example.com", hosts);
        Assert.DoesNotContain("c.example.com", hosts);
    }

    [Theory]
    [InlineData("upstream_1", true, 1)]
    [InlineData("UPSTREAM_23", true, 23)]
    [InlineData("upstream_x", false, 0)]
    [InlineData("cluster_1", false, 0)]
    public void TryExtractSiteIdFromUpstreamKey_ParsesExpectedValues(string key, bool expectedOk, long expectedId)
    {
        var ok = CachePurgePlanner.TryExtractSiteIdFromUpstreamKey(key, out var siteId);

        Assert.Equal(expectedOk, ok);
        Assert.Equal(expectedId, siteId);
    }
}
