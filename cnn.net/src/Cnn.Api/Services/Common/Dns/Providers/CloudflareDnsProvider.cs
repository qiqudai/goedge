using System.Net;
using System.Net.Http.Json;
using System.Text;
using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class CloudflareDnsProvider : IDnsRecordProvider
{
    private readonly string _email;
    private readonly string _apiKey;

    private CloudflareDnsProvider(string email, string apiKey)
    {
        _email = email;
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
            var email = root.TryGetProperty("email", out var emailProp) ? emailProp.GetString() : null;
            var apiKey = root.TryGetProperty("api_key", out var keyProp) ? keyProp.GetString() : null;
            email = (email ?? string.Empty).Trim();
            apiKey = (apiKey ?? string.Empty).Trim();
            return new CloudflareDnsProvider(email, apiKey);
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

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        var zoneId = await GetZoneIdAsync(domain);
        if (string.IsNullOrWhiteSpace(zoneId))
        {
            throw new InvalidOperationException($"cloudflare zone not found: {domain}");
        }

        var fullName = BuildFullName(domain, record.Name);
        var payload = new Dictionary<string, object?>
        {
            ["type"] = record.Type,
            ["name"] = fullName,
            ["content"] = record.Value,
            ["ttl"] = record.TTL == 0 ? 1 : record.TTL,
            ["proxied"] = false
        };

        var req = new HttpRequestMessage(HttpMethod.Post, $"https://api.cloudflare.com/client/v4/zones/{zoneId}/dns_records")
        {
            Content = new StringContent(JsonSerializer.Serialize(payload), Encoding.UTF8, "application/json")
        };
        SetHeaders(req);

        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            return;
        }

        try
        {
            using var doc = JsonDocument.Parse(body);
            var root = doc.RootElement;
            if (root.TryGetProperty("success", out var success) && success.ValueKind == JsonValueKind.True)
            {
                return;
            }

            var message = ExtractFirstError(root);
            if (!string.IsNullOrWhiteSpace(message) && message.Contains("already exists", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }

            if (!string.IsNullOrWhiteSpace(message))
            {
                throw new InvalidOperationException($"cloudflare error: {message}");
            }
        }
        catch (JsonException)
        {
        }
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        var zoneId = await GetZoneIdAsync(domain);
        if (string.IsNullOrWhiteSpace(zoneId))
        {
            throw new InvalidOperationException($"cloudflare zone not found: {domain}");
        }

        var fullName = BuildFullName(domain, record.Name);
        var listUrl = $"https://api.cloudflare.com/client/v4/zones/{zoneId}/dns_records?type={WebUtility.UrlEncode(record.Type)}&name={WebUtility.UrlEncode(fullName)}&content={WebUtility.UrlEncode(record.Value)}";
        var listReq = new HttpRequestMessage(HttpMethod.Get, listUrl);
        SetHeaders(listReq);

        var (_, listBody) = await DnsHttp.SendAsync(listReq, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(listBody))
        {
            return;
        }

        using var doc = JsonDocument.Parse(listBody);
        if (!doc.RootElement.TryGetProperty("result", out var result) || result.ValueKind != JsonValueKind.Array)
        {
            return;
        }

        foreach (var item in result.EnumerateArray())
        {
            if (!item.TryGetProperty("id", out var idProp))
            {
                continue;
            }
            var id = idProp.GetString();
            if (string.IsNullOrWhiteSpace(id))
            {
                continue;
            }

            var delReq = new HttpRequestMessage(HttpMethod.Delete, $"https://api.cloudflare.com/client/v4/zones/{zoneId}/dns_records/{id}");
            SetHeaders(delReq);
            await DnsHttp.SendAsync(delReq, TimeSpan.FromSeconds(30));
        }
    }

    private async Task<string> GetZoneIdAsync(string domain)
    {
        var zoneId = await LookupZoneIdAsync(domain);
        if (!string.IsNullOrWhiteSpace(zoneId))
        {
            return zoneId;
        }

        var parts = domain.Split('.', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length <= 2)
        {
            return string.Empty;
        }

        var fallback = string.Join('.', parts[^2..]);
        return await LookupZoneIdAsync(fallback);
    }

    private async Task<string> LookupZoneIdAsync(string domain)
    {
        var req = new HttpRequestMessage(HttpMethod.Get, $"https://api.cloudflare.com/client/v4/zones?name={WebUtility.UrlEncode(domain)}");
        SetHeaders(req);

        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            return string.Empty;
        }

        using var doc = JsonDocument.Parse(body);
        var root = doc.RootElement;
        if (!root.TryGetProperty("success", out var success) || success.ValueKind != JsonValueKind.True)
        {
            return string.Empty;
        }

        if (!root.TryGetProperty("result", out var result) || result.ValueKind != JsonValueKind.Array)
        {
            return string.Empty;
        }

        foreach (var item in result.EnumerateArray())
        {
            if (item.TryGetProperty("id", out var idProp))
            {
                return idProp.GetString() ?? string.Empty;
            }
        }

        return string.Empty;
    }

    private void SetHeaders(HttpRequestMessage req)
    {
        req.Headers.TryAddWithoutValidation("Content-Type", "application/json");
        if (!string.IsNullOrWhiteSpace(_email))
        {
            req.Headers.TryAddWithoutValidation("X-Auth-Email", _email);
            req.Headers.TryAddWithoutValidation("X-Auth-Key", _apiKey);
            return;
        }

        req.Headers.TryAddWithoutValidation("Authorization", "Bearer " + _apiKey);
    }

    private static string BuildFullName(string domain, string name)
    {
        if (string.IsNullOrWhiteSpace(name) || name == "@")
        {
            return domain;
        }

        if (name.EndsWith(domain, StringComparison.OrdinalIgnoreCase))
        {
            return name;
        }

        return name + "." + domain;
    }

    private static string ExtractFirstError(JsonElement root)
    {
        if (!root.TryGetProperty("errors", out var errors) || errors.ValueKind != JsonValueKind.Array)
        {
            return string.Empty;
        }

        foreach (var err in errors.EnumerateArray())
        {
            if (err.TryGetProperty("message", out var msg))
            {
                return msg.GetString() ?? string.Empty;
            }
        }

        return string.Empty;
    }
}
