using System.Net;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Stream;

public sealed class StreamRouteCompiler
{
    public (IReadOnlyList<StreamListenerPlan> Plans, IReadOnlyList<string> Errors) Compile(EdgeConfigDto? config)
    {
        if (config?.Streams == null || config.Streams.Count == 0)
        {
            return (Array.Empty<StreamListenerPlan>(), Array.Empty<string>());
        }

        var plans = new List<StreamListenerPlan>();
        var errors = new List<string>();
        var keys = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        foreach (var stream in config.Streams)
        {
            if (stream == null)
            {
                continue;
            }

            var targets = ResolveTargets(stream.Targets);
            if (targets.Count == 0)
            {
                errors.Add($"stream {stream.Id} has no enabled targets");
                continue;
            }

            if (stream.ListenPorts == null || stream.ListenPorts.Count == 0)
            {
                errors.Add($"stream {stream.Id} has empty listen_ports");
                continue;
            }

            foreach (var rawListen in stream.ListenPorts)
            {
                if (!TryParseListen(rawListen, out var ip, out var port, out var parseError))
                {
                    errors.Add($"stream {stream.Id} invalid listen '{rawListen}': {parseError}");
                    continue;
                }

                var key = $"tcp:{ip}:{port}";
                if (!keys.Add(key))
                {
                    errors.Add($"duplicate stream listen {key}");
                    continue;
                }

                var plan = new StreamListenerPlan
                {
                    StreamId = stream.Id,
                    Key = key,
                    ListenIp = ip,
                    ListenPort = port,
                    BalanceWay = NormalizeBalance(stream.BalanceWay),
                    MaxConns = stream.ConnLimit.GetValueOrDefault() > 0 ? stream.ConnLimit!.Value : 20_000,
                    ConnectTimeout = ParseDuration(stream.ProxyConnectTimeout, TimeSpan.FromSeconds(3)),
                    IdleTimeout = ParseDuration(stream.ProxyTimeout, TimeSpan.FromSeconds(60)),
                    Targets = targets
                };

                plans.Add(plan);
            }
        }

        return (plans, errors);
    }

    private static IReadOnlyList<EdgeStreamTargetDto> ResolveTargets(IReadOnlyList<EdgeStreamTargetDto>? targets)
    {
        if (targets == null || targets.Count == 0)
        {
            return Array.Empty<EdgeStreamTargetDto>();
        }

        var hasExplicitEnabled = targets.Any(t => t.Enable);
        var list = new List<EdgeStreamTargetDto>();
        foreach (var target in targets)
        {
            var addr = target?.Addr?.Trim();
            if (string.IsNullOrWhiteSpace(addr))
            {
                continue;
            }

            if (hasExplicitEnabled && !target!.Enable)
            {
                continue;
            }

            if (!TryParseTarget(addr, out _, out _, out _))
            {
                continue;
            }

            list.Add(target!);
        }

        return list;
    }

    public static bool TryParseTarget(string raw, out string host, out int port, out string? error)
    {
        host = string.Empty;
        port = 0;
        error = null;

        var value = raw?.Trim() ?? string.Empty;
        if (value.Length == 0)
        {
            error = "empty";
            return false;
        }

        if (Uri.TryCreate(value.Contains("://", StringComparison.Ordinal) ? value : $"tcp://{value}", UriKind.Absolute, out var uri))
        {
            host = uri.Host;
            port = uri.Port;
        }
        else
        {
            var parts = value.Split(':', 2, StringSplitOptions.TrimEntries | StringSplitOptions.RemoveEmptyEntries);
            if (parts.Length != 2 || !int.TryParse(parts[1], out port))
            {
                error = "invalid host:port";
                return false;
            }

            host = parts[0];
        }

        if (string.IsNullOrWhiteSpace(host))
        {
            error = "empty host";
            return false;
        }

        if (port <= 0 || port > 65535)
        {
            error = "invalid port";
            return false;
        }

        return true;
    }

    private static bool TryParseListen(string? raw, out IPAddress ip, out int port, out string? error)
    {
        ip = IPAddress.Any;
        port = 0;
        error = null;

        var value = raw?.Trim() ?? string.Empty;
        if (value.Length == 0)
        {
            error = "empty";
            return false;
        }

        if (int.TryParse(value, out var onlyPort))
        {
            if (onlyPort <= 0 || onlyPort > 65535)
            {
                error = "port out of range";
                return false;
            }

            ip = IPAddress.Any;
            port = onlyPort;
            return true;
        }

        var parts = value.Split(':', 2, StringSplitOptions.TrimEntries | StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length != 2)
        {
            error = "expected ip:port";
            return false;
        }

        if (!IPAddress.TryParse(parts[0], out ip!))
        {
            if (string.Equals(parts[0], "localhost", StringComparison.OrdinalIgnoreCase))
            {
                ip = IPAddress.Loopback;
            }
            else
            {
                error = "invalid ip";
                return false;
            }
        }

        if (!int.TryParse(parts[1], out port) || port <= 0 || port > 65535)
        {
            error = "invalid port";
            return false;
        }

        return true;
    }

    private static string NormalizeBalance(string? raw)
    {
        return raw?.Trim().ToLowerInvariant() switch
        {
            "round_robin" => "round_robin",
            "rr" => "round_robin",
            _ => "round_robin"
        };
    }

    private static TimeSpan ParseDuration(string? raw, TimeSpan fallback)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return fallback;
        }

        var value = raw.Trim().ToLowerInvariant();
        if (value.EndsWith("ms", StringComparison.Ordinal) && int.TryParse(value[..^2], out var ms) && ms > 0)
        {
            return TimeSpan.FromMilliseconds(ms);
        }

        if (value.EndsWith("s", StringComparison.Ordinal) && int.TryParse(value[..^1], out var sec) && sec > 0)
        {
            return TimeSpan.FromSeconds(sec);
        }

        if (value.EndsWith("m", StringComparison.Ordinal) && int.TryParse(value[..^1], out var minute) && minute > 0)
        {
            return TimeSpan.FromMinutes(minute);
        }

        if (int.TryParse(value, out var asSeconds) && asSeconds > 0)
        {
            return TimeSpan.FromSeconds(asSeconds);
        }

        return fallback;
    }
}
