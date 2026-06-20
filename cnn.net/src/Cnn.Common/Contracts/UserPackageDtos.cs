using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed record UserPackageListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<UserPackageItemDto> List
);

public sealed class UserPackageItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("package")]
    public long PackageId { get; set; }

    [JsonPropertyName("region_id")]
    public long RegionId { get; set; }

    [JsonPropertyName("node_group_id")]
    public long NodeGroupId { get; set; }

    [JsonPropertyName("backup_node_group")]
    public long BackupNodeGroup { get; set; }

    [JsonPropertyName("enable_backup_group")]
    public bool EnableBackupGroup { get; set; }

    [JsonPropertyName("cname_domain")]
    public string? CnameDomain { get; set; }

    [JsonPropertyName("cname_hostname2")]
    public string? CnameHostname2 { get; set; }

    [JsonPropertyName("cname_hostname")]
    public string? CnameHostname { get; set; }

    [JsonPropertyName("cname_mode")]
    public string? CnameMode { get; set; }

    [JsonPropertyName("record_id")]
    public string? RecordId { get; set; }

    [JsonPropertyName("traffic")]
    public long Traffic { get; set; }

    [JsonPropertyName("bandwidth")]
    public string? Bandwidth { get; set; }

    [JsonPropertyName("connection")]
    public long Connection { get; set; }

    [JsonPropertyName("domain")]
    public long Domain { get; set; }

    [JsonPropertyName("main_domain_limit")]
    public long MainDomainLimit { get; set; }

    [JsonPropertyName("http_port")]
    public long HttpPort { get; set; }

    [JsonPropertyName("stream_port")]
    public long StreamPort { get; set; }

    [JsonPropertyName("custom_cc_rule")]
    public bool CustomCcRule { get; set; }

    [JsonPropertyName("websocket")]
    public bool Websocket { get; set; }

    [JsonPropertyName("l2_origin")]
    public bool L2Origin { get; set; }

    [JsonPropertyName("month_price")]
    public long MonthPrice { get; set; }

    [JsonPropertyName("quarter_price")]
    public long QuarterPrice { get; set; }

    [JsonPropertyName("year_price")]
    public long YearPrice { get; set; }

    [JsonPropertyName("create_at")]
    public DateTime? CreateAt { get; set; }

    [JsonPropertyName("start_at")]
    public DateTime? StartAt { get; set; }

    [JsonPropertyName("end_at")]
    public DateTime? EndAt { get; set; }

    [JsonPropertyName("task_id")]
    public long? TaskId { get; set; }

    [JsonPropertyName("version")]
    public int Version { get; set; }

    [JsonPropertyName("is_expired")]
    public bool IsExpired { get; set; }

    [JsonPropertyName("ipv6")]
    public bool IPv6 { get; set; }

    [JsonPropertyName("http3_enabled")]
    public bool Http3Enabled { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }
}

public sealed class UserPackageListQuery
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }
}

public sealed class UserPackageUpdateRequest
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("ipv6")]
    public bool? IPv6 { get; set; }

    [JsonPropertyName("end_at")]
    public string? EndAt { get; set; }

    [JsonPropertyName("region_id")]
    public long? RegionId { get; set; }

    [JsonPropertyName("node_group_id")]
    public long? NodeGroupId { get; set; }

    [JsonPropertyName("backup_group_id")]
    public long? BackupGroupId { get; set; }

    [JsonPropertyName("traffic")]
    public string? Traffic { get; set; }

    [JsonPropertyName("bandwidth")]
    public string? Bandwidth { get; set; }

    [JsonPropertyName("connection")]
    public string? Connection { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("main_domain_limit")]
    public string? MainDomainLimit { get; set; }

    [JsonPropertyName("http_port")]
    public string? HttpPort { get; set; }

    [JsonPropertyName("stream_port")]
    public string? StreamPort { get; set; }

    [JsonPropertyName("custom_cc_rule")]
    public bool? CustomCcRule { get; set; }

    [JsonPropertyName("websocket")]
    public bool? Websocket { get; set; }

    [JsonPropertyName("price_monthly")]
    public double? PriceMonthly { get; set; }

    [JsonPropertyName("price_quarterly")]
    public double? PriceQuarterly { get; set; }

    [JsonPropertyName("price_yearly")]
    public double? PriceYearly { get; set; }

    [JsonPropertyName("cname_hostname")]
    public string? CnameHostname { get; set; }

    [JsonPropertyName("cname_domain")]
    public string? CnameDomain { get; set; }

    [JsonPropertyName("cname_mode")]
    public string? CnameMode { get; set; }

    [JsonPropertyName("http3_enabled")]
    public bool? Http3Enabled { get; set; }
}

public sealed class RenewUserPackageRequest
{
    [JsonPropertyName("period")]
    public string? Period { get; set; }

    [JsonPropertyName("months")]
    public int? Months { get; set; }
}

public sealed class SwitchUserPackageRequest
{
    [JsonPropertyName("package_id")]
    public long? PackageId { get; set; }
}

public sealed record RenewUserPackageResult(
    [property: JsonPropertyName("end_at")] DateTime EndAt
);
