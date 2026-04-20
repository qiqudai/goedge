using System.Globalization;
using System.Net.Http;
using Yarp.ReverseProxy.Forwarder;

namespace Cnn.Agent.Proxy;

public class EdgeForwarderHttpClientFactory : ForwarderHttpClientFactory
{
    protected override void ConfigureHandler(ForwarderHttpClientContext context, SocketsHttpHandler handler)
    {
        base.ConfigureHandler(context, handler);

        var metadata = context.NewMetadata;
        if (metadata == null || metadata.Count == 0)
        {
            return;
        }

        if (metadata.TryGetValue("proxy_connect_timeout", out var connectTimeoutRaw)
            && TryParseDuration(connectTimeoutRaw, out var connectTimeout)
            && connectTimeout > TimeSpan.Zero)
        {
            handler.ConnectTimeout = connectTimeout;
        }

        if (metadata.TryGetValue("upstream_keepalive_timeout", out var keepaliveTimeoutRaw)
            && TryParseDuration(keepaliveTimeoutRaw, out var keepaliveTimeout)
            && keepaliveTimeout > TimeSpan.Zero)
        {
            handler.PooledConnectionIdleTimeout = keepaliveTimeout;
        }
    }

    private static bool TryParseDuration(string? raw, out TimeSpan duration)
    {
        duration = default;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        var value = raw.Trim().ToLowerInvariant();
        if (value.EndsWith("ms", StringComparison.Ordinal)
            && double.TryParse(value[..^2], NumberStyles.Float, CultureInfo.InvariantCulture, out var ms)
            && ms > 0)
        {
            duration = TimeSpan.FromMilliseconds(ms);
            return true;
        }

        if (value.EndsWith("s", StringComparison.Ordinal)
            && double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var sec)
            && sec > 0)
        {
            duration = TimeSpan.FromSeconds(sec);
            return true;
        }

        if (value.EndsWith("m", StringComparison.Ordinal)
            && double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var minute)
            && minute > 0)
        {
            duration = TimeSpan.FromMinutes(minute);
            return true;
        }

        if (value.EndsWith("h", StringComparison.Ordinal)
            && double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var hour)
            && hour > 0)
        {
            duration = TimeSpan.FromHours(hour);
            return true;
        }

        if (double.TryParse(value, NumberStyles.Float, CultureInfo.InvariantCulture, out var seconds) && seconds > 0)
        {
            duration = TimeSpan.FromSeconds(seconds);
            return true;
        }

        if (TimeSpan.TryParse(value, CultureInfo.InvariantCulture, out var parsed) && parsed > TimeSpan.Zero)
        {
            duration = parsed;
            return true;
        }

        return false;
    }
}
