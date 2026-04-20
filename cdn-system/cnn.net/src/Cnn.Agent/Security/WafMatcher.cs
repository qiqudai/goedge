using System.Net;
using System.Text.RegularExpressions;

namespace Cnn.Agent.Security;

public sealed class WafMatcher
{
    private static readonly Regex[] SqlRegexes =
    [
        new Regex(@"(?:\bunion\b\s+\b(?:all\s+)?select\b|\bselect\b.+\bfrom\b|\binsert\b\s+\binto\b|\bdelete\b\s+\bfrom\b|\bdrop\b\s+\btable\b)",
            RegexOptions.IgnoreCase | RegexOptions.Compiled,
            TimeSpan.FromMilliseconds(50)),
        new Regex(@"(?:'|\""|\`)\s*(?:or|and)\s*(?:'?\d+'?\s*=\s*'?\d+'?|true|false)",
            RegexOptions.IgnoreCase | RegexOptions.Compiled,
            TimeSpan.FromMilliseconds(50))
    ];

    private static readonly Regex[] XssRegexes =
    [
        new Regex(@"<\s*script\b", RegexOptions.IgnoreCase | RegexOptions.Compiled, TimeSpan.FromMilliseconds(50)),
        new Regex(@"\bon\w+\s*=", RegexOptions.IgnoreCase | RegexOptions.Compiled, TimeSpan.FromMilliseconds(50)),
        new Regex(@"javascript\s*:", RegexOptions.IgnoreCase | RegexOptions.Compiled, TimeSpan.FromMilliseconds(50))
    ];

    private static readonly string[] ScannerTokens =
    [
        "sqlmap",
        "nmap",
        "nikto",
        "masscan",
        "acunetix",
        "nessus",
        "dirbuster",
        "zgrab",
        "w3af"
    ];

    private static readonly string[] ScannerPathTokens =
    [
        "/.git",
        "/.env",
        "/phpmyadmin",
        "/wp-admin",
        "/vendor/phpunit",
        "/actuator",
        "/boaform",
        "/cgi-bin"
    ];

    public bool TryEvaluate(HttpContext context, WafCompiledConfig config, out SecurityDecision decision)
    {
        decision = SecurityDecision.Allow();
        if (!config.Enabled)
        {
            return false;
        }

        var clientIp = context.Connection.RemoteIpAddress;
        var ua = context.Request.Headers.UserAgent.ToString().Trim();
        var pathAndQuery = (context.Request.Path + context.Request.QueryString).ToString();

        if (clientIp != null && IpRuleMatcher.IsMatch(clientIp, config.WhiteIps))
        {
            return false;
        }

        if (clientIp != null && IpRuleMatcher.IsMatch(clientIp, config.BlackIps))
        {
            decision = SecurityDecision.Block(StatusCodes.Status403Forbidden, "waf", "black_ip", "waf_black_ip");
            return true;
        }

        if (IsRegionBlocked(context, config.RegionBlockCountries, out var country))
        {
            decision = SecurityDecision.Block(
                StatusCodes.Status403Forbidden,
                "waf",
                "region_block",
                $"waf_region_block:{country}");
            return true;
        }

        if (config.BlockEmptyUa && string.IsNullOrWhiteSpace(ua))
        {
            decision = SecurityDecision.Block(StatusCodes.Status403Forbidden, "waf", "empty_ua", "waf_empty_ua");
            return true;
        }

        if (ContainsAny(ua, config.WhiteUaKeywords))
        {
            return false;
        }

        if (ContainsAny(ua, config.BlackUaKeywords))
        {
            decision = SecurityDecision.Block(StatusCodes.Status403Forbidden, "waf", "black_ua", "waf_black_ua");
            return true;
        }

        var whiteUrlMatched = ContainsAny(pathAndQuery, config.WhiteUrlKeywords);
        if (!whiteUrlMatched && ContainsAny(pathAndQuery, config.BlackUrlKeywords))
        {
            decision = SecurityDecision.Block(StatusCodes.Status403Forbidden, "waf", "black_url", "waf_black_url");
            return true;
        }

        if (whiteUrlMatched)
        {
            return false;
        }

        var inspect = BuildInspectText(pathAndQuery, ua, context.Request.Headers.Referer.ToString());

        if (config.SqlInjectionEnabled && IsMatchAny(inspect, SqlRegexes))
        {
            decision = SecurityDecision.Block(StatusCodes.Status403Forbidden, "waf", "sql_injection", "waf_sql_injection");
            return true;
        }

        if (config.XssEnabled && IsMatchAny(inspect, XssRegexes))
        {
            decision = SecurityDecision.Block(StatusCodes.Status403Forbidden, "waf", "xss", "waf_xss");
            return true;
        }

        if (config.ScannerEnabled && (ContainsAny(ua, ScannerTokens) || ContainsAny(pathAndQuery, ScannerPathTokens)))
        {
            decision = SecurityDecision.Block(StatusCodes.Status403Forbidden, "waf", "scanner", "waf_scanner");
            return true;
        }

        return false;
    }

    private static bool IsMatchAny(string text, IReadOnlyList<Regex> regexes)
    {
        if (string.IsNullOrWhiteSpace(text))
        {
            return false;
        }

        foreach (var regex in regexes)
        {
            try
            {
                if (regex.IsMatch(text))
                {
                    return true;
                }
            }
            catch
            {
                // ignore invalid/timeout matches
            }
        }

        return false;
    }

    private static bool ContainsAny(string? source, IReadOnlyList<string> keywords)
    {
        if (string.IsNullOrWhiteSpace(source) || keywords == null || keywords.Count == 0)
        {
            return false;
        }

        foreach (var raw in keywords)
        {
            var keyword = raw?.Trim();
            if (string.IsNullOrWhiteSpace(keyword))
            {
                continue;
            }

            if (source.Contains(keyword, StringComparison.OrdinalIgnoreCase))
            {
                return true;
            }
        }

        return false;
    }

    private static bool IsRegionBlocked(HttpContext context, IReadOnlyList<string> regionBlockCountries, out string country)
    {
        country = string.Empty;
        if (regionBlockCountries == null || regionBlockCountries.Count == 0)
        {
            return false;
        }

        country = ResolveCountryCode(context);
        if (string.IsNullOrWhiteSpace(country))
        {
            return false;
        }

        foreach (var raw in regionBlockCountries)
        {
            var value = raw?.Trim();
            if (string.IsNullOrWhiteSpace(value))
            {
                continue;
            }

            if (string.Equals(value, country, StringComparison.OrdinalIgnoreCase))
            {
                return true;
            }
        }

        return false;
    }

    private static string ResolveCountryCode(HttpContext context)
    {
        var headers = context.Request.Headers;

        if (headers.TryGetValue("CF-IPCountry", out var cfCountry))
        {
            var value = cfCountry.ToString().Trim();
            if (value.Length > 0)
            {
                return value.ToUpperInvariant();
            }
        }

        if (headers.TryGetValue("X-Country-Code", out var xCountry))
        {
            var value = xCountry.ToString().Trim();
            if (value.Length > 0)
            {
                return value.ToUpperInvariant();
            }
        }

        if (headers.TryGetValue("X-Geo-Country", out var geoCountry))
        {
            var value = geoCountry.ToString().Trim();
            if (value.Length > 0)
            {
                return value.ToUpperInvariant();
            }
        }

        return string.Empty;
    }

    private static string BuildInspectText(string pathAndQuery, string ua, string referer)
    {
        var value = $"{pathAndQuery}\n{ua}\n{referer}";
        try
        {
            return Uri.UnescapeDataString(value);
        }
        catch
        {
            return value;
        }
    }
}
