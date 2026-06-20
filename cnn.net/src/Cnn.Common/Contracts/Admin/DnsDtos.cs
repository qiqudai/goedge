using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class DnsProviderListQuery
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}

public sealed record DnsProviderListResult(IReadOnlyList<DnsProviderItem> List);

public sealed class DnsProviderItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("uid")]
    public long? UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("auth")]
    public string? Auth { get; set; }
}

public sealed record DnsProviderTypesResult(IReadOnlyList<DnsProviderTypeItem> Types);

public sealed record DnsProviderTypeItem(
    [property: JsonPropertyName("type")] string Type,
    [property: JsonPropertyName("name")] string Name,
    [property: JsonPropertyName("fields")] IReadOnlyList<string> Fields
);

public sealed class DnsProviderCreateRequest
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("credentials")]
    public string? Credentials { get; set; }
}

public sealed class DnsProviderUpdateRequest
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("credentials")]
    public string? Credentials { get; set; }
}

public sealed record DnsTestResult(string Status);

public sealed record DnsFixResult(string Status);

public sealed record DnsCleanupResult(string Status);
