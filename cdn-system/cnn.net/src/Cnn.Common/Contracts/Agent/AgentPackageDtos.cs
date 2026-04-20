using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Agent;

public sealed class AgentPackagePayloadDto
{
    [JsonPropertyName("packages")]
    public List<AgentPackageItemDto> Packages { get; set; } = new();
}

public sealed class AgentPackageItemDto
{
    [JsonPropertyName("package_id")]
    public long PackageId { get; set; }

    [JsonPropertyName("version")]
    public int Version { get; set; }

    [JsonPropertyName("config")]
    public AgentPackageConfigDto? Config { get; set; }
}

public sealed class AgentPackageConfigDto
{
    [JsonPropertyName("package_id")]
    public int PackageId { get; set; }

    [JsonPropertyName("uid")]
    public int Uid { get; set; }

    [JsonPropertyName("version")]
    public int Version { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("region_id")]
    public int RegionId { get; set; }

    [JsonPropertyName("node_group_id")]
    public int NodeGroupId { get; set; }

    [JsonPropertyName("backup_node_group")]
    public int BackupNodeGroup { get; set; }

    [JsonPropertyName("enable_backup")]
    public int EnableBackup { get; set; }

    [JsonPropertyName("cname")]
    public AgentPackageCnameDto? Cname { get; set; }

    [JsonPropertyName("limits")]
    public AgentPackageLimitsDto? Limits { get; set; }

    [JsonPropertyName("features")]
    public AgentPackageFeaturesDto? Features { get; set; }

    [JsonPropertyName("time")]
    public AgentPackageTimeDto? Time { get; set; }
}

public sealed class AgentPackageCnameDto
{
    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("hostname")]
    public string? Hostname { get; set; }

    [JsonPropertyName("hostname2")]
    public string? Hostname2 { get; set; }

    [JsonPropertyName("mode")]
    public string? Mode { get; set; }

    [JsonPropertyName("record_id")]
    public string? RecordId { get; set; }
}

public sealed class AgentPackageLimitsDto
{
    [JsonPropertyName("traffic")]
    public int Traffic { get; set; }

    [JsonPropertyName("bandwidth")]
    public string? Bandwidth { get; set; }

    [JsonPropertyName("connection")]
    public int Connection { get; set; }

    [JsonPropertyName("domain")]
    public int Domain { get; set; }
}

public sealed class AgentPackageFeaturesDto
{
    [JsonPropertyName("http_port")]
    public int HttpPort { get; set; }

    [JsonPropertyName("stream_port")]
    public int StreamPort { get; set; }

    [JsonPropertyName("websocket")]
    public bool Websocket { get; set; }

    [JsonPropertyName("custom_cc_rule")]
    public bool CustomCcRule { get; set; }

    [JsonPropertyName("l2_origin")]
    public bool L2Origin { get; set; }
}

public sealed class AgentPackageTimeDto
{
    [JsonPropertyName("start_at")]
    public string? StartAt { get; set; }

    [JsonPropertyName("end_at")]
    public string? EndAt { get; set; }
}
