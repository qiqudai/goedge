using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class AnnouncementListQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }
}

public sealed record AnnouncementListResult(IReadOnlyList<AnnouncementItemDto> List, long Total);

public sealed class AnnouncementItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("title")]
    public string? Title { get; set; }

    [JsonPropertyName("content")]
    public string? Content { get; set; }

    [JsonPropertyName("is_show")]
    public bool IsShow { get; set; }

    [JsonPropertyName("is_red")]
    public bool IsRed { get; set; }

    [JsonPropertyName("is_bold")]
    public bool IsBold { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }
}

public sealed class AnnouncementUpsertRequest
{
    [JsonPropertyName("title")]
    public string? Title { get; set; }

    [JsonPropertyName("content")]
    public string? Content { get; set; }

    [JsonPropertyName("is_show")]
    public bool? IsShow { get; set; }

    [JsonPropertyName("is_red")]
    public bool? IsRed { get; set; }

    [JsonPropertyName("is_bold")]
    public bool? IsBold { get; set; }
}
