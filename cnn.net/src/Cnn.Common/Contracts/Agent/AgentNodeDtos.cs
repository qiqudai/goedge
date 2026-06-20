using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Agent;

public sealed class AgentHeartbeatRequest
{
    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("timestamp")]
    public long Timestamp { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }
}

public sealed class AgentHeartbeatResponse
{
    [JsonPropertyName("status")]
    public string Status { get; set; } = "pong";

    [JsonPropertyName("sync_action")]
    public string? SyncAction { get; set; }
}

public sealed class AgentSyncRequest
{
    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("success")]
    public bool Success { get; set; }
}

public sealed class AgentSyncResponse
{
    [JsonPropertyName("status")]
    public string Status { get; set; } = "ok";
}

public sealed class AgentL2HeartbeatRequest
{
    [JsonPropertyName("nodes")]
    public List<long> Nodes { get; set; } = new();
}

public sealed class AgentL2NodesResult
{
    [JsonPropertyName("nodes")]
    public List<AgentL2NodeItem> Nodes { get; set; } = new();
}

public sealed class AgentL2NodeItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("ip")]
    public string? Ip { get; set; }

    [JsonPropertyName("port")]
    public int? Port { get; set; }

    [JsonPropertyName("check_protocol")]
    public string? CheckProtocol { get; set; }

    [JsonPropertyName("check_port")]
    public int? CheckPort { get; set; }

    [JsonPropertyName("check_host")]
    public string? CheckHost { get; set; }

    [JsonPropertyName("check_path")]
    public string? CheckPath { get; set; }

    [JsonPropertyName("check_timeout")]
    public int? CheckTimeout { get; set; }
}
