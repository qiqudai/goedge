namespace Cnn.Agent.Logs;

public static class LogChannels
{
    public const string Access = "access";
    public const string StreamAccess = "stream_access";
    public const string Security = "security";
    public const string System = "system";
    public const string Debug = "debug";
    public const string ManualDebug = "manual_debug";
    public const string Metrics = "metrics";
}

public sealed record LogEvent(
    DateTimeOffset Timestamp,
    string Channel,
    string Level,
    string Event,
    string TraceId,
    IReadOnlyDictionary<string, object?> Fields);
