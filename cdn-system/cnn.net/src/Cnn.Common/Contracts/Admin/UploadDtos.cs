using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class UploadImageResult
{
    [JsonPropertyName("url")]
    public string? Url { get; set; }
}
