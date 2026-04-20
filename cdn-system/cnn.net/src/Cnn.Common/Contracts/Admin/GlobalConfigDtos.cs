using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class GlobalConfigDto
{
    [JsonPropertyName("waf")]
    public WafConfigDto? Waf { get; set; }

    [JsonPropertyName("nginx")]
    public NginxConfigDto? Nginx { get; set; }

    [JsonPropertyName("default_config")]
    public DefaultSiteConfigDto? DefaultConfig { get; set; }

    [JsonPropertyName("error_pages")]
    public Dictionary<string, string>? ErrorPages { get; set; }

    [JsonPropertyName("resources")]
    public GlobalResourceConfigDto? Resources { get; set; }
}

public sealed class WafConfigDto
{
    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("default_block_action")]
    public string? DefaultBlockAction { get; set; }

    [JsonPropertyName("auto_ipset_enable")]
    public bool AutoIpSetEnable { get; set; }

    [JsonPropertyName("auto_ipset_threshold")]
    public int AutoIpSetThreshold { get; set; }

    [JsonPropertyName("block_page_rate_limit_enable")]
    public bool BlockPageRateLimitEnable { get; set; }

    [JsonPropertyName("block_page_rate_limit")]
    public int BlockPageRateLimit { get; set; }

    [JsonPropertyName("block_page_traffic_free")]
    public bool BlockPageTrafficFree { get; set; }

    [JsonPropertyName("blacklist_timeout")]
    public int BlacklistTimeout { get; set; }

    [JsonPropertyName("temp_whitelist_timeout")]
    public int TempWhitelistTimeout { get; set; }

    [JsonPropertyName("temp_whitelist_limit_total")]
    public int TempWhitelistLimitTotal { get; set; }

    [JsonPropertyName("temp_whitelist_limit_url")]
    public int TempWhitelistLimitUrl { get; set; }

    [JsonPropertyName("whitelist_ips")]
    public string? WhitelistIps { get; set; }

    [JsonPropertyName("blacklist_ips")]
    public string? BlacklistIps { get; set; }

    [JsonPropertyName("prevent_tls_handshake")]
    public bool PreventTlsHandshake { get; set; }

    [JsonPropertyName("block_unbound_domain")]
    public bool BlockUnboundDomain { get; set; }

    [JsonPropertyName("disable_ping")]
    public bool DisablePing { get; set; }

    [JsonPropertyName("default_page_protection")]
    public string? DefaultPageProtection { get; set; }

    [JsonPropertyName("default_page_protection_threshold")]
    public int DefaultPageProtectionThreshold { get; set; }

    [JsonPropertyName("secret_key")]
    public string? SecretKey { get; set; }

    [JsonPropertyName("node_log_clean_strategy")]
    public string? NodeLogCleanStrategy { get; set; }

    [JsonPropertyName("cc_rule_auto_switch")]
    public bool CcRuleAutoSwitch { get; set; }

    [JsonPropertyName("anti_cc_image_source")]
    public string? AntiCcImageSource { get; set; }

    [JsonPropertyName("anti_cc_image_custom_url")]
    public string? AntiCcImageCustomUrl { get; set; }

    [JsonPropertyName("anti_cc_type")]
    public string? AntiCcType { get; set; }

    [JsonPropertyName("anti_cc_debug")]
    public bool AntiCcDebug { get; set; }

    [JsonPropertyName("well_known_protection_threshold")]
    public int WellKnownProtectionThreshold { get; set; }

    [JsonPropertyName("resource_protection_enable")]
    public bool ResourceProtectionEnable { get; set; }

    [JsonPropertyName("resource_protection_threshold")]
    public int ResourceProtectionThreshold { get; set; }

    [JsonPropertyName("resource_protection_block_timeout")]
    public int ResourceProtectionBlockTimeout { get; set; }

    [JsonPropertyName("resource_protection_rules")]
    public IReadOnlyList<ResourceRuleDto>? ResourceProtectionRules { get; set; }

    [JsonPropertyName("mode")]
    public string? Mode { get; set; }

    [JsonPropertyName("policy")]
    public string? Policy { get; set; }

    [JsonPropertyName("cc")]
    public CcConfigDto? Cc { get; set; }

    [JsonPropertyName("access_control")]
    public AccessControlDto? AccessControl { get; set; }

    [JsonPropertyName("syntactic")]
    public SyntacticWafDto? Syntactic { get; set; }
}

public sealed record ResourceRuleDto(
    [property: JsonPropertyName("duration")] int Duration,
    [property: JsonPropertyName("max_requests")] int MaxRequests
);

public sealed class CcConfigDto
{
    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("threshold")]
    public int Threshold { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("block_timeout")]
    public int BlockTimeout { get; set; }

    [JsonPropertyName("emergency_mode")]
    public bool EmergencyMode { get; set; }

    [JsonPropertyName("slide_count")]
    public int SlideCount { get; set; }
}

public sealed class AccessControlDto
{
    [JsonPropertyName("black_ip")]
    public IReadOnlyList<string>? BlackIp { get; set; }

    [JsonPropertyName("white_ip")]
    public IReadOnlyList<string>? WhiteIp { get; set; }

    [JsonPropertyName("black_ua")]
    public IReadOnlyList<string>? BlackUa { get; set; }

