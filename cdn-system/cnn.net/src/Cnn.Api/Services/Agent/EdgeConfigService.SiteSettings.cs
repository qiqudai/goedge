using System.Text.Json;
using Cnn.Common.Contracts.Agent;
using Cnn.Domain.Entities;

namespace Cnn.Api.Services.Agent;

public sealed partial class EdgeConfigService
{
    private const string SiteSettingsType = "site_settings";
    private const string SiteSettingsScope = "site";
    private const string SiteSettingsName = "settings";

    private async Task<Dictionary<int, Dictionary<string, JsonElement>>> LoadSiteSettingsMapAsync(
        IReadOnlyList<Site> sites,
        CancellationToken cancellationToken)
    {
        var result = new Dictionary<int, Dictionary<string, JsonElement>>();
        var siteIds = sites.Select(s => s.Id).Distinct().ToList();
        if (siteIds.Count == 0)
        {
            return result;
        }

        var rows = await _db.Queryable<Config>()
            .Where(c => c.Type == SiteSettingsType && c.ScopeName == SiteSettingsScope && c.ScopeId.HasValue && siteIds.Contains(c.ScopeId.Value))
            .ToListAsync();

        if (rows.Count == 0)
        {
            return result;
        }

        var grouped = rows.GroupBy(r => r.ScopeId!.Value);
        foreach (var group in grouped)
        {
            var candidate = group
                .Where(r => string.Equals(r.Name, SiteSettingsName, StringComparison.OrdinalIgnoreCase) ||
                            string.Equals(r.Name, SiteSettingsType, StringComparison.OrdinalIgnoreCase))
                .OrderByDescending(r => r.UpdateAt ?? r.CreateAt)
                .FirstOrDefault();

            candidate ??= group.OrderByDescending(r => r.UpdateAt ?? r.CreateAt).FirstOrDefault();
            if (candidate == null || string.IsNullOrWhiteSpace(candidate.Value))
            {
                continue;
            }

            try
            {
                var settings = JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(candidate.Value, JsonOptions);
                if (settings != null && settings.Count > 0)
                {
                    result[group.Key] = settings;
                }
            }
            catch
            {
            }
        }

        return result;
    }

    private static Dictionary<string, JsonElement>? GetSiteSettings(
        Dictionary<int, Dictionary<string, JsonElement>> map,
        int siteId)
    {
        return map.TryGetValue(siteId, out var settings) ? settings : null;
    }

    private static Dictionary<string, JsonElement>? GetSettingsMap(
        Dictionary<string, JsonElement>? root,
        string key)
    {
        if (root == null)
        {
            return null;
        }

        if (!TryGetEntry(root, key, out var value) || value.ValueKind != JsonValueKind.Object)
        {
            return null;
        }

        var map = new Dictionary<string, JsonElement>(StringComparer.OrdinalIgnoreCase);
        foreach (var prop in value.EnumerateObject())
        {
            map[prop.Name] = prop.Value;
        }
        return map;
    }

    private static bool? ParseBoolSetting(Dictionary<string, JsonElement>? map, string key)
    {
        if (map == null)
        {
            return null;
        }

        if (!TryGetEntry(map, key, out var value))
        {
            return null;
        }

        return ParseBool(value, false);
    }

    private static string? ParseStringSetting(Dictionary<string, JsonElement>? map, string key)
    {
        if (map == null)
        {
            return null;
        }

        if (!TryGetEntry(map, key, out var value))
        {
            return null;
        }

        var text = ParseString(value);
        return string.IsNullOrWhiteSpace(text) ? null : text;
    }

    private static int? ParseIntSetting(Dictionary<string, JsonElement>? map, string key)
    {
        if (map == null)
        {
            return null;
        }

        if (!TryGetEntry(map, key, out var value))
        {
            return null;
        }

        return ParseInt(value, null);
    }

