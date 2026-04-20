namespace Cnn.Agent.Logs;

public sealed class LogPipelineOptions
{
    public int BatchSize { get; set; } = 512;
    public int FlushIntervalMs { get; set; } = 1000;
    public int MaxQueue { get; set; } = 200_000;
    public int HighPriorityWriteTimeoutMs { get; set; } = 5;
    public int DropSummaryIntervalSeconds { get; set; } = 30;
    public int CleanupIntervalMinutes { get; set; } = 60;
    public int DiskHighWatermarkPercent { get; set; } = 85;
    public Dictionary<string, int> RetentionDays { get; set; } = new(StringComparer.OrdinalIgnoreCase);
}
