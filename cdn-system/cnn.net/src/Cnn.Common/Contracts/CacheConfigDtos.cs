using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class CacheConfigDto
{
    [JsonPropertyName("profiles")]
    public Dictionary<string, CacheProfileDto>? Profiles { get; set; }

    [JsonPropertyName("rules")]
    public List<CacheRuleDto>? Rules { get; set; }
}

public sealed class CacheProfileDto
{
    [JsonPropertyName("ttl")]
    public int Ttl { get; set; }

    [JsonPropertyName("ignore_query")]
    public bool IgnoreQuery { get; set; }

    [JsonPropertyName("force_cache")]
    public bool ForceCache { get; set; }

    [JsonPropertyName("query_ignore_list")]
    public List<string>? QueryIgnoreList { get; set; }
}

public sealed class CacheRuleDto
{
    [JsonPropertyName("host")]
    public string? Host { get; set; }

    [JsonPropertyName("path_prefix")]
    public string? PathPrefix { get; set; }

    [JsonPropertyName("path_regex")]
    public string? PathRegex { get; set; }

    [JsonPropertyName("profile")]
    public string? Profile { get; set; }
}

public sealed class CacheSiteConfigDto
{
    [JsonPropertyName("site_id")]
    public int SiteId { get; set; }

    [JsonPropertyName("version")]
    public int Version { get; set; }

    [JsonPropertyName("hosts")]
    public List<string>? Hosts { get; set; }

    [JsonPropertyName("profiles")]
    public Dictionary<string, CacheProfileDto>? Profiles { get; set; }

    [JsonPropertyName("rules")]
    public List<CacheRuleDto>? Rules { get; set; }
}
