using System.Net.Http.Headers;
using System.Text;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Stats;

public sealed record ClickHouseHttpConfig(string BaseUrl, string? User, string? Pass, string? Database);

public static class ClickHouseHttpHelper
{
    private static readonly HttpClient HttpClient = new()
    {
        Timeout = TimeSpan.FromSeconds(5)
    };

    public static ClickHouseHttpConfig? ResolveConfig(IConfiguration configuration)
    {
        var dsn = configuration["ClickHouse:Dsn"]
            ?? configuration["ClickHouse:DSN"]
            ?? configuration["ClickHouse:HttpDsn"]
            ?? configuration["ClickHouse:HttpDSN"];
        if (string.IsNullOrWhiteSpace(dsn))
        {
            return null;
        }

        if (!Uri.TryCreate(dsn.Trim(), UriKind.Absolute, out var uri))
        {
            return null;
        }

        if (!string.Equals(uri.Scheme, "http", StringComparison.OrdinalIgnoreCase) &&
            !string.Equals(uri.Scheme, "https", StringComparison.OrdinalIgnoreCase))
        {
            return null;
        }

        var database = uri.AbsolutePath.Trim('/');
        if (string.IsNullOrWhiteSpace(database))
        {
            var query = uri.Query.TrimStart('?');
            if (!string.IsNullOrWhiteSpace(query))
            {
                foreach (var pair in query.Split('&', StringSplitOptions.RemoveEmptyEntries))
                {
                    var kv = pair.Split('=', 2);
                    if (kv.Length == 2 && string.Equals(kv[0], "database", StringComparison.OrdinalIgnoreCase))
                    {
                        database = Uri.UnescapeDataString(kv[1]);
                        break;
                    }
                }
            }
        }

        string? user = null;
        string? pass = null;
        if (!string.IsNullOrWhiteSpace(uri.UserInfo))
        {
            var parts = uri.UserInfo.Split(':', 2);
            user = Uri.UnescapeDataString(parts[0]);
            if (parts.Length > 1)
            {
                pass = Uri.UnescapeDataString(parts[1]);
            }
        }

        var baseUrl = uri.GetLeftPart(UriPartial.Authority);
        return new ClickHouseHttpConfig(baseUrl, user, pass, string.IsNullOrWhiteSpace(database) ? null : database);
    }

    public static async Task<string[]?> QueryRowsAsync(ClickHouseHttpConfig config, string query, CancellationToken cancellationToken)
    {
        var builder = new StringBuilder();
        builder.Append(config.BaseUrl.TrimEnd('/'));
        builder.Append("/?query=").Append(Uri.EscapeDataString(query));
        if (!string.IsNullOrWhiteSpace(config.Database))
        {
            builder.Append("&database=").Append(Uri.EscapeDataString(config.Database));
        }

        using var request = new HttpRequestMessage(HttpMethod.Post, builder.ToString())
        {
            // ClickHouse 22.1 requires Content-Length for POST requests.
            Content = new StringContent(string.Empty, Encoding.UTF8, "text/plain")
        };
        if (!string.IsNullOrWhiteSpace(config.User))
        {
            var token = Convert.ToBase64String(Encoding.UTF8.GetBytes($"{config.User}:{config.Pass ?? string.Empty}"));
            request.Headers.Authorization = new AuthenticationHeaderValue("Basic", token);
        }

        using var response = await HttpClient.SendAsync(request, cancellationToken);
        if (!response.IsSuccessStatusCode)
        {
            return null;
        }

        var body = await response.Content.ReadAsStringAsync(cancellationToken);
        if (string.IsNullOrWhiteSpace(body))
        {
            return Array.Empty<string>();
        }

        return body.Split('\n', StringSplitOptions.RemoveEmptyEntries);
    }

    public static async Task<bool> ExecuteAsync(ClickHouseHttpConfig config, string query, CancellationToken cancellationToken)
    {
        var builder = new StringBuilder();
        builder.Append(config.BaseUrl.TrimEnd('/'));
        builder.Append("/?query=").Append(Uri.EscapeDataString(query));
        if (!string.IsNullOrWhiteSpace(config.Database))
        {
            builder.Append("&database=").Append(Uri.EscapeDataString(config.Database));
        }

        using var request = new HttpRequestMessage(HttpMethod.Post, builder.ToString())
        {
            Content = new StringContent(string.Empty, Encoding.UTF8, "text/plain")
        };
        if (!string.IsNullOrWhiteSpace(config.User))
        {
            var token = Convert.ToBase64String(Encoding.UTF8.GetBytes($"{config.User}:{config.Pass ?? string.Empty}"));
            request.Headers.Authorization = new AuthenticationHeaderValue("Basic", token);
        }

        using var response = await HttpClient.SendAsync(request, cancellationToken);
        return response.IsSuccessStatusCode;
    }

    public static string QuoteString(string value)
    {
        if (string.IsNullOrEmpty(value))
        {
            return "''";
        }

        var escaped = value.Replace("\\", "\\\\").Replace("'", "\\'");
        return $"'{escaped}'";
    }
}
