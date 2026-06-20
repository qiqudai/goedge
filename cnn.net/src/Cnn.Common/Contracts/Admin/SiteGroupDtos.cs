using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed record SiteGroupListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<SiteGroupDto> List,
    [property: JsonPropertyName("total")] int Total
);

public sealed class SiteGroupDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed class SiteGroupListQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}

public sealed class SiteGroupUpsertRequest
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}
