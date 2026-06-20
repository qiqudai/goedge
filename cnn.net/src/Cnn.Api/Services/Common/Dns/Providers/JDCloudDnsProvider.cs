using System.Collections.Concurrent;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.RegularExpressions;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class JDCloudDnsProvider : IDnsRecordProvider
{
    private const string ServiceName = "clouddnsservice";
    private const string Endpoint = "clouddnsservice.jdcloud-api.com";
    private const string ApiVersion = "v1";

    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly string _accessKey;
    private readonly string _secretKey;
    private readonly string _region;
    private readonly ConcurrentDictionary<string, int> _domainIdCache = new(StringComparer.OrdinalIgnoreCase);

    private JDCloudDnsProvider(string accessKey, string secretKey, string region)
    {
        _accessKey = accessKey;
        _secretKey = secretKey;
        _region = string.IsNullOrWhiteSpace(region) ? "cn-north-1" : region;
    }

    public static IDnsRecordProvider? TryCreate(string credentials)
    {
        if (string.IsNullOrWhiteSpace(credentials))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(credentials);
            var root = doc.RootElement;
            var accessKey = GetString(root, "access_key") ?? GetString(root, "accessKey") ?? string.Empty;
            var secretKey = GetString(root, "secret_key") ?? GetString(root, "secretKey") ?? string.Empty;
            var region = GetString(root, "region") ?? GetString(root, "region_id") ?? string.Empty;

            return new JDCloudDnsProvider(accessKey.Trim(), secretKey.Trim(), region.Trim());
        }
        catch
        {
            return null;
        }
    }

    public async Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain)
    {
        var domainId = await GetDomainIdAsync(domain);
        if (domainId <= 0)
        {
            throw new InvalidOperationException($"jdcloud domain not found: {domain}");
        }

        var records = new List<DnsRecord>();
        var pageNumber = 1;
        var pageSize = 100;

        while (true)
        {
            var query = new Dictionary<string, string>
            {
                ["pageNumber"] = pageNumber.ToString(),
                ["pageSize"] = pageSize.ToString()
            };
            using var doc = await SendAsync(HttpMethod.Get, $"/regions/{_region}/domain/{domainId}/RR", query, null);
            EnsureSuccess(doc);

            var result = GetResultElement(doc);
            if (result.ValueKind != JsonValueKind.Object ||
                !result.TryGetProperty("dataList", out var list) ||
                list.ValueKind != JsonValueKind.Array)
            {
                break;
            }

            foreach (var item in list.EnumerateArray())
            {
                var type = GetString(item, "type");
                var name = GetString(item, "hostRecord");
                var value = GetString(item, "hostValue");
                var ttl = GetInt(item, "ttl");
                var weight = GetInt(item, "weight");
                var line = GetLineValue(item);
                if (string.IsNullOrWhiteSpace(type) || string.IsNullOrWhiteSpace(value))
                {
                    continue;
                }

                records.Add(new DnsRecord
                {
                    Type = type,
                    Name = NormalizeHost(name),
                    Value = value,
                    TTL = ttl <= 0 ? 300 : ttl,
                    Weight = weight,
                    Line = line
                });
            }

            var totalPage = GetInt(result, "totalPage");
            if (totalPage <= 0 || pageNumber >= totalPage)
            {
                break;
            }

            pageNumber++;
        }

        return records;
    }

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        var domainId = await GetDomainIdAsync(domain);
        if (domainId <= 0)
        {
            throw new InvalidOperationException($"jdcloud domain not found: {domain}");
        }

        var payload = new
        {
            req = new
            {
                hostRecord = NormalizeHost(record.Name),
                hostValue = record.Value,
                type = record.Type,
                ttl = record.TTL <= 0 ? 300 : record.TTL,
                viewValue = ResolveViewValue(record.Line),
                weight = record.Weight > 0 ? record.Weight : (int?)null
            }
        };

        using var doc = await SendAsync(HttpMethod.Post, $"/regions/{_region}/domain/{domainId}/RRAdd", null, payload);
        EnsureSuccess(doc);
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        var domainId = await GetDomainIdAsync(domain);
        if (domainId <= 0)
        {
            throw new InvalidOperationException($"jdcloud domain not found: {domain}");
        }

        var ids = await FindRecordIdsAsync(domainId, record);
        if (ids.Count == 0)
        {
            return;
        }

        var payload = new
        {
            ids = ids,
            action = "del"
        };

        using var doc = await SendAsync(HttpMethod.Post, $"/regions/{_region}/domain/{domainId}/RROperate", null, payload);
        EnsureSuccess(doc);
    }

    private async Task<List<int>> FindRecordIdsAsync(int domainId, DnsRecord record)
    {
        var ids = new List<int>();
        var pageNumber = 1;
        var pageSize = 100;
        var targetHost = NormalizeHost(record.Name);

        while (true)
        {
            var query = new Dictionary<string, string>
            {
                ["pageNumber"] = pageNumber.ToString(),
                ["pageSize"] = pageSize.ToString()
            };
            using var doc = await SendAsync(HttpMethod.Get, $"/regions/{_region}/domain/{domainId}/RR", query, null);
            EnsureSuccess(doc);

            var result = GetResultElement(doc);
            if (result.ValueKind != JsonValueKind.Object ||
                !result.TryGetProperty("dataList", out var list) ||
                list.ValueKind != JsonValueKind.Array)
            {
                break;
            }

            foreach (var item in list.EnumerateArray())
            {
                var id = GetInt(item, "id");
                if (id <= 0)
                {
                    continue;
                }

                var type = GetString(item, "type");
                var name = NormalizeHost(GetString(item, "hostRecord"));
                var value = GetString(item, "hostValue");
                if (!string.Equals(type, record.Type, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
                if (!string.Equals(name, targetHost, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
                if (!string.Equals(value, record.Value, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }

                ids.Add(id);
            }

            var totalPage = GetInt(result, "totalPage");
            if (totalPage <= 0 || pageNumber >= totalPage)
            {
                break;
            }

            pageNumber++;
        }

        return ids;
    }

    private async Task<int> GetDomainIdAsync(string domain)
    {
        var normalized = NormalizeDomain(domain);
        if (string.IsNullOrWhiteSpace(normalized))
        {
            return 0;
        }

        if (_domainIdCache.TryGetValue(normalized, out var cached))
        {
            return cached;
        }

        foreach (var candidate in GetDomainCandidates(normalized))
        {
            var found = await FindDomainIdAsync(candidate, normalized);
            if (found > 0)
            {
                _domainIdCache[normalized] = found;
                return found;
            }
        }

        var fallback = await FindDomainIdAsync(null, normalized);
        if (fallback > 0)
        {
            _domainIdCache[normalized] = fallback;
        }

        return fallback;
    }

    private async Task<int> FindDomainIdAsync(string? queryDomain, string target)
    {
        var pageNumber = 1;
        var pageSize = 50;
        var bestId = 0;
        var bestLen = 0;

        while (true)
        {
            var query = new Dictionary<string, string>
            {
                ["pageNumber"] = pageNumber.ToString(),
                ["pageSize"] = pageSize.ToString()
            };
            if (!string.IsNullOrWhiteSpace(queryDomain))
            {
                query["domainName"] = queryDomain;
            }

            using var doc = await SendAsync(HttpMethod.Get, $"/regions/{_region}/domain", query, null);
            EnsureSuccess(doc);

            var result = GetResultElement(doc);
            if (result.ValueKind != JsonValueKind.Object ||
                !result.TryGetProperty("dataList", out var list) ||
                list.ValueKind != JsonValueKind.Array)
            {
                return bestId;
            }

            foreach (var item in list.EnumerateArray())
            {
                var id = GetInt(item, "id");
                var name = NormalizeDomain(GetString(item, "domainName"));
                if (id <= 0 || string.IsNullOrWhiteSpace(name))
                {
                    continue;
                }

                if (string.Equals(target, name, StringComparison.OrdinalIgnoreCase) ||
                    target.EndsWith("." + name, StringComparison.OrdinalIgnoreCase))
                {
                    if (name.Length > bestLen)
                    {
                        bestLen = name.Length;
                        bestId = id;
                    }
                }
            }

            var totalPage = GetInt(result, "totalPage");
            if (totalPage <= 0 || pageNumber >= totalPage)
            {
                return bestId;
            }

            pageNumber++;
        }
    }

    private static IEnumerable<string> GetDomainCandidates(string domain)
    {
        var parts = domain.Split('.', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length < 2)
        {
            yield break;
        }

        yield return parts[^2] + "." + parts[^1];
        if (parts.Length >= 3)
        {
            yield return parts[^3] + "." + parts[^2] + "." + parts[^1];
        }
    }

    private async Task<JsonDocument> SendAsync(HttpMethod method, string path, Dictionary<string, string>? query, object? payload)
    {
        var url = BuildUrl(path, query);
        var req = new HttpRequestMessage(method, url);
        var body = string.Empty;

        req.Headers.Host = Endpoint;

        if (payload != null)
        {
            body = JsonSerializer.Serialize(payload, JsonOptions);
            req.Content = new StringContent(body, Encoding.UTF8, "application/json");
        }
        else
        {
            req.Headers.TryAddWithoutValidation("Content-Type", "application/json");
        }

        ApplySignature(req, body);

        var (_, responseBody) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(responseBody))
        {
            throw new InvalidOperationException("jdcloud empty response");
        }

        return JsonDocument.Parse(responseBody);
    }

    private static Uri BuildUrl(string path, Dictionary<string, string>? query)
    {
        var builder = new UriBuilder("https", Endpoint)
        {
            Path = "/" + ApiVersion + (path.StartsWith("/", StringComparison.Ordinal) ? path : "/" + path)
        };

        if (query != null && query.Count > 0)
        {
            builder.Query = BuildQuery(query);
        }

        return builder.Uri;
    }

    private void ApplySignature(HttpRequestMessage request, string body)
    {
        var now = DateTime.UtcNow;
        var formattedTime = now.ToString("yyyyMMdd'T'HHmmss'Z'");
        var shortDate = now.ToString("yyyyMMdd");

        request.Headers.Remove("x-jdcloud-date");
        request.Headers.Remove("x-jdcloud-nonce");
        request.Headers.TryAddWithoutValidation("x-jdcloud-date", formattedTime);
        request.Headers.TryAddWithoutValidation("x-jdcloud-nonce", Guid.NewGuid().ToString());

        var bodyHash = Sha256Hex(string.IsNullOrEmpty(body) ? string.Empty : body);
        var canonicalHeaders = BuildCanonicalHeaders(request, out var signedHeaders);
        var canonicalRequest = string.Join("\n", new[]
        {
            request.Method.Method,
            request.RequestUri?.AbsolutePath ?? "/",
            request.RequestUri?.Query.TrimStart('?') ?? string.Empty,
            canonicalHeaders + "\n",
            signedHeaders,
            bodyHash
        });

        var credentialScope = string.Join("/", new[]
        {
            shortDate,
            _region,
            ServiceName,
            "jdcloud2_request"
        });

        var stringToSign = string.Join("\n", new[]
        {
            "JDCLOUD2-HMAC-SHA256",
            formattedTime,
            credentialScope,
            Sha256Hex(canonicalRequest)
        });

        var signature = Sign(stringToSign, shortDate);
        var authHeader = $"JDCLOUD2-HMAC-SHA256 Credential={_accessKey}/{credentialScope}, SignedHeaders={signedHeaders}, Signature={signature}";
        request.Headers.Remove("Authorization");
        request.Headers.TryAddWithoutValidation("Authorization", authHeader);
    }

    private string Sign(string stringToSign, string shortDate)
    {
        var kDate = HmacSha256(Encoding.UTF8.GetBytes("JDCLOUD2" + _secretKey), shortDate);
        var kRegion = HmacSha256(kDate, _region);
        var kService = HmacSha256(kRegion, ServiceName);
        var kSigning = HmacSha256(kService, "jdcloud2_request");
        return ToHex(HmacSha256(kSigning, stringToSign));
    }

    private static string BuildCanonicalHeaders(HttpRequestMessage request, out string signedHeaders)
    {
        var headers = new SortedDictionary<string, string>(StringComparer.Ordinal);
        var host = request.Headers.Host;
        if (string.IsNullOrWhiteSpace(host))
        {
            host = request.RequestUri?.Host ?? string.Empty;
        }
        headers["host"] = host;

        AddHeaders(headers, request.Headers);
        if (request.Content != null)
        {
            AddHeaders(headers, request.Content.Headers);
        }

        var keys = headers.Keys.ToList();
        signedHeaders = string.Join(";", keys);

        var parts = new List<string>(keys.Count);
        foreach (var key in keys)
        {
            parts.Add(key + ":" + StripSpaces(headers[key]));
        }

        return string.Join("\n", parts);
    }

    private static void AddHeaders(SortedDictionary<string, string> headers, IEnumerable<KeyValuePair<string, IEnumerable<string>>> source)
    {
        foreach (var header in source)
        {
            var name = header.Key.ToLowerInvariant();
            if (IsIgnoredHeader(name))
            {
                continue;
            }

            var value = string.Join(",", header.Value);
            if (headers.TryGetValue(name, out var existing))
            {
                headers[name] = existing + "," + value;
            }
            else
            {
                headers[name] = value;
            }
        }
    }

    private static bool IsIgnoredHeader(string name)
    {
        return string.Equals(name, "authorization", StringComparison.OrdinalIgnoreCase) ||
               string.Equals(name, "user-agent", StringComparison.OrdinalIgnoreCase) ||
               string.Equals(name, "x-jdcloud-request-id", StringComparison.OrdinalIgnoreCase) ||
               string.Equals(name, "content-length", StringComparison.OrdinalIgnoreCase);
    }

    private static string StripSpaces(string value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return string.Empty;
        }

        return Regex.Replace(value.Trim(), "\\s+", " ");
    }

    private static string Sha256Hex(string data)
    {
        var bytes = Encoding.UTF8.GetBytes(data);
        var hash = SHA256.HashData(bytes);
        return ToHex(hash);
    }

    private static byte[] HmacSha256(byte[] key, string data)
    {
        using var hmac = new HMACSHA256(key);
        return hmac.ComputeHash(Encoding.UTF8.GetBytes(data));
    }

    private static string ToHex(byte[] bytes)
    {
        var sb = new StringBuilder(bytes.Length * 2);
        foreach (var b in bytes)
        {
            sb.Append(b.ToString("x2"));
        }
        return sb.ToString();
    }

    private static string BuildQuery(Dictionary<string, string> values)
    {
        var parts = new List<string>(values.Count);
        foreach (var key in values.Keys.OrderBy(k => k, StringComparer.Ordinal))
        {
            var value = values[key] ?? string.Empty;
            parts.Add(Uri.EscapeDataString(key) + "=" + Uri.EscapeDataString(value));
        }

        return string.Join("&", parts);
    }

    private static void EnsureSuccess(JsonDocument doc)
    {
        if (!doc.RootElement.TryGetProperty("error", out var error) ||
            error.ValueKind != JsonValueKind.Object)
        {
            return;
        }

        var code = GetInt(error, "code");
        if (code == 0)
        {
            return;
        }

        var message = GetString(error, "message") ?? GetString(error, "status") ?? "jdcloud error";
        throw new InvalidOperationException($"jdcloud error: {message}");
    }

    private static JsonElement GetResultElement(JsonDocument doc)
    {
        return doc.RootElement.TryGetProperty("result", out var result) ? result : default;
    }

    private static int ResolveViewValue(string? line)
    {
        if (int.TryParse(line, out var value))
        {
            return value;
        }

        return 1;
    }

    private static string GetLineValue(JsonElement element)
    {
        if (!element.TryGetProperty("viewValue", out var prop))
        {
            return string.Empty;
        }

        if (prop.ValueKind == JsonValueKind.Array)
        {
            foreach (var item in prop.EnumerateArray())
            {
                if (item.ValueKind == JsonValueKind.Number && item.TryGetInt32(out var value))
                {
                    return value.ToString();
                }
            }
        }

        if (prop.ValueKind == JsonValueKind.Number && prop.TryGetInt32(out var single))
        {
            return single.ToString();
        }

        return string.Empty;
    }

    private static string NormalizeHost(string? host)
    {
        var value = (host ?? string.Empty).Trim();
        return string.IsNullOrWhiteSpace(value) ? "@" : value;
    }

    private static string NormalizeDomain(string? input)
    {
        var value = (input ?? string.Empty).Trim().TrimEnd('.');
        return value.ToLowerInvariant();
    }

    private static string? GetString(JsonElement element, string name)
    {
        return element.TryGetProperty(name, out var prop) ? prop.GetString() : null;
    }

    private static int GetInt(JsonElement element, string name)
    {
        if (!element.TryGetProperty(name, out var prop))
        {
            return 0;
        }

        if (prop.ValueKind == JsonValueKind.Number && prop.TryGetInt32(out var value))
        {
            return value;
        }

        if (prop.ValueKind == JsonValueKind.String && int.TryParse(prop.GetString(), out var parsed))
        {
            return parsed;
        }

        return 0;
    }
}