    [JsonPropertyName("white_ua")]
    public IReadOnlyList<string>? WhiteUa { get; set; }

    [JsonPropertyName("black_url")]
    public IReadOnlyList<string>? BlackUrl { get; set; }

    [JsonPropertyName("white_url")]
    public IReadOnlyList<string>? WhiteUrl { get; set; }

    [JsonPropertyName("region_block")]
    public IReadOnlyList<string>? RegionBlock { get; set; }

    [JsonPropertyName("block_empty_ua")]
    public bool BlockEmptyUa { get; set; }
}

public sealed class SyntacticWafDto
{
    [JsonPropertyName("sql_injection")]
    public bool SqlInjection { get; set; }

    [JsonPropertyName("xss")]
    public bool Xss { get; set; }

    [JsonPropertyName("scanner")]
    public bool Scanner { get; set; }
}

public sealed class NginxConfigDto
{
    [JsonPropertyName("worker_processes")]
    public string? WorkerProcesses { get; set; }

    [JsonPropertyName("worker_connections")]
    public int WorkerConnections { get; set; }

    [JsonPropertyName("worker_rlimit_nofile")]
    public int WorkerRlimitNofile { get; set; }

    [JsonPropertyName("worker_shutdown_timeout")]
    public string? WorkerShutdownTimeout { get; set; }

    [JsonPropertyName("log_directory")]
    public string? LogDirectory { get; set; }

    [JsonPropertyName("keepalive_timeout")]
    public int KeepaliveTimeout { get; set; }

    [JsonPropertyName("gzip")]
    public bool Gzip { get; set; }

    [JsonPropertyName("custom_snippet")]
    public string? CustomSnippet { get; set; }
}

public sealed class DefaultSiteConfigDto
{
    [JsonPropertyName("website")]
    public SiteTemplateDto? Website { get; set; }

    [JsonPropertyName("api")]
    public SiteTemplateDto? Api { get; set; }

    [JsonPropertyName("download")]
    public SiteTemplateDto? Download { get; set; }
}

public sealed class SiteTemplateDto
{
    [JsonPropertyName("cache_enable")]
    public bool CacheEnable { get; set; }

    [JsonPropertyName("cache_ttl")]
    public int CacheTtl { get; set; }

    [JsonPropertyName("gzip")]
    public bool Gzip { get; set; }

    [JsonPropertyName("waf_enable")]
    public bool WafEnable { get; set; }

    [JsonPropertyName("ssl_ciphers")]
    public string? SslCiphers { get; set; }
}

public sealed class GlobalResourceConfigDto
{
    [JsonPropertyName("website")]
    public WebsiteResourceConfigDto? Website { get; set; }

    [JsonPropertyName("forward")]
    public ForwardResourceConfigDto? Forward { get; set; }

    [JsonPropertyName("public")]
    public PublicResourceConfigDto? Public { get; set; }
}

public sealed class WebsiteResourceConfigDto
{
    [JsonPropertyName("min_limit")]
    public int MinLimit { get; set; }

    [JsonPropertyName("max_limit_multiplier")]
    public int MaxLimitMultiplier { get; set; }

    [JsonPropertyName("max_blacklist_ips")]
    public int MaxBlacklistIps { get; set; }

    [JsonPropertyName("max_whitelist_ips")]
    public int MaxWhitelistIps { get; set; }

    [JsonPropertyName("daily_url_purge_limit")]
    public int DailyUrlPurgeLimit { get; set; }

    [JsonPropertyName("daily_dir_purge_limit")]
    public int DailyDirPurgeLimit { get; set; }

    [JsonPropertyName("daily_preload_limit")]
    public int DailyPreloadLimit { get; set; }

    [JsonPropertyName("daily_unlock_ip_limit")]
    public int DailyUnlockIpLimit { get; set; }

    [JsonPropertyName("unlock_ip_batch_limit")]
    public int UnlockIpBatchLimit { get; set; }

    [JsonPropertyName("max_cc_rules_per_group")]
    public int MaxCcRulesPerGroup { get; set; }

    [JsonPropertyName("max_acl_rules")]
    public int MaxAclRules { get; set; }

    [JsonPropertyName("daily_log_download_limit")]
    public int DailyLogDownloadLimit { get; set; }

    [JsonPropertyName("log_storage_dir")]
    public string? LogStorageDir { get; set; }

    [JsonPropertyName("log_storage_hours")]
    public int LogStorageHours { get; set; }

    [JsonPropertyName("max_domains_per_site")]
    public int MaxDomainsPerSite { get; set; }

    [JsonPropertyName("default_listen_80")]
    public bool DefaultListen80 { get; set; }
}

public sealed class ForwardResourceConfigDto
{
    [JsonPropertyName("disabled_ports")]
    public string? DisabledPorts { get; set; }

    [JsonPropertyName("min_limit")]
    public int MinLimit { get; set; }

    [JsonPropertyName("max_limit_multiplier")]
    public int MaxLimitMultiplier { get; set; }

    [JsonPropertyName("max_acl_rules")]
    public int MaxAclRules { get; set; }
}

public sealed class PublicResourceConfigDto
{
    [JsonPropertyName("disabled_custom_ports")]
    public string? DisabledCustomPorts { get; set; }

    [JsonPropertyName("allowed_custom_ports")]
    public string? AllowedCustomPorts { get; set; }
}
