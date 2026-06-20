using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Text;
using System.Threading;
using Cnn.Agent.Config;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Security;

public interface ITlsCertificateStore
{
    X509Certificate2? GetForHost(string? host);
    void Reload(EdgeConfigDto config);
}

public sealed class TlsCertificateStore : ITlsCertificateStore
{
    private readonly AgentRuntimePaths _paths;
    private readonly X509Certificate2 _emergencyFallback;
    private readonly object _reloadLock = new();
    private TlsSnapshot _snapshot = TlsSnapshot.Empty;

    public TlsCertificateStore(AgentRuntimePaths paths)
    {
        _paths = paths;
        _emergencyFallback = CreateEmergencyFallbackCertificate();
    }

    public X509Certificate2? GetForHost(string? host)
    {
        var snapshot = Volatile.Read(ref _snapshot);
        var normalized = NormalizeHost(host);
        if (!string.IsNullOrWhiteSpace(normalized) && snapshot.Exact.TryGetValue(normalized, out var exact))
        {
            return exact;
        }

        if (!string.IsNullOrWhiteSpace(normalized))
        {
            foreach (var item in snapshot.Wildcards)
            {
                if (normalized.EndsWith(item.Suffix, StringComparison.OrdinalIgnoreCase))
                {
                    return item.Certificate;
                }
            }
        }

        return snapshot.Fallback ?? _emergencyFallback;
    }

    public void Reload(EdgeConfigDto config)
    {
        if (config == null)
        {
            return;
        }

        lock (_reloadLock)
        {
            var previous = Volatile.Read(ref _snapshot);

            var exact = new Dictionary<string, X509Certificate2>(StringComparer.OrdinalIgnoreCase);
            var wildcards = new List<(string Suffix, X509Certificate2 Certificate)>();
            var cacheEntries = new Dictionary<string, X509Certificate2>(StringComparer.Ordinal);

            foreach (var domain in config.Domains)
            {
                var cert = LoadDomainCertificate(domain, previous, cacheEntries);
                if (cert == null)
                {
                    continue;
                }

                foreach (var host in ExpandHosts(domain.Name))
                {
                    if (host.StartsWith("*.", StringComparison.Ordinal))
                    {
                        var suffix = host.Substring(1);
                        if (suffix.Length > 1)
                        {
                            wildcards.Add((suffix, cert));
                        }
                        continue;
                    }

                    exact[host] = cert;
                }
            }

            wildcards.Sort(static (a, b) => b.Suffix.Length.CompareTo(a.Suffix.Length));
            var fallback = LoadFallbackCertificate(config, previous, cacheEntries);

            var next = new TlsSnapshot(exact, wildcards, fallback, cacheEntries);
            Volatile.Write(ref _snapshot, next);

            DisposeObsoleteCertificates(previous, next);
        }
    }

    private X509Certificate2? LoadDomainCertificate(
        EdgeDomainDto domain,
        TlsSnapshot previous,
        Dictionary<string, X509Certificate2> nextCache)
    {
        if (!string.IsNullOrWhiteSpace(domain.SslCertData) && !string.IsNullOrWhiteSpace(domain.SslKeyData))
        {
            var key = BuildInlineSourceKey("domain:inline", domain.SslCertData, domain.SslKeyData);
            return GetOrLoad(key, previous, nextCache, () => TryLoadPemPair(domain.SslCertData, domain.SslKeyData));
        }

        if (!string.IsNullOrWhiteSpace(domain.SslCertPath) && !string.IsNullOrWhiteSpace(domain.SslKeyPath))
        {
            var key = BuildFileSourceKey("domain:file", domain.SslCertPath, domain.SslKeyPath);
            return GetOrLoad(key, previous, nextCache, () => TryLoadPemFiles(domain.SslCertPath, domain.SslKeyPath));
        }

        return null;
    }

    private X509Certificate2? LoadFallbackCertificate(
        EdgeConfigDto config,
        TlsSnapshot previous,
        Dictionary<string, X509Certificate2> nextCache)
    {
        if (!string.IsNullOrWhiteSpace(config.FallbackCertData) && !string.IsNullOrWhiteSpace(config.FallbackKeyData))
        {
            var key = BuildInlineSourceKey("fallback:inline", config.FallbackCertData, config.FallbackKeyData);
            return GetOrLoad(key, previous, nextCache, () => TryLoadPemPair(config.FallbackCertData, config.FallbackKeyData));
        }

        var pemPath = Path.Combine(_paths.CertDir, "fallback.pem");
        var keyPath = Path.Combine(_paths.CertDir, "fallback.key");
        var fallbackKey = BuildFileSourceKey("fallback:file", pemPath, keyPath);
        return GetOrLoad(fallbackKey, previous, nextCache, () => TryLoadPemFiles(pemPath, keyPath));
    }

    private static X509Certificate2? GetOrLoad(
        string key,
        TlsSnapshot previous,
        Dictionary<string, X509Certificate2> nextCache,
        Func<X509Certificate2?> loader)
    {
        if (nextCache.TryGetValue(key, out var current))
        {
            return current;
        }

        if (previous.CacheEntries.TryGetValue(key, out var reused))
        {
            nextCache[key] = reused;
            return reused;
        }

        var loaded = loader();
        if (loaded != null)
        {
            nextCache[key] = loaded;
        }

        return loaded;
    }

    private static X509Certificate2? TryLoadPemPair(string? certPem, string? keyPem)
    {
        if (string.IsNullOrWhiteSpace(certPem) || string.IsNullOrWhiteSpace(keyPem))
        {
            return null;
        }

        try
        {
            using var cert = X509Certificate2.CreateFromPem(certPem.Trim(), keyPem.Trim());
            var pfx = cert.Export(X509ContentType.Pkcs12);
            return X509CertificateLoader.LoadPkcs12(pfx, password: null);
        }
        catch
        {
            return null;
        }
    }

