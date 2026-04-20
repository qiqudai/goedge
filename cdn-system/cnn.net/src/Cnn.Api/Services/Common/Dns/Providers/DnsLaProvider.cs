using System.Net;
using System.Text;
using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class DnsLaProvider : IDnsRecordProvider
{
    private const string BaseUrl = "https://api.dns.la";
    private const int DefaultTtl = 600;
    private const int PageSize = 200;

    private static readonly IReadOnlyDictionary<string, int> TypeNameToCode = new Dictionary<string, int>(StringComparer.OrdinalIgnoreCase)
    {
        ["A"] = 1,
        ["NS"] = 2,
        ["CNAME"] = 5,
        ["SOA"] = 6,
        ["PTR"] = 12,
        ["MX"] = 15,
        ["TXT"] = 16,
        ["AAAA"] = 28,
        ["SRV"] = 33,
        ["NAPTR"] = 35,
        ["SPF"] = 99,
        ["SVCB"] = 64,
        ["HTTPS"] = 65,
        ["CAA"] = 257
    };

    private static readonly IReadOnlyDictionary<int, string> TypeCodeToName = TypeNameToCode.ToDictionary(k => k.Value, v => v.Key);

    private readonly string _token;

    private DnsLaProvider(string token)
    {
        _token = token;
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
            var id = GetString(root, "api_id");
            var secret = GetString(root, "api_pass");
            if (string.IsNullOrWhiteSpace(id) || string.IsNullOrWhiteSpace(secret))
            {
                id ??= GetString(root, "id");
                secret ??= GetString(root, "secret");
            }

            id = (id ?? string.Empty).Trim();
            secret = (secret ?? string.Empty).Trim();
            if (string.IsNullOrWhiteSpace(id) || string.IsNullOrWhiteSpace(secret))
            {
                return null;
            }

            var token = Convert.ToBase64String(Encoding.UTF8.GetBytes(id + ":" + secret));
            return new DnsLaProvider(token);
        }
        catch
        {
            return null;
        }
    }

    public async Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain)
    {
        domain = NormalizeDomain(domain);
        if (string.IsNullOrWhiteSpace(domain))
        {
            throw new InvalidOperationException("dnsla domain required");
        }

        var pageIndex = 1;
        var records = new List<DnsRecord>();
        while (true)
        {
            var response = await ListRecordsAsync(domain, pageIndex, PageSize);
            foreach (var item in response.Data.Results)
            {
                var line = (item.LineCode ?? string.Empty).Trim();
                if (string.IsNullOrWhiteSpace(line))
                {
                    line = (item.LineName ?? string.Empty).Trim();
                }

                records.Add(new DnsRecord
                {
                    Type = TypeCodeToName.TryGetValue(item.Type, out var name) ? name : item.Type.ToString(),
                    Name = NormalizeRecordName(domain, item.Host),
                    Value = (item.Data ?? string.Empty).Trim(),
                    TTL = item.Ttl,
                    Line = line
                });
            }

            if (response.Data.Results.Count == 0)
            {
                break;
            }
            if (response.Data.Total > 0 && records.Count >= response.Data.Total)
            {
                break;
            }
            if (response.Data.Results.Count < PageSize)
            {
                break;
            }

            pageIndex++;
        }

        return records;
    }

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        domain = NormalizeDomain(domain);
        if (string.IsNullOrWhiteSpace(domain))
        {
            throw new InvalidOperationException("dnsla domain required");
        }

        if (!TypeNameToCode.TryGetValue(record.Type.Trim(), out var typeCode))
        {
            throw new InvalidOperationException($"dnsla unsupported record type: {record.Type}");
        }

        var host = NormalizeRecordName(domain, record.Name);
        if (string.IsNullOrWhiteSpace(host))
        {
            host = "@";
        }

        var ttl = record.TTL > 0 ? record.TTL : DefaultTtl;
        var payload = new Dictionary<string, object?>
        {
            ["domain"] = domain,
            ["host"] = host,
            ["type"] = typeCode,
            ["data"] = record.Value?.Trim(),
            ["ttl"] = ttl
        };

        if (!string.IsNullOrWhiteSpace(record.Line))
        {
            payload["lineCode"] = record.Line.Trim();
            payload["lineId"] = record.Line.Trim();
        }
        if (record.Weight > 0)
        {
            payload["weight"] = record.Weight;
        }

        var body = await DoRequestAsync(HttpMethod.Post, "/api/record", null, payload);
        var resp = JsonSerializer.Deserialize<DnsLaCreateRecordResponse>(body) ?? new DnsLaCreateRecordResponse();
        if (resp.Code != 200)
        {
            var msg = resp.Msg ?? string.Empty;
            if (!string.IsNullOrWhiteSpace(msg))
            {
                var lower = msg.ToLowerInvariant();
                if (msg.Contains("exists", StringComparison.OrdinalIgnoreCase) || lower.Contains("exists"))
                {
                    return;
                }
            }

            throw new InvalidOperationException($"dnsla add record error: {resp.Code} - {resp.Msg}");
        }
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        domain = NormalizeDomain(domain);
        if (string.IsNullOrWhiteSpace(domain))
        {
            throw new InvalidOperationException("dnsla domain required");
        }

        var ids = await FindRecordIdsAsync(domain, record);
        if (ids.Count == 0)
        {
            return;
        }

        foreach (var id in ids)
        {
            var query = new Dictionary<string, string> { ["id"] = id };
            var body = await DoRequestAsync(HttpMethod.Delete, "/api/record", query, null);
            var resp = JsonSerializer.Deserialize<DnsLaCommonResponse>(body) ?? new DnsLaCommonResponse();
            if (resp.Code != 200)
            {
                throw new InvalidOperationException($"dnsla delete record error: {resp.Code} - {resp.Msg}");
            }
        }
    }

    private async Task<DnsLaRecordListResponse> ListRecordsAsync(string domain, int pageIndex, int pageSize)
    {
        if (pageIndex <= 0)
        {
            pageIndex = 1;
        }
        if (pageSize <= 0)
        {
            pageSize = PageSize;
        }

        var query = new Dictionary<string, string>
        {
            ["domain"] = domain,
            ["pageIndex"] = pageIndex.ToString(),
            ["pageSize"] = pageSize.ToString()
        };

        var body = await DoRequestAsync(HttpMethod.Get, "/api/recordList", query, null);
        var resp = JsonSerializer.Deserialize<DnsLaRecordListResponse>(body) ?? new DnsLaRecordListResponse();
        if (resp.Code != 200)
        {
            throw new InvalidOperationException($"dnsla recordList error: {resp.Code} - {resp.Msg}");
        }

        return resp;
    }

    private async Task<List<string>> FindRecordIdsAsync(string domain, DnsRecord record)
    {
        if (!TypeNameToCode.TryGetValue(record.Type.Trim(), out var typeCode))
        {
            throw new InvalidOperationException($"dnsla unsupported record type: {record.Type}");
        }

        var desiredName = NormalizeRecordName(domain, record.Name);
        var desiredValue = record.Value?.Trim() ?? string.Empty;
        var desiredLine = record.Line?.Trim() ?? string.Empty;

        var pageIndex = 1;
        var matches = new List<string>();
        while (true)
        {
            var resp = await ListRecordsAsync(domain, pageIndex, PageSize);
            foreach (var item in resp.Data.Results)
            {
                if (!string.Equals(NormalizeRecordName(domain, item.Host), desiredName, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
                if (item.Type != typeCode)
                {
                    continue;
                }
                if (!string.IsNullOrWhiteSpace(desiredValue) && !string.Equals(item.Data?.Trim(), desiredValue, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
                if (!string.IsNullOrWhiteSpace(desiredLine))
                {
                    var lineCode = item.LineCode?.Trim() ?? string.Empty;
                    var lineName = item.LineName?.Trim() ?? string.Empty;
                    if (!string.Equals(desiredLine, lineCode, StringComparison.OrdinalIgnoreCase) &&
                        !string.Equals(desiredLine, lineName, StringComparison.OrdinalIgnoreCase))
                    {
                        continue;
                    }
                }

                if (!string.IsNullOrWhiteSpace(item.Id))
                {
                    matches.Add(item.Id!);
                }
            }

            if (resp.Data.Results.Count == 0)
            {
                break;
            }
            if (resp.Data.Total > 0 && pageIndex * PageSize >= resp.Data.Total)
            {
                break;
            }
            if (resp.Data.Results.Count < PageSize)
            {
                break;
            }

            pageIndex++;
        }

        return matches;
    }

    private async Task<string> DoRequestAsync(HttpMethod method, string path, Dictionary<string, string>? query, object? payload)
    {
        if (!path.StartsWith("/", StringComparison.Ordinal))
        {
            path = "/" + path;
        }

        var url = BaseUrl.TrimEnd('/') + path;
        if (query is { Count: > 0 })
        {
            var queryString = string.Join("&", query.Select(kvp => $"{WebUtility.UrlEncode(kvp.Key)}={WebUtility.UrlEncode(kvp.Value)}"));
            url += "?" + queryString;
        }

        HttpContent? content = null;
        if (payload != null)
        {
            var json = JsonSerializer.Serialize(payload);
            content = new StringContent(json, Encoding.UTF8, "application/json");
        }

        var req = new HttpRequestMessage(method, url)
        {
            Content = content
        };
        req.Headers.TryAddWithoutValidation("Authorization", "Basic " + _token);
        req.Headers.TryAddWithoutValidation("Accept", "application/json");

        var (status, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            throw new InvalidOperationException($"dnsla empty response, status={status}");
        }

        return body;
    }

    private static string NormalizeDomain(string domain)
    {
        return (domain ?? string.Empty).Trim().TrimEnd('.');
    }

    private static string NormalizeRecordName(string domain, string? name)
    {
        domain = NormalizeDomain(domain);
        var host = (name ?? string.Empty).Trim().TrimEnd('.');
        if (string.IsNullOrWhiteSpace(host))
        {
            return "@";
        }

        if (string.IsNullOrWhiteSpace(domain))
        {
            return host;
        }

        if (string.Equals(host, domain, StringComparison.OrdinalIgnoreCase))
        {
            return "@";
        }

        var suffix = "." + domain;
        if (host.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
        {
            var trimmed = host[..^suffix.Length];
            return string.IsNullOrWhiteSpace(trimmed) ? "@" : trimmed;
        }

        return host;
    }

    private static string? GetString(JsonElement root, string name)
    {
        return root.TryGetProperty(name, out var prop) ? prop.GetString() : null;
    }

    private sealed class DnsLaRecordListResponse
    {
        public int Code { get; set; }
        public string? Msg { get; set; }
        public DnsLaRecordListData Data { get; set; } = new();
    }

    private sealed class DnsLaRecordListData
    {
        public int Total { get; set; }
        public List<DnsLaRecordItem> Results { get; set; } = new();
    }

    private sealed class DnsLaRecordItem
    {
        public string? Id { get; set; }
        public string? Host { get; set; }
        public int Type { get; set; }
        public string? Data { get; set; }
        public int Ttl { get; set; }
        public string? LineId { get; set; }
        public string? LineCode { get; set; }
        public string? LineName { get; set; }
    }

    private sealed class DnsLaCreateRecordResponse
    {
        public int Code { get; set; }
        public string? Msg { get; set; }
    }

    private sealed class DnsLaCommonResponse
    {
        public int Code { get; set; }
        public string? Msg { get; set; }
        public object? Data { get; set; }
    }
}
