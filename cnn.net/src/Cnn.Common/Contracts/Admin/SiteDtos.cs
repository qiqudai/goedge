using System.Text.Json;
using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class SiteListQuery
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
    public string? GroupId { get; set; }

    [JsonPropertyName("node_group_id")]
    public long? NodeGroupId { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("https")]
    public string? Https { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("pageSize")]
    public int PageSize { get; set; } = 10;

    [JsonPropertyName("size")]
    public int? Size { get; set; }
}

public sealed record SiteListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<SiteListItem> List,
    [property: JsonPropertyName("total")] long Total
);

public class SiteListItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_name")]
    public string? UserName { get; set; }

    [JsonPropertyName("domains")]
    public IReadOnlyList<string> Domains { get; set; } = Array.Empty<string>();

    [JsonPropertyName("domain_display")]
    public string? DomainDisplay { get; set; }

    [JsonPropertyName("listen_ports")]
    public string? ListenPorts { get; set; }

    [JsonPropertyName("http_listen")]
    public IReadOnlyList<string> HttpListen { get; set; } = Array.Empty<string>();

    [JsonPropertyName("https_listen")]
    public IReadOnlyList<string> HttpsListen { get; set; } = Array.Empty<string>();

    [JsonPropertyName("origin_display")]
    public string? OriginDisplay { get; set; }

    [JsonPropertyName("cname")]
    public string? Cname { get; set; }

    [JsonPropertyName("backends")]
    public IReadOnlyList<string> Backends { get; set; } = Array.Empty<string>();

    [JsonPropertyName("https")]
    public bool Https { get; set; }

    [JsonPropertyName("cert_id")]
    public long CertId { get; set; }

    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("user_package_name")]
    public string? UserPackageName { get; set; }

    [JsonPropertyName("dns_provider_id")]
    public long DnsProviderId { get; set; }

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

    [JsonPropertyName("region_id")]
    public long RegionId { get; set; }

    [JsonPropertyName("region_name")]
    public string? RegionName { get; set; }

    [JsonPropertyName("status")]
    public bool Status { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("settings")]
    public Dictionary<string, object?>? Settings { get; set; }

    [JsonPropertyName("expire_time")]
    public string? ExpireTime { get; set; }

    [JsonPropertyName("created_at")]
    public DateTime? CreatedAt { get; set; }

    [JsonPropertyName("updated_at")]
    public DateTime? UpdatedAt { get; set; }
}

public sealed class SiteDetailDto : SiteListItem
{
}

public sealed class SiteCreateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("dns_provider_id")]
    public long DnsProviderId { get; set; }

    [JsonPropertyName("group_id")]
    public long GroupId { get; set; }

    [JsonPropertyName("group_ids")]
    public IReadOnlyList<long>? GroupIds { get; set; }

    [JsonPropertyName("node_group_id")]
    public long NodeGroupId { get; set; }

    [JsonPropertyName("site_type")]
    public string? SiteType { get; set; }

    [JsonPropertyName("domains")]
    public IReadOnlyList<string>? Domains { get; set; }

    [JsonPropertyName("domains_input")]
    public string? DomainsInput { get; set; }

    [JsonPropertyName("backends")]
    public IReadOnlyList<string>? Backends { get; set; }

    [JsonPropertyName("backends_input")]
    public string? BackendsInput { get; set; }
}

public sealed class SiteUpdateRequest
{
    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }

    [JsonPropertyName("user_package_id")]
    public long? UserPackageId { get; set; }

    [JsonPropertyName("group_id")]
    public long? GroupId { get; set; }

    [JsonPropertyName("group_ids")]
    public IReadOnlyList<long>? GroupIds { get; set; }

    [JsonPropertyName("dns_provider_id")]
    public long? DnsProviderId { get; set; }

    [JsonPropertyName("http_listen")]
    public IReadOnlyList<string>? HttpListen { get; set; }

    [JsonPropertyName("https_listen")]
    public IReadOnlyList<string>? HttpsListen { get; set; }

    [JsonPropertyName("balance_way")]
    public string? BalanceWay { get; set; }

    [JsonPropertyName("backend_protocol")]
    public string? BackendProtocol { get; set; }

    [JsonPropertyName("domains")]
    public IReadOnlyList<string>? Domains { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("backends")]
    public IReadOnlyList<string>? Backends { get; set; }

    [JsonPropertyName("settings")]
    public Dictionary<string, object?>? Settings { get; set; }
}

