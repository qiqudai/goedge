using Cnn.Agent.Config;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;

namespace Cnn.Agent.Logs;

public sealed record LogRetentionResult(
    int RemovedFiles,
    long ReclaimedBytes,
    int DiskUsagePercent);

public interface ILogRetentionService
{
    Task<LogRetentionResult> RunOnceAsync(CancellationToken cancellationToken);
}

public sealed class LogRetentionService : ILogRetentionService
{
    private readonly AgentRuntimePaths _paths;
    private readonly IOptionsMonitor<LogPipelineOptions> _options;
    private readonly ILogger<LogRetentionService> _logger;

    public LogRetentionService(
        AgentRuntimePaths paths,
        IOptionsMonitor<LogPipelineOptions> options,
        ILogger<LogRetentionService> logger)
    {
        _paths = paths;
        _options = options;
        _logger = logger;
    }

    public Task<LogRetentionResult> RunOnceAsync(CancellationToken cancellationToken)
    {
        Directory.CreateDirectory(_paths.LogsDir);
        var now = DateTimeOffset.UtcNow;
        var options = _options.CurrentValue ?? new LogPipelineOptions();
        var retentionDays = BuildRetentionDays(options);

        var removedFiles = 0;
        long reclaimedBytes = 0;

        foreach (var path in Directory.EnumerateFiles(_paths.LogsDir))
        {
            cancellationToken.ThrowIfCancellationRequested();

            var fileName = Path.GetFileName(path);
            if (!LogChannelCatalog.TryResolveChannelFromFileName(fileName, out var channel))
            {
                continue;
            }

            if (!retentionDays.TryGetValue(channel, out var days) || days <= 0)
            {
                continue;
            }

            var info = new FileInfo(path);
            if (info.LastWriteTimeUtc >= now.AddDays(-days))
            {
                continue;
            }

            reclaimedBytes += SafeDelete(info);
            removedFiles++;

            var offsetPath = Path.Combine(_paths.LogsDir, Path.GetFileNameWithoutExtension(path) + ".offset");
            reclaimedBytes += SafeDelete(new FileInfo(offsetPath));
        }

        var usage = ResolveDiskUsagePercent(_paths.LogsDir);
        if (usage >= options.DiskHighWatermarkPercent)
        {
            var pressure = ApplyPressureCleanup(cancellationToken);
            removedFiles += pressure.RemovedFiles;
            reclaimedBytes += pressure.ReclaimedBytes;
            usage = ResolveDiskUsagePercent(_paths.LogsDir);
        }

        if (removedFiles > 0)
        {
            _logger.LogInformation(
                "log retention cleaned removed_files={RemovedFiles} reclaimed_bytes={ReclaimedBytes} disk_usage={DiskUsagePercent}",
                removedFiles,
                reclaimedBytes,
                usage);
        }

        return Task.FromResult(new LogRetentionResult(removedFiles, reclaimedBytes, usage));
    }

    private LogRetentionResult ApplyPressureCleanup(CancellationToken cancellationToken)
    {
        var candidates = Directory
            .EnumerateFiles(_paths.LogsDir)
            .Select(path => new FileInfo(path))
            .Where(info => LogChannelCatalog.TryResolveChannelFromFileName(info.Name, out var channel)
                && LogChannelCatalog.IsPressureDropPreferred(channel)
                && info.Exists
                && info.Length > 0)
            .OrderBy(info => info.LastWriteTimeUtc)
            .ToArray();

        var removedFiles = 0;
        long reclaimedBytes = 0;
        foreach (var info in candidates)
        {
            cancellationToken.ThrowIfCancellationRequested();
            reclaimedBytes += SafeDelete(info);
            removedFiles++;

            var offsetPath = Path.Combine(_paths.LogsDir, Path.GetFileNameWithoutExtension(info.Name) + ".offset");
            reclaimedBytes += SafeDelete(new FileInfo(offsetPath));

            var usage = ResolveDiskUsagePercent(_paths.LogsDir);
            if (usage <= 80)
            {
                break;
            }
        }

        return new LogRetentionResult(removedFiles, reclaimedBytes, ResolveDiskUsagePercent(_paths.LogsDir));
    }

    private static Dictionary<string, int> BuildRetentionDays(LogPipelineOptions options)
    {
        var map = LogChannelCatalog.RetentionDays
            .ToDictionary(pair => pair.Key, pair => pair.Value, StringComparer.OrdinalIgnoreCase);

        if (options.RetentionDays == null || options.RetentionDays.Count == 0)
        {
            return map;
        }

        foreach (var pair in options.RetentionDays)
        {
            if (string.IsNullOrWhiteSpace(pair.Key))
            {
                continue;
            }

            map[LogChannelCatalog.NormalizeChannel(pair.Key)] = pair.Value;
        }

        return map;
    }

    private static long SafeDelete(FileInfo info)
    {
        if (info == null || !info.Exists)
        {
            return 0;
        }

        try
        {
            var bytes = info.Length;
            info.Delete();
            return bytes;
        }
        catch
        {
            return 0;
        }
    }

    private static int ResolveDiskUsagePercent(string path)
    {
        try
        {
            var full = Path.GetFullPath(path);
            var root = Path.GetPathRoot(full);
            if (string.IsNullOrWhiteSpace(root))
            {
                return 0;
            }

            var drive = new DriveInfo(root);
            if (!drive.IsReady || drive.TotalSize <= 0)
            {
                return 0;
            }

            var used = drive.TotalSize - drive.AvailableFreeSpace;
            return (int)Math.Round(used * 100d / drive.TotalSize, MidpointRounding.AwayFromZero);
        }
        catch
        {
            return 0;
        }
    }
}

public sealed class LogRetentionWorker : BackgroundService
{
    private readonly ILogRetentionService _retentionService;
    private readonly IOptionsMonitor<LogPipelineOptions> _options;
    private readonly ILogger<LogRetentionWorker> _logger;

    public LogRetentionWorker(
        ILogRetentionService retentionService,
        IOptionsMonitor<LogPipelineOptions> options,
        ILogger<LogRetentionWorker> logger)
    {
        _retentionService = retentionService;
        _options = options;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await _retentionService.RunOnceAsync(stoppingToken);
            }
            catch (OperationCanceledException)
            {
                // graceful stop
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "log retention run failed");
            }

            var minutes = _options.CurrentValue?.CleanupIntervalMinutes ?? 60;
            if (minutes < 5)
            {
                minutes = 5;
            }

            try
            {
                await Task.Delay(TimeSpan.FromMinutes(minutes), stoppingToken);
            }
            catch (OperationCanceledException)
            {
                // graceful stop
            }
        }
    }
}
