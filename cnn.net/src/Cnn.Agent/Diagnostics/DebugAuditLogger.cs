using Cnn.Agent.Logs;

namespace Cnn.Agent.Diagnostics;

public interface IDebugAuditLogger
{
    void WriteSwitchUpdate(string actor, string? reason, int? ttlSeconds, DebugOptions options, IReadOnlyDictionary<string, bool> switches);
}

public sealed class DebugAuditLogger : IDebugAuditLogger
{
    private readonly ILogEventWriter _logWriter;

    public DebugAuditLogger(ILogEventWriter logWriter)
    {
        _logWriter = logWriter;
    }

    public void WriteSwitchUpdate(string actor, string? reason, int? ttlSeconds, DebugOptions options, IReadOnlyDictionary<string, bool> switches)
    {
        var payload = new Dictionary<string, object?>
        {
            ["actor"] = string.IsNullOrWhiteSpace(actor) ? "unknown" : actor.Trim(),
            ["reason"] = reason,
            ["ttl_seconds"] = ttlSeconds,
            ["debug_enabled"] = options.Enabled,
            ["sample_rate"] = options.SampleRate,
            ["max_events_per_sec"] = options.MaxEventsPerSec,
            ["internal_ip_only"] = options.InternalIpOnly,
            ["allow_header_token"] = options.AllowHeaderToken,
            ["allow_query_flag"] = options.AllowQueryFlag,
            ["modules"] = options.Modules,
            ["switches"] = switches
        };

        _ = _logWriter.TryWrite(new LogEvent(
            DateTimeOffset.UtcNow,
            LogChannels.Debug,
            "information",
            "debug_switch_audit",
            Guid.NewGuid().ToString("N"),
            DebugLogSanitizer.Sanitize(payload)));
    }
}
