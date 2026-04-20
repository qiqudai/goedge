using System.Text;
using System.Text.Json;
using Cnn.Agent.Config;

namespace Cnn.Agent.Logs;

public sealed class FileLogSink : ILogSink
{
    private readonly AgentRuntimePaths _paths;
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);

    public FileLogSink(AgentRuntimePaths paths)
    {
        _paths = paths;
    }

    public ValueTask WriteBatchAsync(IReadOnlyList<LogEvent> events, CancellationToken cancellationToken)
    {
        if (events == null || events.Count == 0)
        {
            return ValueTask.CompletedTask;
        }

        Directory.CreateDirectory(_paths.LogsDir);

        var builders = new Dictionary<string, StringBuilder>(StringComparer.OrdinalIgnoreCase);
        foreach (var item in events)
        {
            cancellationToken.ThrowIfCancellationRequested();

            var channel = NormalizeChannel(item.Channel);
            var filePath = ResolvePath(channel);
            if (!builders.TryGetValue(filePath, out var builder))
            {
                builder = new StringBuilder();
                builders[filePath] = builder;
            }

            var payload = new Dictionary<string, object?>
            {
                ["timestamp"] = item.Timestamp,
                ["channel"] = channel,
                ["level"] = string.IsNullOrWhiteSpace(item.Level) ? "information" : item.Level,
                ["event"] = string.IsNullOrWhiteSpace(item.Event) ? "event" : item.Event,
                ["trace_id"] = item.TraceId,
                ["fields"] = item.Fields
            };

            builder.AppendLine(JsonSerializer.Serialize(payload, JsonOptions));
        }

        foreach (var pair in builders)
        {
            File.AppendAllText(pair.Key, pair.Value.ToString());
        }

        return ValueTask.CompletedTask;
    }

    private string ResolvePath(string channel)
    {
        var fileName = LogChannelCatalog.ResolveFileName(channel);
        return Path.Combine(_paths.LogsDir, fileName);
    }

    private static string NormalizeChannel(string? channel)
    {
        return LogChannelCatalog.NormalizeChannel(channel);
    }
}
