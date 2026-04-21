using System.Net;
using System.Text.Json;
using Task = System.Threading.Tasks.Task;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed partial class SiteService
{
    private static List<long> ParseGroupIds(string? input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return new List<long>();
        }

        var parts = input.Split(new[] { ',', '|', ' ', ';' }, StringSplitOptions.RemoveEmptyEntries);
        var result = new List<long>(parts.Length);
        foreach (var part in parts)
        {
            if (long.TryParse(part.Trim(), out var id) && id > 0)
            {
                result.Add(id);
            }
        }

        return result.Distinct().ToList();
    }

    private static List<string> SplitFields(string raw)
    {
        var normalized = raw.Replace(",", " ").Replace(";", " ").Replace("\n", " ").Replace("\r", " ");
        return normalized.Split(' ', StringSplitOptions.RemoveEmptyEntries)
            .Select(part => part.Trim())
            .Where(part => !string.IsNullOrWhiteSpace(part))
            .ToList();
    }

    private async Task TrySyncUserDnsRecordsAsync(Site? oldSite, Site? newSite)
    {
        try
        {
            await _dnsSyncService.SyncUserDnsRecordsAsync(oldSite, newSite);
        }
        catch
        {
        }
    }

    private static List<string> SplitLines(string raw)
    {
        return raw.Split(new[] { "\r\n", "\n", "\r" }, StringSplitOptions.RemoveEmptyEntries)
            .Select(line => line.Trim())
            .Where(line => !string.IsNullOrWhiteSpace(line))
            .ToList();
    }

    private static List<string> SplitByComma(string raw)
    {
        return raw.Split(',', StringSplitOptions.RemoveEmptyEntries)
            .Select(item => item.Trim())
            .Where(item => !string.IsNullOrWhiteSpace(item))
            .ToList();
    }

    private static List<string> DecodeStringList(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new List<string>();
        }

        var trimmed = raw.Trim();
        if (trimmed.StartsWith("[", StringComparison.Ordinal))
        {
            try
            {
                var list = JsonSerializer.Deserialize<List<string>>(trimmed, JsonOptions);
                if (list != null)
                {
                    return list.Select(item => item?.Trim() ?? string.Empty)
                        .Where(item => !string.IsNullOrWhiteSpace(item))
                        .ToList();
                }
            }
            catch
            {
            }
        }

        return SplitFields(trimmed);
    }

    private static string EncodeStringList(IEnumerable<string> items)
    {
        var list = items.Select(item => item?.Trim() ?? string.Empty)
            .Where(item => !string.IsNullOrWhiteSpace(item))
            .ToList();
        return list.Count == 0 ? string.Empty : JsonSerializer.Serialize(list, JsonOptions);
    }

    private static List<string> NormalizeStringList(IEnumerable<string> items)
    {
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var output = new List<string>();
        foreach (var item in items)
        {
            var value = item?.Trim() ?? string.Empty;
            if (string.IsNullOrWhiteSpace(value))
            {
                continue;
            }
            if (seen.Add(value))
            {
                output.Add(value);
            }
        }
        return output;
    }

    private async Task<List<long>> FindSiteIdsByGroupIdsAsync(IReadOnlyList<long> groupIds)
    {
        if (groupIds == null || groupIds.Count == 0)
        {
            return new List<long>();
        }

        var ids = groupIds.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return new List<long>();
        }

        return await _db.Queryable<MergeSiteGroup>()
            .Where(r => r.GroupId.HasValue && ids.Contains(r.GroupId.Value))
            .Select(r => (long)r.SiteId!.Value)
            .ToListAsync();
    }

    private async Task<List<long>> FindSiteIdsByGroupNameAsync(string keyword)
    {
        if (string.IsNullOrWhiteSpace(keyword))
        {
            return new List<long>();
        }

        var groupIds = await _db.Queryable<SiteGroup>()
            .Where(g => SqlFunc.Contains(g.Name, keyword))
            .Select(g => (long)g.Id)
            .ToListAsync();

        return await FindSiteIdsByGroupIdsAsync(groupIds);
    }

    private async Task<List<long>> FindUserPackageIdsByNameAsync(string keyword)
    {
        if (string.IsNullOrWhiteSpace(keyword))
        {
            return new List<long>();
        }

        return await _db.Queryable<UserPackage>()
            .Where(p => SqlFunc.Contains(p.Name, keyword))
            .Select(p => (long)p.Id)
            .ToListAsync();
    }

    private async Task<List<long>> FindUserIdsByKeywordAsync(string keyword)
    {
        if (string.IsNullOrWhiteSpace(keyword))
        {
            return new List<long>();
        }

        return await _db.Queryable<User>()
            .Where(u => SqlFunc.Contains(u.Name, keyword) || SqlFunc.Contains(u.Email, keyword) || SqlFunc.Contains(u.Phone, keyword))
            .Select(u => (long)u.Id)
            .ToListAsync();
    }

    private async Task<Dictionary<long, string>> LoadUserNameMapAsync(IReadOnlyList<Site> sites)
    {
        var ids = sites.Select(s => (long)(s.Uid ?? 0)).Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return new Dictionary<long, string>();
        }

        var users = await _db.Queryable<User>().Where(u => ids.Contains(u.Id)).ToListAsync();
        return users.ToDictionary(u => (long)u.Id, u => u.Name ?? string.Empty);
    }

    private async Task<Dictionary<long, UserPackage>> LoadUserPackageMapAsync(IReadOnlyList<Site> sites)
    {
        var ids = sites.Select(s => (long)(s.UserPackage ?? 0)).Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return new Dictionary<long, UserPackage>();
        }

        var packages = await _db.Queryable<UserPackage>().Where(p => ids.Contains(p.Id)).ToListAsync();
        return packages.ToDictionary(p => (long)p.Id, p => p);
    }

    private async Task<(Dictionary<long, string> GroupMap, Dictionary<long, List<long>> RelMap)> LoadSiteGroupMapAsync(IReadOnlyList<Site> sites)
    {
        var siteIds = sites.Select(s => (long)s.Id).Where(id => id > 0).Distinct().ToList();
        var relMap = new Dictionary<long, List<long>>();
        if (siteIds.Count == 0)
        {
            return (new Dictionary<long, string>(), relMap);
        }

        var relations = await _db.Queryable<MergeSiteGroup>()
            .Where(r => r.SiteId.HasValue && siteIds.Contains(r.SiteId.Value))
            .ToListAsync();

        var groupIds = new HashSet<long>();
        foreach (var rel in relations)
        {
            if (rel.SiteId is null || rel.GroupId is null)
            {
                continue;
            }

            var siteId = (long)rel.SiteId.Value;
            var groupId = (long)rel.GroupId.Value;
            if (!relMap.TryGetValue(siteId, out var list))
            {
                list = new List<long>();
                relMap[siteId] = list;
            }
            list.Add(groupId);
            groupIds.Add(groupId);
        }

        var groupMap = new Dictionary<long, string>();
        if (groupIds.Count == 0)
        {
            return (groupMap, relMap);
        }

        var groups = await _db.Queryable<SiteGroup>().Where(g => groupIds.Contains(g.Id)).ToListAsync();
        foreach (var group in groups)
        {
            groupMap[group.Id] = group.Name ?? string.Empty;
        }

        return (groupMap, relMap);
    }

    private async Task<Dictionary<long, string>> LoadNodeGroupMapAsync(IReadOnlyList<Site> sites)
    {
        var ids = sites.Select(s => (long)(s.NodeGroupId ?? 0)).Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return new Dictionary<long, string>();
        }

        var groups = await _db.Queryable<NodeGroup>().Where(g => ids.Contains(g.Id)).ToListAsync();
        return groups.ToDictionary(g => (long)g.Id, g => g.Name ?? string.Empty);
    }

    private async Task<Dictionary<long, string>> LoadRegionMapAsync(IReadOnlyList<Site> sites)
    {
        var ids = sites.Select(s => (long)(s.RegionId ?? 0)).Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return new Dictionary<long, string>();
        }

        var regions = await _db.Queryable<Region>().Where(r => ids.Contains(r.Id)).ToListAsync();
        return regions.ToDictionary(r => (long)r.Id, r => r.Name ?? string.Empty);
    }

    private Task<Dictionary<long, Dictionary<string, object?>>> LoadSiteSettingsMapAsync(IReadOnlyList<long> siteIds)
        => _siteSettingsStore.LoadSettingsMapAsync(siteIds);

    private Task<Dictionary<long, string>> LoadSiteTypeMetaMapAsync(IReadOnlyList<long> siteIds)
        => _siteSettingsStore.LoadSiteTypeMapAsync(siteIds);

    private async Task<GlobalConfigDto?> LoadGlobalDefaultConfigAsync()
    {
        var result = await _globalConfigService.GetAsync(CancellationToken.None);
        return result.Success ? result.Data : null;
    }

    private async Task<Dictionary<string, object?>> EnsureSiteSettingsAsync(
        Site site,
        long groupId,
        string siteType,
        Dictionary<string, object?> settings,
        GlobalConfigDto? globalDefaults,
        Dictionary<(long, long), Dictionary<string, string>> defaultCache,
        Dictionary<(long, long), Dictionary<string, string>> scopedCache)
    {
        settings ??= new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        var settingsEmpty = settings.Count == 0 || (settings.Count == 1 && settings.ContainsKey("site_type"));

        if (!settings.TryGetValue("site_type", out var typeRaw) || string.IsNullOrWhiteSpace(ParseString(typeRaw)))
        {
            siteType = siteType?.Trim().ToLowerInvariant() ?? string.Empty;
            if (string.IsNullOrWhiteSpace(siteType))
            {
                siteType = "website";
            }
            settings["site_type"] = siteType;
        }

        ApplySiteTemplateDefaultsByType(settings, siteType, globalDefaults);

        var cacheKey = (site.Uid ?? 0, groupId);
        if (!defaultCache.TryGetValue(cacheKey, out var defaults))
        {
            defaults = await LoadSiteDefaultMapWithGroupAsync(site.Uid ?? 0, groupId);
            defaultCache[cacheKey] = defaults;
        }
        ApplySiteDefaults(site, settings, defaults);

        if (settingsEmpty)
        {
            if (!scopedCache.TryGetValue(cacheKey, out var scoped))
            {
                scoped = await LoadSiteScopedDefaultMapAsync(site.Uid ?? 0, groupId) ?? new Dictionary<string, string>();
                scopedCache[cacheKey] = scoped;
            }
            if (scoped.Count > 0)
            {
                ApplySiteDefaultsScopedOverrides(settings, scoped);
            }
        }

        return SiteSettingsNormalizer.Normalize(settings);
    }

    private async Task<Dictionary<string, string>> LoadSiteDefaultMapWithGroupAsync(long userId, long groupId)
    {
        var global = await LoadConfigMapAsync("site_default_config", "global", 0);
        var certGlobal = await LoadConfigMapAsync("cert_default_config", "global", 0);
        if (certGlobal.Count > 0)
        {
            foreach (var pair in certGlobal)
            {
                global[pair.Key] = pair.Value;
            }
        }

        var legacyUser = userId > 0 ? await LoadConfigMapAsync("site_default_config", "user", (int)userId) : new Dictionary<string, string>();
        var userGlobal = userId > 0 ? await LoadConfigMapAsync("site_default_config", "global", (int)userId) : new Dictionary<string, string>();

        var merged = new Dictionary<string, string>(global, StringComparer.OrdinalIgnoreCase);
        foreach (var pair in legacyUser)
        {
            merged[pair.Key] = pair.Value;
        }
        foreach (var pair in userGlobal)
        {
            merged[pair.Key] = pair.Value;
        }

        if (groupId > 0)
        {
            var groupDefaults = await LoadConfigMapAsync("site_default_config", "group", (int)groupId);
            foreach (var pair in groupDefaults)
            {
                merged[pair.Key] = pair.Value;
            }
        }

        return merged;
    }

    private async Task<Dictionary<string, string>?> LoadSiteScopedDefaultMapAsync(long userId, long groupId)
    {
        var scoped = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        if (userId > 0)
        {
            var legacyUser = await LoadConfigMapAsync("site_default_config", "user", (int)userId);
            foreach (var pair in legacyUser)
            {
                scoped[pair.Key] = pair.Value;
            }

            var userGlobal = await LoadConfigMapAsync("site_default_config", "global", (int)userId);
            foreach (var pair in userGlobal)
            {
                scoped[pair.Key] = pair.Value;
            }
        }

        if (groupId > 0)
        {
            var groupDefaults = await LoadConfigMapAsync("site_default_config", "group", (int)groupId);
            foreach (var pair in groupDefaults)
            {
                scoped[pair.Key] = pair.Value;
            }
        }

        return scoped.Count == 0 ? null : scoped;
    }

    private async Task<Dictionary<string, string>> LoadConfigMapAsync(string type, string scopeName, int scopeId)
    {
        var rows = await _db.Queryable<Config>()
            .Where(c => c.Type == type && c.ScopeName == scopeName && c.ScopeId == scopeId)
            .ToListAsync();

        var result = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row.Name) || row.Enable == false)
            {
                continue;
            }
            result[row.Name] = row.Value ?? string.Empty;
        }

        return result;
    }

    private static void ApplySiteTemplateDefaultsByType(Dictionary<string, object?> settings, string siteType, GlobalConfigDto? globalDefaults)
    {
        if (settings == null || globalDefaults?.DefaultConfig == null)
        {
            return;
        }

        var template = siteType switch
        {
            "api" => globalDefaults.DefaultConfig.Api,
            "download" => globalDefaults.DefaultConfig.Download,
            _ => globalDefaults.DefaultConfig.Website
        };

        if (template == null)
        {
            return;
        }

        var cacheCfg = GetSubMap(settings, "cache");
        if (!cacheCfg.ContainsKey("enable"))
        {
            cacheCfg["enable"] = template.CacheEnable;
        }
        if (template.CacheTtl > 0 && !cacheCfg.ContainsKey("ttl"))
        {
            cacheCfg["ttl"] = template.CacheTtl;
        }

        var advCfg = GetSubMap(settings, "advanced");
        if (!advCfg.ContainsKey("gzip"))
        {
            advCfg["gzip"] = template.Gzip;
        }

        var httpsCfg = GetSubMap(settings, "https");
        if (!string.IsNullOrWhiteSpace(template.SslCiphers) && !httpsCfg.ContainsKey("ssl_ciphers"))
        {
            httpsCfg["ssl_ciphers"] = template.SslCiphers;
        }

        var securityCfg = GetSubMap(settings, "security");
        if (!securityCfg.ContainsKey("waf_enable"))
        {
            securityCfg["waf_enable"] = template.WafEnable;
        }
    }

    private static void ApplySiteDefaults(Site site, Dictionary<string, object?> settings, Dictionary<string, string> defaults)
    {
        if (defaults == null || defaults.Count == 0)
        {
            return;
        }

        var httpEnable = true;
        if (settings.TryGetValue("http_enable", out var httpEnableRaw))
        {
            httpEnable = ParseBool(httpEnableRaw, true);
        }
        if (httpEnable)
        {
            if (string.IsNullOrWhiteSpace(site.HttpListen) || site.HttpListen == "[\"80\"]")
            {
                if (defaults.TryGetValue("http_listen-port", out var value) && !string.IsNullOrWhiteSpace(value))
                {
                    site.HttpListen = EncodeStringList(SplitFields(value));
                }
            }
        }

        var httpsCfg = GetSubMap(settings, "https");
        if (!httpsCfg.ContainsKey("enable"))
        {
            httpsCfg["enable"] = false;
        }
        if (string.IsNullOrWhiteSpace(site.HttpsListen) && ParseBool(httpsCfg["enable"], false))
        {
            if (defaults.TryGetValue("https_listen-port", out var httpsPort) && !string.IsNullOrWhiteSpace(httpsPort))
            {
                site.HttpsListen = EncodeStringList(SplitFields(httpsPort));
            }
        }

        if (string.IsNullOrWhiteSpace(site.BalanceWay) && defaults.TryGetValue("balance_way", out var balanceWay))
        {
            site.BalanceWay = balanceWay;
        }
        if (string.IsNullOrWhiteSpace(site.BackendProtocol) && defaults.TryGetValue("backend_protocol", out var backendProtocol))
        {
            site.BackendProtocol = backendProtocol;
        }
        if ((site.CcDefaultRule ?? 0) == 0 && defaults.TryGetValue("cc_default_rule", out var ccDefault))
        {
            site.CcDefaultRule = ParseInt(ccDefault);
        }
        if ((site.DnsProviderId ?? 0) == 0 && defaults.TryGetValue("dns_provider_id", out var dnsProvider))
        {
            site.DnsProviderId = ParseInt(dnsProvider);
        }
        if (string.IsNullOrWhiteSpace(site.BlackIp) && defaults.TryGetValue("black_ip", out var blackIp))
        {
            site.BlackIp = blackIp;
        }
        if (string.IsNullOrWhiteSpace(site.WhiteIp) && defaults.TryGetValue("white_ip", out var whiteIp))
        {
            site.WhiteIp = whiteIp;
        }

        SetIfMissing(httpsCfg, "force", ParseBool(defaults.GetValueOrDefault("https_listen-force_ssl_enable"), false));
        SetIfMissing(httpsCfg, "redirect_port", defaults.GetValueOrDefault("https_listen-port"));
        SetIfMissing(httpsCfg, "hsts", ParseBool(defaults.GetValueOrDefault("https_listen-hsts"), false));
        SetIfMissing(httpsCfg, "http2", ParseBool(defaults.GetValueOrDefault("https_listen-http2"), false));
        SetIfMissing(httpsCfg, "http3", ParseBool(defaults.GetValueOrDefault("https_listen-http3"), false));
        SetIfMissing(httpsCfg, "ocsp_stapling", ParseBool(defaults.GetValueOrDefault("https_listen-ocsp_stapling"), false));
        SetIfMissing(httpsCfg, "ssl_protocols", defaults.GetValueOrDefault("https_listen-ssl_protocols"));
        SetIfMissing(httpsCfg, "ssl_ciphers", defaults.GetValueOrDefault("https_listen-ssl_ciphers"));
        SetIfMissing(httpsCfg, "ssl_prefer_server_ciphers", defaults.GetValueOrDefault("https_listen-ssl_prefer_server_ciphers"));

        var certCfg = GetSubMap(settings, "cert");
        SetIfMissing(certCfg, "type", defaults.GetValueOrDefault("cert_default_type"));
        SetIfMissing(certCfg, "dnsapi_type", defaults.GetValueOrDefault("cert_default_dnsapi_type"));
        if (defaults.TryGetValue("cert_default_dnsapi_data", out var certData) && !string.IsNullOrWhiteSpace(certData))
        {
            try
            {
                var map = JsonSerializer.Deserialize<Dictionary<string, object?>>(certData, JsonOptions);
                if (map != null)
                {
                    SetIfMissing(certCfg, "dnsapi_data", map);
                }
            }
            catch
            {
            }
        }

        var backsourceCfg = GetSubMap(settings, "backsource");
        SetIfMissing(backsourceCfg, "protocol", defaults.GetValueOrDefault("backend_protocol"));
        SetIfMissing(backsourceCfg, "http_port", defaults.GetValueOrDefault("backend_http_port"));
        SetIfMissing(backsourceCfg, "https_port", defaults.GetValueOrDefault("backend_https_port"));
        SetIfMissing(backsourceCfg, "timeout", defaults.GetValueOrDefault("proxy_timeout"));
        SetIfMissing(backsourceCfg, "connect_timeout", defaults.GetValueOrDefault("connect_timeout"));

        var cacheCfg = GetSubMap(settings, "cache");
        if (!cacheCfg.ContainsKey("enable"))
        {
            var raw = defaults.GetValueOrDefault("proxy_cache")?.Trim() ?? string.Empty;
            cacheCfg["enable"] = !string.IsNullOrWhiteSpace(raw) && raw != "[]";
        }
        if (!cacheCfg.ContainsKey("rules") && defaults.TryGetValue("proxy_cache", out var cacheRules))
        {
            cacheCfg["rules"] = ParseCacheRules(cacheRules);
        }

        var securityCfg = GetSubMap(settings, "security");
        SetIfMissing(securityCfg, "default_rule", site.CcDefaultRule ?? 0);
        if (defaults.TryGetValue("security_bot", out var botAction))
        {
            SetIfMissing(securityCfg, "crawlers_action", botAction);
        }
        if (defaults.TryGetValue("black_ip", out var secBlack))
        {
            SetIfMissing(securityCfg, "blacklist", SplitFields(secBlack));
        }
        if (defaults.TryGetValue("white_ip", out var secWhite))
        {
            SetIfMissing(securityCfg, "whitelist", SplitFields(secWhite));
        }
        if (defaults.TryGetValue("security_black_time", out var blackTime))
        {
            var parsed = ParseInt(blackTime);
            SetIfMissing(securityCfg, "ip_black_timeout", parsed);
            SetIfMissing(securityCfg, "black_time_mode", "custom");
            SetIfMissing(securityCfg, "black_time_custom", parsed);
        }
        if (defaults.TryGetValue("security_white_time", out var whiteTime))
        {
            var parsed = ParseInt(whiteTime);
            SetIfMissing(securityCfg, "ip_white_timeout", parsed);
            SetIfMissing(securityCfg, "white_time_mode", "custom");
            SetIfMissing(securityCfg, "white_time_custom", parsed);
        }
        if (defaults.TryGetValue("security_shield_proxy", out var shieldProxy))
        {
            SetIfMissing(securityCfg, "block_transparent_proxy", ParseBool(shieldProxy, false));
        }
        if (defaults.TryGetValue("block_region", out var blockRegion))
        {
            if (string.IsNullOrWhiteSpace(site.BlockRegion))
            {
                site.BlockRegion = blockRegion;
            }
            if (!securityCfg.ContainsKey("region_block"))
            {
                securityCfg["region_block"] = blockRegion == "none" ? new List<string>() : SplitByComma(blockRegion);
            }
        }

        var advCfg = GetSubMap(settings, "advanced");
        if (defaults.TryGetValue("gzip_enable", out var gzipEnable))
        {
            SetIfMissing(advCfg, "gzip", ParseBool(gzipEnable, false));
        }
        if (defaults.TryGetValue("gzip_types", out var gzipTypes))
        {
            SetIfMissing(advCfg, "gzip_types", gzipTypes);
        }
        if (defaults.TryGetValue("websocket_enable", out var websocketEnable))
        {
            SetIfMissing(advCfg, "websocket", ParseBool(websocketEnable, false));
        }
        if (defaults.TryGetValue("ipv6_enable", out var ipv6Enable))
        {
            SetIfMissing(advCfg, "ipv6", ParseBool(ipv6Enable, false));
        }
        if (defaults.TryGetValue("range", out var rangeEnable))
        {
            SetIfMissing(advCfg, "range", ParseBool(rangeEnable, false));
        }
        if (defaults.TryGetValue("proxy_http_version", out var httpVersion))
        {
            SetIfMissing(advCfg, "proxy_http_version", httpVersion);
        }
        if (defaults.TryGetValue("proxy_ssl_protocols", out var sslProtocols))
        {
            SetIfMissing(advCfg, "proxy_ssl_protocols", sslProtocols);
        }
        if (defaults.TryGetValue("ups_keepalive", out var keepalive))
        {
            SetIfMissing(advCfg, "ups_keepalive", ParseBool(keepalive, false));
        }
        if (defaults.TryGetValue("ups_keepalive_conn", out var keepaliveConn))
        {
            SetIfMissing(advCfg, "ups_keepalive_conn", ParseInt(keepaliveConn));
        }
        if (defaults.TryGetValue("ups_keepalive_timeout", out var keepaliveTimeout))
        {
            SetIfMissing(advCfg, "ups_keepalive_timeout", ParseInt(keepaliveTimeout));
        }
        if (defaults.TryGetValue("post_size_limit", out var postSize))
        {
            SetIfMissing(advCfg, "body_limit", ParseInt(postSize));
            SetIfMissing(advCfg, "body_limit_unit", "kb");
        }
        if (defaults.TryGetValue("log_request_header", out var logRequestHeader))
        {
            SetIfMissing(advCfg, "log_request_header", ParseBool(logRequestHeader, false));
        }
        if (defaults.TryGetValue("log_response_header", out var logResponseHeader))
        {
            SetIfMissing(advCfg, "log_response_header", ParseBool(logResponseHeader, false));
        }
        if (defaults.TryGetValue("log_request_body", out var logRequestBody))
        {
            SetIfMissing(advCfg, "log_request_body", ParseBool(logRequestBody, false));
        }
        if (defaults.TryGetValue("realtime_send", out var realtimeSend))
        {
            SetIfMissing(advCfg, "realtime_send", ParseBool(realtimeSend, false));
        }
        if (defaults.TryGetValue("realtime_return", out var realtimeReturn))
        {
            SetIfMissing(advCfg, "realtime_return", ParseBool(realtimeReturn, false));
        }
        if (defaults.TryGetValue("origin_headers", out var originHeaders))
        {
            SetIfMissing(advCfg, "origin_headers", ParseHeaderList(originHeaders));
        }
    }

    private static void EnsureSitePersistenceDefaults(Site site)
    {
        var now = DateTime.Now;

        site.Uid ??= 0;
        site.UserPackage ??= 0;
        site.RegionId ??= null;
        site.NodeGroupId ??= null;
        site.BackupNodeGroup ??= null;
        site.EnableBackupGroup ??= false;
        site.DnsProviderId ??= null;

        site.PlatformDnsRecordId ??= string.Empty;
        site.UserDnsRecordId ??= string.Empty;
        site.CnameDomain ??= string.Empty;
        site.CnameHostname2 ??= string.Empty;
        site.CnameMode ??= "site";
        site.CnameHostname ??= string.Empty;
        site.Domain ??= string.Empty;
        site.HttpListen ??= string.Empty;
        site.HttpsListen ??= string.Empty;

        site.BalanceWay ??= "round_robin";
        site.Backend ??= string.Empty;
        site.BackendProtocol ??= "http";
        site.BackendHttpsPort ??= "443";
        site.BackendHttpPort ??= "80";
        site.ProxyTimeout ??= "60";
        site.BackendPortMapping ??= false;
        site.HealthCheck ??= string.Empty;
        site.UpsKeepalive ??= false;
        site.UpsKeepaliveConn ??= 0;
        site.UpsKeepaliveTimeout ??= 0;
        site.ProxyHttpVersion ??= string.Empty;
        site.ProxySslProtocols ??= string.Empty;
        site.BackendHost ??= string.Empty;
        site.Range ??= false;
        site.ProxyCache ??= string.Empty;

        site.CcDefaultRule ??= null;
        site.CcSwitch ??= string.Empty;
        site.ExtraCcRule ??= string.Empty;
        site.BlockProxy ??= false;
        site.BlockRegion ??= string.Empty;
        site.BlackIp ??= string.Empty;
        site.WhiteIp ??= string.Empty;
        site.SpiderAllow ??= string.Empty;
        site.Acl ??= null;
        site.Hotlink ??= string.Empty;
        site.Cors ??= string.Empty;
        site.RespHeader ??= string.Empty;
        site.ReqHeader ??= string.Empty;
        site.Page404 ??= string.Empty;
        site.Page50x ??= string.Empty;
        site.UrlRewrite ??= string.Empty;
        site.GzipEnable ??= false;
        site.GzipTypes ??= string.Empty;
        site.WebsocketEnable ??= true;
        site.AcmeProxyToOrgin ??= false;
        site.PostSizeLimit ??= 0;

        site.CreateAt ??= now;
        site.UpdateAt ??= now;
        site.Version ??= 1;
        site.Enable ??= true;
        site.RecordId ??= Guid.NewGuid().ToString("N")[..8];
        site.State ??= "running";
    }

    private static void ApplySiteDefaultsScopedOverrides(Dictionary<string, object?> settings, Dictionary<string, string> defaults)
    {
        if (defaults == null || defaults.Count == 0)
        {
            return;
        }

        if (defaults.TryGetValue("gzip_enable", out var gzipEnable))
        {
            var advCfg = GetSubMap(settings, "advanced");
            advCfg["gzip"] = ParseBool(gzipEnable, false);
        }
        if (defaults.TryGetValue("https_listen-ssl_ciphers", out var sslCiphers))
        {
            var httpsCfg = GetSubMap(settings, "https");
            httpsCfg["ssl_ciphers"] = sslCiphers;
        }
        if (defaults.TryGetValue("proxy_cache", out var proxyCache))
        {
            var cacheCfg = GetSubMap(settings, "cache");
            var raw = proxyCache.Trim();
            cacheCfg["enable"] = !string.IsNullOrWhiteSpace(raw) && raw != "[]";
            cacheCfg["rules"] = ParseCacheRules(proxyCache);
        }
    }

    private static Dictionary<string, object?> MergeSettingsMaps(Dictionary<string, object?>? dst, Dictionary<string, object?>? src)
    {
        dst ??= new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        if (src == null)
        {
            return dst;
        }

        foreach (var (key, value) in src)
        {
            if (value == null)
            {
                dst[key] = null;
                continue;
            }

            if (value is Dictionary<string, object?> srcMap)
            {
                if (dst.TryGetValue(key, out var dstValue) && dstValue is Dictionary<string, object?> dstMap)
                {
                    dst[key] = MergeSettingsMaps(dstMap, srcMap);
                }
                else
                {
                    dst[key] = MergeSettingsMaps(null, srcMap);
                }
                continue;
            }

            dst[key] = value;
        }

        return dst;
    }

    private static Dictionary<string, object?> GetSubMap(Dictionary<string, object?> root, string key)
    {
        if (root.TryGetValue(key, out var value))
        {
            var map = AsDictionary(value);
            if (map != null)
            {
                root[key] = map;
                return map;
            }
        }

        var created = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        root[key] = created;
        return created;
    }

    private static Dictionary<string, object?>? AsDictionary(object? raw)
    {
        if (raw is Dictionary<string, object?> dict)
        {
            return dict;
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
        return element.ValueKind switch
        {
            JsonValueKind.Object => ToDictionary(element),
            JsonValueKind.Array => element.EnumerateArray().Select(ConvertJsonElement).ToList(),
            JsonValueKind.String => element.GetString(),
            JsonValueKind.Number => element.TryGetInt64(out var l) ? l : element.TryGetDecimal(out var d) ? d : element.ToString(),
            JsonValueKind.True => true,
            JsonValueKind.False => false,
            _ => null
        };
    }

    private static string ParseString(object? raw)
    {
        if (raw == null)
        {
            return string.Empty;
        }
        if (raw is string text)
        {
            return text;
        }
        if (raw is JsonElement element)
        {
            return element.ValueKind == JsonValueKind.String ? element.GetString() ?? string.Empty : element.ToString();
        }
        return raw.ToString() ?? string.Empty;
    }

    private static bool ParseBool(object? raw, bool fallback)
    {
        if (raw == null)
        {
            return fallback;
        }
        if (raw is bool b)
        {
            return b;
        }
        if (raw is JsonElement element)
        {
            if (element.ValueKind == JsonValueKind.True)
            {
                return true;
            }
            if (element.ValueKind == JsonValueKind.False)
            {
                return false;
            }
            if (element.ValueKind == JsonValueKind.String && bool.TryParse(element.GetString(), out var parsed))
            {
                return parsed;
            }
        }
        var str = raw.ToString()?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(str))
        {
            return fallback;
        }
        return str is "1" or "true" or "yes" or "on";
    }

    private static int ParseInt(string? raw)
    {
        return int.TryParse(raw?.Trim(), out var parsed) ? parsed : 0;
    }

    private static int ParseInt(object? raw)
    {
        if (raw is int i)
        {
            return i;
        }
        if (raw is long l)
        {
            return l > int.MaxValue ? int.MaxValue : (int)l;
        }
        if (raw is JsonElement element)
        {
            if (element.ValueKind == JsonValueKind.Number && element.TryGetInt32(out var number))
            {
                return number;
            }
            if (element.ValueKind == JsonValueKind.String && int.TryParse(element.GetString(), out var parsed))
            {
                return parsed;
            }
        }
        return ParseInt(raw?.ToString());
    }

    private static void SetIfMissing(Dictionary<string, object?> target, string key, object? value)
    {
        if (value == null)
        {
            return;
        }
        if (!target.ContainsKey(key))
        {
            target[key] = value;
        }
    }

    private static List<Dictionary<string, object?>> ParseCacheRules(string raw)
    {
        var trimmed = raw?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(trimmed))
        {
            return new List<Dictionary<string, object?>>();
        }
        try
        {
            var rules = JsonSerializer.Deserialize<List<Dictionary<string, object?>>>(trimmed, JsonOptions);
            return rules ?? new List<Dictionary<string, object?>>();
        }
        catch
        {
            return new List<Dictionary<string, object?>>();
        }
    }

    private static List<Dictionary<string, string>> ParseHeaderList(string raw)
    {
        var trimmed = raw?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(trimmed))
        {
            return new List<Dictionary<string, string>>();
        }
        try
        {
            var headers = JsonSerializer.Deserialize<List<Dictionary<string, string>>>(trimmed, JsonOptions);
            return headers ?? new List<Dictionary<string, string>>();
        }
        catch
        {
            return new List<Dictionary<string, string>>();
        }
    }

    private static Dictionary<string, object?> DeserializeSettings(string raw)
    {
        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
            }

            var result = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
            foreach (var prop in doc.RootElement.EnumerateObject())
            {
                result[prop.Name] = ConvertJsonElement(prop.Value);
            }
            return result;
        }
        catch
        {
            return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        }
    }

    private static (long RuleId, bool Has) ExtractCcDefaultRule(Dictionary<string, object?>? settings)
    {
        if (settings == null)
        {
            return (0, false);
        }

        if (!settings.TryGetValue("security", out var securityRaw))
        {
            return (0, false);
        }

        var security = AsDictionary(securityRaw);
        if (security == null || !security.TryGetValue("default_rule", out var raw))
        {
            return (0, false);
        }

        var value = ParseInt(raw);
        return (value, true);
    }

    private static void SetCcDefaultRuleInSettings(Dictionary<string, object?> settings, long ruleId)
    {
        var security = GetSubMap(settings, "security");
        security["default_rule"] = ruleId;
    }

    private static (List<string> List, bool Has) ExtractSecurityIpList(Dictionary<string, object?>? settings, string key)
    {
        if (settings == null)
        {
            return (new List<string>(), false);
        }

        if (!settings.TryGetValue("security", out var securityRaw))
        {
            return (new List<string>(), false);
        }

        var security = AsDictionary(securityRaw);
        if (security == null || !security.TryGetValue(key, out var raw))
        {
            return (new List<string>(), false);
        }

        return (ParseStringList(raw), true);
    }

    private static void SetSecurityIpList(Dictionary<string, object?> settings, string key, IReadOnlyList<string> list)
    {
        var security = GetSubMap(settings, "security");
        security[key] = list.ToList();
    }

    private static void MergeSecurityIpList(Dictionary<string, object?> settings, string key, string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return;
        }

        var list = DecodeStringList(raw);
        if (list.Count == 0)
        {
            return;
        }

        var security = GetSubMap(settings, "security");
        if (security.TryGetValue(key, out var existingRaw))
        {
            var existing = ParseStringList(existingRaw);
            if (existing.Count > 0)
            {
                list.AddRange(existing);
            }
        }

        security[key] = NormalizeStringList(list);
    }

    private static List<string> ParseStringList(object? raw)
    {
        if (raw == null)
        {
            return new List<string>();
        }
        if (raw is List<string> list)
        {
            return NormalizeStringList(list);
        }
        if (raw is JsonElement element)
        {
            if (element.ValueKind == JsonValueKind.Array)
            {
                var values = new List<string>();
                foreach (var item in element.EnumerateArray())
                {
                    if (item.ValueKind == JsonValueKind.String)
                    {
                        values.Add(item.GetString() ?? string.Empty);
                    }
                    else
                    {
                        values.Add(item.ToString());
                    }
                }
                return NormalizeStringList(values);
            }
            if (element.ValueKind == JsonValueKind.String)
            {
                return SplitFields(element.GetString() ?? string.Empty);
            }
        }
        if (raw is string text)
        {
            return SplitFields(text);
        }
        if (raw is IEnumerable<object?> items)
        {
            var listItems = new List<string>();
            foreach (var item in items)
            {
                listItems.Add(item?.ToString() ?? string.Empty);
            }
            return NormalizeStringList(listItems);
        }
        return new List<string>();
    }

    private async Task<(int DomainLimit, int MainDomainLimit)?> LoadDomainLimitsAsync(long userPackageId)
    {
        if (userPackageId <= 0)
        {
            return null;
        }

        var pack = await _db.Queryable<UserPackage>().Where(p => p.Id == userPackageId).FirstAsync();
        if (pack == null)
        {
            return null;
        }

        var totalLimit = pack.Domain ?? 0;
        var mainLimit = pack.MainDomainLimit ?? 0;

        if (mainLimit <= 0)
        {
            var cfg = await _db.Queryable<Config>()
                .Where(c => c.Type == "user_package_config" && c.ScopeName == "user_package" && c.ScopeId == userPackageId && c.Name == "main_domain_limit")
                .FirstAsync();

            if (cfg != null)
            {
                mainLimit = ParseInt(cfg.Value);
            }
        }

        return (totalLimit, mainLimit);
    }

    private async Task<(HashSet<string> DomainSet, HashSet<string> MainSet)> LoadUserDomainSetsAsync(long userId, long? excludeSiteId)
    {
        var domainSet = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var mainSet = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        if (userId <= 0)
        {
            return (domainSet, mainSet);
        }

        var query = _db.Queryable<Site>().Where(s => s.Uid == (int)userId);
        if (excludeSiteId is > 0)
        {
            query = query.Where(s => s.Id != excludeSiteId.Value);
        }

        var sites = await query.ToListAsync();
        foreach (var site in sites)
        {
            AddDomains(domainSet, mainSet, DomainParser.ParseDomains(site.Domain));
        }

        return (domainSet, mainSet);
    }

    private static void AddDomains(HashSet<string> domainSet, HashSet<string> mainSet, IReadOnlyList<string> domains)
    {
        foreach (var domain in domains)
        {
            var normalized = DomainParser.NormalizeDomain(domain);
            if (string.IsNullOrWhiteSpace(normalized))
            {
                continue;
            }

            domainSet.Add(normalized);
            var mainKey = MainDomainKey(normalized);
            if (!string.IsNullOrWhiteSpace(mainKey))
            {
                mainSet.Add(mainKey);
            }
        }
    }

    private static string MainDomainKey(string domain)
    {
        if (string.IsNullOrWhiteSpace(domain))
        {
            return string.Empty;
        }

        if (IPAddress.TryParse(domain, out _))
        {
            return domain;
        }

        var parts = domain.Split('.', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length < 2)
        {
            return domain;
        }

        return $"{parts[^2]}.{parts[^1]}";
    }

    private static string ComputeSiteCnameHostname(Site site, UserPackage pkg, string? overrideMode, string? overrideDomain)
    {
        var newCnameHostname = (site.CnameHostname ?? string.Empty).Trim();

        var siteMode = (site.CnameMode ?? string.Empty).Trim();
        if (!string.IsNullOrWhiteSpace(overrideMode))
        {
            siteMode = overrideMode.Trim();
        }

        var cnameDomain = (site.CnameDomain ?? string.Empty).Trim();
        if (!string.IsNullOrWhiteSpace(overrideDomain))
        {
            cnameDomain = overrideDomain.Trim();
        }

        var pkgDomain = (pkg.CnameDomain ?? string.Empty).Trim();
        var pkgMode = (pkg.CnameMode ?? string.Empty).Trim();

        if (string.Equals(siteMode, "package", StringComparison.OrdinalIgnoreCase) ||
            (string.IsNullOrWhiteSpace(siteMode) && string.Equals(pkgMode, "package", StringComparison.OrdinalIgnoreCase)))
        {
            var pkgHost = (pkg.CnameHostname ?? string.Empty).Trim();
            if (!string.IsNullOrWhiteSpace(pkgHost))
            {
                newCnameHostname = pkgHost;
                if (!string.IsNullOrWhiteSpace(pkgDomain))
                {
                    newCnameHostname += "." + pkgDomain;
                }
                else if (!string.IsNullOrWhiteSpace(cnameDomain))
                {
                    newCnameHostname += "." + cnameDomain;
                }
                else
                {
                    newCnameHostname += "." + DefaultCnameDomain;
                }
            }

            return newCnameHostname;
        }

        var effectiveDomain = cnameDomain;
        if (string.IsNullOrWhiteSpace(effectiveDomain))
        {
            effectiveDomain = pkgDomain;
        }
        if (string.IsNullOrWhiteSpace(effectiveDomain))
        {
            effectiveDomain = DefaultCnameDomain;
        }

        var domains = DomainParser.ParseDomains(site.Domain);
        if (domains.Count > 0 && !string.IsNullOrWhiteSpace(effectiveDomain))
        {
            return domains[0] + "." + effectiveDomain;
        }

        return newCnameHostname;
    }

    private async Task<bool> RefreshSiteCnameHostnameAsync(Site site, string? overrideMode, string? overrideDomain)
    {
        if (site.UserPackage is null or <= 0)
        {
            return false;
        }
        if (overrideMode == null && overrideDomain == null && !string.IsNullOrWhiteSpace(site.CnameHostname))
        {
            return false;
        }

        var pkg = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == site.UserPackage.Value)
            .Select(p => new { p.CnameMode, p.CnameHostname, p.CnameDomain })
            .FirstAsync();
        if (pkg == null)
        {
            return false;
        }

        var updateDomain = false;
        if (overrideDomain == null && string.IsNullOrWhiteSpace(site.CnameDomain) && !string.IsNullOrWhiteSpace(pkg.CnameDomain))
        {
            site.CnameDomain = pkg.CnameDomain?.Trim();
            updateDomain = true;
        }

        var pkgEntity = new UserPackage { CnameMode = pkg.CnameMode, CnameHostname = pkg.CnameHostname, CnameDomain = pkg.CnameDomain };
        var newCnameHostname = ComputeSiteCnameHostname(site, pkgEntity, overrideMode, overrideDomain);
        if (string.IsNullOrWhiteSpace(newCnameHostname) || (newCnameHostname == site.CnameHostname && !updateDomain))
        {
            return false;
        }

        var updates = new Site { CnameHostname = newCnameHostname };
        if (updateDomain)
        {
            updates.CnameDomain = site.CnameDomain;
        }

        await _db.Updateable(updates)
            .UpdateColumns(s => new { s.CnameHostname, s.CnameDomain })
            .Where(s => s.Id == site.Id)
            .ExecuteCommandAsync();

        site.CnameHostname = newCnameHostname;
        return true;
    }

    private static bool ShouldResyncSiteCname(Site oldSite, Site newSite)
    {
        if (oldSite.UserPackage != newSite.UserPackage)
        {
            return true;
        }
        if (!string.Equals(oldSite.CnameDomain?.Trim(), newSite.CnameDomain?.Trim(), StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (!string.Equals(oldSite.CnameMode?.Trim(), newSite.CnameMode?.Trim(), StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (!string.Equals(oldSite.CnameHostname?.Trim(), newSite.CnameHostname?.Trim(), StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (oldSite.NodeGroupId != newSite.NodeGroupId)
        {
            return true;
        }
        if (oldSite.BackupNodeGroup != newSite.BackupNodeGroup)
        {
            return true;
        }
        if (oldSite.EnableBackupGroup != newSite.EnableBackupGroup)
        {
            return true;
        }
        return false;
    }

    private async Task ResyncSiteCnameForSiteAsync(Site site)
    {
        if (site.UserPackage is null or <= 0)
        {
            return;
        }

        if (!await ShouldSyncSiteCnameAsync(site))
        {
            return;
        }

        var groupId = await ResolveGroupIdFromSiteAsync(site);
        if (groupId > 0)
        {
            await ResyncGroupLineCnamesAsync(groupId);
        }

        var backupGroup = site.BackupNodeGroup ?? 0;
        var enableBackup = site.EnableBackupGroup ?? false;
        if (!enableBackup && site.UserPackage is > 0)
        {
            var pkg = await _db.Queryable<UserPackage>()
                .Where(p => p.Id == site.UserPackage)
                .Select(p => new { p.BackupNodeGroup, p.EnableBackupGroup })
                .FirstAsync();
            if (pkg != null)
            {
                if (backupGroup == 0)
                {
                    backupGroup = pkg.BackupNodeGroup ?? 0;
                }
                enableBackup = pkg.EnableBackupGroup ?? false;
            }
        }

        if (enableBackup && backupGroup > 0)
        {
            await ResyncGroupLineCnamesAsync(backupGroup);
        }
    }

    private async Task<bool> ShouldSyncSiteCnameAsync(Site site)
    {
        if (site.UserPackage is null or <= 0)
        {
            return false;
        }

        var mode = site.CnameMode?.Trim();
        if (!string.IsNullOrWhiteSpace(mode))
        {
            return !string.Equals(mode, "package", StringComparison.OrdinalIgnoreCase);
        }

        var pkg = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == site.UserPackage)
            .Select(p => new { p.CnameMode })
            .FirstAsync();

        if (pkg == null)
        {
            return true;
        }

        return !string.Equals(pkg.CnameMode?.Trim(), "package", StringComparison.OrdinalIgnoreCase);
    }

    private async Task<long> ResolveGroupIdFromSiteAsync(Site site)
    {
        if (site.NodeGroupId is > 0)
        {
            return site.NodeGroupId.Value;
        }

        if (site.UserPackage is null or <= 0)
        {
            return 0;
        }

        var pkg = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == site.UserPackage)
            .Select(p => new { p.NodeGroupId })
            .FirstAsync();

        return pkg?.NodeGroupId ?? 0;
    }

    private async Task ResyncGroupLineCnamesAsync(long groupId)
    {
        if (groupId <= 0)
        {
            return;
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId == groupId)
            .Select(l => new { l.LineId, l.LineName })
            .ToListAsync();

        var lineMap = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var line in lines)
        {
            var lineId = line.LineId?.Trim();
            if (string.IsNullOrWhiteSpace(lineId))
            {
                lineId = "default";
            }

            var lineName = line.LineName?.Trim();
            if (string.IsNullOrWhiteSpace(lineName))
            {
                lineName = lineId;
            }

            if (!lineMap.ContainsKey(lineId))
            {
                lineMap[lineId] = lineName;
            }
        }

        foreach (var pair in lineMap)
        {
            await _dnsSyncService.SyncPackageCnameForLineChangeAsync(groupId, pair.Key, pair.Value, Array.Empty<long>(), "resync");
        }
    }

    private Task SaveSiteSettingsAsync(long siteId, Dictionary<string, object?> settings)
        => _siteSettingsStore.SaveSettingsAsync(siteId, settings);

    private Task UpsertSiteTypeMetaAsync(long siteId, string siteType)
        => _siteSettingsStore.SaveSiteTypeAsync(siteId, siteType);
}
