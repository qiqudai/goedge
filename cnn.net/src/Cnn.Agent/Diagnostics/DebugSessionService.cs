using System.Net;
using System.Security.Cryptography;
using System.Text;

namespace Cnn.Agent.Diagnostics;

public interface IDebugSessionService
{
    bool IsEnabled(string module, HttpContext context);
    bool TryAllowEvent(string module, HttpContext context, out string? sessionId);
    string? GetSessionId(HttpContext context);
    void Update(DebugOptions options, TimeSpan ttl);
}

public sealed class DebugSessionService : IDebugSessionService
{
    private readonly object _lock = new();
    private DebugOptions _options = DebugOptions.Disabled;
    private DateTimeOffset? _expiresAt;
    private long _windowSecond;
    private int _windowCount;

    public bool IsEnabled(string module, HttpContext context)
    {
        var normalizedModule = NormalizeModule(module);
        lock (_lock)
        {
            EnsureNotExpiredLocked();

            if (IsRequestDebugAllowedLocked(context))
            {
                return true;
            }

            if (!_options.Enabled)
            {
                return false;
            }

            return _options.Modules.TryGetValue(normalizedModule, out var enabled) && enabled;
        }
    }

    public bool TryAllowEvent(string module, HttpContext context, out string? sessionId)
    {
        sessionId = null;
        if (!IsEnabled(module, context))
        {
            return false;
        }

        lock (_lock)
        {
            EnsureNotExpiredLocked();

            var sampleRate = Math.Clamp(_options.SampleRate, 0d, 1d);
            if (sampleRate < 1d && Random.Shared.NextDouble() > sampleRate)
            {
                return false;
            }

            var maxEvents = _options.MaxEventsPerSec <= 0 ? 200 : _options.MaxEventsPerSec;
            var nowSec = DateTimeOffset.UtcNow.ToUnixTimeSeconds();
            if (_windowSecond != nowSec)
            {
                _windowSecond = nowSec;
                _windowCount = 0;
            }

            if (_windowCount >= maxEvents)
            {
                return false;
            }

            _windowCount++;
            sessionId = GetSessionIdLocked(context);
            return true;
        }
    }

    public string? GetSessionId(HttpContext context)
    {
        lock (_lock)
        {
            EnsureNotExpiredLocked();
            return GetSessionIdLocked(context);
        }
    }

    public void Update(DebugOptions options, TimeSpan ttl)
    {
        options ??= DebugOptions.Disabled;
        var normalized = new DebugOptions
        {
            Enabled = options.Enabled,
            InternalIpOnly = options.InternalIpOnly,
            Token = string.IsNullOrWhiteSpace(options.Token) ? null : options.Token.Trim(),
            AllowHeaderToken = options.AllowHeaderToken,
            AllowQueryFlag = options.AllowQueryFlag,
            SampleRate = Math.Clamp(options.SampleRate <= 0 ? 0.01d : options.SampleRate, 0d, 1d),
            MaxEventsPerSec = options.MaxEventsPerSec <= 0 ? 200 : options.MaxEventsPerSec,
            Modules = new Dictionary<string, bool>(StringComparer.OrdinalIgnoreCase)
        };

        if (options.Modules != null)
        {
            foreach (var (key, value) in options.Modules)
            {
                var module = NormalizeModule(key);
                if (!string.IsNullOrWhiteSpace(module))
                {
                    normalized.Modules[module] = value;
                }
            }
        }

        lock (_lock)
        {
            _options = normalized;
            if (ttl > TimeSpan.Zero)
            {
                _expiresAt = DateTimeOffset.UtcNow.Add(ttl);
            }
            else
            {
                _expiresAt = null;
            }

            _windowSecond = 0;
            _windowCount = 0;
        }
    }

    private bool IsRequestDebugAllowedLocked(HttpContext context)
    {
        return IsTokenDebugAllowedLocked(context) || IsQueryDebugAllowedLocked(context);
    }

