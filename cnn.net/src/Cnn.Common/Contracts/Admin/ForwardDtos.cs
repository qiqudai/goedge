using System.Text.Json;
using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class ForwardListQuery
{
    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("search_field")]
    public string? SearchField { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("user_package_id")]
    public long? UserPackageId { get; set; }

    [JsonPropertyName("group_id")]
    public long? GroupId { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("pageSize")]
    public int PageSize { get; set; } = 10;
}

public sealed record ForwardListResult(IReadOnlyList<ForwardListItem> List, long Total);

public sealed class ForwardListItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_name")]
    public string? UserName { get; set; }

    [JsonPropertyName("listen_ports")]
    public string? ListenPorts { get; set; }

    [JsonPropertyName("origin_display")]
    public string? OriginDisplay { get; set; }

    [JsonPropertyName("origin")]
    public string? Origin { get; set; }

    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("user_package_name")]
    public string? UserPackageName { get; set; }

    [JsonPropertyName("group_id")]
    public long GroupId { get; set; }

    [JsonPropertyName("group_ids")]
    public IReadOnlyList<long> GroupIds { get; set; } = Array.Empty<long>();

    [JsonPropertyName("group_name")]
    public string? GroupName { get; set; }

    [JsonPropertyName("node_group_id")]
    public long NodeGroupId { get; set; }

    [JsonPropertyName("node_group_name")]
    public string? NodeGroupName { get; set; }

    [JsonPropertyName("cname")]
    public string? Cname { get; set; }

    [JsonPropertyName("status")]
    public bool Status { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("created_at")]
    public DateTime? CreatedAt { get; set; }
}

public sealed class ForwardDetailDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("region_id")]
    public long RegionId { get; set; }

    [JsonPropertyName("node_group_id")]
    public long NodeGroupId { get; set; }

    [JsonPropertyName("backup_node_group")]
    public long BackupNodeGroup { get; set; }

    [JsonPropertyName("enable_backup_group")]
    public bool EnableBackupGroup { get; set; }

    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("cname_domain")]
    public string? CnameDomain { get; set; }

    [JsonPropertyName("cname_hostname2")]
    public string? CnameHostname2 { get; set; }

    [JsonPropertyName("cname_mode")]
    public string? CnameMode { get; set; }

    [JsonPropertyName("cname")]
    public string? Cname { get; set; }

    [JsonPropertyName("listen_ports")]
    public IReadOnlyList<string> ListenPorts { get; set; } = Array.Empty<string>();

    [JsonPropertyName("origins")]
    public IReadOnlyList<ForwardOriginDto> Origins { get; set; } = Array.Empty<ForwardOriginDto>();

    [JsonPropertyName("backend_port")]
    public string? BackendPort { get; set; }

    [JsonPropertyName("balance_way")]
    public string? BalanceWay { get; set; }

    [JsonPropertyName("proxy_protocol")]
    public bool ProxyProtocol { get; set; }

    [JsonPropertyName("conn_limit")]
    public string? ConnLimit { get; set; }

    [JsonPropertyName("settings")]
    public Dictionary<string, JsonElement>? Settings { get; set; }

    [JsonPropertyName("create_at")]
    public DateTime? CreatedAt { get; set; }

    [JsonPropertyName("update_at")]
    public DateTime? UpdatedAt { get; set; }
}

public sealed class ForwardOriginDto
{
    [JsonPropertyName("address")]
    public string? Address { get; set; }

    [JsonPropertyName("weight")]
    public int Weight { get; set; }

    [JsonPropertyName("enable")]
    public bool Enable { get; set; }
}

