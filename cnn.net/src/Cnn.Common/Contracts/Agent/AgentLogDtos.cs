using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Agent;

public sealed class AgentAccessLogRequest
{
    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("node_ip")]
    public string? NodeIp { get; set; }

    [JsonPropertyName("lines")]
    public List<string> Lines { get; set; } = new();
}

public sealed class AgentMetricLogRequest
{
    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("node_ip")]
    public string? NodeIp { get; set; }

    [JsonPropertyName("content")]
    public string? Content { get; set; }
}

public sealed class AgentEventLogRequest
{
    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("node_ip")]
    public string? NodeIp { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("payloads")]
    public List<string> Payloads { get; set; } = new();
}
