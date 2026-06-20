namespace Cnn.Agent.Logs;

public static class LogChannelCatalog
{
    private static readonly Dictionary<string, string> ChannelFiles = new(StringComparer.OrdinalIgnoreCase)
    {
        [LogChannels.Access] = "access.json",
        [LogChannels.StreamAccess] = "stream_access.json",
        [LogChannels.Security] = "security.json",
        [LogChannels.System] = "system.json",
        [LogChannels.Debug] = "debug.json",
        [LogChannels.ManualDebug] = "manual_debug.jsonl",
        [LogChannels.Metrics] = "metrics.json"
    };

    private static readonly HashSet<string> HighPriorityChannels = new(StringComparer.OrdinalIgnoreCase)
    {
        LogChannels.Security,
        LogChannels.System,
        LogChannels.ManualDebug
    };

    private static readonly HashSet<string> PressureDropPreferredChannels = new(StringComparer.OrdinalIgnoreCase)
    {
        LogChannels.Access,
        LogChannels.StreamAccess,
        LogChannels.Debug,
        LogChannels.Metrics,
        LogChannels.ManualDebug
    };

    private static readonly Dictionary<string, int> DefaultRetentionDays = new(StringComparer.OrdinalIgnoreCase)
    {
        [LogChannels.Access] = 14,
        [LogChannels.StreamAccess] = 14,
        [LogChannels.Security] = 60,
        [LogChannels.System] = 30,
        [LogChannels.Debug] = 7,
        [LogChannels.ManualDebug] = 14,
        [LogChannels.Metrics] = 7
    };

    public static IReadOnlyDictionary<string, int> RetentionDays => DefaultRetentionDays;

    public static string NormalizeChannel(string? channel)
    {
        if (string.IsNullOrWhiteSpace(channel))
        {
            return LogChannels.System;
        }

        return channel.Trim().ToLowerInvariant();
    }

    public static bool IsHighPriority(string? channel)
    {
        var normalized = NormalizeChannel(channel);
        return HighPriorityChannels.Contains(normalized);
    }

    public static bool IsPressureDropPreferred(string? channel)
    {
        var normalized = NormalizeChannel(channel);
        return PressureDropPreferredChannels.Contains(normalized);
    }

    public static IEnumerable<string> ListChannels()
    {
        return ChannelFiles.Keys;
    }

    public static bool TryResolveFileName(string? channel, out string fileName)
    {
        var normalized = NormalizeChannel(channel);
        return ChannelFiles.TryGetValue(normalized, out fileName!);
    }

    public static string ResolveFileName(string? channel)
    {
        if (TryResolveFileName(channel, out var fileName))
        {
            return fileName;
        }

        return NormalizeChannel(channel) + ".json";
    }

    public static bool TryResolveChannelFromFileName(string? fileName, out string channel)
    {
        channel = string.Empty;
        if (string.IsNullOrWhiteSpace(fileName))
        {
            return false;
        }

        foreach (var pair in ChannelFiles)
        {
            if (string.Equals(pair.Value, fileName, StringComparison.OrdinalIgnoreCase))
            {
                channel = pair.Key;
                return true;
            }
        }

        return false;
    }
}
