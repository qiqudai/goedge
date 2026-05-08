using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class LoginLogQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }
}

public sealed record LoginLogListResult(IReadOnlyList<LoginLogItem> List, long Total);

public sealed class LoginLogItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("username")]
    public string? Username { get; set; }

    [JsonPropertyName("ip")]
    public string? Ip { get; set; }

    [JsonPropertyName("success")]
    public bool Success { get; set; }

    [JsonPropertyName("post_content")]
    public string? PostContent { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }
}

public sealed class OperationLogQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }
}

public sealed record OperationLogListResult(IReadOnlyList<OperationLogItem> List, long Total);

public sealed class OperationLogItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("content")]
    public string? Content { get; set; }

    [JsonPropertyName("diff")]
    public string? Diff { get; set; }

    [JsonPropertyName("ip")]
    public string? Ip { get; set; }

    [JsonPropertyName("process")]
    public string? Process { get; set; }

    [JsonPropertyName("description")]
    public string? Description { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("username")]
    public string? Username { get; set; }
}

public sealed class BackupLogQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }
}

public sealed record BackupLogListResult(IReadOnlyList<BackupLogItem> List, long Total);

public sealed class BackupLogItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("finished_at")]
    public string? FinishedAt { get; set; }

    [JsonPropertyName("status")]
    public int Status { get; set; }

    [JsonPropertyName("result")]
    public string? Result { get; set; }
}

public sealed class MailLogQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }
}

public sealed record MailLogListResult(IReadOnlyList<MailLogItem> List, long Total);

public sealed class MailLogItem
{
    [JsonPropertyName("message_id")]
    public long MessageId { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("subject")]
    public string? Subject { get; set; }

    [JsonPropertyName("medium")]
    public string? Medium { get; set; }

    [JsonPropertyName("fails")]
    public int Fails { get; set; }

    [JsonPropertyName("status")]
    public int Status { get; set; }

    [JsonPropertyName("reason")]
    public string? Reason { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }
}

public sealed class AccessLogQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("domain_mode")]
    public string? DomainMode { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("client_ip")]
    public string? ClientIp { get; set; }

    [JsonPropertyName("uri")]
    public string? Uri { get; set; }

    [JsonPropertyName("uri_mode")]
    public string? UriMode { get; set; }

    [JsonPropertyName("method")]
    public string? Method { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("status_min")]
    public string? StatusMin { get; set; }

    [JsonPropertyName("status_max")]
    public string? StatusMax { get; set; }

    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("node_ip")]
    public string? NodeIp { get; set; }

    [JsonPropertyName("port")]
    public string? Port { get; set; }

    [JsonPropertyName("scheme")]
    public string? Scheme { get; set; }

    [JsonPropertyName("cache_status")]
    public string? CacheStatus { get; set; }

    [JsonPropertyName("referer")]
    public string? Referer { get; set; }

    [JsonPropertyName("user_agent")]
    public string? UserAgent { get; set; }

    [JsonPropertyName("ssl_protocol")]
    public string? SslProtocol { get; set; }

    [JsonPropertyName("ssl_cipher")]
    public string? SslCipher { get; set; }
}

public sealed record AccessLogListResult(IReadOnlyList<AccessLogItem> List, long Total);

public sealed class AccessLogItem
{
    [JsonPropertyName("timestamp")]
    public string? Timestamp { get; set; }

    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("node_ip")]
    public string? NodeIp { get; set; }

    [JsonPropertyName("remote_addr")]
    public string? RemoteAddr { get; set; }

    [JsonPropertyName("host")]
    public string? Host { get; set; }

    [JsonPropertyName("method")]
    public string? Method { get; set; }

    [JsonPropertyName("uri")]
    public string? Uri { get; set; }

    [JsonPropertyName("status")]
    public int Status { get; set; }

    [JsonPropertyName("bytes")]
    public long Bytes { get; set; }

    [JsonPropertyName("request_time")]
    public double RequestTime { get; set; }

    [JsonPropertyName("upstream_addr")]
    public string? UpstreamAddr { get; set; }

    [JsonPropertyName("upstream_response_time")]
    public double UpstreamResponseTime { get; set; }

    [JsonPropertyName("upstream_cache_status")]
    public string? UpstreamCacheStatus { get; set; }

    [JsonPropertyName("http_referer")]
    public string? HttpReferer { get; set; }

    [JsonPropertyName("http_user_agent")]
    public string? HttpUserAgent { get; set; }

    [JsonPropertyName("scheme")]
    public string? Scheme { get; set; }

    [JsonPropertyName("ssl_protocol")]
    public string? SslProtocol { get; set; }

    [JsonPropertyName("ssl_cipher")]
    public string? SslCipher { get; set; }
}

