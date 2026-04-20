using System.Formats.Asn1;
using System.Net;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Text.Json;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public sealed partial class CertService
{
    private static string? ResolveCertState(string? taskState, string? certType)
    {
        var normalized = (taskState ?? string.Empty).Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(normalized))
        {
            return string.Empty;
        }

        return normalized switch
        {
            "running" => "issuing",
            "success" => "ready",
            _ => normalized
        };
    }

    private static bool ShouldExposeCertData(string? certType, string? state)
    {
        if (string.Equals(NormalizeCertType(certType), "upload", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }

        var normalized = (state ?? string.Empty).Trim().ToLowerInvariant();
        return normalized is "ready" or "success";
    }

    private string? EncryptKey(string? key)
    {
        if (string.IsNullOrWhiteSpace(key))
        {
            return string.Empty;
        }

        var encrypted = _cryptoService.Encrypt(key);
        return string.IsNullOrWhiteSpace(encrypted) ? key : encrypted;
    }

    private string? DecryptKey(string? key)
    {
        if (string.IsNullOrWhiteSpace(key))
        {
            return string.Empty;
        }

        var decrypted = _cryptoService.Decrypt(key);
        return string.IsNullOrWhiteSpace(decrypted) ? key : decrypted;
    }

    internal static string NormalizeCertType(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return string.Empty;
        }

        var normalized = value.Trim().ToLowerInvariant();
        return normalized switch
        {
            "upload" or "self" => "upload",
            "letsencrypt" or "let's encrypt" or "lets encrypt" or "lets" => "letsencrypt",
            "zerossl" => "zerossl",
            "buypass" => "buypass",
            "google" => "google",
            _ => normalized
        };
    }

    internal static string BuildCaDirUrl(string certType)
    {
        return certType switch
        {
            "letsencrypt" => "https://acme-v02.api.letsencrypt.org/directory",
            "zerossl" => "https://acme.zerossl.com/v2/DV90",
            "buypass" => "https://api.buypass.com/acme/directory",
            "google" => "https://dv.acme-v02.api.pki.goog/directory",
            _ => "https://acme-v02.api.letsencrypt.org/directory"
        };
    }

    private static string NormalizeCertDomainKey(string? raw)
    {
        var domains = SplitCertDomains(raw);
        return domains.Count > 0 ? domains[0] : string.Empty;
    }

    internal static List<string> SplitCertDomains(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new List<string>();
        }

        var parts = SplitFields(raw);
        return NormalizeDomains(parts, out _);
    }

    private static bool CertDomainMatches(string domainKey, string? raw)
    {
        if (string.IsNullOrWhiteSpace(domainKey) || string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        var normalizedKey = NormalizeDomain(domainKey);
        foreach (var domain in SplitCertDomains(raw))
        {
            if (NormalizeDomain(domain) == normalizedKey)
            {
                return true;
            }
        }

        return false;
    }

    private static string NormalizeDomain(string? input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return string.Empty;
        }

        var trimmed = input.Trim();
        if (trimmed.StartsWith("*.", StringComparison.Ordinal))
        {
            var baseHost = DomainHelper.NormalizeDomainInput(trimmed[2..]);
            return string.IsNullOrWhiteSpace(baseHost) ? string.Empty : "*." + baseHost;
        }

        return DomainHelper.NormalizeDomainInput(trimmed);
    }

    private static List<string> NormalizeDomainsFromInput(string? raw, out string? errorKey)
    {
        errorKey = null;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new List<string>();
        }

        var parts = SplitFields(raw);
        return NormalizeDomains(parts, out errorKey);
    }

    private static List<string> NormalizeDomainsFromJson(JsonElement element, out string? errorKey)
    {
        errorKey = null;
        if (element.ValueKind == JsonValueKind.Undefined || element.ValueKind == JsonValueKind.Null)
        {
            return new List<string>();
        }

        if (element.ValueKind == JsonValueKind.String)
        {
            return NormalizeDomainsFromInput(element.GetString(), out errorKey);
        }

        if (element.ValueKind == JsonValueKind.Array)
        {
            var list = new List<string>();
            foreach (var item in element.EnumerateArray())
            {
                if (item.ValueKind == JsonValueKind.String)
                {
                    list.Add(item.GetString() ?? string.Empty);
                }
            }
            return NormalizeDomains(list, out errorKey);
        }

        errorKey = "cert_batch_domains_required";
        return new List<string>();
    }

    private static List<string> NormalizeDomains(IEnumerable<string> domains, out string? errorKey)
    {
        errorKey = null;
        var set = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var domain in domains)
        {
            var normalized = NormalizeDomain(domain);
            if (string.IsNullOrWhiteSpace(normalized))
            {
                continue;
            }

            if (IsIpDomain(normalized))
            {
                errorKey = "invalid_domain";
                return new List<string>();
            }

            var toValidate = normalized.StartsWith("*.", StringComparison.Ordinal) ? normalized[2..] : normalized;
            if (!DomainHelper.IsValidDomain(toValidate))
            {
                errorKey = "invalid_domain";
                return new List<string>();
            }

            set.Add(normalized);
        }

        var result = set.ToList();
        result.Sort(StringComparer.OrdinalIgnoreCase);
        return result;
    }

    internal static bool HasWildcard(IEnumerable<string> domains)
    {
        return domains.Any(domain => domain.Trim().StartsWith("*.", StringComparison.Ordinal));
    }

    private static bool IsIpDomain(string domain)
    {
        var trimmed = domain.StartsWith("*.", StringComparison.Ordinal) ? domain[2..] : domain;
        return IPAddress.TryParse(trimmed, out _);
    }

    private static List<string> SplitFields(string raw)
    {
        return raw
            .Split(new[] { ',', ';', '\n', '\r', '\t', ' ' }, StringSplitOptions.RemoveEmptyEntries)
            .Select(item => item.Trim())
            .Where(item => !string.IsNullOrWhiteSpace(item))
            .ToList();
    }

    internal static bool TryParseCert(string certPem, out List<string> domains, out DateTime notBefore, out DateTime notAfter)
    {
        domains = new List<string>();
        notBefore = DateTime.MinValue;
        notAfter = DateTime.MinValue;

        if (string.IsNullOrWhiteSpace(certPem))
        {
            return false;
        }

        try
        {
            using var cert = LoadCertificate(certPem);
            notBefore = cert.NotBefore;
            notAfter = cert.NotAfter;

            var set = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
            var cn = cert.GetNameInfo(X509NameType.DnsName, false);
            if (!string.IsNullOrWhiteSpace(cn))
            {
                set.Add(cn.Trim());
            }

            foreach (var name in EnumerateSubjectAltNames(cert))
            {
                if (!string.IsNullOrWhiteSpace(name))
                {
                    set.Add(name.Trim());
                }
            }

            var normalized = NormalizeDomains(set, out _);
            domains = normalized;
            return domains.Count > 0;
        }
        catch
        {
            return false;
        }
    }

    private static X509Certificate2 LoadCertificate(string certPem)
    {
        var span = certPem.AsSpan();
        while (PemEncoding.TryFind(span, out var fields))
        {
            var label = span[fields.Label].ToString();
            if (string.Equals(label, "CERTIFICATE", StringComparison.OrdinalIgnoreCase))
            {
                var base64 = span[fields.Base64Data].ToString();
                var raw = Convert.FromBase64String(base64);
                return X509CertificateLoader.LoadCertificate(raw);
            }

            span = span[fields.Location.End..];
        }

        throw new InvalidOperationException("invalid cert pem");
    }

    private static IEnumerable<string> EnumerateSubjectAltNames(X509Certificate2 cert)
    {
        foreach (var extension in cert.Extensions)
        {
            if (extension.Oid?.Value != "2.5.29.17")
            {
                continue;
            }

            var reader = new AsnReader(extension.RawData, AsnEncodingRules.DER);
            var sequence = reader.ReadSequence();
            while (sequence.HasData)
            {
                var tag = sequence.PeekTag();
                if (tag.TagClass == TagClass.ContextSpecific && tag.TagValue == 2)
                {
                    var dns = sequence.ReadCharacterString(UniversalTagNumber.IA5String, new Asn1Tag(TagClass.ContextSpecific, 2));
                    if (!string.IsNullOrWhiteSpace(dns))
                    {
                        yield return dns;
                    }
                }
                else
                {
                    sequence.ReadEncodedValue();
                }
            }
        }
    }

    private static string DefaultCertName(string domain)
    {
        if (string.IsNullOrWhiteSpace(domain))
        {
            return "免费证书";
        }

        return domain + "免费证书";
    }

    private static bool TryParseLong(JsonElement element, out long value)
    {
        value = 0;
        if (element.ValueKind == JsonValueKind.Number)
        {
            return element.TryGetInt64(out value);
        }
        if (element.ValueKind == JsonValueKind.String)
        {
            return long.TryParse(element.GetString(), out value);
        }
        return false;
    }

    private static string SanitizeFilename(string input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return string.Empty;
        }

        var chars = input.Select(ch =>
        {
            if (char.IsLetterOrDigit(ch))
            {
                return ch;
            }

            return ch is '.' or '-' or '_' ? ch : '_';
        }).ToArray();

        return new string(chars).Trim(' ', '.', '-', '_');
    }
}


