using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class WsDispatchRequest
{
    [JsonPropertyName("node_id")]
    public long? NodeId { get; set; }

    [JsonPropertyName("task_type")]
    public string? TaskType { get; set; }

    [JsonPropertyName("payload")]
    public string? Payload { get; set; }

    [JsonPropertyName("wait_seconds")]
    public int? WaitSeconds { get; set; }
}

public sealed class WsDispatchResponse
{
    [JsonPropertyName("node_id")]
    public long NodeId { get; set; }

    [JsonPropertyName("connected")]
    public bool Connected { get; set; }

    [JsonPropertyName("task_id")]
    public long TaskId { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("error")]
    public string? Error { get; set; }
}
