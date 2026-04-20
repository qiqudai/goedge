using System.Text.Json;
using Cnn.Agent.Config;

namespace Cnn.Agent.Logs;

public sealed record LogQueryRequest(
    DateTimeOffset? From = null,
    DateTimeOffset? To = null,
    string? Channel = null,
    string? Level = null,
    string? TraceId = null,
    string? NodeId = null,
    string? Host = null,
    int? Status = null,
    int Page = 1,
    int PageSize = 100);

public sealed record LogQueryItem(
    DateTimeOffset Timestamp,
    string Channel,
    string Level,
    string Event,
    string TraceId,
    IReadOnlyDictionary<string, object?> Fields,
    string Raw);

public sealed record LogQueryResult(
    int Total,
    int Page,
    int PageSize,
    IReadOnlyList<LogQueryItem> Items);

public interface ILogQueryService
{
    Task<LogQueryResult> QueryAsync(LogQueryRequest request, CancellationToken cancellationToken);
}

public sealed class FileLogQueryService : ILogQueryService
{
    private const int MaxPageSize = 1000;
    private readonly AgentRuntimePaths _paths;

    public FileLogQueryService(AgentRuntimePaths paths)
    {
        _paths = paths;
    }

    public Task<LogQueryResult> QueryAsync(LogQueryRequest request, CancellationToken cancellationToken)
    {
        request ??= new LogQueryRequest();
        var page = request.Page <= 0 ? 1 : request.Page;
        var pageSize = request.PageSize <= 0 ? 100 : Math.Min(request.PageSize, MaxPageSize);
        var skip = (page - 1) * pageSize;

        var files = ResolveLogFiles(request.Channel);
        var total = 0;
        var items = new List<LogQueryItem>(pageSize);

        foreach (var file in files)
        {
            cancellationToken.ThrowIfCancellationRequested();

            if (!File.Exists(file.Path))
            {
                continue;
            }

            foreach (var raw in File.ReadLines(file.Path))
            {
                cancellationToken.ThrowIfCancellationRequested();

                var line = raw?.Trim();
                if (string.IsNullOrWhiteSpace(line))
                {
                    continue;
                }

                if (!TryParse(line, file.Channel, out var item))
                {
                    continue;
                }

                if (!Match(item!, request))
                {
                    continue;
                }

                total++;
                if (total <= skip || items.Count >= pageSize)
                {
                    continue;
                }

                items.Add(item!);
            }
        }

        return Task.FromResult(new LogQueryResult(total, page, pageSize, items));
    }

    private IEnumerable<(string Channel, string Path)> ResolveLogFiles(string? channel)
    {
        if (!Directory.Exists(_paths.LogsDir))
        {
            return [];
        }

        if (!string.IsNullOrWhiteSpace(channel))
        {
            var normalized = LogChannelCatalog.NormalizeChannel(channel);
            var fileName = LogChannelCatalog.ResolveFileName(normalized);
            return [(normalized, Path.Combine(_paths.LogsDir, fileName))];
        }

        return LogChannelCatalog.ListChannels()
            .Select(ch => (ch, Path.Combine(_paths.LogsDir, LogChannelCatalog.ResolveFileName(ch))));
    }