    private static X509Certificate2 CreateEmergencyFallbackCertificate()
    {
        using var rsa = RSA.Create(2048);
        var req = new CertificateRequest(
            "CN=cnn-agent-fallback",
            rsa,
            HashAlgorithmName.SHA256,
            RSASignaturePadding.Pkcs1);
        req.CertificateExtensions.Add(new X509BasicConstraintsExtension(false, false, 0, false));
        req.CertificateExtensions.Add(new X509KeyUsageExtension(
            X509KeyUsageFlags.DigitalSignature | X509KeyUsageFlags.KeyEncipherment,
            critical: true));
        req.CertificateExtensions.Add(new X509SubjectKeyIdentifierExtension(req.PublicKey, false));

        using var issued = req.CreateSelfSigned(
            DateTimeOffset.UtcNow.AddHours(-1),
            DateTimeOffset.UtcNow.AddYears(5));
        var pfx = issued.Export(X509ContentType.Pkcs12);
        return X509CertificateLoader.LoadPkcs12(pfx, password: null);
    }

    private static X509Certificate2? TryLoadPemFiles(string? certPath, string? keyPath)
    {
        if (string.IsNullOrWhiteSpace(certPath) || string.IsNullOrWhiteSpace(keyPath))
        {
            return null;
        }

        try
        {
            if (!File.Exists(certPath) || !File.Exists(keyPath))
            {
                return null;
            }

            var certPem = File.ReadAllText(certPath);
            var keyPem = File.ReadAllText(keyPath);
            return TryLoadPemPair(certPem, keyPem);
        }
        catch
        {
            return null;
        }
    }

    private static IEnumerable<string> ExpandHosts(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            yield break;
        }

        var parts = raw.Split([',', ';', '\n', '\r', '\t', ' '], StringSplitOptions.RemoveEmptyEntries);
        foreach (var item in parts)
        {
            var host = NormalizeHost(item);
            if (!string.IsNullOrWhiteSpace(host))
            {
                yield return host;
            }
        }
    }

    private static string NormalizeHost(string? host)
    {
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        return host.Trim().TrimEnd('.').ToLowerInvariant();
    }

    private static string BuildInlineSourceKey(string prefix, string? certPem, string? keyPem)
    {
        return $"{prefix}:{ComputeTextHash(certPem)}:{ComputeTextHash(keyPem)}";
    }

    private static string BuildFileSourceKey(string prefix, string? certPath, string? keyPath)
    {
        var cert = BuildFileStamp(certPath);
        var key = BuildFileStamp(keyPath);
        return $"{prefix}:{cert}:{key}";
    }

    private static string BuildFileStamp(string? path)
    {
        if (string.IsNullOrWhiteSpace(path))
        {
            return "null";
        }

        try
        {
            var info = new FileInfo(path);
            if (!info.Exists)
            {
                return $"missing:{path}";
            }

            return $"{path}:{info.Length}:{info.LastWriteTimeUtc.Ticks}";
        }
        catch
        {
            return $"error:{path}";
        }
    }

    private static string ComputeTextHash(string? input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return "null";
        }

        var normalized = input.Trim();
        var hash = SHA256.HashData(Encoding.UTF8.GetBytes(normalized));
        return Convert.ToHexString(hash);
    }

    private static void DisposeObsoleteCertificates(TlsSnapshot previous, TlsSnapshot next)
    {
        var nextRefs = CollectCertificates(next).ToHashSet(ReferenceEqualityComparer.Instance);
        foreach (var certificate in CollectCertificates(previous))
        {
            if (nextRefs.Contains(certificate))
            {
                continue;
            }

            try
            {
                certificate.Dispose();
            }
            catch
            {
                // ignore
            }
        }
    }

    private static IEnumerable<X509Certificate2> CollectCertificates(TlsSnapshot snapshot)
    {
        var seen = new HashSet<X509Certificate2>(ReferenceEqualityComparer.Instance);

        foreach (var cert in snapshot.Exact.Values)
        {
            if (cert != null && seen.Add(cert))
            {
                yield return cert;
            }
        }

        foreach (var (_, cert) in snapshot.Wildcards)
        {
            if (cert != null && seen.Add(cert))
            {
                yield return cert;
            }
        }

        if (snapshot.Fallback != null && seen.Add(snapshot.Fallback))
        {
            yield return snapshot.Fallback;
        }

        foreach (var cert in snapshot.CacheEntries.Values)
        {
            if (cert != null && seen.Add(cert))
            {
                yield return cert;
            }
        }
    }

    private sealed class TlsSnapshot
    {
        public static readonly TlsSnapshot Empty = new(
            exact: new Dictionary<string, X509Certificate2>(StringComparer.OrdinalIgnoreCase),
            wildcards: [],
            fallback: null,
            cacheEntries: new Dictionary<string, X509Certificate2>(StringComparer.Ordinal));

        public TlsSnapshot(
            IReadOnlyDictionary<string, X509Certificate2> exact,
            IReadOnlyList<(string Suffix, X509Certificate2 Certificate)> wildcards,
            X509Certificate2? fallback,
            IReadOnlyDictionary<string, X509Certificate2> cacheEntries)
        {
            Exact = exact;
            Wildcards = wildcards;
            Fallback = fallback;
            CacheEntries = cacheEntries;
        }

        public IReadOnlyDictionary<string, X509Certificate2> Exact { get; }
        public IReadOnlyList<(string Suffix, X509Certificate2 Certificate)> Wildcards { get; }
        public X509Certificate2? Fallback { get; }
        public IReadOnlyDictionary<string, X509Certificate2> CacheEntries { get; }
    }
}
