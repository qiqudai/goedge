using System.Security.Authentication;
using System.Net.Security;
using Cnn.Agent.Security;
using Cnn.Common.Contracts.Agent;
using Xunit;

namespace Cnn.Agent.Tests;

public sealed class TlsRuntimePolicyStoreTests
{
    [Fact]
    public void Reload_AggregatesProtocolsAndOcspFromDomains()
    {
        var store = new TlsRuntimePolicyStore();
        var config = new EdgeConfigDto
        {
            Version = 1,
            Domains =
            [
                new EdgeDomainDto
                {
                    Name = "a.example.com",
                    HttpsSslProtocols = "TLSv1.2",
                    HttpsOcsp = false
                },
                new EdgeDomainDto
                {
                    Name = "b.example.com",
                    HttpsSslProtocols = "TLSv1.3",
                    HttpsOcsp = true
                }
            ],
            Upstreams = new List<EdgeUpstreamDto>()
        };

        store.Reload(config);
        var policy = store.GetCurrent();

        Assert.True((policy.SslProtocols & SslProtocols.Tls12) != 0);
        Assert.True((policy.SslProtocols & SslProtocols.Tls13) != 0);
        Assert.True(policy.CheckCertificateRevocation);
    }

    [Fact]
    public void Reload_ParsesCipherAliasesIntoRuntimePolicy()
    {
        var store = new TlsRuntimePolicyStore();
        var config = new EdgeConfigDto
        {
            Version = 1,
            Domains =
            [
                new EdgeDomainDto
                {
                    Name = "example.com",
                    HttpsSslCiphers = "ECDHE-ECDSA-AES128-GCM-SHA256:TLS_AES_256_GCM_SHA384"
                }
            ],
            Upstreams = new List<EdgeUpstreamDto>()
        };

        store.Reload(config);
        var policy = store.GetCurrent();

        Assert.Contains(TlsCipherSuite.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, policy.CipherSuites);
        Assert.Contains(TlsCipherSuite.TLS_AES_256_GCM_SHA384, policy.CipherSuites);

        if (!OperatingSystem.IsWindows())
        {
            Assert.NotNull(policy.CipherSuitesPolicy);
        }
    }

    [Fact]
    public void Reload_IgnoresInvalidProtocolAndCipherTokens()
    {
        var store = new TlsRuntimePolicyStore();
        var config = new EdgeConfigDto
        {
            Version = 1,
            Domains =
            [
                new EdgeDomainDto
                {
                    Name = "example.com",
                    HttpsSslProtocols = "SSLv3,TLSv1.1",
                    HttpsSslCiphers = "INVALID_CIPHER"
                }
            ],
            Upstreams = new List<EdgeUpstreamDto>()
        };

        store.Reload(config);
        var policy = store.GetCurrent();

        Assert.Equal(SslProtocols.None, policy.SslProtocols);
        Assert.Empty(policy.CipherSuites);
    }
}
