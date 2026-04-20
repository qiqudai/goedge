namespace Cnn.Agent.Plugin;

public enum PluginBreakerState
{
    Closed = 0,
    Open = 1,
    HalfOpen = 2
}

public sealed class PluginBreakerOptions
{
    public int FailThreshold { get; set; } = 20;
    public int WindowSeconds { get; set; } = 60;
    public int OpenSeconds { get; set; } = 120;
}

public sealed class PluginCircuitBreaker
{
    private readonly object _lock = new();
    private readonly int _failThreshold;
    private readonly int _windowSeconds;
    private readonly int _openSeconds;

    private PluginBreakerState _state = PluginBreakerState.Closed;
    private DateTimeOffset _windowStart = DateTimeOffset.UtcNow;
    private DateTimeOffset _openUntil = DateTimeOffset.MinValue;
    private int _failures;
    private bool _halfOpenProbeInFlight;

    public PluginCircuitBreaker(PluginBreakerOptions? options)
    {
        _failThreshold = Math.Max(1, options?.FailThreshold ?? 20);
        _windowSeconds = Math.Max(1, options?.WindowSeconds ?? 60);
        _openSeconds = Math.Max(1, options?.OpenSeconds ?? 120);
    }

    public bool TryEnter(DateTimeOffset now, out PluginBreakerState state)
    {
        lock (_lock)
        {
            if (_state == PluginBreakerState.Open)
            {
                if (now < _openUntil)
                {
                    state = _state;
                    return false;
                }

                _state = PluginBreakerState.HalfOpen;
                _halfOpenProbeInFlight = false;
            }

            if (_state == PluginBreakerState.HalfOpen)
            {
                if (_halfOpenProbeInFlight)
                {
                    state = _state;
                    return false;
                }

                _halfOpenProbeInFlight = true;
                state = _state;
                return true;
            }

            state = _state;
            return true;
        }
    }

    public void RecordSuccess(DateTimeOffset now)
    {
        lock (_lock)
        {
            _state = PluginBreakerState.Closed;
            _windowStart = now;
            _failures = 0;
            _halfOpenProbeInFlight = false;
            _openUntil = DateTimeOffset.MinValue;
        }
    }

    public void RecordFailure(DateTimeOffset now)
    {
        lock (_lock)
        {
            if (_state == PluginBreakerState.HalfOpen)
            {
                Open(now);
                return;
            }

            if (now - _windowStart >= TimeSpan.FromSeconds(_windowSeconds))
            {
                _windowStart = now;
                _failures = 0;
            }

            _failures++;
            if (_failures >= _failThreshold)
            {
                Open(now);
            }
        }
    }

    public PluginBreakerState GetState(DateTimeOffset now)
    {
        lock (_lock)
        {
            if (_state == PluginBreakerState.Open && now >= _openUntil)
            {
                _state = PluginBreakerState.HalfOpen;
                _halfOpenProbeInFlight = false;
            }

            return _state;
        }
    }

    private void Open(DateTimeOffset now)
    {
        _state = PluginBreakerState.Open;
        _openUntil = now.AddSeconds(_openSeconds);
        _windowStart = now;
        _failures = 0;
        _halfOpenProbeInFlight = false;
    }
}