public sealed class SiteBatchCreateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("group_id")]
    public long GroupId { get; set; }

    [JsonPropertyName("dns_provider_id")]
    public long DnsProviderId { get; set; }

    [JsonPropertyName("node_group_id")]
    public long NodeGroupId { get; set; }

    [JsonPropertyName("data")]
    public string? Data { get; set; }

    [JsonPropertyName("ignore_error")]
    public bool IgnoreError { get; set; }
}

public sealed class SiteBatchTaskItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }
}

public sealed record SiteBatchCreateResult(
    [property: JsonPropertyName("batch_id")] string BatchId,
    [property: JsonPropertyName("created")] int Created,
    [property: JsonPropertyName("tasks")] IReadOnlyList<SiteBatchTaskItem> Tasks
);

public sealed class SiteBatchFailItem
{
    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("reason")]
    public string? Reason { get; set; }
}

public sealed class SiteBatchProgressResult
{
    [JsonPropertyName("total")]
    public int Total { get; set; }

    [JsonPropertyName("success")]
    public int Success { get; set; }

    [JsonPropertyName("fail")]
    public int Fail { get; set; }

    [JsonPropertyName("running")]
    public int Running { get; set; }

    [JsonPropertyName("pending")]
    public int Pending { get; set; }

    [JsonPropertyName("done")]
    public int Done { get; set; }

    [JsonPropertyName("percent")]
    public int Percent { get; set; }

    [JsonPropertyName("fail_items")]
    public IReadOnlyList<SiteBatchFailItem> FailItems { get; set; } = Array.Empty<SiteBatchFailItem>();
}

public sealed class SiteBatchUpdateRequest
{
    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }

    [JsonPropertyName("user_package_id")]
    public long? UserPackageId { get; set; }

    [JsonPropertyName("group_id")]
    public long? GroupId { get; set; }

    [JsonPropertyName("group_ids")]
    public IReadOnlyList<long>? GroupIds { get; set; }

    [JsonPropertyName("dns_provider_id")]
    public long? DnsProviderId { get; set; }

    [JsonPropertyName("http_listen")]
    public IReadOnlyList<string>? HttpListen { get; set; }

    [JsonPropertyName("https_listen")]
    public IReadOnlyList<string>? HttpsListen { get; set; }

    [JsonPropertyName("balance_way")]
    public string? BalanceWay { get; set; }

    [JsonPropertyName("backend_protocol")]
    public string? BackendProtocol { get; set; }

    [JsonPropertyName("backends")]
    public IReadOnlyList<string>? Backends { get; set; }

    [JsonPropertyName("cc_default_rule")]
    public long? CcDefaultRule { get; set; }

    [JsonPropertyName("black_ip")]
    public string? BlackIp { get; set; }

    [JsonPropertyName("white_ip")]
    public string? WhiteIp { get; set; }

    [JsonPropertyName("block_region")]
    public string? BlockRegion { get; set; }

    [JsonPropertyName("settings")]
    public Dictionary<string, object?>? Settings { get; set; }

    [JsonPropertyName("cname_domain")]
    public string? CnameDomain { get; set; }

    [JsonPropertyName("cname_mode")]
    public string? CnameMode { get; set; }

    [JsonPropertyName("region_id")]
    public long? RegionId { get; set; }

    [JsonPropertyName("node_group_id")]
    public long? NodeGroupId { get; set; }

    [JsonPropertyName("backup_node_group_id")]
    public long? BackupNodeGroupId { get; set; }

    [JsonPropertyName("enable_backup_group")]
    public bool? EnableBackupGroup { get; set; }
}

public sealed class SiteBatchActionRequest
{
    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }
}

public sealed record SiteBatchActionResult(
    [property: JsonPropertyName("task_id")] long TaskId
);

public sealed class SiteApplyCertRequest
{
    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }
}

public sealed class SiteApplyCertSkipItem
{
    [JsonPropertyName("site_id")]
    public long SiteId { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("reason")]
    public string? Reason { get; set; }
}

public sealed record SiteApplyCertResult(
    [property: JsonPropertyName("created_ids")] IReadOnlyList<long> CreatedIds,
    [property: JsonPropertyName("skipped")] IReadOnlyList<SiteApplyCertSkipItem> Skipped
);

public sealed class SiteResolveResult
{
    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("cname")]
    public string? Cname { get; set; }

    [JsonPropertyName("ips")]
    public IReadOnlyList<string> Ips { get; set; } = Array.Empty<string>();
}

public sealed record SiteExportResult(
    [property: JsonPropertyName("file_name")] string FileName,
    [property: JsonPropertyName("content")] string Content
);
