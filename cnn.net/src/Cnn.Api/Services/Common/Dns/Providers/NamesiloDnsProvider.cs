using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class NamesiloDnsProvider : IDnsRecordProvider
{
    private readonly string _apiKey;

    private NamesiloDnsProvider(string apiKey)
    {
        _apiKey = apiKey;
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
            var key = root.TryGetProperty("api_key", out var keyProp) ? keyProp.GetString() : null;
            return new NamesiloDnsProvider((key ?? string.Empty).Trim());
        }
        catch
        {
            return null;
        }
    }

    public Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain)
    {
        return GetRecordsInternalAsync(domain);
    }

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        var ttl = record.TTL == 0 ? 3600 : record.TTL;
        var query = new Dictionary<string, string>
        {
            ["version"] = "1",
            ["type"] = "json",
            ["key"] = _apiKey,
            ["domain"] = domain,
            ["rrtype"] = record.Type,
            ["rrhost"] = NormalizeHost(record.Name),
            ["rrvalue"] = record.Value,
            ["rrttl"] = ttl.ToString()
        };

        using var doc = await SendAsync("dnsAddRecord", query);
        EnsureSuccess(doc);
    }

    public Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        return DeleteRecordInternalAsync(domain, record);
    }

    private static string BuildQuery(Dictionary<string, string> values)
    {
        return string.Join("&", values.Select(kvp => $"{Uri.EscapeDataString(kvp.Key)}={Uri.EscapeDataString(kvp.Value ?? string.Empty)}"));
    }

    private async Task<IReadOnlyList<DnsRecord>> GetRecordsInternalAsync(string domain)
    {
        var query = new Dictionary<string, string>
        {
            ["version"] = "1",
            ["type"] = "json",
            ["key"] = _apiKey,
            ["domain"] = domain
        };

        using var doc = await SendAsync("dnsListRecords", query);
        var reply = doc.RootElement.TryGetProperty("reply", out var replyElement) ? replyElement : default;
        if (reply.ValueKind != JsonValueKind.Object)
        {
            return Array.Empty<DnsRecord>();
        }

        var resources = reply.TryGetProperty("resource_record", out var records) ? records : default;
        if (resources.ValueKind != JsonValueKind.Array)
        {
            return Array.Empty<DnsRecord>();
        }

        var output = new List<DnsRecord>();
        foreach (var item in resources.EnumerateArray())
        {
            var type = GetString(item, "type");
            var host = GetString(item, "host");
            var value = GetString(item, "value");
            var ttl = GetInt(item, "ttl");
            if (string.IsNullOrWhiteSpace(type) || string.IsNullOrWhiteSpace(value))
            {
                continue;
            }

            output.Add(new DnsRecord
            {
                Type = type,
                Name = NormalizeNamesiloHost(host, domain),
                Value = value,
                TTL = ttl <= 0 ? 3600 : ttl
            });
        }

        return output;
    }

    private async Task DeleteRecordInternalAsync(string domain, DnsRecord record)
    {
        var query = new Dictionary<string, string>
        {
            ["version"] = "1",
            ["type"] = "json",
            ["key"] = _apiKey,
            ["domain"] = domain
        };

        using var doc = await SendAsync("dnsListRecords", query);
        var reply = doc.RootElement.TryGetProperty("reply", out var replyElement) ? replyElement : default;
        if (reply.ValueKind != JsonValueKind.Object)
        {
            return;
        }

        var resources = reply.TryGetProperty("resource_record", out var records) ? records : default;
        if (resources.ValueKind != JsonValueKind.Array)
        {
            return;
        }

        var targetHost = NormalizeHost(record.Name);
        foreach (var item in resources.EnumerateArray())
        {
            var id = GetString(item, "record_id");
            if (string.IsNullOrWhiteSpace(id))
            {
                continue;
            }

            var type = GetString(item, "type");
            var host = GetString(item, "host");
            var value = GetString(item, "value");
            if (!string.Equals(type, record.Type, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            if (!IsHostMatch(host, domain, targetHost))
            {
                continue;
            }

            if (!string.Equals(value, record.Value, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            var deleteQuery = new Dictionary<string, string>
            {
                ["version"] = "1",
                ["type"] = "json",
                ["key"] = _apiKey,
                ["domain"] = domain,
                ["rrid"] = id
            };
            using var deleteDoc = await SendAsync("dnsDeleteRecord", deleteQuery);
            EnsureSuccess(deleteDoc);
        }
    }

    private static bool IsHostMatch(string? host, string domain, string target)
    {
        var normalized = NormalizeNamesiloHost(host, domain);
        return string.Equals(normalized, target, StringComparison.OrdinalIgnoreCase);
    }

    private static string NormalizeNamesiloHost(string? host, string domain)
    {
        var normalized = (host ?? string.Empty).Trim().TrimEnd('.');
        var domainKey = (domain ?? string.Empty).Trim().TrimEnd('.');
        if (string.IsNullOrWhiteSpace(normalized))
        {
            return "@";
        }

        if (string.Equals(normalized, domainKey, StringComparison.OrdinalIgnoreCase))
        {
            return "@";
        }

        if (!string.IsNullOrWhiteSpace(domainKey))
        {
            var suffix = "." + domainKey;
            if (normalized.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
            {
                return normalized[..^suffix.Length];
            }
        }

        return normalized;
    }

    private static string NormalizeHost(string? host)
    {
        var value = (host ?? string.Empty).Trim();
        return string.IsNullOrWhiteSpace(value) ? "@" : value;
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

    private static string? GetString(JsonElement element, string name)
    {
        return element.TryGetProperty(name, out var prop) ? prop.GetString() : null;
    }

    private static void EnsureSuccess(JsonDocument doc)
    {
        var reply = doc.RootElement.TryGetProperty("reply", out var replyElement) ? replyElement : default;
        if (reply.ValueKind != JsonValueKind.Object)
        {
            throw new InvalidOperationException("namesilo invalid response");
        }

        var code = GetString(reply, "code");
        if (string.Equals(code, "300", StringComparison.OrdinalIgnoreCase) ||
            string.Equals(code, "280", StringComparison.OrdinalIgnoreCase))
        {
            return;
        }

        var detail = GetString(reply, "detail");
        if (!string.IsNullOrWhiteSpace(detail))
        {
            throw new InvalidOperationException($"namesilo error: {detail}");
        }

        throw new InvalidOperationException("namesilo error: " + (code ?? "unknown"));
    }

    private async Task<JsonDocument> SendAsync(string operation, Dictionary<string, string> query)
    {
        var url = "https://www.namesilo.com/api/" + operation + "?" + BuildQuery(query);
        var req = new HttpRequestMessage(HttpMethod.Get, url);
        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            throw new InvalidOperationException("namesilo empty response");
        }

        return JsonDocument.Parse(body);
    }
}
