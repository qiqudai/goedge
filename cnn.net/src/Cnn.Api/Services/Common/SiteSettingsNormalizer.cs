using System.Text.Json;

namespace Cnn.Api.Services.Common;

public static class SiteSettingsNormalizer
{
    public static Dictionary<string, object?> Normalize(Dictionary<string, object?> settings)
    {
        if (settings == null)
        {
            return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        }

        if (TryGetMap(settings, "cache", out var cacheCfg) && cacheCfg != null)
        {
            if (cacheCfg.TryGetValue("rules", out var raw))
            {
                var normalized = NormalizeCacheRulesRaw(raw);
                if (normalized == null || normalized.Count == 0)
                {
                    cacheCfg.Remove("rules");
                }
                else
                {
                    cacheCfg["rules"] = normalized;
                }
            }
        }

        if (TryGetMap(settings, "advanced", out var adv) && adv != null)
        {
            if (adv.TryGetValue("url_redirects", out var redirectsRaw))
            {
                var redirects = NormalizeUrlRedirectsRaw(redirectsRaw);
                if (redirects == null || redirects.Count == 0)
                {
                    adv.Remove("url_redirects");
                }
                else
                {
                    adv["url_redirects"] = redirects;
                }
            }

            if (adv.TryGetValue("req_headers", out var reqHeadersRaw))
            {
                var headers = NormalizeHeaderRulesRaw(reqHeadersRaw);
                if (headers == null || headers.Count == 0)
                {
                    adv.Remove("req_headers");
                }
                else
                {
                    adv["req_headers"] = headers;
                }
            }

            if (adv.TryGetValue("res_headers", out var resHeadersRaw))
            {
                var headers = NormalizeHeaderRulesRaw(resHeadersRaw);
                if (headers == null || headers.Count == 0)
                {
                    adv.Remove("res_headers");
                }
                else
                {
                    adv["res_headers"] = headers;
                }
            }
        }

        return settings;
    }

    private static List<Dictionary<string, object?>>? NormalizeCacheRulesRaw(object? raw)
    {
        var items = NormalizeMapSlice(raw);
        if (items == null || items.Count == 0)
        {
            return null;
        }

        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var output = new List<Dictionary<string, object?>>(items.Count);
        for (var i = items.Count - 1; i >= 0; i--)
        {
            var item = items[i];
            var normalized = NormalizeCacheRuleItem(item, seen);
            if (normalized != null)
            {
                output.Add(normalized);
            }
        }

        output.Reverse();
        return output.Count == 0 ? null : output;
    }

