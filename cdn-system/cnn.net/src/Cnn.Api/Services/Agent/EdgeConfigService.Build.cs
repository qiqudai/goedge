
using System.Text.Json;
using Cnn.Common.Contracts.Agent;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;
using StreamEntity = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Agent;

public sealed partial class EdgeConfigService
{
    private const string PlaceholderCertPath = "/usr/local/goedge/edge/configs/placeholder.crt";
    private const string PlaceholderKeyPath = "/usr/local/goedge/edge/configs/placeholder.key";

    private sealed record L2Target(long NodeId, string Ip);

    private sealed record AdvancedConfig(
        bool? Gzip,
        string? GzipTypes,
        bool? Websocket,
        bool? RangeEnabled,
        string? ProxyHttpVersion,
        string? ProxySslProtocols,
        long BodyLimit,
        bool? Keepalive,
        int KeepaliveConn,
        int KeepaliveTimeout);

    private sealed record HealthRuntimeConfig(
        bool? ActiveEnabled,
        string? ActivePath,
        string? ActiveInterval,
        string? ActiveTimeout,
        string? ActivePolicy,
        int? ActiveThreshold,
        bool? PassiveEnabled,
        string? PassiveReactivation,
        string? PassivePolicy,
        double? PassiveRateLimit,
        string? AvailablePolicy);

    private async Task PopulateNodeConfigAsync(
        EdgeConfigDto config,
        Node node,
        Dictionary<string, string> systemCfg,
        CancellationToken cancellationToken)
    {
        config.Nginx = await LoadNginxConfigAsync(cancellationToken);

        var expireCloseEnabled = true;
        if (systemCfg.TryGetValue("package_expire_close_site", out var expireFlag))
        {
            expireCloseEnabled = ParseBoolFlag(expireFlag, true);
        }

        var groupIds = await LoadNodeGroupIdsAsync(node);
        if (groupIds.Count == 0)
        {
            return;
        }

        var groupL2Config = await LoadNodeGroupL2ConfigAsync(groupIds);
        var l2TargetsByGroup = new Dictionary<long, List<L2Target>>();
        var l2UpstreamKeyByGroup = new Dictionary<long, string>();
        if ((node.Level ?? 0) == 1)
        {
            l2TargetsByGroup = await LoadL2TargetsByGroupAsync(groupIds);
            foreach (var pair in l2TargetsByGroup)
            {
                if (pair.Value.Count == 0)
                {
                    continue;
                }

                var upstreamKey = $"l2_upstream_{pair.Key}";
                l2UpstreamKeyByGroup[pair.Key] = upstreamKey;
                var upstreamTargets = new List<EdgeUpstreamTargetDto>();
                foreach (var target in pair.Value)
                {
                    if (string.IsNullOrWhiteSpace(target.Ip))
                    {
                        continue;
                    }

                    upstreamTargets.Add(new EdgeUpstreamTargetDto
                    {
                        Addr = target.Ip,
                        Weight = 1,
                        NodeId = target.NodeId
                    });
                }

                if (upstreamTargets.Count > 0)
                {
                    config.Upstreams.Add(new EdgeUpstreamDto
                    {
                        Id = upstreamKey,
                        Targets = upstreamTargets
                    });
                }
            }
        }

        var sites = await _db.Queryable<Site>()
            .Where(s => s.NodeGroupId.HasValue && groupIds.Contains(s.NodeGroupId.Value))
            .ToListAsync();

        var nodeGroupCounts = await LoadNodeGroupCountsAsync(groupIds);
        var userPackageMap = await LoadUserPackageMapAsync(sites);
        var domainCountByUserGroup = BuildDomainCountByUserGroup(sites);
        var certs = await _db.Queryable<Cert>().Where(c => c.Enable == true).ToListAsync();
        DecryptCertKeys(certs);
        var (siteDefaultsGlobal, siteDefaultsByUser) = await LoadSiteDefaultConfigAsync(sites);
        var siteSettingsMap = await LoadSiteSettingsMapAsync(sites, cancellationToken);

        var now = DateTime.Now;
        foreach (var site in sites)
        {
            var domains = SplitDomainList(site.Domain);
            if (domains.Count == 0)
            {
                continue;
            }

            var defaults = ResolveSiteDefaults(site.Uid, siteDefaultsGlobal, siteDefaultsByUser);
            var settings = GetSiteSettings(siteSettingsMap, site.Id);

            var status = ResolveSiteStatus(site, userPackageMap, expireCloseEnabled, now);
            var httpListen = SplitPortList(site.HttpListen);
            var httpsListen = SplitPortList(site.HttpsListen);
            var httpEnable = ResolveHttpEnable(settings) ?? true;
            if (httpListen.Count == 0 && httpEnable)
            {
                var defaultHttpPort = GetDefaultValue(defaults, "http_listen-port");
                if (!string.IsNullOrWhiteSpace(defaultHttpPort))
                {
                    httpListen.Add(defaultHttpPort.Trim());
                }
            }
            var httpsConfig = ExtractHttpsConfig(settings);
            var httpsEnable = httpsConfig.Enable ?? false;
            if (httpsListen.Count == 0 && httpsEnable)
            {
                var defaultHttpsPort = GetDefaultValue(defaults, "https_listen-port");
                if (!string.IsNullOrWhiteSpace(defaultHttpsPort))
                {
                    httpsListen.Add(defaultHttpsPort.Trim());
                }
            }
            var hasHttps = httpsListen.Count > 0;

            var backsource = GetSettingsMap(settings, "backsource");
            var settingsProtocol = ParseStringSetting(backsource, "protocol");
            var settingsHttpPort = ParseStringSetting(backsource, "http_port");
            var settingsHttpsPort = ParseStringSetting(backsource, "https_port");

            var originProtocol = string.IsNullOrWhiteSpace(site.BackendProtocol)
                ? GetDefaultValue(defaults, "backend_protocol") ?? string.Empty
                : site.BackendProtocol.Trim();
            if (string.IsNullOrWhiteSpace(originProtocol))
            {
                originProtocol = settingsProtocol?.Trim() ?? string.Empty;
            }
            if (string.IsNullOrWhiteSpace(originProtocol))
            {
                originProtocol = "follow";
            }
            var originHttpPort = settingsHttpPort;
            if (string.IsNullOrWhiteSpace(originHttpPort))
            {
                originHttpPort = site.BackendHttpPort?.Trim();
                if (string.IsNullOrWhiteSpace(originHttpPort))
                {
                    originHttpPort = GetDefaultValue(defaults, "backend_http_port")?.Trim() ?? string.Empty;
                }
            }
            var originHttpsPort = settingsHttpsPort;
            if (string.IsNullOrWhiteSpace(originHttpsPort))
            {
                originHttpsPort = site.BackendHttpsPort?.Trim();
                if (string.IsNullOrWhiteSpace(originHttpsPort))
                {
                    originHttpsPort = GetDefaultValue(defaults, "backend_https_port")?.Trim() ?? string.Empty;
                }
            }

            var l2Enabled = false;
            var l2UpstreamKey = string.Empty;
            if ((node.Level ?? 0) == 1)
            {
                var groupId = site.NodeGroupId ?? 0;
                var groupConfig = groupL2Config.TryGetValue(groupId, out var value) ? value : string.Empty;
                var packageEnabled = true;
                var l2Mode = ResolveL2Mode(settings) ?? string.Empty;
                l2Enabled = ResolveL2Enabled(l2Mode, groupConfig, packageEnabled);
                if (l2Enabled)
                {
                    l2UpstreamKey = l2UpstreamKeyByGroup.TryGetValue(groupId, out var key) ? key : string.Empty;
                    if (string.IsNullOrWhiteSpace(l2UpstreamKey))
                    {
                        l2Enabled = false;
                    }
                }
            }

            var l2HttpPort = l2Enabled ? ResolveListenPort(httpListen, "80") : null;
            var l2HttpsPort = l2Enabled ? ResolveListenPort(httpsListen, string.Empty) : null;

            var upstreamKey = $"upstream_{site.Id}";
            var targets = BuildUpstreamTargetsFromSettings(settings, originProtocol, originHttpPort, originHttpsPort);
            if (targets.Count == 0)
            {
                targets = BuildUpstreamTargets(site, originProtocol, originHttpPort, originHttpsPort);
            }
            if (targets.Count > 0)
            {
                config.Upstreams.Add(new EdgeUpstreamDto
                {
                    Id = upstreamKey,
                    Targets = targets
                });
            }

            var balanceWay = string.IsNullOrWhiteSpace(site.BalanceWay)
                ? GetDefaultValue(defaults, "balance_way")
                : site.BalanceWay;
            var policy = MapBalancePolicy(balanceWay);
            var headers = BuildHeaderMapFromSettings(settings, site) ?? BuildRequestHeaders(site);
            var responseHeaders = BuildResponseHeaderMapFromSettings(settings) ?? BuildResponseHeaders(site);
            var siteType = ParseStringSetting(settings, "site_type")?.Trim().ToLowerInvariant();
            if (string.IsNullOrWhiteSpace(siteType))
            {
                siteType = "website";
            }

            var aclId = ResolveAclId(settings, site.Acl);
            var (aclDefault, aclRules) = await BuildAclForSiteAsync(aclId);
            var blockRegionRaw = string.IsNullOrWhiteSpace(site.BlockRegion)
                ? GetDefaultValue(defaults, "block_region")
                : site.BlockRegion;
            var regionBlock = ExtractRegionBlock(settings, blockRegionRaw);
            var hotlink = ExtractHotlinkConfig(settings, site.Hotlink);
            var cors = ExtractCorsConfig(settings, site.Cors);
            var cookie = ExtractCookieConfig(settings);
            var blockTransparentProxy = ExtractBlockTransparentProxy(
                settings,
                site.BlockProxy ?? ParseDefaultBool(defaults, "security_shield_proxy"));
            var crawlerAction = ExtractCrawlerAction(settings, site.SpiderAllow, defaults);
            var (guardPassTtl, guardBlockTtl) = ExtractGuardTtls(settings, defaults);
            var urlRedirects = ExtractUrlRedirects(settings, site.UrlRewrite);
            var originConditions = ExtractOriginConditions(settings) ?? ParseOriginConditions(site);
            var ccRuleId = ResolveCcRuleId(site, defaults, settings);

            var cacheRaw = string.IsNullOrWhiteSpace(site.ProxyCache)
                ? GetDefaultValue(defaults, "proxy_cache")
                : site.ProxyCache;
            var cacheConfig = ExtractCacheConfig(settings, cacheRaw);
            var blackIps = ParseIpList(string.IsNullOrWhiteSpace(site.BlackIp) ? GetDefaultValue(defaults, "black_ip") : site.BlackIp);
            var whiteIps = ParseIpList(string.IsNullOrWhiteSpace(site.WhiteIp) ? GetDefaultValue(defaults, "white_ip") : site.WhiteIp);
            var timeoutRaw = string.IsNullOrWhiteSpace(site.ProxyTimeout)
                ? GetDefaultValue(defaults, "proxy_timeout")
                : site.ProxyTimeout;
            var (proxyConnectTimeout, proxyReadTimeout, proxySendTimeout) = ExtractProxyTimeouts(settings, timeoutRaw);
            var adv = BuildAdvancedConfig(site, defaults, settings);
            var health = BuildHealthRuntimeConfig(settings, defaults);

            foreach (var domain in domains)
            {
                var limitRate = CalcDomainLimitRate(site, userPackageMap);
                var connLimit = CalcDomainConnLimit(site, userPackageMap, domainCountByUserGroup, nodeGroupCounts);

                var domainConf = new EdgeDomainDto
                {
                    Name = domain,
                    UpstreamKey = upstreamKey,
                    L2UpstreamKey = l2Enabled ? l2UpstreamKey : null,
                    UseL2 = l2Enabled ? true : null,
                    L2HttpPort = l2HttpPort,
                    L2HttpsPort = l2HttpsPort,
                    LoadBalancePolicy = policy,
                    Headers = headers,
                    ResponseHeaders = responseHeaders,
                    Hotlink = hotlink,
                    Cors = cors,
                    Cookie = cookie,
                    BlockTransparentProxy = blockTransparentProxy ? true : null,
                    CrawlerAction = string.IsNullOrWhiteSpace(crawlerAction) ? null : crawlerAction,
                    GuardPassTtl = guardPassTtl > 0 ? guardPassTtl : null,
                    GuardBlockTtl = guardBlockTtl > 0 ? guardBlockTtl : null,
                    UrlRedirects = urlRedirects,
                    OriginConditions = originConditions,
                    Status = status,
                    SiteType = siteType,
                    ConnLimit = connLimit > 0 ? connLimit : null,
                    AclDefaultAction = string.IsNullOrWhiteSpace(aclDefault) ? null : aclDefault,
                    AclRules = aclRules,
                    BlackIps = blackIps,
                    WhiteIps = whiteIps,
                    RegionBlock = regionBlock,
                    CcRuleId = ccRuleId,
                    OriginProtocol = originProtocol,
                    OriginHttpPort = string.IsNullOrWhiteSpace(originHttpPort) ? null : originHttpPort,
                    OriginHttpsPort = string.IsNullOrWhiteSpace(originHttpsPort) ? null : originHttpsPort,
                    Cache = cacheConfig,
                    HttpListen = httpListen.Count == 0 ? null : httpListen,
                    HttpsListen = httpsListen.Count == 0 ? null : httpsListen,
                    HttpsForce = hasHttps ? (httpsConfig.Force ?? ParseDefaultBool(defaults, "https_listen-force_ssl_enable")) : null,
                    HttpsRedirectPort = hasHttps ? (httpsConfig.RedirectPort ?? GetDefaultValue(defaults, "https_listen-port")) : null,
                    HttpsHsts = hasHttps ? (httpsConfig.Hsts ?? ParseDefaultBool(defaults, "https_listen-hsts")) : null,
                    HttpsHttp2 = hasHttps ? (httpsConfig.Http2 ?? ParseDefaultBool(defaults, "https_listen-http2")) : null,
                    HttpsOcsp = hasHttps ? httpsConfig.Ocsp : null,
                    HttpsHttp3 = hasHttps ? httpsConfig.Http3 : null,
                    HttpsSslProtocols = hasHttps ? (SanitizeNginxValue(httpsConfig.SslProtocols) ??
                                                    SanitizeNginxValue(GetDefaultValue(defaults, "https_listen-ssl_protocols"))) : null,
                    HttpsSslCiphers = hasHttps ? (SanitizeNginxValue(httpsConfig.SslCiphers) ??
                                                  SanitizeNginxValue(GetDefaultValue(defaults, "https_listen-ssl_ciphers"))) : null,
                    HttpsSslPreferServerCiphers = hasHttps ? (httpsConfig.SslPreferServerCiphers ??
                                                              NormalizeOnOff(GetDefaultValue(defaults, "https_listen-ssl_prefer_server_ciphers"))) : null,
                    ProxyConnectTimeout = proxyConnectTimeout,
                    ProxyReadTimeout = proxyReadTimeout,
                    ProxySendTimeout = proxySendTimeout,
                    ProxyHttpVersion = adv.ProxyHttpVersion,
                    ProxySslProtocols = adv.ProxySslProtocols,
                    EnableGzip = adv.Gzip,
                    GzipTypes = adv.GzipTypes,
                    EnableWebsocket = adv.Websocket,
                    EnableRange = adv.RangeEnabled,
                    BodyLimit = adv.BodyLimit > 0 ? adv.BodyLimit : null,
                    LimitRate = limitRate > 0 ? limitRate : null,
                    UpstreamKeepalive = adv.Keepalive,
                    UpstreamKeepaliveConn = adv.KeepaliveConn > 0 ? adv.KeepaliveConn : null,
                    UpstreamKeepaliveTimeout = adv.KeepaliveTimeout > 0 ? adv.KeepaliveTimeout : null,
                    UpstreamActiveHealthCheck = health.ActiveEnabled,
                    UpstreamActiveHealthCheckPath = health.ActivePath,
                    UpstreamActiveHealthCheckInterval = health.ActiveInterval,
                    UpstreamActiveHealthCheckTimeout = health.ActiveTimeout,
                    UpstreamActiveHealthCheckPolicy = health.ActivePolicy,
                    UpstreamActiveHealthCheckThreshold = health.ActiveThreshold,
                    UpstreamPassiveHealthCheck = health.PassiveEnabled,
                    UpstreamPassiveHealthCheckReactivation = health.PassiveReactivation,
                    UpstreamPassiveHealthCheckPolicy = health.PassivePolicy,
                    UpstreamPassiveHealthCheckRateLimit = health.PassiveRateLimit,
                    UpstreamAvailableDestinationsPolicy = health.AvailablePolicy
                };

                if (hasHttps)
                {
                    var cert = FindCertForDomain(domain, certs);
                    if (cert != null)
                    {
                        domainConf.SslCertData = cert.CertPem;
                        domainConf.SslKeyData = cert.Key;
                    }
                    else
                    {
                        domainConf.SslCertPath = PlaceholderCertPath;
                        domainConf.SslKeyPath = PlaceholderKeyPath;
                    }
                }

                config.Domains.Add(domainConf);
            }
        }

        config.Streams = await BuildStreamsForNodeAsync(node, groupIds, l2TargetsByGroup, groupL2Config);
    }
    private async Task<List<long>> LoadNodeGroupIdsAsync(Node node)
    {
        var lineGroups = await _db.Queryable<Line>()
            .Where(l => l.NodeId == node.Id)
            .Select(l => l.NodeGroupId)
            .ToListAsync();

        var groupIds = new HashSet<long>();
        foreach (var groupId in lineGroups)
        {
            if (groupId.HasValue && groupId.Value > 0)
            {
                groupIds.Add(groupId.Value);
            }
        }

        foreach (var pending in await LoadPendingGroupIdsAsync(node.Id))
        {
            if (pending > 0)
            {
                groupIds.Add(pending);
            }
        }

        return groupIds.ToList();
    }