public sealed class AccessLogDownloadApplyRequest
{
    [JsonPropertyName("query")]
    public AccessLogQuery? Query { get; set; }

    [JsonPropertyName("start_time")]
    public string? StartTime { get; set; }

    [JsonPropertyName("end_time")]
    public string? EndTime { get; set; }

    [JsonPropertyName("file_name")]
    public string? FileName { get; set; }
}

public sealed record AccessLogDownloadApplyResult(
    [property: JsonPropertyName("id")] long Id,
    [property: JsonPropertyName("state")] string State
);

public sealed class AccessLogDownloadCompleteRequest
{
    [JsonPropertyName("success")]
    public bool Success { get; set; } = true;

    [JsonPropertyName("rows")]
    public long? Rows { get; set; }

    [JsonPropertyName("error")]
    public string? Error { get; set; }
}

public sealed class AccessLogDownloadQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }
}

public sealed record AccessLogDownloadListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<AccessLogDownloadItem> List,
    [property: JsonPropertyName("total")] long Total
);

public sealed class AccessLogDownloadItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("file_name")]
    public string? FileName { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("rows")]
    public long Rows { get; set; }

    [JsonPropertyName("error")]
    public string? Error { get; set; }

    [JsonPropertyName("requester_user_id")]
    public long RequesterUserId { get; set; }

    [JsonPropertyName("scope")]
    public string? Scope { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("finished_at")]
    public string? FinishedAt { get; set; }
}

public sealed class BlockLogQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("range")]
    public string? Range { get; set; }

    [JsonPropertyName("time_range")]
    public string? TimeRange { get; set; }

    [JsonPropertyName("start_time")]
    public string? StartTime { get; set; }

    [JsonPropertyName("end_time")]
    public string? EndTime { get; set; }
}

public sealed record BlockCurrentListResult(IReadOnlyList<BlockCurrentItem> List, long Total);

public sealed class BlockCurrentItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("site_id")]
    public long SiteId { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("ip")]
    public string? Ip { get; set; }

    [JsonPropertyName("location")]
    public string? Location { get; set; }

    [JsonPropertyName("filter")]
    public string? Filter { get; set; }

    [JsonPropertyName("block_module")]
    public string? BlockModule { get; set; }

    [JsonPropertyName("block_rule")]
    public string? BlockRule { get; set; }

    [JsonPropertyName("block_rule_id")]
    public long BlockRuleId { get; set; }

    [JsonPropertyName("block_config")]
    public string? BlockConfig { get; set; }

    [JsonPropertyName("block_source")]
    public string? BlockSource { get; set; }

    [JsonPropertyName("block_time")]
    public string? BlockTime { get; set; }

    [JsonPropertyName("release_time")]
    public string? ReleaseTime { get; set; }
}

public sealed record BlockStatListResult(IReadOnlyList<BlockStatItem> List, long Total);

public sealed class BlockStatItem
{
    [JsonPropertyName("site_id")]
    public long SiteId { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("count")]
    public long Count { get; set; }
}

public sealed record BlockHistoryListResult(IReadOnlyList<BlockHistoryItem> List, long Total);

