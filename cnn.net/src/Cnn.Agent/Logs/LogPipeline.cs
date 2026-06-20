using System.Collections.Concurrent;
using System.Diagnostics;
using System.Threading.Channels;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;

namespace Cnn.Agent.Logs;

public interface ILogEventWriter
{
    bool TryWrite(LogEvent logEvent);
}

public interface ILogSink
{
    ValueTask WriteBatchAsync(IReadOnlyList<LogEvent> events, CancellationToken cancellationToken);
}

public interface ILogPipelineStats
{
    IReadOnlyDictionary<string, long> GetDroppedByChannel();
}

public sealed class LogPipeline : BackgroundService, ILogEventWriter, ILogPipelineStats
{
    private readonly Channel<LogEvent> _channel;
    private readonly IReadOnlyList<ILogSink> _sinks;
    private readonly ILogger<LogPipeline> _logger;
    private readonly LogPipelineOptions _options;
    private readonly ConcurrentDictionary<string, long> _droppedByChannel = new(StringComparer.OrdinalIgnoreCase);
    private DateTimeOffset _lastDropLogAt = DateTimeOffset.MinValue;

    public LogPipeline(IEnumerable<ILogSink> sinks, IOptions<LogPipelineOptions> options, ILogger<LogPipeline> logger)
    {
        _sinks = sinks?.ToList() ?? [];
        _logger = logger;
        _options = options?.Value ?? new LogPipelineOptions();
        if (_options.BatchSize <= 0)
        {
            _options.BatchSize = 512;
        }

        if (_options.FlushIntervalMs <= 0)
        {
            _options.FlushIntervalMs = 1000;
        }

        if (_options.MaxQueue <= 0)
        {
            _options.MaxQueue = 200_000;
        }

        if (_options.HighPriorityWriteTimeoutMs < 0)
        {
            _options.HighPriorityWriteTimeoutMs = 0;
        }

        if (_options.DropSummaryIntervalSeconds <= 0)
        {
            _options.DropSummaryIntervalSeconds = 30;
        }

        _channel = Channel.CreateBounded<LogEvent>(new BoundedChannelOptions(_options.MaxQueue)
        {
            SingleReader = true,
            SingleWriter = false,
            FullMode = BoundedChannelFullMode.Wait
        });
    }

    public bool TryWrite(LogEvent logEvent)
    {
        if (logEvent == null)
        {
            return false;
        }

        var channel = LogChannelCatalog.NormalizeChannel(logEvent.Channel);
        var normalized = string.Equals(logEvent.Channel, channel, StringComparison.Ordinal)
            ? logEvent
            : logEvent with { Channel = channel };

        if (_channel.Writer.TryWrite(normalized))
        {
            return true;
        }

        if (LogChannelCatalog.IsHighPriority(channel) && TryWriteHighPriority(normalized))
        {
            return true;
        }

        RecordDrop(channel);
        return false;
    }

    public IReadOnlyDictionary<string, long> GetDroppedByChannel()
    {
        return _droppedByChannel.ToDictionary(pair => pair.Key, pair => pair.Value, StringComparer.OrdinalIgnoreCase);
    }

    private bool TryWriteHighPriority(LogEvent logEvent)
    {
        if (_options.HighPriorityWriteTimeoutMs <= 0)
        {
            return false;
        }

        var timeout = TimeSpan.FromMilliseconds(_options.HighPriorityWriteTimeoutMs);
        var stopwatch = Stopwatch.StartNew();
        var spinner = new SpinWait();

        while (stopwatch.Elapsed < timeout)
        {
            if (_channel.Writer.TryWrite(logEvent))
            {
                return true;
            }

            spinner.SpinOnce();
        }

        return false;
    }

    private void RecordDrop(string? channel)
    {
        _droppedByChannel.AddOrUpdate(channel ?? "unknown", 1, static (_, value) => value + 1);
        TryLogDropSummary();
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        var batch = new List<LogEvent>(_options.BatchSize);
        var flushInterval = TimeSpan.FromMilliseconds(_options.FlushIntervalMs);
        var lastFlushAt = DateTimeOffset.UtcNow;
        var timer = new PeriodicTimer(TimeSpan.FromMilliseconds(Math.Max(100, _options.FlushIntervalMs / 2)));

        try
        {
            while (!stoppingToken.IsCancellationRequested)
            {
                while (_channel.Reader.TryRead(out var item))
                {
                    batch.Add(item);
                    if (batch.Count >= _options.BatchSize)
                    {
                        await FlushBatchAsync(batch, stoppingToken);
                        lastFlushAt = DateTimeOffset.UtcNow;
                    }
                }

                if (batch.Count > 0 && DateTimeOffset.UtcNow - lastFlushAt >= flushInterval)
                {
                    await FlushBatchAsync(batch, stoppingToken);
                    lastFlushAt = DateTimeOffset.UtcNow;
                }

                var ticked = await timer.WaitForNextTickAsync(stoppingToken);
                if (!ticked)
                {
                    break;
                }

                TryLogDropSummary();
            }
        }
        catch (OperationCanceledException)
        {
            // graceful stop
        }
        finally
        {
            timer.Dispose();
            if (batch.Count > 0)
            {
                await FlushBatchAsync(batch, CancellationToken.None);
            }
        }
    }

    private async Task FlushBatchAsync(List<LogEvent> batch, CancellationToken cancellationToken)
    {
        if (batch.Count == 0)
        {
            return;
        }

        var toWrite = batch.ToArray();
        batch.Clear();

        foreach (var sink in _sinks)
        {
            try
            {
                await sink.WriteBatchAsync(toWrite, cancellationToken);
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "log sink write failed sink={Sink}", sink.GetType().Name);
            }
        }
    }

    private void TryLogDropSummary()
    {
        var now = DateTimeOffset.UtcNow;
        if (now - _lastDropLogAt < TimeSpan.FromSeconds(_options.DropSummaryIntervalSeconds))
        {
            return;
        }

        if (_droppedByChannel.IsEmpty)
        {
            return;
        }

        _lastDropLogAt = now;
        var snapshot = _droppedByChannel.ToDictionary(pair => pair.Key, pair => pair.Value, StringComparer.OrdinalIgnoreCase);
        _logger.LogWarning("log pipeline dropped events: {@DroppedByChannel}", snapshot);
    }
}