    private bool IsTokenDebugAllowedLocked(HttpContext context)
    {
        if (!_options.AllowHeaderToken)
        {
            return false;
        }

        if (string.IsNullOrWhiteSpace(_options.Token))
        {
            return false;
        }

        if (!context.Request.Headers.TryGetValue("X-Debug-Token", out var tokenValues))
        {
            return false;
        }

        var token = tokenValues.ToString().Trim();
        if (!string.Equals(token, _options.Token, StringComparison.Ordinal))
        {
            return false;
        }

        if (!_options.InternalIpOnly)
        {
            return true;
        }

        return IsInternalIp(context.Connection.RemoteIpAddress);
    }

    private bool IsQueryDebugAllowedLocked(HttpContext context)
    {
        if (!_options.AllowQueryFlag)
        {
            return false;
        }

        if (!context.Request.Query.TryGetValue("__debug", out var values) || values.Count == 0)
        {
            return false;
        }

        var raw = values[0]?.Trim();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        var enabled = raw.Equals("1", StringComparison.OrdinalIgnoreCase)
                      || raw.Equals("true", StringComparison.OrdinalIgnoreCase)
                      || raw.Equals("on", StringComparison.OrdinalIgnoreCase)
                      || raw.Equals("yes", StringComparison.OrdinalIgnoreCase);

        if (!enabled)
        {
            return false;
        }

        if (!_options.InternalIpOnly)
        {
            return true;
        }

        return IsInternalIp(context.Connection.RemoteIpAddress);
    }

    private static bool IsInternalIp(IPAddress? ip)
    {
        if (ip == null)
        {
            return false;
        }

        if (IPAddress.IsLoopback(ip))
        {
            return true;
        }

        if (ip.IsIPv4MappedToIPv6)
        {
            ip = ip.MapToIPv4();
        }

        if (ip.AddressFamily == System.Net.Sockets.AddressFamily.InterNetwork)
        {
            var bytes = ip.GetAddressBytes();
            if (bytes.Length != 4)
            {
                return false;
            }

            if (bytes[0] == 10)
            {
                return true;
            }

            if (bytes[0] == 172 && bytes[1] >= 16 && bytes[1] <= 31)
            {
                return true;
            }

            if (bytes[0] == 192 && bytes[1] == 168)
            {
                return true;
            }

            return false;
        }

        if (ip.AddressFamily == System.Net.Sockets.AddressFamily.InterNetworkV6)
        {
            if (ip.IsIPv6LinkLocal || ip.IsIPv6SiteLocal)
            {
                return true;
            }

            var bytes = ip.GetAddressBytes();
            return bytes.Length > 0 && (bytes[0] & 0xFE) == 0xFC;
        }

        return false;
    }

    private static string NormalizeModule(string? module)
    {
        if (string.IsNullOrWhiteSpace(module))
        {
            return string.Empty;
        }

        return module.Trim().ToLowerInvariant();
    }

    private void EnsureNotExpiredLocked()
    {
        if (!_expiresAt.HasValue || _expiresAt.Value > DateTimeOffset.UtcNow)
        {
            return;
        }

        _options = DebugOptions.Disabled;
        _expiresAt = null;
        _windowSecond = 0;
        _windowCount = 0;
    }

    private string? GetSessionIdLocked(HttpContext context)
    {
        if (context.Request.Headers.TryGetValue("X-Debug-Token", out var tokenValues))
        {
            var token = tokenValues.ToString().Trim();
            if (!string.IsNullOrWhiteSpace(token))
            {
                var hash = SHA256.HashData(Encoding.UTF8.GetBytes(token));
                return "dbg_" + Convert.ToHexString(hash[..6]).ToLowerInvariant();
            }
        }

        if (!string.IsNullOrWhiteSpace(context.TraceIdentifier))
        {
            return context.TraceIdentifier;
        }

        return null;
    }
}
