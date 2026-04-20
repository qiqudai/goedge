using System.Text.Json;
using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class CertListQuery
{
    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("search_field")]
    public string? SearchField { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("pageSize")]
    public int PageSize { get; set; } = 10;
}

public sealed record CertListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<CertItemDto> List,
    [property: JsonPropertyName("total")] long Total
);

public sealed class CertItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("uid")]
    public long UserId { get; set; }

    [JsonPropertyName("user_name")]
    public string? UserName { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Description { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("dnsapi")]
    public int DnsApi { get; set; }

    [JsonPropertyName("cert")]
    public string? CertPem { get; set; }

    [JsonPropertyName("key")]
    public string? KeyPem { get; set; }

    [JsonPropertyName("start_time")]
    public DateTime? StartTime { get; set; }

    [JsonPropertyName("expire_time")]
    public DateTime? ExpireTime { get; set; }

    [JsonPropertyName("auto_renew")]
    public bool AutoRenew { get; set; }

    [JsonPropertyName("create_at")]
    public DateTime? CreateAt { get; set; }

    [JsonPropertyName("update_at")]
    public DateTime? UpdateAt { get; set; }

    [JsonPropertyName("enable")]
    public bool Enable { get; set; }

    [JsonPropertyName("task_id")]
    public long? TaskId { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }

    [JsonPropertyName("version")]
    public int? Version { get; set; }

    [JsonPropertyName("issue_task_ret")]
    public string? IssueTaskRet { get; set; }

    [JsonPropertyName("issue_task_state")]
    public string? IssueTaskState { get; set; }

    [JsonPropertyName("issue_task_retry_at")]
    public DateTime? IssueTaskRetryAt { get; set; }

    [JsonPropertyName("issue_task_err_times")]
    public int? IssueTaskErrTimes { get; set; }
}

public sealed class CertCreateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Description { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("dnsapi")]
    public int DnsApi { get; set; }

    [JsonPropertyName("cert")]
    public string? CertPem { get; set; }

    [JsonPropertyName("key")]
    public string? KeyPem { get; set; }

    [JsonPropertyName("auto_renew")]
    public bool AutoRenew { get; set; } = true;
}

public sealed class CertUpdateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("des")]
    public string? Description { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("dnsapi")]
    public int DnsApi { get; set; }

    [JsonPropertyName("cert")]
    public string? CertPem { get; set; }

    [JsonPropertyName("key")]
    public string? KeyPem { get; set; }

    [JsonPropertyName("auto_renew")]
    public bool AutoRenew { get; set; }
}

public sealed class CertBatchCreateRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("dnsapi")]
    public int DnsApi { get; set; }

    [JsonPropertyName("auto_renew")]
    public bool AutoRenew { get; set; } = true;

    [JsonPropertyName("domains")]
    public JsonElement Domains { get; set; }
}

public sealed record CertBatchCreateResult(
    [property: JsonPropertyName("batch_id")] string BatchId,
    [property: JsonPropertyName("count")] int Count,
    [property: JsonPropertyName("ids")] IReadOnlyList<long> Ids
);

public sealed class CertBatchFailItem
{
    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("reason")]
    public string? Reason { get; set; }
}

public sealed class CertBatchProgressResult
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
    public IReadOnlyList<CertBatchFailItem> FailItems { get; set; } = Array.Empty<CertBatchFailItem>();
}

public sealed class CertWildcardRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("dnsapi")]
    public int DnsApi { get; set; }

    [JsonPropertyName("auto_renew")]
    public bool AutoRenew { get; set; } = true;

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }
}

public sealed record CertWildcardResult(
    [property: JsonPropertyName("id")] long Id
);

public sealed class CertBatchActionRequest
{
    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }
}

public sealed record CertBatchActionResult(
    [property: JsonPropertyName("task_id")] long TaskId
);

public sealed class CertReissueRequest
{
    [JsonPropertyName("ids")]
    public IReadOnlyList<long>? Ids { get; set; }
}

public sealed class CertDefaultSettingsRequest
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("dnsapi")]
    public int DnsApi { get; set; }
}

public sealed class CertDefaultSettingsDto
{
    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("dnsapi")]
    public int DnsApi { get; set; }
}

public sealed class DnsChallengeInfoDto
{
    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("fqdn")]
    public string? Fqdn { get; set; }

    [JsonPropertyName("record_name")]
    public string? RecordName { get; set; }

    [JsonPropertyName("record_value")]
    public string? RecordValue { get; set; }

    [JsonPropertyName("record_type")]
    public string? RecordType { get; set; }

    [JsonPropertyName("zone")]
    public string? Zone { get; set; }
}
