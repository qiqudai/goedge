using System.Security.Cryptography;
using System.Text;

namespace Cnn.Api.Services.Auth;

public static class PasswordHasher
{
    public static bool PasswordLooksHashed(string? value)
    {
        value = value?.Trim();
        if (string.IsNullOrWhiteSpace(value) || value.Length != 64)
        {
            return false;
        }

        foreach (var ch in value)
        {
            if (ch is >= '0' and <= '9')
            {
                continue;
            }
            if (ch is >= 'a' and <= 'f')
            {
                continue;
            }
            if (ch is >= 'A' and <= 'F')
            {
                continue;
            }
            return false;
        }

        return true;
    }

    public static string HashPasswordForStorage(string input)
    {
        var normalized = NormalizePasswordInput(input);
        try
        {
            return BCrypt.Net.BCrypt.HashPassword(normalized);
        }
        catch
        {
            return string.Empty;
        }
    }

    public static (bool Ok, bool Upgrade) VerifyPassword(string? stored, string provided, bool providedHashed)
    {
        stored = stored?.Trim();
        provided = provided.Trim();
        if (string.IsNullOrWhiteSpace(stored) || string.IsNullOrWhiteSpace(provided))
        {
            return (false, false);
        }

        if (IsBcryptHash(stored))
        {
            if (providedHashed)
            {
                return (BCrypt.Net.BCrypt.Verify(provided.ToLowerInvariant(), stored), false);
            }

            var normalized = NormalizePasswordInput(provided);
            if (BCrypt.Net.BCrypt.Verify(normalized, stored))
            {
                return (true, false);
            }
            if (BCrypt.Net.BCrypt.Verify(provided, stored))
            {
                return (true, true);
            }
            return (false, false);
        }

        if (providedHashed)
        {
            return (NormalizePasswordInput(stored) == provided.ToLowerInvariant(), true);
        }

        return (string.Equals(stored, provided, StringComparison.Ordinal), string.Equals(stored, provided, StringComparison.Ordinal));
    }

    private static string NormalizePasswordInput(string input)
    {
        var trimmed = input.Trim();
        if (PasswordLooksHashed(trimmed))
        {
            return trimmed.ToLowerInvariant();
        }

        using var sha = SHA256.Create();
        var bytes = sha.ComputeHash(Encoding.UTF8.GetBytes(trimmed));
        var builder = new StringBuilder(bytes.Length * 2);
        foreach (var b in bytes)
        {
            builder.Append(b.ToString("x2"));
        }
        return builder.ToString();
    }

    private static bool IsBcryptHash(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return false;
        }

        return value.StartsWith("$2a$", StringComparison.Ordinal) ||
               value.StartsWith("$2b$", StringComparison.Ordinal) ||
               value.StartsWith("$2y$", StringComparison.Ordinal);
    }
}
