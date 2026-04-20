using System.Net;

namespace Cnn.Api.Services.Common.Dns;

internal static class DnsHttp
{
    public static async Task<(HttpStatusCode StatusCode, string Body)> SendAsync(HttpRequestMessage request, TimeSpan timeout)
    {
        using var client = new HttpClient
        {
            Timeout = timeout
        };

        using var response = await client.SendAsync(request);
        var body = await response.Content.ReadAsStringAsync();
        return (response.StatusCode, body);
    }
}