public sealed class ForwardCreateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("node_group_id")]
    public long NodeGroupId { get; set; }

    [JsonPropertyName("group_id")]
    public long GroupId { get; set; }

    [JsonPropertyName("group_ids")]
    public IReadOnlyList<long>? GroupIds { get; set; }

    [JsonPropertyName("listen_ports")]
    public IReadOnlyList<string>? ListenPorts { get; set; }

    [JsonPropertyName("listen_ports_input")]
    public string? ListenPortsInput { get; set; }

    [JsonPropertyName("origin_input")]
    public string? OriginInput { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed class ForwardUpdateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("listen_ports")]
    public string? ListenPorts { get; set; }

    [JsonPropertyName("listen_ports_input")]
    public string? ListenPortsInput { get; set; }

    [JsonPropertyName("origin")]
    public string? Origin { get; set; }

    [JsonPropertyName("origin_input")]
    public string? OriginInput { get; set; }

    [JsonPropertyName("group_id")]
    public long GroupId { get; set; }

    [JsonPropertyName("group_ids")]
    public IReadOnlyList<long>? GroupIds { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed class ForwardBatchCreateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("group_id")]
    public long GroupId { get; set; }

    [JsonPropertyName("group_ids")]
    public IReadOnlyList<long>? GroupIds { get; set; }

    [JsonPropertyName("data")]
    public string? Data { get; set; }

    [JsonPropertyName("ignore_error")]
    public bool IgnoreError { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed record ForwardBatchCreateResult(
    [property: JsonPropertyName("created")] int Created
);

public sealed class ForwardBatchUpdateRequest
{
    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }

    [JsonPropertyName("user_package_id")]
    public long? UserPackageId { get; set; }

    [JsonPropertyName("group_id")]
    public long? GroupId { get; set; }

    [JsonPropertyName("group_ids")]
    public IReadOnlyList<long>? GroupIds { get; set; }

    [JsonPropertyName("listen_ports")]
    public IReadOnlyList<string>? ListenPorts { get; set; }

    [JsonPropertyName("origins")]
    public IReadOnlyList<ForwardOriginDto>? Origins { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("settings")]
    public Dictionary<string, JsonElement>? Settings { get; set; }
}

public sealed class ForwardBatchActionRequest
{
    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }
}

public sealed record ForwardBatchActionResult(
    [property: JsonPropertyName("task_id")] long TaskId
);

public sealed class ForwardGroupDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed record ForwardGroupListResult(IReadOnlyList<ForwardGroupDto> List);

public sealed class ForwardGroupUpsertRequest
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed class ForwardGroupDeleteRequest
{
    [JsonPropertyName("id")]
    public long Id { get; set; }
}

public sealed class ForwardDefaultItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("id_str")]
    public string? IdStr { get; set; }

    [JsonPropertyName("key")]
    public string? Key { get; set; }

    [JsonPropertyName("value")]
    public object? Value { get; set; }

    [JsonPropertyName("scope")]
    public string? Scope { get; set; }

    [JsonPropertyName("group_id")]
    public long GroupId { get; set; }

    [JsonPropertyName("group_name")]
    public string? GroupName { get; set; }
}

public sealed record ForwardDefaultListResult(IReadOnlyList<ForwardDefaultItemDto> List);

public sealed class ForwardDefaultCreateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("key")]
    public string? Key { get; set; }

    [JsonPropertyName("value")]
    public JsonElement? Value { get; set; }

    [JsonPropertyName("scope")]
    public string? Scope { get; set; }

    [JsonPropertyName("group_id")]
    public long GroupId { get; set; }
}

public sealed class ForwardDefaultDeleteRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("id_str")]
    public string? IdStr { get; set; }
}

public sealed class ForwardTrafficResult
{
    [JsonPropertyName("x_axis")]
    public IReadOnlyList<string> XAxis { get; set; } = Array.Empty<string>();

    [JsonPropertyName("bandwidth")]
    public IReadOnlyList<double> Bandwidth { get; set; } = Array.Empty<double>();

    [JsonPropertyName("traffic")]
    public IReadOnlyList<double> Traffic { get; set; } = Array.Empty<double>();
}

public sealed class ForwardRankingItemDto
{
    [JsonPropertyName("port")]
    public string? Port { get; set; }

    [JsonPropertyName("connections")]
    public ulong Connections { get; set; }

    [JsonPropertyName("traffic")]
    public string? Traffic { get; set; }
}

public sealed record ForwardRankingResult(IReadOnlyList<ForwardRankingItemDto> List);
