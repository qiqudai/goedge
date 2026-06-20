using System.Globalization;

namespace Cnn.Api.Services.Common;

public static class DomainHelper
{
    public static string NormalizeDomainInput(string? input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return string.Empty;
        }

        var value = input.Trim().ToLowerInvariant();
        value = value.Replace("http://", string.Empty).Replace("https://", string.Empty);

        var slashIndex = value.IndexOfAny(new[] { '/', '?', '#' });
        if (slashIndex >= 0)
        {
            value = value[..slashIndex];
        }

        var colonIndex = value.IndexOf(':');
        if (colonIndex >= 0)
        {
            value = value[..colonIndex];
        }

        while (value.EndsWith(".", StringComparison.Ordinal))
        {
            value = value[..^1];
        }

        return value;
    }

    public static bool IsValidDomain(string? domain)
    {
        if (string.IsNullOrWhiteSpace(domain))
        {
            return false;
        }

        if (domain.StartsWith("*.", StringComparison.Ordinal))
        {
            domain = domain[2..];
        }

        if (domain.Length > 253)
        {
            return false;
        }

        var parts = domain.Split('.', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length < 2)
        {
            return false;
        }

        foreach (var part in parts)
        {
            if (part.Length < 1 || part.Length > 63)
            {
                return false;
            }

            if (part.StartsWith("-", StringComparison.Ordinal) || part.EndsWith("-", StringComparison.Ordinal))
            {
                return false;
            }

            foreach (var c in part)
            {
                if (char.IsLetterOrDigit(c))
                {
                    continue;
                }

                if (c != '-')
                {
                    return false;
                }
            }
        }

        return true;
    }

    public static string GenerateToken(int length)
    {
        const string chars = "abcdefghijklmnopqrstuvwxyz0123456789";
        var buffer = new char[length];
        var rng = Random.Shared;
        for (var i = 0; i < length; i++)
        {
            buffer[i] = chars[rng.Next(chars.Length)];
        }

        return new string(buffer);
    }
}