    private static Dictionary<string, object?>? NormalizeCacheRuleItem(Dictionary<string, object?> item, HashSet<string> seen)
    {
        if (item == null)
        {
            return null;
        }

        var ruleExpr = ParseString(item.TryGetValue("rule", out var ruleRaw) ? ruleRaw : null).Trim();
        var uri = ParseString(item.TryGetValue("uri", out var uriRaw) ? uriRaw : null).Trim();
        var prefix = ParseString(item.TryGetValue("prefix", out var prefixRaw) ? prefixRaw : null).Trim();
        var ext = ParseString(item.TryGetValue("ext", out var extRaw) ? extRaw : null).Trim();
        var ruleType = ParseString(item.TryGetValue("type", out var typeRaw) ? typeRaw : null).Trim().ToLowerInvariant();
        var rawValue = ParseString(item.TryGetValue("value", out var valueRaw) ? valueRaw : null);

        bool TryKeep(string location)
        {
            var key = NormalizeLocationKey(location);
            if (string.IsNullOrWhiteSpace(key))
            {
                return false;
            }
            if (seen.Contains(key))
            {
                return false;
            }
            seen.Add(key);
            return true;
        }

        if (!string.IsNullOrWhiteSpace(ruleExpr))
        {
            var location = NormalizeRuleLocation(ruleExpr);
            if (!string.IsNullOrWhiteSpace(location) && TryKeep(location))
            {
                item["rule"] = ruleExpr;
                return item;
            }
            return null;
        }

        if (!string.IsNullOrWhiteSpace(uri))
        {
            var path = NormalizeCachePath(uri);
            var location = string.IsNullOrWhiteSpace(path) ? string.Empty : "= " + path;
            if (!string.IsNullOrWhiteSpace(location) && TryKeep(location))
            {
                item["uri"] = path;
                return item;
            }
            return null;
        }

        if (!string.IsNullOrWhiteSpace(prefix))
        {
            var path = NormalizeCachePath(prefix);
            var location = string.IsNullOrWhiteSpace(path) ? string.Empty : "^~ " + path;
            if (!string.IsNullOrWhiteSpace(location) && TryKeep(location))
            {
                item["prefix"] = path;
                return item;
            }
            return null;
        }

        if (!string.IsNullOrWhiteSpace(ext))
        {
            var extValue = NormalizeCacheExtValue(ext);
            var location = string.IsNullOrWhiteSpace(extValue) ? string.Empty : "~* \\." + extValue + "$";
            if (!string.IsNullOrWhiteSpace(location) && TryKeep(location))
            {
                item["ext"] = extValue;
                return item;
            }
            return null;
        }

        switch (ruleType)
        {
            case "all":
                return TryKeep("^~ /") ? item : null;
            case "index":
                return TryKeep("= /") ? item : null;
            case "dir":
            case "path":
            case "suffix":
            {
                var values = SplitCacheRuleValues(rawValue);
                if (values.Count == 0)
                {
                    return null;
                }

                var kept = new List<string>(values.Count);
                foreach (var value in values)
                {
                    switch (ruleType)
                    {
                        case "suffix":
                        {
                            var extValue = NormalizeCacheExtValue(value);
                            if (string.IsNullOrWhiteSpace(extValue))
                            {
                                continue;
                            }
                            var location = "~* \\." + extValue + "$";
                            if (!TryKeep(location))
                            {
                                continue;
                            }
                            kept.Add(extValue);
                            break;
                        }
                        case "dir":
                        {
                            var path = NormalizeCachePath(value);
                            if (string.IsNullOrWhiteSpace(path))
                            {
                                continue;
                            }
                            var location = "^~ " + path;
                            if (!TryKeep(location))
                            {
                                continue;
                            }
                            kept.Add(path);
                            break;
                        }
                        default:
                        {
                            var path = NormalizeCachePath(value);
                            if (string.IsNullOrWhiteSpace(path))
                            {
                                continue;
                            }
                            var location = "= " + path;
                            if (!TryKeep(location))
                            {
                                continue;
                            }
                            kept.Add(path);
                            break;
                        }
                    }
                }

                if (kept.Count == 0)
                {
                    return null;
                }

                item["value"] = string.Join("|", kept);
                return item;
            }
            default:
                return item;
        }
    }

