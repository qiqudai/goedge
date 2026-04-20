using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts.Admin;

namespace Cnn.Common.Contracts.Agent;

public sealed class EdgeConfigDto
{
    [JsonPropertyName("version")]
    public long Version { get; set; }

    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("node_level")]
    public int? NodeLevel { get; set; }

    [JsonPropertyName("domains")]
    public List<EdgeDomainDto> Domains { get; set; } = new();

    [JsonPropertyName("upstreams")]
    public List<EdgeUpstreamDto> Upstreams { get; set; } = new();

    [JsonPropertyName("waf")]
    public WafConfigDto? Waf { get; set; }

    [JsonPropertyName("resources")]
    public GlobalResourceConfigDto? Resources { get; set; }

    [JsonPropertyName("error_pages")]
    public Dictionary<string, string>? ErrorPages { get; set; }

    [JsonPropertyName("default_config")]
    public DefaultSiteConfigDto? DefaultConfig { get; set; }

    [JsonPropertyName("cc_rules")]
    public Dictionary<long, List<EdgeCCRuleItemDto>>? CcRules { get; set; }

    [JsonPropertyName("cc_matchers")]
    public Dictionary<long, EdgeCCMatcherDto>? CcMatchers { get; set; }

    [JsonPropertyName("cc_filters")]
    public Dictionary<long, EdgeCCFilterDto>? CcFilters { get; set; }

    [JsonPropertyName("streams")]
    public List<EdgeStreamDto>? Streams { get; set; }

    [JsonPropertyName("nginx")]
    public EdgeNginxConfigDto? Nginx { get; set; }

    [JsonPropertyName("fallback_cert_data")]
    public string? FallbackCertData { get; set; }

    [JsonPropertyName("fallback_key_data")]
    public string? FallbackKeyData { get; set; }
}

