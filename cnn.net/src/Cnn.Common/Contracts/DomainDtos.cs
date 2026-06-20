using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class DomainListQuery
{
    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("pageSize")]
    public int PageSize { get; set; } = 20;
}

public sealed record DomainListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<DomainDto> List,
    [property: JsonPropertyName("total")] int Total
);

public sealed class DomainDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("cname")]
    public string? Cname { get; set; }

    [JsonPropertyName("status")]
    public int Status { get; set; }

    [JsonPropertyName("origins")]
    public IReadOnlyList<DomainOriginDto> Origins { get; set; } = Array.Empty<DomainOriginDto>();

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("updated_at")]
    public string? UpdatedAt { get; set; }
}

public sealed class DomainOriginDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("domain_id")]
    public long DomainId { get; set; }

    [JsonPropertyName("addr")]
    public string? Addr { get; set; }

    [JsonPropertyName("port")]
    public int Port { get; set; }

    [JsonPropertyName("weight")]
    public int Weight { get; set; }

    [JsonPropertyName("protocol")]
    public string? Protocol { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("updated_at")]
    public string? UpdatedAt { get; set; }
}

public sealed class CreateDomainRequest
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("origins")]
    public List<DomainOriginDto>? Origins { get; set; }
}

public sealed class DomainConfigDto
{
    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("origins")]
    public IReadOnlyList<DomainOriginDto> Origins { get; set; } = Array.Empty<DomainOriginDto>();

    [JsonPropertyName("https_on")]
    public bool HttpsOn { get; set; }

    [JsonPropertyName("cache_rules")]
    public IReadOnlyList<DomainCacheRuleDto> CacheRules { get; set; } = Array.Empty<DomainCacheRuleDto>();
}

public sealed class DomainCacheRuleDto
{
    [JsonPropertyName("ext")]
    public string? Ext { get; set; }

    [JsonPropertyName("ttl")]
    public int Ttl { get; set; }
}
