namespace Cnn.Agent.Diagnostics;

public static class DebugLogSanitizer
{
    private static readonly HashSet<string> SensitiveKeys = new(StringComparer.OrdinalIgnoreCase)
    {
        "authorization",
        "cookie",
        "token",
        "password",
        "secret",
        "access_token",
        "refresh_token"
    };

    public static IReadOnlyDictionary<string, object?> Sanitize(IReadOnlyDictionary<string, object?> fields)
    {
        if (fields == null || fields.Count == 0)
        {
            return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        }

        var result = new Dictionary<string, object?>(fields.Count, StringComparer.OrdinalIgnoreCase);
        foreach (var (key, value) in fields)
        {
            if (IsSensitive(key))
            {
                result[key] = "***";
                continue;
            }

            result[key] = value;
        }

        return result;
    }

    public static string MaskToken(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return string.Empty;
        }

        return "***";
    }

    private static bool IsSensitive(string? key)
    {
        if (string.IsNullOrWhiteSpace(key))
        {
            return false;
        }

        var normalized = key.Trim();
        if (SensitiveKeys.Contains(normalized))
        {
            return true;
        }

        return normalized.Contains("token", StringComparison.OrdinalIgnoreCase)
            || normalized.Contains("password", StringComparison.OrdinalIgnoreCase)
            || normalized.Contains("secret", StringComparison.OrdinalIgnoreCase)
            || normalized.Contains("authorization", StringComparison.OrdinalIgnoreCase)
            || normalized.Contains("cookie", StringComparison.OrdinalIgnoreCase);
    }
}
