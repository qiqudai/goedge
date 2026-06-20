using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class NodeGroupListQuery
{
    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("region_id")]
    public long? RegionId { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("limit")]
    public int Limit { get; set; } = 20;
}

public sealed record NodeGroupListResult(IReadOnlyList<NodeGroupListItem> List, long Total);

public sealed class NodeGroupListItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("region_id")]
    public long? RegionId { get; set; }

    [JsonPropertyName("resolution")]
    public string? CnameHostname { get; set; }

    [JsonPropertyName("cname_domain")]
    public string? CnameDomain { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("spare_ip_switch")]
    public string? SpareIpSwitch { get; set; }

    [JsonPropertyName("backup_switch_policy")]
    public string? BackupSwitchPolicy { get; set; }

    [JsonPropertyName("ipv4_resolution")]
    public string? Ipv4Resolution { get; set; }

    [JsonPropertyName("l2_config")]
    public string? L2Config { get; set; }

    [JsonPropertyName("sort_order")]
    public int? SortOrder { get; set; }

    [JsonPropertyName("node_count")]
    public long NodeCount { get; set; }

    [JsonPropertyName("site_count")]
    public long SiteCount { get; set; }

    [JsonPropertyName("forward_count")]
    public long ForwardCount { get; set; }
}

public sealed class NodeGroupUpsertRequest
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("region_id")]
    public long? RegionId { get; set; }

    [JsonPropertyName("resolution")]
    public string? CnameHostname { get; set; }

    [JsonPropertyName("cname_domain")]
    public string? CnameDomain { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("spare_ip_switch")]
    public string? SpareIpSwitch { get; set; }

    [JsonPropertyName("backup_switch_policy")]
    public string? BackupSwitchPolicy { get; set; }

    [JsonPropertyName("ipv4_resolution")]
    public string? Ipv4Resolution { get; set; }

    [JsonPropertyName("l2_config")]
    public string? L2Config { get; set; }

    [JsonPropertyName("sort_order")]
    public int? SortOrder { get; set; }
}

public sealed class NodeGroupResolutionQuery
{
    [JsonPropertyName("line_id")]
    public string? LineId { get; set; }
}

public sealed record NodeGroupResolutionResult(
    NodeGroupResolutionMeta Group,
    NodeGroupResolutionLine Line,
    IReadOnlyList<NodeGroupResolutionItem> Available,
    IReadOnlyList<NodeGroupResolutionAssigned> Assigned
);

public sealed record NodeGroupResolutionMeta(long Id, string? Name, string? RegionName);

public sealed record NodeGroupResolutionLine(string? Id, string? Name);

public sealed record NodeGroupResolutionItem(long NodeId, long NodeIpId, string? Name, string? Ip, bool Online);

public sealed record NodeGroupResolutionAssigned(
    long Id,
    long NodeId,
    long NodeIpId,
    string? LineId,
    string? LineName,
    string? Name,
    string? Ip,
    bool Online,
    bool IsOn,
    bool NodeIsOn,
    bool IsBackup,
    bool IsBackupDefaultLine,
    string? Weight,
    int? SortOrder
);

public sealed class NodeGroupAssignRequest
{
    [JsonPropertyName("line_id")]
    public string? LineId { get; set; }

    [JsonPropertyName("line_name")]
    public string? LineName { get; set; }

    [JsonPropertyName("items")]
    public IReadOnlyList<NodeGroupAssignItem>? Items { get; set; }
}

public sealed record NodeGroupAssignItem(
    [property: JsonPropertyName("node_id")] long NodeId,
    [property: JsonPropertyName("node_ip_id")] long NodeIpId,
    [property: JsonPropertyName("name")] string? Name,
    [property: JsonPropertyName("ip")] string? Ip
);

public sealed class NodeGroupActionRequest
{
    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }

    [JsonPropertyName("value")]
    public string? Value { get; set; }
}
