using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class NodeListQuery
{
    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("region_id")]
    public long? RegionId { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("node_type")]
    public int? NodeType { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("pageSize")]
    public int PageSize { get; set; } = 20;
}

public sealed record NodeListResult(IReadOnlyList<NodeListItem> List, long Total);

public sealed class NodeListItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("pid")]
    public long? Pid { get; set; }

    [JsonPropertyName("region_id")]
    public long? RegionId { get; set; }

    [JsonPropertyName("region_name")]
    public string? RegionName { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("ip")]
    public string? Ip { get; set; }

    [JsonPropertyName("token")]
    public string? Token { get; set; }

    [JsonPropertyName("host")]
    public string? Host { get; set; }

    [JsonPropertyName("port")]
    public int? Port { get; set; }

    [JsonPropertyName("http_proxy")]
    public string? HttpProxy { get; set; }

    [JsonPropertyName("is_mgmt")]
    public bool? IsMgmt { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("config_task")]
    public string? ConfigTask { get; set; }

    [JsonPropertyName("check_on")]
    public bool? CheckOn { get; set; }

    [JsonPropertyName("check_protocol")]
    public string? CheckProtocol { get; set; }

    [JsonPropertyName("check_timeout")]
    public int? CheckTimeout { get; set; }

    [JsonPropertyName("check_port")]
    public int? CheckPort { get; set; }

    [JsonPropertyName("check_host")]
    public string? CheckHost { get; set; }

    [JsonPropertyName("check_path")]
    public string? CheckPath { get; set; }

    [JsonPropertyName("check_node_group")]
    public string? CheckNodeGroup { get; set; }

    [JsonPropertyName("check_action")]
    public string? CheckAction { get; set; }

    [JsonPropertyName("bw_limit")]
    public string? BwLimit { get; set; }

    [JsonPropertyName("type")]
    public int? Type { get; set; }

    [JsonPropertyName("sort_order")]
    public int? SortOrder { get; set; }

    [JsonPropertyName("cache_dir")]
    public string? CacheDir { get; set; }

    [JsonPropertyName("cache_limit")]
    public int? CacheLimit { get; set; }

    [JsonPropertyName("log_dir")]
    public string? LogDir { get; set; }

    [JsonPropertyName("ssh_host")]
    public string? SshHost { get; set; }

    [JsonPropertyName("ssh_port")]
    public int? SshPort { get; set; }

    [JsonPropertyName("ssh_user")]
    public string? SshUser { get; set; }

    [JsonPropertyName("ssh_auth_type")]
    public string? SshAuthType { get; set; }

    [JsonPropertyName("ssh_password")]
    public string? SshPassword { get; set; }

    [JsonPropertyName("ssh_key")]
    public string? SshKey { get; set; }

    [JsonPropertyName("work_dir")]
    public string? WorkDir { get; set; }

    [JsonPropertyName("auto_install")]
    public bool? AutoInstall { get; set; }

    [JsonPropertyName("install_status")]
    public string? InstallStatus { get; set; }

    [JsonPropertyName("install_error")]
    public string? InstallError { get; set; }

    [JsonPropertyName("install_at")]
    public DateTime? InstallAt { get; set; }

    [JsonPropertyName("sub_ips")]
    public IReadOnlyList<NodeSubIp>? SubIps { get; set; }

    [JsonPropertyName("line_count")]
    public long LineCount { get; set; }

    [JsonPropertyName("online")]
    public bool Online { get; set; }

    [JsonPropertyName("install_stage")]
    public string? InstallStage { get; set; }

    [JsonPropertyName("install_progress")]
    public int? InstallProgress { get; set; }

    [JsonPropertyName("install_progress_bytes")]
    public long? InstallProgressBytes { get; set; }

    [JsonPropertyName("install_progress_total")]
    public long? InstallProgressTotal { get; set; }

    [JsonPropertyName("anti_blocking")]
    public bool AntiBlocking { get; set; } = true;

    [JsonPropertyName("reported_anti_blocking")]
    public bool? ReportedAntiBlocking { get; set; }

    [JsonPropertyName("config_drift")]
    public bool ConfigDrift { get; set; }

    [JsonPropertyName("config_drift_fields")]
    public IReadOnlyList<string>? ConfigDriftFields { get; set; }
}

public sealed class NodeMonitorLogQuery
{
    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("start")]
    public string? Start { get; set; }

    [JsonPropertyName("end")]
    public string? End { get; set; }

    [JsonPropertyName("timeRange")]
    public string[]? TimeRange { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("pageSize")]
    public int PageSize { get; set; } = 20;
}

public sealed record NodeMonitorLogResult(IReadOnlyList<NodeMonitorLogItem> List, long Total);

public sealed record NodeMonitorLogItem(DateTime? CheckedAt, long FailCount, long TotalCount);

public class NodeCreateRequest
{
    [JsonPropertyName("region_id")]
    public long? RegionId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("ip")]
    public string? Ip { get; set; }

    [JsonPropertyName("host")]
    public string? Host { get; set; }

    [JsonPropertyName("port")]
    public int? Port { get; set; }

    [JsonPropertyName("http_proxy")]
    public string? HttpProxy { get; set; }

    [JsonPropertyName("is_mgmt")]
    public bool? IsMgmt { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("check_on")]
    public bool? CheckOn { get; set; }

    [JsonPropertyName("check_protocol")]
    public string? CheckProtocol { get; set; }

    [JsonPropertyName("check_timeout")]
    public int? CheckTimeout { get; set; }

    [JsonPropertyName("check_port")]
    public int? CheckPort { get; set; }

    [JsonPropertyName("check_host")]
    public string? CheckHost { get; set; }

    [JsonPropertyName("check_path")]
    public string? CheckPath { get; set; }

    [JsonPropertyName("check_node_group")]
    public string? CheckNodeGroup { get; set; }

    [JsonPropertyName("check_action")]
    public string? CheckAction { get; set; }

    [JsonPropertyName("bw_limit")]
    public string? BwLimit { get; set; }

    [JsonPropertyName("type")]
    public int? Type { get; set; }

    [JsonPropertyName("sort_order")]
    public int? SortOrder { get; set; }

    [JsonPropertyName("cache_dir")]
    public string? CacheDir { get; set; }

    [JsonPropertyName("cache_limit")]
    public int? CacheLimit { get; set; }

    [JsonPropertyName("log_dir")]
    public string? LogDir { get; set; }

    [JsonPropertyName("ssh_host")]
    public string? SshHost { get; set; }

    [JsonPropertyName("ssh_port")]
    public int? SshPort { get; set; }

    [JsonPropertyName("ssh_user")]
    public string? SshUser { get; set; }

    [JsonPropertyName("ssh_auth_type")]
    public string? SshAuthType { get; set; }

    [JsonPropertyName("ssh_password")]
    public string? SshPassword { get; set; }

    [JsonPropertyName("ssh_key")]
    public string? SshKey { get; set; }

    [JsonPropertyName("work_dir")]
    public string? WorkDir { get; set; }

    [JsonPropertyName("auto_install")]
    public bool? AutoInstall { get; set; }

    [JsonPropertyName("sub_ips")]
    public IReadOnlyList<NodeSubIp>? SubIps { get; set; }
}

public sealed class NodeUpdateRequest : NodeCreateRequest
{
}

public sealed class NodeStatusRequest
{
    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }
}

public sealed class NodeAntiBlockingRequest
{
    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }
}

public sealed class NodeBatchRequest
{
    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }
}

public sealed record NodeInstallResult(string? InstallStatus);

public sealed record NodeSubIp(long Id, string? Ip);
