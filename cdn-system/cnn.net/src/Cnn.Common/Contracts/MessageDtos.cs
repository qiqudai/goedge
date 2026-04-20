using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class MessageListQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }
}

public sealed record MessageListResult(IReadOnlyList<MessageItemDto> List, long Total);

public sealed class MessageItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("type_label")]
    public string? TypeLabel { get; set; }

    [JsonPropertyName("title")]
    public string? Title { get; set; }

    [JsonPropertyName("content")]
    public string? Content { get; set; }

    [JsonPropertyName("phone")]
    public string? Phone { get; set; }

    [JsonPropertyName("site_id")]
    public long? SiteId { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("is_read")]
    public bool IsRead { get; set; }
}

public sealed record MessageUnreadResult(long Count, MessageItemDto? Latest);

public sealed record MessageSubListResult(IReadOnlyList<MessageSubItemDto> List);

public sealed class MessageSubItemDto
{
    [JsonPropertyName("msg_type")]
    public string? MsgType { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("phone")]
    public bool? Phone { get; set; }

    [JsonPropertyName("email")]
    public bool? Email { get; set; }
}

public sealed class MessageSubUpdateRequest
{
    [JsonPropertyName("list")]
    public List<MessageSubUpdateItem> List { get; set; } = new();
}

public sealed class MessageSubUpdateItem
{
    [JsonPropertyName("msg_type")]
    public string? MsgType { get; set; }

    [JsonPropertyName("phone")]
    public bool? Phone { get; set; }

    [JsonPropertyName("email")]
    public bool? Email { get; set; }
}
