using Microsoft.AspNetCore.Http;
using System.Linq;

namespace Cnn.Common.Localization;

public static class LanguageResolver
{
    public static string Resolve(HttpContext context, string defaultLanguage)
    {
        var fromQuery = TryNormalize(context.Request.Query["lang"].FirstOrDefault());
        if (!string.IsNullOrWhiteSpace(fromQuery))
        {
            return fromQuery!;
        }

        var fromHeader = TryNormalize(context.Request.Headers["Accept-Language"].ToString());
        if (!string.IsNullOrWhiteSpace(fromHeader))
        {
            return fromHeader!;
        }

        return defaultLanguage;
    }

    public static string NormalizeOrDefault(string? value, string defaultLanguage)
    {
        return TryNormalize(value) ?? defaultLanguage;
    }

    private static string? TryNormalize(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return null;
        }

        var token = value.Split(',')[0].Split(';')[0].Trim();
        if (token.Length == 0)
        {
            return null;
        }

        if (token.StartsWith("zh", StringComparison.OrdinalIgnoreCase))
        {
            return "zh-CN";
        }

        if (token.StartsWith("en", StringComparison.OrdinalIgnoreCase))
        {
            return "en-US";
        }

        return null;
    }
}
