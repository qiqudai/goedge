using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed record SiteDefaultListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<SiteDefaultItemDto> List
);

public sealed class SiteDefaultItemDto
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("value")]
    public string? Value { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("scope_id")]
    public long ScopeId { get; set; }

    [JsonPropertyName("scope_name")]
    public string? ScopeName { get; set; }

    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_name")]
    public string? UserName { get; set; }

    [JsonPropertyName("group_name")]
    public string? GroupName { get; set; }
}

public sealed class SiteDefaultListQuery
{
    [JsonPropertyName("scope_name")]
    public string? ScopeName { get; set; }

    [JsonPropertyName("scope_id")]
    public long? ScopeId { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}

public sealed class SiteDefaultCreateRequest
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("value")]
    public string? Value { get; set; }

    [JsonPropertyName("scope_name")]
    public string? ScopeName { get; set; }

    [JsonPropertyName("scope_id")]
    public long? ScopeId { get; set; }

    [JsonPropertyName("data")]
    public Dictionary<string, object?>? Data { get; set; }
}

public sealed class SiteDefaultUpdateRequest
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("value")]
    public string? Value { get; set; }

    [JsonPropertyName("scope_name")]
    public string? ScopeName { get; set; }

    [JsonPropertyName("scope_id")]
    public long? ScopeId { get; set; }

    [JsonPropertyName("old_scope_name")]
    public string? OldScopeName { get; set; }

    [JsonPropertyName("old_scope_id")]
    public long? OldScopeId { get; set; }
}