    private static string NormalizeCacheExtValue(string value)
    {
        var raw = value?.Trim().ToLowerInvariant() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return string.Empty;
        }
        raw = raw.TrimStart('*');
        raw = raw.TrimStart('.');
        return raw;
    }

    private static List<Dictionary<string, object?>>? NormalizeHeaderRulesRaw(object? raw)
    {
        var items = NormalizeMapSlice(raw);
        if (items == null || items.Count == 0)
        {
            return null;
        }

        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var output = new List<Dictionary<string, object?>>(items.Count);
        for (var i = items.Count - 1; i >= 0; i--)
        {
            var item = items[i];
            var name = ParseString(item.TryGetValue("name", out var nameRaw) ? nameRaw : null).Trim();
            if (string.IsNullOrWhiteSpace(name))
            {
                continue;
            }
            if (!seen.Add(name))
            {
                continue;
            }
            item["name"] = name;
            output.Add(item);
        }

        output.Reverse();
        return output.Count == 0 ? null : output;
    }

    private static List<Dictionary<string, object?>>? NormalizeUrlRedirectsRaw(object? raw)
    {
        var items = NormalizeMapSlice(raw);
        if (items == null || items.Count == 0)
        {
            return null;
        }

        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var output = new List<Dictionary<string, object?>>(items.Count);
        for (var i = items.Count - 1; i >= 0; i--)
        {
            var item = items[i];
            var match = ParseString(item.TryGetValue("match", out var matchRaw) ? matchRaw : null).Trim();
            var redirect = ParseString(item.TryGetValue("redirect", out var redirectRaw) ? redirectRaw : null).Trim();
            if (string.IsNullOrWhiteSpace(match) || string.IsNullOrWhiteSpace(redirect))
            {
                continue;
            }

            var domain = ParseString(item.TryGetValue("domain", out var domainRaw) ? domainRaw : null).Trim();
            var code = ParseString(item.TryGetValue("code", out var codeRaw) ? codeRaw : null).Trim();
            var condKey = BuildRedirectConditionKey(item.TryGetValue("conditions", out var condRaw) ? condRaw : null);
            var key = domain.ToLowerInvariant() + "|" + match + "|" + redirect + "|" + code + "|" + condKey;
            if (!seen.Add(key))
            {
                continue;
            }

            item["domain"] = domain;
            item["match"] = match;
            item["redirect"] = redirect;
            item["code"] = code;
            output.Add(item);
        }

        output.Reverse();
        return output.Count == 0 ? null : output;
    }

    private static string BuildRedirectConditionKey(object? raw)
    {
        var items = NormalizeMapSlice(raw);
        if (items == null || items.Count == 0)
        {
            return string.Empty;
        }

        var parts = new List<string>(items.Count);
        foreach (var item in items)
        {
            var key = ParseString(item.TryGetValue("key", out var keyRaw) ? keyRaw : null).Trim();
            if (string.IsNullOrWhiteSpace(key))
            {
                key = ParseString(item.TryGetValue("item", out var itemRaw) ? itemRaw : null).Trim();
            }
            var value = ParseString(item.TryGetValue("value", out var valueRaw) ? valueRaw : null).Trim();
            if (string.IsNullOrWhiteSpace(key) && string.IsNullOrWhiteSpace(value))
            {
                continue;
            }
            parts.Add(key.ToLowerInvariant() + "=" + value);
        }

        if (parts.Count == 0)
        {
            return string.Empty;
        }

        parts.Sort(StringComparer.Ordinal);
        return string.Join("&", parts);
    }

    private static List<Dictionary<string, object?>>? NormalizeMapSlice(object? raw)
    {
        if (raw == null)
        {
            return null;
        }

        if (raw is List<Dictionary<string, object?>> list)
        {
            return list.Count == 0 ? null : list;
        }

        if (raw is IReadOnlyList<Dictionary<string, object?>> roList)
        {
            return roList.Count == 0 ? null : roList.ToList();
        }

        if (raw is IEnumerable<object?> items)
        {
            var output = new List<Dictionary<string, object?>>();
            foreach (var item in items)
            {
                var map = AsDictionary(item);
                if (map != null)
                {
                    output.Add(map);
                }
            }
            return output.Count == 0 ? null : output;
        }

        if (raw is JsonElement element)
        {
            if (element.ValueKind == JsonValueKind.Array)
            {
                var output = new List<Dictionary<string, object?>>();
                foreach (var item in element.EnumerateArray())
                {
                    var map = AsDictionary(item);
                    if (map != null)
                    {
                        output.Add(map);
                    }
                }
                return output.Count == 0 ? null : output;
            }

            if (element.ValueKind == JsonValueKind.Object)
            {
                var map = AsDictionary(element);
                if (map != null)
                {
                    return new List<Dictionary<string, object?>> { map };
                }
            }
        }

        return null;
    }

    private static string NormalizeCachePath(string value)
    {
        var item = value?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(item))
        {
            return string.Empty;
        }
        return item.StartsWith("/", StringComparison.Ordinal) ? item : "/" + item;
    }

    private static string NormalizeRuleLocation(string rule)
    {
        var raw = rule?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return string.Empty;
        }
        if (raw.StartsWith("=", StringComparison.Ordinal) || raw.StartsWith("^~", StringComparison.Ordinal) || raw.StartsWith("~", StringComparison.Ordinal))
        {
            return raw;
        }
        if (raw.StartsWith("/", StringComparison.Ordinal))
        {
            return "^~ " + raw;
        }
        if (raw.StartsWith(".", StringComparison.Ordinal))
        {
            return "~* \\" + raw + "$";
        }
        return "~* " + raw;
    }

    private static string NormalizeLocationKey(string location)
    {
        var raw = location?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return string.Empty;
        }
        var parts = raw.Split(' ', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length == 0)
        {
            return string.Empty;
        }

        if (parts[0] == "=")
        {
            return parts.Length < 2 ? "exact" : "exact " + string.Join(' ', parts[1..]);
        }
        if (parts[0] == "^~")
        {
            return parts.Length < 2 ? "prefix" : "prefix " + string.Join(' ', parts[1..]);
        }
        if (parts[0].StartsWith("~", StringComparison.Ordinal))
        {
            return parts.Length < 2 ? "regex " + parts[0] : "regex " + parts[0] + " " + string.Join(' ', parts[1..]);
        }

        return "prefix " + string.Join(' ', parts);
    }

    private static List<string> SplitCacheRuleValues(string value)
    {
        var raw = value?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new List<string>();
        }

        string[] parts;
        if (raw.Contains('|'))
        {
            parts = raw.Split('|', StringSplitOptions.RemoveEmptyEntries);
        }
        else
        {
            parts = raw.Split(' ', StringSplitOptions.RemoveEmptyEntries);
        }

        var output = new List<string>(parts.Length);
        foreach (var part in parts)
        {
            var item = part.Trim();
            if (!string.IsNullOrWhiteSpace(item))
            {
                output.Add(item);
            }
        }

        return output;
    }

    private static string ParseString(object? value)
    {
        if (value == null)
        {
            return string.Empty;
        }

        if (value is string s)
        {
            return s;
        }

        if (value is JsonElement element)
        {
            return element.ValueKind == JsonValueKind.String ? element.GetString() ?? string.Empty : element.ToString();
        }

        return value.ToString() ?? string.Empty;
    }

    private static bool TryGetMap(Dictionary<string, object?> root, string key, out Dictionary<string, object?>? map)
    {
        map = null;
        if (!root.TryGetValue(key, out var raw))
        {
            return false;
        }

        map = AsDictionary(raw);
        if (map == null)
        {
            return false;
        }

        root[key] = map;
        return true;
    }

    private static Dictionary<string, object?>? AsDictionary(object? raw)
    {
        if (raw is Dictionary<string, object?> dict)
        {
            return dict;
        }

        if (raw is IDictionary<string, object?> generic)
        {
            return new Dictionary<string, object?>(generic, StringComparer.OrdinalIgnoreCase);
        }

        if (raw is JsonElement element && element.ValueKind == JsonValueKind.Object)
        {
            return ToDictionary(element);
        }

        return null;
    }

    private static Dictionary<string, object?> ToDictionary(JsonElement element)
    {
        var result = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        foreach (var prop in element.EnumerateObject())
        {
            result[prop.Name] = ConvertJsonElement(prop.Value);
        }
        return result;
    }

    private static object? ConvertJsonElement(JsonElement element)
    {
        switch (element.ValueKind)
        {
            case JsonValueKind.Object:
                return ToDictionary(element);
            case JsonValueKind.Array:
                var list = new List<object?>();
                foreach (var item in element.EnumerateArray())
                {
                    list.Add(ConvertJsonElement(item));
                }
                return list;
            case JsonValueKind.String:
                return element.GetString();
            case JsonValueKind.Number:
                if (element.TryGetInt64(out var l))
                {
                    return l;
                }
                if (element.TryGetDecimal(out var d))
                {
                    return d;
                }
                return element.ToString();
            case JsonValueKind.True:
                return true;
            case JsonValueKind.False:
                return false;
            case JsonValueKind.Null:
            case JsonValueKind.Undefined:
                return null;
            default:
                return element.ToString();
        }
    }
}
