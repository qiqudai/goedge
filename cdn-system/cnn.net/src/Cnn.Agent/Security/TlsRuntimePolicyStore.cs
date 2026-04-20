using System.Net.Security;
using System.Security.Authentication;
using System.Threading;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Security;

public interface ITlsRuntimePolicyStore
{
    TlsRuntimePolicy GetCurrent();
    void Reload(EdgeConfigDto config);
}

public sealed class TlsRuntimePolicyStore : ITlsRuntimePolicyStore
{
    private static readonly IReadOnlyDictionary<string, TlsCipherSuite> CipherAliasMap =
        new Dictionary<string, TlsCipherSuite>(StringComparer.OrdinalIgnoreCase)
        {
            ["ECDHE-ECDSA-AES128-GCM-SHA256"] = TlsCipherSuite.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            ["ECDHE-RSA-AES128-GCM-SHA256"] = TlsCipherSuite.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
            ["ECDHE-ECDSA-AES256-GCM-SHA384"] = TlsCipherSuite.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            ["ECDHE-RSA-AES256-GCM-SHA384"] = TlsCipherSuite.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            ["ECDHE-ECDSA-CHACHA20-POLY1305"] = TlsCipherSuite.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
            ["ECDHE-RSA-CHACHA20-POLY1305"] = TlsCipherSuite.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
            ["TLS_AES_128_GCM_SHA256"] = TlsCipherSuite.TLS_AES_128_GCM_SHA256,
            ["TLS_AES_256_GCM_SHA384"] = TlsCipherSuite.TLS_AES_256_GCM_SHA384,
            ["TLS_CHACHA20_POLY1305_SHA256"] = TlsCipherSuite.TLS_CHACHA20_POLY1305_SHA256
        };

    private TlsRuntimePolicy _current = TlsRuntimePolicy.Default;

    public TlsRuntimePolicy GetCurrent()
    {
        return Volatile.Read(ref _current);
    }

    public void Reload(EdgeConfigDto config)
    {
        if (config == null)
        {
            return;
        }

        var protocols = SslProtocols.None;
        var hasProtocolOverride = false;
        var checkCertificateRevocation = false;
        var cipherSuites = new List<TlsCipherSuite>();
        var seenCiphers = new HashSet<TlsCipherSuite>();

        foreach (var domain in config.Domains)
        {
            if (!string.IsNullOrWhiteSpace(domain.HttpsSslProtocols))
            {
                var parsed = ParseProtocols(domain.HttpsSslProtocols);
                if (parsed != SslProtocols.None)
                {
                    protocols |= parsed;
                    hasProtocolOverride = true;
                }
            }

            if (domain.HttpsOcsp.GetValueOrDefault())
            {
                checkCertificateRevocation = true;
            }

            foreach (var suite in ParseCipherSuites(domain.HttpsSslCiphers))
            {
                if (seenCiphers.Add(suite))
                {
                    cipherSuites.Add(suite);
                }
            }
        }

        var finalProtocols = hasProtocolOverride ? protocols : SslProtocols.None;
        var cipherPolicy = BuildCipherSuitesPolicy(cipherSuites);

        var next = new TlsRuntimePolicy(
            SslProtocols: finalProtocols,
            CheckCertificateRevocation: checkCertificateRevocation,
            CipherSuites: cipherSuites,
            CipherSuitesPolicy: cipherPolicy);

        Volatile.Write(ref _current, next);
    }

    private static SslProtocols ParseProtocols(string raw)
    {
        var result = SslProtocols.None;
        foreach (var token in SplitTokens(raw))
        {
            var normalized = token.Trim().ToLowerInvariant();
            switch (normalized)
            {
                case "tls1":
                case "tlsv1":
                case "tls1.0":
                case "tlsv1.0":
                    // TLS1.0 is obsolete and intentionally ignored.
                    break;
                case "tls1.1":
                case "tlsv1.1":
                    // TLS1.1 is obsolete and intentionally ignored.
                    break;
                case "tls1.2":
                case "tlsv1.2":
                    result |= SslProtocols.Tls12;
                    break;
                case "tls1.3":
                case "tlsv1.3":
                    result |= SslProtocols.Tls13;
                    break;
            }
        }

        return result;
    }

    private static IEnumerable<TlsCipherSuite> ParseCipherSuites(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            yield break;
        }

        foreach (var token in SplitTokens(raw))
        {
            var trimmed = token.Trim();
            if (trimmed.Length == 0)
            {
                continue;
            }

            if (CipherAliasMap.TryGetValue(trimmed, out var mapped))
            {
                yield return mapped;
                continue;
            }

            if (Enum.TryParse<TlsCipherSuite>(trimmed, ignoreCase: true, out var direct))
            {
                yield return direct;
                continue;
            }

            var normalized = NormalizeCipherToken(trimmed);
            if (Enum.TryParse<TlsCipherSuite>(normalized, ignoreCase: true, out var parsed))
            {
                yield return parsed;
            }
        }
    }

    private static string NormalizeCipherToken(string token)
    {
        var value = token.Trim().ToUpperInvariant().Replace("-", "_", StringComparison.Ordinal);
        if (value.StartsWith("TLS_", StringComparison.Ordinal))
        {
            return value;
        }

        return "TLS_" + value;
    }

    private static IEnumerable<string> SplitTokens(string raw)
    {
        return raw.Split([',', ';', '|', ' ', '\n', '\r', '\t', ':'], StringSplitOptions.RemoveEmptyEntries);
    }

    private static CipherSuitesPolicy? BuildCipherSuitesPolicy(IReadOnlyList<TlsCipherSuite> suites)
    {
        if (suites == null || suites.Count == 0)
        {
            return null;
        }

        if (OperatingSystem.IsWindows())
        {
            return null;
        }

        try
        {
            return new CipherSuitesPolicy(suites);
        }
        catch (PlatformNotSupportedException)
        {
            return null;
        }
    }
}

public sealed record TlsRuntimePolicy(
    SslProtocols SslProtocols,
    bool CheckCertificateRevocation,
    IReadOnlyList<TlsCipherSuite> CipherSuites,
    CipherSuitesPolicy? CipherSuitesPolicy)
{
    public static readonly TlsRuntimePolicy Default = new(
        SslProtocols: SslProtocols.None,
        CheckCertificateRevocation: false,
        CipherSuites: Array.Empty<TlsCipherSuite>(),
        CipherSuitesPolicy: null);
}
