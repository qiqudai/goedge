using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class AgentPackageItemDto
{
    [JsonPropertyName("version")]
    public string? Version { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("gray_percent")]
    public int GrayPercent { get; set; }

    [JsonPropertyName("upload_time")]
    public string? UploadTime { get; set; }

    [JsonPropertyName("filename")]
    public string? Filename { get; set; }

    [JsonPropertyName("size")]
    public long Size { get; set; }

    [JsonPropertyName("sha256")]
    public string? Sha256 { get; set; }
}

public sealed record AgentPackageListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<AgentPackageItemDto> List
);

public sealed class AgentPackageGrayRequest
{
    [JsonPropertyName("version")]
    public string? Version { get; set; }

    [JsonPropertyName("percent")]
    public int Percent { get; set; }
}

public sealed class AgentPackageStableRequest
{
    [JsonPropertyName("version")]
    public string? Version { get; set; }
}

public sealed class AgentPackageNodeDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("ip")]
    public string? Ip { get; set; }

    [JsonPropertyName("region_id")]
    public int? RegionId { get; set; }

    [JsonPropertyName("region_name")]
    public string? RegionName { get; set; }

    [JsonPropertyName("group_name")]
    public string? GroupName { get; set; }

    [JsonPropertyName("current_version")]
    public string? CurrentVersion { get; set; }

    [JsonPropertyName("latest_version")]
    public string? LatestVersion { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("online")]
    public bool Online { get; set; }
}

public sealed record AgentPackageNodeListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<AgentPackageNodeDto> List
);

public sealed class AgentPackageUpgradeRequest
{
    [JsonPropertyName("version")]
    public string? Version { get; set; }

    [JsonPropertyName("node_ids")]
    public List<long>? NodeIds { get; set; }

    [JsonPropertyName("group_ids")]
    public List<long>? GroupIds { get; set; }

    [JsonPropertyName("region_ids")]
    public List<long>? RegionIds { get; set; }
}

public sealed class AgentPackageUpgradeResult
{
    [JsonPropertyName("task_id")]
    public long TaskId { get; set; }
}

public sealed class AgentPackageUpgradeNodeDto
{
    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("progress")]
    public int Progress { get; set; }

    [JsonPropertyName("message")]
    public string? Message { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }

    [JsonPropertyName("last_at")]
    public long? LastAt { get; set; }
}

public sealed class AgentPackageUpgradeStatusResult
{
    [JsonPropertyName("task_id")]
    public long TaskId { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("nodes")]
    public IReadOnlyList<AgentPackageUpgradeNodeDto> Nodes { get; set; } = Array.Empty<AgentPackageUpgradeNodeDto>();
}

public sealed class AgentPackageDownloadResult
{
    public string? FilePath { get; set; }

    public string? FileName { get; set; }
}
