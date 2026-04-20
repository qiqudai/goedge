using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class Dns51Provider : IDnsRecordProvider
{
    private Dns51Provider()
    {
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
            var id = root.TryGetProperty("app_id", out var idProp) ? idProp.GetString() : null;
            var secret = root.TryGetProperty("app_secret", out var secProp) ? secProp.GetString() : null;
            if (string.IsNullOrWhiteSpace(id) || string.IsNullOrWhiteSpace(secret))
            {
                id = root.TryGetProperty("id", out var id2) ? id2.GetString() : null;
                secret = root.TryGetProperty("secret", out var sec2) ? sec2.GetString() : null;
            }

            return new Dns51Provider();
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

    public Task AddRecordAsync(string domain, DnsRecord record)
    {
        return Task.CompletedTask;
    }

    public Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        return Task.CompletedTask;
    }
}