    private async Task<List<long>> LoadPendingGroupIdsAsync(int nodeId)
    {
        if (nodeId <= 0)
        {
            return new List<long>();
        }

        if (!_db.DbMaintenance.IsAnyTable("line_delete_queue"))
        {
            return new List<long>();
        }

        var now = DateTime.Now;
        try
        {
            await _db.Deleteable<LineDeleteQueue>()
                .Where(q => q.DeleteAt != null && q.DeleteAt <= now)
                .ExecuteCommandAsync();
        }
        catch
        {
        }

        var rows = await _db.Queryable<LineDeleteQueue>()
            .Where(q => q.NodeId == nodeId && q.DeleteAt != null && q.DeleteAt > now)
            .ToListAsync();

        var result = new HashSet<long>();
        foreach (var row in rows)
        {
            if (row.NodeGroupId.HasValue && row.NodeGroupId.Value > 0)
            {
                result.Add(row.NodeGroupId.Value);
            }
        }

        return result.ToList();
    }

    private async Task<Dictionary<long, string>> LoadNodeGroupL2ConfigAsync(IReadOnlyList<long> groupIds)
    {
        var result = new Dictionary<long, string>();
        if (groupIds.Count == 0)
        {
            return result;
        }

        var groups = await _db.Queryable<NodeGroup>()
            .Where(g => groupIds.Contains(g.Id))
            .Select(g => new { g.Id, g.BackupSwitchPolicy })
            .ToListAsync();

        foreach (var group in groups)
        {
            var cfg = string.Empty;
            if (!string.IsNullOrWhiteSpace(group.BackupSwitchPolicy))
            {
                try
                {
                    using var doc = JsonDocument.Parse(group.BackupSwitchPolicy);
                    if (doc.RootElement.ValueKind == JsonValueKind.Object &&
                        doc.RootElement.TryGetProperty("l2_config", out var value) &&
                        value.ValueKind == JsonValueKind.String)
                    {
                        cfg = value.GetString() ?? string.Empty;
                    }
                }
                catch
                {
                }
            }

            result[group.Id] = cfg;
        }

        return result;
    }

    private async Task<Dictionary<long, List<L2Target>>> LoadL2TargetsByGroupAsync(IReadOnlyList<long> groupIds)
    {
        var result = new Dictionary<long, List<L2Target>>();
        if (groupIds.Count == 0)
        {
            return result;
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId.HasValue && groupIds.Contains(l.NodeGroupId.Value) && l.Enable == true)
            .Select(l => new { l.NodeGroupId, l.NodeId })
            .ToListAsync();

        var nodeSet = new HashSet<int>();
        foreach (var line in lines)
        {
            if (line.NodeId.HasValue && line.NodeId.Value > 0)
            {
                nodeSet.Add(line.NodeId.Value);
            }
        }

        if (nodeSet.Count == 0)
        {
            return result;
        }

        var nodes = await _db.Queryable<Node>()
            .Where(n => nodeSet.Contains(n.Id) && n.Level == 2 && n.Enable == true)
            .Select(n => new { n.Id, n.Ip })
            .ToListAsync();

        var nodeMap = nodes
            .Where(n => !string.IsNullOrWhiteSpace(n.Ip))
            .ToDictionary(n => n.Id, n => n.Ip!);

        var added = new HashSet<string>();
        foreach (var line in lines)
        {
            if (!line.NodeGroupId.HasValue || !line.NodeId.HasValue)
            {
                continue;
            }

            var groupId = line.NodeGroupId.Value;
            var nodeId = line.NodeId.Value;
            if (!nodeMap.TryGetValue(nodeId, out var ip))
            {
                continue;
            }

            var key = $"{groupId}:{nodeId}";
            if (!added.Add(key))
            {
                continue;
            }

            if (!result.TryGetValue(groupId, out var list))
            {
                list = new List<L2Target>();
                result[groupId] = list;
            }

            list.Add(new L2Target(nodeId, ip));
        }

        return result;
    }

