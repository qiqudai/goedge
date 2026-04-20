using System.Net;
using System.Text.Json;

namespace Cnn.Api.Services.Common;

public interface ISpiderIpAllowlistService
{
    bool IsSpiderIp(string? raw);
}

public sealed class SpiderIpAllowlistService : ISpiderIpAllowlistService
{
    private const string AllowlistFileName = "spider_ip_allowlist.json";

    private readonly object _syncRoot = new();
    private SpiderAllowlist _allowlist = SpiderAllowlist.Empty;
    private DateTime _mtime = DateTime.MinValue;
    private string _path = string.Empty;

    public bool IsSpiderIp(string? raw)
    {
        var ip = NormalizeIPv4(raw);
        if (string.IsNullOrWhiteSpace(ip))
        {
            return false;
        }

        var allowlist = LoadAllowlist();
        return allowlist.Match(ip);
    }

    private SpiderAllowlist LoadAllowlist()
    {
        var path = ResolveAllowlistPath();
        try
        {
            var info = new FileInfo(path);
            if (!info.Exists)
            {
                return SpiderAllowlist.Empty;
            }

            lock (_syncRoot)
            {
                if (string.Equals(_path, path, StringComparison.OrdinalIgnoreCase) && info.LastWriteTimeUtc == _mtime)
                {
                    return _allowlist;
                }

                var json = File.ReadAllText(path);
                _allowlist = ParseAllowlist(json);
                _mtime = info.LastWriteTimeUtc;
                _path = path;
                return _allowlist;
            }
        }
        catch
        {
            return SpiderAllowlist.Empty;
        }
    }

    private static string ResolveAllowlistPath()
    {
        var baseDir = AppContext.BaseDirectory;
        if (string.IsNullOrWhiteSpace(baseDir))
        {
            var processPath = Environment.ProcessPath;
            if (!string.IsNullOrWhiteSpace(processPath))
            {
                baseDir = Path.GetDirectoryName(processPath) ?? string.Empty;
            }
        }

        return Path.Combine(baseDir, AllowlistFileName);
    }

    private static SpiderAllowlist ParseAllowlist(string? json)
    {
        if (string.IsNullOrWhiteSpace(json))
        {
            return SpiderAllowlist.Empty;
        }

        try
        {
            var map = JsonSerializer.Deserialize<Dictionary<string, List<string>>>(json);
            if (map == null || map.Count == 0)
            {
                return SpiderAllowlist.Empty;
            }

            var exact = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
            var prefixes = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
            var cidrs = new List<IpCidr>();

            foreach (var entries in map.Values)
            {
                if (entries == null)
                {
                    continue;
                }

                foreach (var entry in entries)
                {
                    var token = entry?.Trim();
                    if (string.IsNullOrWhiteSpace(token))
                    {
                        continue;
                    }

                    if (token.Contains('/'))
                    {
                        if (IpCidr.TryParse(token, out var cidr))
                        {
                            cidrs.Add(cidr);
                            continue;
                        }
                    }

                    var parts = token.Split('.', StringSplitOptions.RemoveEmptyEntries);
                    if (parts.Length == 4)
                    {
                        var ip = NormalizeIPv4(token);
                        if (!string.IsNullOrWhiteSpace(ip))
                        {
                            exact.Add(ip);
                        }
                        continue;
                    }

                    if (parts.Length == 3 && TryNormalizePrefix(parts, out var prefix))
                    {
                        prefixes.Add(prefix);
                    }
                }
            }

            return new SpiderAllowlist(exact, prefixes.ToList(), cidrs);
        }
        catch
        {
            return SpiderAllowlist.Empty;
        }
    }

    private static string NormalizeIPv4(string? raw)
    {
        var value = raw?.Trim();
        if (string.IsNullOrWhiteSpace(value))
        {
            return string.Empty;
        }

        if (value.Contains(':'))
        {
            var parts = value.Split(':', StringSplitOptions.RemoveEmptyEntries);
            if (parts.Length == 2)
            {
                value = parts[0];
            }
            else
            {
                return string.Empty;
            }
        }

        if (!IPAddress.TryParse(value, out var ip))
        {
            return string.Empty;
        }

        var v4 = ip.MapToIPv4();
        return v4.ToString();
    }

    private static bool TryNormalizePrefix(string[] parts, out string prefix)
    {
        prefix = string.Empty;
        if (parts.Length != 3)
        {
            return false;
        }

        for (var i = 0; i < parts.Length; i++)
        {
            if (!int.TryParse(parts[i], out var value) || value < 0 || value > 255)
            {
                return false;
            }
            parts[i] = value.ToString();
        }

        prefix = string.Join(".", parts) + ".";
        return true;
    }

    private sealed record SpiderAllowlist(HashSet<string> Exact, List<string> Prefixes, List<IpCidr> Cidrs)
    {
        public static readonly SpiderAllowlist Empty = new(new HashSet<string>(), new List<string>(), new List<IpCidr>());

        public bool Match(string ip)
        {
            if (string.IsNullOrWhiteSpace(ip))
            {
                return false;
            }

            if (Exact.Contains(ip))
            {
                return true;
            }

            if (Cidrs.Count > 0 && IPAddress.TryParse(ip, out var parsed))
            {
                foreach (var cidr in Cidrs)
                {
                    if (cidr.Contains(parsed))
                    {
                        return true;
                    }
                }
            }

            foreach (var prefix in Prefixes)
            {
                if (ip.StartsWith(prefix, StringComparison.OrdinalIgnoreCase))
                {
                    return true;
                }
            }

            return false;
        }
    }

    private sealed class IpCidr
    {
        private readonly uint _mask;
        private readonly uint _network;

        private IpCidr(uint network, int prefixLength)
        {
            _network = network;
            _mask = prefixLength == 0 ? 0u : uint.MaxValue << (32 - prefixLength);
        }

        public bool Contains(IPAddress ip)
        {
            if (ip.AddressFamily != System.Net.Sockets.AddressFamily.InterNetwork)
            {
                return false;
            }

            var bytes = ip.GetAddressBytes();
            var value = ((uint)bytes[0] << 24) | ((uint)bytes[1] << 16) | ((uint)bytes[2] << 8) | bytes[3];
            return (value & _mask) == (_network & _mask);
        }

        public static bool TryParse(string raw, out IpCidr cidr)
        {
            cidr = null!;
            if (string.IsNullOrWhiteSpace(raw))
            {
                return false;
            }

            var parts = raw.Split('/', StringSplitOptions.RemoveEmptyEntries);
            if (parts.Length != 2)
            {
                return false;
            }

            if (!IPAddress.TryParse(parts[0].Trim(), out var ip))
            {
                return false;
            }

            if (ip.AddressFamily != System.Net.Sockets.AddressFamily.InterNetwork)
            {
                return false;
            }

            if (!int.TryParse(parts[1].Trim(), out var prefixLength) || prefixLength < 0 || prefixLength > 32)
            {
                return false;
            }

            var bytes = ip.MapToIPv4().GetAddressBytes();
            var value = ((uint)bytes[0] << 24) | ((uint)bytes[1] << 16) | ((uint)bytes[2] << 8) | bytes[3];
            cidr = new IpCidr(value, prefixLength);
            return true;
        }
    }
}
