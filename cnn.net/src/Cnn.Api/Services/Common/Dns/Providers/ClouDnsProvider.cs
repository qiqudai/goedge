using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class ClouDnsProvider : IDnsRecordProvider
{
    private static readonly int[] AllowedTtlValues =
    {
        60, 300, 900, 1800, 3600, 21600, 43200, 86400, 172800, 259200, 604800, 1209600, 2419200
    };

    private readonly string _authId;
    private readonly string _authPassword;

    private ClouDnsProvider(string authId, string authPassword)
    {
        _authId = authId;
        _authPassword = authPassword;
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
            var authId = root.TryGetProperty("auth_id", out var idProp) ? idProp.GetString() : null;
            var authPassword = root.TryGetProperty("auth_password", out var passProp) ? passProp.GetString() : null;
            return new ClouDnsProvider((authId ?? string.Empty).Trim(), (authPassword ?? string.Empty).Trim());
        }
        catch
        {
            return null;
        }
    }

    public async Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain)
    {
        var root = await SendAsync("dns/records.json", new Dictionary<string, string>
        {
            ["auth-id"] = _authId,
            ["auth-password"] = _authPassword,
            ["domain-name"] = domain
        });

        var records = new List<DnsRecord>();
        foreach (var (id, item) in EnumerateRecordObjects(root))
        {
            var type = GetString(item, "type");
            var host = GetString(item, "host");
            var value = GetString(item, "record");
            if (string.IsNullOrWhiteSpace(type) || string.IsNullOrWhiteSpace(value))
            {
                continue;
            }

            var ttl = GetInt(item, "ttl");
            records.Add(new DnsRecord
            {
                Type = type,
                Name = NormalizeHost(host),
                Value = value,
                TTL = ttl <= 0 ? 3600 : ttl
            });
        }

        return records;
    }

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        var ttl = NormalizeTtl(record.TTL);
        var root = await SendAsync("dns/add-record.json", new Dictionary<string, string>
        {
            ["auth-id"] = _authId,
            ["auth-password"] = _authPassword,
            ["domain-name"] = domain,
            ["record-type"] = record.Type,
            ["host"] = NormalizeHost(record.Name),
            ["record"] = record.Value,
            ["ttl"] = ttl.ToString()
        });

        var status = GetString(root, "status");
        if (!string.Equals(status, "Success", StringComparison.OrdinalIgnoreCase))
        {
            var message = GetString(root, "statusDescription");
            if (!string.IsNullOrWhiteSpace(message) &&
                message.Contains("exist", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }

            if (!string.IsNullOrWhiteSpace(message))
            {
                throw new InvalidOperationException($"cloudns error: {message}");
            }
        }
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        var root = await SendAsync("dns/records.json", new Dictionary<string, string>
        {
            ["auth-id"] = _authId,
            ["auth-password"] = _authPassword,
            ["domain-name"] = domain
        });

        var host = NormalizeHost(record.Name);
        foreach (var (id, item) in EnumerateRecordObjects(root))
        {
            var type = GetString(item, "type");
            var name = NormalizeHost(GetString(item, "host"));
            var value = GetString(item, "record");
            if (!string.Equals(type, record.Type, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.Equals(name, host, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.Equals(value, record.Value, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            var deleteRoot = await SendAsync("dns/delete-record.json", new Dictionary<string, string>
            {
                ["auth-id"] = _authId,
                ["auth-password"] = _authPassword,
                ["domain-name"] = domain,
                ["record-id"] = id
            });

            var status = GetString(deleteRoot, "status");
            if (string.Equals(status, "Success", StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            var message = GetString(deleteRoot, "statusDescription");
            if (!string.IsNullOrWhiteSpace(message))
            {
                throw new InvalidOperationException($"cloudns error: {message}");
            }
        }
    }

    private static string NormalizeHost(string? host)
    {
        var value = (host ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(value))
        {
            return "@";
        }

        return value;
    }

    private static int NormalizeTtl(int ttl)
    {
        if (ttl <= 0)
        {
            return 3600;
        }

        foreach (var allowed in AllowedTtlValues)
        {
            if (ttl <= allowed)
            {
                return allowed;
            }
        }

        return AllowedTtlValues[^1];
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

    private static IEnumerable<(string Id, JsonElement Record)> EnumerateRecordObjects(JsonElement root)
    {
        if (root.ValueKind != JsonValueKind.Object)
        {
            yield break;
        }

        if (root.TryGetProperty("status", out var status) &&
            status.ValueKind == JsonValueKind.String &&
            !string.Equals(status.GetString(), "Success", StringComparison.OrdinalIgnoreCase))
        {
            var message = GetString(root, "statusDescription");
            if (!string.IsNullOrWhiteSpace(message))
            {
                throw new InvalidOperationException($"cloudns error: {message}");
            }
        }

        foreach (var prop in root.EnumerateObject())
        {
            if (prop.Value.ValueKind != JsonValueKind.Object)
            {
                continue;
            }

            var id = prop.Value.TryGetProperty("id", out var idProp) ? idProp.GetString() : null;
            if (string.IsNullOrWhiteSpace(id))
            {
                id = prop.Name;
            }
            if (string.IsNullOrWhiteSpace(id))
            {
                continue;
            }

            yield return (id, prop.Value);
        }
    }

    private static async Task<JsonElement> SendAsync(string path, Dictionary<string, string> query)
    {
        var url = "https://api.cloudns.net/" + path;
        if (query.Count > 0)
        {
            url += "?" + BuildQuery(query);
        }

        var req = new HttpRequestMessage(HttpMethod.Get, url);
        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            throw new InvalidOperationException("cloudns empty response");
        }

        using var doc = JsonDocument.Parse(body);
        return doc.RootElement.Clone();
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
}
