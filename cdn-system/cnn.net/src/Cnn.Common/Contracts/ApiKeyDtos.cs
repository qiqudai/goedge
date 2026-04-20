using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class ApiKeyDto
{
    [JsonPropertyName("api_key")]
    public string? ApiKey { get; set; }

    [JsonPropertyName("api_secret")]
    public string? ApiSecret { get; set; }

    [JsonPropertyName("api_ip")]
    public string? ApiIp { get; set; }
}

public sealed class ApiKeyUpdateRequest
{
    [JsonPropertyName("api_ip")]
    public string? ApiIp { get; set; }
}

public sealed class ApiKeySecretDto
{
    [JsonPropertyName("api_secret")]
    public string? ApiSecret { get; set; }
}
