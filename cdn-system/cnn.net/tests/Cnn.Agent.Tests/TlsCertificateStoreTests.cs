using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using Cnn.Agent.Config;
using Cnn.Agent.Security;
using Cnn.Common.Contracts.Agent;
using Microsoft.Extensions.Configuration;
using Xunit;

namespace Cnn.Agent.Tests;

public sealed class TlsCertificateStoreTests
{
    [Fact]
    public void GetForHost_ReturnsEmergencyFallback_WhenNoCertConfigured()
    {
        using var fixture = new RuntimeFixture();
        var store = new TlsCertificateStore(fixture.Paths);

        store.Reload(new EdgeConfigDto
        {
            Version = 1,
            Domains = new List<EdgeDomainDto>(),
            Upstreams = new List<EdgeUpstreamDto>()
        });

        var cert = store.GetForHost("unknown.example.com");

        Assert.NotNull(cert);
        Assert.Contains("cnn-agent-fallback", cert!.Subject, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void GetForHost_UsesDomainCert_WhenAvailable_AndFallbackForUnknownHost()
    {
        using var fixture = new RuntimeFixture();
        var store = new TlsCertificateStore(fixture.Paths);
        var (certPem, keyPem) = CreatePemPair("example.com");

        store.Reload(new EdgeConfigDto
        {
            Version = 1,
            Domains = new List<EdgeDomainDto>
            {
                new()
                {
                    Name = "example.com",
                    SslCertData = certPem,
                    SslKeyData = keyPem
                }
            },
            Upstreams = new List<EdgeUpstreamDto>()
        });

        var exact = store.GetForHost("example.com");
        var other = store.GetForHost("other.example.com");

        Assert.NotNull(exact);
        Assert.NotNull(other);
        Assert.NotEqual(other!.Thumbprint, exact!.Thumbprint);
        Assert.Contains("CN=example.com", exact.Subject, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public void Reload_DoesNotBreakWhenFallbackPemInvalid()
    {
        using var fixture = new RuntimeFixture();
        var store = new TlsCertificateStore(fixture.Paths);

        store.Reload(new EdgeConfigDto
        {
            Version = 1,
            FallbackCertData = "bad cert",
            FallbackKeyData = "bad key",
            Domains = new List<EdgeDomainDto>(),
            Upstreams = new List<EdgeUpstreamDto>()
        });

        var cert = store.GetForHost("any.host");

        Assert.NotNull(cert);
        Assert.Contains("cnn-agent-fallback", cert!.Subject, StringComparison.OrdinalIgnoreCase);
    }

    private static (string CertPem, string KeyPem) CreatePemPair(string commonName)
    {
        using var rsa = RSA.Create(2048);
        var req = new CertificateRequest(
            $"CN={commonName}",
            rsa,
            HashAlgorithmName.SHA256,
            RSASignaturePadding.Pkcs1);
        req.CertificateExtensions.Add(new X509BasicConstraintsExtension(false, false, 0, false));
        req.CertificateExtensions.Add(new X509KeyUsageExtension(
            X509KeyUsageFlags.DigitalSignature | X509KeyUsageFlags.KeyEncipherment,
            critical: true));
        req.CertificateExtensions.Add(new X509SubjectKeyIdentifierExtension(req.PublicKey, false));

        using var cert = req.CreateSelfSigned(DateTimeOffset.UtcNow.AddDays(-1), DateTimeOffset.UtcNow.AddDays(30));
        return (cert.ExportCertificatePem(), rsa.ExportPkcs8PrivateKeyPem());
    }

    private sealed class RuntimeFixture : IDisposable
    {
        private readonly string _tempRoot;

        public RuntimeFixture()
        {
            _tempRoot = Path.Combine(Path.GetTempPath(), "cnn-agent-tests", Guid.NewGuid().ToString("N"));
            Directory.CreateDirectory(_tempRoot);
            var runtimeRoot = Path.Combine(_tempRoot, "runtime");
            Directory.CreateDirectory(runtimeRoot);

            var cfg = new ConfigurationBuilder()
                .AddInMemoryCollection(new Dictionary<string, string?>
                {
                    ["Agent:RuntimeRoot"] = runtimeRoot
                })
                .Build();

            Paths = new AgentRuntimePaths(cfg);
        }

        public AgentRuntimePaths Paths { get; }

        public void Dispose()
        {
            try
            {
                if (Directory.Exists(_tempRoot))
                {
                    Directory.Delete(_tempRoot, recursive: true);
                }
            }
            catch
            {
                // ignore cleanup errors in tests
            }
        }
    }
}
