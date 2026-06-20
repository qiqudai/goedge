using System.Diagnostics;
using StreamBase = System.IO.Stream;

namespace Cnn.Agent.Security;

public sealed class RateLimitedWriteStream : StreamBase
{
    private readonly StreamBase _inner;
    private readonly double _bytesPerSecond;
    private readonly Func<long> _timestampProvider;
    private readonly Func<TimeSpan, CancellationToken, Task> _delayAsync;
    private readonly Action<TimeSpan> _delaySync;
    private readonly object _lock = new();

    private long _nextAvailableTick;
    private bool _disposed;

    public RateLimitedWriteStream(
        StreamBase inner,
        long bytesPerSecond,
        Func<long>? timestampProvider = null,
        Func<TimeSpan, CancellationToken, Task>? delayAsync = null,
        Action<TimeSpan>? delaySync = null)
    {
        _inner = inner ?? throw new ArgumentNullException(nameof(inner));
        _bytesPerSecond = bytesPerSecond <= 0 ? 0d : bytesPerSecond;
        _timestampProvider = timestampProvider ?? Stopwatch.GetTimestamp;
        _delayAsync = delayAsync ?? ((delay, token) => Task.Delay(delay, token));
        _delaySync = delaySync ?? Thread.Sleep;
    }

    public override bool CanRead => false;
    public override bool CanSeek => false;
    public override bool CanWrite => !_disposed && _inner.CanWrite;
    public override long Length => throw new NotSupportedException();
    public override long Position
    {
        get => throw new NotSupportedException();
        set => throw new NotSupportedException();
    }

    public override void Flush()
    {
        ThrowIfDisposed();
        _inner.Flush();
    }

    public override Task FlushAsync(CancellationToken cancellationToken)
    {
        ThrowIfDisposed();
        return _inner.FlushAsync(cancellationToken);
    }

    public override int Read(byte[] buffer, int offset, int count) => throw new NotSupportedException();
    public override long Seek(long offset, SeekOrigin origin) => throw new NotSupportedException();
    public override void SetLength(long value) => throw new NotSupportedException();

    public override void Write(byte[] buffer, int offset, int count)
    {
        ThrowIfDisposed();
        if (count <= 0)
        {
            return;
        }

        ThrottleSync(count);
        _inner.Write(buffer, offset, count);
    }

    public override async ValueTask WriteAsync(ReadOnlyMemory<byte> buffer, CancellationToken cancellationToken = default)
    {
        ThrowIfDisposed();
        if (buffer.Length <= 0)
        {
            return;
        }

        await ThrottleAsync(buffer.Length, cancellationToken);
        await _inner.WriteAsync(buffer, cancellationToken);
    }

    public override async Task WriteAsync(byte[] buffer, int offset, int count, CancellationToken cancellationToken)
    {
        ThrowIfDisposed();
        if (count <= 0)
        {
            return;
        }

        await ThrottleAsync(count, cancellationToken);
        await _inner.WriteAsync(buffer, offset, count, cancellationToken);
    }

    protected override void Dispose(bool disposing)
    {
        _disposed = true;
        base.Dispose(disposing);
    }

    private void ThrottleSync(int bytes)
    {
        var delay = Schedule(bytes);
        if (delay > TimeSpan.Zero)
        {
            _delaySync(delay);
        }
    }

    private async Task ThrottleAsync(int bytes, CancellationToken cancellationToken)
    {
        var delay = Schedule(bytes);
        if (delay > TimeSpan.Zero)
        {
            await _delayAsync(delay, cancellationToken);
        }
    }

    private TimeSpan Schedule(int bytes)
    {
        if (_bytesPerSecond <= 0d)
        {
            return TimeSpan.Zero;
        }

        var now = _timestampProvider();
        long start;
        lock (_lock)
        {
            start = _nextAvailableTick > now ? _nextAvailableTick : now;
            var durationTicks = (long)Math.Ceiling((bytes / _bytesPerSecond) * Stopwatch.Frequency);
            if (durationTicks < 1)
            {
                durationTicks = 1;
            }

            _nextAvailableTick = start + durationTicks;
        }

        var delayTicks = start - now;
        return delayTicks <= 0
            ? TimeSpan.Zero
            : TimeSpan.FromSeconds(delayTicks / (double)Stopwatch.Frequency);
    }

    private void ThrowIfDisposed()
    {
        if (_disposed)
        {
            throw new ObjectDisposedException(nameof(RateLimitedWriteStream));
        }
    }
}
