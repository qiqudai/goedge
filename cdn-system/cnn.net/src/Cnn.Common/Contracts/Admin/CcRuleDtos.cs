using System.Text.Json;
using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class CcListQuery
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }
}

public sealed record CcListResult<T>(IReadOnlyList<T> List, long Total);

public sealed record CcUserInfo(
    [property: JsonPropertyName("username")] string? Username,
    [property: JsonPropertyName("id")] long Id
);

public sealed class CcRuleGroupListItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("user")]
    public CcUserInfo? User { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("is_system")]
    public bool IsSystem { get; set; }

    [JsonPropertyName("is_on")]
    public bool? IsOn { get; set; }

    [JsonPropertyName("is_show")]
    public bool? IsShow { get; set; }

    [JsonPropertyName("is_visible")]
    public bool? IsVisible { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("sort_order")]
    public int? SortOrder { get; set; }

    [JsonPropertyName("create_time")]
    public string? CreateTime { get; set; }

    [JsonPropertyName("created_at")]
    public long? CreatedAt { get; set; }
}

public sealed class CcRuleGroupDetailDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("internal")]
    public bool? Internal { get; set; }

    [JsonPropertyName("is_system")]
    public bool IsSystem { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("is_on")]
    public bool? IsOn { get; set; }

    [JsonPropertyName("is_visible")]
    public bool? IsVisible { get; set; }

    [JsonPropertyName("is_show")]
    public bool? IsShow { get; set; }

    [JsonPropertyName("sort_order")]
    public int? SortOrder { get; set; }

    [JsonPropertyName("rules")]
    public IReadOnlyList<JsonElement>? Rules { get; set; }

    [JsonPropertyName("visible_users")]
    public IReadOnlyList<long>? VisibleUsers { get; set; }
}

public sealed class CcRuleGroupUpsertRequest
{
    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("rules")]
    public IReadOnlyList<JsonElement>? Rules { get; set; }

    [JsonPropertyName("is_visible")]
    public bool IsVisible { get; set; }

    [JsonPropertyName("visible_users")]
    public IReadOnlyList<long>? VisibleUsers { get; set; }

    [JsonPropertyName("sort_order")]
    public int SortOrder { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}

public sealed class CcMatcherListItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("user")]
    public CcUserInfo? User { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("is_system")]
    public bool IsSystem { get; set; }

    [JsonPropertyName("is_on")]
    public bool? IsOn { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("create_time")]
    public string? CreateTime { get; set; }

    [JsonPropertyName("created_at")]
    public long? CreatedAt { get; set; }
}

public sealed class CcMatcherDetailDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("internal")]
    public bool? Internal { get; set; }

    [JsonPropertyName("is_system")]
    public bool IsSystem { get; set; }

    [JsonPropertyName("is_on")]
    public bool? IsOn { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("rules")]
    public IReadOnlyList<JsonElement>? Rules { get; set; }
}

public sealed class CcMatcherUpsertRequest
{
    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("is_on")]
    public bool IsOn { get; set; }

    [JsonPropertyName("rules")]
    public IReadOnlyList<JsonElement>? Rules { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}

public sealed class CcFilterListItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("user")]
    public CcUserInfo? User { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("is_system")]
    public bool IsSystem { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("is_on")]
    public bool? IsOn { get; set; }

    [JsonPropertyName("create_time")]
    public string? CreateTime { get; set; }

    [JsonPropertyName("created_at")]
    public long? CreatedAt { get; set; }
}

public sealed class CcFilterDetailDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("internal")]
    public bool? Internal { get; set; }

    [JsonPropertyName("is_system")]
    public bool IsSystem { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("match_mode")]
    public string? MatchMode { get; set; }

    [JsonPropertyName("blacklist")]
    public bool? Blacklist { get; set; }

    [JsonPropertyName("within_second")]
    public int? WithinSecond { get; set; }

    [JsonPropertyName("max_req")]
    public int? MaxReq { get; set; }

    [JsonPropertyName("max_req_per_uri")]
    public int? MaxReqPerUri { get; set; }

    [JsonPropertyName("auth")]
    public JsonElement? Auth { get; set; }
}

public sealed class CcFilterUpsertRequest
{
    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("match_mode")]
    public string? MatchMode { get; set; }

    [JsonPropertyName("blacklist")]
    public bool Blacklist { get; set; }

    [JsonPropertyName("within_second")]
    public int WithinSecond { get; set; }

    [JsonPropertyName("max_req")]
    public int MaxReq { get; set; }

    [JsonPropertyName("max_req_per_uri")]
    public int MaxReqPerUri { get; set; }

    [JsonPropertyName("auth")]
    public JsonElement? Auth { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}