    private static int? ParseInt(JsonElement value, int? fallback)
    {
        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var parsed))
        {
            return parsed;
        }

        if (value.ValueKind == JsonValueKind.String && int.TryParse(value.GetString(), out var parsedString))
        {
            return parsedString;
        }

        return fallback;
    }

    private sealed record HttpsConfig(
        bool? Enable,
        bool? Force,
        string? RedirectPort,
        bool? Hsts,
        bool? Http2,
        bool? Http3,
        bool? Ocsp,
        string? SslProtocols,
        string? SslCiphers,
        string? SslPreferServerCiphers);

    private static HttpsConfig ExtractHttpsConfig(Dictionary<string, JsonElement>? settings)
    {
        var https = GetSettingsMap(settings, "https");
        if (https == null)
        {
            return new HttpsConfig(null, null, null, null, null, null, null, null, null, null);
        }

        var enable = ParseBoolSetting(https, "enable");
        var force = ParseBoolSetting(https, "force");
        var redirectPort = ParseStringSetting(https, "redirect_port");
        var hsts = ParseBoolSetting(https, "hsts");
        var http2 = ParseBoolSetting(https, "http2");
        var http3 = ParseBoolSetting(https, "http3");
        var ocsp = ParseBoolSetting(https, "ocsp_stapling");
        var sslProtocols = ParseStringSetting(https, "ssl_protocols");
        var sslCiphers = ParseStringSetting(https, "ssl_ciphers");
        var sslPrefer = NormalizeOnOff(ParseStringSetting(https, "ssl_prefer_server_ciphers"));
        var profile = ParseStringSetting(https, "ssl_profile")?.Trim().ToLowerInvariant() ?? string.Empty;

        if (profile == "modern")
        {
            if (string.IsNullOrWhiteSpace(sslProtocols))
            {
                sslProtocols = "TLSv1.2 TLSv1.3";
            }
            if (string.IsNullOrWhiteSpace(sslCiphers))
            {
                sslCiphers = "ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256";
            }
        }
        else if (profile == "compat")
        {
            if (string.IsNullOrWhiteSpace(sslProtocols))
            {
                sslProtocols = "TLSv1 TLSv1.1 TLSv1.2 TLSv1.3";
            }
        }

        return new HttpsConfig(enable, force, redirectPort, hsts, http2, http3, ocsp, sslProtocols, sslCiphers, sslPrefer);
    }

    private static bool? ResolveHttpEnable(Dictionary<string, JsonElement>? settings)
    {
        if (settings == null)
        {
            return null;
        }

        if (TryGetEntry(settings, "http_enable", out var value))
        {
            return ParseBool(value, true);
        }

        return null;
    }

    private static bool? ResolveHttpsEnable(Dictionary<string, JsonElement>? settings)
    {
        var https = GetSettingsMap(settings, "https");
        return ParseBoolSetting(https, "enable");
    }

    private static string? ResolveL2Mode(Dictionary<string, JsonElement>? settings)
    {
        if (settings == null)
        {
            return null;
        }

        if (TryGetEntry(settings, "l2_config", out var value))
        {
            var raw = ParseString(value);
            if (!string.IsNullOrWhiteSpace(raw))
            {
                return raw;
            }
        }

        var adv = GetSettingsMap(settings, "advanced");
        var advValue = ParseStringSetting(adv, "l2_config");
        return string.IsNullOrWhiteSpace(advValue) ? null : advValue;
    }

    private static EdgeCacheConfigDto? ExtractCacheConfig(
        Dictionary<string, JsonElement>? settings,
        string? fallbackRaw)
    {
        var cache = GetSettingsMap(settings, "cache");
        if (cache == null)
        {
            return ParseCacheConfig(fallbackRaw);
        }

        if (TryGetEntry(cache, "profiles", out _))
        {
            return ParseCacheConfig(fallbackRaw);
        }

        var enable = ParseBoolSetting(cache, "enable") ?? true;
        var ttl = ParseIntSetting(cache, "ttl");
        List<EdgeCacheRuleDto>? rules = null;
        if (TryGetEntry(cache, "rules", out var rulesValue))
        {
            rules = ParseCacheRulesFromElement(rulesValue);
        }

        if (!enable && (rules == null || rules.Count == 0) && ttl == null)
        {
            return null;
        }

        return new EdgeCacheConfigDto
        {
            Enable = enable,
            DefaultTtl = ttl,
            Rules = rules
        };
    }

    private static (string? ConnectTimeout, string? ReadTimeout, string? SendTimeout) ExtractProxyTimeouts(
        Dictionary<string, JsonElement>? settings,
        string? fallbackRaw)
    {
        var backsource = GetSettingsMap(settings, "backsource");
        if (backsource != null)
        {
            var connect = NormalizeTimeout(ParseStringSetting(backsource, "connect_timeout"));
            if (string.IsNullOrWhiteSpace(connect))
            {
                connect = NormalizeTimeout(ParseStringSetting(backsource, "timeout"));
            }
            var timeout = NormalizeTimeout(ParseStringSetting(backsource, "timeout"));
            return (connect, timeout, timeout);
        }

        return ParseProxyTimeouts(fallbackRaw);
    }

    private static EdgeHotlinkConfigDto? ExtractHotlinkConfig(
        Dictionary<string, JsonElement>? settings,
        string? fallbackRaw)
    {
        var access = GetSettingsMap(settings, "access");
        var hotlink = GetSettingsMap(access, "hotlink");
        if (hotlink != null)
        {
            var enable = ParseBoolSetting(hotlink, "enable") ?? false;
            if (!enable)
            {
                return null;
            }

            return new EdgeHotlinkConfigDto
            {
                Enable = true,
                Scope = ParseStringSetting(hotlink, "scope"),
                Value = ParseStringSetting(hotlink, "value"),
                AllowEmpty = ParseBoolSetting(hotlink, "allowEmpty") ?? true,
                Domains = ParseStringListFromElement(hotlink, "domains")
            };
        }

        return ParseHotlinkConfig(fallbackRaw);
    }

    private static EdgeCorsConfigDto? ExtractCorsConfig(
        Dictionary<string, JsonElement>? settings,
        string? fallbackRaw)
    {
        var access = GetSettingsMap(settings, "access");
        var cors = GetSettingsMap(access, "cors");
        if (cors != null)
        {
            var enable = ParseBoolSetting(cors, "enable") ?? false;
            if (!enable)
            {
                return null;
            }

            return new EdgeCorsConfigDto
            {
                Enable = true,
                AllowOrigin = ParseStringSetting(cors, "allowOrigin"),
                AllowMethods = ParseStringSetting(cors, "allowMethods"),
                AllowHeaders = ParseStringSetting(cors, "allowHeaders"),
                ExposeHeaders = ParseStringSetting(cors, "exposeHeaders"),
                AllowCredentials = ParseBoolSetting(cors, "allowCredentials") ?? false,
                MaxAge = ParseStringSetting(cors, "maxAge")
            };
        }

        return ParseCorsConfig(fallbackRaw);
    }

    private static EdgeCookieConfigDto? ExtractCookieConfig(Dictionary<string, JsonElement>? settings)
    {
        var security = GetSettingsMap(settings, "security");
        var cookie = GetSettingsMap(security, "cookie");
        if (cookie == null)
        {
            return null;
        }

        var enable = ParseBoolSetting(cookie, "enable") ?? false;
        var domain = ParseStringSetting(cookie, "domain");
        if (!enable || string.IsNullOrWhiteSpace(domain))
        {
            return null;
        }

        return new EdgeCookieConfigDto
        {
            Enable = true,
            Domain = domain
        };
    }

    private static bool ExtractBlockTransparentProxy(
        Dictionary<string, JsonElement>? settings,
        bool? fallback)
    {
        var security = GetSettingsMap(settings, "security");
        var value = ParseBoolSetting(security, "block_transparent_proxy");
        return value ?? (fallback ?? false);
    }

    private static string ExtractCrawlerAction(
        Dictionary<string, JsonElement>? settings,
        string? fallback,
        Dictionary<string, string> defaults)
    {
        var security = GetSettingsMap(settings, "security");
        var action = ParseStringSetting(security, "crawlers_action");
        if (string.IsNullOrWhiteSpace(action))
        {
            action = fallback;
        }
        if (string.IsNullOrWhiteSpace(action))
        {
            action = GetDefaultValue(defaults, "security_bot");
        }

        if (string.IsNullOrWhiteSpace(action))
        {
            return string.Empty;
        }

        var normalized = action.Trim().ToLowerInvariant();
        return normalized is "allow" or "deny" or "block" ? normalized : string.Empty;
    }

    private static (int PassTtl, int BlockTtl) ExtractGuardTtls(
        Dictionary<string, JsonElement>? settings,
        Dictionary<string, string> defaults)
    {
        var security = GetSettingsMap(settings, "security");
        var pass = ParseIntSetting(security, "ip_white_timeout") ?? 0;
        var block = ParseIntSetting(security, "ip_black_timeout") ?? 0;
        if (pass == 0)
        {
            pass = ParseDefaultInt(defaults, "security_white_time") ?? 0;
        }
        if (block == 0)
        {
            block = ParseDefaultInt(defaults, "security_black_time") ?? 0;
        }
        return (pass, block);
    }

    private static List<Dictionary<string, JsonElement>>? ExtractUrlRedirects(
        Dictionary<string, JsonElement>? settings,
        string? fallbackRaw)
    {
        var adv = GetSettingsMap(settings, "advanced");
        if (adv != null && TryGetEntry(adv, "url_redirects", out var value))
        {
            return ParseMapList(value);
        }

        return ParseUrlRedirects(fallbackRaw);
    }

    private static List<Dictionary<string, JsonElement>>? ExtractOriginConditions(
        Dictionary<string, JsonElement>? settings)
    {
        var origin = GetSettingsMap(settings, "origin");
        List<Dictionary<string, JsonElement>>? conditions = null;
        if (origin != null && TryGetEntry(origin, "conditions", out var value))
        {
            conditions = ParseMapList(value);
        }

        return WithSearchEngineOriginCondition(settings, conditions);
    }

    private static List<Dictionary<string, JsonElement>>? WithSearchEngineOriginCondition(
        Dictionary<string, JsonElement>? settings,
        List<Dictionary<string, JsonElement>>? conditions)
    {
        if (settings == null)
        {
            return conditions;
        }

        if (!TryGetEntry(settings, "search_engine_origin", out var enabledValue) || !ParseBool(enabledValue, false))
        {
            return conditions;
        }

        var originIp = string.Empty;
        if (TryGetEntry(settings, "search_engine_origin_ip", out var originValue))
        {
            originIp = ParseString(originValue).Trim();
        }

        if (string.IsNullOrWhiteSpace(originIp))
        {
            return conditions;
        }

        var cond = new Dictionary<string, JsonElement>();
        var json = "{\"item\":\"header\",\"header\":\"user-agent\",\"operator\":\"contains\",\"value\":\"baiduspider|googlebot|bingbot|yandex|sogou|360spider|bytespider|duckduckbot|slurp|facebot|ia_archiver|semrushbot\",\"origin\":\"" + originIp + "\"}";
        try
        {
            cond = JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(json, JsonOptions) ?? new Dictionary<string, JsonElement>();
        }
        catch
        {
        }

        if (cond.Count == 0)
        {
            return conditions;
        }

        if (conditions == null || conditions.Count == 0)
        {
            return new List<Dictionary<string, JsonElement>> { cond };
        }

        return new List<Dictionary<string, JsonElement>> { cond }.Concat(conditions).ToList();
    }

    private static List<Dictionary<string, JsonElement>>? ParseMapList(JsonElement element)
    {
        if (element.ValueKind == JsonValueKind.Null || element.ValueKind == JsonValueKind.Undefined)
        {
            return null;
        }

        if (element.ValueKind != JsonValueKind.Array)
        {
            return null;
        }

        try
        {
            var list = JsonSerializer.Deserialize<List<Dictionary<string, JsonElement>>>(element.GetRawText(), JsonOptions);
            return list == null || list.Count == 0 ? null : list;
        }
        catch
        {
            return null;
        }
    }

    private static List<string>? ParseStringListFromElement(Dictionary<string, JsonElement>? map, string key)
    {
        if (map == null)
        {
            return null;
        }

        if (!TryGetEntry(map, key, out var value))
        {
            return null;
        }

        return ParseStringListFromElement(value);
    }

    private static List<string>? ParseStringListFromElement(JsonElement element)
    {
        if (element.ValueKind == JsonValueKind.Array)
        {
            var list = new List<string>();
            foreach (var item in element.EnumerateArray())
            {
                if (item.ValueKind == JsonValueKind.String)
                {
                    list.Add(item.GetString() ?? string.Empty);
                }
                else
                {
                    list.Add(item.ToString());
                }
            }
            return list.Count == 0 ? null : list;
        }

        if (element.ValueKind == JsonValueKind.String)
        {
            var raw = element.GetString();
            if (string.IsNullOrWhiteSpace(raw))
            {
                return null;
            }
            var list = SplitFields(raw);
            return list.Count == 0 ? null : list;
        }

        return null;
    }

    private static long? ResolveAclId(Dictionary<string, JsonElement>? settings, long? fallback)
    {
        var access = GetSettingsMap(settings, "access");
        if (access != null && TryGetEntry(access, "acl", out var value))
        {
            var id = ParseId(value);
            if (id > 0)
            {
                return id;
            }
        }

        return fallback;
    }

    private static List<string>? ExtractRegionBlock(
        Dictionary<string, JsonElement>? settings,
        string? fallback)
    {
        if (settings != null)
        {
            var access = GetSettingsMap(settings, "access");
            if (access != null && TryGetEntry(access, "region_block", out var value))
            {
                var list = ParseRegionListValue(value);
                if (list != null && list.Count > 0)
                {
                    return list;
                }
            }

            var security = GetSettingsMap(settings, "security");
            if (security != null)
            {
                if (TryGetEntry(security, "region_block", out var securityValue))
                {
                    var list = ParseRegionListValue(securityValue);
                    if (list != null && list.Count > 0)
                    {
                        return list;
                    }
                }
                if (TryGetEntry(security, "region_custom", out var customValue))
                {
                    var list = ParseRegionListValue(customValue);
                    if (list != null && list.Count > 0)
                    {
                        return list;
                    }
                }
            }
        }

        return ParseRegionList(fallback);
    }

    private static List<string>? ParseRegionListValue(JsonElement element)
    {
        if (element.ValueKind == JsonValueKind.Array)
        {
            var list = new List<string>();
            foreach (var item in element.EnumerateArray())
            {
                list.Add(item.ToString());
            }
            return NormalizeRegionList(list);
        }

        if (element.ValueKind == JsonValueKind.String)
        {
            var raw = element.GetString();
            if (string.IsNullOrWhiteSpace(raw))
            {
                return null;
            }
            var list = SplitFields(raw);
            return NormalizeRegionList(list);
        }

        return null;
    }

    private static string? ResolveOriginHost(
        Dictionary<string, JsonElement>? settings,
        Site site)
    {
        if (settings == null)
        {
            return site.BackendHost?.Trim();
        }

        var backsource = GetSettingsMap(settings, "backsource");
        if (backsource != null)
        {
            var mode = ParseStringSetting(backsource, "host_mode")?.ToLowerInvariant();
            switch (mode)
            {
                case "follow":
                case "":
                case null:
                    return string.Empty;
                case "domain":
                    return FirstDomain(SplitDomainList(site.Domain));
                case "custom":
                    return ParseStringSetting(backsource, "host_custom");
                default:
                    if (!string.IsNullOrWhiteSpace(mode))
                    {
                        return mode;
                    }
                    break;
            }
        }

        if (TryGetEntry(settings, "origin_host", out var originHost))
        {
            var rawHost = ParseString(originHost);
            if (!string.IsNullOrWhiteSpace(rawHost))
            {
                return rawHost.Trim();
            }
        }

        var origin = GetSettingsMap(settings, "origin");
        var originValue = ParseStringSetting(origin, "host");
        return originValue;
    }

    private static string FirstDomain(List<string> domains)
    {
        foreach (var domain in domains)
        {
            var trimmed = domain.Trim();
            if (!string.IsNullOrWhiteSpace(trimmed))
            {
                return trimmed;
            }
        }
        return string.Empty;
    }

    private static List<EdgeUpstreamTargetDto> BuildUpstreamTargetsFromSettings(
        Dictionary<string, JsonElement>? settings,
        string originProtocol,
        string originHttpPort,
        string originHttpsPort)
    {
        var origin = GetSettingsMap(settings, "origin");
        if (origin == null || !TryGetEntry(origin, "list", out var listElement) || listElement.ValueKind != JsonValueKind.Array)
        {
            return new List<EdgeUpstreamTargetDto>();
        }

        var targets = new List<EdgeUpstreamTargetDto>();
        foreach (var item in listElement.EnumerateArray())
        {
            if (item.ValueKind != JsonValueKind.Object)
            {
                continue;
            }

            var map = new Dictionary<string, JsonElement>(StringComparer.OrdinalIgnoreCase);
            foreach (var prop in item.EnumerateObject())
            {
                map[prop.Name] = prop.Value;
            }

            var addr = ParseStringFromMap(map, "address");
            if (string.IsNullOrWhiteSpace(addr))
            {
                continue;
            }

            if (TryGetEntry(map, "enable", out var enableValue) && !ParseBool(enableValue, true))
            {
                continue;
            }

            var weight = ParseIntFromMap(map, "weight") ?? 10;
            if (weight <= 0)
            {
                weight = 10;
            }

            var normalized = NormalizeOriginAddr(addr, originProtocol, originHttpPort, originHttpsPort);
            if (string.IsNullOrWhiteSpace(normalized))
            {
                continue;
            }

            targets.Add(new EdgeUpstreamTargetDto
            {
                Addr = normalized,
                Weight = weight
            });
        }

        return targets;
    }

    private static Dictionary<string, string>? BuildHeaderMapFromSettings(
        Dictionary<string, JsonElement>? settings,
        Site site)
    {
        if (settings == null)
        {
            return null;
        }

        var result = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        if (TryGetEntry(settings, "headers", out var headersValue) && headersValue.ValueKind == JsonValueKind.Object)
        {
            foreach (var prop in headersValue.EnumerateObject())
            {
                var name = SanitizeHeaderName(prop.Name);
                if (string.IsNullOrWhiteSpace(name))
                {
                    continue;
                }

                var value = prop.Value.ValueKind == JsonValueKind.String ? prop.Value.GetString() : prop.Value.ToString();
                value = SanitizeHeaderValue(value);
                if (string.IsNullOrWhiteSpace(value))
                {
                    continue;
                }

                result[name] = value;
            }
        }

        var adv = GetSettingsMap(settings, "advanced");
        if (adv != null && TryGetEntry(adv, "origin_headers", out var originHeaders) && originHeaders.ValueKind == JsonValueKind.Array)
        {
            foreach (var item in originHeaders.EnumerateArray())
            {
                if (item.ValueKind != JsonValueKind.Object)
                {
                    continue;
                }

                var name = item.TryGetProperty("name", out var nameValue) ? nameValue.GetString() : null;
                var value = item.TryGetProperty("value", out var valueValue) ? valueValue.GetString() : null;
                name = SanitizeHeaderName(name);
                value = SanitizeHeaderValue(value);
                if (string.IsNullOrWhiteSpace(name) || string.IsNullOrWhiteSpace(value))
                {
                    continue;
                }

                result[name] = value;
            }
        }

        if (!result.ContainsKey("Host"))
        {
            var originHost = ResolveOriginHost(settings, site);
            if (!string.IsNullOrWhiteSpace(originHost))
            {
                var value = SanitizeHeaderValue(originHost);
                if (!string.IsNullOrWhiteSpace(value))
                {
                    result["Host"] = value;
                }
            }
        }

        return result.Count == 0 ? null : result;
    }

    private static Dictionary<string, string>? BuildResponseHeaderMapFromSettings(Dictionary<string, JsonElement>? settings)
    {
        var adv = GetSettingsMap(settings, "advanced");
        if (adv == null || !TryGetEntry(adv, "cdn_headers", out var headers) || headers.ValueKind != JsonValueKind.Array)
        {
            return null;
        }

        var result = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var item in headers.EnumerateArray())
        {
            if (item.ValueKind != JsonValueKind.Object)
            {
                continue;
            }

            var name = item.TryGetProperty("name", out var nameValue) ? nameValue.GetString() : null;
            var value = item.TryGetProperty("value", out var valueValue) ? valueValue.GetString() : null;
            name = SanitizeHeaderName(name);
            value = SanitizeHeaderValue(value);
            if (string.IsNullOrWhiteSpace(name) || string.IsNullOrWhiteSpace(value))
            {
                continue;
            }

            result[name] = value;
        }

        return result.Count == 0 ? null : result;
    }
}
