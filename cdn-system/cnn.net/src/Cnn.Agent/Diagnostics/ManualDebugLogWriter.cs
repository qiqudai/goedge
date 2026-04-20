using Cnn.Agent.Config;
using Cnn.Agent.Logs;

namespace Cnn.Agent.Diagnostics;

public interface IManualDebugLogWriter
{
    void Write(string category, string message, object? data, string? actor);
}

public sealed class ManualDebugLogWriter : IManualDebugLogWriter
{
    private readonly IDebugSwitchStore _switches;
    private readonly ILogEventWriter _logWriter;

    public ManualDebugLogWriter(
        AgentRuntimePaths paths,
        IDebugSwitchStore switches,
        ILogEventWriter logWriter)
    {
        _ = paths;
        _switches = switches;
        _logWriter = logWriter;
    }

    public void Write(string category, string message, object? data, string? actor)
    {
        if (!_switches.IsEnabled(DebugSwitchKeys.ManualDebugLog))
        {
            return;
        }

        if (string.IsNullOrWhiteSpace(message))
        {
            return;
        }

        var record = new Dictionary<string, object?>
        {
            ["timestamp"] = DateTimeOffset.UtcNow,
            ["category"] = string.IsNullOrWhiteSpace(category) ? "manual" : category.Trim(),
            ["message"] = message.Trim(),
            ["actor"] = string.IsNullOrWhiteSpace(actor) ? "unknown" : actor.Trim(),
            ["data"] = data
        };

        var traceId = Guid.NewGuid().ToString("N");
        var payload = DebugLogSanitizer.Sanitize(record);
        _ = _logWriter.TryWrite(new LogEvent(
            DateTimeOffset.UtcNow,
            LogChannels.ManualDebug,
            "debug",
            "manual_debug_log",
            traceId,
            payload));
    }
}
