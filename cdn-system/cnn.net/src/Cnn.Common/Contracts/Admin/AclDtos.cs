using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class AclListQuery
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}

public sealed record AclListResult(IReadOnlyList<AclListItem> List, long Total);

public sealed class AclListItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("user")]
    public AclUserInfo? User { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Description { get; set; }

    [JsonPropertyName("default_action")]
    public string? DefaultAction { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("create_time")]
    public string? CreateTime { get; set; }
}

public sealed record AclUserInfo(
    [property: JsonPropertyName("username")] string? Username,
    [property: JsonPropertyName("id")] long Id
);

public sealed class AclDetailDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Description { get; set; }

    [JsonPropertyName("default_action")]
    public string? DefaultAction { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("rules")]
    public IReadOnlyList<AclRuleDto>? Rules { get; set; }

    [JsonPropertyName("default_deny_status")]
    public int DefaultDenyStatus { get; set; }

    [JsonPropertyName("default_redirect_url")]
    public string? DefaultRedirectUrl { get; set; }
}

public sealed class AclUpsertRequest
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Description { get; set; }

    [JsonPropertyName("default_action")]
    public string? DefaultAction { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("rules")]
    public IReadOnlyList<AclRuleDto>? Rules { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("default_deny_status")]
    public int DefaultDenyStatus { get; set; }

    [JsonPropertyName("default_redirect_url")]
    public string? DefaultRedirectUrl { get; set; }
}

public sealed class AclConditionDto
{
    [JsonPropertyName("item")]
    public string? Item { get; set; }

    [JsonPropertyName("operator")]
    public string? Operator { get; set; }

    [JsonPropertyName("value")]
    public string? Value { get; set; }
}

public sealed class AclRuleDto
{
    [JsonPropertyName("conditions")]
    public IReadOnlyList<AclConditionDto>? Conditions { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("deny_status")]
    public int DenyStatus { get; set; }

    [JsonPropertyName("redirect_url")]
    public string? RedirectUrl { get; set; }
}
