using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class ConfigItemDto
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("value")]
    public string? Value { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("scope_name")]
    public string? ScopeName { get; set; }

    [JsonPropertyName("scope_id")]
    public int? ScopeId { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }
}

public sealed class ConfigItemPayloadDto
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("value")]
    public string? Value { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }
}

public sealed class ConfigItemUpsertRequest
{
    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("scope_name")]
    public string? ScopeName { get; set; }

    [JsonPropertyName("scope_id")]
    public int? ScopeId { get; set; }

    [JsonPropertyName("items")]
    public List<ConfigItemPayloadDto>? Items { get; set; }
}
