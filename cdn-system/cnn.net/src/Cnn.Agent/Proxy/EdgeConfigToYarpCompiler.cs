using System.Globalization;
using System.Net;
using System.Net.Http;
using System.Security.Authentication;
using System.Security.Cryptography;
using System.Text;
using Cnn.Common.Contracts.Agent;
using Yarp.ReverseProxy.Forwarder;
using Yarp.ReverseProxy.Configuration;
using Yarp.ReverseProxy.Health;

namespace Cnn.Agent.Proxy;

public sealed class EdgeConfigToYarpCompiler
{
    public ProxySnapshot Compile(EdgeConfigDto config)
    {
        var domains = config.Domains ?? [];
        if (domains.Count == 0)
        {
            return ProxySnapshot.CreateFallback() with { Version = config.Version, Hash = ComputeHash(config), CreatedAt = DateTimeOffset.UtcNow };
        }

        var routes = new List<RouteConfig>(domains.Count);
        var upstreamDestinations = BuildUpstreamDestinations(config);
        var clusters = new List<ClusterConfig>();
        var clusterByUpstreamAndPolicy = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);

        var routeIds = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var domain in domains)
        {
            var host = NormalizeHost(domain.Name);
            if (string.IsNullOrWhiteSpace(host))
            {
                continue;
            }

            var routeId = BuildUniqueRouteId(routeIds, $"route:{host}");
            var upstreamKey = domain.UpstreamKey?.Trim() ?? string.Empty;
            if (!upstreamDestinations.TryGetValue(upstreamKey, out var destinations))
            {
                continue;
            }

            var lbPolicy = ResolveLoadBalancingPolicy(domain.LoadBalancePolicy, destinations);
            var runtimeOptions = CompileRuntimeOptions(domain);
            var clusterId = EnsureCluster(
                clusters,
                clusterByUpstreamAndPolicy,
                upstreamKey,
                lbPolicy,
                destinations,
                domain,
                runtimeOptions);
            var transforms = BuildTransforms(domain);

            routes.Add(new RouteConfig
            {
                RouteId = routeId,
                ClusterId = clusterId,
                Match = new RouteMatch
                {
                    Hosts = new[] { host },
                    Path = "/{**catch-all}"
                },
                Timeout = ParseDuration(domain.ProxyReadTimeout),
                Transforms = transforms
            });
        }

        if (routes.Count == 0)
        {
            return ProxySnapshot.CreateFallback() with { Version = config.Version, Hash = ComputeHash(config), CreatedAt = DateTimeOffset.UtcNow };
        }

        return new ProxySnapshot(
            Version: config.Version,
            Hash: ComputeHash(config),
            CreatedAt: DateTimeOffset.UtcNow,
            Routes: routes,
            Clusters: clusters,
            DomainCount: domains.Count,
            UpstreamCount: upstreamDestinations.Count,
            IsFallbackMode: false);
    }

    private static Dictionary<string, IReadOnlyList<UpstreamTargetPlan>> BuildUpstreamDestinations(EdgeConfigDto config)
    {
        var upstreamDestinations = new Dictionary<string, IReadOnlyList<UpstreamTargetPlan>>(StringComparer.OrdinalIgnoreCase);

        foreach (var upstream in config.Upstreams ?? [])
        {
            var upstreamId = upstream.Id?.Trim() ?? string.Empty;
            if (string.IsNullOrWhiteSpace(upstreamId))
            {
                continue;
            }

            var targets = new List<UpstreamTargetPlan>();
            foreach (var target in upstream.Targets ?? [])
            {
                var addr = target.Addr?.Trim();
                if (string.IsNullOrWhiteSpace(addr))
                {
                    continue;
                }

                targets.Add(new UpstreamTargetPlan(addr, NormalizeWeight(target.Weight), target.NodeId));
            }

            if (targets.Count == 0)
            {
                continue;
            }

            upstreamDestinations[upstreamId] = targets;
        }

        return upstreamDestinations;
    }

    private static List<IReadOnlyDictionary<string, string>> BuildTransforms(EdgeDomainDto domain)
    {
        var transforms = new List<IReadOnlyDictionary<string, string>>();
        var siteType = domain.SiteType?.Trim().ToLowerInvariant() ?? "website";
        var isWebsite = string.Equals(siteType, "website", StringComparison.OrdinalIgnoreCase);
        var isWebSocket = domain.EnableWebsocket == true;

        if (domain.Headers != null)
        {
            foreach (var header in domain.Headers)
            {
                if (string.IsNullOrWhiteSpace(header.Key))
                {
                    continue;
                }

                transforms.Add(new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
                {
                    ["RequestHeader"] = header.Key.Trim(),
                    ["Set"] = header.Value ?? string.Empty
                });
            }
        }

        if (domain.ResponseHeaders != null)
        {
            foreach (var header in domain.ResponseHeaders)
            {
                if (string.IsNullOrWhiteSpace(header.Key))
                {
                    continue;
                }

                transforms.Add(new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
                {
                    ["ResponseHeader"] = header.Key.Trim(),
                    ["Set"] = header.Value ?? string.Empty
                });
            }
        }

        if (isWebsite && !isWebSocket)
        {
            transforms.Add(new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
            {
                ["ResponseHeaderRemove"] = "Upgrade"
            });
            transforms.Add(new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
            {
                ["ResponseHeaderRemove"] = "Connection"
            });
            transforms.Add(new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
            {
                ["ResponseHeaderRemove"] = "Keep-Alive"
            });
            transforms.Add(new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
            {
                ["ResponseHeaderRemove"] = "Proxy-Connection"
            });
        }

        return transforms;
    }

    private static string BuildUniqueRouteId(HashSet<string> existing, string baseId)
    {
        if (existing.Add(baseId))
        {
            return baseId;
        }

        var i = 1;
        while (true)
        {
            var candidate = $"{baseId}:{i}";
            if (existing.Add(candidate))
            {
                return candidate;
            }

            i++;
        }
    }

    private static string NormalizeHost(string host)
    {
        var normalized = host.Trim().TrimEnd('.').ToLowerInvariant();
        if (normalized.Length == 0)
        {
            return string.Empty;
        }

        try
        {
            return s_idn.GetAscii(normalized);
        }
        catch
        {
            return normalized;
        }
    }

    private static string NormalizeAddress(UpstreamTargetPlan target, EdgeDomainDto domain)
    {
        var addr = target.RawAddress.Trim();
        if (addr.Contains("://", StringComparison.Ordinal))
        {
            return addr;
        }

        var originScheme = ParseOriginScheme(domain.OriginProtocol);
        var originPort = ResolveOriginPort(addr, originScheme, domain.OriginHttpPort, domain.OriginHttpsPort);
        if (originPort.HasValue)
        {
            return $"{originScheme}://{addr}:{originPort.Value}";
        }

        return $"{originScheme}://{addr}";
    }

    private static string ComputeHash(EdgeConfigDto config)
    {
        var raw = $"{config.Version}:{config.Domains.Count}:{config.Upstreams.Count}";
        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes(raw));
        return Convert.ToHexString(bytes);
    }

    private static string EnsureCluster(
        List<ClusterConfig> clusters,
        Dictionary<string, string> cache,
        string upstreamKey,
        string yarpPolicy,
        IReadOnlyList<UpstreamTargetPlan> targets,
        EdgeDomainDto domain,
        ClusterRuntimeOptions runtimeOptions)
    {
        var key = $"{upstreamKey}|{yarpPolicy}|{runtimeOptions.CacheKey}";
        if (cache.TryGetValue(key, out var existingClusterId))
        {
            return existingClusterId;
        }

        var policyTag = yarpPolicy.ToLowerInvariant();
        var clusterId = $"cluster:{upstreamKey}:{policyTag}:{ShortHash(key)}";
        cache[key] = clusterId;

        var clonedDestinations = new Dictionary<string, DestinationConfig>(StringComparer.OrdinalIgnoreCase);
        for (var i = 0; i < targets.Count; i++)
        {
            var target = targets[i];
            var destinationMetadata = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
            if (target.Weight > 0)
            {
                destinationMetadata["weight"] = target.Weight.ToString(CultureInfo.InvariantCulture);
            }

            if (target.NodeId.HasValue)
            {
                destinationMetadata["node_id"] = target.NodeId.Value.ToString(CultureInfo.InvariantCulture);
            }

            clonedDestinations[$"dest-{i}"] = new DestinationConfig
            {
                Address = NormalizeAddress(target, domain),
                Metadata = destinationMetadata.Count == 0 ? null : destinationMetadata
            };
        }

        var metadata = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
        {
            ["upstream_key"] = upstreamKey,
            ["load_balance_policy"] = yarpPolicy
        };
        foreach (var (k, v) in runtimeOptions.Metadata)
        {
            metadata[k] = v;
        }

        clusters.Add(new ClusterConfig
        {
            ClusterId = clusterId,
            LoadBalancingPolicy = yarpPolicy,
            Destinations = clonedDestinations,
            HttpClient = runtimeOptions.HttpClient,
            HttpRequest = runtimeOptions.HttpRequest,
            HealthCheck = runtimeOptions.HealthCheck,
            Metadata = metadata
        });

        return clusterId;
    }

    private static ClusterRuntimeOptions CompileRuntimeOptions(EdgeDomainDto domain)
    {
        var connectTimeout = ParseDuration(domain.ProxyConnectTimeout);
        var readTimeout = ParseDuration(domain.ProxyReadTimeout);
        var sendTimeout = ParseDuration(domain.ProxySendTimeout);
        var activityTimeout = BuildActivityTimeout(readTimeout, sendTimeout);

        var parsedHttpVersion = ParseHttpVersion(domain.ProxyHttpVersion);
        ForwarderRequestConfig? requestConfig = null;
        if (activityTimeout.HasValue || parsedHttpVersion != null)
        {
            requestConfig = new ForwarderRequestConfig
            {
                ActivityTimeout = activityTimeout,
                Version = parsedHttpVersion?.Version,
                VersionPolicy = parsedHttpVersion?.Policy
            };
        }

        var sslProtocols = ParseSslProtocols(domain.ProxySslProtocols);
        var maxConnectionsPerServer = domain.UpstreamKeepaliveConn.GetValueOrDefault() > 0
            ? domain.UpstreamKeepaliveConn
            : null;
        HttpClientConfig? httpClientConfig = null;
        if (sslProtocols.HasValue || maxConnectionsPerServer.HasValue)
        {
            httpClientConfig = new HttpClientConfig
            {
                SslProtocols = sslProtocols,
                MaxConnectionsPerServer = maxConnectionsPerServer
            };
        }

        var healthCheck = CompileHealthCheck(domain);
        var metadata = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        AddIfNotEmpty(metadata, "origin_protocol", ParseOriginScheme(domain.OriginProtocol));
        AddIfNotEmpty(metadata, "origin_http_port", domain.OriginHttpPort);
        AddIfNotEmpty(metadata, "origin_https_port", domain.OriginHttpsPort);
        AddIfNotEmpty(metadata, "proxy_connect_timeout", domain.ProxyConnectTimeout);
        AddIfNotEmpty(metadata, "proxy_read_timeout", domain.ProxyReadTimeout);
        AddIfNotEmpty(metadata, "proxy_send_timeout", domain.ProxySendTimeout);
        if (activityTimeout.HasValue)
        {
            metadata["proxy_activity_timeout_ms"] = Math.Round(activityTimeout.Value.TotalMilliseconds)
                .ToString(CultureInfo.InvariantCulture);
        }

        AddIfNotEmpty(metadata, "proxy_http_version", parsedHttpVersion?.Tag ?? string.Empty);
        if (sslProtocols.HasValue)
        {
            metadata["proxy_ssl_protocols"] = sslProtocols.Value.ToString();
        }

        if (maxConnectionsPerServer.HasValue)
        {
            metadata["upstream_keepalive_conn"] = maxConnectionsPerServer.Value.ToString(CultureInfo.InvariantCulture);
        }

        if (domain.UpstreamKeepalive.HasValue)
        {
            metadata["upstream_keepalive"] = domain.UpstreamKeepalive.Value ? "true" : "false";
        }

        if (domain.UpstreamKeepaliveTimeout.HasValue && domain.UpstreamKeepaliveTimeout.Value > 0)
        {
            metadata["upstream_keepalive_timeout"] = domain.UpstreamKeepaliveTimeout.Value.ToString(CultureInfo.InvariantCulture);
        }

        if (healthCheck?.Active?.Enabled == true)
        {
            metadata["upstream_active_health_check"] = "true";
            AddIfNotEmpty(metadata, "upstream_active_health_check_path", healthCheck.Active.Path);
            AddIfNotEmpty(metadata, "upstream_active_health_check_query", healthCheck.Active.Query);
            AddIfNotEmpty(metadata, "upstream_active_health_check_policy", healthCheck.Active.Policy);
            if (domain.UpstreamActiveHealthCheckThreshold.HasValue && domain.UpstreamActiveHealthCheckThreshold.Value > 0)
            {
                var threshold = domain.UpstreamActiveHealthCheckThreshold.Value.ToString(CultureInfo.InvariantCulture);
                metadata["upstream_active_health_check_threshold"] = threshold;
                metadata[ConsecutiveFailuresHealthPolicyOptions.ThresholdMetadataName] = threshold;
            }

            if (healthCheck.Active.Interval.HasValue)
            {
                metadata["upstream_active_health_check_interval"] = FormatDuration(healthCheck.Active.Interval);
            }

            if (healthCheck.Active.Timeout.HasValue)
            {
                metadata["upstream_active_health_check_timeout"] = FormatDuration(healthCheck.Active.Timeout);
            }
        }

        if (healthCheck?.Passive?.Enabled == true)
        {
            metadata["upstream_passive_health_check"] = "true";
            AddIfNotEmpty(metadata, "upstream_passive_health_check_policy", healthCheck.Passive.Policy);
            if (domain.UpstreamPassiveHealthCheckRateLimit.HasValue &&
                domain.UpstreamPassiveHealthCheckRateLimit.Value > 0 &&
                domain.UpstreamPassiveHealthCheckRateLimit.Value < 1)
            {
                var rateLimit = domain.UpstreamPassiveHealthCheckRateLimit.Value.ToString("0.###", CultureInfo.InvariantCulture);
                metadata["upstream_passive_health_check_rate_limit"] = rateLimit;
                metadata[TransportFailureRateHealthPolicyOptions.FailureRateLimitMetadataName] = rateLimit;
            }

            if (healthCheck.Passive.ReactivationPeriod.HasValue)
            {
                metadata["upstream_passive_health_check_reactivation"] = FormatDuration(healthCheck.Passive.ReactivationPeriod);
            }
        }

        AddIfNotEmpty(metadata, "upstream_available_destinations_policy", healthCheck?.AvailableDestinationsPolicy);

        var cacheKey = string.Create(
            CultureInfo.InvariantCulture,
            $"origin={ParseOriginScheme(domain.OriginProtocol)};" +
            $"httpPort={NormalizeNumberOrEmpty(domain.OriginHttpPort)};" +
            $"httpsPort={NormalizeNumberOrEmpty(domain.OriginHttpsPort)};" +
            $"connect={FormatDuration(connectTimeout)};" +
            $"read={FormatDuration(readTimeout)};" +
            $"send={FormatDuration(sendTimeout)};" +
            $"activity={FormatDuration(activityTimeout)};" +
            $"httpVersion={parsedHttpVersion?.Tag ?? "default"};" +
            $"ssl={(int)(sslProtocols ?? SslProtocols.None)};" +
            $"keepaliveConn={maxConnectionsPerServer?.ToString(CultureInfo.InvariantCulture) ?? "0"};" +
            $"keepalive={domain.UpstreamKeepalive?.ToString() ?? "null"};" +
            $"keepaliveTimeout={domain.UpstreamKeepaliveTimeout?.ToString(CultureInfo.InvariantCulture) ?? "0"};" +
            $"activeHealth={(healthCheck?.Active?.Enabled == true ? 1 : 0)};" +
            $"activePolicy={healthCheck?.Active?.Policy ?? "none"};" +
            $"activeThreshold={domain.UpstreamActiveHealthCheckThreshold?.ToString(CultureInfo.InvariantCulture) ?? "none"};" +
            $"activePath={healthCheck?.Active?.Path ?? "none"};" +
            $"activeInterval={FormatDuration(healthCheck?.Active?.Interval)};" +
            $"activeTimeout={FormatDuration(healthCheck?.Active?.Timeout)};" +
            $"passiveHealth={(healthCheck?.Passive?.Enabled == true ? 1 : 0)};" +
            $"passivePolicy={healthCheck?.Passive?.Policy ?? "none"};" +
            $"passiveRateLimit={domain.UpstreamPassiveHealthCheckRateLimit?.ToString("0.###", CultureInfo.InvariantCulture) ?? "none"};" +
            $"passiveReactivation={FormatDuration(healthCheck?.Passive?.ReactivationPeriod)};" +
            $"availablePolicy={healthCheck?.AvailableDestinationsPolicy ?? "none"}");

        return new ClusterRuntimeOptions(cacheKey, httpClientConfig, requestConfig, healthCheck, metadata);
    }

    private static string ResolveLoadBalancingPolicy(string? policy, IReadOnlyList<UpstreamTargetPlan> targets)
    {
        var mapped = MapLoadBalancingPolicy(policy);
        if (mapped == "RoundRobin" && ShouldUseWeightedRoundRobin(targets))
        {
            return WeightedRoundRobinLoadBalancingPolicy.PolicyName;
        }

        return mapped;
    }

    private static string MapLoadBalancingPolicy(string? policy)
    {
        return NormalizePolicy(policy) switch
        {
            "least_conn" => "LeastRequests",
            "least_requests" => "LeastRequests",
            "random" => "Random",
            "ip_hash" => ClientIpHashLoadBalancingPolicy.PolicyName,
            "consistent_hash" => ClientIpHashLoadBalancingPolicy.PolicyName,
            _ => "RoundRobin"
        };
    }

    private static bool ShouldUseWeightedRoundRobin(IReadOnlyList<UpstreamTargetPlan> targets)
    {
        if (targets == null || targets.Count <= 1)
        {
            return false;
        }

        var baseline = targets[0].Weight;
        for (var i = 1; i < targets.Count; i++)
        {
            if (targets[i].Weight != baseline)
            {
                return true;
            }
        }

        return false;
    }

    private static string NormalizePolicy(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return "round_robin";
        }

        return value
            .Trim()
            .Replace("-", "_", StringComparison.Ordinal)
            .Replace(" ", "_", StringComparison.Ordinal)
            .ToLowerInvariant();
    }

    private static TimeSpan? ParseDuration(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        var value = raw.Trim().ToLowerInvariant();
        if (value.EndsWith("ms", StringComparison.Ordinal)
            && double.TryParse(value[..^2], NumberStyles.Float, CultureInfo.InvariantCulture, out var ms)
            && ms > 0)
        {
            return TimeSpan.FromMilliseconds(ms);
        }

        if (value.EndsWith("s", StringComparison.Ordinal)
            && double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var sec)
            && sec > 0)
        {
            return TimeSpan.FromSeconds(sec);
        }

        if (value.EndsWith("m", StringComparison.Ordinal)
            && double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var minute)
            && minute > 0)
        {
            return TimeSpan.FromMinutes(minute);
        }

        if (value.EndsWith("h", StringComparison.Ordinal)
            && double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var hour)
            && hour > 0)
        {
            return TimeSpan.FromHours(hour);
        }

        if (double.TryParse(value, NumberStyles.Float, CultureInfo.InvariantCulture, out var seconds) && seconds > 0)
        {
            return TimeSpan.FromSeconds(seconds);
        }

        if (TimeSpan.TryParse(value, CultureInfo.InvariantCulture, out var parsed) && parsed > TimeSpan.Zero)
        {
            return parsed;
        }

        return null;
    }

    private static TimeSpan? BuildActivityTimeout(TimeSpan? readTimeout, TimeSpan? sendTimeout)
    {
        if (readTimeout.HasValue && sendTimeout.HasValue)
        {
            return readTimeout.Value >= sendTimeout.Value ? readTimeout.Value : sendTimeout.Value;
        }

        return readTimeout ?? sendTimeout;
    }

    private static ParsedHttpVersion? ParseHttpVersion(string? raw)
    {
        var normalized = NormalizeVersion(raw);
        return normalized switch
        {
            "1" or "1.0" or "http/1" or "http/1.0" => new ParsedHttpVersion(HttpVersion.Version10, HttpVersionPolicy.RequestVersionExact, "1.0"),
            "1.1" or "http/1.1" => new ParsedHttpVersion(HttpVersion.Version11, HttpVersionPolicy.RequestVersionExact, "1.1"),
            "2" or "2.0" or "http/2" or "h2" => new ParsedHttpVersion(HttpVersion.Version20, HttpVersionPolicy.RequestVersionOrLower, "2"),
            "3" or "3.0" or "http/3" or "h3" => new ParsedHttpVersion(HttpVersion.Version30, HttpVersionPolicy.RequestVersionOrLower, "3"),
            _ => null
        };
    }

    private static HealthCheckConfig? CompileHealthCheck(EdgeDomainDto domain)
    {
        var activeEnabled = domain.UpstreamActiveHealthCheck.GetValueOrDefault();
        var passiveEnabled = domain.UpstreamPassiveHealthCheck.GetValueOrDefault();
        if (!activeEnabled && !passiveEnabled)
        {
            return null;
        }

        ActiveHealthCheckConfig? active = null;
        if (activeEnabled)
        {
            var probePath = ParseHealthProbePath(domain.UpstreamActiveHealthCheckPath);
            var interval = ParseDuration(domain.UpstreamActiveHealthCheckInterval) ?? TimeSpan.FromSeconds(10);
            var timeout = ParseDuration(domain.UpstreamActiveHealthCheckTimeout) ?? TimeSpan.FromSeconds(3);
            active = new ActiveHealthCheckConfig
            {
                Enabled = true,
                Path = probePath.Path,
                Query = probePath.Query,
                Interval = interval,
                Timeout = timeout,
                Policy = MapActiveHealthPolicy(domain.UpstreamActiveHealthCheckPolicy)
            };
        }

        PassiveHealthCheckConfig? passive = null;
        if (passiveEnabled)
        {
            var reactivation = ParseDuration(domain.UpstreamPassiveHealthCheckReactivation) ?? TimeSpan.FromSeconds(30);
            passive = new PassiveHealthCheckConfig
            {
                Enabled = true,
                Policy = MapPassiveHealthPolicy(domain.UpstreamPassiveHealthCheckPolicy),
                ReactivationPeriod = reactivation
            };
        }

        return new HealthCheckConfig
        {
            Active = active,
            Passive = passive,
            AvailableDestinationsPolicy = MapAvailableDestinationsPolicy(domain.UpstreamAvailableDestinationsPolicy)
        };
    }

    private static SslProtocols? ParseSslProtocols(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        var parsed = SslProtocols.None;
        var parts = raw.Split([',', ';', '|', ' ', '\n', '\r', '\t'], StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
        foreach (var token in parts)
        {
            switch (NormalizeVersion(token))
            {
                case "tls1.2":
                case "tlsv1.2":
                case "tls12":
                case "tlsv12":
                    parsed |= SslProtocols.Tls12;
                    break;
                case "tls1.3":
                case "tlsv1.3":
                case "tls13":
                case "tlsv13":
                    parsed |= SslProtocols.Tls13;
                    break;
            }
        }

        return parsed == SslProtocols.None ? null : parsed;
    }

    private static string ParseOriginScheme(string? raw)
    {
        return raw?.Trim().ToLowerInvariant() switch
        {
            "https" => "https",
            _ => "http"
        };
    }

    private static int? ResolveOriginPort(string rawAddress, string originScheme, string? originHttpPort, string? originHttpsPort)
    {
        if (HasExplicitPort(rawAddress))
        {
            return null;
        }

        return originScheme == "https"
            ? ParsePort(originHttpsPort)
            : ParsePort(originHttpPort);
    }

    private static bool HasExplicitPort(string rawAddress)
    {
        if (!Uri.TryCreate("tcp://" + rawAddress, UriKind.Absolute, out var uri))
        {
            return false;
        }

        return uri.Port > 0;
    }

    private static int? ParsePort(string? rawPort)
    {
        if (string.IsNullOrWhiteSpace(rawPort))
        {
            return null;
        }

        return int.TryParse(rawPort.Trim(), NumberStyles.Integer, CultureInfo.InvariantCulture, out var port)
               && port > 0
               && port <= 65535
            ? port
            : null;
    }

    private static string NormalizeVersion(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return string.Empty;
        }

        return value.Trim()
            .ToLowerInvariant()
            .Replace("_", ".", StringComparison.Ordinal)
            .Replace("tlsv", "tlsv", StringComparison.Ordinal)
            .Replace("http ", "http/", StringComparison.Ordinal)
            .Replace("http:", "http/", StringComparison.Ordinal)
            .Replace("http//", "http/", StringComparison.Ordinal);
    }

    private static HealthProbePath ParseHealthProbePath(string? path)
    {
        if (string.IsNullOrWhiteSpace(path))
        {
            return new HealthProbePath("/", null);
        }

        var normalized = path.Trim();
        if (!normalized.StartsWith("/", StringComparison.Ordinal))
        {
            normalized = "/" + normalized;
        }

        var queryIndex = normalized.IndexOf('?', StringComparison.Ordinal);
        if (queryIndex < 0)
        {
            return new HealthProbePath(normalized, null);
        }

        var normalizedPath = normalized[..queryIndex];
        if (string.IsNullOrWhiteSpace(normalizedPath))
        {
            normalizedPath = "/";
        }

        var query = queryIndex + 1 < normalized.Length ? normalized[(queryIndex + 1)..] : string.Empty;
        return new HealthProbePath(normalizedPath, query);
    }

    private static string MapActiveHealthPolicy(string? raw)
    {
        return raw?.Trim().ToLowerInvariant() switch
        {
            "consecutive_failures" or "consecutivefailures" => HealthCheckConstants.ActivePolicy.ConsecutiveFailures,
            _ => HealthCheckConstants.ActivePolicy.ConsecutiveFailures
        };
    }

    private static string MapPassiveHealthPolicy(string? raw)
    {
        return raw?.Trim().ToLowerInvariant() switch
        {
            "transport_failure_rate" or "transportfailurerate" => HealthCheckConstants.PassivePolicy.TransportFailureRate,
            _ => HealthCheckConstants.PassivePolicy.TransportFailureRate
        };
    }

    private static string MapAvailableDestinationsPolicy(string? raw)
    {
        return raw?.Trim().ToLowerInvariant() switch
        {
            "healthyandunknown" or "healthy_and_unknown" => HealthCheckConstants.AvailableDestinations.HealthyAndUnknown,
            "healthyorpanic" or "healthy_or_panic" => HealthCheckConstants.AvailableDestinations.HealthyOrPanic,
            _ => HealthCheckConstants.AvailableDestinations.HealthyOrPanic
        };
    }

    private static void AddIfNotEmpty(Dictionary<string, string> target, string key, string? value)
    {
        if (!string.IsNullOrWhiteSpace(value))
        {
            target[key] = value.Trim();
        }
    }

    private static string NormalizeNumberOrEmpty(string? value)
    {
        return int.TryParse(value, NumberStyles.Integer, CultureInfo.InvariantCulture, out var number) && number > 0
            ? number.ToString(CultureInfo.InvariantCulture)
            : string.Empty;
    }

    private static string FormatDuration(TimeSpan? value)
    {
        return value.HasValue
            ? Math.Round(value.Value.TotalMilliseconds).ToString(CultureInfo.InvariantCulture)
            : "0";
    }

    private static string ShortHash(string raw)
    {
        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes(raw));
        return Convert.ToHexString(bytes[..6]).ToLowerInvariant();
    }

    private static int NormalizeWeight(int rawWeight)
    {
        if (rawWeight <= 0)
        {
            return 100;
        }

        return Math.Min(rawWeight, 10_000);
    }

    private static readonly IdnMapping s_idn = new();

    private sealed record UpstreamTargetPlan(string RawAddress, int Weight, long? NodeId);

    private sealed record ClusterRuntimeOptions(
        string CacheKey,
        HttpClientConfig? HttpClient,
        ForwarderRequestConfig? HttpRequest,
        HealthCheckConfig? HealthCheck,
        IReadOnlyDictionary<string, string> Metadata);

    private sealed record HealthProbePath(string Path, string? Query);

    private sealed record ParsedHttpVersion(Version Version, HttpVersionPolicy Policy, string Tag);
}
