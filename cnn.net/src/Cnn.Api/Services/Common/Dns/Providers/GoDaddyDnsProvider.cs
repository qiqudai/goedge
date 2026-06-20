using System.Net.Http.Json;
using System.Text;
using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class GoDaddyDnsProvider : IDnsRecordProvider
{
    private readonly string _apiKey;
    private readonly string _apiSecret;

    private GoDaddyDnsProvider(string apiKey, string apiSecret)
    {
        _apiKey = apiKey;
        _apiSecret = apiSecret;
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
            var apiKey = root.TryGetProperty("api_key", out var keyProp) ? keyProp.GetString() : null;
            var apiSecret = root.TryGetProperty("api_secret", out var secProp) ? secProp.GetString() : null;
            apiKey = (apiKey ?? string.Empty).Trim();
            apiSecret = (apiSecret ?? string.Empty).Trim();
            return new GoDaddyDnsProvider(apiKey, apiSecret);
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
        var payload = new[]
        {
            new Dictionary<string, object?>
            {
                ["type"] = record.Type,
                ["name"] = record.Name,
                ["data"] = record.Value,
                ["ttl"] = record.TTL == 0 ? 600 : record.TTL
            }
        };

        var req = new HttpRequestMessage(HttpMethod.Patch, $"https://api.godaddy.com/v1/domains/{domain}/records")
        {
            Content = new StringContent(JsonSerializer.Serialize(payload), Encoding.UTF8, "application/json")
        };
        SetHeaders(req);

        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            return;
        }

        TryThrowError(body, "godaddy");
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        var getReq = new HttpRequestMessage(HttpMethod.Get,
            $"https://api.godaddy.com/v1/domains/{domain}/records/{record.Type}/{record.Name}");
        SetHeaders(getReq);

        var (_, body) = await DnsHttp.SendAsync(getReq, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            return;
        }

        var existing = JsonSerializer.Deserialize<List<GoDaddyRecord>>(body) ?? new List<GoDaddyRecord>();
        var remaining = new List<Dictionary<string, object?>>();
        var found = false;
        foreach (var item in existing)
        {
            if (string.Equals(item.Data, record.Value, StringComparison.OrdinalIgnoreCase))
            {
                found = true;
                continue;
            }

            remaining.Add(new Dictionary<string, object?>
            {
                ["type"] = item.Type,
                ["name"] = item.Name,
                ["data"] = item.Data,
                ["ttl"] = item.Ttl
            });
        }

        if (!found)
        {
            return;
        }

        var patchReq = new HttpRequestMessage(HttpMethod.Patch,
            $"https://api.godaddy.com/v1/domains/{domain}/records/{record.Type}/{record.Name}")
        {
            Content = new StringContent(JsonSerializer.Serialize(remaining), Encoding.UTF8, "application/json")
        };
        SetHeaders(patchReq);

        var (_, patchBody) = await DnsHttp.SendAsync(patchReq, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(patchBody))
        {
            return;
        }

        TryThrowError(patchBody, "godaddy");
    }

    private void SetHeaders(HttpRequestMessage req)
    {
        req.Headers.TryAddWithoutValidation("Authorization", $"sso-key {_apiKey}:{_apiSecret}");
        req.Headers.TryAddWithoutValidation("Accept", "application/json");
    }

    private static void TryThrowError(string body, string provider)
    {
        try
        {
            using var doc = JsonDocument.Parse(body);
            var root = doc.RootElement;
            if (root.TryGetProperty("code", out var codeProp) && codeProp.ValueKind == JsonValueKind.String)
            {
                var code = codeProp.GetString() ?? string.Empty;
                var message = root.TryGetProperty("message", out var msgProp) ? msgProp.GetString() : null;
                if (!string.IsNullOrWhiteSpace(code))
                {
                    throw new InvalidOperationException($"{provider} error: {code} - {message}");
                }
            }
        }
        catch (JsonException)
        {
        }
    }

    private sealed class GoDaddyRecord
    {
        public string? Data { get; set; }
        public string? Name { get; set; }
        public string? Type { get; set; }
        public int Ttl { get; set; }
    }
}
