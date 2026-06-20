using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class NameComDnsProvider : IDnsRecordProvider
{
    private readonly string _username;
    private readonly string _token;

    private NameComDnsProvider(string username, string token)
    {
        _username = username;
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
            var username = root.TryGetProperty("username", out var userProp) ? userProp.GetString() : null;
            var token = root.TryGetProperty("api_token", out var tokenProp) ? tokenProp.GetString() : null;
            if (string.IsNullOrWhiteSpace(token))
            {
                token = root.TryGetProperty("token", out var tokenAlt) ? tokenAlt.GetString() : null;
            }

            return new NameComDnsProvider((username ?? string.Empty).Trim(), (token ?? string.Empty).Trim());
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
        var payload = new Dictionary<string, object?>
        {
            ["host"] = record.Name,
            ["type"] = record.Type,
            ["answer"] = record.Value,
            ["ttl"] = record.TTL == 0 ? 300 : record.TTL
        };

        var body = JsonSerializer.Serialize(payload);
        var req = new HttpRequestMessage(HttpMethod.Post, $"https://api.name.com/v4/domains/{domain}/records")
        {
            Content = new StringContent(body, Encoding.UTF8, "application/json")
        };
        SetAuth(req);

        var (_, respBody) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(respBody))
        {
            return;
        }

        try
        {
            using var doc = JsonDocument.Parse(respBody);
            var root = doc.RootElement;
            var id = root.TryGetProperty("id", out var idProp) ? idProp.GetInt32() : 0;
            if (id != 0)
            {
                return;
            }

            var message = root.TryGetProperty("message", out var msgProp) ? msgProp.GetString() ?? string.Empty : string.Empty;
            var title = root.TryGetProperty("title", out var titleProp) ? titleProp.GetString() ?? string.Empty : string.Empty;
            if (message.Contains("duplicate", StringComparison.OrdinalIgnoreCase) ||
                title.Contains("duplicate", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }

            if (!string.IsNullOrWhiteSpace(message))
            {
                throw new InvalidOperationException($"name.com error: {message}");
            }

            if (respBody.Contains("permission", StringComparison.OrdinalIgnoreCase))
            {
                throw new InvalidOperationException($"name.com permission error: {respBody}");
            }
        }
        catch (JsonException)
        {
        }
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        var listReq = new HttpRequestMessage(HttpMethod.Get, $"https://api.name.com/v4/domains/{domain}/records");
        SetAuth(listReq);

        var (_, body) = await DnsHttp.SendAsync(listReq, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            return;
        }

        try
        {
            using var doc = JsonDocument.Parse(body);
            if (!doc.RootElement.TryGetProperty("records", out var records) || records.ValueKind != JsonValueKind.Array)
            {
                return;
            }

            foreach (var item in records.EnumerateArray())
            {
                var id = item.TryGetProperty("id", out var idProp) ? idProp.GetInt32() : 0;
                var type = item.TryGetProperty("type", out var typeProp) ? typeProp.GetString() : null;
                var host = item.TryGetProperty("host", out var hostProp) ? hostProp.GetString() : null;
                var answer = item.TryGetProperty("answer", out var answerProp) ? answerProp.GetString() : null;

                if (id == 0)
                {
                    continue;
                }
                if (!string.Equals(type, record.Type, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
                if (!string.Equals(host, record.Name, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
                if (!string.Equals(answer, record.Value, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }

                var delReq = new HttpRequestMessage(HttpMethod.Delete, $"https://api.name.com/v4/domains/{domain}/records/{id}");
                SetAuth(delReq);
                await DnsHttp.SendAsync(delReq, TimeSpan.FromSeconds(30));
                return;
            }
        }
        catch (JsonException)
        {
        }
    }

    private void SetAuth(HttpRequestMessage req)
    {
        if (string.IsNullOrWhiteSpace(_username))
        {
            return;
        }
        var token = Convert.ToBase64String(Encoding.UTF8.GetBytes(_username + ":" + _token));
        req.Headers.Authorization = new AuthenticationHeaderValue("Basic", token);
    }
}
