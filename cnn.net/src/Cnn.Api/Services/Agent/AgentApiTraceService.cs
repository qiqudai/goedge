using System.Text.Json;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Agent;

public interface IAgentApiTraceService
{
    Task TraceAsync(AgentApiTraceRecord record, CancellationToken cancellationToken);
}

public sealed class AgentApiTraceRecord
{
    public string? NodeId { get; set; }
    public string? NodeIp { get; set; }
    public string? Direction { get; set; }
    public string? Channel { get; set; }
    public string? Kind { get; set; }
    public string? Path { get; set; }
    public string? Method { get; set; }
    public int? StatusCode { get; set; }
    public string? TraceId { get; set; }
    public string? Payload { get; set; }
}

public sealed class AgentApiTraceService : IAgentApiTraceService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);

    private readonly ISystemConfigService _systemConfigService;
    private readonly IAgentLogService _agentLogService;

    private readonly Lock _cacheLock = new();
    private readonly Lock _rateLock = new();
    private TraceOptions _cached = TraceOptions.Default;
    private DateTime _cacheUntil = DateTime.MinValue;
    private long _rateWindowSecond;
    private int _rateWindowCount;
    private int _rateWindowDropped;

    public AgentApiTraceService(ISystemConfigService systemConfigService, IAgentLogService agentLogService)
    {
        _systemConfigService = systemConfigService;
        _agentLogService = agentLogService;
    }

    public async Task TraceAsync(AgentApiTraceRecord record, CancellationToken cancellationToken)
    {
        if (record == null)
        {
            return;
        }

        var options = await LoadOptionsAsync(cancellationToken);
        if (!options.Enabled)
        {
            return;
        }

        var accepted = TryConsumeRateQuota(options.MaxEventsPerSec, out var rateSummary);
        if (rateSummary != null)
        {
            await WriteRateSummaryAsync(record, rateSummary, cancellationToken);
        }

        if (!accepted)
        {
            return;
        }

        var payload = record.Payload ?? string.Empty;
        var payloadTruncated = false;
        if (!options.PayloadEnabled)
        {
            payload = string.Empty;
        }
        else if (payload.Length > options.MaxPayload)
        {
            payload = payload[..options.MaxPayload];
            payloadTruncated = true;
        }

        var data = JsonSerializer.Serialize(new
        {
            direction = Normalize(record.Direction, "in"),
            channel = Normalize(record.Channel, "ws"),
            kind = Normalize(record.Kind, "unknown"),
            path = NullIfWhiteSpace(record.Path),
            method = Normalize(record.Method, string.Empty),
            status_code = record.StatusCode,
            trace_id = NullIfWhiteSpace(record.TraceId),
            payload = options.PayloadEnabled ? payload : null,
            payload_truncated = options.PayloadEnabled ? payloadTruncated : (bool?)null
        }, JsonOptions);

        _ = await _agentLogService.InsertEventLogsAsync(
            record.NodeId,
            record.NodeIp,
            "agent_api_trace",
            new[] { data },
            cancellationToken);
    }

    private async Task<TraceOptions> LoadOptionsAsync(CancellationToken cancellationToken)
    {
        var now = DateTime.UtcNow;
        lock (_cacheLock)
        {
            if (now <= _cacheUntil)
            {
                return _cached;
            }
        }

        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        var options = new TraceOptions
        {
            Enabled = ReadBool(cfg, DebugSwitchKeys.AgentApiTraceEnabled, false),
            PayloadEnabled = ReadBool(cfg, DebugSwitchKeys.AgentApiTracePayloadEnabled, false),
            MaxPayload = ReadInt(cfg, DebugSwitchKeys.AgentApiTraceMaxPayload, 2048, 256, 65536),
            MaxEventsPerSec = ReadInt(cfg, DebugSwitchKeys.AgentApiTraceMaxEventsPerSec, 0, 0, 50000)
        };

        lock (_cacheLock)
        {
            _cached = options;
            _cacheUntil = DateTime.UtcNow.AddSeconds(5);
            return _cached;
        }
    }

    private static bool ReadBool(IReadOnlyDictionary<string, string> map, string key, bool defaultValue)
    {
        if (map.TryGetValue(key, out var value))
        {
            var normalized = value?.Trim().ToLowerInvariant();
            if (normalized is "1" or "true" or "yes" or "on")
            {
                return true;
            }

            if (normalized is "0" or "false" or "no" or "off")
            {
                return false;
            }
        }

        var alias = key.Replace('-', '_');
        if (!string.Equals(alias, key, StringComparison.Ordinal) && map.TryGetValue(alias, out value))
        {
            var normalized = value?.Trim().ToLowerInvariant();
            if (normalized is "1" or "true" or "yes" or "on")
            {
                return true;
            }

            if (normalized is "0" or "false" or "no" or "off")
            {
                return false;
            }
        }

        return defaultValue;
    }

    private static int ReadInt(IReadOnlyDictionary<string, string> map, string key, int defaultValue, int min, int max)
    {
        if (map.TryGetValue(key, out var value) && int.TryParse(value?.Trim(), out var parsed))
        {
            return Math.Clamp(parsed, min, max);
        }

        var alias = key.Replace('-', '_');
        if (!string.Equals(alias, key, StringComparison.Ordinal) &&
            map.TryGetValue(alias, out value) &&
            int.TryParse(value?.Trim(), out parsed))
        {
            return Math.Clamp(parsed, min, max);
        }

        return Math.Clamp(defaultValue, min, max);
    }

    private static string Normalize(string? value, string fallback)
    {
        var normalized = value?.Trim();
        return string.IsNullOrWhiteSpace(normalized) ? fallback : normalized;
    }

    private static string? NullIfWhiteSpace(string? value)
    {
        var normalized = value?.Trim();
        return string.IsNullOrWhiteSpace(normalized) ? null : normalized;
    }

    private bool TryConsumeRateQuota(int maxEventsPerSec, out RateLimitSummary? summary)
    {
        summary = null;
        if (maxEventsPerSec <= 0)
        {
            return true;
        }

        var currentSecond = DateTimeOffset.UtcNow.ToUnixTimeSeconds();
        lock (_rateLock)
        {
            if (_rateWindowSecond != currentSecond)
            {
                if (_rateWindowSecond > 0 && _rateWindowDropped > 0)
                {
                    summary = new RateLimitSummary
                    {
                        WindowSecond = _rateWindowSecond,
                        AcceptedEvents = _rateWindowCount,
                        DroppedEvents = _rateWindowDropped,
                        MaxEventsPerSec = maxEventsPerSec
                    };
                }

                _rateWindowSecond = currentSecond;
                _rateWindowCount = 0;
                _rateWindowDropped = 0;
            }

            if (_rateWindowCount >= maxEventsPerSec)
            {
                _rateWindowDropped++;
                return false;
            }

            _rateWindowCount++;
            return true;
        }
    }

    private async Task WriteRateSummaryAsync(AgentApiTraceRecord record, RateLimitSummary summary, CancellationToken cancellationToken)
    {
        var data = JsonSerializer.Serialize(new
        {
            direction = "internal",
            channel = "ws",
            kind = "rate_limit_summary",
            trace_id = NullIfWhiteSpace(record.TraceId),
            rate_limit_max_events_per_sec = summary.MaxEventsPerSec,
            window_second = summary.WindowSecond,
            accepted_events = summary.AcceptedEvents,
            dropped_events = summary.DroppedEvents
        }, JsonOptions);

        _ = await _agentLogService.InsertEventLogsAsync(
            record.NodeId,
            record.NodeIp,
            "agent_api_trace",
            new[] { data },
            cancellationToken);
    }

    private sealed class TraceOptions
    {
        public static readonly TraceOptions Default = new();

        public bool Enabled { get; set; }
        public bool PayloadEnabled { get; set; }
        public int MaxPayload { get; set; } = 2048;
        public int MaxEventsPerSec { get; set; }
    }

    private sealed class RateLimitSummary
    {
        public long WindowSecond { get; set; }
        public int AcceptedEvents { get; set; }
        public int DroppedEvents { get; set; }
        public int MaxEventsPerSec { get; set; }
    }
}
