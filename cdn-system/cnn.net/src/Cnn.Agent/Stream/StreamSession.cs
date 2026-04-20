using System.Buffers;
using System.Net.Sockets;

namespace Cnn.Agent.Stream;

public static class StreamSession
{
    public static async Task RunAsync(TcpClient source, TcpClient destination, TimeSpan idleTimeout, CancellationToken cancellationToken)
    {
        using var linkedCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        var token = linkedCts.Token;

        using var sourceStream = source.GetStream();
        using var destinationStream = destination.GetStream();

        var pumpA = PumpAsync(sourceStream, destinationStream, idleTimeout, token);
        var pumpB = PumpAsync(destinationStream, sourceStream, idleTimeout, token);

        var completed = await Task.WhenAny(pumpA, pumpB);
        linkedCts.Cancel();

        try
        {
            await completed;
        }
        catch
        {
            // ignore session level errors
        }

        try
        {
            await Task.WhenAll(pumpA, pumpB);
        }
        catch
        {
            // ignore
        }
    }

    private static async Task PumpAsync(System.IO.Stream input, System.IO.Stream output, TimeSpan idleTimeout, CancellationToken cancellationToken)
    {
        var buffer = ArrayPool<byte>.Shared.Rent(16 * 1024);
        try
        {
            while (!cancellationToken.IsCancellationRequested)
            {
                var read = await ReadWithIdleTimeoutAsync(input, buffer, idleTimeout, cancellationToken);
                if (read <= 0)
                {
                    break;
                }

                await output.WriteAsync(buffer.AsMemory(0, read), cancellationToken);
                await output.FlushAsync(cancellationToken);
            }
        }
        finally
        {
            ArrayPool<byte>.Shared.Return(buffer);
        }
    }

    private static async Task<int> ReadWithIdleTimeoutAsync(System.IO.Stream stream, byte[] buffer, TimeSpan idleTimeout, CancellationToken cancellationToken)
    {
        var readTask = stream.ReadAsync(buffer.AsMemory(0, buffer.Length), cancellationToken).AsTask();
        var delayTask = Task.Delay(idleTimeout, cancellationToken);
        var completed = await Task.WhenAny(readTask, delayTask);
        if (completed == delayTask)
        {
            throw new TimeoutException("stream idle timeout");
        }

        return await readTask;
    }
}