public sealed class EdgeDomainDto
{
    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("upstream_key")]
    public string UpstreamKey { get; set; } = string.Empty;

    [JsonPropertyName("l2_upstream_key")]
    public string? L2UpstreamKey { get; set; }

    [JsonPropertyName("use_l2")]
    public bool? UseL2 { get; set; }

    [JsonPropertyName("l2_http_port")]
    public string? L2HttpPort { get; set; }

    [JsonPropertyName("l2_https_port")]
    public string? L2HttpsPort { get; set; }

    [JsonPropertyName("load_balance_policy")]
    public string? LoadBalancePolicy { get; set; }

    [JsonPropertyName("headers")]
    public Dictionary<string, string>? Headers { get; set; }

    [JsonPropertyName("response_headers")]
    public Dictionary<string, string>? ResponseHeaders { get; set; }

    [JsonPropertyName("hotlink")]
    public EdgeHotlinkConfigDto? Hotlink { get; set; }

    [JsonPropertyName("cors")]
    public EdgeCorsConfigDto? Cors { get; set; }

    [JsonPropertyName("cookie")]
    public EdgeCookieConfigDto? Cookie { get; set; }

    [JsonPropertyName("block_transparent_proxy")]
    public bool? BlockTransparentProxy { get; set; }

    [JsonPropertyName("crawler_action")]
    public string? CrawlerAction { get; set; }

    [JsonPropertyName("guard_pass_ttl")]
    public int? GuardPassTtl { get; set; }

    [JsonPropertyName("guard_block_ttl")]
    public int? GuardBlockTtl { get; set; }

    [JsonPropertyName("url_redirects")]
    public List<Dictionary<string, JsonElement>>? UrlRedirects { get; set; }

    [JsonPropertyName("origin_conditions")]
    public List<Dictionary<string, JsonElement>>? OriginConditions { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("conn_limit")]
    public int? ConnLimit { get; set; }

    [JsonPropertyName("ssl_cert_data")]
    public string? SslCertData { get; set; }

    [JsonPropertyName("ssl_key_data")]
    public string? SslKeyData { get; set; }

    [JsonPropertyName("ssl_cert_path")]
    public string? SslCertPath { get; set; }

    [JsonPropertyName("ssl_key_path")]
    public string? SslKeyPath { get; set; }

    [JsonPropertyName("acl_default_action")]
    public string? AclDefaultAction { get; set; }

    [JsonPropertyName("acl_rules")]
    public List<EdgeAclRuleDto>? AclRules { get; set; }

    [JsonPropertyName("black_ips")]
    public List<string>? BlackIps { get; set; }

    [JsonPropertyName("white_ips")]
    public List<string>? WhiteIps { get; set; }

    [JsonPropertyName("region_block")]
    public List<string>? RegionBlock { get; set; }

    [JsonPropertyName("cc_rule_id")]
    public long? CcRuleId { get; set; }

    [JsonPropertyName("origin_protocol")]
    public string? OriginProtocol { get; set; }

    [JsonPropertyName("origin_http_port")]
    public string? OriginHttpPort { get; set; }

    [JsonPropertyName("origin_https_port")]
    public string? OriginHttpsPort { get; set; }

    [JsonPropertyName("cache")]
    public EdgeCacheConfigDto? Cache { get; set; }

    [JsonPropertyName("http_listen")]
    public List<string>? HttpListen { get; set; }

    [JsonPropertyName("https_listen")]
    public List<string>? HttpsListen { get; set; }

    [JsonPropertyName("https_force")]
    public bool? HttpsForce { get; set; }

    [JsonPropertyName("https_redirect_port")]
    public string? HttpsRedirectPort { get; set; }

    [JsonPropertyName("https_hsts")]
    public bool? HttpsHsts { get; set; }

    [JsonPropertyName("https_http2")]
    public bool? HttpsHttp2 { get; set; }

    [JsonPropertyName("https_ocsp")]
    public bool? HttpsOcsp { get; set; }

    [JsonPropertyName("https_http3")]
    public bool? HttpsHttp3 { get; set; }

    [JsonPropertyName("https_ssl_protocols")]
    public string? HttpsSslProtocols { get; set; }

    [JsonPropertyName("https_ssl_ciphers")]
    public string? HttpsSslCiphers { get; set; }

    [JsonPropertyName("https_ssl_prefer_server_ciphers")]
    public string? HttpsSslPreferServerCiphers { get; set; }

    [JsonPropertyName("proxy_connect_timeout")]
    public string? ProxyConnectTimeout { get; set; }

    [JsonPropertyName("proxy_read_timeout")]
    public string? ProxyReadTimeout { get; set; }

    [JsonPropertyName("proxy_send_timeout")]
    public string? ProxySendTimeout { get; set; }

    [JsonPropertyName("proxy_http_version")]
    public string? ProxyHttpVersion { get; set; }

    [JsonPropertyName("proxy_ssl_protocols")]
    public string? ProxySslProtocols { get; set; }

    [JsonPropertyName("enable_gzip")]
    public bool? EnableGzip { get; set; }

    [JsonPropertyName("gzip_types")]
    public string? GzipTypes { get; set; }

    [JsonPropertyName("enable_websocket")]
    public bool? EnableWebsocket { get; set; }

    [JsonPropertyName("enable_range")]
    public bool? EnableRange { get; set; }

    [JsonPropertyName("body_limit")]
    public long? BodyLimit { get; set; }

    [JsonPropertyName("limit_rate")]
    public long? LimitRate { get; set; }

    [JsonPropertyName("upstream_keepalive")]
    public bool? UpstreamKeepalive { get; set; }

    [JsonPropertyName("upstream_keepalive_conn")]
    public int? UpstreamKeepaliveConn { get; set; }

    [JsonPropertyName("upstream_keepalive_timeout")]
    public int? UpstreamKeepaliveTimeout { get; set; }

    [JsonPropertyName("upstream_active_health_check")]
    public bool? UpstreamActiveHealthCheck { get; set; }

    [JsonPropertyName("upstream_active_health_check_path")]
    public string? UpstreamActiveHealthCheckPath { get; set; }

    [JsonPropertyName("upstream_active_health_check_interval")]
    public string? UpstreamActiveHealthCheckInterval { get; set; }

    [JsonPropertyName("upstream_active_health_check_timeout")]
    public string? UpstreamActiveHealthCheckTimeout { get; set; }

    [JsonPropertyName("upstream_active_health_check_policy")]
    public string? UpstreamActiveHealthCheckPolicy { get; set; }

    [JsonPropertyName("upstream_active_health_check_threshold")]
    public int? UpstreamActiveHealthCheckThreshold { get; set; }

    [JsonPropertyName("upstream_passive_health_check")]
    public bool? UpstreamPassiveHealthCheck { get; set; }

    [JsonPropertyName("upstream_passive_health_check_reactivation")]
    public string? UpstreamPassiveHealthCheckReactivation { get; set; }

    [JsonPropertyName("upstream_passive_health_check_policy")]
    public string? UpstreamPassiveHealthCheckPolicy { get; set; }

    [JsonPropertyName("upstream_passive_health_check_rate_limit")]
    public double? UpstreamPassiveHealthCheckRateLimit { get; set; }

    [JsonPropertyName("upstream_available_destinations_policy")]
    public string? UpstreamAvailableDestinationsPolicy { get; set; }
}

public sealed class EdgeHotlinkConfigDto
{
    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("scope")]
    public string? Scope { get; set; }

    [JsonPropertyName("value")]
    public string? Value { get; set; }

    [JsonPropertyName("allow_empty")]
    public bool? AllowEmpty { get; set; }

    [JsonPropertyName("domains")]
    public List<string>? Domains { get; set; }
}

public sealed class EdgeCorsConfigDto
{
    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("allow_origin")]
    public string? AllowOrigin { get; set; }

    [JsonPropertyName("allow_methods")]
    public string? AllowMethods { get; set; }

    [JsonPropertyName("allow_headers")]
    public string? AllowHeaders { get; set; }

    [JsonPropertyName("expose_headers")]
    public string? ExposeHeaders { get; set; }

    [JsonPropertyName("allow_credentials")]
    public bool? AllowCredentials { get; set; }

    [JsonPropertyName("max_age")]
    public string? MaxAge { get; set; }
}

