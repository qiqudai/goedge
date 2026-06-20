using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed record UserGroupListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<UserGroupDto> List
);

public sealed class UserGroupDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Des { get; set; }
}

public sealed class UserGroupUpsertRequest
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Des { get; set; }
}

