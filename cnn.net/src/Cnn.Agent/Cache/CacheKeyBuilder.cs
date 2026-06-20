using System.Security.Cryptography;
using System.Text;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.Primitives;

namespace Cnn.Agent.Cache;

public static class CacheKeyBuilder
{
    private const string RootPathToken = "_root";

    public static string? BuildRelativeKey(HttpContext context, CacheDecision decision)
    {
        var host = context.Request.Host.Host;
        if (string.IsNullOrWhiteSpace(host))
        {
            return null;
        }

        var path = context.Request.Path.Value?.TrimStart('/') ?? string.Empty;
        if (string.IsNullOrWhiteSpace(path))
        {
            path = RootPathToken;
        }

        var baseKey = string.IsNullOrWhiteSpace(path) ? host : $"{host}/{path}";
        if (decision.IgnoreQuery)
        {
            return baseKey;
        }

        var normalizedQuery = NormalizeQuery(context.Request.Query, decision.QueryIgnoreList);
        if (string.IsNullOrWhiteSpace(normalizedQuery))
        {
            return baseKey;
        }

        var hash = ComputeHash(normalizedQuery);
        return $"{baseKey}__q={hash}";
    }

    public static StringValues BuildQueryKeys(HttpContext context, CacheDecision decision)
    {
        if (decision.IgnoreQuery)
        {
            return StringValues.Empty;
        }

        if (context.Request.Query.Count == 0)
        {
            return StringValues.Empty;
        }

        var keys = new List<string>();
        foreach (var kvp in context.Request.Query)
        {
            if (IsIgnored(kvp.Key, decision.QueryIgnoreList))
            {
                continue;
            }

            keys.Add(kvp.Key);
        }

        if (keys.Count == 0)
        {
            return StringValues.Empty;
        }

        return new StringValues(keys.ToArray());
    }

    private static string? NormalizeQuery(IQueryCollection query, IReadOnlyList<string> ignoreList)
    {
        if (query.Count == 0)
        {
            return null;
        }

        var pairs = new List<(string Key, string Value)>();
        foreach (var kvp in query)
        {
            if (IsIgnored(kvp.Key, ignoreList))
            {
                continue;
            }

            foreach (var value in kvp.Value)
            {
                pairs.Add((kvp.Key, value ?? string.Empty));
            }
        }

        if (pairs.Count == 0)
        {
            return null;
        }

        pairs.Sort((a, b) =>
        {
            var keyCompare = string.Compare(a.Key, b.Key, StringComparison.OrdinalIgnoreCase);
            if (keyCompare != 0)
            {
                return keyCompare;
            }

            return string.Compare(a.Value, b.Value, StringComparison.OrdinalIgnoreCase);
        });

        var builder = new StringBuilder();
        for (var i = 0; i < pairs.Count; i++)
        {
            if (i > 0)
            {
                builder.Append('&');
            }

            builder.Append(Uri.EscapeDataString(pairs[i].Key));
            builder.Append('=');
            builder.Append(Uri.EscapeDataString(pairs[i].Value));
        }

        return builder.ToString();
    }

    private static bool IsIgnored(string key, IReadOnlyList<string> ignoreList)
    {
        if (ignoreList == null || ignoreList.Count == 0)
        {
            return false;
        }

        foreach (var rule in ignoreList)
        {
            if (string.IsNullOrWhiteSpace(rule))
            {
                continue;
            }

            if (rule.EndsWith("*", StringComparison.Ordinal))
            {
                var prefix = rule.Substring(0, rule.Length - 1);
                if (key.StartsWith(prefix, StringComparison.OrdinalIgnoreCase))
                {
                    return true;
                }
                continue;
            }

            if (string.Equals(key, rule, StringComparison.OrdinalIgnoreCase))
            {
                return true;
            }
        }

        return false;
    }

    private static string ComputeHash(string input)
    {
        var bytes = Encoding.UTF8.GetBytes(input);
        var hash = SHA256.HashData(bytes);
        return Convert.ToHexString(hash).ToLowerInvariant();
    }
}