    private static bool TryParse(string raw, string fallbackChannel, out LogQueryItem? item)
    {
        item = null;
        try
        {
            using var document = JsonDocument.Parse(raw);
            var root = document.RootElement;
            if (root.ValueKind != JsonValueKind.Object)
            {
                return false;
            }

            var timestamp = DateTimeOffset.UtcNow;
            if (root.TryGetProperty("timestamp", out var tsElement))
            {
                if (tsElement.ValueKind == JsonValueKind.String)
                {
                    if (DateTimeOffset.TryParse(tsElement.GetString(), out var parsed))
                    {
                        timestamp = parsed;
                    }
                }
                else if (tsElement.ValueKind == JsonValueKind.Number && tsElement.TryGetInt64(out var unix))
                {
                    timestamp = DateTimeOffset.FromUnixTimeSeconds(unix);
                }
            }

            var channel = root.TryGetProperty("channel", out var chElement)
                ? LogChannelCatalog.NormalizeChannel(chElement.GetString())
                : LogChannelCatalog.NormalizeChannel(fallbackChannel);

            var level = root.TryGetProperty("level", out var lvElement)
                ? lvElement.GetString() ?? string.Empty
                : string.Empty;
            var eventName = root.TryGetProperty("event", out var eventElement)
                ? eventElement.GetString() ?? string.Empty
                : string.Empty;
            var traceId = root.TryGetProperty("trace_id", out var traceElement)
                ? traceElement.GetString() ?? string.Empty
                : string.Empty;

            var fields = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
            if (root.TryGetProperty("fields", out var fieldsElement) && fieldsElement.ValueKind == JsonValueKind.Object)
            {
                foreach (var property in fieldsElement.EnumerateObject())
                {
                    fields[property.Name] = ReadValue(property.Value);
                }
            }

            item = new LogQueryItem(timestamp, channel, level, eventName, traceId, fields, raw);
            return true;
        }
        catch
        {
            return false;
        }
    }

    private static bool Match(LogQueryItem item, LogQueryRequest request)
    {
        if (request.From.HasValue && item.Timestamp < request.From.Value)
        {
            return false;
        }

        if (request.To.HasValue && item.Timestamp > request.To.Value)
        {
            return false;
        }

        if (!string.IsNullOrWhiteSpace(request.Channel) &&
            !string.Equals(item.Channel, LogChannelCatalog.NormalizeChannel(request.Channel), StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        if (!string.IsNullOrWhiteSpace(request.Level) &&
            !string.Equals(item.Level, request.Level, StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        if (!string.IsNullOrWhiteSpace(request.TraceId) &&
            !string.Equals(item.TraceId, request.TraceId, StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        if (!MatchField(item, "node_id", request.NodeId))
        {
            return false;
        }

        if (!MatchField(item, "host", request.Host))
        {
            return false;
        }

        if (request.Status.HasValue)
        {
            if (!TryReadInt(item, "status", out var status) || status != request.Status.Value)
            {
                return false;
            }
        }

        return true;
    }

    private static bool MatchField(LogQueryItem item, string key, string? expected)
    {
        if (string.IsNullOrWhiteSpace(expected))
        {
            return true;
        }

        if (!item.Fields.TryGetValue(key, out var value) || value == null)
        {
            return false;
        }

        return string.Equals(Convert.ToString(value), expected.Trim(), StringComparison.OrdinalIgnoreCase);
    }

    private static bool TryReadInt(LogQueryItem item, string key, out int value)
    {
        value = 0;
        if (!item.Fields.TryGetValue(key, out var raw) || raw == null)
        {
            return false;
        }

        return raw switch
        {
            int i => Assign(i, out value),
            long l when l is >= int.MinValue and <= int.MaxValue => Assign((int)l, out value),
            double d when d is >= int.MinValue and <= int.MaxValue => Assign((int)d, out value),
            _ => int.TryParse(Convert.ToString(raw), out value)
        };
    }

    private static bool Assign(int input, out int output)
    {
        output = input;
        return true;
    }

    private static object? ReadValue(JsonElement element)
    {
        return element.ValueKind switch
        {
            JsonValueKind.Null => null,
            JsonValueKind.True => true,
            JsonValueKind.False => false,
            JsonValueKind.Number => ReadNumber(element),
            JsonValueKind.String => element.GetString(),
            JsonValueKind.Array => element.EnumerateArray().Select(ReadValue).ToArray(),
            JsonValueKind.Object => element.EnumerateObject().ToDictionary(
                static p => p.Name,
                static p => ReadValue(p.Value),
                StringComparer.OrdinalIgnoreCase),
            _ => element.ToString()
        };
    }

    private static object ReadNumber(JsonElement element)
    {
        if (element.TryGetInt64(out var i64))
        {
            return i64;
        }

        if (element.TryGetDecimal(out var dec))
        {
            return dec;
        }

        return element.GetDouble();
    }
}