public sealed class EdgeCookieConfigDto
{
    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }
}

public sealed class EdgeUpstreamDto
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("targets")]
    public List<EdgeUpstreamTargetDto> Targets { get; set; } = new();
}

public sealed class EdgeUpstreamTargetDto
{
    [JsonPropertyName("addr")]
    public string Addr { get; set; } = string.Empty;

    [JsonPropertyName("weight")]
    public int Weight { get; set; }

    [JsonPropertyName("node_id")]
    public long? NodeId { get; set; }
}

public sealed class EdgeAclRuleDto
{
    [JsonPropertyName("ip")]
    public string Ip { get; set; } = string.Empty;

    [JsonPropertyName("action")]
    public string Action { get; set; } = string.Empty;
}

public sealed class EdgeCCRuleItemDto
{
    [JsonPropertyName("matcher_id")]
    public long? MatcherId { get; set; }

    [JsonPropertyName("filter_id")]
    public long? FilterId { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("enabled")]
    public bool Enabled { get; set; }
}

public sealed class EdgeCCMatcherDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("data")]
    public string? Data { get; set; }
}

public sealed class EdgeCCFilterDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("within_second")]
    public int WithinSecond { get; set; }

    [JsonPropertyName("max_req")]
    public int MaxReq { get; set; }

    [JsonPropertyName("max_req_per_uri")]
    public int MaxReqPerUri { get; set; }

    [JsonPropertyName("extra")]
    public string? Extra { get; set; }
}

public sealed class EdgeCacheRuleDto
{
    [JsonPropertyName("rule")]
    public string? Rule { get; set; }

    [JsonPropertyName("ext")]
    public string? Ext { get; set; }

    [JsonPropertyName("uri")]
    public string? Uri { get; set; }

    [JsonPropertyName("prefix")]
    public string? Prefix { get; set; }

    [JsonPropertyName("ttl")]
    public int? Ttl { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("no_cache")]
    public bool? NoCache { get; set; }

    [JsonPropertyName("force_cache")]
    public bool? ForceCache { get; set; }

    [JsonPropertyName("priority")]
    public int? Priority { get; set; }

    [JsonPropertyName("ignore_args")]
    public bool? IgnoreArgs { get; set; }

    [JsonPropertyName("cache_key")]
    public string? CacheKey { get; set; }
}

public sealed class EdgeCacheConfigDto
{
    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("default_ttl")]
    public int? DefaultTtl { get; set; }

    [JsonPropertyName("rules")]
    public List<EdgeCacheRuleDto>? Rules { get; set; }
}

public sealed class EdgeStreamDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("listen_ports")]
    public List<string> ListenPorts { get; set; } = new();

    [JsonPropertyName("targets")]
    public List<EdgeStreamTargetDto> Targets { get; set; } = new();

    [JsonPropertyName("use_listen_port")]
    public bool? UseListenPort { get; set; }

    [JsonPropertyName("balance_way")]
    public string? BalanceWay { get; set; }

    [JsonPropertyName("proxy_protocol")]
    public bool? ProxyProtocol { get; set; }

    [JsonPropertyName("proxy_connect_timeout")]
    public string? ProxyConnectTimeout { get; set; }

    [JsonPropertyName("proxy_timeout")]
    public string? ProxyTimeout { get; set; }

    [JsonPropertyName("conn_limit")]
    public int? ConnLimit { get; set; }
}

public sealed class EdgeStreamTargetDto
{
    [JsonPropertyName("addr")]
    public string Addr { get; set; } = string.Empty;

    [JsonPropertyName("weight")]
    public int Weight { get; set; }

    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("node_id")]
    public long? NodeId { get; set; }

    [JsonPropertyName("backup")]
    public bool? Backup { get; set; }
}

public sealed class EdgeNginxConfigDto
{
    [JsonPropertyName("logs_dir")]
    public string? LogsDir { get; set; }

    [JsonPropertyName("worker_processes")]
    public string? WorkerProcesses { get; set; }

    [JsonPropertyName("worker_connections")]
    public int? WorkerConnections { get; set; }

    [JsonPropertyName("worker_rlimit_nofile")]
    public int? WorkerRlimitNofile { get; set; }

    [JsonPropertyName("worker_shutdown_timeout")]
    public string? WorkerShutdownTimeout { get; set; }

    [JsonPropertyName("resolver")]
    public string? Resolver { get; set; }

    [JsonPropertyName("resolver_timeout")]
    public string? ResolverTimeout { get; set; }

    [JsonPropertyName("http")]
    public Dictionary<string, JsonElement>? Http { get; set; }

    [JsonPropertyName("stream")]
    public Dictionary<string, JsonElement>? Stream { get; set; }
}
