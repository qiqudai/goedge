using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Agent;

public sealed class AgentUpgradeInfo
{
    [JsonPropertyName("api_version")]
    public string ApiVersion { get; set; } = string.Empty;

    [JsonPropertyName("node_version")]
    public string? NodeVersion { get; set; }

    [JsonPropertyName("auto_upgrade")]
    public bool AutoUpgrade { get; set; }

    [JsonPropertyName("need_upgrade")]
    public bool NeedUpgrade { get; set; }

    [JsonPropertyName("should_upgrade")]
    public bool ShouldUpgrade { get; set; }
}
