using System.Text.Json;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Stats;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Admin;

public interface IEventLogService
{
    Task<ServiceResult<EventLogListResult>> ListAsync(
        EventLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken);
}

public sealed class EventLogService : IEventLogService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;
    private readonly IConfiguration _configuration;
    private readonly ISiteHostIndexService _siteHostIndexService;

    public EventLogService(IConfiguration configuration, ISiteHostIndexService siteHostIndexService)
    {
        _configuration = configuration;
        _siteHostIndexService = siteHostIndexService;
    }

    public async Task<ServiceResult<EventLogListResult>> ListAsync(
        EventLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new EventLogQuery();
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return ServiceResult<EventLogListResult>.Ok(new EventLogListResult(Array.Empty<EventLogItem>(), 0));
        }

        HostFilter? hostFilter = null;
        if (!isAdmin)
        {
            if (!userId.HasValue || userId.Value <= 0)
            {
                return ServiceResult<EventLogListResult>.Ok(new EventLogListResult(Array.Empty<EventLogItem>(), 0));
            }

            var index = await _siteHostIndexService.LoadAsync(userId.Value, cancellationToken);
            if (index.Filter.Empty)
            {
                return ServiceResult<EventLogListResult>.Ok(new EventLogListResult(Array.Empty<EventLogItem>(), 0));
            }

            hostFilter = index.Filter;
        }

        var where = BuildWhere(query, startTime, endTime, hostFilter);
        var offset = (page - 1) * pageSize;

        var total = await QueryCountAsync(cfg, where, cancellationToken);
        if (total == 0)
        {
            return ServiceResult<EventLogListResult>.Ok(new EventLogListResult(Array.Empty<EventLogItem>(), 0));
        }

        var rows = await QueryRowsAsync(cfg, where, pageSize, offset, cancellationToken);
        return ServiceResult<EventLogListResult>.Ok(new EventLogListResult(rows, total));
    }

    private static string BuildWhere(EventLogQuery query, DateTime? startTime, DateTime? endTime, HostFilter? hostFilter)
    {
        var conditions = new List<string> { "1=1" };
        if (startTime.HasValue && endTime.HasValue)
        {
            var start = startTime.Value.ToString("yyyy-MM-dd HH:mm:ss");
            var end = endTime.Value.ToString("yyyy-MM-dd HH:mm:ss");
            conditions.Add($"ts >= toDateTime('{start}') AND ts <= toDateTime('{end}')");
        }

        var eventType = query.EventType?.Trim();
        if (!string.IsNullOrWhiteSpace(eventType))
        {
            conditions.Add("event_type = " + ClickHouseHttpHelper.QuoteString(eventType));
        }

        var nodeId = query.NodeId?.Trim();
        if (!string.IsNullOrWhiteSpace(nodeId))
        {
            conditions.Add("node_id = " + ClickHouseHttpHelper.QuoteString(nodeId));
        }

        var nodeIp = query.NodeIp?.Trim();
        if (!string.IsNullOrWhiteSpace(nodeIp))
        {
            conditions.Add("node_ip = " + ClickHouseHttpHelper.QuoteString(nodeIp));
        }

        var traceId = query.TraceId?.Trim();
        if (!string.IsNullOrWhiteSpace(traceId))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"trace_id\":\"" + traceId + "\"%"));
        }

        var host = query.Host?.Trim();
        if (!string.IsNullOrWhiteSpace(host))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"host\":\"" + host + "\"%"));
        }

        if (query.Status.HasValue)
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"status\":" + query.Status.Value + "%"));
        }

        if (query.StatusCode.HasValue)
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"status_code\":" + query.StatusCode.Value + "%"));
        }

        var direction = query.Direction?.Trim();
        if (!string.IsNullOrWhiteSpace(direction))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"direction\":\"" + direction + "\"%"));
        }

        var channel = query.Channel?.Trim();
        if (!string.IsNullOrWhiteSpace(channel))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"channel\":\"" + channel + "\"%"));
        }

        var kind = query.Kind?.Trim();
        if (!string.IsNullOrWhiteSpace(kind))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"kind\":\"" + kind + "\"%"));
        }

        var method = query.Method?.Trim();
        if (!string.IsNullOrWhiteSpace(method))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"method\":\"" + method + "\"%"));
        }

        var path = query.Path?.Trim();
        if (!string.IsNullOrWhiteSpace(path))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"path\":\"" + path + "\"%"));
        }

        var msgId = query.MsgId?.Trim();
        if (!string.IsNullOrWhiteSpace(msgId))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"msg_id\":\"" + msgId + "\"%"));
        }

        if (query.TaskId.HasValue)
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"task_id\":" + query.TaskId.Value + "%"));
        }

        var taskType = query.TaskType?.Trim();
        if (!string.IsNullOrWhiteSpace(taskType))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"task_type\":\"" + taskType + "\"%"));
        }

        var syncAction = query.SyncAction?.Trim();
        if (!string.IsNullOrWhiteSpace(syncAction))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"sync_action\":\"" + syncAction + "\"%"));
        }

        if (query.RateLimitWindowSecond.HasValue)
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"window_second\":" + query.RateLimitWindowSecond.Value + "%"));
        }

        if (query.RateLimitDroppedMin.HasValue)
        {
            conditions.Add("toInt32OrZero(extract(payload, '\"dropped_events\":([0-9]+)')) >= " + query.RateLimitDroppedMin.Value);
        }

        if (query.RateLimitDroppedMax.HasValue)
        {
            conditions.Add("toInt32OrZero(extract(payload, '\"dropped_events\":([0-9]+)')) <= " + query.RateLimitDroppedMax.Value);
        }

        if (hostFilter != null && !hostFilter.Empty)
        {
            var hostConditions = new List<string>();
            foreach (var exact in hostFilter.Exact)
            {
                hostConditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"host\":\"" + exact + "\"%"));
            }

            foreach (var suffix in hostFilter.Wildcards)
            {
                hostConditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%\"host\":\"%" + suffix + "\"%"));
            }

            if (hostConditions.Count == 0)
            {
                return "0=1";
            }

            conditions.Add("(" + string.Join(" OR ", hostConditions) + ")");
        }

        var keyword = query.Keyword?.Trim();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            conditions.Add("payload LIKE " + ClickHouseHttpHelper.QuoteString("%" + keyword + "%"));
        }

        return string.Join(" AND ", conditions);
    }

    private static async Task<long> QueryCountAsync(ClickHouseHttpConfig cfg, string where, CancellationToken cancellationToken)
    {
        var query = $"SELECT count() AS total FROM node_events WHERE {where} FORMAT JSONEachRow";
        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return 0;
        }

        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            try
            {
                using var doc = JsonDocument.Parse(row);
                if (doc.RootElement.TryGetProperty("total", out var value) && value.TryGetInt64(out var total))
                {
                    return total;
                }
            }
            catch
            {
                return 0;
            }
        }

        return 0;
    }

    private static async Task<List<EventLogItem>> QueryRowsAsync(
        ClickHouseHttpConfig cfg,
        string where,
        int pageSize,
        int offset,
        CancellationToken cancellationToken)
    {
        var query = "SELECT ts, node_id, node_ip, event_type, payload " +
                    $"FROM node_events WHERE {where} ORDER BY ts DESC LIMIT {pageSize} OFFSET {offset} FORMAT JSONEachRow";
        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return [];
        }

        var result = new List<EventLogItem>(rows.Length);
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            try
            {
                using var doc = JsonDocument.Parse(row);
                var root = doc.RootElement;
                var payload = ReadString(root, "payload");
                using var payloadDoc = TryParsePayload(payload);

                result.Add(new EventLogItem
                {
                    Timestamp = ReadString(root, "ts"),
                    NodeId = ReadString(root, "node_id"),
                    NodeIp = ReadString(root, "node_ip"),
                    EventType = ReadString(root, "event_type"),
                    Payload = payload,
                    TraceId = ReadPayloadString(payloadDoc, "trace_id"),
                    Host = ReadPayloadString(payloadDoc, "host", "fields.host"),
                    Status = ReadPayloadInt(payloadDoc, "status", "fields.status"),
                    StatusCode = ReadPayloadInt(payloadDoc, "status_code"),
                    Direction = ReadPayloadString(payloadDoc, "direction"),
                    Channel = ReadPayloadString(payloadDoc, "channel"),
                    Kind = ReadPayloadString(payloadDoc, "kind"),
                    Method = ReadPayloadString(payloadDoc, "method"),
                    Path = ReadPayloadString(payloadDoc, "path"),
                    MsgId = ReadPayloadString(payloadDoc, "msg_id"),
                    TaskId = ReadPayloadLong(payloadDoc, "task_id"),
                    TaskType = ReadPayloadString(payloadDoc, "task_type"),
                    SyncAction = ReadPayloadString(payloadDoc, "sync_action", "action"),
                    RateLimitMaxEventsPerSec = ReadPayloadInt(payloadDoc, "rate_limit_max_events_per_sec"),
                    RateLimitWindowSecond = ReadPayloadLong(payloadDoc, "window_second"),
                    RateLimitAcceptedEvents = ReadPayloadInt(payloadDoc, "accepted_events"),
                    RateLimitDroppedEvents = ReadPayloadInt(payloadDoc, "dropped_events")
                });
            }
            catch
            {
                // ignore malformed row
            }
        }

        return result;
    }

    private static JsonDocument? TryParsePayload(string? payload)
    {
        if (string.IsNullOrWhiteSpace(payload))
        {
            return null;
        }

        try
        {
            return JsonDocument.Parse(payload);
        }
        catch
        {
            return null;
        }
    }

    private static string? ReadString(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var value))
        {
            return null;
        }

        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString(),
            JsonValueKind.Number => value.ToString(),
            JsonValueKind.True => "true",
            JsonValueKind.False => "false",
            _ => value.ToString()
        };
    }

    private static string? ReadPayloadString(JsonDocument? payloadDoc, params string[] paths)
    {
        if (payloadDoc == null || paths == null || paths.Length == 0)
        {
            return null;
        }

        foreach (var path in paths)
        {
            if (TryReadPath(payloadDoc.RootElement, path, out var value))
            {
                return value.ValueKind switch
                {
                    JsonValueKind.String => value.GetString(),
                    JsonValueKind.Number => value.ToString(),
                    JsonValueKind.True => "true",
                    JsonValueKind.False => "false",
                    _ => value.ToString()
                };
            }
        }

        return null;
    }

    private static int? ReadPayloadInt(JsonDocument? payloadDoc, params string[] paths)
    {
        if (payloadDoc == null || paths == null || paths.Length == 0)
        {
            return null;
        }

        foreach (var path in paths)
        {
            if (!TryReadPath(payloadDoc.RootElement, path, out var value))
            {
                continue;
            }

            if (value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var intValue))
            {
                return intValue;
            }

            if (value.ValueKind == JsonValueKind.String && int.TryParse(value.GetString(), out intValue))
            {
                return intValue;
            }
        }

        return null;
    }

    private static long? ReadPayloadLong(JsonDocument? payloadDoc, params string[] paths)
    {
        if (payloadDoc == null || paths == null || paths.Length == 0)
        {
            return null;
        }

        foreach (var path in paths)
        {
            if (!TryReadPath(payloadDoc.RootElement, path, out var value))
            {
                continue;
            }

            if (value.ValueKind == JsonValueKind.Number && value.TryGetInt64(out var longValue))
            {
                return longValue;
            }

            if (value.ValueKind == JsonValueKind.String && long.TryParse(value.GetString(), out longValue))
            {
                return longValue;
            }
        }

        return null;
    }

    private static bool TryReadPath(JsonElement root, string path, out JsonElement value)
    {
        value = root;
        if (string.IsNullOrWhiteSpace(path))
        {
            return false;
        }

        var current = root;
        var parts = path.Split('.', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
        foreach (var part in parts)
        {
            if (current.ValueKind != JsonValueKind.Object || !current.TryGetProperty(part, out current))
            {
                return false;
            }
        }

        value = current;
        return true;
    }
}
