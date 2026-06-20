using Microsoft.AspNetCore.Http;

namespace Cnn.Agent.Cache;

public static class HostPathCacheKeyProvider
{
    public static string? CreateStorageKey(HttpContext? context)
    {
        if (context == null)
        {
            return null;
        }

        var request = context.Request;
        var host = request.Host.Host;
        if (string.IsNullOrWhiteSpace(host))
        {
            return null;
        }

        var path = request.Path.Value?.TrimStart('/') ?? string.Empty;
        return string.IsNullOrWhiteSpace(path) ? host : $"{host}/{path}";
    }
}
