using System.Net;
using System.Net.Sockets;

namespace Cnn.Agent.Security;

public static class IpRuleMatcher
{
    public static bool IsMatch(IPAddress? address, IEnumerable<string>? rules)
    {
        if (address == null || rules == null)
        {
            return false;
        }

        foreach (var rule in rules)
        {
            if (IsMatch(address, rule))
            {
                return true;
            }
        }

        return false;
    }

    public static bool IsMatch(IPAddress? address, string? rule)
    {
        if (address == null || string.IsNullOrWhiteSpace(rule))
        {
            return false;
        }

        var value = rule.Trim();
        if (value.Contains('/', StringComparison.Ordinal))
        {
            return IsCidrMatch(address, value);
        }

        if (value.Contains('*'))
        {
            return IsWildcardMatch(address, value);
        }

        if (!IPAddress.TryParse(value, out var target))
        {
            return false;
        }

        target = Normalize(target);
        address = Normalize(address);
        return address.Equals(target);
    }

    private static bool IsCidrMatch(IPAddress address, string cidr)
    {
        var parts = cidr.Split('/', 2, StringSplitOptions.TrimEntries | StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length != 2)
        {
            return false;
        }

        if (!IPAddress.TryParse(parts[0], out var network))
        {
            return false;
        }

        if (!int.TryParse(parts[1], out var prefixLength))
        {
            return false;
        }

        network = Normalize(network);
        address = Normalize(address);

        if (network.AddressFamily != address.AddressFamily)
        {
            return false;
        }

        var totalBits = network.AddressFamily == AddressFamily.InterNetwork ? 32 : 128;
        if (prefixLength < 0 || prefixLength > totalBits)
        {
            return false;
        }

        var addressBytes = address.GetAddressBytes();
        var networkBytes = network.GetAddressBytes();

        var fullBytes = prefixLength / 8;
        var remainBits = prefixLength % 8;

        for (var i = 0; i < fullBytes; i++)
        {
            if (addressBytes[i] != networkBytes[i])
            {
                return false;
            }
        }

        if (remainBits == 0)
        {
            return true;
        }

        var mask = (byte)~(0xFF >> remainBits);
        return (addressBytes[fullBytes] & mask) == (networkBytes[fullBytes] & mask);
    }

    private static bool IsWildcardMatch(IPAddress address, string pattern)
    {
        address = Normalize(address);
        if (address.AddressFamily != AddressFamily.InterNetwork)
        {
            return false;
        }

        var addressParts = address.ToString().Split('.');
        var patternParts = pattern.Split('.');
        if (patternParts.Length != 4 || addressParts.Length != 4)
        {
            return false;
        }

        for (var i = 0; i < 4; i++)
        {
            var rule = patternParts[i].Trim();
            if (rule == "*")
            {
                continue;
            }

            if (!string.Equals(rule, addressParts[i], StringComparison.Ordinal))
            {
                return false;
            }
        }

        return true;
    }

    private static IPAddress Normalize(IPAddress address)
    {
        if (address.IsIPv4MappedToIPv6)
        {
            return address.MapToIPv4();
        }

        return address;
    }
}
