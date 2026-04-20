using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed record PlanListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<PlanItemDto> List,
    [property: JsonPropertyName("total")] int Total
);

public sealed class PlanItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("desc")]
    public string? Desc { get; set; }

    [JsonPropertyName("group")]
    public string? Group { get; set; }

    [JsonPropertyName("region")]
    public long Region { get; set; }

    [JsonPropertyName("line_group")]
    public long LineGroup { get; set; }

    [JsonPropertyName("backup_group")]
    public long BackupGroup { get; set; }

    [JsonPropertyName("traffic_limit")]
    public long TrafficLimit { get; set; }

    [JsonPropertyName("bandwidth_limit")]
    public string? BandwidthLimit { get; set; }

    [JsonPropertyName("connection_limit")]
    public long ConnectionLimit { get; set; }

    [JsonPropertyName("domain_limit")]
    public long DomainLimit { get; set; }

    [JsonPropertyName("custom_cc_rules")]
    public bool CustomCcRules { get; set; }

    [JsonPropertyName("websocket")]
    public bool Websocket { get; set; }

    [JsonPropertyName("l2_origin")]
    public bool L2Origin { get; set; }

    [JsonPropertyName("price_monthly")]
    public long PriceMonthly { get; set; }

    [JsonPropertyName("price_quarterly")]
    public long PriceQuarterly { get; set; }

    [JsonPropertyName("price_yearly")]
    public long PriceYearly { get; set; }

    [JsonPropertyName("sort_order")]
    public int SortOrder { get; set; }

    [JsonPropertyName("status")]
    public bool Status { get; set; }
}

public sealed class PlanDetailDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("desc")]
    public string? Desc { get; set; }

    [JsonPropertyName("group")]
    public string? Group { get; set; }

    [JsonPropertyName("region")]
    public long Region { get; set; }

    [JsonPropertyName("line_group")]
    public long LineGroup { get; set; }

    [JsonPropertyName("backup_group")]
    public long BackupGroup { get; set; }

    [JsonPropertyName("traffic_limit")]
    public long TrafficLimit { get; set; }

    [JsonPropertyName("bandwidth_limit")]
    public string? BandwidthLimit { get; set; }

    [JsonPropertyName("connection_limit")]
    public long ConnectionLimit { get; set; }

    [JsonPropertyName("domain_limit")]
    public long DomainLimit { get; set; }

    [JsonPropertyName("custom_cc_rules")]
    public bool CustomCcRules { get; set; }

    [JsonPropertyName("websocket")]
    public bool Websocket { get; set; }

    [JsonPropertyName("l2_origin")]
    public bool L2Origin { get; set; }

    [JsonPropertyName("price_monthly")]
    public long PriceMonthly { get; set; }

    [JsonPropertyName("price_quarterly")]
    public long PriceQuarterly { get; set; }

    [JsonPropertyName("price_yearly")]
    public long PriceYearly { get; set; }

    [JsonPropertyName("sort_order")]
    public int SortOrder { get; set; }

    [JsonPropertyName("status")]
    public bool Status { get; set; }

    [JsonPropertyName("http_port")]
    public long HttpPort { get; set; }

    [JsonPropertyName("stream_port")]
    public long StreamPort { get; set; }

    [JsonPropertyName("cname_domain")]
    public string? CnameDomain { get; set; }

    [JsonPropertyName("cname_hostname2")]
    public string? CnameHostname2 { get; set; }

    [JsonPropertyName("cname_mode")]
    public string? CnameMode { get; set; }

    [JsonPropertyName("buy_num_limit")]
    public long BuyNumLimit { get; set; }

    [JsonPropertyName("backend_ip_limit")]
    public string? BackendIpLimit { get; set; }

    [JsonPropertyName("id_verify")]
    public bool IdVerify { get; set; }

    [JsonPropertyName("before_exp_days_renew")]
    public long BeforeExpDaysRenew { get; set; }

    [JsonPropertyName("expire")]
    public DateTime? Expire { get; set; }

    [JsonPropertyName("owner")]
    public string? Owner { get; set; }
}

public sealed record UserPlanListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<UserPlanItemDto> List
);

public sealed class UserPlanItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("user_name")]
    public string? UserName { get; set; }

    [JsonPropertyName("package_id")]
    public long PackageId { get; set; }

    [JsonPropertyName("package_name")]
    public string? PackageName { get; set; }

    [JsonPropertyName("plan_name")]
    public string? PlanName { get; set; }

    [JsonPropertyName("record_id")]
    public string? RecordId { get; set; }

    [JsonPropertyName("region_id")]
    public long RegionId { get; set; }

    [JsonPropertyName("node_group_id")]
    public long NodeGroupId { get; set; }

    [JsonPropertyName("backup_group_id")]
    public long BackupGroupId { get; set; }

    [JsonPropertyName("enable_backup_group")]
    public bool EnableBackupGroup { get; set; }

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

    [JsonPropertyName("http3_enabled")]
    public bool Http3Enabled { get; set; }
    public string? CnameDomain { get; set; }

    [JsonPropertyName("cname_hostname")]
    public string? CnameHostname { get; set; }

    [JsonPropertyName("cname_hostname2")]
    public string? CnameHostname2 { get; set; }

    [JsonPropertyName("cname_mode")]
    public string? CnameMode { get; set; }

    [JsonPropertyName("start_at")]
    public DateTime? StartAt { get; set; }

    [JsonPropertyName("end_at")]
    public DateTime? EndAt { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("created_at")]
    public DateTime? CreatedAt { get; set; }
}

public sealed class AssignUserPlanRequest
{
    [JsonPropertyName("plan_id")]
    public long? PlanId { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("duration_months")]
    public int? DurationMonths { get; set; }

    [JsonPropertyName("end_at")]
    public string? EndAt { get; set; }
}

public sealed class DeleteUserPlansRequest
{
    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }
}
