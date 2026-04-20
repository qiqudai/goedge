using System.Net;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class HuaweiDnsProvider : IDnsRecordProvider, IDnsRecordSetUpdater
{
    private readonly string _accessKeyId;
    private readonly string _secretAccessKey;
    private string _region;
    private readonly bool _regionProvided;

    private HuaweiDnsProvider(string accessKeyId, string secretAccessKey, string region, bool regionProvided)
    {
        _accessKeyId = accessKeyId;
        _secretAccessKey = secretAccessKey;
        _region = string.IsNullOrWhiteSpace(region) ? "cn-north-1" : region;
        _regionProvided = regionProvided;
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
            var accessKeyId = GetString(root, "access_key_id") ?? string.Empty;
            var secretAccessKey = GetString(root, "secret_access_key") ?? string.Empty;
            var id = GetString(root, "id") ?? string.Empty;
            var secret = GetString(root, "secret") ?? string.Empty;
            var region = GetString(root, "region") ?? string.Empty;
            var regionProvided = !string.IsNullOrWhiteSpace(region);

            if (string.IsNullOrWhiteSpace(accessKeyId) && !string.IsNullOrWhiteSpace(id))
            {
                accessKeyId = id;
            }
            if (string.IsNullOrWhiteSpace(secretAccessKey) && !string.IsNullOrWhiteSpace(secret))
            {
                secretAccessKey = secret;
            }

            accessKeyId = accessKeyId.Trim();
            secretAccessKey = secretAccessKey.Trim();
            return new HuaweiDnsProvider(accessKeyId, secretAccessKey, region.Trim(), regionProvided);
        }
        catch
        {
            return null;
        }
    }

    public Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain)
    {
        return Task.FromResult<IReadOnlyList<DnsRecord>>(Array.Empty<DnsRecord>());
    }

    public async Task UpsertRecordSetAsync(string domain, DnsRecord record, IReadOnlyList<string> values)
    {
        var zoneId = await GetZoneIdAsync(domain);
        if (string.IsNullOrWhiteSpace(zoneId))
        {
            throw new InvalidOperationException($"huawei zone not found: {domain}");
        }

        var name = BuildRecordName(domain, record.Name);
        var normalized = values
            .Where(v => !string.IsNullOrWhiteSpace(v))
            .Select(v => v.Trim())
            .Distinct(StringComparer.OrdinalIgnoreCase)
            .ToList();

        var listUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets?name={WebUtility.UrlEncode(name)}&type={record.Type}";
        var listBody = await SendRequestAsync(HttpMethod.Get, listUrl, null);
        var listResp = JsonSerializer.Deserialize<HuaweiRecordSetList>(listBody) ?? new HuaweiRecordSetList();

        if (normalized.Count == 0)
        {
            foreach (var rs in listResp.Recordsets)
            {
                var delUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets/{rs.Id}";
                await SendRequestAsync(HttpMethod.Delete, delUrl, null);
            }
            return;
        }

        var ttl = record.TTL == 0 ? 300 : record.TTL;
        if (listResp.Recordsets.Count > 0)
        {
            var first = listResp.Recordsets[0];
            var updateUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets/{first.Id}";
            var payload = new Dictionary<string, object?>
            {
                ["name"] = first.Name,
                ["type"] = record.Type,
                ["ttl"] = ttl,
                ["records"] = normalized
            };
            await SendRequestAsync(HttpMethod.Put, updateUrl, JsonSerializer.SerializeToUtf8Bytes(payload));

            foreach (var extra in listResp.Recordsets.Skip(1))
            {
                var delUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets/{extra.Id}";
                await SendRequestAsync(HttpMethod.Delete, delUrl, null);
            }

            return;
        }

        var createUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets";
        var createPayload = new Dictionary<string, object?>
        {
            ["name"] = name,
            ["type"] = record.Type,
            ["ttl"] = ttl,
            ["description"] = "Created by CDN",
            ["records"] = normalized
        };
        await SendRequestAsync(HttpMethod.Post, createUrl, JsonSerializer.SerializeToUtf8Bytes(createPayload));
    }

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        var zoneId = await GetZoneIdAsync(domain);
        if (string.IsNullOrWhiteSpace(zoneId))
        {
            throw new InvalidOperationException($"huawei zone not found: {domain}");
        }

        var url = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets";
        var payload = new Dictionary<string, object?>
        {
            ["name"] = BuildRecordName(domain, record.Name),
            ["type"] = record.Type,
            ["ttl"] = record.TTL == 0 ? 300 : record.TTL,
            ["description"] = "Created by CDN",
            ["records"] = new[] { record.Value }
        };

        var body = await SendRequestAsync(HttpMethod.Post, url, JsonSerializer.SerializeToUtf8Bytes(payload));
        var parsed = JsonSerializer.Deserialize<HuaweiErrorResponse>(body) ?? new HuaweiErrorResponse();
        if (!string.IsNullOrWhiteSpace(parsed.Code))
        {
            if (parsed.Code.Contains("Duplicate", StringComparison.OrdinalIgnoreCase) ||
                (parsed.Message ?? string.Empty).Contains("already exists", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }
            throw new InvalidOperationException($"huawei error: {parsed.Code} - {parsed.Message}");
        }
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        var zoneId = await GetZoneIdAsync(domain);
        if (string.IsNullOrWhiteSpace(zoneId))
        {
            throw new InvalidOperationException($"huawei zone not found: {domain}");
        }

        var listUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets?name={WebUtility.UrlEncode(BuildRecordName(domain, record.Name))}&type={record.Type}";
        var listBody = await SendRequestAsync(HttpMethod.Get, listUrl, null);
        var listResp = JsonSerializer.Deserialize<HuaweiRecordSetList>(listBody) ?? new HuaweiRecordSetList();

        foreach (var rs in listResp.Recordsets)
        {
            if (rs.Records == null || rs.Records.Count == 0)
            {
                continue;
            }

            if (!rs.Records.Any(r => string.Equals(r, record.Value, StringComparison.OrdinalIgnoreCase)))
            {
                continue;
            }

            if (rs.Records.Count == 1)
            {
                var delUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets/{rs.Id}";
                await SendRequestAsync(HttpMethod.Delete, delUrl, null);
                continue;
            }

            var remaining = rs.Records.Where(v => !string.Equals(v, record.Value, StringComparison.OrdinalIgnoreCase)).ToList();
            var updateUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones/{zoneId}/recordsets/{rs.Id}";
            var payload = new Dictionary<string, object?>
            {
                ["name"] = rs.Name,
                ["type"] = record.Type,
                ["ttl"] = rs.Ttl == 0 ? 300 : rs.Ttl,
                ["records"] = remaining
            };
            await SendRequestAsync(HttpMethod.Put, updateUrl, JsonSerializer.SerializeToUtf8Bytes(payload));
        }
    }

    private async Task<string> GetZoneIdAsync(string domain)
    {
        var trimmed = (domain ?? string.Empty).Trim().TrimEnd('.');
        if (string.IsNullOrWhiteSpace(trimmed))
        {
            return string.Empty;
        }

        var candidates = new List<string>
        {
            trimmed,
            trimmed + "."
        };

        var parts = trimmed.Split('.', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length > 2)
        {
            var baseDomain = string.Join('.', parts[^2..]);
            if (!string.Equals(baseDomain, trimmed, StringComparison.OrdinalIgnoreCase))
            {
                candidates.Add(baseDomain);
                candidates.Add(baseDomain + ".");
            }
        }

        var region = _region;
        var regions = new List<string> { _region };
        if (!_regionProvided)
        {
            foreach (var r in new[] { "cn-north-4", "cn-east-2", "cn-east-3", "cn-south-1", "cn-south-4" })
            {
                if (!string.Equals(r, _region, StringComparison.OrdinalIgnoreCase))
                {
                    regions.Add(r);
                }
            }
        }

        foreach (var r in regions)
        {
            _region = r;
            var id = await LookupZoneIdCandidatesAsync(trimmed, candidates);
            if (!string.IsNullOrWhiteSpace(id))
            {
                return id;
            }
        }

        _region = region;
        return string.Empty;
    }

    private async Task<string> LookupZoneIdCandidatesAsync(string domain, List<string> candidates)
    {
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var name in candidates)
        {
            if (string.IsNullOrWhiteSpace(name))
            {
                continue;
            }
            if (!seen.Add(name))
            {
                continue;
            }

            var zones = await ListZonesAsync(name);
            var id = MatchZoneId(zones, domain);
            if (!string.IsNullOrWhiteSpace(id))
            {
                return id;
            }
        }

        var all = await ListZonesAsync(string.Empty);
        return MatchZoneId(all, domain);
    }

    private async Task<List<HuaweiZone>> ListZonesAsync(string name)
    {
        var baseUrl = $"https://dns.{_region}.myhuaweicloud.com/v2/zones";
        if (!string.IsNullOrWhiteSpace(name))
        {
            baseUrl += "?name=" + WebUtility.UrlEncode(name);
        }

        var zones = new List<HuaweiZone>();
        var nextUrl = baseUrl;
        while (!string.IsNullOrWhiteSpace(nextUrl))
        {
            var body = await SendRequestAsync(HttpMethod.Get, nextUrl, null);
            var parsed = JsonSerializer.Deserialize<HuaweiZonesResponse>(body) ?? new HuaweiZonesResponse();
            zones.AddRange(parsed.Zones);
            if (parsed.Links != null && parsed.Links.TryGetValue("next", out var next) && !string.IsNullOrWhiteSpace(next))
            {
                nextUrl = next;
            }
            else
            {
                nextUrl = string.Empty;
            }
        }

        return zones;
    }

    private static string MatchZoneId(List<HuaweiZone> zones, string domain)
    {
        var best = string.Empty;
        var bestLen = 0;
        var normalized = (domain ?? string.Empty).Trim().TrimEnd('.');
        foreach (var zone in zones)
        {
            var name = (zone.Name ?? string.Empty).Trim().TrimEnd('.');
            if (string.IsNullOrWhiteSpace(name))
            {
                continue;
            }
            if (normalized == name || normalized.EndsWith("." + name, StringComparison.OrdinalIgnoreCase))
            {
                if (name.Length > bestLen)
                {
                    best = zone.Id ?? string.Empty;
                    bestLen = name.Length;
                }
            }
        }

        return best;
    }

    private async Task<byte[]> SendRequestAsync(HttpMethod method, string url, byte[]? body)
    {
        var req = new HttpRequestMessage(method, url)
        {
            Content = body == null ? null : new ByteArrayContent(body)
        };

        req.Headers.TryAddWithoutValidation("content-type", "application/json");
        var uri = new Uri(url);
        req.Headers.Host = uri.Host;

        Sign(req, body ?? Array.Empty<byte>());

        var (status, respBody) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (status < HttpStatusCode.OK || status >= HttpStatusCode.MultipleChoices)
        {
            var message = HuaweiErrorMessageFromBody(respBody);
            if (!string.IsNullOrWhiteSpace(message))
            {
                throw new InvalidOperationException($"huawei api error: {message}");
            }

            throw new InvalidOperationException($"huawei api error: status {(int)status}");
        }

        return Encoding.UTF8.GetBytes(respBody);
    }

    private static string BuildRecordName(string domain, string name)
    {
        if (string.IsNullOrWhiteSpace(name) || name == "@")
        {
            return domain + ".";
        }

        return name + "." + domain + ".";
    }

    private void Sign(HttpRequestMessage req, byte[] body)
    {
        const string Algorithm = "SDK-HMAC-SHA256";
        const string Terminator = "sdk_request";

        var t = DateTime.UtcNow;
        var xSdkDate = t.ToString("yyyyMMdd'T'HHmmss'Z'");
        var date = t.ToString("yyyyMMdd");

        req.Headers.TryAddWithoutValidation("X-Sdk-Date", xSdkDate);

        var uri = req.RequestUri ?? new Uri("/");
        var path = uri.AbsolutePath;
        if (string.IsNullOrWhiteSpace(path))
        {
            path = "/";
        }
        if (!path.EndsWith("/", StringComparison.Ordinal))
        {
            path += "/";
        }

        var canonicalRequest = new StringBuilder();
        canonicalRequest.Append(req.Method.Method).Append('\n');
        canonicalRequest.Append(path).Append('\n');

        var query = WebUtility.UrlDecode(uri.Query).TrimStart('?');
        var queryPairs = new List<string>();
        if (!string.IsNullOrWhiteSpace(query))
        {
            foreach (var pair in query.Split('&', StringSplitOptions.RemoveEmptyEntries))
            {
                var parts = pair.Split('=', 2);
                var key = parts[0];
                var value = parts.Length > 1 ? parts[1] : string.Empty;
                queryPairs.Add(key + "=" + WebUtility.UrlEncode(value));
            }
        }
        queryPairs.Sort(StringComparer.Ordinal);
        canonicalRequest.Append(string.Join("&", queryPairs)).Append('\n');

        var signedHeaders = new[] { "content-type", "host", "x-sdk-date" };
        var canonicalHeaders = new StringBuilder();
        foreach (var header in signedHeaders)
        {
            var value = header switch
            {
                "content-type" => "application/json",
                "host" => req.Headers.Host ?? string.Empty,
                "x-sdk-date" => xSdkDate,
                _ => string.Empty
            };
            canonicalHeaders.Append(header).Append(':').Append(value.Trim()).Append('\n');
        }

        canonicalRequest.Append(canonicalHeaders).Append('\n');
        canonicalRequest.Append(string.Join(";", signedHeaders)).Append('\n');
        canonicalRequest.Append(Sha256Hex(body));

        var credentialScope = date + "/" + _region + "/dns/" + Terminator;
        var stringToSign = Algorithm + "\n" + xSdkDate + "\n" + credentialScope + "\n" + Sha256Hex(canonicalRequest.ToString());

        var kSecret = Encoding.UTF8.GetBytes("SDK" + _secretAccessKey);
        var kDate = HmacSha256(kSecret, date);
        var kRegion = HmacSha256(kDate, _region);
        var kService = HmacSha256(kRegion, "dns");
        var kSigning = HmacSha256(kService, Terminator);
        var signature = ToHex(HmacSha256(kSigning, stringToSign));

        var authHeader = $"{Algorithm} Credential={_accessKeyId}/{credentialScope}, SignedHeaders={string.Join(";", signedHeaders)}, Signature={signature}";
        req.Headers.TryAddWithoutValidation("Authorization", authHeader);
    }

    private static string HuaweiErrorMessageFromBody(string body)
    {
        try
        {
            var parsed = JsonSerializer.Deserialize<HuaweiErrorResponse>(body);
            if (parsed == null)
            {
                return string.Empty;
            }

            if (!string.IsNullOrWhiteSpace(parsed.Code) || !string.IsNullOrWhiteSpace(parsed.Message))
            {
                return string.IsNullOrWhiteSpace(parsed.Code) ? parsed.Message ?? string.Empty :
                    string.IsNullOrWhiteSpace(parsed.Message) ? parsed.Code : parsed.Code + " - " + parsed.Message;
            }

            if (!string.IsNullOrWhiteSpace(parsed.ErrorCode) || !string.IsNullOrWhiteSpace(parsed.ErrorMsg))
            {
                return string.IsNullOrWhiteSpace(parsed.ErrorCode) ? parsed.ErrorMsg ?? string.Empty :
                    string.IsNullOrWhiteSpace(parsed.ErrorMsg) ? parsed.ErrorCode : parsed.ErrorCode + " - " + parsed.ErrorMsg;
            }
        }
        catch
        {
        }

        return string.Empty;
    }

    private static string Sha256Hex(string data)
    {
        return Sha256Hex(Encoding.UTF8.GetBytes(data));
    }

    private static string Sha256Hex(byte[] data)
    {
        var hash = SHA256.HashData(data);
        return ToHex(hash);
    }

    private static byte[] HmacSha256(byte[] key, string msg)
    {
        using var hmac = new HMACSHA256(key);
        return hmac.ComputeHash(Encoding.UTF8.GetBytes(msg));
    }

    private static string ToHex(byte[] buffer)
    {
        var sb = new StringBuilder(buffer.Length * 2);
        foreach (var b in buffer)
        {
            sb.Append(b.ToString("x2"));
        }
        return sb.ToString();
    }

    private static string? GetString(JsonElement root, string name)
    {
        return root.TryGetProperty(name, out var prop) ? prop.GetString() : null;
    }

    private sealed class HuaweiRecordSetList
    {
        public List<HuaweiRecordSet> Recordsets { get; set; } = new();
    }

    private sealed class HuaweiRecordSet
    {
        public string? Id { get; set; }
        public string? Name { get; set; }
        public int Ttl { get; set; }
        public List<string> Records { get; set; } = new();
    }

    private sealed class HuaweiZone
    {
        public string? Id { get; set; }
        public string? Name { get; set; }
    }

    private sealed class HuaweiZonesResponse
    {
        public List<HuaweiZone> Zones { get; set; } = new();
        public Dictionary<string, string>? Links { get; set; }
        public string? Code { get; set; }
        public string? Message { get; set; }
        public string? ErrorCode { get; set; }
        public string? ErrorMsg { get; set; }
    }

    private sealed class HuaweiErrorResponse
    {
        public string? Code { get; set; }
        public string? Message { get; set; }
        public string? ErrorCode { get; set; }
        public string? ErrorMsg { get; set; }
    }
}
