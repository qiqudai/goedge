using System.Net;
using System.Net.Sockets;
using Cnn.Agent.Logs;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Stream;

public sealed class StreamListener
{
    private readonly StreamListenerPlan _plan;
    private readonly ILogger<StreamListener> _logger;
    private readonly ILogEventWriter _logWriter;
    private readonly List<(string Host, int Port)> _targets;
    private readonly object _stateLock = new();
    private TcpListener? _listener;
    private CancellationTokenSource? _cts;
    private Task? _loopTask;
    private string? _lastError;
    private int _activeConnections;
    private int _rrIndex;

    public StreamListener(StreamListenerPlan plan, ILogEventWriter logWriter, ILogger<StreamListener> logger)
    {
        _plan = plan;
        _logWriter = logWriter;
        _logger = logger;
        _targets = ExpandTargets(plan.Targets);
    }

    public string Key => _plan.Key;

    public string Signature => BuildSignature(_plan);

    public async Task<bool> StartAsync(CancellationToken cancellationToken)
    {
        lock (_stateLock)
        {
            if (_listener != null)
            {
                return true;
            }

            _cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            _listener = new TcpListener(_plan.ListenIp, _plan.ListenPort);
        }

        try
        {
            _listener.Start(512);
            _loopTask = Task.Run(() => AcceptLoopAsync(_cts!.Token), CancellationToken.None);
            _logger.LogInformation("stream listener started key={Key}", _plan.Key);
            return true;
        }
        catch (Exception ex)
        {
            _lastError = ex.Message;
            _logger.LogWarning(ex, "stream listener start failed key={Key}", _plan.Key);
            await StopAsync(CancellationToken.None);
            return false;
        }
    }

    public async Task StopAsync(CancellationToken cancellationToken)
    {
        CancellationTokenSource? cts;
        Task? loop;
        TcpListener? listener;
        lock (_stateLock)
        {
            cts = _cts;
            loop = _loopTask;
            listener = _listener;
            _cts = null;
            _loopTask = null;
            _listener = null;
        }

        try
        {
            cts?.Cancel();
        }
        catch
        {
            // ignore
        }

        try
        {
            listener?.Stop();
        }
        catch
        {
            // ignore
        }

        if (loop != null)
        {
            try
            {
                using var timeoutCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
                timeoutCts.CancelAfter(TimeSpan.FromSeconds(2));
                await loop.WaitAsync(timeoutCts.Token);
            }
            catch
            {
                // ignore
            }
        }

        cts?.Dispose();
    }

    public StreamListenerState GetState()
    {
        var running = _listener != null;
        return new StreamListenerState(
            Key: _plan.Key,
            StreamId: _plan.StreamId,
            Listen: $"{_plan.ListenIp}:{_plan.ListenPort}",
            Running: running,
            ActiveConnections: Volatile.Read(ref _activeConnections),
            LastError: _lastError);
    }

    private async Task AcceptLoopAsync(CancellationToken cancellationToken)
    {
        while (!cancellationToken.IsCancellationRequested)
        {
            TcpClient? client = null;
            try
            {
                var listener = _listener;
                if (listener == null)
                {
                    break;
                }

                client = await listener.AcceptTcpClientAsync(cancellationToken);
            }
            catch (OperationCanceledException)
            {
                break;
            }
            catch (Exception ex)
            {
                _lastError = ex.Message;
                _logger.LogWarning(ex, "stream accept failed key={Key}", _plan.Key);
                await Task.Delay(TimeSpan.FromMilliseconds(200), cancellationToken);
                continue;
            }

            if (client == null)
            {
                continue;
            }

            var active = Interlocked.Increment(ref _activeConnections);
            if (active > _plan.MaxConns)
            {
                Interlocked.Decrement(ref _activeConnections);
                client.Dispose();
                continue;
            }

            _ = HandleSessionAsync(client, cancellationToken);
        }
    }

    private async Task HandleSessionAsync(TcpClient downstream, CancellationToken cancellationToken)
    {
        var started = DateTimeOffset.UtcNow;
        string status = "ok";
        string? targetAddr = null;
        try
        {
            var target = SelectTarget();
            if (target == null)
            {
                status = "no_target";
                return;
            }

            targetAddr = $"{target.Value.Host}:{target.Value.Port}";

            using var upstream = new TcpClient();
            var connectTask = upstream.ConnectAsync(target.Value.Host, target.Value.Port, cancellationToken).AsTask();
            var timeoutTask = Task.Delay(_plan.ConnectTimeout, cancellationToken);
            var completed = await Task.WhenAny(connectTask, timeoutTask);
            if (completed == timeoutTask)
            {
                status = "connect_timeout";
                return;
            }

            await connectTask;

            await StreamSession.RunAsync(downstream, upstream, _plan.IdleTimeout, cancellationToken);
        }
        catch (TimeoutException)
        {
            status = "idle_timeout";
        }
        catch (OperationCanceledException)
        {
            status = "cancelled";
        }
        catch (Exception ex)
        {
            _lastError = ex.Message;
            status = "error";
            _logger.LogDebug(ex, "stream session failed key={Key}", _plan.Key);
        }
        finally
        {
            try
            {
                downstream.Dispose();
            }
            catch
            {
                // ignore
            }

            Interlocked.Decrement(ref _activeConnections);
            var latencyMs = (DateTimeOffset.UtcNow - started).TotalMilliseconds;
            _ = _logWriter.TryWrite(new LogEvent(
                DateTimeOffset.UtcNow,
                LogChannels.StreamAccess,
                "information",
                "stream_session",
                Guid.NewGuid().ToString("N"),
                new Dictionary<string, object?>
                {
                    ["stream_id"] = _plan.StreamId,
                    ["listen"] = $"{_plan.ListenIp}:{_plan.ListenPort}",
                    ["target"] = targetAddr ?? string.Empty,
                    ["status"] = status,
                    ["latency_ms"] = Math.Round(latencyMs, 3)
                }));
        }
    }

    private (string Host, int Port)? SelectTarget()
    {
        if (_targets.Count == 0)
        {
            return null;
        }

        var next = Interlocked.Increment(ref _rrIndex);
        var index = Math.Abs(next) % _targets.Count;
        return _targets[index];
    }

    private static List<(string Host, int Port)> ExpandTargets(IReadOnlyList<EdgeStreamTargetDto> targets)
    {
        var result = new List<(string Host, int Port)>();
        foreach (var item in targets)
        {
            if (item == null)
            {
                continue;
            }

            var target = item;
            var raw = target.Addr?.Trim() ?? string.Empty;
            if (!StreamRouteCompiler.TryParseTarget(raw, out var host, out var port, out _))
            {
                continue;
            }

            var weight = target.Weight > 0 ? target.Weight : 1;
            if (weight > 32)
            {
                weight = 32;
            }

            for (var i = 0; i < weight; i++)
            {
                result.Add((host, port));
            }
        }

        return result;
    }

    public static string BuildSignature(StreamListenerPlan plan)
    {
        var targets = string.Join(",", plan.Targets.Select(t => $"{t.Addr}:{t.Weight}:{t.Enable}"));
        return $"{plan.Key}|{plan.StreamId}|{plan.BalanceWay}|{plan.MaxConns}|{plan.ConnectTimeout.TotalMilliseconds}|{plan.IdleTimeout.TotalMilliseconds}|{targets}";
    }
}
