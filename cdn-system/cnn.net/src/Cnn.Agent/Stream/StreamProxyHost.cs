using Cnn.Agent.Config;
using Cnn.Common.Contracts.Agent;
using Microsoft.Extensions.Hosting;

namespace Cnn.Agent.Stream;

public sealed class StreamProxyHost : BackgroundService
{
    private readonly EdgeConfigStore _edgeConfigStore;
    private readonly IStreamRuntime _streamRuntime;
    private readonly ILogger<StreamProxyHost> _logger;
    private EdgeConfigDto? _lastConfigRef;

    public StreamProxyHost(
        EdgeConfigStore edgeConfigStore,
        IStreamRuntime streamRuntime,
        ILogger<StreamProxyHost> logger)
    {
        _edgeConfigStore = edgeConfigStore;
        _streamRuntime = streamRuntime;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        TryApplyCurrent(force: true);

        var timer = new PeriodicTimer(TimeSpan.FromSeconds(1));
        try
        {
            while (await timer.WaitForNextTickAsync(stoppingToken))
            {
                TryApplyCurrent(force: false);
            }
        }
        catch (OperationCanceledException)
        {
            // ignore
        }
        finally
        {
            timer.Dispose();
        }
    }

    public override Task StopAsync(CancellationToken cancellationToken)
    {
        try
        {
            var stopConfig = new EdgeConfigDto
            {
                Version = DateTimeOffset.UtcNow.ToUnixTimeSeconds(),
                Streams = new List<EdgeStreamDto>()
            };

            _streamRuntime.Apply(stopConfig);
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "stream host stop cleanup failed");
        }

        return base.StopAsync(cancellationToken);
    }

    private void TryApplyCurrent(bool force)
    {
        var config = _edgeConfigStore.Current;
        if (config == null)
        {
            return;
        }

        if (!force && ReferenceEquals(config, _lastConfigRef))
        {
            return;
        }

        var result = _streamRuntime.Apply(config);
        _lastConfigRef = config;

        if (result.Success)
        {
            if (result.Started > 0 || result.Stopped > 0 || result.Restarted > 0)
            {
                _logger.LogInformation(
                    "stream runtime updated started={Started} stopped={Stopped} restarted={Restarted}",
                    result.Started,
                    result.Stopped,
                    result.Restarted);
            }

            return;
        }

        if (result.Errors.Count > 0)
        {
            _logger.LogWarning("stream runtime apply has errors: {Errors}", string.Join("; ", result.Errors.Take(3)));
        }
    }
}