public sealed class BlockHistoryItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("site_id")]
    public long SiteId { get; set; }

    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("ip")]
    public string? Ip { get; set; }

    [JsonPropertyName("location")]
    public string? Location { get; set; }

    [JsonPropertyName("filter")]
    public string? Filter { get; set; }

    [JsonPropertyName("block_module")]
    public string? BlockModule { get; set; }

    [JsonPropertyName("block_rule")]
    public string? BlockRule { get; set; }

    [JsonPropertyName("block_rule_id")]
    public long BlockRuleId { get; set; }

    [JsonPropertyName("block_config")]
    public string? BlockConfig { get; set; }

    [JsonPropertyName("block_source")]
    public string? BlockSource { get; set; }

    [JsonPropertyName("block_time")]
    public string? BlockTime { get; set; }

    [JsonPropertyName("is_manual")]
    public bool IsManual { get; set; }
}

public sealed class EventLogQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("event_type")]
    public string? EventType { get; set; }

    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("node_ip")]
    public string? NodeIp { get; set; }

    [JsonPropertyName("trace_id")]
    public string? TraceId { get; set; }

    [JsonPropertyName("host")]
    public string? Host { get; set; }

    [JsonPropertyName("status")]
    public int? Status { get; set; }

    [JsonPropertyName("status_code")]
    public int? StatusCode { get; set; }

    [JsonPropertyName("direction")]
    public string? Direction { get; set; }

    [JsonPropertyName("channel")]
    public string? Channel { get; set; }

    [JsonPropertyName("kind")]
    public string? Kind { get; set; }

    [JsonPropertyName("method")]
    public string? Method { get; set; }

    [JsonPropertyName("path")]
    public string? Path { get; set; }

    [JsonPropertyName("msg_id")]
    public string? MsgId { get; set; }

    [JsonPropertyName("task_id")]
    public long? TaskId { get; set; }

    [JsonPropertyName("task_type")]
    public string? TaskType { get; set; }

    [JsonPropertyName("sync_action")]
    public string? SyncAction { get; set; }

    [JsonPropertyName("rate_limit_window_second")]
    public long? RateLimitWindowSecond { get; set; }

    [JsonPropertyName("rate_limit_dropped_min")]
    public int? RateLimitDroppedMin { get; set; }

    [JsonPropertyName("rate_limit_dropped_max")]
    public int? RateLimitDroppedMax { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }
}

public sealed record EventLogListResult(IReadOnlyList<EventLogItem> List, long Total);

public sealed class EventLogItem
{
    [JsonPropertyName("timestamp")]
    public string? Timestamp { get; set; }

    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("node_ip")]
    public string? NodeIp { get; set; }

    [JsonPropertyName("event_type")]
    public string? EventType { get; set; }

    [JsonPropertyName("payload")]
    public string? Payload { get; set; }

    [JsonPropertyName("trace_id")]
    public string? TraceId { get; set; }

    [JsonPropertyName("host")]
    public string? Host { get; set; }

    [JsonPropertyName("status")]
    public int? Status { get; set; }

    [JsonPropertyName("status_code")]
    public int? StatusCode { get; set; }

    [JsonPropertyName("direction")]
    public string? Direction { get; set; }

    [JsonPropertyName("channel")]
    public string? Channel { get; set; }

    [JsonPropertyName("kind")]
    public string? Kind { get; set; }

    [JsonPropertyName("method")]
    public string? Method { get; set; }

    [JsonPropertyName("path")]
    public string? Path { get; set; }

    [JsonPropertyName("msg_id")]
    public string? MsgId { get; set; }

    [JsonPropertyName("task_id")]
    public long? TaskId { get; set; }

    [JsonPropertyName("task_type")]
    public string? TaskType { get; set; }

    [JsonPropertyName("sync_action")]
    public string? SyncAction { get; set; }

    [JsonPropertyName("rate_limit_max_events_per_sec")]
    public int? RateLimitMaxEventsPerSec { get; set; }

    [JsonPropertyName("rate_limit_window_second")]
    public long? RateLimitWindowSecond { get; set; }

    [JsonPropertyName("rate_limit_accepted_events")]
    public int? RateLimitAcceptedEvents { get; set; }

    [JsonPropertyName("rate_limit_dropped_events")]
    public int? RateLimitDroppedEvents { get; set; }
}
