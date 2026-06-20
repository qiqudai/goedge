using System.Text.Json;
using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class DebugSwitchDispatchRequest
{
    [JsonPropertyName("node_id")]
    public long? NodeId { get; set; }

    [JsonPropertyName("switches")]
    public Dictionary<string, bool>? Switches { get; set; }

    [JsonPropertyName("ttl_seconds")]
    public int? TtlSeconds { get; set; }

    [JsonPropertyName("sample_rate")]
    public double? SampleRate { get; set; }

    [JsonPropertyName("max_events_per_sec")]
    public int? MaxEventsPerSec { get; set; }

    [JsonPropertyName("debug_enabled")]
    public bool? DebugEnabled { get; set; }

    [JsonPropertyName("internal_ip_only")]
    public bool? InternalIpOnly { get; set; }

    [JsonPropertyName("debug_token")]
    public string? DebugToken { get; set; }

    [JsonPropertyName("allow_header_token")]
    public bool? AllowHeaderToken { get; set; }

    [JsonPropertyName("allow_query_flag")]
    public bool? AllowQueryFlag { get; set; }

    [JsonPropertyName("modules")]
    public Dictionary<string, bool>? Modules { get; set; }

    [JsonPropertyName("reason")]
    public string? Reason { get; set; }

    [JsonPropertyName("wait_seconds")]
    public int? WaitSeconds { get; set; }
}

public sealed class ManualDebugLogDispatchRequest
{
    [JsonPropertyName("node_id")]
    public long? NodeId { get; set; }

    [JsonPropertyName("category")]
    public string? Category { get; set; }

    [JsonPropertyName("message")]
    public string? Message { get; set; }

    [JsonPropertyName("data")]
    public JsonElement? Data { get; set; }

    [JsonPropertyName("wait_seconds")]
    public int? WaitSeconds { get; set; }
}

public sealed class ServerDebugSwitchesDto
{
    [JsonPropertyName("operation_log_enabled")]
    public bool OperationLogEnabled { get; set; }

    [JsonPropertyName("agent_api_trace_enabled")]
    public bool AgentApiTraceEnabled { get; set; }

    [JsonPropertyName("agent_api_trace_payload_enabled")]
    public bool AgentApiTracePayloadEnabled { get; set; }

    [JsonPropertyName("agent_api_trace_max_payload")]
    public int AgentApiTraceMaxPayload { get; set; }

    [JsonPropertyName("agent_api_trace_max_events_per_sec")]
    public int AgentApiTraceMaxEventsPerSec { get; set; }
}

public sealed class ServerDebugSwitchesUpdateRequest
{
    [JsonPropertyName("operation_log_enabled")]
    public bool? OperationLogEnabled { get; set; }

    [JsonPropertyName("agent_api_trace_enabled")]
    public bool? AgentApiTraceEnabled { get; set; }

    [JsonPropertyName("agent_api_trace_payload_enabled")]
    public bool? AgentApiTracePayloadEnabled { get; set; }

    [JsonPropertyName("agent_api_trace_max_payload")]
    public int? AgentApiTraceMaxPayload { get; set; }

    [JsonPropertyName("agent_api_trace_max_events_per_sec")]
    public int? AgentApiTraceMaxEventsPerSec { get; set; }
}