    private async Task<Dictionary<long, long>> LoadNodeGroupCountsAsync(IReadOnlyList<long> groupIds)
    {
        var map = new Dictionary<long, long>();
        if (groupIds.Count == 0)
        {
            return map;
        }

        var rows = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId.HasValue && groupIds.Contains(l.NodeGroupId.Value))
            .GroupBy(l => l.NodeGroupId)
            .Select(l => new { NodeGroupId = l.NodeGroupId!.Value, Count = SqlFunc.AggregateDistinctCount(l.NodeId) })
            .ToListAsync();

        foreach (var row in rows)
        {
            map[row.NodeGroupId] = row.Count;
        }

        return map;
    }

    private async Task<Dictionary<int, UserPackage>> LoadUserPackageMapAsync(IReadOnlyList<Site> sites)
    {
        var ids = new HashSet<int>();
        foreach (var site in sites)
        {
            if (site.UserPackage.HasValue && site.UserPackage.Value > 0)
            {
                ids.Add(site.UserPackage.Value);
            }
        }

        var result = new Dictionary<int, UserPackage>();
        if (ids.Count == 0)
        {
            return result;
        }

        var packages = await _db.Queryable<UserPackage>()
            .Where(p => ids.Contains(p.Id))
            .ToListAsync();

        foreach (var pkg in packages)
        {
            result[pkg.Id] = pkg;
        }

        return result;
    }

    private async Task<(Dictionary<string, string> Global, Dictionary<int, Dictionary<string, string>> ByUser)> LoadSiteDefaultConfigAsync(
        IReadOnlyList<Site> sites)
    {
        var global = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        var byUser = new Dictionary<int, Dictionary<string, string>>();
        var userIds = sites
            .Select(s => s.Uid ?? 0)
            .Where(id => id > 0)
            .Distinct()
            .ToList();

        if (userIds.Count == 0)
        {
            var rows = await _db.Queryable<Config>()
                .Where(c => c.Type == "site_default_config" && c.ScopeName == "global" && c.ScopeId == 0)
                .ToListAsync();
            foreach (var row in rows)
            {
                if (string.IsNullOrWhiteSpace(row.Name))
                {
                    continue;
                }

                global[row.Name] = row.Value ?? string.Empty;
            }

            return (global, byUser);
        }

        var items = await _db.Queryable<Config>()
            .Where(c => c.Type == "site_default_config" &&
                        ((c.ScopeName == "global" && c.ScopeId == 0) ||
                         (c.ScopeName == "user" && c.ScopeId.HasValue && userIds.Contains(c.ScopeId.Value))))
            .ToListAsync();

        var userOverrides = new Dictionary<int, Dictionary<string, string>>();
        foreach (var item in items)
        {
            if (string.IsNullOrWhiteSpace(item.Name))
            {
                continue;
            }

            if (string.Equals(item.ScopeName, "global", StringComparison.OrdinalIgnoreCase) && item.ScopeId == 0)
            {
                global[item.Name] = item.Value ?? string.Empty;
                continue;
            }

            if (string.Equals(item.ScopeName, "user", StringComparison.OrdinalIgnoreCase) && item.ScopeId.HasValue && item.ScopeId.Value > 0)
            {
                if (!userOverrides.TryGetValue(item.ScopeId.Value, out var map))
                {
                    map = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
                    userOverrides[item.ScopeId.Value] = map;
                }

                map[item.Name] = item.Value ?? string.Empty;
            }
        }

        foreach (var userId in userIds)
        {
            var merged = new Dictionary<string, string>(global, StringComparer.OrdinalIgnoreCase);
            if (userOverrides.TryGetValue(userId, out var overrides))
            {
                foreach (var pair in overrides)
                {
                    merged[pair.Key] = pair.Value;
                }
            }
            byUser[userId] = merged;
        }

        return (global, byUser);
    }

    private async Task<(Dictionary<string, string> Global, Dictionary<int, Dictionary<string, string>> ByUser)>
        LoadStreamDefaultConfigAsync(IReadOnlyList<StreamEntity> streams)
    {
        var global = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        var byUser = new Dictionary<int, Dictionary<string, string>>();
        var userIds = streams
            .Select(s => s.Uid ?? 0)
            .Where(id => id > 0)
            .Distinct()
            .ToList();

        ISugarQueryable<Config> itemsQuery = _db.Queryable<Config>()
            .Where(c => c.Type == "stream_default_config" &&
                        ((c.ScopeName == "global" && c.ScopeId == 0) ||
                         (c.ScopeName == "user" && c.ScopeId.HasValue && userIds.Contains(c.ScopeId.Value))));
        if (userIds.Count == 0)
        {
            itemsQuery = _db.Queryable<Config>()
                .Where(c => c.Type == "stream_default_config" && c.ScopeName == "global" && c.ScopeId == 0);
        }

        var items = await itemsQuery.ToListAsync();

        var userOverrides = new Dictionary<int, Dictionary<string, string>>();
        foreach (var item in items)
        {
            if (string.IsNullOrWhiteSpace(item.Name))
            {
                continue;
            }

            if (string.Equals(item.ScopeName, "global", StringComparison.OrdinalIgnoreCase) && item.ScopeId == 0)
            {
                global[item.Name] = item.Value ?? string.Empty;
                continue;
            }

            if (string.Equals(item.ScopeName, "user", StringComparison.OrdinalIgnoreCase) && item.ScopeId.HasValue && item.ScopeId.Value > 0)
            {
                if (!userOverrides.TryGetValue(item.ScopeId.Value, out var map))
                {
                    map = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
                    userOverrides[item.ScopeId.Value] = map;
                }

                map[item.Name] = item.Value ?? string.Empty;
            }
        }

        var forwardDefaults = await LoadForwardDefaultSettingsAsync();
        if (forwardDefaults.Count > 0)
        {
            foreach (var pair in forwardDefaults)
            {
                global[pair.Key] = pair.Value;
            }
        }

        if (userIds.Count == 0)
        {
            return (global, byUser);
        }

        foreach (var userId in userIds)
        {
            var merged = new Dictionary<string, string>(global, StringComparer.OrdinalIgnoreCase);
            if (userOverrides.TryGetValue(userId, out var overrides))
            {
                foreach (var pair in overrides)
                {
                    merged[pair.Key] = pair.Value;
                }
            }
            byUser[userId] = merged;
        }

        return (global, byUser);
    }

    private async Task<Dictionary<string, string>> LoadForwardDefaultSettingsAsync()
    {
        var result = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        var config = await _db.Queryable<Config>()
            .Where(c => c.Type == "system" && c.Name == "forward_default_settings")
            .FirstAsync();
        if (config == null || string.IsNullOrWhiteSpace(config.Value))
        {
            return result;
        }

        try
        {
            using var doc = JsonDocument.Parse(config.Value);
            if (doc.RootElement.ValueKind != JsonValueKind.Array)
            {
                return result;
            }

            foreach (var item in doc.RootElement.EnumerateArray())
            {
                if (item.ValueKind != JsonValueKind.Object)
                {
                    continue;
                }

                var key = ReadJsonString(item, "key");
                if (string.IsNullOrWhiteSpace(key))
                {
                    continue;
                }

                var scope = ReadJsonString(item, "scope");
                if (!string.IsNullOrWhiteSpace(scope) &&
                    !string.Equals(scope.Trim(), "global", StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }

                if (!item.TryGetProperty("value", out var value))
                {
                    result[key] = string.Empty;
                    continue;
                }

                result[key] = EncodeForwardDefaultValue(value);
            }
        }
        catch
        {
        }

        return result;
    }

    private static string EncodeForwardDefaultValue(JsonElement value)
    {
        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString() ?? string.Empty,
            JsonValueKind.True => "true",
            JsonValueKind.False => "false",
            JsonValueKind.Number => EncodeNumber(value),
            JsonValueKind.Null => string.Empty,
            _ => JsonSerializer.Serialize(value, JsonOptions)
        };
    }

    private static string EncodeNumber(JsonElement value)
    {
        if (value.TryGetInt64(out var parsed))
        {
            return parsed.ToString();
        }

        if (value.TryGetDouble(out var dbl))
        {
            return ((long)dbl).ToString();
        }

        return value.ToString();
    }

    private async Task<Dictionary<int, UserPackage>> LoadUserPackageMapAsync(IReadOnlyList<StreamEntity> streams)
    {
        var ids = new HashSet<int>();
        foreach (var stream in streams)
        {
            if (stream.UserPackage.HasValue && stream.UserPackage.Value > 0)
            {
                ids.Add(stream.UserPackage.Value);
            }
        }

        var result = new Dictionary<int, UserPackage>();
        if (ids.Count == 0)
        {
            return result;
        }

        var packages = await _db.Queryable<UserPackage>()
            .Where(p => ids.Contains(p.Id))
            .ToListAsync();

        foreach (var pkg in packages)
        {
            result[pkg.Id] = pkg;
        }

        return result;
    }

    private static Dictionary<string, string> ResolveSiteDefaults(
        int? userId,
        Dictionary<string, string> global,
        Dictionary<int, Dictionary<string, string>> byUser)
    {
        if (userId.HasValue && userId.Value > 0 && byUser.TryGetValue(userId.Value, out var map))
        {
            return map;
        }

        return global;
    }

    private static string? GetDefaultValue(Dictionary<string, string> defaults, string key)
    {
        return defaults.TryGetValue(key, out var value) ? value : null;
    }

    private static bool? ParseDefaultBool(Dictionary<string, string> defaults, string key)
    {
        var raw = GetDefaultValue(defaults, key);
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        var normalized = raw.Trim().ToLowerInvariant();
        return normalized switch
        {
            "1" or "true" or "yes" or "on" => true,
            "0" or "false" or "no" or "off" => false,
            _ => null
        };
    }

    private static int? ParseDefaultInt(Dictionary<string, string> defaults, string key)
    {
        var raw = GetDefaultValue(defaults, key);
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        return int.TryParse(raw.Trim(), out var parsed) ? parsed : null;
    }

    private static double? ParseDefaultDouble(Dictionary<string, string> defaults, string key)
    {
        var raw = GetDefaultValue(defaults, key);
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        return double.TryParse(raw.Trim(), System.Globalization.NumberStyles.Float, System.Globalization.CultureInfo.InvariantCulture, out var parsed)
            ? parsed
            : null;
    }

    private static string? NormalizeOnOff(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        var normalized = raw.Trim().ToLowerInvariant();
        if (normalized is "1" or "true" or "yes" or "on")
        {
            return "on";
        }
        if (normalized is "0" or "false" or "no" or "off")
        {
            return "off";
        }

        return SanitizeNginxValue(raw);
    }

    private static Dictionary<int, Dictionary<int, int>> BuildDomainCountByUserGroup(IReadOnlyList<Site> sites)
    {
        var result = new Dictionary<int, Dictionary<int, int>>();
        foreach (var site in sites)
        {
            if (site.Enable != true)
            {
                continue;
            }

            var groupId = site.NodeGroupId ?? 0;
            var userId = site.Uid ?? 0;
            if (groupId <= 0 || userId <= 0)
            {
                continue;
            }

            if (!result.TryGetValue(groupId, out var groupMap))
            {
                groupMap = new Dictionary<int, int>();
                result[groupId] = groupMap;
            }

            var domains = SplitDomainList(site.Domain);
            var count = domains.Count;
            if (count == 0 && !string.IsNullOrWhiteSpace(site.Domain))
            {
                count = 1;
            }

            if (!groupMap.ContainsKey(userId))
            {
                groupMap[userId] = 0;
            }

            groupMap[userId] += count;
        }

        return result;
    }

    private static string ResolveSiteStatus(
        Site site,
        Dictionary<int, UserPackage> userPackageMap,
        bool expireCloseEnabled,
        DateTime now)
    {
        var status = "active";
        var state = site.State?.Trim().ToLowerInvariant() ?? string.Empty;
        switch (state)
        {
            case "stop":
            case "locked":
            case "site_locked":
                status = "locked";
                break;
            case "traffic_limit":
                status = "traffic_limit";
                break;
            case "conn_limit":
                status = "conn_limit";
                break;
            case "expired":
            case "timeout":
                status = "expired";
                break;
        }

        if (site.Enable == false)
        {
            status = "locked";
        }

        if (expireCloseEnabled && site.UserPackage.HasValue && userPackageMap.TryGetValue(site.UserPackage.Value, out var pkg))
        {
            if (pkg.EndAt.HasValue && pkg.EndAt.Value < now)
            {
                status = "expired";
            }
        }

        return status;
    }

    private static List<EdgeUpstreamTargetDto> BuildUpstreamTargets(Site site, string originProtocol, string originHttpPort, string originHttpsPort)
    {
        var targets = new List<EdgeUpstreamTargetDto>();
        foreach (var backend in SplitOriginList(site.Backend))
        {
            if (string.IsNullOrWhiteSpace(backend))
            {
                continue;
            }

            var addr = NormalizeOriginAddr(backend, originProtocol, originHttpPort, originHttpsPort);
            if (string.IsNullOrWhiteSpace(addr))
            {
                continue;
            }

            targets.Add(new EdgeUpstreamTargetDto
            {
                Addr = addr,
                Weight = 10
            });
        }

        return targets;
    }

    private static string NormalizeOriginAddr(string addr, string originProtocol, string originHttpPort, string originHttpsPort)
    {
        addr = addr.Trim();
        if (addr.Length == 0)
        {
            return addr;
        }

        if (addr.Contains(':'))
        {
            return addr;
        }

        var protocol = originProtocol.Trim().ToLowerInvariant();
        switch (protocol)
        {
            case "http":
                if (!string.IsNullOrWhiteSpace(originHttpPort))
                {
                    return $"{addr}:{originHttpPort}";
                }
                break;
            case "https":
                if (!string.IsNullOrWhiteSpace(originHttpsPort))
                {
                    return $"{addr}:{originHttpsPort}";
                }
                break;
        }

        return addr;
    }

    private static string MapBalancePolicy(string? way)
    {
        return way?.Trim().ToLowerInvariant() switch
        {
            "ip_hash" => "ip_hash",
            "random" => "random",
            _ => "round_robin"
        };
    }

    private static bool ResolveL2Enabled(string mode, string groupConfig, bool packageEnabled)
    {
        mode = mode.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(mode))
        {
            mode = "current";
        }
        if (mode == "none")
        {
            return false;
        }

        groupConfig = groupConfig.Trim().ToLowerInvariant();
        if (groupConfig == "none")
        {
            return false;
        }

        return mode != "current" || packageEnabled;
    }

    private static string ResolveListenPort(IReadOnlyList<string> ports, string fallback)
    {
        foreach (var port in ports)
        {
            var trimmed = port.Trim();
            if (!string.IsNullOrWhiteSpace(trimmed))
            {
                return trimmed;
            }
        }

        return fallback;
    }

    private static Dictionary<string, string>? BuildRequestHeaders(Site site)
    {
        var headers = ParseHeaderMap(site.ReqHeader);
        if (headers == null)
        {
            headers = new Dictionary<string, string>();
        }

        if (!headers.ContainsKey("Host"))
        {
            var originHost = site.BackendHost?.Trim();
            if (!string.IsNullOrWhiteSpace(originHost))
            {
                var value = SanitizeHeaderValue(originHost);
                if (!string.IsNullOrWhiteSpace(value))
                {
                    headers["Host"] = value;
                }
            }
        }

        return headers.Count == 0 ? null : headers;
    }

    private static Dictionary<string, string>? BuildResponseHeaders(Site site)
    {
        var headers = ParseHeaderMap(site.RespHeader);
        return headers == null || headers.Count == 0 ? null : headers;
    }

    private async Task<(string? DefaultAction, List<EdgeAclRuleDto>? Rules)> BuildAclForSiteAsync(long? aclId)
    {
        var id = aclId ?? 0;
        if (id <= 0)
        {
            return (null, null);
        }

        var acl = await _db.Queryable<Acl>().Where(a => a.Id == id).FirstAsync();
        if (acl == null)
        {
            return (null, null);
        }

        var action = string.IsNullOrWhiteSpace(acl.DefaultAction) ? "allow" : acl.DefaultAction.Trim();
        var rules = ParseAclRules(acl.Data);
        return (action, rules);
    }
    private static List<EdgeAclRuleDto>? ParseAclRules(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            var direct = JsonSerializer.Deserialize<List<EdgeAclRuleDto>>(raw, JsonOptions);
            if (direct != null && direct.Count > 0)
            {
                return direct;
            }
        }
        catch
        {
        }

        try
        {
            var wrapper = JsonSerializer.Deserialize<AclRuleWrapper>(raw, JsonOptions);
            if (wrapper?.Rules != null && wrapper.Rules.Count > 0)
            {
                return ExtractAclIpRules(wrapper.Rules);
            }
        }
        catch
        {
        }

        try
        {
            var rules = JsonSerializer.Deserialize<List<AclRule>>(raw, JsonOptions);
            if (rules != null && rules.Count > 0)
            {
                return ExtractAclIpRules(rules);
            }
        }
        catch
        {
        }

        try
        {
            var list = JsonSerializer.Deserialize<List<Dictionary<string, JsonElement>>>(raw, JsonOptions);
            if (list == null || list.Count == 0)
            {
                return null;
            }

            var result = new List<EdgeAclRuleDto>();
            foreach (var entry in list)
            {
                if (!TryGetEntry(entry, "ip", out var ipValue))
                {
                    continue;
                }

                var ip = ParseString(ipValue);
                if (string.IsNullOrWhiteSpace(ip))
                {
                    continue;
                }

                var action = "allow";
                if (TryGetEntry(entry, "action", out var actionValue))
                {
                    var parsed = ParseString(actionValue);
                    if (!string.IsNullOrWhiteSpace(parsed))
                    {
                        action = parsed;
                    }
                }

                result.Add(new EdgeAclRuleDto { Ip = ip, Action = action });
            }

            return result.Count == 0 ? null : result;
        }
        catch
        {
        }

        return null;
    }

    private sealed class AclRuleWrapper
    {
        public List<AclRule>? Rules { get; set; }
    }

    private sealed class AclRule
    {
        public List<AclCondition>? Conditions { get; set; }
        public string? Action { get; set; }
    }

    private sealed class AclCondition
    {
        public string? Item { get; set; }
        public string? Operator { get; set; }
        public string? Value { get; set; }
    }

    private static List<EdgeAclRuleDto>? ExtractAclIpRules(List<AclRule> rules)
    {
        if (rules.Count == 0)
        {
            return null;
        }

        var result = new List<EdgeAclRuleDto>();
        foreach (var rule in rules)
        {
            var action = string.IsNullOrWhiteSpace(rule.Action) ? "allow" : rule.Action!.Trim();
            if (rule.Conditions == null || rule.Conditions.Count == 0)
            {
                continue;
            }

            var ipOnly = true;
            foreach (var cond in rule.Conditions)
            {
                var item = cond.Item?.Trim().ToLowerInvariant() ?? string.Empty;
                if (item != "ip")
                {
                    ipOnly = false;
                    break;
                }

                var op = cond.Operator?.Trim().ToLowerInvariant() ?? string.Empty;
                if (op != "eq" && op != "=")
                {
                    ipOnly = false;
                    break;
                }

                var ip = cond.Value?.Trim();
                if (string.IsNullOrWhiteSpace(ip))
                {
                    continue;
                }

                result.Add(new EdgeAclRuleDto { Ip = ip, Action = action });
            }

            if (!ipOnly)
            {
                continue;
            }
        }

        return result.Count == 0 ? null : result;
    }

    private static List<string>? ParseRegionList(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        List<string> items;
        var trimmed = raw.Trim();
        if (trimmed.StartsWith("["))
        {
            try
            {
                var parsed = JsonSerializer.Deserialize<List<string>>(trimmed, JsonOptions);
                if (parsed != null)
                {
                    items = parsed;
                }
                else
                {
                    items = new List<string>();
                }
            }
            catch
            {
                items = SplitFields(trimmed);
            }
        }
        else
        {
            items = SplitFields(trimmed);
        }

        return NormalizeRegionList(items);
    }

    private static List<string>? NormalizeRegionList(List<string> list)
    {
        if (list.Count == 0)
        {
            return null;
        }

        var result = new List<string>();
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var item in list)
        {
            var code = item?.Trim().ToUpperInvariant() ?? string.Empty;
            if (string.IsNullOrWhiteSpace(code))
            {
                continue;
            }

            var idx = code.IndexOf('-', StringComparison.Ordinal);
            if (idx > 0)
            {
                code = code[..idx];
            }

            if (seen.Add(code))
            {
                result.Add(code);
            }
        }

        return result.Count == 0 ? null : result;
    }

    private static List<string>? ParseIpList(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        var trimmed = raw.Trim();
        List<string> items;
        if (trimmed.StartsWith("["))
        {
            try
            {
                var parsed = JsonSerializer.Deserialize<List<string>>(trimmed, JsonOptions);
                items = parsed ?? new List<string>();
            }
            catch
            {
                items = SplitFields(trimmed);
            }
        }
        else
        {
            items = SplitFields(trimmed);
        }

        return NormalizeIpList(items);
    }

    private static List<string>? NormalizeIpList(List<string> list)
    {
        if (list.Count == 0)
        {
            return null;
        }

        var result = new List<string>();
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var item in list)
        {
            var ip = item?.Trim() ?? string.Empty;
            if (string.IsNullOrWhiteSpace(ip))
            {
                continue;
            }

            if (seen.Add(ip))
            {
                result.Add(ip);
            }
        }

        return result.Count == 0 ? null : result;
    }
    private static EdgeHotlinkConfigDto? ParseHotlinkConfig(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                return null;
            }

            var root = doc.RootElement;
            if (root.TryGetProperty("enable", out var enabledValue) && !ParseBool(enabledValue, false))
            {
                return null;
            }

            var cfg = new EdgeHotlinkConfigDto
            {
                Enable = true
            };

            if (root.TryGetProperty("scope", out var scope))
            {
                cfg.Scope = scope.GetString();
            }

            if (root.TryGetProperty("value", out var value))
            {
                cfg.Value = value.GetString();
            }

            if (root.TryGetProperty("allow_empty", out var allowEmpty))
            {
                cfg.AllowEmpty = ParseBool(allowEmpty, true);
            }

            if (root.TryGetProperty("allowEmpty", out var allowEmpty2))
            {
                cfg.AllowEmpty = ParseBool(allowEmpty2, cfg.AllowEmpty ?? true);
            }

            if (root.TryGetProperty("domains", out var domains))
            {
                cfg.Domains = ParseJsonStringList(domains);
            }

            return cfg;
        }
        catch
        {
            return null;
        }
    }

    private static EdgeCorsConfigDto? ParseCorsConfig(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                return null;
            }

            var root = doc.RootElement;
            if (root.TryGetProperty("enable", out var enabledValue) && !ParseBool(enabledValue, false))
            {
                return null;
            }

            var cfg = new EdgeCorsConfigDto
            {
                Enable = true
            };

            if (root.TryGetProperty("allowOrigin", out var allowOrigin))
            {
                cfg.AllowOrigin = allowOrigin.GetString();
            }
            if (root.TryGetProperty("allowMethods", out var allowMethods))
            {
                cfg.AllowMethods = allowMethods.GetString();
            }
            if (root.TryGetProperty("allowHeaders", out var allowHeaders))
            {
                cfg.AllowHeaders = allowHeaders.GetString();
            }
            if (root.TryGetProperty("exposeHeaders", out var exposeHeaders))
            {
                cfg.ExposeHeaders = exposeHeaders.GetString();
            }
            if (root.TryGetProperty("allowCredentials", out var allowCredentials))
            {
                cfg.AllowCredentials = ParseBool(allowCredentials, false);
            }
            if (root.TryGetProperty("maxAge", out var maxAge))
            {
                cfg.MaxAge = maxAge.GetString();
            }

            return cfg;
        }
        catch
        {
            return null;
        }
    }

    private static EdgeCookieConfigDto? ParseCookieConfig(Site site)
    {
        return null;
    }

    private static string ParseCrawlerAction(string? raw, Dictionary<string, string> defaults)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            raw = GetDefaultValue(defaults, "security_bot");
            if (string.IsNullOrWhiteSpace(raw))
            {
                return string.Empty;
            }
        }

        var normalized = raw.Trim().ToLowerInvariant();
        return normalized is "allow" or "deny" or "block" ? normalized : string.Empty;
    }

    private static (int PassTtl, int BlockTtl) ParseGuardTtls(Site site, Dictionary<string, string> defaults)
    {
        var passTtl = ParseDefaultInt(defaults, "security_white_time") ?? 0;
        var blockTtl = ParseDefaultInt(defaults, "security_black_time") ?? 0;
        return (passTtl, blockTtl);
    }

    private static List<Dictionary<string, JsonElement>>? ParseUrlRedirects(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            var list = JsonSerializer.Deserialize<List<Dictionary<string, JsonElement>>>(raw, JsonOptions);
            return list == null || list.Count == 0 ? null : list;
        }
        catch
        {
            return null;
        }
    }

    private static List<Dictionary<string, JsonElement>>? ParseOriginConditions(Site site)
    {
        return null;
    }

    private static long? ResolveCcRuleId(
        Site site,
        Dictionary<string, string> defaults,
        Dictionary<string, JsonElement>? settings)
    {
        var ruleId = 0L;
        var security = GetSettingsMap(settings, "security");
        if (security != null && TryGetEntry(security, "default_rule", out var defaultValue))
        {
            ruleId = ParseId(defaultValue);
        }

        if (ruleId <= 0)
        {
            ruleId = site.CcDefaultRule ?? 0;
        }
        if (ruleId <= 0)
        {
            var defaultRule = ParseDefaultInt(defaults, "cc_default_rule") ?? 0;
            ruleId = defaultRule;
        }
        if (ruleId <= 0)
        {
            return null;
        }

        var switchValue = site.CcSwitch?.Trim();
        if (!string.IsNullOrWhiteSpace(switchValue) && !ParseBoolFlag(switchValue, true))
        {
            return null;
        }

        return ruleId;
    }
    private static EdgeCacheConfigDto? ParseCacheConfig(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        var trimmed = raw.Trim();
        if (trimmed == "[]")
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(trimmed);
            if (doc.RootElement.ValueKind == JsonValueKind.Object)
            {
                var rules = ParseCacheRulesFromElement(doc.RootElement);
                var enable = true;
                if (doc.RootElement.TryGetProperty("enable", out var enabledValue))
                {
                    enable = ParseBool(enabledValue, true);
                }

                var ttl = (int?)null;
                if (doc.RootElement.TryGetProperty("ttl", out var ttlValue) && ttlValue.ValueKind == JsonValueKind.Number)
                {
                    ttl = ttlValue.GetInt32();
                }
                if (doc.RootElement.TryGetProperty("default_ttl", out var ttlValue2) && ttlValue2.ValueKind == JsonValueKind.Number)
                {
                    ttl = ttlValue2.GetInt32();
                }

                if ((rules == null || rules.Count == 0) && ttl == null && !enable)
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

            if (doc.RootElement.ValueKind == JsonValueKind.Array)
            {
                var rules = ParseCacheRulesFromElement(doc.RootElement);
                if (rules == null || rules.Count == 0)
                {
                    return null;
                }

                return new EdgeCacheConfigDto
                {
                    Enable = true,
                    Rules = rules
                };
            }
        }
        catch
        {
        }

        return null;
    }

    private static List<EdgeCacheRuleDto>? ParseCacheRulesFromElement(JsonElement element)
    {
        var rules = new List<EdgeCacheRuleDto>();
        if (element.ValueKind == JsonValueKind.Object)
        {
            if (!element.TryGetProperty("rules", out var rulesElement))
            {
                return null;
            }

            if (rulesElement.ValueKind != JsonValueKind.Array)
            {
                return null;
            }

            element = rulesElement;
        }

        if (element.ValueKind != JsonValueKind.Array)
        {
            return null;
        }

        foreach (var item in element.EnumerateArray())
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

            foreach (var rule in MapToCacheRules(map))
            {
                rules.Add(rule);
            }
        }

        if (rules.Count == 0)
        {
            return null;
        }

        rules = DedupeCacheRules(rules);
        return rules.Count == 0 ? null : rules;
    }

    private static List<EdgeCacheRuleDto> MapToCacheRules(Dictionary<string, JsonElement> raw)
    {
        var baseRule = MapToCacheRule(raw);
        if (!string.IsNullOrWhiteSpace(baseRule.Rule) ||
            !string.IsNullOrWhiteSpace(baseRule.Ext) ||
            !string.IsNullOrWhiteSpace(baseRule.Uri) ||
            !string.IsNullOrWhiteSpace(baseRule.Prefix))
        {
            return new List<EdgeCacheRuleDto> { baseRule };
        }

        var ruleType = ParseStringFromMap(raw, "type").ToLowerInvariant();
        var values = SplitCacheRuleValues(ParseStringFromMap(raw, "value"));
        if (values.Count == 0)
        {
            values.Add(string.Empty);
        }

        switch (ruleType)
        {
            case "suffix":
                return BuildCacheRulesWithExt(baseRule, values);
            case "dir":
                return BuildCacheRulesWithPrefix(baseRule, values);
            case "path":
                return BuildCacheRulesWithUri(baseRule, values);
            case "all":
            {
                var rule = CloneCacheRule(baseRule);
                rule.Prefix = "/";
                return new List<EdgeCacheRuleDto> { rule };
            }
            case "index":
            {
                var rule = CloneCacheRule(baseRule);
                rule.Uri = "/";
                return new List<EdgeCacheRuleDto> { rule };
            }
            default:
                return new List<EdgeCacheRuleDto> { baseRule };
        }
    }

    private static EdgeCacheRuleDto MapToCacheRule(Dictionary<string, JsonElement> raw)
    {
        var ignoreArgs = ParseBoolFromMap(raw, "ignore_args");
        if (ignoreArgs == null)
        {
            ignoreArgs = ParseBoolFromMap(raw, "ignore_query");
        }

        return new EdgeCacheRuleDto
        {
            Rule = NullIfEmpty(ParseStringFromMap(raw, "rule")),
            Ext = NullIfEmpty(ParseStringFromMap(raw, "ext")),
            Uri = NullIfEmpty(ParseStringFromMap(raw, "uri")),
            Prefix = NullIfEmpty(ParseStringFromMap(raw, "prefix")),
            Ttl = ParseIntFromMap(raw, "ttl"),
            Enable = ParseBoolFromMap(raw, "enable"),
            NoCache = ParseBoolFromMap(raw, "no_cache"),
            ForceCache = ParseBoolFromMap(raw, "force_cache"),
            Priority = ParseIntFromMap(raw, "priority"),
            IgnoreArgs = ignoreArgs,
            CacheKey = NullIfEmpty(ParseStringFromMap(raw, "cache_key"))
        };
    }

    private static EdgeCacheRuleDto CloneCacheRule(EdgeCacheRuleDto source)
    {
        return new EdgeCacheRuleDto
        {
            Rule = source.Rule,
            Ext = source.Ext,
            Uri = source.Uri,
            Prefix = source.Prefix,
            Ttl = source.Ttl,
            Enable = source.Enable,
            NoCache = source.NoCache,
            ForceCache = source.ForceCache,
            Priority = source.Priority,
            IgnoreArgs = source.IgnoreArgs,
            CacheKey = source.CacheKey
        };
    }

    private static List<EdgeCacheRuleDto> BuildCacheRulesWithExt(EdgeCacheRuleDto baseRule, List<string> values)
    {
        var rules = new List<EdgeCacheRuleDto>();
        foreach (var value in values)
        {
            var item = value.Trim();
            if (string.IsNullOrWhiteSpace(item))
            {
                continue;
            }

            item = item.TrimStart('*').TrimStart('.');
            if (string.IsNullOrWhiteSpace(item))
            {
                continue;
            }

            var rule = CloneCacheRule(baseRule);
            rule.Ext = item;
            rules.Add(rule);
        }

        return rules.Count == 0 ? new List<EdgeCacheRuleDto> { baseRule } : rules;
    }

    private static List<EdgeCacheRuleDto> BuildCacheRulesWithPrefix(EdgeCacheRuleDto baseRule, List<string> values)
    {
        var rules = new List<EdgeCacheRuleDto>();
        foreach (var value in values)
        {
            var item = NormalizeCachePath(value);
            if (string.IsNullOrWhiteSpace(item))
            {
                continue;
            }

            var rule = CloneCacheRule(baseRule);
            rule.Prefix = item;
            rules.Add(rule);
        }

        return rules.Count == 0 ? new List<EdgeCacheRuleDto> { baseRule } : rules;
    }

    private static List<EdgeCacheRuleDto> BuildCacheRulesWithUri(EdgeCacheRuleDto baseRule, List<string> values)
    {
        var rules = new List<EdgeCacheRuleDto>();
        foreach (var value in values)
        {
            var item = NormalizeCachePath(value);
            if (string.IsNullOrWhiteSpace(item))
            {
                continue;
            }

            var rule = CloneCacheRule(baseRule);
            rule.Uri = item;
            rules.Add(rule);
        }

        return rules.Count == 0 ? new List<EdgeCacheRuleDto> { baseRule } : rules;
    }

    private static string NormalizeCachePath(string value)
    {
        var item = value.Trim();
        if (string.IsNullOrWhiteSpace(item))
        {
            return string.Empty;
        }

        return item.StartsWith("/", StringComparison.Ordinal) ? item : "/" + item;
    }

    private static List<string> SplitCacheRuleValues(string value)
    {
        value = value.Trim();
        if (string.IsNullOrWhiteSpace(value))
        {
            return new List<string>();
        }

        string[] parts;
        if (value.Contains('|'))
        {
            parts = value.Split('|', StringSplitOptions.RemoveEmptyEntries);
        }
        else
        {
            parts = value.Split(new[] { ' ', '\t', '\r', '\n' }, StringSplitOptions.RemoveEmptyEntries);
        }

        var result = new List<string>();
        foreach (var part in parts)
        {
            var item = part.Trim();
            if (!string.IsNullOrWhiteSpace(item))
            {
                result.Add(item);
            }
        }

        return result;
    }

    private static List<EdgeCacheRuleDto> DedupeCacheRules(List<EdgeCacheRuleDto> rules)
    {
        if (rules.Count == 0)
        {
            return rules;
        }

        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var result = new List<EdgeCacheRuleDto>();
        foreach (var rule in rules)
        {
            var key = CacheRuleLocationKey(rule);
            if (string.IsNullOrWhiteSpace(key))
            {
                continue;
            }

            if (seen.Add(key))
            {
                result.Add(rule);
            }
        }

        return result;
    }

    private static string CacheRuleLocationKey(EdgeCacheRuleDto rule)
    {
        var location = CacheRuleLocation(rule);
        if (string.IsNullOrWhiteSpace(location))
        {
            return string.Empty;
        }

        return NormalizeLocationKey(location);
    }

    private static string CacheRuleLocation(EdgeCacheRuleDto rule)
    {
        if (!string.IsNullOrWhiteSpace(rule.Rule))
        {
            return NormalizeRuleLocation(rule.Rule!);
        }
        if (!string.IsNullOrWhiteSpace(rule.Uri))
        {
            var uri = rule.Uri!.Trim();
            if (!uri.StartsWith("/", StringComparison.Ordinal))
            {
                return string.Empty;
            }
            return "= " + uri;
        }
        if (!string.IsNullOrWhiteSpace(rule.Prefix))
        {
            var prefix = rule.Prefix!.Trim();
            if (!prefix.StartsWith("/", StringComparison.Ordinal))
            {
                return string.Empty;
            }
            return "^~ " + prefix;
        }
        if (!string.IsNullOrWhiteSpace(rule.Ext))
        {
            var ext = rule.Ext!.Trim().TrimStart('*').TrimStart('.');
            if (string.IsNullOrWhiteSpace(ext))
            {
                return string.Empty;
            }
            return "~* \\." + ext + "$";
        }
        return string.Empty;
    }

    private static string NormalizeRuleLocation(string rule)
    {
        rule = rule.Trim();
        if (string.IsNullOrWhiteSpace(rule))
        {
            return string.Empty;
        }
        if (rule.StartsWith("=", StringComparison.Ordinal) ||
            rule.StartsWith("^~", StringComparison.Ordinal) ||
            rule.StartsWith("~", StringComparison.Ordinal))
        {
            return rule;
        }
        if (rule.StartsWith("/", StringComparison.Ordinal))
        {
            return "^~ " + rule;
        }
        if (rule.StartsWith(".", StringComparison.Ordinal))
        {
            return "~* \\"
                + rule + "$";
        }
        return "~* " + rule;
    }

    private static string NormalizeLocationKey(string location)
    {
        location = location.Trim();
        if (string.IsNullOrWhiteSpace(location))
        {
            return string.Empty;
        }

        var parts = location.Split(' ', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length == 0)
        {
            return string.Empty;
        }

        return parts[0] switch
        {
            "=" => parts.Length < 2 ? "exact" : "exact " + string.Join(" ", parts[1..]),
            "^~" => parts.Length < 2 ? "prefix" : "prefix " + string.Join(" ", parts[1..]),
            _ => parts[0].StartsWith("~", StringComparison.Ordinal)
                ? parts.Length < 2 ? "regex " + parts[0] : "regex " + parts[0] + " " + string.Join(" ", parts[1..])
                : "prefix " + string.Join(" ", parts)
        };
    }
    private static string NullIfEmpty(string value)
    {
        return string.IsNullOrWhiteSpace(value) ? string.Empty : value;
    }

    private static string ParseStringFromMap(Dictionary<string, JsonElement> map, string key)
    {
        return TryGetEntry(map, key, out var value) ? ParseString(value) : string.Empty;
    }

    private static bool? ParseBoolFromMap(Dictionary<string, JsonElement> map, string key)
    {
        if (!TryGetEntry(map, key, out var value))
        {
            return null;
        }

        return ParseBool(value, false);
    }

    private static int? ParseIntFromMap(Dictionary<string, JsonElement> map, string key)
    {
        if (!TryGetEntry(map, key, out var value))
        {
            return null;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var parsed))
        {
            return parsed;
        }

        if (value.ValueKind == JsonValueKind.String && int.TryParse(value.GetString(), out var parsedString))
        {
            return parsedString;
        }

        return null;
    }

    private static (string? ConnectTimeout, string? ReadTimeout, string? SendTimeout) ParseProxyTimeouts(string? raw)
    {
        var normalized = NormalizeTimeout(raw);
        if (string.IsNullOrWhiteSpace(normalized))
        {
            return (null, null, null);
        }

        return (normalized, normalized, normalized);
    }

    private static string? NormalizeTimeout(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        var value = raw.Trim().ToLowerInvariant();
        if (value.EndsWith("s") || value.EndsWith("m") || value.EndsWith("h"))
        {
            return value;
        }

        if (int.TryParse(value, out var seconds) && seconds > 0)
        {
            return $"{seconds}s";
        }

        return null;
    }

    private static AdvancedConfig BuildAdvancedConfig(
        Site site,
        Dictionary<string, string> defaults,
        Dictionary<string, JsonElement>? settings)
    {
        var adv = GetSettingsMap(settings, "advanced");

        var gzipEnable = ParseBoolSetting(adv, "gzip") ?? site.GzipEnable ?? ParseDefaultBool(defaults, "gzip_enable");
        var gzipTypes = ParseStringSetting(adv, "gzip_types");
        if (string.IsNullOrWhiteSpace(gzipTypes))
        {
            gzipTypes = string.IsNullOrWhiteSpace(site.GzipTypes)
                ? GetDefaultValue(defaults, "gzip_types")
                : site.GzipTypes;
        }

        var websocket = ParseBoolSetting(adv, "websocket") ?? site.WebsocketEnable ?? ParseDefaultBool(defaults, "websocket_enable");
        var rangeEnabled = ParseBoolSetting(adv, "range") ?? site.Range ?? ParseDefaultBool(defaults, "range");

        var proxyHttpVersion = ParseStringSetting(adv, "proxy_http_version");
        if (string.IsNullOrWhiteSpace(proxyHttpVersion))
        {
            proxyHttpVersion = string.IsNullOrWhiteSpace(site.ProxyHttpVersion)
                ? GetDefaultValue(defaults, "proxy_http_version")
                : site.ProxyHttpVersion;
        }

        var proxySslProtocols = ParseStringSetting(adv, "proxy_ssl_protocols");
        if (string.IsNullOrWhiteSpace(proxySslProtocols))
        {
            proxySslProtocols = string.IsNullOrWhiteSpace(site.ProxySslProtocols)
                ? GetDefaultValue(defaults, "proxy_ssl_protocols")
                : site.ProxySslProtocols;
        }

        var bodyLimit = ParseIntSetting(adv, "body_limit") ??
                        site.PostSizeLimit ??
                        ParseDefaultInt(defaults, "post_size_limit") ??
                        0;
        var keepalive = ParseBoolSetting(adv, "ups_keepalive") ?? site.UpsKeepalive ?? ParseDefaultBool(defaults, "ups_keepalive");
        var keepaliveConn = ParseIntSetting(adv, "ups_keepalive_conn") ??
                            site.UpsKeepaliveConn ??
                            ParseDefaultInt(defaults, "ups_keepalive_conn") ??
                            0;
        var keepaliveTimeout = ParseIntSetting(adv, "ups_keepalive_timeout") ??
                               site.UpsKeepaliveTimeout ??
                               ParseDefaultInt(defaults, "ups_keepalive_timeout") ??
                               0;

        return new AdvancedConfig(
            gzipEnable,
            SanitizeNginxValue(gzipTypes),
            websocket,
            rangeEnabled,
            SanitizeProxyHttpVersion(proxyHttpVersion),
            SanitizeNginxValue(proxySslProtocols),
            bodyLimit,
            keepalive,
            keepaliveConn,
            keepaliveTimeout);
    }

    private static HealthRuntimeConfig BuildHealthRuntimeConfig(
        Dictionary<string, JsonElement>? settings,
        Dictionary<string, string> defaults)
    {
        var backsource = GetSettingsMap(settings, "backsource");

        var activeEnabled = ParseBoolSetting(backsource, "upstream_active_health_check")
            ?? ParseDefaultBool(defaults, "upstream_active_health_check");
        var activePath = ParseStringSetting(backsource, "upstream_active_health_check_path")
            ?? GetDefaultValue(defaults, "upstream_active_health_check_path");
        var activeInterval = ParseStringSetting(backsource, "upstream_active_health_check_interval")
            ?? GetDefaultValue(defaults, "upstream_active_health_check_interval");
        var activeTimeout = ParseStringSetting(backsource, "upstream_active_health_check_timeout")
            ?? GetDefaultValue(defaults, "upstream_active_health_check_timeout");
        var activePolicy = ParseStringSetting(backsource, "upstream_active_health_check_policy")
            ?? GetDefaultValue(defaults, "upstream_active_health_check_policy");
        var activeThreshold = ParseIntSetting(backsource, "upstream_active_health_check_threshold")
            ?? ParseDefaultInt(defaults, "upstream_active_health_check_threshold");

        var passiveEnabled = ParseBoolSetting(backsource, "upstream_passive_health_check")
            ?? ParseDefaultBool(defaults, "upstream_passive_health_check");
        var passiveReactivation = ParseStringSetting(backsource, "upstream_passive_health_check_reactivation")
            ?? GetDefaultValue(defaults, "upstream_passive_health_check_reactivation");
        var passivePolicy = ParseStringSetting(backsource, "upstream_passive_health_check_policy")
            ?? GetDefaultValue(defaults, "upstream_passive_health_check_policy");
        var passiveRateLimit = ParseDoubleSetting(backsource, "upstream_passive_health_check_rate_limit")
            ?? ParseDefaultDouble(defaults, "upstream_passive_health_check_rate_limit");
        var availablePolicy = ParseStringSetting(backsource, "upstream_available_destinations_policy")
            ?? GetDefaultValue(defaults, "upstream_available_destinations_policy");

        return new HealthRuntimeConfig(
            activeEnabled,
            SanitizeNginxValue(activePath),
            SanitizeNginxValue(activeInterval),
            SanitizeNginxValue(activeTimeout),
            SanitizeNginxValue(activePolicy),
            activeThreshold.HasValue && activeThreshold.Value > 0 ? activeThreshold : null,
            passiveEnabled,
            SanitizeNginxValue(passiveReactivation),
            SanitizeNginxValue(passivePolicy),
            passiveRateLimit.HasValue && passiveRateLimit.Value > 0 && passiveRateLimit.Value < 1 ? passiveRateLimit : null,
            SanitizeNginxValue(availablePolicy));
    }

    private static double? ParseDoubleSetting(Dictionary<string, JsonElement>? map, string key)
    {
        if (map == null || !TryGetEntry(map, key, out var value))
        {
            return null;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetDouble(out var parsed))
        {
            return parsed;
        }

        if (value.ValueKind == JsonValueKind.String &&
            double.TryParse(value.GetString(), System.Globalization.NumberStyles.Float, System.Globalization.CultureInfo.InvariantCulture, out var parsedString))
        {
            return parsedString;
        }

        return null;
    }

    private static long CalcDomainLimitRate(
        Site site,
        Dictionary<int, UserPackage> userPackageMap)
    {
        if (!site.UserPackage.HasValue || !userPackageMap.TryGetValue(site.UserPackage.Value, out var pkg))
        {
            return 0;
        }

        var mbps = ParseBandwidthMbps(pkg.Bandwidth);
        if (mbps <= 0)
        {
            return 0;
        }

        return MbpsToLimitRate(mbps);
    }

    private static int CalcDomainConnLimit(
        Site site,
        Dictionary<int, UserPackage> userPackageMap,
        Dictionary<int, Dictionary<int, int>> domainCountByUserGroup,
        Dictionary<long, long> nodeGroupCounts)
    {
        var userId = site.Uid ?? 0;
        var groupId = site.NodeGroupId ?? 0;
        if (userId <= 0 || groupId <= 0)
        {
            return 0;
        }

        if (!site.UserPackage.HasValue || !userPackageMap.TryGetValue(site.UserPackage.Value, out var pkg))
        {
            return 0;
        }

        var connLimit = pkg.Connection ?? 0;
        if (connLimit <= 0)
        {
            return 0;
        }

        if (!nodeGroupCounts.TryGetValue(groupId, out var nodeCount) || nodeCount <= 0)
        {
            return 0;
        }

        if (!domainCountByUserGroup.TryGetValue(groupId, out var groupMap))
        {
            return 0;
        }

        if (!groupMap.TryGetValue(userId, out var domainCount) || domainCount <= 0)
        {
            return 0;
        }

        var perNode = (double)connLimit / nodeCount;
        var perDomain = perNode / domainCount;
        if (perDomain < 1)
        {
            return 1;
        }

        return (int)perDomain;
    }

    private static double ParseBandwidthMbps(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return 0;
        }

        var value = raw.Trim().ToLowerInvariant();
        if (value is "0" or "unlimited" or "unlimit")
        {
            return 0;
        }

        var multiplier = 1.0;
        if (value.EndsWith("g"))
        {
            multiplier = 1024;
            value = value[..^1];
        }
        else if (value.EndsWith("m"))
        {
            value = value[..^1];
        }
        else if (value.EndsWith("k"))
        {
            multiplier = 1.0 / 1024;
            value = value[..^1];
        }

        return double.TryParse(value.Trim(), out var parsed) ? parsed * multiplier : 0;
    }

    private static long MbpsToLimitRate(double mbps)
    {
        if (mbps <= 0)
        {
            return 0;
        }

        return (long)(mbps * 1024 * 1024 / 8);
    }

    private void DecryptCertKeys(IReadOnlyList<Cert> certs)
    {
        if (certs.Count == 0)
        {
            return;
        }

        foreach (var cert in certs)
        {
            if (string.IsNullOrWhiteSpace(cert.Key))
            {
                continue;
            }

            var decrypted = _cryptoService.Decrypt(cert.Key);
            if (!string.IsNullOrWhiteSpace(decrypted))
            {
                cert.Key = decrypted;
            }
        }
    }

    private static Cert? FindCertForDomain(string domain, IReadOnlyList<Cert> certs)
    {
        foreach (var cert in certs)
        {
            if (string.IsNullOrWhiteSpace(cert.Domain))
            {
                continue;
            }

            var list = cert.Domain.Split(',', StringSplitOptions.RemoveEmptyEntries);
            foreach (var entry in list)
            {
                if (string.Equals(entry.Trim(), domain, StringComparison.OrdinalIgnoreCase))
                {
                    return cert;
                }
            }
        }

        return null;
    }
    private async Task<List<EdgeStreamDto>?> BuildStreamsForNodeAsync(
        Node node,
        IReadOnlyList<long> groupIds,
        Dictionary<long, List<L2Target>> l2TargetsByGroup,
        Dictionary<long, string> groupL2Config)
    {
        if (groupIds.Count == 0)
        {
            return null;
        }

        var streams = await _db.Queryable<StreamEntity>()
            .Where(s => s.NodeGroupId.HasValue && groupIds.Contains(s.NodeGroupId.Value) && s.Enable == true)
            .ToListAsync();

        if (streams.Count == 0)
        {
            return null;
        }

        var (streamDefaults, streamDefaultsByUser) = await LoadStreamDefaultConfigAsync(streams);
        var userPackageMap = await LoadUserPackageMapAsync(streams);

        var result = new List<EdgeStreamDto>();
        foreach (var stream in streams)
        {
            var listenPorts = SplitPortList(stream.Listen);
            if (listenPorts.Count == 0)
            {
                continue;
            }

            var defaults = ResolveSiteDefaults(stream.Uid, streamDefaults, streamDefaultsByUser);
            var balanceWay = string.IsNullOrWhiteSpace(stream.BalanceWay)
                ? GetDefaultValue(defaults, "balance_way")
                : stream.BalanceWay;
            var proxyProtocol = stream.ProxyProtocol ?? false;
            var proxyProtocolDefault = ParseDefaultBool(defaults, "proxy_protocol");
            if (proxyProtocolDefault.HasValue)
            {
                proxyProtocol = proxyProtocolDefault.Value;
            }

            var settings = ParseJsonObject(stream.Acl);
            var originSettings = GetJsonObject(settings, "origin");

            var connectTimeout = ReadJsonString(originSettings, "connect_timeout");
            if (string.IsNullOrWhiteSpace(connectTimeout))
            {
                connectTimeout = "10s";
            }

            var proxyTimeout = ReadJsonString(originSettings, "proxy_timeout");
            if (string.IsNullOrWhiteSpace(proxyTimeout))
            {
                proxyTimeout = "60s";
            }

            var connLimit = ReadJsonInt(originSettings, "conn_limit");

            var targets = new List<EdgeStreamTargetDto>();
            foreach (var origin in ParseForwardOrigins(stream.Backend))
            {
                if (!origin.Enable)
                {
                    continue;
                }

                if (string.IsNullOrWhiteSpace(origin.Address))
                {
                    continue;
                }

                targets.Add(new EdgeStreamTargetDto
                {
                    Addr = origin.Address,
                    Weight = origin.Weight,
                    Enable = origin.Enable
                });
            }

            if (targets.Count == 0)
            {
                continue;
            }

            var useL2 = false;
            if ((node.Level ?? 0) == 1)
            {
                var groupId = stream.NodeGroupId ?? 0;
                var groupConfig = groupL2Config.TryGetValue(groupId, out var config) ? config : string.Empty;
                var packageEnabled = false;
                if (stream.UserPackage.HasValue && userPackageMap.TryGetValue(stream.UserPackage.Value, out var pkg))
                {
                    packageEnabled = pkg.L2Origin ?? false;
                }

                useL2 = ResolveL2Enabled("current", groupConfig, packageEnabled) &&
                        l2TargetsByGroup.TryGetValue(groupId, out var l2Targets) &&
                        l2Targets.Count > 0;

                if (useL2)
                {
                    var l2List = new List<EdgeStreamTargetDto>();
                    foreach (var l2Node in l2TargetsByGroup[groupId])
                    {
                        if (string.IsNullOrWhiteSpace(l2Node.Ip))
                        {
                            continue;
                        }

                        l2List.Add(new EdgeStreamTargetDto
                        {
                            Addr = l2Node.Ip,
                            Weight = 1,
                            Enable = true,
                            NodeId = l2Node.NodeId
                        });
                    }

                    foreach (var origin in targets)
                    {
                        origin.Backup = true;
                        l2List.Add(origin);
                    }

                    targets = l2List;
                }
            }

            result.Add(new EdgeStreamDto
            {
                Id = stream.Id,
                ListenPorts = listenPorts,
                Targets = targets,
                UseListenPort = useL2 ? true : null,
                BalanceWay = string.IsNullOrWhiteSpace(balanceWay) ? null : balanceWay.Trim(),
                ProxyProtocol = proxyProtocol,
                ProxyConnectTimeout = connectTimeout,
                ProxyTimeout = proxyTimeout,
                ConnLimit = connLimit > 0 ? connLimit : null
            });
        }

        return result.Count == 0 ? null : result;
    }

    private sealed record ForwardOrigin(string Address, int Weight, bool Enable);

    private static Dictionary<string, JsonElement>? ParseJsonObject(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            return JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(raw, JsonOptions);
        }
        catch
        {
            return null;
        }
    }

    private static JsonElement? GetJsonObject(Dictionary<string, JsonElement>? map, string key)
    {
        if (map == null)
        {
            return null;
        }

        return map.TryGetValue(key, out var value) && value.ValueKind == JsonValueKind.Object ? value : null;
    }

    private static string? ReadJsonString(JsonElement? element, string key)
    {
        if (element == null)
        {
            return null;
        }

        return ReadJsonString(element.Value, key);
    }

    private static string? ReadJsonString(JsonElement element, string key)
    {
        if (element.ValueKind != JsonValueKind.Object)
        {
            return null;
        }

        if (!element.TryGetProperty(key, out var value))
        {
            return null;
        }

        return value.ValueKind == JsonValueKind.String ? value.GetString() : value.ToString();
    }

    private static int ReadJsonInt(JsonElement? element, string key)
    {
        if (element == null)
        {
            return 0;
        }

        return ReadJsonInt(element.Value, key, 0);
    }

    private static int ReadJsonInt(JsonElement element, string key, int fallback)
    {
        if (element.ValueKind != JsonValueKind.Object)
        {
            return fallback;
        }

        if (!element.TryGetProperty(key, out var value))
        {
            return fallback;
        }

        return value.ValueKind switch
        {
            JsonValueKind.Number when value.TryGetInt32(out var parsed) => parsed,
            JsonValueKind.String when int.TryParse(value.GetString(), out var parsed) => parsed,
            _ => fallback
        };
    }

    private static bool ReadJsonBool(JsonElement element, string key, bool fallback)
    {
        if (element.ValueKind != JsonValueKind.Object)
        {
            return fallback;
        }

        if (!element.TryGetProperty(key, out var value))
        {
            return fallback;
        }

        return value.ValueKind switch
        {
            JsonValueKind.True => true,
            JsonValueKind.False => false,
            JsonValueKind.Number => value.TryGetInt32(out var parsed) && parsed != 0,
            JsonValueKind.String => ParseBoolFlag(value.GetString(), fallback),
            _ => fallback
        };
    }

    private static List<ForwardOrigin> ParseForwardOrigins(string? raw)
    {
        var result = new List<ForwardOrigin>();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return result;
        }

        var trimmed = raw.Trim();
        if (trimmed.StartsWith("["))
        {
            try
            {
                using var doc = JsonDocument.Parse(trimmed);
                if (doc.RootElement.ValueKind == JsonValueKind.Array)
                {
                    foreach (var item in doc.RootElement.EnumerateArray())
                    {
                        if (item.ValueKind == JsonValueKind.Object)
                        {
                            var address = ReadJsonString(item, "address") ?? ReadJsonString(item, "addr");
                            if (string.IsNullOrWhiteSpace(address))
                            {
                                continue;
                            }

                            var weight = ReadJsonInt(item, "weight", 1);
                            var enable = ReadJsonBool(item, "enable", true);
                            result.Add(new ForwardOrigin(address, weight, enable));
                            continue;
                        }

                        if (item.ValueKind == JsonValueKind.String)
                        {
                            var address = item.GetString();
                            if (!string.IsNullOrWhiteSpace(address))
                            {
                                result.Add(new ForwardOrigin(address, 1, true));
                            }
                        }
                    }

                    return result;
                }
            }
            catch
            {
            }
        }

        foreach (var item in SplitFields(trimmed))
        {
            if (!string.IsNullOrWhiteSpace(item))
            {
                result.Add(new ForwardOrigin(item, 1, true));
            }
        }

        return result;
    }

    private static int ParseIntValue(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return 0;
        }

        return int.TryParse(raw.Trim(), out var parsed) ? parsed : 0;
    }

    private static List<string> SplitDomainList(string? raw)
    {
        var list = SplitStringList(raw);
        var result = new List<string>();
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var item in list)
        {
            var domain = DomainHelper.NormalizeDomainInput(item);
            if (string.IsNullOrWhiteSpace(domain))
            {
                continue;
            }

            if (seen.Add(domain))
            {
                result.Add(domain);
            }
        }

        return result;
    }

    private static List<string> SplitPortList(string? raw)
    {
        var list = SplitStringList(raw);
        var result = new List<string>();
        foreach (var item in list)
        {
            var trimmed = item.Trim();
            if (!string.IsNullOrWhiteSpace(trimmed))
            {
                result.Add(trimmed);
            }
        }

        return result;
    }

    private static List<string> SplitOriginList(string? raw)
    {
        var list = SplitStringList(raw);
        var result = new List<string>();
        foreach (var item in list)
        {
            var trimmed = item.Trim();
            if (!string.IsNullOrWhiteSpace(trimmed))
            {
                result.Add(trimmed);
            }
        }

        return result;
    }

    private static List<string> SplitStringList(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new List<string>();
        }

        var trimmed = raw.Trim();
        if (trimmed.StartsWith("["))
        {
            try
            {
                var list = JsonSerializer.Deserialize<List<string>>(trimmed, JsonOptions);
                if (list != null)
                {
                    return list;
                }
            }
            catch
            {
            }

            try
            {
                using var doc = JsonDocument.Parse(trimmed);
                if (doc.RootElement.ValueKind == JsonValueKind.Array)
                {
                    var list = new List<string>();
                    foreach (var item in doc.RootElement.EnumerateArray())
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

                    return list;
                }
            }
            catch
            {
            }
        }

        return SplitFields(trimmed);
    }

    private static List<string> SplitFields(string raw)
    {
        var parts = raw.Split(new[] { ',', ';', '\n', '\r', '\t', ' ' }, StringSplitOptions.RemoveEmptyEntries);
        var result = new List<string>();
        foreach (var part in parts)
        {
            var item = part.Trim();
            if (!string.IsNullOrWhiteSpace(item))
            {
                result.Add(item);
            }
        }
        return result;
    }

    private static List<string>? ParseJsonStringList(JsonElement element)
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
    private async Task<EdgeNginxConfigDto?> LoadNginxConfigAsync(CancellationToken cancellationToken)
    {
        var values = await _db.Queryable<Config>()
            .Where(c => c.Type == "nginx_config" && c.ScopeName == "global" && c.ScopeId == 0)
            .ToListAsync();

        if (values.Count == 0)
        {
            return null;
        }

        var map = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var item in values)
        {
            if (string.IsNullOrWhiteSpace(item.Name))
            {
                continue;
            }

            map[item.Name] = item.Value ?? string.Empty;
        }

        if (!map.TryGetValue("nginx-config-file", out var raw) || string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            return JsonSerializer.Deserialize<EdgeNginxConfigDto>(raw, JsonOptions);
        }
        catch
        {
            return null;
        }
    }

    private static bool ParseBoolFlag(string? raw, bool fallback)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return fallback;
        }

        var normalized = raw.Trim().ToLowerInvariant();
        return normalized is "1" or "true" or "yes" or "on";
    }

    private static string SanitizeHeaderName(string? name)
    {
        if (string.IsNullOrWhiteSpace(name))
        {
            return string.Empty;
        }

        foreach (var c in name)
        {
            if (char.IsLetterOrDigit(c) || c == '-' || c == '_')
            {
                continue;
            }

            return string.Empty;
        }

        return name.Trim();
    }

    private static string SanitizeHeaderValue(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return string.Empty;
        }

        var trimmed = value.Trim();
        return trimmed.IndexOfAny(new[] { '\r', '\n', ';' }) >= 0 ? string.Empty : trimmed;
    }

    private static string? SanitizeNginxValue(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return null;
        }

        var trimmed = value.Trim();
        return trimmed.IndexOfAny(new[] { '\r', '\n', ';' }) >= 0 ? null : trimmed;
    }

    private static string? SanitizeProxyHttpVersion(string? value)
    {
        var normalized = SanitizeNginxValue(value);
        return normalized is "1.0" or "1.1" ? normalized : null;
    }

    private static Dictionary<string, string>? ParseHeaderMap(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            var result = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
            if (doc.RootElement.ValueKind == JsonValueKind.Object)
            {
                foreach (var prop in doc.RootElement.EnumerateObject())
                {
                    var name = SanitizeHeaderName(prop.Name);
                    if (string.IsNullOrWhiteSpace(name))
                    {
                        continue;
                    }

                    var value = prop.Value.ValueKind == JsonValueKind.String
                        ? prop.Value.GetString()
                        : prop.Value.ToString();
                    value = SanitizeHeaderValue(value);
                    if (string.IsNullOrWhiteSpace(value))
                    {
                        continue;
                    }

                    result[name] = value;
                }
            }
            else if (doc.RootElement.ValueKind == JsonValueKind.Array)
            {
                foreach (var item in doc.RootElement.EnumerateArray())
                {
                    if (item.ValueKind != JsonValueKind.Object)
                    {
                        continue;
                    }

                    if (!item.TryGetProperty("name", out var nameValue))
                    {
                        continue;
                    }

                    var name = SanitizeHeaderName(nameValue.GetString());
                    if (string.IsNullOrWhiteSpace(name))
                    {
                        continue;
                    }

                    if (!item.TryGetProperty("value", out var valueValue))
                    {
                        continue;
                    }

                    var value = SanitizeHeaderValue(valueValue.GetString());
                    if (string.IsNullOrWhiteSpace(value))
                    {
                        continue;
                    }

                    result[name] = value;
                }
            }

            return result.Count == 0 ? null : result;
        }
        catch
        {
            return null;
        }
    }
}
